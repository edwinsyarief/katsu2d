package katsu2d

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type canvas struct {
	buffer                  *ebiten.Image
	renderers               []func(*ebiten.Image)
	topLeft, scale          Vector
	width, height           int
	stretched, pixelPerfect bool
	filter                  scalingFilter
}

func newCanvas(width, height int, stretched, pixelPerfect bool) *canvas {
	result := &canvas{
		width:        width,
		height:       height,
		stretched:    stretched,
		pixelPerfect: pixelPerfect,
		buffer:       ebiten.NewImage(width, height),
		renderers:    make([]func(*ebiten.Image), 0),
	}
	result.SetFilter(_AASamplingSoft)
	return result
}
func (self *canvas) SetFilter(filter scalingFilter) {
	self.filter = filter
	if shaders[filter] == nil {
		compileShader(filter)
	}
}
func (self *canvas) AddRenderer(renderer func(*ebiten.Image)) {
	self.renderers = append(self.renderers, renderer)
}
func (self *canvas) GetTopLeft() Vector {
	return self.topLeft
}
func (self *canvas) GetScale() Vector {
	return self.scale
}
func (self *canvas) Resize(width, height int) {
	scale := V(
		float64(width)/float64(self.width),
		float64(height)/float64(self.height),
	)
	if scale.X < scale.Y {
		scale.Y = scale.X
	} else {
		scale.X = scale.Y
	}
	if self.pixelPerfect {
		scale = scale.Floor()
	}
	self.scale = scale
	self.topLeft = V2(0)
	if !self.stretched {
		self.topLeft = V(
			(float64(width)-float64(self.width)*self.scale.X)/2,
			(float64(height)-float64(self.height)*self.scale.Y)/2,
		)
	}
}
func (self *canvas) ScaleCursorPosition() Vector {
	x, y := ebiten.CursorPosition()
	return V(
		(float64(x)-self.topLeft.X)/self.scale.X,
		(float64(y)-self.topLeft.Y)/self.scale.Y,
	)
}
func (self *canvas) Draw(source, destination *ebiten.Image) {
	p0 := self.topLeft
	p1 := p0.Add(V(self.scale.X*float64(self.width), 0))
	p2 := p0.Add(V(self.scale.X*float64(self.width), self.scale.Y*float64(self.height)))
	p3 := p0.Add(V(0, self.scale.Y*float64(self.height)))
	shaderVertices[0].DstX = float32(p0.X)
	shaderVertices[0].DstY = float32(p0.Y)
	shaderVertices[1].DstX = float32(p1.X)
	shaderVertices[1].DstY = float32(p1.Y)
	shaderVertices[2].DstX = float32(p2.X)
	shaderVertices[2].DstY = float32(p2.Y)
	shaderVertices[3].DstX = float32(p3.X)
	shaderVertices[3].DstY = float32(p3.Y)
	sourceBounds := source.Bounds()
	shaderVertices[0].SrcX = float32(sourceBounds.Min.X)
	shaderVertices[0].SrcY = float32(sourceBounds.Min.Y)
	shaderVertices[1].SrcX = float32(sourceBounds.Max.X)
	shaderVertices[1].SrcY = shaderVertices[0].SrcY
	shaderVertices[2].SrcX = shaderVertices[1].SrcX
	shaderVertices[2].SrcY = float32(sourceBounds.Max.Y)
	shaderVertices[3].SrcX = shaderVertices[0].SrcX
	shaderVertices[3].SrcY = shaderVertices[2].SrcY
	shaderOpts.Images[0] = source
	shaderOpts.Uniforms["SourceRelativeTextureUnitX"] = float32(float64(self.width) / float64(destination.Bounds().Dx()))
	shaderOpts.Uniforms["SourceRelativeTextureUnitY"] = float32(float64(self.height) / float64(destination.Bounds().Dy()))
	destination.DrawTrianglesShader(
		shaderVertices, shaderVertIndices,
		shaders[self.filter], &shaderOpts,
	)
	shaderOpts.Images[0] = nil
}
