package katsu2d

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/edwinsyarief/katsu2d/opensimplex"
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	worldSaveVersion        = 1
	defaultWorldName        = "world"
	defaultChunksX          = 8
	defaultChunksY          = 8
	defaultChunkSize        = 16
	defaultTileSize         = 32
	defaultLoadMargin       = 1
	defaultUnloadMargin     = 2
	defaultMaxPooledImages  = 64
	defaultElevationScale   = 0.022
	defaultMoistureScale    = 0.018
	defaultTemperatureScale = 0.014
	defaultOctaves          = 4
)

// BiomeID identifies a biome type stored per tile and per chunk.
type BiomeID uint8

// BiomeDef describes a biome's tile palette, climate thresholds, minimap color,
// and optional user-defined metadata.
type BiomeDef struct {
	ID             BiomeID
	Name           string
	BaseTileID     uint16
	VariantTileIDs []uint16
	MinElevation   float64
	MaxElevation   float64
	MinMoisture    float64
	MaxMoisture    float64
	MinTemperature float64
	MaxTemperature float64
	Color          color.RGBA
	Metadata       map[string]any
}

// BiomeClassifier maps tile coordinates and normalized [0, 1] environmental noise
// values (elevation, moisture, temperature) to a BiomeID.
type BiomeClassifier func(tx, ty int, elevation, moisture, temperature float64) BiomeID

// WorldGenConfig configures procedural chunked biome world generation, caching,
// and streaming margins.
type WorldGenConfig struct {
	WorldName          string
	Seed               int64
	StorageDir         string
	ChunksX            int
	ChunksY            int
	ChunkSize          int
	TileSize           int
	LoadMarginChunks   int
	UnloadMarginChunks int
	MaxPooledImages    int
	ElevationScale     float64
	MoistureScale      float64
	TemperatureScale   float64
	Octaves            int
	Biomes             []BiomeDef
	Classifier         BiomeClassifier
}

// WorldChunk holds tile, biome, streaming, and GPU backing state for a single
// square chunk of the world.
type WorldChunk struct {
	CX               int
	CY               int
	Size             int
	TileSize         int
	WorldX           float64
	WorldY           float64
	Seed             int64
	Tiles            []uint16
	Biomes           []BiomeID
	DominantBiome    BiomeID
	Dirty            bool
	Loaded           bool
	StreamedEntities []teishoku.Entity
	image            *ebiten.Image
}

// TileAt returns the tile ID at local chunk coordinates (lx, ly), or 0 if out of bounds.
func (c *WorldChunk) TileAt(lx, ly int) uint16 {
	if c == nil || lx < 0 || ly < 0 || lx >= c.Size || ly >= c.Size {
		return 0
	}
	idx := ly*c.Size + lx
	if idx < 0 || idx >= len(c.Tiles) {
		return 0
	}
	return c.Tiles[idx]
}

// BiomeAt returns the BiomeID at local chunk coordinates (lx, ly), or 0 if out of bounds.
func (c *WorldChunk) BiomeAt(lx, ly int) BiomeID {
	if c == nil || lx < 0 || ly < 0 || lx >= c.Size || ly >= c.Size {
		return 0
	}
	idx := ly*c.Size + lx
	if idx < 0 || idx >= len(c.Biomes) {
		return 0
	}
	return c.Biomes[idx]
}

// Bounds returns the world-space pixel bounding rectangle of the chunk.
func (c *WorldChunk) Bounds() Rectangle {
	if c == nil {
		return Rectangle{}
	}
	span := float64(c.Size * c.TileSize)
	return NewRectangle(c.WorldX, c.WorldY, c.WorldX+span, c.WorldY+span)
}

type worldSaveFile struct {
	Version   int      `json:"version"`
	WorldName string   `json:"world_name"`
	Seed      int64    `json:"seed"`
	ChunksX   int      `json:"chunks_x"`
	ChunksY   int      `json:"chunks_y"`
	ChunkSize int      `json:"chunk_size"`
	TileSize  int      `json:"tile_size"`
	TilesRLE  []uint32 `json:"tiles_rle"`
	BiomesRLE []uint32 `json:"biomes_rle"`
}

// WorldGenerator manages deterministic multi-octave OpenSimplex biome world
// generation, RLE JSON persistence, runtime tile edits, and minimap rendering.
type WorldGenerator struct {
	config          WorldGenConfig
	cachePath       string
	loadedFromCache bool
	chunks          []WorldChunk
	biomeIndex      [256]int
}

// NewWorldGenerator creates a WorldGenerator from cfg. If StorageDir is non-empty
// and a matching cached JSON file exists, the world is loaded directly without
// regenerating; otherwise, the world is deterministically generated and saved.
func NewWorldGenerator(cfg WorldGenConfig) (*WorldGenerator, error) {
	cfg = normalizeWorldGenConfig(cfg)

	g := &WorldGenerator{
		config: cfg,
	}
	g.rebuildBiomeIndex()

	if cfg.StorageDir != "" {
		safeName := sanitizeWorldName(cfg.WorldName)
		g.cachePath = filepath.Join(cfg.StorageDir, safeName+".json")
	}

	totalChunks := cfg.ChunksX * cfg.ChunksY
	g.chunks = make([]WorldChunk, totalChunks)
	g.initChunkMetadata()

	if g.cachePath != "" {
		if _, err := os.Stat(g.cachePath); err == nil {
			if err := g.loadFromCache(); err != nil {
				return nil, err
			}
			g.loadedFromCache = true
			return g, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("katsu2d: stat world cache %q: %w", g.cachePath, err)
		}
	}

	g.generateAllChunks(false)
	g.loadedFromCache = false

	if g.cachePath != "" {
		if err := g.Save(); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// LoadedFromCache reports whether the world was loaded from an existing cache file.
func (g *WorldGenerator) LoadedFromCache() bool {
	return g.loadedFromCache
}

// CachePath returns the resolved JSON cache path, or "" if StorageDir is empty.
func (g *WorldGenerator) CachePath() string {
	return g.cachePath
}

// Config returns a copy of the generator's normalized configuration.
func (g *WorldGenerator) Config() WorldGenConfig {
	cfg := g.config
	if len(g.config.Biomes) > 0 {
		cfg.Biomes = make([]BiomeDef, len(g.config.Biomes))
		copy(cfg.Biomes, g.config.Biomes)
	}
	return cfg
}

// WorldName returns the configured world name.
func (g *WorldGenerator) WorldName() string {
	return g.config.WorldName
}

// Seed returns the active world generation seed.
func (g *WorldGenerator) Seed() int64 {
	return g.config.Seed
}

// WorldPixelSize returns the full world dimensions in pixels (width, height).
func (g *WorldGenerator) WorldPixelSize() (float64, float64) {
	w := float64(g.config.ChunksX * g.config.ChunkSize * g.config.TileSize)
	h := float64(g.config.ChunksY * g.config.ChunkSize * g.config.TileSize)
	return w, h
}

// WorldTileSize returns the full world dimensions in tiles (width, height).
func (g *WorldGenerator) WorldTileSize() (int, int) {
	return g.config.ChunksX * g.config.ChunkSize, g.config.ChunksY * g.config.ChunkSize
}

// ChunkCount returns the total number of chunks in the world.
func (g *WorldGenerator) ChunkCount() int {
	return len(g.chunks)
}

// ChunkAt returns the chunk at chunk coordinates (cx, cy), or nil if out of bounds.
func (g *WorldGenerator) ChunkAt(cx, cy int) *WorldChunk {
	if cx < 0 || cy < 0 || cx >= g.config.ChunksX || cy >= g.config.ChunksY {
		return nil
	}
	return &g.chunks[cy*g.config.ChunksX+cx]
}

// TileAt returns the tile ID at global tile coordinates (tx, ty), or 0 if out of bounds.
func (g *WorldGenerator) TileAt(tx, ty int) uint16 {
	w, h := g.WorldTileSize()
	if tx < 0 || ty < 0 || tx >= w || ty >= h {
		return 0
	}
	cs := g.config.ChunkSize
	chunk := &g.chunks[(ty/cs)*g.config.ChunksX+(tx/cs)]
	return chunk.TileAt(tx%cs, ty%cs)
}

// SetTile updates the tile ID at global tile coordinates (tx, ty) and marks
// its owning chunk dirty for GPU re-baking. Returns false if out of bounds.
func (g *WorldGenerator) SetTile(tx, ty int, tileID uint16) bool {
	w, h := g.WorldTileSize()
	if tx < 0 || ty < 0 || tx >= w || ty >= h {
		return false
	}
	cs := g.config.ChunkSize
	chunk := &g.chunks[(ty/cs)*g.config.ChunksX+(tx/cs)]
	idx := (ty%cs)*cs + (tx % cs)
	if chunk.Tiles[idx] != tileID {
		chunk.Tiles[idx] = tileID
		chunk.Dirty = true
	}
	return true
}

// SetDualGridTile updates the tile ID at global tile coordinates (tx, ty) via
// SetTile and also marks adjacent chunks dirty when (tx, ty) lies on a chunk
// border so dual-grid transitions update seamlessly across chunk seams.
func (g *WorldGenerator) SetDualGridTile(tx, ty int, tileID uint16) bool {
	oldTile := g.TileAt(tx, ty)
	if !g.SetTile(tx, ty, tileID) {
		return false
	}
	if oldTile == tileID {
		return true
	}

	cs := g.config.ChunkSize
	cx := tx / cs
	cy := ty / cs
	lx := tx % cs
	ly := ty % cs

	minDX, maxDX := 0, 0
	if lx == 0 {
		minDX = -1
	}
	if lx == cs-1 {
		maxDX = 1
	}
	minDY, maxDY := 0, 0
	if ly == 0 {
		minDY = -1
	}
	if ly == cs-1 {
		maxDY = 1
	}

	for oy := minDY; oy <= maxDY; oy++ {
		for ox := minDX; ox <= maxDX; ox++ {
			if ox == 0 && oy == 0 {
				continue
			}
			if nc := g.ChunkAt(cx+ox, cy+oy); nc != nil {
				nc.Dirty = true
			}
		}
	}
	return true
}

// BiomeAt returns the BiomeID at global tile coordinates (tx, ty), or 0 if out of bounds.
func (g *WorldGenerator) BiomeAt(tx, ty int) BiomeID {
	w, h := g.WorldTileSize()
	if tx < 0 || ty < 0 || tx >= w || ty >= h {
		return 0
	}
	cs := g.config.ChunkSize
	chunk := &g.chunks[(ty/cs)*g.config.ChunksX+(tx/cs)]
	return chunk.BiomeAt(tx%cs, ty%cs)
}

// BiomeAtWorld returns the BiomeID at world-space pixel coordinates (wx, wy).
func (g *WorldGenerator) BiomeAtWorld(wx, wy float64) BiomeID {
	ts := float64(g.config.TileSize)
	tx := int(math.Floor(wx / ts))
	ty := int(math.Floor(wy / ts))
	return g.BiomeAt(tx, ty)
}

// BiomeDefByID returns the BiomeDef matching id, or (BiomeDef{}, false) if not registered.
func (g *WorldGenerator) BiomeDefByID(id BiomeID) (BiomeDef, bool) {
	idx := g.biomeIndex[id]
	if idx < 0 || idx >= len(g.config.Biomes) {
		return BiomeDef{}, false
	}
	return g.config.Biomes[idx], true
}

// Save atomically persists the current world data as RLE-encoded JSON to CachePath().
func (g *WorldGenerator) Save() error {
	if g.cachePath == "" {
		return errors.New("katsu2d: cannot save world with empty StorageDir")
	}

	dir := filepath.Dir(g.cachePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("katsu2d: create world cache dir %q: %w", dir, err)
	}

	tilesRLE, biomesRLE := g.encodeRLE()
	save := worldSaveFile{
		Version:   worldSaveVersion,
		WorldName: g.config.WorldName,
		Seed:      g.config.Seed,
		ChunksX:   g.config.ChunksX,
		ChunksY:   g.config.ChunksY,
		ChunkSize: g.config.ChunkSize,
		TileSize:  g.config.TileSize,
		TilesRLE:  tilesRLE,
		BiomesRLE: biomesRLE,
	}

	data, err := json.Marshal(save)
	if err != nil {
		return fmt.Errorf("katsu2d: marshal world cache: %w", err)
	}

	tmpPath := g.cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("katsu2d: write temp world cache %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, g.cachePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("katsu2d: rename world cache %q: %w", g.cachePath, err)
	}

	return nil
}

// Regenerate rebuilds all world chunks with newSeed, marks every chunk dirty,
// clears LoadedFromCache, and saves to disk if StorageDir is configured.
func (g *WorldGenerator) Regenerate(newSeed int64) error {
	g.config.Seed = newSeed
	g.generateAllChunks(true)
	g.loadedFromCache = false

	if g.cachePath != "" {
		if err := g.Save(); err != nil {
			return err
		}
	}
	return nil
}

// BuildMinimap allocates a WidthTiles x HeightTiles pixel-art overview image
// colored by each tile's BiomeDef.Color. The caller owns the returned image.
func (g *WorldGenerator) BuildMinimap() *ebiten.Image {
	w, h := g.WorldTileSize()
	if w <= 0 || h <= 0 {
		return nil
	}

	pixels := make([]byte, w*h*4)
	fallback := color.RGBA{R: 128, G: 128, B: 128, A: 255}
	cs := g.config.ChunkSize

	for i := range g.chunks {
		c := &g.chunks[i]
		baseX := c.CX * cs
		baseY := c.CY * cs
		for ly := range cs {
			rowBase := (baseY + ly) * w
			tileRow := ly * cs
			for lx := range cs {
				biomeID := c.Biomes[tileRow+lx]
				col := fallback
				if bIdx := g.biomeIndex[biomeID]; bIdx >= 0 {
					col = g.config.Biomes[bIdx].Color
				}
				pIdx := (rowBase + baseX + lx) * 4
				pixels[pIdx] = col.R
				pixels[pIdx+1] = col.G
				pixels[pIdx+2] = col.B
				pixels[pIdx+3] = col.A
			}
		}
	}

	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}

func normalizeWorldGenConfig(cfg WorldGenConfig) WorldGenConfig {
	if cfg.WorldName == "" {
		cfg.WorldName = defaultWorldName
	}
	if cfg.ChunksX <= 0 {
		cfg.ChunksX = defaultChunksX
	}
	if cfg.ChunksY <= 0 {
		cfg.ChunksY = defaultChunksY
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = defaultChunkSize
	}
	if cfg.TileSize <= 0 {
		cfg.TileSize = defaultTileSize
	}
	if cfg.LoadMarginChunks <= 0 {
		cfg.LoadMarginChunks = defaultLoadMargin
	}
	if cfg.UnloadMarginChunks <= 0 {
		cfg.UnloadMarginChunks = defaultUnloadMargin
	}
	if cfg.UnloadMarginChunks < cfg.LoadMarginChunks {
		cfg.UnloadMarginChunks = cfg.LoadMarginChunks + 1
	}
	if cfg.MaxPooledImages <= 0 {
		cfg.MaxPooledImages = defaultMaxPooledImages
	}
	if cfg.ElevationScale <= 0 {
		cfg.ElevationScale = defaultElevationScale
	}
	if cfg.MoistureScale <= 0 {
		cfg.MoistureScale = defaultMoistureScale
	}
	if cfg.TemperatureScale <= 0 {
		cfg.TemperatureScale = defaultTemperatureScale
	}
	if cfg.Octaves <= 0 {
		cfg.Octaves = defaultOctaves
	}
	if len(cfg.Biomes) > 0 {
		copied := make([]BiomeDef, len(cfg.Biomes))
		copy(copied, cfg.Biomes)
		cfg.Biomes = copied
	}
	return cfg
}

func sanitizeWorldName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		ch := name[i]
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if isAlphaNum || ch == '_' || ch == '-' {
			b.WriteByte(ch)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return defaultWorldName
	}
	return out
}

func (g *WorldGenerator) rebuildBiomeIndex() {
	for i := range g.biomeIndex {
		g.biomeIndex[i] = -1
	}
	for i, b := range g.config.Biomes {
		if g.biomeIndex[b.ID] == -1 {
			g.biomeIndex[b.ID] = i
		}
	}
}

func (g *WorldGenerator) initChunkMetadata() {
	cs := g.config.ChunkSize
	ts := g.config.TileSize
	tileCount := cs * cs
	chunkPixelSpan := float64(cs * ts)

	for cy := range g.config.ChunksY {
		for cx := range g.config.ChunksX {
			c := &g.chunks[cy*g.config.ChunksX+cx]
			c.CX = cx
			c.CY = cy
			c.Size = cs
			c.TileSize = ts
			c.WorldX = float64(cx) * chunkPixelSpan
			c.WorldY = float64(cy) * chunkPixelSpan
			c.Seed = deriveChunkSeed(g.config.Seed, cx, cy)
			if len(c.Tiles) != tileCount {
				c.Tiles = make([]uint16, tileCount)
			}
			if len(c.Biomes) != tileCount {
				c.Biomes = make([]BiomeID, tileCount)
			}
		}
	}
}

func (g *WorldGenerator) generateAllChunks(markDirty bool) {
	elevNoise := opensimplex.NewNormalized(mixSeed(g.config.Seed, 0x9e3779b97f4a7c15))
	moistNoise := opensimplex.NewNormalized(mixSeed(g.config.Seed, 0xc2b2ae3d27d4eb4f))
	tempNoise := opensimplex.NewNormalized(mixSeed(g.config.Seed, 0x165667b19e3779f9))
	detailNoise := opensimplex.NewNormalized(mixSeed(g.config.Seed, 0x85ebca77c2b2ae63))

	cs := g.config.ChunkSize
	elevScale := g.config.ElevationScale
	moistScale := g.config.MoistureScale
	tempScale := g.config.TemperatureScale
	octaves := g.config.Octaves

	for i := range g.chunks {
		c := &g.chunks[i]
		c.Seed = deriveChunkSeed(g.config.Seed, c.CX, c.CY)
		baseTX := c.CX * cs
		baseTY := c.CY * cs

		var biomeCounts [256]int
		for ly := range cs {
			ty := baseTY + ly
			fy := float64(ty)
			rowOffset := ly * cs
			for lx := range cs {
				tx := baseTX + lx
				fx := float64(tx)

				elev := sampleFBM(elevNoise, fx, fy, elevScale, octaves)
				moist := sampleFBM(moistNoise, fx, fy, moistScale, octaves)
				temp := sampleFBM(tempNoise, fx, fy, tempScale, octaves)

				biomeID := g.classifyBiome(tx, ty, elev, moist, temp)
				c.Biomes[rowOffset+lx] = biomeID
				biomeCounts[biomeID]++

				c.Tiles[rowOffset+lx] = g.pickTileForBiome(biomeID, tx, ty, detailNoise)
			}
		}

		c.DominantBiome = dominantBiomeFromCounts(&biomeCounts)
		if markDirty {
			c.Dirty = true
		}
	}
}

func (g *WorldGenerator) classifyBiome(tx, ty int, elev, moist, temp float64) BiomeID {
	if g.config.Classifier != nil {
		return g.config.Classifier(tx, ty, elev, moist, temp)
	}
	if len(g.config.Biomes) == 0 {
		return 0
	}

	for i := range g.config.Biomes {
		b := &g.config.Biomes[i]
		if rangeMatches(elev, b.MinElevation, b.MaxElevation) &&
			rangeMatches(moist, b.MinMoisture, b.MaxMoisture) &&
			rangeMatches(temp, b.MinTemperature, b.MaxTemperature) {
			return b.ID
		}
	}

	// Fallback: nearest biome by squared distance to configured environmental bounds.
	bestID := g.config.Biomes[0].ID
	bestDist := math.Inf(1)
	for i := range g.config.Biomes {
		b := &g.config.Biomes[i]
		de := rangeDistance(elev, b.MinElevation, b.MaxElevation)
		dm := rangeDistance(moist, b.MinMoisture, b.MaxMoisture)
		dt := rangeDistance(temp, b.MinTemperature, b.MaxTemperature)
		dist := de*de + dm*dm + dt*dt
		if dist < bestDist {
			bestDist = dist
			bestID = b.ID
		}
	}
	return bestID
}

func (g *WorldGenerator) pickTileForBiome(biomeID BiomeID, tx, ty int, detail opensimplex.Noise) uint16 {
	bIdx := g.biomeIndex[biomeID]
	if bIdx < 0 {
		return 0
	}
	b := &g.config.Biomes[bIdx]
	if len(b.VariantTileIDs) == 0 {
		return b.BaseTileID
	}

	n := detail.Eval2(float64(tx)*0.35, float64(ty)*0.35)
	if n > 0.68 {
		vIdx := int(((n - 0.68) / 0.32) * float64(len(b.VariantTileIDs)))
		if vIdx >= len(b.VariantTileIDs) {
			vIdx = len(b.VariantTileIDs) - 1
		}
		if vIdx < 0 {
			vIdx = 0
		}
		return b.VariantTileIDs[vIdx]
	}
	return b.BaseTileID
}

func (g *WorldGenerator) encodeRLE() (tilesRLE, biomesRLE []uint32) {
	for i := range g.chunks {
		c := &g.chunks[i]
		tilesRLE = appendRLEUint16(tilesRLE, c.Tiles)
		biomesRLE = appendRLEBiomes(biomesRLE, c.Biomes)
	}
	return tilesRLE, biomesRLE
}

func (g *WorldGenerator) loadFromCache() error {
	data, err := os.ReadFile(g.cachePath)
	if err != nil {
		return fmt.Errorf("katsu2d: read world cache %q: %w", g.cachePath, err)
	}

	var save worldSaveFile
	if err := json.Unmarshal(data, &save); err != nil {
		return fmt.Errorf("katsu2d: decode world cache %q: %w", g.cachePath, err)
	}

	if save.Version != worldSaveVersion {
		return fmt.Errorf("katsu2d: unsupported world cache version %d (want %d)", save.Version, worldSaveVersion)
	}
	if save.ChunksX != g.config.ChunksX ||
		save.ChunksY != g.config.ChunksY ||
		save.ChunkSize != g.config.ChunkSize ||
		save.TileSize != g.config.TileSize {
		return fmt.Errorf(
			"katsu2d: world cache dimensions mismatch: got %dx%d chunks (size %d, tile %d), want %dx%d (size %d, tile %d)",
			save.ChunksX, save.ChunksY, save.ChunkSize, save.TileSize,
			g.config.ChunksX, g.config.ChunksY, g.config.ChunkSize, g.config.TileSize,
		)
	}

	totalTiles := g.config.ChunksX * g.config.ChunksY * g.config.ChunkSize * g.config.ChunkSize
	if err := decodeRLEIntoChunks(save.TilesRLE, save.BiomesRLE, g.chunks, totalTiles); err != nil {
		return fmt.Errorf("katsu2d: corrupt world cache %q: %w", g.cachePath, err)
	}

	g.config.Seed = save.Seed
	for i := range g.chunks {
		c := &g.chunks[i]
		c.Seed = deriveChunkSeed(save.Seed, c.CX, c.CY)
		var counts [256]int
		for _, b := range c.Biomes {
			counts[b]++
		}
		c.DominantBiome = dominantBiomeFromCounts(&counts)
		c.Dirty = false
	}
	return nil
}

func appendRLEUint16(dst []uint32, src []uint16) []uint32 {
	for _, v := range src {
		val := uint32(v)
		n := len(dst)
		if n >= 2 && dst[n-1] == val {
			dst[n-2]++
		} else {
			dst = append(dst, 1, val)
		}
	}
	return dst
}

func appendRLEBiomes(dst []uint32, src []BiomeID) []uint32 {
	for _, v := range src {
		val := uint32(v)
		n := len(dst)
		if n >= 2 && dst[n-1] == val {
			dst[n-2]++
		} else {
			dst = append(dst, 1, val)
		}
	}
	return dst
}

func decodeRLEIntoChunks(tilesRLE, biomesRLE []uint32, chunks []WorldChunk, totalTiles int) error {
	if len(tilesRLE)%2 != 0 || len(biomesRLE)%2 != 0 {
		return errors.New("odd RLE stream length")
	}

	chunkTileCap := 0
	if len(chunks) > 0 {
		chunkTileCap = len(chunks[0].Tiles)
	}
	if chunkTileCap == 0 {
		return errors.New("empty chunk buffer")
	}

	decodedTiles := 0
	chunkIdx := 0
	localIdx := 0
	for i := 0; i < len(tilesRLE); i += 2 {
		count := int(tilesRLE[i])
		val := uint16(tilesRLE[i+1])
		if count <= 0 || decodedTiles+count > totalTiles {
			return errors.New("invalid tile RLE run length")
		}
		decodedTiles += count
		for count > 0 {
			rem := chunkTileCap - localIdx
			take := min(count, rem)
			dst := chunks[chunkIdx].Tiles[localIdx : localIdx+take]
			for k := range dst {
				dst[k] = val
			}
			localIdx += take
			count -= take
			if localIdx == chunkTileCap {
				chunkIdx++
				localIdx = 0
			}
		}
	}
	if decodedTiles != totalTiles {
		return fmt.Errorf("decoded %d tiles, expected %d", decodedTiles, totalTiles)
	}

	decodedBiomes := 0
	chunkIdx = 0
	localIdx = 0
	for i := 0; i < len(biomesRLE); i += 2 {
		count := int(biomesRLE[i])
		val := BiomeID(biomesRLE[i+1])
		if count <= 0 || decodedBiomes+count > totalTiles {
			return errors.New("invalid biome RLE run length")
		}
		decodedBiomes += count
		for count > 0 {
			rem := chunkTileCap - localIdx
			take := min(count, rem)
			dst := chunks[chunkIdx].Biomes[localIdx : localIdx+take]
			for k := range dst {
				dst[k] = val
			}
			localIdx += take
			count -= take
			if localIdx == chunkTileCap {
				chunkIdx++
				localIdx = 0
			}
		}
	}
	if decodedBiomes != totalTiles {
		return fmt.Errorf("decoded %d biomes, expected %d", decodedBiomes, totalTiles)
	}

	return nil
}

func sampleFBM(n opensimplex.Noise, x, y, scale float64, octaves int) float64 {
	var total, maxAmp float64
	amplitude := 1.0
	freq := scale
	for range octaves {
		total += n.Eval2(x*freq, y*freq) * amplitude
		maxAmp += amplitude
		amplitude *= 0.5
		freq *= 2.0
	}
	if maxAmp <= 0 {
		return 0
	}
	return Clamp(total/maxAmp, 0.0, 1.0)
}

func mixSeed(seed int64, salt uint64) int64 {
	z := uint64(seed) + salt
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z = z ^ (z >> 31)
	return int64(z)
}

func deriveChunkSeed(worldSeed int64, cx, cy int) int64 {
	h := uint64(worldSeed) ^ 0x9e3779b97f4a7c15
	h ^= uint64(int64(cx)) * 0xbf58476d1ce4e5b9
	h = (h ^ (h >> 30)) * 0x94d049bb133111eb
	h ^= uint64(int64(cy)) * 0x517cc1b727220a95
	h = (h ^ (h >> 27)) * 0xbf58476d1ce4e5b9
	return int64(h ^ (h >> 31))
}

func rangeMatches(val, minV, maxV float64) bool {
	if minV == 0 && maxV == 0 {
		return true
	}
	return val >= minV && val <= maxV
}

func rangeDistance(val, minV, maxV float64) float64 {
	if minV == 0 && maxV == 0 {
		return 0
	}
	if val < minV {
		return minV - val
	}
	if val > maxV {
		return val - maxV
	}
	return 0
}

func dominantBiomeFromCounts(counts *[256]int) BiomeID {
	var best BiomeID
	bestCount := -1
	for id, cnt := range counts {
		if cnt > bestCount {
			bestCount = cnt
			best = BiomeID(id)
		}
	}
	return best
}
