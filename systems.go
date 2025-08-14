package katsu2d

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Update(*World, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Draw(*World, *BatchRenderer)
}
