package main

import (
	"sync"

	"github.com/gonutz/w32/v2"
)

// GestureInterpreter tracks a window drag and maps cursor position
// to a WindowAction via snap zones.
// Translates Rectangle's SnappingManager drag detection logic.
//
// Threading: Begin/Update/End/CurrentAction are called from the WinEvent
// callback (main OS thread). UpdateConfig is called from the settings
// save path (WebView goroutine). The cfgMu read-write mutex makes cfg
// access safe across those two paths.
type GestureInterpreter struct {
	cfgMu sync.RWMutex
	cfg   *AppConfig
	state gestureState
}

// bottomZoneState tracks cursor horizontal extent while in the bottom snap zone.
// This enables the "compound thirds" behaviour: drag across multiple thirds to
// snap the window to the combined area (like Rectangle's ThirdsCompoundCalculation).
type bottomZoneState struct {
	active bool
	xMin   int32 // leftmost cursor X seen while in bottom zone
	xMax   int32 // rightmost cursor X seen while in bottom zone
}

type gestureState struct {
	active     bool
	hwnd       w32.HWND
	lastAction WindowAction
	zones      []SnapZone
	work       w32.RECT
	bottom     bottomZoneState
}

func NewGestureInterpreter(cfg *AppConfig) *GestureInterpreter {
	return &GestureInterpreter{cfg: cfg}
}

// UpdateConfig replaces the active config. Safe to call from any goroutine.
func (g *GestureInterpreter) UpdateConfig(cfg *AppConfig) {
	g.cfgMu.Lock()
	g.cfg = cfg
	g.cfgMu.Unlock()
}

func (g *GestureInterpreter) readCfg() *AppConfig {
	g.cfgMu.RLock()
	defer g.cfgMu.RUnlock()
	return g.cfg
}

func (g *GestureInterpreter) Begin(hwnd w32.HWND, work w32.RECT) {
	cfg := g.readCfg()
	g.state = gestureState{
		active:     true,
		hwnd:       hwnd,
		lastAction: ActionNone,
		zones:      SnapZonesForWork(work, cfg),
		work:       work,
	}
}

func (g *GestureInterpreter) End() { g.state = gestureState{} }

func (g *GestureInterpreter) CurrentAction() WindowAction { return g.state.lastAction }

// Update evaluates the cursor position and returns the matching WindowAction.
func (g *GestureInterpreter) Update(cursor w32.POINT, work w32.RECT) WindowAction {
	if !g.state.active {
		return ActionNone
	}
	cfg := g.readCfg()

	// Rebuild zones if the monitor changed.
	if len(g.state.zones) == 0 || !rectEqual(&g.state.work, &work) {
		g.state.zones = SnapZonesForWork(work, cfg)
		g.state.work = work
	}

	action := ActionForCursor(cursor, g.state.zones)

	// ── Compound bottom thirds ────────────────────────────────────────────
	// When the cursor is in the bottom snap zone, track horizontal extent
	// so that dragging across multiple thirds combines them.
	if action == ActionBottomHalf || action == ActionFirstThird ||
		action == ActionCenterThird || action == ActionLastThird ||
		action == ActionFirstTwoThirds || action == ActionLastTwoThirds {

		inBottom := cursorInBottomZone(cursor, work, cfg)
		if inBottom {
			if !g.state.bottom.active {
				// Entering the bottom zone — start tracking.
				g.state.bottom = bottomZoneState{
					active: true,
					xMin:   cursor.X,
					xMax:   cursor.X,
				}
			} else {
				// Extend the tracked range.
				if cursor.X < g.state.bottom.xMin {
					g.state.bottom.xMin = cursor.X
				}
				if cursor.X > g.state.bottom.xMax {
					g.state.bottom.xMax = cursor.X
				}
			}
			// Calculate action based on horizontal extent.
			action = bottomThirdsAction(work, g.state.bottom.xMin, g.state.bottom.xMax)
		} else {
			g.state.bottom.active = false
		}
	} else {
		// Left the bottom zone — reset tracking so next entry starts fresh.
		g.state.bottom.active = false
	}

	g.state.lastAction = action
	return action
}

// cursorInBottomZone returns true if the cursor is in the bottom snap edge area.
func cursorInBottomZone(cursor w32.POINT, work w32.RECT, cfg *AppConfig) bool {
	edgeW := int32(cfg.EdgeThresholdPx)
	if edgeW <= 0 {
		edgeW = 46
	}
	corner := int32(cfg.CornerSnapAreaSize)
	if corner <= 0 {
		corner = 20
	}
	return cursor.Y >= work.Bottom-edgeW &&
		cursor.X > work.Left+corner &&
		cursor.X < work.Right-corner
}

// bottomThirdsAction calculates which snap action to apply based on how far
// the cursor has been dragged horizontally within the bottom zone.
//
// Rules (matching Rectangle's compound thirds behaviour):
//
//	Left third only              → FirstThird   (left 1/3 of screen)
//	Center third only            → CenterThird
//	Right third only             → LastThird    (right 1/3 of screen)
//	Left + Center                → FirstTwoThirds
//	Center + Right               → LastTwoThirds
//	All three (left→right span)  → Maximize
func bottomThirdsAction(work w32.RECT, xMin, xMax int32) WindowAction {
	W := work.Width()
	t1 := work.Left + W/3   // end of first third
	t2 := work.Left + W*2/3 // end of second third

	coversFirst := xMin < t1
	coversCenter := xMax > work.Left+W/3 && xMin < t2
	coversLast := xMax >= t2

	switch {
	case coversFirst && coversCenter && coversLast:
		return ActionMaximize
	case coversFirst && coversCenter:
		return ActionFirstTwoThirds
	case coversCenter && coversLast:
		return ActionLastTwoThirds
	case coversFirst:
		return ActionFirstThird
	case coversLast:
		return ActionLastThird
	default:
		return ActionCenterThird
	}
}
