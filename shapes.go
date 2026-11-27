package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Shape represents a rebuildable vector shape that exposes triangle mesh buffers.
type Shape interface {
	Rebuild()
	GetVertices() []ebiten.Vertex
	GetIndices() []uint16
}

type shapePoint struct {
	x, y float32
}

type radii struct {
	tl, tr, bl, br float32
}

type cornerSegments struct {
	tl, tr, bl, br int
}

// RectangleShape defines an axis-aligned rectangle with optional rounded corners,
// per-corner fill/stroke gradients, and outer stroke geometry.
type RectangleShape struct {
	Vertices          []ebiten.Vertex
	Indices           []uint16
	Width, Height     float32
	TopLeftRadius     float32
	TopRightRadius    float32
	BottomLeftRadius  float32
	BottomRightRadius float32
	StrokeWidth       float32
	FillColors        [4]color.RGBA // 0:TL, 1:TR, 2:BR, 3:BL
	StrokeColors      [4]color.RGBA // 0:TL, 1:TR, 2:BR, 3:BL
	Dirty             bool
}

// NewRectangleShape creates a RectangleShape with the given dimensions and solid color.
func NewRectangleShape(width, height float32, col color.RGBA) *RectangleShape {
	return &RectangleShape{
		Width:        width,
		Height:       height,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		Dirty:        true,
	}
}

// SetColor sets the four corner fill colors (TL, TR, BR, BL) and marks the shape dirty.
func (r *RectangleShape) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	r.FillColors[0] = topLeft
	r.FillColors[1] = topRight
	r.FillColors[2] = bottomRight
	r.FillColors[3] = bottomLeft
	r.Dirty = true
}

// SetStrokeColor sets the four corner stroke colors (TL, TR, BR, BL) and marks the shape dirty.
func (r *RectangleShape) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	r.StrokeColors[0] = topLeft
	r.StrokeColors[1] = topRight
	r.StrokeColors[2] = bottomRight
	r.StrokeColors[3] = bottomLeft
	r.Dirty = true
}

// SetCornerRadius sets the corner radii for all four corners and marks the shape dirty.
func (r *RectangleShape) SetCornerRadius(topLeft, topRight, bottomLeft, bottomRight float32) {
	r.TopLeftRadius = topLeft
	r.TopRightRadius = topRight
	r.BottomLeftRadius = bottomLeft
	r.BottomRightRadius = bottomRight
	r.Dirty = true
}

// SetStroke sets the stroke width and uniform stroke color, marking the shape dirty.
func (r *RectangleShape) SetStroke(width float32, col color.RGBA) {
	r.StrokeWidth = width
	r.StrokeColors = [4]color.RGBA{col, col, col, col}
	r.Dirty = true
}

// Rebuild regenerates the rectangle's vertex and index buffers if dirty.
func (r *RectangleShape) Rebuild() {
	if !r.Dirty {
		return
	}
	r.Dirty = false
	r.Vertices = r.Vertices[:0]
	r.Indices = r.Indices[:0]
	r.generateFill()
	if r.StrokeWidth > 0 {
		r.generateStroke()
	}
}

func (r *RectangleShape) generateFill() {
	innerRadii := radii{r.TopLeftRadius, r.TopRightRadius, r.BottomLeftRadius, r.BottomRightRadius}
	seg := cornerSegments{
		tl: r.calculateSegments(innerRadii.tl),
		tr: r.calculateSegments(innerRadii.tr),
		bl: r.calculateSegments(innerRadii.bl),
		br: r.calculateSegments(innerRadii.br),
	}
	path := r.generatePath(r.Width, r.Height, innerRadii, r.FillColors, seg)
	r.Vertices, r.Indices = appendFillFan(r.Vertices, r.Indices, r.Width/2, r.Height/2, r.Width, r.Height, r.FillColors, path)
}

func (r *RectangleShape) generateStroke() {
	sw := r.StrokeWidth
	innerRadii := radii{r.TopLeftRadius, r.TopRightRadius, r.BottomLeftRadius, r.BottomRightRadius}
	outerRadii := innerRadii
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
	seg := cornerSegments{
		tl: r.calculateSegments(outerRadii.tl),
		tr: r.calculateSegments(outerRadii.tr),
		bl: r.calculateSegments(outerRadii.bl),
		br: r.calculateSegments(outerRadii.br),
	}
	innerPath := r.generatePath(r.Width, r.Height, innerRadii, r.StrokeColors, seg)
	outerPath := r.generatePath(r.Width+sw*2, r.Height+sw*2, outerRadii, r.StrokeColors, seg)
	for i := range outerPath {
		outerPath[i].DstX -= sw
		outerPath[i].DstY -= sw
	}
	r.Vertices, r.Indices = appendStrokeStrip(r.Vertices, r.Indices, outerPath, innerPath)
}

func (r *RectangleShape) calculateSegments(radius float32) int {
	if radius <= 0 {
		return 1
	}
	arcLength := float32(radius * math.Pi / 2)
	segs := int(arcLength / 1.5)
	if segs < 8 {
		segs = 8
	}
	if segs > 50 {
		segs = 50
	}
	return segs
}

func (r *RectangleShape) generatePath(width, height float32, rd radii, colors [4]color.RGBA, seg cornerSegments) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0, seg.tl+seg.tr+seg.br+seg.bl)
	path = r.appendCorner(path, width, height, rd.tl, rd.tl, rd.tl, 180, 270, colors, seg.tl)
	path = r.appendCorner(path, width, height, width-rd.tr, rd.tr, rd.tr, 270, 360, colors, seg.tr)
	path = r.appendCorner(path, width, height, width-rd.br, height-rd.br, rd.br, 0, 90, colors, seg.br)
	path = r.appendCorner(path, width, height, rd.bl, height-rd.bl, rd.bl, 90, 180, colors, seg.bl)
	return path
}

func (r *RectangleShape) appendCorner(dst []ebiten.Vertex, rectW, rectH, cx, cy, radius, startAngle, endAngle float32, colors [4]color.RGBA, segs int) []ebiten.Vertex {
	for i := 0; i < segs; i++ {
		angle := float64(startAngle)
		if segs > 1 {
			angle += float64(i) * float64(endAngle-startAngle) / float64(segs-1)
		}
		rad := angle * math.Pi / 180
		x := cx + radius*float32(math.Cos(rad))
		y := cy + radius*float32(math.Sin(rad))
		vColor := interpolateColor(x, y, rectW, rectH, colors)
		dst = append(dst, makeShapeVertex(x, y, vColor))
	}
	return dst
}

// GetVertices returns the rectangle's mesh vertices.
func (r *RectangleShape) GetVertices() []ebiten.Vertex {
	return r.Vertices
}

// GetIndices returns the rectangle's mesh triangle indices.
func (r *RectangleShape) GetIndices() []uint16 {
	return r.Indices
}

// CircleShape defines a circle with optional gradient fill and outer stroke geometry.
type CircleShape struct {
	Vertices     []ebiten.Vertex
	Indices      []uint16
	Radius       float32
	StrokeWidth  float32
	FillColors   [4]color.RGBA
	StrokeColors [4]color.RGBA
	Dirty        bool
}

// NewCircleShape creates a CircleShape with the given radius and solid color.
func NewCircleShape(radius float32, col color.RGBA) *CircleShape {
	return &CircleShape{
		Radius:       radius,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		Dirty:        true,
	}
}

// SetColor sets the four corner fill colors (TL, TR, BR, BL) and marks the shape dirty.
func (c *CircleShape) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	c.FillColors[0] = topLeft
	c.FillColors[1] = topRight
	c.FillColors[2] = bottomRight
	c.FillColors[3] = bottomLeft
	c.Dirty = true
}

// SetStrokeColor sets the four corner stroke colors (TL, TR, BR, BL) and marks the shape dirty.
func (c *CircleShape) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	c.StrokeColors[0] = topLeft
	c.StrokeColors[1] = topRight
	c.StrokeColors[2] = bottomRight
	c.StrokeColors[3] = bottomLeft
	c.Dirty = true
}

// SetStroke sets the stroke width and uniform stroke color, marking the shape dirty.
func (c *CircleShape) SetStroke(width float32, col color.RGBA) {
	c.StrokeWidth = width
	c.StrokeColors = [4]color.RGBA{col, col, col, col}
	c.Dirty = true
}

// GetVertices returns the circle's mesh vertices.
func (c *CircleShape) GetVertices() []ebiten.Vertex {
	return c.Vertices
}

// GetIndices returns the circle's mesh triangle indices.
func (c *CircleShape) GetIndices() []uint16 {
	return c.Indices
}

// Rebuild regenerates the circle's vertex and index buffers if dirty.
func (c *CircleShape) Rebuild() {
	if !c.Dirty {
		return
	}
	c.Dirty = false
	c.Vertices = c.Vertices[:0]
	c.Indices = c.Indices[:0]
	c.generateFill()
	if c.StrokeWidth > 0 {
		c.generateStroke()
	}
}

func (c *CircleShape) generateFill() {
	path := c.generatePath(c.Radius, c.FillColors)
	c.Vertices, c.Indices = appendFillFan(c.Vertices, c.Indices, c.Radius, c.Radius, c.Radius*2, c.Radius*2, c.FillColors, path)
}

func (c *CircleShape) generateStroke() {
	sw := c.StrokeWidth
	innerRadius := c.Radius
	outerRadius := c.Radius + sw
	innerPath := c.generatePath(innerRadius, c.StrokeColors)
	outerPath := c.generatePath(outerRadius, c.StrokeColors)
	for i := range outerPath {
		outerPath[i].DstX -= sw
		outerPath[i].DstY -= sw
	}
	c.Vertices, c.Indices = appendStrokeStrip(c.Vertices, c.Indices, outerPath, innerPath)
}

func (c *CircleShape) generatePath(radius float32, colors [4]color.RGBA) []ebiten.Vertex {
	segs := c.calculateSegments(radius)
	path := make([]ebiten.Vertex, 0, segs)
	for i := 0; i < segs; i++ {
		angle := 2 * math.Pi * float32(i) / float32(segs)
		x := radius + radius*float32(math.Cos(float64(angle)))
		y := radius + radius*float32(math.Sin(float64(angle)))
		vColor := interpolateColor(x, y, radius*2, radius*2, colors)
		path = append(path, makeShapeVertex(x, y, vColor))
	}
	return path
}

func (c *CircleShape) calculateSegments(radius float32) int {
	arcLength := 2 * math.Pi * radius
	segs := int(arcLength / 1.5)
	if segs < 12 {
		segs = 12
	}
	if segs > 200 {
		segs = 200
	}
	return segs
}

// TriangleShape defines an isosceles triangle fitted to Width x Height with optional rounded corners and stroke.
type TriangleShape struct {
	Vertices      []ebiten.Vertex
	Indices       []uint16
	Width, Height float32
	StrokeWidth   float32
	CornerRadius  float32
	FillColors    [4]color.RGBA // 0: Top, 1: Right, 2: Left
	StrokeColors  [4]color.RGBA
	Dirty         bool
}

// NewTriangleShape creates a TriangleShape with the given bounding dimensions and solid color.
func NewTriangleShape(width, height float32, col color.RGBA) *TriangleShape {
	return &TriangleShape{
		Width:        width,
		Height:       height,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		Dirty:        true,
	}
}

// SetColor sets the triangle's corner fill colors and marks the shape dirty.
func (t *TriangleShape) SetColor(top, right, left color.RGBA) {
	t.FillColors[0] = top
	t.FillColors[1] = right
	t.FillColors[2] = left
	t.Dirty = true
}

// SetStrokeColor sets the triangle's corner stroke colors and marks the shape dirty.
func (t *TriangleShape) SetStrokeColor(top, right, left color.RGBA) {
	t.StrokeColors[0] = top
	t.StrokeColors[1] = right
	t.StrokeColors[2] = left
	t.Dirty = true
}

// SetStroke sets the stroke width and uniform stroke color, marking the shape dirty.
func (t *TriangleShape) SetStroke(width float32, col color.RGBA) {
	t.StrokeWidth = width
	t.StrokeColors = [4]color.RGBA{col, col, col, col}
	t.Dirty = true
}

// SetCornerRadius sets the corner radius for all vertices and marks the shape dirty.
func (t *TriangleShape) SetCornerRadius(radius float32) {
	t.CornerRadius = radius
	t.Dirty = true
}

// GetVertices returns the triangle's mesh vertices.
func (t *TriangleShape) GetVertices() []ebiten.Vertex {
	return t.Vertices
}

// GetIndices returns the triangle's mesh triangle indices.
func (t *TriangleShape) GetIndices() []uint16 {
	return t.Indices
}

// Rebuild regenerates the triangle's vertex and index buffers if dirty.
func (t *TriangleShape) Rebuild() {
	if !t.Dirty {
		return
	}
	t.Dirty = false
	t.Vertices = t.Vertices[:0]
	t.Indices = t.Indices[:0]
	t.generateFill()
	if t.StrokeWidth > 0 {
		t.generateStroke()
	}
}

func (t *TriangleShape) generateFill() {
	path := t.generatePath(t.Width, t.Height, t.CornerRadius, t.FillColors)
	t.Vertices, t.Indices = appendFillFan(t.Vertices, t.Indices, t.Width/2, t.Height/2, t.Width, t.Height, t.FillColors, path)
}

func (t *TriangleShape) generateStroke() {
	sw := t.StrokeWidth
	innerRadius := t.CornerRadius
	outerRadius := innerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}
	seg := t.calculateSegments(outerRadius)
	var innerPath, outerPath []ebiten.Vertex
	innerPoints := []shapePoint{
		{t.Width / 2, 0},
		{t.Width, t.Height},
		{0, t.Height},
	}
	if t.CornerRadius > 0 {
		innerPath = generateRoundedPathForPolygon(innerPoints, innerRadius, t.StrokeColors, t.Width, t.Height, seg, t.calculateSegments)
		innerSharpVerts := t.generateSharpPath(innerPoints, t.StrokeColors)
		outerSharpVerts := generateMiterPathForPolygon(innerSharpVerts, sw)
		outerPoints := make([]shapePoint, len(outerSharpVerts))
		for i, v := range outerSharpVerts {
			outerPoints[i] = shapePoint{x: v.DstX, y: v.DstY}
		}
		outerPath = generateRoundedPathForPolygon(outerPoints, outerRadius, t.StrokeColors, t.Width, t.Height, seg, t.calculateSegments)
	} else {
		innerPath = t.generatePath(t.Width, t.Height, 0, t.StrokeColors)
		outerPath = generateMiterPathForPolygon(innerPath, sw)
	}
	t.Vertices, t.Indices = appendStrokeStrip(t.Vertices, t.Indices, outerPath, innerPath)
}

func (t *TriangleShape) generatePath(width, height, radius float32, colors [4]color.RGBA, segs ...int) []ebiten.Vertex {
	points := []shapePoint{
		{width / 2, 0},
		{width, height},
		{0, height},
	}
	if radius > 0 {
		var seg int
		if len(segs) > 0 {
			seg = segs[0]
		}
		return generateRoundedPathForPolygon(points, radius, colors, width, height, seg, t.calculateSegments)
	}
	return t.generateSharpPath(points, colors)
}

func (t *TriangleShape) generateSharpPath(points []shapePoint, colors [4]color.RGBA) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	for i, p := range points {
		vColor := interpolateColor(p.x, p.y, t.Width, t.Height, colors)
		path[i] = makeShapeVertex(p.x, p.y, vColor)
	}
	return path
}

func (t *TriangleShape) calculateSegments(radius float32) int {
	return calculatePolygonCornerSegments(radius)
}

// PolygonShape defines a regular N-sided polygon with optional rounded corners and stroke.
type PolygonShape struct {
	Vertices     []ebiten.Vertex
	Indices      []uint16
	Sides        int
	Radius       float32
	StrokeWidth  float32
	CornerRadius float32
	FillColors   [4]color.RGBA
	StrokeColors [4]color.RGBA
	Dirty        bool
}

// NewPolygonShape creates a regular PolygonShape with the given side count, radius, and solid color.
func NewPolygonShape(sides int, radius float32, col color.RGBA) *PolygonShape {
	return &PolygonShape{
		Sides:        sides,
		Radius:       radius,
		FillColors:   [4]color.RGBA{col, col, col, col},
		StrokeColors: [4]color.RGBA{col, col, col, col},
		Dirty:        true,
	}
}

// SetColor sets the four corner fill colors (TL, TR, BR, BL) and marks the shape dirty.
func (p *PolygonShape) SetColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	p.FillColors[0] = topLeft
	p.FillColors[1] = topRight
	p.FillColors[2] = bottomRight
	p.FillColors[3] = bottomLeft
	p.Dirty = true
}

// SetStrokeColor sets the four corner stroke colors (TL, TR, BR, BL) and marks the shape dirty.
func (p *PolygonShape) SetStrokeColor(topLeft, topRight, bottomRight, bottomLeft color.RGBA) {
	p.StrokeColors[0] = topLeft
	p.StrokeColors[1] = topRight
	p.StrokeColors[2] = bottomRight
	p.StrokeColors[3] = bottomLeft
	p.Dirty = true
}

// SetStroke sets the stroke width and uniform stroke color, marking the shape dirty.
func (p *PolygonShape) SetStroke(width float32, col color.RGBA) {
	p.StrokeWidth = width
	p.StrokeColors = [4]color.RGBA{col, col, col, col}
	p.Dirty = true
}

// SetCornerRadius sets the corner radius for all polygon vertices and marks the shape dirty.
func (p *PolygonShape) SetCornerRadius(radius float32) {
	p.CornerRadius = radius
	p.Dirty = true
}

// GetVertices returns the polygon's mesh vertices.
func (p *PolygonShape) GetVertices() []ebiten.Vertex {
	return p.Vertices
}

// GetIndices returns the polygon's mesh triangle indices.
func (p *PolygonShape) GetIndices() []uint16 {
	return p.Indices
}

// PentagonShape defines a 5-sided regular polygon.
type PentagonShape struct {
	PolygonShape
}

// NewPentagonShape creates a regular pentagon with the given radius and solid color.
func NewPentagonShape(radius float32, col color.RGBA) *PentagonShape {
	p := NewPolygonShape(5, radius, col)
	return &PentagonShape{PolygonShape: *p}
}

// HexagonShape defines a 6-sided regular polygon.
type HexagonShape struct {
	PolygonShape
}

// NewHexagonShape creates a regular hexagon with the given radius and solid color.
func NewHexagonShape(radius float32, col color.RGBA) *HexagonShape {
	p := NewPolygonShape(6, radius, col)
	return &HexagonShape{PolygonShape: *p}
}

// Rebuild regenerates the polygon's vertex and index buffers if dirty.
func (p *PolygonShape) Rebuild() {
	if !p.Dirty {
		return
	}
	p.Dirty = false
	p.Vertices = p.Vertices[:0]
	p.Indices = p.Indices[:0]
	p.generateFill()
	if p.StrokeWidth > 0 {
		p.generateStroke()
	}
}

func (p *PolygonShape) generateFill() {
	path := p.generatePath(p.Radius, p.CornerRadius, p.FillColors)
	p.Vertices, p.Indices = appendFillFan(p.Vertices, p.Indices, p.Radius, p.Radius, p.Radius*2, p.Radius*2, p.FillColors, path)
}

func (p *PolygonShape) generateStroke() {
	if p.Sides < 1 {
		return
	}
	sw := p.StrokeWidth
	innerRadius := p.CornerRadius
	outerRadius := innerRadius
	if innerRadius > 0 {
		outerRadius += sw
	}
	seg := p.calculateSegments(outerRadius)
	innerPoints := p.regularPoints(p.Radius)

	var innerPath, outerPath []ebiten.Vertex
	if p.CornerRadius > 0 {
		innerPath = generateRoundedPathForPolygon(innerPoints, innerRadius, p.StrokeColors, p.Radius*2, p.Radius*2, seg, p.calculateSegments)
		innerSharpVerts := p.generateSharpPath(innerPoints, p.StrokeColors, p.Radius)
		outerSharpVerts := generateMiterPathForPolygon(innerSharpVerts, sw)
		outerPoints := make([]shapePoint, len(outerSharpVerts))
		for i, v := range outerSharpVerts {
			outerPoints[i] = shapePoint{x: v.DstX, y: v.DstY}
		}
		outerPath = generateRoundedPathForPolygon(outerPoints, outerRadius, p.StrokeColors, p.Radius*2, p.Radius*2, seg, p.calculateSegments)
	} else {
		innerPath = p.generatePath(p.Radius, 0, p.StrokeColors)
		outerPath = generateMiterPathForPolygon(innerPath, sw)
	}
	p.Vertices, p.Indices = appendStrokeStrip(p.Vertices, p.Indices, outerPath, innerPath)
}

func (p *PolygonShape) regularPoints(radius float32) []shapePoint {
	if p.Sides < 1 {
		return nil
	}
	points := make([]shapePoint, p.Sides)
	angleStep := 2 * math.Pi / float32(p.Sides)
	for i := 0; i < p.Sides; i++ {
		angle := float32(i)*angleStep - float32(math.Pi/2)
		points[i] = shapePoint{
			x: radius + radius*float32(math.Cos(float64(angle))),
			y: radius + radius*float32(math.Sin(float64(angle))),
		}
	}
	return points
}

func (p *PolygonShape) generatePath(radius, cornerRadius float32, colors [4]color.RGBA, segs ...int) []ebiten.Vertex {
	points := p.regularPoints(radius)
	if cornerRadius > 0 {
		var seg int
		if len(segs) > 0 {
			seg = segs[0]
		}
		return generateRoundedPathForPolygon(points, cornerRadius, colors, radius*2, radius*2, seg, p.calculateSegments)
	}
	return p.generateSharpPath(points, colors, radius)
}

func (p *PolygonShape) generateSharpPath(points []shapePoint, colors [4]color.RGBA, radius float32) []ebiten.Vertex {
	path := make([]ebiten.Vertex, len(points))
	for i, pt := range points {
		vColor := interpolateColor(pt.x, pt.y, radius*2, radius*2, colors)
		path[i] = makeShapeVertex(pt.x, pt.y, vColor)
	}
	return path
}

func (p *PolygonShape) calculateSegments(radius float32) int {
	return calculatePolygonCornerSegments(radius)
}

func calculatePolygonCornerSegments(radius float32) int {
	if radius <= 0 {
		return 1
	}
	arcLength := float32(radius * math.Pi / 2)
	segs := int(arcLength / 1.5)
	if segs < 4 {
		segs = 4
	}
	if segs > 25 {
		segs = 25
	}
	return segs
}

func makeShapeVertex(x, y float32, col color.RGBA) ebiten.Vertex {
	cr, cg, cb, ca := col.RGBA()
	return ebiten.Vertex{
		DstX:   x,
		DstY:   y,
		ColorR: float32(cr) / 0xffff,
		ColorG: float32(cg) / 0xffff,
		ColorB: float32(cb) / 0xffff,
		ColorA: float32(ca) / 0xffff,
	}
}

func appendFillFan(vertices []ebiten.Vertex, indices []uint16, cx, cy, width, height float32, colors [4]color.RGBA, fillPath []ebiten.Vertex) ([]ebiten.Vertex, []uint16) {
	baseIndex := uint16(len(vertices))
	avgColor := interpolateColor(cx, cy, width, height, colors)
	vertices = append(vertices, makeShapeVertex(cx, cy, avgColor))
	vertices = append(vertices, fillPath...)
	numPerimeterVerts := len(fillPath)
	for i := range numPerimeterVerts {
		p1 := baseIndex + 1 + uint16(i)
		p2 := baseIndex + 1 + uint16((i+1)%numPerimeterVerts)
		indices = append(indices, baseIndex, p1, p2)
	}
	return vertices, indices
}

func appendStrokeStrip(vertices []ebiten.Vertex, indices []uint16, outerPath, innerPath []ebiten.Vertex) ([]ebiten.Vertex, []uint16) {
	if len(outerPath) != len(innerPath) || len(outerPath) == 0 {
		return vertices, indices
	}
	baseIndex := uint16(len(vertices))
	numVerts := len(outerPath)
	vertices = append(vertices, outerPath...)
	vertices = append(vertices, innerPath...)
	for i := range numVerts {
		p0 := baseIndex + uint16(i)
		p1 := baseIndex + uint16((i+1)%numVerts)
		p2 := baseIndex + uint16(i) + uint16(numVerts)
		p3 := baseIndex + uint16((i+1)%numVerts) + uint16(numVerts)
		indices = append(indices, p0, p2, p1, p1, p2, p3)
	}
	return vertices, indices
}

func generateRoundedPathForPolygon(points []shapePoint, cornerRadius float32, colors [4]color.RGBA, width, height float32, segs int, calcSegs func(radius float32) int) []ebiten.Vertex {
	path := make([]ebiten.Vertex, 0)
	numPoints := len(points)
	if numPoints < 3 {
		return path
	}
	if segs == 0 {
		segs = calcSegs(cornerRadius)
	}
	if segs < 1 {
		segs = 1
	}

	type cornerInfo struct {
		p, center, start, end Vector
		startAngle, endAngle  float32
		isDegenerate          bool
	}
	infos := make([]cornerInfo, numPoints)

	for i := range numPoints {
		prev := points[(i+numPoints-1)%numPoints]
		curr := points[i]
		next := points[(i+1)%numPoints]
		pPrev := Vector{X: float64(prev.x), Y: float64(prev.y)}
		pCurr := Vector{X: float64(curr.x), Y: float64(curr.y)}
		pNext := Vector{X: float64(next.x), Y: float64(next.y)}

		v1 := pPrev.Sub(pCurr)
		v2 := pNext.Sub(pCurr)
		len1 := v1.Length()
		len2 := v2.Length()
		if len1 > 0 {
			v1 = v1.ScaleF(1 / len1)
		}
		if len2 > 0 {
			v2 = v2.ScaleF(1 / len2)
		}

		angle := math.Acos(v1.Dot(v2))
		info := cornerInfo{p: pCurr, isDegenerate: true}
		if !math.IsNaN(angle) && angle > 1e-4 {
			maxDist := math.Min(len1/2, len2/2)
			dist := float64(cornerRadius) / math.Tan(angle/2)
			if dist > maxDist {
				dist = maxDist
			}
			cappedRadius := dist * math.Tan(angle/2)
			if angle < math.Pi-1e-4 {
				info.isDegenerate = false
				info.start = pCurr.Add(v1.ScaleF(dist))
				info.end = pCurr.Add(v2.ScaleF(dist))
				bisector := v1.Add(v2).Normalize()
				centerDist := cappedRadius / math.Sin(angle/2)
				info.center = pCurr.Add(bisector.ScaleF(centerDist))
				vStart := info.start.Sub(info.center)
				vEnd := info.end.Sub(info.center)
				info.startAngle = float32(math.Atan2(vStart.Y, vStart.X))
				info.endAngle = float32(math.Atan2(vEnd.Y, vEnd.X))
				if v1.Cross(v2) < 0 {
					if info.endAngle < info.startAngle {
						info.endAngle += 2 * math.Pi
					}
				} else {
					if info.endAngle > info.startAngle {
						info.endAngle -= 2 * math.Pi
					}
				}
			}
		}
		infos[i] = info
	}

	for _, info := range infos {
		if info.isDegenerate {
			vColor := interpolateColor(float32(info.p.X), float32(info.p.Y), width, height, colors)
			path = append(path, makeShapeVertex(float32(info.p.X), float32(info.p.Y), vColor))
			continue
		}
		radius := info.start.DistanceTo(info.center)
		for j := 0; j < segs; j++ {
			ratio := float32(0)
			if segs > 1 {
				ratio = float32(j) / float32(segs-1)
			}
			interpAngle := info.startAngle + ratio*(info.endAngle-info.startAngle)
			x := info.center.X + radius*math.Cos(float64(interpAngle))
			y := info.center.Y + radius*math.Sin(float64(interpAngle))
			vColor := interpolateColor(float32(x), float32(y), width, height, colors)
			path = append(path, makeShapeVertex(float32(x), float32(y), vColor))
		}
	}
	return path
}

func generateMiterPathForPolygon(points []ebiten.Vertex, strokeWidth float32) []ebiten.Vertex {
	numPoints := len(points)
	if numPoints < 3 {
		return points
	}
	path := make([]ebiten.Vertex, numPoints)
	for i := range numPoints {
		pPrev := points[(i+numPoints-1)%numPoints]
		pCurr := points[i]
		pNext := points[(i+1)%numPoints]
		e1 := shapePoint{pCurr.DstX - pPrev.DstX, pCurr.DstY - pPrev.DstY}
		e2 := shapePoint{pNext.DstX - pCurr.DstX, pNext.DstY - pCurr.DstY}
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
		// Outward normals for CCW polygon: (dy, -dx).
		normal1 := shapePoint{e1.y, -e1.x}
		normal2 := shapePoint{e2.y, -e2.x}
		miterVec := shapePoint{normal1.x + normal2.x, normal1.y + normal2.y}
		lenMiter := float32(math.Sqrt(float64(miterVec.x*miterVec.x + miterVec.y*miterVec.y)))
		if lenMiter > 1e-6 {
			miterVec.x /= lenMiter
			miterVec.y /= lenMiter
		} else {
			miterVec = normal1
		}
		dotProduct := miterVec.x*normal1.x + miterVec.y*normal1.y
		miterLen := strokeWidth
		if math.Abs(float64(dotProduct)) > 1e-6 {
			miterLen = strokeWidth / dotProduct
		}
		path[i] = ebiten.Vertex{
			DstX:   pCurr.DstX + miterVec.x*miterLen,
			DstY:   pCurr.DstY + miterVec.y*miterLen,
			ColorR: pCurr.ColorR,
			ColorG: pCurr.ColorG,
			ColorB: pCurr.ColorB,
			ColorA: pCurr.ColorA,
		}
	}
	return path
}

func interpolateColor(x, y, width, height float32, colors [4]color.RGBA) color.RGBA {
	var u, v float32
	if width > 0 {
		u = min(max(x/width, 0), 1)
	}
	if height > 0 {
		v = min(max(y/height, 0), 1)
	}
	c00 := colors[0] // TL
	c10 := colors[1] // TR
	c01 := colors[3] // BL
	c11 := colors[2] // BR
	r00, g00, b00, a00 := c00.RGBA()
	r10, g10, b10, a10 := c10.RGBA()
	r01, g01, b01, a01 := c01.RGBA()
	r11, g11, b11, a11 := c11.RGBA()
	rInterp := float32(r00)*(1-u)*(1-v) + float32(r10)*u*(1-v) + float32(r01)*(1-u)*v + float32(r11)*u*v
	gInterp := float32(g00)*(1-u)*(1-v) + float32(g10)*u*(1-v) + float32(g01)*(1-u)*v + float32(g11)*u*v
	bInterp := float32(b00)*(1-u)*(1-v) + float32(b10)*u*(1-v) + float32(b01)*(1-u)*v + float32(b11)*u*v
	aInterp := float32(a00)*(1-u)*(1-v) + float32(a10)*u*(1-v) + float32(a01)*(1-u)*v + float32(a11)*u*v
	return color.RGBA{
		R: uint8(rInterp / 256),
		G: uint8(gInterp / 256),
		B: uint8(bInterp / 256),
		A: uint8(aInterp / 256),
	}
}
