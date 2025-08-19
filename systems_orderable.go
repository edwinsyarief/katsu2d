package katsu2d

import (
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// OrderableSystem renders sprite components sorted by their orderable index.
type OrderableSystem struct {
	world            *World
	tm               *TextureManager
	drawableEntities []Entity
}

// NewOrderableSystem creates a new RenderOrderSystem.
func NewOrderableSystem(world *World, tm *TextureManager) *OrderableSystem {
	return &OrderableSystem{
		world: world,
		tm:    tm,
	}
}

// Update queries for all renderable entities and sorts them.
func (self *OrderableSystem) Update(world *World, dt float64) {
	self.drawableEntities = world.Query(CTOrderable, CTSprite, CTTransform)

	sort.SliceStable(self.drawableEntities, func(i, j int) bool {
		orderable1, _ := world.GetComponent(self.drawableEntities[i], CTOrderable)
		o1 := orderable1.(*OrderableComponent)

		orderable2, _ := world.GetComponent(self.drawableEntities[j], CTOrderable)
		o2 := orderable2.(*OrderableComponent)

		index1 := o1.Index()
		index2 := o2.Index()

		if index1 != index2 {
			return index1 < index2
		}
		return self.drawableEntities[i].ID < self.drawableEntities[j].ID
	})
}

// Draw renders all sprites in the world, using the pre-sorted list.
func (self *OrderableSystem) Draw(world *World, renderer *BatchRenderer) {
	for _, entity := range self.drawableEntities {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		if s.dirty {
			s.GenerateMesh()
		}

		if s.MeshType == SpriteMeshTypeGrid {
			worldVertices := make([]ebiten.Vertex, len(s.Vertices))
			transformMatrix := t.Matrix()

			for i, v := range s.Vertices {
				v.ColorR = float32(s.Color.R) / 255
				v.ColorG = float32(s.Color.G) / 255
				v.ColorB = float32(s.Color.B) / 255
				v.ColorA = float32(s.Color.A) / 255 * s.Opacity
				vx, vy := (&transformMatrix).Apply(float64(v.DstX), float64(v.DstY))
				v.DstX = float32(vx)
				v.DstY = float32(vy)
				worldVertices[i] = v
			}
			renderer.AddVertices(worldVertices, s.Indices, img)
		} else {
			imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
			srcRect := s.GetSourceRect(imgW, imgH)

			effColor := s.Color
			effColor.A = uint8(float32(s.Color.A) * s.Opacity)

			realPos := ebimath.V2(0).Apply(t.Matrix())
			if !t.Origin().IsZero() {
				realPos = realPos.Sub(t.Origin())
			}

			renderer.DrawQuad(
				realPos,
				t.Scale(),
				t.Rotation(),
				img,
				effColor,
				float32(srcRect.Min.X), float32(srcRect.Min.Y), float32(srcRect.Max.X), float32(srcRect.Max.Y),
				float64(s.DstW), float64(s.DstH),
			)
		}
	}
}
