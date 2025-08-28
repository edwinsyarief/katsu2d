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
		perp := ebimath.V(-dir.Y, dir.X).Normalized().ScaleF(p1.width / 2)

		// Calculate texture coordinates
		tempTextureSize := l.calculateTextureSize()
		uv1, uv2, uv3, uv4 := l.calculateTextureCoords(i, totalSegments, tempTextureSize)

		// Create segment vertices
		verts := []ebiten.Vertex{
			utils.CreateVertexWithOpacity(p1.position.Add(perp), uv1, col1, l.opacity),
			utils.CreateVertexWithOpacity(p1.position.Sub(perp), uv2, col1, l.opacity),
			utils.CreateVertexWithOpacity(p2.position.Add(perp.ScaleF(p2.width/p1.width)), uv3, col2, l.opacity),
			utils.CreateVertexWithOpacity(p2.position.Sub(perp.ScaleF(p2.width/p1.width)), uv4, col2, l.opacity),
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

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized().ScaleF(p1.width / 2)
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized().ScaleF(p1.width / 2)

			cross := dir1.X*dir2.Y - dir1.Y*dir2.X
			if cross > 0 {
				perp1 = perp1.Invert()
				perp2 = perp2.Invert()
			}

			// Create join vertices with proper bevel connection
			joinVertex := utils.CreateVertexWithOpacity(p1.position, ebimath.V(0, 0), col1, l.opacity)
			joinVertexIndex := uint16(len(vertices))
			vertices = append(vertices, joinVertex)

			v1 := utils.CreateVertexWithOpacity(p1.position.Add(perp1), ebimath.V(0, 0), col1, l.opacity)
			v2 := utils.CreateVertexWithOpacity(p1.position.Add(perp2), ebimath.V(0, 0), col1, l.opacity)
			vertices = append(vertices, v1, v2)
			indices = append(indices, joinVertexIndex, joinVertexIndex+1, joinVertexIndex+2)
		}
	}

	return vertices, indices
}
