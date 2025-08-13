package katsu2d

import (
	"image"
	"image/color"

	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// Transform component defines position, offset, origin, scale, rotation, and z-index.
type Transform struct {
	*ebimath.Transform
	Z float64 // for draw order
}

// NewTransform creates a new Transform component with default values.
func NewTransform() *Transform {
	return &Transform{
		Transform: ebimath.T(),
	}
}

// Sprite component defines texture, source rect, destination size, and color tint.
type Sprite struct {
	TextureID int
	SrcX      float32
	SrcY      float32
	SrcW      float32 // if 0, use whole texture width
	SrcH      float32 // if 0, use whole texture height
	DstW      float32 // if 0, use SrcW
	DstH      float32 // if 0, use SrcH
	Color     color.RGBA
	Opacity   float32
}

// NewSprite creates a new Sprite component for a given texture and destination size.
func NewSprite(textureID, width, height int) *Sprite {
	return &Sprite{
		TextureID: textureID,
		DstW:      float32(width),
		DstH:      float32(height),
		Color:     color.RGBA{255, 255, 255, 255},
		Opacity:   1.0,
	}
}

// GetSourceRect calculates the source rectangle coordinates and size.
func (self *Sprite) GetSourceRect(textureWidth, textureHeight float32) (x, y, w, h float32) {
	x, y = self.SrcX, self.SrcY
	w, h = self.SrcW, self.SrcH
	if w == 0 || h == 0 {
		w, h = textureWidth, textureHeight
	}
	return
}

// GetDestSize calculates the destination size, using source size as a fallback.
func (self *Sprite) GetDestSize(sourceWidth, sourceHeight float32) (w, h float32) {
	w, h = self.DstW, self.DstH
	if w == 0 {
		w = sourceWidth
	}
	if h == 0 {
		h = sourceHeight
	}
	return
}

// AnimMode defines animation playback modes.
type AnimMode int

const (
	AnimLoop      AnimMode = iota // Loop forever
	AnimOnce                      // Play once and stop
	AnimBoomerang                 // Forward then backward loop
)

// Animation component for sprite frame animations.
type Animation struct {
	Frames    []image.Rectangle
	Speed     float64  // Seconds per frame
	Elapsed   float64  // Time since last frame
	Current   int      // Current frame index
	Mode      AnimMode // Animation mode
	Direction bool     // For boomerang: true forward, false backward
	Active    bool     // Is animation playing
}

// TextAlignment defines text alignment modes.
type TextAlignment int

const (
	TextAlignmentTopLeft TextAlignment = iota
	TextAlignmentMiddleLeft
	TextAlignmentBottomLeft
	TextAlignmentTopCenter
	TextAlignmentMiddleCenter
	TextAlignmentBottomCenter
	TextAlignmentTopRight
	TextAlignmentMiddleRight
	TextAlignmentBottomRight
)

// alignmentOffsets stores pre-calculated offset functions
var alignmentOffsets = map[TextAlignment]func(w, h float64) (float64, float64){
	TextAlignmentMiddleLeft:   func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomLeft:   func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentTopCenter:    func(w, h float64) (float64, float64) { return -w / 2, 0 },
	TextAlignmentMiddleCenter: func(w, h float64) (float64, float64) { return -w / 2, -h / 2 },
	TextAlignmentBottomCenter: func(w, h float64) (float64, float64) { return -w / 2, -h },
	TextAlignmentTopRight:     func(w, h float64) (float64, float64) { return -w, 0 },
	TextAlignmentMiddleRight:  func(w, h float64) (float64, float64) { return -w, -h / 2 },
	TextAlignmentBottomRight:  func(w, h float64) (float64, float64) { return -w, -h },
	TextAlignmentTopLeft:      func(w, h float64) (float64, float64) { return 0, 0 },
}

// Text component for drawing text.
type Text struct {
	Alignment TextAlignment
	Caption   string
	Size      float64
	Color     color.RGBA

	fontFace *text.GoTextFace

	// Cache for text measurements
	cachedWidth, cachedHeight float64
	cachedText                string
}

// NewText creates a new Text component with the specified font source, caption, size, and color.
func NewText(source *text.GoTextFaceSource, caption string, size float64, col color.RGBA) *Text {
	fontFace := &text.GoTextFace{
		Source:    source,
		Direction: text.DirectionLeftToRight,
		Size:      size,
		Language:  language.English,
	}
	result := &Text{
		Caption:  caption,
		Size:     size,
		Color:    col,
		fontFace: fontFace,
	}
	result.updateCache()
	return result
}

// updateCache updates the cached measurements for the text if the caption has changed.
func (self *Text) updateCache() {
	if self.cachedText != self.Caption {
		self.cachedWidth, self.cachedHeight = text.Measure(self.Caption, self.fontFace, 0)
		self.cachedText = self.Caption
	}
}

// SetAlignment sets the alignment for the text and returns the Text for chaining.
func (self *Text) SetAlignment(alignment TextAlignment) *Text {
	self.Alignment = alignment
	return self
}

// SetText sets the caption for the text and returns the Text for chaining.
func (self *Text) SetText(text string) *Text {
	if self.Caption != text {
		self.Caption = text
		self.updateCache()
	}
	return self
}

// SetFontFace sets the font face for the text and returns the Text for chaining.
func (self *Text) SetFontFace(fontFace *text.GoTextFace) *Text {
	self.fontFace = fontFace
	self.updateCache()
	return self
}

// SetSize sets the size for the text and returns the Text for chaining.
func (self *Text) SetSize(size float64) *Text {
	self.fontFace.Size = size
	self.updateCache()
	return self
}

// SetColor sets the color for the text and returns the Text for chaining.
func (self *Text) SetColor(color color.RGBA) *Text {
	self.Color = color
	return self
}

// SetOpacity sets the opacity by adjusting the alpha channel of the color and returns the Text for chaining.
func (self *Text) SetOpacity(opacity float64) *Text {
	val := ebimath.Min(ebimath.Max(opacity, 0.0), 1.0)

	col := self.Color
	col.A = uint8(255 * val)
	self.SetColor(col)

	return self
}

// Draw method for the Text component.
// This is not intended to be called directly by a user, but by a DrawSystem.
func (self *Text) Draw(transform *ebimath.Transform, screen *ebiten.Image) {
	self.updateCache()

	op := &text.DrawOptions{}
	op.GeoM = transform.Matrix()

	if offsetFunc, ok := alignmentOffsets[self.Alignment]; ok {
		offsetX, offsetY := offsetFunc(float64(self.cachedWidth), float64(self.cachedHeight))
		op.GeoM.Translate(offsetX, offsetY)
	}

	op.ColorScale = utils.RGBAToColorScale(self.Color)
	text.Draw(screen, self.Caption, self.fontFace, op)
}
