package katsu2d

import (
	"image"
	"image/color"

	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/edwinsyarief/katsu2d/tween"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// Transform component defines position, offset, origin, scale, rotation, and z-index
type Transform struct {
	*ebimath.Transform
	Z float64 // for draw order
}

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
}

func NewSprite(textureID, width, height int) *Sprite {
	return &Sprite{
		TextureID: textureID,
		DstW:      float32(width),
		DstH:      float32(height),
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

// Tween component for interpolating a value over time.
type Tween struct {
	*tween.Tween
}

// NewTween creates a new Tween instance with the specified start value, end value,
// duration, and easing function. The tween starts at the initial value with a
// default delay of 0.
func NewTween(start, end, duration float32, easing ease.EaseFunc) *Tween {
	return &Tween{
		Tween: tween.New(start, end, duration, easing),
	}
}

// Sequence component for chaining tweens.
type Sequence struct {
	*tween.Sequence
}

func NewSequence(tweens ...*tween.Tween) *Sequence {
	return &Sequence{
		Sequence: tween.NewSequence(tweens...),
	}
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
