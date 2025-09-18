// Package atlas provides a texture atlas implementation for Ebitengine.
// A texture atlas is a large image containing a collection of smaller sub-images.
// This is a common optimization technique in 2D games to reduce the number of
// draw calls sent to the GPU, improving rendering performance.
package atlas

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Atlas is a minimal write-only container for sub-images. It manages a single
// large *ebiten.Image and allocates rectangular regions within it for smaller
// images. This is useful for batching draw calls.
type Atlas struct {
	// native is the underlying Ebiten image that holds the entire atlas texture.
	native *ebiten.Image
	// set manages the allocation and deallocation of rectangular regions within the atlas.
	set *Set
	// unmanaged tracks whether the underlying native image is unmanaged.
	// See ebiten.NewImageOptions for more details on unmanaged images.
	unmanaged bool
}

// NewAtlasOptions provides optional configuration for creating a new Atlas.
type NewAtlasOptions struct {
	// MinSize is an optional hint for the minimum size of images that will be
	// allocated on the atlas. Providing this can improve the performance
	// of the allocation algorithm.
	MinSize image.Point
	// Unmanaged specifies whether the underlying atlas image should be "unmanaged".
	// Unmanaged images are not automatically cleared on screen clears and can
	// be useful for persistent textures like an atlas.
	Unmanaged bool
}

// New creates a new Atlas with the specified dimensions and options.
func New(width, height int, opts *NewAtlasOptions) *Atlas {
	var setOpts *NewSetOptions
	var unmanaged bool
	if opts != nil {
		setOpts = &NewSetOptions{
			MinSize: opts.MinSize,
		}
		unmanaged = opts.Unmanaged
	}

	// Create the main Ebiten image that will serve as the texture atlas.
	nativeImage := ebiten.NewImageWithOptions(
		image.Rect(0, 0, width, height),
		&ebiten.NewImageOptions{
			Unmanaged: unmanaged,
		},
	)

	// Initialize the Atlas struct with the native image and the allocation set.
	return &Atlas{
		native:    nativeImage,
		set:       NewSet(width, height, setOpts),
		unmanaged: unmanaged,
	}
}

// Image returns the underlying *ebiten.Image for the entire atlas.
// This is the image that should be used as the source for drawing operations.
func (self *Atlas) Image() *ebiten.Image {
	return self.native
}

// Bounds returns the dimensions of the entire atlas as an image.Rectangle.
func (self *Atlas) Bounds() image.Rectangle {
	return self.native.Bounds()
}

// NewImage finds and allocates a new rectangular area on the atlas for an
// image of the given width and height.
// It returns a new *Image representing the allocated region.
// If there is not enough free space on the atlas, it returns nil.
func (self *Atlas) NewImage(width, height int) *Image {
	// Define the rectangle for the new image.
	r := image.Rect(0, 0, width, height)
	img := &Image{
		atlas:  self,
		bounds: &r,
	}

	// Try to insert the image's bounds into the allocation set.
	if !self.set.Insert(img.bounds) {
		// If insertion fails, it means there's no space.
		return nil
	}

	// Return the new image handle on success.
	return img
}

// SubImage returns an *Image handle for a specific, predefined region of the atlas.
// This method does not perform any new allocation. It's useful when the atlas
// is created from an existing spritesheet where the locations of sub-images
// are already known.
func (self *Atlas) SubImage(bounds image.Rectangle) *Image {
	return &Image{
		atlas:  self,
		bounds: &bounds,
	}
}

// Free deallocates an image's region on the atlas, making its space
// available for future allocations. It also clears the pixels in that
// region to prevent graphical artifacts.
func (self *Atlas) Free(img *Image) {
	// Clear the image data in the specified region to avoid stale graphics.
	img.Image().Clear()
	// Mark the region as free in the allocation set.
	self.set.Free(img.bounds)
}
