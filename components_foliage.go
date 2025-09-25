package katsu2d

import (
	"time"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/opensimplex"
)

// FoliageComponent stores the unique state and properties for a single piece of foliage.
// To use it, add this component along with SpriteComponent and TransformComponent to an entity.
type FoliageComponent struct {
	// TextureID is the identifier for the foliage sprite's texture.
	TextureID int
	// SwaySeed is a random seed used to create a unique, pseudo-random sway for this foliage.
	SwaySeed float64
	// Angle is the current rotational angle of the foliage sprite, in radians.
	Angle float64
	// Pivot is the point around which the foliage rotates when swaying.
	Pivot ebimath.Vector
}

// FoliageControllerComponent manages the overall state and logic for the foliage simulation.
// It handles wind effects and other global parameters that affect all foliage.
type FoliageControllerComponent struct {
	// WindTime is a continuously increasing value used to drive the time-based wind simulation.
	WindTime float64
	// WindDirection is the dominant vector for the ambient wind, normalized to a unit vector.
	WindDirection ebimath.Vector
	// WindForce controls the maximum amplitude of the wind-induced sway.
	WindForce float64
	// WindSpeed determines how quickly the wind effect progresses over time.
	WindSpeed float64
	// RippleStrength sets the intensity of the rippling effect that travels through the foliage.
	RippleStrength float64
	// Noise is the Perlin noise generator used to create complex, natural-looking wind patterns.
	Noise opensimplex.Noise
}

// FoliageOption is a functional option type for configuring a FoliageControllerComponent.
// This allows for flexible and readable component initialization.
type FoliageOption func(*FoliageControllerComponent)

// WithFoliageWindForce sets the maximum amplitude of the foliage's sway due to wind.
func WithFoliageWindForce(force float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.WindForce = force
	}
}

// WithFoliageWindSpeed sets how fast the foliage sways due to the wind effect.
func WithFoliageWindSpeed(speed float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.WindSpeed = speed
	}
}

// WithFoliageWindDirection sets the dominant direction of the ambient wind. The vector will be normalized.
func WithFoliageWindDirection(x, y float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.WindDirection = ebimath.Vector{X: x, Y: y}.Normalize()
	}
}

// WithFoliageRippleStrength sets the strength of the rippling effect that travels through the foliage.
func WithFoliageRippleStrength(strength float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.RippleStrength = strength
	}
}

func NewFoliageControllerComponent(opts ...FoliageOption) *FoliageControllerComponent {
	res := &FoliageControllerComponent{}
	res.Noise = opensimplex.New(time.Now().UnixNano())
	res.WindDirection = ebimath.V(1, 0)
	res.WindForce = 10
	res.WindSpeed = 1
	res.RippleStrength = 1
	res.WindTime = 0

	// Apply all functional options to customize the component.
	for _, opt := range opts {
		opt(res)
	}

	return res
}
