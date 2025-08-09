package graphics

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Vertex represents a single vertex with position, texture coordinates, and color.
type Vertex struct {
	X, Y, Z    float64 // Position
	U, V       float64 // Texture coordinates
	R, G, B, A float64 // Color
}

func (self *Vertex) ToEbitenVertex() ebiten.Vertex {
	return ebiten.Vertex{
		DstX:   float32(self.X),
		DstY:   float32(self.Y),
		SrcX:   float32(self.U),
		SrcY:   float32(self.V),
		ColorR: float32(self.R),
		ColorG: float32(self.G),
		ColorB: float32(self.B),
		ColorA: float32(self.A),
	}
}

var vertexBufferPool = sync.Pool{
	New: func() interface{} {
		return &VertexBuffer{
			vertices: make([]Vertex, 0, defaultCapacity),
			indices:  make([]uint16, 0, defaultCapacity*6/4),
			capacity: defaultCapacity,
		}
	},
}

const defaultCapacity = 1024

// VertexBuffer is a data structure for storing vertices and indices for batching.
type VertexBuffer struct {
	vertices []Vertex
	indices  []uint16
	capacity int
}

// NewVertexBuffer creates a new VertexBuffer with a given capacity.
func NewVertexBuffer(capacity int) *VertexBuffer {
	if capacity <= 0 {
		capacity = defaultCapacity
	}

	vb := vertexBufferPool.Get().(*VertexBuffer)
	if cap(vb.vertices) < capacity {
		// If the pooled buffer is too small, create a new one
		vb.vertices = make([]Vertex, 0, capacity)
		vb.indices = make([]uint16, 0, capacity*6/4)
		vb.capacity = capacity
	} else {
		// Reuse the existing buffer but reset its length
		vb.vertices = vb.vertices[:0]
		vb.indices = vb.indices[:0]
		vb.capacity = capacity
	}
	return vb
}

// Release returns the VertexBuffer to the pool.
func (self *VertexBuffer) Release() {
	self.Clear()
	vertexBufferPool.Put(self)
}

// Clear resets the buffer.
func (self *VertexBuffer) Clear() {
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
}

// AddQuad adds a single quad to the vertex buffer.
func (self *VertexBuffer) AddQuad(x, y, w, h float64, u1, v1, u2, v2 float64, r, g, b, a float64) {
	if len(self.vertices)+4 > self.capacity {
		return // Buffer full
	}

	baseIndex := uint16(len(self.vertices))

	// Add 4 vertices for quad
	self.vertices = append(self.vertices,
		Vertex{X: x, Y: y, Z: 0, U: u1, V: v1, R: r, G: g, B: b, A: a},         // Top-left
		Vertex{X: x + w, Y: y, Z: 0, U: u2, V: v1, R: r, G: g, B: b, A: a},     // Top-right
		Vertex{X: x + w, Y: y + h, Z: 0, U: u2, V: v2, R: r, G: g, B: b, A: a}, // Bottom-right
		Vertex{X: x, Y: y + h, Z: 0, U: u1, V: v2, R: r, G: g, B: b, A: a},     // Bottom-left
	)

	// Add 6 indices for 2 triangles (quad)
	self.indices = append(self.indices,
		baseIndex, baseIndex+1, baseIndex+2, // First triangle
		baseIndex, baseIndex+2, baseIndex+3, // Second triangle
	)
}

// GetVertices returns the stored vertices.
func (self *VertexBuffer) GetVertices() []Vertex {
	return self.vertices
}

// GetIndices returns the stored indices.
func (self *VertexBuffer) GetIndices() []uint16 {
	return self.indices
}

// IsFull checks if the buffer is at capacity.
func (self *VertexBuffer) IsFull(additionalVertices int) bool {
	return len(self.vertices)+additionalVertices >= self.capacity
}

// IsEmpty checks if the buffer is empty.
func (self *VertexBuffer) IsEmpty() bool {
	return len(self.vertices) == 0
}
