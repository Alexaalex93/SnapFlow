package main

import (
	"sync"

	"github.com/gonutz/w32/v2"
)

// WindowHistory tracks the last position of each window so ActionRestore works.
// Translates Rectangle's WindowHistory.
type WindowHistory struct {
	mu      sync.Mutex
	entries map[w32.HWND]w32.RECT
}

var windowHistory = &WindowHistory{
	entries: make(map[w32.HWND]w32.RECT),
}

func (h *WindowHistory) Save(hwnd w32.HWND, rect w32.RECT) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries[hwnd] = rect
}

func (h *WindowHistory) Pop(hwnd w32.HWND) (w32.RECT, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.entries[hwnd]
	if ok {
		delete(h.entries, hwnd)
	}
	return r, ok
}
