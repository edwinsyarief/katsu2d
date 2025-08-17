package katsu2d

import (
	"image/color"
	"time"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

type ParticleComponent struct {
	Gravity, Velocity ebimath.Vector
	Lifetime          float64
	TotalLifetime     float64
	InitialColor      color.RGBA
	TargetColor       color.RGBA
	InitialScale      float64
	TargetScale       float64
	InitialRotation   float64
	TargetRotation    float64
}

type ParticleEmitterComponent struct {
	Active                                           bool
	BurstCount                                       int
	EmitRate                                         float64
	MaxParticles                                     int
	ParticleLifetime                                 float64
	ParticleSpawnOffset                              ebimath.Vector
	InitialParticleSpeedMin, InitialParticleSpeedMax float64
	InitialColorMin, InitialColorMax                 color.RGBA
	TargetColorMin, TargetColorMax                   color.RGBA
	FadeOut                                          bool
	Gravity                                          ebimath.Vector
	TextureIDs                                       []int
	BlendMode                                        ebiten.Blend
	MinScale, MaxScale                               float64
	TargetScaleMin, TargetScaleMax                   float64
	MinRotation, MaxRotation                         float64
	EndRotationMin, EndRotationMax                   float64
	lastEmitTime                                     time.Time
	spawnCounter                                     float64
}

func NewParticleEmitterComponent(textureIDs []int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		TextureIDs:      textureIDs,
		lastEmitTime:    time.Now(),
		InitialColorMin: color.RGBA{255, 255, 255, 255},
		InitialColorMax: color.RGBA{255, 255, 255, 255},
		TargetColorMin:  color.RGBA{255, 255, 255, 255},
		TargetColorMax:  color.RGBA{255, 255, 255, 255},
		BlendMode:       ebiten.BlendSourceOver,
		MinScale:        1.0,
		MaxScale:        1.0,
		TargetScaleMin:  1.0,
		TargetScaleMax:  1.0,
		MinRotation:     0,
		MaxRotation:     0,
		EndRotationMin:  0,
		EndRotationMax:  0,
	}
}
