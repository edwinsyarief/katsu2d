package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxVertices = 65532
)

// BatchRenderer batches 2D quad, mesh, and triangle-strip draw calls into
// minimal GPU submissions grouped by texture and blend mode.
type BatchRenderer struct {
	screen        *ebiten.Image
	currentImage  *ebiten.Image
	currentBlend  ebiten.Blend
	drawOpts      ebiten.DrawTrianglesOptions
	vertices      []ebiten.Vertex
	indices       []uint16
	viewMatrix    Matrix
	va, vb, vc    float64
	vd, vtx, vty  float64
	hasViewMatrix bool
}

// NewBatchRenderer creates a new BatchRenderer with pre-allocated vertex and index buffers.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{
		currentBlend: ebiten.BlendSourceOver,
		vertices:     make([]ebiten.Vertex, 0, 4096),
		indices:      make([]uint16, 0, 6144),
	}
}

// GetScreen returns the current target image being rendered to.
func (r *BatchRenderer) GetScreen() *ebiten.Image {
	return r.screen
}

// SetViewMatrix flushes any pending geometry and applies m as the active camera view transform.
func (r *BatchRenderer) SetViewMatrix(m Matrix) {
	if len(r.vertices) > 0 {
		r.Flush()
	}
	r.viewMatrix = m
	r.hasViewMatrix = true
	r.va = m.Element(0, 0)
	r.vb = m.Element(0, 1)
	r.vtx = m.Element(0, 2)
	r.vc = m.Element(1, 0)
	r.vd = m.Element(1, 1)
	r.vty = m.Element(1, 2)
}

// ResetViewMatrix flushes any pending geometry if a view matrix is active and clears it.
func (r *BatchRenderer) ResetViewMatrix() {
	if r.hasViewMatrix && len(r.vertices) > 0 {
		r.Flush()
	}
	r.hasViewMatrix = false
	r.viewMatrix.Reset()
}

// GetViewMatrix returns the active view matrix and whether one is currently set.
func (r *BatchRenderer) GetViewMatrix() (Matrix, bool) {
	return r.viewMatrix, r.hasViewMatrix
}

// HasViewMatrix reports whether a view matrix is currently active.
func (r *BatchRenderer) HasViewMatrix() bool {
	return r.hasViewMatrix
}

// Begin prepares the renderer for a new frame on the given screen image.
func (r *BatchRenderer) Begin(screen *ebiten.Image) {
	r.screen = screen
	r.vertices = r.vertices[:0]
	r.indices = r.indices[:0]
	r.currentImage = nil
	r.currentBlend = ebiten.BlendSourceOver
	r.hasViewMatrix = false
	r.viewMatrix.Reset()
}

// Flush submits all accumulated geometry to the target image and resets the batch.
func (r *BatchRenderer) Flush() {
	if len(r.vertices) == 0 || r.currentImage == nil {
		r.vertices = r.vertices[:0]
		r.indices = r.indices[:0]
		r.currentImage = nil
		return
	}
	r.drawOpts.Blend = r.currentBlend
	r.screen.DrawTriangles(r.vertices, r.indices, r.currentImage, &r.drawOpts)
	r.vertices = r.vertices[:0]
	r.indices = r.indices[:0]
	r.currentImage = nil
}

func (r *BatchRenderer) ensureBatch(img *ebiten.Image, blend ebiten.Blend, vertexCount int) {
	if (r.currentImage != nil && (img != r.currentImage || blend != r.currentBlend)) ||
		len(r.vertices)+vertexCount >= maxVertices {
		r.Flush()
	}
	r.currentImage = img
	r.currentBlend = blend
}

// AddCustomMeshes appends arbitrary vertices and triangle indices using BlendSourceOver.
func (r *BatchRenderer) AddCustomMeshes(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image) {
	r.AddCustomMeshesWithBlend(verts, inds, img, ebiten.BlendSourceOver)
}

// AddCustomMeshesWithBlend appends arbitrary vertices and triangle indices with a custom blend mode.
func (r *BatchRenderer) AddCustomMeshesWithBlend(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image, blend ebiten.Blend) {
	if len(verts) == 0 || len(inds) == 0 || img == nil {
		return
	}
	r.ensureBatch(img, blend, len(verts))
	offset := uint16(len(r.vertices))
	if r.hasViewMatrix {
		va, vb, vtx := r.va, r.vb, r.vtx
		vc, vd, vty := r.vc, r.vd, r.vty
		for _, v := range verts {
			x, y := float64(v.DstX), float64(v.DstY)
			v.DstX = float32(va*x + vb*y + vtx)
			v.DstY = float32(vc*x + vd*y + vty)
			r.vertices = append(r.vertices, v)
		}
	} else {
		r.vertices = append(r.vertices, verts...)
	}
	for _, idx := range inds {
		r.indices = append(r.indices, offset+idx)
	}
}

// AddQuad draws a transformed quad using BlendSourceOver.
// Transformation order matches Transform.Matrix(): Scale -> Offset pivot -> Rotate -> Translate(pos - origin).
func (r *BatchRenderer) AddQuad(
	pos, offset, origin, scale Vector, rotation float64,
	img *ebiten.Image, clr color.RGBA,
	srcMinX, srcMinY, srcMaxX, srcMaxY float32,
	destW, destH float64,
) {
	r.AddQuadWithBlend(
		pos, offset, origin, scale, rotation,
		img, clr, ebiten.BlendSourceOver,
		srcMinX, srcMinY, srcMaxX, srcMaxY,
		destW, destH,
	)
}

// AddQuadWithBlend draws a transformed quad with a custom blend mode.
// Transformation order matches Transform.Matrix(): Scale -> Offset pivot -> Rotate -> Translate(pos - origin).
func (r *BatchRenderer) AddQuadWithBlend(
	pos, offset, origin, scale Vector, rotation float64,
	img *ebiten.Image, clr color.RGBA, blend ebiten.Blend,
	srcMinX, srcMinY, srcMaxX, srcMaxY float32,
	destW, destH float64,
) {
	if img == nil {
		return
	}
	r.ensureBatch(img, blend, 4)

	tx := pos.X - origin.X
	ty := pos.Y - origin.Y

	// Local corners after scaling and subtracting the scaled pivot offset.
	x0 := -offset.X * scale.X
	y0 := -offset.Y * scale.Y
	x1 := (destW - offset.X) * scale.X
	y1 := (destH - offset.Y) * scale.Y

	var p0x, p0y, p1x, p1y, p2x, p2y, p3x, p3y float64
	if rotation == 0 {
		p0x, p0y = x0+tx, y0+ty
		p1x, p1y = x1+tx, y0+ty
		p2x, p2y = x1+tx, y1+ty
		p3x, p3y = x0+tx, y1+ty
	} else {
		sin, cos := math.Sincos(rotation)
		x0c, x0s := x0*cos, x0*sin
		x1c, x1s := x1*cos, x1*sin
		y0c, y0s := y0*cos, y0*sin
		y1c, y1s := y1*cos, y1*sin

		p0x, p0y = x0c-y0s+tx, x0s+y0c+ty
		p1x, p1y = x1c-y0s+tx, x1s+y0c+ty
		p2x, p2y = x1c-y1s+tx, x1s+y1c+ty
		p3x, p3y = x0c-y1s+tx, x0s+y1c+ty
	}

	if r.hasViewMatrix {
		va, vb, vtx := r.va, r.vb, r.vtx
		vc, vd, vty := r.vc, r.vd, r.vty
		p0x, p0y = va*p0x+vb*p0y+vtx, vc*p0x+vd*p0y+vty
		p1x, p1y = va*p1x+vb*p1y+vtx, vc*p1x+vd*p1y+vty
		p2x, p2y = va*p2x+vb*p2y+vtx, vc*p2x+vd*p2y+vty
		p3x, p3y = va*p3x+vb*p3y+vtx, vc*p3x+vd*p3y+vty
	}

	const inv255 = float32(1.0 / 255.0)
	cr := float32(clr.R) * inv255
	cg := float32(clr.G) * inv255
	cb := float32(clr.B) * inv255
	ca := float32(clr.A) * inv255

	vertIndex := uint16(len(r.vertices))
	r.vertices = append(r.vertices,
		ebiten.Vertex{DstX: float32(p0x), DstY: float32(p0y), SrcX: srcMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p1x), DstY: float32(p1y), SrcX: srcMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p2x), DstY: float32(p2y), SrcX: srcMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(p3x), DstY: float32(p3y), SrcX: srcMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	r.indices = append(r.indices,
		vertIndex, vertIndex+1, vertIndex+2,
		vertIndex, vertIndex+2, vertIndex+3,
	)
}

// AddTriangleStrip draws a triangle strip using BlendSourceOver.
func (r *BatchRenderer) AddTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
	if len(verts) < 3 || img == nil {
		return
	}
	r.ensureBatch(img, ebiten.BlendSourceOver, len(verts))
	offset := len(r.vertices)
	if r.hasViewMatrix {
		va, vb, vtx := r.va, r.vb, r.vtx
		vc, vd, vty := r.vc, r.vd, r.vty
		for _, v := range verts {
			x, y := float64(v.DstX), float64(v.DstY)
			v.DstX = float32(va*x + vb*y + vtx)
			v.DstY = float32(vc*x + vd*y + vty)
			r.vertices = append(r.vertices, v)
		}
	} else {
		r.vertices = append(r.vertices, verts...)
	}
	for i := 0; i < len(verts)-2; i++ {
		a := uint16(offset + i)
		b := uint16(offset + i + 1)
		c := uint16(offset + i + 2)
		if i%2 == 0 {
			r.indices = append(r.indices, a, b, c)
		} else {
			r.indices = append(r.indices, a, c, b)
		}
	}
}
