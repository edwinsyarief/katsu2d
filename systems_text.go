package katsu2d

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/mlange-42/ark/ecs"
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
	filter      *ecs.Filter2[TransformComponent, TextComponent]
	drawOpts    *text.DrawOptions
	transform   *Transform
	fontFaceMap map[ecs.Entity]*text.GoTextFace
	initialized bool
}

func NewTextSystem() *TextSystem {
	return &TextSystem{
		drawOpts:  &text.DrawOptions{},
		transform: T(),
	}
}
func (self *TextSystem) Initialize(w *ecs.World) {
	if self.initialized {
		return
	}

	self.filter = self.filter.New(w)
	self.fm = GetFontManager(w)
	self.initialized = true
}
func (self *TextSystem) Update(w *ecs.World, dt float64) {
	self.fontFaceMap = make(map[ecs.Entity]*text.GoTextFace)
	query := self.filter.Query()
	for query.Next() {
		_, txt := query.Get()
		f := self.getFontFace(query, txt.FontID, txt.Size)
		self.updateCache(txt, f)
	}
}
func (self *TextSystem) Draw(w *ecs.World, rdr *BatchRenderer) {
	rdr.Flush()
	query := self.filter.Query()
	for query.Next() {
		t, txt := query.Get()
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
		text.Draw(rdr.screen, txt.Caption, self.fontFaceMap[query.Entity()], self.drawOpts)
	}
}
func (self *TextSystem) updateCache(txt *TextComponent, fontFace *text.GoTextFace) {
	if txt.CachedText != txt.Caption {
		txt.CachedWidth, txt.CachedHeight = text.Measure(txt.Caption, fontFace, txt.LineSpacing)
		txt.CachedText = txt.Caption
	}
}
func (self *TextSystem) getFontFace(query ecs.Query2[TransformComponent, TextComponent], fontID int, size float64) *text.GoTextFace {
	fontFace, ok := self.fontFaceMap[query.Entity()]
	if !ok {
		source := self.fm.Get(fontID)
		fontFace = &text.GoTextFace{
			Source:    source,
			Direction: text.DirectionLeftToRight,
			Size:      size,
			Language:  language.English,
		}
		self.fontFaceMap[query.Entity()] = fontFace
	}
	return fontFace
}
