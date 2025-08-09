package ecs

import (
	"iter"
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
	// A new slice to store entity IDs in the order they were created.
	// This is the key to preserving a stable iteration order.
	entityOrder []EntityID

	entities     map[EntityID]bool
	components   map[EntityID]map[int]Component
	signatures   map[EntityID]Signature
	deadEntities []EntityID
}

// NewEntityManager creates a new EntityManager instance.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entityOrder:  make([]EntityID, 0),
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

	// Add the new entity ID to our ordered slice.
	self.entityOrder = append(self.entityOrder, id)

	return id
}

// DestroyEntity marks an entity for removal.
func (self *EntityManager) DestroyEntity(id EntityID) {
	if _, exists := self.entities[id]; exists {
		delete(self.entities, id)
		delete(self.components, id)
		delete(self.signatures, id) // Also remove the signature
		self.deadEntities = append(self.deadEntities, id)

		// Remove the entity from our ordered slice to preserve consistency.
		for i, entityID := range self.entityOrder {
			if entityID == id {
				self.entityOrder = append(self.entityOrder[:i], self.entityOrder[i+1:]...)
				break
			}
		}
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

// GetEntitiesWithComponents finds all entities that have all specified component types,
// returning an iter.Seq for memory efficiency.
func (self *EntityManager) GetEntitiesWithComponents(typeIDs ...int) iter.Seq[EntityID] {
	var targetSignature Signature
	for _, typeID := range typeIDs {
		targetSignature.Set(typeID)
	}

	return func(yield func(EntityID) bool) {
		for _, entityID := range self.entityOrder {
			if signature, exists := self.signatures[entityID]; exists {
				if signature&targetSignature == targetSignature {
					if !yield(entityID) {
						return
					}
				}
			}
		}
	}
}

// GetEntitiesWithAnyComponent finds all entities that have at least one of the
// specified component types, returning an iter.Seq.
func (self *EntityManager) GetEntitiesWithAnyComponent(typeIDs ...int) iter.Seq[EntityID] {
	var targetSignature Signature
	for _, typeID := range typeIDs {
		targetSignature.Set(typeID)
	}

	return func(yield func(EntityID) bool) {
		for _, entityID := range self.entityOrder {
			if signature, exists := self.signatures[entityID]; exists {
				if signature&targetSignature != 0 {
					if !yield(entityID) {
						return
					}
				}
			}
		}
	}
}

// CleanupDeadEntities clears the list of entities to be destroyed.
func (self *EntityManager) CleanupDeadEntities() {
	self.deadEntities = self.deadEntities[:0]
}
