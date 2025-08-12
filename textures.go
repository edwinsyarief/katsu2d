package katsu2d

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TextureManager manages loading and retrieving textures.
type TextureManager struct {
	textures []*ebiten.Image
}

// NewTextureManager creates a manager with a default white texture.
func NewTextureManager() *TextureManager {
	tm := &TextureManager{}
	white := ebiten.NewImage(1, 1)
	white.Fill(color.White)
	tm.textures = append(tm.textures, white)
	return tm
}

// Load loads an image from file and returns its ID.
func (self *TextureManager) Load(path string) (int, error) {
	var result *ebiten.Image
	_, img, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		panic(err)
	}

	result = ebiten.NewImageFromImage(img)
	id := len(self.textures)
	self.textures = append(self.textures, result)

	return id, nil
}

// LoadEmbedded loads an image from embedded file and returns its ID.
func (self *TextureManager) LoadEmbedded(path string) int {
	b := openEmbeddedFile(path)
	return self.fromByte(b)
}

func (self *TextureManager) LoadFromAssetPacker(path string) int {
	b := openBundledFile(path)
	return self.fromByte(b)
}

func (self *TextureManager) fromByte(content []byte) int {
	img := ebiten.NewImageFromImage(*self.decodeImage(&content))
	id := len(self.textures)
	self.textures = append(self.textures, img)
	return id
}

func (self *TextureManager) decodeImage(rawImage *[]byte) *image.Image {
	img, _, _ := image.Decode(bytes.NewReader(*rawImage))
	return &img
}

// Get retrieves a texture by ID, falling back to white if invalid.
func (self *TextureManager) Get(id int) *ebiten.Image {
	if id < 0 || id >= len(self.textures) {
		return self.textures[0] // Fallback to white
	}
	return self.textures[id]
}
