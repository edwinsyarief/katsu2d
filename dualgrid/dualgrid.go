// dualgrid.go

package dualgrid

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type Grid [][]TileType

func NewGrid(width, height int) Grid {
	grid := make([][]TileType, width)
	for x := range grid {
		grid[x] = make([]TileType, height)
	}
	return grid
}

func NewGridWithValue(width, height int, value TileType) Grid {
	grid := make([][]TileType, width)
	for x := range grid {
		grid[x] = make([]TileType, height)
		for y := range grid[x] {
			grid[x][y] = value
		}
	}
	return grid
}

type TileType uint8

type Material [16]*ebiten.Image

type DualGrid struct {
	Materials       []Material
	DefaultMaterial TileType
	TileSize        int
	WorldGrid       Grid
	GridWidth       int
	GridHeight      int
	atlas           *ebiten.Image
}

func NewDualGrid(width, height, tileSize int, defaultMaterial TileType) DualGrid {
	return DualGrid{
		Materials:       []Material{},
		DefaultMaterial: defaultMaterial,
		TileSize:        tileSize,
		WorldGrid:       NewGridWithValue(width, height, defaultMaterial),
		GridWidth:       width,
		GridHeight:      height,
	}
}

func (g *DualGrid) AddMaterial(material, mask *ebiten.Image) {
	if mask == nil && (material.Bounds().Dx() != 4*g.TileSize || material.Bounds().Dy() != 4*g.TileSize) {
		log.Fatal(fmt.Errorf("Material without mask must have the correct size: %dx%d", 4*g.TileSize, 4*g.TileSize))
	}
	if material.Bounds().Dx() != g.TileSize || material.Bounds().Dy() != g.TileSize {
		log.Fatal(errors.New("Material isnt the right dimension"))
	}
	if mask.Bounds().Dx() != 4*g.TileSize || mask.Bounds().Dy() != 4*g.TileSize {
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

	order := []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}

	newMaterial := Material{}
	tempImage := ebiten.NewImage(g.TileSize, g.TileSize)
	for i := range 16 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		finalImage := ebiten.NewImage(g.TileSize, g.TileSize)

		if mask != nil {
			tempImage.DrawImage(material, opts)
			tempImage.DrawImage(mask.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image), multiplyOpts)
		} else {
			tempImage.DrawImage(material.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image), opts)
		}

		finalImage.DrawImage(tempImage, opts)

		newMaterial[order[i]] = finalImage
	}
	g.Materials = append(g.Materials, newMaterial)
	g.atlas = nil
}

func (g *DualGrid) buildAtlas() {
	numMats := len(g.Materials)
	if numMats == 0 {
		return
	}
	atlasW := 16 * g.TileSize
	atlasH := numMats * g.TileSize
	g.atlas = ebiten.NewImage(atlasW, atlasH)
	opts := &ebiten.DrawImageOptions{}
	for m, mat := range g.Materials {
		for b := 0; b < 16; b++ {
			opts.GeoM.Reset()
			opts.GeoM.Translate(float64(b*g.TileSize), float64(m*g.TileSize))
			g.atlas.DrawImage(mat[b], opts)
		}
	}
}

func (g DualGrid) DrawTo(img *ebiten.Image) {
	if g.atlas == nil {
		g.buildAtlas()
	}
	if g.atlas == nil {
		return
	}

	img.Fill(color.Transparent)

	verts := make([]ebiten.Vertex, 0, (g.GridWidth+1)*(g.GridHeight+1)*16)
	indices := make([]uint16, 0, (g.GridWidth+1)*(g.GridHeight+1)*24)
	var idx uint16

	var xPos, yPos float64
	var tl, tr, bl, br TileType

	matTypeMask := make([]bool, len(g.Materials))
	ts := float32(g.TileSize)

	for x := 0; x < g.GridWidth+1; x++ {
		xPos = float64(x * g.TileSize)
		for y := 0; y < g.GridHeight+1; y++ {
			yPos = float64(y * g.TileSize)

			tl = g.DefaultMaterial
			tr = g.DefaultMaterial
			bl = g.DefaultMaterial
			br = g.DefaultMaterial

			if x >= 1 && y >= 1 {
				tl = g.WorldGrid[x-1][y-1]
			}
			if x < g.GridWidth && y >= 1 {
				tr = g.WorldGrid[x][y-1]
			}
			if x >= 1 && y < g.GridHeight {
				bl = g.WorldGrid[x-1][y]
			}
			if x < g.GridWidth && y < g.GridHeight {
				br = g.WorldGrid[x][y]
			}

			for i := range matTypeMask {
				matTypeMask[i] = false
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			for i := range len(g.Materials) {
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

				srcX := float32(bitmask * g.TileSize)
				srcY := float32(int(matType) * g.TileSize)
				maxSrcX := srcX + ts
				maxSrcY := srcY + ts

				dx := float32(xPos)
				dy := float32(yPos)
				dx2 := dx + ts
				dy2 := dy + ts

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

	img.DrawTriangles(verts, indices, g.atlas, nil)
}
