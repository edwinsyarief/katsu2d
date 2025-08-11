package managers

import (
	"fmt"
	"slices"
)

const (
	DEFAULT_COOLDOWN_LIMIT = 512
)

// Cool down instance
type cooldown struct {
	id       string
	duration float64
	initial  float64
	callback func()
}

func (self *cooldown) getRemainingRatio() float64 {
	if self.initial == 0 {
		return 0
	}

	return self.duration / self.initial
}

func (self *cooldown) getProgressRatio() float64 {
	if self.initial == 0 {
		return 0
	}

	return 1 - self.getRemainingRatio()
}

// Cool down manager
type CooldownManager struct {
	cds []*cooldown

	maxSize int
}

// New CoolDown manager
func NewCooldownManager(maxSize int) *CooldownManager {
	result := &CooldownManager{
		maxSize: maxSize,
	}

	if result.maxSize == 0 {
		result.maxSize = DEFAULT_COOLDOWN_LIMIT
	}

	result.cds = []*cooldown{}

	return result
}

func (self *CooldownManager) Has(id string) bool {
	hasId := slices.ContainsFunc(self.cds, func(c *cooldown) bool {
		if c == nil {
			return false
		}

		return c.id == id
	})

	return hasId
}

func (self *CooldownManager) GetProgress(id string) float64 {
	if !self.Has(id) {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}

	c := self.get(id)
	if c.duration <= 0 {
		return 1
	}

	return c.getProgressRatio()
}

func (self *CooldownManager) GetRatio(id string) float64 {
	if !self.Has(id) {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}

	c := self.get(id)
	if c.duration <= 0 {
		return 0
	}

	return c.getRemainingRatio()
}

func (self *CooldownManager) Set(id string, duration float64, callback func()) {
	if duration <= 0 {
		return
	}

	if self.Has(id) {
		return
	}

	self.cds = append(self.cds, &cooldown{
		id:       id,
		duration: duration,
		initial:  duration,
		callback: callback,
	})
}

func (self *CooldownManager) Override(id string, duration float64) {
	if duration <= 0 {
		return
	}

	if !self.Has(id) {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}

	c := self.get(id)
	c.duration = duration
	c.initial = duration
}

func (self *CooldownManager) Update(delta float64) {
	for _, c := range self.cds {
		if c == nil {
			continue
		}

		c.duration -= delta
		if c.duration <= 0 {
			cb := c.callback
			self.remove(c.id)
			if cb != nil {
				cb()
			}
		}
	}
}

func (self *CooldownManager) Reset() {
	self.cds = nil
	self.cds = make([]*cooldown, self.maxSize)
}

func (self *CooldownManager) get(id string) *cooldown {
	index := slices.IndexFunc(self.cds, func(c *cooldown) bool {
		return c.id == id
	})
	return self.cds[index]
}

func (self *CooldownManager) remove(id string) {
	self.cds = slices.DeleteFunc(self.cds, func(c *cooldown) bool {
		return c.id == id
	})
}
