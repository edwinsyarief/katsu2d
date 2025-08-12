package katsu2d

import (
	"embed"

	"github.com/edwinsyarief/katsu2d/input"
	"github.com/hajimehoshi/ebiten/v2"
	dinput "github.com/quasilyte/ebitengine-input"
)

// Engine manages the game loop, world, systems, and window settings.
type Engine struct {
	world                *World
	tm                   *TextureManager
	fm                   *FontManager
	sm                   *SceneManager
	systems              []System
	timeScale            float64
	windowWidth          int
	windowHeight         int
	windowTitle          string
	windowResizeMode     ebiten.WindowResizingModeType
	fullScreen, vsync    bool
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

// WithWindowResizeMode sets whether the window is resizeable
func WithWindowResizeMode(mode ebiten.WindowResizingModeType) Option {
	return func(e *Engine) {
		e.windowResizeMode = mode
	}
}

// WithFullScreen sets whether the window is fullscreen
func WithFullScreen(full bool) Option {
	return func(e *Engine) {
		e.fullScreen = full
	}
}

// WithVsyncEnable sets whether the vsync is enabled
func WithVsyncEnabled(vsync bool) Option {
	return func(e *Engine) {
		e.vsync = vsync
	}
}

// WithCursorMode sets the cursor mode
func WithCursorMode(mode ebiten.CursorModeType) Option {
	return func(e *Engine) {
		e.cursorMode = mode
	}
}

func WithClearScreenEachFrame(clear bool) Option {
	return func(e *Engine) {
		e.clearScreenEachFrame = clear
	}
}

// NewEngine creates a new engine with default or optional settings.
func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		world:                NewWorld(),
		tm:                   NewTextureManager(),
		fm:                   NewFontManager(),
		sm:                   NewSceneManager(),
		timeScale:            1.0,
		windowWidth:          800,
		windowHeight:         600,
		windowTitle:          "github.com/edwinsyarief/katsu2d",
		windowResizeMode:     ebiten.WindowResizingModeEnabled,
		fullScreen:           false,
		vsync:                false,
		clearScreenEachFrame: false,
		cursorMode:           ebiten.CursorModeVisible,
	}
	for _, opt := range opts {
		opt(e)
	}
	ebiten.SetWindowSize(e.windowWidth, e.windowHeight)
	ebiten.SetWindowTitle(e.windowTitle)
	ebiten.SetWindowResizingMode(e.windowResizeMode)
	ebiten.SetFullscreen(e.fullScreen)
	ebiten.SetVsyncEnabled(e.vsync)
	ebiten.SetScreenClearedEveryFrame(e.clearScreenEachFrame)
	ebiten.SetCursorMode(e.cursorMode)
	return e
}

func (self *Engine) InitFS(fs embed.FS) {
	initFS(fs)
}

func (self *Engine) InitAssetReader(path string, key []byte) {
	initAssetReader(path, key)
}

func (self *Engine) InitInput(id uint8, keymaps dinput.Keymap) {
	input.Initialize(id, keymaps)
}

// World returns the ECS world (note: scenes may use their own worlds).
func (self *Engine) World() *World {
	return self.world
}

// TextureManager returns the texture manager.
func (self *Engine) TextureManager() *TextureManager {
	return self.tm
}

// FontManager returns the font manager.
func (self *Engine) FontManager() *FontManager {
	return self.fm
}

// AddSystem add systems
func (self *Engine) AddSystem(s System) {
	self.systems = append(self.systems, s)
}

// SetScene switches to a new scene.
func (self *Engine) SetScene(s Scene) {
	self.sm.SetScene(s)
}

// SetLoadingScene switches to a loading scene with function to execute and callback when finishes
func (self *Engine) SetLoadingScene(task func() []any, callback func([]any), drawFunc func(*ebiten.Image)) {
	self.sm.SetLoadingScene(task, callback, drawFunc)
}

// SetTimeScale adjusts the game speed.
func (self *Engine) SetTimeScale(ts float64) {
	self.timeScale = ts
}

// ScreenWidth returns the screen width.
func (self *Engine) ScreenWidth() int {
	return self.windowWidth
}

// ScreenWidth returns the screen height.
func (self *Engine) ScreenHeight() int {
	return self.windowHeight
}

// Update runs scene manager update with delta time.
func (self *Engine) Update() error {
	dt := 1.0 / 60.0 * self.timeScale // Fixed timestep assumption (Ebitengine targets 60 FPS)

	input.Update()

	self.sm.Update(dt, self)
	for _, sys := range self.systems {
		sys.Update(self.world, dt)
	}
	return nil
}

// Draw all objects in scene
func (self *Engine) Draw(screen *ebiten.Image) {
	screen.Clear() // Optional: clear screen every frame

	self.sm.Draw(screen)
	for _, sys := range self.systems {
		sys.Draw(self.world, screen)
	}
}

// Layout returns window dimension
func (self *Engine) Layout(outsideWidth, outsideHeight int) (int, int) {
	return self.windowWidth, self.windowHeight
}

// Run engine
func (self *Engine) Run() error {
	return ebiten.RunGame(self)
}
