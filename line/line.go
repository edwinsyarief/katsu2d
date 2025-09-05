package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// Line represents a drawable line with various styling options.
type Line struct {
	// Mesh data
	vertices []ebiten.Vertex
	indices  []uint16
	isDirty  bool

	// Line properties
	points         []ebimath.Vector
	pointLimit     int
	width          float64
	defaultColor   color.RGBA
	IsClosed       bool
	JointMode      LineJointMode
	BeginCapMode   LineCapMode
	EndCapMode     LineCapMode
	SharpLimit     float64
	RoundPrecision int

	// Width interpolation
	Widths []float64

	// Color interpolation
	Colors []color.RGBA

	// Texture properties
	texture     *ebiten.Image
	whiteDot    *ebiten.Image
	TextureMode LineTextureMode
	TileAspect  float64
}

// NewLine creates a new Line instance.
func NewLine() *Line {
	line := &Line{
		defaultColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		width:          10,
		points:         make([]ebimath.Vector, 0),
		vertices:       make([]ebiten.Vertex, 0),
		indices:        make([]uint16, 0),
		isDirty:        true,
		JointMode:      LineJointSharp,
		BeginCapMode:   LineCapNone,
		EndCapMode:     LineCapNone,
		SharpLimit:     2.0,
		RoundPrecision: 8,
		TileAspect:     1.0,
	}
	line.whiteDot = ebiten.NewImage(1, 1)
	line.whiteDot.Fill(color.White)
	return line
}

// Point management
func (l *Line) AddPoint(pos ebimath.Vector) {
	if len(l.points) > 0 && l.points[len(l.points)-1].Equals(pos) {
		return
	}
	l.isDirty = true
	l.points = append(l.points, pos)

	if l.pointLimit > 0 && len(l.points) > l.pointLimit {
		l.points = l.points[1:]
	}
}

func (l *Line) SetPoint(index int, position ebimath.Vector) {
	if index < 0 || index >= len(l.points) {
		return
	}
	l.isDirty = true
	l.points[index] = position
}

func (l *Line) GetPoints() []ebimath.Vector {
	return l.points
}

func (l *Line) ClearPoints() {
	l.points = l.points[:0]
	l.isDirty = true
}

// Style setting methods
func (l *Line) SetJointMode(mode LineJointMode) {
	l.isDirty = true
	l.JointMode = mode
}

func (l *Line) SetCapMode(begin, end LineCapMode) {
	l.isDirty = true
	l.BeginCapMode = begin
	l.EndCapMode = end
}

func (l *Line) SetIsClosed(isClosed bool) {
	l.isDirty = true
	l.IsClosed = isClosed
}

func (l *Line) SetTexture(img *ebiten.Image) {
	l.isDirty = true
	l.texture = img
	if img != nil {
		bounds := img.Bounds()
		l.TileAspect = float64(bounds.Dx()) / float64(bounds.Dy())
	}
}

func (l *Line) SetWidth(width float64) {
	l.isDirty = true
	l.width = width
	l.Widths = nil // Clear interpolated widths
}

func (l *Line) SetDefaultColor(c color.RGBA) {
	l.isDirty = true
	l.defaultColor = c
	l.Colors = nil // Clear interpolated colors
}

func (l *Line) SetInterpolatedColors(colors []color.RGBA) {
	l.isDirty = true
	l.Colors = colors
}

func (l *Line) SetInterpolatedWidths(widths []float64) {
	l.isDirty = true
	l.Widths = widths
}

// BuildMesh generates the vertices and indices for the line.
func (l *Line) BuildMesh() {
	if !l.isDirty || len(l.points) < 2 {
		return
	}
	l.ResetMesh()

	builder := NewLineBuilder()
	builder.Points = l.points
	builder.Closed = l.IsClosed
	builder.Width = l.width
	builder.Widths = l.Widths
	builder.DefaultColor = l.defaultColor
	builder.Colors = l.Colors
	builder.JointMode = l.JointMode
	builder.BeginCapMode = l.BeginCapMode
	builder.EndCapMode = l.EndCapMode
	builder.SharpLimit = l.SharpLimit
	builder.RoundPrecision = l.RoundPrecision
	builder.TextureMode = l.TextureMode
	builder.TileAspect = l.TileAspect

	builder.Build()

	l.vertices = builder.Vertices
	l.indices = builder.Indices

	l.isDirty = false
}

// Draw renders the line to the screen.
func (l *Line) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	l.BuildMesh()
	if len(l.vertices) == 0 {
		return
	}

	img := l.whiteDot
	if l.TextureMode != LineTextureNone && l.texture != nil {
		img = l.texture
	}

	screen.DrawTriangles(l.vertices, l.indices, img, op)
}

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
	// Add width to bounds
	h_width := l.width / 2
	return image.Rect(int(minX-h_width), int(minY-h_width), int(maxX+h_width), int(maxY+h_width))
}

func (l *Line) ResetMesh() {
	l.vertices = l.vertices[:0]
	l.indices = l.indices[:0]
	l.isDirty = true
}
