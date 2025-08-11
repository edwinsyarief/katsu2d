package katsu2d

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene defines the interface for game scenes.
type Scene interface {
	OnEnter(engine *Engine)    // Called when entering the scene (load/setup)
	Update(dt float64)         // Update logic (delegates to systems)
	Draw(screen *ebiten.Image) // Draw logic (delegates to systems)
	OnExit()                   // Called when exiting the scene (cleanup)
}

// SceneManager manages scene switching.
type SceneManager struct {
	current Scene // Current active scene
	next    Scene // Next scene to switch to
}

// Update handles scene switching and updates the current scene.
func (sm *SceneManager) Update(dt float64, engine *Engine) {
	if sm.next != nil {
		if sm.current != nil {
			sm.current.OnExit()
		}
		sm.current = sm.next
		sm.next = nil
		sm.current.OnEnter(engine)
	}
	if sm.current != nil {
		sm.current.Update(dt)
	}
}

// Draw draws the current scene.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	if sm.current != nil {
		sm.current.Draw(screen)
	}
}

// SetScene queues a scene switch for the next update.
func (sm *SceneManager) SetScene(s Scene) {
	sm.next = s
}
