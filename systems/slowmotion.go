package systems

import (
	"katsu2d/ecs"

	"github.com/hajimehoshi/ebiten/v2"
)

// SlowMotionSystem is a placeholder system to demonstrate that the timeScale can be handled by
// the engine itself, and all systems will respect it.
type SlowMotionSystem struct {
	priority int
}

// NewSlowMotionSystem creates a new SlowMotionSystem.
func NewSlowMotionSystem() *SlowMotionSystem {
	return &SlowMotionSystem{
		priority: 0, // This is a very high priority system
	}
}

// Update is not implemented for the slow motion system because the time scale
// is handled by the main GameScene object.
func (self *SlowMotionSystem) Update(world *ecs.World, timeScale float64) error {
	return nil
}

// Draw is not implemented for the slow motion system.
func (self *SlowMotionSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// This system doesn't draw anything
}

// GetPriority returns the system's priority.
func (self *SlowMotionSystem) GetPriority() int {
	return self.priority
}
