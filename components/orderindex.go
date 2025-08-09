package components

import "katsu2d/constants"

type OrderIndex struct {
	Index float64
}

// GetTypeID returns the component type ID.
func (self *OrderIndex) GetTypeID() int {
	return constants.ComponentOrderIndex
}
