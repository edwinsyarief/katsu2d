package graphics

import (
	"image/color"
	"math"

	"katsu2d/constants"

	"github.com/hajimehoshi/ebiten/v2"
)

type Batcher struct {
	buffer         *VertexBuffer
	whitePixel     *ebiten.Image
	currentTexture *ebiten.Image
}

func NewBatcher() *Batcher {
	whitePixel := ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)

	return &Batcher{
		buffer:     NewVertexBuffer(constants.MaxVertices),
		whitePixel: whitePixel,
	}
}

func (self *Batcher) Begin(texture *ebiten.Image) {
	self.buffer.Release()
	self.currentTexture = texture
	if self.currentTexture == nil {
		self.currentTexture = self.whitePixel
	}
}

func (self *Batcher) AddQuad(x, y, w, h float64, u1, v1, u2, v2 float64, r, g, b, a float64) {
	if self.buffer.IsFull(4) {
		return
	}
	self.buffer.AddQuad(x, y, w, h, u1, v1, u2, v2, r, g, b, a)
}

func (self *Batcher) AddTransformedQuad(
	x, y float64,
	width, height float64,
	scaleX, scaleY float64,
	rotation float64,
	u1, v1, u2, v2 float64,
	r, g, b, a float64,
) {
	if self.buffer.IsFull(4) {
		return
	}

	// Calculate half dimensions and center point
	hw, hh := width*scaleX/2, height*scaleY/2
	sin, cos := math.Sin(rotation), math.Cos(rotation)
	cx, cy := x+hw, y+hh

	// Calculate transformed corners
	p1x := cos*(-hw) - sin*(-hh) + cx
	p1y := sin*(-hw) + cos*(-hh) + cy

	p2x := cos*(hw) - sin*(-hh) + cx
	p2y := sin*(hw) + cos*(-hh) + cy

	p3x := cos*(hw) - sin*(hh) + cx
	p3y := sin*(hw) + cos*(hh) + cy

	p4x := cos*(-hw) - sin*(hh) + cx
	p4y := sin*(-hw) + cos*(hh) + cy

	baseIndex := uint16(len(self.buffer.vertices))

	// Add vertices directly
	self.buffer.vertices = append(self.buffer.vertices,
		Vertex{X: p1x, Y: p1y, U: u1, V: v1, R: r, G: g, B: b, A: a}, // Top-left
		Vertex{X: p2x, Y: p2y, U: u2, V: v1, R: r, G: g, B: b, A: a}, // Top-right
		Vertex{X: p3x, Y: p3y, U: u2, V: v2, R: r, G: g, B: b, A: a}, // Bottom-right
		Vertex{X: p4x, Y: p4y, U: u1, V: v2, R: r, G: g, B: b, A: a}, // Bottom-left
	)

	// Add indices for two triangles
	self.buffer.indices = append(self.buffer.indices,
		baseIndex, baseIndex+1, baseIndex+2,
		baseIndex, baseIndex+2, baseIndex+3,
	)
}

func (self *Batcher) AddTriangle(
	p1x, p1y, p2x, p2y, p3x, p3y float64,
	u1, v1, u2, v2, u3, v3 float64,
	r, g, b, a float64,
) {
	if self.buffer.IsFull(3) {
		return
	}

	baseIndex := uint16(len(self.buffer.vertices))

	self.buffer.vertices = append(self.buffer.vertices,
		Vertex{X: p1x, Y: p1y, U: u1, V: v1, R: r, G: g, B: b, A: a},
		Vertex{X: p2x, Y: p2y, U: u2, V: v2, R: r, G: g, B: b, A: a},
		Vertex{X: p3x, Y: p3y, U: u3, V: v3, R: r, G: g, B: b, A: a},
	)

	self.buffer.indices = append(self.buffer.indices,
		baseIndex, baseIndex+1, baseIndex+2,
	)
}

func (self *Batcher) Flush(screen *ebiten.Image) {
	if screen == nil || self.buffer.IsEmpty() {
		return
	}

	vertices := self.buffer.GetVertices()
	indices := self.buffer.GetIndices()

	if len(vertices) == 0 || len(indices) == 0 {
		return
	}

	ebitenVertices := make([]ebiten.Vertex, len(vertices))
	for i, v := range vertices {
		ebitenVertices[i] = v.ToEbitenVertex()
		ebitenVertices[i].SrcX = float32(v.U)
		ebitenVertices[i].SrcY = float32(v.V)
	}

	opts := &ebiten.DrawTrianglesOptions{
		Address: ebiten.AddressClampToZero,
		Filter:  ebiten.FilterNearest,
	}
	screen.DrawTriangles(ebitenVertices, indices, self.currentTexture, opts)
}

func (self *Batcher) End(screen *ebiten.Image) {
	self.Flush(screen)
	self.buffer.Release()
}

func (self *Batcher) IsFull(additionalVertices int) bool {
	return self.buffer.IsFull(additionalVertices)
}

func (self *Batcher) IsEmpty() bool {
	return self.buffer.IsEmpty()
}
