package atlas

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Image struct {
	atlas *Atlas

	bounds *image.Rectangle
}

func (i *Image) Atlas() *Atlas {
	return i.atlas
}

func (i *Image) Bounds() image.Rectangle {
	return *i.bounds
}

func (i *Image) Image() *ebiten.Image {
	return i.atlas.native.SubImage(*i.bounds).(*ebiten.Image)
}

func (i *Image) DrawImage(src *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if opts == nil {
		opts = &ebiten.DrawImageOptions{}
	}
	dst := i.Image()
	geom := opts.GeoM
	opts.GeoM.Translate(
		float64(dst.Bounds().Min.X),
		float64(dst.Bounds().Min.Y),
	)
	opts.GeoM.Concat(geom)
	dst.DrawImage(src, opts)
}

func (i *Image) SubImage(bounds image.Rectangle) *Image {
	bounds = bounds.Add(i.bounds.Min)

	return &Image{
		atlas: i.atlas,

		bounds: &bounds,
	}
}
