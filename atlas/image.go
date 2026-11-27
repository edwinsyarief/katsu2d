package atlas

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Image represents a single sub-image or region within an Atlas.
// It holds a reference to its parent Atlas and its specific bounds
// within the atlas's main texture.
type Image struct {
	// atlas is a pointer to the parent Atlas that this image belongs to.
	atlas *Atlas

	// bounds defines the rectangular region of this image within the parent atlas.
	bounds *image.Rectangle
}

// Atlas returns the parent Atlas to which this image belongs.
func (i *Image) Atlas() *Atlas {
	return i.atlas
}

// Bounds returns the rectangular region of the image within its parent atlas.
func (i *Image) Bounds() image.Rectangle {
	return *i.bounds
}

// Image returns an *ebiten.Image that represents this specific sub-image.
// It achieves this by taking a sub-image of the parent atlas's native image.
func (i *Image) Image() *ebiten.Image {
	if i.atlas == nil || i.atlas.native == nil {
		return nil
	}
	return i.atlas.native.SubImage(*i.bounds).(*ebiten.Image)
}

// DrawImage draws the given source image onto this atlas sub-image.
// It correctly translates the drawing options to ensure the source image
// is rendered within the sub-image's bounds on the main atlas texture.
func (i *Image) DrawImage(src *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if src == nil {
		return
	}
	dst := i.Image()
	if dst == nil {
		return
	}
	var drawOpts ebiten.DrawImageOptions
	if opts != nil {
		drawOpts = *opts
	}
	// Translate to the top-left corner of this sub-image's bounds on the atlas
	// without mutating the caller's DrawImageOptions.
	drawOpts.GeoM.Translate(
		float64(i.bounds.Min.X),
		float64(i.bounds.Min.Y),
	)
	// Draw the source image to the destination (the sub-image region on the atlas).
	dst.DrawImage(src, &drawOpts)
}

// SubImage returns a new *atlas.Image that represents a region
// within the current image. This is useful for creating sprites from a
// smaller section of a larger sprite within the atlas.
func (i *Image) SubImage(bounds image.Rectangle) *Image {
	// Adjust the bounds of the new sub-image to be relative to the parent atlas.
	bounds = bounds.Add(i.bounds.Min)

	// Return a new Image struct representing this nested region.
	return &Image{
		atlas:  i.atlas,
		bounds: &bounds,
	}
}
