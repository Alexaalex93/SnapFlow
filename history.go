package main

import (
	"sync"

	"github.com/gonutz/w32/v2"
)

// WindowHistory remembers, per window, the free-floating frame it had before
// SnapFlow first snapped it ("restore rect") and the frame SnapFlow last put
// it in ("snapped rect"). Translates Rectangle's WindowHistory.
//
// Snapping an already-snapped window again (e.g. Left Half → Top Left) keeps
// the ORIGINAL restore rect, so Restore and "restore size when unsnapped"
// always bring back the size the user chose, not an intermediate snap.
//
// All rects are visible-frame rects (see frameRect), never GetWindowRect.
type WindowHistory struct {
	mu      sync.Mutex
	entries map[w32.HWND]*historyEntry
}

type historyEntry struct {
	restore w32.RECT
	snapped w32.RECT
	// cycle state for repeated half shortcuts (½ → ⅔ → ⅓).
	cycleBase  WindowAction
	cycleIndex int
}

var windowHistory = &WindowHistory{entries: make(map[w32.HWND]*historyEntry)}

// RecordSnap is called after SnapFlow placed hwnd. before is the frame just
// before this snap, applied the frame achieved.
func (h *WindowHistory) RecordSnap(hwnd w32.HWND, before, applied w32.RECT) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.entries[hwnd]
	if ok && rectNear(before, e.snapped, 3) {
		// Was still in our last snapped position → keep the original restore rect.
		e.snapped = applied
		return
	}
	if !ok {
		e = &historyEntry{}
		h.entries[hwnd] = e
		h.pruneLocked()
	}
	e.restore = before
	e.snapped = applied
	e.cycleBase, e.cycleIndex = ActionNone, 0
}

// Snapped returns the restore rect if hwnd is still exactly where SnapFlow
// last put it (i.e. it is currently "snapped").
func (h *WindowHistory) Snapped(hwnd w32.HWND, current w32.RECT) (restore w32.RECT, ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, found := h.entries[hwnd]
	if !found || !rectNear(current, e.snapped, 3) {
		return w32.RECT{}, false
	}
	return e.restore, true
}

// Pop removes and returns the restore rect (used by ActionRestore and by
// unsnap-restore).
func (h *WindowHistory) Pop(hwnd w32.HWND) (w32.RECT, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.entries[hwnd]
	if !ok {
		return w32.RECT{}, false
	}
	delete(h.entries, hwnd)
	return e.restore, true
}

// Forget drops any state for hwnd (window was moved freely by the user).
func (h *WindowHistory) Forget(hwnd w32.HWND) {
	h.mu.Lock()
	delete(h.entries, hwnd)
	h.mu.Unlock()
}

// NextCycle returns which step of a size cycle to apply for base: if hwnd is
// still at the rect of the previous step of the same cycle, advance; else 0.
func (h *WindowHistory) NextCycle(hwnd w32.HWND, base WindowAction, current w32.RECT, steps int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.entries[hwnd]
	if !ok || e.cycleBase != base || !rectNear(current, e.snapped, 3) {
		return 0
	}
	return (e.cycleIndex + 1) % steps
}

// SetCycle stores the cycle step that was just applied.
func (h *WindowHistory) SetCycle(hwnd w32.HWND, base WindowAction, index int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if e, ok := h.entries[hwnd]; ok {
		e.cycleBase, e.cycleIndex = base, index
	}
}

// pruneLocked drops entries for windows that no longer exist so the map
// can't grow without bound over a long session.
func (h *WindowHistory) pruneLocked() {
	if len(h.entries) < 128 {
		return
	}
	for hwnd := range h.entries {
		if !w32.IsWindow(hwnd) {
			delete(h.entries, hwnd)
		}
	}
}
