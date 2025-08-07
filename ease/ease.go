// Package ease provides easing functions for smooth interpolation in animations.
// Each function takes time (t), start value (b), change in value (c), and duration (d),
// returning the interpolated value. All functions handle t in [0, d] and return values in [b, b+c].
package ease

import "math"

// EaseFunc defines the signature for easing functions.
// Parameters:
//   t: Current time (0 to d).
//   b: Start value.
//   c: Change in value (end - start).
//   d: Total duration.
// Returns the interpolated value.
type EaseFunc func(t, b, c, d float32) float32

// Constants for trigonometric calculations
const (
	pi     = float32(math.Pi)
	halfPi = pi / 2
	twoPi  = 2 * pi
	backS  = 1.70158 // Overshoot factor for Back easing
)

// Linear provides linear interpolation with no easing.
func Linear(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	return c*t/d + b
}

// QuadIn provides quadratic ease-in (accelerating).
func QuadIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t + b
}

// QuadOut provides quadratic ease-out (decelerating).
func QuadOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return -c*t*(t-2) + b
}

// QuadInOut provides quadratic ease-in for the first half and ease-out for the second half.
func QuadInOut(t, b, c, d float32) float32 {
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
func QuadOutIn(t, b, c, d float32) float32 {
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
func CubicIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t + b
}

// CubicOut provides cubic ease-out (decelerating).
func CubicOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*(t*t*t+1) + b
}

// CubicInOut provides cubic ease-in for the first half and ease-out for the second half.
func CubicInOut(t, b, c, d float32) float32 {
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
func CubicOutIn(t, b, c, d float32) float32 {
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
func QuartIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t*t + b
}

// QuartOut provides quartic ease-out (decelerating).
func QuartOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return -c*(t*t*t*t-1) + b
}

// QuartInOut provides quartic ease-in for the first half and ease-out for the second half.
func QuartInOut(t, b, c, d float32) float32 {
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
func QuartOutIn(t, b, c, d float32) float32 {
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
func QuintIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return c*t*t*t*t*t + b
}

// QuintOut provides quintic ease-out (decelerating).
func QuintOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*(t*t*t*t*t+1) + b
}

// QuintInOut provides quintic ease-in for the first half and ease-out for the second half.
func QuintInOut(t, b, c, d float32) float32 {
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
func QuintOutIn(t, b, c, d float32) float32 {
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
func SineIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return -c*float32(math.Cos(float64(t)*float64(halfPi))) + c + b
}

// SineOut provides sinusoidal ease-out (decelerating).
func SineOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return c*float32(math.Sin(float64(t)*float64(halfPi))) + b
}

// SineInOut provides sinusoidal ease-in for the first half and ease-out for the second half.
func SineInOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	return -c/2*(float32(math.Cos(float64(t/d)*float64(pi)))-1) + b
}

// SineOutIn provides sinusoidal ease-out for the first half and ease-in for the second half.
func SineOutIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return c/2*float32(math.Sin(float64(t)*float64(halfPi))) + b
	}
	t--
	return -c/2*float32(math.Cos(float64(t)*float64(halfPi))) + c/2 + b + c/2
}

// ExpoIn provides exponential ease-in (accelerating).
// ExpoIn provides exponential ease-in (accelerating).
func ExpoIn(t, b, c, d float32) float32 {
	if d == 0 || t == 0 {
		return b
	}
	return c*float32(math.Pow(2, float64(10*(t/d-1)))) + b - c*0.001
}

// ExpoOut provides exponential ease-out (decelerating).
func ExpoOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	return c*1.001*(-float32(math.Pow(2, float64(-10*t/d)))+1) + b
}

// ExpoInOut provides exponential ease-in for the first half and ease-out for the second half.
func ExpoInOut(t, b, c, d float32) float32 {
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
		return c/2*float32(math.Pow(2, float64(10*(t-1)))) + b - c*0.0005
	}
	t--
	return c/2*1.0005*(2-float32(math.Pow(2, float64(-10*t)))) + b
}

// ExpoOutIn provides exponential ease-out for the first half and ease-in for the second half.
func ExpoOutIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	if t < d/2 {
		return ExpoOut(t*2, b, c/2, d)
	}
	return ExpoIn((t*2)-d, b+c/2, c/2, d)
}

// CircIn provides circular ease-in (accelerating).
func CircIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	return -c*(float32(math.Sqrt(float64(1-t*t)))-1) + b
}

// CircOut provides circular ease-out (decelerating).
func CircOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t = t/d - 1
	return c*float32(math.Sqrt(float64(1-t*t))) + b
}

// CircInOut provides circular ease-in for the first half and ease-out for the second half.
func CircInOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return -c/2*(float32(math.Sqrt(float64(1-t*t)))-1) + b
	}
	t -= 2
	return c/2*(float32(math.Sqrt(float64(1-t*t)))+1) + b
}

// CircOutIn provides circular ease-out for the first half and ease-in for the second half.
func CircOutIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		t--
		return c/2*float32(math.Sqrt(float64(1-t*t))) + b
	}
	t--
	return -c/2*(float32(math.Sqrt(float64(1-t*t)))-1) + b + c/2
}

// BackIn provides back ease-in (overshooting backward before moving forward).
func BackIn(t, b, c, d float32) float32 {
	t /= d
	return c*t*t*((backS+1)*t-backS) + b
}

// BackOut provides back ease-out (overshooting forward before settling).
func BackOut(t, b, c, d float32) float32 {
	t = t/d - 1
	return c*(t*t*((backS+1)*t+backS)+1) + b
}

// BackInOut provides back ease-in for the first half and ease-out for the second half.
func BackInOut(t, b, c, d float32) float32 {
	s := float32(backS * 1.525)
	t = t / d * 2
	if t < 1 {
		return c/2*(t*t*((s+1)*t-s)) + b
	}
	t -= 2
	return c/2*(t*t*((s+1)*t+s)+2) + b
}

// BackOutIn provides back ease-out for the first half and ease-in for the second half.
func BackOutIn(t, b, c, d float32) float32 {
	if t < (d / 2) {
		return BackOut(t*2, b, c/2, d)
	}
	return BackIn((t*2)-d, b+c/2, c/2, d)
}

// BounceIn provides bounce ease-in (simulating bounces with increasing amplitude).
func BounceIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	return c - BounceOut(d-t, 0, c, d) + b
}

// BounceOut provides bounce ease-out (simulating bounces with decreasing amplitude).
func BounceOut(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d
	if t < 1/2.75 {
		return c*(7.5625*t*t) + b
	} else if t < 2/2.75 {
		t -= 1.5 / 2.75
		return c*(7.5625*t*t+0.75) + b
	} else if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return c*(7.5625*t*t+0.9375) + b
	}
	t -= 2.625 / 2.75
	return c*(7.5625*t*t+0.984375) + b
}

// BounceInOut provides bounce ease-in for the first half and ease-out for the second half.
func BounceInOut(t, b, c, d float32) float32 {
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
func BounceOutIn(t, b, c, d float32) float32 {
	if d == 0 {
		return b
	}
	t /= d / 2
	if t < 1 {
		return BounceOut(t*d, 0, c, d)/2 + b
	}
	return BounceIn(t*d-d, 0, c, d)/2 + c/2 + b
}

// ElasticIn provides elastic ease-in (oscillating with increasing amplitude).
func ElasticIn(t, b, c, d float32) float32 {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	p := d * 0.3
	s := p / 4
	t = t/d - 1
	return -(c * float32(math.Pow(2, float64(10*t))) * float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p)))) + b
}

// ElasticOut provides elastic ease-out (oscillating with decreasing amplitude).
func ElasticOut(t, b, c, d float32) float32 {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	p := d * 0.3
	s := p / 4
	t /= d
	return c*float32(math.Pow(2, float64(-10*t)))*float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p))) + c + b
}

// ElasticInOut provides elastic ease-in for the first half and ease-out for the second half.
func ElasticInOut(t, b, c, d float32) float32 {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	p := d * 0.3 * 1.5
	s := p / 4
	t /= d / 2
	if t < 1 {
		t--
		return -0.5*(c*float32(math.Pow(2, float64(10*t)))*float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p)))) + b
	}
	t--
	return c*float32(math.Pow(2, float64(-10*t)))*float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p)))*0.5 + c + b
}

// ElasticOutIn provides elastic ease-out for the first half and ease-in for the second half.
func ElasticOutIn(t, b, c, d float32) float32 {
	if d == 0 || t == 0 {
		return b
	}
	if t == d {
		return b + c
	}
	p := d * 0.3 * 1.5
	s := p / 4
	t /= d / 2
	if t < 1 {
		return c/2*float32(math.Pow(2, float64(-10*t)))*float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p))) + c/2 + b
	}
	t--
	return -(c / 2 * float32(math.Pow(2, float64(10*t))) * float32(math.Sin(float64(t*d-s)*float64(twoPi)/float64(p)))) + b + c/2
}
