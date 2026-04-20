package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ahmetb/RectangleWin/w32ex"
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
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080
)

var (
	overlayUser32          = syscall.NewLazyDLL("user32.dll")
	overlayKernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateWindowExW    = overlayUser32.NewProc("CreateWindowExW")
	procDestroyWindow      = overlayUser32.NewProc("DestroyWindow")
	procSetWindowPos       = overlayUser32.NewProc("SetWindowPos")
	procGetModuleHandleW   = overlayKernel32.NewProc("GetModuleHandleW")
)

type OverlayRenderer struct {
	hwnd    w32.HWND
	visible bool
}

func NewOverlayRenderer() (*OverlayRenderer, error) {
	instance, _, _ := procGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr("STATIC")
	windowName := syscall.StringToUTF16Ptr(ProductName + " Preview")

	exStyle := uintptr(wsExTopmost | wsExTransparent | wsExToolWindow | wsExLayered | wsExNoActivate)
	style := uintptr(wsPopup)
	r1, _, e1 := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		style,
		0,
		0,
		1,
		1,
		0,
		0,
		instance,
		0,
	)
	if r1 == 0 {
		return nil, fmt.Errorf("create overlay window: %v", e1)
	}
	hwnd := w32.HWND(r1)
	if !w32ex.SetLayeredWindowAttributes(hwnd, 0, 76, w32ex.LWA_ALPHA) {
		return nil, fmt.Errorf("set layered alpha failed: %d", w32.GetLastError())
	}
	return &OverlayRenderer{hwnd: hwnd}, nil
}

func (o *OverlayRenderer) Show(bounds w32.RECT) error {
	if o == nil || o.hwnd == 0 {
		return fmt.Errorf("overlay not initialized")
	}
	w := bounds.Width()
	h := bounds.Height()
	if w <= 0 || h <= 0 {
		return nil
	}
	r1, _, e1 := procSetWindowPos.Call(
		uintptr(o.hwnd),
		uintptr(w32.HWND_TOPMOST),
		uintptr(int32(bounds.Left)),
		uintptr(int32(bounds.Top)),
		uintptr(int32(w)),
		uintptr(int32(h)),
		uintptr(swpNoActivate|swpShowWindow),
	)
	if r1 == 0 {
		return fmt.Errorf("show overlay failed: %v", e1)
	}
	o.visible = true
	return nil
}

func (o *OverlayRenderer) Hide() error {
	if o == nil || o.hwnd == 0 || !o.visible {
		return nil
	}
	r1, _, e1 := procSetWindowPos.Call(
		uintptr(o.hwnd),
		0,
		0,
		0,
		0,
		0,
		uintptr(swpNoActivate|swpHideWindow),
	)
	if r1 == 0 {
		return fmt.Errorf("hide overlay failed: %v", e1)
	}
	o.visible = false
	return nil
}

func (o *OverlayRenderer) Close() {
	if o == nil || o.hwnd == 0 {
		return
	}
	_, _, _ = procDestroyWindow.Call(uintptr(o.hwnd))
	o.hwnd = 0
	o.visible = false
}
