package main

import (
	"fmt"
	"strings"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	gwlExStyle = -20
	gwlStyle   = -16
)

type resizeFunc func(disp, cur w32.RECT) w32.RECT

// applyResize moves hwnd by calling f(workArea, currentFrame) → newFrame.
// It accounts for Windows 10/11 invisible border offsets and DPI scaling.
func applyResize(hwnd w32.HWND, f resizeFunc) (bool, error) {
	if !isZonableWindow(hwnd) {
		return false, nil
	}
	rect := w32.GetWindowRect(hwnd)
	mon := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	hdc := w32.GetDC(hwnd)
	displayDPI := w32.GetDeviceCaps(hdc, w32.LOGPIXELSY)
	w32.ReleaseDC(hwnd, hdc)

	var monInfo w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &monInfo) {
		return false, fmt.Errorf("GetMonitorInfo: %d", w32.GetLastError())
	}
	ok, frame := w32.DwmGetWindowAttributeEXTENDED_FRAME_BOUNDS(hwnd)
	if !ok {
		return false, fmt.Errorf("DwmGetWindowAttribute: %d", w32.GetLastError())
	}
	windowDPI := w32ex.GetDpiForWindow(hwnd)
	adjusted := scaleRect(frame, int32(windowDPI), int32(displayDPI))

	// Invisible border compensation (Windows 10/11 shadow frames).
	lExtra := adjusted.Left - rect.Left
	rExtra := -adjusted.Right + rect.Right
	tExtra := adjusted.Top - rect.Top
	bExtra := -adjusted.Bottom + rect.Bottom

	newPos := f(monInfo.RcWork, adjusted)
	newPos.Left -= lExtra
	newPos.Top -= tExtra
	newPos.Right += rExtra
	newPos.Bottom += bExtra

	if rectEqual(rect, &newPos) {
		return false, nil
	}
	w32.ShowWindow(hwnd, w32.SW_SHOWNORMAL)
	if !w32.SetWindowPos(hwnd, 0, int(newPos.Left), int(newPos.Top), int(newPos.Width()), int(newPos.Height()), w32.SWP_NOZORDER|w32.SWP_NOACTIVATE) {
		return false, fmt.Errorf("SetWindowPos: %d", w32.GetLastError())
	}
	return true, nil
}

// resizeToRect moves hwnd to target (in work-area coordinates).
func resizeToRect(hwnd w32.HWND, target w32.RECT) (bool, error) {
	return applyResize(hwnd, func(_, _ w32.RECT) w32.RECT { return target })
}

func maximize(hwnd w32.HWND) error {
	if !isZonableWindow(hwnd) {
		return nil
	}
	w32.ShowWindow(hwnd, w32.SW_MAXIMIZE)
	return nil
}

func toggleAlwaysOnTop(hwnd w32.HWND) {
	if !isZonableWindow(hwnd) {
		return
	}
	if w32.GetWindowLong(hwnd, w32.GWL_EXSTYLE)&w32.WS_EX_TOPMOST != 0 {
		w32.SetWindowPos(hwnd, w32.HWND_NOTOPMOST, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE)
	} else {
		w32.SetWindowPos(hwnd, w32.HWND_TOPMOST, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE)
	}
}

// moveToDisplay moves hwnd to the next or previous monitor,
// scaling the window position proportionally.
// Translates Rectangle's NextPrevDisplayCalculation.
func moveToDisplay(hwnd w32.HWND, next bool) {
	if !isZonableWindow(hwnd) {
		return
	}
	var monitors []w32.HMONITOR
	EnumMonitors(func(m w32.HMONITOR) bool {
		monitors = append(monitors, m)
		return true
	})
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
	var nextIdx int
	if next {
		nextIdx = (curIdx + 1) % len(monitors)
	} else {
		nextIdx = (curIdx - 1 + len(monitors)) % len(monitors)
	}
	moveToMonitor(hwnd, monitors[curIdx], monitors[nextIdx])
}

func moveToDisplayByIndex(hwnd w32.HWND, idx int) {
	if !isZonableWindow(hwnd) {
		return
	}
	var monitors []w32.HMONITOR
	EnumMonitors(func(m w32.HMONITOR) bool {
		monitors = append(monitors, m)
		return true
	})
	if idx < 0 || idx >= len(monitors) {
		return
	}
	cur := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	moveToMonitor(hwnd, cur, monitors[idx])
}

func moveToMonitor(hwnd w32.HWND, from, to w32.HMONITOR) {
	var fromInfo, toInfo w32.MONITORINFO
	w32.GetMonitorInfo(from, &fromInfo)
	w32.GetMonitorInfo(to, &toInfo)

	rect := w32.GetWindowRect(hwnd)
	if rect == nil {
		return
	}
	fw, fh := float64(fromInfo.RcWork.Width()), float64(fromInfo.RcWork.Height())
	tw, th := float64(toInfo.RcWork.Width()), float64(toInfo.RcWork.Height())

	relX := float64(rect.Left-fromInfo.RcWork.Left) / fw
	relY := float64(rect.Top-fromInfo.RcWork.Top) / fh
	scaleW := tw / fw
	scaleH := th / fh

	newLeft := toInfo.RcWork.Left + int32(relX*tw)
	newTop := toInfo.RcWork.Top + int32(relY*th)
	newW := int32(float64(rect.Width()) * scaleW)
	newH := int32(float64(rect.Height()) * scaleH)

	w32.ShowWindow(hwnd, w32.SW_SHOWNORMAL)
	w32.SetWindowPos(hwnd, 0, int(newLeft), int(newTop), int(newW), int(newH), w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
}

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

func scaleRect(src w32.RECT, from, to int32) w32.RECT {
	return w32.RECT{
		Left:   src.Left * to / from,
		Right:  src.Right * to / from,
		Top:    src.Top * to / from,
		Bottom: src.Bottom * to / from,
	}
}

func rectEqual(a, b *w32.RECT) bool {
	return a != nil && b != nil &&
		a.Left == b.Left && a.Top == b.Top && a.Right == b.Right && a.Bottom == b.Bottom
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

	// Skip genuine floating tool palettes (WS_EX_TOOLWINDOW without a caption).
	// Exception: ApplicationFrameWindow is the UWP/Store app host — always allow.
	if exStyle&w32.WS_EX_TOOLWINDOW == w32.WS_EX_TOOLWINDOW &&
		!strings.EqualFold(className, "ApplicationFrameWindow") {
		return false
	}

	// Skip popup windows that have no min/max buttons AND no caption —
	// these are typically menus, tooltips, or transient overlays.
	// (WS_THICKFRAME alone doesn't make something a real resizable window.)
	hasCaption := style&w32.WS_CAPTION == w32.WS_CAPTION
	isPopup := uint32(style)&w32.WS_POPUP == w32.WS_POPUP
	if isPopup && !hasCaption {
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
	}
	for _, c := range blocked {
		if strings.EqualFold(c, name) {
			return true
		}
	}
	return false
}
