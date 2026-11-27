package katsu2d

import (
	"image/color"
	"math"
	"testing"

	"github.com/edwinsyarief/teishoku"
)

func TestClothExtremeGustNeverInverts(t *testing.T) {
	const (
		cols   = 8
		rows   = 14
		width  = 92
		height = 185
		dt     = 1.0 / 60.0
	)

	w := teishoku.NewWorld(16)
	sys := NewClothSystem()
	sys.Initialize(w)

	tr := NewTransformComponent()
	tr.Position = Point{X: 200, Y: 100}
	spr := SpriteComponent{
		Width:   width,
		Height:  height,
		Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity: 1.0,
	}
	clothCfg := NewClothComponent(
		cols, rows,
		WithClothPinMode(ClothPinTopEdge),
		WithClothGravity(0, 240),
		WithClothWind(V(-1, 0), 1000, 3.5, 160),
		WithClothShading(0.35),
	)

	ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
		tr,
		spr,
		NewGridMeshComponent(cols, rows),
		clothCfg,
	)

	sys.AddWindGust(StrongWindGust{
		StartPos:        V(360, 190),
		EndPos:          V(40, 190),
		Direction:       V(-1, 0),
		Width:           360,
		Length:          320,
		Strength:        640,
		Duration:        3.0,
		FadeInDuration:  0.15,
		FadeOutDuration: 0.35,
	})

	cloth := teishoku.GetComponent[ClothComponent](w, ent)
	mesh := teishoku.GetComponent[MeshComponent](w, ent)
	stride := cols + 1

	assertBannerHealthy := func(phase string, frame int) {
		t.Helper()
		for r := 0; r <= rows; r++ {
			for col := 0; col <= cols; col++ {
				idx := r*stride + col
				px := cloth.posX[idx]
				py := cloth.posY[idx]
				if math.IsNaN(px) || math.IsInf(px, 0) || math.IsNaN(py) || math.IsInf(py, 0) {
					t.Fatalf("%s frame %d: non-finite vertex (%d,%d): (%v,%v)", phase, frame, col, r, px, py)
				}
				if !cloth.pinned[idx] && py < 0 {
					t.Fatalf("%s frame %d: non-pinned vertex (%d,%d) flipped above top bar: posY=%f", phase, frame, col, r, py)
				}
				if r > 0 {
					aboveY := cloth.posY[(r-1)*stride+col]
					if py <= aboveY {
						t.Fatalf("%s frame %d: row %d inverted above row %d at col %d: rowY=%f aboveY=%f",
							phase, frame, r, r-1, col, py, aboveY)
					}
				}
				vx := float64(mesh.Vertices[idx].DstX)
				vy := float64(mesh.Vertices[idx].DstY)
				if math.IsNaN(vx) || math.IsInf(vx, 0) || math.IsNaN(vy) || math.IsInf(vy, 0) {
					t.Fatalf("%s frame %d: non-finite mesh vertex (%d,%d)", phase, frame, col, r)
				}
			}
		}
	}

	// Blast banner with 1000 px/s^2 West wind + 640-strength gust for 240 frames (4 seconds).
	for frame := range 240 {
		sys.Update(w, dt)
		assertBannerHealthy("gust", frame)
	}

	// Let wind settle to calm for 120 frames (2 seconds).
	cloth.SetWind(V(-1, 0), 0)
	cloth.WindTurbulence = 0
	for frame := range 120 {
		sys.Update(w, dt)
		assertBannerHealthy("settle", frame)
	}

	// Bottom row must hang below 120px (near its 185px rest height, not stuck folded up).
	bottomOff := rows * stride
	for col := 0; col <= cols; col++ {
		bottomY := cloth.posY[bottomOff+col]
		if bottomY <= 120.0 {
			t.Fatalf("bottom row vertex (%d,%d) stuck folded up after settle: posY=%f (want > 120.0)", col, rows, bottomY)
		}
	}
}

func TestClothPoleAndCornerWindStability(t *testing.T) {
	const dt = 1.0 / 60.0

	t.Run("TopCornersExtremeWestGust", func(t *testing.T) {
		const cols, rows = 10, 10
		w := teishoku.NewWorld(16)
		sys := NewClothSystem()
		sys.Initialize(w)

		tr := NewTransformComponent()
		tr.Position = Point{X: 120, Y: 80}
		spr := SpriteComponent{
			Width:   120,
			Height:  120,
			Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Opacity: 1.0,
		}
		ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
			tr,
			spr,
			NewGridMeshComponent(cols, rows),
			NewClothComponent(
				cols, rows,
				WithClothPinMode(ClothPinTopCorners),
				WithClothGravity(0, 240),
				WithClothWind(V(-1, 0), 900, 3.0, 140),
			),
		)

		sys.AddWindGust(StrongWindGust{
			StartPos:        V(280, 140),
			EndPos:          V(20, 140),
			Direction:       V(-1, 0),
			Width:           280,
			Length:          280,
			Strength:        640,
			Duration:        2.0,
			FadeInDuration:  0.1,
			FadeOutDuration: 0.2,
		})

		cloth := teishoku.GetComponent[ClothComponent](w, ent)
		stride := cols + 1
		for range 180 {
			sys.Update(w, dt)
			for r := 0; r <= rows; r++ {
				for col := 0; col <= cols; col++ {
					idx := r*stride + col
					if math.IsNaN(cloth.posX[idx]) || math.IsNaN(cloth.posY[idx]) ||
						math.IsInf(cloth.posX[idx], 0) || math.IsInf(cloth.posY[idx], 0) {
						t.Fatalf("non-finite coordinate at (%d,%d)", col, r)
					}
					if !cloth.pinned[idx] && cloth.posY[idx] < 0 {
						t.Fatalf("ClothPinTopCorners flipped above top bar at (%d,%d): posY=%f", col, r, cloth.posY[idx])
					}
				}
			}
		}

		cloth.SetWind(V(-1, 0), 0)
		cloth.WindTurbulence = 0
		for range 120 {
			sys.Update(w, dt)
		}
		bottomOff := rows * stride
		for col := 0; col <= cols; col++ {
			if cloth.posY[bottomOff+col] <= 80.0 {
				t.Fatalf("ClothPinTopCorners bottom vertex (%d,%d) stuck high: posY=%f", col, rows, cloth.posY[bottomOff+col])
			}
		}
	})

	t.Run("LeftAndRightEdgeWestAndEastWind", func(t *testing.T) {
		cases := []struct {
			name        string
			pinMode     ClothPinMode
			primaryDir  Vector
			opposingDir Vector
			freeCol     int
			wantMinX    float64
			wantMaxX    float64
		}{
			{
				name:        "LeftEdgeEastWindWithWestGustRecovery",
				pinMode:     ClothPinLeftEdge,
				primaryDir:  V(1, 0),
				opposingDir: V(-1, 0),
				freeCol:     12,
				wantMinX:    100,
				wantMaxX:    125,
			},
			{
				name:        "RightEdgeWestWindWithEastGustRecovery",
				pinMode:     ClothPinRightEdge,
				primaryDir:  V(-1, 0),
				opposingDir: V(1, 0),
				freeCol:     0,
				wantMinX:    -5,
				wantMaxX:    20,
			},
		}

		const cols, rows = 12, 8
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := teishoku.NewWorld(16)
				sys := NewClothSystem()
				sys.Initialize(w)

				tr := NewTransformComponent()
				tr.Position = Point{X: 200, Y: 200}
				spr := SpriteComponent{
					Width:   120,
					Height:  80,
					Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
					Opacity: 1.0,
				}
				ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
					tr,
					spr,
					NewGridMeshComponent(cols, rows),
					NewClothComponent(
						cols, rows,
						WithClothPinMode(tc.pinMode),
						WithClothGravity(0, 180),
						WithClothWind(tc.primaryDir, 500, 3.0, 120),
					),
				)

				cloth := teishoku.GetComponent[ClothComponent](w, ent)

				// Blast with an opposing gust while reversing wind, then restore primary wind.
				cloth.SetWind(tc.opposingDir, 450)
				sys.AddWindGust(StrongWindGust{
					StartPos:        V(260, 240),
					EndPos:          V(260, 240).Add(tc.opposingDir.ScaleF(200)),
					Direction:       tc.opposingDir,
					Width:           260,
					Length:          260,
					Strength:        500,
					Duration:        0.8,
					FadeInDuration:  0.1,
					FadeOutDuration: 0.2,
				})
				for range 60 {
					sys.Update(w, dt)
					for i := range cloth.posX {
						if math.IsNaN(cloth.posX[i]) || math.IsNaN(cloth.posY[i]) ||
							math.IsInf(cloth.posX[i], 0) || math.IsInf(cloth.posY[i], 0) {
							t.Fatalf("%s opposing gust: non-finite vertex %d", tc.name, i)
						}
					}
				}

				// Restore primary wind and verify the free edge recovers cleanly without fold-lock.
				cloth.SetWind(tc.primaryDir, 600)
				for range 120 {
					sys.Update(w, dt)
					for i := range cloth.posX {
						if math.IsNaN(cloth.posX[i]) || math.IsNaN(cloth.posY[i]) ||
							math.IsInf(cloth.posX[i], 0) || math.IsInf(cloth.posY[i], 0) {
							t.Fatalf("%s recovery: non-finite vertex %d", tc.name, i)
						}
					}
				}

				stride := cols + 1
				midIdx := (rows/2)*stride + tc.freeCol
				freeX := cloth.posX[midIdx]
				if freeX < tc.wantMinX || freeX > tc.wantMaxX {
					t.Fatalf("%s: free edge mid-row posX=%f out of recovered range [%f, %f]",
						tc.name, freeX, tc.wantMinX, tc.wantMaxX)
				}
			})
		}
	})
}

func TestClothLeftFlagDroopsUnderZeroWind(t *testing.T) {
	const (
		cols   = 16
		rows   = 10
		width  = 220
		height = 130
		dt     = 1.0 / 60.0
	)

	w := teishoku.NewWorld(16)
	sys := NewClothSystem()
	sys.Initialize(w)

	tr := NewTransformComponent()
	tr.Position = Point{X: 100, Y: 100}
	spr := SpriteComponent{
		Width:   width,
		Height:  height,
		Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity: 1.0,
	}
	clothCfg := NewClothComponent(
		cols, rows,
		WithClothPinMode(ClothPinLeftEdge),
		WithClothGravity(0, 260),
		WithClothWind(V(1, 0), 0, 3.0, 120),
		WithClothShading(0.35),
	)

	ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
		tr,
		spr,
		NewGridMeshComponent(cols, rows),
		clothCfg,
	)

	cloth := teishoku.GetComponent[ClothComponent](w, ent)
	mesh := teishoku.GetComponent[MeshComponent](w, ent)
	topRightIdx := cols // row 0, col = cols

	// Step 180 frames (3s) at zero wind: flag must drape limp under gravity.
	for frame := range 180 {
		sys.Update(w, dt)
		for i := range cloth.posX {
			if math.IsNaN(cloth.posX[i]) || math.IsNaN(cloth.posY[i]) ||
				math.IsInf(cloth.posX[i], 0) || math.IsInf(cloth.posY[i], 0) {
				t.Fatalf("zero-wind frame %d: non-finite vertex %d", frame, i)
			}
		}
	}

	droopX := cloth.posX[topRightIdx]
	droopY := cloth.posY[topRightIdx]
	if droopY <= 55.0 {
		t.Fatalf("expected top-right corner to droop significantly under zero wind (posY > 55), got posY=%f (posX=%f)", droopY, droopX)
	}
	if droopX >= 200.0 {
		t.Fatalf("expected top-right corner to pull inward under zero wind (posX < 200), got posX=%f (posY=%f)", droopX, droopY)
	}

	// Verify fold shading modulates brightness on the draped cloth.
	minShade := float32(2.0)
	maxShade := float32(0.0)
	for i := range mesh.Vertices {
		r := mesh.Vertices[i].ColorR
		if r < minShade {
			minShade = r
		}
		if r > maxShade {
			maxShade = r
		}
	}
	if maxShade-minShade < 0.05 {
		t.Fatalf("expected visible fold shading contrast when draped limp, got min=%f max=%f", minShade, maxShade)
	}

	// Blow East wind at 420 for 180 frames (3s): flag must unfurl back up and outward.
	cloth.SetWind(V(1, 0), 420)
	for frame := range 180 {
		sys.Update(w, dt)
		for i := range cloth.posX {
			if math.IsNaN(cloth.posX[i]) || math.IsNaN(cloth.posY[i]) ||
				math.IsInf(cloth.posX[i], 0) || math.IsInf(cloth.posY[i], 0) {
				t.Fatalf("wind-unfurl frame %d: non-finite vertex %d", frame, i)
			}
		}
	}

	unfurlX := cloth.posX[topRightIdx]
	unfurlY := cloth.posY[topRightIdx]
	if unfurlX <= 180.0 {
		t.Fatalf("expected top-right corner to unfurl outward under 420 East wind (posX > 180), got posX=%f (posY=%f)", unfurlX, unfurlY)
	}
	if unfurlY >= 50.0 {
		t.Fatalf("expected top-right corner to lift back up under 420 East wind (posY < 50), got posY=%f (posX=%f)", unfurlY, unfurlX)
	}
}

func TestClothFoldDepthSortingAndZeroAlloc(t *testing.T) {
	const (
		cols   = 16
		rows   = 10
		width  = 220
		height = 130
		dt     = 1.0 / 60.0
	)

	w := teishoku.NewWorld(16)
	sys := NewClothSystem()
	sys.Initialize(w)

	tr := NewTransformComponent()
	tr.Position = Point{X: 100, Y: 100}
	spr := SpriteComponent{
		Width:   width,
		Height:  height,
		Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity: 1.0,
	}
	clothCfg := NewClothComponent(
		cols, rows,
		WithClothPinMode(ClothPinLeftEdge),
		WithClothGravity(0, 260),
		WithClothWind(V(1, 0), 0, 3.0, 120),
		WithClothShading(0.35),
	)

	ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
		tr,
		spr,
		NewGridMeshComponent(rows, cols),
		clothCfg,
	)

	mesh := teishoku.GetComponent[MeshComponent](w, ent)
	cloth := teishoku.GetComponent[ClothComponent](w, ent)

	// Warm lazy buffers and verify zero allocations per Update call.
	sys.Update(w, dt)
	allocs := testing.AllocsPerRun(100, func() {
		sys.Update(w, dt)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per ClothSystem.Update, got %v", allocs)
	}

	// Brief gust + droop to reliably fold inner quads while outer columns drape in front.
	cloth.SetWind(V(-1, 0), 320)
	sawFoldedBackAndFront := false
	totalQuads := cols * rows
	wantIndices := totalQuads * 6

	checkOrder := func(phase string, frame int) {
		t.Helper()
		if len(mesh.Indices) != wantIndices {
			t.Fatalf("%s frame %d: len(mesh.Indices)=%d, want %d", phase, frame, len(mesh.Indices), wantIndices)
		}
		seenFront := false
		backCount := 0
		frontCount := 0
		for q := range totalQuads {
			base := q * 6
			tl := mesh.Indices[base]
			tr := mesh.Indices[base+1]
			bl := mesh.Indices[base+2]
			br := mesh.Indices[base+4]

			xTL, yTL := mesh.Vertices[tl].DstX, mesh.Vertices[tl].DstY
			xTR, yTR := mesh.Vertices[tr].DstX, mesh.Vertices[tr].DstY
			xBL, yBL := mesh.Vertices[bl].DstX, mesh.Vertices[bl].DstY
			xBR, yBR := mesh.Vertices[br].DstX, mesh.Vertices[br].DstY

			cross1 := (xTR-xTL)*(yBL-yTL) - (yTR-yTL)*(xBL-xTL)
			cross2 := (xTR-xBL)*(yBR-yBL) - (yTR-yBL)*(xBR-xBL)
			signedArea := float32(0.5) * (cross1 + cross2)

			if signedArea < 0 {
				backCount++
				if seenFront {
					t.Fatalf("%s frame %d: back-facing quad %d (signedArea=%f) appeared after front-facing quad",
						phase, frame, q, signedArea)
				}
			} else {
				frontCount++
				seenFront = true
			}
		}
		if backCount > 0 && frontCount > 0 {
			sawFoldedBackAndFront = true
		}
	}

	for frame := range 45 {
		sys.Update(w, dt)
		checkOrder("fold", frame)
	}
	cloth.SetWind(V(1, 0), 0)
	for frame := range 120 {
		sys.Update(w, dt)
		checkOrder("droop", frame)
	}

	if !sawFoldedBackAndFront {
		t.Fatal("expected drooping/folded flag to produce both back-facing and front-facing quads for depth-sort verification")
	}
}
