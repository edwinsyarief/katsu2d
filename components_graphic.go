package katsu2d

import (
	"image"
	"image/color"

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
	result.updateCache()
	return result
}

func (self *TextComponent) updateCache() {
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
	self.updateCache()
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
		self.updateCache()
	}
	return self
}

func (self *TextComponent) SetFontFace(fontFace *text.GoTextFace) *TextComponent {
	self.fontFace = fontFace
	self.updateCache()
	return self
}

func (self *TextComponent) FontFace() *text.GoTextFace {
	return self.fontFace
}

func (self *TextComponent) SetSize(size float64) *TextComponent {
	self.fontFace.Size = size
	self.updateCache()
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
