package katsu2d

import (
	"bytes"
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed internal_assets/fonts/pixel_code.otf
var _DefaultFont []byte

type FontManager struct {
	fonts []*text.GoTextFaceSource
}

// NewFontManager creates a font manager.
func NewFontManager() *FontManager {
	fm := &FontManager{
		fonts: make([]*text.GoTextFaceSource, 0),
	}
	return fm
}

func (self *FontManager) Load(path string) int {
	b := readFile(path)
	return self.fromByte(b)
}

func (self *FontManager) LoadEmbedded(path string) int {
	b := openEmbeddedFile(path)
	return self.fromByte(b)
}

func (self *FontManager) LoadFromAssetPacker(path string) int {
	b := openBundledFile(path)
	return self.fromByte(b)
}

func (self *FontManager) fromByte(content []byte) int {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(content))
	if err != nil {
		panic(err)
	}
	id := len(self.fonts)
	self.fonts = append(self.fonts, font)
	return id
}

// Get retrieves a font by ID, falling back to white if invalid.
func (self *FontManager) Get(id int) *text.GoTextFaceSource {
	if id < 0 || id >= len(self.fonts) {
		return self.fonts[0] // Fallback to white
	}
	return self.fonts[id]
}
