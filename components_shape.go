package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type RectangleComponent struct {
	Width, Height     float32
	FillColors        [4]color.RGBA // 0:TL, 1:TR, 2:BR, 3:BL
	TopLeftRadius     float32
	TopRightRadius    float32
	BottomLeftRadius  float32
	BottomRightRadius float32
	StrokeWidth       float32
	StrokeColors      [4]color.RGBA // 0:TL, 1:TR, 2:BR, 3:BL
	Vertices          []ebiten.Vertex
	Indices           []uint16
	dirty             bool
}

func NewRectangleComponent(width, height float32, col color.RGBA) *RectangleComponent {
	return &RectangleComponent{
		Width:        width,
		Height:       height,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		dirty:        true,
	}
}

func (self *RectangleComponent) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.FillColors[0] = topLeft
	self.FillColors[1] = topRight
	self.FillColors[2] = bottomRight
	self.FillColors[3] = bottomLeft
	self.dirty = true
}

func (self *RectangleComponent) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.StrokeColors[0] = topLeft
	self.StrokeColors[1] = topRight
	self.StrokeColors[2] = bottomRight
	self.StrokeColors[3] = bottomLeft
	self.dirty = true
}

func (self *RectangleComponent) SetRadii(topLeft, topRight, bottomLeft, bottomRight float32) {
	self.TopLeftRadius = topLeft
	self.TopRightRadius = topRight
	self.BottomLeftRadius = bottomLeft
	self.BottomRightRadius = bottomRight
	self.dirty = true
}

func (self *RectangleComponent) SetStroke(width float32, col color.RGBA) {
	self.StrokeWidth = width
	self.StrokeColors = [4]color.RGBA{col, col, col, col}
	self.dirty = true
}

func (self *RectangleComponent) Rebuild() {
	if !self.dirty {
		return
	}
	self.dirty = false

	self.Vertices = nil
	self.Indices = nil

	self.generateFill()

	if self.StrokeWidth > 0 {
		self.generateStroke()
	}
}

type radii struct{ tl, tr, bl, br float32 }
type segments struct{ tl, tr, bl, br int }

func (self *RectangleComponent) generateFill() {
	innerRadii := radii{self.TopLeftRadius, self.TopRightRadius, self.BottomLeftRadius, self.BottomRightRadius}
	seg := segments{
		tl: self.calculateSegments(innerRadii.tl),
		tr: self.calculateSegments(innerRadii.tr),
		bl: self.calculateSegments(innerRadii.bl),
		br: self.calculateSegments(innerRadii.br),
	}
	path := self.generatePath(self.Width, self.Height, innerRadii, self.FillColors, seg)
	self.triangulateFill(path, self.FillColors)
}

func (self *RectangleComponent) generateStroke() {
	sw := self.StrokeWidth
	innerRadii := radii{self.TopLeftRadius, self.TopRightRadius, self.BottomLeftRadius, self.BottomRightRadius}
	outerRadii := radii{innerRadii.tl, innerRadii.tr, innerRadii.bl, innerRadii.br}
	if innerRadii.tl > 0 {
		outerRadii.tl += sw
	}
	if innerRadii.tr > 0 {
		outerRadii.tr += sw
	}
	if innerRadii.bl > 0 {
		outerRadii.bl += sw
	}
	if innerRadii.br > 0 {
		outerRadii.br += sw
	}

	seg := segments{
		tl: self.calculateSegments(outerRadii.tl),
		tr: self.calculateSegments(outerRadii.tr),
		bl: self.calculateSegments(outerRadii.bl),
		br: self.calculateSegments(outerRadii.br),
	}

	innerPath := self.generatePath(self.Width, self.Height, innerRadii, self.StrokeColors, seg)
	outerPath := self.generatePath(self.Width+sw*2, self.Height+sw*2, outerRadii, self.StrokeColors, seg)

	self.triangulateStroke(outerPath, innerPath)
}

func (self *RectangleComponent) calculateSegments(radius float32) int {
	if radius <= 0 {
		return 1
	}
	arcLength := float32(radius * math.Pi / 2)
	segments := int(arcLength / 1.5)
	if segments < 8 {
		segments = 8
	}
	if segments > 50 {
		segments = 50
	}
	return segments
}

func (self *RectangleComponent) generatePath(width, height float32, rd radii, colors [4]color.RGBA, seg segments) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0)
	path = append(path, self.generateCorner(width, height, rd.tl, rd.tl, rd.tl, 180, 270, colors, seg.tl)...)
	path = append(path, self.generateCorner(width, height, width-rd.tr, rd.tr, rd.tr, 270, 360, colors, seg.tr)...)
	path = append(path, self.generateCorner(width, height, width-rd.br, height-rd.br, rd.br, 0, 90, colors, seg.br)...)
	path = append(path, self.generateCorner(width, height, rd.bl, height-rd.bl, rd.bl, 90, 180, colors, seg.bl)...)
	return path
}

func (self *RectangleComponent) generateCorner(rectW, rectH, cx, cy, radius, startAngle, endAngle float32, colors [4]color.RGBA, segments int) []ebiten.Vertex {
	cornerVerts := make([]ebiten.Vertex, 0, segments)
	for i := 0; i < segments; i++ {
		angle := float64(startAngle)
		if segments > 1 {
			angle += float64(i) * (float64(endAngle - startAngle)) / float64(segments-1)
		}
		rad := angle * math.Pi / 180
		x := cx + radius*float32(math.Cos(rad))
		y := cy + radius*float32(math.Sin(rad))

		vColor := self.interpolateColor(x, y, rectW, rectH, colors)
		cr, cg, cb, ca := vColor.RGBA()

		cornerVerts = append(cornerVerts, ebiten.Vertex{
			DstX: x, DstY: y,
			ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
		})
	}
	return cornerVerts
}

func (self *RectangleComponent) triangulateStroke(outerPath, innerPath []ebiten.Vertex) {
	baseIndex := uint16(len(self.Vertices))

	if len(outerPath) != len(innerPath) {
		return
	}
	numVerts := len(outerPath)

	sw := self.StrokeWidth
	for _, v := range outerPath {
		v.DstX -= sw
		v.DstY -= sw
		self.Vertices = append(self.Vertices, v)
	}
	self.Vertices = append(self.Vertices, innerPath...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		self.Indices = append(self.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (self *RectangleComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(self.Vertices))

	avgColor := self.interpolateColor(self.Width/2, self.Height/2, self.Width, self.Height, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: self.Width / 2, DstY: self.Height / 2,
		ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
	}
	self.Vertices = append(self.Vertices, center)

	self.Vertices = append(self.Vertices, fillPath...)

	numPerimeterVerts := len(fillPath)
	for i := 0; i < numPerimeterVerts; i++ {
		p1 := baseIndex + 1 + uint16(i)
		p2 := baseIndex + 1 + uint16((i+1)%numPerimeterVerts)
		self.Indices = append(self.Indices, baseIndex, p1, p2)
	}
}

func (self *RectangleComponent) interpolateColor(x, y, width, height float32, colors [4]color.RGBA) color.RGBA {
	u := x / width
	v := y / height
	if u < 0 {
		u = 0
	}
	if u > 1 {
		u = 1
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}

	c00 := colors[0] // TL
	c10 := colors[1] // TR
	c01 := colors[3] // BL
	c11 := colors[2] // BR

	r00, g00, b00, a00 := c00.RGBA()
	r10, g10, b10, a10 := c10.RGBA()
	r01, g01, b01, a01 := c01.RGBA()
	r11, g11, b11, a11 := c11.RGBA()

	r_interp := float32(r00)*(1-u)*(1-v) + float32(r10)*u*(1-v) + float32(r01)*(1-u)*v + float32(r11)*u*v
	g_interp := float32(g00)*(1-u)*(1-v) + float32(g10)*u*(1-v) + float32(g01)*(1-u)*v + float32(g11)*u*v
	b_interp := float32(b00)*(1-u)*(1-v) + float32(b10)*u*(1-v) + float32(b01)*(1-u)*v + float32(b11)*u*v
	a_interp := float32(a00)*(1-u)*(1-v) + float32(a10)*u*(1-v) + float32(a01)*(1-u)*v + float32(a11)*u*v

	return color.RGBA{
		R: uint8(r_interp / 256),
		G: uint8(g_interp / 256),
		B: uint8(b_interp / 256),
		A: uint8(a_interp / 256),
	}
}
