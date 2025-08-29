package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// Constants and types
const (
	defaultMiterLimit = 10.0
	roundJoinSegments = 10
)

type (
	LineJoinType         int
	LineTextureMode      int
	LineTextureDirection int
)

const (
	LineJoinMiter LineJoinType = iota
	LineJoinBevel
	LineJoinRound
)

const (
	LineTextureNone LineTextureMode = iota
	LineTextureStretch
	LineTextureTile
)

const (
	LineTextureHorizontal LineTextureDirection = iota
	LineTextureVertical
)

// LinePoint represents a single point on the line
type LinePoint struct {
	position ebimath.Vector
	color    color.RGBA
	width    float64
}

type LinePoints []*LinePoint

// Line represents a drawable line with various styling options
type Line struct {
	// Mesh data
	vertices []ebiten.Vertex
	indices  []uint16
	isDirty  bool

	// Line properties
	points     LinePoints
	pointLimit int
	width      float64
	color      color.RGBA
	opacity    float64
	lineJoin   LineJoinType
	IsClosed   bool

	// Width interpolation
	interpolateWidth     bool
	startWidth, endWidth float64

	// Color interpolation
	interpolateColor bool
	colors           []color.RGBA

	// Texture properties
	texture          *ebiten.Image
	whiteDot         *ebiten.Image
	textureMode      LineTextureMode
	textureSize      ebimath.Vector
	textureDirection LineTextureDirection
	textureScale     float64
	tileScale        float64
}

// NewLine creates a new Line instance
func NewLine(col color.RGBA, width float64) *Line {
	line := &Line{
		color:        col,
		opacity:      1.0,
		width:        width,
		points:       make(LinePoints, 0),
		tileScale:    1.0,
		textureMode:  LineTextureNone,
		textureScale: 1.0,
		vertices:     make([]ebiten.Vertex, 0),
		indices:      make([]uint16, 0),
		lineJoin:     LineJoinMiter,
		isDirty:      true,
		IsClosed:     false,
	}
	line.whiteDot = ebiten.NewImage(1, 1)
	line.whiteDot.Fill(color.White)
	return line
}

// Point management methods
func createLinePoint(position ebimath.Vector, col color.RGBA, width float64) *LinePoint {
	return &LinePoint{
		position: position,
		color:    col,
		width:    width,
	}
}

func (l *Line) AddPoint(pos ebimath.Vector) {
	if len(l.points) > 0 && l.points[len(l.points)-1].position.Equals(pos) {
		return
	}
	l.isDirty = true
	l.points = append(l.points, createLinePoint(pos, l.color, l.width))

	if l.pointLimit > 0 && len(l.points) > l.pointLimit {
		l.points = l.points[1:]
	}
}

func (l *Line) SetPosition(index int, position ebimath.Vector) {
	if !l.isValidIndex(index) {
		return
	}
	l.isDirty = true
	l.points[index].position = position
}

// Style setting methods
func (l *Line) SetLineJoin(join LineJoinType) {
	l.isDirty = true
	l.lineJoin = join
}

func (l *Line) SetIsClosed(isClosed bool) {
	l.isDirty = true
	l.IsClosed = isClosed
}

func (l *Line) SetTexture(img *ebiten.Image) {
	if img == nil {
		return
	}
	l.isDirty = true
	l.texture = img
	bounds := img.Bounds()
	l.textureSize = ebimath.V(float64(bounds.Dx()), float64(bounds.Dy()))
}

func (l *Line) SetTextureProperties(mode LineTextureMode, direction LineTextureDirection, scale float64) {
	l.isDirty = true
	l.textureMode = mode
	l.textureDirection = direction
	l.textureScale = scale
}

func (l *Line) SetWidth(index int, width float64) {
	if !l.isValidIndex(index) {
		return
	}
	l.isDirty = true
	l.points[index].width = width
}

// Color management methods
func (l *Line) ApplyAllColor(color color.RGBA) {
	l.isDirty = true
	l.interpolateColor = false
	for i := range l.points {
		l.points[i].color = color
	}
}

func (l *Line) InterpolateColors(colors ...color.RGBA) {
	l.isDirty = true
	l.interpolateColor = true
	l.colors = colors
}

// Width management methods
func (l *Line) ApplyAllWidth(width float64) {
	l.isDirty = true
	l.interpolateWidth = false
	l.width = width
	for _, p := range l.points {
		p.width = width
	}
}

func (l *Line) InterpolateWidth(start, end float64) {
	l.isDirty = true
	l.interpolateWidth = true
	l.width = start
	l.startWidth = start
	l.endWidth = end
}

// Mesh building methods
func (l *Line) BuildMesh() {
	if !l.isDirty || len(l.points) < 2 {
		return
	}

	l.ResetMesh()
	l.updateInterpolatedWidths()

	joiner := getJoiner(l.lineJoin)
	l.vertices, l.indices = joiner.BuildMesh(l)

	l.isDirty = false
}

// Helper methods
func (l *Line) isValidIndex(index int) bool {
	return index >= 0 && index < len(l.points)
}

func (l *Line) updateInterpolatedWidths() {
	if !l.interpolateWidth {
		return
	}
	for i, p := range l.points {
		t := float64(i) / float64(len(l.points)-1)
		p.width = l.startWidth + t*(l.endWidth-l.startWidth)
	}
}

func (l *Line) lerpColor(t float64) color.RGBA {
	if !l.interpolateColor || len(l.colors) == 0 {
		return l.color
	}
	if len(l.colors) == 1 {
		return l.colors[0]
	}

	colors := l.colors
	if l.IsClosed {
		colors = append(colors, l.colors[0])
	}

	pos := t * float64(len(colors)-1)
	segment := int(pos)
	tInSegment := pos - float64(segment)

	if segment >= len(colors)-1 {
		segment = len(colors) - 2
		tInSegment = 1.0
	}
	if segment < 0 {
		segment = 0
		tInSegment = 0.0
	}

	return utils.LerpPremultipliedRGBA(colors[segment], colors[segment+1], tInSegment)
}

// Drawing methods
func (l *Line) Draw(screen *ebiten.Image) {
	l.BuildMesh()
	if len(l.vertices) == 0 {
		return
	}

	img := l.whiteDot
	if l.textureMode != LineTextureNone && l.texture != nil {
		img = l.texture
	}

	op := &ebiten.DrawTrianglesOptions{}
	screen.DrawTriangles(l.vertices, l.indices, img, op)
}

func (l *Line) GetBounds() image.Rectangle {
	if len(l.points) == 0 {
		return image.Rectangle{}
	}

	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := -math.MaxInt, -math.MaxInt

	for _, p := range l.points {
		pos := p.position
		x := int(math.Round(pos.X))
		y := int(math.Round(pos.Y))
		minX = min(minX, x)
		minY = min(minY, y)
		maxX = max(maxX, x)
		maxY = max(maxY, y)
	}

	return image.Rect(minX, minY, maxX, maxY)
}

func (l *Line) ResetMesh() {
	l.vertices = l.vertices[:0]
	l.indices = l.indices[:0]
}
