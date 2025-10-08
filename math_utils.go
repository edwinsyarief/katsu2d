package katsu2d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Type Aliases
// ------------
type Matrix = ebiten.GeoM

// Constants
// ---------
const (
	Pi       = 3.141592653589793
	Epsilon  = 1e-9
	degToRad = math.Pi / 180.0
	radToDeg = 180.0 / math.Pi
)

type Number interface {
	Float | ~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Float interface {
	~float32 | ~float64
}

// Utility Functions for Floating Point Comparisons
// ------------------------------------------------
// EqualsApproximately checks if two numbers of a generic float type are approximately equal.
func EqualsApproximately[T Float](a, b T) bool {
	// Check for exact equality first, required to handle "infinity" values.
	if a == b {
		return true
	}
	// Then check for approximate equality.
	tolerance := Epsilon * Abs(T(a))
	if tolerance < Epsilon {
		tolerance = Epsilon
	}
	return Abs(T(a-b)) < tolerance
}

// Math Utilities
// --------------
// Repeat normalizes a value within a range.
func Repeat(t, length float64) float64 {
	return Clamp(t-math.Floor(t/length)*length, 0, length)
}

// CubicInterpolate performs cubic interpolation between values.
func CubicInterpolate(from, to, pre, post, t float64) float64 {
	return 0.5 *
		((from * 2.0) +
			(-pre+to)*t +
			(2.0*pre-5.0*from+4.0*to-post)*(t*t) +
			(-pre+3.0*from-3.0*to+post)*(t*t*t))
}

// Angle Conversion
// ----------------
// ToDegrees converts radians to degrees.
func ToDegrees(radians float64) float64 {
	return radians * radToDeg
}

// ToRadians converts degrees to radians.
func ToRadians(degrees float64) float64 {
	return degrees * degToRad
}

// Linear Interpolation
// --------------------
// Lerp performs linear interpolation.
func Lerp[T Number](from, to, t T) T {
	return from + ((to - from) * t)
}

// Clamping and Rounding
// ---------------------
// Clamp restricts a value to be within specified bounds.
func Clamp[T Number](value, min, max T) T {
	if value <= min {
		return min
	}
	if value >= max {
		return max
	}
	return value
}

// FastFloor performs a fast floor operation for floating-point numbers.
func FastFloor[T Float, U Number](value T) U {
	return U((value + 32768.0) - 32768)
}

// ClampTowardsZero clamps a value towards zero based on another value's sign.
func ClampTowardsZero[T Number](value, clampReference T) T {
	if clampReference > 0 {
		return Min(value, clampReference)
	}
	return Max(value, clampReference)
}

// Absolute and Sign Functions
// ---------------------------
// Abs returns the absolute value.
func Abs[T Number](value T) T {
	if value >= 0 {
		return value
	}
	return -value
}

// Sign returns +1 for positive numbers, -1 for negative numbers, ignoring zero.
func Sign[T Number](value T) T {
	if value >= 0 {
		return +1
	}
	return -1
}

// Min and Max Functions
// ---------------------
// Max returns the larger of two values.
func Max[T Number](v1, v2 T) T {
	if v1 > v2 {
		return v1
	}
	return v2
}

// Min returns the smaller of two values.
func Min[T Number](v1, v2 T) T {
	if v1 < v2 {
		return v1
	}
	return v2
}

// Vector and Angle
// ----------------
// AngleToVector converts an angle in radians to a vector with a given length.
func AngleToVector(angleRadians float64, length float64) Vector {
	return V(math.Cos(angleRadians)*length, math.Sin(angleRadians)*length)
}

// Pixel Adjustment
// ----------------
// AdjustDestinationPixel adjusts the pixel position to avoid center issues in rendering.
func AdjustDestinationPixel(x float32) float32 {
	// Avoid the center of the pixel, which is problematic (#929, #1171).
	// Instead, align the vertices with about 1/3 pixels.
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
