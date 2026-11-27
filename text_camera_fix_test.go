package katsu2d

import (
	"math"
	"testing"

	"github.com/edwinsyarief/teishoku"
)

func TestTextAlignmentOffsets(t *testing.T) {
	const (
		textW = 120.0
		textH = 40.0
	)

	cases := []struct {
		name      string
		alignment TextAlignment
		wantOffY  float64
		wantMatTY float64
	}{
		{name: "TopLeft", alignment: TextAlignmentTopLeft, wantOffY: textH, wantMatTY: -textH},
		{name: "TopCenter", alignment: TextAlignmentTopCenter, wantOffY: textH, wantMatTY: -textH},
		{name: "TopRight", alignment: TextAlignmentTopRight, wantOffY: textH, wantMatTY: -textH},
		{name: "MiddleLeft", alignment: TextAlignmentMiddleLeft, wantOffY: textH / 2, wantMatTY: -textH / 2},
		{name: "MiddleCenter", alignment: TextAlignmentMiddleCenter, wantOffY: textH / 2, wantMatTY: -textH / 2},
		{name: "MiddleRight", alignment: TextAlignmentMiddleRight, wantOffY: textH / 2, wantMatTY: -textH / 2},
		{name: "BottomLeft", alignment: TextAlignmentBottomLeft, wantOffY: 0, wantMatTY: 0},
		{name: "BottomCenter", alignment: TextAlignmentBottomCenter, wantOffY: 0, wantMatTY: 0},
		{name: "BottomRight", alignment: TextAlignmentBottomRight, wantOffY: 0, wantMatTY: 0},
	}

	tr := T()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, ok := AlignmentOffsets[tc.alignment]
			if !ok {
				t.Fatalf("missing AlignmentOffsets entry for %v", tc.alignment)
			}

			offX, offY := fn(textW, textH)
			if offX != 0 {
				t.Fatalf("expected offsetX=0, got %v", offX)
			}
			if math.Abs(offY-tc.wantOffY) > 1e-9 {
				t.Fatalf("expected offsetY=%v, got %v", tc.wantOffY, offY)
			}

			comp := TransformComponent{
				Position: Point{X: 100, Y: 200},
				Offset:   Point(V(offX, offY)),
				Scale:    Point{X: 1, Y: 1},
			}
			tr.SetFromComponent(&comp)
			mat := tr.Matrix()
			gotDeltaTY := mat.Element(1, 2) - comp.Position.Y
			if math.Abs(gotDeltaTY-tc.wantMatTY) > 1e-9 {
				t.Fatalf("expected Transform.Matrix() Y translation delta=%v, got %v", tc.wantMatTY, gotDeltaTY)
			}
		})
	}
}

func TestCameraSystemAutoResize(t *testing.T) {
	t.Run("AutoResizeEnabledOnLayoutEvent", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		camBuilder := teishoku.NewBuilder[CameraComponent](w)

		camInit := NewCameraComponent(800, 600, WithCameraPosition(V(100, 50)))
		if !camInit.AutoResizeViewport {
			t.Fatal("expected AutoResizeViewport to default to true")
		}
		camEnt := camBuilder.NewEntityWith(*camInit)

		sys := NewCameraSystem()
		sys.Initialize(w)
		sys.Update(w, 1.0/60.0)

		Publish(w, EngineLayoutChangedEvent{Width: 1200, Height: 750})
		sys.Update(w, 1.0/60.0)

		cam := camBuilder.Get(camEnt)
		if cam == nil {
			t.Fatal("expected camera component to exist")
		}
		if cam.ViewportWidth != 1200 || cam.ViewportHeight != 750 {
			t.Fatalf("expected viewport 1200x750, got %vx%v", cam.ViewportWidth, cam.ViewportHeight)
		}

		active := GetActiveCamera(w)
		if active == nil || !active.HasCamera {
			t.Fatal("expected active camera resource to be populated")
		}
		if active.ViewportWidth != 1200 || active.ViewportHeight != 750 {
			t.Fatalf("expected ActiveCamera viewport 1200x750, got %vx%v", active.ViewportWidth, active.ViewportHeight)
		}

		wantBounds := NewRectangle(100-600, 50-375, 100+600, 50+375)
		if math.Abs(active.ViewBounds.Min.X-wantBounds.Min.X) > 1e-9 ||
			math.Abs(active.ViewBounds.Min.Y-wantBounds.Min.Y) > 1e-9 ||
			math.Abs(active.ViewBounds.Max.X-wantBounds.Max.X) > 1e-9 ||
			math.Abs(active.ViewBounds.Max.Y-wantBounds.Max.Y) > 1e-9 {
			t.Fatalf("expected ViewBounds %+v, got %+v", wantBounds, active.ViewBounds)
		}
	})

	t.Run("AutoResizeDisabledOnLayoutEvent", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		camBuilder := teishoku.NewBuilder[CameraComponent](w)

		camInit := NewCameraComponent(
			800, 600,
			WithCameraPosition(V(100, 50)),
			WithCameraAutoResize(false),
		)
		if camInit.AutoResizeViewport {
			t.Fatal("expected AutoResizeViewport to be false")
		}
		camEnt := camBuilder.NewEntityWith(*camInit)

		sys := NewCameraSystem()
		sys.Initialize(w)
		sys.Update(w, 1.0/60.0)

		Publish(w, EngineLayoutChangedEvent{Width: 1200, Height: 750})
		sys.Update(w, 1.0/60.0)

		cam := camBuilder.Get(camEnt)
		if cam == nil {
			t.Fatal("expected camera component to exist")
		}
		if cam.ViewportWidth != 800 || cam.ViewportHeight != 600 {
			t.Fatalf("expected viewport to remain 800x600, got %vx%v", cam.ViewportWidth, cam.ViewportHeight)
		}

		active := GetActiveCamera(w)
		if active == nil || !active.HasCamera {
			t.Fatal("expected active camera resource to be populated")
		}
		wantBounds := NewRectangle(100-400, 50-300, 100+400, 50+300)
		if math.Abs(active.ViewBounds.Min.X-wantBounds.Min.X) > 1e-9 ||
			math.Abs(active.ViewBounds.Min.Y-wantBounds.Min.Y) > 1e-9 ||
			math.Abs(active.ViewBounds.Max.X-wantBounds.Max.X) > 1e-9 ||
			math.Abs(active.ViewBounds.Max.Y-wantBounds.Max.Y) > 1e-9 {
			t.Fatalf("expected ViewBounds %+v, got %+v", wantBounds, active.ViewBounds)
		}
	})

	t.Run("InitialSyncFromHiResDisplayInfo", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		updateHiResDisplayResource(w, 1600, 900)

		camBuilder := teishoku.NewBuilder[CameraComponent](w)
		autoEnt := camBuilder.NewEntityWith(*NewCameraComponent(800, 600))
		fixedEnt := camBuilder.NewEntityWith(*NewCameraComponent(640, 480, WithCameraAutoResize(false)))

		sys := NewCameraSystem()
		sys.Initialize(w)
		sys.Update(w, 1.0/60.0)

		autoCam := camBuilder.Get(autoEnt)
		if autoCam.ViewportWidth != 1600 || autoCam.ViewportHeight != 900 {
			t.Fatalf("expected auto camera viewport 1600x900, got %vx%v", autoCam.ViewportWidth, autoCam.ViewportHeight)
		}
		fixedCam := camBuilder.Get(fixedEnt)
		if fixedCam.ViewportWidth != 640 || fixedCam.ViewportHeight != 480 {
			t.Fatalf("expected fixed camera viewport 640x480, got %vx%v", fixedCam.ViewportWidth, fixedCam.ViewportHeight)
		}
	})
}
