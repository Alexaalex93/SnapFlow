package main

import "github.com/gonutz/w32/v2"

// SnapZone maps a rectangular hot-region on screen to a WindowAction.
// Translates Rectangle's SnapArea struct.
type SnapZone struct {
	Region w32.RECT
	Action WindowAction
}

// SnapZonesForWork returns the active snap zones for a monitor work area.
// This translates Rectangle's SnapAreaModel + SnappingManager.directionalLocationOfCursor.
//
// Default landscape layout (matches Rectangle's defaults):
//   - Top edge center  → Maximize
//   - Top-left corner  → Top Left
//   - Top-right corner → Top Right
//   - Bottom-left corner → Bottom Left
//   - Bottom-right corner → Bottom Right
//   - Left edge (upper) → Top Left (or Top-Left Third)
//   - Left edge (middle) → Left Half
//   - Left edge (lower) → Bottom Left (or Bottom-Left Third)
//   - Right edge (upper) → Top Right
//   - Right edge (middle) → Right Half
//   - Right edge (lower) → Bottom Right
//   - Bottom edge → Bottom Half
//
// zoneNameToAction maps config zone name strings to WindowAction constants.
// "compoundThirds" is a UI alias for ActionBottomHalf: when the bottom zone
// uses this value the gesture engine activates the compound-thirds tracker,
// snapping to the 1/3, 2/3 or full-width column the cursor sweeps through.
var zoneNameToAction = map[string]WindowAction{
	// Special / UI aliases
	"none": ActionNone, "compoundThirds": ActionBottomHalf,
	// Halves
	"leftHalf": ActionLeftHalf, "rightHalf": ActionRightHalf,
	"topHalf": ActionTopHalf, "bottomHalf": ActionBottomHalf,
	"centerHalf": ActionCenterHalf,
	// Quarters
	"topLeft": ActionTopLeft, "topRight": ActionTopRight,
	"bottomLeft": ActionBottomLeft, "bottomRight": ActionBottomRight,
	// Horizontal thirds
	"firstThird": ActionFirstThird, "centerThird": ActionCenterThird, "lastThird": ActionLastThird,
	"firstTwoThirds": ActionFirstTwoThirds, "centerTwoThirds": ActionCenterTwoThirds, "lastTwoThirds": ActionLastTwoThirds,
	// Vertical thirds
	"topVerticalThird": ActionTopVerticalThird, "middleVerticalThird": ActionMiddleVerticalThird,
	"bottomVerticalThird":  ActionBottomVerticalThird,
	"topVerticalTwoThirds": ActionTopVerticalTwoThirds, "bottomVerticalTwoThirds": ActionBottomVerticalTwoThirds,
	// Horizontal fourths
	"firstFourth": ActionFirstFourth, "secondFourth": ActionSecondFourth,
	"thirdFourth": ActionThirdFourth, "lastFourth": ActionLastFourth,
	"firstThreeFourths": ActionFirstThreeFourths, "centerThreeFourths": ActionCenterThreeFourths,
	"lastThreeFourths": ActionLastThreeFourths,
	// Sixths (3 cols × 2 rows)
	"topLeftSixth": ActionTopLeftSixth, "topCenterSixth": ActionTopCenterSixth, "topRightSixth": ActionTopRightSixth,
	"bottomLeftSixth": ActionBottomLeftSixth, "bottomCenterSixth": ActionBottomCenterSixth, "bottomRightSixth": ActionBottomRightSixth,
	// Corner thirds (1/3 wide × 1/2 tall)
	"topLeftThird": ActionTopLeftThird, "topRightThird": ActionTopRightThird,
	"bottomLeftThird": ActionBottomLeftThird, "bottomRightThird": ActionBottomRightThird,
	// Maximize family
	"maximize": ActionMaximize, "almostMaximize": ActionAlmostMaximize,
	"maximizeHeight": ActionMaximizeHeight,
	// Center
	"center": ActionCenter, "centerProminently": ActionCenterProminently,
}

func zoneAction(name string) WindowAction {
	if a, ok := zoneNameToAction[name]; ok {
		return a
	}
	return ActionNone
}

func SnapZonesForWork(work w32.RECT, cfg *AppConfig) []SnapZone {
	if !cfg.SnapByDragging {
		return nil
	}
	W, H := work.Width(), work.Height()

	corner := int32(cfg.CornerSnapAreaSize)
	if corner <= 0 {
		corner = 20
	}
	shortEdge := int32(cfg.ShortEdgeSnapAreaSize)
	if shortEdge <= 0 {
		shortEdge = 145
	}
	edgeW := int32(cfg.EdgeThresholdPx)
	if edgeW <= 0 {
		edgeW = 46
	}

	// Clamp so zones never overlap each other badly on small screens.
	if corner*2 > H/2 {
		corner = H / 4
	}
	if corner*2 > W/2 {
		corner = W / 4
	}
	if shortEdge > H/2-corner {
		shortEdge = H/2 - corner
	}

	// ── Corner snap areas ─────────────────────────────────────────────────
	// These are square areas at each corner of the screen.
	topLeftCorner := w32.RECT{
		Left: work.Left, Top: work.Top,
		Right: work.Left + corner, Bottom: work.Top + corner,
	}
	topRightCorner := w32.RECT{
		Left: work.Right - corner, Top: work.Top,
		Right: work.Right, Bottom: work.Top + corner,
	}
	bottomLeftCorner := w32.RECT{
		Left: work.Left, Top: work.Bottom - corner,
		Right: work.Left + corner, Bottom: work.Bottom,
	}
	bottomRightCorner := w32.RECT{
		Left: work.Right - corner, Top: work.Bottom - corner,
		Right: work.Right, Bottom: work.Bottom,
	}

	// ── Top edge ──────────────────────────────────────────────────────────
	// In Rectangle: top edge → Maximize (center portion, excluding corners)
	topEdge := w32.RECT{
		Left: work.Left + corner, Top: work.Top,
		Right: work.Right - corner, Bottom: work.Top + edgeW,
	}

	// ── Left edge: compound halves ────────────────────────────────────────
	// Upper zone → Top Left quadrant
	// Middle zone → Left Half
	// Lower zone → Bottom Left quadrant
	leftEdgeTop := w32.RECT{
		Left: work.Left, Top: work.Top + corner,
		Right: work.Left + edgeW, Bottom: work.Top + corner + shortEdge,
	}
	leftEdgeMid := w32.RECT{
		Left: work.Left, Top: work.Top + corner + shortEdge,
		Right: work.Left + edgeW, Bottom: work.Bottom - corner - shortEdge,
	}
	leftEdgeBot := w32.RECT{
		Left: work.Left, Top: work.Bottom - corner - shortEdge,
		Right: work.Left + edgeW, Bottom: work.Bottom - corner,
	}

	// ── Right edge: compound halves ───────────────────────────────────────
	rightEdgeTop := w32.RECT{
		Left: work.Right - edgeW, Top: work.Top + corner,
		Right: work.Right, Bottom: work.Top + corner + shortEdge,
	}
	rightEdgeMid := w32.RECT{
		Left: work.Right - edgeW, Top: work.Top + corner + shortEdge,
		Right: work.Right, Bottom: work.Bottom - corner - shortEdge,
	}
	rightEdgeBot := w32.RECT{
		Left: work.Right - edgeW, Top: work.Bottom - corner - shortEdge,
		Right: work.Right, Bottom: work.Bottom - corner,
	}

	// ── Bottom edge ───────────────────────────────────────────────────────
	bottomEdge := w32.RECT{
		Left: work.Left + corner, Top: work.Bottom - edgeW,
		Right: work.Right - corner, Bottom: work.Bottom,
	}

	// If the screen is too narrow for the left/right edge zones to have a
	// meaningful middle section, collapse to just the half.
	midH := leftEdgeMid.Height()

	// Read per-zone actions from config (with fallback to defaults).
	aTopLeft := firstNonNone(zoneAction(cfg.ZoneTopLeft), ActionTopLeft)
	aTop := firstNonNone(zoneAction(cfg.ZoneTop), ActionMaximize)
	aTopRight := firstNonNone(zoneAction(cfg.ZoneTopRight), ActionTopRight)
	aLeft := firstNonNone(zoneAction(cfg.ZoneLeft), ActionLeftHalf)
	aRight := firstNonNone(zoneAction(cfg.ZoneRight), ActionRightHalf)
	aBottomLeft := firstNonNone(zoneAction(cfg.ZoneBottomLeft), ActionBottomLeft)
	aBottom := firstNonNone(zoneAction(cfg.ZoneBottom), ActionBottomHalf)
	aBottomRight := firstNonNone(zoneAction(cfg.ZoneBottomRight), ActionBottomRight)

	zones := []SnapZone{}

	// Corners first (highest priority).
	if aTopLeft != ActionNone {
		zones = append(zones, SnapZone{topLeftCorner, aTopLeft})
	}
	if aTopRight != ActionNone {
		zones = append(zones, SnapZone{topRightCorner, aTopRight})
	}
	if aBottomLeft != ActionNone {
		zones = append(zones, SnapZone{bottomLeftCorner, aBottomLeft})
	}
	if aBottomRight != ActionNone {
		zones = append(zones, SnapZone{bottomRightCorner, aBottomRight})
	}

	// Top edge.
	if aTop != ActionNone {
		zones = append(zones, SnapZone{topEdge, aTop})
	}

	// Left edge: compound if action is leftHalf — show TopLeft/LeftHalf/BottomLeft.
	if aLeft != ActionNone {
		if midH > 0 && aLeft == ActionLeftHalf {
			zones = append(zones,
				SnapZone{leftEdgeTop, aTopLeft},
				SnapZone{leftEdgeMid, ActionLeftHalf},
				SnapZone{leftEdgeBot, aBottomLeft},
			)
		} else {
			zones = append(zones, SnapZone{
				w32.RECT{Left: work.Left, Top: work.Top + corner, Right: work.Left + edgeW, Bottom: work.Bottom - corner},
				aLeft,
			})
		}
	}

	// Right edge: compound if rightHalf.
	if aRight != ActionNone {
		if midH > 0 && aRight == ActionRightHalf {
			zones = append(zones,
				SnapZone{rightEdgeTop, aTopRight},
				SnapZone{rightEdgeMid, ActionRightHalf},
				SnapZone{rightEdgeBot, aBottomRight},
			)
		} else {
			zones = append(zones, SnapZone{
				w32.RECT{Left: work.Right - edgeW, Top: work.Top + corner, Right: work.Right, Bottom: work.Bottom - corner},
				aRight,
			})
		}
	}

	// Bottom edge.
	if aBottom != ActionNone {
		zones = append(zones, SnapZone{bottomEdge, aBottom})
	}

	_ = W
	_ = H
	return zones
}

// ActionForCursor returns the WindowAction for the given cursor position,
// or ActionNone if the cursor is not in any snap zone.
func ActionForCursor(cursor w32.POINT, zones []SnapZone) WindowAction {
	for _, z := range zones {
		if pointInRect(cursor, z.Region) {
			return z.Action
		}
	}
	return ActionNone
}

func firstNonNone(a, b WindowAction) WindowAction {
	if a != ActionNone {
		return a
	}
	return b
}

func pointInRect(p w32.POINT, r w32.RECT) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}
