package components

import "katsu2d/constants"

// Cooldown component tracks a cooldown timer for an entity.
type Cooldown struct {
	Total     float64
	Remaining float64
	Active    bool
}

// GetTypeID returns the component type ID.
func (self *Cooldown) GetTypeID() int {
	return constants.ComponentCooldown
}

// Start begins a new cooldown period.
func (self *Cooldown) Start(duration float64) {
	self.Remaining = duration
	self.Active = true
}

// IsReady checks if the cooldown is over.
func (self *Cooldown) IsReady() bool {
	return !self.Active || self.Remaining <= 0
}
