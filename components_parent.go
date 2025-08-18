package katsu2d

// ParentComponent holds a reference to a parent entity.
type ParentComponent struct {
	Parent Entity
}

// NewParentComponent creates a new ParentComponent.
func NewParentComponent(parent Entity) *ParentComponent {
	return &ParentComponent{Parent: parent}
}
