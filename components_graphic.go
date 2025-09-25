package katsu2d

import (
	"bytes"
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
	BaseVertices []ebiten.Vertex // Base vertices for the mesh
	Indices      []uint16
	Dirty        bool
}

func NewSpriteComponent(textureID int, bounds image.Rectangle) *SpriteComponent {
	res := &SpriteComponent{}
	res.TextureID = textureID
	res.DstW = float32(bounds.Dx())
	res.DstH = float32(bounds.Dy())
	res.SrcRect = &bounds
	res.Color = color.RGBA{255, 255, 255, 255}
	res.Opacity = 1.0
	res.Rows = 1
	res.Cols = 1
	res.MeshType = SpriteMeshTypeQuad
	res.TexMode = SpriteTextureModeStretch
	res.Dirty = true
	return res
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
		self.BaseVertices = nil

		self.Dirty = true
	}
	return self
}

func (self *SpriteComponent) SetVerticesAndIndices(vertices []ebiten.Vertex, indices []uint16) {
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	self.Vertices = vertices
	self.Indices = indices
	self.BaseVertices = make([]ebiten.Vertex, len(vertices))
	copy(self.BaseVertices, vertices)
	self.Dirty = false
}

// GenerateMesh creates the vertices and indices for the sprite's mesh.
func (self *SpriteComponent) GenerateMesh() {
	if !self.Dirty || self.MeshType == SpriteMeshTypeCustom {
		return
	}

	var srcRect image.Rectangle
	if self.SrcRect != nil {
		srcRect = *self.SrcRect
	} else {
		srcRect = image.Rect(0, 0, int(self.DstW), int(self.DstH))
	}

	numVertices := (self.Rows + 1) * (self.Cols + 1)
	self.BaseVertices = make([]ebiten.Vertex, numVertices)
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

	self.BaseVertices = make([]ebiten.Vertex, len(self.Vertices))
	copy(self.BaseVertices, self.Vertices)

	self.Dirty = false
}

func (self *SpriteComponent) ResetMesh() {
	self.Vertices = make([]ebiten.Vertex, len(self.BaseVertices))
	copy(self.Vertices, self.BaseVertices)

	self.Dirty = false
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
	Size, LineSpacing float64
	Color             color.RGBA
	FontFace          *text.GoTextFace
	CachedWidth       float64
	CachedHeight      float64
	CachedText        string
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
		FontFace: fontFace,
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
	if self.CachedText != self.Caption {
		self.CachedWidth, self.CachedHeight = text.Measure(self.Caption, self.FontFace, self.LineSpacing)
		self.CachedText = self.Caption
	}
}

func (self *TextComponent) SetLineSpacing(spacing float64) *TextComponent {
	self.LineSpacing = spacing
	self.UpdateCache()
	return self
}

func (self *TextComponent) SetAlignment(alignment TextAlignment) *TextComponent {
	self.Alignment = alignment
	return self
}

func (self *TextComponent) GetOffset() (float64, float64) {
	if offsetFunc, ok := alignmentOffsets[self.Alignment]; ok {
		offsetX, offsetY := offsetFunc(self.CachedWidth, self.CachedHeight)
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
	self.FontFace = fontFace
	self.FontFace.Size = self.Size
	self.UpdateCache()
	return self
}

func (self *TextComponent) SetSize(size float64) *TextComponent {
	self.FontFace.Size = size
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
