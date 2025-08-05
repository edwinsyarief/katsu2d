package components

import (
	"katsu2d/constants"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// Transform is a component that holds a 2D transformation matrix.
type Transform struct {
	*ebimath.Transform
}

// GetTypeID returns the component type ID.
func (self *Transform) GetTypeID() int {
	return constants.ComponentTransform
}
