package dualgrid

import (
	"os"
	"testing"
)

func TestNewGrid(t *testing.T) {
	grid := NewGrid(10, 20)
	if grid.Width != 10 {
		t.Errorf("Expected width 10, got %d", grid.Width)
	}
	if grid.Height != 20 {
		t.Errorf("Expected height 20, got %d", grid.Height)
	}
	if len(grid.Tiles) != 200 {
		t.Errorf("Expected 200 tiles, got %d", len(grid.Tiles))
	}
}

func TestGridGetSet(t *testing.T) {
	grid := NewGrid(5, 5)
	tile := NewTile(1)

	// Test in-bounds Set and Get
	grid.Set(2, 2, tile)
	retrievedTile := grid.Get(2, 2)
	if retrievedTile != tile {
		t.Errorf("Expected tile %v, got %v", tile, retrievedTile)
	}

	// Test out-of-bounds Get
	if grid.Get(-1, 0) != nil {
		t.Errorf("Expected nil for out-of-bounds Get")
	}
	if grid.Get(5, 0) != nil {
		t.Errorf("Expected nil for out-of-bounds Get")
	}

	// Test out-of-bounds Set
	grid.Set(-1, 0, tile) // should not panic
	grid.Set(5, 0, tile)  // should not panic
}

func TestNewDualGridTileMap(t *testing.T) {
	tilemap := NewDualGridTileMap(10, 10)
	if tilemap.Width != 10 {
		t.Errorf("Expected width 10, got %d", tilemap.Width)
	}
	if tilemap.Height != 10 {
		t.Errorf("Expected height 10, got %d", tilemap.Height)
	}
	if tilemap.LowerGrid == nil || tilemap.UpperGrid == nil {
		t.Errorf("Expected grids to be initialized")
	}
}

func TestLoadFromJSON(t *testing.T) {
	data, err := os.ReadFile("../testdata/map.json")
	if err != nil {
		t.Fatalf("Failed to read test map data: %v", err)
	}

	tilemap, err := LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	if tilemap.Width != 5 || tilemap.Height != 5 {
		t.Errorf("Expected map dimensions 5x5, got %dx%d", tilemap.Width, tilemap.Height)
	}

	// Check a tile on the lower grid
	grassTile := tilemap.LowerGrid.Get(0, 0)
	if grassTile == nil || grassTile.ID != 1 {
		t.Errorf("Expected tile with ID 1 at (0,0) on lower grid")
	}

	// Check a tile on the upper grid
	treeTile := tilemap.UpperGrid.Get(1, 1)
	if treeTile == nil || treeTile.ID != 3 {
		t.Errorf("Expected tile with ID 3 at (1,1) on upper grid")
	}

	// Check an empty tile
	emptyTile := tilemap.UpperGrid.Get(0, 0)
	if emptyTile != nil {
		t.Errorf("Expected nil for empty tile at (0,0) on upper grid")
	}
}

func TestLoadFromJSON_InvalidData(t *testing.T) {
	testCases := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Invalid JSON",
			data:    []byte("{"),
			wantErr: true,
		},
		{
			name: "Grid size mismatch",
			data: []byte(`{"width": 2, "height": 2, "tileset": {}, "lower_grid": [1, 2, 3], "upper_grid": [1, 2, 3, 4]}`),
			wantErr: true,
		},
		{
			name: "Unknown tile ID",
			data: []byte(`{"width": 1, "height": 1, "tileset": {"1": {}}, "lower_grid": [2], "upper_grid": [1]}`),
			wantErr: true,
		},
		{
			name:    "Negative dimensions",
			data:    []byte(`{"width": -1, "height": 1, "tileset": {}, "lower_grid": [], "upper_grid": []}`),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFromJSON(tc.data)
			if (err != nil) != tc.wantErr {
				t.Errorf("LoadFromJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
