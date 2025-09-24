package katsu2d

import (
	"math"
	"time"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/opensimplex"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/edwinsyarief/lazyecs"
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
func (self *ParticleEmitterSystem) Update(world *lazyecs.World, dt float64) {
	query1 := world.Query(CTParticle)
	currentParticles := 0
	for query1.Next() {
		currentParticles += query1.Count()
	}

	query2 := world.Query(CTTransform, CTParticleEmitter)
	for query2.Next() {
		for _, entity := range query2.Entities() {
			t, _ := lazyecs.GetComponent[TransformComponent](world, entity)
			em, _ := lazyecs.GetComponent[ParticleEmitterComponent](world, entity)

			particlesToSpawn := 0
			if em.Active {
				em.SpawnCounter += em.EmitRate * dt
				particlesToSpawn = int(em.SpawnCounter)
				em.SpawnCounter -= float64(particlesToSpawn)
			} else if em.BurstCount > 0 {
				particlesToSpawn = em.BurstCount
				em.BurstCount = 0
			}

			if particlesToSpawn > 0 {
				for i := 0; i < particlesToSpawn; i++ {
					if currentParticles+i >= em.MaxParticles {
						break
					}
					self.spawnParticle(world, entity, t, em)
				}
			}
		}
	}
}

// spawnParticle creates a single new particle entity and configures its components.
func (self *ParticleEmitterSystem) spawnParticle(world *lazyecs.World, emitterEntity lazyecs.Entity, emitterTransform *TransformComponent, emitter *ParticleEmitterComponent) {
	texID := 0
	if len(emitter.TextureIDs) > 1 {
		ebimath.RandomChoose(self.r, emitter.TextureIDs)
	} else {
		texID = emitter.TextureIDs[0]
	}

	newParticleEntity := world.CreateEntity()

	// Add all components first
	lazyecs.AddComponent[TransformComponent](world, newParticleEntity)
	lazyecs.AddComponent[SpriteComponent](world, newParticleEntity)
	lazyecs.AddComponent[ParticleComponent](world, newParticleEntity)
	lazyecs.AddComponent[ParentComponent](world, newParticleEntity)
	lazyecs.AddComponent[OrderableComponent](world, newParticleEntity)

	// Now get pointers to the actual components in storage
	parentComponent, _ := lazyecs.GetComponent[ParentComponent](world, newParticleEntity)
	parentComponent.Init(emitterEntity)
	particleTransform, _ := lazyecs.GetComponent[TransformComponent](world, newParticleEntity)
	particleTransform.Init()
	particleSprite, _ := lazyecs.GetComponent[SpriteComponent](world, newParticleEntity)
	tex := self.tm.Get(texID)
	particleSprite.Init(texID, tex.Bounds())
	particleData, _ := lazyecs.GetComponent[ParticleComponent](world, newParticleEntity)

	particleOrderable, _ := lazyecs.GetComponent[OrderableComponent](world, newParticleEntity)
	if parentOrderable, ok := lazyecs.GetComponent[OrderableComponent](world, emitterEntity); ok {
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

	particleSprite.Color = particleData.InitialColor
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
func (self *ParticleUpdateSystem) Update(world *lazyecs.World, dt float64) {
	self.time += dt
	query := world.Query(CTTransform, CTParticle, CTSprite, CTParent)
	for query.Next() {
		for _, entity := range query.Entities() {
			transform, _ := lazyecs.GetComponent[TransformComponent](world, entity)
			particle, _ := lazyecs.GetComponent[ParticleComponent](world, entity)
			sprite, _ := lazyecs.GetComponent[SpriteComponent](world, entity)
			parent, _ := lazyecs.GetComponent[ParentComponent](world, entity)
			emitter, _ := lazyecs.GetComponent[ParticleEmitterComponent](world, parent.Parent)

			particle.Velocity = particle.Velocity.Add(particle.Gravity.ScaleF(dt))

			switch emitter.DirectionMode {
			case ParticleDirectionModeZigZag:
				perp := ebimath.V(-particle.Velocity.Y, particle.Velocity.X).Normalize()
				zigZag := math.Sin(self.time*emitter.ZigZagFrequency) * emitter.ZigZagMagnitude
				particle.Velocity = particle.Velocity.Add(perp.ScaleF(zigZag))
			case ParticleDirectionModeNoise:
				noiseValX := self.noise.Eval2(particle.NoiseOffsetX, self.time*emitter.NoiseFactor)
				noiseValY := self.noise.Eval2(particle.NoiseOffsetY, self.time*emitter.NoiseFactor)
				particle.Velocity = particle.Velocity.Add(ebimath.V(noiseValX, noiseValY))
			}

			transform.SetPosition(transform.Position().Add(particle.Velocity.ScaleF(dt)))
			particle.Lifetime -= dt

			tp := 1.0 - (particle.Lifetime / particle.TotalLifetime)

			// Fading
			initialA := particle.InitialColor.A
			targetA := particle.TargetColor.A
			var currentA uint8
			switch emitter.FadeMode {
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
			sprite.Color = utils.LerpPremultipliedRGBA(particle.InitialColor, particle.TargetColor, tp)
			sprite.Color.A = currentA

			// Scaling
			var currentScale float64
			switch emitter.ScaleMode {
			case ParticleScaleModeScaleIn:
				currentScale = ebimath.Lerp(0, particle.InitialScale, tp)
			case ParticleScaleModeScaleOut:
				currentScale = ebimath.Lerp(particle.InitialScale, particle.TargetScale, tp)
			case ParticleScaleModeScaleInOut:
				if tp < 0.5 {
					currentScale = ebimath.Lerp(0, particle.InitialScale, tp*2)
				} else {
					currentScale = ebimath.Lerp(particle.InitialScale, particle.TargetScale, (tp-0.5)*2)
				}
			default: // ParticleScaleModeNone
				currentScale = particle.InitialScale
			}
			transform.SetScale(ebimath.V(currentScale, currentScale))

			// Rotation
			if emitter.RotationSpeedMin != 0 || emitter.RotationSpeedMax != 0 {
				transform.SetRotation(transform.Rotation() + particle.RotationSpeed*dt)
			} else {
				transform.SetRotation(ebimath.Lerp(particle.InitialRotation, particle.TargetRotation, tp))
			}

			if particle.Lifetime <= 0 {
				world.RemoveEntity(entity)
			}
		}
	}
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
func (self *ParticleRenderSystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	query := world.Query(CTTransform, CTSprite, CTParticle)
	for query.Next() {
		transforms, _ := lazyecs.GetComponentSlice[TransformComponent](query)
		sprites, _ := lazyecs.GetComponentSlice[SpriteComponent](query)

		for i, t := range transforms {
			s := sprites[i]
			img := self.tm.Get(s.TextureID)
			if img == nil {
				continue
			}

			srcRect := s.SrcRect
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
}
