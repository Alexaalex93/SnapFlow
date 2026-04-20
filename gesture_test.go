package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestGesture_LeftEdgeMorphsToTwoThirdsOnLowerDrag(t *testing.T) {
	g := NewGestureInterpreter(NewEntitlements(true))
	work := w32.RECT{Left: 0, Top: 0, Right: 1200, Bottom: 900}

	g.Begin(1, w32.POINT{X: 8, Y: 450})
	if got := g.Update(work, w32.POINT{X: 8, Y: 450}); got != SnapLeftHalf {
		t.Fatalf("expected left half, got %v", got)
	}
	if got := g.Update(work, w32.POINT{X: 9, Y: 780}); got != SnapLeftTwoThirds {
		t.Fatalf("expected left two-thirds, got %v", got)
	}
}

func TestGesture_FreeTierLocksToHalfOnEdge(t *testing.T) {
	g := NewGestureInterpreter(NewEntitlements(false))
	work := w32.RECT{Left: 0, Top: 0, Right: 1200, Bottom: 900}

	g.Begin(1, w32.POINT{X: 8, Y: 450})
	if got := g.Update(work, w32.POINT{X: 8, Y: 780}); got != SnapLeftHalf {
		t.Fatalf("expected left half in free tier, got %v", got)
	}
}
