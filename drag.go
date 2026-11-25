package main

import (
	"sync"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// DragManager detects window drags via WinEvent hooks and applies snap actions.
// Translates Rectangle's SnappingManager.
//
// All hook callbacks run on the main OS thread (WINEVENT_OUTOFCONTEXT hooks
// installed from it). The mutex only protects the fields read by
// IsDragging(), which background goroutines call.
type DragManager struct {
	overlay *OverlayRenderer
	gest    *GestureInterpreter

	startHook w32ex.WinEventHook
	endHook   w32ex.WinEventHook
	// locHook is installed per drag and scoped to the dragged window's
	// process. A global LOCATIONCHANGE hook fires for every caret blink,
	// progress bar and tooltip on the whole desktop; scoping it removes
	// that constant background cost.
	locHook w32ex.WinEventHook

	mu     sync.Mutex
	active bool
	hwnd   w32.HWND

	startCursor w32.POINT
	startFrame  w32.RECT // frame when the drag began (after unsnap-restore)
	moved       bool     // cursor travelled far enough to count as a drag

	// unsnap-restore bookkeeping (see onMoveStart)
	unsnapped   bool
	restoreSize w32.RECT
}

const dragDeadZone = 4 // px the cursor must travel before snapping is considered

func NewDragManager(gest *GestureInterpreter, overlay *OverlayRenderer) *DragManager {
	return &DragManager{gest: gest, overlay: overlay}
}

func (d *DragManager) Start() error {
	flags := uint32(w32ex.WINEVENT_OUTOFCONTEXT | w32ex.WINEVENT_SKIPOWNPROCESS)
	d.startHook = w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZESTART, w32ex.EVENT_SYSTEM_MOVESIZESTART, 0, 0, flags,
		func(_ w32ex.WinEventHook, _ uint32, hwnd w32.HWND, idObject int32, _ int32, _ uint32, _ uint32) {
			if idObject == w32ex.OBJID_WINDOW {
				d.onMoveStart(hwnd)
			}
		})
	d.endHook = w32ex.SetWinEventHook(w32ex.EVENT_SYSTEM_MOVESIZEEND, w32ex.EVENT_SYSTEM_MOVESIZEEND, 0, 0, flags,
		func(_ w32ex.WinEventHook, _ uint32, hwnd w32.HWND, idObject int32, _ int32, _ uint32, _ uint32) {
			if idObject == w32ex.OBJID_WINDOW {
				d.onMoveEnd(hwnd)
			}
		})
	if d.startHook == 0 || d.endHook == 0 {
		d.Stop()
		return errHookInstall
	}
	return nil
}

func (d *DragManager) Stop() {
	w32ex.UnhookWinEvent(d.startHook)
	w32ex.UnhookWinEvent(d.endHook)
	d.startHook, d.endHook = 0, 0
	d.endDrag()
	d.overlay.Hide()
}

// IsDragging reports whether hwnd is currently being dragged. Safe from any goroutine.
func (d *DragManager) IsDragging(hwnd w32.HWND) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active && d.hwnd == hwnd
}

func (d *DragManager) endDrag() {
	if d.locHook != 0 {
		w32ex.UnhookWinEvent(d.locHook)
		d.locHook = 0
	}
	d.mu.Lock()
	d.active = false
	d.hwnd = 0
	d.mu.Unlock()
	d.gest.End()
}

// isResizeHit asks the window what part of it is under the cursor. Dragging
// a sizing border also fires EVENT_SYSTEM_MOVESIZESTART; without this check
// resizing a window towards a screen edge would show the overlay and snap
// it on release.
func isResizeHit(hwnd w32.HWND, pt w32.POINT) bool {
	lp := uintptr(uint16(int16(pt.X))) | uintptr(uint16(int16(pt.Y)))<<16
	hit, ok := w32ex.SendMessageTimeout(hwnd, w32ex.WM_NCHITTEST, 0, lp, 60)
	if !ok {
		return false // hung or refused: assume it's a move
	}
	const (
		htLeft        = 10
		htBottomRight = 17
		htBorder      = 18
	)
	return (hit >= htLeft && hit <= htBottomRight) || hit == htBorder
}

func (d *DragManager) onMoveStart(hwnd w32.HWND) {
	if d.locHook != 0 { // a previous drag never got its END event
		d.endDrag()
	}
	cfg := appCfg
	if !cfg.SnapByDragging && !cfg.RestoreSize {
		return
	}
	if !isZonableWindow(hwnd) {
		return
	}
	pt, ok := w32ex.GetCursorPos()
	if !ok {
		return
	}
	if isResizeHit(hwnd, pt) {
		return
	}
	frame, ok := frameRect(hwnd)
	if !ok {
		return
	}
	work, ok := monitorWorkArea(pt)
	if !ok {
		return
	}

	// ── Restore size when unsnapped ──────────────────────────────────────
	// If SnapFlow snapped this window and it's still in that spot, give it
	// back its original size right away, keeping the cursor at the same
	// relative spot of the title bar (like Rectangle / Windows Snap).
	d.unsnapped = false
	if cfg.RestoreSize && !w32ex.IsZoomed(hwnd) {
		if restore, ok := windowHistory.Snapped(hwnd, frame); ok &&
			(restore.Width() != frame.Width() || restore.Height() != frame.Height()) {
			nf := unsnapFrame(frame, restore, pt)
			if in, ok := windowInsets(hwnd); ok && setWindowFrame(hwnd, nf, in, 0) {
				windowHistory.Forget(hwnd)
				d.unsnapped = true
				d.restoreSize = restore
				if f, ok := frameRect(hwnd); ok {
					frame = f
				}
			}
		}
	}

	if !cfg.SnapByDragging {
		return // only the unsnap-restore was wanted
	}

	_, pid := w32.GetWindowThreadProcessId(hwnd)
	d.locHook = w32ex.SetWinEventHook(w32ex.EVENT_OBJECT_LOCATIONCHANGE, w32ex.EVENT_OBJECT_LOCATIONCHANGE,
		uint32(pid), 0, uint32(w32ex.WINEVENT_OUTOFCONTEXT|w32ex.WINEVENT_SKIPOWNPROCESS),
		func(_ w32ex.WinEventHook, _ uint32, h w32.HWND, idObject int32, _ int32, _ uint32, _ uint32) {
			if idObject == w32ex.OBJID_WINDOW {
				d.onLocationChange(h)
			}
		})

	d.mu.Lock()
	d.active = true
	d.hwnd = hwnd
	d.mu.Unlock()
	d.startCursor = pt
	d.startFrame = frame
	d.moved = false
	d.gest.Begin(hwnd, work)
}

// unsnapFrame returns a frame of restore's size positioned so the cursor keeps
// the same relative horizontal position inside the title bar. Pure geometry.
func unsnapFrame(frame, restore w32.RECT, cursor w32.POINT) w32.RECT {
	rw, rh := restore.Width(), restore.Height()
	rel := 0.5
	if fw := frame.Width(); fw > 0 {
		rel = float64(cursor.X-frame.Left) / float64(fw)
	}
	left := cursor.X - int32(rel*float64(rw))
	top := frame.Top
	// Keep the grab point inside the new (smaller) title bar.
	if cursor.Y-top > rh/2 {
		top = cursor.Y - 16
	}
	return w32.RECT{Left: left, Top: top, Right: left + rw, Bottom: top + rh}
}

func (d *DragManager) onLocationChange(hwnd w32.HWND) {
	if !d.IsDragging(hwnd) {
		return
	}
	pt, ok := w32ex.GetCursorPos()
	if !ok {
		return
	}
	if !d.moved {
		dx, dy := pt.X-d.startCursor.X, pt.Y-d.startCursor.Y
		if dx*dx+dy*dy < dragDeadZone*dragDeadZone {
			return
		}
		d.moved = true
	}
	work, ok := monitorWorkArea(pt)
	if !ok {
		return
	}

	action := d.gest.Update(pt, work)
	if action == ActionNone || !action.IsDragSnappable() {
		d.overlay.Hide()
		return
	}
	d.overlay.Show(d.previewRect(hwnd, action, work))
}

func (d *DragManager) previewRect(hwnd w32.HWND, action WindowAction, work w32.RECT) w32.RECT {
	cur := d.startFrame
	if action.UsesCurrentRect() {
		if f, ok := frameRect(hwnd); ok {
			cur = f
		}
	}
	return action.Calculate(work, cur, appCfg)
}

func (d *DragManager) onMoveEnd(hwnd w32.HWND) {
	if !d.IsDragging(hwnd) {
		return
	}
	pt, _ := w32ex.GetCursorPos()
	work, workOK := monitorWorkArea(pt)

	// Final evaluation at the release point (covers "show window contents
	// while dragging" = off, where no LOCATIONCHANGE events arrive).
	action := ActionNone
	if workOK {
		action = d.gest.Update(pt, work)
	}
	dx, dy := pt.X-d.startCursor.X, pt.Y-d.startCursor.Y
	moved := d.moved || dx*dx+dy*dy >= dragDeadZone*dragDeadZone

	frameNow, _ := frameRect(hwnd)
	// Esc during a drag makes Windows put the window back where it started.
	cancelled := moved && rectNear(frameNow, d.startFrame, 0)

	before := d.startFrame
	unsnapped, restoreSize := d.unsnapped, d.restoreSize
	d.endDrag()

	if !moved || cancelled || !workOK || action == ActionNone || !action.IsDragSnappable() {
		d.overlay.Hide()
		// Fallback for apps whose move loop overrode our unsnap resize.
		if unsnapped && !cancelled && (frameNow.Width() != restoreSize.Width() || frameNow.Height() != restoreSize.Height()) {
			nf := unsnapFrame(frameNow, restoreSize, pt)
			resizeToRect(hwnd, nf)
		}
		return
	}

	if action == ActionMaximize {
		maximize(hwnd)
		d.overlay.Hide()
		return
	}

	cur := before
	if action.UsesCurrentRect() {
		cur = frameNow
	}
	target := action.Calculate(work, cur, appCfg)
	got, err := resizeToRect(hwnd, target)
	if err == nil && got.Width() > 0 {
		windowHistory.RecordSnap(hwnd, before, got)
	}
	// Hide AFTER placing: the preview fades out on top of the window as it
	// lands, which masks the app's repaint and feels much smoother.
	d.overlay.Hide()
}

func monitorWorkArea(pt w32.POINT) (w32.RECT, bool) {
	return monitorWork(w32ex.MonitorFromPoint(pt, w32ex.MONITOR_DEFAULTTONEAREST))
}
