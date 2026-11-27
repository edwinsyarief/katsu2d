package managers

import (
	"cmp"
	"fmt"
	"slices"
)

type delayState int

const (
	delayTaskIdle delayState = iota
	delayTaskActive
	delayTaskDone
)

type delayTask struct {
	callback func()
	id       string
	time     float64
	state    delayState
}

// DelayManager schedules named one-shot delayed callbacks.
type DelayManager struct {
	delays     []*delayTask
	pendingCbs []func()
}

// NewDelayManager creates and returns a new DelayManager.
func NewDelayManager() *DelayManager {
	return &DelayManager{
		delays: []*delayTask{},
	}
}

// Update advances active delay tasks by delta, removes completed tasks, and invokes their callbacks.
func (m *DelayManager) Update(delta float64) {
	m.pendingCbs = m.pendingCbs[:0]
	n := 0
	for _, t := range m.delays {
		if t == nil {
			continue
		}
		if t.state == delayTaskActive {
			t.time -= delta
			if t.time <= 0 {
				t.state = delayTaskDone
				if t.callback != nil {
					m.pendingCbs = append(m.pendingCbs, t.callback)
				}
				continue
			}
		}
		if t.state != delayTaskDone {
			m.delays[n] = t
			n++
		}
	}
	clear(m.delays[n:])
	m.delays = m.delays[:n]

	for i, cb := range m.pendingCbs {
		if cb != nil {
			cb()
			m.pendingCbs[i] = nil
		}
	}
	m.pendingCbs = m.pendingCbs[:0]
}

// Add registers or replaces a delayed task in idle state.
func (m *DelayManager) Add(id string, time float64, callback func()) {
	exist, index := m.HasId(id)
	task := &delayTask{id: id, time: time, callback: callback, state: delayTaskIdle}
	if exist {
		m.delays[index] = task
	} else {
		m.delays = append(m.delays, task)
	}
	m.sortTasks()
}

// HasId reports whether a task with the given id exists and returns its index, or -1 if absent.
func (m *DelayManager) HasId(id string) (bool, int) {
	index := slices.IndexFunc(m.delays, func(t *delayTask) bool {
		return t != nil && t.id == id
	})
	return index != -1, index
}

// Activate marks the task with the given id as active so it counts down during Update.
func (m *DelayManager) Activate(id string) {
	has, index := m.HasId(id)
	if !has {
		panic(fmt.Sprintf("Delayer with task id: '%s' doesn't exist", id))
	}
	m.delays[index].state = delayTaskActive
}

func (m *DelayManager) sortTasks() {
	slices.SortFunc(m.delays, func(t1, t2 *delayTask) int {
		return cmp.Compare(t1.time, t2.time)
	})
}
