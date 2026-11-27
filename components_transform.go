package katsu2d

// TransformComponent stores 2D spatial transform properties for an entity.
type TransformComponent struct {
	Position, Scale, Offset, Origin Point
	Rotation, Z                     float64
	IsDirty                         bool
}

// NewTransformComponent creates a TransformComponent with default unit scale (1, 1).
func NewTransformComponent() TransformComponent {
	return TransformComponent{
		Scale: Point{X: 1, Y: 1},
	}
}
