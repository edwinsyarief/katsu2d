package camera

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/ease"
)

// InterpolationMode defines the interpolation profiles for camera movements.
type InterpolationMode int

const (
	Instant InterpolationMode = iota
	Linear
	QuadIn
	QuadOut
	QuadInOut
	QuadOutIn
	CubicIn
	CubicOut
	CubicInOut
	CubicOutIn
	QuartIn
	QuartOut
	QuartInOut
	QuartOutIn
	QuintIn
	QuintOut
	QuintInOut
	QuintOutIn
	SineIn
	SineOut
	SineInOut
	SineOutIn
	ExpoIn
	ExpoOut
	ExpoInOut
	ExpoOutIn
	CircIn
	CircOut
	CircInOut
	CircOutIn
	BackIn
	BackOut
	BackInOut
	BackOutIn
	BounceIn
	BounceOut
	BounceInOut
	BounceOutIn
	ElasticIn
	ElasticOut
	ElasticInOut
	ElasticOutIn
	Spring
)

// easingFuncs maps InterpolationMode to the corresponding ease.EaseFunc
var easingFuncs = map[InterpolationMode]ease.EaseFunc{
	Linear:       ease.Linear,
	QuadIn:       ease.QuadIn,
	QuadOut:      ease.QuadOut,
	QuadInOut:    ease.QuadInOut,
	QuadOutIn:    ease.QuadOutIn,
	CubicIn:      ease.CubicIn,
	CubicOut:     ease.CubicOut,
	CubicInOut:   ease.CubicInOut,
	CubicOutIn:   ease.CubicOutIn,
	QuartIn:      ease.QuartIn,
	QuartOut:     ease.QuartOut,
	QuartInOut:   ease.QuartInOut,
	QuartOutIn:   ease.QuartOutIn,
	QuintIn:      ease.QuintIn,
	QuintOut:     ease.QuintOut,
	QuintInOut:   ease.QuintInOut,
	QuintOutIn:   ease.QuintOutIn,
	SineIn:       ease.SineIn,
	SineOut:      ease.SineOut,
	SineInOut:    ease.SineInOut,
	SineOutIn:    ease.SineOutIn,
	ExpoIn:       ease.ExpoIn,
	ExpoOut:      ease.ExpoOut,
	ExpoInOut:    ease.ExpoInOut,
	ExpoOutIn:    ease.ExpoOutIn,
	CircIn:       ease.CircIn,
	CircOut:      ease.CircOut,
	CircInOut:    ease.CircInOut,
	CircOutIn:    ease.CircOutIn,
	BackIn:       ease.BackIn,
	BackOut:      ease.BackOut,
	BackInOut:    ease.BackInOut,
	BackOutIn:    ease.BackOutIn,
	BounceIn:     ease.BounceIn,
	BounceOut:    ease.BounceOut,
	BounceInOut:  ease.BounceInOut,
	BounceOutIn:  ease.BounceOutIn,
	ElasticIn:    ease.ElasticIn,
	ElasticOut:   ease.ElasticOut,
	ElasticInOut: ease.ElasticInOut,
	ElasticOutIn: ease.ElasticOutIn,
}

// easingFunc computes the interpolation factor for timed modes based on progress (t in [0, 1]).
func easingFunc(mode InterpolationMode, t float64) float64 {
	t = ebimath.Clamp(t, 0, 1) // Ensure t is bounded
	if fn, ok := easingFuncs[mode]; ok {
		return float64(fn(float32(t), 0, 1, 1))
	}
	return t // Default to linear for unmapped modes or if function not found
}

// Interpolator manages smooth transitions for a single scalar value (e.g., position, zoom).
type Interpolator struct {
	current   float64           // Current value
	target    float64           // Target value
	mode      InterpolationMode // Interpolation type
	duration  float64           // Duration for timed interpolations (seconds)
	elapsed   float64           // Elapsed time for timed interpolations
	velocity  float64           // Velocity for spring interpolation
	start     float64           // Starting value for timed interpolations
	stiffness float64           // Spring stiffness (for Spring mode)
	damping   float64           // Damping factor (for Spring mode)
}

// SetTarget sets a new target value with the specified mode and duration.
func (self *Interpolator) SetTarget(newTarget float64, mode InterpolationMode, duration float64) {
	self.target = newTarget
	self.mode = mode
	switch self.mode {
	case Instant:
		self.current = newTarget
		self.velocity = 0
	case Spring:
		// Initialize stiffness and damping if not set (provide sensible defaults)
		if self.stiffness == 0 {
			self.stiffness = 100.0 // Default stiffness
		}
		if self.damping == 0 {
			self.damping = 10.0 // Default damping
		}
		// Preserve velocity for continuous spring motion, don't reset unless explicitly desired
	default: // All easing modes (Linear, QuadIn, etc.)
		if duration <= 0 {
			duration = 0.001 // Prevent division by zero for timed modes
		}
		self.start = self.current
		self.duration = duration
		self.elapsed = 0
		self.velocity = 0 // Reset velocity for non-spring modes
	}
}

// Update advances the interpolation based on elapsed time (deltaTime in seconds).
func (self *Interpolator) Update(deltaTime float64) {
	switch self.mode {
	case Instant:
		// When mode is Instant, current is already at target, no update needed.
		return
	case Spring:
		displacement := self.target - self.current
		// If very close to target and velocity is low, snap to target
		if math.Abs(displacement) < 0.001 && math.Abs(self.velocity) < 0.001 {
			self.current = self.target
			self.velocity = 0
			return
		}
		acceleration := self.stiffness*displacement - self.damping*self.velocity
		self.velocity += acceleration * deltaTime
		self.current += self.velocity * deltaTime
	default: // All easing modes (Linear, QuadIn, etc.)
		self.elapsed += deltaTime
		if self.elapsed >= self.duration {
			self.current = self.target
			self.mode = Instant // Snap to target and stop interpolation
			return
		}
		t := self.elapsed / self.duration
		easing := easingFunc(self.mode, t)
		self.current = self.start + (self.target-self.start)*easing
	}
}
