package katsu2d

import (
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
)

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

// NewTweenSystem creates a new TweenSystem.
func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

// Update updates all standalone tweens and sequences in the world.
func (self *TweenSystem) Update(world *World, dt float64) {
	// Standalone tweens
	entities := world.Query(CTTween)
	for _, e := range entities {
		twAny, _ := world.GetComponent(e, CTTween)
		tw := twAny.(*tween.Tween)
		tw.Update(float32(dt))
	}

	// Standalone Sequences
	entities = world.Query(CTSequence)
	for _, e := range entities {
		seqAny, _ := world.GetComponent(e, CTSequence)
		seq := seqAny.(*tween.Sequence)
		seq.Update(float32(dt))
	}
}

// CooldownSystem manages cooldowns.
type CooldownSystem struct{}

// NewCooldownSystem creates a new CooldownSystem.
func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

// Update advances all cooldown managers in the world by the given delta time.
func (self *CooldownSystem) Update(world *World, dt float64) {
	entities := world.Query(CTCooldown)
	for _, e := range entities {
		cmAny, _ := world.GetComponent(e, CTCooldown)
		cm := cmAny.(*managers.CooldownManager)
		cm.Update(dt)
	}
}

// DelaySystem manages delays.
type DelaySystem struct{}

// NewDelaySystem creates a new DelaySystem.
func NewDelaySystem() *DelaySystem {
	return &DelaySystem{}
}

// Update advances all delay managers in the world by the given delta time.
func (self *DelaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTDelayer)
	for _, e := range entities {
		delayAny, _ := world.GetComponent(e, CTDelayer)
		delay := delayAny.(*managers.DelayManager)
		delay.Update(dt)
	}
}
