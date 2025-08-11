package managers

import (
	"fmt"
	"slices"
)

type delayState int

const (
	delay_task_idle = iota
	delay_task_active
	delay_task_done
)

type delayTask struct {
	id       string
	time     float64
	callback func()
	state    delayState
}

type Delayer struct {
	delays []*delayTask
}

func NewDelayer() *Delayer {
	return &Delayer{
		delays: []*delayTask{},
	}
}

func (self *Delayer) Update(delta float64) {
	for _, t := range self.delays {
		if t.state == delay_task_active {
			t.time -= delta

			if t.time <= 0 {
				t.callback()
				t.state = delay_task_done
			}
		}
	}

	self.delays = slices.DeleteFunc(self.delays, func(t *delayTask) bool {
		return t.state == delay_task_done
	})
}

func (self *Delayer) Add(id string, time float64, callback func()) {
	var exist, index = self.HasId(id)
	if exist {
		self.delays[index] = &delayTask{id: id, time: time, callback: callback, state: delay_task_idle}
	} else {
		self.delays = append(self.delays, &delayTask{id: id, time: time, callback: callback, state: delay_task_idle})
	}

	self.sortTasks()
}

func (self *Delayer) HasId(id string) (bool, int) {
	var index = slices.IndexFunc(self.delays, func(t *delayTask) bool {
		return t.id == id
	})

	return index != -1, index
}

func (self *Delayer) Activate(id string) {
	var has, index = self.HasId(id)
	if !has {
		panic(fmt.Sprintf("Delayer with task id: '%s' doesn't exist", id))
	}

	self.delays[index].state = delay_task_active
}

func (self *Delayer) sortTasks() {
	slices.SortFunc(self.delays, func(t1 *delayTask, t2 *delayTask) int {
		if t1.time > t2.time {
			return 1
		} else if t1.time < t2.time {
			return -1
		} else {
			return 0
		}
	})
}
