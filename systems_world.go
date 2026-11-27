package katsu2d

import (
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// WorldStreamStats reports live chunk streaming and pool metrics for WorldSystem.
type WorldStreamStats struct {
	WorldName        string
	Seed             int64
	LoadedFromCache  bool
	TotalChunks      int
	LoadedChunks     int
	VisibleChunks    int
	StreamedEntities int
	PooledImages     int
}

// TilePainter renders a chunk's tiles onto dst using the given tile size in pixels.
type TilePainter func(dst *ebiten.Image, chunk *WorldChunk, tileSize int)

// ChunkLoadHook is invoked when a chunk is streamed in and returns any chunk-owned
// entities that should be automatically removed when the chunk unloads.
type ChunkLoadHook func(w *teishoku.World, chunk *WorldChunk) []teishoku.Entity

// ChunkUnloadHook is invoked immediately before a chunk's streamed entities are
// removed and its backing GPU image is returned to the pool or deallocated.
type ChunkUnloadHook func(w *teishoku.World, chunk *WorldChunk)

// WorldSystemOption configures a WorldSystem.
type WorldSystemOption func(*WorldSystem)

// WithWorldTileTextures configures the tile ID to texture lookup map used by the
// default chunk tile baker.
func WithWorldTileTextures(textures map[uint16]*ebiten.Image) WorldSystemOption {
	return func(s *WorldSystem) {
		if len(textures) == 0 {
			s.tileTextures = nil
			return
		}
		copied := make(map[uint16]*ebiten.Image, len(textures))
		for k, v := range textures {
			copied[k] = v
		}
		s.tileTextures = copied
	}
}

// WithWorldTilePainter overrides the default tile texture baker with a custom
// chunk painter (for example, autotiling, blending, or procedural overlays).
func WithWorldTilePainter(painter TilePainter) WorldSystemOption {
	return func(s *WorldSystem) {
		s.painter = painter
	}
}

// WithWorldOnChunkLoad registers a hook called whenever a chunk is streamed in.
func WithWorldOnChunkLoad(fn ChunkLoadHook) WorldSystemOption {
	return func(s *WorldSystem) {
		s.onChunkLoad = fn
	}
}

// WithWorldOnChunkUnload registers a hook called whenever a chunk is unloaded.
func WithWorldOnChunkUnload(fn ChunkUnloadHook) WorldSystemOption {
	return func(s *WorldSystem) {
		s.onChunkUnload = fn
	}
}

// WithWorldFallbackColor sets the solid fill color drawn for tiles lacking a
// matching entry in the tile texture map.
func WithWorldFallbackColor(col color.RGBA) WorldSystemOption {
	return func(s *WorldSystem) {
		s.fallbackColor = col
	}
}

// WorldSystem streams, bakes, caches, and draws visible WorldGenerator chunks
// around the active camera using a bounded GPU image pool.
type WorldSystem struct {
	gen             *WorldGenerator
	activeCamHandle teishoku.ResourceHandle[ActiveCamera]
	initialized     bool

	tileTextures  map[uint16]*ebiten.Image
	painter       TilePainter
	onChunkLoad   ChunkLoadHook
	onChunkUnload ChunkUnloadHook
	fallbackColor color.RGBA

	imagePool       []*ebiten.Image
	maxPooledImages int
	loadedChunks    []*WorldChunk
	drawGeom        ebiten.DrawImageOptions
	tileFallbackImg *ebiten.Image

	stats WorldStreamStats
}

// NewWorldSystem creates a WorldSystem bound to gen and configured with opts.
func NewWorldSystem(gen *WorldGenerator, opts ...WorldSystemOption) *WorldSystem {
	s := &WorldSystem{
		gen:             gen,
		fallbackColor:   color.RGBA{R: 48, G: 52, B: 64, A: 255},
		maxPooledImages: defaultMaxPooledImages,
	}

	if gen != nil {
		cfg := gen.config
		s.maxPooledImages = cfg.MaxPooledImages
		s.stats = WorldStreamStats{
			WorldName:       cfg.WorldName,
			Seed:            cfg.Seed,
			LoadedFromCache: gen.loadedFromCache,
			TotalChunks:     len(gen.chunks),
		}
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

// Initialize caches the world's ActiveCamera resource handle once.
func (s *WorldSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.activeCamHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	s.initialized = true
}

// Update streams chunks in and out around the active camera, re-bakes dirty
// loaded chunks, and updates WorldStreamStats with zero steady-state allocations.
func (s *WorldSystem) Update(w *teishoku.World, dt float64) {
	if s.gen == nil {
		return
	}
	if !s.initialized && w != nil {
		s.Initialize(w)
	}
	if w != nil && !s.activeCamHandle.Has() {
		s.activeCamHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	}

	viewBounds := s.currentViewBounds()
	cfg := s.gen.config
	chunkPixelSize := float64(cfg.ChunkSize * cfg.TileSize)

	rawVisMinCX, rawVisMinCY, rawVisMaxCX, rawVisMaxCY, hasVis := visibleChunkRange(viewBounds, chunkPixelSize)
	visMinCX := max(0, rawVisMinCX)
	visMinCY := max(0, rawVisMinCY)
	visMaxCX := min(cfg.ChunksX-1, rawVisMaxCX)
	visMaxCY := min(cfg.ChunksY-1, rawVisMaxCY)

	visibleCount := 0
	if hasVis && visMinCX <= visMaxCX && visMinCY <= visMaxCY {
		visibleCount = (visMaxCX - visMinCX + 1) * (visMaxCY - visMinCY + 1)
	}

	unloadMinCX := rawVisMinCX - cfg.UnloadMarginChunks
	unloadMinCY := rawVisMinCY - cfg.UnloadMarginChunks
	unloadMaxCX := rawVisMaxCX + cfg.UnloadMarginChunks
	unloadMaxCY := rawVisMaxCY + cfg.UnloadMarginChunks

	chunkPx := cfg.ChunkSize * cfg.TileSize

	// Unload chunks outside the unload margin and re-bake any dirty loaded chunks.
	writeIdx := 0
	for _, c := range s.loadedChunks {
		if !hasVis || c.CX < unloadMinCX || c.CX > unloadMaxCX || c.CY < unloadMinCY || c.CY > unloadMaxCY {
			s.unloadChunk(w, c)
			continue
		}
		if c.Dirty {
			if c.image == nil {
				c.image = s.acquireChunkImage(chunkPx)
			}
			s.bakeChunk(c.image, c)
			c.Dirty = false
		}
		s.loadedChunks[writeIdx] = c
		writeIdx++
	}
	if writeIdx < len(s.loadedChunks) {
		clear(s.loadedChunks[writeIdx:])
		s.loadedChunks = s.loadedChunks[:writeIdx]
	}

	// Stream in any unloaded chunks within the load margin.
	if hasVis {
		loadMinCX := max(0, rawVisMinCX-cfg.LoadMarginChunks)
		loadMinCY := max(0, rawVisMinCY-cfg.LoadMarginChunks)
		loadMaxCX := min(cfg.ChunksX-1, rawVisMaxCX+cfg.LoadMarginChunks)
		loadMaxCY := min(cfg.ChunksY-1, rawVisMaxCY+cfg.LoadMarginChunks)

		for cy := loadMinCY; cy <= loadMaxCY; cy++ {
			for cx := loadMinCX; cx <= loadMaxCX; cx++ {
				c := s.gen.ChunkAt(cx, cy)
				if c == nil || c.Loaded {
					continue
				}
				img := s.acquireChunkImage(chunkPx)
				s.bakeChunk(img, c)
				c.image = img
				c.Loaded = true
				c.Dirty = false
				if s.onChunkLoad != nil && w != nil {
					entities := s.onChunkLoad(w, c)
					if len(entities) > 0 {
						c.StreamedEntities = append(c.StreamedEntities[:0], entities...)
					} else {
						c.StreamedEntities = c.StreamedEntities[:0]
					}
				}
				s.loadedChunks = append(s.loadedChunks, c)
			}
		}
	}

	streamedEntities := 0
	for _, c := range s.loadedChunks {
		streamedEntities += len(c.StreamedEntities)
	}
	s.stats = WorldStreamStats{
		WorldName:        cfg.WorldName,
		Seed:             cfg.Seed,
		LoadedFromCache:  s.gen.loadedFromCache,
		TotalChunks:      len(s.gen.chunks),
		LoadedChunks:     len(s.loadedChunks),
		VisibleChunks:    visibleCount,
		StreamedEntities: streamedEntities,
		PooledImages:     len(s.imagePool),
	}
}

// Draw submits only the loaded chunks overlapping ActiveCamera.ViewBounds to rdr.
func (s *WorldSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if s.gen == nil || rdr == nil {
		return
	}
	if !s.initialized && w != nil {
		s.Initialize(w)
	}

	viewBounds := s.currentViewBounds()
	cfg := s.gen.config
	chunkPixelSize := float64(cfg.ChunkSize * cfg.TileSize)

	rawVisMinCX, rawVisMinCY, rawVisMaxCX, rawVisMaxCY, hasVis := visibleChunkRange(viewBounds, chunkPixelSize)
	if !hasVis {
		return
	}

	visMinCX := max(0, rawVisMinCX)
	visMinCY := max(0, rawVisMinCY)
	visMaxCX := min(cfg.ChunksX-1, rawVisMaxCX)
	visMaxCY := min(cfg.ChunksY-1, rawVisMaxCY)
	if visMinCX > visMaxCX || visMinCY > visMaxCY {
		return
	}

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for cy := visMinCY; cy <= visMaxCY; cy++ {
		for cx := visMinCX; cx <= visMaxCX; cx++ {
			c := s.gen.ChunkAt(cx, cy)
			if c == nil || !c.Loaded || c.image == nil {
				continue
			}
			b := c.image.Bounds()
			rdr.AddQuad(
				V(c.WorldX, c.WorldY),
				ZeroVector,
				ZeroVector,
				V(1, 1),
				0,
				c.image,
				white,
				float32(b.Min.X), float32(b.Min.Y), float32(b.Max.X), float32(b.Max.Y),
				chunkPixelSize, chunkPixelSize,
			)
		}
	}
}

// UnloadAll cleanly unloads all active chunks, removes their streamed entities,
// and returns chunk images to the bounded pool (or deallocates overflow).
func (s *WorldSystem) UnloadAll(w *teishoku.World) {
	for _, c := range s.loadedChunks {
		s.unloadChunk(w, c)
	}
	clear(s.loadedChunks)
	s.loadedChunks = s.loadedChunks[:0]

	s.stats.LoadedChunks = 0
	s.stats.VisibleChunks = 0
	s.stats.StreamedEntities = 0
	s.stats.PooledImages = len(s.imagePool)
}

// Dispose deallocates every active chunk image, every pooled image, and any
// internal fallback image held by the system.
func (s *WorldSystem) Dispose() {
	for _, c := range s.loadedChunks {
		if c.image != nil {
			c.image.Deallocate()
			c.image = nil
		}
		clear(c.StreamedEntities)
		c.StreamedEntities = c.StreamedEntities[:0]
		c.Loaded = false
	}
	clear(s.loadedChunks)
	s.loadedChunks = s.loadedChunks[:0]

	for i, img := range s.imagePool {
		if img != nil {
			img.Deallocate()
			s.imagePool[i] = nil
		}
	}
	s.imagePool = s.imagePool[:0]

	if s.tileFallbackImg != nil {
		s.tileFallbackImg.Deallocate()
		s.tileFallbackImg = nil
	}

	s.stats.LoadedChunks = 0
	s.stats.VisibleChunks = 0
	s.stats.StreamedEntities = 0
	s.stats.PooledImages = 0
}

// Stats returns the latest WorldStreamStats snapshot.
func (s *WorldSystem) Stats() WorldStreamStats {
	stats := s.stats
	if s.gen != nil {
		stats.WorldName = s.gen.config.WorldName
		stats.Seed = s.gen.config.Seed
		stats.LoadedFromCache = s.gen.loadedFromCache
		stats.TotalChunks = len(s.gen.chunks)
	}
	stats.PooledImages = len(s.imagePool)
	return stats
}

// Generator returns the underlying WorldGenerator.
func (s *WorldSystem) Generator() *WorldGenerator {
	return s.gen
}

func (s *WorldSystem) currentViewBounds() Rectangle {
	if s.activeCamHandle.Has() {
		if cam := s.activeCamHandle.Get(); cam != nil && cam.HasCamera {
			return cam.ViewBounds
		}
	}
	return NewRectangle(0, 0, 800, 600)
}

func visibleChunkRange(viewBounds Rectangle, chunkPixelSize float64) (minCX, minCY, maxCX, maxCY int, valid bool) {
	if viewBounds.IsEmpty() || chunkPixelSize <= 0 {
		return 0, 0, -1, -1, false
	}
	minCX = int(math.Floor(viewBounds.Min.X / chunkPixelSize))
	minCY = int(math.Floor(viewBounds.Min.Y / chunkPixelSize))
	maxCX = int(math.Ceil(viewBounds.Max.X/chunkPixelSize)) - 1
	maxCY = int(math.Ceil(viewBounds.Max.Y/chunkPixelSize)) - 1
	return minCX, minCY, maxCX, maxCY, true
}

func (s *WorldSystem) acquireChunkImage(chunkPx int) *ebiten.Image {
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

func (s *WorldSystem) unloadChunk(w *teishoku.World, c *WorldChunk) {
	if c == nil || !c.Loaded {
		return
	}
	if s.onChunkUnload != nil {
		s.onChunkUnload(w, c)
	}
	if w != nil && len(c.StreamedEntities) > 0 {
		w.RemoveEntities(c.StreamedEntities)
	}
	clear(c.StreamedEntities)
	c.StreamedEntities = c.StreamedEntities[:0]

	if c.image != nil {
		if len(s.imagePool) < s.maxPooledImages {
			s.imagePool = append(s.imagePool, c.image)
		} else {
			c.image.Deallocate()
		}
		c.image = nil
	}
	c.Loaded = false
}

func (s *WorldSystem) bakeChunk(dst *ebiten.Image, chunk *WorldChunk) {
	if dst == nil || chunk == nil {
		return
	}
	dst.Clear()

	if s.painter != nil {
		s.painter(dst, chunk, chunk.TileSize)
		return
	}

	size := chunk.Size
	ts := chunk.TileSize
	if size <= 0 || ts <= 0 {
		return
	}

	for ly := range size {
		row := ly * size
		py := float64(ly * ts)
		for lx := range size {
			tileID := chunk.Tiles[row+lx]
			var tex *ebiten.Image
			if s.tileTextures != nil {
				tex = s.tileTextures[tileID]
			}
			px := float64(lx * ts)

			if tex != nil {
				b := tex.Bounds()
				s.drawGeom.GeoM.Reset()
				if b.Dx() > 0 && b.Dy() > 0 && (b.Dx() != ts || b.Dy() != ts) {
					s.drawGeom.GeoM.Scale(float64(ts)/float64(b.Dx()), float64(ts)/float64(b.Dy()))
				}
				s.drawGeom.GeoM.Translate(px, py)
				s.drawGeom.ColorScale.Reset()
				dst.DrawImage(tex, &s.drawGeom)
			} else if s.fallbackColor.A > 0 {
				if s.tileFallbackImg == nil {
					s.tileFallbackImg = ebiten.NewImage(1, 1)
					s.tileFallbackImg.Fill(color.White)
				}
				s.drawGeom.GeoM.Reset()
				s.drawGeom.GeoM.Scale(float64(ts), float64(ts))
				s.drawGeom.GeoM.Translate(px, py)
				s.drawGeom.ColorScale = RGBAToColorScale(s.fallbackColor)
				dst.DrawImage(s.tileFallbackImg, &s.drawGeom)
			}
		}
	}
}
