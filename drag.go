package main

import (
	"fmt"
	"sync"

	"github.com/ahmetb/RectangleWin/w32ex"
	"github.com/gonutz/w32/v2"
)

type DragManager struct {
	overlay *OverlayRenderer
	engine  *SnapEngine
	gest    *GestureInterpreter

	mu       sync.Mutex
	hooks    []w32ex.WinEventHook
	active   bool
	hwnd     w32.HWND
	workArea w32.RECT
}

func NewDragManager(engine *SnapEngine, gest *GestureInterpreter, overlay *OverlayRenderer) *DragManager {
	return &DragManager{engine: engine, gest: gest, overlay: overlay}
}

func (d *DragManager) Start() error {
	if d == nil {
		return nil
	}
	cb := func(_ w32ex.WinEventHook, event uint32, hwnd w32.HWND, idObject int32, _ int32, _ uint32, _ uint32) {
		if idObject != int32(w32ex.OBJID_WINDOW) {
			return
		}
		switch event {
		case w32ex.EVENT_SYSTEM_MOVESIZESTART:
			d.onMoveSizeStart(hwnd)
		case w32ex.EVENT_OBJECT_LOCATIONCHANGE:
			d.onLocationChange(hwnd)
		case w32ex.EVENT_SYSTEM_MOVESIZEEND:
			d.onMoveSizeEnd(hwnd)
		}
	}

	d.hooks = append(d.hooks,
		w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZESTART, w32ex.EVENT_SYSTEM_MOVESIZESTART, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
		w32ex.SetWinEventHook(w32ex.EVENT_OBJECT_LOCATIONCHANGE, w32ex.EVENT_OBJECT_LOCATIONCHANGE, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
		w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZEEND, w32ex.EVENT_SYSTEM_MOVESIZEEND, 0, 0, w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS, cb),
	)
	for _, h := range d.hooks {
		if h == 0 {
			return fmt.Errorf("failed to install one or more WinEvent hooks")
		}
	}
	return nil
}

func (d *DragManager) Stop() {
	if d == nil {
		return
	}
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
	_ = d.overlay.Hide()
}

func (d *DragManager) onMoveSizeStart(hwnd w32.HWND) {
	if !isZonableWindow(hwnd) {
		return
	}
	pt, ok := w32ex.GetCursorPos()
	if !ok {
		return
	}
	work, ok := monitorWorkAreaAtPoint(pt)
	if !ok {
		return
	}
	d.mu.Lock()
	d.active = true
	d.hwnd = hwnd
	d.workArea = work
	d.mu.Unlock()
	d.gest.Begin(hwnd, pt)
	_ = d.overlay.Hide()
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
	work, ok := monitorWorkAreaAtPoint(pt)
	if !ok {
		return
	}
	d.mu.Lock()
	d.workArea = work
	d.mu.Unlock()

	target := d.gest.Update(work, pt)
	if target == SnapNone || !d.engine.Allowed(target) {
		_ = d.overlay.Hide()
		return
	}
	_ = d.overlay.Show(d.engine.Bounds(work, target))
}

func (d *DragManager) onMoveSizeEnd(hwnd w32.HWND) {
	d.mu.Lock()
	active := d.active && d.hwnd == hwnd
	d.active = false
	d.hwnd = 0
	d.mu.Unlock()
	if !active {
		return
	}
	defer d.gest.End()
	defer d.overlay.Hide()

	target := d.gest.CurrentTarget()
	if target == SnapNone || !d.engine.Allowed(target) {
		return
	}
	if _, err := d.engine.Apply(hwnd, target); err != nil {
		fmt.Printf("warn: drag apply failed: %v\n", err)
		return
	}
}

func monitorWorkAreaAtPoint(pt w32.POINT) (w32.RECT, bool) {
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
