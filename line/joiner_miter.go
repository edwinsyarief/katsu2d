package line

import (
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

type MiterJoiner struct{}

// createJoint computes miter join points per side using bisector math.
// Each side is handled independently, so one can bevel while the other miters.
func (self *MiterJoiner) createJoint(
	segment1, segment2 PolySegment,
	end1, end2, nextStart1, nextStart2 *ebimath.Vector,
) {
	// Directions of incoming and outgoing segments
	d1 := segment1.Center.Direction(true).Normalize()
	d2 := segment2.Center.Direction(true).Normalize()

	// Bisector direction (normalized)
	bisector := d1.Add(d2).Normalize()
	if bisector.Length() == 0 {
		// 180° turn — just bevel both sides
		*end1 = segment1.Edge1.B
		*end2 = segment1.Edge2.B
		*nextStart1 = segment2.Edge1.A
		*nextStart2 = segment2.Edge2.A
		return
	}

	// Perpendiculars for each segment (left/right offsets)
	perp1 := ebimath.V(-d1.Y, d1.X)
	perp2 := ebimath.V(-d2.Y, d2.X)

	// --- Side 1 (Edge1 continuity) ---
	hw1 := segment1.Edge1.B.Sub(segment1.Center.B).Length()
	miterDir1 := perp1.Add(perp2).Normalize()
	scale1 := hw1 / miterDir1.Dot(perp1) // miter length factor
	miterPoint1 := segment1.Center.B.Add(miterDir1.ScaleF(scale1))

	if miterPoint1.Sub(segment1.Center.B).Length() > hw1*defaultMiterLimit {
		// Bevel fallback
		*end1 = segment1.Edge1.B
		*nextStart1 = segment2.Edge1.A
	} else {
		*end1 = miterPoint1
		*nextStart1 = miterPoint1
	}

	// --- Side 2 (Edge2 continuity) ---
	hw2 := segment1.Edge2.B.Sub(segment1.Center.B).Length()
	miterDir2 := perp1.ScaleF(-1).Add(perp2.ScaleF(-1)).Normalize()
	scale2 := hw2 / miterDir2.Dot(perp1.ScaleF(-1))
	miterPoint2 := segment1.Center.B.Add(miterDir2.ScaleF(scale2))

	if miterPoint2.Sub(segment1.Center.B).Length() > hw2*defaultMiterLimit {
		// Bevel fallback
		*end2 = segment1.Edge2.B
		*nextStart2 = segment2.Edge2.A
	} else {
		*end2 = miterPoint2
		*nextStart2 = miterPoint2
	}
}

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

		if isFirstSegment {
			start1 = segment.Edge1.A
			start2 = segment.Edge2.A
		} else {
			start1 = nextStart1
			start2 = nextStart2
		}

		if l.IsClosed || !isLastSegment {
			nextSegmentIndex := (i + 1) % len(segments)
			self.createJoint(segment, segments[nextSegmentIndex], &end1, &end2, &nextStart1, &nextStart2)
		} else {
			end1 = segment.Edge1.B
			end2 = segment.Edge2.B
		}

		if l.IsClosed && isFirstSegment {
			var dummyEnd1, dummyEnd2 ebimath.Vector
			prevSegmentIndex := len(segments) - 1
			self.createJoint(segments[prevSegmentIndex], segment, &dummyEnd1, &dummyEnd2, &start1, &start2)
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
	}
	return vertices, indices
}
