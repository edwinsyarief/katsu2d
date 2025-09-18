package dualgrid

import (
	"errors"
	"fmt"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// grid represents a 2D slice of TileType.
type grid [][]TileType

// newGridWithValue creates a new grid of the specified dimensions and fills it with a single value.
func newGridWithValue(width, height int, value TileType) grid {
	grid := make([][]TileType, width)
	for x := range grid {
		grid[x] = make([]TileType, height)
		for y := range grid[x] {
			grid[x][y] = value
		}
	}
	return grid
}

// Get returns the TileType at the specified coordinates.
func (self grid) Get(x, y int) TileType {
	return self[x][y]
}

// Set sets the TileType at the specified coordinates to a new value.
func (self grid) Set(x, y int, value TileType) {
	self[x][y] = value
}

// TileType is a custom type for grid tile values.
type TileType uint8

// Material is a slice of Ebiten images.
type Material []*ebiten.Image

// GridMode defines the drawing mode of the grid.
type GridMode int

const (
	// Square mode is the only supported grid mode.
	Square GridMode = iota
)

// DualGrid manages the grid data, materials, and drawing functionality.
type DualGrid struct {
	materials       []Material
	defaultMaterial TileType
	tileSize        int
	worldGrid       grid
	gridWidth       int
	gridHeight      int
	mode            GridMode
	atlas           *ebiten.Image
}

// NewDualGrid creates and returns a new DualGrid instance.
func NewDualGrid(width, height, tileSize int, defaultMaterial TileType) DualGrid {
	g := DualGrid{
		materials:       []Material{},
		defaultMaterial: defaultMaterial,
		tileSize:        tileSize,
		worldGrid:       newGridWithValue(width, height, defaultMaterial),
		gridWidth:       width,
		gridHeight:      height,
		mode:            Square,
	}
	return g
}

// Reset re-initializes the grid with new dimensions and default material.
func (self *DualGrid) Reset(width, height int, defaultMaterial TileType) {
	self.worldGrid = newGridWithValue(width, height, defaultMaterial)
	self.defaultMaterial = defaultMaterial
}

// Materials returns the slice of materials.
func (self *DualGrid) Materials() []Material {
	return self.materials
}

// DefaultMaterial returns the default tile type.
func (self *DualGrid) DefaultMaterial() TileType {
	return self.defaultMaterial
}

// GridSize returns the width and height of the grid.
func (self *DualGrid) GridSize() (int, int) {
	return self.gridWidth, self.gridHeight
}

// TileSize returns the size of each tile.
func (self *DualGrid) TileSize() int {
	return self.tileSize
}

// Mode returns the current grid mode.
func (self *DualGrid) Mode() GridMode {
	return self.mode
}

// SetMode sets the grid drawing mode. Note: Only Square is currently supported.
func (self *DualGrid) SetMode(mode GridMode) {
	self.mode = mode
}

// GetTile returns the tile type at a specific coordinate.
func (self *DualGrid) GetTile(x, y int) TileType {
	return self.worldGrid.Get(x, y)
}

// SetTile sets the tile type at a specific coordinate.
func (self *DualGrid) SetTile(x, y int, value TileType) {
	self.worldGrid.Set(x, y, value)
}

// AddMaterial adds a new material to the grid, creating the necessary variants from a mask.
func (self *DualGrid) AddMaterial(material, mask *ebiten.Image) {
	if material.Bounds().Dx() != self.tileSize || material.Bounds().Dy() != self.tileSize {
		errMsg := fmt.Sprintf("Material isn't the right dimension: %dx%d, tile size: %d", material.Bounds().Dx(), material.Bounds().Dy(), self.tileSize)
		log.Fatal(errors.New(errMsg))
	}

	side := 4
	order := []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}
	num := side * side

	if mask.Bounds().Dx() != side*self.tileSize || mask.Bounds().Dy() != side*self.tileSize {
		errMsg := fmt.Sprintf("Mask isn't the right dimension: %dx%d, tile size: %d", mask.Bounds().Dx(), mask.Bounds().Dy(), self.tileSize)
		log.Fatal(errors.New(errMsg))
	}

	opts := &ebiten.DrawImageOptions{}
	multiplyOpts := &ebiten.DrawImageOptions{
		Blend: ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorZero,
			BlendFactorSourceAlpha:      ebiten.BlendFactorSourceAlpha,
			BlendFactorDestinationRGB:   ebiten.BlendFactorSourceColor,
			BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		},
	}

	newMaterial := make(Material, num)
	tempImage := ebiten.NewImage(self.tileSize, self.tileSize)
	for i := 0; i < num; i++ {
		x := (i % side) * self.tileSize
		y := (i / side) * self.tileSize

		finalImage := ebiten.NewImage(self.tileSize, self.tileSize)

		if mask != nil {
			tempImage.DrawImage(material, opts)
			tempImage.DrawImage(mask.SubImage(image.Rect(x, y, x+self.tileSize, y+self.tileSize)).(*ebiten.Image), multiplyOpts)

			finalImage.DrawImage(tempImage, opts)
		} else {
			finalImage.DrawImage(material.SubImage(image.Rect(x, y, x+self.tileSize, y+self.tileSize)).(*ebiten.Image), opts)
		}

		idx := order[i]
		newMaterial[idx] = finalImage
	}
	self.materials = append(self.materials, newMaterial)
	self.atlas = nil
}

// buildAtlas pre-renders all material variants onto a single texture atlas for efficient drawing.
func (self *DualGrid) buildAtlas() {
	numMats := len(self.materials)
	if numMats == 0 {
		return
	}
	numVars := len(self.materials[0])
	atlasW := numVars * self.tileSize
	atlasH := numMats * self.tileSize
	self.atlas = ebiten.NewImage(atlasW, atlasH)
	opts := &ebiten.DrawImageOptions{}
	for m, mat := range self.materials {
		for b := 0; b < len(mat); b++ {
			opts.GeoM.Reset()
			opts.GeoM.Translate(float64(b*self.tileSize), float64(m*self.tileSize))
			self.atlas.DrawImage(mat[b], opts)
		}
	}
}

// DrawTo draws the entire grid to a given Ebiten image.
func (self DualGrid) DrawTo(img *ebiten.Image) {
	if self.atlas == nil {
		self.buildAtlas()
	}
	if self.atlas == nil {
		return
	}

	img.Clear()

	verts := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	var idx uint16 = 0

	matTypeMask := make([]bool, len(self.materials))
	ts := float32(self.tileSize)

	var tl, tr, bl, br TileType

	for x := 0; x < self.gridWidth+1; x++ {
		for y := 0; y < self.gridHeight+1; y++ {
			xPos := float64(x * self.tileSize)
			yPos := float64(y * self.tileSize)

			tl = self.defaultMaterial
			tr = self.defaultMaterial
			bl = self.defaultMaterial
			br = self.defaultMaterial

			if x >= 1 && y >= 1 {
				tl = self.worldGrid[x-1][y-1]
			}
			if x < self.gridWidth && y >= 1 {
				tr = self.worldGrid[x][y-1]
			}
			if x >= 1 && y < self.gridHeight {
				bl = self.worldGrid[x-1][y]
			}
			if x < self.gridWidth && y < self.gridHeight {
				br = self.worldGrid[x][y]
			}

			for i := range matTypeMask {
				matTypeMask[i] = false
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			for i := 0; i < len(self.materials); i++ {
				if !matTypeMask[i] {
					continue
				}
				matType := TileType(i)
				bitmask := 0
				if tl == matType || tl > matType {
					bitmask |= 1 << 3
				}
				if tr == matType || tr > matType {
					bitmask |= 1 << 2
				}
				if bl == matType || bl > matType {
					bitmask |= 1 << 1
				}
				if br == matType || br > matType {
					bitmask |= 1 << 0
				}

				dx := float32(xPos)
				dy := float32(yPos)
				dx2 := dx + ts
				dy2 := dy + ts

				srcX := float32(bitmask * self.tileSize)
				srcY := float32(int(matType) * self.tileSize)
				maxSrcX := srcX + ts
				maxSrcY := srcY + ts

				verts = append(verts,
					ebiten.Vertex{DstX: dx, DstY: dy, SrcX: srcX, SrcY: srcY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dx2, DstY: dy, SrcX: maxSrcX, SrcY: srcY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dx, DstY: dy2, SrcX: srcX, SrcY: maxSrcY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dx2, DstY: dy2, SrcX: maxSrcX, SrcY: maxSrcY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
				)
				indices = append(indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
				idx += 4
			}
		}
	}

	img.DrawTriangles(verts, indices, self.atlas, nil)
}

// WorldSize returns the dimensions of the world in pixels.
func (self DualGrid) WorldSize() (int, int) {
	ts := self.tileSize
	return (self.gridWidth + 1) * ts, (self.gridHeight + 1) * ts
}

// WorldCoordFromScreen converts screen coordinates to world grid coordinates.
func (self DualGrid) WorldCoordFromScreen(sx, sy int) (wx, wy int, ok bool) {
	wx = (sx - self.tileSize/2) / self.tileSize
	wy = (sy - self.tileSize/2) / self.tileSize

	if wx < 0 || wx >= self.gridWidth || wy < 0 || wy >= self.gridHeight {
		return 0, 0, false
	}
	return wx, wy, true
}
