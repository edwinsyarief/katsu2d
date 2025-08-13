package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- SYSTEMS ---

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Update(*Engine, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Draw(*Engine, *BatchRenderer)
}

// --- RENDERER ---

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
	self.currentImage = img
	offset := len(self.vertices)
	self.vertices = append(self.vertices, verts...)
	for _, i := range inds {
		self.indices = append(self.indices, uint16(offset)+i)
	}
}

// DrawQuad draws a quad (sprite).
func (self *BatchRenderer) DrawQuad(pos, scale, offset, origin ebimath.Vector, rotation float64, img *ebiten.Image, clr color.RGBA) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img

	w, h := float64(img.Bounds().Dx())*scale.X, float64(img.Bounds().Dy())*scale.Y
	ox, oy := float64(origin.X), float64(origin.Y)

	p0 := ebimath.V(-ox, -oy)
	p1 := ebimath.V(w-ox, -oy)
	p2 := ebimath.V(w-ox, h-oy)
	p3 := ebimath.V(-ox, h-oy)

	if rotation != 0 {
		c, s := math.Cos(rotation), math.Sin(rotation)
		p0 = ebimath.V(p0.X*c-p0.Y*s, p0.X*s+p0.Y*c)
		p1 = ebimath.V(p1.X*c-p1.Y*s, p1.X*s+p1.Y*c)
		p2 = ebimath.V(p2.X*c-p2.Y*s, p2.X*s+p2.Y*c)
		p3 = ebimath.V(p3.X*c-p3.Y*s, p3.X*s+p3.Y*c)
	}

	p0 = p0.Add(pos)
	p1 = p1.Add(pos)
	p2 = p2.Add(pos)
	p3 = p3.Add(pos)

	minX, minY := float32(img.Bounds().Min.X), float32(img.Bounds().Min.Y)
	maxX, maxY := float32(img.Bounds().Max.X), float32(img.Bounds().Max.Y)

	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255

	vertIndex := len(self.vertices)
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: float32(p0.X), DstY: float32(p0.Y), SrcX: minX, SrcY: minY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p1.X), DstY: float32(p1.Y), SrcX: maxX, SrcY: minY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p2.X), DstY: float32(p2.Y), SrcX: maxX, SrcY: maxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p3.X), DstY: float32(p3.Y), SrcX: minX, SrcY: maxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	self.indices = append(self.indices, uint16(vertIndex), uint16(vertIndex+1), uint16(vertIndex+2), uint16(vertIndex), uint16(vertIndex+2), uint16(vertIndex+3))
}

// DrawTriangleStrip draws a triangle strip.
func (self *BatchRenderer) DrawTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)
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

// DrawText draws text, flushing the batch first to maintain render order.
// This is necessary because text.Draw is a separate operation from DrawTriangles.
func (self *BatchRenderer) DrawText(txt *Text, transform *ebimath.Transform) {
	self.Flush()
	txt.Draw(transform, self.screen)
}
