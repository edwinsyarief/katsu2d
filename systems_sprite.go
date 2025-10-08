package katsu2d

import (
	"sort"

	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteSystem struct {
	transform         *Transform
	filter            *lazyecs.Filter2[TransformComponent, SpriteComponent]
	lastFrameEntities map[lazyecs.Entity]struct{}
	entities          []lazyecs.Entity
	zSortNeeded       bool
}

func NewSpriteSystem() *SpriteSystem {
	return &SpriteSystem{
		transform:         T(),
		lastFrameEntities: make(map[lazyecs.Entity]struct{}),
		entities:          make([]lazyecs.Entity, 0),
	}
}
func (self *SpriteSystem) Initialize(w *lazyecs.World) {
	self.filter = self.filter.New(w)
}
func (self *SpriteSystem) Update(w *lazyecs.World, dt float64) {
	currentEntities := make([]lazyecs.Entity, 0)
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
			t1 := lazyecs.GetComponent[TransformComponent](w, self.entities[i])
			t2 := lazyecs.GetComponent[TransformComponent](w, self.entities[j])
			return t1.Z < t2.Z
		})
		self.zSortNeeded = false
	}

	self.lastFrameEntities = make(map[lazyecs.Entity]struct{}, len(currentEntities))
	for _, entity := range currentEntities {
		self.lastFrameEntities[entity] = struct{}{}
	}
}
func (self *SpriteSystem) Draw(w *lazyecs.World, rdr *BatchRenderer) {
	tm := GetTextureManager(w)
	for _, e := range self.entities {
		t, s := lazyecs.GetComponent2[TransformComponent, SpriteComponent](w, e)
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

		matrix := self.transform.Matrix()
		if m := lazyecs.GetComponent[MeshComponent](w, e); m != nil {
			GenerateMesh(m, s)
			worldVertices := make([]ebiten.Vertex, len(m.Vertices))
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
			pos := V2(0).Apply(matrix)
			col := s.Color
			col.A = uint8(float64(col.A) * s.Opacity)
			rdr.AddQuad(pos,
				self.transform.Scale(), self.transform.Rotation(),
				img, col,
				float32(s.Bound.Min.X), float32(s.Bound.Min.Y),
				float32(s.Bound.Max.X), float32(s.Bound.Max.Y),
				float64(s.Width), float64(s.Height))
		}
	}
}
