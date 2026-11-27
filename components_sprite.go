package katsu2d

import "image/color"

// SpriteComponent stores texture, dimension, source bound, tint color, and opacity for rendering a sprite.
type SpriteComponent struct {
	TextureID     int
	Width, Height int
	Bound         Bound
	Color         color.RGBA
	Opacity       float64
}

// NewSpriteComponent creates a SpriteComponent with full white tint, full opacity (1.0),
// and a source Bound covering (0, 0) to (width, height).
func NewSpriteComponent(textureID, width, height int) SpriteComponent {
	return SpriteComponent{
		TextureID: textureID,
		Width:     width,
		Height:    height,
		Bound: Bound{
			Max: Point{X: float64(width), Y: float64(height)},
		},
		Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity: 1.0,
	}
}
