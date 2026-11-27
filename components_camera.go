package katsu2d

import (
	"math"

	"github.com/edwinsyarief/teishoku"
)

// CameraFollowMode defines how a camera tracks its target entity.
type CameraFollowMode int

const (
	// CameraFollowInstant snaps the camera directly to the target position each update.
	CameraFollowInstant CameraFollowMode = iota
	// CameraFollowSmooth moves the camera toward the target using frame-rate-independent
	// exponential smoothing: 1 - math.Exp(-FollowSpeed * dt).
	CameraFollowSmooth
	// CameraFollowSpring tracks the target with a damped harmonic spring using
	// SpringStiffness, SpringDamping, and Velocity.
	CameraFollowSpring
	// CameraFollowDeadZone tracks the target using a normalized viewport dead zone
	// (e.g. (0.04, 0.10)) with velocity/friction and edge braking near Bounds.
	CameraFollowDeadZone
)

// ActiveCamera is a world resource storing the active camera's view matrices,
// visible world bounds, and state for rendering and culling systems.
type ActiveCamera struct {
	ViewMatrix     Matrix
	InvViewMatrix  Matrix
	ViewBounds     Rectangle
	Position       Vector
	Zoom           float64
	Rotation       float64
	ViewportWidth  float64
	ViewportHeight float64
	HasCamera      bool
}

// WorldToScreen converts a world-space point into screen-space coordinates.
func (ac *ActiveCamera) WorldToScreen(worldPos Vector) Vector {
	if ac == nil || !ac.HasCamera {
		return worldPos
	}
	return worldPos.Apply(ac.ViewMatrix)
}

// ScreenToWorld converts a screen-space point into world-space coordinates.
func (ac *ActiveCamera) ScreenToWorld(screenPos Vector) Vector {
	if ac == nil || !ac.HasCamera {
		return screenPos
	}
	return screenPos.Apply(ac.InvViewMatrix)
}

// IsVisible reports whether the given world-space rectangle intersects the camera's view bounds.
func (ac *ActiveCamera) IsVisible(bounds Rectangle) bool {
	if ac == nil || !ac.HasCamera {
		return true
	}
	return ac.ViewBounds.Intersects(bounds)
}

// IsPointVisible reports whether a world-space point (expanded by margin) is within the view bounds.
func (ac *ActiveCamera) IsPointVisible(point Vector, margin float64) bool {
	if ac == nil || !ac.HasCamera {
		return true
	}
	return ac.ViewBounds.IntersectsCircle(point, math.Max(0, margin))
}

type cameraScalarTween struct {
	start    float64
	end      float64
	duration float64
	elapsed  float64
	easeType EaseType
	active   bool
}

type cameraVecTween struct {
	start    Vector
	end      Vector
	duration float64
	elapsed  float64
	easeType EaseType
	active   bool
}

// CameraComponent stores pure-data 2D camera state for ECS entities, including
// follow tracking, world bounds clamping, eased tweening, shake, and bump effects.
type CameraComponent struct {
	// Core
	Position           Vector
	Zoom               float64
	MinZoom            float64
	MaxZoom            float64
	Rotation           float64
	ViewportWidth      float64
	ViewportHeight     float64
	Active             bool
	PixelSnap          bool
	AutoResizeViewport bool

	// Follow
	FollowMode       CameraFollowMode
	HasTarget        bool
	TargetEntity     teishoku.Entity
	TargetOffset     Vector
	FollowSpeed      float64
	SpringStiffness  float64
	SpringDamping    float64
	DeadZone         Vector
	Velocity         Vector
	TrackingFriction float64
	snapNextUpdate   bool

	// Bounds
	Bounds            Rectangle
	ClampBounds       bool
	EdgeBrakeDistance float64

	// Position/Zoom/Rotation tweening & zoom bump
	posTween        cameraVecTween
	zoomTween       cameraScalarTween
	rotTween        cameraScalarTween
	BumpZoomValue   float64
	BumpZoomDamping float64

	// Shake & Bump
	BumpOffset     Vector
	BumpDamping    float64
	ShakeDuration  float64
	ShakeRemaining float64
	ShakePower     float64
	ShakeRotation  float64
	shakeTime      float64
	ShakeFreqX     float64
	ShakeFreqY     float64
	shakeOffset    Vector
	shakeRot       float64

	// Cached outputs
	viewMatrix    Matrix
	invViewMatrix Matrix
	viewBounds    Rectangle
	effectivePos  Vector
	effectiveZoom float64
	effectiveRot  float64
	matricesValid bool
}

// CameraOption is a functional option for configuring a CameraComponent.
type CameraOption func(*CameraComponent)

// WithCameraPosition sets the initial world position of the camera.
func WithCameraPosition(pos Vector) CameraOption {
	return func(c *CameraComponent) {
		c.Position = pos
	}
}

// WithCameraZoom sets the initial zoom factor of the camera.
func WithCameraZoom(zoom float64) CameraOption {
	return func(c *CameraComponent) {
		c.Zoom = zoom
	}
}

// WithCameraZoomLimits sets the minimum and maximum zoom bounds of the camera.
func WithCameraZoomLimits(min, max float64) CameraOption {
	return func(c *CameraComponent) {
		c.MinZoom = min
		c.MaxZoom = max
	}
}

// WithCameraRotation sets the initial rotation of the camera in radians.
func WithCameraRotation(rad float64) CameraOption {
	return func(c *CameraComponent) {
		c.Rotation = rad
	}
}

// WithCameraBounds sets the world bounds and enables bounds clamping.
func WithCameraBounds(bounds Rectangle) CameraOption {
	return func(c *CameraComponent) {
		c.Bounds = bounds
		c.ClampBounds = true
	}
}

// WithCameraFollowMode sets the target-following mode.
func WithCameraFollowMode(mode CameraFollowMode) CameraOption {
	return func(c *CameraComponent) {
		c.FollowMode = mode
	}
}

// WithCameraDeadZone sets the normalized dead-zone proportions for CameraFollowDeadZone.
func WithCameraDeadZone(deadZone Vector) CameraOption {
	return func(c *CameraComponent) {
		c.DeadZone = deadZone
	}
}

// WithCameraFollowSpeed sets the smoothing/tracking speed for following modes.
func WithCameraFollowSpeed(speed float64) CameraOption {
	return func(c *CameraComponent) {
		c.FollowSpeed = speed
	}
}

// WithCameraSpring sets the stiffness and damping constants for CameraFollowSpring.
func WithCameraSpring(stiffness, damping float64) CameraOption {
	return func(c *CameraComponent) {
		c.SpringStiffness = stiffness
		c.SpringDamping = damping
	}
}

// WithCameraPixelSnap enables or disables pixel-snapping on the final view translation.
func WithCameraPixelSnap(snap bool) CameraOption {
	return func(c *CameraComponent) {
		c.PixelSnap = snap
	}
}

// WithCameraAutoResize enables or disables automatic viewport resizing on EngineLayoutChangedEvent.
func WithCameraAutoResize(enabled bool) CameraOption {
	return func(c *CameraComponent) {
		c.AutoResizeViewport = enabled
	}
}

// NewCameraComponent creates a new CameraComponent configured with sensible defaults
// and any provided functional options.
func NewCameraComponent(viewportW, viewportH float64, opts ...CameraOption) *CameraComponent {
	c := &CameraComponent{
		Zoom:               1.0,
		MinZoom:            0.1,
		MaxZoom:            10.0,
		ViewportWidth:      viewportW,
		ViewportHeight:     viewportH,
		Active:             true,
		PixelSnap:          true,
		AutoResizeViewport: true,
		FollowMode:        CameraFollowSmooth,
		FollowSpeed:       8.0,
		SpringStiffness:   120.0,
		SpringDamping:     16.0,
		DeadZone:          V(0.04, 0.10),
		TrackingFriction:  0.85,
		EdgeBrakeDistance: 0.1,
		BumpDamping:       10.0,
		BumpZoomDamping:   10.0,
		ShakeFreqX:        28.0,
		ShakeFreqY:        37.0,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if c.MinZoom <= 0 {
		c.MinZoom = 0.0001
	}
	if c.MaxZoom < c.MinZoom {
		c.MaxZoom = c.MinZoom
	}
	c.Zoom = c.clampZoom(c.Zoom)
	c.RebuildMatrices()
	return c
}

func (c *CameraComponent) clampZoom(zoom float64) float64 {
	minZ := c.MinZoom
	if minZ <= 0 {
		minZ = 0.0001
	}
	maxZ := c.MaxZoom
	if maxZ < minZ {
		return math.Max(minZ, zoom)
	}
	return Clamp(zoom, minZ, maxZ)
}

// FollowEntity configures the camera to follow the given entity with a world offset.
func (c *CameraComponent) FollowEntity(entity teishoku.Entity, offset Vector) {
	c.TargetEntity = entity
	c.TargetOffset = offset
	c.HasTarget = true
	c.posTween.active = false
}

// FollowEntityImmediate configures the camera to follow the given entity and snaps
// directly to its position on the next update.
func (c *CameraComponent) FollowEntityImmediate(entity teishoku.Entity, offset Vector) {
	c.TargetEntity = entity
	c.TargetOffset = offset
	c.HasTarget = true
	c.snapNextUpdate = true
	c.posTween.active = false
}

// StopFollowing clears the current follow target and resets follow velocity.
func (c *CameraComponent) StopFollowing() {
	c.HasTarget = false
	c.TargetEntity = teishoku.Entity{}
	c.Velocity = Vector{}
	c.snapNextUpdate = false
}

// SnapToTarget marks the camera to snap immediately to its follow target on the next update.
func (c *CameraComponent) SnapToTarget() {
	c.snapNextUpdate = true
}

// SetPosition immediately moves the camera to pos and cancels any active position tween.
func (c *CameraComponent) SetPosition(pos Vector) {
	c.Position = pos
	c.posTween.active = false
	c.Velocity = Vector{}
	c.matricesValid = false
}

// Pan offsets the camera's position by delta and cancels any active follow target or position tween.
func (c *CameraComponent) Pan(delta Vector) {
	c.Position = c.Position.Add(delta)
	c.HasTarget = false
	c.posTween.active = false
	c.Velocity = Vector{}
	c.matricesValid = false
}

// MoveTo smoothly transitions the camera position to target over duration using easeType,
// stopping any active target following.
func (c *CameraComponent) MoveTo(target Vector, duration float64, easeType EaseType) {
	c.HasTarget = false
	c.Velocity = Vector{}
	if duration <= 0 {
		c.Position = target
		c.posTween.active = false
		c.matricesValid = false
		return
	}
	c.posTween = cameraVecTween{
		start:    c.Position,
		end:      target,
		duration: duration,
		easeType: easeType,
		active:   true,
	}
}

// SetZoom immediately sets the camera's zoom level (clamped to [MinZoom, MaxZoom]).
func (c *CameraComponent) SetZoom(zoom float64) {
	c.Zoom = c.clampZoom(zoom)
	c.zoomTween.active = false
	c.matricesValid = false
}

// ZoomTo smoothly transitions the camera zoom to the clamped target over duration using easeType.
func (c *CameraComponent) ZoomTo(zoom, duration float64, easeType EaseType) {
	zoom = c.clampZoom(zoom)
	if duration <= 0 {
		c.Zoom = zoom
		c.zoomTween.active = false
		c.matricesValid = false
		return
	}
	c.zoomTween = cameraScalarTween{
		start:    c.Zoom,
		end:      zoom,
		duration: duration,
		easeType: easeType,
		active:   true,
	}
}

// BumpZoom adds a temporary decaying zoom punch delta.
func (c *CameraComponent) BumpZoom(delta float64) {
	c.BumpZoomValue += delta
	c.matricesValid = false
}

// SetRotation immediately sets the camera's rotation in radians.
func (c *CameraComponent) SetRotation(rad float64) {
	c.Rotation = rad
	c.rotTween.active = false
	c.matricesValid = false
}

// RotateTo smoothly transitions the camera's rotation to rad over duration using easeType.
func (c *CameraComponent) RotateTo(rad, duration float64, easeType EaseType) {
	if duration <= 0 {
		c.Rotation = rad
		c.rotTween.active = false
		c.matricesValid = false
		return
	}
	c.rotTween = cameraScalarTween{
		start:    c.Rotation,
		end:      rad,
		duration: duration,
		easeType: easeType,
		active:   true,
	}
}

// SetBounds sets the camera's world bounds and enables bounds clamping.
func (c *CameraComponent) SetBounds(bounds Rectangle) {
	c.Bounds = bounds
	c.ClampBounds = true
}

// ClearBounds disables world bounds clamping.
func (c *CameraComponent) ClearBounds() {
	c.ClampBounds = false
}

// Shake triggers a positional camera shake with the given duration and amplitude.
func (c *CameraComponent) Shake(duration, power float64) {
	c.ShakeWithRotation(duration, power, 0)
}

// ShakeWithRotation triggers a positional and rotational camera shake.
func (c *CameraComponent) ShakeWithRotation(duration, power, rotPower float64) {
	if duration <= 0 || (power == 0 && rotPower == 0) {
		c.ShakeDuration = 0
		c.ShakeRemaining = 0
		c.ShakePower = 0
		c.ShakeRotation = 0
		c.shakeOffset = Vector{}
		c.shakeRot = 0
		c.matricesValid = false
		return
	}
	if !c.IsShaking() {
		c.shakeTime = 0
	}
	c.ShakeDuration = duration
	c.ShakeRemaining = duration
	c.ShakePower = power
	c.ShakeRotation = rotPower
}

// IsShaking reports whether the camera currently has an active shake effect.
func (c *CameraComponent) IsShaking() bool {
	return c.ShakeRemaining > 0
}

// Bump applies an instantaneous decaying world-space offset impulse.
func (c *CameraComponent) Bump(offset Vector) {
	c.BumpOffset = c.BumpOffset.Add(offset)
	c.matricesValid = false
}

// BumpAngular applies an instantaneous decaying offset impulse along angle (radians) by distance.
func (c *CameraComponent) BumpAngular(angle, distance float64) {
	c.BumpOffset = c.BumpOffset.Add(V(math.Cos(angle)*distance, math.Sin(angle)*distance))
	c.matricesValid = false
}

// SetViewportSize updates the camera's viewport dimensions.
func (c *CameraComponent) SetViewportSize(w, h float64) {
	c.ViewportWidth = w
	c.ViewportHeight = h
	c.matricesValid = false
}

// RebuildMatrices recomputes the camera's view matrix, inverse view matrix, and world-space view bounds.
func (c *CameraComponent) RebuildMatrices() {
	c.effectivePos = c.Position.Add(c.BumpOffset).Add(c.shakeOffset)
	c.effectiveZoom = c.Zoom + c.BumpZoomValue
	if c.effectiveZoom < 0.0001 {
		c.effectiveZoom = 0.0001
	}
	c.effectiveRot = c.Rotation + c.shakeRot

	var m Matrix
	m.Translate(-c.effectivePos.X, -c.effectivePos.Y)
	if c.effectiveRot != 0 {
		m.Rotate(-c.effectiveRot)
	}
	if c.effectiveZoom != 1 {
		m.Scale(c.effectiveZoom, c.effectiveZoom)
	}
	m.Translate(c.ViewportWidth*0.5, c.ViewportHeight*0.5)
	if c.PixelSnap && c.effectiveRot == 0 {
		m.SetElement(0, 2, math.Round(m.Element(0, 2)))
		m.SetElement(1, 2, math.Round(m.Element(1, 2)))
	}

	c.viewMatrix = m
	inv := m
	if inv.IsInvertible() {
		inv.Invert()
	}
	c.invViewMatrix = inv

	x0, y0 := inv.Apply(0, 0)
	x1, y1 := inv.Apply(c.ViewportWidth, 0)
	x2, y2 := inv.Apply(c.ViewportWidth, c.ViewportHeight)
	x3, y3 := inv.Apply(0, c.ViewportHeight)

	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))

	c.viewBounds = NewRectangle(minX, minY, maxX, maxY)
	c.matricesValid = true
}

// ViewMatrix returns the camera's world-to-screen view transformation matrix.
func (c *CameraComponent) ViewMatrix() Matrix {
	if !c.matricesValid {
		c.RebuildMatrices()
	}
	return c.viewMatrix
}

// InvViewMatrix returns the camera's screen-to-world inverse transformation matrix.
func (c *CameraComponent) InvViewMatrix() Matrix {
	if !c.matricesValid {
		c.RebuildMatrices()
	}
	return c.invViewMatrix
}

// ViewBounds returns the axis-aligned world-space bounding box of the camera's viewport.
func (c *CameraComponent) ViewBounds() Rectangle {
	if !c.matricesValid {
		c.RebuildMatrices()
	}
	return c.viewBounds
}

// WorldToScreen converts a world-space point into screen-space coordinates.
func (c *CameraComponent) WorldToScreen(worldPos Vector) Vector {
	return worldPos.Apply(c.ViewMatrix())
}

// ScreenToWorld converts a screen-space point into world-space coordinates.
func (c *CameraComponent) ScreenToWorld(screenPos Vector) Vector {
	return screenPos.Apply(c.InvViewMatrix())
}

// IsVisible reports whether the given world-space rectangle intersects the camera's view bounds.
func (c *CameraComponent) IsVisible(bounds Rectangle) bool {
	return c.ViewBounds().Intersects(bounds)
}

// IsPointVisible reports whether a world-space point (expanded by margin) is within the view bounds.
func (c *CameraComponent) IsPointVisible(point Vector, margin float64) bool {
	return c.ViewBounds().IntersectsCircle(point, math.Max(0, margin))
}
