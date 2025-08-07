package graphics

import (
	"image/color"
	"katsu2d/components"

	"github.com/hajimehoshi/ebiten/v2"
)

// Renderer provides a high-level interface for drawing primitives.
type Renderer struct {
	batcher *Batcher
}

// NewRenderer creates a new Renderer instance.
func NewRenderer() *Renderer {
	return &Renderer{
		batcher: NewBatcher(),
	}
}

// Begin starts a new drawing batch.
func (self *Renderer) Begin(texture *ebiten.Image) {
	self.batcher.Begin(texture)
}

// DrawQuad adds a non-textured quad to the batcher.
func (self *Renderer) DrawQuad(x, y, w, h float64, col color.RGBA) {
	self.batcher.AddQuad(x, y, w, h, 0, 0, 1, 1, float64(col.R)/255.0, float64(col.G)/255.0, float64(col.B)/255.0, float64(col.A)/255.0)
}

// DrawTransformedQuad adds a non-textured quad with rotation and scaling.
func (self *Renderer) DrawTransformedQuad(transform *components.Transform, w, h float64, col color.RGBA) {
	self.batcher.AddTransformedQuad(transform, w, h, 0, 0, 1, 1, float64(col.R)/255.0, float64(col.G)/255.0, float64(col.B)/255.0, float64(col.A)/255.0)
}

// DrawTexturedQuad adds a textured quad to the batcher.
func (self *Renderer) DrawTexturedQuad(transform *components.Transform, w, h, u1, v1, u2, v2 float64, col color.RGBA) {
	self.batcher.AddTransformedQuad(transform, w, h, u1, v1, u2, v2, float64(col.R)/255.0, float64(col.G)/255.0, float64(col.B)/255.0, float64(col.A)/255.0)
}

// End flushes the current batch to the screen.
func (self *Renderer) End(screen *ebiten.Image) {
	self.batcher.End(screen)
}

// Flush flushes the current batch without ending the batch session.
func (self *Renderer) Flush(screen *ebiten.Image) {
	self.batcher.Flush(screen)
}

// GetBatcher returns the internal batcher.
func (self *Renderer) GetBatcher() *Batcher {
	return self.batcher
}
