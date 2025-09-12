package katsu2d

import (
	"math"
	"time"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/opensimplex"
	"github.com/edwinsyarief/katsu2d/utils"
)

// ParticleEmitterSystem is an UpdateSystem that handles spawning new particles.
type ParticleEmitterSystem struct {
	tm *TextureManager
	r  *ebimath.Rand
}

// NewParticleEmitterSystem creates a new system for particle emission.
func NewParticleEmitterSystem(tm *TextureManager) *ParticleEmitterSystem {
	return &ParticleEmitterSystem{
		tm: tm,
		r:  ebimath.Random(),
	}
}

// Update implements the UpdateSystem interface to handle particle spawning.
func (self *ParticleEmitterSystem) Update(world *World, dt float64) {
	for _, entity := range world.Query(CTTransform, CTParticleEmitter) {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		emitter, _ := world.GetComponent(entity, CTParticleEmitter)
		em := emitter.(*ParticleEmitterComponent)

		particlesToSpawn := 0
		if em.Active {
			em.spawnCounter += em.EmitRate * dt
			particlesToSpawn = int(em.spawnCounter)
			em.spawnCounter -= float64(particlesToSpawn)
		} else if em.BurstCount > 0 {
			particlesToSpawn = em.BurstCount
			em.BurstCount = 0
		}

		if particlesToSpawn > 0 {
			ptEntities := world.Query(CTParticle)
			currentParticles := len(ptEntities)
			for i := 0; i < particlesToSpawn; i++ {
				if currentParticles+i >= em.MaxParticles {
					break
				}
				self.spawnParticle(world, entity, t, em)
			}
		}
	}
}

// spawnParticle creates a single new particle entity and configures its components.
func (self *ParticleEmitterSystem) spawnParticle(world *World, emitterEntity Entity, emitterTransform *TransformComponent, emitter *ParticleEmitterComponent) {
	texID := 0
	if len(emitter.TextureIDs) > 1 {
		ebimath.RandomChoose(self.r, emitter.TextureIDs)
	} else {
		texID = emitter.TextureIDs[0]
	}

	tex := self.tm.Get(texID)
	newParticle := world.CreateEntity()
	particleTransform := GetTransformComponent()
	particleSprite := GetSpriteComponent()
	particleData := GetParticleComponent()
	parentComponent := NewParentComponent(emitterEntity)
	particleOrderable := &OrderableComponent{}

	if parentOrderableAny, ok := world.GetComponent(emitterEntity, CTOrderable); ok {
		parentOrderable := parentOrderableAny.(*OrderableComponent)
		particleOrderable.SetIndex(parentOrderable.Index())
	}

	particleTransform.Z = emitterTransform.Z
	randAngle := self.r.Float64() * 2 * math.Pi
	randSpeed := self.r.FloatRange(
		emitter.InitialParticleSpeedMin,
		emitter.InitialParticleSpeedMax)
	randOffset := ebimath.V(self.r.Float64()*emitter.ParticleSpawnOffset.X, self.r.Float64()*emitter.ParticleSpawnOffset.Y)

	particleTransform.SetPosition(emitterTransform.Position().Add(randOffset))
	particleTransform.SetRotation(ebimath.Lerp(emitter.MinRotation, emitter.MaxRotation, self.r.Float64()))
	particleTransform.SetScale(ebimath.V(
		ebimath.Lerp(emitter.MinScale, emitter.MaxScale, self.r.Float64()),
		ebimath.Lerp(emitter.MinScale, emitter.MaxScale, self.r.Float64()),
	))

	particleData.Velocity = ebimath.V(randSpeed*math.Cos(randAngle), randSpeed*math.Sin(randAngle))
	particleData.Lifetime = emitter.ParticleLifetime
	particleData.TotalLifetime = emitter.ParticleLifetime
	particleData.InitialColor = utils.GetInterpolatedColor(emitter.InitialColorMin, emitter.InitialColorMax)
	particleData.TargetColor = utils.GetInterpolatedColor(emitter.TargetColorMin, emitter.TargetColorMax)
	particleData.InitialScale = particleTransform.Scale().X
	particleData.TargetScale = ebimath.Lerp(emitter.TargetScaleMin, emitter.TargetScaleMax, self.r.Float64())
	particleData.InitialRotation = particleTransform.Rotation()
	particleData.TargetRotation = ebimath.Lerp(emitter.EndRotationMin, emitter.EndRotationMax, self.r.Float64())
	particleData.RotationSpeed = ebimath.Lerp(emitter.RotationSpeedMin, emitter.RotationSpeedMax, self.r.Float64())
	particleData.Gravity = emitter.Gravity
	particleData.NoiseOffsetX = self.r.Float64() * 1000
	particleData.NoiseOffsetY = self.r.Float64() * 1000

	particleSprite.TextureID = texID
	particleSprite.Color = particleData.InitialColor
	bounds := tex.Bounds()
	particleSprite.SrcRect = &bounds
	particleSprite.DstW = float32(tex.Bounds().Dx())
	particleSprite.DstH = float32(tex.Bounds().Dy())

	world.AddComponent(newParticle, particleTransform)
	world.AddComponent(newParticle, particleSprite)
	world.AddComponent(newParticle, particleData)
	world.AddComponent(newParticle, parentComponent)
	world.AddComponent(newParticle, particleOrderable)
}

// ParticleUpdateSystem is an UpdateSystem that handles the movement and lifecycle of particles.
type ParticleUpdateSystem struct {
	noise opensimplex.Noise
	time  float64
}

// NewParticleUpdateSystem creates a new system for updating particles.
func NewParticleUpdateSystem() *ParticleUpdateSystem {
	return &ParticleUpdateSystem{
		noise: opensimplex.New(time.Now().Unix()),
	}
}

// Update implements the UpdateSystem interface.
func (self *ParticleUpdateSystem) Update(world *World, dt float64) {
	self.time += dt
	var toRemove []Entity
	for _, entity := range world.Query(CTTransform, CTParticle, CTSprite, CTParent) {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)
		parent, _ := world.GetComponent(entity, CTParent)
		parentComponent := parent.(*ParentComponent)
		emitter, ok := world.GetComponent(parentComponent.Parent, CTParticleEmitter)
		if !ok {
			continue
		}
		em := emitter.(*ParticleEmitterComponent)

		p.Velocity = p.Velocity.Add(p.Gravity.ScaleF(dt))

		switch em.DirectionMode {
		case ParticleDirectionModeZigZag:
			perp := ebimath.V(-p.Velocity.Y, p.Velocity.X).Normalize()
			zigZag := math.Sin(self.time*em.ZigZagFrequency) * em.ZigZagMagnitude
			p.Velocity = p.Velocity.Add(perp.ScaleF(zigZag))
		case ParticleDirectionModeNoise:
			noiseValX := self.noise.Eval2(p.NoiseOffsetX, self.time*em.NoiseFactor)
			noiseValY := self.noise.Eval2(p.NoiseOffsetY, self.time*em.NoiseFactor)
			p.Velocity = p.Velocity.Add(ebimath.V(noiseValX, noiseValY))
		}

		t.SetPosition(t.Position().Add(p.Velocity.ScaleF(dt)))
		p.Lifetime -= dt

		tp := 1.0 - (p.Lifetime / p.TotalLifetime)

		// Fading
		initialA := p.InitialColor.A
		targetA := p.TargetColor.A
		var currentA uint8
		switch em.FadeMode {
		case ParticleFadeModeFadeIn:
			currentA = uint8(ebimath.Lerp(0, float64(initialA), tp))
		case ParticleFadeModeFadeOut:
			currentA = uint8(ebimath.Lerp(float64(initialA), float64(targetA), tp))
		case ParticleFadeModeFadeInOut:
			if tp < 0.5 {
				currentA = uint8(ebimath.Lerp(0, float64(initialA), tp*2))
			} else {
				currentA = uint8(ebimath.Lerp(float64(initialA), float64(targetA), (tp-0.5)*2))
			}
		default: // ParticleFadeModeNone
			currentA = initialA
		}
		s.Color = utils.LerpPremultipliedRGBA(p.InitialColor, p.TargetColor, tp)
		s.Color.A = currentA

		// Scaling
		var currentScale float64
		switch em.ScaleMode {
		case ParticleScaleModeScaleIn:
			currentScale = ebimath.Lerp(0, p.InitialScale, tp)
		case ParticleScaleModeScaleOut:
			currentScale = ebimath.Lerp(p.InitialScale, p.TargetScale, tp)
		case ParticleScaleModeScaleInOut:
			if tp < 0.5 {
				currentScale = ebimath.Lerp(0, p.InitialScale, tp*2)
			} else {
				currentScale = ebimath.Lerp(p.InitialScale, p.TargetScale, (tp-0.5)*2)
			}
		default: // ParticleScaleModeNone
			currentScale = p.InitialScale
		}
		t.SetScale(ebimath.V(currentScale, currentScale))

		// Rotation
		if em.RotationSpeedMin != 0 || em.RotationSpeedMax != 0 {
			t.SetRotation(t.Rotation() + p.RotationSpeed*dt)
		} else {
			t.SetRotation(ebimath.Lerp(p.InitialRotation, p.TargetRotation, tp))
		}

		if p.Lifetime <= 0 {
			toRemove = append(toRemove, entity)
		}
	}

	for _, entity := range toRemove {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		PutTransformComponent(t)
		PutParticleComponent(p)
		PutSpriteComponent(s)
	}
	world.BatchRemoveEntities(toRemove...)
}

// ParticleRenderSystem is a DrawSystem that renders all particles to the screen.
type ParticleRenderSystem struct {
	tm *TextureManager
}

// NewParticleRenderSystem creates a new system for rendering particles.
func NewParticleRenderSystem(tm *TextureManager) *ParticleRenderSystem {
	return &ParticleRenderSystem{tm: tm}
}

// Draw implements the DrawSystem interface.
func (self *ParticleRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	for _, entity := range world.Query(CTTransform, CTSprite, CTParticle) {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		//imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
		srcRect := s.GetSourceRect()
		effColor := s.Color
		effColor.A = uint8(float32(s.Color.A) * s.Opacity)
		realPos := ebimath.V2(0).Apply(t.Matrix())
		if !t.Origin().IsZero() {
			realPos = realPos.Sub(t.Origin())
		}
		renderer.AddQuad(
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
