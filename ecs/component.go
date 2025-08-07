package ecs

// Component is the base interface for all components.
// All components in the engine must implement this interface.
type Component interface {
	GetTypeID() int
}

// ComponentID is a unique identifier for a component type.
// It's a small integer that can be used as an index or part of a bitmask.
// The constants.ComponentType enum (in constants/constants.go) maps component names
// to these unique IDs.
type ComponentID uint16

// ComponentMask is a bitmask used to represent a set of components.
// Each bit corresponds to a unique ComponentID.
// This allows for efficient storage and querying of which components an entity has.
type ComponentMask uint64
