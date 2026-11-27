package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// FirePreset creates a glowing additive fire particle emitter preset.
func FirePreset(textureID int) *ParticleEmitterComponent {
	em := NewParticleEmitterComponent([]int{textureID})
	em.Active = true
	em.EmitRate = 200
	em.MaxParticles = 500
	em.ParticleLifetime = 1.0
	em.InitialParticleSpeedMin = 3
	em.InitialParticleSpeedMax = 5
	em.ParticleSpawnOffset = V(8, 8)
	em.InitialColorMin = color.RGBA{255, 255, 0, 255}
	em.InitialColorMax = color.RGBA{255, 128, 0, 255}
	em.TargetColorMin = color.RGBA{255, 0, 0, 0}
	em.TargetColorMax = color.RGBA{255, 0, 0, 0}
	em.FadeMode = ParticleFadeModeFadeOut
	em.BlendMode = ebiten.BlendLighter
	em.MinScale = 1
	em.MaxScale = 1.5
	em.MinRotation = 0
	em.MaxRotation = 1.5 * math.Pi
	em.Gravity = V(0, -100)
	return em
}

// RainPreset creates a downward-falling rain particle emitter preset.
func RainPreset(textureID int) *ParticleEmitterComponent {
	em := NewParticleEmitterComponent([]int{textureID})
	em.Active = true
	em.EmitRate = 200
	em.MaxParticles = 1000
	em.ParticleLifetime = 1.0
	em.InitialParticleSpeedMin = 200
	em.InitialParticleSpeedMax = 300
	em.ParticleSpawnOffset = V(100, 0)
	em.InitialColorMin = color.RGBA{100, 150, 255, 200}
	em.InitialColorMax = color.RGBA{100, 150, 255, 200}
	em.TargetColorMin = color.RGBA{100, 150, 255, 0}
	em.TargetColorMax = color.RGBA{100, 150, 255, 0}
	em.FadeMode = ParticleFadeModeFadeOut
	em.Gravity = V(0, 150)
	em.BlendMode = ebiten.BlendSourceOver
	em.MinScale = 0.2
	em.MaxScale = 0.5
	em.ScaleMode = ParticleScaleModeNone
	em.MinRotation = 0
	em.MaxRotation = 0
	return em
}

// SmokePreset creates a soft fading smoke particle emitter preset.
func SmokePreset(textureIDs []int) *ParticleEmitterComponent {
	em := NewParticleEmitterComponent(textureIDs)
	em.Active = true
	em.EmitRate = 10
	em.MaxParticles = 100
	em.ParticleLifetime = 5.0
	em.InitialParticleSpeedMin = 10
	em.InitialParticleSpeedMax = 20
	em.ParticleSpawnOffset = V(5, 5)
	em.InitialColorMin = color.RGBA{150, 150, 150, 100}
	em.InitialColorMax = color.RGBA{200, 200, 200, 100}
	em.TargetColorMin = color.RGBA{50, 50, 50, 0}
	em.TargetColorMax = color.RGBA{50, 50, 50, 0}
	em.FadeMode = ParticleFadeModeFadeOut
	em.BlendMode = ebiten.BlendSourceOver
	em.MinScale = 0.5
	em.MaxScale = 2.0
	em.MinRotation = 0
	em.MaxRotation = 2 * math.Pi
	return em
}

// ExplosionPreset creates a one-shot burst explosion particle emitter preset.
func ExplosionPreset(textureIDs []int) *ParticleEmitterComponent {
	em := NewParticleEmitterComponent(textureIDs)
	em.Active = false
	em.BurstCount = 100
	em.MaxParticles = 100
	em.ParticleLifetime = 0.8
	em.InitialParticleSpeedMin = 50
	em.InitialParticleSpeedMax = 100
	em.InitialColorMin = color.RGBA{255, 150, 0, 255}
	em.InitialColorMax = color.RGBA{255, 255, 0, 255}
	em.TargetColorMin = color.RGBA{255, 0, 0, 0}
	em.TargetColorMax = color.RGBA{255, 0, 0, 0}
	em.FadeMode = ParticleFadeModeFadeOut
	em.BlendMode = ebiten.BlendLighter
	em.MinScale = 0.5
	em.MaxScale = 1.5
	em.MinRotation = 0
	em.MaxRotation = 2 * math.Pi
	return em
}

// WhimsicalPreset creates a swirling, noise-driven glowing particle emitter preset.
func WhimsicalPreset(textureID int) *ParticleEmitterComponent {
	em := NewParticleEmitterComponent([]int{textureID})
	em.Active = true
	em.EmitRate = 50
	em.MaxParticles = 200
	em.ParticleLifetime = 3.0
	em.InitialParticleSpeedMin = 20
	em.InitialParticleSpeedMax = 40
	em.ParticleSpawnOffset = V(10, 10)
	em.InitialColorMin = color.RGBA{255, 0, 255, 255}
	em.InitialColorMax = color.RGBA{0, 255, 255, 255}
	em.TargetColorMin = color.RGBA{255, 255, 0, 0}
	em.TargetColorMax = color.RGBA{0, 255, 0, 0}
	em.FadeMode = ParticleFadeModeFadeInOut
	em.ScaleMode = ParticleScaleModeScaleInOut
	em.MinScale = 0.2
	em.MaxScale = 0.8
	em.TargetScaleMin = 0.1
	em.TargetScaleMax = 0.4
	em.DirectionMode = ParticleDirectionModeNoise
	em.NoiseFactor = 0.1
	em.RotationSpeedMin = -math.Pi
	em.RotationSpeedMax = math.Pi
	em.BlendMode = ebiten.BlendLighter
	return em
}
