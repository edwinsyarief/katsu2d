package katsu2d

import (
	"image/color"

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

// Update rebuilds the mesh for any dirty shapes.
func (self *ShapeRenderSystem) Update(world *World, dt float64) {
	entities := world.Query(CTRectangle)
	for _, e := range entities {
		rectAny, _ := world.GetComponent(e, CTRectangle)
		rect := rectAny.(*RectangleComponent)
		rect.Rebuild()
	}
}

// Draw renders all shape components in the world.
func (self *ShapeRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTTransform, CTRectangle)
	for _, e := range entities {
		rectAny, _ := world.GetComponent(e, CTRectangle)
		rect := rectAny.(*RectangleComponent)

		tAny, _ := world.GetComponent(e, CTTransform)
		t := tAny.(*TransformComponent)

		if len(rect.Vertices) == 0 {
			continue
		}

		worldVertices := make([]ebiten.Vertex, len(rect.Vertices))
		transformMatrix := t.Matrix()
		for i, v := range rect.Vertices {
			vx, vy := (&transformMatrix).Apply(float64(v.DstX), float64(v.DstY))
			v.DstX = float32(vx)
			v.DstY = float32(vy)
			v.SrcX = 0
			v.SrcY = 0
			worldVertices[i] = v
		}
		renderer.AddCustomMeshes(worldVertices, rect.Indices, self.img)
	}
}
