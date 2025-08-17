package katsu2d

import (
	"math"
	"math/rand"
	"time"

	"github.com/aquilax/go-perlin"
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/quadtree"
	"github.com/hajimehoshi/ebiten/v2"
)

// GrassComponent holds the physics-related data for a single blade of grass.
type GrassComponent struct {
	SwaySeed         float64
	InteractionSway  float64
	SwayVelocity     float64
	AccumulatedForce float64
}

type ForceSource struct {
	Position ebimath.Vector
	Radius   float64
	Strength float64
}

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

type Area struct {
	TexturesIDs    []int
	X1, Y1, X2, Y2 int
}

type activeGustFrameData struct {
	gust               *StrongWindGust
	pos                ebimath.Vector
	strengthMultiplier float64
}

// Option is a function type for configuring GrassControllerComponent.
type GrassOption func(*GrassControllerComponent)

// WithTileSize sets the size of the grid cells for grass placement.
func WithTileSize(size int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.tileSize = size
	}
}

// WithGrassDensity sets the number of grass blades per tile.
func WithGrassDensity(density int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.grassDensity = density
	}
}

// WithWindForce sets the maximum amplitude of wind-induced sway.
func WithWindForce(force float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.windForce = force
	}
}

// WithWindSpeed sets how fast the grass sways due to wind.
func WithWindSpeed(speed float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.windSpeed = speed
	}
}

// WithWindDirection sets the dominant direction of the wind. The vector will be normalized.
func WithWindDirection(x, y float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.windDirection = ebimath.Vector{X: x, Y: y}.Normalized()
	}
}

// WithNoiseMapSize sets the dimension of the generated wind noise map.
func WithNoiseMapSize(size int) GrassOption {
	return func(self *GrassControllerComponent) {
		self.noiseMapSize = size
	}
}

// WithNoiseFrequency controls the "zoom" of the Perlin noise for wind.
func WithNoiseFrequency(freq float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.noiseFrequency = freq
	}
}

// WithForceBaseAcceleration sets how quickly the grass reacts to external forces.
func WithForceBaseAcceleration(accel float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.forceBaseAcceleration = accel
	}
}

// WithSwaySpringStrength sets how strongly grass tries to return to 0 (like a spring constant).
func WithSwaySpringStrength(strength float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.swaySpringStrength = strength
	}
}

// WithSwayDamping sets the damping factor for grass recovery.
func WithSwayDamping(damping float64) GrassOption {
	return func(self *GrassControllerComponent) {
		self.swayDamping = damping
	}
}

// WithGrassAreas sets the specific tile areas where grass should be generated.
func WithGrassAreas(areas []Area) GrassOption {
	return func(self *GrassControllerComponent) {
		self.grassAreas = areas
	}
}

// GrassControllerComponent holds the configuration and state of the grass system.
type GrassControllerComponent struct {
	worldWidth            int
	worldHeight           int
	tileSize              int
	grassDensity          int
	quadtree              *quadtree.Quadtree
	tm                    *TextureManager
	externalForceSources  []ForceSource
	strongWindGusts       []*StrongWindGust
	windScroll            ebimath.Vector
	windTime              float64
	noiseImage            *ebiten.Image
	noiseMapSize          int
	noiseFrequency        float64
	swaySpringStrength    float64
	swayDamping           float64
	forceBaseAcceleration float64
	grassAreas            []Area
	renderArea            ebimath.Rectangle
	windDirection         ebimath.Vector
	windForce             float64
	windSpeed             float64
	TextureID             int
	Z                     float64
}

// NewGrassControllerComponent creates and initializes a new grass controller component.
func NewGrassControllerComponent(world *World, tm *TextureManager, worldWidth, worldHeight int, textureID int, z float64, opts ...GrassOption) *GrassControllerComponent {
	rand.Seed(time.Now().UnixNano())
	self := &GrassControllerComponent{
		worldWidth:            worldWidth,
		worldHeight:           worldHeight,
		tileSize:              32,
		grassDensity:          20,
		tm:                    tm,
		noiseMapSize:          128,
		noiseFrequency:        20.0,
		swaySpringStrength:    10.0,
		swayDamping:           0.88,
		forceBaseAcceleration: 1800.0,
		windDirection:         ebimath.Vector{X: 1.0, Y: 0.0},
		windForce:             0.8,
		windSpeed:             0.5,
		TextureID:             textureID,
		Z:                     z,
	}
	for _, opt := range opts {
		opt(self)
	}
	self.noiseImage = self.generatePerlinNoiseImage(self.noiseMapSize, self.noiseMapSize, self.noiseFrequency)
	self.initGrass(world)
	bounds := ebimath.Rectangle{
		Min: ebimath.Vector{X: 0, Y: 0},
		Max: ebimath.Vector{X: float64(worldWidth), Y: float64(worldHeight)},
	}
	self.quadtree = quadtree.New(bounds)
	return self
}

// initGrass generates grass blades across specified areas.
func (self *GrassControllerComponent) initGrass(world *World) {
	areasToGenerate := self.grassAreas
	if len(areasToGenerate) == 0 {
		areasToGenerate = []Area{
			{X1: 0, Y1: 0, X2: self.worldWidth / self.tileSize, Y2: self.worldHeight / self.tileSize},
		}
	}

	for _, area := range areasToGenerate {
		startX := int(math.Max(0, float64(area.X1)))
		startY := int(math.Max(0, float64(area.Y1)))
		endX := int(math.Min(float64(self.worldWidth/self.tileSize), float64(area.X2)))
		endY := int(math.Min(float64(self.worldHeight/self.tileSize), float64(area.Y2)))

		for yTile := startY; yTile < endY; yTile++ {
			for xTile := startX; xTile < endX; xTile++ {
				for i := 0; i < self.grassDensity; i++ {
					posX := float64(xTile*self.tileSize) + rand.Float64()*float64(self.tileSize)
					posY := float64(yTile*self.tileSize) + rand.Float64()*float64(self.tileSize)

					grassComp := &GrassComponent{
						SwaySeed:        rand.Float64() * 2 * math.Pi,
						InteractionSway: 0,
						SwayVelocity:    0,
					}
					entity := world.CreateEntity()
					transform := NewTransformComponent()
					transform.SetPosition(ebimath.V(posX, posY))
					transform.Z = self.Z

					textureID := self.TextureID
					if len(area.TexturesIDs) > 0 {
						textureID = area.TexturesIDs[rand.Intn(len(area.TexturesIDs))]
					}

					img := self.tm.Get(textureID)
					width, height := img.Bounds().Dx(), img.Bounds().Dy()
					sprite := NewSpriteComponent(textureID, width, height)

					world.AddComponent(entity, grassComp)
					world.AddComponent(entity, transform)
					world.AddComponent(entity, sprite)
				}
			}
		}
	}
}

// generatePerlinNoiseImage generates a Perlin noise image for wind simulation.
func (self *GrassControllerComponent) generatePerlinNoiseImage(width, height int, frequency float64) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	p := perlin.NewPerlin(2, 2, 3, rand.Int63())
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noiseVal := p.Noise2D(float64(x)/frequency, float64(y)/frequency)
			gray := byte((noiseVal + 1) * 127.5)
			idx := (y*width + x) * 4
			pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = gray, gray, gray, 255
		}
	}
	img.WritePixels(pixels)
	return img
}

// getWindForceAt samples the wind force at a given position from the noise map.
func (self *GrassControllerComponent) getWindForceAt(x, y float64) float64 {
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

// SetForcePositions updates the external force sources affecting the grass.
func (self *GrassControllerComponent) SetForcePositions(sources []ForceSource) {
	self.externalForceSources = sources
}

// AddStrongWindGust adds a new strong wind gust to the system.
func (self *GrassControllerComponent) AddStrongWindGust(gust StrongWindGust) {
	gust.Direction = gust.EndPos.Sub(gust.StartPos).Normalized()
	gust.Active = true
	self.strongWindGusts = append(self.strongWindGusts, &gust)
}
