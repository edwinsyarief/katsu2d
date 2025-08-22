package katsu2d

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LayerRendererOption is a function type for configuring LayerRendererSystem
// using the functional options pattern
type LayerRendererOption func(*LayerRendererSytem)

// AddDrawSystem creates an option to add a drawing system to the layer renderer.
// This allows for modular addition of different drawing systems.
func AddDrawSystem(ds DrawSystem) LayerRendererOption {
	return func(ls *LayerRendererSytem) {
		ls.drawSystems = append(ls.drawSystems, ds)
	}
}

// LayerRendererSytem manages the rendering of multiple drawing systems
// onto a single buffer layer. It handles scaling and positioning of the final output.
type LayerRendererSytem struct {
	batchRenderer *BatchRenderer // Handles batch rendering operations
	buffer        *ebiten.Image  // Off-screen buffer for compositing
	drawSystems   []DrawSystem   // Collection of drawing systems to be executed
}

// NewLayerRendererSystem creates a new layer renderer with specified dimensions
// and optional configuration through LayerRendererOptions
func NewLayerRendererSystem(width, height int, opts ...LayerRendererOption) *LayerRendererSytem {
	buffer := ebiten.NewImage(width, height)
	ls := &LayerRendererSytem{
		batchRenderer: NewBatchRenderer(),
		buffer:        buffer,
		drawSystems:   make([]DrawSystem, 0),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(ls)
	}

	return ls
}

// Draw executes all registered render systems and handles scaling of the final output.
// It maintains aspect ratio while scaling and centers the result on the screen.
func (self *LayerRendererSytem) Draw(world *World, renderer *BatchRenderer) {
	// Clear the buffer with transparency
	self.buffer.Fill(color.Transparent)

	// Begin batch rendering to the buffer
	self.batchRenderer.Begin(self.buffer, nil)

	// Execute all registered drawing systems
	for _, ds := range self.drawSystems {
		ds.Draw(world, self.batchRenderer)
	}

	// Ensure all batched operations are executed
	self.batchRenderer.Flush()
	renderer.Flush()

	// Calculate dimensions for source and destination
	srcWidth := float64(self.batchRenderer.screen.Bounds().Dx())
	srcHeight := float64(self.batchRenderer.screen.Bounds().Dy())
	dstWidth := float64(renderer.screen.Bounds().Dx())
	dstHeight := float64(renderer.screen.Bounds().Dy())

	// Calculate scale factors while preserving aspect ratio
	scaleX := dstWidth / srcWidth
	scaleY := dstHeight / srcHeight

	// Choose the smaller scale to maintain aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Set up transformation for the final draw operation
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)

	// Calculate position to center the scaled image
	scaledWidth := srcWidth * scale
	scaledHeight := srcHeight * scale
	x := (dstWidth - scaledWidth) / 2
	y := (dstHeight - scaledHeight) / 2
	op.GeoM.Translate(x, y)

	// Draw the final scaled and positioned buffer to the screen
	renderer.screen.DrawImage(self.buffer, op)
}
