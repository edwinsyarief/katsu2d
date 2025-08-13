package katsu2d

import (
	"embed"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- ENGINE ---

// Engine manages the game loop, global state, and scene transitions.
type Engine struct {
	// Managers
	world    *World
	tm       *TextureManager
	fm       *FontManager
	am       *AudioManager
	sm       *SceneManager
	renderer *BatchRenderer

	// Engine-level systems
	updateSystems         []UpdateSystem
	backgroundDrawSystems []DrawSystem // New: For backgrounds and pre-scene rendering
	overlayDrawSystems    []DrawSystem // New: For UI and post-scene rendering

	// Game settings
	timeScale            float64
	windowWidth          int
	windowHeight         int
	windowTitle          string
	windowResizeMode     ebiten.WindowResizingModeType
	fullScreen           bool
	vsync                bool
	clearScreenEachFrame bool
	cursorMode           ebiten.CursorModeType
}

// Option is a functional option for configuring the engine.
type Option func(*Engine)

// WithWindowSize sets the window dimensions.
func WithWindowSize(w, h int) Option {
	return func(e *Engine) {
		e.windowWidth = w
		e.windowHeight = h
	}
}

// WithWindowTitle sets the window title.
func WithWindowTitle(title string) Option {
	return func(e *Engine) {
		e.windowTitle = title
	}
}

// WithWindowResizeMode sets the window resizing mode.
func WithWindowResizeMode(mode ebiten.WindowResizingModeType) Option {
	return func(e *Engine) {
		e.windowResizeMode = mode
	}
}

// WithFullScreen sets fullscreen mode.
func WithFullScreen(full bool) Option {
	return func(e *Engine) {
		e.fullScreen = full
	}
}

// WithVsyncEnabled sets vsync.
func WithVsyncEnabled(vsync bool) Option {
	return func(e *Engine) {
		e.vsync = vsync
	}
}

// WithCursorMode sets the cursor mode.
func WithCursorMode(mode ebiten.CursorModeType) Option {
	return func(e *Engine) {
		e.cursorMode = mode
	}
}

// WithClearScreenEachFrame sets whether to clear the screen each frame.
func WithClearScreenEachFrame(clear bool) Option {
	return func(e *Engine) {
		e.clearScreenEachFrame = clear
	}
}

// WithBackgroundDrawSystem adds a DrawSystem that renders before the scene.
func WithBackgroundDrawSystem(sys DrawSystem) Option {
	return func(e *Engine) {
		e.backgroundDrawSystems = append(e.backgroundDrawSystems, sys)
	}
}

// WithOverlayDrawSystem adds a DrawSystem that renders after the scene (on top).
func WithOverlayDrawSystem(sys DrawSystem) Option {
	return func(e *Engine) {
		e.overlayDrawSystems = append(e.overlayDrawSystems, sys)
	}
}

// NewEngine creates a new engine with options.
func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		world:    NewWorld(),
		tm:       NewTextureManager(),
		fm:       NewFontManager(),
		am:       NewAudioManager(44100),
		sm:       NewSceneManager(),
		renderer: NewBatchRenderer(),
		// We can add global update systems here, such as input handlers.
		updateSystems: make([]UpdateSystem, 0),
		timeScale:     1.0,
		windowWidth:   800,
		windowHeight:  600,
		windowTitle:   "Game",
		// ... default settings
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (self *Engine) InitFS(fs embed.FS) {
	initFS(fs)
}

func (self *Engine) InitAssetReader(path string, key []byte) {
	initAssetReader(path, key)
}

func (self *Engine) World() *World {
	return self.world
}

func (self *Engine) TextureManager() *TextureManager {
	return self.tm
}

func (self *Engine) AudioManager() *AudioManager {
	return self.am
}

func (self *Engine) SceneManager() *SceneManager {
	return self.sm
}

func (self *Engine) FontManager() *FontManager {
	return self.fm
}

// SwitchScene switches to a named scene.
func (self *Engine) SwitchScene(name string) {
	self.sm.SwitchTo(self, name)
}

func (self *Engine) AddScene(name string, scene *Scene) {
	self.sm.AddScene(name, scene)
}

// SetTimeScale adjusts the game speed.
func (self *Engine) SetTimeScale(ts float64) {
	self.timeScale = ts
}

// Update implements ebiten.Game.Update.
func (self *Engine) Update() error {
	dt := (1.0 / 60.0) * self.timeScale

	// 1. Update the engine's global systems first.
	// These could be input handlers, etc.
	for _, us := range self.updateSystems {
		us.Update(self, dt)
	}

	// 2. Then, update the active scene's systems.
	if self.sm.current != nil {
		self.sm.current.Update(self, dt)
	}

	// 3. Process deferred removals for the engine's world.
	self.World().processRemovals()

	return nil
}

// Draw implements ebiten.Game.Draw. This method orchestrates the entire rendering pipeline.
func (self *Engine) Draw(screen *ebiten.Image) {
	if self.clearScreenEachFrame {
		screen.Fill(color.Transparent)
	}

	// Begin the batch renderer cycle. All drawing will be batched from this point.
	self.renderer.Begin(screen)

	// 1. Draw the engine's background systems (bottom-most layer).
	for _, ds := range self.backgroundDrawSystems {
		ds.Draw(self, self.renderer)
	}

	// 2. Draw the active scene's content (the main game world).
	if self.sm.current != nil {
		self.sm.current.Draw(self, self.renderer)
	}

	// 3. Draw the engine's overlay systems (UI, HUD, FPS counter - top-most layer).
	for _, ds := range self.overlayDrawSystems {
		ds.Draw(self, self.renderer)
	}

	// 4. Flush all batched draw calls at once.
	self.renderer.Flush()
}

// Layout implements ebiten.Game.Layout.
func (self *Engine) Layout(outWidth, outHeight int) (int, int) {
	return self.windowWidth, self.windowHeight
}

// Run runs the game.
func (self *Engine) Run() error {
	ebiten.SetWindowSize(self.windowWidth, self.windowHeight)
	ebiten.SetWindowTitle(self.windowTitle)
	ebiten.SetWindowResizingMode(self.windowResizeMode)
	ebiten.SetFullscreen(self.fullScreen)
	ebiten.SetVsyncEnabled(self.vsync)
	ebiten.SetCursorMode(self.cursorMode)
	ebiten.SetScreenClearedEveryFrame(self.clearScreenEachFrame)
	return ebiten.RunGame(self)
}
