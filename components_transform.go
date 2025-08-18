package katsu2d

import ebimath "github.com/edwinsyarief/ebi-math"

// TransformComponent extends the basic ebimath Transform with a Z-coordinate
// for managing 2D depth/layering. It handles position, rotation, and scale in 2D space.
type TransformComponent struct {
	*ebimath.Transform         // Embedded transform for basic 2D transformations
	Z                  float64 // Z-coordinate for depth sorting
}

// Position returns the current 2D position vector of the transform
func (self *TransformComponent) Position() ebimath.Vector {
	return self.Transform.Position()
}

// NewTransformComponent creates a new transform component with default values
// Initializes with a new ebimath Transform and Z=0
func NewTransformComponent() *TransformComponent {
	return &TransformComponent{
		Transform: ebimath.T(),
	}
}

// SetZ updates the Z-coordinate of the transform and marks the world's Z-ordering as dirty
// This ensures proper depth sorting will occur on the next update
// world: The game world context
// z: The new Z-coordinate value
func (self *TransformComponent) SetZ(world *World, z float64) {
	if self.Z != z {
		self.Z = z
		world.MarkZDirty() // Notify world that Z-ordering needs to be recalculated
	}
}
