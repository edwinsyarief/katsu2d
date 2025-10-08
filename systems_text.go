package katsu2d

import (
	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

var AlignmentOffsets = map[TextAlignment]func(w, h float64) (float64, float64){
	TextAlignmentTopRight:     func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleRight:  func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomRight:  func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopCenter:    func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleCenter: func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomCenter: func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentTopLeft:      func(w, h float64) (float64, float64) { return 0, -h },
	TextAlignmentMiddleLeft:   func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomLeft:   func(w, h float64) (float64, float64) { return 0, 0 },
}

type TextSystem struct {
	fm          *FontManager
	filter      *lazyecs.Filter2[TransformComponent, TextComponent]
	drawOpts    *text.DrawOptions
	transform   *Transform
	fontFaceMap map[lazyecs.Entity]*text.GoTextFace
	entities    []lazyecs.Entity
}

func NewTextSystem() *TextSystem {
	return &TextSystem{
		drawOpts:  &text.DrawOptions{},
		transform: T(),
	}
}
func (self *TextSystem) Initialize(w *lazyecs.World) {
	self.filter = self.filter.New(w)
	self.fm = GetFontManager(w)
}
func (self *TextSystem) Update(w *lazyecs.World, dt float64) {
	self.entities = make([]lazyecs.Entity, 0)
	self.fontFaceMap = make(map[lazyecs.Entity]*text.GoTextFace)
	self.filter.Reset()
	for self.filter.Next() {
		self.entities = append(self.entities, self.filter.Entity())
		_, txt := self.filter.Get()
		f := self.getFontFace(txt.FontID, txt.Size)
		self.updateCache(txt, f)
	}
}
func (self *TextSystem) Draw(w *lazyecs.World, rdr *BatchRenderer) {
	rdr.Flush()
	for _, e := range self.entities {
		t, txt := lazyecs.GetComponent2[TransformComponent, TextComponent](w, e)
		self.drawOpts.LineSpacing = txt.LineSpacing
		switch txt.Alignment {
		case TextAlignmentTopRight, TextAlignmentMiddleRight, TextAlignmentBottomRight:
			self.drawOpts.PrimaryAlign = text.AlignStart
		case TextAlignmentTopCenter, TextAlignmentMiddleCenter, TextAlignmentBottomCenter:
			self.drawOpts.PrimaryAlign = text.AlignCenter
		default:
			self.drawOpts.PrimaryAlign = text.AlignEnd
		}
		offsetX, offsetY := AlignmentOffsets[txt.Alignment](txt.CachedWidth, txt.CachedHeight)
		t.Offset = Point(V(offsetX, offsetY))
		self.transform.SetFromComponent(t)
		self.drawOpts.GeoM = self.transform.Matrix()
		self.drawOpts.ColorScale = RGBAToColorScale(txt.Color)
		text.Draw(rdr.screen, txt.Caption, self.fontFaceMap[e], self.drawOpts)
	}
}
func (self *TextSystem) updateCache(txt *TextComponent, fontFace *text.GoTextFace) {
	if txt.CachedText != txt.Caption {
		txt.CachedWidth, txt.CachedHeight = text.Measure(txt.Caption, fontFace, txt.LineSpacing)
		txt.CachedText = txt.Caption
	}
}
func (self *TextSystem) getFontFace(fontID int, size float64) *text.GoTextFace {
	fontFace, ok := self.fontFaceMap[self.filter.Entity()]
	if !ok {
		source := self.fm.Get(fontID)
		fontFace = &text.GoTextFace{
			Source:    source,
			Direction: text.DirectionLeftToRight,
			Size:      size,
			Language:  language.English,
		}
		self.fontFaceMap[self.filter.Entity()] = fontFace
	}
	return fontFace
}
