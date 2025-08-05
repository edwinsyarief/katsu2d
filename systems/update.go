package systems

import (
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"

	"github.com/hajimehoshi/ebiten/v2"
)

// UpdateSystem processes all updatable components.
type UpdateSystem struct {
	priority int
}

// NewUpdateSystem creates a new UpdateSystem instance.
func NewUpdateSystem() *UpdateSystem {
	return &UpdateSystem{
		priority: 1,
	}
}

// Update runs the update function for each updatable entity.
func (self *UpdateSystem) Update(world *ecs.World, timeScale float64) error {
	// Get all entities with Updatable component
	entities := world.GetEntitiesWithComponents(constants.ComponentUpdatable)
	for _, entityID := range entities {
		if component, exists := world.GetComponent(entityID, constants.ComponentUpdatable); exists {
			if updatable, ok := component.(*components.Updatable); ok && updatable.UpdateFunc != nil {
				// We now use the timeScale to control the speed of updates.
				updatable.UpdateFunc((1.0/60.0)*timeScale, world, entityID)
			}
		}
	}
	return nil
}

// Draw is not implemented for the update system.
func (self *UpdateSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Update system doesn't draw
}

// GetPriority returns the system's priority.
func (self *UpdateSystem) GetPriority() int {
	return self.priority
}
