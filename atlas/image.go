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
func (self *Image) Atlas() *Atlas {
	return self.atlas
}

// Bounds returns the rectangular region of the image within its parent atlas.
func (self *Image) Bounds() image.Rectangle {
	return *self.bounds
}

// Image returns an *ebiten.Image that represents this specific sub-image.
// It achieves this by taking a sub-image of the parent atlas's native image.
func (self *Image) Image() *ebiten.Image {
	return self.atlas.native.SubImage(*self.bounds).(*ebiten.Image)
}

// DrawImage draws the given source image onto this atlas sub-image.
// It correctly translates the drawing options to ensure the source image
// is rendered within the sub-image's bounds on the main atlas texture.
func (self *Image) DrawImage(src *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if opts == nil {
		opts = &ebiten.DrawImageOptions{}
	}
	// Get the target sub-image from the atlas.
	dst := self.Image()
	// Store the original GeoM to preserve it.
	geom := opts.GeoM
	// Translate the geometry matrix to the top-left corner of this sub-image's bounds.
	opts.GeoM.Translate(
		float64(dst.Bounds().Min.X),
		float64(dst.Bounds().Min.Y),
	)
	// Apply the original transformation.
	opts.GeoM.Concat(geom)
	// Draw the source image to the destination (the sub-image region on the atlas).
	dst.DrawImage(src, opts)
}

// SubImage returns a new *atlas.Image that represents a region
// within the current image. This is useful for creating sprites from a
// smaller section of a larger sprite within the atlas.
func (self *Image) SubImage(bounds image.Rectangle) *Image {
	// Adjust the bounds of the new sub-image to be relative to the parent atlas.
	bounds = bounds.Add(self.bounds.Min)

	// Return a new Image struct representing this nested region.
	return &Image{
		atlas:  self.atlas,
		bounds: &bounds,
	}
}
