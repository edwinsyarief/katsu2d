package katsu2d

import (
	"image/color"

	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShapeRenderSystem renders shape components.
type ShapeRenderSystem struct {
	img *ebiten.Image
}

// NewShapeRenderSystem creates a new ShapeRenderSystem.
func NewShapeRenderSystem() *ShapeRenderSystem {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return &ShapeRenderSystem{
		img: img,
	}
}

// Draw renders all shape components in the world.
func (self *ShapeRenderSystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	query := world.Query(CTShape, CTTransform)
	for query.Next() {
		transforms, _ := lazyecs.GetComponentSlice[TransformComponent](query)
		for i, entity := range query.Entities() {
			shape, _ := lazyecs.GetComponent[ShapeComponent](world, entity)
			shape.Shape.Rebuild()

			t := transforms[i]
			vertices := shape.Shape.GetVertices()
			indices := shape.Shape.GetIndices()

			if len(vertices) == 0 {
				continue
			}

			worldVertices := make([]ebiten.Vertex, len(vertices))
			transformMatrix := t.Matrix()
			for i, v := range vertices {
				vx, vy := transformMatrix.Apply(float64(v.DstX), float64(v.DstY))
				v.DstX = float32(vx)
				v.DstY = float32(vy)
				v.SrcX = 0
				v.SrcY = 0
				worldVertices[i] = v
			}
			renderer.AddCustomMeshes(worldVertices, indices, self.img)
		}
	}
}
