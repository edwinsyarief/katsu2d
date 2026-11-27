package katsu2d

import (
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// LayerOption is a function type for configuring LayerSystem
// using the functional options pattern.
type LayerOption func(*LayerSystem)

// AddSystem creates an option to add an update and/or draw system to the layer.
func AddSystem(sys any) LayerOption {
	return func(ls *LayerSystem) {
		if us, ok := sys.(UpdateSystem); ok {
			ls.updateSystems = append(ls.updateSystems, us)
		}
		if ds, ok := sys.(DrawSystem); ok {
			ls.drawSystems = append(ls.drawSystems, ds)
		}
	}
}

// Stretched configures whether the layer stretches to fill the target screen.
func Stretched(stretch bool) LayerOption {
	return func(ls *LayerSystem) {
		ls.stretched = stretch
	}
}

// PixelPerfect configures whether the layer snaps scaling to integer multiples.
func PixelPerfect(pixelPerfect bool) LayerOption {
	return func(ls *LayerSystem) {
		ls.pixelPerfect = pixelPerfect
	}
}

// IgnoreCamera configures whether the layer ignores the active camera's view matrix.
func IgnoreCamera(ignore bool) LayerOption {
	return func(ls *LayerSystem) {
		ls.ignoreCamera = ignore
	}
}

// LayerSystem manages the rendering of multiple drawing systems
// onto a single buffer layer. It handles scaling and positioning of the final output.
type LayerSystem struct {
	batchRenderer                                      *BatchRenderer // Handles batch rendering operations
	buffer                                             *ebiten.Image  // Off-screen buffer for compositing
	canvas                                             *canvas
	layoutCh                                           teishoku.EventChannel[EngineLayoutChangedEvent]
	drawSystems                                        []DrawSystem   // Collection of drawing systems to be executed
	updateSystems                                      []UpdateSystem // Collection of update systems to be executed
	width, height                                      int
	screenWidth, screenHeight                          int
	stretched, pixelPerfect, ignoreCamera, initialized bool
}

// NewLayerSystem creates a new layer renderer with specified dimensions
// and optional configuration through LayerOptions.
func NewLayerSystem(width, height int, opts ...LayerOption) *LayerSystem {
	buffer := ebiten.NewImage(width, height)
	ls := &LayerSystem{
		batchRenderer: NewBatchRenderer(),
		buffer:        buffer,
		drawSystems:   make([]DrawSystem, 0),
		width:         width,
		height:        height,
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(ls)
	}

	ls.canvas = newCanvas(width, height, ls.stretched, ls.pixelPerfect)

	return ls
}

// SetFilter configures the scaling filter applied when presenting the layer buffer.
func (s *LayerSystem) SetFilter(filter ScalingFilter) *LayerSystem {
	s.canvas.SetFilter(filter)
	return s
}

// Buffer returns the off-screen buffer image used by this layer.
func (s *LayerSystem) Buffer() *ebiten.Image {
	return s.buffer
}

// ScreenToCanvas converts screen-space coordinates (x, y) into layer canvas coordinates.
func (s *LayerSystem) ScreenToCanvas(x, y float64) Vector {
	if s.width <= 0 || s.height <= 0 || s.screenWidth <= 0 || s.screenHeight <= 0 {
		return V(x, y)
	}
	if s.stretched {
		scaleX := float64(s.screenWidth) / float64(s.width)
		scaleY := float64(s.screenHeight) / float64(s.height)
		return V(x/scaleX, y/scaleY)
	}

	scaleX := float64(s.screenWidth) / float64(s.width)
	scaleY := float64(s.screenHeight) / float64(s.height)
	scale := math.Min(scaleX, scaleY)
	if s.pixelPerfect && scale >= 1.0 {
		scale = math.Floor(scale)
	}
	if scale <= 0 {
		return V(x, y)
	}
	drawW := float64(s.width) * scale
	drawH := float64(s.height) * scale
	offsetX := (float64(s.screenWidth) - drawW) * 0.5
	offsetY := (float64(s.screenHeight) - drawH) * 0.5
	return V((x-offsetX)/scale, (y-offsetY)/scale)
}

// ScaleCursorPosition returns the current Ebitengine cursor position mapped into canvas coordinates.
func (s *LayerSystem) ScaleCursorPosition() Vector {
	cx, cy := ebiten.CursorPosition()
	return s.ScreenToCanvas(float64(cx), float64(cy))
}

// Dispose deallocates the layer's off-screen buffer image.
func (s *LayerSystem) Dispose() {
	if s.buffer != nil {
		s.buffer.Deallocate()
		s.buffer = nil
	}
}

// Initialize initializes all child systems, syncs initial display size if present,
// and subscribes to layout change events.
func (s *LayerSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}

	for _, us := range s.updateSystems {
		us.Initialize(w)
	}
	for _, ds := range s.drawSystems {
		ds.Initialize(w)
	}

	if info := GetHiResDisplayInfo(w); info != nil && info.Width > 0 && info.Height > 0 {
		s.screenWidth = info.Width
		s.screenHeight = info.Height
		s.canvas.Resize(info.Width, info.Height)
	}

	s.layoutCh = teishoku.NewEventChannel[EngineLayoutChangedEvent](getEventBus(w))
	s.layoutCh.Subscribe(s.onEngineLayoutChanged)

	s.initialized = true
}

func (s *LayerSystem) onEngineLayoutChanged(data EngineLayoutChangedEvent) {
	s.screenWidth = data.Width
	s.screenHeight = data.Height
	s.canvas.Resize(data.Width, data.Height)
}

// Update advances all registered child update systems.
func (s *LayerSystem) Update(w *teishoku.World, dt float64) {
	for _, us := range s.updateSystems {
		us.Update(w, dt)
	}
}

// Draw executes all registered render systems and handles scaling of the final output.
// It maintains aspect ratio while scaling and centers the result on the screen.
func (s *LayerSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if s.buffer == nil {
		return
	}

	// Clear the buffer with transparency
	s.buffer.Fill(color.Transparent)

	// Begin batch rendering to the buffer
	s.batchRenderer.Begin(s.buffer)

	if !s.ignoreCamera {
		if cam := GetActiveCamera(w); cam != nil {
			s.batchRenderer.SetViewMatrix(cam.ViewMatrix)
		}
	}

	// Execute all registered drawing systems
	for _, ds := range s.drawSystems {
		ds.Draw(w, s.batchRenderer)
	}

	// Ensure all batched operations are executed
	s.batchRenderer.Flush()
	rdr.Flush()

	s.canvas.Draw(s.buffer, rdr.screen)
}
