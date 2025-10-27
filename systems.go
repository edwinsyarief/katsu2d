package katsu2d

import "github.com/mlange-42/ark/ecs"

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Initialize(*ecs.World)
	Update(*ecs.World, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Initialize(*ecs.World)
	Draw(*ecs.World, *BatchRenderer)
}
