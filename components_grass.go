package katsu2d

import (
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
)

// GrassComponent stores the unique simulation state for a single blade of grass.
// It is a pure value struct with no pointers so ECS archetype storage requires no GC scanning.
type GrassComponent struct {
	// SwaySeed is a random phase offset used to create natural variation per blade.
	SwaySeed float64
	// InteractionSway is the current sway angle caused by external forces.
	InteractionSway float64
	// SwayVelocity is the angular velocity of the interaction sway.
	SwayVelocity float64
	// AccumulatedForce is the total external force applied to this blade during the current frame.
	AccumulatedForce float64
	// CurrentSway is the final blended sway angle applied to the blade's rotation.
	CurrentSway float64
}

// ForceSource defines a radial external force (such as a character moving through grass).
type ForceSource struct {
	Position Vector
	Radius   float64
	Strength float64
}

// StrongWindGust defines a temporary, localized gust of wind that travels across the world.
type StrongWindGust struct {
	StartPos        Vector
	EndPos          Vector
	Direction       Vector
	Width           float64
	Length          float64
	Strength        float64
	Duration        float64
	FadeInDuration  float64
	FadeOutDuration float64
	ElapsedTime     float64
	Active          bool
}

// Area defines a rectangular tile region in the world where grass should be generated.
type Area struct {
	// TexturesIDs is a list of texture IDs randomly chosen from for grass blades in this area.
	TexturesIDs []int
	// X1, Y1, X2, Y2 define the inclusive-exclusive tile coordinates of the rectangular area.
	X1, Y1, X2, Y2 int
}

// GrassOption configures a GrassControllerComponent during construction.
type GrassOption func(*GrassControllerComponent)

// WithGrassTileSize sets the size of the grid cells (in pixels) for grass placement.
func WithGrassTileSize(size int) GrassOption {
	return func(c *GrassControllerComponent) {
		c.TileSize = size
	}
}

// WithGrassDensity sets the number of grass blades generated per tile.
func WithGrassDensity(density int) GrassOption {
	return func(c *GrassControllerComponent) {
		c.GrassDensity = density
	}
}

// WithGrassWindForce sets the maximum amplitude of wind-induced sway.
func WithGrassWindForce(force float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.WindForce = force
	}
}

// WithGrassWindSpeed sets how fast the grass oscillates due to ambient wind.
func WithGrassWindSpeed(speed float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.WindSpeed = speed
	}
}

// WithGrassWindDirection sets the dominant direction of ambient wind (normalized automatically).
func WithGrassWindDirection(x, y float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.WindDirection = Vector{X: x, Y: y}.Normalize()
	}
}

// WithGrassNoiseMapSize sets the dimension of the repeating Perlin noise map used for wind.
func WithGrassNoiseMapSize(size int) GrassOption {
	return func(c *GrassControllerComponent) {
		c.NoiseMapSize = size
	}
}

// WithGrassNoiseFrequency controls the spatial frequency of the Perlin wind noise map.
func WithGrassNoiseFrequency(freq float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.NoiseFrequency = freq
	}
}

// WithGrassForceBaseAcceleration sets the base acceleration multiplier for external forces.
func WithGrassForceBaseAcceleration(accel float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.ForceBaseAcceleration = accel
	}
}

// WithGrassSwaySpringStrength sets how strongly displaced grass pulls back toward upright.
func WithGrassSwaySpringStrength(strength float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.SwaySpringStrength = strength
	}
}

// WithGrassSwayDamping sets the per-frame velocity damping factor for recovery motion.
func WithGrassSwayDamping(damping float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.SwayDamping = damping
	}
}

// WithGrassAreas sets the specific tile areas where grass should be generated.
// If empty, grass is generated across the entire world.
func WithGrassAreas(areas []Area) GrassOption {
	return func(c *GrassControllerComponent) {
		c.GrassAreas = areas
	}
}

// WithGrassOrderable enables Y-based OrderableComponent generation for Z-sorted sprite rendering.
func WithGrassOrderable(orderable bool) GrassOption {
	return func(c *GrassControllerComponent) {
		c.Orderable = orderable
	}
}

// WithGrassWindShading sets the strength of wind-wave sunlit crest tinting on grass blades.
func WithGrassWindShading(strength float64) GrassOption {
	return func(c *GrassControllerComponent) {
		c.WindShadingStrength = strength
	}
}

// GrassControllerComponent holds the configuration and simulation state for a grass field.
type GrassControllerComponent struct {
	WorldWidth            int
	WorldHeight           int
	TileSize              int
	GrassDensity          int
	Quadtree              *Quadtree
	Tm                    *TextureManager
	ExternalForceSources  []ForceSource
	StrongWindGusts       []StrongWindGust
	WindScroll            Vector
	WindTime              float64
	noiseMap              []float64
	NoiseMapSize          int
	NoiseFrequency        float64
	SwaySpringStrength    float64
	SwayDamping           float64
	ForceBaseAcceleration float64
	GrassAreas            []Area
	RenderArea            Rectangle
	WindDirection         Vector
	WindForce             float64
	WindSpeed             float64
	WindShadingStrength   float64
	TextureID             int
	Orderable             bool
	Z                     float64
}

// NewGrassControllerComponent creates a GrassControllerComponent, builds its spatial quadtree
// and procedural wind noise map, and spawns all grass blade entities into world.
func NewGrassControllerComponent(
	world *teishoku.World,
	tm *TextureManager,
	worldWidth, worldHeight int,
	textureID int,
	z float64,
	opts ...GrassOption,
) *GrassControllerComponent {
	res := &GrassControllerComponent{
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
		WindDirection:         Vector{X: 1.0, Y: 0.0},
		WindForce:             0.3,
		WindSpeed:             0.5,
		WindShadingStrength:   0.14,
		TextureID:             textureID,
		Z:                     z,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(res)
		}
	}

	if res.TileSize <= 0 {
		res.TileSize = 32
	}
	if res.NoiseMapSize <= 0 {
		res.NoiseMapSize = 512
	}
	if res.GrassDensity < 0 {
		res.GrassDensity = 0
	}

	bounds := Rectangle{
		Min: Vector{X: 0, Y: 0},
		Max: Vector{X: float64(worldWidth), Y: float64(worldHeight)},
	}
	res.Quadtree = NewQuadtree(world, bounds)
	res.noiseMap = GeneratePerlinNoiseMap(res.NoiseMapSize, res.NoiseMapSize, res.NoiseFrequency)
	res.initGrass(world)

	return res
}

func (c *GrassControllerComponent) initGrass(world *teishoku.World) {
	if c.GrassDensity <= 0 || c.WorldWidth <= 0 || c.WorldHeight <= 0 {
		return
	}

	var defaultAreas [1]Area
	areasToGenerate := c.GrassAreas
	if len(areasToGenerate) == 0 {
		defaultAreas[0] = Area{
			X1: 0,
			Y1: 0,
			X2: c.WorldWidth / c.TileSize,
			Y2: c.WorldHeight / c.TileSize,
		}
		areasToGenerate = defaultAreas[:]
	}

	var builder3 *teishoku.Builder3[TransformComponent, SpriteComponent, GrassComponent]
	var builder4 *teishoku.Builder4[TransformComponent, SpriteComponent, GrassComponent, OrderableComponent]
	if c.Orderable {
		builder4 = teishoku.NewBuilder4[TransformComponent, SpriteComponent, GrassComponent, OrderableComponent](world)
	} else {
		builder3 = teishoku.NewBuilder3[TransformComponent, SpriteComponent, GrassComponent](world)
	}

	r := Random()
	maxTilesX := c.WorldWidth / c.TileSize
	maxTilesY := c.WorldHeight / c.TileSize
	tileSizeF := float64(c.TileSize)

	for _, area := range areasToGenerate {
		startX := max(0, area.X1)
		startY := max(0, area.Y1)
		endX := min(maxTilesX, area.X2)
		endY := min(maxTilesY, area.Y2)

		for yTile := startY; yTile < endY; yTile++ {
			baseY := float64(yTile) * tileSizeF
			for xTile := startX; xTile < endX; xTile++ {
				baseX := float64(xTile) * tileSizeF
				for range c.GrassDensity {
					posX := baseX + r.Float64()*tileSizeF
					posY := baseY + r.Float64()*tileSizeF

					texID := c.TextureID
					if len(area.TexturesIDs) > 0 {
						texID = area.TexturesIDs[r.IntRange(0, len(area.TexturesIDs)-1)]
					}

					var w, h int
					if c.Tm != nil {
						if img := c.Tm.Get(texID); img != nil {
							b := img.Bounds()
							w, h = b.Dx(), b.Dy()
						}
					}

					transform := TransformComponent{
						Position: Point{X: posX, Y: posY},
						Scale:    Point{X: 1, Y: 1},
						Offset:   Point{X: float64(w) / 2, Y: float64(h)},
						Z:        c.Z,
					}
					sprite := SpriteComponent{
						TextureID: texID,
						Width:     w,
						Height:    h,
						Bound:     Bound{Max: Point{X: float64(w), Y: float64(h)}},
						Color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
						Opacity:   1.0,
					}
					grass := GrassComponent{
						SwaySeed: r.Rad(),
					}

					var entity teishoku.Entity
					if c.Orderable {
						entity = builder4.NewEntityWith(transform, sprite, grass, OrderableComponent{Index: posY})
					} else {
						entity = builder3.NewEntityWith(transform, sprite, grass)
					}

					c.Quadtree.Insert(entity)
				}
			}
		}
	}
}

// getWindForceAt performs an O(1) tiled bilinear lookup into the precomputed Perlin noise map.
func (c *GrassControllerComponent) getWindForceAt(x, y float64) float64 {
	size := c.NoiseMapSize
	if size <= 0 || len(c.noiseMap) < size*size {
		return 0
	}

	sizeF := float64(size)
	invSize := 1.0 / sizeF

	sx := x + c.WindScroll.X
	sy := y + c.WindScroll.Y
	sx -= math.Floor(sx*invSize) * sizeF
	sy -= math.Floor(sy*invSize) * sizeF
	if sx >= sizeF || sx < 0 {
		sx = 0
	}
	if sy >= sizeF || sy < 0 {
		sy = 0
	}

	x0 := int(sx)
	y0 := int(sy)
	fx := sx - float64(x0)
	fy := sy - float64(y0)
	x1 := x0 + 1
	if x1 >= size {
		x1 = 0
	}
	y1 := y0 + 1
	if y1 >= size {
		y1 = 0
	}

	row0 := y0 * size
	row1 := y1 * size
	v00 := c.noiseMap[row0+x0]
	v10 := c.noiseMap[row0+x1]
	v01 := c.noiseMap[row1+x0]
	v11 := c.noiseMap[row1+x1]

	top := v00 + (v10-v00)*fx
	bot := v01 + (v11-v01)*fx
	return top + (bot-top)*fy
}

// SetForcePositions replaces the active external force sources while reusing slice capacity.
func (c *GrassControllerComponent) SetForcePositions(sources ...ForceSource) {
	c.ExternalForceSources = append(c.ExternalForceSources[:0], sources...)
}

// AddStrongWindGust normalizes the gust direction, activates it, and appends it to the simulation.
func (c *GrassControllerComponent) AddStrongWindGust(gust StrongWindGust) {
	gust.Direction = gust.EndPos.Sub(gust.StartPos).Normalize()
	gust.Active = true
	c.StrongWindGusts = append(c.StrongWindGusts, gust)
}

// Dispose clears and releases the spatial quadtree without allocating a fresh root node,
// and releases internal simulation slices.
func (c *GrassControllerComponent) Dispose() {
	if c.Quadtree != nil {
		if c.Quadtree.root != nil {
			c.Quadtree.root.release()
			c.Quadtree.root = nil
		}
		c.Quadtree = nil
	}
	c.noiseMap = nil
	c.ExternalForceSources = nil
	c.StrongWindGusts = nil
}

// DisposeEntities despawns all grass blade entities belonging to this controller
// from w and disposes internal quadtree and simulation resources.
func (c *GrassControllerComponent) DisposeEntities(w *teishoku.World) {
	if w != nil && c.Quadtree != nil && c.Quadtree.root != nil {
		entities := c.Quadtree.Query(c.Quadtree.Bounds())
		if len(entities) > 0 {
			w.RemoveEntities(entities)
		}
	}
	c.Dispose()
}
