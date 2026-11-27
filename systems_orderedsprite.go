package katsu2d

import (
	"cmp"
	"slices"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// OrderedSpriteSystemOption configures an OrderedSpriteSystem.
type OrderedSpriteSystemOption func(*OrderedSpriteSystem)

// WithOrderedSpriteFrustumMargin sets the extra world-space padding around ActiveCamera.ViewBounds
// used for ordered sprite frustum culling (default 96).
func WithOrderedSpriteFrustumMargin(margin float64) OrderedSpriteSystemOption {
	return func(s *OrderedSpriteSystem) {
		if margin < 0 {
			margin = 0
		}
		s.frustumMargin = margin
	}
}

// OrderedSpriteSystem sorts and renders entities possessing TransformComponent,
// SpriteComponent, and OrderableComponent, ordered by Z, then OrderableComponent.Index, then Entity.ID.
type OrderedSpriteSystem struct {
	transform       *Transform
	filter          *teishoku.Filter3[TransformComponent, SpriteComponent, OrderableComponent]
	tsBuilder       *teishoku.Builder2[TransformComponent, SpriteComponent]
	meshBuilder     *teishoku.Builder[MeshComponent]
	particleBuilder *teishoku.Builder[ParticleComponent]
	tmHandle        teishoku.ResourceHandle[TextureManager]
	camHandle       teishoku.ResourceHandle[ActiveCamera]
	frustumMargin   float64
	entries         []spriteRenderEntry
	worldVertices   []ebiten.Vertex
	initialized     bool
}

// NewOrderedSpriteSystem creates a new OrderedSpriteSystem.
func NewOrderedSpriteSystem(opts ...OrderedSpriteSystemOption) *OrderedSpriteSystem {
	s := &OrderedSpriteSystem{
		transform:     T(),
		frustumMargin: defaultSpriteFrustumMargin,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// Initialize prepares the OrderedSpriteSystem's ECS filter, builders, and resource handles.
func (s *OrderedSpriteSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.filter = teishoku.NewFilter3[TransformComponent, SpriteComponent, OrderableComponent](w)
	s.tsBuilder = teishoku.NewBuilder2[TransformComponent, SpriteComponent](w)
	s.meshBuilder = teishoku.NewBuilder[MeshComponent](w)
	s.particleBuilder = teishoku.NewBuilder[ParticleComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	s.initialized = true
}

// Update collects visible orderable sprite entities, updates Y-sorted indices, and sorts when out of order.
func (s *OrderedSpriteSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		s.Initialize(w)
	}
	cam := s.camHandle.Get()
	if cam == nil {
		if c := GetActiveCamera(w); c != nil {
			s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
			cam = c
		}
	}
	useCull := cam != nil && cam.HasCamera

	s.entries = s.entries[:0]
	chunks := s.filter.Chunks()
	for chunks.Next() {
		entities := chunks.Entities()
		transforms, sprites, orderables := chunks.Get()
		for i := range entities {
			o := &orderables[i]
			if o.AutoSortY {
				o.Index = transforms[i].Position.Y + o.YOffset
			}
			if useCull && !cam.IsVisible(spriteWorldBounds(&transforms[i], &sprites[i], s.frustumMargin)) {
				continue
			}
			s.entries = append(s.entries, spriteRenderEntry{
				entity: entities[i],
				z:      transforms[i].Z,
				order:  o.Index,
			})
		}
	}

	sorted := true
	for i := 1; i < len(s.entries); i++ {
		prev := s.entries[i-1]
		curr := s.entries[i]
		if curr.z < prev.z {
			sorted = false
			break
		}
		if curr.z == prev.z {
			if curr.order < prev.order || (curr.order == prev.order && curr.entity.ID < prev.entity.ID) {
				sorted = false
				break
			}
		}
	}
	if !sorted {
		slices.SortFunc(s.entries, func(a, b spriteRenderEntry) int {
			if c := cmp.Compare(a.z, b.z); c != 0 {
				return c
			}
			if c := cmp.Compare(a.order, b.order); c != 0 {
				return c
			}
			return cmp.Compare(a.entity.ID, b.entity.ID)
		})
	}
}

// Draw submits all sorted orderable sprites and sprite meshes to the BatchRenderer.
func (s *OrderedSpriteSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	s.worldVertices = drawSpriteEntries(
		s.entries,
		s.tsBuilder,
		s.meshBuilder,
		s.particleBuilder,
		s.tmHandle.Get(),
		s.transform,
		s.worldVertices,
		rdr,
	)
}
