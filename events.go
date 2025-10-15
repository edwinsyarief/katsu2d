package katsu2d

import "github.com/edwinsyarief/teishoku"

type EngineLayoutChangedEvent struct {
	Width, Height int
}

type TweenFinishedEvent struct {
	Entity teishoku.Entity
	ID     string
}

type TimerFinishedEvent struct {
	Entity teishoku.Entity
	ID     string
}
