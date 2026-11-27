package katsu2d

// OrderableComponent controls draw ordering within a Z layer.
// When AutoSortY is true, OrderedSpriteSystem updates Index each frame to Position.Y + YOffset.
type OrderableComponent struct {
	Index     float64
	AutoSortY bool
	YOffset   float64
}

// NewOrderableComponent creates an OrderableComponent with a fixed sort index.
func NewOrderableComponent(index float64) OrderableComponent {
	return OrderableComponent{
		Index: index,
	}
}

// NewYOrderableComponent creates an OrderableComponent that automatically sorts by Position.Y + yOffset.
func NewYOrderableComponent(yOffset float64) OrderableComponent {
	return OrderableComponent{
		AutoSortY: true,
		YOffset:   yOffset,
	}
}
