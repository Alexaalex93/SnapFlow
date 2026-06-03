package main

import "github.com/gonutz/w32/v2"

// WindowAction translates Rectangle's WindowAction enum.
// Raw values match Rectangle's Int rawValue for config compatibility.
type WindowAction int

const (
	ActionNone WindowAction = -1

	ActionLeftHalf   WindowAction = 0
	ActionRightHalf  WindowAction = 1
	ActionMaximize   WindowAction = 2
	ActionMaximizeHeight WindowAction = 3
	ActionPreviousDisplay WindowAction = 4
	ActionNextDisplay     WindowAction = 5
	ActionLarger      WindowAction = 8
	ActionSmaller     WindowAction = 9
	ActionBottomHalf  WindowAction = 10
	ActionTopHalf     WindowAction = 11
	ActionCenter      WindowAction = 12
	ActionBottomLeft  WindowAction = 13
	ActionBottomRight WindowAction = 14
	ActionTopLeft     WindowAction = 15
	ActionTopRight    WindowAction = 16
	ActionRestore     WindowAction = 19
	ActionFirstThird     WindowAction = 20
	ActionFirstTwoThirds WindowAction = 21
	ActionCenterThird    WindowAction = 22
	ActionLastTwoThirds  WindowAction = 23
	ActionLastThird      WindowAction = 24
	ActionMoveLeft  WindowAction = 25
	ActionMoveRight WindowAction = 26
	ActionMoveUp    WindowAction = 27
	ActionMoveDown  WindowAction = 28
	ActionAlmostMaximize    WindowAction = 29
	ActionCenterHalf        WindowAction = 30
	ActionFirstFourth       WindowAction = 31
	ActionSecondFourth      WindowAction = 32
	ActionThirdFourth       WindowAction = 33
	ActionLastFourth        WindowAction = 34
	ActionFirstThreeFourths WindowAction = 35
	ActionLastThreeFourths  WindowAction = 36
	ActionTopLeftSixth    WindowAction = 37
	ActionTopCenterSixth  WindowAction = 38
	ActionTopRightSixth   WindowAction = 39
	ActionBottomLeftSixth   WindowAction = 40
	ActionBottomCenterSixth WindowAction = 41
	ActionBottomRightSixth  WindowAction = 42
	ActionTopLeftNinth     WindowAction = 45
	ActionTopCenterNinth   WindowAction = 46
	ActionTopRightNinth    WindowAction = 47
	ActionMiddleLeftNinth  WindowAction = 48
	ActionMiddleCenterNinth WindowAction = 49
	ActionMiddleRightNinth  WindowAction = 50
	ActionBottomLeftNinth   WindowAction = 51
	ActionBottomCenterNinth WindowAction = 52
	ActionBottomRightNinth  WindowAction = 53
	ActionTopLeftThird    WindowAction = 54
	ActionTopRightThird   WindowAction = 55
	ActionBottomLeftThird WindowAction = 56
	ActionBottomRightThird WindowAction = 57
	ActionTopLeftEighth        WindowAction = 58
	ActionTopCenterLeftEighth  WindowAction = 59
	ActionTopCenterRightEighth WindowAction = 60
	ActionTopRightEighth       WindowAction = 61
	ActionBottomLeftEighth        WindowAction = 62
	ActionBottomCenterLeftEighth  WindowAction = 63
	ActionBottomCenterRightEighth WindowAction = 64
	ActionBottomRightEighth       WindowAction = 65
	ActionCenterProminently WindowAction = 71
	ActionDoubleHeightUp    WindowAction = 72
	ActionDoubleHeightDown  WindowAction = 73
	ActionDoubleWidthLeft   WindowAction = 74
	ActionDoubleWidthRight  WindowAction = 75
	ActionHalveHeightUp     WindowAction = 76
	ActionHalveHeightDown   WindowAction = 77
	ActionHalveWidthLeft    WindowAction = 78
	ActionHalveWidthRight   WindowAction = 79
	ActionLargerWidth  WindowAction = 80
	ActionSmallerWidth WindowAction = 81
	ActionLargerHeight  WindowAction = 82
	ActionSmallerHeight WindowAction = 83
	ActionCenterTwoThirds   WindowAction = 84
	ActionCenterThreeFourths WindowAction = 85
	ActionTopVerticalThird      WindowAction = 87
	ActionMiddleVerticalThird   WindowAction = 88
	ActionBottomVerticalThird   WindowAction = 89
	ActionTopVerticalTwoThirds  WindowAction = 90
	ActionBottomVerticalTwoThirds WindowAction = 91
	ActionTopLeftTwelfth        WindowAction = 92
	ActionTopCenterLeftTwelfth  WindowAction = 93
	ActionTopCenterRightTwelfth WindowAction = 94
	ActionTopRightTwelfth       WindowAction = 95
	ActionMiddleLeftTwelfth        WindowAction = 96
	ActionMiddleCenterLeftTwelfth  WindowAction = 97
	ActionMiddleCenterRightTwelfth WindowAction = 98
	ActionMiddleRightTwelfth       WindowAction = 99
	ActionBottomLeftTwelfth        WindowAction = 100
	ActionBottomCenterLeftTwelfth  WindowAction = 101
	ActionBottomCenterRightTwelfth WindowAction = 102
	ActionBottomRightTwelfth       WindowAction = 103
	ActionTopLeftSixteenth             WindowAction = 104
	ActionTopCenterLeftSixteenth       WindowAction = 105
	ActionTopCenterRightSixteenth      WindowAction = 106
	ActionTopRightSixteenth            WindowAction = 107
	ActionUpperMiddleLeftSixteenth          WindowAction = 108
	ActionUpperMiddleCenterLeftSixteenth    WindowAction = 109
	ActionUpperMiddleCenterRightSixteenth   WindowAction = 110
	ActionUpperMiddleRightSixteenth         WindowAction = 111
	ActionLowerMiddleLeftSixteenth          WindowAction = 112
	ActionLowerMiddleCenterLeftSixteenth    WindowAction = 113
	ActionLowerMiddleCenterRightSixteenth   WindowAction = 114
	ActionLowerMiddleRightSixteenth         WindowAction = 115
	ActionBottomLeftSixteenth             WindowAction = 116
	ActionBottomCenterLeftSixteenth       WindowAction = 117
	ActionBottomCenterRightSixteenth      WindowAction = 118
	ActionBottomRightSixteenth            WindowAction = 119
	ActionDisplayOne   WindowAction = 120
	ActionDisplayTwo   WindowAction = 121
	ActionDisplayThree WindowAction = 122
	ActionDisplayFour  WindowAction = 123
	ActionDisplayFive  WindowAction = 124
	ActionDisplaySix   WindowAction = 125
	ActionDisplaySeven WindowAction = 126
	ActionDisplayEight WindowAction = 127
	ActionDisplayNine  WindowAction = 128
)

// ── Gap edges ─────────────────────────────────────────────────────────────────

type gapEdge int

const (
	gapEdgeNone   gapEdge = 0
	gapEdgeTop    gapEdge = 1
	gapEdgeBottom gapEdge = 2
	gapEdgeLeft   gapEdge = 4
	gapEdgeRight  gapEdge = 8
)

func applyGaps(r w32.RECT, edges gapEdge, half int32) w32.RECT {
	if half <= 0 {
		return r
	}
	if edges&gapEdgeTop != 0 {
		r.Top += half
	}
	if edges&gapEdgeBottom != 0 {
		r.Bottom -= half
	}
	if edges&gapEdgeLeft != 0 {
		r.Left += half
	}
	if edges&gapEdgeRight != 0 {
		r.Right -= half
	}
	return r
}

// ── Grid helpers ──────────────────────────────────────────────────────────────

func colRect(work w32.RECT, c, totalCols int32) (left, right int32) {
	w := work.Width()
	return work.Left + w*c/totalCols, work.Left + w*(c+1)/totalCols
}

func rowRect(work w32.RECT, r, totalRows int32) (top, bottom int32) {
	h := work.Height()
	return work.Top + h*r/totalRows, work.Top + h*(r+1)/totalRows
}

func gridCell(work w32.RECT, col, row, totalCols, totalRows int32) w32.RECT {
	l, r := colRect(work, col, totalCols)
	t, b := rowRect(work, row, totalRows)
	return w32.RECT{Left: l, Top: t, Right: r, Bottom: b}
}

// ── Calculate ─────────────────────────────────────────────────────────────────

// Calculate returns the target RECT for this action on the given work area.
// cur is the current window rect (used for resize/move actions).
// cfg provides gap and sizing preferences.
func (a WindowAction) Calculate(work, cur w32.RECT, cfg *AppConfig) w32.RECT {
	g := int32(cfg.GapSize) / 2
	W, H := work.Width(), work.Height()
	midX := work.Left + W/2
	midY := work.Top + H/2
	_ = midX
	_ = midY

	switch a {

	// ── Halves ──────────────────────────────────────────────────────────────
	case ActionLeftHalf:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: midX, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeRight, g)
	case ActionRightHalf:
		r := w32.RECT{Left: midX, Top: work.Top, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft, g)
	case ActionTopHalf:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: work.Right, Bottom: midY}
		return applyGaps(r, gapEdgeBottom, g)
	case ActionBottomHalf:
		r := w32.RECT{Left: work.Left, Top: midY, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeTop, g)
	case ActionCenterHalf:
		r := w32.RECT{Left: work.Left + W/4, Top: work.Top, Right: work.Right - W/4, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft|gapEdgeRight, g)

	// ── Quarters ────────────────────────────────────────────────────────────
	case ActionTopLeft:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: midX, Bottom: midY}
		return applyGaps(r, gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRight:
		r := w32.RECT{Left: midX, Top: work.Top, Right: work.Right, Bottom: midY}
		return applyGaps(r, gapEdgeLeft|gapEdgeBottom, g)
	case ActionBottomLeft:
		r := w32.RECT{Left: work.Left, Top: midY, Right: midX, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRight:
		r := w32.RECT{Left: midX, Top: midY, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft|gapEdgeTop, g)

	// ── Horizontal thirds ────────────────────────────────────────────────────
	case ActionFirstThird:
		l, rr := colRect(work, 0, 3)
		r := w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeRight, g)
	case ActionCenterThird:
		l, rr := colRect(work, 1, 3)
		r := w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft|gapEdgeRight, g)
	case ActionLastThird:
		l, rr := colRect(work, 2, 3)
		r := w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft, g)
	case ActionFirstTwoThirds:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: work.Left + W*2/3, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeRight, g)
	case ActionCenterTwoThirds:
		r := w32.RECT{Left: work.Left + W/6, Top: work.Top, Right: work.Left + W*5/6, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft|gapEdgeRight, g)
	case ActionLastTwoThirds:
		r := w32.RECT{Left: work.Left + W/3, Top: work.Top, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft, g)

	// ── Vertical thirds ──────────────────────────────────────────────────────
	case ActionTopVerticalThird:
		t, b := rowRect(work, 0, 3)
		r := w32.RECT{Left: work.Left, Top: t, Right: work.Right, Bottom: b}
		return applyGaps(r, gapEdgeBottom, g)
	case ActionMiddleVerticalThird:
		t, b := rowRect(work, 1, 3)
		r := w32.RECT{Left: work.Left, Top: t, Right: work.Right, Bottom: b}
		return applyGaps(r, gapEdgeTop|gapEdgeBottom, g)
	case ActionBottomVerticalThird:
		t, b := rowRect(work, 2, 3)
		r := w32.RECT{Left: work.Left, Top: t, Right: work.Right, Bottom: b}
		return applyGaps(r, gapEdgeTop, g)
	case ActionTopVerticalTwoThirds:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: work.Right, Bottom: work.Top + H*2/3}
		return applyGaps(r, gapEdgeBottom, g)
	case ActionBottomVerticalTwoThirds:
		r := w32.RECT{Left: work.Left, Top: work.Top + H/3, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeTop, g)

	// ── Fourths ──────────────────────────────────────────────────────────────
	case ActionFirstFourth:
		l, rr := colRect(work, 0, 4)
		return applyGaps(w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}, gapEdgeRight, g)
	case ActionSecondFourth:
		l, rr := colRect(work, 1, 4)
		return applyGaps(w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}, gapEdgeLeft|gapEdgeRight, g)
	case ActionThirdFourth:
		l, rr := colRect(work, 2, 4)
		return applyGaps(w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}, gapEdgeLeft|gapEdgeRight, g)
	case ActionLastFourth:
		l, rr := colRect(work, 3, 4)
		return applyGaps(w32.RECT{Left: l, Top: work.Top, Right: rr, Bottom: work.Bottom}, gapEdgeLeft, g)
	case ActionFirstThreeFourths:
		r := w32.RECT{Left: work.Left, Top: work.Top, Right: work.Left + W*3/4, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeRight, g)
	case ActionCenterThreeFourths:
		r := w32.RECT{Left: work.Left + W/8, Top: work.Top, Right: work.Right - W/8, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft|gapEdgeRight, g)
	case ActionLastThreeFourths:
		r := w32.RECT{Left: work.Left + W/4, Top: work.Top, Right: work.Right, Bottom: work.Bottom}
		return applyGaps(r, gapEdgeLeft, g)

	// ── Sixths (3 cols × 2 rows) ──────────────────────────────────────────
	case ActionTopLeftSixth:
		return applyGaps(gridCell(work, 0, 0, 3, 2), gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterSixth:
		return applyGaps(gridCell(work, 1, 0, 3, 2), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightSixth:
		return applyGaps(gridCell(work, 2, 0, 3, 2), gapEdgeLeft|gapEdgeBottom, g)
	case ActionBottomLeftSixth:
		return applyGaps(gridCell(work, 0, 1, 3, 2), gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterSixth:
		return applyGaps(gridCell(work, 1, 1, 3, 2), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightSixth:
		return applyGaps(gridCell(work, 2, 1, 3, 2), gapEdgeLeft|gapEdgeTop, g)

	// ── Eighths (4 cols × 2 rows) ─────────────────────────────────────────
	case ActionTopLeftEighth:
		return applyGaps(gridCell(work, 0, 0, 4, 2), gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterLeftEighth:
		return applyGaps(gridCell(work, 1, 0, 4, 2), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterRightEighth:
		return applyGaps(gridCell(work, 2, 0, 4, 2), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightEighth:
		return applyGaps(gridCell(work, 3, 0, 4, 2), gapEdgeLeft|gapEdgeBottom, g)
	case ActionBottomLeftEighth:
		return applyGaps(gridCell(work, 0, 1, 4, 2), gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterLeftEighth:
		return applyGaps(gridCell(work, 1, 1, 4, 2), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterRightEighth:
		return applyGaps(gridCell(work, 2, 1, 4, 2), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightEighth:
		return applyGaps(gridCell(work, 3, 1, 4, 2), gapEdgeLeft|gapEdgeTop, g)

	// ── Ninths (3 cols × 3 rows) ──────────────────────────────────────────
	case ActionTopLeftNinth:
		return applyGaps(gridCell(work, 0, 0, 3, 3), gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterNinth:
		return applyGaps(gridCell(work, 1, 0, 3, 3), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightNinth:
		return applyGaps(gridCell(work, 2, 0, 3, 3), gapEdgeLeft|gapEdgeBottom, g)
	case ActionMiddleLeftNinth:
		return applyGaps(gridCell(work, 0, 1, 3, 3), gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionMiddleCenterNinth:
		return applyGaps(gridCell(work, 1, 1, 3, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionMiddleRightNinth:
		return applyGaps(gridCell(work, 2, 1, 3, 3), gapEdgeLeft|gapEdgeTop|gapEdgeBottom, g)
	case ActionBottomLeftNinth:
		return applyGaps(gridCell(work, 0, 2, 3, 3), gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterNinth:
		return applyGaps(gridCell(work, 1, 2, 3, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightNinth:
		return applyGaps(gridCell(work, 2, 2, 3, 3), gapEdgeLeft|gapEdgeTop, g)

	// ── Corner thirds (1/3 wide, 1/2 tall) ───────────────────────────────
	case ActionTopLeftThird:
		return applyGaps(w32.RECT{Left: work.Left, Top: work.Top, Right: work.Left + W/3, Bottom: midY}, gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightThird:
		return applyGaps(w32.RECT{Left: work.Right - W/3, Top: work.Top, Right: work.Right, Bottom: midY}, gapEdgeLeft|gapEdgeBottom, g)
	case ActionBottomLeftThird:
		return applyGaps(w32.RECT{Left: work.Left, Top: midY, Right: work.Left + W/3, Bottom: work.Bottom}, gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightThird:
		return applyGaps(w32.RECT{Left: work.Right - W/3, Top: midY, Right: work.Right, Bottom: work.Bottom}, gapEdgeLeft|gapEdgeTop, g)

	// ── Twelfths (4 cols × 3 rows) ────────────────────────────────────────
	case ActionTopLeftTwelfth:
		return applyGaps(gridCell(work, 0, 0, 4, 3), gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterLeftTwelfth:
		return applyGaps(gridCell(work, 1, 0, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterRightTwelfth:
		return applyGaps(gridCell(work, 2, 0, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightTwelfth:
		return applyGaps(gridCell(work, 3, 0, 4, 3), gapEdgeLeft|gapEdgeBottom, g)
	case ActionMiddleLeftTwelfth:
		return applyGaps(gridCell(work, 0, 1, 4, 3), gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionMiddleCenterLeftTwelfth:
		return applyGaps(gridCell(work, 1, 1, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionMiddleCenterRightTwelfth:
		return applyGaps(gridCell(work, 2, 1, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionMiddleRightTwelfth:
		return applyGaps(gridCell(work, 3, 1, 4, 3), gapEdgeLeft|gapEdgeTop|gapEdgeBottom, g)
	case ActionBottomLeftTwelfth:
		return applyGaps(gridCell(work, 0, 2, 4, 3), gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterLeftTwelfth:
		return applyGaps(gridCell(work, 1, 2, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterRightTwelfth:
		return applyGaps(gridCell(work, 2, 2, 4, 3), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightTwelfth:
		return applyGaps(gridCell(work, 3, 2, 4, 3), gapEdgeLeft|gapEdgeTop, g)

	// ── Sixteenths (4 cols × 4 rows) ─────────────────────────────────────
	case ActionTopLeftSixteenth:
		return applyGaps(gridCell(work, 0, 0, 4, 4), gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterLeftSixteenth:
		return applyGaps(gridCell(work, 1, 0, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopCenterRightSixteenth:
		return applyGaps(gridCell(work, 2, 0, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeBottom, g)
	case ActionTopRightSixteenth:
		return applyGaps(gridCell(work, 3, 0, 4, 4), gapEdgeLeft|gapEdgeBottom, g)
	case ActionUpperMiddleLeftSixteenth:
		return applyGaps(gridCell(work, 0, 1, 4, 4), gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionUpperMiddleCenterLeftSixteenth:
		return applyGaps(gridCell(work, 1, 1, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionUpperMiddleCenterRightSixteenth:
		return applyGaps(gridCell(work, 2, 1, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionUpperMiddleRightSixteenth:
		return applyGaps(gridCell(work, 3, 1, 4, 4), gapEdgeLeft|gapEdgeTop|gapEdgeBottom, g)
	case ActionLowerMiddleLeftSixteenth:
		return applyGaps(gridCell(work, 0, 2, 4, 4), gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionLowerMiddleCenterLeftSixteenth:
		return applyGaps(gridCell(work, 1, 2, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionLowerMiddleCenterRightSixteenth:
		return applyGaps(gridCell(work, 2, 2, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop|gapEdgeBottom, g)
	case ActionLowerMiddleRightSixteenth:
		return applyGaps(gridCell(work, 3, 2, 4, 4), gapEdgeLeft|gapEdgeTop|gapEdgeBottom, g)
	case ActionBottomLeftSixteenth:
		return applyGaps(gridCell(work, 0, 3, 4, 4), gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterLeftSixteenth:
		return applyGaps(gridCell(work, 1, 3, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomCenterRightSixteenth:
		return applyGaps(gridCell(work, 2, 3, 4, 4), gapEdgeLeft|gapEdgeRight|gapEdgeTop, g)
	case ActionBottomRightSixteenth:
		return applyGaps(gridCell(work, 3, 3, 4, 4), gapEdgeLeft|gapEdgeTop, g)

	// ── Maximize family ───────────────────────────────────────────────────
	case ActionMaximize:
		return work // SW_MAXIMIZE handles this, but return work for preview

	case ActionAlmostMaximize:
		mw := float64(cfg.AlmostMaximizeWidth)
		mh := float64(cfg.AlmostMaximizeHeight)
		if mw <= 0 {
			mw = 0.9
		}
		if mh <= 0 {
			mh = 0.9
		}
		nw := int32(float64(W) * mw)
		nh := int32(float64(H) * mh)
		return w32.RECT{
			Left:   work.Left + (W-nw)/2,
			Top:    work.Top + (H-nh)/2,
			Right:  work.Left + (W-nw)/2 + nw,
			Bottom: work.Top + (H-nh)/2 + nh,
		}

	case ActionMaximizeHeight:
		return w32.RECT{Left: cur.Left, Top: work.Top, Right: cur.Right, Bottom: work.Bottom}

	// ── Center ────────────────────────────────────────────────────────────
	case ActionCenter:
		return centerRect(work, cur)

	case ActionCenterProminently:
		nw := W * 3 / 4
		nh := H * 3 / 4
		return w32.RECT{
			Left:   work.Left + (W-nw)/2,
			Top:    work.Top + (H-nh)/2,
			Right:  work.Left + (W-nw)/2 + nw,
			Bottom: work.Top + (H-nh)/2 + nh,
		}

	// ── Resize (larger/smaller) ───────────────────────────────────────────
	case ActionLarger:
		step := int32(cfg.SizeOffset)
		if step <= 0 {
			step = 30
		}
		return clampToWork(w32.RECT{
			Left:   cur.Left - step,
			Top:    cur.Top - step,
			Right:  cur.Right + step,
			Bottom: cur.Bottom + step,
		}, work)

	case ActionSmaller:
		step := int32(cfg.SizeOffset)
		if step <= 0 {
			step = 30
		}
		minW, minH := int32(100), int32(100)
		nw := cur.Width() - step*2
		nh := cur.Height() - step*2
		if nw < minW {
			nw = minW
		}
		if nh < minH {
			nh = minH
		}
		cx := cur.Left + cur.Width()/2
		cy := cur.Top + cur.Height()/2
		return w32.RECT{Left: cx - nw/2, Top: cy - nh/2, Right: cx + nw/2, Bottom: cy + nh/2}

	case ActionLargerWidth:
		step := int32(cfg.WidthStepSize)
		if step <= 0 {
			step = 30
		}
		return clampToWork(w32.RECT{Left: cur.Left - step, Top: cur.Top, Right: cur.Right + step, Bottom: cur.Bottom}, work)

	case ActionSmallerWidth:
		step := int32(cfg.WidthStepSize)
		if step <= 0 {
			step = 30
		}
		nw := cur.Width() - step*2
		if nw < 100 {
			nw = 100
		}
		cx := cur.Left + cur.Width()/2
		return w32.RECT{Left: cx - nw/2, Top: cur.Top, Right: cx + nw/2, Bottom: cur.Bottom}

	case ActionLargerHeight:
		step := int32(cfg.SizeOffset)
		if step <= 0 {
			step = 30
		}
		return clampToWork(w32.RECT{Left: cur.Left, Top: cur.Top - step, Right: cur.Right, Bottom: cur.Bottom + step}, work)

	case ActionSmallerHeight:
		step := int32(cfg.SizeOffset)
		if step <= 0 {
			step = 30
		}
		nh := cur.Height() - step*2
		if nh < 100 {
			nh = 100
		}
		cy := cur.Top + cur.Height()/2
		return w32.RECT{Left: cur.Left, Top: cy - nh/2, Right: cur.Right, Bottom: cy + nh/2}

	// ── Double / Halve dimensions ─────────────────────────────────────────
	case ActionDoubleHeightUp:
		nh := cur.Height() * 2
		return clampToWork(w32.RECT{Left: cur.Left, Top: cur.Bottom - nh, Right: cur.Right, Bottom: cur.Bottom}, work)
	case ActionDoubleHeightDown:
		nh := cur.Height() * 2
		return clampToWork(w32.RECT{Left: cur.Left, Top: cur.Top, Right: cur.Right, Bottom: cur.Top + nh}, work)
	case ActionDoubleWidthLeft:
		nw := cur.Width() * 2
		return clampToWork(w32.RECT{Left: cur.Right - nw, Top: cur.Top, Right: cur.Right, Bottom: cur.Bottom}, work)
	case ActionDoubleWidthRight:
		nw := cur.Width() * 2
		return clampToWork(w32.RECT{Left: cur.Left, Top: cur.Top, Right: cur.Left + nw, Bottom: cur.Bottom}, work)
	case ActionHalveHeightUp:
		nh := cur.Height() / 2
		return w32.RECT{Left: cur.Left, Top: cur.Bottom - nh, Right: cur.Right, Bottom: cur.Bottom}
	case ActionHalveHeightDown:
		nh := cur.Height() / 2
		return w32.RECT{Left: cur.Left, Top: cur.Top, Right: cur.Right, Bottom: cur.Top + nh}
	case ActionHalveWidthLeft:
		nw := cur.Width() / 2
		return w32.RECT{Left: cur.Right - nw, Top: cur.Top, Right: cur.Right, Bottom: cur.Bottom}
	case ActionHalveWidthRight:
		nw := cur.Width() / 2
		return w32.RECT{Left: cur.Left, Top: cur.Top, Right: cur.Left + nw, Bottom: cur.Bottom}

	// ── Move to edge ──────────────────────────────────────────────────────
	case ActionMoveLeft:
		return w32.RECT{Left: work.Left, Top: cur.Top, Right: work.Left + cur.Width(), Bottom: cur.Bottom}
	case ActionMoveRight:
		return w32.RECT{Left: work.Right - cur.Width(), Top: cur.Top, Right: work.Right, Bottom: cur.Bottom}
	case ActionMoveUp:
		return w32.RECT{Left: cur.Left, Top: work.Top, Right: cur.Right, Bottom: work.Top + cur.Height()}
	case ActionMoveDown:
		return w32.RECT{Left: cur.Left, Top: work.Bottom - cur.Height(), Right: cur.Right, Bottom: work.Bottom}
	}

	return work
}

func clampToWork(r, work w32.RECT) w32.RECT {
	if r.Left < work.Left {
		r.Right += work.Left - r.Left
		r.Left = work.Left
	}
	if r.Right > work.Right {
		r.Left -= r.Right - work.Right
		r.Right = work.Right
	}
	if r.Top < work.Top {
		r.Bottom += work.Top - r.Top
		r.Top = work.Top
	}
	if r.Bottom > work.Bottom {
		r.Top -= r.Bottom - work.Bottom
		r.Bottom = work.Bottom
	}
	return r
}

// IsDragSnappable returns true if this action can be triggered by drag-to-snap.
// Matches Rectangle's isDragSnappable property.
func (a WindowAction) IsDragSnappable() bool {
	switch a {
	case ActionRestore, ActionPreviousDisplay, ActionNextDisplay,
		ActionMoveUp, ActionMoveDown, ActionMoveLeft, ActionMoveRight,
		ActionLarger, ActionSmaller, ActionLargerWidth, ActionSmallerWidth,
		ActionTopLeftNinth, ActionTopCenterNinth, ActionTopRightNinth,
		ActionMiddleLeftNinth, ActionMiddleCenterNinth, ActionMiddleRightNinth,
		ActionBottomLeftNinth, ActionBottomCenterNinth, ActionBottomRightNinth,
		ActionTopLeftThird, ActionTopRightThird, ActionBottomLeftThird, ActionBottomRightThird,
		ActionDisplayOne, ActionDisplayTwo, ActionDisplayThree, ActionDisplayFour,
		ActionDisplayFive, ActionDisplaySix, ActionDisplaySeven, ActionDisplayEight, ActionDisplayNine:
		return false
	}
	return true
}

// DisplayName returns the human-readable label for this action.
func (a WindowAction) DisplayName() string {
	names := map[WindowAction]string{
		ActionLeftHalf: "Left Half", ActionRightHalf: "Right Half",
		ActionTopHalf: "Top Half", ActionBottomHalf: "Bottom Half",
		ActionCenterHalf: "Center Half",
		ActionTopLeft: "Top Left", ActionTopRight: "Top Right",
		ActionBottomLeft: "Bottom Left", ActionBottomRight: "Bottom Right",
		ActionFirstThird: "First Third", ActionCenterThird: "Center Third",
		ActionLastThird: "Last Third", ActionFirstTwoThirds: "First Two Thirds",
		ActionCenterTwoThirds: "Center Two Thirds", ActionLastTwoThirds: "Last Two Thirds",
		ActionTopVerticalThird: "Top Third", ActionMiddleVerticalThird: "Middle Third",
		ActionBottomVerticalThird: "Bottom Third",
		ActionTopVerticalTwoThirds: "Top Two Thirds", ActionBottomVerticalTwoThirds: "Bottom Two Thirds",
		ActionFirstFourth: "First Fourth", ActionSecondFourth: "Second Fourth",
		ActionThirdFourth: "Third Fourth", ActionLastFourth: "Last Fourth",
		ActionFirstThreeFourths: "First 3/4", ActionCenterThreeFourths: "Center 3/4", ActionLastThreeFourths: "Last 3/4",
		ActionTopLeftSixth: "Top Left Sixth", ActionTopCenterSixth: "Top Center Sixth", ActionTopRightSixth: "Top Right Sixth",
		ActionBottomLeftSixth: "Bottom Left Sixth", ActionBottomCenterSixth: "Bottom Center Sixth", ActionBottomRightSixth: "Bottom Right Sixth",
		ActionTopLeftEighth: "Top Left Eighth", ActionTopCenterLeftEighth: "Top Center-Left Eighth",
		ActionTopCenterRightEighth: "Top Center-Right Eighth", ActionTopRightEighth: "Top Right Eighth",
		ActionBottomLeftEighth: "Bottom Left Eighth", ActionBottomCenterLeftEighth: "Bottom Center-Left Eighth",
		ActionBottomCenterRightEighth: "Bottom Center-Right Eighth", ActionBottomRightEighth: "Bottom Right Eighth",
		ActionTopLeftNinth: "Top Left Ninth", ActionTopCenterNinth: "Top Center Ninth", ActionTopRightNinth: "Top Right Ninth",
		ActionMiddleLeftNinth: "Middle Left Ninth", ActionMiddleCenterNinth: "Middle Center Ninth", ActionMiddleRightNinth: "Middle Right Ninth",
		ActionBottomLeftNinth: "Bottom Left Ninth", ActionBottomCenterNinth: "Bottom Center Ninth", ActionBottomRightNinth: "Bottom Right Ninth",
		ActionTopLeftTwelfth: "Top Left Twelfth", ActionTopCenterLeftTwelfth: "Top Center-Left Twelfth",
		ActionTopCenterRightTwelfth: "Top Center-Right Twelfth", ActionTopRightTwelfth: "Top Right Twelfth",
		ActionMiddleLeftTwelfth: "Middle Left Twelfth", ActionMiddleCenterLeftTwelfth: "Middle Center-Left Twelfth",
		ActionMiddleCenterRightTwelfth: "Middle Center-Right Twelfth", ActionMiddleRightTwelfth: "Middle Right Twelfth",
		ActionBottomLeftTwelfth: "Bottom Left Twelfth", ActionBottomCenterLeftTwelfth: "Bottom Center-Left Twelfth",
		ActionBottomCenterRightTwelfth: "Bottom Center-Right Twelfth", ActionBottomRightTwelfth: "Bottom Right Twelfth",
		ActionTopLeftSixteenth: "Top Left Sixteenth", ActionTopCenterLeftSixteenth: "Top Center-Left Sixteenth",
		ActionTopCenterRightSixteenth: "Top Center-Right Sixteenth", ActionTopRightSixteenth: "Top Right Sixteenth",
		ActionUpperMiddleLeftSixteenth: "Upper-Mid Left Sixteenth", ActionUpperMiddleCenterLeftSixteenth: "Upper-Mid Center-Left Sixteenth",
		ActionUpperMiddleCenterRightSixteenth: "Upper-Mid Center-Right Sixteenth", ActionUpperMiddleRightSixteenth: "Upper-Mid Right Sixteenth",
		ActionLowerMiddleLeftSixteenth: "Lower-Mid Left Sixteenth", ActionLowerMiddleCenterLeftSixteenth: "Lower-Mid Center-Left Sixteenth",
		ActionLowerMiddleCenterRightSixteenth: "Lower-Mid Center-Right Sixteenth", ActionLowerMiddleRightSixteenth: "Lower-Mid Right Sixteenth",
		ActionBottomLeftSixteenth: "Bottom Left Sixteenth", ActionBottomCenterLeftSixteenth: "Bottom Center-Left Sixteenth",
		ActionBottomCenterRightSixteenth: "Bottom Center-Right Sixteenth", ActionBottomRightSixteenth: "Bottom Right Sixteenth",
		ActionMaximize: "Maximize", ActionAlmostMaximize: "Almost Maximize", ActionMaximizeHeight: "Maximize Height",
		ActionCenter: "Center", ActionCenterProminently: "Center Prominently",
		ActionLarger: "Larger", ActionSmaller: "Smaller",
		ActionLargerWidth: "Larger Width", ActionSmallerWidth: "Smaller Width",
		ActionLargerHeight: "Larger Height", ActionSmallerHeight: "Smaller Height",
		ActionDoubleHeightUp: "Double Height Up", ActionDoubleHeightDown: "Double Height Down",
		ActionDoubleWidthLeft: "Double Width Left", ActionDoubleWidthRight: "Double Width Right",
		ActionHalveHeightUp: "Halve Height Up", ActionHalveHeightDown: "Halve Height Down",
		ActionHalveWidthLeft: "Halve Width Left", ActionHalveWidthRight: "Halve Width Right",
		ActionMoveLeft: "Move Left", ActionMoveRight: "Move Right",
		ActionMoveUp: "Move Up", ActionMoveDown: "Move Down",
		ActionNextDisplay: "Next Display", ActionPreviousDisplay: "Previous Display",
		ActionDisplayOne: "Display 1", ActionDisplayTwo: "Display 2", ActionDisplayThree: "Display 3",
		ActionDisplayFour: "Display 4", ActionDisplayFive: "Display 5",
		ActionRestore: "Restore",
	}
	if n, ok := names[a]; ok {
		return n
	}
	return ""
}
