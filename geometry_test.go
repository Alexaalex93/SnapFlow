package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestExtendFlushEdges(t *testing.T) {
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	// Left half: left/top/bottom are flush, right edge is interior.
	got := extendFlushEdges(w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 1040}, work)
	want := w32.RECT{Left: -1, Top: -1, Right: 960, Bottom: 1041}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestReanchor_MinSizeRightAnchored(t *testing.T) {
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	want := w32.RECT{Left: 1440, Top: -1, Right: 1921, Bottom: 1041} // right fourth (extended)
	got := w32.RECT{Left: 1440, Top: -1, Right: 1440 + 700, Bottom: 1041}
	r := reanchor(got, want, work)
	if r.Right != 1921 || r.Width() != 700 {
		t.Fatalf("right-anchored window should hug the right edge: %+v", r)
	}
}

func TestReanchor_MinSizeLeftAnchored(t *testing.T) {
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	want := w32.RECT{Left: -1, Top: -1, Right: 480, Bottom: 1041}
	got := w32.RECT{Left: -1, Top: -1, Right: 699, Bottom: 1041}
	if r := reanchor(got, want, work); r.Left != -1 {
		t.Fatalf("left-anchored window should stay at the left edge: %+v", r)
	}
}

func TestMapRectBetween_HalfStaysHalf(t *testing.T) {
	from := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	to := w32.RECT{Left: 1920, Top: 0, Right: 1920 + 2560, Bottom: 1400}
	got := mapRectBetween(w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 1040}, from, to)
	want := w32.RECT{Left: 1920, Top: 0, Right: 1920 + 1280, Bottom: 1400}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestUnsnapFrame_KeepsCursorRelative(t *testing.T) {
	frame := w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 1040}
	restore := w32.RECT{Left: 200, Top: 200, Right: 1000, Bottom: 800} // 800×600
	cursor := w32.POINT{X: 480, Y: 10}                                 // middle of title bar
	got := unsnapFrame(frame, restore, cursor)
	if got.Width() != 800 || got.Height() != 600 {
		t.Fatalf("size not restored: %+v", got)
	}
	if mid := got.Left + got.Width()/2; mid != cursor.X {
		t.Fatalf("cursor should stay centred in the title bar: left=%d mid=%d", got.Left, mid)
	}
}

func TestRectNear(t *testing.T) {
	a := w32.RECT{Left: 10, Top: 10, Right: 100, Bottom: 100}
	if !rectNear(a, w32.RECT{Left: 12, Top: 9, Right: 101, Bottom: 100}, 2) {
		t.Fatal("expected near")
	}
	if rectNear(a, w32.RECT{Left: 14, Top: 10, Right: 100, Bottom: 100}, 2) {
		t.Fatal("expected not near")
	}
}

func TestCalculate_HalvesCoverWorkArea(t *testing.T) {
	cfg := defaultConfig()
	work := w32.RECT{Left: -1920, Top: 0, Right: 0, Bottom: 1041} // odd height, negative coords
	l := ActionLeftHalf.Calculate(work, work, cfg)
	r := ActionRightHalf.Calculate(work, work, cfg)
	if l.Left != work.Left || r.Right != work.Right || l.Right != r.Left {
		t.Fatalf("halves must tile the work area: %+v %+v", l, r)
	}
	ft := ActionFirstThird.Calculate(work, work, cfg)
	ct := ActionCenterThird.Calculate(work, work, cfg)
	lt := ActionLastThird.Calculate(work, work, cfg)
	if ft.Right != ct.Left || ct.Right != lt.Left || lt.Right != work.Right {
		t.Fatalf("thirds must tile without gaps: %+v %+v %+v", ft, ct, lt)
	}
}

func TestCalculate_GapOnlyBetweenWindows(t *testing.T) {
	cfg := defaultConfig()
	cfg.GapSize = 10
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	l := ActionLeftHalf.Calculate(work, work, cfg)
	r := ActionRightHalf.Calculate(work, work, cfg)
	if l.Left != 0 || r.Right != 1920 || r.Left-l.Right != 10 {
		t.Fatalf("gap should be 10px between halves and 0 at the screen edges: %+v %+v", l, r)
	}
}

func TestActionsUsingCurrentRect(t *testing.T) {
	if !ActionCenter.UsesCurrentRect() || ActionLeftHalf.UsesCurrentRect() {
		t.Fatal("UsesCurrentRect classification is wrong")
	}
	if actionCompoundThirds.IsDragSnappable() || ActionNone.IsDragSnappable() {
		t.Fatal("internal markers must never be snappable")
	}
}

func TestIsTiling(t *testing.T) {
	for _, a := range []WindowAction{ActionLeftHalf, ActionFirstThird, ActionTopLeft, ActionBottomRightSixth, ActionMaximize} {
		if !a.IsTiling() {
			t.Errorf("action %d should be tiling (square corners)", a)
		}
	}
	for _, a := range []WindowAction{ActionCenter, ActionAlmostMaximize, ActionLarger, ActionRestore, ActionNone, actionCompoundThirds} {
		if a.IsTiling() {
			t.Errorf("action %d leaves the window floating (rounded corners)", a)
		}
	}
}

func TestGrabFrame_KeepsCursorInTitleBar(t *testing.T) {
	maxFrame := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	normal := w32.RECT{Left: 300, Top: 200, Right: 1100, Bottom: 800} // 800×600
	down := w32.POINT{X: 1440, Y: 12}                                 // 75% across the title bar
	cursor := w32.POINT{X: 1450, Y: 20}
	got := grabFrame(maxFrame, normal, down, cursor)
	if got.Width() != 800 || got.Height() != 600 {
		t.Fatalf("restored size wrong: %+v", got)
	}
	if rel := float64(cursor.X-got.Left) / float64(got.Width()); rel < 0.74 || rel > 0.76 {
		t.Fatalf("cursor should stay at 75%% of the title bar, is at %.2f", rel)
	}
	if cursor.Y-got.Top != 12 {
		t.Fatalf("cursor should stay at the same title-bar height, offset=%d", cursor.Y-got.Top)
	}
}
