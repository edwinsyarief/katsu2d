package katsu2d

import ebimath "github.com/edwinsyarief/ebi-math"

// TransformComponent extends the basic ebimath Transform with a Z-coordinate
// for managing 2D depth/layering. It handles position, rotation, and scale in 2D space.
type TransformComponent struct {
	*ebimath.Transform         // Embedded transform for basic 2D transformations
	Z                  float64 // Z-coordinate for depth sorting
}

func (self *TransformComponent) Init() *TransformComponent {
	self.Transform = ebimath.T()
	return self
}
