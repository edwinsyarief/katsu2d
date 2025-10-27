package katsu2d

import "github.com/mlange-42/ark/ecs"

type TweenSystem struct {
	filter *ecs.Filter1[TweenComponent]
}

func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

func (self *TweenSystem) Initialize(w *ecs.World) {
	self.filter = self.filter.New(w)
}

func (self *TweenSystem) Update(w *ecs.World, dt float64) {
	query := self.filter.Query()
	for query.Next() {
		tw := query.Get()
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
				Entity: query.Entity(),
				ID:     tw.ID,
			})

			continue
		}

		tw.Current = EaseTypes[float64](tw.EaseType)(tw.Time-tw.Delay, tw.Start, tw.End-tw.Start, tw.Duration)
	}
}
