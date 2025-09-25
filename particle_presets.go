package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

func FirePreset(textureID int) *ParticleEmitterComponent {
	res := NewParticleEmitterComponent([]int{textureID})
	res.Active = true
	res.EmitRate = 200
	res.MaxParticles = 500
	res.ParticleLifetime = 1.0
	res.InitialParticleSpeedMin = 3
	res.InitialParticleSpeedMax = 5
	res.ParticleSpawnOffset = ebimath.V(8, 8)
	res.InitialColorMin = color.RGBA{255, 255, 0, 255} // Yellow
	res.InitialColorMax = color.RGBA{255, 128, 0, 255} // Orange
	res.TargetColorMin = color.RGBA{255, 0, 0, 0}      // Red, transparent
	res.TargetColorMax = color.RGBA{255, 0, 0, 0}      // Red, transparent
	res.FadeMode = ParticleFadeModeFadeOut
	res.BlendMode = ebiten.BlendLighter // Additive blending for a glowing effect
	res.MinScale = 1
	res.MaxScale = 1.5
	res.MinRotation = 0
	res.MaxRotation = 1.5 * math.Pi
	res.Gravity = ebimath.V(0, -100)
	return res
}

func RainPreset(textureID int) *ParticleEmitterComponent {
	res := NewParticleEmitterComponent([]int{textureID})
	res.Active = true
	res.EmitRate = 200
	res.MaxParticles = 1000
	res.ParticleLifetime = 1.0
	res.InitialParticleSpeedMin = 200
	res.InitialParticleSpeedMax = 300
	res.ParticleSpawnOffset = ebimath.V(100, 0) // Wide spawn area
	res.InitialColorMin = color.RGBA{100, 150, 255, 200}
	res.InitialColorMax = color.RGBA{100, 150, 255, 200}
	res.TargetColorMin = color.RGBA{100, 150, 255, 0}
	res.TargetColorMax = color.RGBA{100, 150, 255, 0}
	res.FadeMode = ParticleFadeModeFadeOut
	res.Gravity = ebimath.V(0, 150) // Gravity pulls particles down
	res.BlendMode = ebiten.BlendSourceOver
	res.MinScale = 0.2
	res.MaxScale = 0.5
	res.ScaleMode = ParticleScaleModeNone
	res.MinRotation = 0
	res.MaxRotation = 0
	return res
}

func SmokePreset(textureIDs []int) *ParticleEmitterComponent {
	res := NewParticleEmitterComponent(textureIDs)
	res.Active = true
	res.EmitRate = 10
	res.MaxParticles = 100
	res.ParticleLifetime = 5.0
	res.InitialParticleSpeedMin = 10
	res.InitialParticleSpeedMax = 20
	res.ParticleSpawnOffset = ebimath.V(5, 5)
	res.InitialColorMin = color.RGBA{150, 150, 150, 100}
	res.InitialColorMax = color.RGBA{200, 200, 200, 100}
	res.TargetColorMin = color.RGBA{50, 50, 50, 0}
	res.TargetColorMax = color.RGBA{50, 50, 50, 0}
	res.FadeMode = ParticleFadeModeFadeOut
	res.BlendMode = ebiten.BlendSourceOver
	res.MinScale = 0.5
	res.MaxScale = 2.0
	res.MinRotation = 0
	res.MaxRotation = 2 * math.Pi
	return res
}

func ExplosionPreset(textureIDs []int) *ParticleEmitterComponent {
	res := NewParticleEmitterComponent(textureIDs)
	res.Active = false
	res.BurstCount = 100
	res.MaxParticles = 100
	res.ParticleLifetime = 0.8
	res.InitialParticleSpeedMin = 50
	res.InitialParticleSpeedMax = 100
	res.InitialColorMin = color.RGBA{255, 150, 0, 255}
	res.InitialColorMax = color.RGBA{255, 255, 0, 255}
	res.TargetColorMin = color.RGBA{255, 0, 0, 0}
	res.TargetColorMax = color.RGBA{255, 0, 0, 0}
	res.FadeMode = ParticleFadeModeFadeOut
	res.BlendMode = ebiten.BlendLighter
	res.MinScale = 0.5
	res.MaxScale = 1.5
	res.MinRotation = 0
	res.MaxRotation = 2 * math.Pi
	return res
}

func WhimsicalPreset(textureID int) *ParticleEmitterComponent {
	res := NewParticleEmitterComponent([]int{textureID})
	res.Active = true
	res.EmitRate = 50
	res.MaxParticles = 200
	res.ParticleLifetime = 3.0
	res.InitialParticleSpeedMin = 20
	res.InitialParticleSpeedMax = 40
	res.ParticleSpawnOffset = ebimath.V(10, 10)
	res.InitialColorMin = color.RGBA{255, 0, 255, 255} // Magenta
	res.InitialColorMax = color.RGBA{0, 255, 255, 255} // Cyan
	res.TargetColorMin = color.RGBA{255, 255, 0, 0}    // Yellow, transparent
	res.TargetColorMax = color.RGBA{0, 255, 0, 0}      // Green, transparent
	res.FadeMode = ParticleFadeModeFadeInOut
	res.ScaleMode = ParticleScaleModeScaleInOut
	res.MinScale = 0.2
	res.MaxScale = 0.8
	res.TargetScaleMin = 0.1
	res.TargetScaleMax = 0.4
	res.DirectionMode = ParticleDirectionModeNoise
	res.NoiseFactor = 0.1
	res.RotationSpeedMin = -math.Pi
	res.RotationSpeedMax = math.Pi
	res.BlendMode = ebiten.BlendLighter
	return res
}
