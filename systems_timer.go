package katsu2d

import "github.com/edwinsyarief/teishoku"

// TimerSystem updates countdown timers and publishes TimerFinishedEvent on completion.
type TimerSystem struct {
	filter      *teishoku.Filter[TimerComponent]
	finishedCh  teishoku.EventChannel[TimerFinishedEvent]
	initialized bool
}

// NewTimerSystem creates a new TimerSystem.
func NewTimerSystem() *TimerSystem {
	return &TimerSystem{}
}

// Initialize prepares the TimerSystem's ECS filter and event channel.
func (s *TimerSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter[TimerComponent](w)
	s.finishedCh = teishoku.NewEventChannel[TimerFinishedEvent](getEventBus(w))
	s.initialized = true
}

// Update advances active timers and publishes TimerFinishedEvent when they expire.
func (s *TimerSystem) Update(w *teishoku.World, dt float64) {
	chunks := s.filter.Chunks()
	for chunks.Next() {
		entities := chunks.Entities()
		timers := chunks.Get()
		for i := range timers {
			t := &timers[i]
			if t.State == TimerStateActive {
				t.Time -= dt
				if t.Time <= 0 {
					t.Time = 0
					t.State = TimerStateDone
					s.finishedCh.Publish(TimerFinishedEvent{
						Entity: entities[i],
						ID:     t.ID,
					})
				}
			}
		}
	}
}
