package line

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// MiterJoiner implements miter line joins
type MiterJoiner struct{}

func (self *MiterJoiner) createJoint(segment1, segment2 PolySegment, end1, end2, nextStart1, nextStart2 *ebimath.Vector) {

	dir1 := segment1.Center.Direction(true)
	dir2 := segment2.Center.Direction(true)
	perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
	perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

	miter := perp1.Add(perp2)
	miterLength := miter.Length()

	var miterVector ebimath.Vector
	if miterLength == 0 {
		// Lines are parallel, use first perpendicular
		miterVector = perp1.ScaleF(segment1.Edge1.A.Sub(segment1.Center.A).Length())
	} else {
		dot := dir1.Dot(dir2)
		// Check for potential division by zero
		if dot < -0.9999 {
			dot = -0.9999
		} else if dot > 0.9999 {
			dot = 0.9999
		}
		cosHalfAngle := math.Sqrt((dot + 1.0) / 2.0)
		miterScale := segment1.Edge1.A.Sub(segment1.Center.A).Length() / cosHalfAngle

		// Apply miter limit
		if miterScale > segment1.Edge1.A.Sub(segment1.Center.A).Length()*defaultMiterLimit {
			miterScale = segment1.Edge1.A.Sub(segment1.Center.A).Length() * defaultMiterLimit
		}

		miterVector = miter.Normalized().ScaleF(miterScale)
	}

	if segment1.Center.Normal().Dot(miterVector) > 0 {
		*end1 = segment1.Center.B.Add(miterVector)
		*end2 = segment1.Center.B.Sub(miterVector)
		*nextStart1 = segment2.Center.A.Add(miterVector)
		*nextStart2 = segment2.Center.A.Sub(miterVector)
	} else {
		*end1 = segment1.Center.B.Sub(miterVector)
		*end2 = segment1.Center.B.Add(miterVector)
		*nextStart1 = segment2.Center.A.Sub(miterVector)
		*nextStart2 = segment2.Center.A.Add(miterVector)
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

	// End caps
	start1 = segments[0].Edge1.A
	start2 = segments[0].Edge2.A
	end1 = segments[len(segments)-1].Edge1.B
	end2 = segments[len(segments)-1].Edge2.B

	// Loop through each segment to build the mesh
	for i := 0; i < len(segments); i++ {
		segment := segments[i]

		col_start := l.points[i].color
		if l.interpolateColor {
			col_start = l.lerpColor(float64(i) / float64(totalSegments))
		}
		col_end := l.points[i+1].color
		if l.interpolateColor {
			col_end = l.lerpColor(float64(i+1) / float64(totalSegments))
		}

		if i < len(segments)-1 {
			self.createJoint(segment, segments[i+1], &end1, &end2, &nextStart1, &nextStart2)
		} else {
			end1 = segments[i].Edge1.B
			end2 = segments[i].Edge2.B
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

		start1 = nextStart1
		start2 = nextStart2
	}
	return vertices, indices
}
