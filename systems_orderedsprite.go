package katsu2d

import (
	"sort"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

type OrderedSpriteSystem struct {
	transform                *Transform
	filter                   *teishoku.Filter3[TransformComponent, SpriteComponent, OrderableComponent]
	lastFrameEntities        map[teishoku.Entity]struct{}
	entities                 []teishoku.Entity
	zSortNeeded, initialized bool
}

func NewOrderedSpriteSystem() *OrderedSpriteSystem {
	return &OrderedSpriteSystem{
		transform:         T(),
		lastFrameEntities: make(map[teishoku.Entity]struct{}),
		entities:          make([]teishoku.Entity, 0),
	}
}
func (self *OrderedSpriteSystem) Initialize(w *teishoku.World) {
	if self.initialized {
		return
	}

	self.filter = self.filter.New(w)
	self.initialized = true
}
func (self *OrderedSpriteSystem) Update(w *teishoku.World, dt float64) {
	currentEntities := make([]teishoku.Entity, 0)
	self.filter.Reset()
	for self.filter.Next() {
		currentEntities = append(currentEntities, self.filter.Entity())
	}

	zSortNeeded := self.zSortNeeded || len(currentEntities) != len(self.lastFrameEntities)
	if !zSortNeeded && len(currentEntities) > 0 {
		for _, entity := range currentEntities {
			if _, ok := self.lastFrameEntities[entity]; !ok {
				zSortNeeded = true
				break
			}
		}
	}

	if zSortNeeded {
		self.entities = currentEntities
		sort.SliceStable(self.entities, func(i, j int) bool {
			t1 := teishoku.GetComponent[TransformComponent](w, self.entities[i])
			t2 := teishoku.GetComponent[TransformComponent](w, self.entities[j])
			if t1.Z != t2.Z {
				return t1.Z < t2.Z
			}

			o1 := teishoku.GetComponent[OrderableComponent](w, self.entities[i])
			o2 := teishoku.GetComponent[OrderableComponent](w, self.entities[j])
			index1 := o1.Index
			index2 := o2.Index
			if index1 != index2 {
				return index1 < index2
			}

			return self.entities[i].ID < self.entities[j].ID
		})
		self.zSortNeeded = false
	}

	self.lastFrameEntities = make(map[teishoku.Entity]struct{}, len(currentEntities))
	for _, entity := range currentEntities {
		self.lastFrameEntities[entity] = struct{}{}
	}
}
func (self *OrderedSpriteSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	tm := GetTextureManager(w)
	for _, e := range self.entities {
		t, s := teishoku.GetComponent2[TransformComponent, SpriteComponent](w, e)
		self.transform.SetFromComponent(t)
		img := tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		if IsBoundEmpty(s.Bound) {
			s.Bound = Bound{
				Min: Point{X: 0, Y: 0},
				Max: Point{X: float64(s.Width), Y: float64(s.Height)},
			}
		}

		if m := teishoku.GetComponent[MeshComponent](w, e); m != nil {
			GenerateMesh(m, s)

			worldVertices := make([]ebiten.Vertex, len(m.Vertices))
			matrix := self.transform.Matrix()
			for i, v := range m.Vertices {
				v.ColorR = float32(s.Color.R) / 255
				v.ColorG = float32(s.Color.G) / 255
				v.ColorB = float32(s.Color.B) / 255
				v.ColorA = (float32(s.Color.A) / 255) * float32(s.Opacity)
				vx, vy := (&matrix).Apply(float64(v.DstX), float64(v.DstY))
				v.DstX = float32(vx)
				v.DstY = float32(vy)
				worldVertices[i] = v
			}
			rdr.AddCustomMeshes(worldVertices, m.Indices, img)
		} else {
			col := s.Color
			col.A = uint8((float64(col.A) / 255.0) * s.Opacity)
			rdr.AddQuad(self.transform.Position(),
				self.transform.Offset(),
				self.transform.Origin(),
				self.transform.Scale(), self.transform.Rotation(),
				img, col,
				float32(s.Bound.Min.X), float32(s.Bound.Min.Y),
				float32(s.Bound.Max.X), float32(s.Bound.Max.Y),
				float64(s.Width), float64(s.Height))
		}
	}
}
