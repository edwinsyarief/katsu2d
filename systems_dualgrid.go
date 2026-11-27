package katsu2d

import (
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	defaultDualGridLoadMargin   = 1
	defaultDualGridUnloadMargin = 2
	defaultDualGridMaxPool      = 64
	dualGridMaxBatchVerts       = maxVertices
)

// DualGridComponent attaches a DualGridMap and DualGridTileset to an entity
// for chunk-pooled dual-grid rendering and optional editor overlays.
type DualGridComponent struct {
	Map             *DualGridMap
	Tileset         *DualGridTileset
	Visible         bool
	ShowLogicalGrid bool
	ShowDualCorners bool
	HasHover        bool
	HoverX          int
	HoverY          int
}

// NewDualGridComponent creates a visible DualGridComponent for gridMap and tileset.
func NewDualGridComponent(gridMap *DualGridMap, tileset *DualGridTileset) DualGridComponent {
	return DualGridComponent{
		Map:     gridMap,
		Tileset: tileset,
		Visible: true,
	}
}

// DualGridStats reports live chunk and image pool metrics for DualGridSystem.
type DualGridStats struct {
	ActiveMaps   int
	LoadedChunks int
	PooledImages int
}

// DualGridSystemOption configures a DualGridSystem.
type DualGridSystemOption func(*DualGridSystem)

// WithDualGridMaxPooledImages sets the maximum number of offscreen chunk images
// retained in DualGridSystem's reuse pool (default 64).
func WithDualGridMaxPooledImages(n int) DualGridSystemOption {
	return func(s *DualGridSystem) {
		if n >= 0 {
			s.maxPooledImages = n
		}
	}
}

// WithDualGridLoadMargin sets the chunk radius around ActiveCamera.ViewBounds
// that DualGridSystem streams in and bakes (default 1).
func WithDualGridLoadMargin(chunks int) DualGridSystemOption {
	return func(s *DualGridSystem) {
		if chunks >= 0 {
			s.loadMargin = chunks
		}
	}
}

// WithDualGridUnloadMargin sets the chunk radius beyond which loaded chunks are
// recycled back into the image pool (default 2).
func WithDualGridUnloadMargin(chunks int) DualGridSystemOption {
	return func(s *DualGridSystem) {
		if chunks >= 0 {
			s.unloadMargin = chunks
		}
	}
}

type dualGridEntityState struct {
	gridMap       *DualGridMap
	tileset       *DualGridTileset
	atlasVersion  uint64
	width         int
	height        int
	tileSize      int
	chunkSize     int
	chunksX       int
	chunksY       int
	chunkImages   []*ebiten.Image
	loadedFlags   []bool
	lastSeenFrame uint64
}

// DualGridSystem streams, bakes, pools, and renders DualGridComponent entities
// with camera frustum culling and zero steady-state allocations.
type DualGridSystem struct {
	filter          *teishoku.Filter[DualGridComponent]
	activeCamHandle teishoku.ResourceHandle[ActiveCamera]
	initialized     bool

	maxPooledImages int
	loadMargin      int
	unloadMargin    int

	imagePool    []*ebiten.Image
	states       map[teishoku.Entity]*dualGridEntityState
	frameID      uint64
	verts        []ebiten.Vertex
	inds         []uint16
	loadedChunks int
}

// NewDualGridSystem creates a DualGridSystem configured with opts.
func NewDualGridSystem(opts ...DualGridSystemOption) *DualGridSystem {
	s := &DualGridSystem{
		maxPooledImages: defaultDualGridMaxPool,
		loadMargin:      defaultDualGridLoadMargin,
		unloadMargin:    defaultDualGridUnloadMargin,
		states:          make(map[teishoku.Entity]*dualGridEntityState, 4),
		verts:           make([]ebiten.Vertex, 0, 4096),
		inds:            make([]uint16, 0, 6144),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.unloadMargin < s.loadMargin {
		s.unloadMargin = s.loadMargin
	}
	return s
}

// Initialize caches the world's ECS filter and ActiveCamera resource handle once.
func (s *DualGridSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.filter = teishoku.NewFilter[DualGridComponent](w)
	s.activeCamHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	s.initialized = true
}

// Update streams dual-grid chunks in and out around ActiveCamera.ViewBounds,
// re-bakes dirty visible chunks, and recycles distant chunk images into the pool.
func (s *DualGridSystem) Update(w *teishoku.World, dt float64) {
	if w == nil {
		return
	}
	if !s.initialized {
		s.Initialize(w)
	}
	if !s.activeCamHandle.Has() {
		s.activeCamHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	}

	s.frameID++
	viewBounds := s.currentViewBounds()
	totalLoaded := 0

	chunks := s.filter.Chunks()
	for chunks.Next() {
		entities := chunks.Entities()
		comps := chunks.Get()
		for i := range entities {
			ent := entities[i]
			comp := &comps[i]
			if comp.Map == nil || comp.Tileset == nil {
				if state, ok := s.states[ent]; ok {
					s.releaseStateImages(state)
					delete(s.states, ent)
				}
				continue
			}

			state := s.ensureState(ent, comp.Map, comp.Tileset)
			state.lastSeenFrame = s.frameID

			m := comp.Map
			ts := comp.Tileset
			atlas := ts.Atlas()
			if atlas == nil {
				continue
			}

			tilePx := m.tileSize
			cs := m.chunkSize
			chunkPx := cs * tilePx
			halfTS := float64(tilePx) * 0.5
			chunkPixelSpan := float64(chunkPx)

			shifted := NewRectangle(
				viewBounds.Min.X+halfTS,
				viewBounds.Min.Y+halfTS,
				viewBounds.Max.X+halfTS,
				viewBounds.Max.Y+halfTS,
			)
			rawMinCX, rawMinCY, rawMaxCX, rawMaxCY, hasVis := visibleChunkRange(shifted, chunkPixelSpan)

			unloadMinCX := rawMinCX - s.unloadMargin
			unloadMinCY := rawMinCY - s.unloadMargin
			unloadMaxCX := rawMaxCX + s.unloadMargin
			unloadMaxCY := rawMaxCY + s.unloadMargin

			// Unload distant chunks and re-bake dirty loaded chunks.
			for cy := range m.chunksY {
				rowOff := cy * m.chunksX
				for cx := range m.chunksX {
					idx := rowOff + cx
					if !state.loadedFlags[idx] {
						continue
					}
					if !hasVis || cx < unloadMinCX || cx > unloadMaxCX || cy < unloadMinCY || cy > unloadMaxCY {
						s.recycleChunkImage(state.chunkImages[idx])
						state.chunkImages[idx] = nil
						state.loadedFlags[idx] = false
						continue
					}
					if m.dirtyChunks[idx] {
						if state.chunkImages[idx] == nil {
							state.chunkImages[idx] = s.acquireChunkImage(chunkPx)
						}
						s.bakeChunk(state.chunkImages[idx], m, ts, atlas, cx, cy)
						m.dirtyChunks[idx] = false
					}
					totalLoaded++
				}
			}

			// Load and bake any unloaded chunks inside the load margin.
			if hasVis {
				loadMinCX := max(0, rawMinCX-s.loadMargin)
				loadMinCY := max(0, rawMinCY-s.loadMargin)
				loadMaxCX := min(m.chunksX-1, rawMaxCX+s.loadMargin)
				loadMaxCY := min(m.chunksY-1, rawMaxCY+s.loadMargin)

				for cy := loadMinCY; cy <= loadMaxCY; cy++ {
					rowOff := cy * m.chunksX
					for cx := loadMinCX; cx <= loadMaxCX; cx++ {
						idx := rowOff + cx
						if state.loadedFlags[idx] {
							continue
						}
						img := s.acquireChunkImage(chunkPx)
						s.bakeChunk(img, m, ts, atlas, cx, cy)
						state.chunkImages[idx] = img
						state.loadedFlags[idx] = true
						m.dirtyChunks[idx] = false
						totalLoaded++
					}
				}
			}
		}
	}

	// Evict states belonging to destroyed entities or entities that lost DualGridComponent.
	for ent, state := range s.states {
		if state.lastSeenFrame != s.frameID {
			s.releaseStateImages(state)
			delete(s.states, ent)
		}
	}

	s.loadedChunks = totalLoaded
}

// Draw submits visible dual-grid chunks overlapping ActiveCamera.ViewBounds and
// any enabled editor overlays (logical grid, dual-corner dots, hover outline) to rdr.
func (s *DualGridSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if w == nil || rdr == nil {
		return
	}
	if !s.initialized {
		s.Initialize(w)
	}

	viewBounds := s.currentViewBounds()
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	chunks := s.filter.Chunks()
	for chunks.Next() {
		entities := chunks.Entities()
		comps := chunks.Get()
		for i := range entities {
			comp := &comps[i]
			if !comp.Visible || comp.Map == nil || comp.Tileset == nil {
				continue
			}
			state := s.states[entities[i]]
			if state == nil {
				continue
			}

			m := comp.Map
			tilePx := m.tileSize
			cs := m.chunkSize
			halfTS := float64(tilePx) * 0.5
			chunkPixelSpan := float64(cs * tilePx)

			shifted := NewRectangle(
				viewBounds.Min.X+halfTS,
				viewBounds.Min.Y+halfTS,
				viewBounds.Max.X+halfTS,
				viewBounds.Max.Y+halfTS,
			)
			rawMinCX, rawMinCY, rawMaxCX, rawMaxCY, hasVis := visibleChunkRange(shifted, chunkPixelSpan)
			if !hasVis {
				continue
			}

			visMinCX := max(0, rawMinCX)
			visMinCY := max(0, rawMinCY)
			visMaxCX := min(m.chunksX-1, rawMaxCX)
			visMaxCY := min(m.chunksY-1, rawMaxCY)

			if visMinCX <= visMaxCX && visMinCY <= visMaxCY {
				for cy := visMinCY; cy <= visMaxCY; cy++ {
					dyStart := cy * cs
					dyEnd := min(dyStart+cs, m.height+1)
					drawH := float64((dyEnd - dyStart) * tilePx)
					worldY := float64(dyStart*tilePx) - halfTS
					rowOff := cy * m.chunksX

					for cx := visMinCX; cx <= visMaxCX; cx++ {
						idx := rowOff + cx
						if !state.loadedFlags[idx] || state.chunkImages[idx] == nil {
							continue
						}
						dxStart := cx * cs
						dxEnd := min(dxStart+cs, m.width+1)
						drawW := float64((dxEnd - dxStart) * tilePx)
						worldX := float64(dxStart*tilePx) - halfTS

						rdr.AddQuad(
							V(worldX, worldY),
							ZeroVector,
							ZeroVector,
							V(1, 1),
							0,
							state.chunkImages[idx],
							white,
							0, 0, float32(drawW), float32(drawH),
							drawW, drawH,
						)
					}
				}
			}

			if comp.ShowLogicalGrid || comp.ShowDualCorners || comp.HasHover {
				s.drawOverlays(comp, viewBounds, rdr)
			}
		}
	}
}

// Dispose deallocates all active chunk images and every pooled image held by DualGridSystem.
func (s *DualGridSystem) Dispose() {
	for ent, state := range s.states {
		for i, img := range state.chunkImages {
			if img != nil {
				img.Deallocate()
				state.chunkImages[i] = nil
			}
		}
		clear(state.loadedFlags)
		delete(s.states, ent)
	}
	for i, img := range s.imagePool {
		if img != nil {
			img.Deallocate()
			s.imagePool[i] = nil
		}
	}
	s.imagePool = s.imagePool[:0]
	s.loadedChunks = 0
}

// Stats returns the current DualGridStats snapshot.
func (s *DualGridSystem) Stats() DualGridStats {
	return DualGridStats{
		ActiveMaps:   len(s.states),
		LoadedChunks: s.loadedChunks,
		PooledImages: len(s.imagePool),
	}
}

func (s *DualGridSystem) currentViewBounds() Rectangle {
	if s.activeCamHandle.Has() {
		if cam := s.activeCamHandle.Get(); cam != nil && cam.HasCamera {
			return cam.ViewBounds
		}
	}
	return NewRectangle(0, 0, 800, 600)
}

func (s *DualGridSystem) ensureState(ent teishoku.Entity, m *DualGridMap, ts *DualGridTileset) *dualGridEntityState {
	state, ok := s.states[ent]
	if !ok {
		total := m.chunksX * m.chunksY
		state = &dualGridEntityState{
			gridMap:      m,
			tileset:      ts,
			atlasVersion: ts.version,
			width:        m.width,
			height:       m.height,
			tileSize:     m.tileSize,
			chunkSize:    m.chunkSize,
			chunksX:      m.chunksX,
			chunksY:      m.chunksY,
			chunkImages:  make([]*ebiten.Image, total),
			loadedFlags:  make([]bool, total),
		}
		m.MarkAllDirty()
		s.states[ent] = state
		return state
	}

	dimsChanged := state.gridMap != m ||
		state.width != m.width ||
		state.height != m.height ||
		state.tileSize != m.tileSize ||
		state.chunkSize != m.chunkSize ||
		state.chunksX != m.chunksX ||
		state.chunksY != m.chunksY

	if dimsChanged {
		s.releaseStateImages(state)
		total := m.chunksX * m.chunksY
		if cap(state.chunkImages) < total {
			state.chunkImages = make([]*ebiten.Image, total)
		} else {
			state.chunkImages = state.chunkImages[:total]
			clear(state.chunkImages)
		}
		if cap(state.loadedFlags) < total {
			state.loadedFlags = make([]bool, total)
		} else {
			state.loadedFlags = state.loadedFlags[:total]
			clear(state.loadedFlags)
		}
		state.gridMap = m
		state.tileset = ts
		state.atlasVersion = ts.version
		state.width = m.width
		state.height = m.height
		state.tileSize = m.tileSize
		state.chunkSize = m.chunkSize
		state.chunksX = m.chunksX
		state.chunksY = m.chunksY
		m.MarkAllDirty()
		return state
	}

	if state.tileset != ts || state.atlasVersion != ts.version {
		state.tileset = ts
		state.atlasVersion = ts.version
		m.MarkAllDirty()
	}

	return state
}

func (s *DualGridSystem) releaseStateImages(state *dualGridEntityState) {
	for i, img := range state.chunkImages {
		if img != nil {
			s.recycleChunkImage(img)
			state.chunkImages[i] = nil
		}
		state.loadedFlags[i] = false
	}
}

func (s *DualGridSystem) acquireChunkImage(chunkPx int) *ebiten.Image {
	if chunkPx <= 0 {
		chunkPx = 1
	}
	for len(s.imagePool) > 0 {
		last := len(s.imagePool) - 1
		img := s.imagePool[last]
		s.imagePool[last] = nil
		s.imagePool = s.imagePool[:last]
		if img == nil {
			continue
		}
		b := img.Bounds()
		if b.Dx() == chunkPx && b.Dy() == chunkPx {
			return img
		}
		img.Deallocate()
	}
	return ebiten.NewImage(chunkPx, chunkPx)
}

func (s *DualGridSystem) recycleChunkImage(img *ebiten.Image) {
	if img == nil {
		return
	}
	if len(s.imagePool) < s.maxPooledImages {
		s.imagePool = append(s.imagePool, img)
		return
	}
	img.Deallocate()
}

func (s *DualGridSystem) bakeChunk(
	dst *ebiten.Image,
	m *DualGridMap,
	tileset *DualGridTileset,
	atlas *ebiten.Image,
	cx, cy int,
) {
	if dst == nil || m == nil || tileset == nil || atlas == nil {
		return
	}
	dst.Clear()

	cs := m.chunkSize
	dxStart := cx * cs
	dyStart := cy * cs
	dxEnd := min(dxStart+cs, m.width+1)
	dyEnd := min(dyStart+cs, m.height+1)
	if dxStart >= dxEnd || dyStart >= dyEnd {
		return
	}

	matCount := tileset.MaterialCount()
	if matCount == 0 {
		return
	}

	dstTS := float32(m.tileSize)
	atlasTS := float32(tileset.TileSize())

	s.verts = s.verts[:0]
	s.inds = s.inds[:0]

	for dy := dyStart; dy < dyEnd; dy++ {
		ly := dy - dyStart
		y0 := float32(ly) * dstTS
		y1 := y0 + dstTS
		for dx := dxStart; dx < dxEnd; dx++ {
			if len(s.verts)+16 > dualGridMaxBatchVerts {
				dst.DrawTriangles(s.verts, s.inds, atlas, nil)
				s.verts = s.verts[:0]
				s.inds = s.inds[:0]
			}
			lx := dx - dxStart
			x0 := float32(lx) * dstTS
			x1 := x0 + dstTS

			tl := m.TileAt(dx-1, dy-1)
			tr := m.TileAt(dx, dy-1)
			bl := m.TileAt(dx-1, dy)
			br := m.TileAt(dx, dy)

			s.verts, s.inds = appendDualGridCellQuads(
				s.verts, s.inds,
				tl, tr, bl, br,
				matCount,
				x0, y0, x1, y1,
				atlasTS,
			)
		}
	}

	if len(s.verts) > 0 {
		dst.DrawTriangles(s.verts, s.inds, atlas, nil)
	}
}

func (s *DualGridSystem) drawOverlays(comp *DualGridComponent, viewBounds Rectangle, rdr *BatchRenderer) {
	m := comp.Map
	ts := float64(m.tileSize)
	if ts <= 0 {
		return
	}
	whiteImg := defaultWhiteImage()
	invTS := 1.0 / ts

	if comp.ShowLogicalGrid {
		minTX := max(0, int(math.Floor(viewBounds.Min.X*invTS)))
		maxTX := min(m.width, int(math.Ceil(viewBounds.Max.X*invTS)))
		minTY := max(0, int(math.Floor(viewBounds.Min.Y*invTS)))
		maxTY := min(m.height, int(math.Ceil(viewBounds.Max.Y*invTS)))
		gridCol := color.RGBA{R: 255, G: 255, B: 255, A: 48}
		spanY := float64(maxTY-minTY) * ts
		spanX := float64(maxTX-minTX) * ts

		if spanY > 0 {
			y0 := float64(minTY) * ts
			for tx := minTX; tx <= maxTX; tx++ {
				x := float64(tx) * ts
				rdr.AddQuad(V(x, y0), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, gridCol, 0, 0, 1, 1, 1, spanY)
			}
		}
		if spanX > 0 {
			x0 := float64(minTX) * ts
			for ty := minTY; ty <= maxTY; ty++ {
				y := float64(ty) * ts
				rdr.AddQuad(V(x0, y), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, gridCol, 0, 0, 1, 1, spanX, 1)
			}
		}
	}

	if comp.ShowDualCorners {
		minDX := max(0, int(math.Floor(viewBounds.Min.X*invTS)))
		maxDX := min(m.width, int(math.Ceil(viewBounds.Max.X*invTS)))
		minDY := max(0, int(math.Floor(viewBounds.Min.Y*invTS)))
		maxDY := min(m.height, int(math.Ceil(viewBounds.Max.Y*invTS)))
		dotCol := color.RGBA{R: 255, G: 220, B: 80, A: 180}
		dotSize := math.Max(2.0, math.Min(4.0, ts*0.125))
		halfDot := dotSize * 0.5
		for dy := minDY; dy <= maxDY; dy++ {
			y := float64(dy)*ts - halfDot
			for dx := minDX; dx <= maxDX; dx++ {
				x := float64(dx)*ts - halfDot
				rdr.AddQuad(V(x, y), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, dotCol, 0, 0, 1, 1, dotSize, dotSize)
			}
		}
	}

	if comp.HasHover && comp.HoverX >= 0 && comp.HoverY >= 0 && comp.HoverX < m.width && comp.HoverY < m.height {
		x := float64(comp.HoverX) * ts
		y := float64(comp.HoverY) * ts
		hoverCol := color.RGBA{R: 255, G: 240, B: 120, A: 220}
		const border = 2.0
		rdr.AddQuad(V(x, y), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, hoverCol, 0, 0, 1, 1, ts, border)
		rdr.AddQuad(V(x, y+ts-border), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, hoverCol, 0, 0, 1, 1, ts, border)
		rdr.AddQuad(V(x, y), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, hoverCol, 0, 0, 1, 1, border, ts)
		rdr.AddQuad(V(x+ts-border, y), ZeroVector, ZeroVector, V(1, 1), 0, whiteImg, hoverCol, 0, 0, 1, 1, border, ts)
	}
}

// NewDualGridTilePainter creates a WorldSystem TilePainter that renders chunks
// seamlessly using a DualGridTileset and a logical-to-material mapping function.
// Every chunk is rendered in a single dst.DrawTriangles batch (splitting only
// if vertex count exceeds maxVertices), with zero steady-state allocations.
func NewDualGridTilePainter(
	gen *WorldGenerator,
	tileset *DualGridTileset,
	mapTile func(tx, ty int, tileID uint16, biome BiomeID) DualMaterialID,
) TilePainter {
	if gen == nil || tileset == nil {
		return func(*ebiten.Image, *WorldChunk, int) {}
	}
	if mapTile == nil {
		mapTile = func(_, _ int, tileID uint16, _ BiomeID) DualMaterialID {
			return DualMaterialID(tileID)
		}
	}

	verts := make([]ebiten.Vertex, 0, 2048)
	inds := make([]uint16, 0, 3072)

	return func(dst *ebiten.Image, chunk *WorldChunk, tileSize int) {
		if dst == nil || chunk == nil || chunk.Size <= 0 || tileSize <= 0 {
			return
		}
		atlas := tileset.Atlas()
		if atlas == nil {
			return
		}
		matCount := tileset.MaterialCount()
		if matCount == 0 {
			return
		}

		worldW, worldH := gen.WorldTileSize()
		if worldW <= 0 || worldH <= 0 {
			return
		}

		sampleMat := func(tx, ty int) DualMaterialID {
			if tx < 0 {
				tx = 0
			} else if tx >= worldW {
				tx = worldW - 1
			}
			if ty < 0 {
				ty = 0
			} else if ty >= worldH {
				ty = worldH - 1
			}
			return mapTile(tx, ty, gen.TileAt(tx, ty), gen.BiomeAt(tx, ty))
		}

		baseTX := chunk.CX * chunk.Size
		baseTY := chunk.CY * chunk.Size
		dstTS := float32(tileSize)
		halfTS := dstTS * 0.5
		atlasTS := float32(tileset.TileSize())

		verts = verts[:0]
		inds = inds[:0]

		for ly := 0; ly <= chunk.Size; ly++ {
			ty := baseTY + ly
			y0 := float32(ly)*dstTS - halfTS
			y1 := y0 + dstTS
			for lx := 0; lx <= chunk.Size; lx++ {
				if len(verts)+16 > dualGridMaxBatchVerts {
					dst.DrawTriangles(verts, inds, atlas, nil)
					verts = verts[:0]
					inds = inds[:0]
				}
				tx := baseTX + lx
				x0 := float32(lx)*dstTS - halfTS
				x1 := x0 + dstTS

				tl := sampleMat(tx-1, ty-1)
				tr := sampleMat(tx, ty-1)
				bl := sampleMat(tx-1, ty)
				br := sampleMat(tx, ty)

				verts, inds = appendDualGridCellQuads(
					verts, inds,
					tl, tr, bl, br,
					matCount,
					x0, y0, x1, y1,
					atlasTS,
				)
			}
		}

		if len(verts) > 0 {
			dst.DrawTriangles(verts, inds, atlas, nil)
		}
	}
}

// buildDualGridChunkVerts populates verts and inds with all dual-grid cell quads
// for chunk (cx, cy) of m, using reusable caller-provided buffers.
func buildDualGridChunkVerts(
	verts []ebiten.Vertex,
	inds []uint16,
	m *DualGridMap,
	matCount int,
	atlasTileSize float32,
	cx, cy int,
) ([]ebiten.Vertex, []uint16) {
	verts = verts[:0]
	inds = inds[:0]
	if m == nil || matCount <= 0 || cx < 0 || cy < 0 || cx >= m.chunksX || cy >= m.chunksY {
		return verts, inds
	}

	cs := m.chunkSize
	dxStart := cx * cs
	dyStart := cy * cs
	dxEnd := min(dxStart+cs, m.width+1)
	dyEnd := min(dyStart+cs, m.height+1)
	dstTS := float32(m.tileSize)

	for dy := dyStart; dy < dyEnd; dy++ {
		ly := dy - dyStart
		y0 := float32(ly) * dstTS
		y1 := y0 + dstTS
		for dx := dxStart; dx < dxEnd; dx++ {
			lx := dx - dxStart
			x0 := float32(lx) * dstTS
			x1 := x0 + dstTS

			tl := m.TileAt(dx-1, dy-1)
			tr := m.TileAt(dx, dy-1)
			bl := m.TileAt(dx-1, dy)
			br := m.TileAt(dx, dy)

			verts, inds = appendDualGridCellQuads(
				verts, inds,
				tl, tr, bl, br,
				matCount,
				x0, y0, x1, y1,
				atlasTileSize,
			)
		}
	}
	return verts, inds
}

func appendDualGridCellQuads(
	verts []ebiten.Vertex,
	inds []uint16,
	tl, tr, bl, br DualMaterialID,
	matCount int,
	dstX0, dstY0, dstX1, dstY1 float32,
	atlasTileSize float32,
) ([]ebiten.Vertex, []uint16) {
	if tl == tr && tl == bl && tl == br {
		if int(tl) < matCount {
			verts, inds = appendDualGridMaskQuad(verts, inds, 15, tl, dstX0, dstY0, dstX1, dstY1, atlasTileSize)
		}
		return verts, inds
	}

	var mats [4]DualMaterialID
	nMats := 0
	corners := [4]DualMaterialID{tl, tr, bl, br}
	for _, c := range corners {
		inserted := false
		duplicate := false
		for i := 0; i < nMats; i++ {
			if mats[i] == c {
				duplicate = true
				break
			}
			if c < mats[i] {
				for j := nMats; j > i; j-- {
					mats[j] = mats[j-1]
				}
				mats[i] = c
				nMats++
				inserted = true
				break
			}
		}
		if !duplicate && !inserted {
			mats[nMats] = c
			nMats++
		}
	}

	for i := 0; i < nMats; i++ {
		mat := mats[i]
		if int(mat) >= matCount {
			continue
		}
		mask := 0
		if tl >= mat {
			mask |= 8
		}
		if tr >= mat {
			mask |= 4
		}
		if bl >= mat {
			mask |= 2
		}
		if br >= mat {
			mask |= 1
		}
		if mask == 0 {
			continue
		}
		verts, inds = appendDualGridMaskQuad(verts, inds, mask, mat, dstX0, dstY0, dstX1, dstY1, atlasTileSize)
	}
	return verts, inds
}

func appendDualGridMaskQuad(
	verts []ebiten.Vertex,
	inds []uint16,
	bitmask int,
	mat DualMaterialID,
	dstX0, dstY0, dstX1, dstY1 float32,
	atlasTileSize float32,
) ([]ebiten.Vertex, []uint16) {
	srcX0 := float32(bitmask) * atlasTileSize
	srcY0 := float32(mat) * atlasTileSize
	srcX1 := srcX0 + atlasTileSize
	srcY1 := srcY0 + atlasTileSize

	base := uint16(len(verts))
	verts = append(verts,
		ebiten.Vertex{DstX: dstX0, DstY: dstY0, SrcX: srcX0, SrcY: srcY0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: dstX1, DstY: dstY0, SrcX: srcX1, SrcY: srcY0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: dstX1, DstY: dstY1, SrcX: srcX1, SrcY: srcY1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: dstX0, DstY: dstY1, SrcX: srcX0, SrcY: srcY1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	)
	inds = append(inds, base, base+1, base+2, base, base+2, base+3)
	return verts, inds
}
