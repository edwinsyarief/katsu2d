package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// System is the base interface for all systems.
type System interface {
	Update(world *World, timeScale float64) error
	Draw(world *World, screen *ebiten.Image)
	GetPriority() int
}

// SystemManager manages and runs all systems in the world.
type SystemManager struct {
	systems []System
}

// NewSystemManager creates a new SystemManager instance.
func NewSystemManager() *SystemManager {
	return &SystemManager{
		systems: make([]System, 0),
	}
}

// AddSystem adds a system to the manager and sorts it by priority.
func (self *SystemManager) AddSystem(system System) {
	self.systems = append(self.systems, system)
	// Sort by priority after adding a new system
	self.sortSystemsByPriority()
}

// RemoveSystem removes a system from the manager.
func (self *SystemManager) RemoveSystem(system System) {
	for i, s := range self.systems {
		if s == system {
			self.systems = append(self.systems[:i], self.systems[i+1:]...)
			break
		}
	}
}

// Update calls the Update method for all systems.
func (self *SystemManager) Update(world *World, timeScale float64) error {
	for _, system := range self.systems {
		if err := system.Update(world, timeScale); err != nil {
			return err
		}
	}
	return nil
}

// Draw calls the Draw method for all systems.
func (self *SystemManager) Draw(world *World, screen *ebiten.Image) {
	for _, system := range self.systems {
		system.Draw(world, screen)
	}
}

// sortSystemsByPriority sorts the systems based on their priority.
func (self *SystemManager) sortSystemsByPriority() {
	// Simple bubble sort by priority
	n := len(self.systems)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if self.systems[j].GetPriority() > self.systems[j+1].GetPriority() {
				self.systems[j], self.systems[j+1] = self.systems[j+1], self.systems[j]
			}
		}
	}
}
