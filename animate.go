package main

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// ── Animated window movement ──────────────────────────────────────────────────
//
// Windows doesn't let other apps use the compositor animation its own Snap
// uses (cloaking another process's window is access-denied), and resizing a
// real window 60 times a second makes heavy apps (Chrome, VS Code, Office)
// stutter and flash. Instead SnapFlow animates a *proxy*:
//
//  1. a live DWM thumbnail of the window (GPU-scaled, no app repaint) is
//     shown in a transparent topmost window exactly over the window,
//  2. the real window is parked just off-screen (same size, same monitor, so
//     no DPI change and its content stays valid),
//  3. the thumbnail glides/grows from the old frame to the target frame
//     (ease-out, vsync-paced),
//  4. the real window is placed at the target and the proxy disappears.
//
// Bookkeeping (history, corners) is done synchronously by the caller using
// the expected frame; only the visual part runs on the animator thread.
// A new request for the same window mid-flight retargets smoothly.

const (
	windowAnimDur = 230 * time.Millisecond

	wsExNoRedirectionBitmap = 0x00200000
	proxyClassName          = "SnapFlowWindowProxy"

	dwmTnpRectDestination  = 0x01
	dwmTnpRectSource       = 0x02
	dwmTnpOpacity          = 0x04
	dwmTnpVisible          = 0x08
	dwmTnpSourceClientOnly = 0x10

	spiGetClientAreaAnimation = 0x1042
)

var (
	dwmapiDLL                        = syscall.NewLazyDLL("dwmapi.dll")
	procDwmRegisterThumbnail         = dwmapiDLL.NewProc("DwmRegisterThumbnail")
	procDwmUnregisterThumbnail       = dwmapiDLL.NewProc("DwmUnregisterThumbnail")
	procDwmUpdateThumbnailProperties = dwmapiDLL.NewProc("DwmUpdateThumbnailProperties")
	procGetWindowDisplayAffinity     = user32DLL.NewProc("GetWindowDisplayAffinity")
)

type dwmThumbnailProperties struct {
	flags      uint32
	dst, src   w32.RECT
	opacity    byte
	visible    int32
	clientOnly int32
}

// windowAnim is the process-wide animator (nil if it couldn't start).
var windowAnim *WindowAnimator

type animReq struct {
	hwnd     w32.HWND
	from     w32.RECT // visible frame now
	target   w32.RECT // frame to reach
	expected w32.RECT // what placement will actually produce (flush edges extended)
	off      w32.POINT
}

type WindowAnimator struct {
	threadID uint32
	reqCh    chan animReq
	done     chan struct{}
	proxy    w32.HWND

	mu      sync.Mutex
	targets map[w32.HWND]w32.RECT // logical frame of windows in flight
}

func NewWindowAnimator() (*WindowAnimator, error) {
	a := &WindowAnimator{
		reqCh:   make(chan animReq, 32),
		done:    make(chan struct{}),
		targets: map[w32.HWND]w32.RECT{},
	}
	if procDwmRegisterThumbnail.Find() != nil {
		return nil, errors.New("DWM thumbnails unavailable")
	}
	ready := make(chan error, 1)
	go a.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return a, nil
}

// LogicalFrame returns where hwnd is (or is about to be): its animation
// target while in flight — the real window is parked off-screen then.
func (a *WindowAnimator) LogicalFrame(hwnd w32.HWND) (w32.RECT, bool) {
	if a != nil {
		a.mu.Lock()
		r, ok := a.targets[hwnd]
		a.mu.Unlock()
		if ok {
			return r, true
		}
	}
	return frameRect(hwnd)
}

func (a *WindowAnimator) inFlight(hwnd w32.HWND) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.targets[hwnd]
	return ok
}

// Animate starts (or retargets) an animated move of hwnd's frame to target.
// Returns false when the window can't/shouldn't be animated — the caller
// then places it directly.
func (a *WindowAnimator) Animate(hwnd w32.HWND, from, target, work w32.RECT) bool {
	if a == nil {
		return false
	}
	expected := extendFlushEdges(target, work)
	if a.inFlight(hwnd) {
		a.mu.Lock()
		a.targets[hwnd] = target
		a.mu.Unlock()
		a.send(animReq{hwnd: hwnd, target: target, expected: expected})
		return true
	}
	if rectNear(from, expected, 2) || w32ex.IsIconic(hwnd) || w32ex.IsZoomed(hwnd) {
		return false
	}
	if !systemAnimationsEnabled() || hasDisplayAffinity(hwnd) {
		return false // honour "Animation effects: off"; protected content would show black
	}
	wr := w32.GetWindowRect(hwnd)
	if wr == nil {
		return false
	}
	off, ok := offscreenSpot(*wr, w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST))
	if !ok {
		return false
	}
	a.mu.Lock()
	a.targets[hwnd] = target
	a.mu.Unlock()
	a.send(animReq{hwnd: hwnd, from: from, target: target, expected: expected, off: off})
	return true
}

func (a *WindowAnimator) send(r animReq) {
	a.reqCh <- r // buffered; the animator drains it every frame
	procPostThreadMessageW.Call(uintptr(a.threadID), wmOverlayWake, 0, 0)
}

// Close finishes any animation in flight (the window is placed) and stops.
func (a *WindowAnimator) Close() {
	if a == nil {
		return
	}
	procPostThreadMessageW.Call(uintptr(a.threadID), wmQuit, 0, 0)
	select {
	case <-a.done:
	case <-time.After(time.Second):
	}
}

func systemAnimationsEnabled() bool {
	var on int32 = 1
	w32ex.SystemParametersInfo(spiGetClientAreaAnimation, 0, unsafe.Pointer(&on), 0)
	return on != 0
}

func hasDisplayAffinity(hwnd w32.HWND) bool {
	var aff uint32
	r, _, _ := procGetWindowDisplayAffinity.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&aff)))
	return r != 0 && aff != 0
}

// offscreenSpot finds a position for a window of rect wr that is on no
// monitor but whose nearest monitor is still mon (so parking it there never
// triggers a DPI change). Pure Win32 geometry queries.
func offscreenSpot(wr w32.RECT, mon w32.HMONITOR) (w32.POINT, bool) {
	var info w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &info) {
		return w32.POINT{}, false
	}
	m := info.RcMonitor
	w, h := wr.Width(), wr.Height()
	const gap = 64
	cands := []w32.POINT{
		{X: m.Right + gap, Y: m.Top},
		{X: m.Left - w - gap, Y: m.Top},
		{X: m.Left, Y: m.Bottom + gap},
		{X: m.Left, Y: m.Top - h - gap},
	}
	for _, p := range cands {
		r := w32.RECT{Left: p.X, Top: p.Y, Right: p.X + w, Bottom: p.Y + h}
		if w32.MonitorFromRect(&r, w32ex.MONITOR_DEFAULTTONULL) == 0 &&
			w32.MonitorFromRect(&r, w32.MONITOR_DEFAULTTONEAREST) == mon {
			return p, true
		}
	}
	return w32.POINT{}, false
}

// ── Animator thread ───────────────────────────────────────────────────────────

type flight struct {
	animReq
	start  time.Time
	cur    w32.RECT // currently displayed rect
	thumb  uintptr
	origin w32.POINT // proxy window origin
	src    w32.RECT  // thumbnail source rect (frame inside the window rect)
}

func (a *WindowAnimator) run(ready chan<- error) {
	runtime.LockOSThread()
	defer close(a.done)
	a.threadID = w32ex.GetCurrentThreadID()
	proxy, err := createProxyWindow()
	if err != nil {
		ready <- err
		return
	}
	a.proxy = proxy
	ready <- nil
	defer procDestroyWindow.Call(uintptr(a.proxy))

	var f *flight
	var m w32.MSG
	quit := false
	for !quit {
		if f == nil {
			r := w32.GetMessage(&m, 0, 0, 0)
			if r == 0 || r == -1 {
				return
			}
			if !(m.Hwnd == 0 && m.Message == wmOverlayWake) {
				w32.TranslateMessage(&m)
				w32.DispatchMessage(&m)
			}
		} else {
			for peekMessage(&m) {
				if m.Message == wmQuit {
					quit = true
					break
				}
				if !(m.Hwnd == 0 && m.Message == wmOverlayWake) {
					w32.TranslateMessage(&m)
					w32.DispatchMessage(&m)
				}
			}
		}

		// Apply requests.
	drain:
		for {
			select {
			case r := <-a.reqCh:
				switch {
				case f != nil && r.hwnd == f.hwnd:
					a.retarget(f, r)
				default:
					if f != nil {
						a.finish(f)
						f = nil
					}
					if r.from.Width() > 0 {
						f = a.begin(r)
					} else {
						// Retarget for a flight that already landed: just place it.
						a.placeDirect(r)
					}
				}
			default:
				break drain
			}
		}

		if f != nil {
			if quit || a.step(f) {
				a.finish(f)
				f = nil
			} else if w32.DwmFlush() != 0 {
				time.Sleep(8 * time.Millisecond)
			}
		}
	}
}

func createProxyWindow() (w32.HWND, error) {
	instance, _, _ := procGetModuleHandleW.Call(0)
	name := syscall.StringToUTF16Ptr(proxyClassName)
	wc := wndClassExW{lpfnWndProc: procDefWindowProcW.Addr(), hInstance: instance, lpszClassName: name}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 && e != syscall.Errno(1410) {
		return 0, errors.New("RegisterClassExW: " + e.Error())
	}
	// No redirection surface: the window itself is fully transparent and
	// only the DWM thumbnail is visible.
	ex := uintptr(wsExTopmost | wsExToolWindow | wsExNoActivate | wsExNoRedirectionBitmap)
	r, _, e := procCreateWindowExW.Call(ex, uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SnapFlow Motion"))),
		uintptr(wsPopup), 0, 0, 1, 1, 0, 0, instance, 0)
	if r == 0 {
		return 0, errors.New("CreateWindowExW: " + e.Error())
	}
	return w32.HWND(r), nil
}

func unionRect(a, b w32.RECT) w32.RECT {
	u := a
	if b.Left < u.Left {
		u.Left = b.Left
	}
	if b.Top < u.Top {
		u.Top = b.Top
	}
	if b.Right > u.Right {
		u.Right = b.Right
	}
	if b.Bottom > u.Bottom {
		u.Bottom = b.Bottom
	}
	return u
}

func offsetRect(r w32.RECT, o w32.POINT) w32.RECT {
	return w32.RECT{Left: r.Left - o.X, Top: r.Top - o.Y, Right: r.Right - o.X, Bottom: r.Bottom - o.Y}
}

func (a *WindowAnimator) coverProxy(f *flight, from, to w32.RECT) {
	u := unionRect(from, to)
	f.origin = w32.POINT{X: u.Left, Y: u.Top}
	w32.SetWindowPos(a.proxy, w32.HWND_TOPMOST, int(u.Left), int(u.Top), int(u.Width()), int(u.Height()),
		swpNoActivate|w32.SWP_SHOWWINDOW)
}

func (a *WindowAnimator) updateThumb(f *flight, dst w32.RECT) {
	p := dwmThumbnailProperties{
		flags:   dwmTnpRectDestination | dwmTnpRectSource | dwmTnpOpacity | dwmTnpVisible | dwmTnpSourceClientOnly,
		dst:     offsetRect(dst, f.origin),
		src:     f.src,
		opacity: 255,
		visible: 1,
	}
	procDwmUpdateThumbnailProperties.Call(f.thumb, uintptr(unsafe.Pointer(&p)))
}

// begin covers the window with its thumbnail, parks the real window
// off-screen and starts the clock. Returns nil if animation isn't possible
// (the window is then placed directly).
func (a *WindowAnimator) begin(r animReq) *flight {
	f := &flight{animReq: r, cur: r.from}
	wr := w32.GetWindowRect(r.hwnd)
	if wr == nil || !w32.IsWindow(r.hwnd) {
		a.placeDirect(r)
		return nil
	}
	f.src = offsetRect(r.from, w32.POINT{X: wr.Left, Y: wr.Top})

	a.coverProxy(f, r.from, r.expected)
	hr, _, _ := procDwmRegisterThumbnail.Call(uintptr(a.proxy), uintptr(r.hwnd), uintptr(unsafe.Pointer(&f.thumb)))
	if hr != 0 {
		w32.ShowWindow(a.proxy, swHide)
		a.placeDirect(r)
		return nil
	}
	a.updateThumb(f, r.from)
	// Make sure the thumbnail is on screen before the window moves away,
	// so there's never a frame with neither visible.
	w32.DwmFlush()
	w32.DwmFlush()

	if !w32.SetWindowPos(r.hwnd, 0, int(r.off.X), int(r.off.Y), 0, 0,
		w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE|swpNoOwnerZOrder) {
		a.teardown(f)
		a.placeDirect(r)
		return nil
	}
	f.start = time.Now()
	return f
}

// retarget continues a flight towards a new target from where it is now.
func (a *WindowAnimator) retarget(f *flight, r animReq) {
	f.from = f.cur
	f.target, f.expected = r.target, r.expected
	f.start = time.Now()
	a.coverProxy(f, f.from, f.expected)
	a.updateThumb(f, f.cur)
}

// step advances the animation; true when it reached the end.
func (a *WindowAnimator) step(f *flight) bool {
	if !w32.IsWindow(f.hwnd) {
		return true
	}
	t := progress(f.start, windowAnimDur, time.Now())
	e := easeOutCubic(t)
	ff, ft := toFrect(f.from), toFrect(f.expected)
	f.cur = lerpRect(ff, ft, e).round()
	a.updateThumb(f, f.cur)
	return t >= 1
}

// finish places the real window at the target and removes the proxy.
func (a *WindowAnimator) finish(f *flight) {
	if w32.IsWindow(f.hwnd) {
		got, err := resizeToRect(f.hwnd, f.target)
		if err != nil || got.Width() <= 0 {
			// Never leave a window parked off-screen.
			if in, ok := windowInsets(f.hwnd); ok {
				setWindowFrame(f.hwnd, f.from, in, 0)
			}
		} else {
			if !rectNear(got, f.expected, 0) {
				windowHistory.UpdateSnapped(f.hwnd, f.expected, got)
			}
			// Point the thumbnail at the window's final frame so the hand-over
			// to the real window is pixel-identical, then let DWM show it.
			if wr := w32.GetWindowRect(f.hwnd); wr != nil {
				f.src = offsetRect(got, w32.POINT{X: wr.Left, Y: wr.Top})
				a.updateThumb(f, got)
			}
			w32.DwmFlush()
		}
	}
	a.teardown(f)
	a.mu.Lock()
	if t, ok := a.targets[f.hwnd]; ok && t == f.target {
		delete(a.targets, f.hwnd)
	}
	a.mu.Unlock()
}

func (a *WindowAnimator) teardown(f *flight) {
	if f.thumb != 0 {
		procDwmUnregisterThumbnail.Call(f.thumb)
		f.thumb = 0
	}
	w32.ShowWindow(a.proxy, swHide)
}

func (a *WindowAnimator) placeDirect(r animReq) {
	got, err := resizeToRect(r.hwnd, r.target)
	if err == nil && got.Width() > 0 && !rectNear(got, r.expected, 0) {
		windowHistory.UpdateSnapped(r.hwnd, r.expected, got)
	}
	a.mu.Lock()
	delete(a.targets, r.hwnd)
	a.mu.Unlock()
}

// ── Entry point used by actions and drags ─────────────────────────────────────

// placeWindow moves hwnd's visible frame from `from` to `target`, animated
// when enabled and possible. It returns the frame the window ends up in
// (for animations: the expected frame — the animator corrects history if
// an app's minimum size changes it). Main thread only (reads appCfg).
func placeWindow(hwnd w32.HWND, from, target w32.RECT) (w32.RECT, error) {
	if appCfg.AnimateWindows && windowAnim != nil {
		if work, ok := monitorWork(w32.MonitorFromRect(&target, w32.MONITOR_DEFAULTTONEAREST)); ok &&
			isZonableWindow(hwnd) && windowAnim.Animate(hwnd, from, target, work) {
			return extendFlushEdges(target, work), nil
		}
	}
	return resizeToRect(hwnd, target)
}

// currentFrame is frameRect, but aware of windows that are mid-animation.
func currentFrame(hwnd w32.HWND) (w32.RECT, bool) {
	if windowAnim != nil {
		return windowAnim.LogicalFrame(hwnd)
	}
	return frameRect(hwnd)
}
