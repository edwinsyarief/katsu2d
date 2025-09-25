package katsu2d

// OrderableComponent determines the rendering order of an entity.
type OrderableComponent struct {
	IndexFunc   func() float64
	StaticIndex float64
	IsStatic    bool
}

func NewOrderableComponent(indexFunc func() float64) *OrderableComponent {
	return &OrderableComponent{
		IndexFunc: indexFunc,
	}
}

// SetIndex sets a static sorting index for the entity.
// This will override the index function.
func (self *OrderableComponent) SetIndex(index float64) *OrderableComponent {
	self.StaticIndex = index
	self.IsStatic = true
	self.IndexFunc = nil // Clear the function to save memory
	return self
}

// Index returns the sorting index for the entity.
// It will execute the index function if it exists, otherwise it returns the static index.
func (self *OrderableComponent) Index() float64 {
	if self.IsStatic {
		return self.StaticIndex
	}
	if self.IndexFunc != nil {
		return self.IndexFunc()
	}
	return 0 // Default index
}
