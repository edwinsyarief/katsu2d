package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a self-contained game scene with its own ECS World,
// systems, and lifecycle hooks.
type Scene struct {
	world         *teishoku.World
	layoutEventCh teishoku.EventChannel[EngineLayoutChangedEvent]
	OnEnter       func(*Engine)
	OnExit        func(*Engine)
	OnUpdate      func(float64)
	OnBeforeDraw  func(*ebiten.Image)
	OnAfterDraw   func(*ebiten.Image)
	UpdateSystems []UpdateSystem
	DrawSystems   []DrawSystem
	Width, Height int
}

// NewScene creates a new Scene with the default initial entity capacity.
func NewScene() *Scene {
	return NewSceneWithInitialCapacity(defaultCapacity)
}

// NewSceneWithInitialCapacity creates a new Scene with a dedicated World of the given capacity.
func NewSceneWithInitialCapacity(capacity int) *Scene {
	w := teishoku.NewWorld(capacity)
	return &Scene{
		world:         w,
		layoutEventCh: teishoku.NewEventChannel[EngineLayoutChangedEvent](getEventBus(w)),
	}
}

// World returns the scene's dedicated ECS world.
func (s *Scene) World() *teishoku.World {
	return s.world
}

// AddSystem adds an UpdateSystem and/or DrawSystem to the scene.
func (s *Scene) AddSystem(sys any) {
	if sys == nil {
		return
	}
	if us, ok := sys.(UpdateSystem); ok {
		s.AddUpdateSystem(us)
	}
	if ds, ok := sys.(DrawSystem); ok {
		s.AddDrawSystem(ds)
	}
}

// AddUpdateSystem appends an UpdateSystem to the scene.
func (s *Scene) AddUpdateSystem(us UpdateSystem) {
	if us == nil {
		return
	}
	s.UpdateSystems = append(s.UpdateSystems, us)
}

// AddDrawSystem appends a DrawSystem to the scene.
func (s *Scene) AddDrawSystem(ds DrawSystem) {
	if ds == nil {
		return
	}
	s.DrawSystems = append(s.DrawSystems, ds)
}

// ClearSystems removes all registered update and draw systems from the scene.
func (s *Scene) ClearSystems() {
	s.UpdateSystems = s.UpdateSystems[:0]
	s.DrawSystems = s.DrawSystems[:0]
}

// Update runs all the scene's update systems.
func (s *Scene) Update(dt float64) {
	for _, us := range s.UpdateSystems {
		us.Update(s.world, dt)
	}
}

// OnLayoutChanged updates the scene dimensions and publishes an EngineLayoutChangedEvent.
func (s *Scene) OnLayoutChanged(width, height int) {
	s.Width = width
	s.Height = height
	s.layoutEventCh.Publish(EngineLayoutChangedEvent{
		Width:  width,
		Height: height,
	})
}

// Draw runs all the scene's draw systems using the shared BatchRenderer.
func (s *Scene) Draw(world *teishoku.World, renderer *BatchRenderer) {
	for _, ds := range s.DrawSystems {
		ds.Draw(world, renderer)
	}
}

// SceneManager manages named scenes and active scene transitions.
type SceneManager struct {
	engine      *Engine
	scenes      map[string]*Scene
	current     *Scene
	transitions *sceneTransitionController
}

// NewSceneManager creates a new SceneManager bound to an Engine.
func NewSceneManager(e *Engine) *SceneManager {
	return &SceneManager{
		engine:      e,
		scenes:      make(map[string]*Scene),
		transitions: newSceneTransitionController(),
	}
}

// AddScene registers a scene by name.
func (sm *SceneManager) AddScene(name string, scene *Scene) {
	sm.scenes[name] = scene
}

// GetScene retrieves a registered scene by name.
func (sm *SceneManager) GetScene(name string) *Scene {
	return sm.scenes[name]
}

// CurrentScene returns the currently active scene, or nil if none is active.
func (sm *SceneManager) CurrentScene() *Scene {
	return sm.current
}

// SwitchTo transitions immediately to a named scene, cancelling any active transition,
// invoking OnExit on the previous scene, injecting engine resources, invoking OnEnter,
// and initializing the new scene's systems.
func (sm *SceneManager) SwitchTo(name string) {
	if _, ok := sm.scenes[name]; !ok {
		return
	}
	if sm.transitions != nil && sm.transitions.isTransitioning() {
		sm.CancelTransition()
	}
	sm.switchToInternal(name)
}

func (sm *SceneManager) switchToInternal(name string) {
	newScene, ok := sm.scenes[name]
	if !ok {
		return
	}
	if sm.current != nil && sm.current.OnExit != nil {
		sm.current.OnExit(sm.engine)
	}
	sm.current = newScene
	initializeAssetManagers(
		sm.current.World(),
		sm.engine.TextureManager(),
		sm.engine.FontManager(),
		sm.engine.AudioManager(),
		sm.engine.ShaderManager(),
		sm,
	)
	w, h := sm.engine.HiResSize()
	updateHiResDisplayResource(sm.current.World(), w, h)
	if sm.current.OnEnter != nil {
		sm.current.OnEnter(sm.engine)
	}
	for _, us := range sm.current.UpdateSystems {
		us.Initialize(sm.current.World())
	}
	for _, ds := range sm.current.DrawSystems {
		ds.Initialize(sm.current.World())
	}
	if w > 0 && h > 0 {
		sm.current.OnLayoutChanged(w, h)
	}
}

// TransitionTo starts a configured scene transition to the named scene.
// Returns false if a transition is already active or the target scene is not registered.
func (sm *SceneManager) TransitionTo(name string, cfg TransitionConfig) bool {
	if sm.transitions == nil {
		sm.transitions = newSceneTransitionController()
	}
	started := sm.transitions.start(sm, name, cfg)
	if started {
		sm.syncInputBlockState()
	}
	return started
}

// IsTransitioning reports whether a scene transition is currently in progress.
func (sm *SceneManager) IsTransitioning() bool {
	return sm.transitions != nil && sm.transitions.isTransitioning()
}

// CancelTransition aborts any active scene transition without firing completion callbacks
// and releases any transition input block.
func (sm *SceneManager) CancelTransition() {
	if sm.transitions == nil {
		return
	}
	sm.transitions.cancel()
	sm.syncInputBlockState()
}

// Dispose releases any offscreen buffers allocated for scene transitions.
func (sm *SceneManager) Dispose() {
	if sm.transitions != nil {
		sm.transitions.dispose()
	}
}

func (sm *SceneManager) updateTransition(rawDt, scaledDt float64) {
	if sm.transitions == nil || !sm.transitions.isTransitioning() {
		return
	}
	dt := scaledDt
	if sm.transitions.usesUnscaledTime() {
		dt = rawDt
	}
	sm.transitions.update(sm, dt)
}

func (sm *SceneManager) isTransitionInputBlocked() bool {
	return sm.transitions != nil && sm.transitions.isInputBlocked()
}

func (sm *SceneManager) syncInputBlockState() {
	blocked := sm.isTransitionInputBlocked()
	if sm.engine != nil {
		setWorldInputBlocked(sm.engine.world, blocked)
	}
	if sm.current != nil {
		setWorldInputBlocked(sm.current.World(), blocked)
	}
}
