package canvas

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

type Canvas struct {
	width, height           int
	stretched, pixelPerfect bool
	topLeft, scale          ebimath.Vector
	buffer                  *ebiten.Image
	filter                  ScalingFilter
	renderers               []func(*ebiten.Image)
}

func NewCanvas(width, height int, stretched, pixelPerfect bool) *Canvas {
	result := &Canvas{
		width:        width,
		height:       height,
		stretched:    stretched,
		pixelPerfect: pixelPerfect,
		buffer:       ebiten.NewImage(width, height),
		renderers:    make([]func(*ebiten.Image), 0),
	}

	result.SetFilter(AASamplingSoft)

	return result
}

// GetHeight - return 16:9 height from specified width
func GetHeight(width int) int {
	return int(math.Floor(float64(width) / (16.0 / 9.0)))
}

func (self *Canvas) SetFilter(filter ScalingFilter) {
	self.filter = filter

	if shaders[filter] == nil {
		compileShader(filter)
	}
}

func (self *Canvas) AddRenderer(renderer func(*ebiten.Image)) {
	self.renderers = append(self.renderers, renderer)
}

func (self *Canvas) GetTopLeft() ebimath.Vector {
	return self.topLeft
}

func (self *Canvas) GetScale() ebimath.Vector {
	return self.scale
}

func (self *Canvas) Resize(width, height int) {
	scale := ebimath.V(
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

	self.topLeft = ebimath.V2(0)
	if !self.stretched {
		self.topLeft = ebimath.V(
			(float64(width)-float64(self.width)*self.scale.X)/2,
			(float64(height)-float64(self.height)*self.scale.Y)/2,
		)
	}
}

func (self *Canvas) ScaleCursorPosition() ebimath.Vector {
	x, y := ebiten.CursorPosition()
	return ebimath.V(
		(float64(x)-self.topLeft.X)/self.scale.X,
		(float64(y)-self.topLeft.Y)/self.scale.Y,
	)
}

func (self *Canvas) Draw(screen *ebiten.Image) {
	self.buffer.Fill(color.Transparent)

	for _, r := range self.renderers {
		r(self.buffer)
	}

	p0 := self.topLeft
	p1 := p0.Add(ebimath.V(self.scale.X*float64(self.width), 0))
	p2 := p0.Add(ebimath.V(self.scale.X*float64(self.width), self.scale.Y*float64(self.height)))
	p3 := p0.Add(ebimath.V(0, self.scale.Y*float64(self.height)))

	shaderVertices[0].DstX = float32(p0.X)
	shaderVertices[0].DstY = float32(p0.Y)
	shaderVertices[1].DstX = float32(p1.X)
	shaderVertices[1].DstY = float32(p1.Y)
	shaderVertices[2].DstX = float32(p2.X)
	shaderVertices[2].DstY = float32(p2.Y)
	shaderVertices[3].DstX = float32(p3.X)
	shaderVertices[3].DstY = float32(p3.Y)

	sourceBounds := self.buffer.Bounds()
	shaderVertices[0].SrcX = float32(sourceBounds.Min.X)
	shaderVertices[0].SrcY = float32(sourceBounds.Min.Y)
	shaderVertices[1].SrcX = float32(sourceBounds.Max.X)
	shaderVertices[1].SrcY = shaderVertices[0].SrcY
	shaderVertices[2].SrcX = shaderVertices[1].SrcX
	shaderVertices[2].SrcY = float32(sourceBounds.Max.Y)
	shaderVertices[3].SrcX = shaderVertices[0].SrcX
	shaderVertices[3].SrcY = shaderVertices[2].SrcY

	shaderOpts.Images[0] = self.buffer
	shaderOpts.Uniforms["SourceRelativeTextureUnitX"] = float32(float64(self.width) / float64(screen.Bounds().Dx()))
	shaderOpts.Uniforms["SourceRelativeTextureUnitY"] = float32(float64(self.height) / float64(screen.Bounds().Dy()))
	screen.DrawTrianglesShader(
		shaderVertices, shaderVertIndices,
		shaders[self.filter], &shaderOpts,
	)
	shaderOpts.Images[0] = nil
}
