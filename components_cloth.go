package katsu2d

import "math"

// ClothPinMode defines which vertices of a cloth grid are pinned in place.
type ClothPinMode int

const (
	// ClothPinLeftEdge pins column 0 (horizontal flag waving from a left pole).
	ClothPinLeftEdge ClothPinMode = iota
	// ClothPinRightEdge pins the rightmost column (col == cols).
	ClothPinRightEdge
	// ClothPinTopEdge pins row 0 (hanging vertical banner or curtain).
	ClothPinTopEdge
	// ClothPinTopCorners pins only the top-left (0, 0) and top-right (cols, 0) corners
	// (draped tapestry or festival bunting).
	ClothPinTopCorners
	// ClothPinCustom delegates pin selection to ClothComponent.PinFunc.
	ClothPinCustom
)

const (
	defaultClothCols                 = 14
	defaultClothRows                 = 8
	defaultClothGravityX             = 0.0
	defaultClothGravityY             = 240.0
	defaultClothWindDirX             = 1.0
	defaultClothWindDirY             = 0.0
	defaultClothWindForce            = 340.0
	defaultClothWindSpeed            = 3.0
	defaultClothWindTurbulence       = 120.0
	defaultClothDamping              = 0.985
	defaultClothStiffness            = 0.92
	defaultClothConstraintIterations = 5
	defaultClothShadeStrength        = 0.30
)

type clothConstraint struct {
	a, b           int
	restLen        float64
	stiffnessScale float64
	minRatio       float64
	compressScale  float64
}

type clothQuadDepth struct {
	tl, tr, bl, br uint16
	depth          float32
}

// ClothComponent stores configuration and Verlet simulation buffers for a 2D cloth or flag mesh.
// Pair this component with TransformComponent, SpriteComponent, and MeshComponent on an entity.
type ClothComponent struct {
	// Cols and Rows specify the cloth grid subdivision counts (defaults to 14 x 8).
	Cols int
	Rows int

	// Gravity is the constant acceleration applied to non-pinned vertices in pixels/s^2.
	Gravity Vector
	// WindDirection is the normalized ambient wind direction vector.
	WindDirection Vector
	// WindForce is the ambient wind acceleration magnitude in pixels/s^2.
	WindForce float64
	// WindSpeed controls how fast traveling flutter waves propagate across the cloth.
	WindSpeed float64
	// WindTurbulence controls the amplitude of perpendicular flutter waves.
	WindTurbulence float64
	// Damping is the per-60Hz-frame velocity retention factor in [0..1], applied frame-rate independently.
	Damping float64
	// Stiffness controls constraint correction strength in [0.1..1.0].
	Stiffness float64
	// ConstraintIterations sets the number of relaxation passes per update.
	ConstraintIterations int
	// PinMode selects which grid vertices remain anchored at their rest positions.
	PinMode ClothPinMode
	// PinFunc determines pinned vertices when PinMode is ClothPinCustom.
	PinFunc func(col, row, cols, rows int) bool
	// ShadeStrength modulates MeshComponent.Vertices RGB brightness based on cloth folds (0 disables).
	ShadeStrength float64
	// WindPhase offsets the traveling wave phase so multiple cloths do not wave identically.
	WindPhase float64
	// ExternalForces holds optional per-cloth radial force sources in local sprite coordinates.
	ExternalForces []ForceSource

	// Internal zero-allocation solver buffers (initialized lazily on first update).
	posX          []float64
	posY          []float64
	prevX         []float64
	prevY         []float64
	restX         []float64
	restY         []float64
	pinned        []bool
	pinWeight     []float64
	constraints   []clothConstraint
	quadOrder     []clothQuadDepth
	backFaceAccum []float32
	simTime       float64
	initialized   bool
	cachedW       int
	cachedH       int
}

// ClothOption configures a ClothComponent during construction.
type ClothOption func(*ClothComponent)

// WithClothPinMode sets the pin mode for the cloth.
func WithClothPinMode(mode ClothPinMode) ClothOption {
	return func(c *ClothComponent) {
		c.PinMode = mode
	}
}

// WithClothPinFunc sets a custom pin predicate and switches PinMode to ClothPinCustom when non-nil.
func WithClothPinFunc(fn func(col, row, cols, rows int) bool) ClothOption {
	return func(c *ClothComponent) {
		c.PinFunc = fn
		if fn != nil {
			c.PinMode = ClothPinCustom
		}
	}
}

// WithClothGravity sets the gravity acceleration vector in pixels/s^2.
func WithClothGravity(gx, gy float64) ClothOption {
	return func(c *ClothComponent) {
		c.Gravity = V(gx, gy)
	}
}

// WithClothWind configures ambient wind direction (normalized), force, wave speed, and turbulence.
func WithClothWind(dir Vector, force, speed, turbulence float64) ClothOption {
	return func(c *ClothComponent) {
		if !dir.IsZero() {
			c.WindDirection = dir.Normalize()
		} else {
			c.WindDirection = Vector{}
		}
		c.WindForce = force
		c.WindSpeed = speed
		c.WindTurbulence = turbulence
	}
}

// WithClothDamping sets the frame-rate-independent velocity retention factor in [0..1].
func WithClothDamping(damping float64) ClothOption {
	return func(c *ClothComponent) {
		c.Damping = Clamp(damping, 0.0, 1.0)
	}
}

// WithClothStiffness sets constraint stiffness (clamped to [0.1..1.0]) and solver iteration count.
func WithClothStiffness(stiffness float64, iterations int) ClothOption {
	return func(c *ClothComponent) {
		c.Stiffness = Clamp(stiffness, 0.1, 1.0)
		if iterations > 0 {
			c.ConstraintIterations = iterations
		} else {
			c.ConstraintIterations = 1
		}
	}
}

// WithClothShading sets the fold lighting modulation strength (0 disables vertex color shading).
func WithClothShading(strength float64) ClothOption {
	return func(c *ClothComponent) {
		c.ShadeStrength = math.Max(0.0, strength)
	}
}

// WithClothPhase sets the traveling wind wave phase offset in radians.
func WithClothPhase(phase float64) ClothOption {
	return func(c *ClothComponent) {
		c.WindPhase = phase
	}
}

// NewClothComponent creates a ClothComponent with the given grid dimensions and default flag physics,
// applying any provided ClothOptions. Non-positive cols/rows default to 14 x 8.
func NewClothComponent(cols, rows int, opts ...ClothOption) ClothComponent {
	if cols <= 0 {
		cols = defaultClothCols
	}
	if rows <= 0 {
		rows = defaultClothRows
	}
	c := ClothComponent{
		Cols:                 cols,
		Rows:                 rows,
		Gravity:              V(defaultClothGravityX, defaultClothGravityY),
		WindDirection:        V(defaultClothWindDirX, defaultClothWindDirY),
		WindForce:            defaultClothWindForce,
		WindSpeed:            defaultClothWindSpeed,
		WindTurbulence:       defaultClothWindTurbulence,
		Damping:              defaultClothDamping,
		Stiffness:            defaultClothStiffness,
		ConstraintIterations: defaultClothConstraintIterations,
		PinMode:              ClothPinLeftEdge,
		ShadeStrength:        defaultClothShadeStrength,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&c)
		}
	}
	return c
}

// SetWind updates the ambient wind direction (normalized if non-zero) and force magnitude.
func (c *ClothComponent) SetWind(dir Vector, force float64) {
	if !dir.IsZero() {
		c.WindDirection = dir.Normalize()
	} else {
		c.WindDirection = Vector{}
	}
	c.WindForce = force
}

// Reset restores all cloth vertices to their rest pose with zero velocity and resets simulation time.
func (c *ClothComponent) Reset() {
	c.simTime = 0
	if !c.initialized {
		return
	}
	copy(c.posX, c.restX)
	copy(c.posY, c.restY)
	copy(c.prevX, c.restX)
	copy(c.prevY, c.restY)
	clear(c.backFaceAccum)
}
