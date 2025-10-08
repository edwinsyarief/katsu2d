package katsu2d

import (
	"image/color"
	"maps"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxVertices = 65534
)

// BatchRenderer batches draw calls for performance.
type BatchRenderer struct {
	screen        *ebiten.Image
	currentImage  *ebiten.Image
	currentShader *ebiten.Shader
	dOpt          *ebiten.DrawTrianglesOptions
	dsOpt         *ebiten.DrawTrianglesShaderOptions
	vertices      []ebiten.Vertex
	indices       []uint16
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
func (self *BatchRenderer) Begin(screen *ebiten.Image, shader *ebiten.Shader) {
	self.screen = screen
	self.currentShader = shader
	self.dsOpt = nil
	self.dOpt = nil
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

type DrawOptions func(*ebiten.DrawTrianglesOptions)

func WithRenderBlendMode(mode ebiten.Blend) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.Blend = mode
	}
}
func WithRenderColorScaleMode(mode ebiten.ColorScaleMode) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.ColorScaleMode = mode
	}
}
func WithRenderFilterMode(mode ebiten.Filter) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.Filter = mode
	}
}
func WithRenderAddressMode(mode ebiten.Address) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.Address = mode
	}
}
func WithRenderFillRule(rule ebiten.FillRule) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.FillRule = rule
	}
}
func WithRenderAntiAlias(aa bool) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.AntiAlias = aa
	}
}
func WithRenderDisableMipmaps(disable bool) DrawOptions {
	return func(opt *ebiten.DrawTrianglesOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesOptions{}
		}
		opt.DisableMipmaps = disable
	}
}

// SetDrawOptions sets the draw options for the batch renderer.
func (self *BatchRenderer) SetDrawOptions(opts ...DrawOptions) {
	for _, opt := range opts {
		opt(self.dOpt)
	}
}

type DrawShaderOptions func(*ebiten.DrawTrianglesShaderOptions)

func WithShaderRenderBlendMode(mode ebiten.Blend) DrawShaderOptions {
	return func(opt *ebiten.DrawTrianglesShaderOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesShaderOptions{}
		}
		opt.Blend = mode
	}
}
func WithShaderRenderFillRule(rule ebiten.FillRule) DrawShaderOptions {
	return func(opt *ebiten.DrawTrianglesShaderOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesShaderOptions{}
		}
		opt.FillRule = rule
	}
}
func WithShaderRenderAntiAlias(aa bool) DrawShaderOptions {
	return func(opt *ebiten.DrawTrianglesShaderOptions) {
		if opt == nil {
			opt = &ebiten.DrawTrianglesShaderOptions{}
		}
		opt.AntiAlias = aa
	}
}
func WithShaderRenderImages(images ...*ebiten.Image) DrawShaderOptions {
	return func(opt *ebiten.DrawTrianglesShaderOptions) {
		if len(images) == 0 {
			return
		}
		if opt == nil {
			opt = &ebiten.DrawTrianglesShaderOptions{}
		}
		copy(opt.Images[:], images)
	}
}
func WithShaderRenderUniforms(uniforms map[string]interface{}) DrawShaderOptions {
	return func(opt *ebiten.DrawTrianglesShaderOptions) {
		if uniforms == nil {
			return
		}
		if opt == nil {
			opt = &ebiten.DrawTrianglesShaderOptions{}
		}
		maps.Copy(opt.Uniforms, uniforms)
	}
}

// SetDrawShaderOptions sets the shader options for the batch renderer.
func (self *BatchRenderer) SetDrawShaderOptions(opt ...DrawShaderOptions) {
	if self.currentShader == nil {
		panic("Set a shader using Begin before setting shader options")
	}
	for _, o := range opt {
		o(self.dsOpt)
	}
}

// Flush draws the current batch.
func (self *BatchRenderer) Flush() {
	if len(self.vertices) == 0 {
		return
	}
	if self.currentShader != nil {
		if self.dsOpt == nil {
			panic("SetDrawShaderOptions must be called before Flush when using a shader")
		}
		self.screen.DrawTrianglesShader(self.vertices, self.indices, self.currentShader, self.dsOpt)
		self.vertices = self.vertices[:0]
		self.indices = self.indices[:0]
		self.dsOpt = nil
		self.currentImage = nil
	} else {
		self.screen.DrawTriangles(self.vertices, self.indices, self.currentImage, self.dOpt)
		self.vertices = self.vertices[:0]
		self.indices = self.indices[:0]
		self.dOpt = nil
		self.currentImage = nil
	}
}

// AddCustomMeshes adds custom vertices and indices to the batch.
func (self *BatchRenderer) AddCustomMeshes(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image) {
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
		v.DstX = AdjustDestinationPixel(v.DstX)
		v.DstY = AdjustDestinationPixel(v.DstY)
	}
	self.vertices = append(self.vertices, verts...)
	for _, i := range inds {
		self.indices = append(self.indices, uint16(offset)+i)
	}
}

// AddQuad draws a quad (sprite) with specified source rectangle and destination size.
func (self *BatchRenderer) AddQuad(
	pos, scale Vector, rotation float64, // transform parameters
	img *ebiten.Image, clr color.RGBA, // image & color parameters
	srcMinX, srcMinY, srcMaxX, srcMaxY float32, // source rectangle
	// destination size
	destW, destH float64) {
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
	p0 := V(float64(left), float64(top))
	p1 := V(float64(right), float64(top))
	p2 := V(float64(right), float64(bottom))
	p3 := V(float64(left), float64(bottom))
	if rotation != 0 {
		srcOffset := V(srcProjMinX, srcProjMinY)
		p0 = p0.RotateAround(srcOffset, rotation)
		p1 = p1.RotateAround(srcOffset, rotation)
		p2 = p2.RotateAround(srcOffset, rotation)
		p3 = p3.RotateAround(srcOffset, rotation)
	}
	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255
	vertIndex := len(self.vertices)
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: AdjustDestinationPixel(float32(p0.X)), DstY: AdjustDestinationPixel(float32(p0.Y)), SrcX: srcMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: AdjustDestinationPixel(float32(p1.X)), DstY: AdjustDestinationPixel(float32(p1.Y)), SrcX: srcMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: AdjustDestinationPixel(float32(p2.X)), DstY: AdjustDestinationPixel(float32(p2.Y)), SrcX: srcMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: AdjustDestinationPixel(float32(p3.X)), DstY: AdjustDestinationPixel(float32(p3.Y)), SrcX: srcMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	self.indices = append(self.indices, uint16(vertIndex), uint16(vertIndex+1), uint16(vertIndex+2), uint16(vertIndex), uint16(vertIndex+2), uint16(vertIndex+3))
}

// AddTriangleStrip draws a triangle strip.
func (self *BatchRenderer) AddTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
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
		v.DstX = AdjustDestinationPixel(v.DstX)
		v.DstY = AdjustDestinationPixel(v.DstY)
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
