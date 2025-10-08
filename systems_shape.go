package katsu2d

import (
	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShapeRenderSystem renders shape components.
type ShapeRenderSystem struct {
	transform *Transform
	filter    *lazyecs.Filter2[TransformComponent, ShapeComponent]
}

// NewShapeRenderSystem creates a new ShapeRenderSystem.
func NewShapeRenderSystem() *ShapeRenderSystem {
	return &ShapeRenderSystem{
		transform: T(),
	}
}

func (self *ShapeRenderSystem) Initialize(w *lazyecs.World) {
	self.filter = self.filter.New(w)
}

// Draw renders all shape components in the world.
func (self *ShapeRenderSystem) Draw(w *lazyecs.World, rdr *BatchRenderer) {
	tm := GetTextureManager(w)
	for self.filter.Next() {
		transform, shape := self.filter.Get()
		shape.Shape.Rebuild()
		vertices := shape.Shape.GetVertices()
		indices := shape.Shape.GetIndices()

		if len(vertices) == 0 {
			continue
		}

		self.transform.SetFromComponent(transform)

		worldVertices := make([]ebiten.Vertex, len(vertices))
		transformMatrix := self.transform.Matrix()
		for i, v := range vertices {
			vx, vy := transformMatrix.Apply(float64(v.DstX), float64(v.DstY))
			v.DstX = float32(vx)
			v.DstY = float32(vy)
			v.SrcX = 0
			v.SrcY = 0
			worldVertices[i] = v
		}
		img := tm.Get(0)
		rdr.AddCustomMeshes(worldVertices, indices, img)
	}
	self.filter.Reset()
}
