package katsu2d

import "github.com/edwinsyarief/katsu2d/dualgrid"

// TileMapComponent is a component that holds a reference to a dual-grid tilemap
// and the Z-layer for its grids.
type TileMapComponent struct {
	TileMap    *dualgrid.DualGridTileMap
	LowerGridZ float64
	UpperGridZ float64
}

// NewTileMapComponent creates a new TileMapComponent with default Z-layers.
func NewTileMapComponent(tileMap *dualgrid.DualGridTileMap) *TileMapComponent {
	return &TileMapComponent{
		TileMap:    tileMap,
		LowerGridZ: 0,
		UpperGridZ: 2, // Default to a higher layer than most sprites
	}
}

// SetZ sets the Z-layers for the tilemap grids and returns the component for chaining.
func (self *TileMapComponent) SetZ(lower, upper float64) *TileMapComponent {
	self.LowerGridZ = lower
	self.UpperGridZ = upper
	return self
}
