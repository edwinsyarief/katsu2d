// joiner_round.go
package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// RoundJoiner implements round line joins.
type RoundJoiner struct{}

// signedAngle returns the minimal signed angle from a to b in (-π, π].
func signedAngle(ax, ay, bx, by float64) float64 {
	// angle = atan2(cross, dot)
	cross := ax*by - ay*bx
	dot := ax*bx + ay*by
	return math.Atan2(cross, dot)
}

func (self *RoundJoiner) createJoint(
	vertices *[]ebiten.Vertex, indices *[]uint16,
	segment1, segment2 PolySegment, col color.RGBA, opacity float64,
	end1, end2, nextStart1, nextStart2 *ebimath.Vector,
	addGeometry bool,
) {
	dir1 := segment1.Center.Direction(true)
	dir2 := segment2.Center.Direction(true)
	clockwise := (dir1.X*dir2.Y - dir1.Y*dir2.X) < 0

	// Choose outer/inner edges based on turn direction.
	var inner1, inner2, outer1, outer2 *LineSegment
	if clockwise {
		outer1, outer2 = &segment1.Edge1, &segment2.Edge1
		inner1, inner2 = &segment1.Edge2, &segment2.Edge2
	} else {
		outer1, outer2 = &segment1.Edge2, &segment2.Edge2
		inner1, inner2 = &segment1.Edge1, &segment2.Edge1
	}

	// Intersection on inner side (apex for our fan).
	innerSec, ok := inner1.Intersection(*inner2, false)
	if !ok {
		// Fallback: pick a stable point near the joint to avoid huge skinny triangles.
		innerSec = inner1.B
	}

	// Stitching points for segment rectangles.
	if clockwise {
		*end1, *end2 = outer1.B, innerSec
		*nextStart1, *nextStart2 = outer2.A, innerSec
	} else {
		*end1, *end2 = innerSec, outer1.B
		*nextStart1, *nextStart2 = innerSec, outer2.A
	}

	if !addGeometry {
		return
	}

	// Build the outer arc between the exact edge endpoints.
	center := segment1.Center.B
	startPt := outer1.B
	endPt := outer2.A

	vsx, vsy := startPt.X-center.X, startPt.Y-center.Y
	vex, vey := endPt.X-center.X, endPt.Y-center.Y

	startR := math.Hypot(vsx, vsy)
	endR := math.Hypot(vex, vey)

	// If nearly colinear, skip geometry (the segment quads meet exactly).
	const eps = 1e-6
	if startR < eps || endR < eps {
		return
	}

	sweep := signedAngle(vsx, vsy, vex, vey) // minimal sweep in (-π, π]
	absSweep := math.Abs(sweep)
	if absSweep < 1e-3 { // ~0.057 degrees
		return
	}

	// Adaptive tessellation: about 11.25° per step.
	steps := int(math.Ceil(absSweep / (math.Pi / 16.0)))
	if steps < 2 {
		steps = 2
	}

	// Generate arc points (including endpoints), radii interpolate for width changes.
	arcStartAngle := math.Atan2(vsy, vsx)
	// We advance from start to end using minimal sweep.
	arcPts := make([]ebimath.Vector, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		angle := arcStartAngle + sweep*t
		r := (1.0-t)*startR + t*endR
		arcPts[i] = ebimath.Vector{
			X: center.X + math.Cos(angle)*r,
			Y: center.Y + math.Sin(angle)*r,
		}
	}

	// Ensure exact endpoints (avoid any rounding drift).
	arcPts[0] = startPt
	arcPts[len(arcPts)-1] = endPt

	// Triangulate the join as a single fan from innerSec to the arc.
	// Maintain consistent winding. We’ll use:
	// - CCW: (innerSec, arc[i], arc[i+1])
	// - CW:  (innerSec, arc[i+1], arc[i])
	ccw := !clockwise

	innerIdx := uint16(len(*vertices))
	*vertices = append(*vertices, utils.CreateVertexWithOpacity(innerSec, ebimath.Vector{}, col, opacity))

	// First arc point
	prevIdx := uint16(len(*vertices))
	*vertices = append(*vertices, utils.CreateVertexWithOpacity(arcPts[0], ebimath.Vector{}, col, opacity))

	for i := 1; i < len(arcPts); i++ {
		currIdx := uint16(len(*vertices))
		*vertices = append(*vertices, utils.CreateVertexWithOpacity(arcPts[i], ebimath.Vector{}, col, opacity))

		if ccw {
			*indices = append(*indices, innerIdx, prevIdx, currIdx)
		} else {
			*indices = append(*indices, innerIdx, currIdx, prevIdx)
		}
		prevIdx = currIdx
	}
}

// BuildMesh generates vertices and indices for a round-joined line.
func (self *RoundJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
	if len(l.points) < 2 {
		return nil, nil
	}

	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	totalSegments := len(l.points) - 1
	if l.IsClosed {
		totalSegments = len(l.points)
	}

	// Create PolySegments with per-end thickness.
	segments := make([]PolySegment, 0)
	for i := 0; i < len(l.points)-1; i++ {
		p1 := l.points[i]
		p2 := l.points[i+1]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(
				p1.position, p2.position,
				p1.width/2, p2.width/2,
			))
		}
	}
	if l.IsClosed && len(l.points) > 1 {
		p1 := l.points[len(l.points)-1]
		p2 := l.points[0]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(
				p1.position, p2.position,
				p1.width/2, p2.width/2,
			))
		}
	}

	if len(segments) == 0 {
		return nil, nil
	}

	var start1, start2 ebimath.Vector
	var end1, end2 ebimath.Vector
	var nextStart1, nextStart2 ebimath.Vector

	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		isFirst := i == 0
		isLast := i == len(segments)-1

		var p1idx, p2idx int
		p1idx = i
		if i == len(l.points)-1 {
			p2idx = 0
		} else {
			p2idx = i + 1
		}

		colStart := l.points[p1idx].color
		if l.interpolateColor {
			colStart = l.lerpColor(float64(i) / float64(totalSegments))
		}
		colEnd := l.points[p2idx].color
		if l.interpolateColor {
			colEnd = l.lerpColor(float64(i+1) / float64(totalSegments))
		}

		if isFirst {
			start1 = segment.Edge1.A
			start2 = segment.Edge2.A
		} else {
			start1 = nextStart1
			start2 = nextStart2
		}

		if l.IsClosed || !isLast {
			nextIdx := (i + 1) % len(segments)
			colJoin := l.points[p2idx].color
			if l.interpolateColor {
				colJoin = l.lerpColor(float64(i+1) / float64(totalSegments))
			}
			self.createJoint(&vertices, &indices, segment, segments[nextIdx], colJoin, l.opacity, &end1, &end2, &nextStart1, &nextStart2, true)
		} else {
			end1 = segment.Edge1.B
			end2 = segment.Edge2.B
		}

		// For closed polylines, precompute the first segment’s start alignment using the last joint.
		if l.IsClosed && isFirst {
			var dummyEnd1, dummyEnd2 ebimath.Vector
			prevIdx := len(segments) - 1
			colJoin := l.points[p1idx].color
			if l.interpolateColor {
				colJoin = l.lerpColor(float64(i) / float64(totalSegments))
			}
			self.createJoint(&vertices, &indices, segments[prevIdx], segment, colJoin, l.opacity, &dummyEnd1, &dummyEnd2, &start1, &start2, false)
		}

		// Segment rectangle (two triangles).
		vStart1 := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(start1, ebimath.Vector{}, colStart, l.opacity))
		vStart2 := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(start2, ebimath.Vector{}, colStart, l.opacity))
		vEnd1 := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(end1, ebimath.Vector{}, colEnd, l.opacity))
		vEnd2 := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(end2, ebimath.Vector{}, colEnd, l.opacity))

		indices = append(indices, vStart1, vStart2, vEnd1)
		indices = append(indices, vEnd1, vStart2, vEnd2)
	}

	return vertices, indices
}
