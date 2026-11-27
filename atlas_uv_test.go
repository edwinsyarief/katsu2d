package katsu2d

import (
	"image"
	"image/color"
	"testing"

	"github.com/edwinsyarief/katsu2d/atlas"
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTextureAtlasSubImageUVs(t *testing.T) {
	tm := NewTextureManager(WithAtlas(true))

	redSrc := ebiten.NewImage(32, 32)
	redSrc.Fill(color.RGBA{R: 255, A: 255})
	redID := tm.Add(redSrc)

	greenSrc := ebiten.NewImage(48, 48)
	greenSrc.Fill(color.RGBA{G: 255, A: 255})
	greenID := tm.Add(greenSrc)

	blueSrc := ebiten.NewImage(64, 64)
	blueSrc.Fill(color.RGBA{B: 255, A: 255})
	blueID := tm.Add(blueSrc)

	cases := []struct {
		name string
		id   int
		w, h int
	}{
		{name: "red32x32", id: redID, w: 32, h: 32},
		{name: "green48x48", id: greenID, w: 48, h: 48},
		{name: "blue64x64", id: blueID, w: 64, h: 64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := tm.Get(tc.id)
			if img == nil {
				t.Fatalf("tm.Get(%d) returned nil", tc.id)
			}
			b := img.Bounds()
			if b.Dx() != tc.w || b.Dy() != tc.h {
				t.Fatalf("expected bounds size %dx%d, got %v", tc.w, tc.h, b)
			}
			if b.Min.X == 0 && b.Min.Y == 0 {
				t.Fatalf("expected non-zero atlas offset for packed texture %d, got %v", tc.id, b)
			}

			// Default NewSpriteComponent stores texture-local Bound (0,0)-(w,h).
			sp := NewSpriteComponent(tc.id, tc.w, tc.h)
			minX, minY, maxX, maxY := resolveSpriteSourceBounds(&sp, img)
			if minX != float32(b.Min.X) || minY != float32(b.Min.Y) || maxX != float32(b.Max.X) || maxY != float32(b.Max.Y) {
				t.Fatalf("default sprite UVs mismatch: got (%.1f,%.1f)-(%.1f,%.1f), want %v",
					minX, minY, maxX, maxY, b)
			}
			if sp.Bound != (Bound{Max: Point{X: float64(tc.w), Y: float64(tc.h)}}) {
				t.Fatalf("expected sp.Bound to remain texture-local, got %+v", sp.Bound)
			}

			// Empty Bound resolves to full sub-image bounds without mutating sp.Bound.
			spEmpty := SpriteComponent{TextureID: tc.id, Width: tc.w, Height: tc.h}
			minX, minY, maxX, maxY = resolveSpriteSourceBounds(&spEmpty, img)
			if minX != float32(b.Min.X) || minY != float32(b.Min.Y) || maxX != float32(b.Max.X) || maxY != float32(b.Max.Y) {
				t.Fatalf("empty bound sprite UVs mismatch: got (%.1f,%.1f)-(%.1f,%.1f), want %v",
					minX, minY, maxX, maxY, b)
			}
			if !IsBoundEmpty(spEmpty.Bound) {
				t.Fatalf("expected empty spEmpty.Bound not to be mutated, got %+v", spEmpty.Bound)
			}

			// Custom local animation frame Bound{Min:{16,0}, Max:{32,16}} offsets by atlas origin.
			spFrame := NewSpriteComponent(tc.id, tc.w, tc.h)
			spFrame.Bound = Bound{
				Min: Point{X: 16, Y: 0},
				Max: Point{X: 32, Y: 16},
			}
			minX, minY, maxX, maxY = resolveSpriteSourceBounds(&spFrame, img)
			wantMinX := float32(b.Min.X + 16)
			wantMinY := float32(b.Min.Y)
			wantMaxX := float32(b.Min.X + 32)
			wantMaxY := float32(b.Min.Y + 16)
			if minX != wantMinX || minY != wantMinY || maxX != wantMaxX || maxY != wantMaxY {
				t.Fatalf("frame UVs mismatch: got (%.1f,%.1f)-(%.1f,%.1f), want (%.1f,%.1f)-(%.1f,%.1f)",
					minX, minY, maxX, maxY, wantMinX, wantMinY, wantMaxX, wantMaxY)
			}
		})
	}

	t.Run("SyncMeshUVsPreservesDeformationAndUpdatesOnSwap", func(t *testing.T) {
		mesh := NewGridMeshComponent(2, 2)
		sp := NewSpriteComponent(redID, 32, 32)
		GenerateMesh(&mesh, &sp)

		const centerIdx = 4
		origDstX := mesh.Vertices[centerIdx].DstX + 7.5
		origDstY := mesh.Vertices[centerIdx].DstY - 3.25
		mesh.Vertices[centerIdx].DstX = origDstX
		mesh.Vertices[centerIdx].DstY = origDstY
		mesh.Vertices[centerIdx].ColorR = 0.4
		mesh.Vertices[centerIdx].ColorG = 0.6
		mesh.Vertices[centerIdx].ColorB = 0.8
		mesh.Vertices[centerIdx].ColorA = 0.9

		redImg := tm.Get(redID)
		redB := redImg.Bounds()
		SyncMeshUVs(&mesh, &sp, redImg)

		assertGridUVs := func(rect image.Rectangle) {
			t.Helper()
			if len(mesh.Vertices) != 9 || len(mesh.BaseVertices) != 9 {
				t.Fatalf("unexpected vertex counts: verts=%d base=%d", len(mesh.Vertices), len(mesh.BaseVertices))
			}
			for r := 0; r <= 2; r++ {
				wantV := float32(rect.Min.Y) + float32(rect.Dy())*(float32(r)*0.5)
				for c := 0; c <= 2; c++ {
					wantU := float32(rect.Min.X) + float32(rect.Dx())*(float32(c)*0.5)
					idx := r*3 + c
					if mesh.Vertices[idx].SrcX != wantU || mesh.Vertices[idx].SrcY != wantV {
						t.Fatalf("vertex[%d] UV = (%.2f,%.2f), want (%.2f,%.2f)",
							idx, mesh.Vertices[idx].SrcX, mesh.Vertices[idx].SrcY, wantU, wantV)
					}
					if mesh.BaseVertices[idx].SrcX != wantU || mesh.BaseVertices[idx].SrcY != wantV {
						t.Fatalf("baseVertex[%d] UV = (%.2f,%.2f), want (%.2f,%.2f)",
							idx, mesh.BaseVertices[idx].SrcX, mesh.BaseVertices[idx].SrcY, wantU, wantV)
					}
				}
			}
			v := mesh.Vertices[centerIdx]
			if v.DstX != origDstX || v.DstY != origDstY ||
				v.ColorR != 0.4 || v.ColorG != 0.6 || v.ColorB != 0.8 || v.ColorA != 0.9 {
				t.Fatalf("SyncMeshUVs clobbered deformed vertex attributes: %+v", v)
			}
		}

		assertGridUVs(redB)

		// Swap TextureID to blue and verify UVs update cleanly without clobbering Dst/Color.
		blueImg := tm.Get(blueID)
		blueB := blueImg.Bounds()
		sp.TextureID = blueID
		sp.Width = 64
		sp.Height = 64
		sp.Bound = Bound{Max: Point{X: 64, Y: 64}}
		SyncMeshUVs(&mesh, &sp, blueImg)
		assertGridUVs(blueB)

		// Update local animation frame Bound and verify UVs shift to the frame sub-rect.
		sp.Bound = Bound{Min: Point{X: 16, Y: 0}, Max: Point{X: 32, Y: 16}}
		SyncMeshUVs(&mesh, &sp, blueImg)
		assertGridUVs(image.Rect(blueB.Min.X+16, blueB.Min.Y, blueB.Min.X+32, blueB.Min.Y+16))
	})

	t.Run("ParticleEmitterLocalBounds", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		teishoku.AddResource(w.Resources(), tm)

		emitterSys := NewParticleEmitterSystem()
		emitterSys.Initialize(w)

		teishoku.NewBuilder2[TransformComponent, ParticleEmitterComponent](w).NewEntityWith(
			TransformComponent{Scale: Point{X: 1, Y: 1}},
			ParticleEmitterComponent{
				TextureIDs:       []int{redID},
				MaxParticles:     4,
				BurstCount:       1,
				ParticleLifetime: 1.0,
				MinScale:         1.0,
				MaxScale:         1.0,
			},
		)

		emitterSys.Update(w, 1.0/60.0)

		pFilter := teishoku.NewFilter2[SpriteComponent, ParticleComponent](w)
		count := 0
		pFilter.EachChunk(func(sprites []SpriteComponent, _ []ParticleComponent) {
			for i := range sprites {
				count++
				sp := &sprites[i]
				if sp.Bound != (Bound{Max: Point{X: 32, Y: 32}}) {
					t.Fatalf("expected texture-local particle bound (0,0)-(32,32), got %+v", sp.Bound)
				}
				redB := tm.Get(redID).Bounds()
				minX, minY, maxX, maxY := resolveSpriteSourceBounds(sp, tm.Get(redID))
				if minX != float32(redB.Min.X) || minY != float32(redB.Min.Y) || maxX != float32(redB.Max.X) || maxY != float32(redB.Max.Y) {
					t.Fatalf("resolved particle UVs mismatch: got (%.1f,%.1f)-(%.1f,%.1f), want %v",
						minX, minY, maxX, maxY, redB)
				}
			}
		})
		if count != 1 {
			t.Fatalf("expected 1 spawned particle, got %d", count)
		}
	})

	t.Run("AtlasImageDrawImagePreservesCallerOpts", func(t *testing.T) {
		a := atlas.New(128, 128, nil)
		_ = a.NewImage(16, 16)
		sub := a.NewImage(32, 32)
		if sub == nil {
			t.Fatal("failed to allocate atlas sub-image")
		}
		src := ebiten.NewImage(16, 16)
		src.Fill(color.White)

		var opts ebiten.DrawImageOptions
		opts.GeoM.Scale(2, 2)
		origGeoM := opts.GeoM

		sub.DrawImage(src, &opts)
		if opts.GeoM != origGeoM {
			t.Fatalf("atlas.Image.DrawImage mutated caller's GeoM: got %+v, want %+v", opts.GeoM, origGeoM)
		}
	})
}
