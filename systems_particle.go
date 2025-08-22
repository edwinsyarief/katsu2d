package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
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
	if emitter.EnableScaling {
		particleData.TargetScale = ebimath.Lerp(emitter.TargetScaleMin, emitter.TargetScaleMax, self.r.Float64())
	} else {
		particleData.TargetScale = particleData.InitialScale
	}
	particleData.InitialRotation = particleTransform.Rotation()
	particleData.TargetRotation = ebimath.Lerp(emitter.EndRotationMin, emitter.EndRotationMax, self.r.Float64())
	particleData.Gravity = emitter.Gravity

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
type ParticleUpdateSystem struct{}

// NewParticleUpdateSystem creates a new system for updating particles.
func NewParticleUpdateSystem() *ParticleUpdateSystem {
	return &ParticleUpdateSystem{}
}

// Update implements the UpdateSystem interface.
func (self *ParticleUpdateSystem) Update(world *World, dt float64) {
	var toRemove []Entity
	for _, entity := range world.Query(CTTransform, CTParticle, CTSprite) {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		p.Velocity = p.Velocity.Add(p.Gravity.MulF(dt))
		t.SetPosition(t.Position().Add(p.Velocity.MulF(dt)))
		p.Lifetime -= dt

		tp := 1.0 - (p.Lifetime / p.TotalLifetime)
		s.Color = utils.LerpPremultipliedRGBA(p.InitialColor, p.TargetColor, tp)

		t.SetScale(ebimath.V(
			ebimath.Lerp(p.InitialScale, p.TargetScale, tp),
			ebimath.Lerp(p.InitialScale, p.TargetScale, tp),
		))
		t.SetRotation(ebimath.Lerp(p.InitialRotation, p.TargetRotation, tp))

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
