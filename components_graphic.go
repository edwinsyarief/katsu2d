package katsu2d

import (
	"bytes"
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

type SpriteMeshType int

const (
	// SpriteMeshTypeQuad represents a simple quad mesh.
	SpriteMeshTypeQuad SpriteMeshType = iota
	// SpriteMeshTypeGrid represents a grid mesh.
	SpriteMeshTypeGrid
	// SpriteMeshTypeCustom represents a custom mesh defined by the user.
	SpriteMeshTypeCustom
)

// SpriteTextureMode defines how the texture is applied to the mesh.
type SpriteTextureMode int

const (
	// SpriteTextureModeStretch stretches the texture to fit the entire mesh.
	SpriteTextureModeStretch SpriteTextureMode = iota
	// SpriteTextureModeTile is not yet implemented.
	SpriteTextureModeTile
)

type SpriteComponent struct {
	TextureID int
	DstW      float32
	DstH      float32
	SrcRect   *image.Rectangle
	Color     color.RGBA
	Opacity   float32

	// Mesh properties
	Rows         int
	Cols         int
	MeshType     SpriteMeshType
	TexMode      SpriteTextureMode
	Vertices     []ebiten.Vertex
	baseVertices []ebiten.Vertex // Base vertices for the mesh
	Indices      []uint16
	dirty        bool
}

// NewSpriteComponent creates a new sprite component for a simple quad.
func NewSpriteComponent(textureID int, bounds image.Rectangle) *SpriteComponent {
	s := &SpriteComponent{
		TextureID: textureID,
		DstW:      float32(bounds.Dx()),
		DstH:      float32(bounds.Dy()),
		SrcRect:   &bounds,
		Color:     color.RGBA{255, 255, 255, 255},
		Opacity:   1.0,
		Rows:      1,
		Cols:      1,
		MeshType:  SpriteMeshTypeQuad,
		TexMode:   SpriteTextureModeStretch,
		dirty:     true,
	}
	return s
}

// SetGrid sets the number of rows and columns for the mesh.
func (self *SpriteComponent) SetGrid(rows, cols int) *SpriteComponent {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	if self.Rows != rows || self.Cols != cols {
		self.Rows = rows
		self.Cols = cols
		if self.Cols > 1 || self.Rows > 1 {
			self.MeshType = SpriteMeshTypeGrid
		} else {
			self.MeshType = SpriteMeshTypeQuad
		}
		self.Vertices = nil
		self.Indices = nil
		self.baseVertices = nil

		self.dirty = true
	}
	return self
}

// IsDirty returns true if the mesh needs to be regeerated.
func (self *SpriteComponent) IsDirty() bool {
	return self.dirty
}

// GetSourceRect returns the source rectangle for drawing.
func (self *SpriteComponent) GetSourceRect() image.Rectangle {
	if self.SrcRect != nil {
		return *self.SrcRect
	}
	return image.Rect(0, 0, int(self.DstW), int(self.DstH))
}

func (self *SpriteComponent) SetVerticesAndIndices(vertices []ebiten.Vertex, indices []uint16) {
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	self.Vertices = vertices
	self.Indices = indices
	self.baseVertices = make([]ebiten.Vertex, len(vertices))
	copy(self.baseVertices, vertices)
	self.dirty = false
}

// GenerateMesh creates the vertices and indices for the sprite's mesh.
func (self *SpriteComponent) GenerateMesh() {
	if !self.dirty || self.MeshType == SpriteMeshTypeCustom {
		return
	}

	srcRect := self.GetSourceRect()

	numVertices := (self.Rows + 1) * (self.Cols + 1)
	self.baseVertices = make([]ebiten.Vertex, numVertices)
	self.Vertices = make([]ebiten.Vertex, numVertices)

	for r := 0; r <= self.Rows; r++ {
		for c := 0; c <= self.Cols; c++ {
			idx := r*(self.Cols+1) + c
			vx := float32(self.DstW * (float32(c) / float32(self.Cols)))
			vy := float32(self.DstH * (float32(r) / float32(self.Rows)))
			u := float32(srcRect.Min.X) + float32(srcRect.Dx())*(float32(c)/float32(self.Cols))
			v := float32(srcRect.Min.Y) + float32(srcRect.Dy())*(float32(r)/float32(self.Rows))

			self.Vertices[idx] = ebiten.Vertex{
				DstX:   vx,
				DstY:   vy,
				SrcX:   u,
				SrcY:   v,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			}
		}
	}

	numQuads := self.Rows * self.Cols
	self.Indices = make([]uint16, numQuads*6)
	i := 0
	for r := 0; r < self.Rows; r++ {
		for c := 0; c < self.Cols; c++ {
			topLeft := uint16(r*(self.Cols+1) + c)
			topRight := topLeft + 1
			bottomLeft := uint16((r+1)*(self.Cols+1) + c)
			bottomRight := bottomLeft + 1
			self.Indices[i] = topLeft
			self.Indices[i+1] = topRight
			self.Indices[i+2] = bottomLeft
			self.Indices[i+3] = bottomLeft
			self.Indices[i+4] = topRight
			self.Indices[i+5] = bottomRight
			i += 6
		}
	}

	self.baseVertices = make([]ebiten.Vertex, len(self.Vertices))
	copy(self.baseVertices, self.Vertices)

	self.dirty = false
}

func (self *SpriteComponent) ResetMesh() {
	self.Vertices = make([]ebiten.Vertex, len(self.baseVertices))
	copy(self.Vertices, self.baseVertices)

	self.dirty = false
}

type AnimMode int

const (
	AnimLoop AnimMode = iota
	AnimOnce
	AnimBoomerang
)

type AnimationComponent struct {
	Frames    []image.Rectangle
	Speed     float64
	Elapsed   float64
	Current   int
	Mode      AnimMode
	Direction bool
	Active    bool
}

type TextAlignment int

const (
	TextAlignmentBottomRight TextAlignment = iota
	TextAlignmentMiddleRight
	TextAlignmentTopRight
	TextAlignmentBottomCenter
	TextAlignmentMiddleCenter
	TextAlignmentTopCenter
	TextAlignmentBottomLeft
	TextAlignmentMiddleLeft
	TextAlignmentTopLeft
)

var alignmentOffsets = map[TextAlignment]func(w, h float64) (float64, float64){
	TextAlignmentTopRight:     func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleRight:  func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomRight:  func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopCenter:    func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleCenter: func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomCenter: func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopLeft:      func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleLeft:   func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomLeft:   func(w, h float64) (float64, float64) { return 0, 0 },
}

type TextComponent struct {
	Alignment         TextAlignment
	Caption           string
	Size, lineSpacing float64
	Color             color.RGBA
	fontFace          *text.GoTextFace
	cachedWidth       float64
	cachedHeight      float64
	cachedText        string
}

func NewTextComponent(source *text.GoTextFaceSource, caption string, size float64, col color.RGBA) *TextComponent {
	fontFace := &text.GoTextFace{
		Source:    source,
		Direction: text.DirectionLeftToRight,
		Size:      size,
		Language:  language.English,
	}
	result := &TextComponent{
		Caption:  caption,
		Size:     size,
		Color:    col,
		fontFace: fontFace,
	}
	result.UpdateCache()
	return result
}

func NewDefaultTextComponent(caption string, size float64, col color.RGBA) *TextComponent {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(_DefaultFont))
	if err != nil {
		panic(err)
	}
	return NewTextComponent(font, caption, size, col)
}

func (self *TextComponent) UpdateCache() {
	if self.cachedText != self.Caption {
		self.cachedWidth, self.cachedHeight = text.Measure(self.Caption, self.fontFace, self.lineSpacing)
		self.cachedText = self.Caption
	}
}

func (self *TextComponent) LineSpacing() float64 {
	return self.lineSpacing
}

func (self *TextComponent) SetLineSpacing(spacing float64) *TextComponent {
	self.lineSpacing = spacing
	self.UpdateCache()
	return self
}

func (self *TextComponent) SetAlignment(alignment TextAlignment) *TextComponent {
	self.Alignment = alignment
	return self
}

func (self *TextComponent) GetOffset() (float64, float64) {
	if offsetFunc, ok := alignmentOffsets[self.Alignment]; ok {
		offsetX, offsetY := offsetFunc(self.cachedWidth, self.cachedHeight)
		return offsetX, offsetY
	}
	return 0, 0
}

func (self *TextComponent) SetText(text string) *TextComponent {
	if self.Caption != text {
		self.Caption = text
		self.UpdateCache()
	}
	return self
}

func (self *TextComponent) SetFontFace(fontFace *text.GoTextFace) *TextComponent {
	self.fontFace = fontFace
	self.UpdateCache()
	return self
}

func (self *TextComponent) FontFace() *text.GoTextFace {
	return self.fontFace
}

func (self *TextComponent) SetSize(size float64) *TextComponent {
	self.fontFace.Size = size
	self.UpdateCache()
	return self
}

func (self *TextComponent) SetColor(color color.RGBA) *TextComponent {
	self.Color = color
	return self
}

func (self *TextComponent) SetOpacity(opacity float64) *TextComponent {
	val := ebimath.Min(ebimath.Max(opacity, 0.0), 1.0)
	col := self.Color
	col.A = uint8(255 * val)
	self.SetColor(col)
	return self
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

type rectCornerRadii struct{ tl, tr, bl, br float32 }
type rectCornerSegments struct{ tl, tr, bl, br int }

func (self *RectangleComponent) generateFill() {
	innerRadii := rectCornerRadii{self.TopLeftRadius, self.TopRightRadius, self.BottomLeftRadius, self.BottomRightRadius}
	seg := rectCornerSegments{
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
	innerRadii := rectCornerRadii{self.TopLeftRadius, self.TopRightRadius, self.BottomLeftRadius, self.BottomRightRadius}
	outerRadii := rectCornerRadii{innerRadii.tl, innerRadii.tr, innerRadii.bl, innerRadii.br}
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

	seg := rectCornerSegments{
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

func (self *RectangleComponent) generatePath(width, height float32, rd rectCornerRadii, colors [4]color.RGBA, seg rectCornerSegments) []ebiten.Vertex {
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
