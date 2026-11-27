package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

// AlignmentOffsets maps each TextAlignment to a function returning the local pivot offset for a measured text block.
var AlignmentOffsets = map[TextAlignment]func(w, h float64) (float64, float64){
	TextAlignmentTopRight:     func(w, h float64) (float64, float64) { return 0, h },
	TextAlignmentMiddleRight:  func(w, h float64) (float64, float64) { return 0, h / 2 },
	TextAlignmentBottomRight:  func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopCenter:    func(w, h float64) (float64, float64) { return 0, h },
	TextAlignmentMiddleCenter: func(w, h float64) (float64, float64) { return 0, h / 2 },
	TextAlignmentBottomCenter: func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopLeft:      func(w, h float64) (float64, float64) { return 0, h },
	TextAlignmentMiddleLeft:   func(w, h float64) (float64, float64) { return 0, h / 2 },
	TextAlignmentBottomLeft:   func(w, h float64) (float64, float64) { return 0, 0 },
}

type fontFaceKey struct {
	fontID int
	size   float64
}

// TextSystem measures and renders TextComponent entities using cached GoTextFace instances.
type TextSystem struct {
	fmHandle    teishoku.ResourceHandle[FontManager]
	filter      *teishoku.Filter2[TransformComponent, TextComponent]
	drawOpts    *text.DrawOptions
	transform   *Transform
	fontFaces   map[fontFaceKey]*text.GoTextFace
	initialized bool
}

// NewTextSystem creates a new TextSystem.
func NewTextSystem() *TextSystem {
	return &TextSystem{
		drawOpts:  &text.DrawOptions{},
		transform: T(),
		fontFaces: make(map[fontFaceKey]*text.GoTextFace),
	}
}

// Initialize prepares the TextSystem's ECS filter and FontManager resource handle.
func (s *TextSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter2[TransformComponent, TextComponent](w)
	s.fmHandle = teishoku.UseResource[FontManager](w.Resources())
	if s.fontFaces == nil {
		s.fontFaces = make(map[fontFaceKey]*text.GoTextFace)
	}
	s.initialized = true
}

// Update refreshes cached text measurements for all TextComponent entities.
func (s *TextSystem) Update(w *teishoku.World, dt float64) {
	fm := s.fmHandle.Get()
	if fm == nil {
		return
	}
	s.filter.EachChunk(func(_ []TransformComponent, texts []TextComponent) {
		for i := range texts {
			txt := &texts[i]
			f := s.getFontFace(fm, txt.FontID, txt.Size)
			if f == nil {
				continue
			}
			s.updateCache(txt, f)
		}
	})
}

// Draw flushes the BatchRenderer and draws all TextComponent entities.
func (s *TextSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	fm := s.fmHandle.Get()
	if fm == nil {
		return
	}
	rdr.Flush()
	s.filter.EachChunk(func(transforms []TransformComponent, texts []TextComponent) {
		for i := range texts {
			txt := &texts[i]
			f := s.getFontFace(fm, txt.FontID, txt.Size)
			if f == nil {
				continue
			}
			s.updateCache(txt, f)

			s.drawOpts.LineSpacing = txt.LineSpacing
			switch txt.Alignment {
			case TextAlignmentTopRight, TextAlignmentMiddleRight, TextAlignmentBottomRight:
				s.drawOpts.PrimaryAlign = text.AlignStart
			case TextAlignmentTopCenter, TextAlignmentMiddleCenter, TextAlignmentBottomCenter:
				s.drawOpts.PrimaryAlign = text.AlignCenter
			default:
				s.drawOpts.PrimaryAlign = text.AlignEnd
			}

			tmp := transforms[i]
			if fn, ok := AlignmentOffsets[txt.Alignment]; ok {
				offsetX, offsetY := fn(txt.CachedWidth, txt.CachedHeight)
				tmp.Offset = Point(V(offsetX, offsetY))
			}
			s.transform.SetFromComponent(&tmp)
			s.drawOpts.GeoM = s.transform.Matrix()
			if vm, ok := rdr.GetViewMatrix(); ok {
				s.drawOpts.GeoM.Concat(vm)
			}
			s.drawOpts.ColorScale = RGBAToColorScale(txt.Color)
			text.Draw(rdr.screen, txt.Caption, f, s.drawOpts)
		}
	})
}

func (s *TextSystem) updateCache(txt *TextComponent, fontFace *text.GoTextFace) {
	if txt.CachedText != txt.Caption ||
		txt.cachedFontID != txt.FontID ||
		txt.cachedSize != txt.Size ||
		txt.cachedLineSpacing != txt.LineSpacing {
		txt.CachedWidth, txt.CachedHeight = text.Measure(txt.Caption, fontFace, txt.LineSpacing)
		txt.CachedText = txt.Caption
		txt.cachedFontID = txt.FontID
		txt.cachedSize = txt.Size
		txt.cachedLineSpacing = txt.LineSpacing
	}
}

func (s *TextSystem) getFontFace(fm *FontManager, fontID int, size float64) *text.GoTextFace {
	key := fontFaceKey{fontID: fontID, size: size}
	if fontFace, ok := s.fontFaces[key]; ok {
		return fontFace
	}
	source := fm.Get(fontID)
	if source == nil {
		return nil
	}
	fontFace := &text.GoTextFace{
		Source:    source,
		Direction: text.DirectionLeftToRight,
		Size:      size,
		Language:  language.English,
	}
	s.fontFaces[key] = fontFace
	return fontFace
}
