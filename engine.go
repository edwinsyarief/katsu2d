package katsu2d

import (
	"image/color"
	"log"

	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"
	"katsu2d/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	layoutWidth  = 1280
	layoutHeight = 720
)

// game is the main Ebitengine game struct.
type game struct {
	engine *Engine
}

// Update handles game logic updates.
func (self *game) Update() error {
	return self.engine.Update()
}

// Draw handles rendering to the screen.
func (self *game) Draw(screen *ebiten.Image) {
	if !ebiten.IsScreenClearedEveryFrame() {
		screen.Fill(color.RGBA{0, 0, 0, 255}) // Clear the screen with black if not cleared every frame
	}

	self.engine.Draw(screen)
}

// Layout sets the game screen size.
func (self *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return layoutWidth, layoutHeight
}

// Option is a function type for configuring GrassSystem.
type Option func(*Engine)

// WithTileSize sets the size of the grid cells for grass placement.
func WithWindowSize(width, height int) Option {
	return func(_ *Engine) {
		ebiten.SetWindowSize(width, height)

		layoutWidth = width
		layoutHeight = height
	}
}

// WithWindowResizingMode sets the resizing mode for the game window.
func WithWindowsResizeMode(mode ebiten.WindowResizingModeType) Option {
	return func(_ *Engine) {
		ebiten.SetWindowResizingMode(mode)
	}
}

// WithWindowSize sets the size of the game window.
func WithWindowTitle(title string) Option {
	return func(_ *Engine) {
		ebiten.SetWindowTitle(title)
	}
}

// WithWindowFullscreen enables fullscreen mode for the game window.
func WithWindowFullscreen(fullscreen bool) Option {
	return func(_ *Engine) {
		ebiten.SetFullscreen(fullscreen)
	}
}

// WithWindowVsync enables or disables VSync for the game window.
func WithWindowVsync(vsync bool) Option {
	return func(_ *Engine) {
		ebiten.SetVsyncEnabled(vsync)
	}
}

// WithScreenClearedEveryFrame sets whether the screen should be cleared every frame.
func WithScreenClearedEveryFrame(clear bool) Option {
	return func(_ *Engine) {
		ebiten.SetScreenClearedEveryFrame(clear)
	}
}

// WithCursorMode sets the cursor mode for the game window.
func WithCursorMode(mode ebiten.CursorModeType) Option {
	return func(_ *Engine) {
		ebiten.SetCursorMode(mode)
	}
}

// Engine is the main game interface, providing a simplified API over the ECS.
type Engine struct {
	world           *ecs.World
	sceneManager    *scene.Manager
	timeScale       float64
	targetTimeScale float64
	lerpSpeed       float64

	game *game // Reference to the Ebitengine game struct
}

// NewEngine creates a new Engine instance.
func NewEngine(opts ...Option) *Engine {
	// Initialize the underlying ECS world and scene manager
	world := ecs.NewWorld()
	sceneManager := scene.NewManager()

	engine := &Engine{
		world:           world,
		sceneManager:    sceneManager,
		timeScale:       1.0,
		targetTimeScale: 1.0,
		lerpSpeed:       0.1,
	}

	for _, opt := range opts {
		opt(engine)
	}

	engine.game = &game{engine: engine}

	return engine
}

// RunGame starts the game loop using Ebitengine.
func (self *Engine) RunGame() error {
	return ebiten.RunGame(self.game)
}

// World returns the ECS World.
func (self *Engine) World() *ecs.World {
	return self.world
}

func (self *Engine) NewWorld() *ecs.World {
	self.world = ecs.NewWorld()
	return self.world
}

// SceneManager returns the SceneManager.
func (self *Engine) SceneManager() *scene.Manager {
	return self.sceneManager
}

// AddSystem adds a system to the game.
func (self *Engine) AddSystem(system ecs.System) {
	self.world.AddSystem(system)
}

// AddEntity creates a new entity and adds the provided components.
func (self *Engine) AddEntity(comps ...ecs.Component) ecs.EntityID {
	entityID := self.world.CreateEntity()
	for _, comp := range comps {
		self.world.AddComponent(entityID, comp)
	}
	return entityID
}

// Update runs the game logic for the current frame.
func (self *Engine) Update() error {
	// Lerp the time scale smoothly towards the target
	self.timeScale = self.timeScale + (self.targetTimeScale-self.timeScale)*self.lerpSpeed

	// Update current scene
	if err := self.sceneManager.Update(); err != nil {
		return err
	}

	// Update ECS systems
	return self.world.Update(self.timeScale)
}

// Draw renders the game state to the screen.
func (self *Engine) Draw(screen *ebiten.Image) {
	// Draw current scene
	self.sceneManager.Draw(screen)

	// Draw ECS systems
	self.world.Draw(screen)
}

// SetTimeScale sets the target time scale for the game.
func (self *Engine) SetTimeScale(scale float64) {
	self.targetTimeScale = scale
}

// ToggleSlowMotion switches between normal speed and a slow motion effect.
func (self *Engine) ToggleSlowMotion() {
	if self.targetTimeScale == 1.0 {
		self.targetTimeScale = 0.2 // Slow motion
		log.Println("Slow motion enabled.")
	} else {
		self.targetTimeScale = 1.0 // Normal speed
		log.Println("Slow motion disabled.")
	}
}

// FindEntityByTag finds an entity by its tag.
func (self *Engine) FindEntityByTag(tag string) (ecs.EntityID, bool) {
	for entityID := range self.world.GetEntitiesWithComponents(constants.ComponentTag) {
		if comp, exists := self.world.GetComponent(entityID, constants.ComponentTag); exists {
			if tagComp, ok := comp.(*components.Tag); ok && tagComp.Name == tag {
				return entityID, true
			}
		}
	}
	return constants.InvalidEntityID, false
}

// GetComponent functions to easily get a component
func (self *Engine) GetTransform(entityID ecs.EntityID) (*components.Transform, bool) {
	if comp, exists := self.world.GetComponent(entityID, constants.ComponentTransform); exists {
		if transform, ok := comp.(*components.Transform); ok {
			return transform, true
		}
	}
	return nil, false
}

// GetCooldown retrieves the cooldown component for an entity.
func (self *Engine) GetCooldown(entityID ecs.EntityID) (*components.Cooldown, bool) {
	if comp, exists := self.world.GetComponent(entityID, constants.ComponentCooldown); exists {
		if cooldown, ok := comp.(*components.Cooldown); ok {
			return cooldown, true
		}
	}
	return nil, false
}
