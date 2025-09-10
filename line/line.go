package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// Line represents a drawable line with various styling options.
// It provides functionality for creating and rendering lines with different
// visual properties like width, color, texture, and joint/cap styles.
type Line struct {
	// Mesh data for rendering the line
	vertices []ebiten.Vertex // Vertex data for the triangle mesh
	indices  []uint16        // Triangle indices for rendering
	isDirty  bool            // Tracks if the mesh needs to be rebuilt

	// Core line properties
	points         []ebimath.Vector // Array of points defining the line path
	pointLimit     int              // Maximum number of points (0 = unlimited)
	width          float64          // Base width of the line
	defaultColor   color.RGBA       // Default color for the entire line
	isClosed       bool             // Whether the line forms a closed loop
	jointMode      LineJointMode    // How line segments are joined
	beginCapMode   LineCapMode      // Style of the start cap
	endCapMode     LineCapMode      // Style of the end cap
	sharpLimit     float64          // Angle threshold for sharp corners
	roundPrecision int              // Number of segments in rounded corners

	// Width interpolation properties
	widths []float64 // Per-point width values for width interpolation

	// Color interpolation properties
	colors []color.RGBA // Per-point colors for color interpolation

	// Texture properties
	texture     *ebiten.Image   // Optional texture for the line
	whiteDot    *ebiten.Image   // Fallback texture for untextured lines
	textureMode LineTextureMode // How textures are applied
	tileAspect  float64         // Aspect ratio for texture tiling

	// Advanced line generation properties
	stepDistance    float64          // Distance between points after resampling (0 = disabled)
	catmullRom      bool             // If true, the line will be smoothed using a Catmull-Rom spline
	splinePrecision float64          // Number of points to generate between each original point for the spline
	debugDraw       bool             // If true, draws debug points and lines to visualize the mesh
	debugPoints     []ebimath.Vector // The points used for building the mesh after processing
}

// NewLine creates a new Line instance with default settings.
// Returns a line with a white color, 10px width, sharp joints, and no caps.
func NewLine() *Line {
	line := &Line{
		defaultColor:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
		width:           10,
		points:          make([]ebimath.Vector, 0),
		vertices:        make([]ebiten.Vertex, 0),
		indices:         make([]uint16, 0),
		isDirty:         true,
		jointMode:       LineJointSharp,
		beginCapMode:    LineCapNone,
		endCapMode:      LineCapNone,
		sharpLimit:      2.0,
		roundPrecision:  8,
		tileAspect:      1.0,
		stepDistance:    0.0,
		catmullRom:      false,
		splinePrecision: 10,
		debugDraw:       false,
		debugPoints:     make([]ebimath.Vector, 0),
	}
	// A small white image used as a default texture for untextured lines.
	line.whiteDot = ebiten.NewImage(1, 1)
	line.whiteDot.Fill(color.White)

	return line
}

// GetMesh returns the current vertex and index data for the line's mesh.
func (self *Line) GetMesh() ([]ebiten.Vertex, []uint16) {
	return self.vertices, self.indices
}

// AddPoint adds a new point to the line if it is not a duplicate of the last point.
// If a point limit is set, the oldest points are removed to maintain the limit.
func (self *Line) AddPoint(pos ebimath.Vector) {
	if len(self.points) > 0 && self.points[len(self.points)-1].Equals(pos) {
		return
	}
	self.isDirty = true
	self.points = append(self.points, pos)

	if self.pointLimit > 0 && len(self.points) > self.pointLimit {
		self.points = self.points[1:]
	}
}

// SetPoint updates the position of an existing point at the given index.
// This function does nothing if the index is out of bounds.
func (self *Line) SetPoint(index int, position ebimath.Vector) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index] = position
}

// GetPoints returns the current array of line points.
func (self *Line) GetPoints() []ebimath.Vector {
	return self.points
}

// ClearPoints removes all points from the line.
func (self *Line) ClearPoints() {
	self.points = self.points[:0]
	self.isDirty = true
}

// SetJointMode defines how line segments are joined together.
func (self *Line) SetJointMode(mode LineJointMode) {
	self.isDirty = true
	self.jointMode = mode
}

// SetCapMode sets the style for both the start and end caps of the line.
func (self *Line) SetCapMode(begin, end LineCapMode) {
	self.isDirty = true
	self.beginCapMode = begin
	self.endCapMode = end
}

// SetIsClosed determines if the line should form a closed loop.
func (self *Line) SetIsClosed(isClosed bool) {
	self.isDirty = true
	self.isClosed = isClosed
}

// SetTexture assigns a texture to the line and updates the tile aspect ratio.
// Passing a nil image will disable texturing for the line.
func (self *Line) SetTexture(img *ebiten.Image) {
	self.isDirty = true
	self.texture = img
	if img != nil {
		bounds := img.Bounds()
		self.tileAspect = float64(bounds.Dx()) / float64(bounds.Dy())
	}
}

// SetWidth sets a uniform width for the entire line and clears any interpolated widths.
func (self *Line) SetWidth(width float64) {
	self.isDirty = true
	self.width = width
	self.widths = nil // Clear interpolated widths
}

// SetDefaultColor sets a uniform color for the entire line and clears any interpolated colors.
func (self *Line) SetDefaultColor(c color.RGBA) {
	self.isDirty = true
	self.defaultColor = c
	self.colors = nil // Clear interpolated colors
}

// SetInterpolatedColors sets per-point colors for creating a gradient effect.
func (self *Line) SetInterpolatedColors(colors ...color.RGBA) {
	self.isDirty = true
	self.colors = colors
}

// SetInterpolatedWidths sets per-point widths for creating a line with variable thickness.
func (self *Line) SetInterpolatedWidths(widths ...float64) {
	self.isDirty = true
	self.widths = widths
}

// SetStepDistance enables point resampling to ensure a minimum distance between vertices.
// A value of 0.0 disables this feature.
func (self *Line) SetStepDistance(distance float64) {
	self.isDirty = true
	self.stepDistance = distance
}

// EnableCatmullRom enables or disables Catmull-Rom spline smoothing for the line.
// The precision parameter controls the number of intermediate points generated for the curve.
func (self *Line) EnableCatmullRom(enable bool, precision float64) {
	self.isDirty = true
	self.catmullRom = enable
	self.splinePrecision = precision
}

// SetDebugDraw enables or disables the drawing of debug points and lines to visualize the mesh.
func (self *Line) SetDebugDraw(enable bool) {
	self.debugDraw = enable
}

// BuildMesh generates the triangle mesh for rendering the line.
// It performs spline smoothing and point resampling before building the final mesh.
// The mesh is only rebuilt if the line is marked as dirty and has at least 2 points.
func (self *Line) BuildMesh() {
	if !self.isDirty || len(self.points) < 2 {
		return
	}
	self.resetMesh()

	// Process points based on Catmull-Rom and resampling settings.
	pointsToProcess := self.points
	if self.catmullRom && len(self.points) >= 4 {
		pointsToProcess = self.generateSplinePoints(self.points, int(self.splinePrecision))
	}

	self.debugPoints = pointsToProcess
	if self.stepDistance > 0 && len(pointsToProcess) > 1 {
		self.debugPoints = self.resamplePoints(pointsToProcess, self.stepDistance)
	}

	// Use a LineBuilder to construct the mesh from the final set of points.
	builder := NewLineBuilder()
	builder.points = self.debugPoints
	builder.closed = self.isClosed
	builder.width = self.width
	builder.widths = self.widths
	builder.defaultColor = self.defaultColor
	builder.colors = self.colors
	builder.jointMode = self.jointMode
	builder.beginCapMode = self.beginCapMode
	builder.endCapMode = self.endCapMode
	builder.sharpLimit = self.sharpLimit
	builder.roundPrecision = self.roundPrecision
	builder.textureMode = self.textureMode
	builder.tileAspect = self.tileAspect

	builder.Build()

	self.vertices = builder.vertices
	self.indices = builder.indices

	self.isDirty = false
}

// Draw renders the line to the specified screen using the provided options.
// It uses a texture if one is set; otherwise, it falls back to a solid color.
// If debugDraw is enabled, it will also draw the processed points and segments.
func (self *Line) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	self.BuildMesh()
	if len(self.vertices) == 0 {
		return
	}

	img := self.whiteDot
	if self.textureMode != LineTextureNone && self.texture != nil {
		img = self.texture
	}

	screen.DrawTriangles(self.vertices, self.indices, img, op)

	// Draw debug points and lines if enabled.
	if self.debugDraw && len(self.debugPoints) > 0 {
		debugVerts := make([]ebiten.Vertex, 0)
		debugIndices := make([]uint16, 0)
		red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}

		// Draw points as small red squares.
		for _, p := range self.debugPoints {
			v1 := p.Add(ebimath.V(-2, -2))
			v2 := p.Add(ebimath.V(2, -2))
			v3 := p.Add(ebimath.V(-2, 2))
			v4 := p.Add(ebimath.V(2, 2))
			idx := uint16(len(debugVerts))
			debugVerts = append(debugVerts,
				ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
			)
			debugIndices = append(debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
		}

		// Draw lines as thin yellow segments.
		for i := 0; i < len(self.debugPoints)-1; i++ {
			p1 := self.debugPoints[i]
			p2 := self.debugPoints[i+1]
			dir := p2.Sub(p1).Normalize()
			normal := dir.Orthogonal().ScaleF(0.5) // 1px wide line

			v1 := p1.Add(normal)
			v2 := p1.Sub(normal)
			v3 := p2.Add(normal)
			v4 := p2.Sub(normal)
			idx := uint16(len(debugVerts))
			debugVerts = append(debugVerts,
				ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
			)
			debugIndices = append(debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
		}

		if len(debugVerts) > 0 {
			screen.DrawTriangles(debugVerts, debugIndices, self.whiteDot, op)
		}
	}
}

// GetBounds calculates the bounding rectangle of the line, taking its width into account.
func (self *Line) GetBounds() image.Rectangle {
	if len(self.points) == 0 {
		return image.Rectangle{}
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, p := range self.points {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	// Expand the bounds by half the line width to ensure the entire line is contained.
	h_width := self.width / 2
	return image.Rect(int(minX-h_width), int(minY-h_width), int(maxX+h_width), int(maxY+h_width))
}

// Reset clears the line's points and mesh data.
func (self *Line) Reset() {
	self.points = self.points[:0]
	self.resetMesh()
}

// resetMesh clears the current mesh data and marks the line as dirty.
func (self *Line) resetMesh() {
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.isDirty = true
}

// generateSplinePoints creates a new set of points from the original points
// by interpolating a smooth Catmull-Rom spline.
func (self *Line) generateSplinePoints(points []ebimath.Vector, precision int) []ebimath.Vector {
	if len(points) < 2 {
		return points
	}

	newPoints := make([]ebimath.Vector, 0)
	newPoints = append(newPoints, points[0])

	for i := 0; i < len(points)-1; i++ {
		p0 := points[i]
		if i > 0 {
			p0 = points[i-1]
		}
		p1 := points[i]
		p2 := points[i+1]
		p3 := points[i+1]
		if i < len(points)-2 {
			p3 = points[i+2]
		}

		for j := 1; j <= precision; j++ {
			t := float64(j) / float64(precision)
			newPos := catmullRomPoint(p0, p1, p2, p3, t)
			newPoints = append(newPoints, newPos)
		}
	}

	return newPoints
}

// resamplePoints creates a new slice of points with a uniform distance between them.
// This is useful for effects like dashed lines or uniform texture tiling.
func (self *Line) resamplePoints(points []ebimath.Vector, step float64) []ebimath.Vector {
	if len(points) < 2 || step <= 0 {
		return points
	}

	newPoints := make([]ebimath.Vector, 0, len(points))
	newPoints = append(newPoints, points[0])

	distanceNeeded := step
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]

		segmentVec := p2.Sub(p1)
		segmentLen := segmentVec.Length()

		if segmentLen < 1e-6 {
			continue
		}

		for segmentLen >= distanceNeeded {
			t := distanceNeeded / segmentLen
			newPos := p1.Lerp(p2, t)
			newPoints = append(newPoints, newPos)

			segmentLen -= distanceNeeded
			p1 = newPoints[len(newPoints)-1]
			distanceNeeded = step
		}
		distanceNeeded -= segmentLen
	}
	return newPoints
}
