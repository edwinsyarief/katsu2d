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
	// Loop over all entities that have a Transform and an Emitter component.
	for _, entity := range world.Query(CTTransform, CTParticleEmitter) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		emitter, _ := world.GetComponent(entity, CTParticleEmitter)
		em := emitter.(*ParticleEmitterComponent)

		// Determine the number of particles to spawn this frame.
		particlesToSpawn := 0
		if em.Active {
			// Continuous emission
			em.spawnCounter += em.EmitRate * dt
			particlesToSpawn = int(em.spawnCounter)
			em.spawnCounter -= float64(particlesToSpawn)
		} else if em.BurstCount > 0 {
			// One-time burst
			particlesToSpawn = em.BurstCount
			em.BurstCount = 0 // Reset burst count after spawning
		}

		// Spawn new particles, respecting the max particle limit.
		if particlesToSpawn > 0 {
			ptEntities := world.Query(CTParticle)
			currentParticles := len(ptEntities)
			for i := 0; i < particlesToSpawn; i++ {
				if currentParticles+i >= em.MaxParticles {
					break
				}
				self.spawnParticle(world, t, em)
			}
		}
	}
}

// spawnParticle creates a single new particle entity and configures its components.
func (self *ParticleEmitterSystem) spawnParticle(world *World, emitterTransform *TransformComponent, emitter *ParticleEmitterComponent) {
	texID := 0
	if len(emitter.TextureIDs) > 1 {
		ebimath.RandomChoose(self.r, emitter.TextureIDs)
	} else {
		texID = emitter.TextureIDs[0]
	}

	tex := self.tm.Get(texID)
	width, height := tex.Bounds().Dx(), tex.Bounds().Dy()
	// Create a new entity and retrieve components from the pool for efficiency.
	newParticle := world.CreateEntity()
	particleTransform := GetTransformComponent()
	particleSprite := GetSpriteComponent()
	particleData := GetParticleComponent()

	// Particle must have the same Z with the emitter
	particleTransform.Z = emitterTransform.Z

	// Randomize particle properties based on the emitter's configuration.
	randAngle := self.r.Float64() * 2 * math.Pi // 0 to 2*pi
	randSpeed := self.r.FloatRange(
		emitter.InitialParticleSpeedMin,
		emitter.InitialParticleSpeedMax)
	randOffset := ebimath.V(self.r.Float64()*emitter.ParticleSpawnOffset.X, self.r.Float64()*emitter.ParticleSpawnOffset.Y)

	// Set initial particle state.
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

	// Correctly setting the initial and target scale and rotation based on the emitter's ranges.
	particleData.InitialScale = particleTransform.Scale().X
	particleData.TargetScale = ebimath.Lerp(emitter.TargetScaleMin, emitter.TargetScaleMax, self.r.Float64())
	particleData.InitialRotation = particleTransform.Rotation()
	particleData.TargetRotation = ebimath.Lerp(emitter.EndRotationMin, emitter.EndRotationMax, self.r.Float64())

	particleData.Gravity = emitter.Gravity

	// Configure the sprite component.
	particleSprite.TextureID = texID
	particleSprite.Color = particleData.InitialColor
	particleSprite.SrcW = float32(width)
	particleSprite.SrcH = float32(height)

	world.AddComponent(newParticle, particleTransform)
	world.AddComponent(newParticle, particleSprite)
	world.AddComponent(newParticle, particleData)
}

// ParticleUpdateSystem is an UpdateSystem that handles the movement and lifecycle of particles.
type ParticleUpdateSystem struct {
}

// NewParticleUpdateSystem creates a new system for updating particles.
func NewParticleUpdateSystem() *ParticleUpdateSystem {
	return &ParticleUpdateSystem{}
}

// Update implements the UpdateSystem interface. It moves particles, handles fading, and removes expired particles.
func (self *ParticleUpdateSystem) Update(world *World, dt float64) {
	// A list to hold entities that should be removed at the end of the frame.
	var toRemove []Entity

	for _, entity := range world.Query(CTTransform, CTParticle, CTSprite) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		// Update position and velocity.
		p.Velocity = p.Velocity.Add(p.Gravity.MulF(dt))
		t.SetPosition(t.Position().Add(p.Velocity.MulF(dt)))
		p.Lifetime -= dt

		// Interpolate color, scale, and rotation.
		tp := 1.0 - (p.Lifetime / p.TotalLifetime) // normalized time from 0 to 1
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

	// Remove expired particles and return components to the pool.
	for _, entity := range toRemove {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		PutTransformComponent(t)
		PutParticleComponent(p)
		PutSpriteComponent(s) // Return the sprite component to the pool!
	}

	// Batch remove expired particles
	world.BatchRemoveEntities(toRemove...)
}

// ParticleRenderSystem is a DrawSystem that renders all particles to the screen.
type ParticleRenderSystem struct {
	tm *TextureManager
}

// NewParticleRenderSystem creates a new system for rendering particles.
// It requires a TextureManager to access particle textures.
func NewParticleRenderSystem(tm *TextureManager) *ParticleRenderSystem {
	return &ParticleRenderSystem{tm: tm}
}

// Draw implements the DrawSystem interface. It renders all particles using the BatchRenderer.
func (self *ParticleRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	// Query for all entities that have a Transform and a Sprite, and are also particles.
	// This ensures we only render active particles.
	for _, entity := range world.Query(CTTransform, CTSprite, CTParticle) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
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
