package line

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// RoundJoiner implements round line joins
type RoundJoiner struct{}

// buildRoundMesh generates vertices and indices for a round-joined line
func (self *RoundJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	totalSegments := len(l.points) - 1

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

		// Calculate segment direction and perpendicular
		dir := p2.position.Sub(p1.position).Normalized()
		perp := ebimath.V(-dir.Y, dir.X).Normalized()

		// Calculate texture coordinates
		tempTextureSize := l.calculateTextureSize()
		uv1, uv2, uv3, uv4 := l.calculateTextureCoords(i, totalSegments, tempTextureSize)

		// Create segment vertices
		verts := []ebiten.Vertex{
			utils.CreateVertexWithOpacity(p1.position.Add(perp.ScaleF(p1.width/2)), uv1, col1, l.opacity),
			utils.CreateVertexWithOpacity(p1.position.Sub(perp.ScaleF(p1.width/2)), uv2, col1, l.opacity),
			utils.CreateVertexWithOpacity(p2.position.Add(perp.ScaleF(p2.width/2)), uv3, col2, l.opacity),
			utils.CreateVertexWithOpacity(p2.position.Sub(perp.ScaleF(p2.width/2)), uv4, col2, l.opacity),
		}

		// Add vertices and indices for segment
		vertexIndex := uint16(len(vertices))
		vertices = append(vertices, verts...)
		indices = append(indices,
			vertexIndex+0, vertexIndex+1, vertexIndex+2,
			vertexIndex+2, vertexIndex+1, vertexIndex+3,
		)

		// Add round join for intermediate points
		if i > 0 {
			p0 := l.points[i-1]
			dir1 := p1.position.Sub(p0.position).Normalized()
			dir2 := p2.position.Sub(p1.position).Normalized()

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			// Calculate angles for round join
			startAngle := math.Atan2(perp1.Y, perp1.X)
			endAngle := math.Atan2(perp2.Y, perp2.X)

			// Adjust angles for proper rotation direction
			crossProduct := dir1.X*dir2.Y - dir1.Y*dir2.X
			if crossProduct > 0 {
				if endAngle < startAngle {
					endAngle += 2 * math.Pi
				}
			} else {
				if startAngle < endAngle {
					startAngle += 2 * math.Pi
				}
			}

			// Create center vertex for round join
			joinVertexIndex := uint16(len(vertices))
			vertices = append(vertices, utils.CreateVertexDefaultSrc(p1.position, col1))

			// Create arc vertices
			angleDelta := (endAngle - startAngle) / roundJoinSegments
			for s := 0; s <= roundJoinSegments; s++ {
				arcAngle := startAngle + float64(s)*angleDelta
				arcPerp := ebimath.V(math.Cos(arcAngle), math.Sin(arcAngle)).ScaleF(p1.width / 2)
				arcVertex := utils.CreateVertexDefaultSrc(p1.position.Add(arcPerp), col1)
				vertices = append(vertices, arcVertex)

				if s > 0 {
					indices = append(indices,
						joinVertexIndex,
						joinVertexIndex+uint16(s),
						joinVertexIndex+uint16(s)+1,
					)
				}
			}
		}
	}

	return vertices, indices
}
