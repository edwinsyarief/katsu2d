package katsu2d

import (
	"sort"

	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// OrderableSystem renders sprite components sorted by their orderable index.
type OrderableSystem struct {
	tm               *TextureManager
	drawableEntities []lazyecs.Entity
	transforms       map[lazyecs.Entity]TransformComponent
	sprites          map[lazyecs.Entity]SpriteComponent
	orderables       map[lazyecs.Entity]OrderableComponent
}

// NewOrderableSystem creates a new RenderOrderSystem.
func NewOrderableSystem(tm *TextureManager) *OrderableSystem {
	return &OrderableSystem{
		tm: tm,
	}
}

// Update queries for all renderable entities and sorts them.
func (self *OrderableSystem) Update(world *lazyecs.World, dt float64) {
	self.transforms = make(map[lazyecs.Entity]TransformComponent)
	self.sprites = make(map[lazyecs.Entity]SpriteComponent)
	self.orderables = make(map[lazyecs.Entity]OrderableComponent)

	query := world.Query(CTTransform, CTSprite, CTOrderable)
	currentEntities := make([]lazyecs.Entity, 0)
	lastEntities := make(map[lazyecs.Entity]struct{})
	for query.Next() {
		transforms, _ := lazyecs.GetComponentSlice[TransformComponent](query)
		sprites, _ := lazyecs.GetComponentSlice[SpriteComponent](query)
		orderable, _ := lazyecs.GetComponentSlice[OrderableComponent](query)

		for i, entity := range query.Entities() {
			if _, ok := lastEntities[entity]; !ok {
				currentEntities = append(currentEntities, entity)
			}

			self.transforms[entity] = transforms[i]
			self.sprites[entity] = sprites[i]
			self.orderables[entity] = orderable[i]
		}
	}

	self.drawableEntities = currentEntities

	sort.SliceStable(self.drawableEntities, func(i, j int) bool {
		o1 := self.orderables[self.drawableEntities[i]]
		o2 := self.orderables[self.drawableEntities[j]]

		index1 := o1.Index()
		index2 := o2.Index()

		if index1 != index2 {
			return index1 < index2
		}
		return self.drawableEntities[i].ID < self.drawableEntities[j].ID
	})
}

// Draw renders all sprites in the world, using the pre-sorted list.
func (self *OrderableSystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	for _, entity := range self.drawableEntities {
		t := self.transforms[entity]
		s := self.sprites[entity]

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		if s.Dirty {
			s.GenerateMesh()
		}

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
		renderer.AddCustomMeshes(worldVertices, s.Indices, img)
	}
}
