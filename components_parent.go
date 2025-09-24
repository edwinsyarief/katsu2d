package katsu2d

import "github.com/edwinsyarief/lazyecs"

// ParentComponent holds a reference to a parent entity.
type ParentComponent struct {
	Parent lazyecs.Entity
}

func (self *ParentComponent) Init(parent lazyecs.Entity) {
	self.Parent = parent
}
