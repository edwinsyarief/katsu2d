package katsu2d

import (
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestClothSystemSimulationAndPins(t *testing.T) {
	t.Run("PinModesGravityWindAndFoldShading", func(t *testing.T) {
		pinModes := []struct {
			name     string
			mode     ClothPinMode
			isPinned func(col, row, cols, rows int) bool
		}{
			{
				name: "ClothPinLeftEdge",
				mode: ClothPinLeftEdge,
				isPinned: func(col, _, _, _ int) bool {
					return col == 0
				},
			},
			{
				name: "ClothPinTopEdge",
				mode: ClothPinTopEdge,
				isPinned: func(_, row, _, _ int) bool {
					return row == 0
				},
			},
			{
				name: "ClothPinTopCorners",
				mode: ClothPinTopCorners,
				isPinned: func(col, row, cols, _ int) bool {
					return row == 0 && (col == 0 || col == cols)
				},
			},
		}

		const cols, rows = 8, 6
		for _, tc := range pinModes {
			t.Run(tc.name, func(t *testing.T) {
				w := teishoku.NewWorld(16)
				sys := NewClothSystem()
				sys.Initialize(w)
				sys.Initialize(w) // idempotent guard

				spr := SpriteComponent{
					Width:   96,
					Height:  64,
					Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
					Opacity: 1.0,
				}
				clothCfg := NewClothComponent(
					cols, rows,
					WithClothPinMode(tc.mode),
					WithClothGravity(0, 260),
					WithClothWind(V(1, 0), 380, 3.5, 140),
					WithClothShading(0.35),
				)
				ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
					NewTransformComponent(),
					spr,
					NewGridMeshComponent(cols, rows),
					clothCfg,
				)

				for range 20 {
					sys.Update(w, 1.0/60.0)
				}

				mesh := teishoku.GetComponent[MeshComponent](w, ent)
				stride := cols + 1
				if len(mesh.Vertices) != stride*(rows+1) {
					t.Fatalf("unexpected vertex count %d, want %d", len(mesh.Vertices), stride*(rows+1))
				}

				freeMoved := 0
				shadedCount := 0
				for r := 0; r <= rows; r++ {
					for col := 0; col <= cols; col++ {
						idx := r*stride + col
						base := mesh.BaseVertices[idx]
						curr := mesh.Vertices[idx]
						if tc.isPinned(col, r, cols, rows) {
							if !approxEqual32(curr.DstX, base.DstX, 1e-5) || !approxEqual32(curr.DstY, base.DstY, 1e-5) {
								t.Fatalf("pinned vertex (%d,%d) moved: base=(%f,%f) curr=(%f,%f)", col, r, base.DstX, base.DstY, curr.DstX, curr.DstY)
							}
						} else if math.Abs(float64(curr.DstX-base.DstX)) > 0.1 || math.Abs(float64(curr.DstY-base.DstY)) > 0.1 {
							freeMoved++
						}

						if curr.ColorA != 1.0 {
							t.Fatalf("expected vertex ColorA=1.0, got %f", curr.ColorA)
						}
						if curr.ColorR < float32(clothMinShade)-1e-5 || curr.ColorR > float32(clothMaxShade)+1e-5 {
							t.Fatalf("vertex ColorR out of shade range: %f", curr.ColorR)
						}
						if math.Abs(float64(curr.ColorR-1.0)) > 1e-3 ||
							math.Abs(float64(curr.ColorG-1.0)) > 1e-3 ||
							math.Abs(float64(curr.ColorB-1.0)) > 1e-3 {
							shadedCount++
						}
					}
				}
				if freeMoved == 0 {
					t.Fatal("expected free cloth vertices to move under gravity and wind")
				}
				if shadedCount == 0 {
					t.Fatal("expected fold shading to modulate MeshComponent.Vertices ColorR/G/B")
				}

				// Specifically check top middle vertex moves when only top corners are pinned.
				if tc.mode == ClothPinTopCorners {
					midIdx := cols / 2
					base := mesh.BaseVertices[midIdx]
					curr := mesh.Vertices[midIdx]
					if curr.DstY <= base.DstY+0.5 {
						t.Fatalf("expected top-middle vertex to sag under gravity in ClothPinTopCorners: baseY=%f currY=%f", base.DstY, curr.DstY)
					}
				}
			})
		}
	})

	t.Run("ForcesGustsAndDtSpikes", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewClothSystem()
		sys.Initialize(w)

		tr := NewTransformComponent()
		tr.Position = Point{X: 100, Y: 100}
		spr := SpriteComponent{
			Width:   64,
			Height:  64,
			Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Opacity: 1.0,
		}
		clothCfg := NewClothComponent(
			8, 6,
			WithClothPinMode(ClothPinTopEdge),
			WithClothGravity(0, 0),
			WithClothWind(V(1, 0), 0, 0, 0),
			WithClothShading(0),
		)
		ent := teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
			tr,
			spr,
			NewGridMeshComponent(8, 6),
			clothCfg,
		)

		// Baseline step with no forces: vertices stay at rest and shading is disabled (white).
		sys.Update(w, 1.0/60.0)
		mesh := teishoku.GetComponent[MeshComponent](w, ent)
		for i := range mesh.Vertices {
			if mesh.Vertices[i].ColorR != 1 || mesh.Vertices[i].ColorG != 1 || mesh.Vertices[i].ColorB != 1 {
				t.Fatalf("expected unshaded white vertex when ShadeStrength=0, got %+v", mesh.Vertices[i])
			}
			if !approxEqual32(mesh.Vertices[i].DstX, mesh.BaseVertices[i].DstX, 1e-5) ||
				!approxEqual32(mesh.Vertices[i].DstY, mesh.BaseVertices[i].DstY, 1e-5) {
				t.Fatalf("expected rest pose without forces, v%d moved", i)
			}
		}

		// Apply world-space radial force to the left of the cloth center -> pushes vertices in +X.
		sys.SetForceSources(ForceSource{
			Position: V(96, 148),
			Radius:   128,
			Strength: 45.0,
		})
		for range 5 {
			sys.Update(w, 1.0/60.0)
		}
		displacedByForce := 0
		for i := range mesh.Vertices {
			if mesh.Vertices[i].DstX > mesh.BaseVertices[i].DstX+0.25 {
				displacedByForce++
			}
		}
		if displacedByForce == 0 {
			t.Fatal("expected radial ForceSource to displace free cloth vertices")
		}

		// Reset cloth state and clear force sources.
		sys.SetForceSources()
		cloth := teishoku.GetComponent[ClothComponent](w, ent)
		cloth.Reset()
		sys.Update(w, 1.0/60.0)
		for i := range mesh.Vertices {
			if !approxEqual32(mesh.Vertices[i].DstX, mesh.BaseVertices[i].DstX, 1e-4) ||
				!approxEqual32(mesh.Vertices[i].DstY, mesh.BaseVertices[i].DstY, 1e-4) {
				t.Fatalf("expected Reset() to restore rest pose, v%d=(%f,%f) base=(%f,%f)",
					i, mesh.Vertices[i].DstX, mesh.Vertices[i].DstY, mesh.BaseVertices[i].DstX, mesh.BaseVertices[i].DstY)
			}
		}

		// Traveling world-space StrongWindGust displaces vertices along gust direction and expires.
		sys.AddWindGust(StrongWindGust{
			StartPos:        V(80, 132),
			EndPos:          V(180, 132),
			Width:           160,
			Length:          160,
			Strength:        55.0,
			Duration:        0.2,
			FadeInDuration:  0.05,
			FadeOutDuration: 0.05,
		})
		for range 4 {
			sys.Update(w, 0.025)
		}
		displacedByGust := 0
		for i := range mesh.Vertices {
			if mesh.Vertices[i].DstX > mesh.BaseVertices[i].DstX+0.25 {
				displacedByGust++
			}
		}
		if displacedByGust == 0 {
			t.Fatal("expected AddWindGust to displace free cloth vertices")
		}

		// Advance past gust duration (0.1 + 0.15 = 0.25 >= 0.2) and test large dt spikes.
		cloth.Gravity = V(0, 240)
		cloth.SetWind(V(1, 0), 340)
		for range 10 {
			sys.Update(w, 5.0) // dt spike clamped internally to 1/30s
		}
		if len(sys.windGusts) != 0 {
			t.Fatalf("expected expired wind gusts to be removed, got %d", len(sys.windGusts))
		}
		for i, v := range mesh.Vertices {
			vals := []float64{
				float64(v.DstX), float64(v.DstY),
				float64(v.ColorR), float64(v.ColorG), float64(v.ColorB), float64(v.ColorA),
			}
			for _, val := range vals {
				if math.IsNaN(val) || math.IsInf(val, 0) {
					t.Fatalf("non-finite vertex value at index %d after dt spike: %+v", i, v)
				}
			}
			if math.Hypot(float64(v.DstX-mesh.BaseVertices[i].DstX), float64(v.DstY-mesh.BaseVertices[i].DstY)) > 512 {
				t.Fatalf("cloth vertex %d exploded after dt spike: %+v", i, v)
			}
		}
	})
}

func BenchmarkClothUpdate(b *testing.B) {
	w := teishoku.NewWorld(16)
	sys := NewClothSystem()
	sys.Initialize(w)

	spr := SpriteComponent{
		Width:   128,
		Height:  80,
		Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity: 1.0,
	}
	teishoku.NewBuilder4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w).NewEntityWith(
		NewTransformComponent(),
		spr,
		NewGridMeshComponent(16, 10),
		NewClothComponent(16, 10),
	)

	sys.Update(w, 1.0/60.0) // warm lazy solver buffers

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sys.Update(w, 1.0/60.0)
	}
}

func testWorldBiomes() []BiomeDef {
	return []BiomeDef{
		{
			ID:           1,
			Name:         "water",
			BaseTileID:   10,
			MaxElevation: 0.42,
			Color:        color.RGBA{R: 40, G: 90, B: 180, A: 255},
		},
		{
			ID:             2,
			Name:           "plains",
			BaseTileID:     20,
			VariantTileIDs: []uint16{21, 22},
			MinElevation:   0.42,
			MaxElevation:   0.70,
			Color:          color.RGBA{R: 90, G: 170, B: 70, A: 255},
		},
		{
			ID:             3,
			Name:           "highlands",
			BaseTileID:     30,
			VariantTileIDs: []uint16{31},
			MinElevation:   0.70,
			MaxElevation:   1.0,
			Color:          color.RGBA{R: 130, G: 120, B: 110, A: 255},
		},
	}
}

func TestWorldGeneratorDeterministicAndJSONCache(t *testing.T) {
	dir := t.TempDir()
	biomes := testWorldBiomes()

	gen1, err := NewWorldGenerator(WorldGenConfig{
		WorldName:  "test_realm",
		Seed:       12345,
		StorageDir: dir,
		ChunksX:    4,
		ChunksY:    4,
		ChunkSize:  8,
		TileSize:   16,
		Biomes:     biomes,
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator(gen1) failed: %v", err)
	}
	if gen1.LoadedFromCache() {
		t.Fatal("expected gen1.LoadedFromCache() == false on initial generation")
	}
	if gen1.CachePath() == "" {
		t.Fatal("expected non-empty gen1.CachePath()")
	}
	if _, err := os.Stat(gen1.CachePath()); err != nil {
		t.Fatalf("expected cache JSON file at %q: %v", gen1.CachePath(), err)
	}

	// Deterministic reproduction in a separate directory with identical seed.
	genDet, err := NewWorldGenerator(WorldGenConfig{
		WorldName:  "test_realm",
		Seed:       12345,
		StorageDir: t.TempDir(),
		ChunksX:    4,
		ChunksY:    4,
		ChunkSize:  8,
		TileSize:   16,
		Biomes:     biomes,
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator(genDet) failed: %v", err)
	}

	// Second generator pointing to same WorldName & StorageDir but different Seed loads cache.
	gen2, err := NewWorldGenerator(WorldGenConfig{
		WorldName:  "test_realm",
		Seed:       54321,
		StorageDir: dir,
		ChunksX:    4,
		ChunksY:    4,
		ChunkSize:  8,
		TileSize:   16,
		Biomes:     biomes,
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator(gen2) failed: %v", err)
	}
	if !gen2.LoadedFromCache() {
		t.Fatal("expected gen2.LoadedFromCache() == true when cache file exists")
	}
	if gen2.Seed() != 12345 {
		t.Fatalf("expected cached gen2.Seed() == 12345, got %d", gen2.Seed())
	}

	tw, th := gen1.WorldTileSize()
	for ty := range th {
		for tx := range tw {
			t1, b1 := gen1.TileAt(tx, ty), gen1.BiomeAt(tx, ty)
			t2, b2 := gen2.TileAt(tx, ty), gen2.BiomeAt(tx, ty)
			tD, bD := genDet.TileAt(tx, ty), genDet.BiomeAt(tx, ty)
			if t1 != t2 || b1 != b2 {
				t.Fatalf("cache mismatch at (%d,%d): gen1=(%d,%d) gen2=(%d,%d)", tx, ty, t1, b1, t2, b2)
			}
			if t1 != tD || b1 != bD {
				t.Fatalf("determinism mismatch at (%d,%d): gen1=(%d,%d) genDet=(%d,%d)", tx, ty, t1, b1, tD, bD)
			}
		}
	}

	// Verify SetTile + Save() persistence.
	if !gen1.SetTile(5, 7, 999) {
		t.Fatal("expected SetTile(5, 7, 999) to succeed")
	}
	if !gen1.ChunkAt(0, 0).Dirty {
		t.Fatal("expected SetTile to mark owning chunk Dirty")
	}
	if err := gen1.Save(); err != nil {
		t.Fatalf("gen1.Save() failed: %v", err)
	}

	gen3, err := NewWorldGenerator(WorldGenConfig{
		WorldName:  "test_realm",
		Seed:       111,
		StorageDir: dir,
		ChunksX:    4,
		ChunksY:    4,
		ChunkSize:  8,
		TileSize:   16,
		Biomes:     biomes,
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator(gen3) failed: %v", err)
	}
	if !gen3.LoadedFromCache() {
		t.Fatal("expected gen3.LoadedFromCache() == true")
	}
	if got := gen3.TileAt(5, 7); got != 999 {
		t.Fatalf("expected persisted tile edit 999 at (5,7), got %d", got)
	}

	// Verify Regenerate(99999) rebuilds chunks, clears LoadedFromCache, and updates cache.
	if err := gen1.Regenerate(99999); err != nil {
		t.Fatalf("gen1.Regenerate(99999) failed: %v", err)
	}
	if gen1.LoadedFromCache() {
		t.Fatal("expected LoadedFromCache() == false after Regenerate")
	}
	if gen1.Seed() != 99999 {
		t.Fatalf("expected Seed() == 99999 after Regenerate, got %d", gen1.Seed())
	}
	for cy := range 4 {
		for cx := range 4 {
			if !gen1.ChunkAt(cx, cy).Dirty {
				t.Fatalf("expected chunk (%d,%d) to be Dirty after Regenerate", cx, cy)
			}
		}
	}

	gen4, err := NewWorldGenerator(WorldGenConfig{
		WorldName:  "test_realm",
		Seed:       12345,
		StorageDir: dir,
		ChunksX:    4,
		ChunksY:    4,
		ChunkSize:  8,
		TileSize:   16,
		Biomes:     biomes,
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator(gen4) failed: %v", err)
	}
	if !gen4.LoadedFromCache() || gen4.Seed() != 99999 {
		t.Fatalf("expected regenerated cache with seed 99999, got cached=%v seed=%d", gen4.LoadedFromCache(), gen4.Seed())
	}
	for ty := range th {
		for tx := range tw {
			if gen4.TileAt(tx, ty) != gen1.TileAt(tx, ty) || gen4.BiomeAt(tx, ty) != gen1.BiomeAt(tx, ty) {
				t.Fatalf("regenerated cache mismatch at (%d,%d)", tx, ty)
			}
		}
	}
}

func TestWorldSystemChunkStreamingAndUnload(t *testing.T) {
	// 16x16 chunks, each 8x8 tiles of 16px = 128x128px per chunk (2048x2048px world).
	gen, err := NewWorldGenerator(WorldGenConfig{
		WorldName:          "stream_realm",
		Seed:               42,
		ChunksX:            16,
		ChunksY:            16,
		ChunkSize:          8,
		TileSize:           16,
		LoadMarginChunks:   1,
		UnloadMarginChunks: 2,
		MaxPooledImages:    64,
		Biomes:             testWorldBiomes(),
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator failed: %v", err)
	}

	w := teishoku.NewWorld(256)
	camSys := NewCameraSystem()
	camSys.Initialize(w)

	// Viewport 256x256 centered at (128, 128) -> ViewBounds [0..256, 0..256] -> visible chunks [0..1]x[0..1].
	// With LoadMarginChunks=1 -> load chunks [0..2]x[0..2] (9 chunks).
	camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(256, 256, WithCameraPosition(V(128, 128))),
	)
	baselineEntities := w.Len()

	loadedMap := make(map[[2]int]bool)
	unloadedCount := 0

	worldSys := NewWorldSystem(
		gen,
		WithWorldOnChunkLoad(func(world *teishoku.World, chunk *WorldChunk) []teishoku.Entity {
			loadedMap[[2]int{chunk.CX, chunk.CY}] = true
			b := teishoku.NewBuilder[TransformComponent](world)
			return []teishoku.Entity{
				b.NewEntityWith(TransformComponent{Position: Point{X: chunk.WorldX + 16, Y: chunk.WorldY + 16}}),
				b.NewEntityWith(TransformComponent{Position: Point{X: chunk.WorldX + 48, Y: chunk.WorldY + 48}}),
			}
		}),
		WithWorldOnChunkUnload(func(_ *teishoku.World, chunk *WorldChunk) {
			delete(loadedMap, [2]int{chunk.CX, chunk.CY})
			unloadedCount++
		}),
	)
	worldSys.Initialize(w)
	worldSys.Initialize(w) // idempotent guard
	defer worldSys.Dispose()

	camSys.Update(w, 1.0/60.0)
	worldSys.Update(w, 1.0/60.0)

	stats := worldSys.Stats()
	if stats.LoadedChunks != 9 {
		t.Fatalf("expected 9 loaded chunks in [0..2]x[0..2], got %d", stats.LoadedChunks)
	}
	if stats.StreamedEntities != 18 {
		t.Fatalf("expected 18 streamed entities, got %d", stats.StreamedEntities)
	}
	if w.Len() != baselineEntities+18 {
		t.Fatalf("expected world entity count %d, got %d", baselineEntities+18, w.Len())
	}
	if gen.ChunkAt(10, 10).Loaded || len(gen.ChunkAt(10, 10).StreamedEntities) != 0 {
		t.Fatal("expected distant chunk (10,10) not to be loaded")
	}

	// Move camera to (1536, 1536) -> ViewBounds [1408..1664] -> chunks [11..12] -> load [10..13]x[10..13] (16 chunks).
	// Unload margin = 2 -> [9..14], so all 9 old chunks [0..2]x[0..2] unload!
	cam := teishoku.GetComponent[CameraComponent](w, camEnt)
	cam.SetPosition(V(1536, 1536))
	camSys.Update(w, 1.0/60.0)
	worldSys.Update(w, 1.0/60.0)

	if unloadedCount != 9 {
		t.Fatalf("expected 9 distant chunks to unload, got %d", unloadedCount)
	}
	for cy := range 3 {
		for cx := range 3 {
			c := gen.ChunkAt(cx, cy)
			if c.Loaded || len(c.StreamedEntities) != 0 || c.image != nil {
				t.Fatalf("expected old chunk (%d,%d) fully unloaded, got loaded=%v ents=%d img=%v",
					cx, cy, c.Loaded, len(c.StreamedEntities), c.image)
			}
		}
	}
	stats = worldSys.Stats()
	if stats.LoadedChunks != 16 || stats.StreamedEntities != 32 {
		t.Fatalf("expected 16 loaded chunks and 32 streamed entities at new position, got loaded=%d ents=%d",
			stats.LoadedChunks, stats.StreamedEntities)
	}
	if w.Len() != baselineEntities+32 {
		t.Fatalf("expected old streamed entities despawned (world Len=%d, want %d)", w.Len(), baselineEntities+32)
	}

	// Move camera to far corner (1984, 1984) -> chunks [14..15] -> load [13..15]x[13..15] (9 chunks),
	// unloading chunks outside [12..17] (7 of the 16 old chunks unload while 4 stay loaded, 5 load -> net 2 images enter pool).
	cam.SetPosition(V(1984, 1984))
	camSys.Update(w, 1.0/60.0)
	worldSys.Update(w, 1.0/60.0)

	stats = worldSys.Stats()
	if stats.PooledImages == 0 {
		t.Fatalf("expected unloaded chunk images to be recycled into pool, got PooledImages=0")
	}

	// Dispose leaves zero active chunks, zero pooled images, and zero active images.
	worldSys.Dispose()
	stats = worldSys.Stats()
	if stats.LoadedChunks != 0 || stats.PooledImages != 0 || stats.StreamedEntities != 0 {
		t.Fatalf("expected all counts zero after Dispose, got %+v", stats)
	}
	for i := range gen.chunks {
		if gen.chunks[i].image != nil || gen.chunks[i].Loaded {
			t.Fatalf("expected chunk %d cleared after Dispose", i)
		}
	}
}

func BenchmarkWorldSystemUpdate(b *testing.B) {
	gen, err := NewWorldGenerator(WorldGenConfig{
		WorldName:          "bench_world",
		Seed:               99,
		ChunksX:            8,
		ChunksY:            8,
		ChunkSize:          8,
		TileSize:           16,
		LoadMarginChunks:   1,
		UnloadMarginChunks: 2,
		MaxPooledImages:    64,
		Biomes:             testWorldBiomes(),
	})
	if err != nil {
		b.Fatalf("NewWorldGenerator failed: %v", err)
	}

	w := teishoku.NewWorld(64)
	camSys := NewCameraSystem()
	camSys.Initialize(w)
	teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(800, 600, WithCameraPosition(V(512, 512))),
	)

	worldSys := NewWorldSystem(gen)
	worldSys.Initialize(w)
	defer worldSys.Dispose()

	camSys.Update(w, 1.0/60.0)
	worldSys.Update(w, 1.0/60.0) // warm steady-state loaded chunks & fallback image

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		worldSys.Update(w, 1.0/60.0)
	}
}

func TestSpriteAndOrderedSpriteFrustumCulling(t *testing.T) {
	w := teishoku.NewWorld(256)
	camSys := NewCameraSystem()
	camSys.Initialize(w)

	camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(800, 600, WithCameraPosition(V(400, 300))),
	)

	b := teishoku.NewBuilder3[TransformComponent, SpriteComponent, OrderableComponent](w)
	// 20 sprites inside ViewBounds [0..800, 0..600].
	for i := range 20 {
		tr := NewTransformComponent()
		tr.Position = Point{X: float64(60 + i*32), Y: float64(100 + (i%4)*80)}
		b.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewOrderableComponent(float64(i)))
	}
	// 180 sprites far outside ViewBounds (beyond default frustumMargin=96).
	for i := range 180 {
		tr := NewTransformComponent()
		tr.Position = Point{X: float64(5000 + (i%15)*64), Y: float64(5000 + (i/15)*64)}
		b.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewOrderableComponent(float64(20+i)))
	}

	spriteSys := NewSpriteSystem()
	orderedSys := NewOrderedSpriteSystem()
	spriteSys.Initialize(w)
	orderedSys.Initialize(w)

	camSys.Update(w, 1.0/60.0)
	spriteSys.Update(w, 1.0/60.0)
	orderedSys.Update(w, 1.0/60.0)

	if len(spriteSys.entries) != 20 {
		t.Fatalf("expected 20 visible entries in SpriteSystem, got %d", len(spriteSys.entries))
	}
	if len(orderedSys.entries) != 20 {
		t.Fatalf("expected 20 visible entries in OrderedSpriteSystem, got %d", len(orderedSys.entries))
	}

	// Pan camera far outside both clusters -> 0 visible sprites.
	cam := teishoku.GetComponent[CameraComponent](w, camEnt)
	cam.SetPosition(V(20000, 20000))
	camSys.Update(w, 1.0/60.0)
	spriteSys.Update(w, 1.0/60.0)
	orderedSys.Update(w, 1.0/60.0)

	if len(spriteSys.entries) != 0 || len(orderedSys.entries) != 0 {
		t.Fatalf("expected 0 visible entries far from sprites, got sprite=%d ordered=%d",
			len(spriteSys.entries), len(orderedSys.entries))
	}
}

func TestShadowRenderSystemPreCullAndBlurDownsample(t *testing.T) {
	t.Run("BlurDownsampleClampingAndTargets", func(t *testing.T) {
		low := NewShadowRenderSystem(WithShadowBlur(4.0), WithShadowBlurDownsample(0.1))
		if !approxEqual(low.BlurDownsample(), 0.25, 1e-9) {
			t.Fatalf("expected blur downsample clamped to 0.25, got %f", low.BlurDownsample())
		}

		high := NewShadowRenderSystem(WithShadowBlur(4.0), WithShadowBlurDownsample(2.0))
		if !approxEqual(high.BlurDownsample(), 1.0, 1e-9) {
			t.Fatalf("expected blur downsample clamped to 1.0, got %f", high.BlurDownsample())
		}

		sys := NewShadowRenderSystem(WithShadowBlur(4.0), WithShadowBlurDownsample(0.5))
		defer sys.Dispose()
		if !approxEqual(sys.BlurDownsample(), 0.5, 1e-9) {
			t.Fatalf("expected BlurDownsample=0.5, got %f", sys.BlurDownsample())
		}

		sys.ensureTargets(200, 100, true)
		if sys.blurTarget == nil || sys.blurTempTarget == nil {
			t.Fatal("expected blurTarget and blurTempTarget allocated at 0.5 downsample")
		}
		if b := sys.blurTarget.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
			t.Fatalf("expected downsampled blurTarget (100x50), got (%dx%d)", b.Dx(), b.Dy())
		}
		if b := sys.blurTempTarget.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
			t.Fatalf("expected downsampled blurTempTarget (100x50), got (%dx%d)", b.Dx(), b.Dy())
		}

		// Setting downsample >= 1.0 releases blurTempTarget cleanly.
		sys.SetBlurDownsample(1.0)
		if !approxEqual(sys.BlurDownsample(), 1.0, 1e-9) {
			t.Fatalf("expected BlurDownsample=1.0, got %f", sys.BlurDownsample())
		}
		if sys.blurTempTarget != nil {
			t.Fatal("expected SetBlurDownsample(1.0) to deallocate and clear blurTempTarget")
		}
	})

	t.Run("CoarsePreCullSkipsOffscreenCasters", func(t *testing.T) {
		w := teishoku.NewWorld(128)
		teishoku.AddResource(w.Resources(), NewTextureManager())

		camSys := NewCameraSystem()
		camSys.Initialize(w)
		teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
			*NewCameraComponent(800, 600, WithCameraPosition(V(400, 300))),
		)
		camSys.Update(w, 1.0/60.0)

		sb := teishoku.NewBuilder3[TransformComponent, SpriteComponent, ShadowComponent](w)
		// 10 visible contact shadows inside ViewBounds.
		for i := range 10 {
			tr := NewTransformComponent()
			tr.Position = Point{X: float64(100 + i*50), Y: 300}
			sb.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewContactShadow(0, 0))
		}
		// 90 far-offscreen contact shadows skipped by coarse culling.
		for i := range 90 {
			tr := NewTransformComponent()
			tr.Position = Point{X: float64(50000 + i*100), Y: 50000}
			sb.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewContactShadow(0, 0))
		}

		shadowSys := NewShadowRenderSystem(WithShadowComposited(false))
		shadowSys.Initialize(w)

		screen := ebiten.NewImage(800, 600)
		defer screen.Deallocate()
		rdr := NewBatchRenderer()
		rdr.Begin(screen)

		shadowSys.Draw(w, rdr)
		if len(rdr.vertices) != 10*4 || len(rdr.indices) != 10*6 {
			t.Fatalf("expected only 10 visible contact shadows staged (40 verts, 60 inds), got verts=%d inds=%d",
				len(rdr.vertices), len(rdr.indices))
		}
	})
}

func BenchmarkOrderedSpriteCull10K(b *testing.B) {
	w := teishoku.NewWorld(10240)
	camSys := NewCameraSystem()
	camSys.Initialize(w)
	teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(800, 600, WithCameraPosition(V(400, 300))),
	)

	eb := teishoku.NewBuilder3[TransformComponent, SpriteComponent, OrderableComponent](w)
	for i := range 10000 {
		tr := NewTransformComponent()
		tr.Position = Point{
			X: float64((i % 100) * 160),
			Y: float64((i / 100) * 160),
		}
		eb.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewOrderableComponent(0))
	}

	orderedSys := NewOrderedSpriteSystem()
	orderedSys.Initialize(w)

	camSys.Update(w, 1.0/60.0)
	orderedSys.Update(w, 1.0/60.0) // warm reusable entries buffer

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		orderedSys.Update(w, 1.0/60.0)
	}
}

func BenchmarkShadowPreCull10K(b *testing.B) {
	w := teishoku.NewWorld(10240)
	teishoku.AddResource(w.Resources(), NewTextureManager())

	camSys := NewCameraSystem()
	camSys.Initialize(w)
	teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(800, 600, WithCameraPosition(V(400, 300))),
	)

	sb := teishoku.NewBuilder3[TransformComponent, SpriteComponent, ShadowComponent](w)
	for i := range 10000 {
		tr := NewTransformComponent()
		tr.Position = Point{
			X: float64((i % 100) * 160),
			Y: float64((i / 100) * 160),
		}
		sb.NewEntityWith(tr, NewSpriteComponent(0, 32, 32), NewContactShadow(0, 0))
	}

	shadowSys := NewShadowRenderSystem(WithShadowComposited(false))
	shadowSys.Initialize(w)

	screen := ebiten.NewImage(800, 600)
	defer screen.Deallocate()
	rdr := NewBatchRenderer()

	camSys.Update(w, 1.0/60.0)
	rdr.Begin(screen)
	shadowSys.Draw(w, rdr) // warm reusable buffers and contact shadow texture

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rdr.Begin(screen)
		shadowSys.Draw(w, rdr)
	}
}
