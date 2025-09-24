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
	Target           *ebimath.Transform
	Scroller         *ebimath.Transform
	TargetOffset     ebimath.Vector
	RawFocus         ebimath.Vector
	ClampedFocus     ebimath.Vector
	OriginalViewport ebimath.Vector
	Viewport         ebimath.Vector
	LevelBounds      ebimath.Rectangle
	BaseZoom         float64
	TargetZoom       float64
	ZoomSpeed        float64
	DestZoom         float64
	BumpZoom         float64
	TrackingSpeed    float64
	DestPoint        ebimath.Vector
	BumpOffset       ebimath.Vector
	ShakePower       float64
	ClampBounds      bool
	Left             int
	Right            int
	Top              int
	Bottom           int
	Center           ebimath.Vector
	Cooldown         *managers.CooldownManager
}

// newBasicCamera creates a new basicCamera with the given viewport and scroller transform.
func newBasicCamera(viewport ebimath.Vector, scroller *ebimath.Transform) *basicCamera {
	if scroller == nil {
		scroller = ebimath.T()
	}
	return &basicCamera{
		Scroller:         scroller,
		TargetOffset:     ebimath.V2(0),
		Viewport:         viewport,
		OriginalViewport: viewport,
		BaseZoom:         1.0,
		TargetZoom:       1.0,
		ZoomSpeed:        0.0014,
		TrackingSpeed:    DefaultTrackingSpeed,
		ClampBounds:      true,
		Cooldown:         managers.NewCooldownManager(10),
	}
}

// SetZoomSpeed sets the speed of zoom interpolation.
func (self *basicCamera) SetZoomSpeed(speed float64) {
	self.ZoomSpeed = speed
}

// ZoomTo sets a target zoom level with smooth interpolation.
func (self *basicCamera) ZoomTo(value float64) {
	self.TargetZoom = ebimath.Clamp(value, MinZoom, MaxZoom)
}

// ForceZoom sets the zoom level immediately without interpolation.
func (self *basicCamera) ForceZoom(value float64) {
	value = ebimath.Clamp(value, MinZoom, MaxZoom)
	self.BaseZoom = value
	self.TargetZoom = value
	self.DestZoom = 0
}

// SetTrackingSpeed sets the speed at which the camera follows its target.
func (self *basicCamera) SetTrackingSpeed(speed float64) {
	self.TrackingSpeed = speed
}

// TrackObject sets a target to follow with optional immediate centering and offset.
func (self *basicCamera) TrackObject(target *ebimath.Transform, immediateFocus bool, trackingSpeed float64, offset ebimath.Vector) {
	self.Target = target
	if trackingSpeed > 0 {
		self.TrackingSpeed = trackingSpeed
	}
	self.TargetOffset = offset
	if self.Target != nil && (immediateFocus || self.RawFocus.IsZero()) {
		self.CenterOnTarget()
	}
}

// CenterOnTarget instantly moves the camera to the target’s position.
func (self *basicCamera) CenterOnTarget() {
	if self.Target != nil {
		self.RawFocus = self.Target.Position().Add(self.TargetOffset.Scale(self.Target.Scale()))
	}
}

// StopTrackObject stops tracking the current target.
func (self *basicCamera) StopTrackObject() {
	self.Target = nil
}

// SetLevelBounds defines the level boundaries for clamping.
func (self *basicCamera) SetLevelBounds(bounds ebimath.Rectangle) {
	self.LevelBounds = bounds
	self.ClampBounds = true
}

// SetClampToLevelBounds toggles clamping to level bounds.
func (self *basicCamera) SetClampToLevelBounds(clamp bool) {
	self.ClampBounds = clamp
}

// Bump applies a temporary position offset in world coordinates.
func (self *basicCamera) Bump(v ebimath.Vector) {
	self.BumpOffset = self.BumpOffset.Add(v)
}

// BumpAngular applies a position offset by angle and distance in world coordinates.
func (self *basicCamera) BumpAngular(angle, distance float64) {
	self.BumpOffset = self.BumpOffset.Add(ebimath.V(math.Cos(angle)*distance, math.Sin(angle)*distance))
}

// Shake applies a shake effect with the given duration and power.
func (self *basicCamera) Shake(duration, power float64) {
	self.ShakePower = power
	if power != 0 {
		self.Cooldown.Set("shaking", duration, nil)
	}
}

// IsShaking returns true if the camera is currently shaking.
func (self *basicCamera) IsShaking() bool {
	return self.Cooldown.Has("shaking")
}

func (self *basicCamera) Update(deltaTime float64) {
	frameCount = (frameCount % math.MaxInt) + 1
	zoom := self.BaseZoom + self.BumpZoom
	self.Viewport = ebimath.V(self.OriginalViewport.X/zoom, self.OriginalViewport.Y/zoom)
	self.Cooldown.Update(deltaTime)
	if zoomDiff := self.TargetZoom - self.BaseZoom; zoomDiff != 0 {
		zoomStep := self.ZoomSpeed * deltaTime
		self.DestZoom = ebimath.Clamp(self.DestZoom+ebimath.Sign(zoomDiff)*zoomStep, math.Min(zoomDiff, 0), math.Max(zoomDiff, 0))
		self.BaseZoom += self.DestZoom
		self.DestZoom *= math.Pow(ZoomFriction, deltaTime)
	}
	self.BumpZoom *= math.Pow(ZoomFriction, deltaTime)
	if self.Target != nil {
		targetPos := self.Target.Position().Add(self.TargetOffset.Scale(self.Target.Scale()))
		angle := float64(self.RawFocus.AngleToPoint(targetPos))
		distX := math.Abs(targetPos.X - self.RawFocus.X)
		distY := math.Abs(targetPos.Y - self.RawFocus.Y)
		spdX := TrackingSpeedFactorX * self.TrackingSpeed * zoom
		spdY := TrackingSpeedFactorY * self.TrackingSpeed * zoom
		if distX >= DeadZonePercentX*self.Viewport.X {
			deltaX := 0.8*distX - DeadZonePercentX*self.Viewport.X
			self.DestPoint.X += math.Cos(angle) * deltaX * spdX * deltaTime
		}
		if distY >= DeadZonePercentY*self.Viewport.Y {
			deltaY := 0.8*distY - DeadZonePercentY*self.Viewport.Y
			self.DestPoint.Y += math.Sin(angle) * deltaY * spdY * deltaTime
		}
	}
	frict := BaseFriction - self.TrackingSpeed*zoom*FrictionTrackingMod*BaseFriction
	if self.ClampBounds {
		self.applyBreaking(&frict, deltaTime)
	}
	self.RawFocus = self.RawFocus.Add(self.DestPoint.ScaleF(deltaTime))
	self.DestPoint = self.DestPoint.ScaleF(math.Pow(frict, deltaTime))
	self.BumpOffset = self.BumpOffset.ScaleF(math.Pow(BumpFriction, deltaTime))
	if self.ClampBounds {
		self.ClampedFocus.X = ebimath.Clamp(self.RawFocus.X, self.Viewport.X*0.5, math.Max(self.LevelBounds.Width()-self.Viewport.X*0.5, self.Viewport.X*0.5))
		self.ClampedFocus.Y = ebimath.Clamp(self.RawFocus.Y, self.Viewport.Y*0.5, math.Max(self.LevelBounds.Height()-self.Viewport.Y*0.5, self.Viewport.Y*0.5))
	} else {
		self.ClampedFocus = self.RawFocus
	}
	self.apply()
}

func (self *basicCamera) applyBreaking(frict *float64, deltaTime float64) {
	brakeDistX := BreakDistance * self.Viewport.X
	if self.DestPoint.X != 0 {
		var brakeRatio float64
		if self.DestPoint.X < 0 {
			brakeRatio = 1 - ebimath.Clamp((self.RawFocus.X-self.Viewport.X*0.5)/brakeDistX, 0, 1)
		} else {
			brakeRatio = 1 - ebimath.Clamp((self.LevelBounds.Width()-self.Viewport.X*0.5-self.RawFocus.X)/brakeDistX, 0, 1)
		}
		*frict *= 1 - 0.9*brakeRatio*deltaTime
	}
	brakeDistY := BreakDistance * self.Viewport.Y
	if self.DestPoint.Y != 0 {
		var brakeRatio float64
		if self.DestPoint.Y < 0 {
			brakeRatio = 1 - ebimath.Clamp((self.RawFocus.Y-self.Viewport.Y*0.5)/brakeDistY, 0, 1)
		} else {
			brakeRatio = 1 - ebimath.Clamp((self.LevelBounds.Height()-self.Viewport.Y*0.5-self.RawFocus.Y)/brakeDistY, 0, 1)
		}
		*frict *= 1 - 0.9*brakeRatio*deltaTime
	}
}

func (self *basicCamera) apply() {
	zoom := self.BaseZoom + self.BumpZoom
	screenCenter := ebimath.V(self.OriginalViewport.X/2, self.OriginalViewport.Y/2)
	t := screenCenter.Sub(self.ClampedFocus.ScaleF(zoom))
	t = t.Sub(self.BumpOffset.ScaleF(zoom))
	if self.Cooldown.Has("shaking") {
		ftime := float64(frameCount)
		shakeAmount := self.ShakePower * self.Cooldown.GetRatio("shaking")
		shakeOffset := ebimath.V(
			math.Cos(ftime*ShakeFreqX)*ShakeAmp*shakeAmount,
			math.Sin(0.3+ftime*ShakeFreqY)*ShakeAmp*shakeAmount,
		)
		t = t.Add(shakeOffset.ScaleF(zoom))
		if self.Cooldown.GetRatio("shaking") == 0 {
			self.ShakePower = 0
		}
	}
	self.Scroller.SetPosition(t.Round())
	self.Scroller.SetScale(ebimath.V2(zoom))
	self.Left = int(self.ClampedFocus.X - self.Viewport.X*0.5)
	self.Right = self.Left + int(self.Viewport.X-1)
	self.Top = int(self.ClampedFocus.Y - self.Viewport.Y*0.5)
	self.Bottom = self.Top + int(self.Viewport.Y-1)
	self.Center = ebimath.V(float64(self.Left+self.Right)*0.5, float64(self.Top+self.Bottom)*0.5)
}

func (self *basicCamera) Area() ebimath.Rectangle {
	return ebimath.NewRectangle(
		float64(self.Left),
		float64(self.Top),
		float64(self.Right-self.Left+1),
		float64(self.Bottom-self.Top+1),
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
	FollowTarget   *ebimath.Vector
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
	self.PositionX = Interpolator{Current: pos.X, Target: pos.X, Mode: Instant}
	self.PositionY = Interpolator{Current: pos.Y, Target: pos.Y, Mode: Instant}
	self.Zoom = Interpolator{Current: 1.0, Target: 1.0, Mode: Instant}
	self.Rotation = Interpolator{Current: t.Rotation(), Target: t.Rotation(), Mode: Instant}
	return self
}

// Follow sets the camera to smoothly follow a target position using Spring interpolation.
func (self *camera) Follow(target *ebimath.Vector) {
	self.FollowTarget = target
	if target != nil {
		self.PositionX.SetTarget(target.X, Spring, 0)
		self.PositionY.SetTarget(target.Y, Spring, 0)
	}
}

// StopFollowing stops following any target and sets position interpolators to Instant.
func (self *camera) StopFollowing() {
	self.FollowTarget = nil
	self.PositionX.Mode = Instant
	self.PositionY.Mode = Instant
	self.PositionX.Current = self.PositionX.Target
	self.PositionY.Current = self.PositionY.Target
}

// Update advances the camera’s interpolators and applies the results to the transform.
func (self *camera) Update(deltaTime float64) {
	if self.FollowTarget != nil {
		self.PositionX.Target = self.FollowTarget.X
		self.PositionY.Target = self.FollowTarget.Y
	}
	self.PositionX.Update(deltaTime)
	self.PositionY.Update(deltaTime)
	self.Zoom.Update(deltaTime)
	self.Rotation.Update(deltaTime)
	self.CurrentZoom = ebimath.Clamp(self.Zoom.Current, MinZoom, MaxZoom)
	self.SetPosition(ebimath.V(self.PositionX.Current, self.PositionY.Current))
	self.SetRotation(self.Rotation.Current)
}

// WorldToScreen converts a world position to screen coordinates.
func (self *camera) WorldToScreen(worldPos ebimath.Vector) ebimath.Vector {
	screenCenter := ebimath.V(float64(self.ViewportWidth)/2, float64(self.ViewportHeight)/2)
	relativePos := worldPos.Sub(self.Position())
	rotatedPos := relativePos.Rotate(-self.Transform.Rotation())
	scaledPos := rotatedPos.ScaleF(self.CurrentZoom)
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
	self.FollowTarget = nil
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
	self.FollowTarget = nil
	newPos := self.Position().Add(delta)
	self.PositionX.SetTarget(newPos.X, Instant, 0)
	self.PositionY.SetTarget(newPos.Y, Instant, 0)
	self.SetPosition(newPos)
}

// SetPositionInstant instantly sets the camera’s position and stops following.
func (self *camera) SetPositionInstant(position ebimath.Vector) {
	self.FollowTarget = nil
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
	Current   float64
	Target    float64
	Mode      InterpolationMode
	Duration  float64
	Elapsed   float64
	Velocity  float64
	Start     float64
	Stiffness float64
	Damping   float64
}

// SetTarget sets a new target value with the specified mode and duration.
func (self *Interpolator) SetTarget(newTarget float64, mode InterpolationMode, duration float64) {
	self.Target = newTarget
	self.Mode = mode
	switch self.Mode {
	case Instant:
		self.Current = newTarget
		self.Velocity = 0
	case Spring:
		if self.Stiffness == 0 {
			self.Stiffness = 100.0
		}
		if self.Damping == 0 {
			self.Damping = 10.0
		}
	default:
		if duration <= 0 {
			duration = 0.001
		}
		self.Start = self.Current
		self.Duration = duration
		self.Elapsed = 0
		self.Velocity = 0
	}
}

// Update advances the interpolation based on elapsed time (deltaTime in seconds).
func (self *Interpolator) Update(deltaTime float64) {
	switch self.Mode {
	case Instant:
		return
	case Spring:
		displacement := self.Target - self.Current
		if math.Abs(displacement) < 0.001 && math.Abs(self.Velocity) < 0.001 {
			self.Current = self.Target
			self.Velocity = 0
			return
		}
		acceleration := self.Stiffness*displacement - self.Damping*self.Velocity
		self.Velocity += acceleration * deltaTime
		self.Current += self.Velocity * deltaTime
	default:
		self.Elapsed += deltaTime
		if self.Elapsed >= self.Duration {
			self.Current = self.Target
			self.Mode = Instant
			return
		}
		t := self.Elapsed / self.Duration
		easing := easingFunc(self.Mode, t)
		self.Current = self.Start + (self.Target-self.Start)*easing
	}
}

// BasicCameraComponent is a component that wraps a basicCamera.
type BasicCameraComponent struct {
	*basicCamera
}

func (self *BasicCameraComponent) Init(viewport ebimath.Vector, scroller *ebimath.Transform) {
	self.basicCamera = newBasicCamera(viewport, scroller)
}

// CameraComponent is a component that wraps a camera.
type CameraComponent struct {
	*camera
}

func (self *CameraComponent) Init(viewportWidth, viewportHeight int) {
	self.camera = newCamera(viewportWidth, viewportHeight)
}
