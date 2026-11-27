package katsu2d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type canvas struct {
	topLeft, scale          Vector
	width, height           int
	stretched, pixelPerfect bool
	filter                  ScalingFilter
}

func newCanvas(width, height int, stretched, pixelPerfect bool) *canvas {
	c := &canvas{
		width:        max(1, width),
		height:       max(1, height),
		scale:        V2(1),
		stretched:    stretched,
		pixelPerfect: pixelPerfect,
	}
	c.SetFilter(SF_AASamplingSoft)
	return c
}

func (c *canvas) SetFilter(filter ScalingFilter) {
	c.filter = filter
	if shaders[filter] == nil {
		compileShader(filter)
	}
}

func (c *canvas) GetTopLeft() Vector {
	return c.topLeft
}

func (c *canvas) GetScale() Vector {
	return c.scale
}

func (c *canvas) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	scale := V(
		float64(width)/float64(c.width),
		float64(height)/float64(c.height),
	)
	if !c.stretched {
		if scale.X < scale.Y {
			scale.Y = scale.X
		} else {
			scale.X = scale.Y
		}
	}
	if c.pixelPerfect {
		scale.X = math.Max(1, math.Floor(scale.X))
		scale.Y = math.Max(1, math.Floor(scale.Y))
	}
	c.scale = scale
	if c.stretched {
		c.topLeft = V2(0)
	} else {
		c.topLeft = V(
			(float64(width)-float64(c.width)*c.scale.X)/2,
			(float64(height)-float64(c.height)*c.scale.Y)/2,
		)
	}
}

func (c *canvas) ScreenToCanvas(x, y float64) Vector {
	if c.scale.X == 0 || c.scale.Y == 0 {
		return V(x, y)
	}
	return V(
		(x-c.topLeft.X)/c.scale.X,
		(y-c.topLeft.Y)/c.scale.Y,
	)
}

func (c *canvas) ScaleCursorPosition() Vector {
	x, y := ebiten.CursorPosition()
	return c.ScreenToCanvas(float64(x), float64(y))
}

func (c *canvas) Draw(source, destination *ebiten.Image) {
	dstW := destination.Bounds().Dx()
	dstH := destination.Bounds().Dy()
	if dstW <= 0 || dstH <= 0 {
		return
	}

	p0 := c.topLeft
	w := c.scale.X * float64(c.width)
	h := c.scale.Y * float64(c.height)
	shaderVertices[0].DstX = float32(p0.X)
	shaderVertices[0].DstY = float32(p0.Y)
	shaderVertices[1].DstX = float32(p0.X + w)
	shaderVertices[1].DstY = float32(p0.Y)
	shaderVertices[2].DstX = float32(p0.X + w)
	shaderVertices[2].DstY = float32(p0.Y + h)
	shaderVertices[3].DstX = float32(p0.X)
	shaderVertices[3].DstY = float32(p0.Y + h)

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
	switch c.filter {
	case SF_CrtLottes:
		tw, th := sourceBounds.Dx(), sourceBounds.Dy()
		shaderOpts.Uniforms["ScreenSize"] = []float32{float32(dstW), float32(dstH)}
		shaderOpts.Uniforms["TextureSize"] = []float32{float32(tw), float32(th)}
	default:
		shaderOpts.Uniforms["SourceRelativeTextureUnitX"] = float32(float64(c.width) / float64(dstW))
		shaderOpts.Uniforms["SourceRelativeTextureUnitY"] = float32(float64(c.height) / float64(dstH))
	}
	destination.DrawTrianglesShader(
		shaderVertices, shaderVertIndices,
		shaders[c.filter], &shaderOpts,
	)
	shaderOpts.Images[0] = nil
}
