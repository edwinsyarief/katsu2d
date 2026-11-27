package katsu2d

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ParticleFadeMode defines the alpha fading behavior of a particle over its lifetime.
type ParticleFadeMode int

const (
	// ParticleFadeModeNone keeps the particle's initial alpha constant.
	ParticleFadeModeNone ParticleFadeMode = iota
	// ParticleFadeModeFadeIn fades the particle alpha from 0 to its initial alpha.
	ParticleFadeModeFadeIn
	// ParticleFadeModeFadeOut fades the particle alpha from its initial alpha to its target alpha.
	ParticleFadeModeFadeOut
	// ParticleFadeModeFadeInOut fades the particle alpha in for the first half of its life and out for the second half.
	ParticleFadeModeFadeInOut
)

// ParticleScaleMode defines the scaling behavior of a particle over its lifetime.
type ParticleScaleMode int

const (
	// ParticleScaleModeNone keeps the particle at its initial scale.
	ParticleScaleModeNone ParticleScaleMode = iota
	// ParticleScaleModeScaleIn scales the particle from 0 up to its initial scale.
	ParticleScaleModeScaleIn
	// ParticleScaleModeScaleOut scales the particle from its initial scale to its target scale.
	ParticleScaleModeScaleOut
	// ParticleScaleModeScaleInOut scales the particle in during the first half of its life and to target scale in the second half.
	ParticleScaleModeScaleInOut
)

// ParticleDirectionMode defines the directional motion modifier applied to a particle.
type ParticleDirectionMode int

const (
	// ParticleDirectionModeLinear moves the particle solely under velocity and gravity.
	ParticleDirectionModeLinear ParticleDirectionMode = iota
	// ParticleDirectionModeZigZag oscillates the particle perpendicular to its velocity.
	ParticleDirectionModeZigZag
	// ParticleDirectionModeNoise perturbs the particle velocity using 2D OpenSimplex noise.
	ParticleDirectionModeNoise
)

// ParticleComponent stores self-contained POD state for an individual particle entity.
// It holds no pointers so archetypes stay cache-friendly and particles survive emitter removal or preset changes.
type ParticleComponent struct {
	Gravity, Velocity                               Vector
	Lifetime, TotalLifetime                         float64
	InitialColor, TargetColor                       color.RGBA
	InitialScale, TargetScale                       float64
	InitialRotation, TargetRotation, RotationSpeed  float64
	NoiseOffsetX, NoiseOffsetY                      float64
	ZigZagFrequency, ZigZagMagnitude, NoiseFactor   float64
	BlendMode                                       ebiten.Blend
	FadeMode                                        ParticleFadeMode
	ScaleMode                                       ParticleScaleMode
	DirectionMode                                   ParticleDirectionMode
	UseRotationSpeed                                bool
}

// ParticleEmitterComponent configures how an emitter spawns and initializes particles.
type ParticleEmitterComponent struct {
	Active              bool
	BurstCount          int
	EmitRate            float64
	MaxParticles        int
	ParticleLifetime    float64
	ParticleSpawnOffset Vector

	InitialParticleSpeedMin float64
	InitialParticleSpeedMax float64

	InitialColorMin, InitialColorMax color.RGBA
	TargetColorMin, TargetColorMax   color.RGBA

	Gravity    Vector
	TextureIDs []int
	BlendMode  ebiten.Blend

	MinScale, MaxScale             float64
	TargetScaleMin, TargetScaleMax float64

	MinRotation, MaxRotation           float64
	EndRotationMin, EndRotationMax     float64
	RotationSpeedMin, RotationSpeedMax float64

	FadeMode      ParticleFadeMode
	ScaleMode     ParticleScaleMode
	DirectionMode ParticleDirectionMode

	ZigZagFrequency float64
	ZigZagMagnitude float64
	NoiseFactor     float64

	SpawnCounter float64
}

// NewParticleEmitterComponent creates a ParticleEmitterComponent with sensible defaults
// (white colors, BlendSourceOver, unit scale, and linear/none modes).
func NewParticleEmitterComponent(textureIDs []int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		TextureIDs:      textureIDs,
		InitialColorMin: color.RGBA{255, 255, 255, 255},
		InitialColorMax: color.RGBA{255, 255, 255, 255},
		TargetColorMin:  color.RGBA{255, 255, 255, 255},
		TargetColorMax:  color.RGBA{255, 255, 255, 255},
		BlendMode:       ebiten.BlendSourceOver,
		MinScale:        1.0,
		MaxScale:        1.0,
		TargetScaleMin:  1.0,
		TargetScaleMax:  1.0,
		FadeMode:        ParticleFadeModeNone,
		ScaleMode:       ParticleScaleModeNone,
		DirectionMode:   ParticleDirectionModeLinear,
	}
}
