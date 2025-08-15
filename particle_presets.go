package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// FirePreset returns a configured emitter for a fire effect.
func FirePreset(textureID int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		Active:                  true,
		EmitRate:                50,
		MaxParticles:            500,
		ParticleLifetime:        1.5,
		InitialParticleSpeedMin: 10,
		InitialParticleSpeedMax: 20,
		ParticleSpawnOffset:     ebimath.V(5, 5),
		InitialColorMin:         color.RGBA{255, 255, 0, 255}, // Yellow
		InitialColorMax:         color.RGBA{255, 128, 0, 255}, // Orange
		TargetColorMin:          color.RGBA{255, 0, 0, 0},     // Red, transparent
		TargetColorMax:          color.RGBA{255, 0, 0, 0},     // Red, transparent
		FadeOut:                 true,
		BlendMode:               ebiten.BlendLighter, // Additive blending for a glowing effect
		MinScale:                0.5,
		MaxScale:                1.5,
		MinRotation:             0,
		MaxRotation:             2 * math.Pi,
		TextureIDs:              []int{textureID},
	}
}

// RainPreset returns a configured emitter for a rain effect.
func RainPreset(textureID int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		Active:                  true,
		EmitRate:                200,
		MaxParticles:            1000,
		ParticleLifetime:        1.0,
		InitialParticleSpeedMin: 200,
		InitialParticleSpeedMax: 300,
		ParticleSpawnOffset:     ebimath.V(100, 0), // Wide spawn area
		InitialColorMin:         color.RGBA{100, 150, 255, 200},
		InitialColorMax:         color.RGBA{100, 150, 255, 200},
		TargetColorMin:          color.RGBA{100, 150, 255, 0},
		TargetColorMax:          color.RGBA{100, 150, 255, 0},
		FadeOut:                 true,
		Gravity:                 ebimath.V(0, 150), // Gravity pulls particles down
		BlendMode:               ebiten.BlendSourceOver,
		MinScale:                0.2,
		MaxScale:                0.5,
		MinRotation:             0,
		MaxRotation:             0,
		TextureIDs:              []int{textureID},
	}
}

// SmokePreset returns a configured emitter for a smoke effect.
func SmokePreset(textureID int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		Active:                  true,
		EmitRate:                10,
		MaxParticles:            100,
		ParticleLifetime:        5.0,
		InitialParticleSpeedMin: 10,
		InitialParticleSpeedMax: 20,
		ParticleSpawnOffset:     ebimath.V(5, 5),
		InitialColorMin:         color.RGBA{150, 150, 150, 100},
		InitialColorMax:         color.RGBA{200, 200, 200, 100},
		TargetColorMin:          color.RGBA{50, 50, 50, 0},
		TargetColorMax:          color.RGBA{50, 50, 50, 0},
		FadeOut:                 true,
		BlendMode:               ebiten.BlendSourceOver,
		MinScale:                0.5,
		MaxScale:                2.0,
		MinRotation:             0,
		MaxRotation:             2 * math.Pi,
		TextureIDs:              []int{textureID},
	}
}

// ExplosionPreset returns a configured emitter for a one-time explosion effect.
func ExplosionPreset(textureID int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		Active:                  false,
		BurstCount:              100,
		MaxParticles:            100,
		ParticleLifetime:        0.8,
		InitialParticleSpeedMin: 50,
		InitialParticleSpeedMax: 100,
		InitialColorMin:         color.RGBA{255, 150, 0, 255},
		InitialColorMax:         color.RGBA{255, 255, 0, 255},
		TargetColorMin:          color.RGBA{255, 0, 0, 0},
		TargetColorMax:          color.RGBA{255, 0, 0, 0},
		FadeOut:                 true,
		BlendMode:               ebiten.BlendLighter,
		MinScale:                0.5,
		MaxScale:                1.5,
		MinRotation:             0,
		MaxRotation:             2 * math.Pi,
		TextureIDs:              []int{textureID},
	}
}
