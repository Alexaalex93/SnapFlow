package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

var testWork = w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

func TestGesture_LeftEdge(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	if action := g.Update(w32.POINT{X: 5, Y: 540}, testWork); action == ActionNone {
		t.Error("expected snap action at left edge, got ActionNone")
	}
}

func TestGesture_TopLeftCorner(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	if action := g.Update(w32.POINT{X: 5, Y: 5}, testWork); action != ActionTopLeft {
		t.Errorf("expected ActionTopLeft at top-left corner, got %d", action)
	}
}

func TestGesture_TopEdge_Maximize(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	if action := g.Update(w32.POINT{X: 960, Y: 2}, testWork); action != ActionMaximize {
		t.Errorf("expected ActionMaximize at top center, got %d", action)
	}
}

func TestGesture_NoEdge(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	if action := g.Update(w32.POINT{X: 960, Y: 540}, testWork); action != ActionNone {
		t.Errorf("expected ActionNone in center, got %d", action)
	}
}

func TestGesture_LeftEdgeMiddle_LeftHalf(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	if action := g.Update(w32.POINT{X: 5, Y: 540}, testWork); action != ActionLeftHalf {
		t.Errorf("expected ActionLeftHalf at left edge middle, got %d", action)
	}
}

// Compound thirds: the span grows from the entry third and SHRINKS again
// when the cursor moves back (the old min/max tracker could only grow).
func TestGesture_CompoundThirds_Reversible(t *testing.T) {
	g := NewGestureInterpreter(defaultConfig())
	g.Begin(1, testWork)
	y := testWork.Bottom - 2
	steps := []struct {
		x    int32
		want WindowAction
	}{
		{300, ActionFirstThird},      // enter in first third
		{900, ActionFirstTwoThirds},  // sweep into the centre
		{1600, ActionMaximize},       // all three
		{900, ActionFirstTwoThirds},  // back: span shrinks
		{300, ActionFirstThird},      // back to the anchor
		{1000, ActionFirstTwoThirds}, // and forward again
	}
	for i, s := range steps {
		if got := g.Update(w32.POINT{X: s.x, Y: y}, testWork); got != s.want {
			t.Fatalf("step %d (x=%d): got %d, want %d", i, s.x, got, s.want)
		}
	}
	// Leaving the bottom zone resets the anchor.
	g.Update(w32.POINT{X: 960, Y: 540}, testWork)
	if got := g.Update(w32.POINT{X: 1600, Y: y}, testWork); got != ActionLastThird {
		t.Fatalf("after re-entry: got %d, want LastThird", got)
	}
}

// A plain "Bottom Half" setting must not behave like compound thirds.
func TestGesture_BottomHalfIsNotThirds(t *testing.T) {
	cfg := defaultConfig()
	cfg.ZoneBottom = "bottomHalf"
	g := NewGestureInterpreter(cfg)
	g.Begin(1, testWork)
	if got := g.Update(w32.POINT{X: 300, Y: testWork.Bottom - 2}, testWork); got != ActionBottomHalf {
		t.Fatalf("got %d, want BottomHalf", got)
	}
}

// "none" must really disable a zone (it used to fall back to the default).
func TestGesture_ZoneNoneDisables(t *testing.T) {
	cfg := defaultConfig()
	cfg.ZoneTop = "none"
	g := NewGestureInterpreter(cfg)
	g.Begin(1, testWork)
	if got := g.Update(w32.POINT{X: 960, Y: 2}, testWork); got != ActionNone {
		t.Fatalf("top zone set to none, got action %d", got)
	}
}

func TestThirdsSpanAction(t *testing.T) {
	cases := []struct {
		a, b int
		want WindowAction
	}{
		{0, 0, ActionFirstThird}, {1, 1, ActionCenterThird}, {2, 2, ActionLastThird},
		{0, 1, ActionFirstTwoThirds}, {1, 0, ActionFirstTwoThirds},
		{1, 2, ActionLastTwoThirds}, {2, 1, ActionLastTwoThirds},
		{0, 2, ActionMaximize}, {2, 0, ActionMaximize},
	}
	for _, c := range cases {
		if got := thirdsSpanAction(c.a, c.b); got != c.want {
			t.Errorf("thirdsSpanAction(%d,%d)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

// Every zone produced for the default config maps to a drag-snappable,
// fully-resolved action (the internal compound marker never leaks).
func TestSnapZones_AllActionsSnappable(t *testing.T) {
	for _, z := range SnapZonesForWork(testWork, defaultConfig()) {
		if z.Action == actionCompoundThirds {
			continue // resolved by the gesture interpreter
		}
		if !z.Action.IsDragSnappable() {
			t.Errorf("zone %+v has non-snappable action %d", z.Region, z.Action)
		}
	}
}
