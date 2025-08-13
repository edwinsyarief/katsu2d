package katsu2d

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// TransformComponent component defines position, offset, origin, scale, rotation, and z-index.
type TransformComponent struct {
	*ebimath.Transform
	Z float64 // for draw order
}

// NewTransformComponent creates a new Transform component with default values.
func NewTransformComponent() *TransformComponent {
	return &TransformComponent{
		Transform: ebimath.T(),
	}
}

// SpriteComponent component defines texture, source rect, destination size, and color tint.
type SpriteComponent struct {
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

// NewSpriteComponent creates a new Sprite component for a given texture and destination size.
func NewSpriteComponent(textureID, width, height int) *SpriteComponent {
	return &SpriteComponent{
		TextureID: textureID,
		DstW:      float32(width),
		DstH:      float32(height),
		Color:     color.RGBA{255, 255, 255, 255},
		Opacity:   1.0,
	}
}

// GetSourceRect calculates the source rectangle coordinates and size.
func (self *SpriteComponent) GetSourceRect(textureWidth, textureHeight float32) (x, y, w, h float32) {
	x, y = self.SrcX, self.SrcY
	w, h = self.SrcW, self.SrcH
	if w == 0 || h == 0 {
		w, h = textureWidth, textureHeight
	}
	return
}

// GetDestSize calculates the destination size, using source size as a fallback.
func (self *SpriteComponent) GetDestSize(sourceWidth, sourceHeight float32) (w, h float32) {
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

// AnimationComponent component for sprite frame animations.
type AnimationComponent struct {
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

// TextComponent component for drawing text.
type TextComponent struct {
	Alignment TextAlignment
	Caption   string
	Size      float64
	Color     color.RGBA

	fontFace *text.GoTextFace

	// Cache for text measurements
	cachedWidth, cachedHeight float64
	cachedText                string
}

// NewTextComponent creates a new Text component with the specified font source, caption, size, and color.
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

// updateCache updates the cached measurements for the text if the caption has changed.
func (self *TextComponent) updateCache() {
	if self.cachedText != self.Caption {
		self.cachedWidth, self.cachedHeight = text.Measure(self.Caption, self.fontFace, 0)
		self.cachedText = self.Caption
	}
}

// SetAlignment sets the alignment for the text and returns the Text for chaining.
func (self *TextComponent) SetAlignment(alignment TextAlignment) *TextComponent {
	self.Alignment = alignment
	return self
}

// SetText sets the caption for the text and returns the Text for chaining.
func (self *TextComponent) SetText(text string) *TextComponent {
	if self.Caption != text {
		self.Caption = text
		self.updateCache()
	}
	return self
}

// SetFontFace sets the font face for the text and returns the Text for chaining.
func (self *TextComponent) SetFontFace(fontFace *text.GoTextFace) *TextComponent {
	self.fontFace = fontFace
	self.updateCache()
	return self
}

func (self *TextComponent) FontFace() *text.GoTextFace {
	return self.fontFace
}

// SetSize sets the size for the text and returns the Text for chaining.
func (self *TextComponent) SetSize(size float64) *TextComponent {
	self.fontFace.Size = size
	self.updateCache()
	return self
}

// SetColor sets the color for the text and returns the Text for chaining.
func (self *TextComponent) SetColor(color color.RGBA) *TextComponent {
	self.Color = color
	return self
}

// SetOpacity sets the opacity by adjusting the alpha channel of the color and returns the Text for chaining.
func (self *TextComponent) SetOpacity(opacity float64) *TextComponent {
	val := ebimath.Min(ebimath.Max(opacity, 0.0), 1.0)

	col := self.Color
	col.A = uint8(255 * val)
	self.SetColor(col)

	return self
}
