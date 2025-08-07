package graphics

import (
	"image/color"
	"katsu2d"

	"katsu2d/constants"

	"github.com/hajimehoshi/ebiten/v2"
)

// Batcher handles drawing multiple primitives with a single draw call.
type Batcher struct {
	buffer         *VertexBuffer
	whitePixel     *ebiten.Image
	currentTexture *ebiten.Image
}

// NewBatcher creates a new Batcher instance.
func NewBatcher() *Batcher {
	// Create a 1x1 white pixel image for colored rendering
	whitePixel := ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)

	return &Batcher{
		buffer:     NewVertexBuffer(constants.MaxVertices),
		whitePixel: whitePixel,
	}
}

// Begin prepares the batcher for a new batch.
func (self *Batcher) Begin(texture *ebiten.Image) {
	self.buffer.Clear()
	self.currentTexture = texture
	if self.currentTexture == nil {
		self.currentTexture = self.whitePixel
	}
}

// AddQuad adds a simple quad to the current batch.
func (self *Batcher) AddQuad(x, y, w, h float64, u1, v1, u2, v2 float64, r, g, b, a float64) {
	// Check if buffer needs flushing
	if self.buffer.IsFull() {
		return
	}
	self.buffer.AddQuad(x, y, w, h, u1, v1, u2, v2, r, g, b, a)
}

// AddTransformedQuad adds a quad with scaling and rotation to the current batch.
// The u1, v1, u2, v2 represent normalized texture coordinates (0.0 to 1.0).
func (self *Batcher) AddTransformedQuad(
	x, y, w, h, scaleX, scaleY, rotation float64,
	u1, v1, u2, v2 float64,
	r, g, b, a float64,
) {
	if self.buffer.IsFull() {
		return
	}

	// Calculate half width and height
	halfW := (w * scaleX) / 2
	halfH := (h * scaleY) / 2

	// Define the local vertices of the quad
	p1 := katsu2d.V(-halfW, -halfH)
	p2 := katsu2d.V(halfW, -halfH)
	p3 := katsu2d.V(halfW, halfH)
	p4 := katsu2d.V(-halfW, halfH)

	// Apply rotation to each local vertex
	p1 = p1.Rotate(rotation)
	p2 = p2.Rotate(rotation)
	p3 = p3.Rotate(rotation)
	p4 = p4.Rotate(rotation)

	// Translate the rotated vertices to the world position
	p1 = p1.Add(katsu2d.V(x, y))
	p2 = p2.Add(katsu2d.V(x, y))
	p3 = p3.Add(katsu2d.V(x, y))
	p4 = p4.Add(katsu2d.V(x, y))

	baseIndex := uint16(len(self.buffer.vertices))

	// Add 4 transformed vertices to the buffer
	self.buffer.vertices = append(self.buffer.vertices,
		// Top-left
		Vertex{X: p1.X, Y: p1.Y, U: u1, V: v1, R: r, G: g, B: b, A: a},
		// Top-right
		Vertex{X: p2.X, Y: p2.Y, U: u2, V: v1, R: r, G: g, B: b, A: a},
		// Bottom-right
		Vertex{X: p3.X, Y: p3.Y, U: u2, V: v2, R: r, G: g, B: b, A: a},
		// Bottom-left
		Vertex{X: p4.X, Y: p4.Y, U: u1, V: v2, R: r, G: g, B: b, A: a},
	)

	// Add 6 indices for 2 triangles (quad)
	self.buffer.indices = append(self.buffer.indices,
		baseIndex, baseIndex+1, baseIndex+2, // First triangle
		baseIndex, baseIndex+2, baseIndex+3, // Second triangle
	)
}

// Flush draws the current batch to the screen.
func (self *Batcher) Flush(screen *ebiten.Image) {
	if screen == nil || self.buffer.IsEmpty() {
		return
	}

	vertices := self.buffer.GetVertices()
	indices := self.buffer.GetIndices()

	if len(vertices) == 0 || len(indices) == 0 {
		return
	}

	// Convert our vertex format to Ebitengine's vertex format
	ebitenVertices := make([]ebiten.Vertex, len(vertices))
	for i, v := range vertices {
		ebitenVertices[i] = v.ToEbitenVertex()
		ebitenVertices[i].SrcX = float32(v.U) * float32(self.currentTexture.Bounds().Dx())
		ebitenVertices[i].SrcY = float32(v.V) * float32(self.currentTexture.Bounds().Dy())
	}

	// Draw using Ebitengine's DrawTriangles
	opts := &ebiten.DrawTrianglesOptions{}
	opts.Address = ebiten.AddressClampToZero
	opts.Filter = ebiten.FilterNearest
	screen.DrawTriangles(ebitenVertices, indices, self.currentTexture, opts)
}

// End flushes the remaining batch and clears the buffer.
func (self *Batcher) End(screen *ebiten.Image) {
	self.Flush(screen)
	self.buffer.Clear()
}

// IsFull checks if the buffer is full.
func (self *Batcher) IsFull() bool {
	return self.buffer.IsFull()
}

// IsEmpty checks if the buffer is empty.
func (self *Batcher) IsEmpty() bool {
	return self.buffer.IsEmpty()
}
