package katsu2d

import (
	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a game scene. It is a self-contained unit
// with its own World, systems, and lifecycle hooks.
type Scene struct {
	World         *lazyecs.World
	Width, Height int
	UpdateSystems []UpdateSystem
	DrawSystems   []DrawSystem
	OnEnter       func(*Engine)
	OnExit        func(*Engine)
	OnUpdate      func(float64)
	OnBeforeDraw  func(*ebiten.Image)
	OnAfterDraw   func(*ebiten.Image)
}

// NewScene creates a new scene with its own dedicated World.
func NewScene() *Scene {
	scn := &Scene{
		World: lazyecs.NewWorld(),
	}
	_ = GetEventBus(scn.World)

	return scn
}

// AddSystem adds an update and/or draw system to the scene.
func (self *Scene) AddSystem(sys any) {
	if us, ok := sys.(UpdateSystem); ok {
		self.AddUpdateSystem(us)
	}
	if ds, ok := sys.(DrawSystem); ok {
		self.AddDrawSystem(ds)
	}
}

func (self *Scene) AddUpdateSystem(us UpdateSystem) {
	self.UpdateSystems = append(self.UpdateSystems, us)
}

func (self *Scene) AddDrawSystem(ds DrawSystem) {
	self.DrawSystems = append(self.DrawSystems, ds)
}

// Update runs all the scene's update systems.
func (self *Scene) Update(world *lazyecs.World, dt float64) {
	// Then, run the scene's own update systems.
	for _, us := range self.UpdateSystems {
		us.Update(world, dt)
	}
	// Process event bus
	ProcessEventBus(self.World)

	// First, process any deferred entity removals for this scene's world.
	self.World.ProcessRemovals()
}

// OnLayoutChanged publishes an engine layout change event
func (self *Scene) OnLayoutChanged(width, height int) {
	self.Width = width
	self.Height = height

	eb := GetEventBus(self.World)
	eb.Publish(EngineLayoutChangedEvent{
		Width:  width,
		Height: height,
	})
}

// Draw runs all the scene's draw systems using the engine's shared renderer.
func (self *Scene) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	for _, ds := range self.DrawSystems {
		ds.Draw(world, renderer)
	}
}

// SceneManager manages scenes and scene transitions.
type SceneManager struct {
	scenes  map[string]*Scene
	current *Scene
}

// NewSceneManager creates a new scene manager.
func NewSceneManager() *SceneManager {
	return &SceneManager{scenes: make(map[string]*Scene)}
}

// AddScene adds a scene by name.
func (self *SceneManager) AddScene(name string, scene *Scene) {
	self.scenes[name] = scene
}

// GetScene retrieves a scene by name.
func (self *SceneManager) GetScene(name string) *Scene {
	return self.scenes[name]
}

// CurrentScene returns the currently active scene.
func (self *SceneManager) CurrentScene() *Scene {
	return self.current
}

// SwitchTo switches to a new scene, running its OnEnter hook and the old scene's OnExit hook.
// This is the only place where the active scene is changed.
func (self *SceneManager) SwitchTo(e *Engine, name string) {
	newScene, ok := self.scenes[name]
	if !ok {
		return
	}
	if self.current != nil && self.current.OnExit != nil {
		self.current.OnExit(e)
	}
	self.current = newScene
	if self.current.OnEnter != nil {
		self.current.OnEnter(e)
		w, h := e.HiResSize()
		self.current.OnLayoutChanged(w, h)
	}
}
