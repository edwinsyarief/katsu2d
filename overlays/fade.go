package overlays

import (
	"image/color"
	"katsu2d/ease"
	"katsu2d/tween"

	"github.com/hajimehoshi/ebiten/v2"
)

type FadeType int

const (
	FadeTypeOut FadeType = iota
	FadeTypeIn
)

// Pre-allocate vertices and indices as they don't change
var (
	overlayIndices  = []uint16{0, 1, 2, 0, 2, 3}
	overlayVertices = make([]ebiten.Vertex, 4)
)

type FadeOverlay struct {
	fadeType    FadeType
	fadeColor   color.RGBA
	overlay     *ebiten.Image
	tween       *tween.Tween
	currentFade float32
	isFinished  bool
	callback    func()
}

func NewFadeOverlay(width, height int, fadeType FadeType, fadeColor color.RGBA, duration float64, callback func()) *FadeOverlay {
	begin, end := float32(0.0), float32(1.0)
	easing := ease.CubicInOut

	// Prepare a full-screen quad to draw the overlay
	overlayVertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	overlayVertices[1] = ebiten.Vertex{DstX: float32(width), DstY: 0, SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	overlayVertices[2] = ebiten.Vertex{DstX: float32(width), DstY: float32(height), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	overlayVertices[3] = ebiten.Vertex{DstX: 0, DstY: float32(height), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}

	if fadeType == FadeTypeIn {
		begin, end = 1.0, 0.0
		easing = ease.QuadInOut
	}

	overlay := ebiten.NewImage(1, 1)
	overlay.Fill(fadeColor)

	return &FadeOverlay{
		fadeType:  fadeType,
		fadeColor: fadeColor,
		overlay:   overlay,
		tween:     tween.New(begin, end, float32(duration), easing),
		callback:  callback,
	}
}

func (self *FadeOverlay) Reset(fadeType FadeType, fadeColor color.RGBA, delay, duration float64, callback func()) {
	self.isFinished = false
	self.currentFade = 0.0
	self.fadeType = fadeType
	self.fadeColor = fadeColor
	self.callback = callback

	begin := 0.0
	end := 1.0
	easing := ease.CubicInOut

	if fadeType == FadeTypeIn {
		begin = 1.0
		end = 0.0
		easing = ease.QuadInOut
	}

	self.tween = tween.New(float32(begin), float32(end), float32(duration), easing)
	self.tween.SetDelay(float32(delay))
}

func (self *FadeOverlay) Update(delta float64) {
	if self.isFinished {
		return
	}

	self.currentFade, self.isFinished = self.tween.Update(float32(delta))

	if self.isFinished && self.callback != nil {
		self.callback()
	}
}

func (self *FadeOverlay) GetType() FadeType {
	return self.fadeType
}

func (self *FadeOverlay) IsFinished() bool {
	return self.isFinished
}

func (self *FadeOverlay) Draw(screen *ebiten.Image) {
	if self.fadeType == FadeTypeIn && self.isFinished {
		return
	}

	// Update the overlay color with the current fade alpha
	fadeColor := self.fadeColor
	fadeColor.A = uint8(255 * self.currentFade)
	self.overlay.Fill(fadeColor)

	// Draw the overlay on the canvas
	screen.DrawTriangles(overlayVertices, overlayIndices, self.overlay, nil)
}
