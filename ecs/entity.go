package ecs

import (
	"sync/atomic"

	"katsu2d/constants"
)

// Signature is a bitmask used to represent a set of components.
// Each bit corresponds to a unique ComponentType ID.
type Signature uint64

// Set sets the bit for a given component type ID in the signature.
func (s *Signature) Set(typeID int) {
	*s |= (1 << typeID)
}

// Unset unsets the bit for a given component type ID in the signature.
func (s *Signature) Unset(typeID int) {
	*s &^= (1 << typeID)
}

// Has checks if the bit for a given component type ID is set.
func (s *Signature) Has(typeID int) bool {
	return *s&(1<<typeID) != 0
}

// EntityID is the unique identifier for an entity.
type EntityID uint32

var nextEntityID uint32 = 1

// NewEntityID generates a new unique entity ID.
func NewEntityID() EntityID {
	return EntityID(atomic.AddUint32(&nextEntityID, 1))
}

// IsValid checks if the entity ID is valid.
func (self EntityID) IsValid() bool {
	return self != constants.InvalidEntityID
}

// EntityManager manages the lifecycle of entities and their components.
type EntityManager struct {
	entities     map[EntityID]bool
	components   map[EntityID]map[int]Component
	signatures   map[EntityID]Signature // New bitmask signatures map for performance
	deadEntities []EntityID
}

// NewEntityManager creates a new EntityManager instance.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities:     make(map[EntityID]bool),
		components:   make(map[EntityID]map[int]Component),
		signatures:   make(map[EntityID]Signature),
		deadEntities: make([]EntityID, 0),
	}
}

// CreateEntity generates and registers a new entity.
func (self *EntityManager) CreateEntity() EntityID {
	id := NewEntityID()
	self.entities[id] = true
	self.components[id] = make(map[int]Component)
	self.signatures[id] = 0 // Initialize signature to zero
	return id
}

// DestroyEntity marks an entity for removal.
func (self *EntityManager) DestroyEntity(id EntityID) {
	if _, exists := self.entities[id]; exists {
		delete(self.entities, id)
		delete(self.components, id)
		delete(self.signatures, id) // Also remove the signature
		self.deadEntities = append(self.deadEntities, id)
	}
}

// AddComponent adds a component to an entity.
func (self *EntityManager) AddComponent(entityID EntityID, component Component) {
	if _, exists := self.entities[entityID]; exists {
		self.components[entityID][component.GetTypeID()] = component
		// Update the entity's signature
		signature := self.signatures[entityID]
		signature.Set(component.GetTypeID())
		self.signatures[entityID] = signature
	}
}

// RemoveComponent removes a component from an entity.
func (self *EntityManager) RemoveComponent(entityID EntityID, typeID int) {
	if components, exists := self.components[entityID]; exists {
		delete(components, typeID)
		// Update the entity's signature
		signature := self.signatures[entityID]
		signature.Unset(typeID)
		self.signatures[entityID] = signature
	}
}

// GetComponent retrieves a component from an entity.
func (self *EntityManager) GetComponent(entityID EntityID, typeID int) (Component, bool) {
	if components, exists := self.components[entityID]; exists {
		component, exists := components[typeID]
		return component, exists
	}
	return nil, false
}

// HasComponent checks if an entity has a specific component.
func (self *EntityManager) HasComponent(entityID EntityID, typeID int) bool {
	if signature, exists := self.signatures[entityID]; exists {
		return signature.Has(typeID)
	}
	return false
}

// GetEntitiesWithComponents finds all entities that have all specified component types.
// This refactored implementation uses bitmasks for a high-performance query.
func (self *EntityManager) GetEntitiesWithComponents(typeIDs ...int) []EntityID {
	var result []EntityID
	var targetSignature Signature

	// Build the target signature from the requested component IDs
	for _, typeID := range typeIDs {
		targetSignature.Set(typeID)
	}

	// Iterate through entity signatures and use a bitwise check
	for entityID, signature := range self.signatures {
		if signature&targetSignature == targetSignature {
			result = append(result, entityID)
		}
	}

	return result
}

// CleanupDeadEntities clears the list of entities to be destroyed.
func (self *EntityManager) CleanupDeadEntities() {
	self.deadEntities = self.deadEntities[:0]
}
