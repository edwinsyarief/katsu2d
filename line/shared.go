package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// LineSegment represents a line segment with a start and end point.
type LineSegment struct {
	A, B ebimath.Vector
}

func (ls *LineSegment) Normal() ebimath.Vector {
	dir := ls.Direction(true)
	return ebimath.V(-dir.Y, dir.X)
}

func (ls *LineSegment) Direction(normalized bool) ebimath.Vector {
	vec := ls.B.Sub(ls.A)
	if normalized {
		return vec.Normalized()
	}
	return vec
}

// Intersection calculates the intersection point of two line segments.
func (ls *LineSegment) Intersection(other LineSegment, infiniteLines bool) (ebimath.Vector, bool) {
	r := ls.Direction(false)
	s := other.Direction(false)
	originDist := other.A.Sub(ls.A)

	r_cross_s := r.X*s.Y - r.Y*s.X
	if math.Abs(r_cross_s) < 0.0001 {
		return ebimath.Vector{}, false
	}

	t := (originDist.X*s.Y - originDist.Y*s.X) / r_cross_s
	u := (originDist.X*r.Y - originDist.Y*r.X) / r_cross_s

	if !infiniteLines && (t < 0 || t > 1 || u < 0 || u > 1) {
		return ebimath.Vector{}, false
	}

	return ls.A.Add(r.ScaleF(t)), true
}

// PolySegment represents a thick line segment with a center and two outer edges.
type PolySegment struct {
	Center LineSegment
	Edge1  LineSegment
	Edge2  LineSegment
}

func NewPolySegment(p1, p2 ebimath.Vector, thickness float64) PolySegment {
	center := LineSegment{A: p1, B: p2}
	normal := center.Normal().ScaleF(thickness)

	edge1 := LineSegment{A: p1.Add(normal), B: p2.Add(normal)}
	edge2 := LineSegment{A: p1.Sub(normal), B: p2.Sub(normal)}

	return PolySegment{
		Center: center,
		Edge1:  edge1,
		Edge2:  edge2,
	}
}

func createTriangleFan(vertices *[]ebiten.Vertex, indices *[]uint16, connectTo, origin, start, end ebimath.Vector, clockwise bool, numSegments int, col color.RGBA, opacity float64) {
	point1 := start.Sub(origin)
	point2 := end.Sub(origin)

	angle1 := math.Atan2(point1.Y, point1.X)
	angle2 := math.Atan2(point2.Y, point2.X)

	if clockwise {
		if angle2 > angle1 {
			angle2 -= 2 * math.Pi
		}
	} else {
		if angle1 > angle2 {
			angle1 -= 2 * math.Pi
		}
	}

	jointAngle := angle2 - angle1
	triAngle := jointAngle / float64(numSegments)

	startPoint := start

	connectToIdx := uint16(len(*vertices))
	*vertices = append(*vertices, utils.CreateVertexWithOpacity(connectTo, ebimath.Vector{}, col, opacity))

	startPointIdx := uint16(len(*vertices))
	*vertices = append(*vertices, utils.CreateVertexWithOpacity(startPoint, ebimath.Vector{}, col, opacity))

	for i := 0; i < numSegments; i++ {
		angle := angle1 + float64(i+1)*triAngle
		endPoint := origin.Add(ebimath.V(math.Cos(angle), math.Sin(angle)).ScaleF(point1.Length()))

		if i+1 == numSegments {
			endPoint = end
		}

		endPointIdx := uint16(len(*vertices))
		*vertices = append(*vertices, utils.CreateVertexWithOpacity(endPoint, ebimath.Vector{}, col, opacity))

		*indices = append(*indices, startPointIdx, endPointIdx, connectToIdx)

		startPointIdx = endPointIdx
	}
}
