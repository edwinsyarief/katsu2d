package katsu2d

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/edwinsyarief/katsu2d/opensimplex"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	defaultCapacity = 5000
)

var (
	sharedWhiteOnce sync.Once
	sharedWhiteImg  *ebiten.Image
)

// defaultWhiteImage returns a lazily initialized shared 1x1 white image
// used as a fallback texture for untextured geometry.
func defaultWhiteImage() *ebiten.Image {
	sharedWhiteOnce.Do(func() {
		sharedWhiteImg = ebiten.NewImage(1, 1)
		sharedWhiteImg.Fill(color.White)
	})
	return sharedWhiteImg
}

// LerpPremultipliedRGBA linearly interpolates two colors using premultiplied alpha.
func LerpPremultipliedRGBA(color1, color2 color.RGBA, t float64) color.RGBA {
	t = Clamp(t, 0.0, 1.0)

	r1 := float64(color1.R) / 255.0
	g1 := float64(color1.G) / 255.0
	b1 := float64(color1.B) / 255.0
	a1 := float64(color1.A) / 255.0

	r2 := float64(color2.R) / 255.0
	g2 := float64(color2.G) / 255.0
	b2 := float64(color2.B) / 255.0
	a2 := float64(color2.A) / 255.0

	a := a1 + t*(a2-a1)
	if a <= 0.0 {
		return color.RGBA{}
	}

	r := (r1 + t*(r2-r1))
	g := (g1 + t*(g2-g1))
	b := (b1 + t*(b2-b1))

	return color.RGBA{
		R: uint8(Clamp(r*255.0, 0.0, 255.0)),
		G: uint8(Clamp(g*255.0, 0.0, 255.0)),
		B: uint8(Clamp(b*255.0, 0.0, 255.0)),
		A: uint8(Clamp(a*255.0, 0.0, 255.0)),
	}
}

// PremultiplyRGBA converts normalized [0, 1] RGBA channels into a premultiplied color.RGBA.
func PremultiplyRGBA(r, g, b, a float64) color.RGBA {
	r = Clamp(r, 0.0, 1.0)
	g = Clamp(g, 0.0, 1.0)
	b = Clamp(b, 0.0, 1.0)
	a = Clamp(a, 0.0, 1.0)

	if a > 0.0 {
		r *= a
		g *= a
		b *= a
	} else {
		r, g, b = 0, 0, 0
	}

	return color.RGBA{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(b * 255.0)),
		A: uint8(math.Round(a * 255.0)),
	}
}

// HexToPremultipliedRGBA converts a 24-bit 0xRRGGBB hex value and alpha [0, 1] to premultiplied color.RGBA.
func HexToPremultipliedRGBA(hex uint32, alpha float64) color.RGBA {
	alpha = Clamp(alpha, 0.0, 1.0)

	r := float64((hex&0xFF0000)>>16) / 255.0
	g := float64((hex&0xFF00)>>8) / 255.0
	b := float64(hex&0xFF) / 255.0

	if alpha > 0.0 {
		r *= alpha
		g *= alpha
		b *= alpha
	} else {
		r, g, b = 0, 0, 0
	}

	return color.RGBA{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(b * 255.0)),
		A: uint8(math.Round(alpha * 255.0)),
	}
}

// HexToRGB converts a 24-bit 0xRRGGBB hex value to an opaque color.RGBA.
func HexToRGB(hex uint32) color.RGBA {
	return color.RGBA{
		R: uint8((hex & 0xFF0000) >> 16),
		G: uint8((hex & 0xFF00) >> 8),
		B: uint8(hex & 0xFF),
		A: 255,
	}
}

// HexToNRGBA converts a 32-bit 0xRRGGBBAA hex value to color.NRGBA.
func HexToNRGBA(hex uint32) color.NRGBA {
	return color.NRGBA{
		R: uint8((hex & 0xFF000000) >> 24),
		G: uint8((hex & 0xFF0000) >> 16),
		B: uint8((hex & 0xFF00) >> 8),
		A: uint8(hex & 0xFF),
	}
}

// HexStringToRGBA parses "#RGB" or "#RRGGBB" into an opaque color.RGBA.
func HexStringToRGBA(s string) (c color.RGBA) {
	c.A = 0xff
	if len(s) == 0 || s[0] != '#' {
		return c
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	}
	return c
}

// RGBAToColorScale converts a color.RGBA into an ebiten.ColorScale.
func RGBAToColorScale(col color.RGBA) ebiten.ColorScale {
	var cs ebiten.ColorScale
	const inv255 = float32(1.0 / 255.0)
	cs.SetR(float32(col.R) * inv255)
	cs.SetG(float32(col.G) * inv255)
	cs.SetB(float32(col.B) * inv255)
	cs.SetA(float32(col.A) * inv255)
	return cs
}

// SecondsToDuration converts seconds as float64 into time.Duration.
func SecondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// BetweenInterval returns true on alternating intervals of length interval.
func BetweenInterval(val, interval float64) bool {
	if interval <= 0 {
		return false
	}
	return math.Mod(val, interval*2) > interval
}

// OnInterval reports whether val crossed a multiple of interval since prevVal.
func OnInterval(val, prevVal, interval float64) bool {
	if interval <= 0 {
		return false
	}
	return int(prevVal/interval) != int(val/interval)
}

// ImageRectangle constructs an image.Rectangle from min and max coordinates.
func ImageRectangle(minX, minY, maxX, maxY int) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{X: minX, Y: minY},
		Max: image.Point{X: maxX, Y: maxY},
	}
}

// CalculateRemainingRatio returns the clamped ratio of distance covered from start to end.
func CalculateRemainingRatio(start, end, current float64) float64 {
	totalDistance := end - start
	if totalDistance == 0 {
		return 1.0
	}
	ratio := (current - start) / totalDistance
	if end < start {
		ratio = 1 - ratio
	}
	return Clamp(ratio, 0.0, 1.0)
}

// CalculateProgressRatio returns 1 - CalculateRemainingRatio(start, end, current).
func CalculateProgressRatio(start, end, current float64) float64 {
	return 1 - CalculateRemainingRatio(start, end, current)
}

// GenerateHorizontalImageRectangles slices a horizontal sprite strip into count rectangles.
func GenerateHorizontalImageRectangles(x, y, width, height, count int) []image.Rectangle {
	result := make([]image.Rectangle, 0, count)
	for i := range count {
		sx := x + i*width
		result = append(result, ImageRectangle(sx, y, sx+width, y+height))
	}
	return result
}

// GetInterpolatedColor picks a random color with each channel independently chosen between c1 and c2.
func GetInterpolatedColor(c1, c2 color.RGBA) color.RGBA {
	return InterpolateColorWithRand(Random(), c1, c2)
}

// InterpolateColorWithRand picks a random color between c1 and c2 using the provided Rand.
func InterpolateColorWithRand(r *Rand, c1, c2 color.RGBA) color.RGBA {
	randChannel := func(a, b uint8) uint8 {
		if a == b {
			return a
		}
		lo, hi := float64(a), float64(b)
		if lo > hi {
			lo, hi = hi, lo
		}
		return uint8(math.Round(r.FloatRange(lo, hi)))
	}
	return color.RGBA{
		R: randChannel(c1.R, c2.R),
		G: randChannel(c1.G, c2.G),
		B: randChannel(c1.B, c2.B),
		A: randChannel(c1.A, c2.A),
	}
}

// GeneratePerlinNoiseMap generates a flat width*height slice of normalized [0, 1] OpenSimplex noise values.
func GeneratePerlinNoiseMap(width, height int, frequency float64) []float64 {
	if width <= 0 || height <= 0 {
		return nil
	}
	if frequency <= 0 {
		frequency = 1.0
	}
	p := opensimplex.New(Random().PositiveInt64())
	values := make([]float64, width*height)
	invFreq := 1.0 / frequency
	for y := range height {
		fy := float64(y) * invFreq
		row := y * width
		for x := range width {
			noiseVal := p.Eval2(float64(x)*invFreq, fy)
			values[row+x] = Clamp((noiseVal+1.0)*0.5, 0.0, 1.0)
		}
	}
	return values
}

// GeneratePerlinNoiseImage generates a grayscale OpenSimplex noise *ebiten.Image.
func GeneratePerlinNoiseImage(width, height int, frequency float64) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	values := GeneratePerlinNoiseMap(width, height, frequency)
	pixels := make([]byte, width*height*4)
	for i, v := range values {
		gray := byte(v * 255.0)
		idx := i * 4
		pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = gray, gray, gray, 255
	}
	img.WritePixels(pixels)
	return img
}

// GetFileExtension extracts the file extension without the leading dot, or "" if none.
func GetFileExtension(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/' && path[i] != '\\'; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return ""
}

// CreateVertex creates an ebiten.Vertex with pixel-adjusted destination coordinates and un-premultiplied RGB color.
func CreateVertex(dst, src Vector, col color.RGBA) ebiten.Vertex {
	const inv255 = float32(1.0 / 255.0)
	r := float32(col.R) * inv255
	g := float32(col.G) * inv255
	b := float32(col.B) * inv255
	a := float32(col.A) * inv255
	if a > 0.0 {
		r /= a
		g /= a
		b /= a
	}
	return ebiten.Vertex{
		DstX:   AdjustDestinationPixel(float32(dst.X)),
		DstY:   AdjustDestinationPixel(float32(dst.Y)),
		SrcX:   float32(src.X),
		SrcY:   float32(src.Y),
		ColorR: r,
		ColorG: g,
		ColorB: b,
		ColorA: a,
	}
}

// CreateVertexWithOpacity creates an ebiten.Vertex with explicit opacity scaling.
func CreateVertexWithOpacity(pos, uv Vector, col color.RGBA, opacity float64) ebiten.Vertex {
	const inv255 = float32(1.0 / 255.0)
	return ebiten.Vertex{
		DstX:   float32(pos.X),
		DstY:   float32(pos.Y),
		SrcX:   float32(uv.X),
		SrcY:   float32(uv.Y),
		ColorR: float32(col.R) * inv255,
		ColorG: float32(col.G) * inv255,
		ColorB: float32(col.B) * inv255,
		ColorA: float32(col.A) * inv255 * float32(opacity),
	}
}

// CreateVertexDefaultSrc creates an ebiten.Vertex with zero source UV coordinates.
func CreateVertexDefaultSrc(dst Vector, col color.RGBA) ebiten.Vertex {
	return CreateVertex(dst, V2(0), col)
}

// UpdateVertexColor returns a copy of vert with updated RGBA color channels.
func UpdateVertexColor(vert ebiten.Vertex, col color.RGBA) ebiten.Vertex {
	const inv255 = float32(1.0 / 255.0)
	vert.ColorR = float32(col.R) * inv255
	vert.ColorG = float32(col.G) * inv255
	vert.ColorB = float32(col.B) * inv255
	vert.ColorA = float32(col.A) * inv255
	return vert
}

// UpdateVertexDst returns a copy of vert with updated destination coordinates.
func UpdateVertexDst(vert ebiten.Vertex, dst Vector) ebiten.Vertex {
	vert.DstX = float32(dst.X)
	vert.DstY = float32(dst.Y)
	return vert
}

// UpdateVertexSrc returns a copy of vert with updated source UV coordinates.
func UpdateVertexSrc(vert ebiten.Vertex, src Vector) ebiten.Vertex {
	vert.SrcX = float32(src.X)
	vert.SrcY = float32(src.Y)
	return vert
}

// IsPointEmpty reports whether both coordinates of point are zero.
func IsPointEmpty(point Point) bool {
	return point.X == 0 && point.Y == 0
}

// IsBoundEmpty reports whether both Min and Max of bound are zero.
func IsBoundEmpty(bound Bound) bool {
	return IsPointEmpty(bound.Min) && IsPointEmpty(bound.Max)
}
