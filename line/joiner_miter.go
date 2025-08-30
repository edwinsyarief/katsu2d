// joiner_miter.go
package line

import (
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// MiterJoiner implements miter line joins
type MiterJoiner struct{}

// createJoint computes the end/start edges at the corner between segment1 -> segment2.
// It computes the outer and inner intersections. If the outer miter exceeds the miter limit,
// it gracefully falls back to a bevel-like corner to avoid spikes.
func (self *MiterJoiner) createJoint(
	segment1, segment2 PolySegment,
	end1, end2, nextStart1, nextStart2 *ebimath.Vector,
) {
	dir1 := segment1.Center.Direction(true)
	dir2 := segment2.Center.Direction(true)
	clockwise := (dir1.X*dir2.Y - dir1.Y*dir2.X) < 0

	var inner1, inner2, outer1, outer2 *LineSegment
	if clockwise {
		outer1 = &segment1.Edge1
		outer2 = &segment2.Edge1
		inner1 = &segment1.Edge2
		inner2 = &segment2.Edge2
	} else {
		outer1 = &segment1.Edge2
		outer2 = &segment2.Edge2
		inner1 = &segment1.Edge1
		inner2 = &segment2.Edge1
	}

	// Intersections on inner and outer sides
	innerSec, okInner := inner1.Intersection(*inner2, false)
	if !okInner {
		innerSec = inner1.B
	}
	outerSec, okOuter := outer1.Intersection(*outer2, true)
	if !okOuter {
		outerSec = outer1.B
	}

	// Miter limit check based on the radius at the joint (end of segment1)
	radiusAtJoint := segment1.Edge1.B.Sub(segment1.Center.B).Length()
	maxMiter := radiusAtJoint * defaultMiterLimit
	outDist := outerSec.Sub(segment1.Center.B).Length()

	useBevelFallback := !okOuter || outDist > maxMiter

	if useBevelFallback {
		// Bevel-like fallback (same layout as BevelJoiner)
		if clockwise {
			*end1 = outer1.B
			*end2 = innerSec
			*nextStart1 = outer2.A
			*nextStart2 = innerSec
		} else {
			*end1 = innerSec
			*end2 = outer1.B
			*nextStart1 = innerSec
			*nextStart2 = outer2.A
		}
		return
	}

	// True miter: use the outer intersection for the outer edge, inner intersection for the inner edge
	if clockwise {
		*end1 = outerSec
		*end2 = innerSec
		*nextStart1 = outerSec
		*nextStart2 = innerSec
	} else {
		*end1 = innerSec
		*end2 = outerSec
		*nextStart1 = innerSec
		*nextStart2 = outerSec
	}
}

// BuildMesh generates vertices and indices for a miter-joined line
func (self *MiterJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
	if len(l.points) < 2 {
		return nil, nil
	}

	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	totalSegments := len(l.points) - 1
	if l.IsClosed {
		totalSegments = len(l.points)
	}

	// Create PolySegments with per-end thickness
	segments := make([]PolySegment, 0)
	for i := 0; i < len(l.points)-1; i++ {
		p1 := l.points[i]
		p2 := l.points[i+1]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2, p2.width/2))
		}
	}
	if l.IsClosed && len(l.points) > 1 {
		p1 := l.points[len(l.points)-1]
		p2 := l.points[0]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2, p2.width/2))
		}
	}

	if len(segments) == 0 {
		return nil, nil
	}

	var start1, start2, end1, end2, nextStart1, nextStart2 ebimath.Vector

	// Loop through each segment to build the mesh
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		isFirstSegment := i == 0
		isLastSegment := i == len(segments)-1

		// Map segment index to point indices (assumes non-degenerate point ordering)
		var p1_idx, p2_idx int
		p1_idx = i
		if i == len(l.points)-1 {
			p2_idx = 0
		} else {
			p2_idx = i + 1
		}

		// Colors (with optional interpolation)
		col_start := l.points[p1_idx].color
		if l.interpolateColor {
			col_start = l.lerpColor(float64(i) / float64(totalSegments))
		}
		col_end := l.points[p2_idx].color
		if l.interpolateColor {
			col_end = l.lerpColor(float64(i+1) / float64(totalSegments))
		}

		// Determine start edge of this segment
		if isFirstSegment {
			start1 = segment.Edge1.A
			start2 = segment.Edge2.A
		} else {
			start1 = nextStart1
			start2 = nextStart2
		}

		// Determine end edge via joint with next segment (or the raw end if last and open)
		if l.IsClosed || !isLastSegment {
			nextSegmentIndex := (i + 1) % len(segments)
			self.createJoint(segment, segments[nextSegmentIndex], &end1, &end2, &nextStart1, &nextStart2)
		} else {
			end1 = segment.Edge1.B
			end2 = segment.Edge2.B
		}

		// If closed, fix the very first start edge by "joining" previous to current without adding geometry
		if l.IsClosed && isFirstSegment {
			var dummyEnd1, dummyEnd2 ebimath.Vector
			prevSegmentIndex := len(segments) - 1
			self.createJoint(segments[prevSegmentIndex], segment, &dummyEnd1, &dummyEnd2, &start1, &start2)
		}

		// Create the quad for the line segment
		v_start1_idx := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(start1, ebimath.Vector{}, col_start, l.opacity))
		v_start2_idx := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(start2, ebimath.Vector{}, col_start, l.opacity))
		v_end1_idx := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(end1, ebimath.Vector{}, col_end, l.opacity))
		v_end2_idx := uint16(len(vertices))
		vertices = append(vertices, utils.CreateVertexWithOpacity(end2, ebimath.Vector{}, col_end, l.opacity))

		indices = append(indices, v_start1_idx, v_start2_idx, v_end1_idx)
		indices = append(indices, v_end1_idx, v_start2_idx, v_end2_idx)
	}
	return vertices, indices
}
