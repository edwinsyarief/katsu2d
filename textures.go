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

// TextureManager manages loading and retrieving textures using a texture atlas
// to optimize rendering performance. It packs multiple small textures into larger
// atlas images to reduce the number of draw calls.
type TextureManager struct {
	// atlases is a slice of texture atlases. A new atlas is created when the
	// existing ones are full.
	atlases []*atlas.Atlas
	// images is a slice of all the sub-images (textures) stored in the atlases.
	// The index of this slice is used as the texture ID.
	images []*atlas.Image
	// atlasWidth is the width of the atlas textures.
	atlasWidth int
	// atlasHeight is the height of the atlas textures.
	atlasHeight int
}

// NewTextureManager creates a manager with a default white texture and a default
// atlas size of 2048x2048.
func NewTextureManager() *TextureManager {
	return NewTextureManagerWithOptions(2048, 2048)
}

// NewTextureManagerWithOptions creates a manager with a default white texture
// and a custom atlas size.
func NewTextureManagerWithOptions(atlasWidth, atlasHeight int) *TextureManager {
	tm := &TextureManager{
		atlasWidth:  atlasWidth,
		atlasHeight: atlasHeight,
	}
	// Create the first atlas.
	tm.atlases = append(tm.atlases, atlas.New(tm.atlasWidth, tm.atlasHeight, nil))

	// Add a 1x1 white texture as the default texture (ID 0). This is useful
	// as a fallback for invalid texture IDs.
	white := ebiten.NewImage(1, 1)
	white.Fill(color.White)
	tm.Add(white)

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
	if id < 0 || id >= len(tm.images) {
		return tm.images[0].Image() // Fallback to white texture (ID 0)
	}
	return tm.images[id].Image()
}

// Add adds a pre-existing *ebiten.Image to an atlas and returns its new ID.
// It iterates through the existing atlases to find a space for the image. If no
// space is found, a new atlas is created. If the image is larger than the
// default atlas size, a new, custom-sized atlas is created for it.
func (tm *TextureManager) Add(img *ebiten.Image) int {
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
