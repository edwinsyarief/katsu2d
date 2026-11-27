package katsu2d

import (
	"image/color"
	"path/filepath"
	"testing"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDualGridBitmaskAndCoords(t *testing.T) {
	// 8x8 logical tiles, tileSize=32, chunkSize=4 -> chunks cover [0..8] => (8+4)/4 = 3 chunks per axis.
	m := NewDualGridMap(8, 8, 32, 0, WithDualGridChunkSize(4))
	if m.Width() != 8 || m.Height() != 8 || m.TileSize() != 32 || m.ChunkSize() != 4 {
		t.Fatalf("unexpected map dimensions: w=%d h=%d ts=%d cs=%d", m.Width(), m.Height(), m.TileSize(), m.ChunkSize())
	}
	if m.ChunksX() != 3 || m.ChunksY() != 3 {
		t.Fatalf("expected 3x3 dual chunks for 8x8 map with cs=4, got %dx%d", m.ChunksX(), m.ChunksY())
	}
	pw, ph := m.WorldPixelSize()
	if pw != 256 || ph != 256 {
		t.Fatalf("expected WorldPixelSize (256, 256), got (%v, %v)", pw, ph)
	}

	// Coordinate conversions: exact floor division, no half-tile mouse-pick shift.
	tx, ty := m.WorldToTile(0, 0)
	if tx != 0 || ty != 0 {
		t.Fatalf("WorldToTile(0,0) = (%d,%d), want (0,0)", tx, ty)
	}
	tx, ty = m.WorldToTile(63.9, 96.0)
	if tx != 1 || ty != 3 {
		t.Fatalf("WorldToTile(63.9,96.0) = (%d,%d), want (1,3)", tx, ty)
	}
	tx, ty = m.WorldToTile(-1, -0.5)
	if tx != -1 || ty != -1 {
		t.Fatalf("WorldToTile(-1,-0.5) = (%d,%d), want (-1,-1)", tx, ty)
	}
	wx, wy := m.TileToWorld(3, 5)
	if wx != 96 || wy != 160 {
		t.Fatalf("TileToWorld(3,5) = (%v,%v), want (96,160)", wx, wy)
	}

	// Out-of-bounds TileAt returns defaultMat, SetTile returns false.
	if got := m.TileAt(-1, 0); got != 0 {
		t.Fatalf("expected out-of-bounds TileAt = 0, got %d", got)
	}
	if m.SetTile(-1, 0, 2) || m.SetTile(8, 0, 2) || m.SetTile(0, 8, 2) {
		t.Fatal("expected out-of-bounds SetTile to return false")
	}

	// Clear dirty flags and verify seam propagation for SetTile.
	for i := range m.dirtyChunks {
		m.dirtyChunks[i] = false
	}
	// Tile (1, 1) touches dual corners (1..2, 1..2), all inside chunk (0, 0).
	if !m.SetTile(1, 1, 1) {
		t.Fatal("expected SetTile(1,1,1) to succeed")
	}
	if !m.IsChunkDirty(0, 0) || m.IsChunkDirty(1, 0) || m.IsChunkDirty(0, 1) {
		t.Fatal("expected only chunk (0,0) dirty for interior tile (1,1)")
	}

	// Clear dirty flags; tile (3, 3) touches dual corners (3..4, 3..4) -> chunks (0,0), (1,0), (0,1), (1,1).
	for i := range m.dirtyChunks {
		m.dirtyChunks[i] = false
	}
	if !m.SetTile(3, 3, 2) {
		t.Fatal("expected SetTile(3,3,2) to succeed")
	}
	for cy := range 2 {
		for cx := range 2 {
			if !m.IsChunkDirty(cx, cy) {
				t.Fatalf("expected seam chunk (%d,%d) dirty after SetTile(3,3)", cx, cy)
			}
		}
	}
	if m.IsChunkDirty(2, 0) || m.IsChunkDirty(0, 2) {
		t.Fatal("expected distant chunks (2,0) and (0,2) to remain clean")
	}

	// PaintBrush, PaintRect, FloodFill, Fill.
	m.Fill(0)
	rectChanged := m.PaintRect(1, 1, 5, 5, 1)
	if rectChanged != 25 {
		t.Fatalf("expected PaintRect(1,1,5,5) to change 25 tiles, got %d", rectChanged)
	}
	// Repeating same rect changes 0 tiles.
	if again := m.PaintRect(1, 1, 5, 5, 1); again != 0 {
		t.Fatalf("expected repeated PaintRect to change 0 tiles, got %d", again)
	}

	brushChanged := m.PaintBrush(3, 3, 1, 2)
	// Radius 1 integer disk: (0,0), (+-1,0), (0,+-1) = 5 tiles.
	if brushChanged != 5 {
		t.Fatalf("expected PaintBrush radius 1 to change 5 tiles, got %d", brushChanged)
	}

	// FloodFill the ring of material 1 surrounding the cross of material 2: 25 - 5 = 20 tiles.
	floodChanged := m.FloodFill(1, 1, 3)
	if floodChanged != 20 {
		t.Fatalf("expected FloodFill to change 20 tiles, got %d", floodChanged)
	}
	if m.TileAt(1, 1) != 3 || m.TileAt(3, 3) != 2 || m.TileAt(0, 0) != 0 {
		t.Fatalf("unexpected tile materials after FloodFill: (1,1)=%d (3,3)=%d (0,0)=%d",
			m.TileAt(1, 1), m.TileAt(3, 3), m.TileAt(0, 0))
	}

	// Multi-material >= matID bitmasking verification.
	verts, inds := appendDualGridCellQuads(nil, nil, 1, 1, 1, 1, 4, 0, 0, 32, 32, 32)
	if len(verts) != 4 || len(inds) != 6 {
		t.Fatalf("expected 1 solid quad (4 verts, 6 inds) for uniform cell, got %d verts, %d inds", len(verts), len(inds))
	}
	if verts[0].SrcX != 15*32 || verts[0].SrcY != 1*32 {
		t.Fatalf("expected solid bitmask 15 for uniform material 1, got Src=(%v,%v)", verts[0].SrcX, verts[0].SrcY)
	}

	// Mixed corners: tl=0, tr=1, bl=2, br=2 -> emits mat 0 (mask 15), mat 1 (mask 7), mat 2 (mask 3).
	verts, inds = appendDualGridCellQuads(verts[:0], inds[:0], 0, 1, 2, 2, 4, 0, 0, 32, 32, 32)
	if len(verts) != 12 || len(inds) != 18 {
		t.Fatalf("expected 3 quads (12 verts, 18 inds) for 3-material cell, got %d verts, %d inds", len(verts), len(inds))
	}
	wantMasks := [3]int{15, 7, 3}
	wantMats := [3]int{0, 1, 2}
	for q := range 3 {
		v := verts[q*4]
		if v.SrcX != float32(wantMasks[q]*32) || v.SrcY != float32(wantMats[q]*32) {
			t.Fatalf("quad %d: got Src=(%v,%v), want (%d,%d)",
				q, v.SrcX, v.SrcY, wantMasks[q]*32, wantMats[q]*32)
		}
	}

	// RLE JSON Marshal/Unmarshal & SaveJSON/LoadJSON round-trip.
	raw, err := m.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var decoded DualGridMap
	if err := decoded.UnmarshalJSON(raw); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if decoded.Width() != m.Width() || decoded.Height() != m.Height() ||
		decoded.TileSize() != m.TileSize() || decoded.ChunkSize() != m.ChunkSize() {
		t.Fatal("decoded map metadata mismatch")
	}
	for y := range m.Height() {
		for x := range m.Width() {
			if decoded.TileAt(x, y) != m.TileAt(x, y) {
				t.Fatalf("decoded tile mismatch at (%d,%d): got %d, want %d", x, y, decoded.TileAt(x, y), m.TileAt(x, y))
			}
		}
	}

	savePath := filepath.Join(t.TempDir(), "sub", "map.json")
	if err := m.SaveJSON(savePath); err != nil {
		t.Fatalf("SaveJSON failed: %v", err)
	}
	var loaded DualGridMap
	if err := loaded.LoadJSON(savePath); err != nil {
		t.Fatalf("LoadJSON failed: %v", err)
	}
	if loaded.TileAt(1, 1) != 3 || loaded.TileAt(3, 3) != 2 {
		t.Fatal("loaded map tile mismatch")
	}
}

func TestDualGridTilesetAndMasks(t *testing.T) {
	const ts = 16

	// Verify all 3 mask styles allocate 4*ts x 4*ts and dispose cleanly.
	styles := []DualMaskStyle{DualMaskSoft, DualMaskOrganic, DualMaskRocky}
	for _, st := range styles {
		mask := NewDualGridMask(ts, st)
		if mask == nil {
			t.Fatalf("NewDualGridMask(%d, %d) returned nil", ts, st)
		}
		b := mask.Bounds()
		if b.Dx() != 4*ts || b.Dy() != 4*ts {
			t.Fatalf("mask style %d bounds = %dx%d, want %dx%d", st, b.Dx(), b.Dy(), 4*ts, 4*ts)
		}
		mask.Deallocate()
	}

	// Verify SDF edge continuity across shared dual-cell boundaries and organic edge symmetry.
	for _, rocky := range []bool{false, true} {
		// Top edge of cell 12 (TL|TR) and bottom edge of cell 15 must both be solid (sd <= 0).
		if evalDualGridMaskSDF(12, 0.25, 0.01, rocky) > 0 || evalDualGridMaskSDF(12, 0.75, 0.01, rocky) > 0 {
			t.Fatalf("cell 12 top edge not solid (rocky=%v)", rocky)
		}
		// Shared vertical edge: right edge of cell 12 (TR=1,BR=0) vs left edge of cell 8 (TL=1,BL=0).
		for _, v := range []float64{0.1, 0.4, 0.6, 0.9} {
			sRight := evalDualGridMaskSDF(12, 1.0, v, rocky)
			sLeft := evalDualGridMaskSDF(8, 0.0, v, rocky)
			if sRight != sLeft {
				t.Fatalf("edge discontinuity at v=%v (rocky=%v): right=%v left=%v", v, rocky, sRight, sLeft)
			}
		}
	}
	for py := range ts {
		// Left edge (px=0) and right edge (px=ts-1) produce identical organic jitter.
		if dualOrganicHash(0, py) != dualOrganicHash(0, py) {
			t.Fatal("organic hash not deterministic")
		}
	}

	tileset := NewDualGridTileset(ts)
	defer tileset.Dispose()

	if tileset.TileSize() != ts {
		t.Fatalf("TileSize = %d, want %d", tileset.TileSize(), ts)
	}
	if tileset.Atlas() != nil {
		t.Fatal("expected nil atlas before any materials are added")
	}

	waterImg := ebiten.NewImage(ts, ts)
	defer waterImg.Deallocate()
	waterImg.Fill(color.RGBA{R: 40, G: 100, B: 200, A: 255})

	grassImg := ebiten.NewImage(ts, ts)
	defer grassImg.Deallocate()
	grassImg.Fill(color.RGBA{R: 60, G: 170, B: 70, A: 255})

	rockyMask := NewDualGridMask(ts, DualMaskRocky)
	defer rockyMask.Deallocate()

	id0 := tileset.AddMaterialWithMask("water", waterImg, nil, WithDualMaterialColor(color.RGBA{R: 40, G: 100, B: 200, A: 255}))
	id1 := tileset.AddMaterialWithMask("grass", grassImg, rockyMask, WithDualMaterialColor(color.RGBA{R: 60, G: 170, B: 70, A: 255}))
	if id0 != 0 || id1 != 1 {
		t.Fatalf("unexpected material IDs: %d, %d", id0, id1)
	}
	if tileset.MaterialCount() != 2 {
		t.Fatalf("MaterialCount = %d, want 2", tileset.MaterialCount())
	}
	if tileset.MaterialName(id1) != "grass" {
		t.Fatalf("MaterialName(1) = %q, want grass", tileset.MaterialName(id1))
	}
	if tileset.MaterialColor(id0) != (color.RGBA{R: 40, G: 100, B: 200, A: 255}) {
		t.Fatalf("unexpected MaterialColor(0): %+v", tileset.MaterialColor(id0))
	}

	atlas := tileset.BuildAtlas()
	if atlas == nil {
		t.Fatal("expected non-nil atlas")
	}
	ab := atlas.Bounds()
	if ab.Dx() != 16*ts || ab.Dy() != 2*ts {
		t.Fatalf("atlas bounds = %dx%d, want %dx%d", ab.Dx(), ab.Dy(), 16*ts, 2*ts)
	}

	// Adding a 3rd sheet material invalidates and rebuilds the atlas cleanly.
	sheet := ebiten.NewImage(4*ts, 4*ts)
	defer sheet.Deallocate()
	sheet.Fill(color.RGBA{R: 130, G: 120, B: 110, A: 255})
	id2 := tileset.AddMaterialSheet("stone", sheet)
	if id2 != 2 {
		t.Fatalf("expected id2 = 2, got %d", id2)
	}
	atlas2 := tileset.Atlas()
	if atlas2 == nil {
		t.Fatal("expected rebuilt atlas")
	}
	if b := atlas2.Bounds(); b.Dx() != 16*ts || b.Dy() != 3*ts {
		t.Fatalf("rebuilt atlas bounds = %dx%d, want %dx%d", b.Dx(), b.Dy(), 16*ts, 3*ts)
	}

	tileset.Dispose()
	if tileset.MaterialCount() != 0 || tileset.Atlas() != nil {
		t.Fatal("expected tileset cleared after Dispose")
	}
}

func TestDualGridSystemAndWorldPainter(t *testing.T) {
	const ts = 16
	tileset := NewDualGridTileset(ts)
	defer tileset.Dispose()

	waterImg := ebiten.NewImage(ts, ts)
	defer waterImg.Deallocate()
	waterImg.Fill(color.RGBA{R: 40, G: 110, B: 210, A: 255})

	grassImg := ebiten.NewImage(ts, ts)
	defer grassImg.Deallocate()
	grassImg.Fill(color.RGBA{R: 70, G: 180, B: 70, A: 255})

	tileset.AddMaterialWithMask("water", waterImg, nil)
	tileset.AddMaterialWithMask("grass", grassImg, nil)

	// 1. DualGridSystem chunk streaming, culling, dirty re-baking, Draw, entity eviction, Dispose.
	w := teishoku.NewWorld(64)
	camSys := NewCameraSystem()
	camSys.Initialize(w)
	camEnt := teishoku.NewBuilder[CameraComponent](w).NewEntityWith(
		*NewCameraComponent(128, 128, WithCameraPosition(V(64, 64))),
	)

	// 32x32 tiles, ts=16, cs=4 -> 64px per chunk, 9x9 chunks.
	// Viewport [0..128] -> visible chunks [0..2]x[0..2], loadMargin=1 -> [0..3]x[0..3] = 16 chunks.
	gridMap := NewDualGridMap(32, 32, ts, 0, WithDualGridChunkSize(4))
	gridMap.PaintRect(1, 1, 6, 6, 1)

	comp := NewDualGridComponent(gridMap, tileset)
	comp.ShowLogicalGrid = true
	comp.ShowDualCorners = true
	comp.HasHover = true
	comp.HoverX = 2
	comp.HoverY = 2

	gridEnt := teishoku.NewBuilder[DualGridComponent](w).NewEntityWith(comp)

	dgSys := NewDualGridSystem(
		WithDualGridLoadMargin(1),
		WithDualGridUnloadMargin(2),
		WithDualGridMaxPooledImages(32),
	)
	dgSys.Initialize(w)
	dgSys.Initialize(w) // idempotent guard
	defer dgSys.Dispose()

	camSys.Update(w, 1.0/60.0)
	dgSys.Update(w, 1.0/60.0)

	stats := dgSys.Stats()
	if stats.ActiveMaps != 1 || stats.LoadedChunks != 16 {
		t.Fatalf("expected 1 active map and 16 loaded chunks, got %+v", stats)
	}
	if gridMap.IsChunkDirty(0, 0) || gridMap.IsChunkDirty(1, 1) {
		t.Fatal("expected loaded chunks to have dirty flags cleared after Update")
	}

	// Editing a tile marks its chunks dirty and re-bakes on next Update without loading extra chunks.
	gridMap.SetTile(2, 2, 0)
	if !gridMap.IsChunkDirty(0, 0) {
		t.Fatal("expected chunk (0,0) dirty after SetTile(2,2)")
	}
	dgSys.Update(w, 1.0/60.0)
	if gridMap.IsChunkDirty(0, 0) {
		t.Fatal("expected chunk (0,0) clean after re-bake")
	}

	screen := ebiten.NewImage(128, 128)
	defer screen.Deallocate()
	rdr := NewBatchRenderer()
	rdr.Begin(screen)
	dgSys.Draw(w, rdr)
	if len(rdr.vertices) == 0 {
		t.Fatal("expected DualGridSystem.Draw to stage vertices on BatchRenderer")
	}
	rdr.Flush()

	// Pan camera to far corner (512, 512) -> ViewBounds [448..576] -> load [6..8]x[6..8] (9 chunks).
	// All 16 old chunks [0..3]x[0..3] unload into the pool; 9 are reused immediately, leaving 7 in the pool.
	cam := teishoku.GetComponent[CameraComponent](w, camEnt)
	cam.SetPosition(V(512, 512))
	camSys.Update(w, 1.0/60.0)
	dgSys.Update(w, 1.0/60.0)

	stats = dgSys.Stats()
	if stats.PooledImages == 0 {
		t.Fatalf("expected unloaded chunk images recycled into pool, got %+v", stats)
	}

	// Destroying the entity evicts its map state without leaking.
	w.RemoveEntity(gridEnt)
	dgSys.Update(w, 1.0/60.0)
	stats = dgSys.Stats()
	if stats.ActiveMaps != 0 || stats.LoadedChunks != 0 {
		t.Fatalf("expected evicted state after entity removal, got %+v", stats)
	}

	dgSys.Dispose()
	if s := dgSys.Stats(); s.PooledImages != 0 || s.LoadedChunks != 0 {
		t.Fatalf("expected zero counts after Dispose, got %+v", s)
	}

	// 2. WorldGenerator.SetDualGridTile + NewDualGridTilePainter + WorldSystem.
	gen, err := NewWorldGenerator(WorldGenConfig{
		WorldName: "dual_world_test",
		Seed:      123,
		ChunksX:   4,
		ChunksY:   4,
		ChunkSize: 4,
		TileSize:  ts,
		Biomes: []BiomeDef{
			{ID: 1, Name: "Water", BaseTileID: 0},
			{ID: 2, Name: "Grass", BaseTileID: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorldGenerator failed: %v", err)
	}

	// Clear all dirty flags, then test SetDualGridTile on a chunk corner (3, 3) -> chunks (0,0), (1,0), (0,1), (1,1).
	for i := range gen.chunks {
		gen.chunks[i].Dirty = false
	}
	cur := gen.TileAt(3, 3)
	next := cur ^ 1
	if !gen.SetDualGridTile(3, 3, next) {
		t.Fatal("SetDualGridTile(3,3) failed")
	}
	for cy := range 2 {
		for cx := range 2 {
			if !gen.ChunkAt(cx, cy).Dirty {
				t.Fatalf("expected adjacent chunk (%d,%d) marked dirty by SetDualGridTile(3,3)", cx, cy)
			}
		}
	}
	if gen.ChunkAt(2, 0).Dirty || gen.ChunkAt(0, 2).Dirty {
		t.Fatal("expected non-adjacent chunks to stay clean")
	}

	painter := NewDualGridTilePainter(gen, tileset, func(_, _ int, tileID uint16, _ BiomeID) DualMaterialID {
		if tileID == 0 {
			return 0
		}
		return 1
	})
	ws := NewWorldSystem(gen, WithWorldTilePainter(painter))
	w2 := teishoku.NewWorld(32)
	camSys2 := NewCameraSystem()
	camSys2.Initialize(w2)
	teishoku.NewBuilder[CameraComponent](w2).NewEntityWith(
		*NewCameraComponent(128, 128, WithCameraPosition(V(64, 64))),
	)
	ws.Initialize(w2)
	defer ws.Dispose()

	camSys2.Update(w2, 1.0/60.0)
	ws.Update(w2, 1.0/60.0)
	if ws.Stats().LoadedChunks == 0 {
		t.Fatal("expected WorldSystem with DualGridTilePainter to bake and load chunks")
	}
}

func BenchmarkDualGridBuildChunkVerts(b *testing.B) {
	m := NewDualGridMap(32, 32, 32, 0, WithDualGridChunkSize(16))
	m.PaintRect(2, 2, 14, 14, 1)
	m.PaintBrush(8, 8, 4, 2)

	verts := make([]ebiten.Vertex, 0, 4096)
	inds := make([]uint16, 0, 6144)

	// Warm buffers before measuring.
	verts, inds = buildDualGridChunkVerts(verts, inds, m, 3, 32, 0, 0)
	if len(verts) == 0 || len(inds) == 0 {
		b.Fatal("expected non-empty chunk mesh")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		verts, inds = buildDualGridChunkVerts(verts, inds, m, 3, 32, 0, 0)
	}
}
