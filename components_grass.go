package katsu2d

import (
	"math"
	"math/rand"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// GrassComponent stores the unique state for a single blade of grass.
// This includes its individual swaying behavior and external forces acting on it.
type GrassComponent struct {
	// SwaySeed is a random seed used to create a unique, natural-looking sway for this blade.
	SwaySeed float64
	// InteractionSway is the current sway value caused by external forces, such as a character walking through it.
	InteractionSway float64
	// SwayVelocity is the current rate of change for the sway, simulating momentum.
	SwayVelocity float64
	// AccumulatedForce is the total force currently applied to this grass blade from all sources.
	AccumulatedForce float64
	// CurrentSway is the total sway of the grass blade, combining natural wind and external forces.
	CurrentSway float64
}

// ForceSource defines a single point of external force that can affect grass.
// This is typically used to simulate objects interacting with the grass field.
type ForceSource struct {
	Position ebimath.Vector
	Radius   float64
	Strength float64
}

// StrongWindGust defines a temporary, localized gust of wind that moves through the world.
type StrongWindGust struct {
	StartPos        ebimath.Vector
	EndPos          ebimath.Vector
	Direction       ebimath.Vector
	Width           float64
	Length          float64
	Strength        float64
	Duration        float64
	FadeInDuration  float64
	FadeOutDuration float64
	ElapsedTime     float64
	Active          bool
}

// Area defines a rectangular region in the world where grass should be generated.
type Area struct {
	// TexturesIDs is a list of texture IDs to randomly choose from for grass in this area.
	TexturesIDs []int
	// X1, Y1, X2, Y2 define the tile coordinates of the rectangular area.
	X1, Y1, X2, Y2 int
}

// activeGustFrameData is a helper struct used to process a wind gust on a per-frame basis.
type activeGustFrameData struct {
	gust               *StrongWindGust
	pos                ebimath.Vector
	strengthMultiplier float64
}

// Option is a function type for configuring a GrassControllerComponent.
// This is a common pattern in Go for providing flexible initialization.
type GrassOption func(*GrassControllerComponent)

// WithGrassTileSize sets the size of the grid cells for grass placement.
func WithGrassTileSize(size int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.TileSize = size
	}
}

// WithGrassDensity sets the number of grass blades to generate per tile.
func WithGrassDensity(density int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.GrassDensity = density
	}
}

// WithGrassWindForce sets the maximum amplitude of wind-induced sway.
func WithGrassWindForce(force float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.WindForce = force
	}
}

// WithGrassWindSpeed sets how fast the grass sways due to the wind effect.
func WithGrassWindSpeed(speed float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.WindSpeed = speed
	}
}

// WithGrassWindDirection sets the dominant direction of the ambient wind.
// The provided vector will be normalized to a unit vector.
func WithGrassWindDirection(x, y float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.WindDirection = ebimath.Vector{X: x, Y: y}.Normalize()
	}
}

// WithGrassNoiseMapSize sets the dimension of the generated Perlin noise map used for wind.
func WithGrassNoiseMapSize(size int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.NoiseMapSize = size
	}
}

// WithGrassNoiseFrequency controls the "zoom" of the Perlin noise, affecting
// the size of the wind patterns. A lower frequency means larger, smoother patterns.
func WithGrassNoiseFrequency(freq float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.NoiseFrequency = freq
	}
}

// WithGrassForceBaseAcceleration sets the base acceleration for grass reaction to external forces.
// A higher value makes the grass react more quickly and stiffly.
func WithGrassForceBaseAcceleration(accel float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.ForceBaseAcceleration = accel
	}
}

// WithGrassSwaySpringStrength sets how strongly a grass blade tries to return to its upright position.
// This acts like a spring constant, pulling the grass back to its base state.
func WithGrassSwaySpringStrength(strength float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.SwaySpringStrength = strength
	}
}

// WithGrassSwayDamping sets the damping factor for the grass's recovery motion.
// This controls how quickly the sway oscillations decay.
func WithGrassSwayDamping(damping float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.SwayDamping = damping
	}
}

// WithGrassAreas sets the specific tile areas where grass should be generated.
// If no areas are provided, grass will be generated across the entire world.
func WithGrassAreas(areas []Area) GrassOption {
	return func(self *GrassControllerComponent) {
		self.GrassAreas = areas
	}
}

// WithGrassOrderable enables Z-sorting for grass sprites, allowing them to be drawn
// in front of or behind other sprites based on their Y-position.
func WithGrassOrderable(orderable bool) GrassOption {
	return func(self *GrassControllerComponent) {
		self.Orderable = orderable
	}
}

// GrassControllerComponent holds the configuration and overall state for the grass simulation system.
type GrassControllerComponent struct {
	WorldWidth   int
	WorldHeight  int
	TileSize     int
	GrassDensity int
	Quadtree     *Quadtree
	Tm           *TextureManager
	// ExternalForceSources are points of force (e.g., player's position) that affect nearby grass.
	ExternalForceSources []ForceSource
	// StrongWindGusts are temporary, moving wind effects.
	StrongWindGusts []*StrongWindGust
	// WindScroll represents the current offset for sampling the Perlin noise map,
	// creating a scrolling wind effect.
	WindScroll ebimath.Vector
	// WindTime is the elapsed time used to update the windScroll.
	WindTime              float64
	NoiseImage            *ebiten.Image
	NoiseMapSize          int
	NoiseFrequency        float64
	SwaySpringStrength    float64
	SwayDamping           float64
	ForceBaseAcceleration float64
	// GrassAreas specifies the regions for grass generation.
	GrassAreas []Area
	// RenderArea defines the current visible area of the world.
	RenderArea    ebimath.Rectangle
	WindDirection ebimath.Vector
	WindForce     float64
	WindSpeed     float64
	// TextureID is the default texture to use for grass blades.
	TextureID int
	// Orderable indicates whether the grass sprites should be Z-sorted for rendering.
	Orderable bool
	// Z is the Z-depth for rendering the grass.
	Z float64
}

// NewGrassControllerComponent creates and initializes a new grass controller.
// It sets up default values and applies any provided options.
func NewGrassControllerComponent(world *World, tm *TextureManager, worldWidth, worldHeight int, textureID int, z float64, opts ...GrassOption) *GrassControllerComponent {
	self := &GrassControllerComponent{
		WorldWidth:            worldWidth,
		WorldHeight:           worldHeight,
		TileSize:              32,
		GrassDensity:          20,
		Tm:                    tm,
		NoiseMapSize:          512,
		NoiseFrequency:        100.0,
		SwaySpringStrength:    0.5,
		SwayDamping:           0.5,
		ForceBaseAcceleration: 800.0,
		WindDirection:         ebimath.Vector{X: 1.0, Y: 0.0},
		WindForce:             0.3,
		WindSpeed:             0.5,
		TextureID:             textureID,
		Z:                     z,
	}
	// Apply all functional options to configure the component.
	for _, opt := range opts {
		opt(self)
	}

	bounds := ebimath.Rectangle{
		Min: ebimath.Vector{X: 0, Y: 0},
		Max: ebimath.Vector{X: float64(worldWidth), Y: float64(worldHeight)},
	}
	self.Quadtree = NewQuadtree(world, bounds)
	// Generate a Perlin noise image to simulate complex wind patterns.
	self.NoiseImage = utils.GeneratePerlinNoiseImage(self.NoiseMapSize, self.NoiseMapSize, self.NoiseFrequency)
	self.initGrass(world)

	return self
}

// initGrass generates and places all grass blades within the specified areas.
func (self *GrassControllerComponent) initGrass(world *World) {
	areasToGenerate := self.GrassAreas
	// If no specific areas are defined, generate grass across the entire world.
	if len(areasToGenerate) == 0 {
		areasToGenerate = []Area{
			{X1: 0, Y1: 0, X2: self.WorldWidth / self.TileSize, Y2: self.WorldHeight / self.TileSize},
		}
	}

	for _, area := range areasToGenerate {
		startX := int(math.Max(0, float64(area.X1)))
		startY := int(math.Max(0, float64(area.Y1)))
		endX := int(math.Min(float64(self.WorldWidth/self.TileSize), float64(area.X2)))
		endY := int(math.Min(float64(self.WorldHeight/self.TileSize), float64(area.Y2)))

		for yTile := startY; yTile < endY; yTile++ {
			for xTile := startX; xTile < endX; xTile++ {
				// Generate grass blades for each tile based on the density setting.
				for i := 0; i < self.GrassDensity; i++ {
					// Calculate a random position within the current tile.
					posX := float64(xTile*self.TileSize) + rand.Float64()*float64(self.TileSize)
					posY := float64(yTile*self.TileSize) + rand.Float64()*float64(self.TileSize)

					grassComp := &GrassComponent{
						// Set a random seed for unique, natural wind sway.
						SwaySeed:        rand.Float64() * 2 * math.Pi,
						InteractionSway: 0,
						SwayVelocity:    0,
						CurrentSway:     0,
					}
					entity := world.CreateEntity()
					transform := NewTransformComponent()
					transform.SetPosition(ebimath.V(posX, posY))
					transform.Z = self.Z

					// If orderable, create and add an orderable component for rendering.
					if self.Orderable {
						orderable := NewOrderableComponent(nil)
						orderable.SetIndex(transform.Position().Y)
						world.AddComponent(entity, orderable)
					}

					textureID := self.TextureID
					// Use a random texture from the area's list if specified.
					if len(area.TexturesIDs) > 0 {
						textureID = area.TexturesIDs[rand.Intn(len(area.TexturesIDs))]
					}

					img := self.Tm.Get(textureID)
					sprite := NewSpriteComponent(textureID, img.Bounds())

					// Manually adjust the vertices to set the anchor to the bottom-center.
					// This ensures the physics position (at the base) aligns with the visual representation.
					sprite.GenerateMesh()
					offsetX := float32(img.Bounds().Dx()) / 2
					offsetY := float32(img.Bounds().Dy())
					for i := range sprite.Vertices {
						sprite.Vertices[i].DstX -= offsetX
						sprite.Vertices[i].DstY -= offsetY
					}

					// Add all necessary components to the new grass entity.
					world.AddComponent(entity, grassComp)
					world.AddComponent(entity, transform)
					world.AddComponent(entity, sprite)

					// Insert the new grass entity into the quadtree for efficient spatial queries.
					self.Quadtree.Insert(entity)
				}
			}
		}
	}
}

// getWindForceAt samples the wind force at a given world position from the noise map.
// The noise map is treated as a seamless, repeating texture.
func (self *GrassControllerComponent) getWindForceAt(x, y float64) float64 {
	sampleX := int(math.Mod(x+self.WindScroll.X, float64(self.NoiseMapSize)))
	sampleY := int(math.Mod(y+self.WindScroll.Y, float64(self.NoiseMapSize)))
	if sampleX < 0 {
		sampleX += self.NoiseMapSize
	}
	if sampleY < 0 {
		sampleY += self.NoiseMapSize
	}
	noiseColor := self.NoiseImage.At(sampleX, sampleY)
	// The color value (red channel) represents the wind force.
	r, _, _, _ := noiseColor.RGBA()
	// Normalize the value to a float between 0.0 and 1.0.
	return float64(r) / 65535.0
}

// SetForcePositions updates the list of external force sources affecting the grass.
func (self *GrassControllerComponent) SetForcePositions(sources ...ForceSource) {
	self.ExternalForceSources = sources
}

// AddStrongWindGust adds a new strong wind gust to the simulation.
func (self *GrassControllerComponent) AddStrongWindGust(gust StrongWindGust) {
	// Normalize the direction vector and set the gust as active.
	gust.Direction = gust.EndPos.Sub(gust.StartPos).Normalize()
	gust.Active = true
	self.StrongWindGusts = append(self.StrongWindGusts, &gust)
}
