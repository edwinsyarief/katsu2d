package katsu2d

// TagComponent attaches a string tag label to an entity for grouping or lookup.
type TagComponent struct {
	Tag string
}

// NewTagComponent creates a TagComponent with the given tag string.
func NewTagComponent(tag string) TagComponent {
	return TagComponent{
		Tag: tag,
	}
}
