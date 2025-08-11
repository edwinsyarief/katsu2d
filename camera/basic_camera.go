// Package camera provides 2D camera systems for game rendering, supporting smooth
// tracking, zooming, and effects like bumps and shakes.
package camera

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/managers"
)

var (
	frameCount = 0
)

// Camera constants
const (
	MinZoom              = 1.0   // Minimum zoom level
	MaxZoom              = 10.0  // Maximum zoom level
	BaseFriction         = 0.8   // Base movement friction
	BumpFriction         = 0.85  // Friction for bump effects
	ZoomFriction         = 0.9   // Friction for zoom interpolation
	DefaultTrackingSpeed = 1.0   // Default speed for tracking
	BreakDistance        = 0.1   // Distance for braking near bounds
	DeadZonePercentX     = 0.04  // X-axis dead zone percentage
	DeadZonePercentY     = 0.10  // Y-axis dead zone percentage
	TrackingSpeedFactorX = 0.015 // X-axis tracking speed factor
	TrackingSpeedFactorY = 0.050 // Y-axis tracking speed factor
	FrictionTrackingMod  = 0.027 // Friction modifier for tracking
	ShakeFreqX           = 1.1   // X-axis shake frequency
	ShakeFreqY           = 1.7   // Y-axis shake frequency
	ShakeAmp             = 2.5   // Shake amplitude
)

// BasicCamera implements a 2D camera with smooth tracking, zoom, bump, and shake effects.
// It follows a target transform, clamps to level bounds, and applies transformations to a scroller.
type BasicCamera struct {
	target                     *ebimath.Transform        // Target transform to track
	scroller                   *ebimath.Transform        // Transform applied to the view
	targetOffset               ebimath.Vector            // Offset from target's position
	rawFocus                   ebimath.Vector            // Unclamped focus point
	clampedFocus               ebimath.Vector            // Focus point after clamping
	originalViewport, viewport ebimath.Vector            // Visible area in world coordinates
	levelBounds                ebimath.Rectangle         // Level boundaries for clamping
	baseZoom                   float64                   // Base zoom level
	targetZoom                 float64                   // Desired zoom level
	zoomSpeed                  float64                   // Zoom interpolation speed
	destZoom                   float64                   // Current zoom velocity
	bumpZoom                   float64                   // Temporary zoom offset
	trackingSpeed              float64                   // Speed of tracking movement
	destPoint                  ebimath.Vector            // Movement velocity
	bumpOffset                 ebimath.Vector            // Temporary position offset
	shakePower                 float64                   // Shake intensity
	clampBounds                bool                      // Whether to clamp to level bounds
	left, right                int                       // Cached bounds (world coordinates)
	top, bottom                int                       // Cached bounds (world coordinates)
	center                     ebimath.Vector            // Cached center (world coordinates)
	cd                         *managers.CooldownManager // Cooldown system for effects
}

// NewBasicCamera creates a new BasicCamera with the given viewport and scroller transform.
func NewBasicCamera(viewport ebimath.Vector, scroller *ebimath.Transform) *BasicCamera {
	if scroller == nil {
		scroller = ebimath.T()
	}
	return &BasicCamera{
		scroller:         scroller,
		targetOffset:     ebimath.V2(0),
		viewport:         viewport,
		originalViewport: viewport,
		baseZoom:         1.0,
		targetZoom:       1.0,
		zoomSpeed:        0.0014,
		trackingSpeed:    DefaultTrackingSpeed,
		clampBounds:      true,
		cd:               managers.NewCooldownManager(10),
	}
}

// GetViewport returns the camera’s viewport size in world coordinates.
func (self *BasicCamera) GetViewport() ebimath.Vector {
	return self.viewport
}

// GetZoom returns the current zoom level (base + bump).
func (self *BasicCamera) GetZoom() float64 {
	return self.baseZoom + self.bumpZoom
}

// SetZoomSpeed sets the speed of zoom interpolation.
func (self *BasicCamera) SetZoomSpeed(speed float64) {
	self.zoomSpeed = speed
}

// ZoomTo sets a target zoom level with smooth interpolation.
func (self *BasicCamera) ZoomTo(value float64) {
	self.targetZoom = ebimath.Clamp(value, MinZoom, MaxZoom)
}

// ForceZoom sets the zoom level immediately without interpolation.
func (self *BasicCamera) ForceZoom(value float64) {
	value = ebimath.Clamp(value, MinZoom, MaxZoom)
	self.baseZoom = value
	self.targetZoom = value
	self.destZoom = 0
}

// BumpZoom applies a temporary zoom offset that decays over time.
func (self *BasicCamera) BumpZoom(value float64) {
	self.bumpZoom = value
}

// SetTrackingSpeed sets the speed at which the camera follows its target.
func (self *BasicCamera) SetTrackingSpeed(speed float64) {
	self.trackingSpeed = speed
}

// TrackObject sets a target to follow with optional immediate centering and offset.
func (self *BasicCamera) TrackObject(target *ebimath.Transform, immediateFocus bool, trackingSpeed float64, offset ebimath.Vector) {
	self.target = target
	if trackingSpeed > 0 {
		self.trackingSpeed = trackingSpeed
	}
	self.targetOffset = offset
	if self.target != nil && (immediateFocus || self.rawFocus.IsZero()) {
		self.CenterOnTarget()
	}
}

// CenterOnTarget instantly moves the camera to the target’s position.
func (self *BasicCamera) CenterOnTarget() {
	if self.target != nil {
		self.rawFocus = self.target.Position().Add(self.targetOffset.Mul(self.target.Scale()))
	}
}

// StopTrackObject stops tracking the current target.
func (self *BasicCamera) StopTrackObject() {
	self.target = nil
}

// SetLevelBounds defines the level boundaries for clamping.
func (self *BasicCamera) SetLevelBounds(bounds ebimath.Rectangle) {
	self.levelBounds = bounds
	self.clampBounds = true
}

// SetClampToLevelBounds toggles clamping to level bounds.
func (self *BasicCamera) SetClampToLevelBounds(clamp bool) {
	self.clampBounds = clamp
}

// Bump applies a temporary position offset in world coordinates.
func (self *BasicCamera) Bump(v ebimath.Vector) {
	self.bumpOffset = self.bumpOffset.Add(v)
}

// BumpAngular applies a position offset by angle and distance in world coordinates.
func (self *BasicCamera) BumpAngular(angle, distance float64) {
	self.bumpOffset = self.bumpOffset.Add(ebimath.V(math.Cos(angle)*distance, math.Sin(angle)*distance))
}

// Shake applies a shake effect with the given duration and power.
func (self *BasicCamera) Shake(duration, power float64) {
	self.shakePower = power
	if power != 0 {
		self.cd.Set("shaking", duration, nil)
	}
}

// IsShaking returns true if the camera is currently shaking.
func (c *BasicCamera) IsShaking() bool {
	return c.cd.Has("shaking")
}

// Update advances the camera’s state based on elapsed time.
func (self *BasicCamera) Update(deltaTime float64) {
	frameCount = (frameCount % math.MaxInt) + 1

	// Update viewport and cooldowns
	zoom := self.GetZoom()
	self.viewport = ebimath.V(self.originalViewport.X/zoom, self.originalViewport.Y/zoom)
	self.cd.Update(deltaTime)

	// Update zoom
	if zoomDiff := self.targetZoom - self.baseZoom; zoomDiff != 0 {
		zoomStep := self.zoomSpeed * deltaTime
		self.destZoom = ebimath.Clamp(self.destZoom+ebimath.Sign(zoomDiff)*zoomStep, math.Min(zoomDiff, 0), math.Max(zoomDiff, 0))
		self.baseZoom += self.destZoom
		self.destZoom *= math.Pow(ZoomFriction, deltaTime)
	}
	self.bumpZoom *= math.Pow(ZoomFriction, deltaTime)

	// Track target
	if self.target != nil {
		targetPos := self.target.Position().Add(self.targetOffset.Mul(self.target.Scale()))
		angle := float64(self.rawFocus.AngleToPoint(targetPos))
		distX := math.Abs(targetPos.X - self.rawFocus.X)
		distY := math.Abs(targetPos.Y - self.rawFocus.Y)
		spdX := TrackingSpeedFactorX * self.trackingSpeed * zoom
		spdY := TrackingSpeedFactorY * self.trackingSpeed * zoom
		if distX >= DeadZonePercentX*self.viewport.X {
			deltaX := 0.8*distX - DeadZonePercentX*self.viewport.X
			self.destPoint.X += math.Cos(angle) * deltaX * spdX * deltaTime
		}
		if distY >= DeadZonePercentY*self.viewport.Y {
			deltaY := 0.8*distY - DeadZonePercentY*self.viewport.Y
			self.destPoint.Y += math.Sin(angle) * deltaY * spdY * deltaTime
		}
	}

	// Apply friction and braking
	frict := BaseFriction - self.trackingSpeed*zoom*FrictionTrackingMod*BaseFriction
	if self.clampBounds {
		self.applyBreaking(&frict, deltaTime)
	}
	self.rawFocus = self.rawFocus.Add(self.destPoint.MulF(deltaTime))
	self.destPoint = self.destPoint.MulF(math.Pow(frict, deltaTime))
	self.bumpOffset = self.bumpOffset.MulF(math.Pow(BumpFriction, deltaTime))

	// Clamp to bounds
	if self.clampBounds {
		self.clampedFocus.X = ebimath.Clamp(self.rawFocus.X, self.viewport.X*0.5, math.Max(self.levelBounds.Width()-self.viewport.X*0.5, self.viewport.X*0.5))
		self.clampedFocus.Y = ebimath.Clamp(self.rawFocus.Y, self.viewport.Y*0.5, math.Max(self.levelBounds.Height()-self.viewport.Y*0.5, self.viewport.Y*0.5))
	} else {
		self.clampedFocus = self.rawFocus
	}

	// Apply transformations
	self.apply()
}

// applyBreaking adjusts friction near level bounds to slow the camera.
func (self *BasicCamera) applyBreaking(frict *float64, deltaTime float64) {
	brakeDistX := BreakDistance * self.viewport.X
	if self.destPoint.X != 0 {
		var brakeRatio float64
		if self.destPoint.X < 0 {
			brakeRatio = 1 - ebimath.Clamp((self.rawFocus.X-self.viewport.X*0.5)/brakeDistX, 0, 1)
		} else {
			brakeRatio = 1 - ebimath.Clamp((self.levelBounds.Width()-self.viewport.X*0.5-self.rawFocus.X)/brakeDistX, 0, 1)
		}
		*frict *= 1 - 0.9*brakeRatio*deltaTime
	}
	brakeDistY := BreakDistance * self.viewport.Y
	if self.destPoint.Y != 0 {
		var brakeRatio float64
		if self.destPoint.Y < 0 {
			brakeRatio = 1 - ebimath.Clamp((self.rawFocus.Y-self.viewport.Y*0.5)/brakeDistY, 0, 1)
		} else {
			brakeRatio = 1 - ebimath.Clamp((self.levelBounds.Height()-self.viewport.Y*0.5-self.rawFocus.Y)/brakeDistY, 0, 1)
		}
		*frict *= 1 - 0.9*brakeRatio*deltaTime
	}
}

// apply updates the scroller transform and cached bounds.
func (self *BasicCamera) apply() {
	zoom := self.GetZoom()
	screenCenter := ebimath.V(self.originalViewport.X/2, self.originalViewport.Y/2)
	t := screenCenter.Sub(self.clampedFocus.MulF(zoom))
	t = t.Sub(self.bumpOffset.MulF(zoom))
	if self.cd.Has("shaking") {
		ftime := float64(frameCount)
		shakeAmount := self.shakePower * self.cd.GetRatio("shaking")
		shakeOffset := ebimath.V(
			math.Cos(ftime*ShakeFreqX)*ShakeAmp*shakeAmount,
			math.Sin(0.3+ftime*ShakeFreqY)*ShakeAmp*shakeAmount,
		)
		t = t.Add(shakeOffset.MulF(zoom))
		if self.cd.GetRatio("shaking") == 0 {
			self.shakePower = 0 // Reset shake power when done
		}
	}
	self.scroller.SetPosition(t.Round())
	self.scroller.SetScale(ebimath.V2(zoom))
	self.left = int(self.clampedFocus.X - self.viewport.X*0.5)
	self.right = self.left + int(self.viewport.X-1)
	self.top = int(self.clampedFocus.Y - self.viewport.Y*0.5)
	self.bottom = self.top + int(self.viewport.Y-1)
	self.center = ebimath.V(float64(self.left+self.right)*0.5, float64(self.top+self.bottom)*0.5)
}

// Area returns camera min and max points based on camera position.
func (self *BasicCamera) Area() ebimath.Rectangle {
	return ebimath.NewRectangle(
		float64(self.left),
		float64(self.top),
		float64(self.right-self.left+1),
		float64(self.bottom-self.top+1),
	)
}
