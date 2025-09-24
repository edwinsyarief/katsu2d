package katsu2d

// TagComponent provides a simple string identifier for entities
// Useful for categorizing and looking up entities by name or type
type TagComponent struct {
	Tag string // The identifying string for this entity
}

func (self *TagComponent) Init(tag string) {
	self.Tag = tag
}
