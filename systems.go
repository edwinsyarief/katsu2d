package katsu2d

import "github.com/edwinsyarief/lazyecs"

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Initialize(*lazyecs.World)
	Update(*lazyecs.World, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Initialize(*lazyecs.World)
	Draw(*lazyecs.World, *BatchRenderer)
}
