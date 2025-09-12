package katsu2d

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
