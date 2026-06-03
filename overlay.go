package main

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExTopmost     = 0x00000008
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000
	wsPopup         = 0x80000000

	swpNoActivate = 0x0010
	swpNoZOrder   = 0x0004
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080

	wmEraseBkgnd = 0x0014
	wmPaint      = 0x000F

	// Snap preview color — blue accent matching Rectangle
	overlayR byte = 37
	overlayG byte = 99
	overlayB byte = 235

	overlayCornerRadius = 14
	overlayTargetAlpha  = byte(165) // ~65% opacity
)

var (
	overlayUser32          = syscall.NewLazyDLL("user32.dll")
	overlayGdi32           = syscall.NewLazyDLL("gdi32.dll")
	overlayKernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateWindowExW    = overlayUser32.NewProc("CreateWindowExW")
	procDestroyWindow      = overlayUser32.NewProc("DestroyWindow")
	procSetWindowPos       = overlayUser32.NewProc("SetWindowPos")
	procSetWindowRgn       = overlayUser32.NewProc("SetWindowRgn") // user32, NOT gdi32
	procCreateRoundRectRgn = overlayGdi32.NewProc("CreateRoundRectRgn")
	procGetModuleHandleW   = overlayKernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW   = overlayUser32.NewProc("RegisterClassExW")
	procDefWindowProcW     = overlayUser32.NewProc("DefWindowProcW")
	procBeginPaint         = overlayUser32.NewProc("BeginPaint")
	procEndPaint           = overlayUser32.NewProc("EndPaint")
	procFillRect           = overlayUser32.NewProc("FillRect")
	procCreateSolidBrush   = overlayGdi32.NewProc("CreateSolidBrush")
	procDeleteObject       = overlayGdi32.NewProc("DeleteObject")
)

// WNDCLASSEXW mirrors the Win32 WNDCLASSEXW struct.
type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

// PAINTSTRUCT mirrors Win32 PAINTSTRUCT.
type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     w32.RECT
	fRestore    int32
	fIncUpdate  int32
	_           [32]byte
}

// overlayWndProc is the custom window procedure for the snap preview overlay.
// It runs on the main OS thread (via DispatchMessage in msgLoop).
var overlayWndProc = syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmEraseBkgnd:
		// Suppress the default (black) background erase; we paint in WM_PAINT.
		return 1

	case wmPaint:
		// With DWM composition, WM_ERASEBKGND is not reliably called on layered
		// windows — paint the blue fill here using the DC from BeginPaint.
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			rc := w32.GetClientRect(w32.HWND(hwnd))
			brush, _, _ := procCreateSolidBrush.Call(uintptr(rgb(overlayR, overlayG, overlayB)))
			if brush != 0 {
				procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
				procDeleteObject.Call(brush)
			}
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
})

const overlayClassName = "SnapFlowOverlay"

func registerOverlayClass(instance uintptr) error {
	name := syscall.StringToUTF16Ptr(overlayClassName)
	wc := wndClassExW{
		lpfnWndProc:   overlayWndProc,
		hInstance:     instance,
		lpszClassName: name,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return fmt.Errorf("RegisterClassExW: %v", err)
	}
	return nil
}

type overlayCmd struct {
	show   bool
	bounds w32.RECT
}

// OverlayRenderer displays the animated snap-preview rectangle.
// Show/Hide are safe to call from any goroutine (non-blocking channel sends).
// The actual Win32 calls happen in animLoop, a background goroutine.
type OverlayRenderer struct {
	hwnd   w32.HWND
	cmdCh  chan overlayCmd
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewOverlayRenderer() (*OverlayRenderer, error) {
	instance, _, _ := procGetModuleHandleW.Call(0)

	if err := registerOverlayClass(instance); err != nil {
		return nil, err
	}

	className := syscall.StringToUTF16Ptr(overlayClassName)
	windowName := syscall.StringToUTF16Ptr("SnapFlow Preview")

	exStyle := uintptr(wsExTopmost | wsExTransparent | wsExToolWindow | wsExLayered | wsExNoActivate)
	r, _, e := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		uintptr(wsPopup),
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if r == 0 {
		return nil, fmt.Errorf("CreateWindowExW: %v", e)
	}
	hwnd := w32.HWND(r)

	// Start fully transparent; animLoop will fade it in.
	if !w32ex.SetLayeredWindowAttributes(hwnd, 0, 0, w32ex.LWA_ALPHA) {
		return nil, fmt.Errorf("SetLayeredWindowAttributes: %d", w32.GetLastError())
	}

	o := &OverlayRenderer{
		hwnd:   hwnd,
		cmdCh:  make(chan overlayCmd, 16),
		stopCh: make(chan struct{}),
	}
	o.wg.Add(1)
	go o.animLoop()
	return o, nil
}

// Show queues the overlay to appear at the given screen coordinates.
func (o *OverlayRenderer) Show(bounds w32.RECT) {
	select {
	case o.cmdCh <- overlayCmd{show: true, bounds: bounds}:
	default:
	}
}

// Hide queues the overlay to fade out.
func (o *OverlayRenderer) Hide() {
	select {
	case o.cmdCh <- overlayCmd{show: false}:
	default:
	}
}

// Close stops the animation goroutine and destroys the overlay window.
func (o *OverlayRenderer) Close() {
	close(o.stopCh)
	o.wg.Wait()
	if o.hwnd != 0 {
		procDestroyWindow.Call(uintptr(o.hwnd))
		o.hwnd = 0
	}
}

func (o *OverlayRenderer) animLoop() {
	defer o.wg.Done()

	const (
		fadeInStep  = byte(30)
		fadeOutStep = byte(24)
		frameDur    = 14 * time.Millisecond // ~70 fps
	)

	var (
		visible     bool
		curAlpha    byte
		targetAlpha byte
		lastBounds  w32.RECT
		lastW, lastH int32
	)

	ticker := time.NewTicker(frameDur)
	ticker.Stop() // stopped until needed
	tickerRunning := false

	startTicker := func() {
		if !tickerRunning {
			ticker.Reset(frameDur)
			tickerRunning = true
		}
	}
	stopTicker := func() {
		if tickerRunning {
			ticker.Stop()
			tickerRunning = false
		}
	}

	for {
		select {
		case <-o.stopCh:
			stopTicker()
			return

		case cmd := <-o.cmdCh:
			// Drain any stale show commands so we always act on the freshest position.
			for len(o.cmdCh) > 0 {
				cmd = <-o.cmdCh
			}

			if cmd.show {
				newW, newH := cmd.bounds.Width(), cmd.bounds.Height()
				if newW <= 0 || newH <= 0 {
					continue
				}
				needReposition := !rectEquals(cmd.bounds, lastBounds)
				needRegion := newW != lastW || newH != lastH

				if needRegion {
					o.applyRoundedRegion(newW, newH)
					lastW, lastH = newW, newH
				}
				if needReposition {
					o.moveTo(cmd.bounds)
					lastBounds = cmd.bounds
				}
				if !visible {
					// Show with alpha=0; fade will make it appear.
					w32ex.SetLayeredWindowAttributes(o.hwnd, 0, 0, w32ex.LWA_ALPHA)
					procSetWindowPos.Call(
						uintptr(o.hwnd),
						uintptr(w32.HWND_TOPMOST),
						uintptr(cmd.bounds.Left),
						uintptr(cmd.bounds.Top),
						uintptr(newW),
						uintptr(newH),
						uintptr(swpNoActivate|swpShowWindow),
					)
					curAlpha = 0
					visible = true
					lastW, lastH = newW, newH
					lastBounds = cmd.bounds
				}
				targetAlpha = overlayTargetAlpha
				startTicker()

			} else {
				targetAlpha = 0
				if visible {
					startTicker()
				}
			}

		case <-ticker.C:
			if curAlpha == targetAlpha {
				stopTicker()
				if curAlpha == 0 && visible {
					procSetWindowPos.Call(
						uintptr(o.hwnd), 0, 0, 0, 0, 0,
						uintptr(swpNoActivate|swpHideWindow),
					)
					visible = false
				}
				continue
			}

			if curAlpha < targetAlpha {
				step := targetAlpha - curAlpha
				if step > fadeInStep {
					step = fadeInStep
				}
				curAlpha += step
			} else {
				step := curAlpha - targetAlpha
				if step > fadeOutStep {
					step = fadeOutStep
				}
				curAlpha -= step
			}
			w32ex.SetLayeredWindowAttributes(o.hwnd, 0, curAlpha, w32ex.LWA_ALPHA)
		}
	}
}

var procInvalidateRect = overlayUser32.NewProc("InvalidateRect")
var procUpdateWindow    = overlayUser32.NewProc("UpdateWindow")

func (o *OverlayRenderer) moveTo(bounds w32.RECT) {
	procSetWindowPos.Call(
		uintptr(o.hwnd),
		uintptr(w32.HWND_TOPMOST),
		uintptr(bounds.Left),
		uintptr(bounds.Top),
		uintptr(bounds.Width()),
		uintptr(bounds.Height()),
		uintptr(swpNoActivate|swpShowWindow),
	)
	// Force an immediate repaint so the blue colour appears before the alpha
	// fade-in starts — prevents the brief black flash on layered windows.
	procInvalidateRect.Call(uintptr(o.hwnd), 0, 1)
	procUpdateWindow.Call(uintptr(o.hwnd))
}

func (o *OverlayRenderer) applyRoundedRegion(w, h int32) {
	rgn, _, _ := procCreateRoundRectRgn.Call(
		0, 0,
		uintptr(w+1), uintptr(h+1),
		uintptr(overlayCornerRadius),
		uintptr(overlayCornerRadius),
	)
	if rgn != 0 {
		procSetWindowRgn.Call(uintptr(o.hwnd), rgn, 1)
	}
}

func rectEquals(a, b w32.RECT) bool {
	return a.Left == b.Left && a.Top == b.Top && a.Right == b.Right && a.Bottom == b.Bottom
}

func rgb(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}
