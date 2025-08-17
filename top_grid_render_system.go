package katsu2d

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// TopGridRenderSystem is a DrawSystem that renders the top-most grid of a tilemap.
// It should be added as an overlay system to ensure it draws on top of all other elements.
type TopGridRenderSystem struct {
	tm         *TextureManager
	tileWidth  int
	tileHeight int
}

// NewTopGridRenderSystem creates a new system for rendering the top grid.
func NewTopGridRenderSystem(tm *TextureManager) *TopGridRenderSystem {
	return &TopGridRenderSystem{
		tm: tm,
	}
}

// Draw draws the top grid tiles.
func (self *TopGridRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTTileMap)
	if len(entities) == 0 {
		return
	}
	mapEntity := entities[0]
	mapCompAny, ok := world.GetComponent(mapEntity, CTTileMap)
	if !ok {
		return
	}
	mapComp := mapCompAny.(*TileMapComponent)
	tilemap := mapComp.TileMap

	if tilemap.TopGrid == nil {
		return
	}

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
		if self.tileWidth == 0 || self.tileHeight == 0 {
			return
		}
	}

	// Draw top grid
	for y := 0; y < tilemap.Height; y++ {
		for x := 0; x < tilemap.Width; x++ {
			tile := tilemap.TopGrid.Get(x, y)
			if tile != nil {
				texture := self.tm.Get(tile.ID)
				if texture == nil {
					continue
				}
				drawX := float64(x * self.tileWidth)
				drawY := float64(y * self.tileHeight)

				renderer.DrawQuad(
					ebimath.V(drawX, drawY),
					ebimath.V(1, 1),
					0.0,
					texture,
					color.RGBA{255, 255, 255, 255},
					0, 0, float32(self.tileWidth), float32(self.tileHeight),
					float64(self.tileWidth), float64(self.tileHeight),
				)
			}
		}
	}
}
