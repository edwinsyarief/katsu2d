package katsu2d

import "image/color"

type SpriteComponent struct {
	TextureID     int
	Width, Height int
	Bound         Bound
	Color         color.RGBA
	Opacity       float64
}
