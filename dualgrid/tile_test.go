package dualgrid

import (
	"testing"
)

func TestNewTile(t *testing.T) {
	tile := NewTile(1)
	if tile.ID != 1 {
		t.Errorf("Expected tile ID 1, got %d", tile.ID)
	}
	if tile.Properties == nil {
		t.Errorf("Expected properties map to be initialized, but it was nil")
	}
}
