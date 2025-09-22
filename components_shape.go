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

func (self *CircleComponent) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.FillColors[0] = topLeft
	self.FillColors[1] = topRight
	self.FillColors[2] = bottomRight
	self.FillColors[3] = bottomLeft
	self.dirty = true
}

func (self *CircleComponent) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.StrokeColors[0] = topLeft
	self.StrokeColors[1] = topRight
	self.StrokeColors[2] = bottomRight
	self.StrokeColors[3] = bottomLeft
	self.dirty = true
}

func (self *CircleComponent) SetStroke(width float32, col color.RGBA) {
	self.StrokeWidth = width
	self.StrokeColors = [4]color.RGBA{col, col, col, col}
	self.dirty = true
}

func (self *CircleComponent) GetVertices() []ebiten.Vertex {
	return self.Vertices
}

func (self *CircleComponent) GetIndices() []uint16 {
	return self.Indices
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

func (self *TriangleComponent) SetColor(top, right, left color.RGBA) {
	self.FillColors[0] = top
	self.FillColors[1] = right
	self.FillColors[2] = left
	self.dirty = true
}

func (self *TriangleComponent) SetStrokeColor(top, right, left color.RGBA) {
	self.StrokeColors[0] = top
	self.StrokeColors[1] = right
	self.StrokeColors[2] = left
	self.dirty = true
}

func (self *TriangleComponent) SetStroke(width float32, col color.RGBA) {
	self.StrokeWidth = width
	self.StrokeColors = [4]color.RGBA{col, col, col, col}
	self.dirty = true
}

func (self *TriangleComponent) SetCornerRadius(radius float32) {
	self.CornerRadius = radius
	self.dirty = true
}

func (self *TriangleComponent) GetVertices() []ebiten.Vertex {
	return self.Vertices
}

func (self *TriangleComponent) GetIndices() []uint16 {
	return self.Indices
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

func (self *PolygonComponent) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.FillColors[0] = topLeft
	self.FillColors[1] = topRight
	self.FillColors[2] = bottomRight
	self.FillColors[3] = bottomLeft
	self.dirty = true
}

func (self *PolygonComponent) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	self.StrokeColors[0] = topLeft
	self.StrokeColors[1] = topRight
	self.StrokeColors[2] = bottomRight
	self.StrokeColors[3] = bottomLeft
	self.dirty = true
}

func (self *PolygonComponent) SetStroke(width float32, col color.RGBA) {
	self.StrokeWidth = width
	self.StrokeColors = [4]color.RGBA{col, col, col, col}
	self.dirty = true
}

func (self *PolygonComponent) SetCornerRadius(radius float32) {
	self.CornerRadius = radius
	self.dirty = true
}

func (self *PolygonComponent) GetVertices() []ebiten.Vertex {
	return self.Vertices
}

func (self *PolygonComponent) GetIndices() []uint16 {
	return self.Indices
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

func (self *CircleComponent) Rebuild() {
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

func (self *CircleComponent) generateFill() {
	path := self.generatePath(self.Radius, self.FillColors)
	self.triangulateFill(path, self.FillColors)
}

func (self *CircleComponent) generateStroke() {
	sw := self.StrokeWidth
	innerRadius := self.Radius
	outerRadius := self.Radius + sw

	innerPath := self.generatePath(innerRadius, self.StrokeColors)
	outerPath := self.generatePath(outerRadius, self.StrokeColors)

	self.triangulateStroke(outerPath, innerPath)
}

func (self *CircleComponent) generatePath(radius float32, colors [4]color.RGBA) []ebiten.Vertex {
	segments := self.calculateSegments(radius)
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

func (self *CircleComponent) calculateSegments(radius float32) int {
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

func (self *CircleComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(self.Vertices))

	avgColor := interpolateColor(self.Radius, self.Radius, self.Radius*2, self.Radius*2, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: self.Radius, DstY: self.Radius,
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

func (self *CircleComponent) triangulateStroke(outerPath, innerPath []ebiten.Vertex) {
	baseIndex := uint16(len(self.Vertices))
	numVerts := len(outerPath)

	sw := self.StrokeWidth
	newOuterPath := make([]ebiten.Vertex, len(outerPath))
	for i, v := range outerPath {
		v.DstX -= sw
		v.DstY -= sw
		newOuterPath[i] = v
	}

	self.Vertices = append(self.Vertices, newOuterPath...)
	self.Vertices = append(self.Vertices, innerPath...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		self.Indices = append(self.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (self *TriangleComponent) Rebuild() {
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

func (self *TriangleComponent) generateFill() {
	path := self.generatePath(self.Width, self.Height, self.CornerRadius, self.FillColors)
	self.triangulateFill(path, self.FillColors)
}

func (self *TriangleComponent) generateStroke() {
	sw := self.StrokeWidth
	innerRadius := self.CornerRadius
	outerRadius := self.CornerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}

	segments := self.calculateSegments(outerRadius)
	var inner_path, outer_path []ebiten.Vertex
	if self.CornerRadius > 0 {
		inner_path = self.generatePath(self.Width, self.Height, innerRadius, self.StrokeColors, segments)
		outer_path = self.generatePath(self.Width, self.Height, outerRadius, self.StrokeColors, segments)
	} else {
		inner_path = self.generatePath(self.Width, self.Height, 0, self.StrokeColors)
		outer_path = self.generateMiterPath(inner_path, sw)
	}

	if len(inner_path) != len(outer_path) {
		return
	}

	baseIndex := uint16(len(self.Vertices))
	numVerts := len(outer_path)

	self.Vertices = append(self.Vertices, outer_path...)
	self.Vertices = append(self.Vertices, inner_path...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		self.Indices = append(self.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (self *TriangleComponent) generatePath(width, height, radius float32, colors [4]color.RGBA, segments ...int) []ebiten.Vertex {
	points := []struct{ x, y float32 }{
		{width / 2, 0},
		{0, height},
		{width, height},
	}

	if radius > 0 {
		if len(segments) > 0 {
			return self.generateRoundedPath(points, radius, colors, segments[0])
		}
		return self.generateRoundedPath(points, radius, colors, 0)
	}
	return self.generateSharpPath(points, colors)
}

func (self *TriangleComponent) generateSharpPath(points []struct{ x, y float32 }, colors [4]color.RGBA) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	for i, p := range points {
		vColor := interpolateColor(p.x, p.y, self.Width, self.Height, colors)
		cr, cg, cb, ca := vColor.RGBA()
		path[i] = ebiten.Vertex{
			DstX: p.x, DstY: p.y,
			ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
		}
	}
	return path
}

func (self *TriangleComponent) generateRoundedPath(points []struct{ x, y float32 }, radius float32, colors [4]color.RGBA, segments int) []ebiten.Vertex {
	return generateRoundedPathForPolygon(points, radius, colors, self.Width, self.Height, segments, self.calculateSegments)
}

func (self *TriangleComponent) generateMiterPath(points []ebiten.Vertex, strokeWidth float32) []ebiten.Vertex {
	return generateMiterPathForPolygon(points, strokeWidth)
}

func (self *TriangleComponent) calculateSegments(radius float32) int {
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

func (self *TriangleComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
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

func (self *PolygonComponent) Rebuild() {
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

func (self *PolygonComponent) generateFill() {
	path := self.generatePath(self.Radius, self.CornerRadius, self.FillColors)
	self.triangulateFill(path, self.FillColors)
}

func (self *PolygonComponent) generateStroke() {
	sw := self.StrokeWidth
	innerRadius := self.CornerRadius
	outerRadius := self.CornerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}

	segments := self.calculateSegments(outerRadius)
	var inner_path, outer_path []ebiten.Vertex
	if self.CornerRadius > 0 {
		inner_path = self.generatePath(self.Radius, innerRadius, self.StrokeColors, segments)
		outer_path = self.generatePath(self.Radius, outerRadius, self.StrokeColors, segments)
	} else {
		inner_path = self.generatePath(self.Radius, 0, self.StrokeColors)
		outer_path = self.generateMiterPath(inner_path, sw)
	}

	if len(inner_path) != len(outer_path) {
		// Fallback or error, for safety, though they should be the same now.
		return
	}

	baseIndex := uint16(len(self.Vertices))
	numVerts := len(outer_path)

	self.Vertices = append(self.Vertices, outer_path...)
	self.Vertices = append(self.Vertices, inner_path...)

	for i := 0; i < numVerts; i++ {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		self.Indices = append(self.Indices, p0, p2, p1, p1, p2, p3)
	}
}

func (self *PolygonComponent) generatePath(radius, cornerRadius float32, colors [4]color.RGBA, segments ...int) []ebiten.Vertex {
	points := make([]struct{ x, y float32 }, self.Sides)
	angleStep := 2 * math.Pi / float32(self.Sides)
	for i := 0; i < self.Sides; i++ {
		angle := float32(i)*angleStep - float32(math.Pi/2)
		points[i] = struct{ x, y float32 }{
			x: radius + radius*float32(math.Cos(float64(angle))),
			y: radius + radius*float32(math.Sin(float64(angle))),
		}
	}

	if cornerRadius > 0 {
		if len(segments) > 0 {
			return self.generateRoundedPath(points, cornerRadius, colors, radius, segments[0])
		}
		return self.generateRoundedPath(points, cornerRadius, colors, radius, 0)
	}
	return self.generateSharpPath(points, colors, radius)
}

func (self *PolygonComponent) generateSharpPath(points []struct{ x, y float32 }, colors [4]color.RGBA, radius float32) []ebiten.Vertex {
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

func (self *PolygonComponent) generateRoundedPath(points []struct{ x, y float32 }, radius float32, colors [4]color.RGBA, polyRadius float32, segments int) []ebiten.Vertex {
	return generateRoundedPathForPolygon(points, radius, colors, polyRadius*2, polyRadius*2, segments, self.calculateSegments)
}

func (self *PolygonComponent) generateMiterPath(points []ebiten.Vertex, strokeWidth float32) []ebiten.Vertex {
	return generateMiterPathForPolygon(points, strokeWidth)
}

func generateRoundedPathForPolygon(points []struct{ x, y float32 }, cornerRadius float32, colors [4]color.RGBA, width, height float32, segments int, calculateSegments func(radius float32) int) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0)
	numPoints := len(points)

	if segments == 0 {
		segments = calculateSegments(cornerRadius)
	}

	for i := 0; i < numPoints; i++ {
		p1 := points[(i+numPoints-1)%numPoints] // Previous
		p2 := points[i]                         // Current (the corner)
		p3 := points[(i+1)%numPoints]           // Next

		v1 := struct{ x, y float32 }{p1.x - p2.x, p1.y - p2.y}
		v2 := struct{ x, y float32 }{p3.x - p2.x, p3.y - p2.y}

		len1 := float32(math.Sqrt(float64(v1.x*v1.x + v1.y*v1.y)))
		if len1 > 0 {
			v1.x /= len1
			v1.y /= len1
		}

		len2 := float32(math.Sqrt(float64(v2.x*v2.x + v2.y*v2.y)))
		if len2 > 0 {
			v2.x /= len2
			v2.y /= len2
		}

		angle := float32(math.Acos(float64(v1.x*v2.x + v1.y*v2.y)))

		// When angle is close to 0 or Pi, it's a straight line or a 180-degree turn.
		// Rounding is not well-defined here and can cause huge floating point errors.
		// So we just draw a sharp corner by adding the vertex p2.
		// To maintain the same number of vertices for stroke calculations, we add it 'segments' times.
		if math.IsNaN(float64(angle)) || angle < 1e-4 || angle > math.Pi-1e-4 {
			vColor := interpolateColor(p2.x, p2.y, width, height, colors)
			cr, cg, cb, ca := vColor.RGBA()
			vert := ebiten.Vertex{
				DstX:   p2.x,
				DstY:   p2.y,
				ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
			}
			for j := 0; j < segments; j++ {
				path = append(path, vert)
			}
			continue
		}

		dist := cornerRadius / float32(math.Tan(float64(angle/2)))
		startPt := struct{ x, y float32 }{p2.x + dist*v1.x, p2.y + dist*v1.y}
		endPt := struct{ x, y float32 }{p2.x + dist*v2.x, p2.y + dist*v2.y}

		bisector := struct{ x, y float32 }{v1.x + v2.x, v1.y + v2.y}
		len_bisector := float32(math.Sqrt(float64(bisector.x*bisector.x + bisector.y*bisector.y)))
		if len_bisector > 0 {
			bisector.x /= len_bisector
			bisector.y /= len_bisector
		}

		center_dist := cornerRadius / float32(math.Sin(float64(angle/2)))
		arcCenter := struct{ x, y float32 }{p2.x + bisector.x*center_dist, p2.y + bisector.y*center_dist}

		startAngle := float32(math.Atan2(float64(startPt.y-arcCenter.y), float64(startPt.x-arcCenter.x)))
		endAngle := float32(math.Atan2(float64(endPt.y-arcCenter.y), float64(endPt.x-arcCenter.x)))

		// For a convex corner on a CCW polygon, the cross product of incoming vectors is negative.
		cross_product := v1.x*v2.y - v1.y*v2.x
		if cross_product < 0 {
			startAngle, endAngle = endAngle, startAngle
		}

		if endAngle < startAngle {
			endAngle += 2 * math.Pi
		}

		for j := 0; j < segments; j++ {
			var ratio float32
			if segments > 1 {
				ratio = float32(j) / float32(segments-1)
			}
			interpAngle := startAngle + ratio*(endAngle-startAngle)

			x := arcCenter.x + cornerRadius*float32(math.Cos(float64(interpAngle)))
			y := arcCenter.y + cornerRadius*float32(math.Sin(float64(interpAngle)))

			vColor := interpolateColor(x, y, width, height, colors)
			cr, cg, cb, ca := vColor.RGBA()
			path = append(path, ebiten.Vertex{
				DstX:   x,
				DstY:   y,
				ColorR: float32(cr) / 0xffff, ColorG: float32(cg) / 0xffff, ColorB: float32(cb) / 0xffff, ColorA: float32(ca) / 0xffff,
			})
		}
	}
	return path
}

func (self *PolygonComponent) calculateSegments(radius float32) int {
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

func (self *PolygonComponent) triangulateFill(fillPath []ebiten.Vertex, colors [4]color.RGBA) {
	baseIndex := uint16(len(self.Vertices))

	avgColor := interpolateColor(self.Radius, self.Radius, self.Radius*2, self.Radius*2, colors)
	cr, cg, cb, ca := avgColor.RGBA()

	center := ebiten.Vertex{
		DstX: self.Radius, DstY: self.Radius,
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

func generateMiterPathForPolygon(points []ebiten.Vertex, strokeWidth float32) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	numPoints := len(points)

	if numPoints < 3 {
		return points
	}

	for i := 0; i < numPoints; i++ {
		p_prev := points[(i+numPoints-1)%numPoints]
		p_curr := points[i]
		p_next := points[(i+1)%numPoints]

		e1 := struct{ x, y float32 }{p_curr.DstX - p_prev.DstX, p_curr.DstY - p_prev.DstY}
		e2 := struct{ x, y float32 }{p_next.DstX - p_curr.DstX, p_next.DstY - p_curr.DstY}

		len1 := float32(math.Sqrt(float64(e1.x*e1.x + e1.y*e1.y)))
		if len1 > 0 {
			e1.x /= len1
			e1.y /= len1
		}

		len2 := float32(math.Sqrt(float64(e2.x*e2.x + e2.y*e2.y)))
		if len2 > 0 {
			e2.x /= len2
			e2.y /= len2
		}

		// Assuming CCW winding order for polygons, outward normal is (-dy, dx)
		normal1 := struct{ x, y float32 }{-e1.y, e1.x}
		normal2 := struct{ x, y float32 }{-e2.y, e2.x}

		miter_vec := struct{ x, y float32 }{normal1.x + normal2.x, normal1.y + normal2.y}
		len_miter := float32(math.Sqrt(float64(miter_vec.x*miter_vec.x + miter_vec.y*miter_vec.y)))

		if len_miter > 1e-6 {
			miter_vec.x /= len_miter
			miter_vec.y /= len_miter
		} else {
			miter_vec = normal1
		}

		dot_product := miter_vec.x*normal1.x + miter_vec.y*normal1.y
		miter_len := strokeWidth
		if math.Abs(float64(dot_product)) > 1e-6 {
			miter_len = strokeWidth / dot_product
		}

		path[i] = ebiten.Vertex{
			DstX:   p_curr.DstX + miter_vec.x*miter_len,
			DstY:   p_curr.DstY + miter_vec.y*miter_len,
			ColorR: p_curr.ColorR, ColorG: p_curr.ColorG, ColorB: p_curr.ColorB, ColorA: p_curr.ColorA,
		}
	}
	return path
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
