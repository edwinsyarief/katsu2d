package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

func (self *ParticleEmitterComponent) FirePreset(textureID int) {
	self.Init([]int{textureID})
	self.Active = true
	self.EmitRate = 200
	self.MaxParticles = 500
	self.ParticleLifetime = 1.0
	self.InitialParticleSpeedMin = 3
	self.InitialParticleSpeedMax = 5
	self.ParticleSpawnOffset = ebimath.V(8, 8)
	self.InitialColorMin = color.RGBA{255, 255, 0, 255} // Yellow
	self.InitialColorMax = color.RGBA{255, 128, 0, 255} // Orange
	self.TargetColorMin = color.RGBA{255, 0, 0, 0}      // Red, transparent
	self.TargetColorMax = color.RGBA{255, 0, 0, 0}      // Red, transparent
	self.FadeMode = ParticleFadeModeFadeOut
	self.BlendMode = ebiten.BlendLighter // Additive blending for a glowing effect
	self.MinScale = 1
	self.MaxScale = 1.5
	self.MinRotation = 0
	self.MaxRotation = 1.5 * math.Pi
	self.Gravity = ebimath.V(0, -100)
}

func (self *ParticleEmitterComponent) RainPreset(textureID int) {
	self.Init([]int{textureID})
	self.Active = true
	self.EmitRate = 200
	self.MaxParticles = 1000
	self.ParticleLifetime = 1.0
	self.InitialParticleSpeedMin = 200
	self.InitialParticleSpeedMax = 300
	self.ParticleSpawnOffset = ebimath.V(100, 0) // Wide spawn area
	self.InitialColorMin = color.RGBA{100, 150, 255, 200}
	self.InitialColorMax = color.RGBA{100, 150, 255, 200}
	self.TargetColorMin = color.RGBA{100, 150, 255, 0}
	self.TargetColorMax = color.RGBA{100, 150, 255, 0}
	self.FadeMode = ParticleFadeModeFadeOut
	self.Gravity = ebimath.V(0, 150) // Gravity pulls particles down
	self.BlendMode = ebiten.BlendSourceOver
	self.MinScale = 0.2
	self.MaxScale = 0.5
	self.ScaleMode = ParticleScaleModeNone
	self.MinRotation = 0
	self.MaxRotation = 0
}

func (self *ParticleEmitterComponent) SmokePreset(textureIDs []int) {
	self.Init(textureIDs)
	self.Active = true
	self.EmitRate = 10
	self.MaxParticles = 100
	self.ParticleLifetime = 5.0
	self.InitialParticleSpeedMin = 10
	self.InitialParticleSpeedMax = 20
	self.ParticleSpawnOffset = ebimath.V(5, 5)
	self.InitialColorMin = color.RGBA{150, 150, 150, 100}
	self.InitialColorMax = color.RGBA{200, 200, 200, 100}
	self.TargetColorMin = color.RGBA{50, 50, 50, 0}
	self.TargetColorMax = color.RGBA{50, 50, 50, 0}
	self.FadeMode = ParticleFadeModeFadeOut
	self.BlendMode = ebiten.BlendSourceOver
	self.MinScale = 0.5
	self.MaxScale = 2.0
	self.MinRotation = 0
	self.MaxRotation = 2 * math.Pi
}

func (self *ParticleEmitterComponent) ExplosionPreset(textureIDs []int) {
	self.Init(textureIDs)
	self.Active = false
	self.BurstCount = 100
	self.MaxParticles = 100
	self.ParticleLifetime = 0.8
	self.InitialParticleSpeedMin = 50
	self.InitialParticleSpeedMax = 100
	self.InitialColorMin = color.RGBA{255, 150, 0, 255}
	self.InitialColorMax = color.RGBA{255, 255, 0, 255}
	self.TargetColorMin = color.RGBA{255, 0, 0, 0}
	self.TargetColorMax = color.RGBA{255, 0, 0, 0}
	self.FadeMode = ParticleFadeModeFadeOut
	self.BlendMode = ebiten.BlendLighter
	self.MinScale = 0.5
	self.MaxScale = 1.5
	self.MinRotation = 0
	self.MaxRotation = 2 * math.Pi
}

func (self *ParticleEmitterComponent) WhimsicalPreset(textureID int) {
	self.Init([]int{textureID})
	self.Active = true
	self.EmitRate = 50
	self.MaxParticles = 200
	self.ParticleLifetime = 3.0
	self.InitialParticleSpeedMin = 20
	self.InitialParticleSpeedMax = 40
	self.ParticleSpawnOffset = ebimath.V(10, 10)
	self.InitialColorMin = color.RGBA{255, 0, 255, 255} // Magenta
	self.InitialColorMax = color.RGBA{0, 255, 255, 255} // Cyan
	self.TargetColorMin = color.RGBA{255, 255, 0, 0}    // Yellow, transparent
	self.TargetColorMax = color.RGBA{0, 255, 0, 0}      // Green, transparent
	self.FadeMode = ParticleFadeModeFadeInOut
	self.ScaleMode = ParticleScaleModeScaleInOut
	self.MinScale = 0.2
	self.MaxScale = 0.8
	self.TargetScaleMin = 0.1
	self.TargetScaleMax = 0.4
	self.DirectionMode = ParticleDirectionModeNoise
	self.NoiseFactor = 0.1
	self.RotationSpeedMin = -math.Pi
	self.RotationSpeedMax = math.Pi
	self.BlendMode = ebiten.BlendLighter
}
