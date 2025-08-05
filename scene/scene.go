package scene

import "github.com/hajimehoshi/ebiten/v2"

// Scene is the interface for a game scene.
type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
	Load() error
	Unload() error
}

// DefaultScene is a basic implementation of a Scene.
type DefaultScene struct{}

// NewDefaultScene creates a new DefaultScene.
func NewDefaultScene() *DefaultScene {
	return &DefaultScene{}
}

// Update runs the scene's update logic.
func (self *DefaultScene) Update() error {
	// Default scene does not have any specific update logic
	return nil
}

// Draw renders the scene.
func (self *DefaultScene) Draw(screen *ebiten.Image) {
	// Default scene does not have any specific draw logic
}

// Load is called when the scene is loaded.
func (self *DefaultScene) Load() error {
	return nil
}

// Unload is called when the scene is unloaded.
func (self *DefaultScene) Unload() error {
	return nil
}

// Manager handles switching and managing game scenes.
type Manager struct {
	scenes       map[string]Scene
	currentScene Scene
}

// NewManager creates a new Manager instance.
func NewManager() *Manager {
	return &Manager{
		scenes: make(map[string]Scene),
	}
}

// AddScene adds a new scene to the manager.
func (self *Manager) AddScene(name string, s Scene) {
	self.scenes[name] = s
}

// SwitchScene switches to a new scene.
func (self *Manager) SwitchScene(name string) {
	if self.currentScene != nil {
		err := self.currentScene.Unload()
		if err != nil {
			panic(err)
		}
	}
	newScene, ok := self.scenes[name]
	if !ok {
		return
	}
	err := newScene.Load()
	if err != nil {
		panic(err)
	}
	self.currentScene = newScene
}

// Update calls the update method of the current scene.
func (self *Manager) Update() error {
	if self.currentScene != nil {
		return self.currentScene.Update()
	}
	return nil
}

// Draw calls the draw method of the current scene.
func (self *Manager) Draw(screen *ebiten.Image) {
	if self.currentScene != nil {
		self.currentScene.Draw(screen)
	}
}
