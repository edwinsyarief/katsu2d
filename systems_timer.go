package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
)

type TimerSystem struct {
	filter *teishoku.Filter[TimerComponent]
}

func NewTimerSystem() *TimerSystem {
	return &TimerSystem{}
}

func (self *TimerSystem) Initialize(w *teishoku.World) {
	self.filter = self.filter.New(w)
}

func (self *TimerSystem) Update(w *teishoku.World, dt float64) {
	self.filter.Reset()
	for self.filter.Next() {
		t := self.filter.Get()
		if t.State == TimerStateActive {
			t.Time -= dt

			if t.Time <= 0 {
				t.Time = 0
				t.State = TimerStateDone
				Publish(w, TimerFinishedEvent{
					Entity: self.filter.Entity(),
					ID:     t.ID,
				})
			}
		}
	}
}
