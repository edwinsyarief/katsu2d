package katsu2d

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/dualgrid"
)

// TileMapRenderSystem is a DrawSystem that renders the dual-grid tilemap.
type TileMapRenderSystem struct {
	tm         *TextureManager
	tileWidth  int
	tileHeight int
}

// NewTileMapRenderSystem creates a new system for rendering the tilemap.
func NewTileMapRenderSystem(tm *TextureManager) *TileMapRenderSystem {
	return &TileMapRenderSystem{
		tm: tm,
	}
}

// Draw draws the lower grid directly to the screen. The upper grid is handled
// by the YSortedRenderSystem for proper sorting with other game objects.
func (self *TileMapRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTTileMap)
	if len(entities) == 0 {
		return
	}
	// Assume one tilemap entity
	mapEntity := entities[0]
	mapCompAny, ok := world.GetComponent(mapEntity, CTTileMap)
	if !ok {
		return
	}
	mapComp := mapCompAny.(*TileMapComponent)
	tilemap := mapComp.TileMap

	// Cache tile size if not already cached
	if self.tileWidth == 0 || self.tileHeight == 0 {
		tileset := tilemap.Tileset()
		for id := range tileset {
			texture := self.tm.Get(id)
			if texture != nil {
				bounds := texture.Bounds()
				self.tileWidth = bounds.Dx()
				self.tileHeight = bounds.Dy()
				break
			}
		}
		// If still no size, can't draw
		if self.tileWidth == 0 || self.tileHeight == 0 {
			return
		}
	}

	// Draw lower grid directly, as it's always in the background.
	for y := 0; y < tilemap.Height; y++ {
		for x := 0; x < tilemap.Width; x++ {
			tile := tilemap.LowerGrid.Get(x, y)
			if tile != nil {
				self.drawTile(renderer, tile, x, y)
			}
		}
	}
}

// drawTile is a helper to draw a single tile directly to the batch renderer.
// This is used for the lower grid, which does not need to be sorted with sprites.
func (self *TileMapRenderSystem) drawTile(renderer *BatchRenderer, tile *dualgrid.Tile, x, y int) {
	texture := self.tm.Get(tile.ID)
	if texture == nil {
		return
	}

	drawX := float64(x * self.tileWidth)
	drawY := float64(y * self.tileHeight)

	srcW := float32(self.tileWidth)
	srcH := float32(self.tileHeight)

	renderer.DrawQuad(
		ebimath.V(drawX, drawY),
		ebimath.V(1, 1),
		0.0,
		texture,
		color.RGBA{255, 255, 255, 255},
		0, 0, srcW, srcH,
		float64(self.tileWidth), float64(self.tileHeight),
	)
}
