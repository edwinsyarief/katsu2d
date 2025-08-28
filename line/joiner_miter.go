package line

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// MiterJoiner implements miter line joins
type MiterJoiner struct{}

// buildMiterMesh generates vertices and indices for a miter-joined line
func (self *MiterJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)

	totalSegments := len(l.points) - 1

	// Pre-calculate vertices for each point
	for i := 0; i < len(l.points); i++ {
		p := l.points[i]

		switch i {
		case 0:
			// First point: use angle to next point
			nextP := l.points[i+1]
			angle := p.position.AngleToPoint(nextP.position)
			perp := ebimath.V(math.Cos(angle+math.Pi/2), math.Sin(angle+math.Pi/2)).
				Normalized().ScaleF(p.width / 2)
			p.top = p.position.Add(perp)
			p.bottom = p.position.Sub(perp)

		case len(l.points) - 1:
			// Last point: use angle from previous point
			prevP := l.points[i-1]
			angle := prevP.position.AngleToPoint(p.position)
			perp := ebimath.V(math.Cos(angle+math.Pi/2), math.Sin(angle+math.Pi/2)).
				Normalized().ScaleF(p.width / 2)
			p.top = p.position.Add(perp)
			p.bottom = p.position.Sub(perp)

		default:
			// Intermediate points: calculate miter joint
			prevP := l.points[i-1]
			nextP := l.points[i+1]

			dir1 := p.position.Sub(prevP.position).Normalized()
			dir2 := nextP.position.Sub(p.position).Normalized()

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			miter := perp1.Add(perp2)
			miterLength := miter.Length()

			if miterLength == 0 {
				// Lines are parallel, use first perpendicular
				p.top = p.position.Add(perp1.ScaleF(p.width / 2))
				p.bottom = p.position.Sub(perp1.ScaleF(p.width / 2))
			} else {
				cosHalfAngle := math.Sqrt((dir1.Dot(dir2) + 1.0) / 2.0)
				miterScale := p.width / (2.0 * cosHalfAngle)

				// Apply miter limit
				if miterScale > p.width*defaultMiterLimit {
					miterScale = p.width * defaultMiterLimit
				}

				miter = miter.Normalized().ScaleF(miterScale)
				p.top = p.position.Add(miter)
				p.bottom = p.position.Sub(miter)
			}
		}
	}

	// Build mesh from pre-calculated vertices
	for i := 0; i < totalSegments; i++ {
		p1 := l.points[i]
		p2 := l.points[i+1]

		// Handle color interpolation
		col1, col2 := p1.color, p2.color
		if l.interpolateColor {
			t1 := float64(i) / float64(totalSegments)
			t2 := float64(i+1) / float64(totalSegments)
			col1 = l.lerpColor(t1)
			col2 = l.lerpColor(t2)
		}

		// Calculate texture coordinates
		tempTextureSize := l.calculateTextureSize()
		uv1, uv2, uv3, uv4 := l.calculateTextureCoords(i, totalSegments, tempTextureSize)

		// Create vertices
		verts := []ebiten.Vertex{
			utils.CreateVertexWithOpacity(p1.top, uv1, col1, l.opacity),
			utils.CreateVertexWithOpacity(p1.bottom, uv2, col1, l.opacity),
			utils.CreateVertexWithOpacity(p2.top, uv3, col2, l.opacity),
			utils.CreateVertexWithOpacity(p2.bottom, uv4, col2, l.opacity),
		}

		// Create indices
		vertexIndex := uint16(len(vertices))
		vertices = append(vertices, verts...)
		indices = append(indices,
			vertexIndex+0, vertexIndex+1, vertexIndex+2,
			vertexIndex+2, vertexIndex+1, vertexIndex+3,
		)
	}

	return vertices, indices
}
