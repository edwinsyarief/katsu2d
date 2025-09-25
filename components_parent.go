package katsu2d

import "github.com/edwinsyarief/lazyecs"

// ParentComponent holds a reference to a parent entity.
type ParentComponent struct {
	Parent lazyecs.Entity
}

func NewParentComponent(parent lazyecs.Entity) *ParentComponent {
	return &ParentComponent{
		Parent: parent,
	}
}
