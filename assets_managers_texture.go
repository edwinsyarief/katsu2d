package katsu2d

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"

	"github.com/edwinsyarief/katsu2d/atlas"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TextureManager manages loading and retrieving textures either as individual
// *ebiten.Images or packed inside texture atlases for draw call batching.
// Texture ID 0 is always a 1x1 solid white pixel.
type TextureManager struct {
	atlases     []*atlas.Atlas
	images      []*atlas.Image
	textures    []*ebiten.Image
	atlasWidth  int
	atlasHeight int
	useAtlas    bool
}

// TextureManagerOption configures a TextureManager.
type TextureManagerOption func(*TextureManager)

// WithAtlas enables or disables the texture atlas system.
func WithAtlas(enabled bool) TextureManagerOption {
	return func(tm *TextureManager) {
		tm.useAtlas = enabled
	}
}

// WithAtlasSize sets atlas dimensions and implicitly enables the atlas system.
func WithAtlasSize(w, h int) TextureManagerOption {
	return func(tm *TextureManager) {
		tm.useAtlas = true
		tm.atlasWidth = w
		tm.atlasHeight = h
	}
}

// NewTextureManager creates a TextureManager with a default 1x1 white texture at ID 0.
func NewTextureManager(opts ...TextureManagerOption) *TextureManager {
	tm := &TextureManager{
		useAtlas:    false,
		atlasWidth:  2048,
		atlasHeight: 2048,
	}
	for _, opt := range opts {
		opt(tm)
	}

	white := ebiten.NewImage(1, 1)
	white.Fill(color.White)

	if tm.useAtlas {
		tm.atlases = append(tm.atlases, atlas.New(tm.atlasWidth, tm.atlasHeight, nil))
		tm.Add(white)
	} else {
		tm.textures = append(tm.textures, white)
	}
	return tm
}

// Load loads an image from disk and returns its texture ID.
func (tm *TextureManager) Load(path string) (int, error) {
	_, img, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return 0, err
	}
	return tm.Add(ebiten.NewImageFromImage(img)), nil
}

// LoadEmbedded loads an image from the embedded filesystem and returns its ID.
func (tm *TextureManager) LoadEmbedded(path string) int {
	id, err := tm.LoadBytes(openEmbeddedFile(path))
	if err != nil {
		panic(err)
	}
	return id
}

// LoadFromAssetPacker loads an image from the asset bundle and returns its ID.
func (tm *TextureManager) LoadFromAssetPacker(path string) int {
	id, err := tm.LoadBytes(openBundledFile(path))
	if err != nil {
		panic(err)
	}
	return id
}

// LoadBytes decodes an image from raw bytes, registers it, and returns its ID.
func (tm *TextureManager) LoadBytes(content []byte) (int, error) {
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return 0, fmt.Errorf("katsu2d: failed to decode image: %w", err)
	}
	return tm.Add(ebiten.NewImageFromImage(img)), nil
}

// Get retrieves a texture by ID, falling back to the 1x1 white texture (ID 0) if out of range.
func (tm *TextureManager) Get(id int) *ebiten.Image {
	if tm.useAtlas {
		if id < 0 || id >= len(tm.images) {
			return tm.images[0].Image()
		}
		return tm.images[id].Image()
	}
	if id < 0 || id >= len(tm.textures) {
		return tm.textures[0]
	}
	return tm.textures[id]
}

// AtlasSize returns the configured atlas dimensions, or (0, 0) when atlas mode is disabled.
func (tm *TextureManager) AtlasSize() (int, int) {
	if !tm.useAtlas {
		return 0, 0
	}
	return tm.atlasWidth, tm.atlasHeight
}

// Add registers a pre-existing *ebiten.Image and returns its assigned texture ID.
func (tm *TextureManager) Add(img *ebiten.Image) int {
	if !tm.useAtlas {
		id := len(tm.textures)
		tm.textures = append(tm.textures, img)
		return id
	}

	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()
	var atlasImg *atlas.Image

	if imgWidth > tm.atlasWidth || imgHeight > tm.atlasHeight {
		newAtlas := atlas.New(imgWidth, imgHeight, nil)
		tm.atlases = append(tm.atlases, newAtlas)
		atlasImg = newAtlas.NewImage(imgWidth, imgHeight)
	} else {
		for _, a := range tm.atlases {
			if atlasImg = a.NewImage(imgWidth, imgHeight); atlasImg != nil {
				break
			}
		}
		if atlasImg == nil {
			newAtlas := atlas.New(tm.atlasWidth, tm.atlasHeight, nil)
			tm.atlases = append(tm.atlases, newAtlas)
			atlasImg = newAtlas.NewImage(imgWidth, imgHeight)
		}
	}

	if atlasImg == nil {
		panic("katsu2d: failed to allocate image on atlas")
	}

	atlasImg.DrawImage(img, nil)
	id := len(tm.images)
	tm.images = append(tm.images, atlasImg)
	return id
}
