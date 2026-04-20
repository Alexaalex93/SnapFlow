package main

import "github.com/gonutz/w32/v2"

type activeEdge int

const (
	edgeNone activeEdge = iota
	edgeLeft
	edgeRight
	edgeTop
	edgeBottom
)

type GestureInterpreter struct {
	entitlements *Entitlements
	state        gestureState
}

type gestureState struct {
	dragging          bool
	hwnd              w32.HWND
	edge              activeEdge
	lastTarget        SnapTarget
	lastCursor        w32.POINT
	initializedCursor bool
}

const (
	edgeThresholdPx      int32 = 36
	cornerThresholdPx    int32 = 52
	hysteresisPaddingPx  int32 = 16
	thirdBandUpperScaled int32 = 36 // 0.36
	thirdBandLowerScaled int32 = 64 // 0.64
)

func NewGestureInterpreter(entitlements *Entitlements) *GestureInterpreter {
	return &GestureInterpreter{entitlements: entitlements}
}

func (g *GestureInterpreter) Begin(hwnd w32.HWND, cursor w32.POINT) {
	g.state = gestureState{
		dragging:          true,
		hwnd:              hwnd,
		edge:              edgeNone,
		lastTarget:        SnapNone,
		lastCursor:        cursor,
		initializedCursor: true,
	}
}

func (g *GestureInterpreter) End() {
	g.state = gestureState{}
}

func (g *GestureInterpreter) CurrentTarget() SnapTarget { return g.state.lastTarget }

func (g *GestureInterpreter) Update(work w32.RECT, cursor w32.POINT) SnapTarget {
	if !g.state.dragging {
		return SnapNone
	}
	dx, dy := int32(0), int32(0)
	if g.state.initializedCursor {
		dx = cursor.X - g.state.lastCursor.X
		dy = cursor.Y - g.state.lastCursor.Y
	}
	g.state.lastCursor = cursor
	g.state.initializedCursor = true

	nextEdge, cornerTarget := detectEdge(work, cursor)
	if cornerTarget != SnapNone {
		g.state.edge = nextEdge
		g.state.lastTarget = cornerTarget
		return cornerTarget
	}
	if nextEdge == edgeNone {
		g.state.edge = edgeNone
		g.state.lastTarget = SnapNone
		return SnapNone
	}

	if g.state.edge != nextEdge && !edgeSwitchAllowed(work, cursor, nextEdge) {
		nextEdge = g.state.edge
	}
	g.state.edge = nextEdge

	target := g.resolveEdgeTarget(work, cursor, dx, dy)
	if g.state.lastTarget != SnapNone && target != g.state.lastTarget {
		if !targetSwitchAllowed(work, cursor, g.state.edge, target, g.state.lastTarget) {
			target = g.state.lastTarget
		}
	}
	g.state.lastTarget = target
	return target
}

func (g *GestureInterpreter) resolveEdgeTarget(work w32.RECT, cursor w32.POINT, dx, dy int32) SnapTarget {
	allowAdvanced := g.entitlements.Enabled(FeatureDynamicMorphing)
	switch g.state.edge {
	case edgeLeft:
		if !allowAdvanced {
			return SnapLeftHalf
		}
		return sideMorphTarget(work, cursor, dx, dy, true)
	case edgeRight:
		if !allowAdvanced {
			return SnapRightHalf
		}
		return sideMorphTarget(work, cursor, dx, dy, false)
	case edgeTop:
		if g.entitlements.Enabled(FeatureThirdsAndTwoThirds) {
			xNorm := scaledAxis(cursor.X-work.Left, work.Width())
			if xNorm < thirdBandUpperScaled {
				return SnapLeftThird
			}
			if xNorm > thirdBandLowerScaled {
				return SnapRightThird
			}
			if abs32(dx) > abs32(dy) {
				return SnapCenterThird
			}
		}
		return SnapTopHalf
	case edgeBottom:
		return SnapBottomHalf
	default:
		return SnapNone
	}
}

func sideMorphTarget(work w32.RECT, cursor w32.POINT, dx, dy int32, left bool) SnapTarget {
	yNorm := scaledAxis(cursor.Y-work.Top, work.Height())
	favorLarge := dy > 0 || (abs32(dx) > abs32(dy) && dx > 0)
	if !left {
		favorLarge = dy < 0 || (abs32(dx) > abs32(dy) && dx < 0)
	}

	if yNorm < thirdBandUpperScaled {
		if left {
			return SnapLeftThird
		}
		return SnapRightThird
	}
	if yNorm > thirdBandLowerScaled || favorLarge {
		if left {
			return SnapLeftTwoThirds
		}
		return SnapRightTwoThirds
	}
	if left {
		return SnapLeftHalf
	}
	return SnapRightHalf
}

func detectEdge(work w32.RECT, cursor w32.POINT) (activeEdge, SnapTarget) {
	nearLeft := cursor.X-work.Left <= edgeThresholdPx
	nearRight := work.Right-cursor.X <= edgeThresholdPx
	nearTop := cursor.Y-work.Top <= edgeThresholdPx
	nearBottom := work.Bottom-cursor.Y <= edgeThresholdPx

	if nearLeft && nearTop && cursor.Y-work.Top <= cornerThresholdPx {
		return edgeLeft, SnapTopLeftQuadrant
	}
	if nearRight && nearTop && cursor.Y-work.Top <= cornerThresholdPx {
		return edgeRight, SnapTopRightQuadrant
	}
	if nearLeft && nearBottom && work.Bottom-cursor.Y <= cornerThresholdPx {
		return edgeLeft, SnapBottomLeftQuadrant
	}
	if nearRight && nearBottom && work.Bottom-cursor.Y <= cornerThresholdPx {
		return edgeRight, SnapBottomRightQuadrant
	}

	switch {
	case nearLeft:
		return edgeLeft, SnapNone
	case nearRight:
		return edgeRight, SnapNone
	case nearTop:
		return edgeTop, SnapNone
	case nearBottom:
		return edgeBottom, SnapNone
	default:
		return edgeNone, SnapNone
	}
}

func edgeSwitchAllowed(work w32.RECT, cursor w32.POINT, to activeEdge) bool {
	switch to {
	case edgeLeft:
		return cursor.X-work.Left <= edgeThresholdPx+hysteresisPaddingPx
	case edgeRight:
		return work.Right-cursor.X <= edgeThresholdPx+hysteresisPaddingPx
	case edgeTop:
		return cursor.Y-work.Top <= edgeThresholdPx+hysteresisPaddingPx
	case edgeBottom:
		return work.Bottom-cursor.Y <= edgeThresholdPx+hysteresisPaddingPx
	default:
		return false
	}
}

func targetSwitchAllowed(work w32.RECT, cursor w32.POINT, edge activeEdge, to, from SnapTarget) bool {
	if edge == edgeNone || to == from {
		return true
	}
	if to == SnapNone {
		return false
	}
	switch edge {
	case edgeLeft, edgeRight:
		return cursor.Y >= work.Top-hysteresisPaddingPx && cursor.Y <= work.Bottom+hysteresisPaddingPx
	case edgeTop, edgeBottom:
		return cursor.X >= work.Left-hysteresisPaddingPx && cursor.X <= work.Right+hysteresisPaddingPx
	default:
		return true
	}
}

func scaledAxis(position, span int32) int32 {
	if span <= 0 {
		return 0
	}
	if position < 0 {
		position = 0
	}
	if position > span {
		position = span
	}
	return (position * 100) / span
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
