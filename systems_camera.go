package katsu2d

import (
	"math"

	"github.com/edwinsyarief/teishoku"
)

// CameraSystem updates CameraComponent entities (tweens, bump/shake decay,
// target following, bounds clamping, and matrix rebuilding) and syncs the
// first active camera into the world's ActiveCamera resource.
type CameraSystem struct {
	filter           *teishoku.Filter[CameraComponent]
	targetBuilder    *teishoku.Builder[TransformComponent]
	activeCamHandle  teishoku.ResourceHandle[ActiveCamera]
	pendingW         int
	pendingH         int
	hasPendingResize bool
	initialized      bool
}

// NewCameraSystem creates a new CameraSystem.
func NewCameraSystem() *CameraSystem {
	return &CameraSystem{}
}

// Initialize prepares the CameraSystem's ECS filter, target builder, and ActiveCamera resource.
func (s *CameraSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	res := w.Resources()
	if ok, _ := teishoku.HasResource[ActiveCamera](res); !ok {
		teishoku.AddResource(res, &ActiveCamera{})
	}
	s.activeCamHandle = teishoku.UseResource[ActiveCamera](res)
	s.filter = teishoku.NewFilter[CameraComponent](w)
	s.targetBuilder = teishoku.NewBuilder[TransformComponent](w)
	if info := GetHiResDisplayInfo(w); info != nil && info.Width > 0 && info.Height > 0 {
		s.pendingW, s.pendingH, s.hasPendingResize = info.Width, info.Height, true
	}
	Subscribe(w, s.onLayoutChanged)
	s.initialized = true
}

func (s *CameraSystem) onLayoutChanged(evt EngineLayoutChangedEvent) {
	if evt.Width > 0 && evt.Height > 0 {
		s.pendingW = evt.Width
		s.pendingH = evt.Height
		s.hasPendingResize = true
	}
}

// Update advances all cameras by dt seconds and syncs the active camera resource.
func (s *CameraSystem) Update(w *teishoku.World, dt float64) {
	if w == nil {
		return
	}
	if !s.initialized {
		s.Initialize(w)
	}

	if s.hasPendingResize {
		wFloat, hFloat := float64(s.pendingW), float64(s.pendingH)
		s.filter.EachChunk(func(cams []CameraComponent) {
			for i := range cams {
				cam := &cams[i]
				if cam.AutoResizeViewport {
					cam.SetViewportSize(wFloat, hFloat)
				}
			}
		})
		s.hasPendingResize = false
	}

	active := s.activeCamHandle.Get()
	foundActive := false

	s.filter.EachChunk(func(cams []CameraComponent) {
		for i := range cams {
			cam := &cams[i]
			updateCameraTweens(cam, dt)
			updateCameraEffects(cam, dt)

			if cam.HasTarget {
				if target := s.targetBuilder.Get(cam.TargetEntity); target != nil {
					updateCameraFollow(cam, target, dt)
				}
			}

			clampCameraToBounds(cam)
			cam.RebuildMatrices()

			if cam.Active && !foundActive && active != nil {
				foundActive = true
				active.ViewMatrix = cam.viewMatrix
				active.InvViewMatrix = cam.invViewMatrix
				active.ViewBounds = cam.viewBounds
				active.Position = cam.effectivePos
				active.Zoom = cam.effectiveZoom
				active.Rotation = cam.effectiveRot
				active.ViewportWidth = cam.ViewportWidth
				active.ViewportHeight = cam.ViewportHeight
				active.HasCamera = true
			}
		}
	})

	if !foundActive && active != nil {
		active.HasCamera = false
	}
}

func updateCameraTweens(cam *CameraComponent, dt float64) {
	if cam.posTween.active {
		if cam.posTween.duration <= 0 {
			cam.Position = cam.posTween.end
			cam.posTween.active = false
		} else {
			cam.posTween.elapsed += dt
			if cam.posTween.elapsed >= cam.posTween.duration {
				cam.Position = cam.posTween.end
				cam.posTween.active = false
			} else {
				factor := EaseTypes[float64](cam.posTween.easeType)(cam.posTween.elapsed, 0, 1, cam.posTween.duration)
				cam.Position = cam.posTween.start.Lerp(cam.posTween.end, factor)
			}
		}
	}

	if cam.zoomTween.active {
		if cam.zoomTween.duration <= 0 {
			cam.Zoom = cam.clampZoom(cam.zoomTween.end)
			cam.zoomTween.active = false
		} else {
			cam.zoomTween.elapsed += dt
			if cam.zoomTween.elapsed >= cam.zoomTween.duration {
				cam.Zoom = cam.clampZoom(cam.zoomTween.end)
				cam.zoomTween.active = false
			} else {
				val := EaseTypes[float64](cam.zoomTween.easeType)(
					cam.zoomTween.elapsed,
					cam.zoomTween.start,
					cam.zoomTween.end-cam.zoomTween.start,
					cam.zoomTween.duration,
				)
				cam.Zoom = cam.clampZoom(val)
			}
		}
	}

	if cam.rotTween.active {
		if cam.rotTween.duration <= 0 {
			cam.Rotation = cam.rotTween.end
			cam.rotTween.active = false
		} else {
			cam.rotTween.elapsed += dt
			if cam.rotTween.elapsed >= cam.rotTween.duration {
				cam.Rotation = cam.rotTween.end
				cam.rotTween.active = false
			} else {
				cam.Rotation = EaseTypes[float64](cam.rotTween.easeType)(
					cam.rotTween.elapsed,
					cam.rotTween.start,
					cam.rotTween.end-cam.rotTween.start,
					cam.rotTween.duration,
				)
			}
		}
	}
}

func updateCameraEffects(cam *CameraComponent, dt float64) {
	if cam.BumpZoomValue != 0 {
		damping := cam.BumpZoomDamping
		if damping <= 0 {
			damping = 10.0
		}
		cam.BumpZoomValue *= math.Exp(-damping * dt)
		if math.Abs(cam.BumpZoomValue) < 1e-5 {
			cam.BumpZoomValue = 0
		}
	}

	if !cam.BumpOffset.IsZero() {
		damping := cam.BumpDamping
		if damping <= 0 {
			damping = 10.0
		}
		decay := math.Exp(-damping * dt)
		cam.BumpOffset.X *= decay
		cam.BumpOffset.Y *= decay
		if cam.BumpOffset.LengthSquared() < 1e-6 {
			cam.BumpOffset = Vector{}
		}
	}

	if cam.ShakeRemaining > 0 {
		cam.ShakeRemaining -= dt
		if cam.ShakeRemaining <= 0 {
			cam.ShakeRemaining = 0
			cam.ShakePower = 0
			cam.ShakeRotation = 0
			cam.shakeOffset = Vector{}
			cam.shakeRot = 0
		} else {
			cam.shakeTime += dt
			ratio := 1.0
			if cam.ShakeDuration > 0 {
				ratio = Clamp(cam.ShakeRemaining/cam.ShakeDuration, 0.0, 1.0)
			}
			freqX := cam.ShakeFreqX
			if freqX == 0 {
				freqX = 28.0
			}
			freqY := cam.ShakeFreqY
			if freqY == 0 {
				freqY = 37.0
			}
			power := cam.ShakePower * ratio
			cam.shakeOffset = V(
				math.Cos(cam.shakeTime*freqX)*power,
				math.Sin(0.3+cam.shakeTime*freqY)*power,
			)
			if cam.ShakeRotation != 0 {
				cam.shakeRot = math.Sin(cam.shakeTime*(freqX*0.73)) * cam.ShakeRotation * ratio
			} else {
				cam.shakeRot = 0
			}
		}
	} else {
		cam.shakeOffset = Vector{}
		cam.shakeRot = 0
	}
}

func updateCameraFollow(cam *CameraComponent, target *TransformComponent, dt float64) {
	targetPos := V(
		target.Position.X+cam.TargetOffset.X,
		target.Position.Y+cam.TargetOffset.Y,
	)

	if cam.snapNextUpdate {
		cam.Position = targetPos
		cam.Velocity = Vector{}
		cam.snapNextUpdate = false
		return
	}

	switch cam.FollowMode {
	case CameraFollowInstant:
		cam.Position = targetPos
		cam.Velocity = Vector{}

	case CameraFollowSmooth:
		speed := cam.FollowSpeed
		if speed <= 0 {
			speed = 8.0
		}
		alpha := 1.0 - math.Exp(-speed*dt)
		cam.Position = cam.Position.Lerp(targetPos, alpha)

	case CameraFollowSpring:
		dx := targetPos.X - cam.Position.X
		dy := targetPos.Y - cam.Position.Y
		ax := cam.SpringStiffness*dx - cam.SpringDamping*cam.Velocity.X
		ay := cam.SpringStiffness*dy - cam.SpringDamping*cam.Velocity.Y
		cam.Velocity.X += ax * dt
		cam.Velocity.Y += ay * dt
		cam.Position.X += cam.Velocity.X * dt
		cam.Position.Y += cam.Velocity.Y * dt

		if math.Abs(targetPos.X-cam.Position.X) < 1e-4 && math.Abs(cam.Velocity.X) < 1e-4 {
			cam.Position.X = targetPos.X
			cam.Velocity.X = 0
		}
		if math.Abs(targetPos.Y-cam.Position.Y) < 1e-4 && math.Abs(cam.Velocity.Y) < 1e-4 {
			cam.Position.Y = targetPos.Y
			cam.Velocity.Y = 0
		}

	case CameraFollowDeadZone:
		effZoom := math.Max(0.0001, cam.Zoom+cam.BumpZoomValue)
		viewW := cam.ViewportWidth / effZoom
		viewH := cam.ViewportHeight / effZoom

		dzX := cam.DeadZone.X * viewW
		dzY := cam.DeadZone.Y * viewH

		diffX := targetPos.X - cam.Position.X
		diffY := targetPos.Y - cam.Position.Y
		distX := math.Abs(diffX)
		distY := math.Abs(diffY)

		spd := cam.FollowSpeed
		if spd <= 0 {
			spd = 8.0
		}

		if distX > dzX {
			excessX := distX - dzX
			cam.Velocity.X += Sign(diffX) * excessX * spd * spd * dt
		}
		if distY > dzY {
			excessY := distY - dzY
			cam.Velocity.Y += Sign(diffY) * excessY * spd * spd * dt
		}

		baseFriction := math.Pow(Clamp(cam.TrackingFriction, 0.01, 0.999), dt*60.0)
		frictionX := baseFriction
		frictionY := baseFriction

		if cam.ClampBounds && cam.EdgeBrakeDistance > 0 {
			brakeDistX := cam.EdgeBrakeDistance * viewW
			if brakeDistX > 0 && cam.Velocity.X != 0 && cam.Bounds.Width() > viewW {
				minX := cam.Bounds.Min.X + viewW*0.5
				maxX := cam.Bounds.Max.X - viewW*0.5
				var brakeRatio float64
				if cam.Velocity.X < 0 {
					brakeRatio = 1.0 - Clamp((cam.Position.X-minX)/brakeDistX, 0.0, 1.0)
				} else {
					brakeRatio = 1.0 - Clamp((maxX-cam.Position.X)/brakeDistX, 0.0, 1.0)
				}
				frictionX *= 1.0 - Clamp(0.9*brakeRatio*dt*60.0, 0.0, 0.95)
			}

			brakeDistY := cam.EdgeBrakeDistance * viewH
			if brakeDistY > 0 && cam.Velocity.Y != 0 && cam.Bounds.Height() > viewH {
				minY := cam.Bounds.Min.Y + viewH*0.5
				maxY := cam.Bounds.Max.Y - viewH*0.5
				var brakeRatio float64
				if cam.Velocity.Y < 0 {
					brakeRatio = 1.0 - Clamp((cam.Position.Y-minY)/brakeDistY, 0.0, 1.0)
				} else {
					brakeRatio = 1.0 - Clamp((maxY-cam.Position.Y)/brakeDistY, 0.0, 1.0)
				}
				frictionY *= 1.0 - Clamp(0.9*brakeRatio*dt*60.0, 0.0, 0.95)
			}
		}

		cam.Position.X += cam.Velocity.X * dt
		cam.Position.Y += cam.Velocity.Y * dt
		cam.Velocity.X *= frictionX
		cam.Velocity.Y *= frictionY

		if math.Abs(cam.Velocity.X) < 1e-3 {
			cam.Velocity.X = 0
		}
		if math.Abs(cam.Velocity.Y) < 1e-3 {
			cam.Velocity.Y = 0
		}
	}
}

func clampCameraToBounds(cam *CameraComponent) {
	if !cam.ClampBounds {
		return
	}
	effZoom := cam.Zoom + cam.BumpZoomValue
	if effZoom < 0.0001 {
		effZoom = 0.0001
	}
	viewW := cam.ViewportWidth / effZoom
	viewH := cam.ViewportHeight / effZoom
	halfW := viewW * 0.5
	halfH := viewH * 0.5

	if cam.Bounds.Width() <= viewW {
		cam.Position.X = (cam.Bounds.Min.X + cam.Bounds.Max.X) * 0.5
		cam.Velocity.X = 0
	} else {
		minX := cam.Bounds.Min.X + halfW
		maxX := cam.Bounds.Max.X - halfW
		if cam.Position.X < minX {
			cam.Position.X = minX
			if cam.Velocity.X < 0 {
				cam.Velocity.X = 0
			}
		} else if cam.Position.X > maxX {
			cam.Position.X = maxX
			if cam.Velocity.X > 0 {
				cam.Velocity.X = 0
			}
		}
	}

	if cam.Bounds.Height() <= viewH {
		cam.Position.Y = (cam.Bounds.Min.Y + cam.Bounds.Max.Y) * 0.5
		cam.Velocity.Y = 0
	} else {
		minY := cam.Bounds.Min.Y + halfH
		maxY := cam.Bounds.Max.Y - halfH
		if cam.Position.Y < minY {
			cam.Position.Y = minY
			if cam.Velocity.Y < 0 {
				cam.Velocity.Y = 0
			}
		} else if cam.Position.Y > maxY {
			cam.Position.Y = maxY
			if cam.Velocity.Y > 0 {
				cam.Velocity.Y = 0
			}
		}
	}
}
