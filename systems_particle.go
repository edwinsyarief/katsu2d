package katsu2d

import (
	"image/color"
	"math"

	"github.com/edwinsyarief/katsu2d/opensimplex"
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// ParticleEmitterSystem is an UpdateSystem that spawns particles from active or bursting emitters.
type ParticleEmitterSystem struct {
	pFilter                *teishoku.Filter[ParticleComponent]
	emFilter               *teishoku.Filter2[TransformComponent, ParticleEmitterComponent]
	orderBuilder           *teishoku.Builder[OrderableComponent]
	particleBuilder        *teishoku.Builder3[TransformComponent, SpriteComponent, ParticleComponent]
	orderedParticleBuilder *teishoku.Builder4[TransformComponent, SpriteComponent, ParticleComponent, OrderableComponent]
	tmHandle               teishoku.ResourceHandle[TextureManager]
	r                      *Rand
	initialized            bool
}

// NewParticleEmitterSystem creates a new ParticleEmitterSystem.
func NewParticleEmitterSystem() *ParticleEmitterSystem {
	return &ParticleEmitterSystem{
		r: Random(),
	}
}

// Initialize caches ECS filters, builders, and the TextureManager resource handle.
func (s *ParticleEmitterSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.pFilter = teishoku.NewFilter[ParticleComponent](w)
	s.emFilter = teishoku.NewFilter2[TransformComponent, ParticleEmitterComponent](w)
	s.orderBuilder = teishoku.NewBuilder[OrderableComponent](w)
	s.particleBuilder = teishoku.NewBuilder3[TransformComponent, SpriteComponent, ParticleComponent](w)
	s.orderedParticleBuilder = teishoku.NewBuilder4[TransformComponent, SpriteComponent, ParticleComponent, OrderableComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.initialized = true
}

// Update advances emitter spawn counters and creates particle entities up to MaxParticles.
func (s *ParticleEmitterSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		s.Initialize(w)
	}
	if !s.tmHandle.Has() {
		s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	}

	currentParticles := s.pFilter.Count()

	c := s.emFilter.Chunks()
	for c.Next() {
		ents := c.Entities()
		transforms, emitters := c.Get()
		for i := range emitters {
			em := &emitters[i]
			particlesToSpawn := 0
			if em.Active {
				em.SpawnCounter += em.EmitRate * dt
				particlesToSpawn = int(em.SpawnCounter)
				em.SpawnCounter -= float64(particlesToSpawn)
			} else if em.BurstCount > 0 {
				particlesToSpawn = em.BurstCount
				em.BurstCount = 0
			}

			if particlesToSpawn <= 0 {
				continue
			}

			emTransform := &transforms[i]
			for range particlesToSpawn {
				if currentParticles >= em.MaxParticles {
					break
				}
				s.spawnParticle(ents[i], emTransform, em)
				currentParticles++
			}
		}
	}
}

func (s *ParticleEmitterSystem) spawnParticle(emitterEnt teishoku.Entity, emitterTransform *TransformComponent, em *ParticleEmitterComponent) {
	texID := 0
	switch len(em.TextureIDs) {
	case 0:
		texID = 0
	case 1:
		texID = em.TextureIDs[0]
	default:
		texID = RandomElement(s.r, em.TextureIDs)
	}

	var texW, texH int
	var bound Bound
	if tm := s.tmHandle.Get(); tm != nil {
		if img := tm.Get(texID); img != nil {
			b := img.Bounds()
			texW = b.Dx()
			texH = b.Dy()
			bound = Bound{
				Max: Point{X: float64(texW), Y: float64(texH)},
			}
		}
	}

	randAngle := s.r.Rad()
	sin, cos := math.Sincos(randAngle)
	randSpeed := s.r.FloatRange(em.InitialParticleSpeedMin, em.InitialParticleSpeedMax)
	scale := Lerp(em.MinScale, em.MaxScale, s.r.Float64())
	rot := Lerp(em.MinRotation, em.MaxRotation, s.r.Float64())

	lifetime := em.ParticleLifetime
	if lifetime <= 0 {
		lifetime = 0.1
	}

	blendMode := em.BlendMode
	if blendMode == (ebiten.Blend{}) {
		blendMode = ebiten.BlendSourceOver
	}

	initColor := InterpolateColorWithRand(s.r, em.InitialColorMin, em.InitialColorMax)
	targetColor := InterpolateColorWithRand(s.r, em.TargetColorMin, em.TargetColorMax)

	pt := TransformComponent{
		Position: Point{
			X: emitterTransform.Position.X + s.r.Float64()*em.ParticleSpawnOffset.X,
			Y: emitterTransform.Position.Y + s.r.Float64()*em.ParticleSpawnOffset.Y,
		},
		Scale:    Point{X: scale, Y: scale},
		Offset:   Point{X: float64(texW) * 0.5, Y: float64(texH) * 0.5},
		Rotation: rot,
		Z:        emitterTransform.Z,
		IsDirty:  true,
	}

	ps := SpriteComponent{
		TextureID: texID,
		Width:     texW,
		Height:    texH,
		Bound:     bound,
		Color:     initColor,
		Opacity:   1.0,
	}

	pd := ParticleComponent{
		Gravity:          em.Gravity,
		Velocity:         V(randSpeed*cos, randSpeed*sin),
		Lifetime:         lifetime,
		TotalLifetime:    lifetime,
		InitialColor:     initColor,
		TargetColor:      targetColor,
		InitialScale:     scale,
		TargetScale:      Lerp(em.TargetScaleMin, em.TargetScaleMax, s.r.Float64()),
		InitialRotation:  rot,
		TargetRotation:   Lerp(em.EndRotationMin, em.EndRotationMax, s.r.Float64()),
		RotationSpeed:    Lerp(em.RotationSpeedMin, em.RotationSpeedMax, s.r.Float64()),
		NoiseOffsetX:     s.r.Float64() * 1000,
		NoiseOffsetY:     s.r.Float64() * 1000,
		ZigZagFrequency:  em.ZigZagFrequency,
		ZigZagMagnitude:  em.ZigZagMagnitude,
		NoiseFactor:      em.NoiseFactor,
		BlendMode:        blendMode,
		FadeMode:         em.FadeMode,
		ScaleMode:        em.ScaleMode,
		DirectionMode:    em.DirectionMode,
		UseRotationSpeed: em.RotationSpeedMin != 0 || em.RotationSpeedMax != 0,
	}

	if emitterOrder := s.orderBuilder.Get(emitterEnt); emitterOrder != nil {
		s.orderedParticleBuilder.NewEntityWith(pt, ps, pd, OrderableComponent{
			Index: emitterOrder.Index,
		})
	} else {
		s.particleBuilder.NewEntityWith(pt, ps, pd)
	}
}

// ParticleUpdateSystem is an UpdateSystem that simulates particle physics, appearance, and lifetime.
type ParticleUpdateSystem struct {
	noise       opensimplex.Noise
	filter      *teishoku.Filter3[TransformComponent, SpriteComponent, ParticleComponent]
	toRemove    []teishoku.Entity
	time        float64
	initialized bool
}

// NewParticleUpdateSystem creates a new ParticleUpdateSystem.
func NewParticleUpdateSystem() *ParticleUpdateSystem {
	return &ParticleUpdateSystem{
		noise: opensimplex.New(Random().PositiveInt64()),
	}
}

// Initialize caches the particle query filter.
func (s *ParticleUpdateSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter3[TransformComponent, SpriteComponent, ParticleComponent](w)
	s.initialized = true
}

// Update advances particle physics, fading, scaling, and rotation, and removes expired particles.
func (s *ParticleUpdateSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		s.Initialize(w)
	}

	s.time += dt
	s.toRemove = s.toRemove[:0]

	c := s.filter.Chunks()
	for c.Next() {
		ents := c.Entities()
		transforms, sprites, particles := c.Get()
		for i := range particles {
			p := &particles[i]
			p.Velocity.X += p.Gravity.X * dt
			p.Velocity.Y += p.Gravity.Y * dt

			switch p.DirectionMode {
			case ParticleDirectionModeZigZag:
				perp := V(-p.Velocity.Y, p.Velocity.X).Normalize()
				zigZag := math.Sin(s.time*p.ZigZagFrequency) * p.ZigZagMagnitude
				p.Velocity.X += perp.X * zigZag
				p.Velocity.Y += perp.Y * zigZag
			case ParticleDirectionModeNoise:
				noiseTime := s.time * p.NoiseFactor
				p.Velocity.X += s.noise.Eval2(p.NoiseOffsetX, noiseTime)
				p.Velocity.Y += s.noise.Eval2(p.NoiseOffsetY, noiseTime)
			}

			t := &transforms[i]
			t.Position.X += p.Velocity.X * dt
			t.Position.Y += p.Velocity.Y * dt

			p.Lifetime -= dt
			if p.Lifetime <= 0 {
				s.toRemove = append(s.toRemove, ents[i])
				continue
			}

			tp := 1.0
			if p.TotalLifetime > 0 {
				tp = Clamp(1.0-(p.Lifetime/p.TotalLifetime), 0.0, 1.0)
			}

			// Fading
			initialA := p.InitialColor.A
			targetA := p.TargetColor.A
			var currentA uint8
			switch p.FadeMode {
			case ParticleFadeModeFadeIn:
				currentA = uint8(Lerp(0, float64(initialA), tp))
			case ParticleFadeModeFadeOut:
				currentA = uint8(Lerp(float64(initialA), float64(targetA), tp))
			case ParticleFadeModeFadeInOut:
				if tp < 0.5 {
					currentA = uint8(Lerp(0, float64(initialA), tp*2))
				} else {
					currentA = uint8(Lerp(float64(initialA), float64(targetA), (tp-0.5)*2))
				}
			default:
				currentA = initialA
			}
			sp := &sprites[i]
			c := LerpPremultipliedRGBA(p.InitialColor, p.TargetColor, tp)
			alphaScale := float64(currentA) / 255.0
			sp.Color = color.RGBA{
				R: uint8(float64(c.R) * alphaScale),
				G: uint8(float64(c.G) * alphaScale),
				B: uint8(float64(c.B) * alphaScale),
				A: currentA,
			}

			// Scaling
			var currentScale float64
			switch p.ScaleMode {
			case ParticleScaleModeScaleIn:
				currentScale = Lerp(0, p.InitialScale, tp)
			case ParticleScaleModeScaleOut:
				currentScale = Lerp(p.InitialScale, p.TargetScale, tp)
			case ParticleScaleModeScaleInOut:
				if tp < 0.5 {
					currentScale = Lerp(0, p.InitialScale, tp*2)
				} else {
					currentScale = Lerp(p.InitialScale, p.TargetScale, (tp-0.5)*2)
				}
			default:
				currentScale = p.InitialScale
			}
			t.Scale.X = currentScale
			t.Scale.Y = currentScale

			// Rotation
			if p.UseRotationSpeed {
				t.Rotation += p.RotationSpeed * dt
			} else {
				t.Rotation = Lerp(p.InitialRotation, p.TargetRotation, tp)
			}
			t.IsDirty = true
		}
	}

	if len(s.toRemove) > 0 {
		w.RemoveEntities(s.toRemove)
	}
}

// ParticleRenderSystem is a DrawSystem that submits unordered particles to the BatchRenderer with their blend modes.
// Particles possessing OrderableComponent are skipped here and rendered in Y/index order by OrderedSpriteSystem.
type ParticleRenderSystem struct {
	filter       *teishoku.Filter3[TransformComponent, SpriteComponent, ParticleComponent]
	orderBuilder *teishoku.Builder[OrderableComponent]
	tmHandle     teishoku.ResourceHandle[TextureManager]
	camHandle    teishoku.ResourceHandle[ActiveCamera]
	initialized  bool
}

// NewParticleRenderSystem creates a new ParticleRenderSystem.
func NewParticleRenderSystem() *ParticleRenderSystem {
	return &ParticleRenderSystem{}
}

// Initialize caches the particle query filter, orderable builder, and resource handles.
func (s *ParticleRenderSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.filter = teishoku.NewFilter3[TransformComponent, SpriteComponent, ParticleComponent](w)
	s.orderBuilder = teishoku.NewBuilder[OrderableComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	s.initialized = true
}

// Draw submits all visible unordered particle quads to the BatchRenderer using each particle's blend mode.
func (s *ParticleRenderSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if !s.initialized {
		s.Initialize(w)
	}

	tm := s.tmHandle.Get()
	if tm == nil {
		s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
		tm = s.tmHandle.Get()
		if tm == nil {
			return
		}
	}

	cam := s.camHandle.Get()
	if cam == nil {
		if c := GetActiveCamera(w); c != nil {
			s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
			cam = c
		}
	}
	useCull := cam != nil && cam.HasCamera

	chunks := s.filter.Chunks()
	for chunks.Next() {
		ents := chunks.Entities()
		transforms, sprites, particles := chunks.Get()
		for i := range particles {
			if s.orderBuilder != nil && s.orderBuilder.Get(ents[i]) != nil {
				continue
			}
			t := &transforms[i]
			sp := &sprites[i]
			if useCull {
				maxExtent := 64.0
				if sp.Width > 0 {
					dim := float64(max(sp.Width, sp.Height))
					scale := math.Max(math.Abs(t.Scale.X), math.Abs(t.Scale.Y))
					maxExtent = dim*scale + 32.0
				}
				if !cam.IsPointVisible(Vector(t.Position), maxExtent) {
					continue
				}
			}

			img := tm.Get(sp.TextureID)
			if img == nil {
				continue
			}

			p := &particles[i]

			col := sp.Color
			col.A = uint8(float64(col.A) * sp.Opacity)

			blend := p.BlendMode
			if blend == (ebiten.Blend{}) {
				blend = ebiten.BlendSourceOver
			}

			dstW, dstH := float64(sp.Width), float64(sp.Height)
			if dstW == 0 || dstH == 0 {
				b := img.Bounds()
				if dstW == 0 {
					dstW = float64(b.Dx())
				}
				if dstH == 0 {
					dstH = float64(b.Dy())
				}
			}
			minX, minY, maxX, maxY := resolveSpriteSourceBounds(sp, img)

			rdr.AddQuadWithBlend(
				Vector(t.Position),
				Vector(t.Offset),
				Vector(t.Origin),
				Vector(t.Scale),
				t.Rotation,
				img,
				col,
				blend,
				minX, minY, maxX, maxY,
				dstW, dstH,
			)
		}
	}
}
