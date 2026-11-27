package katsu2d

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Line represents a drawable line with various styling options.
// It provides functionality for creating and rendering lines with different
// visual properties like width, color, texture, and joint/cap styles.
type Line struct {
	texture      *ebiten.Image
	whiteDot     *ebiten.Image
	builder      *LineBuilder
	vertices     []ebiten.Vertex
	indices      []uint16
	points       []Vector
	widths       []float64
	colors       []color.RGBA
	debugPoints  []Vector
	splineBuf    []Vector
	resampleBuf  []Vector
	debugVerts   []ebiten.Vertex
	debugIndices []uint16

	pointLimit      int
	width           float64
	jointMode       LineJointMode
	beginCapMode    LineCapMode
	endCapMode      LineCapMode
	sharpLimit      float64
	roundPrecision  int
	textureMode     LineTextureMode
	tileAspect      float64
	textureScale    float64
	textureOffset   float64
	stepDistance    float64
	splinePrecision float64
	defaultColor    color.RGBA
	isDirty         bool
	isClosed        bool
	catmullRom      bool
	debugDraw       bool
}

// NewLine creates a new Line instance with default settings.
// Returns a line with a white color, 10px width, sharp joints, and no caps.
func NewLine() *Line {
	return &Line{
		defaultColor:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
		width:           10,
		whiteDot:        defaultWhiteImage(),
		builder:         NewLineBuilder(),
		isDirty:         true,
		jointMode:       LineJointSharp,
		beginCapMode:    LineCapNone,
		endCapMode:      LineCapNone,
		sharpLimit:      2.0,
		roundPrecision:  8,
		tileAspect:      1.0,
		textureScale:    1.0,
		stepDistance:    0.0,
		catmullRom:      false,
		splinePrecision: 10,
		debugDraw:       false,
	}
}

// GetMesh returns the current vertex and index data for the line's mesh.
func (l *Line) GetMesh() ([]ebiten.Vertex, []uint16) {
	return l.vertices, l.indices
}

// AddPoint adds a new point to the line if it is not a duplicate of the last point.
// If a point limit is set, the oldest points are removed in-place to maintain the limit.
func (l *Line) AddPoint(pos Vector) {
	if len(l.points) > 0 && l.points[len(l.points)-1].Equals(pos) {
		return
	}
	l.isDirty = true
	l.points = append(l.points, pos)
	if l.pointLimit > 0 && len(l.points) > l.pointLimit {
		excess := len(l.points) - l.pointLimit
		copy(l.points, l.points[excess:])
		l.points = l.points[:l.pointLimit]
	}
}

// SetPoint updates the position of an existing point at the given index.
// This function does nothing if the index is out of bounds.
func (l *Line) SetPoint(index int, position Vector) {
	if index < 0 || index >= len(l.points) {
		return
	}
	l.isDirty = true
	l.points[index] = position
}

// GetPoints returns the current slice of line points.
func (l *Line) GetPoints() []Vector {
	return l.points
}

// ClearPoints removes all points from the line.
func (l *Line) ClearPoints() {
	l.points = l.points[:0]
	l.resetMesh()
}

// SetPointLimit sets the maximum number of points the line retains.
// A limit <= 0 disables the limit.
func (l *Line) SetPointLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	l.pointLimit = limit
	if l.pointLimit > 0 && len(l.points) > l.pointLimit {
		excess := len(l.points) - l.pointLimit
		copy(l.points, l.points[excess:])
		l.points = l.points[:l.pointLimit]
		l.isDirty = true
	}
}

// SetJointMode defines how line segments are joined together.
func (l *Line) SetJointMode(mode LineJointMode) {
	l.isDirty = true
	l.jointMode = mode
}

// SetCapMode sets the style for both the start and end caps of the line.
func (l *Line) SetCapMode(begin, end LineCapMode) {
	l.isDirty = true
	l.beginCapMode = begin
	l.endCapMode = end
}

// SetIsClosed determines if the line should form a closed loop.
func (l *Line) SetIsClosed(isClosed bool) {
	l.isDirty = true
	l.isClosed = isClosed
}

// SetTexture assigns a texture to the line and updates the tile aspect ratio.
// Passing a nil image disables texturing for the line.
func (l *Line) SetTexture(img *ebiten.Image) {
	l.isDirty = true
	l.texture = img
	if img != nil {
		bounds := img.Bounds()
		if dy := bounds.Dy(); dy > 0 {
			l.tileAspect = float64(bounds.Dx()) / float64(dy)
		}
	}
}

// SetTextureMode defines how textures are applied along the line.
func (l *Line) SetTextureMode(mode LineTextureMode) {
	l.isDirty = true
	l.textureMode = mode
}

// SetTextureScale sets the horizontal tiling scale multiplier for textured lines.
// Values <= 0 are clamped to 1e-4.
func (l *Line) SetTextureScale(scale float64) {
	if scale <= 0 {
		scale = 1e-4
	}
	l.isDirty = true
	l.textureScale = scale
}

// SetTextureOffset sets the horizontal UV offset (in normalized texture widths) along the line.
func (l *Line) SetTextureOffset(offset float64) {
	l.isDirty = true
	l.textureOffset = offset
}

// SetWidth sets a uniform width for the entire line and clears any interpolated widths.
func (l *Line) SetWidth(width float64) {
	l.isDirty = true
	l.width = width
	l.widths = nil
}

// SetDefaultColor sets a uniform color for the entire line and clears any interpolated colors.
func (l *Line) SetDefaultColor(c color.RGBA) {
	l.isDirty = true
	l.defaultColor = c
	l.colors = nil
}

// SetInterpolatedColors sets per-point colors for creating a gradient effect.
func (l *Line) SetInterpolatedColors(colors ...color.RGBA) {
	l.isDirty = true
	l.colors = colors
}

// SetInterpolatedWidths sets per-point widths for creating a line with variable thickness.
func (l *Line) SetInterpolatedWidths(widths ...float64) {
	l.isDirty = true
	l.widths = widths
}

// SetStepDistance enables point resampling to ensure a minimum distance between vertices.
// A value of 0.0 disables this feature.
func (l *Line) SetStepDistance(distance float64) {
	l.isDirty = true
	l.stepDistance = distance
}

// EnableCatmullRom enables or disables Catmull-Rom spline smoothing for the line.
// The precision parameter controls the number of intermediate points generated for the curve.
func (l *Line) EnableCatmullRom(enable bool, precision float64) {
	l.isDirty = true
	l.catmullRom = enable
	l.splinePrecision = precision
}

// SetDebugDraw enables or disables the drawing of debug points and lines to visualize the mesh.
func (l *Line) SetDebugDraw(enable bool) {
	l.debugDraw = enable
}

// BuildMesh generates the triangle mesh for rendering the line.
// It performs spline smoothing and point resampling before building the final mesh.
// The mesh is only rebuilt if the line is marked as dirty and has at least 2 points.
func (l *Line) BuildMesh() {
	if !l.isDirty {
		return
	}
	l.resetMesh()
	if len(l.points) < 2 {
		l.debugPoints = l.debugPoints[:0]
		l.isDirty = false
		return
	}

	pointsToProcess := l.points
	if l.catmullRom && len(l.points) >= 4 {
		pointsToProcess = l.generateSplinePoints(l.points, int(l.splinePrecision))
	}
	if l.stepDistance > 0 && len(pointsToProcess) > 1 {
		pointsToProcess = l.resamplePoints(pointsToProcess, l.stepDistance)
	}
	l.debugPoints = pointsToProcess

	if l.builder == nil {
		l.builder = NewLineBuilder()
	}
	l.builder.Reset()
	l.builder.points = l.debugPoints
	l.builder.closed = l.isClosed
	l.builder.width = l.width
	l.builder.widths = l.widths
	l.builder.defaultColor = l.defaultColor
	l.builder.colors = l.colors
	l.builder.jointMode = l.jointMode
	l.builder.beginCapMode = l.beginCapMode
	l.builder.endCapMode = l.endCapMode
	l.builder.sharpLimit = l.sharpLimit
	l.builder.roundPrecision = l.roundPrecision
	l.builder.textureMode = l.textureMode
	l.builder.tileAspect = l.tileAspect
	l.builder.textureScale = l.textureScale
	l.builder.textureOffset = l.textureOffset
	if l.texture != nil && l.textureMode != LineTextureNone {
		l.builder.texBounds = l.texture.Bounds()
	} else {
		l.builder.texBounds = image.Rectangle{}
	}
	l.builder.Build()

	l.vertices = l.builder.vertices
	l.indices = l.builder.indices
	l.isDirty = false
}

// Draw renders the line to the specified screen using the provided options.
// It uses a texture if one is set; otherwise, it falls back to a solid color.
// If debugDraw is enabled, it also draws the processed points and segments.
func (l *Line) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	l.BuildMesh()
	if len(l.vertices) == 0 {
		return
	}
	if l.whiteDot == nil {
		l.whiteDot = defaultWhiteImage()
	}
	var drawOp ebiten.DrawTrianglesOptions
	if op != nil {
		drawOp = *op
	}
	img := l.whiteDot
	if l.textureMode != LineTextureNone && l.texture != nil {
		img = l.texture
		if drawOp.Address == ebiten.AddressUnsafe && l.textureMode == LineTextureTile {
			drawOp.Address = ebiten.AddressRepeat
		}
	}
	screen.DrawTriangles(l.vertices, l.indices, img, &drawOp)

	if l.debugDraw && len(l.debugPoints) > 0 {
		l.debugVerts = l.debugVerts[:0]
		l.debugIndices = l.debugIndices[:0]
		red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}

		for _, p := range l.debugPoints {
			v1 := p.Add(V(-2, -2))
			v2 := p.Add(V(2, -2))
			v3 := p.Add(V(-2, 2))
			v4 := p.Add(V(2, 2))
			l.appendDebugQuad(v1, v2, v3, v4, red)
		}
		for i := 0; i < len(l.debugPoints)-1; i++ {
			p1 := l.debugPoints[i]
			p2 := l.debugPoints[i+1]
			dir := p2.Sub(p1).Normalize()
			normal := dir.Orthogonal().ScaleF(0.5)
			v1 := p1.Add(normal)
			v2 := p1.Sub(normal)
			v3 := p2.Add(normal)
			v4 := p2.Sub(normal)
			l.appendDebugQuad(v1, v2, v3, v4, yellow)
		}
		if len(l.debugVerts) > 0 {
			screen.DrawTriangles(l.debugVerts, l.debugIndices, l.whiteDot, op)
		}
	}
}

func (l *Line) appendDebugQuad(v1, v2, v3, v4 Vector, clr color.RGBA) {
	const inv255 = float32(1.0 / 255.0)
	cr := float32(clr.R) * inv255
	cg := float32(clr.G) * inv255
	cb := float32(clr.B) * inv255
	ca := float32(clr.A) * inv255
	idx := uint16(len(l.debugVerts))
	l.debugVerts = append(l.debugVerts,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	l.debugIndices = append(l.debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

// GetBounds calculates the bounding rectangle of the line, taking its width into account.
func (l *Line) GetBounds() image.Rectangle {
	if len(l.points) == 0 {
		return image.Rectangle{}
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, p := range l.points {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	halfWidth := l.width / 2
	return image.Rect(int(minX-halfWidth), int(minY-halfWidth), int(maxX+halfWidth), int(maxY+halfWidth))
}

// Reset clears the line's points and mesh data.
func (l *Line) Reset() {
	l.points = l.points[:0]
	l.debugPoints = l.debugPoints[:0]
	l.resetMesh()
}

func (l *Line) resetMesh() {
	l.vertices = l.vertices[:0]
	l.indices = l.indices[:0]
	l.isDirty = true
}

func (l *Line) generateSplinePoints(points []Vector, precision int) []Vector {
	if len(points) < 2 || precision <= 0 {
		return points
	}
	l.splineBuf = l.splineBuf[:0]
	l.splineBuf = append(l.splineBuf, points[0])
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
			l.splineBuf = append(l.splineBuf, catmullRomPoint(p0, p1, p2, p3, t))
		}
	}
	return l.splineBuf
}

func (l *Line) resamplePoints(points []Vector, step float64) []Vector {
	if len(points) < 2 || step <= 0 {
		return points
	}
	l.resampleBuf = l.resampleBuf[:0]
	l.resampleBuf = append(l.resampleBuf, points[0])
	distanceNeeded := step
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]
		segmentLen := p2.Sub(p1).Length()
		if segmentLen < 1e-6 {
			continue
		}
		for segmentLen >= distanceNeeded {
			t := distanceNeeded / segmentLen
			newPos := p1.Lerp(p2, t)
			l.resampleBuf = append(l.resampleBuf, newPos)
			segmentLen -= distanceNeeded
			p1 = newPos
			distanceNeeded = step
		}
		distanceNeeded -= segmentLen
	}
	return l.resampleBuf
}
