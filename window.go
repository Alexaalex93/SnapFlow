package main

import (
	"errors"
	"strings"
	"sync"
	"syscall"
	"time"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	gwlExStyle = -20
	gwlStyle   = -16

	swpNoOwnerZOrder = 0x0200
)

var errAccessDenied = errors.New("access denied (window belongs to an elevated app)")

// ── Frame geometry ────────────────────────────────────────────────────────────
//
// Windows 10/11 windows have an invisible ~7px resize border on the left,
// right and bottom. GetWindowRect includes it; the DWM "extended frame
// bounds" do not. All snap geometry in SnapFlow works on the *visible frame*
// so windows line up edge-to-edge, and the border is added back right before
// SetWindowPos.
//
// SnapFlow is Per-Monitor-V2 DPI aware, so both rects are in physical screen
// pixels and can be subtracted directly. (The previous code rescaled the DWM
// rect by GetDeviceCaps/GetDpiForWindow, which corrupted the position of
// DPI-unaware apps and of windows on a monitor whose scale differs from the
// primary one — one of the "sometimes it doesn't snap right" bugs.)

type insets struct{ l, t, r, b int32 }

// frameRect returns the visible frame of hwnd, falling back to GetWindowRect
// when DWM can't answer (e.g. some legacy or DWM-excluded windows).
func frameRect(hwnd w32.HWND) (w32.RECT, bool) {
	if ok, f := w32.DwmGetWindowAttributeEXTENDED_FRAME_BOUNDS(hwnd); ok && f.Width() > 0 && f.Height() > 0 {
		return f, true
	}
	if r := w32.GetWindowRect(hwnd); r != nil {
		return *r, true
	}
	return w32.RECT{}, false
}

// windowInsets returns the invisible border between GetWindowRect and the
// visible frame. Insets are clamped to sane values so a bogus DWM answer
// can never throw a window off-screen.
func windowInsets(hwnd w32.HWND) (insets, bool) {
	r := w32.GetWindowRect(hwnd)
	if r == nil {
		return insets{}, false
	}
	f, ok := frameRect(hwnd)
	if !ok {
		return insets{}, false
	}
	in := insets{l: f.Left - r.Left, t: f.Top - r.Top, r: r.Right - f.Right, b: r.Bottom - f.Bottom}
	clamp := func(v int32) int32 {
		if v < 0 || v > 64 {
			return 0
		}
		return v
	}
	in.l, in.t, in.r, in.b = clamp(in.l), clamp(in.t), clamp(in.r), clamp(in.b)
	return in, true
}

func setWindowFrame(hwnd w32.HWND, frame w32.RECT, in insets, flags uint) bool {
	x := frame.Left - in.l
	y := frame.Top - in.t
	w := frame.Width() + in.l + in.r
	h := frame.Height() + in.t + in.b
	return w32.SetWindowPos(hwnd, 0, int(x), int(y), int(w), int(h),
		w32.SWP_NOZORDER|w32.SWP_NOACTIVATE|swpNoOwnerZOrder|flags)
}

// extendFlushEdges pushes every edge that sits exactly on the work-area
// border 1px outward, so the 1px accent border Windows 11 draws around
// windows ends up off-screen and the content is truly edge-to-edge.
func extendFlushEdges(target, work w32.RECT) w32.RECT {
	if target.Left == work.Left {
		target.Left--
	}
	if target.Top == work.Top {
		target.Top--
	}
	if target.Right == work.Right {
		target.Right++
	}
	if target.Bottom == work.Bottom {
		target.Bottom++
	}
	return target
}

// ── Placement ─────────────────────────────────────────────────────────────────

// placement generation per window: a delayed verification only acts if no
// newer placement happened in the meantime.
var (
	placeMu  sync.Mutex
	placeGen = map[w32.HWND]uint64{}
)

func bumpPlaceGen(hwnd w32.HWND) uint64 {
	placeMu.Lock()
	defer placeMu.Unlock()
	placeGen[hwnd]++
	if len(placeGen) > 256 {
		for h := range placeGen {
			if !w32.IsWindow(h) {
				delete(placeGen, h)
			}
		}
	}
	return placeGen[hwnd]
}

func currentPlaceGen(hwnd w32.HWND) uint64 {
	placeMu.Lock()
	defer placeMu.Unlock()
	return placeGen[hwnd]
}

// resizeToRect moves hwnd so that its visible frame matches target
// (screen coordinates). Returns the frame actually achieved.
func resizeToRect(hwnd w32.HWND, target w32.RECT) (w32.RECT, error) {
	if !isZonableWindow(hwnd) {
		return w32.RECT{}, nil
	}
	if target.Width() <= 0 || target.Height() <= 0 {
		return w32.RECT{}, nil
	}
	work, ok := monitorWork(w32.MonitorFromRect(&target, w32.MONITOR_DEFAULTTONEAREST))
	if !ok {
		return w32.RECT{}, errors.New("no monitor for target")
	}
	got, err := placeFrame(hwnd, target, work)
	if err != nil {
		if errors.Is(err, errAccessDenied) {
			notifyElevatedOnce()
		}
		return got, err
	}

	// Some apps (Electron, Java, terminals…) re-apply their own bounds right
	// after a move, or receive a late WM_DPICHANGED. Check once shortly
	// afterwards and re-apply if the window drifted.
	gen := bumpPlaceGen(hwnd)
	go func() {
		time.Sleep(150 * time.Millisecond)
		if currentPlaceGen(hwnd) != gen || !w32.IsWindow(hwnd) ||
			w32ex.IsZoomed(hwnd) || w32ex.IsIconic(hwnd) {
			return
		}
		if dragMgr != nil && dragMgr.IsDragging(hwnd) {
			return
		}
		if cur, ok := frameRect(hwnd); ok && !rectNear(cur, got, 2) {
			placeFrame(hwnd, target, work)
		}
	}()
	return got, nil
}

// placeFrame does the actual positioning work:
//  1. un-maximise / un-minimise (border metrics differ in those states),
//  2. position using the current invisible-border insets,
//  3. second pass if the move changed DPI (insets and size change after a
//     cross-monitor move),
//  4. if the app enforces a minimum size bigger than the target, re-anchor
//     the window to the side the target was anchored to and keep it
//     inside the work area instead of letting it hang off-screen.
func placeFrame(hwnd w32.HWND, target, work w32.RECT) (w32.RECT, error) {
	if w32ex.IsIconic(hwnd) || w32ex.IsZoomed(hwnd) {
		w32.ShowWindow(hwnd, w32.SW_RESTORE)
	}
	want := extendFlushEdges(target, work)

	in, ok := windowInsets(hwnd)
	if !ok {
		return w32.RECT{}, errors.New("cannot read window bounds")
	}
	if !setWindowFrame(hwnd, want, in, 0) {
		if syscall.Errno(w32.GetLastError()) == syscall.ERROR_ACCESS_DENIED {
			return w32.RECT{}, errAccessDenied
		}
		return w32.RECT{}, errors.New("SetWindowPos failed")
	}

	got, _ := frameRect(hwnd)
	if !rectNear(got, want, 0) {
		// Second pass: picks up new insets after a DPI change.
		if in2, ok := windowInsets(hwnd); ok {
			setWindowFrame(hwnd, want, in2, 0)
			in = in2
		}
		got, _ = frameRect(hwnd)
	}

	// Minimum-size handling.
	if got.Width() > want.Width()+1 || got.Height() > want.Height()+1 {
		pos := reanchor(got, want, work)
		if pos.Left != got.Left || pos.Top != got.Top {
			setWindowFrame(hwnd, pos, in, w32.SWP_NOSIZE)
			got, _ = frameRect(hwnd)
		}
	}
	return got, nil
}

// reanchor moves an oversized frame `got` so it hugs the same edges as the
// intended rect `want`, staying inside work where possible. Pure geometry.
func reanchor(got, want, work w32.RECT) w32.RECT {
	gw, gh := got.Width(), got.Height()
	left, top := got.Left, got.Top
	if gw > want.Width() {
		switch {
		case want.Right >= work.Right && want.Left > work.Left:
			left = want.Right - gw // right-anchored
		case want.Left <= work.Left:
			left = want.Left
		default:
			left = want.Left + (want.Width()-gw)/2
		}
		if left+gw > work.Right+1 {
			left = work.Right + 1 - gw
		}
		if left < work.Left-1 {
			left = work.Left - 1
		}
	}
	if gh > want.Height() {
		switch {
		case want.Bottom >= work.Bottom && want.Top > work.Top:
			top = want.Bottom - gh
		case want.Top <= work.Top:
			top = want.Top
		default:
			top = want.Top + (want.Height()-gh)/2
		}
		if top+gh > work.Bottom+1 {
			top = work.Bottom + 1 - gh
		}
		if top < work.Top-1 {
			top = work.Top - 1
		}
	}
	return w32.RECT{Left: left, Top: top, Right: left + gw, Bottom: top + gh}
}

func maximize(hwnd w32.HWND) {
	if !isZonableWindow(hwnd) {
		return
	}
	bumpPlaceGen(hwnd) // cancel pending verifications
	w32.ShowWindow(hwnd, w32.SW_MAXIMIZE)
}

// ── Multi-monitor ─────────────────────────────────────────────────────────────

// moveToDisplay moves hwnd to the next or previous monitor.
// Translates Rectangle's NextPrevDisplayCalculation.
func moveToDisplay(hwnd w32.HWND, next bool) {
	if !isZonableWindow(hwnd) {
		return
	}
	monitors := Monitors()
	if len(monitors) <= 1 {
		return
	}
	cur := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	curIdx := -1
	for i, m := range monitors {
		if m == cur {
			curIdx = i
			break
		}
	}
	if curIdx < 0 {
		return
	}
	nextIdx := (curIdx + 1) % len(monitors)
	if !next {
		nextIdx = (curIdx - 1 + len(monitors)) % len(monitors)
	}
	moveToMonitor(hwnd, monitors[curIdx], monitors[nextIdx])
}

func moveToDisplayByIndex(hwnd w32.HWND, idx int) {
	if !isZonableWindow(hwnd) {
		return
	}
	monitors := Monitors()
	if idx < 0 || idx >= len(monitors) {
		return
	}
	cur := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	if cur == monitors[idx] {
		return
	}
	moveToMonitor(hwnd, cur, monitors[idx])
}

// moveToMonitor maps the window's frame proportionally from one work area to
// the other (so a left half stays a left half) and keeps maximised windows
// maximised.
func moveToMonitor(hwnd w32.HWND, from, to w32.HMONITOR) {
	fromWork, ok1 := monitorWork(from)
	toWork, ok2 := monitorWork(to)
	if !ok1 || !ok2 {
		return
	}
	wasMax := w32ex.IsZoomed(hwnd)
	if wasMax || w32ex.IsIconic(hwnd) {
		w32.ShowWindow(hwnd, w32.SW_RESTORE)
	}
	frame, ok := frameRect(hwnd)
	if !ok {
		return
	}
	target := mapRectBetween(frame, fromWork, toWork)
	if _, err := resizeToRect(hwnd, target); err != nil {
		return
	}
	if wasMax {
		w32.ShowWindow(hwnd, w32.SW_MAXIMIZE)
	}
}

// mapRectBetween scales r from one work area into another and clamps it
// inside the destination. Pure geometry (unit-tested).
func mapRectBetween(r, from, to w32.RECT) w32.RECT {
	fw, fh := float64(from.Width()), float64(from.Height())
	tw, th := float64(to.Width()), float64(to.Height())
	if fw <= 0 || fh <= 0 {
		return clampToWork(r, to)
	}
	sx := func(x int32) int32 { return to.Left + int32(float64(x-from.Left)*tw/fw+0.5) }
	sy := func(y int32) int32 { return to.Top + int32(float64(y-from.Top)*th/fh+0.5) }
	out := w32.RECT{Left: sx(r.Left), Top: sy(r.Top), Right: sx(r.Right), Bottom: sy(r.Bottom)}
	return clampToWork(out, to)
}

// ── Small helpers ─────────────────────────────────────────────────────────────

// centerRect centers cur within disp without resizing.
func centerRect(disp, cur w32.RECT) w32.RECT {
	w := (disp.Width() - cur.Width()) / 2
	h := (disp.Height() - cur.Height()) / 2
	return w32.RECT{
		Left:   disp.Left + w,
		Right:  disp.Left + w + cur.Width(),
		Top:    disp.Top + h,
		Bottom: disp.Top + h + cur.Height(),
	}
}

func rectEqual(a, b *w32.RECT) bool {
	return a != nil && b != nil &&
		a.Left == b.Left && a.Top == b.Top && a.Right == b.Right && a.Bottom == b.Bottom
}

// rectNear reports whether every edge of a and b differs by at most tol px.
func rectNear(a, b w32.RECT, tol int32) bool {
	d := func(x, y int32) bool {
		if x > y {
			return x-y <= tol
		}
		return y-x <= tol
	}
	return d(a.Left, b.Left) && d(a.Top, b.Top) && d(a.Right, b.Right) && d(a.Bottom, b.Bottom)
}

// ── Window classification ─────────────────────────────────────────────────────

func isZonableWindow(hwnd w32.HWND) bool {
	return hwnd != 0 && isStandardWindow(hwnd) && hasNoVisibleOwner(hwnd)
}

func hasNoVisibleOwner(hwnd w32.HWND) bool {
	owner := w32.GetWindow(hwnd, w32.GW_OWNER)
	if owner == 0 {
		return true
	}
	if !w32.IsWindowVisible(owner) {
		return true
	}
	rect := w32.GetWindowRect(owner)
	return rect == nil || rect.Width() == 0 || rect.Height() == 0
}

func isStandardWindow(hwnd w32.HWND) bool {
	if w32ex.GetAncestor(hwnd, w32ex.GA_ROOT) != hwnd || !w32.IsWindowVisible(hwnd) {
		return false
	}
	for _, sys := range []w32.HWND{w32.GetDesktopWindow(), w32ex.GetShellWindow()} {
		if hwnd == sys {
			return false
		}
	}
	// Never manage our own windows (overlay, tray, settings).
	if overlay != nil && hwnd == overlay.hwnd {
		return false
	}

	className, ok := w32.GetClassName(hwnd)
	if !ok {
		return false
	}
	if isSystemClassName(className) {
		return false
	}

	style := w32.GetWindowLong(hwnd, gwlStyle)
	exStyle := w32.GetWindowLong(hwnd, gwlExStyle)

	// Always skip child windows and disabled windows.
	if uint32(style)&w32.WS_CHILD == w32.WS_CHILD ||
		style&w32.WS_DISABLED == w32.WS_DISABLED {
		return false
	}

	// Skip genuine floating tool palettes (WS_EX_TOOLWINDOW).
	// Exception: ApplicationFrameWindow is the UWP/Store app host — always allow.
	if exStyle&w32.WS_EX_TOOLWINDOW == w32.WS_EX_TOOLWINDOW &&
		!strings.EqualFold(className, "ApplicationFrameWindow") {
		return false
	}

	// Skip popup windows that have no caption AND no sizing frame —
	// these are menus, tooltips or transient overlays. Popups with a
	// thick frame are custom-chrome apps (Spotify, Discord, VS Code
	// variants…) and must stay snappable.
	hasCaption := style&w32.WS_CAPTION == w32.WS_CAPTION
	hasThickFrame := style&w32.WS_THICKFRAME == w32.WS_THICKFRAME
	isPopup := uint32(style)&w32.WS_POPUP == w32.WS_POPUP
	if isPopup && !hasCaption && !hasThickFrame {
		return false
	}

	return true
}

func isSystemClassName(name string) bool {
	blocked := []string{
		"SysListView32", "WorkerW", "Shell_TrayWnd",
		"Shell_SecondaryTrayWnd", "Progman",
		"Windows.UI.Core.CoreWindow", // UWP inner-content window (not the frame)
		"XamlExplorerHostIslandWindow",
		"PopupHost",
		"TopLevelWindowForOverflowXamlIsland",
		"NotifyIconOverflowWindow",
	}
	for _, c := range blocked {
		if strings.EqualFold(c, name) {
			return true
		}
	}
	return false
}
