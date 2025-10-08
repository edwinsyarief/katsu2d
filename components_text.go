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
}
