package ecs

// Component is the base interface for all components.
type Component interface {
	GetTypeID() int
}

// ComponentRegistry is a map for component types.
type ComponentRegistry struct {
	components map[int]Component
}

// NewComponentRegistry creates a new registry.
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[int]Component),
	}
}

// Register adds a new component type to the registry.
func (self *ComponentRegistry) Register(typeID int, component Component) {
	self.components[typeID] = component
}

// Get retrieves a component prototype from the registry.
func (self *ComponentRegistry) Get(typeID int) (Component, bool) {
	component, exists := self.components[typeID]
	return component, exists
}
