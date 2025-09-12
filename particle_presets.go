package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// FirePreset returns a configured emitter for a fire effect.
func FirePreset(textureID int) *ParticleEmitterComponent {
	emitter := NewParticleEmitterComponent([]int{textureID})
	emitter.Active = true
	emitter.EmitRate = 200
	emitter.MaxParticles = 500
	emitter.ParticleLifetime = 1.0
	emitter.InitialParticleSpeedMin = 3
	emitter.InitialParticleSpeedMax = 5
	emitter.ParticleSpawnOffset = ebimath.V(8, 8)
	emitter.InitialColorMin = color.RGBA{255, 255, 0, 255} // Yellow
	emitter.InitialColorMax = color.RGBA{255, 128, 0, 255} // Orange
	emitter.TargetColorMin = color.RGBA{255, 0, 0, 0}      // Red, transparent
	emitter.TargetColorMax = color.RGBA{255, 0, 0, 0}      // Red, transparent
	emitter.FadeMode = ParticleFadeModeFadeOut
	emitter.BlendMode = ebiten.BlendLighter // Additive blending for a glowing effect
	emitter.MinScale = 1
	emitter.MaxScale = 1.5
	emitter.MinRotation = 0
	emitter.MaxRotation = 1.5 * math.Pi
	emitter.Gravity = ebimath.V(0, -100)
	return emitter
}

// RainPreset returns a configured emitter for a rain effect.
func RainPreset(textureID int) *ParticleEmitterComponent {
	emitter := NewParticleEmitterComponent([]int{textureID})
	emitter.Active = true
	emitter.EmitRate = 200
	emitter.MaxParticles = 1000
	emitter.ParticleLifetime = 1.0
	emitter.InitialParticleSpeedMin = 200
	emitter.InitialParticleSpeedMax = 300
	emitter.ParticleSpawnOffset = ebimath.V(100, 0) // Wide spawn area
	emitter.InitialColorMin = color.RGBA{100, 150, 255, 200}
	emitter.InitialColorMax = color.RGBA{100, 150, 255, 200}
	emitter.TargetColorMin = color.RGBA{100, 150, 255, 0}
	emitter.TargetColorMax = color.RGBA{100, 150, 255, 0}
	emitter.FadeMode = ParticleFadeModeFadeOut
	emitter.Gravity = ebimath.V(0, 150) // Gravity pulls particles down
	emitter.BlendMode = ebiten.BlendSourceOver
	emitter.MinScale = 0.2
	emitter.MaxScale = 0.5
	emitter.ScaleMode = ParticleScaleModeNone
	emitter.MinRotation = 0
	emitter.MaxRotation = 0
	return emitter
}

// SmokePreset returns a configured emitter for a smoke effect.
func SmokePreset(textureID int) *ParticleEmitterComponent {
	emitter := NewParticleEmitterComponent([]int{textureID})
	emitter.Active = true
	emitter.EmitRate = 10
	emitter.MaxParticles = 100
	emitter.ParticleLifetime = 5.0
	emitter.InitialParticleSpeedMin = 10
	emitter.InitialParticleSpeedMax = 20
	emitter.ParticleSpawnOffset = ebimath.V(5, 5)
	emitter.InitialColorMin = color.RGBA{150, 150, 150, 100}
	emitter.InitialColorMax = color.RGBA{200, 200, 200, 100}
	emitter.TargetColorMin = color.RGBA{50, 50, 50, 0}
	emitter.TargetColorMax = color.RGBA{50, 50, 50, 0}
	emitter.FadeMode = ParticleFadeModeFadeOut
	emitter.BlendMode = ebiten.BlendSourceOver
	emitter.MinScale = 0.5
	emitter.MaxScale = 2.0
	emitter.MinRotation = 0
	emitter.MaxRotation = 2 * math.Pi
	return emitter
}

// ExplosionPreset returns a configured emitter for a one-time explosion effect.
func ExplosionPreset(textureID int) *ParticleEmitterComponent {
	emitter := NewParticleEmitterComponent([]int{textureID})
	emitter.Active = false
	emitter.BurstCount = 100
	emitter.MaxParticles = 100
	emitter.ParticleLifetime = 0.8
	emitter.InitialParticleSpeedMin = 50
	emitter.InitialParticleSpeedMax = 100
	emitter.InitialColorMin = color.RGBA{255, 150, 0, 255}
	emitter.InitialColorMax = color.RGBA{255, 255, 0, 255}
	emitter.TargetColorMin = color.RGBA{255, 0, 0, 0}
	emitter.TargetColorMax = color.RGBA{255, 0, 0, 0}
	emitter.FadeMode = ParticleFadeModeFadeOut
	emitter.BlendMode = ebiten.BlendLighter
	emitter.MinScale = 0.5
	emitter.MaxScale = 1.5
	emitter.MinRotation = 0
	emitter.MaxRotation = 2 * math.Pi
	return emitter
}

// WhimsicalPreset returns a configured emitter for a whimsical effect.
func WhimsicalPreset(textureID int) *ParticleEmitterComponent {
	emitter := NewParticleEmitterComponent([]int{textureID})
	emitter.Active = true
	emitter.EmitRate = 50
	emitter.MaxParticles = 200
	emitter.ParticleLifetime = 3.0
	emitter.InitialParticleSpeedMin = 20
	emitter.InitialParticleSpeedMax = 40
	emitter.ParticleSpawnOffset = ebimath.V(10, 10)
	emitter.InitialColorMin = color.RGBA{255, 0, 255, 255} // Magenta
	emitter.InitialColorMax = color.RGBA{0, 255, 255, 255} // Cyan
	emitter.TargetColorMin = color.RGBA{255, 255, 0, 0}    // Yellow, transparent
	emitter.TargetColorMax = color.RGBA{0, 255, 0, 0}      // Green, transparent
	emitter.FadeMode = ParticleFadeModeFadeInOut
	emitter.ScaleMode = ParticleScaleModeScaleInOut
	emitter.MinScale = 0.2
	emitter.MaxScale = 0.8
	emitter.TargetScaleMin = 0.1
	emitter.TargetScaleMax = 0.4
	emitter.DirectionMode = ParticleDirectionModeNoise
	emitter.NoiseFactor = 0.1
	emitter.RotationSpeedMin = -math.Pi
	emitter.RotationSpeedMax = math.Pi
	emitter.BlendMode = ebiten.BlendLighter
	return emitter
}
