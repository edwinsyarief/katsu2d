package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/edwinsyarief/katsu2d/managers"
)

var (
	// frameCount is a global frame counter.
	frameCount = 0
)

// Camera constants
const (
	// MinZoom is the minimum zoom level.
	MinZoom = 1.0
	// MaxZoom is the maximum zoom level.
	MaxZoom = 10.0
	// BaseFriction is the base movement friction.
	BaseFriction = 0.8
	// BumpFriction is the friction for bump effects.
	BumpFriction = 0.85
	// ZoomFriction is the friction for zoom interpolation.
	ZoomFriction = 0.9
	// DefaultTrackingSpeed is the default speed for tracking.
	DefaultTrackingSpeed = 1.0
	// BreakDistance is the distance for braking near bounds.
	BreakDistance = 0.1
	// DeadZonePercentX is the X-axis dead zone percentage.
	DeadZonePercentX = 0.04
	// DeadZonePercentY is the Y-axis dead zone percentage.
	DeadZonePercentY = 0.10
	// TrackingSpeedFactorX is the X-axis tracking speed factor.
	TrackingSpeedFactorX = 0.015
	// TrackingSpeedFactorY is the Y-axis tracking speed factor.
	TrackingSpeedFactorY = 0.050
	// FrictionTrackingMod is the friction modifier for tracking.
	FrictionTrackingMod = 0.027
	// ShakeFreqX is the X-axis shake frequency.
	ShakeFreqX = 1.1
	// ShakeFreqY is the Y-axis shake frequency.
	ShakeFreqY = 1.7
	// ShakeAmp is the shake amplitude.
	ShakeAmp = 2.5
)

// basicCamera implements a 2D camera with smooth tracking, zoom, bump, and shake effects.
// It follows a target transform, clamps to level bounds, and applies transformations to a scroller.
type basicCamera struct {
	target           *ebimath.Transform
	scroller         *ebimath.Transform
	targetOffset     ebimath.Vector
	rawFocus         ebimath.Vector
	clampedFocus     ebimath.Vector
	originalViewport ebimath.Vector
	viewport         ebimath.Vector
	levelBounds      ebimath.Rectangle
	baseZoom         float64
	targetZoom       float64
	zoomSpeed        float64
	destZoom         float64
	bumpZoom         float64
	trackingSpeed    float64
	destPoint        ebimath.Vector
	bumpOffset       ebimath.Vector
	shakePower       float64
	clampBounds      bool
	left             int
	right            int
	top              int
	bottom           int
	center           ebimath.Vector
	cd               *managers.CooldownManager
}

// newBasicCamera creates a new basicCamera with the given viewport and scroller transform.
func newBasicCamera(viewport ebimath.Vector, scroller *ebimath.Transform) *basicCamera {
	if scroller == nil {
		scroller = ebimath.T()
	}
	return &basicCamera{
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
func (self *basicCamera) GetViewport() ebimath.Vector {
	return self.viewport
}

// GetZoom returns the current zoom level (base + bump).
func (self *basicCamera) GetZoom() float64 {
	return self.baseZoom + self.bumpZoom
}

// SetZoomSpeed sets the speed of zoom interpolation.
func (self *basicCamera) SetZoomSpeed(speed float64) {
	self.zoomSpeed = speed
}

// ZoomTo sets a target zoom level with smooth interpolation.
func (self *basicCamera) ZoomTo(value float64) {
	self.targetZoom = ebimath.Clamp(value, MinZoom, MaxZoom)
}

// ForceZoom sets the zoom level immediately without interpolation.
func (self *basicCamera) ForceZoom(value float64) {
	value = ebimath.Clamp(value, MinZoom, MaxZoom)
	self.baseZoom = value
	self.targetZoom = value
	self.destZoom = 0
}

// BumpZoom applies a temporary zoom offset that decays over time.
func (self *basicCamera) BumpZoom(value float64) {
	self.bumpZoom = value
}

// SetTrackingSpeed sets the speed at which the camera follows its target.
func (self *basicCamera) SetTrackingSpeed(speed float64) {
	self.trackingSpeed = speed
}

// TrackObject sets a target to follow with optional immediate centering and offset.
func (self *basicCamera) TrackObject(target *ebimath.Transform, immediateFocus bool, trackingSpeed float64, offset ebimath.Vector) {
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
func (self *basicCamera) CenterOnTarget() {
	if self.target != nil {
		self.rawFocus = self.target.Position().Add(self.targetOffset.Mul(self.target.Scale()))
	}
}

// StopTrackObject stops tracking the current target.
func (self *basicCamera) StopTrackObject() {
	self.target = nil
}

// SetLevelBounds defines the level boundaries for clamping.
func (self *basicCamera) SetLevelBounds(bounds ebimath.Rectangle) {
	self.levelBounds = bounds
	self.clampBounds = true
}

// SetClampToLevelBounds toggles clamping to level bounds.
func (self *basicCamera) SetClampToLevelBounds(clamp bool) {
	self.clampBounds = clamp
}

// Bump applies a temporary position offset in world coordinates.
func (self *basicCamera) Bump(v ebimath.Vector) {
	self.bumpOffset = self.bumpOffset.Add(v)
}

// BumpAngular applies a position offset by angle and distance in world coordinates.
func (self *basicCamera) BumpAngular(angle, distance float64) {
	self.bumpOffset = self.bumpOffset.Add(ebimath.V(math.Cos(angle)*distance, math.Sin(angle)*distance))
}

// Shake applies a shake effect with the given duration and power.
func (self *basicCamera) Shake(duration, power float64) {
	self.shakePower = power
	if power != 0 {
		self.cd.Set("shaking", duration, nil)
	}
}

// IsShaking returns true if the camera is currently shaking.
func (self *basicCamera) IsShaking() bool {
	return self.cd.Has("shaking")
}

func (self *basicCamera) Update(deltaTime float64) {
	frameCount = (frameCount % math.MaxInt) + 1
	zoom := self.GetZoom()
	self.viewport = ebimath.V(self.originalViewport.X/zoom, self.originalViewport.Y/zoom)
	self.cd.Update(deltaTime)
	if zoomDiff := self.targetZoom - self.baseZoom; zoomDiff != 0 {
		zoomStep := self.zoomSpeed * deltaTime
		self.destZoom = ebimath.Clamp(self.destZoom+ebimath.Sign(zoomDiff)*zoomStep, math.Min(zoomDiff, 0), math.Max(zoomDiff, 0))
		self.baseZoom += self.destZoom
		self.destZoom *= math.Pow(ZoomFriction, deltaTime)
	}
	self.bumpZoom *= math.Pow(ZoomFriction, deltaTime)
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
	frict := BaseFriction - self.trackingSpeed*zoom*FrictionTrackingMod*BaseFriction
	if self.clampBounds {
		self.applyBreaking(&frict, deltaTime)
	}
	self.rawFocus = self.rawFocus.Add(self.destPoint.MulF(deltaTime))
	self.destPoint = self.destPoint.MulF(math.Pow(frict, deltaTime))
	self.bumpOffset = self.bumpOffset.MulF(math.Pow(BumpFriction, deltaTime))
	if self.clampBounds {
		self.clampedFocus.X = ebimath.Clamp(self.rawFocus.X, self.viewport.X*0.5, math.Max(self.levelBounds.Width()-self.viewport.X*0.5, self.viewport.X*0.5))
		self.clampedFocus.Y = ebimath.Clamp(self.rawFocus.Y, self.viewport.Y*0.5, math.Max(self.levelBounds.Height()-self.viewport.Y*0.5, self.viewport.Y*0.5))
	} else {
		self.clampedFocus = self.rawFocus
	}
	self.apply()
}

func (self *basicCamera) applyBreaking(frict *float64, deltaTime float64) {
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

func (self *basicCamera) apply() {
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
			self.shakePower = 0
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

func (self *basicCamera) Area() ebimath.Rectangle {
	return ebimath.NewRectangle(
		float64(self.left),
		float64(self.top),
		float64(self.right-self.left+1),
		float64(self.bottom-self.top+1),
	)
}

// camera represents a 2D camera with smooth interpolation for position, zoom, and rotation.
type camera struct {
	*ebimath.Transform
	CurrentZoom    float64
	PositionX      Interpolator
	PositionY      Interpolator
	Zoom           Interpolator
	Rotation       Interpolator
	ViewportWidth  int
	ViewportHeight int
	followTarget   *ebimath.Vector
}

// newCamera creates a new camera with default values and initialized interpolators.
func newCamera(viewportWidth, viewportHeight int) *camera {
	t := ebimath.T()
	pos := t.Position()
	self := &camera{
		Transform:      t,
		CurrentZoom:    1.0,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
	self.PositionX = Interpolator{current: pos.X, target: pos.X, mode: Instant}
	self.PositionY = Interpolator{current: pos.Y, target: pos.Y, mode: Instant}
	self.Zoom = Interpolator{current: 1.0, target: 1.0, mode: Instant}
	self.Rotation = Interpolator{current: t.Rotation(), target: t.Rotation(), mode: Instant}
	return self
}

// Follow sets the camera to smoothly follow a target position using Spring interpolation.
func (self *camera) Follow(target *ebimath.Vector) {
	self.followTarget = target
	if target != nil {
		self.PositionX.SetTarget(target.X, Spring, 0)
		self.PositionY.SetTarget(target.Y, Spring, 0)
	}
}

// StopFollowing stops following any target and sets position interpolators to Instant.
func (self *camera) StopFollowing() {
	self.followTarget = nil
	self.PositionX.mode = Instant
	self.PositionY.mode = Instant
	self.PositionX.current = self.PositionX.target
	self.PositionY.current = self.PositionY.target
}

// Update advances the camera’s interpolators and applies the results to the transform.
func (self *camera) Update(deltaTime float64) {
	if self.followTarget != nil {
		self.PositionX.target = self.followTarget.X
		self.PositionY.target = self.followTarget.Y
	}
	self.PositionX.Update(deltaTime)
	self.PositionY.Update(deltaTime)
	self.Zoom.Update(deltaTime)
	self.Rotation.Update(deltaTime)
	self.CurrentZoom = ebimath.Clamp(self.Zoom.current, MinZoom, MaxZoom)
	self.SetPosition(ebimath.V(self.PositionX.current, self.PositionY.current))
	self.SetRotation(self.Rotation.current)
}

// WorldToScreen converts a world position to screen coordinates.
func (self *camera) WorldToScreen(worldPos ebimath.Vector) ebimath.Vector {
	screenCenter := ebimath.V(float64(self.ViewportWidth)/2, float64(self.ViewportHeight)/2)
	relativePos := worldPos.Sub(self.Position())
	rotatedPos := relativePos.Rotate(-self.Transform.Rotation())
	scaledPos := rotatedPos.MulF(self.CurrentZoom)
	return screenCenter.Add(scaledPos)
}

// ScreenToWorld converts a screen position to world coordinates.
func (self *camera) ScreenToWorld(screenPos ebimath.Vector) ebimath.Vector {
	screenCenter := ebimath.V(float64(self.ViewportWidth)/2, float64(self.ViewportHeight)/2)
	relativePos := screenPos.Sub(screenCenter)
	unscaledPos := relativePos.DivF(self.CurrentZoom)
	unrotatedPos := unscaledPos.Rotate(self.Transform.Rotation())
	return self.Position().Add(unrotatedPos)
}

// SetTargetPosition sets a new target position with interpolation.
// This will stop any direct movement or following.
func (self *camera) SetTargetPosition(position ebimath.Vector, mode InterpolationMode, duration float64) {
	self.followTarget = nil
	self.PositionX.SetTarget(position.X, mode, duration)
	self.PositionY.SetTarget(position.Y, mode, duration)
}

// SetTargetZoom sets a new target zoom level with interpolation.
func (self *camera) SetTargetZoom(zoom float64, mode InterpolationMode, duration float64) {
	zoom = ebimath.Clamp(zoom, MinZoom, MaxZoom)
	self.Zoom.SetTarget(zoom, mode, duration)
}

// SetTargetRotation sets a new target rotation with interpolation.
func (self *camera) SetTargetRotation(rotation float64, mode InterpolationMode, duration float64) {
	self.Rotation.SetTarget(rotation, mode, duration)
}

// Move moves the camera by a delta vector with interpolation.
// This function is intended for one-shot, interpolated movements to a new relative position.
// For continuous movement (e.g., player input), use AddPosition.
func (self *camera) Move(delta ebimath.Vector, mode InterpolationMode, duration float64) {
	currentPos := self.Position()
	self.SetTargetPosition(currentPos.Add(delta), mode, duration)
}

// AddPosition directly adds a delta vector to the camera's current position.
// This is suitable for continuous, player-controlled movement.
// It will stop any active position interpolation or following.
func (self *camera) AddPosition(delta ebimath.Vector) {
	self.followTarget = nil
	newPos := self.Position().Add(delta)
	self.PositionX.SetTarget(newPos.X, Instant, 0)
	self.PositionY.SetTarget(newPos.Y, Instant, 0)
	self.SetPosition(newPos)
}

// SetPositionInstant instantly sets the camera’s position and stops following.
func (self *camera) SetPositionInstant(position ebimath.Vector) {
	self.followTarget = nil
	self.PositionX.SetTarget(position.X, Instant, 0)
	self.PositionY.SetTarget(position.Y, Instant, 0)
	self.SetPosition(position)
}

// SetZoomInstant instantly sets the camera’s zoom level.
func (self *camera) SetZoomInstant(zoom float64) {
	zoom = ebimath.Clamp(zoom, MinZoom, MaxZoom)
	self.Zoom.SetTarget(zoom, Instant, 0)
	self.CurrentZoom = zoom
}

// Rotate rotates the camera by an angle with interpolation.
func (self *camera) Rotate(angle float64, mode InterpolationMode, duration float64) {
	currentRotation := self.Transform.Rotation()
	self.SetTargetRotation(currentRotation+angle, mode, duration)
}

// SetRotationInstant instantly sets the camera’s rotation.
func (self *camera) SetRotationInstant(rotation float64) {
	self.Rotation.SetTarget(rotation, Instant, 0)
	self.SetRotation(rotation)
}

func (self *camera) Area() ebimath.Rectangle {
	halfWidth := float64(self.ViewportWidth) / (2 * self.CurrentZoom)
	halfHeight := float64(self.ViewportHeight) / (2 * self.CurrentZoom)
	topLeft := ebimath.V(-halfWidth, -halfHeight)
	topRight := ebimath.V(halfWidth, -halfHeight)
	bottomLeft := ebimath.V(-halfWidth, halfHeight)
	bottomRight := ebimath.V(halfWidth, halfHeight)
	rotation := self.Transform.Rotation()
	topLeft = topLeft.Rotate(rotation)
	topRight = topRight.Rotate(rotation)
	bottomLeft = bottomLeft.Rotate(rotation)
	bottomRight = bottomRight.Rotate(rotation)
	cameraPos := self.Position()
	topLeft = topLeft.Add(cameraPos)
	topRight = topRight.Add(cameraPos)
	bottomLeft = bottomLeft.Add(cameraPos)
	bottomRight = bottomRight.Add(cameraPos)
	minX := ebimath.Min(ebimath.Min(topLeft.X, topRight.X), ebimath.Min(bottomLeft.X, bottomRight.X))
	minY := ebimath.Min(ebimath.Min(topLeft.Y, topRight.Y), ebimath.Min(bottomLeft.Y, bottomRight.Y))
	maxX := ebimath.Max(ebimath.Max(topLeft.X, topRight.X), ebimath.Max(bottomLeft.X, bottomRight.X))
	maxY := ebimath.Max(ebimath.Max(topLeft.Y, topRight.Y), ebimath.Max(bottomLeft.Y, bottomRight.Y))
	return ebimath.NewRectangle(minX, minY, maxX, maxY)
}

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

func easingFunc(mode InterpolationMode, t float64) float64 {
	t = ebimath.Clamp(t, 0, 1)
	if fn, ok := easingFuncs[mode]; ok {
		return float64(fn(float32(t), 0, 1, 1))
	}
	return t
}

// Interpolator manages smooth transitions for a single scalar value (e.g., position, zoom).
type Interpolator struct {
	current   float64
	target    float64
	mode      InterpolationMode
	duration  float64
	elapsed   float64
	velocity  float64
	start     float64
	stiffness float64
	damping   float64
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
		if self.stiffness == 0 {
			self.stiffness = 100.0
		}
		if self.damping == 0 {
			self.damping = 10.0
		}
	default:
		if duration <= 0 {
			duration = 0.001
		}
		self.start = self.current
		self.duration = duration
		self.elapsed = 0
		self.velocity = 0
	}
}

// Update advances the interpolation based on elapsed time (deltaTime in seconds).
func (self *Interpolator) Update(deltaTime float64) {
	switch self.mode {
	case Instant:
		return
	case Spring:
		displacement := self.target - self.current
		if math.Abs(displacement) < 0.001 && math.Abs(self.velocity) < 0.001 {
			self.current = self.target
			self.velocity = 0
			return
		}
		acceleration := self.stiffness*displacement - self.damping*self.velocity
		self.velocity += acceleration * deltaTime
		self.current += self.velocity * deltaTime
	default:
		self.elapsed += deltaTime
		if self.elapsed >= self.duration {
			self.current = self.target
			self.mode = Instant
			return
		}
		t := self.elapsed / self.duration
		easing := easingFunc(self.mode, t)
		self.current = self.start + (self.target-self.start)*easing
	}
}

// BasicCameraComponent is a component that wraps a basicCamera.
type BasicCameraComponent struct {
	*basicCamera
}

// NewBasicCameraComponent creates a new BasicCameraComponent.
func NewBasicCameraComponent(viewport ebimath.Vector, scroller *ebimath.Transform) *BasicCameraComponent {
	return &BasicCameraComponent{
		basicCamera: newBasicCamera(viewport, scroller),
	}
}

// CameraComponent is a component that wraps a camera.
type CameraComponent struct {
	*camera
}

// NewCameraComponent creates a new CameraComponent.
func NewCameraComponent(viewportWidth, viewportHeight int) *CameraComponent {
	return &CameraComponent{
		camera: newCamera(viewportWidth, viewportHeight),
	}
}
