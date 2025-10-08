// Package ease provides easing functions for smooth interpolation in animations.
// Each function takes time (t), start value (b), change in value (c), and duration (d),
// returning the interpolated value. All functions handle t in [0, d] and return values in [b, b+c].
package ease

import "math"

type Float interface {
	~float32 | ~float64
}

// EaseFunc defines the signature for easing functions.
// Parameters:
//   t: Current time (0 to d).
//   b: Start value.
//   c: Change in value (end - start).
//   d: Total duration.
// Returns the interpolated value.
type EaseFunc[T Float] func(t, b, c, d T) T

// Constants for trigonometric calculations
const (
	pi     = math.Pi
	halfPi = pi / 2
	twoPi  = 2 * pi
	backS  = 1.70158 // Overshoot factor for Back easing
)

// Linear provides linear interpolation with no easing.
func Linear[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	return c*t/d + b
}

// QuadIn provides quadratic ease-in (accelerating).
func QuadIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t + b
}

// QuadOut provides quadratic ease-out (decelerating).
func QuadOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return -c*t*(t-2) + b
}

// QuadInOut provides quadratic ease-in for the first half and ease-out for the second half.
func QuadInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return c/2*t*t + b
	}
	t--
	return -c/2*(t*(t-2)-1) + b
}

// QuadOutIn provides quadratic ease-out for the first half and ease-in for the second half.
func QuadOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return -c/2*t*(t-2) + b
	}
	t--
	return c/2*t*t + b + c/2
}

// CubicIn provides cubic ease-in (accelerating).
func CubicIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t + b
}

// CubicOut provides cubic ease-out (decelerating).
func CubicOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*(t*t*t+1) + b
}

// CubicInOut provides cubic ease-in for the first half and ease-out for the second half.
func CubicInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return c/2*t*t*t + b
	}
	t -= 2
	return c/2*(t*t*t+2) + b
}

// CubicOutIn provides cubic ease-out for the first half and ease-in for the second half.
func CubicOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		t--
		return c/2*(t*t*t+1) + b
	}
	t--
	return c/2*t*t*t + b + c/2
}

// QuartIn provides quartic ease-in (accelerating).
func QuartIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t*t + b
}

// QuartOut provides quartic ease-out (decelerating).
func QuartOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return -c*(t*t*t*t-1) + b
}

// QuartInOut provides quartic ease-in for the first half and ease-out for the second half.
func QuartInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return c/2*t*t*t*t + b
	}
	t -= 2
	return -c/2*(t*t*t*t-2) + b
}

// QuartOutIn provides quartic ease-out for the first half and ease-in for the second half.
func QuartOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		t--
		return -c/2*(t*t*t*t-1) + b
	}
	t--
	return c/2*t*t*t*t + b + c/2
}

// QuintIn provides quintic ease-in (accelerating).
func QuintIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t*t*t + b
}

// QuintOut provides quintic ease-out (decelerating).
func QuintOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*(t*t*t*t*t+1) + b
}

// QuintInOut provides quintic ease-in for the first half and ease-out for the second half.
func QuintInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return c/2*t*t*t*t*t + b
	}
	t -= 2
	return c/2*(t*t*t*t*t+2) + b
}

// QuintOutIn provides quintic ease-out for the first half and ease-in for the second half.
func QuintOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		t--
		return c/2*(t*t*t*t*t+1) + b
	}
	t--
	return c/2*t*t*t*t*t + b + c/2
}

// SineIn provides sinusoidal ease-in (accelerating).
func SineIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	cos := math.Cos(float64(t) * halfPi)
	return -c*T(cos) + c + b
}

// SineOut provides sinusoidal ease-out (decelerating).
func SineOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	sin := math.Sin(float64(t) * halfPi)
	return c*T(sin) + b
}

// SineInOut provides sinusoidal ease-in for the first half and ease-out for the second half.
func SineInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	cos := math.Cos(float64(t/d) * pi)
	return -c/2*(T(cos)-1) + b
}

// SineOutIn provides sinusoidal ease-out for the first half and ease-in for the second half.
func SineOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		sin := math.Sin(float64(t) * halfPi)
		return c/2*T(sin) + b
	}
	t--
	cos := math.Cos(float64(t) * halfPi)
	return -c/2*T(cos) + c + b
}

// ExpoIn provides exponential ease-in (accelerating).
func ExpoIn[T Float](t, b, c, d T) T {
	if d == 0 || t == 0 {
		return b
	}
	t /= d
	pow := math.Pow(2, float64(10*(t-1)))
	return c*T(pow) + b
}

// ExpoOut provides exponential ease-out (decelerating).
func ExpoOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	if t == 1 {
		return b + c
	}
	pow := math.Pow(2, float64(-10*t))
	return c*(1-T(pow)) + b
}

// ExpoInOut provides exponential ease-in for the first half and ease-out for the second half.
func ExpoInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	if t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	t /= d / 2
	if t < 1 {
		pow := math.Pow(2, float64(10*(t-1)))
		return c/2*T(pow) + b
	}
	t--
	pow := math.Pow(2, float64(-10*t))
	return c/2*(2-T(pow)) + b
}

// ExpoOutIn provides exponential ease-out for the first half and ease-in for the second half.
func ExpoOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	if t < d/2 {
		return ExpoOut(t*2, b, c/2, d)
	}
	return ExpoIn((t*2)-d, b+c/2, c/2, d)
}

// CircIn provides circular ease-in (accelerating).
func CircIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	sqrt := math.Sqrt(float64(1 - t*t))
	return -c*(T(sqrt)-1) + b
}

// CircOut provides circular ease-out (decelerating).
func CircOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t = t/d - 1
	sqrt := math.Sqrt(float64(1 - t*t))
	return c*T(sqrt) + b
}

// CircInOut provides circular ease-in for the first half and ease-out for the second half.
func CircInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		sqrt := math.Sqrt(float64(1 - t*t))
		return -c/2*(T(sqrt)-1) + b
	}
	t -= 2
	sqrt := math.Sqrt(float64(1 - t*t))
	return c/2*(T(sqrt)+1) + b
}

// CircOutIn provides circular ease-out for the first half and ease-in for the second half.
func CircOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		t--
		sqrt := math.Sqrt(float64(1 - t*t))
		return c/2*T(sqrt) + b
	}
	t--
	sqrt := math.Sqrt(float64(1 - t*t))
	return -c/2*(T(sqrt)-1) + b + c/2
}

// BackIn provides back ease-in (overshooting backward before moving forward).
func BackIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*((backS+1)*t-backS) + b
}

// BackOut provides back ease-out (overshooting forward before settling).
func BackOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*(t*t*((backS+1)*t+backS)+1) + b
}

// BackInOut provides back ease-in for the first half and ease-out for the second half.
func BackInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	s := T(backS * 1.525)
	t /= d / 2
	if t < 1 {
		return c/2*(t*t*((s+1)*t-s)) + b
	}
	t -= 2
	return c/2*(t*t*((s+1)*t+s)+2) + b
}

// BackOutIn provides back ease-out for the first half and ease-in for the second half.
func BackOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	if t < d/2 {
		return BackOut(t*2, b, c/2, d)
	}
	return BackIn((t*2)-d, b+c/2, c/2, d)
}

// BounceIn provides bounce ease-in (simulating bounces with increasing amplitude).
func BounceIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	return c - BounceOut(d-t, 0, c, d) + b
}

// BounceOut provides bounce ease-out (simulating bounces with decreasing amplitude).
func BounceOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d
	if t < 4.0/11.0 {
		return c*((121.0*t*t)/16.0) + b
	} else if t < 8.0/11.0 {
		return c*((363.0/40.0*t*t)-(99.0/10.0*t)+17.0/5.0) + b
	} else if t < 9.0/10.0 {
		return c*((4356.0/361.0*t*t)-(35442.0/1805.0*t)+16061.0/1805.0) + b
	}
	return c*((54.0/5.0*t*t)-(513.0/25.0*t)+268.0/25.0) + b
}

// BounceInOut provides bounce ease-in for the first half and ease-out for the second half.
func BounceInOut[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return BounceIn(t*d, 0, c, d)/2 + b
	}
	return BounceOut(t*d-d, 0, c, d)/2 + c/2 + b
}

// BounceOutIn provides bounce ease-out for the first half and ease-in for the second half.
func BounceOutIn[T Float](t, b, c, d T) T {
	if d == 0 {
		return b
	}
	if t < d/2 {
		return BounceOut(t*2, b, c/2, d)
	}
	return BounceIn((t*2)-d, b+c/2, c/2, d)
}

// ElasticIn provides elastic ease-in (oscillating with increasing amplitude).
func ElasticIn[T Float](t, b, c, d T) T {
	if d == 0 || t == 0 {
		return b
	}
	t /= d
	if t == 1 {
		return b + c
	}
	p := 0.5
	s := p / 4
	t -= 1
	pow := math.Pow(2, 10*float64(t))
	sin := math.Sin((float64(t) - s) * twoPi / p)
	return -c*T(pow)*T(sin) + b
}

// ElasticOut provides elastic ease-out (oscillating with decreasing amplitude).
func ElasticOut[T Float](t, b, c, d T) T {
	if d == 0 || t == 0 {
		return b
	}
	t /= d
	if t == 1 {
		return b + c
	}
	p := 0.5
	s := p / 4
	pow := math.Pow(2, -10*float64(t))
	sin := math.Sin((float64(t) - s) * twoPi / p)
	return c*T(pow)*T(sin) + c + b
}

// ElasticInOut provides elastic ease-in for the first half and ease-out for the second half.
func ElasticInOut[T Float](t, b, c, d T) T {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	p := 0.5
	s := p / 4
	t /= d / 2
	if t < 1 {
		t -= 1
		pow := math.Pow(2, 10*float64(t))
		sin := math.Sin((float64(t) - s) * twoPi / p)
		return -0.5*(c*T(pow)*T(sin)) + b
	}
	t -= 1
	pow := math.Pow(2, -10*float64(t))
	sin := math.Sin((float64(t) - s) * twoPi / p)
	return c*T(pow)*T(sin)*0.5 + c + b
}

// ElasticOutIn provides elastic ease-out for the first half and ease-in for the second half.
func ElasticOutIn[T Float](t, b, c, d T) T {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	if t < d/2 {
		return ElasticOut(t*2, b, c/2, d)
	}
	return ElasticIn((t*2)-d, b+c/2, c/2, d)
}
