package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShapeRenderSystem renders shape components.
type ShapeRenderSystem struct {
	transform     *Transform
	filter        *teishoku.Filter2[TransformComponent, ShapeComponent]
	tmHandle      teishoku.ResourceHandle[TextureManager]
	worldVertices []ebiten.Vertex
	initialized   bool
}

// NewShapeRenderSystem creates a new ShapeRenderSystem.
func NewShapeRenderSystem() *ShapeRenderSystem {
	return &ShapeRenderSystem{
		transform: T(),
	}
}

// Initialize prepares the ShapeRenderSystem's ECS filter and TextureManager handle.
func (s *ShapeRenderSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter2[TransformComponent, ShapeComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.initialized = true
}

// Draw renders all shape components in the world.
func (s *ShapeRenderSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	var img *ebiten.Image
	if tm := s.tmHandle.Get(); tm != nil {
		img = tm.Get(0)
	}
	if img == nil {
		img = defaultWhiteImage()
	}

	s.filter.EachChunk(func(transforms []TransformComponent, shapes []ShapeComponent) {
		for i := range transforms {
			shape := shapes[i].Shape
			if shape == nil {
				continue
			}
			shape.Rebuild()
			vertices := shape.GetVertices()
			indices := shape.GetIndices()
			if len(vertices) == 0 || len(indices) == 0 {
				continue
			}

			s.transform.SetFromComponent(&transforms[i])
			transformMatrix := s.transform.Matrix()

			s.worldVertices = s.worldVertices[:0]
			for _, v := range vertices {
				vx, vy := transformMatrix.Apply(float64(v.DstX), float64(v.DstY))
				v.DstX = float32(vx)
				v.DstY = float32(vy)
				v.SrcX = 0
				v.SrcY = 0
				s.worldVertices = append(s.worldVertices, v)
			}
			rdr.AddCustomMeshes(s.worldVertices, indices, img)
		}
	})
}
