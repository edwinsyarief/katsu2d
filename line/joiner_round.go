package line

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// RoundJoiner implements round line joins
type RoundJoiner struct{}

func (self *RoundJoiner) createJoint(vertices *[]ebiten.Vertex, indices *[]uint16,
	segment1, segment2 PolySegment, width1, width2 float64, col color.RGBA, opacity float64,
	end1, end2, nextStart1, nextStart2 *ebimath.Vector, addGeometry bool) {

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

	innerSec, ok := inner1.Intersection(*inner2, true)
	if !ok {
		innerSec = inner1.B
	}

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

	// Correctly pass the start and end radii to the triangle fan creation
	startRadius := width1
	endRadius := width2

	if addGeometry {
		// Create a fan of triangles for the round join
		createTriangleFan(vertices, indices, innerSec, segment1.Center.B, outer1.B, outer2.A, startRadius, endRadius, clockwise, roundJoinSegments, col, opacity)
	}
}

// BuildMesh generates vertices and indices for a round-joined line
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

	// Create PolySegments
	segments := make([]PolySegment, 0)
	for i := 0; i < len(l.points)-1; i++ {
		p1 := l.points[i]
		p2 := l.points[i+1]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2))
		}
	}
	if l.IsClosed && len(l.points) > 1 {
		p1 := l.points[len(l.points)-1]
		p2 := l.points[0]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2))
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

		var p1_idx, p2_idx int
		p1_idx = i
		if i == len(l.points)-1 {
			p2_idx = 0
		} else {
			p2_idx = i + 1
		}

		col_start := l.points[p1_idx].color
		if l.interpolateColor {
			col_start = l.lerpColor(float64(i) / float64(totalSegments))
		}
		col_end := l.points[p2_idx].color
		if l.interpolateColor {
			col_end = l.lerpColor(float64(i+1) / float64(totalSegments))
		}

		width_start := l.points[p1_idx].width
		width_end := l.points[p2_idx].width

		if isFirstSegment {
			start1 = segment.Edge1.A
			start2 = segment.Edge2.A
		} else {
			start1 = nextStart1
			start2 = nextStart2
		}

		if l.IsClosed || !isLastSegment {
			nextSegmentIndex := (i + 1) % len(segments)
			col_join := l.points[p2_idx].color
			if l.interpolateColor {
				col_join = l.lerpColor(float64(i+1) / float64(totalSegments))
			}
			self.createJoint(&vertices, &indices, segment, segments[nextSegmentIndex], width_start/2, width_end/2, col_join, l.opacity, &end1, &end2, &nextStart1, &nextStart2, true)
		} else {
			end1 = segment.Edge1.B
			end2 = segment.Edge2.B
		}

		if l.IsClosed && isFirstSegment {
			var dummyEnd1, dummyEnd2 ebimath.Vector
			prevSegmentIndex := len(segments) - 1
			col_join := l.points[p1_idx].color
			if l.interpolateColor {
				col_join = l.lerpColor(float64(i) / float64(totalSegments))
			}
			self.createJoint(&vertices, &indices, segments[prevSegmentIndex], segment, l.points[len(l.points)-1].width/2, width_start/2, col_join, l.opacity, &dummyEnd1, &dummyEnd2, &start1, &start2, false)
		}

		// Create the quad for the line segment, now using the correct start and end points
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
