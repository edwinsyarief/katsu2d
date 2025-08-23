package katsu2d

import (
	"embed"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	backgroundDrawSystems []DrawSystem
	overlayDrawSystems    []DrawSystem

	// Game settings
	timeScale            float64
	windowWidth          int
	windowHeight         int
	windowTitle          string
	windowResizeMode     ebiten.WindowResizingModeType
	fullScreen           bool
	vsync                bool
	clearScreenEachFrame bool
	clearColor           color.Color
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

// WithClearColor sets a custom clear color for the screen.
func WithClearColor(col color.Color) Option {
	return func(e *Engine) {
		e.clearColor = col
	}
}

// WithBackgroundDrawSystem adds a DrawSystem that renders before the scene.
func WithBackgroundDrawSystem(sys any) Option {
	return func(e *Engine) {
		e.AddBackgroundDrawSystem(sys)
	}
}

// WithOverlayDrawSystem adds a DrawSystem that renders after the scene (on top).
func WithOverlayDrawSystem(sys any) Option {
	return func(e *Engine) {
		e.AddOverlayDrawSystem(sys)
	}
}

// WithUpdateSystem adds an UpdateSystem that update before the scene update.
func WithUpdateSystem(sys UpdateSystem) Option {
	return func(e *Engine) {
		e.updateSystems = append(e.updateSystems, sys)
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
		updateSystems:         make([]UpdateSystem, 0),
		backgroundDrawSystems: make([]DrawSystem, 0),
		overlayDrawSystems:    make([]DrawSystem, 0),
		timeScale:             1.0,
		windowWidth:           800,
		windowHeight:          600,
		windowTitle:           "Game",
		// ... default settings
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// InitFS initializes the asset filesystem using an embedded FS.
func (self *Engine) InitFS(fs embed.FS) {
	initFS(fs)
}

// InitAssetReader initializes the asset reader with a path and encryption key.
func (self *Engine) InitAssetReader(path string, key []byte) {
	initAssetReader(path, key)
}

// World returns the engine's global ECS world.
func (self *Engine) World() *World {
	return self.world
}

// TextureManager returns the engine's texture manager.
func (self *Engine) TextureManager() *TextureManager {
	return self.tm
}

// AudioManager returns the engine's audio manager.
func (self *Engine) AudioManager() *AudioManager {
	return self.am
}

// SceneManager returns the engine's scene manager.
func (self *Engine) SceneManager() *SceneManager {
	return self.sm
}

// FontManager returns the engine's font manager.
func (self *Engine) FontManager() *FontManager {
	return self.fm
}

// SwitchScene switches to a named scene.
func (self *Engine) SwitchScene(name string) {
	self.sm.SwitchTo(self, name)
}

// AddScene adds a scene to the scene manager by name.
func (self *Engine) AddScene(name string, scene *Scene) {
	self.sm.AddScene(name, scene)
}

// AddUpdateSystem adds an update system to the engine's global update systems.
func (self *Engine) AddUpdateSystem(sys UpdateSystem) {
	self.updateSystems = append(self.updateSystems, sys)
}

// AddBackgroundDrawSystem adds a system that updates and/or draws before the scene.
func (self *Engine) AddBackgroundDrawSystem(sys any) {
	if us, ok := sys.(UpdateSystem); ok {
		self.updateSystems = append(self.updateSystems, us)
	}
	if ds, ok := sys.(DrawSystem); ok {
		self.backgroundDrawSystems = append(self.backgroundDrawSystems, ds)
	}
}

// AddOverlayDrawSystem adds a system that updates and/or draws after the scene.
func (self *Engine) AddOverlayDrawSystem(sys any) {
	if us, ok := sys.(UpdateSystem); ok {
		self.updateSystems = append(self.updateSystems, us)
	}
	if ds, ok := sys.(DrawSystem); ok {
		self.overlayDrawSystems = append(self.overlayDrawSystems, ds)
	}
}

// SetTimeScale adjusts the game speed.
func (self *Engine) SetTimeScale(ts float64) {
	self.timeScale = ts
}

// Update implements ebiten.Game.Update.
func (self *Engine) Update() error {
	dt := (1.0 / 60.0) * self.timeScale

	// Process deferred removals for the engine's world.
	self.World().processRemovals()

	// Update the engine's global systems first.
	for _, us := range self.updateSystems {
		us.Update(self.world, dt)
	}

	// Then, update the active scene's systems.
	if self.sm.current != nil {
		self.sm.current.Update(self.sm.current.World, dt)

		if self.sm.current.OnUpdate != nil {
			self.sm.current.OnUpdate(dt)
		}
	}

	// Finally, update the audio manager.
	self.am.Update(dt)

	return nil
}

// Draw implements ebiten.Game.Draw. This method orchestrates the entire rendering pipeline.
func (self *Engine) Draw(screen *ebiten.Image) {
	if self.clearColor != nil || !self.clearScreenEachFrame {
		fillColor := self.clearColor
		if fillColor == nil {
			fillColor = color.Black
		}
		screen.Fill(fillColor)
	}

	self.renderer.Begin(screen, nil)

	// Draw the engine's background systems (bottom-most layer).
	for _, ds := range self.backgroundDrawSystems {
		ds.Draw(self.world, self.renderer)
	}

	// Draw the active scene's content (the main game world).
	if self.sm.current != nil {
		if self.sm.current.OnBeforeDraw != nil {
			self.sm.current.OnBeforeDraw(screen)
		}

		self.sm.current.Draw(self.sm.current.World, self.renderer)

		if self.sm.current.OnAfterDraw != nil {
			self.sm.current.OnAfterDraw(screen)
		}
	}

	// Draw the engine's overlay systems (UI, HUD, FPS counter - top-most layer).
	for _, ds := range self.overlayDrawSystems {
		ds.Draw(self.world, self.renderer)
	}

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
	if self.clearColor != nil {
		ebiten.SetScreenClearedEveryFrame(false)
	} else {
		ebiten.SetScreenClearedEveryFrame(self.clearScreenEachFrame)
	}
	return ebiten.RunGame(self)
}
