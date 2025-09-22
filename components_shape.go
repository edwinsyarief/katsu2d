package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Shape interface {
	Rebuild()
	GetVertices() []ebiten.Vertex
	GetIndices() []uint16
}

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

		vColor := interpolateColor(x, y, rectW, rectH, colors)
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

	avgColor := interpolateColor(self.Width/2, self.Height/2, self.Width, self.Height, colors)
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

type CircleComponent struct {
	Radius       float32
	FillColors   [4]color.RGBA
	StrokeWidth  float32
	StrokeColors [4]color.RGBA
	Vertices     []ebiten.Vertex
	Indices      []uint16
	dirty        bool
}

func NewCircleComponent(radius float32, col color.RGBA) *CircleComponent {
	return &CircleComponent{
		Radius:       radius,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		dirty:        true,
	}
}

func (c *CircleComponent) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	c.FillColors[0] = topLeft
	c.FillColors[1] = topRight
	c.FillColors[2] = bottomRight
	c.FillColors[3] = bottomLeft
	c.dirty = true
}

func (c *CircleComponent) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	c.StrokeColors[0] = topLeft
	c.StrokeColors[1] = topRight
	c.StrokeColors[2] = bottomRight
	c.StrokeColors[3] = bottomLeft
	c.dirty = true
}

func (c *CircleComponent) SetStroke(width float32, col color.RGBA) {
	c.StrokeWidth = width
	c.StrokeColors = [4]color.RGBA{col, col, col, col}
	c.dirty = true
}

func (c *CircleComponent) GetVertices() []ebiten.Vertex {
	return c.Vertices
}

func (c *CircleComponent) GetIndices() []uint16 {
	return c.Indices
}

type TriangleComponent struct {
	Width, Height float32       // The dimensions of the bounding box
	FillColors    [4]color.RGBA // 0: Top, 1: Right, 2: Left
	StrokeWidth   float32
	StrokeColors  [4]color.RGBA
	CornerRadius  float32
	Vertices      []ebiten.Vertex
	Indices       []uint16
	dirty         bool
}

func NewTriangleComponent(width, height float32, col color.RGBA) *TriangleComponent {
	return &TriangleComponent{
		Width:        width,
		Height:       height,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		dirty:        true,
	}
}

func (t *TriangleComponent) SetColor(top, right, left color.RGBA) {
	t.FillColors[0] = top
	t.FillColors[1] = right
	t.FillColors[2] = left
	t.dirty = true
}

func (t *TriangleComponent) SetStrokeColor(top, right, left color.RGBA) {
	t.StrokeColors[0] = top
	t.StrokeColors[1] = right
	t.StrokeColors[2] = left
	t.dirty = true
}

func (t *TriangleComponent) SetStroke(width float32, col color.RGBA) {
	t.StrokeWidth = width
	t.StrokeColors = [4]color.RGBA{col, col, col, col}
	t.dirty = true
}

func (t *TriangleComponent) SetCornerRadius(radius float32) {
	t.CornerRadius = radius
	t.dirty = true
}

func (t *TriangleComponent) GetVertices() []ebiten.Vertex {
	return t.Vertices
}

func (t *TriangleComponent) GetIndices() []uint16 {
	return t.Indices
}

type PolygonComponent struct {
	Sides        int
	Radius       float32
	FillColors   [4]color.RGBA
	StrokeWidth  float32
	StrokeColors [4]color.RGBA
	CornerRadius float32
	Vertices     []ebiten.Vertex
	Indices      []uint16
	dirty        bool
}

func NewPolygonComponent(sides int, radius float32, col color.RGBA) *PolygonComponent {
	return &PolygonComponent{
		Sides:        sides,
		Radius:       radius,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		dirty:        true,
	}
}

func (p *PolygonComponent) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	p.FillColors[0] = topLeft
	p.FillColors[1] = topRight
	p.FillColors[2] = bottomRight
	p.FillColors[3] = bottomLeft
	p.dirty = true
}

func (p *PolygonComponent) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	p.StrokeColors[0] = topLeft
	p.StrokeColors[1] = topRight
	p.StrokeColors[2] = bottomRight
	p.StrokeColors[3] = bottomLeft
	p.dirty = true
}

func (p *PolygonComponent) SetStroke(width float32, col color.RGBA) {
	p.StrokeWidth = width
	p.StrokeColors = [4]color.RGBA{col, col, col, col}
	p.dirty = true
}

func (p *PolygonComponent) SetCornerRadius(radius float32) {
	p.CornerRadius = radius
	p.dirty = true
}

func (p *PolygonComponent) GetVertices() []ebiten.Vertex {
	return p.Vertices
}

func (p *PolygonComponent) GetIndices() []uint16 {
	return p.Indices
}

type PentagonComponent struct {
	PolygonComponent
}

func NewPentagonComponent(radius float32, col color.RGBA) *PentagonComponent {
	p := NewPolygonComponent(5, radius, col)
	return &PentagonComponent{PolygonComponent: *p}
}

type HexagonComponent struct {
	PolygonComponent
}

func NewHexagonComponent(radius float32, col color.RGBA) *HexagonComponent {
	p := NewPolygonComponent(6, radius, col)
	return &HexagonComponent{PolygonComponent: *p}
}

func (c *CircleComponent) Rebuild() {
	if !c.dirty {
		return
	}
	c.dirty = false

	c.Vertices = nil
	c.Indices = nil

	c.generateFill()

	if c.StrokeWidth > 0 {
		c.generateStroke()
	}
}

func (c *CircleComponent) generateFill() {
	path := c.generatePath(c.Radius, c.FillColors)
	c.triangulateFill(path, c.FillColors)
}

func (c *CircleComponent) generateStroke() {
	sw := c.StrokeWidth
	innerRadius := c.Radius
	outerRadius := c.Radius + sw

	innerPath := c.generatePath(innerRadius, c.StrokeColors)
	outerPath := c.generatePath(outerRadius, c.StrokeColors)

	c.triangulateStroke(outerPath, innerPath)
}

func (c *CircleComponent) generatePath(radius float32, colors [4]color.RGBA) []ebiten.Vertex {
	segments := c.calculateSegments(radius)
	path := make([]ebiten.Vertex, 0, segments)
	for i := 0; i < segments; i++ {
		angle := 2 * math.Pi * float32(i) / float32(segments)
		x := radius + radius*float32(math.Cos(float64(angle)))
		y := radius + radius*float32(math.Sin(float64(angle)))

		vColor := interpolateColor(x, y, radius*2, radius*2, colors)
		cr, cg, cb, ca := vColor.RGBA()

		path = append(path, ebiten.Vertex{
			DstX: x, DstY: y,
			ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
		})
	}
	return path
}

func (c *CircleComponent) calculateSegments(radius float32) int {
	arcLength := 2 * math.Pi * radius
	segments := int(arcLength / 1.5)
	if segments < 12 {
		segments = 12
	}
	if segments > 200 {
		segments = 200
	}
	return segments
}

func (c *CircleComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(c.Vertices))

	avgColor := interpolateColor(c.Radius, c.Radius, c.Radius*2, c.Radius*2, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: c.Radius, DstY: c.Radius,
		ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
	}
	c.Vertices = append(c.Vertices, center)
	c.Vertices = append(c.Vertices, fillPath...)

	numPerimeterVerts := len(fillPath)
	for i := 0; i < numPerimeterVerts; i++ {
		p1 := baseIndex + 1 + uint16(i)
		p2 := baseIndex + 1 + uint16((i+1)%numPerimeterVerts)
		c.Indices = append(c.Indices, baseIndex, p1, p2)
	}
}

func (c *CircleComponent) triangulateStroke(outerPath, innerPath []ebiten.Vertex) {
	baseIndex := uint16(len(c.Vertices))
	numVerts := len(outerPath)

	sw := c.StrokeWidth
	newOuterPath := make([]ebiten.Vertex, len(outerPath))
	for i, v := range outerPath {
		v.DstX -= sw
		v.DstY -= sw
		newOuterPath[i] = v
	}

	c.Vertices = append(c.Vertices, newOuterPath...)
	c.Vertices = append(c.Vertices, innerPath...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		c.Indices = append(c.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (t *TriangleComponent) Rebuild() {
	if !t.dirty {
		return
	}
	t.dirty = false

	t.Vertices = nil
	t.Indices = nil

	t.generateFill()

	if t.StrokeWidth > 0 {
		t.generateStroke()
	}
}

func (t *TriangleComponent) generateFill() {
	path := t.generatePath(t.Width, t.Height, t.CornerRadius, t.FillColors)
	t.triangulateFill(path, t.FillColors)
}

func (t *TriangleComponent) generateStroke() {
	sw := t.StrokeWidth
	innerRadius := t.CornerRadius
	outerRadius := t.CornerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}

	segments := t.calculateSegments(outerRadius)
	inner_path := t.generatePath(t.Width, t.Height, innerRadius, t.StrokeColors, segments)
	outer_path := t.generatePath(t.Width+sw*2, t.Height+sw*2, outerRadius, t.StrokeColors, segments)

	if len(inner_path) != len(outer_path) {
		return
	}

	baseIndex := uint16(len(t.Vertices))
	numVerts := len(outer_path)

	newOuterPath := make([]ebiten.Vertex, len(outer_path))
	for i, v := range outer_path {
		v.DstX -= sw
		v.DstY -= sw
		newOuterPath[i] = v
	}

	t.Vertices = append(t.Vertices, newOuterPath...)
	t.Vertices = append(t.Vertices, inner_path...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		t.Indices = append(t.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (t *TriangleComponent) generatePath(width, height, radius float32, colors [4]color.RGBA, segments ...int) []ebiten.Vertex {
	points := []struct{ x, y float32 }{
		{width / 2, 0},
		{width, height},
		{0, height},
	}

	if radius > 0 {
		if len(segments) > 0 {
			return t.generateRoundedPath(points, radius, colors, segments[0])
		}
		return t.generateRoundedPath(points, radius, colors, 0)
	}
	return t.generateSharpPath(points, colors)
}

func (t *TriangleComponent) generateSharpPath(points []struct{ x, y float32 }, colors [4]color.RGBA) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	for i, p := range points {
		vColor := interpolateColor(p.x, p.y, t.Width, t.Height, colors)
		cr, cg, cb, ca := vColor.RGBA()
		path[i] = ebiten.Vertex{
			DstX: p.x, DstY: p.y,
			ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
		}
	}
	return path
}

func (t *TriangleComponent) generateRoundedPath(points []struct{ x, y float32 }, radius float32, colors [4]color.RGBA, segments int) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0)
	numPoints := len(points)

	if segments == 0 {
		segments = t.calculateSegments(radius)
	}

	for i := 0; i < numPoints; i++ {
		p1 := points[i]
		p2 := points[(i+1)%numPoints]
		p3 := points[(i+2)%numPoints]

		v1 := struct{ x, y float32 }{p1.x - p2.x, p1.y - p2.y}
		v2 := struct{ x, y float32 }{p3.x - p2.x, p3.y - p2.y}

		len1 := float32(math.Sqrt(float64(v1.x*v1.x + v1.y*v1.y)))
		v1.x /= len1
		v1.y /= len1

		len2 := float32(math.Sqrt(float64(v2.x*v2.x + v2.y*v2.y)))
		v2.x /= len2
		v2.y /= len2

		angle := float32(math.Acos(float64(v1.x*v2.x + v1.y*v2.y)))
		dist := radius / float32(math.Tan(float64(angle/2)))

		startPt := struct{ x, y float32 }{p2.x + dist*v1.x, p2.y + dist*v1.y}
		endPt := struct{ x, y float32 }{p2.x + dist*v2.x, p2.y + dist*v2.y}

		center := struct{ x, y float32 }{startPt.x + (v2.y-v1.y)*radius, startPt.y + (v1.x-v2.x)*radius}

		startAngle := float32(math.Atan2(float64(startPt.y-center.y), float64(startPt.x-center.x)))
		endAngle := float32(math.Atan2(float64(endPt.y-center.y), float64(endPt.x-center.x)))

		if v1.x*v2.y-v1.y*v2.x < 0 {
			startAngle, endAngle = endAngle, startAngle
		}

		for j := 0; j < segments; j++ {
			ratio := float32(j) / float32(segments-1)
			interpAngle := startAngle + ratio*(endAngle-startAngle)
			if endAngle < startAngle {
				interpAngle = startAngle - ratio*(startAngle-endAngle)
			}

			x := center.x + radius*float32(math.Cos(float64(interpAngle)))
			y := center.y + radius*float32(math.Sin(float64(interpAngle)))

			vColor := interpolateColor(x, y, t.Width, t.Height, colors)
			cr, cg, cb, ca := vColor.RGBA()
			path = append(path, ebiten.Vertex{
				DstX: x, DstY: y,
				ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
			})
		}
	}
	return path
}

func (t *TriangleComponent) calculateSegments(radius float32) int {
	if radius <= 0 {
		return 1
	}
	arcLength := float32(radius * math.Pi / 2)
	segments := int(arcLength / 1.5)
	if segments < 4 {
		segments = 4
	}
	if segments > 25 {
		segments = 25
	}
	return segments
}

func (t *TriangleComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(t.Vertices))

	avgColor := interpolateColor(t.Width/2, t.Height/2, t.Width, t.Height, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: t.Width / 2, DstY: t.Height / 2,
		ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
	}
	t.Vertices = append(t.Vertices, center)
	t.Vertices = append(t.Vertices, fillPath...)

	numPerimeterVerts := len(fillPath)
	for i := 0; i < numPerimeterVerts; i++ {
		p1 := baseIndex + 1 + uint16(i)
		p2 := baseIndex + 1 + uint16((i+1)%numPerimeterVerts)
		t.Indices = append(t.Indices, baseIndex, p1, p2)
	}
}

func (p *PolygonComponent) Rebuild() {
	if !p.dirty {
		return
	}
	p.dirty = false

	p.Vertices = nil
	p.Indices = nil

	p.generateFill()

	if p.StrokeWidth > 0 {
		p.generateStroke()
	}
}

func (p *PolygonComponent) generateFill() {
	path := p.generatePath(p.Radius, p.CornerRadius, p.FillColors)
	p.triangulateFill(path, p.FillColors)
}

func (p *PolygonComponent) generateStroke() {
	sw := p.StrokeWidth
	innerRadius := p.CornerRadius
	outerRadius := p.CornerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}

	segments := p.calculateSegments(outerRadius)
	inner_path := p.generatePath(p.Radius, innerRadius, p.StrokeColors, segments)
	outer_path := p.generatePath(p.Radius+sw, outerRadius, p.StrokeColors, segments)

	if len(inner_path) != len(outer_path) {
		// Fallback or error, for safety, though they should be the same now.
		return
	}

	baseIndex := uint16(len(p.Vertices))
	numVerts := len(outer_path)

	newOuterPath := make([]ebiten.Vertex, len(outer_path))
	for i, v := range outer_path {
		v.DstX -= sw
		v.DstY -= sw
		newOuterPath[i] = v
	}

	p.Vertices = append(p.Vertices, newOuterPath...)
	p.Vertices = append(p.Vertices, inner_path...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		p.Indices = append(p.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (p *PolygonComponent) generatePath(radius, cornerRadius float32, colors [4]color.RGBA, segments ...int) []ebiten.Vertex {
	points := make([]struct{ x, y float32 }, p.Sides)
	angleStep := 2 * math.Pi / float32(p.Sides)
	for i := 0; i < p.Sides; i++ {
		angle := float32(i)*angleStep - float32(math.Pi/2)
		points[i] = struct{ x, y float32 }{
			x: radius + radius*float32(math.Cos(float64(angle))),
			y: radius + radius*float32(math.Sin(float64(angle))),
		}
	}

	if cornerRadius > 0 {
		if len(segments) > 0 {
			return p.generateRoundedPath(points, cornerRadius, colors, radius, segments[0])
		}
		return p.generateRoundedPath(points, cornerRadius, colors, radius, 0)
	}
	return p.generateSharpPath(points, colors, radius)
}

func (p *PolygonComponent) generateSharpPath(points []struct{ x, y float32 }, colors [4]color.RGBA, radius float32) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	for i, pt := range points {
		vColor := interpolateColor(pt.x, pt.y, radius*2, radius*2, colors)
		cr, cg, cb, ca := vColor.RGBA()
		path[i] = ebiten.Vertex{
			DstX: pt.x, DstY: pt.y,
			ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
		}
	}
	return path
}

func (p *PolygonComponent) generateRoundedPath(points []struct{ x, y float32 }, radius float32, colors [4]color.RGBA, polyRadius float32, segments int) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0)
	numPoints := len(points)

	if segments == 0 {
		segments = p.calculateSegments(radius)
	}

	for i := 0; i < numPoints; i++ {
		p1 := points[i]
		p2 := points[(i+1)%numPoints]
		p3 := points[(i+2)%numPoints]

		v1 := struct{ x, y float32 }{p1.x - p2.x, p1.y - p2.y}
		v2 := struct{ x, y float32 }{p3.x - p2.x, p3.y - p2.y}

		len1 := float32(math.Sqrt(float64(v1.x*v1.x + v1.y*v1.y)))
		v1.x /= len1
		v1.y /= len1

		len2 := float32(math.Sqrt(float64(v2.x*v2.x + v2.y*v2.y)))
		v2.x /= len2
		v2.y /= len2

		angle := float32(math.Acos(float64(v1.x*v2.x + v1.y*v2.y)))
		dist := radius / float32(math.Tan(float64(angle/2)))

		startPt := struct{ x, y float32 }{p2.x + dist*v1.x, p2.y + dist*v1.y}
		endPt := struct{ x, y float32 }{p2.x + dist*v2.x, p2.y + dist*v2.y}

		center := struct{ x, y float32 }{startPt.x + (v2.y-v1.y)*radius, startPt.y + (v1.x-v2.x)*radius}

		startAngle := float32(math.Atan2(float64(startPt.y-center.y), float64(startPt.x-center.x)))
		endAngle := float32(math.Atan2(float64(endPt.y-center.y), float64(endPt.x-center.x)))

		if v1.x*v2.y-v1.y*v2.x < 0 {
			startAngle, endAngle = endAngle, startAngle
		}

		for j := 0; j < segments; j++ {
			ratio := float32(j) / float32(segments-1)
			interpAngle := startAngle + ratio*(endAngle-startAngle)
			if endAngle < startAngle {
				interpAngle = startAngle - ratio*(startAngle-endAngle)
			}

			x := center.x + radius*float32(math.Cos(float64(interpAngle)))
			y := center.y + radius*float32(math.Sin(float64(interpAngle)))

			vColor := interpolateColor(x, y, polyRadius*2, polyRadius*2, colors)
			cr, cg, cb, ca := vColor.RGBA()
			path = append(path, ebiten.Vertex{
				DstX: x, DstY: y,
				ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
			})
		}
	}
	return path
}

func (p *PolygonComponent) calculateSegments(radius float32) int {
	if radius <= 0 {
		return 1
	}
	arcLength := float32(radius * math.Pi / 2)
	segments := int(arcLength / 1.5)
	if segments < 4 {
		segments = 4
	}
	if segments > 25 {
		segments = 25
	}
	return segments
}

func (p *PolygonComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(p.Vertices))

	avgColor := interpolateColor(p.Radius, p.Radius, p.Radius*2, p.Radius*2, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: p.Radius, DstY: p.Radius,
		ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
	}
	p.Vertices = append(p.Vertices, center)
	p.Vertices = append(p.Vertices, fillPath...)

	numPerimeterVerts := len(fillPath)
	for i := 0; i < numPerimeterVerts; i++ {
		p1 := baseIndex + 1 + uint16(i)
		p2 := baseIndex + 1 + uint16((i+1)%numPerimeterVerts)
		p.Indices = append(p.Indices, baseIndex, p1, p2)
	}
}

func interpolateColor(x, y, width, height float32, colors [4]color.RGBA) color.RGBA {
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
