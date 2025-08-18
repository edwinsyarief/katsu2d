package katsu2d

import (
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
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
// This is done every frame to ensure the order is always correct, as
// the sorting key can be dynamic.
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

		imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
		srcX, srcY, srcW, srcH := s.GetSourceRect(float32(imgW), float32(imgH))
		destW32, destH32 := s.GetDestSize(srcW, srcH)
		destW, destH := float64(destW32), float64(destH32)

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
			srcX, srcY, srcX+srcW, srcY+srcH,
			destW, destH,
		)
	}
}
