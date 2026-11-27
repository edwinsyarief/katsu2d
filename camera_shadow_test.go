package katsu2d

import (
	"image/color"
	"math"
	"testing"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func approxEqual32(a, b, tol float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

func TestCameraFollowModes(t *testing.T) {
	t.Run("CameraFollowInstant", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)
		sys.Initialize(w) // idempotent guard

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 300, Y: 200},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(800, 600, WithCameraFollowMode(CameraFollowInstant))
		camInit.FollowEntity(target, V(15, -25))
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		sys.Update(w, 1.0/60.0)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 315, 1e-9) || !approxEqual(cam.Position.Y, 175, 1e-9) {
			t.Fatalf("expected instant follow to snap to (315, 175), got %+v", cam.Position)
		}
		if cam.Velocity != (Vector{}) {
			t.Fatalf("expected zero velocity in instant mode, got %+v", cam.Velocity)
		}

		// Move target and verify instant re-snap.
		tr := teishoku.GetComponent[TransformComponent](w, target)
		tr.Position = Point{X: -120, Y: 440}
		sys.Update(w, 1.0/60.0)

		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, -105, 1e-9) || !approxEqual(cam.Position.Y, 415, 1e-9) {
			t.Fatalf("expected instant follow to snap to (-105, 415), got %+v", cam.Position)
		}
	})

	t.Run("CameraFollowSmooth", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 400, Y: 300},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(
			800, 600,
			WithCameraFollowMode(CameraFollowSmooth),
			WithCameraFollowSpeed(8.0),
		)
		camInit.FollowEntity(target, Vector{})
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		const dt = 1.0 / 60.0
		sys.Update(w, dt)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		alpha := 1.0 - math.Exp(-8.0*dt)
		wantX := 400.0 * alpha
		wantY := 300.0 * alpha
		if !approxEqual(cam.Position.X, wantX, 1e-6) || !approxEqual(cam.Position.Y, wantY, 1e-6) {
			t.Fatalf("expected smooth step to (%.6f, %.6f), got %+v", wantX, wantY, cam.Position)
		}
		if cam.Position.X >= 400 || cam.Position.Y >= 300 {
			t.Fatalf("expected smooth follow not to overshoot on first step, got %+v", cam.Position)
		}

		for range 180 {
			sys.Update(w, dt)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 400, 1e-3) || !approxEqual(cam.Position.Y, 300, 1e-3) {
			t.Fatalf("expected smooth follow to converge to (400, 300), got %+v", cam.Position)
		}
	})

	t.Run("CameraFollowSpring", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 500, Y: -200},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(
			800, 600,
			WithCameraFollowMode(CameraFollowSpring),
			WithCameraSpring(120.0, 16.0),
		)
		camInit.FollowEntity(target, Vector{})
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		const dt = 1.0 / 60.0
		sys.Update(w, dt)
		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.Velocity.X <= 0 || cam.Velocity.Y >= 0 {
			t.Fatalf("expected spring acceleration toward target, got velocity %+v", cam.Velocity)
		}
		if cam.Position.X <= 0 || cam.Position.Y >= 0 {
			t.Fatalf("expected spring movement toward target, got position %+v", cam.Position)
		}

		for range 240 {
			sys.Update(w, dt)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 500, 1e-3) || !approxEqual(cam.Position.Y, -200, 1e-3) {
			t.Fatalf("expected spring follow to settle at (500, -200), got %+v", cam.Position)
		}
		if !approxEqual(cam.Velocity.X, 0, 1e-3) || !approxEqual(cam.Velocity.Y, 0, 1e-3) {
			t.Fatalf("expected spring velocity to damp to zero, got %+v", cam.Velocity)
		}
	})

	t.Run("CameraFollowDeadZone", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		// Deadzone: 0.1 * 800 = 80px in X, 0.1 * 600 = 60px in Y.
		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 450, Y: 330},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(400, 300)),
			WithCameraFollowMode(CameraFollowDeadZone),
			WithCameraDeadZone(V(0.10, 0.10)),
			WithCameraFollowSpeed(8.0),
		)
		camInit.FollowEntity(target, Vector{})
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		const dt = 1.0 / 60.0
		// Target is at (+50, +30) from camera, well inside (+-80, +-60) deadzone.
		for range 10 {
			sys.Update(w, dt)
		}
		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.Position != V(400, 300) || cam.Velocity != (Vector{}) {
			t.Fatalf("expected target inside deadzone not to move camera, got pos=%+v vel=%+v", cam.Position, cam.Velocity)
		}

		// Move target outside deadzone; camera must accelerate smoothly without snapping.
		tr := teishoku.GetComponent[TransformComponent](w, target)
		tr.Position = Point{X: 600, Y: 450}
		sys.Update(w, dt)

		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.Velocity.X <= 0 || cam.Velocity.Y <= 0 {
			t.Fatalf("expected positive acceleration when target leaves deadzone, got vel=%+v", cam.Velocity)
		}
		if cam.Position.X <= 400 || cam.Position.X >= 600 {
			t.Fatalf("expected smooth non-snapping advance in X, got %f", cam.Position.X)
		}

		for range 120 {
			sys.Update(w, dt)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if math.Abs(600-cam.Position.X) > 80.5 || math.Abs(450-cam.Position.Y) > 60.5 {
			t.Fatalf("expected camera to bring target back into/near deadzone, got pos=%+v", cam.Position)
		}
	})

	t.Run("CameraFollowDeadZone_EdgeBraking", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		// Target placed inside deadzone so only friction + edge braking acts on initial velocity.
		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 1170, Y: 600},
			Scale:    Point{X: 1, Y: 1},
		})

		// World [0..1600] with viewport 800 -> maxX = 1200, brakeDistX = 0.25 * 800 = 200.
		// Position 1170 is 30px from maxX, well inside edge brake zone.
		camBraked := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(1170, 600)),
			WithCameraBounds(NewRectangle(0, 0, 1600, 1200)),
			WithCameraFollowMode(CameraFollowDeadZone),
			WithCameraDeadZone(V(0.10, 0.10)),
		)
		camBraked.EdgeBrakeDistance = 0.25
		camBraked.Velocity = V(300, 0)
		camBraked.FollowEntity(target, Vector{})

		camFree := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(1170, 600)),
			WithCameraFollowMode(CameraFollowDeadZone),
			WithCameraDeadZone(V(0.10, 0.10)),
		)
		camFree.EdgeBrakeDistance = 0
		camFree.Velocity = V(300, 0)
		camFree.FollowEntity(target, Vector{})

		cb := teishoku.NewBuilder[CameraComponent](w)
		entBraked := cb.NewEntityWith(*camBraked)
		entFree := cb.NewEntityWith(*camFree)

		sys.Update(w, 1.0/60.0)

		braked := teishoku.GetComponent[CameraComponent](w, entBraked)
		free := teishoku.GetComponent[CameraComponent](w, entFree)
		if braked.Velocity.X >= free.Velocity.X {
			t.Fatalf("expected edge braking to slow velocity more than unbraked: braked=%f free=%f", braked.Velocity.X, free.Velocity.X)
		}
		if braked.Position.X > 1200 {
			t.Fatalf("expected bounded camera X <= 1200, got %f", braked.Position.X)
		}
	})

	t.Run("ImmediateSnapAndRemovedTarget", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 250, Y: -100},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(
			800, 600,
			WithCameraFollowMode(CameraFollowSmooth),
			WithCameraFollowSpeed(4.0),
		)
		camInit.FollowEntityImmediate(target, V(10, 20))
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		// FollowEntityImmediate snaps on the very next update even in smooth mode.
		sys.Update(w, 1.0/60.0)
		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 260, 1e-9) || !approxEqual(cam.Position.Y, -80, 1e-9) {
			t.Fatalf("expected FollowEntityImmediate to snap to (260, -80), got %+v", cam.Position)
		}

		// Move target far away and call SnapToTarget.
		tr := teishoku.GetComponent[TransformComponent](w, target)
		tr.Position = Point{X: 2000, Y: 1500}
		cam.Velocity = V(50, 50)
		cam.SnapToTarget()

		sys.Update(w, 1.0/60.0)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 2010, 1e-9) || !approxEqual(cam.Position.Y, 1520, 1e-9) {
			t.Fatalf("expected SnapToTarget to snap to (2010, 1520), got %+v", cam.Position)
		}
		if cam.Velocity != (Vector{}) {
			t.Fatalf("expected SnapToTarget to clear velocity, got %+v", cam.Velocity)
		}

		// Remove target entity while camera still has HasTarget=true -> must not panic or produce NaN.
		w.RemoveEntity(target)
		for range 5 {
			sys.Update(w, 1.0/60.0)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 2010, 1e-9) || !approxEqual(cam.Position.Y, 1520, 1e-9) {
			t.Fatalf("expected camera to remain stable after target removal, got %+v", cam.Position)
		}
	})
}

func TestCameraEffectsAndTweens(t *testing.T) {
	t.Run("MoveTo_ZoomTo_RotateTo", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(NewTransformComponent())
		camInit := NewCameraComponent(
			800, 600,
			WithCameraZoom(1.0),
			WithCameraZoomLimits(0.5, 3.0),
		)
		camInit.FollowEntity(target, Vector{})
		camInit.Velocity = V(25, -25)
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		cam.MoveTo(V(400, -200), 0.5, QuadOut)
		if cam.HasTarget {
			t.Fatal("expected MoveTo to clear HasTarget")
		}
		if cam.Velocity != (Vector{}) {
			t.Fatal("expected MoveTo to reset velocity")
		}

		cam.ZoomTo(5.0, 0.4, Linear) // clamped to MaxZoom=3.0
		cam.RotateTo(1.2, 0.4, CubicInOut)

		sys.Update(w, 0.2)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.Position.X <= 0 || cam.Position.X >= 400 {
			t.Fatalf("expected intermediate MoveTo X in (0, 400), got %f", cam.Position.X)
		}
		if !approxEqual(cam.Zoom, 2.0, 1e-6) {
			t.Fatalf("expected linear clamped ZoomTo midpoint=2.0, got %f", cam.Zoom)
		}
		if !approxEqual(cam.Rotation, 0.6, 1e-6) {
			t.Fatalf("expected symmetric CubicInOut RotateTo midpoint=0.6, got %f", cam.Rotation)
		}

		sys.Update(w, 0.2)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Zoom, 3.0, 1e-9) {
			t.Fatalf("expected ZoomTo clamped to MaxZoom=3.0, got %f", cam.Zoom)
		}
		if !approxEqual(cam.Rotation, 1.2, 1e-9) {
			t.Fatalf("expected RotateTo completion=1.2, got %f", cam.Rotation)
		}

		sys.Update(w, 0.1)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 400, 1e-9) || !approxEqual(cam.Position.Y, -200, 1e-9) {
			t.Fatalf("expected MoveTo completion at (400, -200), got %+v", cam.Position)
		}

		// Zero-duration tween snaps immediately and clamps MinZoom.
		cam.ZoomTo(0.1, 0, Linear)
		cam.MoveTo(V(10, 20), 0, Linear)
		cam.RotateTo(-0.5, 0, Linear)
		if !approxEqual(cam.Zoom, 0.5, 1e-9) {
			t.Fatalf("expected zero-duration ZoomTo to clamp to MinZoom=0.5, got %f", cam.Zoom)
		}
		if cam.Position != V(10, 20) || cam.Rotation != -0.5 {
			t.Fatalf("expected zero-duration MoveTo/RotateTo immediate snap, got pos=%+v rot=%f", cam.Position, cam.Rotation)
		}
	})

	t.Run("BumpZoom_Bump_BumpAngular", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		camInit := NewCameraComponent(800, 600, WithCameraZoom(1.0))
		camInit.BumpZoom(0.5)
		camInit.Bump(V(30, -40))
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.BumpZoomValue, 0.5, 1e-9) {
			t.Fatalf("expected initial BumpZoomValue=0.5, got %f", cam.BumpZoomValue)
		}
		if cam.BumpOffset != V(30, -40) {
			t.Fatalf("expected initial BumpOffset=(30, -40), got %+v", cam.BumpOffset)
		}

		// Advance a few frames: values should decay smoothly.
		sys.Update(w, 0.05)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.BumpZoomValue <= 0 || cam.BumpZoomValue >= 0.5 {
			t.Fatalf("expected decaying BumpZoomValue in (0, 0.5), got %f", cam.BumpZoomValue)
		}
		if cam.BumpOffset.Length() <= 0 || cam.BumpOffset.Length() >= 50 {
			t.Fatalf("expected decaying BumpOffset length in (0, 50), got %f", cam.BumpOffset.Length())
		}

		// Advance until threshold snaps both to exact zero.
		for range 120 {
			sys.Update(w, 0.05)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.BumpZoomValue != 0 {
			t.Fatalf("expected BumpZoomValue to decay to 0, got %f", cam.BumpZoomValue)
		}
		if cam.BumpOffset != (Vector{}) {
			t.Fatalf("expected BumpOffset to decay to zero, got %+v", cam.BumpOffset)
		}

		// Test BumpAngular direction and decay.
		angle := math.Pi / 3.0
		dist := 20.0
		cam.BumpAngular(angle, dist)
		wantX := math.Cos(angle) * dist
		wantY := math.Sin(angle) * dist
		if !approxEqual(cam.BumpOffset.X, wantX, 1e-9) || !approxEqual(cam.BumpOffset.Y, wantY, 1e-9) {
			t.Fatalf("expected BumpAngular offset (%.6f, %.6f), got %+v", wantX, wantY, cam.BumpOffset)
		}
		for range 120 {
			sys.Update(w, 0.05)
		}
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.BumpOffset != (Vector{}) {
			t.Fatalf("expected BumpAngular offset to decay to zero, got %+v", cam.BumpOffset)
		}
	})

	t.Run("Shake_And_ShakeWithRotation", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		camInit := NewCameraComponent(800, 600, WithCameraPosition(V(100, 200)), WithCameraRotation(0.25))
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		cam.Shake(0.20, 12.0)
		if !cam.IsShaking() {
			t.Fatal("expected IsShaking() == true immediately after Shake")
		}

		sys.Update(w, 0.05)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !cam.IsShaking() {
			t.Fatal("expected IsShaking() == true mid-shake")
		}
		if cam.shakeOffset.IsZero() {
			t.Fatal("expected non-zero shakeOffset mid-shake")
		}
		if cam.shakeRot != 0 {
			t.Fatalf("expected shakeRot=0 for positional Shake, got %f", cam.shakeRot)
		}

		// Advance past remaining 0.15s.
		sys.Update(w, 0.16)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.IsShaking() {
			t.Fatal("expected IsShaking() == false after duration elapsed")
		}
		if cam.shakeOffset != (Vector{}) || cam.shakeRot != 0 {
			t.Fatalf("expected shake offsets reset to zero, got offset=%+v rot=%f", cam.shakeOffset, cam.shakeRot)
		}
		ac := GetActiveCamera(w)
		if ac == nil || ac.Position != V(100, 200) || ac.Rotation != 0.25 {
			t.Fatalf("expected ActiveCamera restored to base pose, got %+v", ac)
		}

		// ShakeWithRotation.
		cam.ShakeWithRotation(0.20, 10.0, 0.3)
		if !cam.IsShaking() {
			t.Fatal("expected IsShaking() == true for ShakeWithRotation")
		}

		sys.Update(w, 0.03)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !cam.IsShaking() {
			t.Fatal("expected IsShaking() == true mid-ShakeWithRotation")
		}
		if cam.shakeOffset.IsZero() || cam.shakeRot == 0 {
			t.Fatalf("expected both positional and rotational shake offsets mid-shake, got offset=%+v rot=%f", cam.shakeOffset, cam.shakeRot)
		}

		sys.Update(w, 0.20)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if cam.IsShaking() {
			t.Fatal("expected ShakeWithRotation to finish")
		}
		if cam.shakeOffset != (Vector{}) || cam.shakeRot != 0 {
			t.Fatalf("expected ShakeWithRotation offsets reset to zero, got offset=%+v rot=%f", cam.shakeOffset, cam.shakeRot)
		}

		// Zero power/duration clears shake immediately.
		cam.ShakeWithRotation(0.5, 10, 0.2)
		cam.ShakeWithRotation(0, 0, 0)
		if cam.IsShaking() {
			t.Fatal("expected zero ShakeWithRotation to clear active shake")
		}
	})
}

func TestCameraBoundsAndCoordinates(t *testing.T) {
	t.Run("BoundsClampingAndSmallWorldCentering", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		camInit := NewCameraComponent(800, 600)
		camInit.SetBounds(NewRectangle(0, 0, 2000, 1500))
		camInit.SetPosition(V(-500, -500))
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		sys.Update(w, 1.0/60.0)
		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 400, 1e-9) || !approxEqual(cam.Position.Y, 300, 1e-9) {
			t.Fatalf("expected min-bound clamp to (400, 300), got %+v", cam.Position)
		}

		cam.SetPosition(V(5000, 5000))
		sys.Update(w, 1.0/60.0)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 1600, 1e-9) || !approxEqual(cam.Position.Y, 1200, 1e-9) {
			t.Fatalf("expected max-bound clamp to (1600, 1200), got %+v", cam.Position)
		}

		// Zoom=2 halves visible world area (400x300 -> half 200x150).
		cam.SetZoom(2.0)
		cam.SetPosition(V(-500, -500))
		sys.Update(w, 1.0/60.0)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 200, 1e-9) || !approxEqual(cam.Position.Y, 150, 1e-9) {
			t.Fatalf("expected zoomed min-bound clamp to (200, 150), got %+v", cam.Position)
		}

		// World smaller than viewport centers camera along both axes.
		cam.SetZoom(1.0)
		cam.SetBounds(NewRectangle(100, 200, 500, 500)) // 400x300 < 800x600
		cam.Velocity = V(99, -88)
		cam.SetPosition(V(0, 0))
		sys.Update(w, 1.0/60.0)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		if !approxEqual(cam.Position.X, 300, 1e-9) || !approxEqual(cam.Position.Y, 350, 1e-9) {
			t.Fatalf("expected small world centering at (300, 350), got %+v", cam.Position)
		}
	})

	t.Run("WorldScreenRoundTrip_PanZoomRotate", func(t *testing.T) {
		cam := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(123.5, -456.25)),
			WithCameraZoom(2.0),
			WithCameraRotation(0.4),
			WithCameraPixelSnap(false),
		)

		// Camera position must map to exact viewport center.
		centerScreen := cam.WorldToScreen(cam.Position)
		if !approxEqual(centerScreen.X, 400, 1e-9) || !approxEqual(centerScreen.Y, 300, 1e-9) {
			t.Fatalf("expected camera Position to map to screen center (400, 300), got %+v", centerScreen)
		}

		worldSamples := []Vector{
			V(0, 0),
			V(123.5, -456.25),
			V(-312.4, 88.9),
			V(999.1, 42.3),
		}
		for _, wp := range worldSamples {
			sp := cam.WorldToScreen(wp)
			roundTrip := cam.ScreenToWorld(sp)
			if !approxEqual(roundTrip.X, wp.X, 1e-9) || !approxEqual(roundTrip.Y, wp.Y, 1e-9) {
				t.Fatalf("WorldToScreen->ScreenToWorld mismatch for %+v: got %+v", wp, roundTrip)
			}
		}

		screenSamples := []Vector{
			V(0, 0),
			V(800, 0),
			V(800, 600),
			V(0, 600),
			V(250.75, 410.25),
		}
		for _, sp := range screenSamples {
			wp := cam.ScreenToWorld(sp)
			roundTrip := cam.WorldToScreen(wp)
			if !approxEqual(roundTrip.X, sp.X, 1e-9) || !approxEqual(roundTrip.Y, sp.Y, 1e-9) {
				t.Fatalf("ScreenToWorld->WorldToScreen mismatch for %+v: got %+v", sp, roundTrip)
			}
		}
	})

	t.Run("ViewBounds_Visibility_PixelSnap_ActiveCameraSync", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		if got := GetActiveCamera(w); got != nil {
			t.Fatalf("expected GetActiveCamera == nil before update, got %+v", got)
		}
		if got := GetActiveCamera(nil); got != nil {
			t.Fatalf("expected GetActiveCamera(nil) == nil, got %+v", got)
		}

		sys := NewCameraSystem()
		sys.Initialize(w)

		camInit := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(100, 50)),
			WithCameraZoom(2.0),
		)
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)
		sys.Update(w, 1.0/60.0)

		cam := teishoku.GetComponent[CameraComponent](w, camEnt)
		vb := cam.ViewBounds()
		// Viewport 800x600 at zoom 2 -> 400x300 around (100, 50): [-100, -100] .. [300, 200]
		if !approxEqual(vb.Min.X, -100, 1e-9) || !approxEqual(vb.Min.Y, -100, 1e-9) ||
			!approxEqual(vb.Max.X, 300, 1e-9) || !approxEqual(vb.Max.Y, 200, 1e-9) {
			t.Fatalf("unexpected unrotated ViewBounds: %+v", vb)
		}

		if !cam.IsVisible(NewRectangle(50, 0, 150, 100)) {
			t.Fatal("expected inside rect to be visible")
		}
		if cam.IsVisible(NewRectangle(400, 400, 500, 500)) {
			t.Fatal("expected far outside rect not to be visible")
		}
		if !cam.IsPointVisible(V(100, 50), 0) {
			t.Fatal("expected center point to be visible")
		}
		if cam.IsPointVisible(V(310, 50), 0) {
			t.Fatal("expected point outside ViewBounds with margin=0 not to be visible")
		}
		if !cam.IsPointVisible(V(310, 50), 15) {
			t.Fatal("expected point within margin=15 of ViewBounds to be visible")
		}

		// Rotated ViewBounds encloses all 4 screen corners mapped to world space.
		cam.SetRotation(0.4)
		rotVB := cam.ViewBounds()
		corners := []Vector{V(0, 0), V(800, 0), V(800, 600), V(0, 600)}
		for _, c := range corners {
			wc := cam.ScreenToWorld(c)
			if wc.X < rotVB.Min.X-1e-9 || wc.X > rotVB.Max.X+1e-9 ||
				wc.Y < rotVB.Min.Y-1e-9 || wc.Y > rotVB.Max.Y+1e-9 {
				t.Fatalf("rotated corner %+v outside ViewBounds %+v", wc, rotVB)
			}
		}

		// PixelSnap is enabled by default and snaps final view matrix translation components
		// to integer pixels when unrotated.
		snapCam := NewCameraComponent(
			801, 601,
			WithCameraPosition(V(100.37, 200.68)),
		)
		if !snapCam.PixelSnap {
			t.Fatal("expected PixelSnap to default to true")
		}
		vm := snapCam.ViewMatrix()
		tx := vm.Element(0, 2)
		ty := vm.Element(1, 2)
		if tx != math.Round(tx) || ty != math.Round(ty) {
			t.Fatalf("expected PixelSnap integer translation, got tx=%f ty=%f", tx, ty)
		}

		// When camera rotation is non-zero, PixelSnap leaves translation unrounded.
		snapCam.SetRotation(0.25)
		rotVM := snapCam.ViewMatrix()
		rotCamFree := NewCameraComponent(
			801, 601,
			WithCameraPosition(V(100.37, 200.68)),
			WithCameraRotation(0.25),
			WithCameraPixelSnap(false),
		)
		if rotVM != rotCamFree.ViewMatrix() {
			t.Fatalf("expected PixelSnap to skip rounding when Rotation != 0")
		}

		// ActiveCamera resource syncs and delegates WorldToScreen/ScreenToWorld/IsVisible/IsPointVisible.
		sys.Update(w, 1.0/60.0)
		cam = teishoku.GetComponent[CameraComponent](w, camEnt)
		ac := GetActiveCamera(w)
		if ac == nil || !ac.HasCamera {
			t.Fatalf("expected valid GetActiveCamera after Update, got %+v", ac)
		}
		if ac.Position != cam.Position || ac.Zoom != cam.Zoom || ac.Rotation != cam.Rotation {
			t.Fatalf("ActiveCamera state mismatch: got %+v want pos=%+v zoom=%f rot=%f", ac, cam.Position, cam.Zoom, cam.Rotation)
		}
		pt := V(140, 80)
		if !ac.WorldToScreen(pt).Equals(cam.WorldToScreen(pt)) {
			t.Fatalf("ActiveCamera.WorldToScreen mismatch")
		}
		if !ac.ScreenToWorld(V(200, 150)).Equals(cam.ScreenToWorld(V(200, 150))) {
			t.Fatalf("ActiveCamera.ScreenToWorld mismatch")
		}
		if !ac.IsVisible(NewRectangle(90, 40, 110, 60)) || ac.IsVisible(NewRectangle(900, 900, 950, 950)) {
			t.Fatal("ActiveCamera.IsVisible mismatch")
		}
		if !ac.IsPointVisible(V(100, 50), 0) {
			t.Fatal("ActiveCamera.IsPointVisible mismatch")
		}

		// Deactivating the only camera makes GetActiveCamera(w) return nil.
		cam.Active = false
		sys.Update(w, 1.0/60.0)
		if got := GetActiveCamera(w); got != nil {
			t.Fatalf("expected GetActiveCamera(w) == nil after deactivation, got %+v", got)
		}
	})
}

func TestCameraPixelSnapZeroRelativeJitter(t *testing.T) {
	assertZeroRelativeJitter := func(t *testing.T, frame int, cam *CameraComponent) {
		t.Helper()
		vm := cam.ViewMatrix()
		s1x, s1y := vm.Apply(320, 240)
		s2x, s2y := vm.Apply(352, 240)

		if s1x != math.Trunc(s1x) || s1y != math.Trunc(s1y) ||
			s2x != math.Trunc(s2x) || s2y != math.Trunc(s2y) {
			t.Fatalf("frame %d (pos=%+v): expected exact integer screen coords, got p1=(%.17g, %.17g) p2=(%.17g, %.17g)",
				frame, cam.Position, s1x, s1y, s2x, s2y)
		}
		if sepX, sepY := s2x-s1x, s2y-s1y; sepX != 32.0 || sepY != 0.0 {
			t.Fatalf("frame %d (pos=%+v): expected exact 32.0 separation, got (%.17g, %.17g)",
				frame, cam.Position, sepX, sepY)
		}

		// Verify InvViewMatrix stays consistent with the snapped ViewMatrix.
		w1 := cam.ScreenToWorld(V(s1x, s1y))
		w2 := cam.ScreenToWorld(V(s2x, s2y))
		if !approxEqual(w1.X, 320, 1e-9) || !approxEqual(w1.Y, 240, 1e-9) ||
			!approxEqual(w2.X, 352, 1e-9) || !approxEqual(w2.Y, 240, 1e-9) {
			t.Fatalf("frame %d: InvViewMatrix round-trip mismatch: w1=%+v w2=%+v", frame, w1, w2)
		}
	}

	t.Run("SmoothFollow60FramesOddViewport", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewCameraSystem()
		sys.Initialize(w)

		target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
			Position: Point{X: 250.73, Y: 210.61},
			Scale:    Point{X: 1, Y: 1},
		})

		camInit := NewCameraComponent(
			801, 601,
			WithCameraPosition(V(100.13, 150.29)),
			WithCameraFollowMode(CameraFollowSmooth),
			WithCameraFollowSpeed(5.0),
		)
		camInit.FollowEntity(target, Vector{})
		camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(*camInit)

		for frame := range 60 {
			sys.Update(w, 1.0/60.0)
			cam := teishoku.GetComponent[CameraComponent](w, camEnt)
			assertZeroRelativeJitter(t, frame, cam)
		}
	})

	t.Run("ExplicitFractionalPositions60Frames", func(t *testing.T) {
		cam := NewCameraComponent(800, 600, WithCameraPosition(V(100.13, 200.07)))
		fracSteps := []float64{0.34, 0.42, 0.27, 0.19, 0.31}
		pos := cam.Position
		for frame := range 60 {
			cam.SetPosition(pos)
			assertZeroRelativeJitter(t, frame, cam)
			pos.X += fracSteps[frame%len(fracSteps)]
			pos.Y += fracSteps[(frame+2)%len(fracSteps)]
		}
	})
}

func TestShadowVertexBuildersAndLight(t *testing.T) {
	t.Run("ProjectedShadowVertices", func(t *testing.T) {
		var verts [4]ebiten.Vertex
		tr := TransformComponent{
			Position: Point{X: 100, Y: 200},
			Scale:    Point{X: 1, Y: 1},
			Offset:   Point{X: 16, Y: 32},
		}
		sp := NewSpriteComponent(0, 32, 32)
		sh := NewProjectedShadow(0.5, 0.75)
		sh.FadeOut = true
		sh.FadeAlpha = 0.25
		sh.Opacity = 0.8

		buildProjectedShadowVertices(
			&verts, &tr, &sp, &sh,
			sh.Angle, sh.Length,
			0, 0, 32, 32,
			color.RGBA{R: 10, G: 20, B: 40, A: 255},
			sh.Opacity, false,
		)

		// Base anchored at bottom (y=200), x in [84, 116].
		if !approxEqual32(verts[0].DstX, 84, 1e-5) || !approxEqual32(verts[0].DstY, 200, 1e-5) ||
			!approxEqual32(verts[1].DstX, 116, 1e-5) || !approxEqual32(verts[1].DstY, 200, 1e-5) {
			t.Fatalf("unexpected projected shadow base vertices: v0=%+v v1=%+v", verts[0], verts[1])
		}

		// Top skewed by Angle (32 * 0.5 = 16) and Length (32 * 0.75 = 24).
		if !approxEqual32(verts[3].DstX, 100, 1e-5) || !approxEqual32(verts[3].DstY, 224, 1e-5) ||
			!approxEqual32(verts[2].DstX, 132, 1e-5) || !approxEqual32(verts[2].DstY, 224, 1e-5) {
			t.Fatalf("unexpected projected shadow tip vertices: v2=%+v v3=%+v", verts[2], verts[3])
		}

		// FadeOut sets tip alpha to baseAlpha * FadeAlpha (0.8 * 0.25 = 0.2).
		if !approxEqual32(verts[0].ColorA, 0.8, 1e-5) || !approxEqual32(verts[2].ColorA, 0.2, 1e-5) {
			t.Fatalf("expected base alpha 0.8 and tip alpha 0.2, got base=%f tip=%f", verts[0].ColorA, verts[2].ColorA)
		}

		// PreserveColor preserves sprite RGB by using premultiplied white (a, a, a, a).
		refl := NewReflectionShadow(1.0)
		refl.FadeOut = false
		buildProjectedShadowVertices(
			&verts, &tr, &sp, &refl,
			refl.Angle, refl.Length,
			0, 0, 32, 32,
			color.RGBA{R: 0, G: 0, B: 0, A: 255},
			0.5, false,
		)
		if !approxEqual32(verts[0].ColorR, 0.5, 1e-5) || !approxEqual32(verts[0].ColorG, 0.5, 1e-5) ||
			!approxEqual32(verts[0].ColorB, 0.5, 1e-5) || !approxEqual32(verts[0].ColorA, 0.5, 1e-5) {
			t.Fatalf("expected PreserveColor premultiplied (0.5,0.5,0.5,0.5), got %+v", verts[0])
		}

		// Negative Scale.X flips UV horizontally without inverting vertex DstX order.
		trFlip := tr
		trFlip.Scale.X = -1
		buildProjectedShadowVertices(
			&verts, &trFlip, &sp, &sh,
			sh.Angle, sh.Length,
			0, 0, 32, 32,
			sh.Color, sh.Opacity, false,
		)
		if verts[0].DstX >= verts[1].DstX {
			t.Fatalf("expected non-inverted DstX order under negative Scale.X, got v0=%f v1=%f", verts[0].DstX, verts[1].DstX)
		}
		if verts[0].SrcX != 32 || verts[1].SrcX != 0 {
			t.Fatalf("expected flipped SrcX (32 -> 0), got v0=%f v1=%f", verts[0].SrcX, verts[1].SrcX)
		}

		// Elevation shifts base by (Elevation*angle, Elevation*0.5).
		shElev := sh
		shElev.Elevation = 20
		buildProjectedShadowVertices(
			&verts, &tr, &sp, &shElev,
			0.5, shElev.Length,
			0, 0, 32, 32,
			shElev.Color, shElev.Opacity, false,
		)
		if !approxEqual32(verts[0].DstX, 94, 1e-5) || !approxEqual32(verts[0].DstY, 210, 1e-5) {
			t.Fatalf("expected elevation base shift (+10, +10) -> (94, 210), got (%f, %f)", verts[0].DstX, verts[0].DstY)
		}
	})

	t.Run("DropShadowVertices", func(t *testing.T) {
		var verts [4]ebiten.Vertex
		tr := TransformComponent{
			Position: Point{X: 50, Y: 60},
			Scale:    Point{X: 1, Y: 1},
			Rotation: math.Pi / 2.0,
		}
		sp := NewSpriteComponent(0, 20, 10)
		sh := NewDropShadow(V(6, 8))
		sh.Elevation = 10

		buildDropShadowVertices(
			&verts, &tr, &sp, &sh,
			0.4,
			0, 0, 20, 10,
			sh.Color, 0.5, false,
		)

		// tx = 50 + 6 + 10*0.4 = 60, ty = 60 + 8 + 10*0.5 = 73.
		// Rotating (0..20, 0..10) by +pi/2 maps (x, y) -> (-y, x).
		if !approxEqual32(verts[0].DstX, 60, 1e-4) || !approxEqual32(verts[0].DstY, 73, 1e-4) ||
			!approxEqual32(verts[1].DstX, 60, 1e-4) || !approxEqual32(verts[1].DstY, 93, 1e-4) ||
			!approxEqual32(verts[2].DstX, 50, 1e-4) || !approxEqual32(verts[2].DstY, 93, 1e-4) ||
			!approxEqual32(verts[3].DstX, 50, 1e-4) || !approxEqual32(verts[3].DstY, 73, 1e-4) {
			t.Fatalf("unexpected rotated+elevated drop shadow vertices: %+v", verts)
		}
		if !approxEqual32(verts[0].ColorA, 0.5, 1e-5) {
			t.Fatalf("expected drop shadow ColorA=0.5, got %f", verts[0].ColorA)
		}
	})

	t.Run("ContactShadowVertices", func(t *testing.T) {
		var verts [4]ebiten.Vertex
		tr := TransformComponent{
			Position: Point{X: 100, Y: 100},
			Scale:    Point{X: 2, Y: 1},
			Offset:   Point{X: 20, Y: 20},
		}
		sp := NewSpriteComponent(0, 40, 20)
		sh := NewContactShadow(0, 0) // auto radius from sprite

		ok := buildContactShadowVertices(&verts, &tr, &sp, &sh, 0, 0, 64, 64, sh.Color, 0.6)
		if !ok {
			t.Fatal("expected auto-sized contact shadow to return true")
		}

		// rx = 40 * 0.45 * 2 = 36 -> width = 72; ry = 36 * 0.45 = 16.2 -> height = 32.4.
		width0 := verts[1].DstX - verts[0].DstX
		height0 := verts[2].DstY - verts[1].DstY
		alpha0 := verts[0].ColorA
		if !approxEqual32(width0, 72, 1e-4) || !approxEqual32(height0, 32.4, 1e-4) {
			t.Fatalf("expected auto contact dimensions (72, 32.4), got (%f, %f)", width0, height0)
		}

		// Elevation shrinks radius and softens alpha.
		shElev := sh
		shElev.Elevation = 50
		ok = buildContactShadowVertices(&verts, &tr, &sp, &shElev, 0, 0, 64, 64, shElev.Color, 0.6)
		if !ok {
			t.Fatal("expected elevated contact shadow to return true")
		}
		widthElev := verts[1].DstX - verts[0].DstX
		alphaElev := verts[0].ColorA
		if widthElev >= width0 {
			t.Fatalf("expected elevation to shrink radius: base=%f elev=%f", width0, widthElev)
		}
		if alphaElev >= alpha0 {
			t.Fatalf("expected elevation to soften alpha: base=%f elev=%f", alpha0, alphaElev)
		}
		if !approxEqual32(alphaElev, 0.6*0.5, 1e-5) { // 1/(1+50*0.02) = 0.5
			t.Fatalf("expected elevated alpha 0.3, got %f", alphaElev)
		}

		// Zero-area contact shadow returns false.
		zeroSh := NewContactShadow(0, 0)
		if buildContactShadowVertices(&verts, &tr, nil, &zeroSh, 0, 0, 64, 64, zeroSh.Color, 0.6) {
			t.Fatal("expected zero-area contact shadow with nil sprite to return false")
		}
		zeroSp := NewSpriteComponent(0, 0, 0)
		if buildContactShadowVertices(&verts, &tr, &zeroSp, &zeroSh, 0, 0, 64, 64, zeroSh.Color, 0.6) {
			t.Fatal("expected zero-area contact shadow with zero-size sprite to return false")
		}
	})

	t.Run("ShadowRenderSystemOptionsSettersDisposeAndLight", func(t *testing.T) {
		col := color.RGBA{R: 15, G: 25, B: 45, A: 255}
		sys := NewShadowRenderSystem(
			WithShadowComposited(false),
			WithShadowBlur(4.5), // positive blur implicitly enables composited
			WithShadowGlobalOpacity(0.65),
			WithShadowGlobalColor(col),
			WithShadowFrustumMargin(200),
		)

		if !sys.IsComposited() {
			t.Fatal("expected positive WithShadowBlur to enable composited mode")
		}
		if !approxEqual(sys.BlurRadius(), 4.5, 1e-9) {
			t.Fatalf("expected BlurRadius=4.5, got %f", sys.BlurRadius())
		}
		if !approxEqual(sys.globalOpacity, 0.65, 1e-9) {
			t.Fatalf("expected globalOpacity=0.65, got %f", sys.globalOpacity)
		}
		if !sys.hasGlobalColor || sys.globalColor != col {
			t.Fatalf("expected globalColor=%+v, got has=%v col=%+v", col, sys.hasGlobalColor, sys.globalColor)
		}
		if !approxEqual(sys.frustumMargin, 200, 1e-9) {
			t.Fatalf("expected frustumMargin=200, got %f", sys.frustumMargin)
		}

		// Negative clamping and runtime setters.
		sys.SetComposited(false)
		if sys.IsComposited() {
			t.Fatal("expected SetComposited(false) to disable composited mode")
		}
		sys.SetBlurRadius(2.5)
		if !sys.IsComposited() || !approxEqual(sys.BlurRadius(), 2.5, 1e-9) {
			t.Fatalf("expected SetBlurRadius(2.5) to enable composited mode and set radius=2.5")
		}
		sys.SetBlurRadius(-3.0)
		if sys.BlurRadius() != 0 {
			t.Fatalf("expected negative SetBlurRadius to clamp to 0, got %f", sys.BlurRadius())
		}
		sys.SetGlobalOpacity(1.5)
		if sys.globalOpacity != 1.0 {
			t.Fatalf("expected opacity clamp to 1.0, got %f", sys.globalOpacity)
		}
		sys.SetGlobalOpacity(-0.2)
		if sys.globalOpacity != 0.0 {
			t.Fatalf("expected opacity clamp to 0.0, got %f", sys.globalOpacity)
		}

		// Dispose idempotency (with and without allocated offscreen targets).
		sys.Dispose()
		sys.Dispose()
		sys.ensureTargets(64, 64, true)
		if sys.shadowTarget == nil || sys.blurTarget == nil {
			t.Fatal("expected ensureTargets to allocate shadowTarget and blurTarget")
		}
		sys.Dispose()
		sys.Dispose()
		if sys.shadowTarget != nil || sys.blurTarget != nil {
			t.Fatal("expected Dispose to clear shadowTarget and blurTarget")
		}

		// Global ShadowLightComponent constructor & world initialization guard.
		light := NewShadowLightComponent(0.35, 1.25, 0.55)
		if !light.Enabled || light.Angle != 0.35 || light.Length != 1.25 || light.Opacity != 0.55 {
			t.Fatalf("unexpected ShadowLightComponent: %+v", light)
		}
		w := teishoku.NewWorld(16)
		sys.Initialize(w)
		sys.Initialize(w)
	})

	t.Run("ShadowBlurShaderCompiles", func(t *testing.T) {
		shader := getShadowBlurShader()
		if shader == nil {
			t.Fatal("expected getShadowBlurShader() to return a non-nil compiled shader")
		}
	})
}

var (
	benchVertexSink ebiten.Vertex
	benchBoolSink   bool
)

func BenchmarkCameraSystemUpdate(b *testing.B) {
	w := teishoku.NewWorld(16)
	target := teishoku.NewBuilder[TransformComponent](w).NewEntityWith(TransformComponent{
		Position: Point{X: 600, Y: 450},
		Scale:    Point{X: 1, Y: 1},
	})

	camInit := NewCameraComponent(
		800, 600,
		WithCameraPosition(V(500, 400)),
		WithCameraBounds(NewRectangle(0, 0, 2000, 2000)),
		WithCameraFollowMode(CameraFollowSmooth),
	)
	camInit.FollowEntity(target, V(10, 15))
	camInit.ShakeWithRotation(1e9, 6.0, 0.05)
	camInit.Bump(V(4, -3))
	camInit.BumpZoom(0.05)

	camBuilder := teishoku.NewBuilder[CameraComponent](w)
	camEnt := camBuilder.NewEntityWith(*camInit)

	sys := NewCameraSystem()
	sys.Initialize(w)
	sys.Update(w, 1.0/60.0) // warm up

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if cam := camBuilder.Get(camEnt); cam != nil {
			cam.BumpOffset = V(4, -3)
			cam.BumpZoomValue = 0.05
		}
		sys.Update(w, 1.0/60.0)
	}
}

func BenchmarkBuildProjectedShadowVertices1K(b *testing.B) {
	const count = 1000
	transforms := make([]TransformComponent, count)
	sprites := make([]SpriteComponent, count)
	shadows := make([]ShadowComponent, count)
	for i := range count {
		transforms[i] = TransformComponent{
			Position: Point{X: float64(i * 10), Y: float64(i * 5)},
			Scale:    Point{X: 1, Y: 1},
			Offset:   Point{X: 16, Y: 32},
		}
		sprites[i] = NewSpriteComponent(0, 32, 32)
		shadows[i] = NewProjectedShadow(0.4, 0.8)
	}

	var dst [4]ebiten.Vertex
	col := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for i := range count {
			buildProjectedShadowVertices(
				&dst, &transforms[i], &sprites[i], &shadows[i],
				0.4, 0.8,
				0, 0, 32, 32,
				col, 0.4, false,
			)
		}
		benchVertexSink = dst[0]
	}
}

func BenchmarkBuildContactShadowVertices1K(b *testing.B) {
	const count = 1000
	transforms := make([]TransformComponent, count)
	sprites := make([]SpriteComponent, count)
	shadows := make([]ShadowComponent, count)
	for i := range count {
		transforms[i] = TransformComponent{
			Position: Point{X: float64(i * 10), Y: float64(i * 5)},
			Scale:    Point{X: 1, Y: 1},
			Offset:   Point{X: 16, Y: 32},
		}
		sprites[i] = NewSpriteComponent(0, 32, 32)
		shadows[i] = NewContactShadow(0, 0)
		shadows[i].Elevation = float64(i % 20)
	}

	var dst [4]ebiten.Vertex
	col := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for i := range count {
			benchBoolSink = buildContactShadowVertices(
				&dst, &transforms[i], &sprites[i], &shadows[i],
				0, 0, 64, 64,
				col, 0.45,
			)
		}
		benchVertexSink = dst[0]
	}
}
