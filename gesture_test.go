package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestGesture_LeftEdge(t *testing.T) {
	cfg := defaultConfig()
	g := NewGestureInterpreter(cfg)
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	g.Begin(1, work)
	action := g.Update(w32.POINT{X: 5, Y: 540}, work)
	if action == ActionNone {
		t.Error("expected snap action at left edge, got ActionNone")
	}
}

func TestGesture_TopLeftCorner(t *testing.T) {
	cfg := defaultConfig()
	g := NewGestureInterpreter(cfg)
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	g.Begin(1, work)
	action := g.Update(w32.POINT{X: 5, Y: 5}, work)
	if action != ActionTopLeft {
		t.Errorf("expected ActionTopLeft at top-left corner, got %d", action)
	}
}

func TestGesture_TopEdge_Maximize(t *testing.T) {
	cfg := defaultConfig()
	g := NewGestureInterpreter(cfg)
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	g.Begin(1, work)
	action := g.Update(w32.POINT{X: 960, Y: 2}, work)
	if action != ActionMaximize {
		t.Errorf("expected ActionMaximize at top center, got %d", action)
	}
}

func TestGesture_NoEdge(t *testing.T) {
	cfg := defaultConfig()
	g := NewGestureInterpreter(cfg)
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	g.Begin(1, work)
	action := g.Update(w32.POINT{X: 960, Y: 540}, work)
	if action != ActionNone {
		t.Errorf("expected ActionNone in center, got %d", action)
	}
}

func TestGesture_LeftEdgeMiddle_LeftHalf(t *testing.T) {
	cfg := defaultConfig()
	g := NewGestureInterpreter(cfg)
	work := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	g.Begin(1, work)
	// Middle of left edge → LeftHalf
	action := g.Update(w32.POINT{X: 5, Y: 540}, work)
	if action != ActionLeftHalf {
		t.Errorf("expected ActionLeftHalf at left edge middle, got %d", action)
	}
}
