package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestSnapEngineBounds_CenterThird(t *testing.T) {
	engine := NewSnapEngine(NewEntitlements(true))
	work := w32.RECT{Left: 0, Top: 0, Right: 300, Bottom: 900}
	b := engine.Bounds(work, SnapCenterThird)
	if b.Left != 100 || b.Right != 200 || b.Top != 0 || b.Bottom != 900 {
		t.Fatalf("unexpected bounds: %#v", b)
	}
}

func TestSnapEngineAllowed_ProOnly(t *testing.T) {
	free := NewSnapEngine(NewEntitlements(false))
	pro := NewSnapEngine(NewEntitlements(true))
	if free.Allowed(SnapLeftThird) {
		t.Fatalf("free tier should not allow thirds")
	}
	if !pro.Allowed(SnapLeftThird) {
		t.Fatalf("pro tier should allow thirds")
	}
}
