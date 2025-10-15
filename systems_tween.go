package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
)

type TweenSystem struct {
	filter *teishoku.Filter[TweenComponent]
}

func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

func (self *TweenSystem) Initialize(w *teishoku.World) {
	self.filter = self.filter.New(w)
}

func (self *TweenSystem) Update(w *teishoku.World, dt float64) {
	self.filter.Reset()
	for self.filter.Next() {
		tw := self.filter.Get()
		if tw.Finished {
			continue
		}

		tw.Current = tw.Start
		tw.Time += dt

		if tw.Time < tw.Delay {
			continue
		}

		if tw.Time >= tw.Duration+tw.Delay {
			tw.Current = tw.End
			tw.Finished = true

			Publish(w, TweenFinishedEvent{
				Entity: self.filter.Entity(),
				ID:     tw.ID,
			})

			continue
		}

		tw.Current = EaseTypes[float64](tw.EaseType)(tw.Time-tw.Delay, tw.Start, tw.End-tw.Start, tw.Duration)
	}
}
