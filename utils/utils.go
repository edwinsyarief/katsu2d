package utils

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"reflect"
	"time"

	"github.com/aquilax/go-perlin"
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

func ResizeSlices[T any](slice []T, newLen int) []T {
	if newLen > cap(slice) {
		newCap := cap(slice) * 2
		if newCap < newLen {
			newCap = newLen
		}
		newSlice := make([]T, newLen, newCap)
		copy(newSlice, slice)
		return newSlice
	}
	return slice[:newLen]
}

func New[T any]() T {
	var obj T
	return obj
}

func NewPointer[T any]() *T {
	var obj T
	return &obj
}

func NewPointerFromInstance[T any](instance T) T {
	t := reflect.TypeOf(instance).Elem()
	return reflect.New(t).Interface().(T)
}

// InitializeStruct initializes a struct by recursively setting up its embedded structs
// and pointer-to-struct fields, calling their Init() methods if available.
func InitializeStruct(ptr any) {
	v := reflect.ValueOf(ptr).Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		// Handle embedded structs
		if field.Kind() == reflect.Struct && field.CanSet() {
			// Recursively initialize the embedded struct
			InitializeStruct(field.Addr().Interface())
			// Call Init method if it exists
			if initMethod, ok := field.Type().MethodByName("Init"); ok {
				initMethod.Func.Call([]reflect.Value{field.Addr()})
			}
		}
		// Handle pointer-to-struct fields
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct && field.CanSet() {
			if field.IsNil() {
				// Instantiate the pointer if nil
				field.Set(reflect.New(field.Type().Elem()))
			}
			// Recursively initialize the pointed-to struct
			InitializeStruct(field.Interface())
			// Call Init method if it exists
			if initMethod, ok := field.Type().Elem().MethodByName("Init"); ok {
				initMethod.Func.Call([]reflect.Value{field})
			}
		}
	}
}

// InitializeEmbedded creates a new instance of the input struct type and initializes
// its embedded structs and pointer fields with default values via Init methods.
func InitializeEmbedded[T any](obj T) T {
	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Struct {
		return obj
	}
	// Create a new instance of the struct
	v := reflect.New(t).Elem()
	// Initialize it recursively
	InitializeStruct(v.Addr().Interface())
	return v.Interface().(T)
}

func LerpPremultipliedRGBA(color1, color2 color.RGBA, t float64) color.RGBA {
	// Clamp interpolation factor (t) to 0-1 range
	t = ebimath.Clamp(t, 0.0, 1.0)

	// Convert RGBA to float64 (0-1 range) with premultiplied alpha handling
	r1 := float64(color1.R) / 255.0
	g1 := float64(color1.G) / 255.0
	b1 := float64(color1.B) / 255.0
	a1 := float64(color1.A) / 255.0
	if a1 > 0.0 {
		r1 /= a1
		g1 /= a1
		b1 /= a1
	}

	r2 := float64(color2.R) / 255.0
	g2 := float64(color2.G) / 255.0
	b2 := float64(color2.B) / 255.0
	a2 := float64(color2.A) / 255.0
	if a2 > 0.0 {
		r2 /= a2
		g2 /= a2
		b2 /= a2
	}

	// Lerp with premultiplied alpha
	r := (r1*a1 + t*(r2*a2-r1*a1)) / (a1 + t*(a2-a1))
	g := (g1*a1 + t*(g2*a2-g1*a1)) / (a1 + t*(a2-a1))
	b := (b1*a1 + t*(b2*a2-b1*a1)) / (a1 + t*(a2-a1))
	a := a1 + t*(a2-a1)

	// Convert back to uint8 and clamp to 0-255 range
	return color.RGBA{R: uint8(ebimath.Clamp(r*255.0, 0.0, 255.0)),
		G: uint8(ebimath.Clamp(g*255.0, 0.0, 255.0)),
		B: uint8(ebimath.Clamp(b*255.0, 0.0, 255.0)),
		A: uint8(ebimath.Clamp(a*255.0, 0.0, 255.0))}
}

func PremultiplyRGBA(r, g, b, a float64) color.RGBA {
	// Clamp values to 0-1 range in case of slight rounding errors
	r = ebimath.Clamp(r, 0.0, 1.0)
	g = ebimath.Clamp(g, 0.0, 1.0)
	b = ebimath.Clamp(b, 0.0, 1.0)
	a = ebimath.Clamp(a, 0.0, 1.0)

	// Premultiply RGB channels by alpha
	if a > 0.0 {
		r *= a
		g *= a
		b *= a
	}

	// Convert to uint8 and round to nearest integer
	r_uint8 := uint8(math.Round(r * 255.0))
	g_uint8 := uint8(math.Round(g * 255.0))
	b_uint8 := uint8(math.Round(b * 255.0))
	a_uint8 := uint8(math.Round(a * 255.0))

	return color.RGBA{R: r_uint8, G: g_uint8, B: b_uint8, A: a_uint8}
}

func HexToPremultipliedRGBA(hex uint32, alpha float64) color.RGBA {
	// Clamp alpha to 0-1 range
	alpha = ebimath.Clamp(alpha, 0.0, 1.0)

	// Extract RGB channels from hex
	r := float64((hex&0xFF0000)>>16) / 255.0
	g := float64((hex&0xFF00)>>8) / 255.0
	b := float64(hex&0xFF) / 255.0

	// Premultiply RGB channels if alpha is greater than zero
	if alpha > 0.0 {
		r *= alpha
		g *= alpha
		b *= alpha
	}

	// Convert to uint8 and round to nearest integer
	r_uint8 := uint8(math.Round(r * 255.0))
	g_uint8 := uint8(math.Round(g * 255.0))
	b_uint8 := uint8(math.Round(b * 255.0))
	a_uint8 := uint8(math.Round(alpha * 255.0))

	return color.RGBA{R: r_uint8, G: g_uint8, B: b_uint8, A: a_uint8}
}

func HexToRGB(hex uint32) color.RGBA {
	r := uint8((hex & 0xFF0000) >> 16)
	g := uint8((hex & 0xFF00) >> 8)
	b := uint8(hex & 0xFF)

	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func HexToNRGBA(hex uint32) color.NRGBA {
	r := uint8((hex & 0xFF000000) >> 24)
	g := uint8((hex & 0xFF0000) >> 16)
	b := uint8((hex & 0xFF00) >> 8)
	a := uint8(hex & 0xFF)

	return color.NRGBA{r, g, b, a}
}

func HexStringToRGBA(s string) (c color.RGBA) {
	c.A = 0xff

	if s[0] != '#' {
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
	default:
	}
	return
}

func RGBAToColorScale(col color.RGBA) ebiten.ColorScale {
	cs := ebiten.ColorScale{}
	cs.SetR(float32(col.R) / 255.0)
	cs.SetG(float32(col.G) / 255.0)
	cs.SetB(float32(col.B) / 255.0)
	cs.SetA(float32(col.A) / 255.0)
	return cs
}

func RemoveRange[T any](s []T, idx int, count int) []T {
	return append(s[:idx], s[idx+count:]...)
}

func RemoveFront[T any](s []T) []T {
	return RemoveRangeFront(s, 1)
}

func RemoveRangeFront[T any](s []T, count int) []T {
	return append(s[:0], s[count:]...)
}

func PushFront[T any](s []T, e T) []T {
	return append([]T{e}, s...)
}

func PopBack[T any](s []T) []T {
	return PopRangeBack(s, 1)
}

func PopRangeBack[T any](s []T, count int) []T {
	return s[:len(s)-count]
}

func Filter[T any](s []T, f func(T) bool) []T {
	result := make([]T, 0)

	for _, e := range s {
		if f(e) {
			result = append(result, e)
		}
	}

	return result
}

func GenerateString(values ...int) string {
	return fmt.Sprintf("%.d", values)
}

func SecondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

func BetweenInterval(val, interval float64) bool {
	return math.Mod(val, (interval*2)) > interval
}

func OnInterval(val, prevVal, interval float64) bool {
	return int(prevVal/interval) != int(val/interval)
}

func ImageRectangle(minX, minY, maxX, maxY int) image.Rectangle {
	return image.Rectangle{Min: image.Point{X: minX, Y: minY}, Max: image.Point{X: maxX, Y: maxY}}
}

func ClampF(value, min, max float64) float64 {
	return math.Min(math.Max(value, min), max)
}

func CalculateRemainingRatio(start, end, current float64) float64 {
	totalDistance := end - start
	distanceCovered := current - start
	remainingRatio := distanceCovered / totalDistance
	if end < start {
		remainingRatio = 1 - remainingRatio
	}
	remainingRatio = ClampF(remainingRatio, 0.0, 1.0)
	return remainingRatio
}

func CalculateProgressRatio(start, end, current float64) float64 {
	return 1 - CalculateRemainingRatio(start, end, current)
}

func GenerateHorizontalImageRectangles(x, y, width, height, count int) []image.Rectangle {
	result := []image.Rectangle{}

	for i := range count {
		sx, sy := x+i*width, y
		result = append(result, ImageRectangle(sx, sy, sx+width, sy+height))
	}

	return result
}

// AdjustDestinationPixel is the original ebitengine implementation found here:
// https://github.com/hajimehoshi/ebiten/blob/v2.8.0-alpha.1/internal/graphics/vertex.go#L102-L126
func AdjustDestinationPixel(x float32) float32 {
	// Avoid the center of the pixel, which is problematic (#929, #1171).
	// Instead, align the vertices with about 1/3 pixels.
	//
	// The intention here is roughly this code:
	//
	//     float32(math.Floor((float64(x)+1.0/6.0)*3) / 3)
	//
	// The actual implementation is more optimized than the above implementation.
	ix := float32(int(x))
	if x < 0 && x != ix {
		ix -= 1
	}
	frac := x - ix
	switch {
	case frac < 3.0/16.0:
		return ix
	case frac < 8.0/16.0:
		return ix + 5.0/16.0
	case frac < 13.0/16.0:
		return ix + 11.0/16.0
	default:
		return ix + 1
	}
}

// GetInterpolatedColor returns a color between min and max
func GetInterpolatedColor(min, max color.RGBA) color.RGBA {
	r := uint8(float64(min.R) + rand.Float64()*float64(max.R-min.R))
	g := uint8(float64(min.G) + rand.Float64()*float64(max.G-min.G))
	b := uint8(float64(min.B) + rand.Float64()*float64(max.B-min.B))
	a := uint8(float64(min.A) + rand.Float64()*float64(max.A-min.A))
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// generatePerlinNoiseImage generates a Perlin noise image for wind simulation.
func GeneratePerlinNoiseImage(width, height int, frequency float64) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	p := perlin.NewPerlin(2, 2, 3, rand.Int64())
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noiseVal := p.Noise2D(float64(x)/frequency, float64(y)/frequency)
			gray := byte((noiseVal + 1) * 127.5)
			idx := (y*width + x) * 4
			pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = gray, gray, gray, 255
		}
	}
	img.WritePixels(pixels)
	return img
}

func GetFileExtension(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return ""
}

func CreateVertice(src, dst ebimath.Vector, col color.RGBA) ebiten.Vertex {
	r := float32(col.R) / 255.0
	g := float32(col.G) / 255.0
	b := float32(col.B) / 255.0
	a := float32(col.A) / 255.0
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

func GenerateIndices(verticesLength int) []uint16 {
	if verticesLength < 4 || verticesLength%2 != 0 {
		return nil
	}

	result := []uint16{}

	loop := (verticesLength / 2) - 1

	for i := range loop {
		maxIndices := 3 + (i * 2)
		minIndex := maxIndices - 3

		if i == 0 {
			result = append(result, uint16(minIndex), uint16(minIndex+1), uint16(minIndex+2))
			result = append(result, uint16(minIndex), uint16(maxIndices-1), uint16(maxIndices))
		} else {
			result = append(result, uint16(minIndex+1), uint16(minIndex), uint16(minIndex+2))
			result = append(result, uint16(minIndex+1), uint16(maxIndices-1), uint16(maxIndices))
		}
	}

	return result
}

func ToRadians(degrees float64) float64 {
	return math.Pi * degrees / 180
}
