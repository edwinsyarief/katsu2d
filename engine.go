package katsu2d

import (
	"embed"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/ark/ecs"
)

type Engine struct {
	clearColor color.Color
	// Managers
	world       ecs.World
	tm          *TextureManager
	fm          *FontManager
	am          *AudioManager
	shm         *ShaderManager
	scm         *SceneManager
	renderer    *BatchRenderer
	windowTitle string
	// Engine-level systems
	updateSystems         []UpdateSystem
	backgroundDrawSystems []DrawSystem
	overlayDrawSystems    []DrawSystem
	// Game settings
	timeScale            float64
	windowWidth          int
	windowHeight         int
	windowResizeMode     ebiten.WindowResizingModeType
	cursorMode           ebiten.CursorModeType
	atlasWidth           int
	atlasHeight          int
	hiResWidth           int
	hiResHeight          int
	fullScreen           bool
	vsync                bool
	clearScreenEachFrame bool
	// Atlas settings
	useAtlas bool
	// Layout properties
	layoutHasChanged bool
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

// WithTextureAtlas enables or disables the texture atlas system.
func WithTextureAtlas(enabled bool) Option {
	return func(e *Engine) {
		e.useAtlas = enabled
	}
}

// WithTextureAtlasSize sets the dimensions for the internal texture atlases.
// This implicitly enables the atlas system.
func WithTextureAtlasSize(w, h int) Option {
	return func(e *Engine) {
		e.useAtlas = true
		e.atlasWidth = w
		e.atlasHeight = h
	}
}

// WithBackgroundSystem adds a DrawSystem that renders before the scene.
func WithBackgroundSystem(sys any) Option {
	return func(e *Engine) {
		e.AddBackgroundSystem(sys)
	}
}

// WithOverlaySystem adds a DrawSystem that renders after the scene (on top).
func WithOverlaySystem(sys any) Option {
	return func(e *Engine) {
		e.AddOverlaySystem(sys)
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
	return NewEngineWithInitialCapacity(defaultCapacity, opts...)
}
func NewEngineWithInitialCapacity(cap int, opts ...Option) *Engine {
	e := &Engine{
		world:    ecs.NewWorld(cap),
		fm:       NewFontManager(),
		am:       NewAudioManager(44100),
		renderer: NewBatchRenderer(),
		// We can add global update systems here, such as input handlers.
		updateSystems:         make([]UpdateSystem, 0),
		backgroundDrawSystems: make([]DrawSystem, 0),
		overlayDrawSystems:    make([]DrawSystem, 0),
		timeScale:             1.0,
		windowWidth:           800,
		windowHeight:          600,
		windowTitle:           "Game",
		useAtlas:              false, // Default atlas usage
		atlasWidth:            2048,  // Default atlas size
		atlasHeight:           2048,
		// ... default settings
	}
	// Apply all the functional options. This might override the defaults.
	for _, opt := range opts {
		opt(e)
	}
	// Configure and initialize the texture manager.
	var tmOpts []TextureManagerOption
	if e.useAtlas {
		tmOpts = append(tmOpts, WithAtlas(true))
		tmOpts = append(tmOpts, WithAtlasSize(e.atlasWidth, e.atlasHeight))
	}
	e.scm = NewSceneManager(e)
	e.tm = NewTextureManager(tmOpts...)

	initializeAssetManagers(e.World(),
		e.TextureManager(),
		e.FontManager(),
		e.AudioManager(),
		e.ShaderManager(),
		e.SceneManager())

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
func (self *Engine) World() *ecs.World {
	return &self.world
}

// TextureManager returns the engine's texture manager.
func (self *Engine) TextureManager() *TextureManager {
	return self.tm
}

// AudioManager returns the engine's audio manager.
func (self *Engine) AudioManager() *AudioManager {
	return self.am
}

func (self *Engine) ShaderManager() *ShaderManager {
	return self.shm
}

// SceneManager returns the engine's scene manager.
func (self *Engine) SceneManager() *SceneManager {
	return self.scm
}

// FontManager returns the engine's font manager.
func (self *Engine) FontManager() *FontManager {
	return self.fm
}

// SwitchScene switches to a named scene.
func (self *Engine) SwitchScene(name string) {
	self.scm.SwitchTo(name)
}

// AddScene adds a scene to the scene manager by name.
func (self *Engine) AddScene(name string, scene *Scene) {
	self.scm.AddScene(name, scene)
}

// AddUpdateSystem adds an update system to the engine's global update systems.
func (self *Engine) AddUpdateSystem(sys UpdateSystem) {
	self.updateSystems = append(self.updateSystems, sys)
}

// AddBackgroundSystem adds a system that updates and/or draws before the scene.
func (self *Engine) AddBackgroundSystem(sys any) {
	if us, ok := sys.(UpdateSystem); ok {
		self.updateSystems = append(self.updateSystems, us)
	}
	if ds, ok := sys.(DrawSystem); ok {
		self.backgroundDrawSystems = append(self.backgroundDrawSystems, ds)
	}
}

// AddOverlaySystem adds a system that updates and/or draws after the scene.
func (self *Engine) AddOverlaySystem(sys any) {
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
	if self.layoutHasChanged {
		updateHiResDisplayResource(self.World(), self.hiResWidth, self.hiResHeight)
		Publish(self.World(), EngineLayoutChangedEvent{
			Width:  self.hiResWidth,
			Height: self.hiResHeight,
		})
	}
	// Update the engine's global systems first.
	for _, us := range self.updateSystems {
		us.Update(self.World(), dt)
	}

	// Then, update the active scene's systems.
	if self.scm.current != nil {
		if self.layoutHasChanged {
			self.layoutHasChanged = false
			updateHiResDisplayResource(self.scm.current.World(), self.hiResWidth, self.hiResHeight)
			self.scm.current.OnLayoutChanged(self.hiResWidth, self.hiResHeight)
		}
		self.scm.current.Update(dt)
		if self.scm.current.OnUpdate != nil {
			self.scm.current.OnUpdate(dt)
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
		ds.Draw(self.World(), self.renderer)
	}
	// Draw the active scene's content (the main game world).
	if self.scm.current != nil {
		if self.scm.current.OnBeforeDraw != nil {
			self.scm.current.OnBeforeDraw(screen)
		}
		self.scm.current.Draw(self.scm.current.World(), self.renderer)
		if self.scm.current.OnAfterDraw != nil {
			self.scm.current.OnAfterDraw(screen)
		}
	}
	// Draw the engine's overlay systems (UI, HUD, FPS counter - top-most layer).
	for _, ds := range self.overlayDrawSystems {
		ds.Draw(self.World(), self.renderer)
	}
	self.renderer.Flush()
}

// Layout implements ebiten.Game.Layout.
func (self *Engine) Layout(logicWinWidth, logicWinHeight int) (int, int) {
	monitor := ebiten.Monitor()
	scale := monitor.DeviceScaleFactor()
	hiResWidth := int(float64(logicWinWidth) * scale)
	hiResHeight := int(float64(logicWinHeight) * scale)
	if hiResWidth != self.hiResWidth || hiResHeight != self.hiResHeight {
		self.layoutHasChanged = true
		self.hiResWidth, self.hiResHeight = hiResWidth, hiResHeight
	}
	return self.hiResWidth, self.hiResHeight
}

// LayoufF implements ebiten.Game.LayoutF.
func (self *Engine) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	monitor := ebiten.Monitor()
	scale := monitor.DeviceScaleFactor()
	outWidth := math.Ceil(logicWinWidth * scale)
	outHeight := math.Ceil(logicWinHeight * scale)
	if int(outWidth) != self.hiResWidth || int(outHeight) != self.hiResHeight {
		self.layoutHasChanged = true
		self.hiResWidth, self.hiResHeight = int(outWidth), int(outHeight)
	}
	return outWidth, outHeight
}

// HiResSize returns hiResWidth, hiResHeight
func (self *Engine) HiResSize() (int, int) {
	return self.hiResWidth, self.hiResHeight
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
	for _, us := range self.updateSystems {
		us.Initialize(self.World())
	}
	for _, bs := range self.backgroundDrawSystems {
		bs.Initialize(self.World())
	}
	for _, os := range self.overlayDrawSystems {
		os.Initialize(self.World())
	}
	return ebiten.RunGame(self)
}
