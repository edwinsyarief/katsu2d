package katsu2d

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"

	"github.com/edwinsyarief/katsu2d/atlas"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TextureManager manages loading and retrieving textures. It can operate in two
// modes: using a texture atlas for performance, or managing individual textures
// as a simple list.
type TextureManager struct {
	// --- Atlas Mode Fields ---
	// atlases is a slice of texture atlases. A new atlas is created when the
	// existing ones are full. Only used when useAtlas is true.
	atlases []*atlas.Atlas
	// images is a slice of all the sub-images (textures) stored in the atlases.
	// The index of this slice is used as the texture ID. Only used when useAtlas is true.
	images []*atlas.Image
	// --- Non-Atlas Mode Fields ---
	// textures is a simple slice of images. Only used when useAtlas is false.
	textures    []*ebiten.Image
	atlasWidth  int
	atlasHeight int
	// --- Configuration ---
	useAtlas bool
}

// TextureManagerOption is a functional option for configuring a TextureManager.
type TextureManagerOption func(*TextureManager)

// WithAtlas enables or disables the texture atlas system.
func WithAtlas(enabled bool) TextureManagerOption {
	return func(tm *TextureManager) {
		tm.useAtlas = enabled
	}
}

// WithAtlasSize sets the dimensions for the internal texture atlases.
// This implicitly enables the atlas system.
func WithAtlasSize(w, h int) TextureManagerOption {
	return func(tm *TextureManager) {
		tm.useAtlas = true
		tm.atlasWidth = w
		tm.atlasHeight = h
	}
}

// NewTextureManager creates a manager with a default white texture.
// By default, the atlas system is disabled. Use WithAtlas(true) or
// WithAtlasSize(...) to enable it.
func NewTextureManager(opts ...TextureManagerOption) *TextureManager {
	tm := &TextureManager{
		// Default values
		useAtlas:    false,
		atlasWidth:  2048,
		atlasHeight: 2048,
	}
	for _, opt := range opts {
		opt(tm)
	}
	// Add a 1x1 white texture as the default texture (ID 0).
	white := ebiten.NewImage(1, 1)
	white.Fill(color.White)
	// Initialize the appropriate storage and add the white texture.
	if tm.useAtlas {
		tm.atlases = append(tm.atlases, atlas.New(tm.atlasWidth, tm.atlasHeight, nil))
		// The Add method will handle adding the white texture to the atlas.
	} else {
		tm.textures = append(tm.textures, white)
	}
	// If using the atlas, we need to call Add to place the white texture.
	// In non-atlas mode, it's already in the slice.
	if tm.useAtlas {
		tm.Add(white)
	}
	return tm
}

// Load loads an image from a file, adds it to an atlas, and returns its ID.
func (tm *TextureManager) Load(path string) (int, error) {
	_, img, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return 0, err
	}
	return tm.Add(ebiten.NewImageFromImage(img)), nil
}

// LoadEmbedded loads an image from an embedded file, adds it to an atlas, and
// returns its ID.
func (tm *TextureManager) LoadEmbedded(path string) int {
	b := openEmbeddedFile(path)
	return tm.fromByte(b)
}

// LoadFromAssetPacker loads an image from a bundled asset file, adds it to an
// atlas, and returns its ID.
func (tm *TextureManager) LoadFromAssetPacker(path string) int {
	b := openBundledFile(path)
	return tm.fromByte(b)
}

// fromByte decodes an image from a byte slice, adds it to an atlas, and returns
// its ID.
func (tm *TextureManager) fromByte(content []byte) int {
	img := ebiten.NewImageFromImage(*tm.decodeImage(&content))
	return tm.Add(img)
}

// decodeImage decodes a byte slice into an image.Image.
func (tm *TextureManager) decodeImage(rawImage *[]byte) *image.Image {
	img, _, _ := image.Decode(bytes.NewReader(*rawImage))
	return &img
}

// Get retrieves a texture by its ID. If the ID is invalid, it returns the
// default white texture.
func (tm *TextureManager) Get(id int) *ebiten.Image {
	if tm.useAtlas {
		if id < 0 || id >= len(tm.images) {
			return tm.images[0].Image() // Fallback to white texture (ID 0)
		}
		return tm.images[id].Image()
	}
	if id < 0 || id >= len(tm.textures) {
		return tm.textures[0] // Fallback to white texture (ID 0)
	}
	return tm.textures[id]
}

// AtlasSize returns the default dimensions of the atlases created by the manager.
// Returns 0, 0 if the atlas system is not in use.
func (tm *TextureManager) AtlasSize() (int, int) {
	if !tm.useAtlas {
		return 0, 0
	}
	return tm.atlasWidth, tm.atlasHeight
}

// Add adds a pre-existing *ebiten.Image to the manager and returns its new ID.
// The behavior depends on whether the atlas system is enabled.
func (tm *TextureManager) Add(img *ebiten.Image) int {
	if !tm.useAtlas {
		id := len(tm.textures)
		tm.textures = append(tm.textures, img)
		return id
	}
	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()
	var atlasImg *atlas.Image
	// Handle images that are larger than the standard atlas size.
	if imgWidth > tm.atlasWidth || imgHeight > tm.atlasHeight {
		// Create a new atlas specifically for this large image.
		newAtlas := atlas.New(imgWidth, imgHeight, nil)
		tm.atlases = append(tm.atlases, newAtlas)
		atlasImg = newAtlas.NewImage(imgWidth, imgHeight)
	} else {
		// For regular-sized images, try to fit them into an existing atlas.
		for _, a := range tm.atlases {
			atlasImg = a.NewImage(imgWidth, imgHeight)
			if atlasImg != nil {
				break // Found a spot
			}
		}
		// If no space was found, create a new atlas with the default size.
		if atlasImg == nil {
			newAtlas := atlas.New(tm.atlasWidth, tm.atlasHeight, nil)
			tm.atlases = append(tm.atlases, newAtlas)
			atlasImg = newAtlas.NewImage(imgWidth, imgHeight)
		}
	}
	// If atlasImg is still nil, it means we couldn't allocate space even in a
	// new atlas, which indicates a problem.
	if atlasImg == nil {
		panic("katsu2d: failed to allocate image on atlas")
	}
	// Draw the image onto the atlas.
	atlasImg.DrawImage(img, nil)
	// Add the new atlas image to the list and return its ID.
	id := len(tm.images)
	tm.images = append(tm.images, atlasImg)
	return id
}
