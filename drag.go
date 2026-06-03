package main

import (
	"fmt"
	"sync"
	"time"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// DragManager detects window drags via WinEvent hooks and applies snap actions.
// Translates Rectangle's SnappingManager.
type DragManager struct {
	overlay *OverlayRenderer
	gest    *GestureInterpreter

	mu            sync.Mutex
	hooks         []w32ex.WinEventHook
	active        bool
	hwnd          w32.HWND
	workArea      w32.RECT
	preSnap       w32.RECT // window rect before snap, for history
	lastPreviewAt time.Time
}

func NewDragManager(gest *GestureInterpreter, overlay *OverlayRenderer) *DragManager {
	return &DragManager{gest: gest, overlay: overlay}
}

func (d *DragManager) Start() error {
	cb := func(_ w32ex.WinEventHook, event uint32, hwnd w32.HWND, idObject int32, _ int32, _ uint32, _ uint32) {
		if idObject != int32(w32ex.OBJID_WINDOW) {
			return
		}
		switch event {
		case w32ex.EVENT_SYSTEM_MOVESIZESTART:
			d.onMoveStart(hwnd)
		case w32ex.EVENT_OBJECT_LOCATIONCHANGE:
			d.onLocationChange(hwnd)
		case w32ex.EVENT_SYSTEM_MOVESIZEEND:
			d.onMoveEnd(hwnd)
		}
	}

	d.hooks = append(d.hooks,
		w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZESTART, w32ex.EVENT_SYSTEM_MOVESIZESTART, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
		w32ex.SetWinEventHook(w32ex.EVENT_OBJECT_LOCATIONCHANGE, w32ex.EVENT_OBJECT_LOCATIONCHANGE, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
		w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZEEND, w32ex.EVENT_SYSTEM_MOVESIZEEND, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
	)
	for _, h := range d.hooks {
		if h == 0 {
			return fmt.Errorf("failed to install WinEvent hook")
		}
	}
	return nil
}

func (d *DragManager) Stop() {
	for _, h := range d.hooks {
		if h != 0 {
			w32ex.UnhookWinEvent(h)
		}
	}
	d.hooks = nil
	d.mu.Lock()
	d.active = false
	d.hwnd = 0
	d.mu.Unlock()
	d.overlay.Hide()
}

func (d *DragManager) onMoveStart(hwnd w32.HWND) {
	if !isZonableWindow(hwnd) {
		return
	}
	pt, ok := w32ex.GetCursorPos()
	if !ok {
		return
	}
	work, ok := monitorWorkArea(pt)
	if !ok {
		return
	}
	rect := w32.GetWindowRect(hwnd)
	if rect == nil {
		// Can't determine pre-snap position — skip this drag entirely so
		// onMoveEnd never applies a snap to a zero RECT.
		return
	}
	d.mu.Lock()
	d.active = true
	d.hwnd = hwnd
	d.workArea = work
	d.preSnap = *rect
	d.mu.Unlock()
	d.gest.Begin(hwnd, work)
	d.overlay.Hide()
}

func (d *DragManager) onLocationChange(hwnd w32.HWND) {
	d.mu.Lock()
	active := d.active && d.hwnd == hwnd
	d.mu.Unlock()
	if !active {
		return
	}

	pt, ok := w32ex.GetCursorPos()
	if !ok {
		return
	}
	work, ok := monitorWorkArea(pt)
	if !ok {
		return
	}
	d.mu.Lock()
	d.workArea = work
	d.mu.Unlock()

	action := d.gest.Update(pt, work)
	if action == ActionNone || !action.IsDragSnappable() {
		d.mu.Lock()
		lastAt := d.lastPreviewAt
		d.mu.Unlock()
		if time.Since(lastAt) < 80*time.Millisecond {
			return
		}
		d.overlay.Hide()
		return
	}

	// Show footprint at the target position.
	target := action.Calculate(work, d.getPreSnap(), appCfg)
	d.overlay.Show(target)
	d.mu.Lock()
	d.lastPreviewAt = time.Now()
	d.mu.Unlock()
}

func (d *DragManager) onMoveEnd(hwnd w32.HWND) {
	d.mu.Lock()
	active := d.active && d.hwnd == hwnd
	d.active = false
	d.hwnd = 0
	pre := d.preSnap
	work := d.workArea
	d.mu.Unlock()

	d.overlay.Hide()
	action := d.gest.CurrentAction()
	d.gest.End()

	if !active {
		return
	}
	if action == ActionNone || !action.IsDragSnappable() {
		return
	}

	// Save current position to history before snapping.
	if rect := w32.GetWindowRect(hwnd); rect != nil {
		windowHistory.Save(hwnd, *rect)
	}

	target := action.Calculate(work, pre, appCfg)

	if action == ActionMaximize {
		maximize(hwnd)
		return
	}
	if _, err := resizeToRect(hwnd, target); err != nil {
		fmt.Printf("warn: snap apply: %v\n", err)
	}
}

func (d *DragManager) getPreSnap() w32.RECT {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.preSnap
}

func monitorWorkArea(pt w32.POINT) (w32.RECT, bool) {
	mon := w32ex.MonitorFromPoint(pt, w32ex.MONITOR_DEFAULTTONEAREST)
	if mon == 0 {
		return w32.RECT{}, false
	}
	var info w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &info) {
		return w32.RECT{}, false
	}
	return info.RcWork, true
}
