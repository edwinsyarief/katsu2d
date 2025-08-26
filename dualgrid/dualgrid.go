// dualgrid.go

package dualgrid

import (
	"errors"
	"image"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type grid [][]TileType

/* func newGrid(width, height int) grid {
	grid := make([][]TileType, width)
	for x := range grid {
		grid[x] = make([]TileType, height)
	}
	return grid
} */

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

func (self grid) Get(x, y int) TileType {
	return self[x][y]
}

func (self grid) Set(x, y int, value TileType) {
	self[x][y] = value
}

type TileType uint8

type Material []*ebiten.Image

type GridMode int

const (
	Square GridMode = iota
	Isometric
	Hexagonal
)

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

func (self *DualGrid) Reset(width, height int, defaultMaterial TileType) {
	self.worldGrid = newGridWithValue(width, height, defaultMaterial)
	self.defaultMaterial = defaultMaterial
}

func (self *DualGrid) Materials() []Material {
	return self.materials
}

func (self *DualGrid) DefaultMaterial() TileType {
	return self.defaultMaterial
}

func (self *DualGrid) GridSize() (int, int) {
	return self.gridWidth, self.gridHeight
}

func (self *DualGrid) TileSize() int {
	return self.tileSize
}

func (self *DualGrid) Mode() GridMode {
	return self.mode
}

func (self *DualGrid) SetMode(mode GridMode) {
	self.mode = mode
}

func (self *DualGrid) GetTile(x, y int) TileType {
	return self.worldGrid.Get(x, y)
}

func (self *DualGrid) SetTile(x, y int, value TileType) {
	self.worldGrid.Set(x, y, value)
}

func (self *DualGrid) AddMaterial(material, mask *ebiten.Image) {
	if material.Bounds().Dx() != self.tileSize || material.Bounds().Dy() != self.tileSize {
		log.Fatal(errors.New("Material isnt the right dimension"))
	}

	var side int
	var order []int
	if self.mode == Hexagonal {
		side = 8
		order = nil
	} else {
		side = 4
		order = []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}
	}
	num := side * side
	if mask.Bounds().Dx() != side*self.tileSize || mask.Bounds().Dy() != side*self.tileSize {
		log.Fatal(errors.New("Mask isnt the right dimension"))
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

		idx := i
		if order != nil {
			idx = order[i]
		}
		newMaterial[idx] = finalImage
	}
	self.materials = append(self.materials, newMaterial)
	self.atlas = nil
}

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

	var offsetX, offsetY float64
	if self.mode == Isometric {
		offsetX = float64(self.gridHeight+1) * float64(self.tileSize) / 2.0
		offsetY = 0
	}

	var tl, tr, bl, br TileType
	var matType TileType

	if self.mode == Hexagonal {
		for x := 0; x < self.gridWidth; x++ {
			for y := 0; y < self.gridHeight; y++ {
				xPos := float64(x) * 1.5 * float64(self.tileSize)
				yPos := float64(y) * math.Sqrt(3) * float64(self.tileSize)
				if x%2 == 1 {
					yPos += math.Sqrt(3) / 2 * float64(self.tileSize)
				}

				var deltas [6]struct {
					dx, dy int
				}
				if x%2 == 0 {
					deltas = [6]struct {
						dx, dy int
					}{{1, 0}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}, {0, 1}}
				} else {
					deltas = [6]struct {
						dx, dy int
					}{{1, 0}, {1, -1}, {0, -1}, {-1, 0}, {0, 1}, {1, 1}}
				}

				var neighbors [6]TileType
				for i := 0; i < 6; i++ {
					nx := x + deltas[i].dx
					ny := y + deltas[i].dy
					if nx >= 0 && nx < self.gridWidth && ny >= 0 && ny < self.gridHeight {
						neighbors[i] = self.worldGrid[nx][ny]
					} else {
						neighbors[i] = self.defaultMaterial
					}
				}

				for i := range matTypeMask {
					matTypeMask[i] = false
				}
				for _, n := range neighbors {
					matTypeMask[n] = true
				}

				for i := 0; i < len(self.materials); i++ {
					if !matTypeMask[i] {
						continue
					}
					matType = TileType(i)
					bitmask := 0
					for j := 0; j < 6; j++ {
						if neighbors[j] == matType || neighbors[j] > matType {
							bitmask |= 1 << uint(j)
						}
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
	} else {
		for x := 0; x < self.gridWidth+1; x++ {
			for y := 0; y < self.gridHeight+1; y++ {
				var xPos, yPos float64
				if self.mode == Isometric {
					xPos = float64((x-y)*self.tileSize/2) + offsetX
					yPos = float64((x+y)*self.tileSize/4) + offsetY
				} else {
					xPos = float64(x * self.tileSize)
					yPos = float64(y * self.tileSize)
				}

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
					matType = TileType(i)
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
	}

	img.DrawTriangles(verts, indices, self.atlas, nil)
}

func (self DualGrid) WorldSize() (int, int) {
	ts := self.tileSize
	switch self.mode {
	case Hexagonal:
		width := int(float64(self.gridWidth)*1.5*float64(ts)) + ts
		height := int(float64(self.gridHeight)*math.Sqrt(3)*float64(ts)+math.Sqrt(3)/2*float64(ts)) + ts
		return width, height
	case Isometric:
		w := (self.gridWidth + self.gridHeight + 2) * ts / 2
		h := (self.gridWidth+self.gridHeight+2)*ts/4 + ts/2
		return w, h
	default:
		return (self.gridWidth + 1) * ts, (self.gridHeight + 1) * ts
	}
}

func hexRound(h [3]float64) [3]int {
	q := math.Round(h[0])
	r := math.Round(h[1])
	s := math.Round(h[2])
	qd := math.Abs(q - h[0])
	rd := math.Abs(r - h[1])
	sd := math.Abs(s - h[2])
	if qd > rd && qd > sd {
		q = -r - s
	} else if rd > sd {
		r = -q - s
	} else {
		s = -q - r
	}
	return [3]int{int(q), int(r), int(s)}
}

func (self DualGrid) WorldCoordFromScreen(sx, sy int) (wx, wy int, ok bool) {
	tsf := float64(self.tileSize)
	sxf := float64(sx)
	syf := float64(sy)
	switch self.mode {
	case Square:
		wx = (sx - self.tileSize/2) / self.tileSize
		wy = (sy - self.tileSize/2) / self.tileSize
	case Isometric:
		offsetX := float64(self.gridHeight+1) * tsf / 2.0
		adjX := sxf - offsetX
		adjY := syf
		col := (adjX*2/tsf + adjY*4/tsf) / 2
		row := (adjY*4/tsf - adjX*2/tsf) / 2
		wx = int(math.Round(col) - 1)
		wy = int(math.Round(row) - 1)
	case Hexagonal:
		q := (2.0 / 3.0 * sxf) / tsf
		r := (-1.0/3.0*sxf + math.Sqrt(3)/3.0*syf) / tsf
		s := -q - r
		cube := [3]float64{q, r, s}
		rounded := hexRound(cube)
		wx = rounded[0]
		wy = rounded[1]
	}
	if wx < 0 || wx >= self.gridWidth || wy < 0 || wy >= self.gridHeight {
		return 0, 0, false
	}
	return wx, wy, true
}
