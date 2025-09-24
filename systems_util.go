package katsu2d

import (
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/lazyecs"
)

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

// NewTweenSystem creates a new TweenSystem.
func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

// Update updates all standalone tweens and sequences in the world.
func (self *TweenSystem) Update(world *lazyecs.World, dt float64) {
	// Standalone tweens
	query := world.Query(CTTween)
	for query.Next() {
		for _, entity := range query.Entities() {
			tw, _ := lazyecs.GetComponent[tween.Tween](world, entity)
			tw.Update(float32(dt))
		}
	}

	// Standalone Sequences
	query = world.Query(CTSequence)
	for query.Next() {
		for _, entity := range query.Entities() {
			seq, _ := lazyecs.GetComponent[tween.Sequence](world, entity)
			seq.Update(float32(dt))
		}
	}
}

// CooldownSystem manages cooldowns.
type CooldownSystem struct{}

// NewCooldownSystem creates a new CooldownSystem.
func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

// Update advances all cooldown managers in the world by the given delta time.
func (self *CooldownSystem) Update(world *lazyecs.World, dt float64) {
	query := world.Query(CTCooldown)
	for query.Next() {
		for _, entity := range query.Entities() {
			cm, _ := lazyecs.GetComponent[managers.CooldownManager](world, entity)
			cm.Update(dt)
		}
	}
}

// DelaySystem manages delays.
type DelaySystem struct{}

// NewDelaySystem creates a new DelaySystem.
func NewDelaySystem() *DelaySystem {
	return &DelaySystem{}
}

// Update advances all delay managers in the world by the given delta time.
func (self *DelaySystem) Update(world *lazyecs.World, dt float64) {
	query := world.Query(CTDelayer)
	for query.Next() {
		for _, entity := range query.Entities() {
			delay, _ := lazyecs.GetComponent[managers.DelayManager](world, entity)
			delay.Update(dt)
		}
	}
}
