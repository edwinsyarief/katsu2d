package katsu2d

import (
	"bytes"
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed internal_assets/fonts/pixel_code.otf
var defaultFontBytes []byte

// FontManager loads and stores GoTextFaceSource fonts indexed by integer ID.
// ID 0 is always the built-in default font.
type FontManager struct {
	fonts []*text.GoTextFaceSource
}

// NewFontManager creates a FontManager preloaded with the built-in default font at ID 0.
func NewFontManager() *FontManager {
	fm := &FontManager{
		fonts: make([]*text.GoTextFaceSource, 0, 4),
	}
	fm.fromBytes(defaultFontBytes)
	return fm
}

// Load loads a font file from disk and returns its ID.
func (fm *FontManager) Load(path string) int {
	return fm.fromBytes(readFile(path))
}

// LoadEmbedded loads a font file from the embedded filesystem and returns its ID.
func (fm *FontManager) LoadEmbedded(path string) int {
	return fm.fromBytes(openEmbeddedFile(path))
}

// LoadFromAssetPacker loads a font from the encrypted asset bundle and returns its ID.
func (fm *FontManager) LoadFromAssetPacker(path string) int {
	return fm.fromBytes(openBundledFile(path))
}

// LoadBytes parses a font from raw bytes and returns its ID.
func (fm *FontManager) LoadBytes(content []byte) int {
	return fm.fromBytes(content)
}

func (fm *FontManager) fromBytes(content []byte) int {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(content))
	if err != nil {
		panic(err)
	}
	id := len(fm.fonts)
	fm.fonts = append(fm.fonts, font)
	return id
}

// Get retrieves a font by ID, falling back to the built-in default font (ID 0) if invalid.
func (fm *FontManager) Get(id int) *text.GoTextFaceSource {
	if id < 0 || id >= len(fm.fonts) {
		return fm.fonts[0]
	}
	return fm.fonts[id]
}
