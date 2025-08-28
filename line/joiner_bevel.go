package line

import (
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// BevelJoiner implements bevel line joins
type BevelJoiner struct{}

// buildBevelMesh generates vertices and indices for a bevel-joined line
func (self *BevelJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
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

		// Add bevel join for intermediate points
		if i > 0 {
			p0 := l.points[i-1]
			dir1 := p1.position.Sub(p0.position).Normalized()
			dir2 := p2.position.Sub(p1.position).Normalized()

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			isConvex := dir1.X*dir2.Y-dir1.Y*dir2.X < 0

			// Create join vertices
			joinVertex := utils.CreateVertexDefaultSrc(p1.position, col1)
			joinVertexIndex := uint16(len(vertices))
			vertices = append(vertices, joinVertex)

			if isConvex {
				v1 := utils.CreateVertexDefaultSrc(p1.position.Add(perp1.ScaleF(p1.width/2)), col1)
				v2 := utils.CreateVertexDefaultSrc(p1.position.Add(perp2.ScaleF(p1.width/2)), col1)
				vertices = append(vertices, v1, v2)
			} else {
				v1 := utils.CreateVertexDefaultSrc(p1.position.Sub(perp1.ScaleF(p1.width/2)), col1)
				v2 := utils.CreateVertexDefaultSrc(p1.position.Sub(perp2.ScaleF(p1.width/2)), col1)
				vertices = append(vertices, v1, v2)
			}
			indices = append(indices, joinVertexIndex, joinVertexIndex+1, joinVertexIndex+2)
		}
	}

	return vertices, indices
}
