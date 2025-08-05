package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// World is the core of the ECS, holding all entities and systems.
type World struct {
	entityManager *EntityManager
	systemManager *SystemManager
	registry      *ComponentRegistry
}

// NewWorld creates a new World instance.
func NewWorld() *World {
	return &World{
		entityManager: NewEntityManager(),
		systemManager: NewSystemManager(),
		registry:      NewComponentRegistry(),
	}
}

// CreateEntity creates and returns a new entity ID.
func (self *World) CreateEntity() EntityID {
	return self.entityManager.CreateEntity()
}

// DestroyEntity marks an entity for destruction.
func (self *World) DestroyEntity(entityID EntityID) {
	self.entityManager.DestroyEntity(entityID)
}

// AddComponent adds a component to an entity.
func (self *World) AddComponent(entityID EntityID, component Component) {
	self.entityManager.AddComponent(entityID, component)
}

// RemoveComponent removes a component from an entity.
func (self *World) RemoveComponent(entityID EntityID, typeID int) {
	self.entityManager.RemoveComponent(entityID, typeID)
}

// GetComponent retrieves a component from an entity.
func (self *World) GetComponent(entityID EntityID, typeID int) (Component, bool) {
	return self.entityManager.GetComponent(entityID, typeID)
}

// HasComponent checks if an entity has a specific component.
func (self *World) HasComponent(entityID EntityID, typeID int) bool {
	return self.entityManager.HasComponent(entityID, typeID)
}

// GetEntitiesWithComponents finds all entities that have a set of components.
func (self *World) GetEntitiesWithComponents(typeIDs ...int) []EntityID {
	return self.entityManager.GetEntitiesWithComponents(typeIDs...)
}

// AddSystem adds a system to the world.
func (self *World) AddSystem(system System) {
	self.systemManager.AddSystem(system)
}

// RemoveSystem removes a system from the world.
func (self *World) RemoveSystem(system System) {
	self.systemManager.RemoveSystem(system)
}

// Update runs the update logic for all systems in the world.
func (self *World) Update(timeScale float64) error {
	err := self.systemManager.Update(self, timeScale)
	self.entityManager.CleanupDeadEntities()
	return err
}

// Draw runs the draw logic for all systems in the world.
func (self *World) Draw(screen *ebiten.Image) {
	self.systemManager.Draw(self, screen)
}

// Registry returns the component registry.
func (self *World) Registry() *ComponentRegistry {
	return self.registry
}
