package katsu2d

import (
	"embed"
	"image/color"
	"math"

	_ "github.com/silbinarywolf/preferdiscretegpu"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// Engine orchestrates the game loop, asset managers, scene transitions,
// global ECS world, and batched rendering pipeline.
type Engine struct {
	clearColor color.Color

	// Core managers
	world    *teishoku.World
	tm       *TextureManager
	fm       *FontManager
	am       *AudioManager
	shm      *ShaderManager
	scm      *SceneManager
	renderer *BatchRenderer

	// Cached event channel for layout changes on the engine world
	layoutEventCh teishoku.EventChannel[EngineLayoutChangedEvent]

	// Engine-level systems
	updateSystems         []UpdateSystem
	backgroundDrawSystems []DrawSystem
	overlayDrawSystems    []DrawSystem

	// Game settings
	windowTitle          string
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
	useAtlas             bool

	// Lifecycle state
	layoutHasChanged bool
	running          bool
}

// Option is a functional option for configuring the Engine.
type Option func(*Engine)

// WithWindowSize sets the initial window dimensions.
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

// WithVsyncEnabled enables or disables vertical synchronization.
func WithVsyncEnabled(vsync bool) Option {
	return func(e *Engine) {
		e.vsync = vsync
	}
}

// WithCursorMode sets the cursor visibility/capture mode.
func WithCursorMode(mode ebiten.CursorModeType) Option {
	return func(e *Engine) {
		e.cursorMode = mode
	}
}

// WithClearScreenEachFrame sets whether Ebitengine clears the screen each frame.
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

// WithTextureAtlas enables or disables the internal texture atlas system.
func WithTextureAtlas(enabled bool) Option {
	return func(e *Engine) {
		e.useAtlas = enabled
	}
}

// WithTextureAtlasSize sets the dimensions for internal texture atlases
// and implicitly enables the atlas system.
func WithTextureAtlasSize(w, h int) Option {
	return func(e *Engine) {
		e.useAtlas = true
		e.atlasWidth = w
		e.atlasHeight = h
	}
}

// WithBackgroundSystem adds a system that updates and/or draws before the active scene.
func WithBackgroundSystem(sys any) Option {
	return func(e *Engine) {
		e.AddBackgroundSystem(sys)
	}
}

// WithOverlaySystem adds a system that updates and/or draws after the active scene (on top).
func WithOverlaySystem(sys any) Option {
	return func(e *Engine) {
		e.AddOverlaySystem(sys)
	}
}

// WithUpdateSystem adds a global UpdateSystem executed before the active scene updates.
func WithUpdateSystem(sys UpdateSystem) Option {
	return func(e *Engine) {
		e.AddUpdateSystem(sys)
	}
}

// NewEngine creates a new Engine with the default entity capacity and given options.
func NewEngine(opts ...Option) *Engine {
	return NewEngineWithInitialCapacity(defaultCapacity, opts...)
}

// NewEngineWithInitialCapacity creates a new Engine with a specific initial entity capacity.
func NewEngineWithInitialCapacity(capacity int, opts ...Option) *Engine {
	e := &Engine{
		world:                 teishoku.NewWorld(capacity),
		fm:                    NewFontManager(),
		am:                    NewAudioManager(44100),
		shm:                   NewShaderManager(),
		renderer:              NewBatchRenderer(),
		updateSystems:         make([]UpdateSystem, 0),
		backgroundDrawSystems: make([]DrawSystem, 0),
		overlayDrawSystems:    make([]DrawSystem, 0),
		timeScale:             1.0,
		windowWidth:           800,
		windowHeight:          600,
		windowTitle:           "Game",
		vsync:                 true,
		clearScreenEachFrame:  true,
		useAtlas:              false,
		atlasWidth:            2048,
		atlasHeight:           2048,
	}

	for _, opt := range opts {
		opt(e)
	}

	var tmOpts []TextureManagerOption
	if e.useAtlas {
		tmOpts = append(tmOpts, WithAtlas(true), WithAtlasSize(e.atlasWidth, e.atlasHeight))
	}
	e.tm = NewTextureManager(tmOpts...)
	e.scm = NewSceneManager(e)

	initializeAssetManagers(
		e.world,
		e.tm,
		e.fm,
		e.am,
		e.shm,
		e.scm,
	)
	e.layoutEventCh = teishoku.NewEventChannel[EngineLayoutChangedEvent](getEventBus(e.world))

	return e
}

// InitFS initializes the embedded asset filesystem.
func (e *Engine) InitFS(fs embed.FS) {
	initFS(fs)
}

// InitAssetReader initializes the encrypted asset bundle reader.
func (e *Engine) InitAssetReader(path string, key []byte) error {
	return initAssetReader(path, key)
}

// World returns the engine's global ECS world.
func (e *Engine) World() *teishoku.World {
	return e.world
}

// TextureManager returns the engine's texture manager.
func (e *Engine) TextureManager() *TextureManager {
	return e.tm
}

// FontManager returns the engine's font manager.
func (e *Engine) FontManager() *FontManager {
	return e.fm
}

// AudioManager returns the engine's audio manager.
func (e *Engine) AudioManager() *AudioManager {
	return e.am
}

// ShaderManager returns the engine's shader manager.
func (e *Engine) ShaderManager() *ShaderManager {
	return e.shm
}

// SceneManager returns the engine's scene manager.
func (e *Engine) SceneManager() *SceneManager {
	return e.scm
}

// Renderer returns the engine's shared BatchRenderer.
func (e *Engine) Renderer() *BatchRenderer {
	return e.renderer
}

// SwitchScene switches to the named scene.
func (e *Engine) SwitchScene(name string) {
	e.scm.SwitchTo(name)
}

// TransitionTo starts a configured scene transition to the named scene.
// Returns false if a transition is already active or the target scene is not registered.
func (e *Engine) TransitionTo(name string, cfg TransitionConfig) bool {
	return e.scm.TransitionTo(name, cfg)
}

// IsTransitioning reports whether a scene transition is currently in progress.
func (e *Engine) IsTransitioning() bool {
	return e.scm.IsTransitioning()
}

// CancelTransition aborts any active scene transition without firing completion callbacks.
func (e *Engine) CancelTransition() {
	e.scm.CancelTransition()
}

// Dispose releases engine resources held by scene transitions.
func (e *Engine) Dispose() {
	if e.scm != nil {
		e.scm.Dispose()
	}
}

// AddScene registers a scene with the scene manager by name.
func (e *Engine) AddScene(name string, scene *Scene) {
	e.scm.AddScene(name, scene)
}

// AddUpdateSystem adds a global update system.
func (e *Engine) AddUpdateSystem(sys UpdateSystem) {
	if sys == nil {
		return
	}
	e.updateSystems = append(e.updateSystems, sys)
	if e.running {
		sys.Initialize(e.world)
	}
}

// AddBackgroundSystem adds a system that updates and/or draws before the active scene.
func (e *Engine) AddBackgroundSystem(sys any) {
	if sys == nil {
		return
	}
	if us, ok := sys.(UpdateSystem); ok {
		e.updateSystems = append(e.updateSystems, us)
		if e.running {
			us.Initialize(e.world)
		}
	}
	if ds, ok := sys.(DrawSystem); ok {
		e.backgroundDrawSystems = append(e.backgroundDrawSystems, ds)
		if e.running {
			ds.Initialize(e.world)
		}
	}
}

// AddOverlaySystem adds a system that updates and/or draws after the active scene.
func (e *Engine) AddOverlaySystem(sys any) {
	if sys == nil {
		return
	}
	if us, ok := sys.(UpdateSystem); ok {
		e.updateSystems = append(e.updateSystems, us)
		if e.running {
			us.Initialize(e.world)
		}
	}
	if ds, ok := sys.(DrawSystem); ok {
		e.overlayDrawSystems = append(e.overlayDrawSystems, ds)
		if e.running {
			ds.Initialize(e.world)
		}
	}
}

// SetTimeScale adjusts the game simulation speed multiplier.
func (e *Engine) SetTimeScale(ts float64) {
	e.timeScale = ts
}

// TimeScale returns the current simulation speed multiplier.
func (e *Engine) TimeScale() float64 {
	return e.timeScale
}

// Update implements ebiten.Game.Update.
func (e *Engine) Update() error {
	const rawDt = 1.0 / 60.0
	dt := rawDt * e.timeScale

	if e.layoutHasChanged {
		e.layoutHasChanged = false
		updateHiResDisplayResource(e.world, e.hiResWidth, e.hiResHeight)
		e.layoutEventCh.Publish(EngineLayoutChangedEvent{
			Width:  e.hiResWidth,
			Height: e.hiResHeight,
		})
		if e.scm.current != nil {
			updateHiResDisplayResource(e.scm.current.World(), e.hiResWidth, e.hiResHeight)
			e.scm.current.OnLayoutChanged(e.hiResWidth, e.hiResHeight)
		}
	}

	e.scm.updateTransition(rawDt, dt)
	e.scm.syncInputBlockState()

	for _, us := range e.updateSystems {
		us.Update(e.world, dt)
	}

	if cur := e.scm.current; cur != nil {
		cur.Update(dt)
		if cur.OnUpdate != nil {
			cur.OnUpdate(dt)
		}
	}

	e.am.Update(dt)
	return nil
}

// Draw implements ebiten.Game.Draw, orchestrating the full rendering pipeline.
func (e *Engine) Draw(screen *ebiten.Image) {
	tr := e.scm.transitions
	if tr == nil || !tr.isTransitioning() {
		e.drawBackgroundAndScene(screen)
		e.drawOverlays(screen)
		return
	}

	if tr.needsCapture {
		bounds := screen.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		if w > 0 && h > 0 {
			if capBuf := tr.ensureCaptureBuf(w, h); capBuf != nil {
				if e.clearColor == nil {
					tr.fillOpaqueBlack(capBuf)
				}
				e.drawBackgroundAndScene(capBuf)
				if !tr.belowOverlay() {
					e.drawOverlays(capBuf)
				}
			}
		}
		tr.needsCapture = false
		tr.executeMidpoint(e.scm)
		if cur := e.scm.current; cur != nil {
			cur.Update(0)
		}
		if !tr.isTransitioning() {
			e.drawBackgroundAndScene(screen)
			e.drawOverlays(screen)
			return
		}
	}

	e.drawBackgroundAndScene(screen)
	if tr.belowOverlay() {
		tr.draw(screen)
		e.drawOverlays(screen)
	} else {
		e.drawOverlays(screen)
		tr.draw(screen)
	}
}

func (e *Engine) drawBackgroundAndScene(target *ebiten.Image) {
	if e.clearColor != nil {
		target.Fill(e.clearColor)
	} else if !e.clearScreenEachFrame {
		target.Fill(color.Black)
	}

	e.renderer.Begin(target)

	if cam := GetActiveCamera(e.world); cam != nil {
		e.renderer.SetViewMatrix(cam.ViewMatrix)
	} else {
		e.renderer.ResetViewMatrix()
	}

	for _, ds := range e.backgroundDrawSystems {
		ds.Draw(e.world, e.renderer)
	}

	if cur := e.scm.current; cur != nil {
		if cur.OnBeforeDraw != nil {
			e.renderer.Flush()
			cur.OnBeforeDraw(target)
		}
		if cam := GetActiveCamera(cur.World()); cam != nil {
			e.renderer.SetViewMatrix(cam.ViewMatrix)
		} else {
			e.renderer.ResetViewMatrix()
		}
		cur.Draw(cur.World(), e.renderer)
		if cur.OnAfterDraw != nil {
			e.renderer.Flush()
			cur.OnAfterDraw(target)
		}
	}

	e.renderer.Flush()
}

func (e *Engine) drawOverlays(target *ebiten.Image) {
	if len(e.overlayDrawSystems) == 0 {
		return
	}
	e.renderer.Begin(target)
	e.renderer.ResetViewMatrix()
	for _, ds := range e.overlayDrawSystems {
		ds.Draw(e.world, e.renderer)
	}
	e.renderer.Flush()
}

// Layout implements ebiten.Game.Layout.
func (e *Engine) Layout(logicWinWidth, logicWinHeight int) (int, int) {
	scale := 1.0
	if monitor := ebiten.Monitor(); monitor != nil {
		if s := monitor.DeviceScaleFactor(); s > 0 {
			scale = s
		}
	}
	hiResWidth := max(1, int(float64(logicWinWidth)*scale))
	hiResHeight := max(1, int(float64(logicWinHeight)*scale))
	if hiResWidth != e.hiResWidth || hiResHeight != e.hiResHeight {
		e.layoutHasChanged = true
		e.hiResWidth, e.hiResHeight = hiResWidth, hiResHeight
	}
	return e.hiResWidth, e.hiResHeight
}

// LayoutF implements ebiten.LayoutFer for HiDPI fractional scaling.
func (e *Engine) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	scale := 1.0
	if monitor := ebiten.Monitor(); monitor != nil {
		if s := monitor.DeviceScaleFactor(); s > 0 {
			scale = s
		}
	}
	outWidth := math.Max(1, math.Ceil(logicWinWidth*scale))
	outHeight := math.Max(1, math.Ceil(logicWinHeight*scale))
	if int(outWidth) != e.hiResWidth || int(outHeight) != e.hiResHeight {
		e.layoutHasChanged = true
		e.hiResWidth, e.hiResHeight = int(outWidth), int(outHeight)
	}
	return outWidth, outHeight
}

// HiResSize returns the current high-resolution backing buffer dimensions.
func (e *Engine) HiResSize() (int, int) {
	return e.hiResWidth, e.hiResHeight
}

// InitializeSystems initializes all registered global systems.
// This is called automatically by Run(), and can also be called directly in tests.
func (e *Engine) InitializeSystems() {
	e.running = true
	for _, us := range e.updateSystems {
		us.Initialize(e.world)
	}
	for _, bs := range e.backgroundDrawSystems {
		bs.Initialize(e.world)
	}
	for _, os := range e.overlayDrawSystems {
		os.Initialize(e.world)
	}
}

// Run configures the Ebitengine window, initializes all systems, and starts the game loop.
func (e *Engine) Run() error {
	ebiten.SetWindowSize(e.windowWidth, e.windowHeight)
	ebiten.SetWindowTitle(e.windowTitle)
	ebiten.SetWindowResizingMode(e.windowResizeMode)
	ebiten.SetFullscreen(e.fullScreen)
	ebiten.SetVsyncEnabled(e.vsync)
	ebiten.SetCursorMode(e.cursorMode)
	if e.clearColor != nil {
		ebiten.SetScreenClearedEveryFrame(false)
	} else {
		ebiten.SetScreenClearedEveryFrame(e.clearScreenEachFrame)
	}
	e.InitializeSystems()
	return ebiten.RunGame(e)
}
