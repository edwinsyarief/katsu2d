package line

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// BevelJoiner implements bevel line joins
type BevelJoiner struct{}

func (self *BevelJoiner) createJoint(vertices *[]ebiten.Vertex, indices *[]uint16,
	segment1, segment2 PolySegment, col color.RGBA, opacity float64,
	end1, end2, nextStart1, nextStart2 *ebimath.Vector) {

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

	// Add bevel triangle
	idx := uint16(len(*vertices))
	*vertices = append(*vertices,
		utils.CreateVertexWithOpacity(outer1.B, ebimath.Vector{}, col, opacity),
		utils.CreateVertexWithOpacity(outer2.A, ebimath.Vector{}, col, opacity),
		utils.CreateVertexWithOpacity(innerSec, ebimath.Vector{}, col, opacity),
	)
	*indices = append(*indices, idx, idx+1, idx+2)
}

// BuildMesh generates vertices and indices for a bevel-joined line
func (self *BevelJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
	if len(l.points) < 2 {
		return nil, nil
	}

	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	totalSegments := len(l.points) - 1

	// Create PolySegments
	segments := make([]PolySegment, 0)
	for i := 0; i < len(l.points)-1; i++ {
		p1 := l.points[i]
		p2 := l.points[i+1]
		if !p1.position.Equals(p2.position) {
			segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2))
		}
	}

	if len(segments) == 0 {
		return nil, nil
	}

	var start1, start2, end1, end2, nextStart1, nextStart2 ebimath.Vector

	pathStart1 := segments[0].Edge1.A
	pathStart2 := segments[0].Edge2.A
	pathEnd1 := segments[len(segments)-1].Edge1.B
	pathEnd2 := segments[len(segments)-1].Edge2.B

	start1 = pathStart1
	start2 = pathStart2

	for i := 0; i < len(segments); i++ {
		segment := segments[i]

		col_start := l.points[i].color
		if l.interpolateColor {
			col_start = l.lerpColor(float64(i) / float64(totalSegments))
		}

		if i < len(segments)-1 {
			col_join := l.points[i+1].color
			if l.interpolateColor {
				col_join = l.lerpColor(float64(i+1) / float64(totalSegments))
			}
			self.createJoint(&vertices, &indices, segment, segments[i+1], col_join, l.opacity, &end1, &end2, &nextStart1, &nextStart2)
		} else {
			end1 = pathEnd1
			end2 = pathEnd2
		}

		var col_end color.RGBA
		if i < len(segments)-1 {
			col_end = l.points[i+1].color
			if l.interpolateColor {
				col_end = l.lerpColor(float64(i+1) / float64(totalSegments))
			}
		} else {
			col_end = l.points[len(l.points)-1].color
			if l.interpolateColor {
				col_end = l.lerpColor(1.0)
			}
		}

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

		start1 = nextStart1
		start2 = nextStart2
	}

	return vertices, indices
}
