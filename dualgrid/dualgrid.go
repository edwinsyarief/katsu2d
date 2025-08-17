package dualgrid

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Grid represents a 2D grid of tiles.
// Tiles are stored in a 1D slice for performance.
type Grid struct {
	Width  int
	Height int
	Tiles  []*Tile
}

// NewGrid creates a new Grid with the given dimensions.
func NewGrid(width, height int) *Grid {
	return &Grid{
		Width:  width,
		Height: height,
		Tiles:  make([]*Tile, width*height),
	}
}

// Get returns the tile at the given coordinates.
// It returns nil if the coordinates are out of bounds.
func (g *Grid) Get(x, y int) *Tile {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return nil
	}
	return g.Tiles[y*g.Width+x]
}

// Set sets the tile at the given coordinates.
// It does nothing if the coordinates are out of bounds.
func (g *Grid) Set(x, y int, tile *Tile) {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return
	}
	g.Tiles[y*g.Width+x] = tile
}

// DualGridTileMap is the main component.
// It holds two grids: a lower grid for the ground and an upper grid for objects.
type DualGridTileMap struct {
	Width     int
	Height    int
	LowerGrid *Grid
	UpperGrid *Grid
	tileset   map[int]*Tile
}

// NewDualGridTileMap creates a new DualGridTileMap with the given dimensions.
func NewDualGridTileMap(width, height int) *DualGridTileMap {
	return &DualGridTileMap{
		Width:     width,
		Height:    height,
		LowerGrid: NewGrid(width, height),
		UpperGrid: NewGrid(width, height),
		tileset:   make(map[int]*Tile),
	}
}

// mapData is used for unmarshaling the JSON map file.
type mapData struct {
	Width     int                               `json:"width"`
	Height    int                               `json:"height"`
	Tileset   map[string]map[string]interface{} `json:"tileset"`
	LowerGrid []int                             `json:"lower_grid"`
	UpperGrid []int                             `json:"upper_grid"`
}

// LoadFromJSON creates a new DualGridTileMap from a JSON byte slice.
func LoadFromJSON(data []byte) (*DualGridTileMap, error) {
	var md mapData
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("failed to unmarshal map data: %w", err)
	}

	if md.Width <= 0 || md.Height <= 0 {
		return nil, fmt.Errorf("map width and height must be positive")
	}

	tilemap := NewDualGridTileMap(md.Width, md.Height)

	// Create tileset
	for idStr, props := range md.Tileset {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tile ID in tileset: %s", idStr)
		}
		tile := NewTile(id)
		tile.Properties = props
		tilemap.tileset[id] = tile
	}

	// Populate lower grid
	if len(md.LowerGrid) != md.Width*md.Height {
		return nil, fmt.Errorf("lower_grid size (%d) does not match map dimensions (%d x %d = %d)", len(md.LowerGrid), md.Width, md.Height, md.Width*md.Height)
	}
	for i, tileID := range md.LowerGrid {
		if tileID == 0 {
			continue // 0 is an empty tile
		}
		tile, ok := tilemap.tileset[tileID]
		if !ok {
			return nil, fmt.Errorf("tile with ID %d not found in tileset", tileID)
		}
		tilemap.LowerGrid.Tiles[i] = tile
	}

	// Populate upper grid
	if len(md.UpperGrid) != md.Width*md.Height {
		return nil, fmt.Errorf("upper_grid size (%d) does not match map dimensions (%d x %d = %d)", len(md.UpperGrid), md.Width, md.Height, md.Width*md.Height)
	}
	for i, tileID := range md.UpperGrid {
		if tileID == 0 {
			continue // 0 is an empty tile
		}
		tile, ok := tilemap.tileset[tileID]
		if !ok {
			return nil, fmt.Errorf("tile with ID %d not found in tileset", tileID)
		}
		tilemap.UpperGrid.Tiles[i] = tile
	}

	return tilemap, nil
}

// GetTileFromTileset retrieves a tile prototype from the tileset by its ID.
// This is useful for getting tile information without accessing a specific grid cell.
func (tm *DualGridTileMap) GetTileFromTileset(id int) *Tile {
	return tm.tileset[id]
}

// Tileset returns the entire tileset map.
func (tm *DualGridTileMap) Tileset() map[int]*Tile {
	return tm.tileset
}
