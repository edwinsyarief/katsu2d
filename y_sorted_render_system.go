package katsu2d

import (
	"image/color"
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/dualgrid"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// zLayerMultiplier is used to create a composite sort key from Z and Y values.
	// It ensures that the Z layer is the primary sort key.
	zLayerMultiplier = 10000.0
)

// sortableObject holds all the data needed to render and sort an object.
// It's a unified representation for sprites and upper-layer tiles.
type sortableObject struct {
	texture    *ebiten.Image
	position   ebimath.Vector
	scale      ebimath.Vector
	rotation   float64
	color      color.RGBA
	srcX0      float32
	srcY0      float32
	srcX1      float32
	srcY1      float32
	destWidth  float64
	destHeight float64
	sortKey    float64
}

// YSortedRenderSystem is a DrawSystem that collects sprites and upper-layer tiles,
// sorts them by Z-layer and then Y-position, and draws them in a single batch.
type YSortedRenderSystem struct {
	tm             *TextureManager
	tileWidth      int
	tileHeight     int
	sortableObjects []sortableObject
}

// NewYSortedRenderSystem creates a new YSortedRenderSystem.
func NewYSortedRenderSystem(tm *TextureManager) *YSortedRenderSystem {
	return &YSortedRenderSystem{
		tm:             tm,
		sortableObjects: make([]sortableObject, 0, 1024), // Pre-allocate with capacity
	}
}

// Draw collects, sorts, and draws sprites and upper-layer tiles.
func (self *YSortedRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	// 1. Clear the list from the previous frame.
	self.sortableObjects = self.sortableObjects[:0]

	// 2. Collect sprites.
	self.collectSprites(world)

	// 3. Collect upper-layer tiles.
	self.collectUpperTiles(world)

	// 4. Sort the combined list.
	sort.SliceStable(self.sortableObjects, func(i, j int) bool {
		return self.sortableObjects[i].sortKey < self.sortableObjects[j].sortKey
	})

	// 5. Draw the sorted objects.
	for _, obj := range self.sortableObjects {
		renderer.DrawQuad(
			obj.position,
			obj.scale,
			obj.rotation,
			obj.texture,
			obj.color,
			obj.srcX0, obj.srcY0, obj.srcX1, obj.srcY1,
			obj.destWidth, obj.destHeight,
		)
	}
}

func (self *YSortedRenderSystem) collectSprites(world *World) {
	for _, entity := range world.Query(CTSprite, CTTransform) {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		spr := sprite.(*SpriteComponent)

		img := self.tm.Get(spr.TextureID)
		if img == nil {
			continue
		}

		imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
		srcX, srcY, srcW, srcH := spr.GetSourceRect(float32(imgW), float32(imgH))
		destW, destH := spr.GetDestSize(srcW, srcH)

		effColor := spr.Color
		effColor.A = uint8(float32(spr.Color.A) * spr.Opacity)

		realPos := ebimath.V2(0).Apply(t.Matrix())
		if !t.Origin().IsZero() {
			realPos = realPos.Sub(t.Origin())
		}

		self.sortableObjects = append(self.sortableObjects, sortableObject{
			texture:    img,
			position:   realPos,
			scale:      t.Scale(),
			rotation:   t.Rotation(),
			color:      effColor,
			srcX0:      srcX,
			srcY0:      srcY,
			srcX1:      srcX + srcW,
			srcY1:      srcY + srcH,
			destWidth:  float64(destW),
			destHeight: float64(destH),
			sortKey:    (t.Z * zLayerMultiplier) + t.Position().Y,
		})
	}
}

func (self *YSortedRenderSystem) collectUpperTiles(world *World) {
	entities := world.Query(CTTileMap)
	if len(entities) == 0 {
		return
	}
	mapEntity := entities[0]
	mapCompAny, _ := world.GetComponent(mapEntity, CTTileMap)
	mapComp := mapCompAny.(*TileMapComponent)
	tilemap := mapComp.TileMap

	self.cacheTileSize(tilemap)
	if self.tileWidth == 0 || self.tileHeight == 0 {
		return // No tiles to draw
	}

	for y := 0; y < tilemap.Height; y++ {
		for x := 0; x < tilemap.Width; x++ {
			tile := tilemap.UpperGrid.Get(x, y)
			if tile != nil {
				texture := self.tm.Get(tile.ID)
				if texture == nil {
					continue
				}

				drawX := float64(x * self.tileWidth)
				drawY := float64(y * self.tileHeight)

				self.sortableObjects = append(self.sortableObjects, sortableObject{
					texture:    texture,
					position:   ebimath.V(drawX, drawY),
					scale:      ebimath.V(1, 1),
					rotation:   0.0,
					color:      color.RGBA{255, 255, 255, 255},
					srcX0:      0,
					srcY0:      0,
					srcX1:      float32(self.tileWidth),
					srcY1:      float32(self.tileHeight),
					destWidth:  float64(self.tileWidth),
					destHeight: float64(self.tileHeight),
					sortKey:    (mapComp.UpperGridZ * zLayerMultiplier) + drawY,
				})
			}
		}
	}
}

// cacheTileSize determines the tile dimensions from the first available tile texture.
func (self *YSortedRenderSystem) cacheTileSize(tilemap *dualgrid.DualGridTileMap) {
	if self.tileWidth != 0 && self.tileHeight != 0 {
		return // Already cached
	}
	tileset := tilemap.Tileset()
	for id := range tileset {
		texture := self.tm.Get(id)
		if texture != nil {
			bounds := texture.Bounds()
			self.tileWidth = bounds.Dx()
			self.tileHeight = bounds.Dy()
			return
		}
	}
}
