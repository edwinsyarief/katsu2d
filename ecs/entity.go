package ecs

import (
	"sync/atomic"

	"katsu2d/constants"
)

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
	deadEntities []EntityID
}

// NewEntityManager creates a new EntityManager instance.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities:     make(map[EntityID]bool),
		components:   make(map[EntityID]map[int]Component),
		deadEntities: make([]EntityID, 0),
	}
}

// CreateEntity generates and registers a new entity.
func (self *EntityManager) CreateEntity() EntityID {
	id := NewEntityID()
	self.entities[id] = true
	self.components[id] = make(map[int]Component)
	return id
}

// DestroyEntity marks an entity for removal.
func (self *EntityManager) DestroyEntity(id EntityID) {
	if _, exists := self.entities[id]; exists {
		delete(self.entities, id)
		delete(self.components, id)
		self.deadEntities = append(self.deadEntities, id)
	}
}

// AddComponent adds a component to an entity.
func (self *EntityManager) AddComponent(entityID EntityID, component Component) {
	if _, exists := self.entities[entityID]; exists {
		self.components[entityID][component.GetTypeID()] = component
	}
}

// RemoveComponent removes a component from an entity.
func (self *EntityManager) RemoveComponent(entityID EntityID, typeID int) {
	if components, exists := self.components[entityID]; exists {
		delete(components, typeID)
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
	if components, exists := self.components[entityID]; exists {
		_, exists := components[typeID]
		return exists
	}
	return false
}

// GetEntitiesWithComponents finds all entities that have all specified component types.
// This is a naive implementation; for high performance with a large number of entities,
// a component bitmask approach is recommended.
func (self *EntityManager) GetEntitiesWithComponents(typeIDs ...int) []EntityID {
	var result []EntityID

	for entityID := range self.entities {
		hasAll := true
		for _, typeID := range typeIDs {
			if !self.HasComponent(entityID, typeID) {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, entityID)
		}
	}

	return result
}

// CleanupDeadEntities clears the list of entities to be destroyed.
func (self *EntityManager) CleanupDeadEntities() {
	self.deadEntities = self.deadEntities[:0]
}
