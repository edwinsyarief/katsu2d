package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/quadtree"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// FoliageComponent represents a single piece of foliage affected by wind and forces.
type FoliageComponent struct {
	// PivotPoint is a normalized point (0 to 1) relative to the sprite's dimensions,
	// which serves as the anchor for the sway.
	PivotPoint ebimath.Vector
	// OriginalVertices stores the initial, un-swayed positions of the vertices
	// to allow for resetting or calculating sway offsets.
	OriginalVertices     []ebiten.Vertex
	SwaySeed, SwayOffset float64
}

// NewFoliageComponent creates a new FoliageComponent.
func NewFoliageComponent(fixedPoint ebimath.Vector, swaySeed float64) *FoliageComponent {
	return &FoliageComponent{
		PivotPoint: fixedPoint,
		SwaySeed:   swaySeed,
	}
}

// FoliageControllerComponent is a central controller for all foliage in a scene.
type FoliageControllerComponent struct {
	MinRadius, MaxRadius float64
	windTime             float64
	windSpeed            float64
	windForce            float64
	windDirection        ebimath.Vector
	windScroll           ebimath.Vector
	externalForceSources []ForceSource
	strongWindGusts      []*StrongWindGust
	quadtree             *quadtree.Quadtree
	noiseImage           *ebiten.Image
	worldWidth           float64
	worldHeight          float64

	// Internal fields for options
	noiseMapSize   int
	noiseFrequency float64
}

// FoliageControllerOption is a functional option for FoliageControllerComponent.
type FoliageControllerOption func(*FoliageControllerComponent)

// WithNoiseMapSize sets the size of the Perlin noise map.
func WithFoliageNoiseMapSize(size int) FoliageControllerOption {
	return func(c *FoliageControllerComponent) {
		c.noiseMapSize = size
	}
}

// WithNoiseFrequency sets the frequency of the Perlin noise.
func WithFoliageNoiseFrequency(frequency float64) FoliageControllerOption {
	return func(c *FoliageControllerComponent) {
		c.noiseFrequency = frequency
	}
}

// WithWindSpeed sets the speed of the wind scrolling.
func WithFoliageWindSpeed(speed float64) FoliageControllerOption {
	return func(c *FoliageControllerComponent) {
		c.windSpeed = speed
	}
}

// WithWindForce sets the overall strength of the wind.
func WithFoliageWindForce(force float64) FoliageControllerOption {
	return func(c *FoliageControllerComponent) {
		c.windForce = force
	}
}

// NewFoliageControllerComponent creates a new FoliageControllerComponent.
// It uses a functional options pattern to allow for flexible configuration.
func NewFoliageControllerComponent(worldWidth, worldHeight float64, options ...FoliageControllerOption) *FoliageControllerComponent {
	controller := &FoliageControllerComponent{
		worldWidth:           worldWidth,
		worldHeight:          worldHeight,
		windTime:             0,
		windSpeed:            0.6,
		windForce:            1.0,
		windDirection:        ebimath.V(1, 0).Normalized(),
		windScroll:           ebimath.V(0, 0),
		externalForceSources: make([]ForceSource, 0),
		strongWindGusts:      make([]*StrongWindGust, 0),
		quadtree:             quadtree.New(ebimath.Rectangle{Min: ebimath.V(0, 0), Max: ebimath.V(worldWidth, worldHeight)}),
		noiseMapSize:         128,
		noiseFrequency:       100.0,
	}

	// Apply functional options.
	for _, opt := range options {
		opt(controller)
	}

	controller.noiseImage = utils.GeneratePerlinNoiseImage(controller.noiseMapSize, controller.noiseMapSize, controller.noiseFrequency)
	return controller
}

// AddForce adds a single frame point force.
func (self *FoliageControllerComponent) AddForce(position ebimath.Vector, radius, strength float64) {
	self.externalForceSources = append(self.externalForceSources, ForceSource{Position: position, Radius: radius, Strength: strength})
}

// AddStrongGust adds a strong wind gust.
func (self *FoliageControllerComponent) AddStrongGust(gust *StrongWindGust) {
	gust.Active = true
	self.strongWindGusts = append(self.strongWindGusts, gust)
}

// getWindForceAt samples the wind force at a given position from the noise map.
func (self *FoliageControllerComponent) getWindForceAt(x, y float64) float64 {
	sampleX := int(math.Mod(x+self.windScroll.X, float64(self.noiseMapSize)))
	sampleY := int(math.Mod(y+self.windScroll.Y, float64(self.noiseMapSize)))
	if sampleX < 0 {
		sampleX += self.noiseMapSize
	}
	if sampleY < 0 {
		sampleY += self.noiseMapSize
	}
	noiseColor := self.noiseImage.At(sampleX, sampleY)
	r, _, _, _ := noiseColor.RGBA()
	return float64(r) / 65535.0
}
