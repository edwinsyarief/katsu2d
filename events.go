package katsu2d

import "github.com/edwinsyarief/lazyecs"

type EngineLayoutChangedEvent struct {
	Width, Height int
}

type TweenFinishedEvent struct {
	Entity lazyecs.Entity
	ID     string
}

type TimerFinishedEvent struct {
	Entity lazyecs.Entity
	ID     string
}
