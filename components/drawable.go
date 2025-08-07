package components

import (
	"image/color"
	"katsu2d/constants"
)

// Drawable component for rendering a single sprite with its own draw call.
// This is for entities that need individual draw calls, which is less efficient.
type Drawable struct {
	TextureID     int
	Width, Height float64
	Color         color.RGBA
}

// GetTypeID returns the component type ID.
func (d *Drawable) GetTypeID() int {
	return constants.ComponentDrawable
}

// DrawableBatch component for rendering a sprite in a batch.
// This is for entities that can be grouped into a single draw call, which is more efficient.
type DrawableBatch struct {
	TextureID     int
	Width, Height float64
	Color         color.RGBA
	// Source rectangle in the texture atlas
	SrcX, SrcY, SrcW, SrcH float64
}

// GetTypeID returns the component type ID.
func (self *DrawableBatch) GetTypeID() int {
	return constants.ComponentDrawableBatch
}
