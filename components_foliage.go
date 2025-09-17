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
	// windTime is a continuously increasing value used to drive the time-based wind simulation.
	windTime float64
	// windDirection is the dominant vector for the ambient wind, normalized to a unit vector.
	windDirection ebimath.Vector
	// windForce controls the maximum amplitude of the wind-induced sway.
	windForce float64
	// windSpeed determines how quickly the wind effect progresses over time.
	windSpeed float64
	// rippleStrength sets the intensity of the rippling effect that travels through the foliage.
	rippleStrength float64
	// noise is the Perlin noise generator used to create complex, natural-looking wind patterns.
	noise opensimplex.Noise
}

// FoliageOption is a functional option type for configuring a FoliageControllerComponent.
// This allows for flexible and readable component initialization.
type FoliageOption func(*FoliageControllerComponent)

// WithFoliageWindForce sets the maximum amplitude of the foliage's sway due to wind.
func WithFoliageWindForce(force float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windForce = force
	}
}

// WithFoliageWindSpeed sets how fast the foliage sways due to the wind effect.
func WithFoliageWindSpeed(speed float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windSpeed = speed
	}
}

// WithFoliageWindDirection sets the dominant direction of the ambient wind. The vector will be normalized.
func WithFoliageWindDirection(x, y float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windDirection = ebimath.Vector{X: x, Y: y}.Normalize()
	}
}

// WithFoliageRippleStrength sets the strength of the rippling effect that travels through the foliage.
func WithFoliageRippleStrength(strength float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.rippleStrength = strength
	}
}

// NewFoliageControllerComponent creates and initializes a new foliage controller.
// It sets default values and applies any provided options.
func NewFoliageControllerComponent(opts ...FoliageOption) *FoliageControllerComponent {
	// Initialize a new Perlin noise generator with a random seed based on the current time.
	// Alpha, Beta, and N control the frequency, amplitude, and number of octaves,
	// which define the complexity and texture of the wind patterns.
	noiseGenerator := opensimplex.New(time.Now().UnixNano())

	c := &FoliageControllerComponent{
		windDirection:  ebimath.Vector{X: 1.0, Y: 0.0},
		windForce:      10.0, // Default pixels of sway
		windSpeed:      1.0,
		rippleStrength: 1.0,
		windTime:       0,
		noise:          noiseGenerator, // Assign the newly created noise generator
	}
	// Apply all functional options to customize the component.
	for _, opt := range opts {
		opt(c)
	}
	return c
}
