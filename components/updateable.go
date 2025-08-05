package components

import (
	"katsu2d/constants"
	"katsu2d/ecs"
)

// Updatable component with a function to be called every frame.
type Updatable struct {
	UpdateFunc func(float64, *ecs.World, ecs.EntityID)
}

// GetTypeID returns the component type ID.
func (self *Updatable) GetTypeID() int {
	return constants.ComponentUpdatable
}
