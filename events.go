package katsu2d

import (
	"github.com/mlange-42/ark/ecs"
)

type EngineLayoutChangedEvent struct {
	Width, Height int
}

type TweenFinishedEvent struct {
	Entity ecs.Entity
	ID     string
}

type TimerFinishedEvent struct {
	Entity ecs.Entity
	ID     string
}
