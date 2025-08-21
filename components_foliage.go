package katsu2d

import (
	ebimath "github.com/edwinsyarief/ebi-math"
)

// FoliageComponent holds the data for a single piece of foliage.
// To create a foliage entity, add FoliageComponent, SpriteComponent, and TransformComponent to an entity.
// The SpriteComponent should be configured with a grid mesh (e.g., by using sprite.SetGrid(rows, cols)).
type FoliageComponent struct {
	TextureID int
	SwaySeed  float64
	Angle     float64
	Pivot     ebimath.Vector
}

// FoliageControllerComponent holds the configuration and state of the foliage system.
type FoliageControllerComponent struct {
	windTime      float64
	windDirection ebimath.Vector
	windForce     float64
	windSpeed     float64
}

// FoliageOption is a function type for configuring FoliageControllerComponent.
type FoliageOption func(*FoliageControllerComponent)

// WithFoliageWindForce sets the maximum amplitude of wind-induced sway.
func WithFoliageWindForce(force float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windForce = force
	}
}

// WithFoliageWindSpeed sets how fast the foliage sways due to wind.
func WithFoliageWindSpeed(speed float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windSpeed = speed
	}
}

// WithFoliageWindDirection sets the dominant direction of the wind. The vector will be normalized.
func WithFoliageWindDirection(x, y float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.windDirection = ebimath.Vector{X: x, Y: y}.Normalized()
	}
}

// NewFoliageControllerComponent creates and initializes a new foliage controller component.
func NewFoliageControllerComponent(opts ...FoliageOption) *FoliageControllerComponent {
	c := &FoliageControllerComponent{
		windDirection: ebimath.Vector{X: 1.0, Y: 0.0},
		windForce:     10.0, // Pixels of sway
		windSpeed:     1.0,
		windTime:      0,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
