package katsu2d

import (
	"image/color"
	"math"
	"testing"

	"github.com/edwinsyarief/teishoku"
)

func TestGrassControllerSystem(t *testing.T) {
	w := teishoku.NewWorld(256)
	ctrlInit := NewGrassControllerComponent(
		w, nil, 128, 128, 0, 0,
		WithGrassTileSize(32),
		WithGrassDensity(5),
		WithGrassOrderable(true),
	)
	ctrlEnt := teishoku.NewBuilder[GrassControllerComponent](w).NewEntityWith(*ctrlInit)
	ctrl := teishoku.GetComponent[GrassControllerComponent](w, ctrlEnt)

	grassFilter := teishoku.NewFilter2[TransformComponent, GrassComponent](w)
	orderFilter := teishoku.NewFilter2[GrassComponent, OrderableComponent](w)

	const expectedBlades = 4 * 4 * 5 // 80
	if got := grassFilter.Count(); got != expectedBlades {
		t.Fatalf("expected %d grass blades, got %d", expectedBlades, got)
	}
	if got := orderFilter.Count(); got != expectedBlades {
		t.Fatalf("expected %d orderable grass blades, got %d", expectedBlades, got)
	}

	sys := NewGrassControllerSystem()
	sys.Initialize(w)
	sys.Initialize(w) // idempotent guard

	// Advance with ambient wind; blade rotations should become non-zero.
	for range 10 {
		sys.Update(w, 1.0/60.0)
	}

	moved := 0
	grassFilter.EachChunk(func(transforms []TransformComponent, grasses []GrassComponent) {
		for i := range transforms {
			if transforms[i].Rotation != 0 {
				moved++
			}
			if math.IsNaN(transforms[i].Rotation) || math.IsNaN(grasses[i].CurrentSway) {
				t.Fatalf("NaN detected in grass sway state at index %d", i)
			}
		}
	})
	if moved == 0 {
		t.Fatal("expected ambient wind to change grass blade rotations")
	}

	// Add radial ForceSource near the center and verify nearby blades bend.
	ctrl.SetForcePositions(ForceSource{
		Position: V(64, 64),
		Radius:   64,
		Strength: 8.0,
	})
	sys.Update(w, 1.0/60.0)

	bentByForce := 0
	grassFilter.EachChunk(func(_ []TransformComponent, grasses []GrassComponent) {
		for i := range grasses {
			if grasses[i].InteractionSway != 0 || grasses[i].AccumulatedForce != 0 {
				bentByForce++
			}
		}
	})
	if bentByForce == 0 {
		t.Fatal("expected ForceSource to bend nearby grass blades")
	}
	ctrl.SetForcePositions()

	// Add StrongWindGust: active while within duration, then expires cleanly.
	ctrl.AddStrongWindGust(StrongWindGust{
		StartPos:        V(0, 64),
		EndPos:          V(128, 64),
		Width:           128,
		Length:          128,
		Strength:        12.0,
		Duration:        0.2,
		FadeInDuration:  0.05,
		FadeOutDuration: 0.05,
	})
	if len(ctrl.StrongWindGusts) != 1 {
		t.Fatalf("expected 1 active wind gust, got %d", len(ctrl.StrongWindGusts))
	}

	sys.Update(w, 0.1) // mid-gust
	if len(ctrl.StrongWindGusts) != 1 {
		t.Fatalf("expected gust to remain active mid-duration, got %d", len(ctrl.StrongWindGusts))
	}
	gustForceApplied := false
	grassFilter.EachChunk(func(_ []TransformComponent, grasses []GrassComponent) {
		for i := range grasses {
			if grasses[i].AccumulatedForce != 0 {
				gustForceApplied = true
				break
			}
		}
	})
	if !gustForceApplied {
		t.Fatal("expected active StrongWindGust to apply force to blades")
	}

	// Advance past gust duration (0.1 + 0.15 = 0.25 >= 0.2).
	sys.Update(w, 0.15)
	if len(ctrl.StrongWindGusts) != 0 {
		t.Fatalf("expected expired StrongWindGust to be removed, got len=%d", len(ctrl.StrongWindGusts))
	}

	ctrl.Dispose()
	if ctrl.ExternalForceSources != nil || ctrl.StrongWindGusts != nil || ctrl.noiseMap != nil {
		t.Fatal("expected Dispose to release internal slices")
	}
	if ctrl.Quadtree != nil {
		t.Fatalf("expected nil quadtree after Dispose, got %v", ctrl.Quadtree)
	}

	// Verify DisposeEntities removes all spawned grass entities from the world.
	// Remove the first controller's orphaned blades before spawning a clean controller.
	var toRemove []teishoku.Entity
	q := grassFilter.Query()
	for q.Next() {
		toRemove = append(toRemove, q.Entity())
	}
	w.RemoveEntities(toRemove)

	ctrl2Init := NewGrassControllerComponent(
		w, nil, 128, 128, 0, 0,
		WithGrassTileSize(32),
		WithGrassDensity(5),
	)
	ctrl2Ent := teishoku.NewBuilder[GrassControllerComponent](w).NewEntityWith(*ctrl2Init)
	ctrl2 := teishoku.GetComponent[GrassControllerComponent](w, ctrl2Ent)
	if got := grassFilter.Count(); got != expectedBlades {
		t.Fatalf("expected %d grass blades before DisposeEntities, got %d", expectedBlades, got)
	}
	ctrl2.DisposeEntities(w)
	if got := grassFilter.Count(); got != 0 {
		t.Fatalf("expected 0 grass blades after DisposeEntities, got %d", got)
	}
	if ctrl2.Quadtree != nil {
		t.Fatalf("expected nil quadtree after DisposeEntities, got %v", ctrl2.Quadtree)
	}
}

func TestFoliageSystem(t *testing.T) {
	w := teishoku.NewWorld(16)
	sys := NewFoliageSystem()
	sys.Initialize(w)
	sys.Initialize(w) // idempotent guard

	ctrlEnt := teishoku.NewBuilder[FoliageControllerComponent](w).NewEntityWith(*NewFoliageControllerComponent())

	spr := SpriteComponent{
		Width:   64,
		Height:  64,
		Color:   color.RGBA{255, 255, 255, 255},
		Opacity: 1.0,
	}
	e := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, FoliageComponent](w).NewEntityWith(
		NewTransformComponent(),
		spr,
		NewGridMeshComponent(4, 4),
		NewFoliageComponent(1.0, V(0.5, 1.0)),
	)

	sys.Update(w, 0.1)

	mesh := teishoku.GetComponent[MeshComponent](w, e)
	if len(mesh.Vertices) != 25 || len(mesh.BaseVertices) != 25 {
		t.Fatalf("expected 25 vertices for 4x4 grid, got Vertices=%d BaseVertices=%d", len(mesh.Vertices), len(mesh.BaseVertices))
	}

	topDisplaced := 0
	bottomCount := 0
	var pivotColorR, maxTopColorR float32
	hasCrestTint := false
	for i := range mesh.BaseVertices {
		base := mesh.BaseVertices[i]
		curr := mesh.Vertices[i]
		if curr.ColorA != 1 {
			t.Fatalf("expected vertex %d ColorA=1, got %f", i, curr.ColorA)
		}
		if math.Abs(float64(curr.ColorR-curr.ColorB)) > 1e-4 {
			hasCrestTint = true
		}
		if math.Abs(float64(base.DstY)) < 1e-5 { // normalizedY = 0 (top edge)
			if curr.DstX != base.DstX {
				topDisplaced++
			}
			if curr.ColorR > maxTopColorR {
				maxTopColorR = curr.ColorR
			}
		} else if math.Abs(float64(base.DstY-64)) < 1e-5 { // normalizedY = 1 (bottom pivot)
			bottomCount++
			if curr.DstX != base.DstX || curr.DstY != base.DstY {
				t.Fatalf("bottom pivot vertex %d moved from base: base=(%f,%f) curr=(%f,%f)", i, base.DstX, base.DstY, curr.DstX, curr.DstY)
			}
			if curr.ColorR >= 1.0 || math.Abs(float64(curr.ColorR-curr.ColorG)) > 1e-5 || math.Abs(float64(curr.ColorG-curr.ColorB)) > 1e-5 {
				t.Fatalf("expected AO darkened grayscale on bottom pivot vertex %d, got RGB=(%f,%f,%f)", i, curr.ColorR, curr.ColorG, curr.ColorB)
			}
			pivotColorR = curr.ColorR
		}
	}
	if topDisplaced == 0 {
		t.Fatal("expected top vertices (normalizedY=0) to be displaced by wind")
	}
	if bottomCount != 5 {
		t.Fatalf("expected 5 bottom vertices on 4x4 grid, got %d", bottomCount)
	}
	if maxTopColorR <= pivotColorR {
		t.Fatalf("expected top vertices to be brighter than AO pivot base: top=%f pivot=%f", maxTopColorR, pivotColorR)
	}
	if !hasCrestTint {
		t.Fatal("expected per-vertex ColorR/G/B crest shading modulation across foliage mesh")
	}

	// Verify world-space radial forces (SetForcePositions) and traveling gusts (AddStrongWindGust)
	// independently of ambient wind by zeroing ambient wind/ripple amplitudes.
	ctrl := teishoku.GetComponent[FoliageControllerComponent](w, ctrlEnt)
	ctrl.WindForce = 0
	ctrl.RippleStrength = 0

	sys.Update(w, 0.016)
	for i := range mesh.BaseVertices {
		if math.Abs(float64(mesh.Vertices[i].DstX-mesh.BaseVertices[i].DstX)) > 1e-5 {
			t.Fatalf("expected zero X displacement without wind/forces, got v%d=%f want=%f", i, mesh.Vertices[i].DstX, mesh.BaseVertices[i].DstX)
		}
	}

	// Radial force positioned to the left of the foliage pushes free vertices in +X.
	ctrl.SetForcePositions(ForceSource{
		Position: V(-16, 16),
		Radius:   160,
		Strength: 20.0,
	})
	sys.Update(w, 0.016)

	pushedRight := 0
	for i := range mesh.BaseVertices {
		if math.Abs(float64(mesh.BaseVertices[i].DstY)) < 1e-5 && mesh.Vertices[i].DstX > mesh.BaseVertices[i].DstX+0.5 {
			pushedRight++
		}
	}
	if pushedRight == 0 {
		t.Fatal("expected SetForcePositions to displace top foliage vertices away from force source")
	}
	ctrl.SetForcePositions()

	// Strong wind gust traveling across the foliage displaces vertices and expires cleanly.
	ctrl.AddStrongWindGust(StrongWindGust{
		StartPos:        V(0, 32),
		EndPos:          V(64, 32),
		Width:           160,
		Length:          160,
		Strength:        25.0,
		Duration:        0.2,
		FadeInDuration:  0.05,
		FadeOutDuration: 0.05,
	})
	if len(ctrl.StrongWindGusts) != 1 {
		t.Fatalf("expected 1 active foliage wind gust, got %d", len(ctrl.StrongWindGusts))
	}

	sys.Update(w, 0.1) // mid-gust
	if len(ctrl.StrongWindGusts) != 1 {
		t.Fatalf("expected foliage wind gust to remain active mid-duration, got %d", len(ctrl.StrongWindGusts))
	}
	gustDisplaced := 0
	for i := range mesh.BaseVertices {
		if math.Abs(float64(mesh.BaseVertices[i].DstY)) < 1e-5 && mesh.Vertices[i].DstX > mesh.BaseVertices[i].DstX+0.5 {
			gustDisplaced++
		}
	}
	if gustDisplaced == 0 {
		t.Fatal("expected AddStrongWindGust to displace top foliage vertices along gust direction")
	}

	// Advance past gust duration (0.1 + 0.15 = 0.25 >= 0.2).
	sys.Update(w, 0.15)
	if len(ctrl.StrongWindGusts) != 0 {
		t.Fatalf("expected expired foliage StrongWindGust to be removed, got len=%d", len(ctrl.StrongWindGusts))
	}
}

func TestParticleSystem(t *testing.T) {
	t.Run("PresetsSpawnCapsAndLifetimeCleanup", func(t *testing.T) {
		presets := []struct {
			name    string
			emitter *ParticleEmitterComponent
			burst   bool
		}{
			{"Fire", FirePreset(0), false},
			{"Rain", RainPreset(0), false},
			{"Smoke", SmokePreset([]int{0, 1}), false},
			{"Explosion", ExplosionPreset([]int{0}), true},
			{"Whimsical", WhimsicalPreset(0), false},
		}

		for _, tc := range presets {
			t.Run(tc.name, func(t *testing.T) {
				w := teishoku.NewWorld(2048)
				emSys := NewParticleEmitterSystem()
				upSys := NewParticleUpdateSystem()
				emSys.Initialize(w)
				upSys.Initialize(w)

				pFilter := teishoku.NewFilter3[TransformComponent, SpriteComponent, ParticleComponent](w)

				// Baseline entity (e.g. camera/world marker) to verify zero leaks back to baseline.
				_ = teishoku.NewBuilder[TransformComponent](w).NewEntityWith(NewTransformComponent())
				baseline := w.Len()

				tr := NewTransformComponent()
				tr.Position = Point{X: 100, Y: 100}
				emEnt := teishoku.NewBuilder2[TransformComponent, ParticleEmitterComponent](w).NewEntityWith(tr, *tc.emitter)

				// Step emitters aggressively to verify MaxParticles cap.
				for range 20 {
					emSys.Update(w, 0.1)
					if count := pFilter.Count(); count > tc.emitter.MaxParticles {
						t.Fatalf("%s exceeded MaxParticles (%d > %d)", tc.name, count, tc.emitter.MaxParticles)
					}
				}
				spawned := pFilter.Count()
				if spawned == 0 {
					t.Fatalf("%s spawned 0 particles", tc.name)
				}
				if tc.burst && spawned != tc.emitter.MaxParticles {
					t.Fatalf("Explosion burst expected %d particles, got %d", tc.emitter.MaxParticles, spawned)
				}

				// Remove emitter entity and verify particles survive.
				w.RemoveEntity(emEnt)
				if got := pFilter.Count(); got != spawned {
					t.Fatalf("%s particles did not survive emitter removal: got %d, want %d", tc.name, got, spawned)
				}

				// Step once and confirm particles move/update without expiring yet.
				upSys.Update(w, 0.01)
				if pFilter.Count() == 0 {
					t.Fatalf("%s particles expired prematurely", tc.name)
				}

				// Advance well past ParticleLifetime so every particle expires and is removed.
				totalSteps := int(math.Ceil((tc.emitter.ParticleLifetime + 1.0) / 0.1))
				for range totalSteps {
					upSys.Update(w, 0.1)
				}

				if pFilter.Count() != 0 {
					t.Fatalf("%s expected 0 particles after lifetime expiry, got %d", tc.name, pFilter.Count())
				}
				if w.Len() != baseline {
					t.Fatalf("%s entity leak: expected world Len=%d, got %d", tc.name, baseline, w.Len())
				}
			})
		}
	})

	t.Run("MotionFadeScaleRotation", func(t *testing.T) {
		w := teishoku.NewWorld(64)
		emSys := NewParticleEmitterSystem()
		upSys := NewParticleUpdateSystem()
		emSys.Initialize(w)
		upSys.Initialize(w)

		em := WhimsicalPreset(0)
		em.Active = false
		em.BurstCount = 1
		em.MaxParticles = 1
		em.ParticleLifetime = 2.0
		em.InitialParticleSpeedMin = 50
		em.InitialParticleSpeedMax = 50
		em.MinScale = 0.8
		em.MaxScale = 0.8
		em.TargetScaleMin = 0.2
		em.TargetScaleMax = 0.2
		em.RotationSpeedMin = 2.0
		em.RotationSpeedMax = 2.0
		em.InitialColorMin = color.RGBA{200, 100, 50, 255}
		em.InitialColorMax = color.RGBA{200, 100, 50, 255}
		em.TargetColorMin = color.RGBA{50, 200, 255, 0}
		em.TargetColorMax = color.RGBA{50, 200, 255, 0}

		teishoku.NewBuilder2[TransformComponent, ParticleEmitterComponent](w).NewEntityWith(NewTransformComponent(), *em)
		emSys.Update(w, 0.016)

		pFilter := teishoku.NewFilter3[TransformComponent, SpriteComponent, ParticleComponent](w)
		var startPos Point
		var startRot float64
		pFilter.EachChunk(func(trs []TransformComponent, _ []SpriteComponent, _ []ParticleComponent) {
			startPos = trs[0].Position
			startRot = trs[0].Rotation
		})

		// First quarter of lifetime (tp = 0.25): FadeInOut and ScaleInOut ramp up from 0 toward initial.
		upSys.Update(w, 0.5)
		var midPos Point
		var midScale Point
		var midRot float64
		var midAlpha uint8
		pFilter.EachChunk(func(trs []TransformComponent, sps []SpriteComponent, _ []ParticleComponent) {
			midPos = trs[0].Position
			midScale = trs[0].Scale
			midRot = trs[0].Rotation
			midAlpha = sps[0].Color.A
		})

		if midPos == startPos {
			t.Fatal("expected particle position to move")
		}
		if math.Abs(midScale.X-0.4) > 1e-6 || math.Abs(midScale.Y-0.4) > 1e-6 {
			t.Fatalf("expected ScaleInOut at tp=0.25 to be 0.4, got %+v", midScale)
		}
		if math.Abs((midRot-startRot)-1.0) > 1e-6 {
			t.Fatalf("expected rotation delta=1.0 rad, got %f", midRot-startRot)
		}
		if midAlpha == 0 || midAlpha == 255 {
			t.Fatalf("expected intermediate FadeInOut alpha at tp=0.25, got %d", midAlpha)
		}

		// Three quarters of lifetime (tp = 0.75): FadeInOut and ScaleInOut ramp toward target.
		upSys.Update(w, 1.0)
		pFilter.EachChunk(func(trs []TransformComponent, sps []SpriteComponent, _ []ParticleComponent) {
			if math.Abs(trs[0].Scale.X-0.5) > 1e-6 {
				t.Fatalf("expected ScaleInOut at tp=0.75 to be 0.5, got %f", trs[0].Scale.X)
			}
			if sps[0].Color.A >= midAlpha && midAlpha > 130 {
				t.Fatalf("expected FadeInOut alpha to decrease in second half: mid=%d late=%d", midAlpha, sps[0].Color.A)
			}
		})
	})
}

func BenchmarkGrassUpdate1K(b *testing.B) {
	w := teishoku.NewWorld(1200)
	ctrlInit := NewGrassControllerComponent(
		w, nil, 1024, 1024, 0, 0,
		WithGrassTileSize(32),
		WithGrassDensity(1), // 32 * 32 * 1 = 1,024 blades
	)
	ctrlEnt := teishoku.NewBuilder[GrassControllerComponent](w).NewEntityWith(*ctrlInit)
	ctrl := teishoku.GetComponent[GrassControllerComponent](w, ctrlEnt)
	ctrl.SetForcePositions(ForceSource{
		Position: V(512, 512),
		Radius:   128,
		Strength: 5.0,
	})

	sys := NewGrassControllerSystem()
	sys.Initialize(w)
	sys.Update(w, 1.0/60.0) // warm reusable buffers

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sys.Update(w, 1.0/60.0)
	}
}

func BenchmarkFoliageUpdate(b *testing.B) {
	w := teishoku.NewWorld(128)
	teishoku.NewBuilder[FoliageControllerComponent](w).NewEntityWith(*NewFoliageControllerComponent())

	fb := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, FoliageComponent](w)
	for i := range 100 {
		spr := SpriteComponent{
			Width:   64,
			Height:  64,
			Color:   color.RGBA{255, 255, 255, 255},
			Opacity: 1.0,
		}
		fb.NewEntityWith(
			NewTransformComponent(),
			spr,
			NewGridMeshComponent(4, 4), // 5x5 grid vertices
			NewFoliageComponent(float64(i)*0.37, V(0.5, 1.0)),
		)
	}

	sys := NewFoliageSystem()
	sys.Initialize(w)
	sys.Update(w, 1.0/60.0) // warm mesh vertices/indices on first dirty frame

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sys.Update(w, 1.0/60.0)
	}
}

func BenchmarkParticleUpdate1K(b *testing.B) {
	w := teishoku.NewWorld(1200)
	pb := teishoku.NewBuilder3[TransformComponent, SpriteComponent, ParticleComponent](w)

	for i := range 1000 {
		tr := NewTransformComponent()
		tr.Position = Point{X: float64(i % 100), Y: float64(i / 100)}
		sp := NewSpriteComponent(0, 8, 8)
		p := ParticleComponent{
			Gravity:          V(0, 20),
			Velocity:         V(10, -15),
			Lifetime:         1e9,
			TotalLifetime:    1e9,
			InitialColor:     color.RGBA{255, 200, 100, 255},
			TargetColor:      color.RGBA{255, 0, 0, 0},
			InitialScale:     1.0,
			TargetScale:      0.2,
			RotationSpeed:    1.5,
			NoiseOffsetX:     float64(i) * 0.5,
			NoiseOffsetY:     float64(i) * 0.8,
			NoiseFactor:      0.1,
			FadeMode:         ParticleFadeModeFadeOut,
			ScaleMode:        ParticleScaleModeScaleOut,
			DirectionMode:    ParticleDirectionModeNoise,
			UseRotationSpeed: true,
		}
		pb.NewEntityWith(tr, sp, p)
	}

	sys := NewParticleUpdateSystem()
	sys.Initialize(w)
	sys.Update(w, 1.0/60.0) // warm reusable scratch slice

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sys.Update(w, 1.0/60.0)
	}
}
