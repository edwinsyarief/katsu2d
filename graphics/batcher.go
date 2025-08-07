package graphics

import (
	"image/color"
	"katsu2d"

	"katsu2d/components"
	"katsu2d/constants"

	ebimath "github.com/edwinsyarief/ebi-math"
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
	if self.buffer.IsFull(4) {
		return
	}
	self.buffer.AddQuad(x, y, w, h, u1, v1, u2, v2, r, g, b, a)
}

// AddTransformedQuad adds a quad with scaling and rotation to the current batch.
// The u1, v1, u2, v2 represent normalized texture coordinates (0.0 to 1.0).
func (self *Batcher) AddTransformedQuad(
	transform *components.Transform,
	width, height float64,
	u1, v1, u2, v2 float64,
	r, g, b, a float64,
) {
	if self.buffer.IsFull(4) {
		return
	}

	t := transform.GetTransform()
	if t.GetInitialParentTransform() != nil {
		t = t.GetInitialParentTransform()
	}

	realPos := katsu2d.V2(0).Apply(t.Matrix())
	if !t.Origin().IsZero() {
		realPos = realPos.Sub(t.Origin())
	}

	srcProjMinX := realPos.X
	srcProjMinY := realPos.Y
	srcProjMaxX := srcProjMinX + width*t.Scale().X
	srcProjMaxY := srcProjMinY + height*t.Scale().Y

	left, right := float32(srcProjMinX), float32(srcProjMaxX)
	top, bottom := float32(srcProjMinY), float32(srcProjMaxY)

	p1 := ebimath.V(float64(left), float64(top))
	p2 := ebimath.V(float64(right), float64(top))
	p3 := ebimath.V(float64(right), float64(bottom))
	p4 := ebimath.V(float64(left), float64(bottom))

	if t.Rotation() != 0 {
		srcOffset := ebimath.V(srcProjMinX, srcProjMinY)
		p1 = p1.RotateAround(srcOffset, t.Rotation())
		p2 = p2.RotateAround(srcOffset, t.Rotation())
		p3 = p3.RotateAround(srcOffset, t.Rotation())
		p4 = p4.RotateAround(srcOffset, t.Rotation())
	}

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
		ebitenVertices[i].SrcX = float32(v.U)
		ebitenVertices[i].SrcY = float32(v.V)
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
func (self *Batcher) IsFull(additionalVertices int) bool {
	return self.buffer.IsFull(additionalVertices)
}

// IsEmpty checks if the buffer is empty.
func (self *Batcher) IsEmpty() bool {
	return self.buffer.IsEmpty()
}
