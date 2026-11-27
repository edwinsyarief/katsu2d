package katsu2d

import "image/color"

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

type TextComponent struct {
	Caption           string
	CachedText        string
	FontID            int
	Size, LineSpacing float64
	Alignment         TextAlignment
	CachedWidth       float64
	CachedHeight      float64
	Color             color.RGBA

	cachedFontID      int
	cachedSize        float64
	cachedLineSpacing float64
}

// NewTextComponent creates a TextComponent with the given caption, font ID, font size, and color.
func NewTextComponent(caption string, fontID int, size float64, col color.RGBA) TextComponent {
	return TextComponent{
		Caption:     caption,
		FontID:      fontID,
		Size:        size,
		LineSpacing: size,
		Color:       col,
	}
}
