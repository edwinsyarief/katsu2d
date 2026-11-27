package katsu2d

// ShapeComponent associates a procedural Shape with an entity for rendering.
type ShapeComponent struct {
	Shape Shape
}

// NewShapeComponent creates a ShapeComponent wrapping the given Shape.
func NewShapeComponent(shape Shape) ShapeComponent {
	return ShapeComponent{
		Shape: shape,
	}
}
