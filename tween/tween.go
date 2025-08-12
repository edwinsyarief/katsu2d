// Package tween provides functionality for tweening, which is the process of
// generating intermediate values between a start and end state over a specified
// duration, typically using an easing function to create smooth transitions.
package tween

import "github.com/edwinsyarief/katsu2d/ease"

// Tween represents a tweening object that interpolates between a start and end
// value over a specified duration using an easing function. It supports features
// such as a delay before starting, a callback on completion, and methods to
// control and query its state.
type Tween struct {
	delay    float32       // Delay before the tween starts
	start    float32       // Starting value of the tween
	end      float32       // Ending value of the tween
	duration float32       // Duration of the tween in seconds
	time     float32       // Current elapsed time, including delay
	easing   ease.EaseFunc // Easing function to interpolate values
	current  float32       // Current value of the tween
	finished bool          // Indicates if the tween has completed
	onDone   func()        // Callback function called when the tween finishes
	onUpdate func(float32) // Callback function called every update when tween is running
}

// New creates a new Tween instance with the specified start value, end value,
// duration, and easing function. The tween starts at the initial value with a
// default delay of 0.
func New(start, end, duration float32, easing ease.EaseFunc) *Tween {
	return &Tween{
		start:    start,
		current:  start,
		end:      end,
		duration: duration,
		easing:   easing,
	}
}

// Reset resets the tween to its initial state, setting the time to 0, delay to 0,
// current value to start, finished to false, and clears the callback. This
// prepares the tween to restart from the beginning, discarding any previous
// delay or callback settings.
func (self *Tween) Reset(start, end, duration float32, easing ease.EaseFunc) {
	self.start = start
	self.end = end
	self.duration = duration
	self.easing = easing
	self.time = 0
	self.delay = 0
	self.current = self.start
	self.finished = false
	if self.onDone != nil {
		self.onDone = nil // Reset callback
	}
	if self.onUpdate != nil {
		self.onUpdate = nil // Rest callback
	}
}

// IsFinished returns whether the tween has completed its interpolation.
func (self *Tween) IsFinished() bool {
	return self.finished
}

// CurrentValue returns the current interpolated value of the tween.
func (self *Tween) CurrentValue() float32 {
	return self.current
}

// Set sets the internal time to the specified value, accounting for the delay,
// and calculates the current value accordingly. It does not trigger the finished
// state or the callback, even if the time exceeds the duration plus delay.
// This method is useful for manually positioning the tween at a specific point.
func (self *Tween) Set(time float32) float32 {
	self.time = time
	if time < self.delay {
		self.current = self.start
	} else if time >= self.duration+self.delay {
		self.current = self.end
	} else {
		self.current = self.easing(time-self.delay, self.start, self.end-self.start, self.duration)
	}
	return self.current
}

// SetDelay sets the delay before the tween starts and resets the internal time
// to 0 to ensure the delay takes effect from the beginning.
func (self *Tween) SetDelay(delay float32) {
	self.delay = delay
	//self.time = 0 // Reset time when setting a new delay
}

// SetDuration sets the duration of the tween, preserving the previous completeness
// percentage if the tween was in progress. The internal time is scaled to maintain
// the same relative progress, and the current value is recalculated. If the tween
// was finished, it is marked as unfinished to allow further updates.
func (self *Tween) SetDuration(duration float32) {
	if self.duration == 0 || self.time <= self.delay || self.finished {
		// If duration was 0, time is in delay, or tween is finished, restart from beginning
		self.duration = duration
		self.time = 0
		self.current = self.start
		self.finished = false
		return
	}

	// Calculate completeness percentage (0 to 1) based on progress after delay
	progress := (self.time - self.delay) / self.duration
	// Update duration and scale time to maintain the same progress
	self.duration = duration
	self.time = self.delay + progress*duration
	// Recalculate current value using the easing function
	self.current = self.easing(self.time-self.delay, self.start, self.end-self.start, self.duration)
	self.finished = false // Allow the tween to continue
}

// SetCallback sets a callback function to be executed when the tween completes
// naturally through the Update method.
func (self *Tween) SetCallback(callback func()) {
	self.onDone = callback
}

// SetOnUpdate sets a callback function to be executed every tween update called
// when the tween is running
func (self *Tween) SetOnUpdate(updateCallback func(float32)) {
	self.onUpdate = updateCallback
}

// Update advances the tween by the given delta time (in seconds) and returns the
// current value and a boolean indicating whether the tween has finished. If the
// tween completes during this update, the callback is triggered.
func (self *Tween) Update(delta float32) (float32, bool) {
	if self.finished {
		return self.end, true // Already finished
	}

	self.time += delta

	if self.time < self.delay {
		return self.start, false // Still in delay period
	}

	if self.time >= self.duration+self.delay {
		self.current = self.end
		self.finished = true
		if self.onUpdate != nil {
			self.onUpdate(self.current)
		}
		if self.onDone != nil {
			self.onDone() // Call the callback if set
		}
		return self.end, true // Tween is done
	}

	self.current = self.easing(self.time-self.delay, self.start, self.end-self.start, self.duration)
	if self.onUpdate != nil {
		self.onUpdate(self.current)
	}
	return self.current, false // Tween is still in progress
}
