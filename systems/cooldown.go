package systems

import (
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"

	"github.com/hajimehoshi/ebiten/v2"
)

// CooldownSystem processes cooldown timers for all entities with a Cooldown component.
type CooldownSystem struct {
	priority int
}

// NewCooldownSystem creates a new CooldownSystem instance.
func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{
		priority: 2,
	}
}

// Update updates all cooldown timers.
func (self *CooldownSystem) Update(world *ecs.World, timeScale float64) error {
	// Get all entities with Cooldown components
	for entityID := range world.GetEntitiesWithComponents(constants.ComponentCooldown) {
		if component, exists := world.GetComponent(entityID, constants.ComponentCooldown); exists {
			if cooldown, ok := component.(*components.Cooldown); ok {
				if cooldown.Active {
					cooldown.Remaining -= (1.0 / 60.0) * timeScale
					if cooldown.Remaining <= 0 {
						cooldown.Active = false
					}
				}
			}
		}
	}
	return nil
}

// Draw is not implemented for the cooldown system.
func (self *CooldownSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Cooldown system doesn't draw
}

// GetPriority returns the system's priority.
func (self *CooldownSystem) GetPriority() int {
	return self.priority
}
