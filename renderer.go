package katsu2d

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxVertices = 65534
)

// BatchRenderer batches draw calls for performance.
type BatchRenderer struct {
	screen       *ebiten.Image
	vertices     []ebiten.Vertex
	indices      []uint16
	currentImage *ebiten.Image
}

// NewBatchRenderer creates a new batch renderer.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{
		vertices: make([]ebiten.Vertex, 0, 4096),
		indices:  make([]uint16, 0, 6144),
	}
}

// GetScreen returns the current screen image being rendered to.
func (self *BatchRenderer) GetScreen() *ebiten.Image {
	return self.screen
}

// Begin prepares the renderer for a new frame.
func (self *BatchRenderer) Begin(screen *ebiten.Image) {
	self.screen = screen
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// Flush draws the current batch.
func (self *BatchRenderer) Flush() {
	if len(self.vertices) == 0 {
		return
	}
	self.screen.DrawTriangles(self.vertices, self.indices, self.currentImage, nil)
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// AddVertices adds custom vertices and indices to the batch.
func (self *BatchRenderer) AddVertices(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}

	totalEstimation := len(self.vertices) + len(verts)
	if totalEstimation >= maxVertices {
		self.Flush()
	}

	self.currentImage = img
	offset := len(self.vertices)

	for _, v := range verts {
		v.DstX = utils.AdjustDestinationPixel(v.DstX)
		v.DstY = utils.AdjustDestinationPixel(v.DstY)
	}

	self.vertices = append(self.vertices, verts...)
	for _, i := range inds {
		self.indices = append(self.indices, uint16(offset)+i)
	}
}

// DrawQuad draws a quad (sprite) with specified source rectangle and destination size.
func (self *BatchRenderer) DrawQuad(pos, scale ebimath.Vector, rotation float64, img *ebiten.Image, clr color.RGBA, srcMinX, srcMinY, srcMaxX, srcMaxY float32, destW, destH float64) {
	totalEstimation := len(self.vertices) + 4
	if totalEstimation >= maxVertices {
		self.Flush()
	}
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img

	srcProjMinX := pos.X
	srcProjMinY := pos.Y
	srcProjMaxX := srcProjMinX + destW*scale.X
	srcProjMaxY := srcProjMinY + destH*scale.Y

	left, right := float32(srcProjMinX), float32(srcProjMaxX)
	top, bottom := float32(srcProjMinY), float32(srcProjMaxY)

	p0 := ebimath.V(float64(left), float64(top))
	p1 := ebimath.V(float64(right), float64(top))
	p2 := ebimath.V(float64(right), float64(bottom))
	p3 := ebimath.V(float64(left), float64(bottom))

	if rotation != 0 {
		srcOffset := ebimath.V(srcProjMinX, srcProjMinY)
		p0 = p0.RotateAround(srcOffset, rotation)
		p1 = p1.RotateAround(srcOffset, rotation)
		p2 = p2.RotateAround(srcOffset, rotation)
		p3 = p3.RotateAround(srcOffset, rotation)
	}

	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255

	vertIndex := len(self.vertices)
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p0.X)), DstY: utils.AdjustDestinationPixel(float32(p0.Y)), SrcX: srcMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p1.X)), DstY: utils.AdjustDestinationPixel(float32(p1.Y)), SrcX: srcMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p2.X)), DstY: utils.AdjustDestinationPixel(float32(p2.Y)), SrcX: srcMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p3.X)), DstY: utils.AdjustDestinationPixel(float32(p3.Y)), SrcX: srcMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	self.indices = append(self.indices, uint16(vertIndex), uint16(vertIndex+1), uint16(vertIndex+2), uint16(vertIndex), uint16(vertIndex+2), uint16(vertIndex+3))
}

// DrawTriangleStrip draws a triangle strip.
func (self *BatchRenderer) DrawTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
	totalEstimation := len(self.vertices) + len(verts)
	if totalEstimation >= maxVertices {
		self.Flush()
	}
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)

	for _, v := range verts {
		v.DstX = utils.AdjustDestinationPixel(v.DstX)
		v.DstY = utils.AdjustDestinationPixel(v.DstY)
	}

	self.vertices = append(self.vertices, verts...)
	for i := 0; i < len(verts)-2; i++ {
		a := uint16(offset + i)
		bb := uint16(offset + i + 1)
		c := uint16(offset + i + 2)
		if i%2 == 0 {
			self.indices = append(self.indices, a, bb, c)
		} else {
			self.indices = append(self.indices, a, c, bb)
		}
	}
}
