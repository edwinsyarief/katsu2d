package dualgrid

// Tile represents a single tile in the grid.
// It contains an ID for identifying the tile type and a map for custom properties.
type Tile struct {
	ID         int
	Properties map[string]interface{}
}

// NewTile creates a new Tile with the given ID.
func NewTile(id int) *Tile {
	return &Tile{
		ID:         id,
		Properties: make(map[string]interface{}),
	}
}
