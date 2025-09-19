package katsu2d

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LayerOption is a function type for configuring LayerRendererSystem
// using the functional options pattern
type LayerOption func(*LayerSytem)

// AddDrawSystem creates an option to add a drawing system to the layer renderer.
// This allows for modular addition of different drawing systems.
func AddSystem(sys any) LayerOption {
	return func(ls *LayerSytem) {
		if us, ok := sys.(UpdateSystem); ok {
			ls.updateSystems = append(ls.updateSystems, us)
		}
		if ds, ok := sys.(DrawSystem); ok {
			ls.drawSystems = append(ls.drawSystems, ds)
		}
	}
}

func Stretched(stretch bool) LayerOption {
	return func(ls *LayerSytem) {
		ls.stretched = stretch
	}
}

func PixelPerfect(pixelPerfect bool) LayerOption {
	return func(ls *LayerSytem) {
		ls.pixelPerfect = pixelPerfect
	}
}

// LayerSytem manages the rendering of multiple drawing systems
// onto a single buffer layer. It handles scaling and positioning of the final output.
type LayerSytem struct {
	batchRenderer           *BatchRenderer // Handles batch rendering operations
	buffer                  *ebiten.Image  // Off-screen buffer for compositing
	canvas                  *canvas
	drawSystems             []DrawSystem   // Collection of drawing systems to be executed
	updateSystems           []UpdateSystem // Collection of update systems to be executed
	stretched, pixelPerfect bool
}

// NewLayerSystem creates a new layer renderer with specified dimensions
// and optional configuration through LayerRendererOptions
func NewLayerSystem(world *World, width, height int, opts ...LayerOption) *LayerSytem {
	buffer := ebiten.NewImage(width, height)
	ls := &LayerSytem{
		batchRenderer: NewBatchRenderer(),
		buffer:        buffer,
		drawSystems:   make([]DrawSystem, 0),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(ls)
	}

	ls.canvas = newCanvas(width, height, ls.stretched, ls.pixelPerfect)

	eb := world.GetEventBus()
	eb.Subscribe(EngineLayoutChangedEvent{}, ls.onEngineLayoutChanged)

	return ls
}

func (self *LayerSytem) onEngineLayoutChanged(obj interface{}) {
	data := obj.(EngineLayoutChangedEvent)
	self.canvas.Resize(data.Width, data.Height)
}

func (self *LayerSytem) Update(world *World, dt float64) {
	for _, us := range self.updateSystems {
		us.Update(world, dt)
	}
}

// Draw executes all registered render systems and handles scaling of the final output.
// It maintains aspect ratio while scaling and centers the result on the screen.
func (self *LayerSytem) Draw(world *World, renderer *BatchRenderer) {
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

	self.canvas.Draw(self.buffer, renderer.screen)
}
