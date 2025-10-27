package katsu2d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/ark/ecs"
)

// ShapeRenderSystem renders shape components.
type ShapeRenderSystem struct {
	transform   *Transform
	filter      *ecs.Filter2[TransformComponent, ShapeComponent]
	initialized bool
}

// NewShapeRenderSystem creates a new ShapeRenderSystem.
func NewShapeRenderSystem() *ShapeRenderSystem {
	return &ShapeRenderSystem{
		transform: T(),
	}
}

func (self *ShapeRenderSystem) Initialize(w *ecs.World) {
	if self.initialized {
		return
	}

	self.filter = self.filter.New(w)
	self.initialized = true
}

// Draw renders all shape components in the world.
func (self *ShapeRenderSystem) Draw(w *ecs.World, rdr *BatchRenderer) {
	tm := GetTextureManager(w)
	query := self.filter.Query()
	for query.Next() {
		transform, shape := query.Get()
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
}
