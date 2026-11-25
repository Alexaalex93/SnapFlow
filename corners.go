package main

import (
	"sync"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// Square corners for snapped windows.
//
// Windows 11 draws tiled (Snap) windows with square corners and floating ones
// rounded. SnapFlow mirrors that with DWMWA_WINDOW_CORNER_PREFERENCE.
// Only windows SnapFlow squared are ever reset to the default style, so apps
// that chose their own corner style are never touched.

var (
	squaredMu sync.Mutex
	squared   = map[w32.HWND]bool{}
)

// updateCorners applies the corner style for a window that was just placed
// by action. Main thread.
func updateCorners(hwnd w32.HWND, action WindowAction) {
	if appCfg.SquareCorners && action.IsTiling() {
		squareCorners(hwnd)
	} else {
		roundCorners(hwnd)
	}
}

func squareCorners(hwnd w32.HWND) {
	if !w32ex.SetWindowCornerPreference(hwnd, w32ex.DWMWCP_DONOTROUND) {
		return // Windows 10, or the window refused
	}
	squaredMu.Lock()
	squared[hwnd] = true
	if len(squared) > 256 {
		for h := range squared {
			if !w32.IsWindow(h) {
				delete(squared, h)
			}
		}
	}
	squaredMu.Unlock()
}

// roundCorners gives a window SnapFlow had squared its default corners back
// (it was unsnapped, restored, or moved freely).
func roundCorners(hwnd w32.HWND) {
	squaredMu.Lock()
	was := squared[hwnd]
	delete(squared, hwnd)
	squaredMu.Unlock()
	if was {
		w32ex.SetWindowCornerPreference(hwnd, w32ex.DWMWCP_DEFAULT)
	}
}

// roundAllCorners resets every window SnapFlow squared (option turned off,
// or SnapFlow quitting).
func roundAllCorners() {
	squaredMu.Lock()
	list := make([]w32.HWND, 0, len(squared))
	for h := range squared {
		list = append(list, h)
	}
	squared = map[w32.HWND]bool{}
	squaredMu.Unlock()
	for _, h := range list {
		if w32.IsWindow(h) {
			w32ex.SetWindowCornerPreference(h, w32ex.DWMWCP_DEFAULT)
		}
	}
}
