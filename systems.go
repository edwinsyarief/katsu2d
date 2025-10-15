package katsu2d

import "github.com/edwinsyarief/teishoku"

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Initialize(*teishoku.World)
	Update(*teishoku.World, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Initialize(*teishoku.World)
	Draw(*teishoku.World, *BatchRenderer)
}
