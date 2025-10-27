package katsu2d

import "github.com/mlange-42/ark/ecs"

type TimerSystem struct {
	filter *ecs.Filter1[TimerComponent]
}

func NewTimerSystem() *TimerSystem {
	return &TimerSystem{}
}

func (self *TimerSystem) Initialize(w *ecs.World) {
	self.filter = self.filter.New(w)
}

func (self *TimerSystem) Update(w *ecs.World, dt float64) {
	query := self.filter.Query()
	for query.Next() {
		t := query.Get()
		if t.State == TimerStateActive {
			t.Time -= dt

			if t.Time <= 0 {
				t.Time = 0
				t.State = TimerStateDone
				Publish(w, TimerFinishedEvent{
					Entity: query.Entity(),
					ID:     t.ID,
				})
			}
		}
	}
}
