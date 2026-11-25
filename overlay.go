package main

import (
	"errors"
	"math"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows/registry"
)

// ── Snap preview overlay ──────────────────────────────────────────────────────
//
// The overlay is a per-pixel-alpha layered window (UpdateLayeredWindow) drawn
// as an anti-aliased rounded "card": translucent accent fill + crisp border.
//
// It lives on its own locked OS thread with its own message pump, so the
// animation never stutters while the main thread is busy (WinEvent
// callbacks, dialogs, the settings window…), and no call ever crosses
// threads. Frames are paced with DwmFlush(), i.e. synchronised with the
// compositor's refresh, so motion is as smooth as the display allows.
//
// Animations:
//   - appear:   fade-in + slight zoom (94% → 100%)
//   - retarget: the card glides from the previous zone to the new one
//   - hide:     fade-out (on drop it fades out over the snapped window)

const (
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExTopmost     = 0x00000008
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000
	wsPopup         = 0x80000000

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010

	swHide           = 0
	swShowNoActivate = 4

	ulwAlpha = 0x2
	pmRemove = 0x0001
	wmQuit   = 0x0012

	wmOverlayWake = w32.WM_APP + 50

	overlayMoveDur    = 170 * time.Millisecond
	overlayFadeInDur  = 120 * time.Millisecond
	overlayFadeOutDur = 180 * time.Millisecond
	overlayAppearZoom = 0.94

	overlayClassName = "SnapFlowOverlay"
)

type overlayCmdKind int

const (
	cmdShow overlayCmdKind = iota
	cmdHide
)

type overlayCmd struct {
	kind   overlayCmdKind
	bounds w32.RECT
}

// OverlayRenderer displays the animated snap-preview rectangle.
// Show/Hide/Configure are safe to call from any goroutine and never block.
type OverlayRenderer struct {
	hwnd     w32.HWND
	threadID uint32
	cmdCh    chan overlayCmd
	done     chan struct{}

	animate   atomic.Bool
	fillAlpha atomic.Uint32 // math.Float32bits
}

func NewOverlayRenderer() (*OverlayRenderer, error) {
	o := &OverlayRenderer{
		cmdCh: make(chan overlayCmd, 64),
		done:  make(chan struct{}),
	}
	o.animate.Store(true)
	o.fillAlpha.Store(math.Float32bits(0.30))

	ready := make(chan error, 1)
	go o.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return o, nil
}

// Configure applies user preferences (animation on/off, fill opacity).
func (o *OverlayRenderer) Configure(animate bool, fillAlpha float64) {
	if fillAlpha < 0.08 || fillAlpha > 0.95 {
		fillAlpha = 0.30
	}
	o.animate.Store(animate)
	o.fillAlpha.Store(math.Float32bits(float32(fillAlpha)))
}

// Show moves/morphs the preview to bounds (screen coordinates).
func (o *OverlayRenderer) Show(bounds w32.RECT) { o.send(overlayCmd{kind: cmdShow, bounds: bounds}) }

// Hide fades the preview out.
func (o *OverlayRenderer) Hide() { o.send(overlayCmd{kind: cmdHide}) }

func (o *OverlayRenderer) send(c overlayCmd) {
	select {
	case o.cmdCh <- c:
	default:
	}
	procPostThreadMessageW.Call(uintptr(o.threadID), wmOverlayWake, 0, 0)
}

// Close stops the overlay thread and destroys the window.
func (o *OverlayRenderer) Close() {
	procPostThreadMessageW.Call(uintptr(o.threadID), wmQuit, 0, 0)
	select {
	case <-o.done:
	case <-time.After(500 * time.Millisecond):
	}
}

// ── Overlay thread ────────────────────────────────────────────────────────────

func (o *OverlayRenderer) run(ready chan<- error) {
	runtime.LockOSThread() // window + animation stay on this thread forever
	defer close(o.done)

	o.threadID = w32ex.GetCurrentThreadID()
	hwnd, err := createOverlayWindow()
	if err != nil {
		ready <- err
		return
	}
	o.hwnd = hwnd
	ready <- nil

	s := &overlaySurface{dc: w32.CreateCompatibleDC(0)}
	defer s.free()
	defer procDestroyWindow.Call(uintptr(o.hwnd))

	var a overlayAnim
	var m w32.MSG
	for {
		if !a.animating() {
			// Idle: sleep in GetMessage until a command wakes us.
			r := w32.GetMessage(&m, 0, 0, 0)
			if r == 0 || r == -1 {
				return
			}
			o.dispatch(&m)
		} else {
			for peekMessage(&m) {
				if m.Message == wmQuit {
					return
				}
				o.dispatch(&m)
			}
		}

		o.drainCommands(&a)

		if a.animating() {
			o.step(&a, s)
			if a.animating() {
				// Wait for the next compositor frame (vsync-paced).
				if w32.DwmFlush() != 0 {
					time.Sleep(8 * time.Millisecond)
				}
			}
		}
	}
}

func (o *OverlayRenderer) dispatch(m *w32.MSG) {
	if m.Hwnd == 0 && m.Message == wmOverlayWake {
		return
	}
	w32.TranslateMessage(m)
	w32.DispatchMessage(m)
}

func peekMessage(m *w32.MSG) bool {
	r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(m)), 0, 0, 0, pmRemove)
	return r != 0
}

func (o *OverlayRenderer) drainCommands(a *overlayAnim) {
	for {
		select {
		case c := <-o.cmdCh:
			o.apply(a, c)
		default:
			return
		}
	}
}

func createOverlayWindow() (w32.HWND, error) {
	instance, _, _ := procGetModuleHandleW.Call(0)
	name := syscall.StringToUTF16Ptr(overlayClassName)
	if err := procDefWindowProcW.Find(); err != nil {
		return 0, err
	}
	wc := wndClassExW{
		lpfnWndProc:   procDefWindowProcW.Addr(), // nothing to handle: ULW windows need no WM_PAINT
		hInstance:     instance,
		lpszClassName: name,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 && e != syscall.Errno(1410) {
		return 0, errors.New("RegisterClassExW: " + e.Error())
	}
	exStyle := uintptr(wsExTopmost | wsExTransparent | wsExToolWindow | wsExLayered | wsExNoActivate)
	r, _, e := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SnapFlow Preview"))),
		uintptr(wsPopup),
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if r == 0 {
		return 0, errors.New("CreateWindowExW: " + e.Error())
	}
	return w32.HWND(r), nil
}

// ── Animation state machine ───────────────────────────────────────────────────

type frect struct{ l, t, r, b float64 }

func toFrect(r w32.RECT) frect {
	return frect{float64(r.Left), float64(r.Top), float64(r.Right), float64(r.Bottom)}
}

func (f frect) round() w32.RECT {
	return w32.RECT{
		Left: int32(math.Round(f.l)), Top: int32(math.Round(f.t)),
		Right: int32(math.Round(f.r)), Bottom: int32(math.Round(f.b)),
	}
}

func (f frect) scaled(k float64) frect {
	cx, cy := (f.l+f.r)/2, (f.t+f.b)/2
	hw, hh := (f.r-f.l)/2*k, (f.b-f.t)/2*k
	return frect{cx - hw, cy - hh, cx + hw, cy + hh}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func lerpRect(a, b frect, t float64) frect {
	return frect{lerp(a.l, b.l, t), lerp(a.t, b.t, t), lerp(a.r, b.r, t), lerp(a.b, b.b, t)}
}

// easeOutCubic: fast start, gentle landing — feels responsive yet smooth.
func easeOutCubic(t float64) float64 {
	u := 1 - t
	return 1 - u*u*u
}

type overlayAnim struct {
	visible bool

	cur, from, to frect
	moving        bool
	moveStart     time.Time
	moveDur       time.Duration

	alpha, alphaFrom, alphaTo float64
	fading                    bool
	fadeStart                 time.Time
	fadeDur                   time.Duration

	color [3]float32 // accent colour (0..1) captured when the card appears
}

func (a *overlayAnim) animating() bool { return a.moving || a.fading }

func (o *OverlayRenderer) dur(d time.Duration) time.Duration {
	if !o.animate.Load() {
		return 0
	}
	return d
}

func (o *OverlayRenderer) apply(a *overlayAnim, c overlayCmd) {
	now := time.Now()
	switch c.kind {
	case cmdShow:
		if c.bounds.Width() <= 0 || c.bounds.Height() <= 0 {
			return
		}
		b := toFrect(c.bounds)
		appearing := !a.visible || (a.alphaTo == 0 && a.alpha < 0.15)
		if appearing {
			a.color = accentColor()
			a.from = b.scaled(overlayAppearZoom)
			a.cur = a.from
			a.to = b
			a.moving, a.moveStart, a.moveDur = true, now, o.dur(overlayMoveDur)
		} else if b != a.to {
			a.from, a.to = a.cur, b
			a.moving, a.moveStart, a.moveDur = true, now, o.dur(overlayMoveDur)
		}
		if a.alphaTo != 1 {
			a.alphaFrom, a.alphaTo = a.alpha, 1
			a.fading, a.fadeStart, a.fadeDur = true, now, o.dur(overlayFadeInDur)
		}
	case cmdHide:
		if !a.visible && !a.animating() {
			return
		}
		if a.alphaTo != 0 {
			a.alphaFrom, a.alphaTo = a.alpha, 0
			a.fading, a.fadeStart, a.fadeDur = true, now, o.dur(overlayFadeOutDur)
		}
	}
}

func progress(start time.Time, d time.Duration, now time.Time) float64 {
	if d <= 0 {
		return 1
	}
	t := float64(now.Sub(start)) / float64(d)
	if t >= 1 {
		return 1
	}
	if t < 0 {
		return 0
	}
	return t
}

func (o *OverlayRenderer) step(a *overlayAnim, s *overlaySurface) {
	now := time.Now()
	if a.moving {
		t := progress(a.moveStart, a.moveDur, now)
		a.cur = lerpRect(a.from, a.to, easeOutCubic(t))
		if t >= 1 {
			a.cur, a.moving = a.to, false
		}
	}
	if a.fading {
		t := progress(a.fadeStart, a.fadeDur, now)
		a.alpha = lerp(a.alphaFrom, a.alphaTo, easeOutCubic(t))
		if t >= 1 {
			a.alpha, a.fading = a.alphaTo, false
		}
	}

	if a.alphaTo == 0 && !a.fading {
		// Fully faded out → hide and park.
		if a.visible {
			w32.ShowWindow(o.hwnd, swHide)
			a.visible = false
		}
		a.moving = false
		return
	}

	r := a.cur.round()
	w, h := r.Width(), r.Height()
	if w <= 0 || h <= 0 {
		return
	}
	scale := float32(w32ex.GetDpiForMonitor(w32.MonitorFromRect(&r, w32.MONITOR_DEFAULTTONEAREST))) / 96
	st := overlayStyle(scale, a.color, math.Float32frombits(o.fillAlpha.Load()))
	if !s.render(w, h, st) {
		return
	}
	s.present(o.hwnd, r.Left, r.Top, byte(math.Round(a.alpha*255)))

	if !a.visible {
		w32.ShowWindow(o.hwnd, swShowNoActivate)
		w32.SetWindowPos(o.hwnd, w32.HWND_TOPMOST, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		a.visible = true
	}
}

// ── Rendering ─────────────────────────────────────────────────────────────────

type roundRectStyle struct {
	radius, border, inset float32
	fill, edge            [4]float32 // r, g, b, a in 0..1 (straight alpha)
}

func overlayStyle(scale float32, c [3]float32, fillAlpha float32) roundRectStyle {
	if scale <= 0 {
		scale = 1
	}
	// Border is the accent lightened a touch so it reads on dark wallpapers too.
	light := func(v float32) float32 { return v + (1-v)*0.25 }
	return roundRectStyle{
		radius: 10 * scale,
		border: float32(math.Max(1, math.Round(float64(2*scale)))),
		inset:  float32(math.Round(float64(3 * scale))),
		fill:   [4]float32{c[0], c[1], c[2], fillAlpha},
		edge:   [4]float32{light(c[0]), light(c[1]), light(c[2]), 0.92},
	}
}

// renderRoundRect paints an anti-aliased rounded rectangle (fill + border)
// into a premultiplied BGRA buffer. Only the left/right bands containing the
// corners are evaluated per pixel; every row's middle is a single colour and
// is filled with memmove-speed copies, so even 4K previews render in ~1ms.
func renderRoundRect(buf []byte, stride, w, h int, st roundRectStyle) {
	cx, cy := float32(w)/2, float32(h)/2
	ohx, ohy := cx-st.inset, cy-st.inset
	if ohx <= 0 || ohy <= 0 {
		for y := 0; y < h; y++ {
			row := buf[y*stride : y*stride+w*4]
			for i := range row {
				row[i] = 0
			}
		}
		return
	}
	or := minf(st.radius, minf(ohx, ohy))
	bw := st.border
	ihx, ihy := ohx-bw, ohy-bw
	ir := or - bw
	if ir < 0 {
		ir = 0
	}
	hasInner := ihx > 0 && ihy > 0

	pix := func(px, py float32) [4]byte {
		oc := roundRectCoverage(px, py, ohx, ohy, or)
		var ic float32
		if hasInner {
			ic = roundRectCoverage(px, py, ihx, ihy, ir)
		}
		fa := ic * st.fill[3]
		ea := (oc - ic) * st.edge[3]
		if ea < 0 {
			ea = 0
		}
		to := func(v float32) byte {
			v = v*255 + 0.5
			if v > 255 {
				return 255
			}
			if v < 0 {
				return 0
			}
			return byte(v)
		}
		return [4]byte{
			to(fa*st.fill[2] + ea*st.edge[2]), // B
			to(fa*st.fill[1] + ea*st.edge[1]), // G
			to(fa*st.fill[0] + ea*st.edge[0]), // R
			to(fa + ea),                       // A
		}
	}

	band := int(st.inset+or) + 2
	if band*2 > w {
		band = (w + 1) / 2
	}
	for y := 0; y < h; y++ {
		row := buf[y*stride : y*stride+w*4]
		py := float32(y) + 0.5 - cy
		for x := 0; x < band; x++ {
			p := pix(float32(x)+0.5-cx, py)
			copy(row[x*4:], p[:])
		}
		for x := w - band; x < w; x++ {
			if x < band {
				continue
			}
			p := pix(float32(x)+0.5-cx, py)
			copy(row[x*4:], p[:])
		}
		if w-band > band {
			mid := row[band*4 : (w-band)*4]
			p := pix(0, py)
			copy(mid, p[:])
			for n := 4; n < len(mid); n *= 2 {
				copy(mid[n:], mid[:n])
			}
		}
	}
}

// roundRectCoverage returns 0..1 coverage of the pixel centred at (px,py)
// (relative to the rect centre) by a rounded box of half-size (hx,hy) and
// corner radius r, using its signed distance field.
func roundRectCoverage(px, py, hx, hy, r float32) float32 {
	qx := absf(px) - hx + r
	qy := absf(py) - hy + r
	ox, oy := maxf(qx, 0), maxf(qy, 0)
	d := float32(math.Sqrt(float64(ox*ox+oy*oy))) + minf(maxf(qx, qy), 0) - r
	c := 0.5 - d
	if c < 0 {
		return 0
	}
	if c > 1 {
		return 1
	}
	return c
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// overlaySurface is a reusable top-down 32bpp DIB selected into a memory DC.
// It only grows, so animating between sizes never reallocates per frame.
type overlaySurface struct {
	dc         w32.HDC
	bmp        w32.HBITMAP
	old        w32.HGDIOBJ
	bits       unsafe.Pointer
	capW, capH int32

	w, h  int32
	style roundRectStyle
	valid bool
}

func (s *overlaySurface) ensure(w, h int32) bool {
	if s.dc == 0 {
		return false
	}
	if s.bmp != 0 && w <= s.capW && h <= s.capH {
		return true
	}
	nw, nh := w, h
	if s.capW > nw {
		nw = s.capW
	}
	if s.capH > nh {
		nh = s.capH
	}
	var bmi w32.BITMAPINFO
	bmi.BmiHeader = w32.BITMAPINFOHEADER{
		BiSize:     uint32(unsafe.Sizeof(bmi.BmiHeader)),
		BiWidth:    nw,
		BiHeight:   -nh, // top-down
		BiPlanes:   1,
		BiBitCount: 32,
	}
	var bits unsafe.Pointer
	bmp := w32.CreateDIBSection(s.dc, &bmi, 0, &bits, 0, 0)
	if bmp == 0 || bits == nil {
		return false
	}
	prev := w32.SelectObject(s.dc, w32.HGDIOBJ(bmp))
	if s.bmp != 0 {
		procDeleteObject.Call(uintptr(s.bmp))
	} else {
		s.old = prev
	}
	s.bmp, s.bits, s.capW, s.capH = bmp, bits, nw, nh
	s.valid = false
	return true
}

func (s *overlaySurface) render(w, h int32, st roundRectStyle) bool {
	if !s.ensure(w, h) {
		return false
	}
	if s.valid && s.w == w && s.h == h && s.style == st {
		return true
	}
	buf := unsafe.Slice((*byte)(s.bits), int(s.capW)*int(s.capH)*4)
	renderRoundRect(buf, int(s.capW)*4, int(w), int(h), st)
	s.w, s.h, s.style, s.valid = w, h, st, true
	return true
}

func (s *overlaySurface) present(hwnd w32.HWND, x, y int32, alpha byte) {
	dst := w32.POINT{X: x, Y: y}
	size := struct{ cx, cy int32 }{s.w, s.h}
	src := w32.POINT{}
	// BLENDFUNCTION{AC_SRC_OVER, 0, alpha, AC_SRC_ALPHA}
	blend := uint32(alpha)<<16 | 1<<24
	procUpdateLayeredWindow.Call(uintptr(hwnd), 0,
		uintptr(unsafe.Pointer(&dst)), uintptr(unsafe.Pointer(&size)),
		uintptr(s.dc), uintptr(unsafe.Pointer(&src)), 0,
		uintptr(unsafe.Pointer(&blend)), ulwAlpha)
}

func (s *overlaySurface) free() {
	if s.dc == 0 {
		return
	}
	if s.bmp != 0 {
		w32.SelectObject(s.dc, s.old)
		procDeleteObject.Call(uintptr(s.bmp))
	}
	w32.DeleteDC(s.dc)
	s.dc, s.bmp = 0, 0
}

// ── Accent colour ─────────────────────────────────────────────────────────────

// accentColor returns the user's Windows accent colour (Settings →
// Personalisation → Colours), falling back to SnapFlow blue.
func accentColor() [3]float32 {
	fallback := [3]float32{37.0 / 255, 99.0 / 255, 235.0 / 255}
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\DWM`, registry.QUERY_VALUE)
	if err != nil {
		return fallback
	}
	defer k.Close()
	if v, _, err := k.GetIntegerValue("AccentColor"); err == nil { // 0xAABBGGRR
		return [3]float32{float32(v&0xFF) / 255, float32(v>>8&0xFF) / 255, float32(v>>16&0xFF) / 255}
	}
	if v, _, err := k.GetIntegerValue("ColorizationColor"); err == nil { // 0xAARRGGBB
		return [3]float32{float32(v>>16&0xFF) / 255, float32(v>>8&0xFF) / 255, float32(v&0xFF) / 255}
	}
	return fallback
}
