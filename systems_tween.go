package katsu2d

import "github.com/edwinsyarief/teishoku"

// TweenSystem advances value tweens and publishes TweenFinishedEvent upon completion.
type TweenSystem struct {
	filter      *teishoku.Filter[TweenComponent]
	finishedCh  teishoku.EventChannel[TweenFinishedEvent]
	initialized bool
}

// NewTweenSystem creates a new TweenSystem.
func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

// Initialize prepares the TweenSystem's ECS filter and event channel.
func (s *TweenSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter[TweenComponent](w)
	s.finishedCh = teishoku.NewEventChannel[TweenFinishedEvent](getEventBus(w))
	s.initialized = true
}

// Update advances active tweens by dt and emits TweenFinishedEvent when they complete.
func (s *TweenSystem) Update(w *teishoku.World, dt float64) {
	chunks := s.filter.Chunks()
	for chunks.Next() {
		entities := chunks.Entities()
		tweens := chunks.Get()
		for i := range tweens {
			tw := &tweens[i]
			if tw.Finished {
				continue
			}

			tw.Current = tw.Start
			tw.Time += dt

			if tw.Time < tw.Delay {
				continue
			}

			if tw.Duration <= 0 || tw.Time >= tw.Duration+tw.Delay {
				tw.Current = tw.End
				tw.Finished = true
				s.finishedCh.Publish(TweenFinishedEvent{
					Entity: entities[i],
					ID:     tw.ID,
				})
				continue
			}

			tw.Current = EaseTypes[float64](tw.EaseType)(tw.Time-tw.Delay, tw.Start, tw.End-tw.Start, tw.Duration)
		}
	}
}
