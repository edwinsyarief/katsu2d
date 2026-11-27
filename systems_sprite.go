package katsu2d

import (
	"cmp"
	"math"
	"slices"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

const defaultSpriteFrustumMargin = 96.0

// SpriteSystemOption configures a SpriteSystem.
type SpriteSystemOption func(*SpriteSystem)

// WithSpriteFrustumMargin sets the extra world-space padding around ActiveCamera.ViewBounds
// used for sprite frustum culling (default 96).
func WithSpriteFrustumMargin(margin float64) SpriteSystemOption {
	return func(s *SpriteSystem) {
		if margin < 0 {
			margin = 0
		}
		s.frustumMargin = margin
	}
}

type spriteRenderEntry struct {
	entity teishoku.Entity
	z      float64
	order  float64
}

// SpriteSystem sorts and renders entities possessing TransformComponent and SpriteComponent.
type SpriteSystem struct {
	transform       *Transform
	filter          *teishoku.Filter2[TransformComponent, SpriteComponent]
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

// NewSpriteSystem creates a new SpriteSystem.
func NewSpriteSystem(opts ...SpriteSystemOption) *SpriteSystem {
	s := &SpriteSystem{
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

// Initialize prepares the SpriteSystem's ECS filter, builders, and resource handles.
func (s *SpriteSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.filter = teishoku.NewFilter2[TransformComponent, SpriteComponent](w)
	s.tsBuilder = teishoku.NewBuilder2[TransformComponent, SpriteComponent](w)
	s.meshBuilder = teishoku.NewBuilder[MeshComponent](w)
	s.particleBuilder = teishoku.NewBuilder[ParticleComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	s.initialized = true
}

// Update collects visible sprite entities and sorts them by Z depth when out of order.
func (s *SpriteSystem) Update(w *teishoku.World, dt float64) {
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
		transforms, sprites := chunks.Get()
		for i := range entities {
			if useCull && !cam.IsVisible(spriteWorldBounds(&transforms[i], &sprites[i], s.frustumMargin)) {
				continue
			}
			s.entries = append(s.entries, spriteRenderEntry{
				entity: entities[i],
				z:      transforms[i].Z,
			})
		}
	}

	sorted := true
	for i := 1; i < len(s.entries); i++ {
		prev := s.entries[i-1]
		curr := s.entries[i]
		if curr.z < prev.z || (curr.z == prev.z && curr.entity.ID < prev.entity.ID) {
			sorted = false
			break
		}
	}
	if !sorted {
		slices.SortFunc(s.entries, func(a, b spriteRenderEntry) int {
			if c := cmp.Compare(a.z, b.z); c != 0 {
				return c
			}
			return cmp.Compare(a.entity.ID, b.entity.ID)
		})
	}
}

// Draw submits all sorted sprites and sprite meshes to the BatchRenderer.
func (s *SpriteSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
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

func drawSpriteEntries(
	entries []spriteRenderEntry,
	tsBuilder *teishoku.Builder2[TransformComponent, SpriteComponent],
	meshBuilder *teishoku.Builder[MeshComponent],
	particleBuilder *teishoku.Builder[ParticleComponent],
	tm *TextureManager,
	transform *Transform,
	worldVertices []ebiten.Vertex,
	rdr *BatchRenderer,
) []ebiten.Vertex {
	if tm == nil {
		return worldVertices
	}
	const inv255 = float32(1.0 / 255.0)
	for _, entry := range entries {
		t, sp := tsBuilder.Get(entry.entity)
		if t == nil || sp == nil {
			continue
		}
		img := tm.Get(sp.TextureID)
		if img == nil {
			continue
		}
		transform.SetFromComponent(t)

		if m := meshBuilder.Get(entry.entity); m != nil {
			SyncMeshUVs(m, sp, img)
			worldVertices = worldVertices[:0]
			matrix := transform.Matrix()
			op := float32(Clamp(sp.Opacity, 0.0, 1.0))
			cr := float32(sp.Color.R) * inv255 * op
			cg := float32(sp.Color.G) * inv255 * op
			cb := float32(sp.Color.B) * inv255 * op
			ca := float32(sp.Color.A) * inv255 * op
			for _, v := range m.Vertices {
				v.ColorR *= cr
				v.ColorG *= cg
				v.ColorB *= cb
				v.ColorA *= ca
				vx, vy := matrix.Apply(float64(v.DstX), float64(v.DstY))
				v.DstX = float32(vx)
				v.DstY = float32(vy)
				worldVertices = append(worldVertices, v)
			}
			rdr.AddCustomMeshes(worldVertices, m.Indices, img)
		} else {
			op := Clamp(sp.Opacity, 0.0, 1.0)
			col := sp.Color
			if op < 1.0 {
				col.R = uint8(float64(col.R) * op)
				col.G = uint8(float64(col.G) * op)
				col.B = uint8(float64(col.B) * op)
				col.A = uint8(float64(col.A) * op)
			}
			srcMinX, srcMinY, srcMaxX, srcMaxY := resolveSpriteSourceBounds(sp, img)
			if particleBuilder != nil {
				if p := particleBuilder.Get(entry.entity); p != nil {
					blend := p.BlendMode
					if blend == (ebiten.Blend{}) {
						blend = ebiten.BlendSourceOver
					}
					rdr.AddQuadWithBlend(
						transform.Position(),
						transform.Offset(),
						transform.Origin(),
						transform.Scale(),
						transform.Rotation(),
						img, col, blend,
						srcMinX, srcMinY,
						srcMaxX, srcMaxY,
						float64(sp.Width), float64(sp.Height),
					)
					continue
				}
			}
			rdr.AddQuad(
				transform.Position(),
				transform.Offset(),
				transform.Origin(),
				transform.Scale(),
				transform.Rotation(),
				img, col,
				srcMinX, srcMinY,
				srcMaxX, srcMaxY,
				float64(sp.Width), float64(sp.Height),
			)
		}
	}
	return worldVertices
}

func spriteWorldBounds(t *TransformComponent, sp *SpriteComponent, margin float64) Rectangle {
	tx := t.Position.X - t.Origin.X
	ty := t.Position.Y - t.Origin.Y

	if sp.Width <= 0 || sp.Height <= 0 {
		sx := math.Abs(t.Scale.X)
		sy := math.Abs(t.Scale.Y)
		maxScale := math.Max(1.0, math.Max(sx, sy))
		ext := 256.0*maxScale + margin + math.Max(math.Abs(t.Offset.X)*sx, math.Abs(t.Offset.Y)*sy)
		return Rectangle{
			Min: Vector{X: tx - ext, Y: ty - ext},
			Max: Vector{X: tx + ext, Y: ty + ext},
		}
	}

	w := float64(sp.Width)
	h := float64(sp.Height)
	lx0 := -t.Offset.X * t.Scale.X
	lx1 := (w - t.Offset.X) * t.Scale.X
	if lx0 > lx1 {
		lx0, lx1 = lx1, lx0
	}
	ly0 := -t.Offset.Y * t.Scale.Y
	ly1 := (h - t.Offset.Y) * t.Scale.Y
	if ly0 > ly1 {
		ly0, ly1 = ly1, ly0
	}

	if t.Rotation != 0 {
		rx := math.Max(math.Abs(lx0), math.Abs(lx1))
		ry := math.Max(math.Abs(ly0), math.Abs(ly1))
		diag := math.Sqrt(rx*rx+ry*ry) + margin
		return Rectangle{
			Min: Vector{X: tx - diag, Y: ty - diag},
			Max: Vector{X: tx + diag, Y: ty + diag},
		}
	}

	return Rectangle{
		Min: Vector{X: tx + lx0 - margin, Y: ty + ly0 - margin},
		Max: Vector{X: tx + lx1 + margin, Y: ty + ly1 + margin},
	}
}
