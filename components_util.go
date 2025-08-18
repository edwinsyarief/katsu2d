package katsu2d

// TagComponent provides a simple string identifier for entities
// Useful for categorizing and looking up entities by name or type
type TagComponent struct {
	Tag string // The identifying string for this entity
}

// NewTagComponent creates a new tag component with the specified string identifier
func NewTagComponent(tag string) *TagComponent {
	return &TagComponent{Tag: tag}
}
