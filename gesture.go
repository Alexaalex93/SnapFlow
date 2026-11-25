package main

import (
	"sync"

	"github.com/gonutz/w32/v2"
)

// actionCompoundThirds is an internal zone action used only for the bottom
// edge when it is configured as "compoundThirds". The gesture interpreter
// resolves it into a concrete third / two-thirds / maximize action, so it
// never reaches Calculate().
const actionCompoundThirds WindowAction = -2

// GestureInterpreter tracks a window drag and maps cursor position
// to a WindowAction via snap zones.
// Translates Rectangle's SnappingManager drag detection logic.
//
// Threading: Begin/Update/End/CurrentAction are called from the WinEvent
// callback (main OS thread). UpdateConfig may be called from anywhere; the
// cfgMu read-write mutex makes cfg access safe.
type GestureInterpreter struct {
	cfgMu sync.RWMutex
	cfg   *AppConfig
	state gestureState
}

// bottomZoneState implements Rectangle-style compound thirds on the bottom
// edge. The third where the cursor ENTERED the bottom zone is the anchor;
// the window spans from the anchor third to the third under the cursor.
// Moving back towards the anchor shrinks the span again, so the gesture is
// fully reversible (the old min/max tracking could only ever grow).
type bottomZoneState struct {
	active bool
	anchor int // 0, 1 or 2
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

	// Rebuild zones if the monitor changed.
	if len(g.state.zones) == 0 || !rectEqual(&g.state.work, &work) {
		g.state.zones = SnapZonesForWork(work, g.readCfg())
		g.state.work = work
		g.state.bottom.active = false
	}

	action := ActionForCursor(cursor, g.state.zones)

	if action == actionCompoundThirds {
		cur := thirdIndex(work, cursor.X)
		if !g.state.bottom.active {
			g.state.bottom = bottomZoneState{active: true, anchor: cur}
		}
		action = thirdsSpanAction(g.state.bottom.anchor, cur)
	} else {
		// Left the bottom zone — next entry starts a fresh span.
		g.state.bottom.active = false
	}

	g.state.lastAction = action
	return action
}

// thirdIndex returns 0, 1 or 2 for the horizontal third containing x.
func thirdIndex(work w32.RECT, x int32) int {
	W := work.Width()
	if W <= 0 {
		return 1
	}
	i := int((x - work.Left) * 3 / W)
	if i < 0 {
		return 0
	}
	if i > 2 {
		return 2
	}
	return i
}

// thirdsSpanAction maps a span of thirds [a..b] (either order) to an action:
//
//	single third        → First / Center / Last Third
//	first + center      → First Two Thirds
//	center + last       → Last Two Thirds
//	all three           → Maximize
func thirdsSpanAction(a, b int) WindowAction {
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	switch {
	case lo == 0 && hi == 2:
		return ActionMaximize
	case lo == 0 && hi == 1:
		return ActionFirstTwoThirds
	case lo == 1 && hi == 2:
		return ActionLastTwoThirds
	case lo == 0:
		return ActionFirstThird
	case lo == 2:
		return ActionLastThird
	default:
		return ActionCenterThird
	}
}

// bottomThirdsAction is kept for callers/tests that think in x-coordinates:
// it returns the action for a sweep between x positions xa and xb.
func bottomThirdsAction(work w32.RECT, xa, xb int32) WindowAction {
	return thirdsSpanAction(thirdIndex(work, xa), thirdIndex(work, xb))
}
