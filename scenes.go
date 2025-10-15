package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a game scene. It is a self-contained unit
// with its own World, systems, and lifecycle hooks.
type Scene struct {
	world         teishoku.World
	OnEnter       func(*Engine)
	OnExit        func(*Engine)
	OnUpdate      func(float64)
	OnBeforeDraw  func(*ebiten.Image)
	OnAfterDraw   func(*ebiten.Image)
	UpdateSystems []UpdateSystem
	DrawSystems   []DrawSystem
	Width, Height int
}

// NewScene creates a new scene with its own dedicated World.
func NewScene() *Scene {
	return NewSceneWithInitialCapacity(defaultCapacity)
}
func NewSceneWithInitialCapacity(cap int) *Scene {
	scn := &Scene{
		world: teishoku.NewWorld(cap),
	}
	return scn
}

func (self *Scene) World() *teishoku.World {
	return &self.world
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
func (self *Scene) ClearSystems() {
	self.UpdateSystems = self.UpdateSystems[:0]
	self.DrawSystems = self.DrawSystems[:0]
}

// Update runs all the scene's update systems.
func (self *Scene) Update(dt float64) {
	// Then, run the scene's own update systems.
	for _, us := range self.UpdateSystems {
		us.Update(self.World(), dt)
	}
}

// OnLayoutChanged publishes an engine layout change event
func (self *Scene) OnLayoutChanged(width, height int) {
	self.Width = width
	self.Height = height
	Publish(self.World(), EngineLayoutChangedEvent{
		Width:  width,
		Height: height,
	})
}

// Draw runs all the scene's draw systems using the engine's shared renderer.
func (self *Scene) Draw(world *teishoku.World, renderer *BatchRenderer) {
	for _, ds := range self.DrawSystems {
		ds.Draw(world, renderer)
	}
}

// SceneManager manages scenes and scene transitions.
type SceneManager struct {
	engine  *Engine
	scenes  map[string]*Scene
	current *Scene
}

// NewSceneManager creates a new scene manager.
func NewSceneManager(e *Engine) *SceneManager {
	return &SceneManager{
		engine: e,
		scenes: make(map[string]*Scene),
	}
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
func (self *SceneManager) SwitchTo(name string) {
	newScene, ok := self.scenes[name]
	if !ok {
		return
	}
	if self.current != nil && self.current.OnExit != nil {
		self.current.OnExit(self.engine)
	}
	self.current = newScene
	initializeAssetManagers(self.current.World(),
		self.engine.TextureManager(),
		self.engine.FontManager(),
		self.engine.AudioManager(),
		self.engine.ShaderManager(),
		self,
	)
	if self.current.OnEnter != nil {
		self.current.OnEnter(self.engine)
	}
	for _, us := range self.current.UpdateSystems {
		us.Initialize(self.current.World())
	}
	for _, ds := range self.current.DrawSystems {
		ds.Initialize(self.current.World())
	}
	w, h := self.engine.HiResSize()
	self.current.OnLayoutChanged(w, h)
}
