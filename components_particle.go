package katsu2d

import (
	"image/color"
	"time"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// ParticleFadeMode defines the fading behavior of a particle.
type ParticleFadeMode int

const (
	// ParticleFadeModeNone means the particle will not fade.
	ParticleFadeModeNone ParticleFadeMode = iota
	// ParticleFadeModeFadeIn means the particle will fade in.
	ParticleFadeModeFadeIn
	// ParticleFadeModeFadeOut means the particle will fade out.
	ParticleFadeModeFadeOut
	// ParticleFadeModeFadeInOut means the particle will fade in and then out.
	ParticleFadeModeFadeInOut
)

// ParticleScaleMode defines the scaling behavior of a particle.
type ParticleScaleMode int

const (
	// ParticleScaleModeNone means the particle will not scale.
	ParticleScaleModeNone ParticleScaleMode = iota
	// ParticleScaleModeScaleIn means the particle will scale in.
	ParticleScaleModeScaleIn
	// ParticleScaleModeScaleOut means the particle will scale out.
	ParticleScaleModeScaleOut
	// ParticleScaleModeScaleInOut means the particle will scale in and then out.
	ParticleScaleModeScaleInOut
)

// ParticleDirectionMode defines the direction behavior of a particle.
type ParticleDirectionMode int

const (
	// ParticleDirectionModeLinear means the particle will move in a straight line.
	ParticleDirectionModeLinear ParticleDirectionMode = iota
	// ParticleDirectionModeZigZag means the particle will move in a zig-zag pattern.
	ParticleDirectionModeZigZag
	// ParticleDirectionModeNoise means the particle will move randomly using Perlin noise.
	ParticleDirectionModeNoise
)

// ParticleComponent represents an individual particle in the particle system
// It contains all the properties needed for a single particle's behavior and appearance
type ParticleComponent struct {
	Gravity, Velocity ebimath.Vector // Physics properties for particle movement
	Lifetime          float64        // Current lifetime of the particle
	TotalLifetime     float64        // Maximum lifetime of the particle
	InitialColor      color.RGBA     // Starting color of the particle
	TargetColor       color.RGBA     // Final color the particle will interpolate to
	InitialScale      float64        // Starting scale of the particle
	TargetScale       float64        // Final scale the particle will grow/shrink to
	InitialRotation   float64        // Starting rotation angle in radians
	TargetRotation    float64        // Final rotation angle the particle will rotate to
	RotationSpeed     float64        // Speed of rotation
	NoiseOffsetX      float64        // Noise offset for random direction
	NoiseOffsetY      float64        // Noise offset for random direction
}

// ParticleEmitterComponent manages the emission and properties of multiple particles
// It controls how particles are spawned, their appearance, and their behavior
type ParticleEmitterComponent struct {
	Active              bool           // Whether the emitter is currently emitting particles
	BurstCount          int            // Number of particles to emit in a single burst
	EmitRate            float64        // Particles per second to emit
	MaxParticles        int            // Maximum number of particles allowed at once
	ParticleLifetime    float64        // How long each particle lives
	ParticleSpawnOffset ebimath.Vector // Offset from emitter position where particles spawn

	// Velocity range for new particles
	InitialParticleSpeedMin float64
	InitialParticleSpeedMax float64

	// Color ranges for particle interpolation
	InitialColorMin, InitialColorMax color.RGBA // Starting color range
	TargetColorMin, TargetColorMax   color.RGBA // Ending color range

	Gravity    ebimath.Vector // Gravity force applied to particles
	TextureIDs []int          // Available textures for particles
	BlendMode  ebiten.Blend   // How particles blend with the background

	// Scale ranges for particles
	MinScale, MaxScale             float64 // Initial scale range
	TargetScaleMin, TargetScaleMax float64 // Final scale range

	// Rotation ranges for particles (in radians)
	MinRotation, MaxRotation           float64 // Initial rotation range
	EndRotationMin, EndRotationMax     float64 // Final rotation range
	RotationSpeedMin, RotationSpeedMax float64

	// Fade, Scale, Direction modes
	FadeMode      ParticleFadeMode
	ScaleMode     ParticleScaleMode
	DirectionMode ParticleDirectionMode

	// ZigZag movement properties
	ZigZagFrequency float64
	ZigZagMagnitude float64

	// Noise movement properties
	NoiseFactor float64

	// Internal state
	LastEmitTime time.Time // Tracks last particle emission time
	SpawnCounter float64   // Accumulates partial particle spawn credit
}

// NewParticleEmitterComponent creates a new particle emitter with default settings
// textureIDs: slice of texture identifiers available for particles to use
func NewParticleEmitterComponent(textureIDs []int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		TextureIDs:   textureIDs,
		LastEmitTime: time.Now(),
		// Default colors set to white
		InitialColorMin: color.RGBA{255, 255, 255, 255},
		InitialColorMax: color.RGBA{255, 255, 255, 255},
		TargetColorMin:  color.RGBA{255, 255, 255, 255},
		TargetColorMax:  color.RGBA{255, 255, 255, 255},
		// Default blend mode and transformations
		BlendMode:      ebiten.BlendSourceOver,
		MinScale:       1.0,
		MaxScale:       1.0,
		TargetScaleMin: 1.0,
		TargetScaleMax: 1.0,
		MinRotation:    0,
		MaxRotation:    0,
		EndRotationMin: 0,
		EndRotationMax: 0,
		// Default modes
		FadeMode:      ParticleFadeModeNone,
		ScaleMode:     ParticleScaleModeNone,
		DirectionMode: ParticleDirectionModeLinear,
	}
}
