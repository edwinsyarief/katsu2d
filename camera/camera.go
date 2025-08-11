// Package camera provides 2D camera systems for game rendering, supporting smooth
// tracking, zooming, and effects like bumps, shakes, and interpolated movements.
package camera

import (
	ebimath "github.com/edwinsyarief/ebi-math"
)

// Camera represents a 2D camera with smooth interpolation for position, zoom, and rotation.
type Camera struct {
	*ebimath.Transform                 // Embedded transform for position and rotation
	CurrentZoom        float64         // Current zoom level
	PositionX          Interpolator    // Interpolator for X position
	PositionY          Interpolator    // Interpolator for Y position
	Zoom               Interpolator    // Interpolator for zoom
	Rotation           Interpolator    // Interpolator for rotation
	ViewportWidth      int             // Screen width in pixels
	ViewportHeight     int             // Screen height in pixels
	followTarget       *ebimath.Vector // Pointer to target's position (nil if not following)
}

// NewCamera creates a new Camera with default values and initialized interpolators.
func NewCamera(viewportWidth, viewportHeight int) *Camera {
	t := ebimath.T()
	pos := t.Position()
	c := &Camera{
		Transform:      t,
		CurrentZoom:    1.0,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
	c.PositionX = Interpolator{current: pos.X, target: pos.X, mode: Instant}
	c.PositionY = Interpolator{current: pos.Y, target: pos.Y, mode: Instant}
	c.Zoom = Interpolator{current: 1.0, target: 1.0, mode: Instant}
	c.Rotation = Interpolator{current: t.Rotation(), target: t.Rotation(), mode: Instant}
	return c
}

// Follow sets the camera to smoothly follow a target position using Spring interpolation.
func (self *Camera) Follow(target *ebimath.Vector) {
	self.followTarget = target
	if target != nil {
		// When following, ensure position interpolators are set to Spring
		self.PositionX.SetTarget(target.X, Spring, 0) // Duration 0 is fine for Spring, it's not time-based
		self.PositionY.SetTarget(target.Y, Spring, 0)
	}
}

// StopFollowing stops following any target and sets position interpolators to Instant.
func (self *Camera) StopFollowing() {
	self.followTarget = nil
	self.PositionX.mode = Instant
	self.PositionY.mode = Instant
	// Snap current to target to prevent any lingering interpolation from previous follow
	self.PositionX.current = self.PositionX.target
	self.PositionY.current = self.PositionY.target
}

// Update advances the camera’s interpolators and applies the results to the transform.
func (self *Camera) Update(deltaTime float64) {
	if self.followTarget != nil {
		// Continuously update the target for spring interpolation when following
		self.PositionX.target = self.followTarget.X
		self.PositionY.target = self.followTarget.Y
	}

	self.PositionX.Update(deltaTime)
	self.PositionY.Update(deltaTime)
	self.Zoom.Update(deltaTime)
	self.Rotation.Update(deltaTime)

	self.CurrentZoom = ebimath.Clamp(self.Zoom.current, MinZoom, MaxZoom)
	// Apply the interpolated values to the camera's actual transform position
	self.SetPosition(ebimath.V(self.PositionX.current, self.PositionY.current))
	self.SetRotation(self.Rotation.current)
}

// WorldToScreen converts a world position to screen coordinates.
func (self *Camera) WorldToScreen(worldPos ebimath.Vector) ebimath.Vector {
	screenCenter := ebimath.V(float64(self.ViewportWidth)/2, float64(self.ViewportHeight)/2)
	relativePos := worldPos.Sub(self.Position())
	rotatedPos := relativePos.Rotate(-self.Transform.Rotation())
	scaledPos := rotatedPos.MulF(self.CurrentZoom)
	return screenCenter.Add(scaledPos)
}

// ScreenToWorld converts a screen position to world coordinates.
func (self *Camera) ScreenToWorld(screenPos ebimath.Vector) ebimath.Vector {
	screenCenter := ebimath.V(float64(self.ViewportWidth)/2, float64(self.ViewportHeight)/2)
	relativePos := screenPos.Sub(screenCenter)
	unscaledPos := relativePos.DivF(self.CurrentZoom)
	unrotatedPos := unscaledPos.Rotate(self.Transform.Rotation())
	return self.Position().Add(unrotatedPos)
}

// SetTargetPosition sets a new target position with interpolation.
// This will stop any direct movement or following.
func (self *Camera) SetTargetPosition(position ebimath.Vector, mode InterpolationMode, duration float64) {
	self.followTarget = nil // Stop following when a specific target is set
	self.PositionX.SetTarget(position.X, mode, duration)
	self.PositionY.SetTarget(position.Y, mode, duration)
}

// SetTargetZoom sets a new target zoom level with interpolation.
func (self *Camera) SetTargetZoom(zoom float64, mode InterpolationMode, duration float64) {
	zoom = ebimath.Clamp(zoom, MinZoom, MaxZoom)
	self.Zoom.SetTarget(zoom, mode, duration)
}

// SetTargetRotation sets a new target rotation with interpolation.
func (self *Camera) SetTargetRotation(rotation float64, mode InterpolationMode, duration float64) {
	self.Rotation.SetTarget(rotation, mode, duration)
}

// Move moves the camera by a delta vector with interpolation.
// This function is intended for one-shot, interpolated movements to a new relative position.
// For continuous movement (e.g., player input), use AddPosition.
func (self *Camera) Move(delta ebimath.Vector, mode InterpolationMode, duration float64) {
	currentPos := self.Position()
	self.SetTargetPosition(currentPos.Add(delta), mode, duration)
}

// AddPosition directly adds a delta vector to the camera's current position.
// This is suitable for continuous, player-controlled movement.
// It will stop any active position interpolation or following.
func (self *Camera) AddPosition(delta ebimath.Vector) {
	self.followTarget = nil // Stop following when direct movement is applied
	newPos := self.Position().Add(delta)
	// Instantly set interpolator to new position, effectively bypassing ongoing interpolation
	self.PositionX.SetTarget(newPos.X, Instant, 0)
	self.PositionY.SetTarget(newPos.Y, Instant, 0)
	self.SetPosition(newPos) // Directly update the transform
}

// SetPositionInstant instantly sets the camera’s position and stops following.
func (self *Camera) SetPositionInstant(position ebimath.Vector) {
	self.followTarget = nil
	self.PositionX.SetTarget(position.X, Instant, 0)
	self.PositionY.SetTarget(position.Y, Instant, 0)
	self.SetPosition(position) // Ensure transform is also updated instantly
}

// SetZoomInstant instantly sets the camera’s zoom level.
func (self *Camera) SetZoomInstant(zoom float64) {
	zoom = ebimath.Clamp(zoom, MinZoom, MaxZoom)
	self.Zoom.SetTarget(zoom, Instant, 0)
	self.CurrentZoom = zoom // Update CurrentZoom directly
}

// Rotate rotates the camera by an angle with interpolation.
func (self *Camera) Rotate(angle float64, mode InterpolationMode, duration float64) {
	currentRotation := self.Transform.Rotation()
	self.SetTargetRotation(currentRotation+angle, mode, duration)
}

// SetRotationInstant instantly sets the camera’s rotation.
func (self *Camera) SetRotationInstant(rotation float64) {
	self.Rotation.SetTarget(rotation, Instant, 0)
	self.SetRotation(rotation) // Ensure transform is also updated instantly
}

// Area returns the min and max points based on camera's position.
func (self *Camera) Area() ebimath.Rectangle {
	// Calculate half dimensions of the viewport in world space
	halfWidth := float64(self.ViewportWidth) / (2 * self.CurrentZoom)
	halfHeight := float64(self.ViewportHeight) / (2 * self.CurrentZoom)

	// Create vectors for the four corners before rotation
	topLeft := ebimath.V(-halfWidth, -halfHeight)
	topRight := ebimath.V(halfWidth, -halfHeight)
	bottomLeft := ebimath.V(-halfWidth, halfHeight)
	bottomRight := ebimath.V(halfWidth, halfHeight)

	// Apply rotation to all corners
	rotation := self.Transform.Rotation()
	topLeft = topLeft.Rotate(rotation)
	topRight = topRight.Rotate(rotation)
	bottomLeft = bottomLeft.Rotate(rotation)
	bottomRight = bottomRight.Rotate(rotation)

	// Translate corners to world position
	cameraPos := self.Position()
	topLeft = topLeft.Add(cameraPos)
	topRight = topRight.Add(cameraPos)
	bottomLeft = bottomLeft.Add(cameraPos)
	bottomRight = bottomRight.Add(cameraPos)

	// Find min and max points
	minX := ebimath.Min(ebimath.Min(topLeft.X, topRight.X), ebimath.Min(bottomLeft.X, bottomRight.X))
	minY := ebimath.Min(ebimath.Min(topLeft.Y, topRight.Y), ebimath.Min(bottomLeft.Y, bottomRight.Y))
	maxX := ebimath.Max(ebimath.Max(topLeft.X, topRight.X), ebimath.Max(bottomLeft.X, bottomRight.X))
	maxY := ebimath.Max(ebimath.Max(topLeft.Y, topRight.Y), ebimath.Max(bottomLeft.Y, bottomRight.Y))

	return ebimath.NewRectangle(minX, minY, maxX, maxY)
}
