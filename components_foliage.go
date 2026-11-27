package katsu2d

import (
	"math"

	"github.com/edwinsyarief/katsu2d/opensimplex"
)

const (
	foliageNoiseSize = 64
	foliageNoiseMask = foliageNoiseSize - 1
)

// FoliageComponent stores per-entity state and cached vertex bounds for foliage wind simulation.
// Pair this component with TransformComponent, SpriteComponent, and MeshComponent on an entity.
type FoliageComponent struct {
	// SwaySeed is a seed used to create a unique, pseudo-random sway for this foliage.
	SwaySeed float64
	// Pivot is the normalized pivot point in [0..1] around which the foliage sways.
	// When zero, it defaults to (0.5, 1.0) (bottom-center).
	Pivot Vector

	cachedMinX      float32
	cachedMinY      float32
	cachedWidth     float64
	cachedHeight    float64
	cachedVertCount int
	cachedPivot     Vector
	cachedWeights   []float64
}

// NewFoliageComponent creates a FoliageComponent with the given sway seed and normalized pivot.
// A zero pivot defaults to bottom-center (0.5, 1.0).
func NewFoliageComponent(swaySeed float64, pivot Vector) FoliageComponent {
	if pivot.X == 0 && pivot.Y == 0 {
		pivot = V(0.5, 1.0)
	}
	return FoliageComponent{
		SwaySeed: swaySeed,
		Pivot:    pivot,
	}
}

// FoliageControllerComponent manages global wind parameters, interactive forces,
// and vertex shading for all foliage entities.
type FoliageControllerComponent struct {
	// WindTime is a continuously increasing value used to drive the time-based wind simulation.
	WindTime float64
	// WindDirection is the dominant vector for the ambient wind, normalized to a unit vector.
	WindDirection Vector
	// WindForce controls the maximum amplitude of the wind-induced sway.
	WindForce float64
	// WindSpeed determines how quickly the wind effect progresses over time.
	WindSpeed float64
	// RippleStrength sets the intensity of the rippling effect that travels through the foliage.
	RippleStrength float64
	// ShadingStrength controls subtle base-to-tip ambient occlusion and wind-crest brightness.
	ShadingStrength float64
	// ExternalForceSources holds radial external force emitters affecting foliage in world space.
	ExternalForceSources []ForceSource
	// StrongWindGusts holds active localized traveling wind gusts affecting foliage in world space.
	StrongWindGusts []StrongWindGust
	// Noise is the OpenSimplex noise generator used to build the tiled wind lookup table.
	Noise opensimplex.Noise

	noiseTable []float64
}

// FoliageOption is a functional option for configuring a FoliageControllerComponent.
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

// WithFoliageWindDirection sets the dominant direction of the ambient wind. The vector is normalized.
func WithFoliageWindDirection(x, y float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.WindDirection = V(x, y).Normalize()
	}
}

// WithFoliageRippleStrength sets the strength of the rippling effect that travels through the foliage.
func WithFoliageRippleStrength(strength float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.RippleStrength = strength
	}
}

// WithFoliageShading sets the strength of base-to-tip ambient occlusion and wind-crest vertex shading.
func WithFoliageShading(strength float64) FoliageOption {
	return func(c *FoliageControllerComponent) {
		c.ShadingStrength = strength
	}
}

// NewFoliageControllerComponent creates a FoliageControllerComponent with default wind settings
// and applies any provided functional options.
func NewFoliageControllerComponent(opts ...FoliageOption) *FoliageControllerComponent {
	res := &FoliageControllerComponent{
		WindDirection:   V(1, 0),
		WindForce:       10,
		WindSpeed:       1,
		RippleStrength:  1,
		ShadingStrength: 0.22,
		Noise:           opensimplex.New(Random().PositiveInt64()),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(res)
		}
	}
	res.noiseTable = buildFoliageNoiseTable(res.Noise)
	return res
}

// SetForcePositions replaces the active external force sources while reusing slice capacity.
func (c *FoliageControllerComponent) SetForcePositions(sources ...ForceSource) {
	c.ExternalForceSources = append(c.ExternalForceSources[:0], sources...)
}

// AddStrongWindGust normalizes the gust direction, activates it, and appends it to the simulation.
func (c *FoliageControllerComponent) AddStrongWindGust(gust StrongWindGust) {
	gust.Direction = gust.EndPos.Sub(gust.StartPos).Normalize()
	gust.Active = true
	c.StrongWindGusts = append(c.StrongWindGusts, gust)
}

func buildFoliageNoiseTable(n opensimplex.Noise) []float64 {
	if n == nil {
		return nil
	}
	table := make([]float64, foliageNoiseSize*foliageNoiseSize)
	const twoPiOverSize = (2.0 * math.Pi) / float64(foliageNoiseSize)
	const radius = 1.6
	for y := range foliageNoiseSize {
		v := float64(y) * twoPiOverSize
		sv, cv := math.Sincos(v)
		z := cv * radius
		w := sv * radius
		row := y * foliageNoiseSize
		for x := range foliageNoiseSize {
			u := float64(x) * twoPiOverSize
			su, cu := math.Sincos(u)
			table[row+x] = n.Eval4(cu*radius, su*radius, z, w)
		}
	}
	return table
}

// sampleNoise performs an O(1) seamlessly tiled bilinear lookup into the 64x64 noise table.
func (c *FoliageControllerComponent) sampleNoise(x, y float64) float64 {
	if len(c.noiseTable) < foliageNoiseSize*foliageNoiseSize {
		return 0
	}

	const invSize = 1.0 / float64(foliageNoiseSize)
	sx := x - math.Floor(x*invSize)*foliageNoiseSize
	sy := y - math.Floor(y*invSize)*foliageNoiseSize
	if sx >= foliageNoiseSize || sx < 0 {
		sx = 0
	}
	if sy >= foliageNoiseSize || sy < 0 {
		sy = 0
	}

	x0 := int(sx) & foliageNoiseMask
	y0 := int(sy) & foliageNoiseMask
	fx := sx - float64(x0)
	fy := sy - float64(y0)
	x1 := (x0 + 1) & foliageNoiseMask
	y1 := (y0 + 1) & foliageNoiseMask

	row0 := y0 * foliageNoiseSize
	row1 := y1 * foliageNoiseSize
	v00 := c.noiseTable[row0+x0]
	v10 := c.noiseTable[row0+x1]
	v01 := c.noiseTable[row1+x0]
	v11 := c.noiseTable[row1+x1]

	top := v00 + (v10-v00)*fx
	bot := v01 + (v11-v01)*fx
	return top + (bot-top)*fy
}
