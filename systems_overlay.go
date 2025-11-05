package katsu2d

import (
	"image/color"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

type FadeOverlaySystem struct {
	filter      *teishoku.Filter2[FadeOverlayComponent, TweenComponent]
	indices     Indices
	vertices    Vertices
	toRemove    []teishoku.Entity
	initialized bool
}

func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{
		indices:  Indices{0, 1, 2, 0, 2, 3},
		vertices: make(Vertices, 4),
		toRemove: make([]teishoku.Entity, 0),
	}
}

func (self *FadeOverlaySystem) Initialize(w *teishoku.World) {
	if self.initialized {
		return
	}

	self.filter = self.filter.New(w)
	self.initialized = true
}

func (self *FadeOverlaySystem) Update(w *teishoku.World, dt float64) {
	self.toRemove = self.toRemove[:0]
	self.filter.Reset()
	for self.filter.Next() {
		fade, tween := self.filter.Get()
		fade.CurrentFade = tween.Start

		if fade.FadeType == FadeTypeIn && fade.Finished {
			self.toRemove = append(self.toRemove, self.filter.Entity())
			continue
		}

		fade.CurrentFade, fade.Finished = tween.Current, tween.Finished
	}
	for _, e := range self.toRemove {
		w.RemoveEntity(e)
	}
}

func (self *FadeOverlaySystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	rdr.Flush()
	tm := GetTextureManager(w)
	width, height := rdr.screen.Bounds().Dx(), rdr.screen.Bounds().Dy()
	self.filter.Reset()
	for self.filter.Next() {
		fade, _ := self.filter.Get()

		if fade.FadeType == FadeTypeIn && fade.Finished {
			continue
		}

		// Apply alpha to overlay color and draw
		img := tm.Get(0)
		overlayColor := fade.FadeColor
		overlayColor.A = uint8(float64(overlayColor.A) * fade.CurrentFade)
		updateOverlayVertices(self.vertices, width, height, overlayColor)
		rdr.AddCustomMeshes(self.vertices, self.indices, img)
	}
}

func updateOverlayVertices(vertices Vertices, width, height int, col color.RGBA) {
	sR := float32(col.R) / 255.0
	sG := float32(col.G) / 255.0
	sB := float32(col.B) / 255.0
	sA := float32(col.A) / 255.0
	vertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[1] = ebiten.Vertex{DstX: float32(width), DstY: 0, SrcX: 1, SrcY: 0, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[2] = ebiten.Vertex{DstX: float32(width), DstY: float32(height), SrcX: 1, SrcY: 1, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[3] = ebiten.Vertex{DstX: 0, DstY: float32(height), SrcX: 0, SrcY: 1, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
}

type CinematicOverlaySystem struct {
	filter *teishoku.Filter3[CinematicOverlayComponent, TweenComponent, TimerComponent]
	// Pre-allocated buffers for performance
	indices      []uint16
	vertices     []ebiten.Vertex
	spotlightV   []ebiten.Vertex
	spotlightI   []uint16
	spotlightSeg int
	target       *ebiten.Image
	initialized  bool
}

func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	res := &CinematicOverlaySystem{
		indices:      []uint16{0, 1, 2, 0, 2, 3},
		vertices:     make([]ebiten.Vertex, 4),
		spotlightSeg: 64,
	}
	res.spotlightV = make([]ebiten.Vertex, res.spotlightSeg+2)
	res.spotlightI = make([]uint16, res.spotlightSeg*3)
	for i := 0; i < res.spotlightSeg; i++ {
		res.spotlightI[i*3] = 0
		res.spotlightI[i*3+1] = uint16(i + 1)
		res.spotlightI[i*3+2] = uint16(i + 2)
	}
	return res
}

func (self *CinematicOverlaySystem) Initialize(w *teishoku.World) {
	if self.initialized {
		return
	}

	self.filter = self.filter.New(w)

	Subscribe(w, self.onEngineLayoutChanged)
	Subscribe(w, self.onTimerFinished)
	Subscribe(w, self.onTweenFinished)

	self.initialized = true
}

func (self *CinematicOverlaySystem) onEngineLayoutChanged(data EngineLayoutChangedEvent) {
	self.target = ebiten.NewImage(data.Width, data.Height)
}

func (self *CinematicOverlaySystem) onTimerFinished(obj TimerFinishedEvent) {

}

func (self *CinematicOverlaySystem) onTweenFinished(obj TweenFinishedEvent) {

}

func (self *CinematicOverlaySystem) Update(w *teishoku.World, dt float64) {

}

func (self *CinematicOverlaySystem) Draw(w *teishoku.World, rdr *BatchRenderer) {

}
