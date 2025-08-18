package katsu2d

// OrderableComponent determines the rendering order of an entity.
type OrderableComponent struct {
	indexFunc   func() float64
	staticIndex float64
	isStatic    bool
}

// NewOrderableComponent creates a new OrderableComponent with a dynamic index function.
func NewOrderableComponent(indexFunc func() float64) *OrderableComponent {
	return &OrderableComponent{
		indexFunc: indexFunc,
		isStatic:  false,
	}
}

// SetIndex sets a static sorting index for the entity.
// This will override the index function.
func (self *OrderableComponent) SetIndex(index float64) {
	self.staticIndex = index
	self.isStatic = true
	self.indexFunc = nil // Clear the function to save memory
}

// Index returns the sorting index for the entity.
// It will execute the index function if it exists, otherwise it returns the static index.
func (self *OrderableComponent) Index() float64 {
	if self.isStatic {
		return self.staticIndex
	}
	if self.indexFunc != nil {
		return self.indexFunc()
	}
	return 0 // Default index
}
