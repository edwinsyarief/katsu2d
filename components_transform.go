package katsu2d

type TransformComponent struct {
	Position, Scale, Offset, Origin Point
	Rotation, Z                     float64
	IsDirty                         bool
}
