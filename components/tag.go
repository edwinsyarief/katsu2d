package components

import "katsu2d/constants"

// Tag component provides a way to identify entities by name.
type Tag struct {
	Name string
}

// GetTypeID returns the component type ID.
func (self *Tag) GetTypeID() int {
	return constants.ComponentTag
}
