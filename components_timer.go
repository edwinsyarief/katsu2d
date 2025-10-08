package katsu2d

type TimerState int

const (
	TimerStateIdle TimerState = iota
	TimerStateActive
	TimerStateDone
)

type TimerComponent struct {
	ID    string
	Time  float64
	State TimerState
}
