package atlas

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Atlas is a minimal write-only container for sub-images, that
// can be used with a DrawList to batch draw commands as triangles.
type Atlas struct {
	native    *ebiten.Image
	set       *Set
	unmanaged bool
}

type NewAtlasOptions struct {
	// MinSize is the minimum size of images on the atlas.
	// It is an optional hint to improve the allocation time
	// of new images on the atlas.
	MinSize image.Point
	// Unmanaged is the same ebitengine's image option.
	Unmanaged bool
}

func New(width, height int, opts *NewAtlasOptions) *Atlas {
	var setOpts *NewSetOptions
	var unmanaged bool
	if opts != nil {
		setOpts = &NewSetOptions{
			MinSize: opts.MinSize,
		}
		unmanaged = opts.Unmanaged
	}

	return &Atlas{
		native: ebiten.NewImageWithOptions(
			image.Rect(0, 0, width, height),
			&ebiten.NewImageOptions{
				Unmanaged: unmanaged,
			},
		),
		set: NewSet(width, height, setOpts),

		unmanaged: unmanaged,
	}
}

// Image returns the full atlas image.
func (a *Atlas) Image() *ebiten.Image {
	return a.native
}

// Bounds returns the bounds of the atlas.
func (a *Atlas) Bounds() image.Rectangle {
	return a.native.Bounds()
}

// NewImage allocates a rectangle area on the atlas and
// returns the corresponding image.
// It returns a nil if there was no space to allocate the region
// specified by width, height.
func (a *Atlas) NewImage(width, height int) *Image {
	r := image.Rect(0, 0, width, height)
	img := &Image{
		atlas:  a,
		bounds: &r,
	}
	if !a.set.Insert(img.bounds) {
		return nil
	}

	return img
}

// SubImage returns an image matching a specific region of the atlas.
// It is useful if you want to create the atlas from a spritesheet and
// that you know the regions of you sub images.
func (a *Atlas) SubImage(bounds image.Rectangle) *Image {
	return &Image{
		atlas:  a,
		bounds: &bounds,
	}
}

// Free frees a region on the atlas, making it available for next
// allocations.
func (a *Atlas) Free(img *Image) {
	img.Image().Clear()
	a.set.Free(img.bounds)
}
