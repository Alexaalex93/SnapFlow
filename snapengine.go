package main

import "github.com/gonutz/w32/v2"

type SnapTarget int

const (
	SnapNone SnapTarget = iota
	SnapLeftHalf
	SnapRightHalf
	SnapTopHalf
	SnapBottomHalf
	SnapTopLeftQuadrant
	SnapTopRightQuadrant
	SnapBottomLeftQuadrant
	SnapBottomRightQuadrant
	SnapLeftThird
	SnapCenterThird
	SnapRightThird
	SnapLeftTwoThirds
	SnapRightTwoThirds
)

func (t SnapTarget) ProOnly() bool {
	switch t {
	case SnapLeftThird, SnapCenterThird, SnapRightThird, SnapLeftTwoThirds, SnapRightTwoThirds:
		return true
	default:
		return false
	}
}

type SnapEngine struct {
	entitlements *Entitlements
}

func NewSnapEngine(entitlements *Entitlements) *SnapEngine {
	return &SnapEngine{entitlements: entitlements}
}

func (s *SnapEngine) Allowed(t SnapTarget) bool {
	if t == SnapNone {
		return false
	}
	if t.ProOnly() {
		return s.entitlements.Enabled(FeatureThirdsAndTwoThirds)
	}
	return true
}

func (s *SnapEngine) Bounds(work w32.RECT, t SnapTarget) w32.RECT {
	switch t {
	case SnapLeftHalf:
		return toLeft(work, 1, 2)
	case SnapRightHalf:
		return toRight(work, 1, 2)
	case SnapTopHalf:
		return toTop(work, 1, 2)
	case SnapBottomHalf:
		return toBottom(work, 1, 2)
	case SnapTopLeftQuadrant:
		return merge(toLeft(work, 1, 2), toTop(work, 1, 2))
	case SnapTopRightQuadrant:
		return merge(toRight(work, 1, 2), toTop(work, 1, 2))
	case SnapBottomLeftQuadrant:
		return merge(toLeft(work, 1, 2), toBottom(work, 1, 2))
	case SnapBottomRightQuadrant:
		return merge(toRight(work, 1, 2), toBottom(work, 1, 2))
	case SnapLeftThird:
		return toLeft(work, 1, 3)
	case SnapCenterThird:
		return w32.RECT{Left: work.Left + work.Width()/3, Top: work.Top, Right: work.Left + (work.Width()*2)/3, Bottom: work.Bottom}
	case SnapRightThird:
		return toRight(work, 1, 3)
	case SnapLeftTwoThirds:
		return toLeft(work, 2, 3)
	case SnapRightTwoThirds:
		return toRight(work, 2, 3)
	default:
		return work
	}
}

func (s *SnapEngine) Apply(hwnd w32.HWND, t SnapTarget) (bool, error) {
	if !s.Allowed(t) {
		return false, nil
	}
	return resize(hwnd, func(disp, _ w32.RECT) w32.RECT { return s.Bounds(disp, t) })
}
