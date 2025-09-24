package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// FadeOverlaySystem manages fade overlays.
type FadeOverlaySystem struct {
	indices         []uint16
	vertices        []ebiten.Vertex
	overlayEntities []lazyecs.Entity
	overlays        map[lazyecs.Entity]*FadeOverlayComponent
}

// NewFadeOverlaySystem creates a new FadeOverlaySystem.
func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{
		indices:  []uint16{0, 1, 2, 0, 2, 3},
		vertices: make([]ebiten.Vertex, 4),
	}
}

// Update updates all fade overlays in the world.
func (self *FadeOverlaySystem) Update(world *lazyecs.World, dt float64) {
	toRemove := []lazyecs.Entity{}
	query := world.Query(CTFadeOverlay)
	for query.Next() {
		for _, entity := range query.Entities() {
			fade, _ := lazyecs.GetComponent[FadeOverlayComponent](world, entity)
			self.overlays[entity] = fade

			if fade.Finished {
				if fade.FadeType == FadeTypeIn {
					toRemove = append(toRemove, entity)
				}
				continue
			}

			fade.CurrentFade, fade.Finished = fade.Tween.Update(float32(dt))

			if fade.Finished && fade.Callback != nil {
				fade.Callback()
			}

			self.overlayEntities = append(self.overlayEntities, entity)
		}
	}

	for _, entity := range toRemove {
		delete(self.overlays, entity)
		world.RemoveEntity(entity)
	}
}

func (self *FadeOverlaySystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	for _, entity := range self.overlayEntities {
		fade := self.overlays[entity]

		if fade.FadeType == FadeTypeIn && fade.Finished {
			continue
		}

		fade.Overlay.Clear()
		width, height := renderer.screen.Bounds().Dx(), renderer.screen.Bounds().Dy()
		self.updateOverlayVertices(width, height)

		// Apply alpha to overlay color and draw
		overlayColor := fade.FadeColor
		overlayColor.A = uint8(255 * fade.CurrentFade)
		fade.Overlay.Fill(overlayColor)

		renderer.Flush()
		renderer.screen.DrawTriangles(self.vertices, self.indices, fade.Overlay, nil)
	}
}

func (self *FadeOverlaySystem) updateOverlayVertices(width, height int) {
	self.vertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[1] = ebiten.Vertex{DstX: float32(width), DstY: 0, SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[2] = ebiten.Vertex{DstX: float32(width), DstY: float32(height), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[3] = ebiten.Vertex{DstX: 0, DstY: float32(height), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
}

// CinematicOverlaySystem manages cinematic overlays.
type CinematicOverlaySystem struct {
	// Pre-allocated buffers for performance
	indices         []uint16
	vertices        []ebiten.Vertex
	spotlightV      []ebiten.Vertex
	spotlightI      []uint16
	spotlightSeg    int
	toRemove        []lazyecs.Entity
	overlayEntities []lazyecs.Entity
	overlays        map[lazyecs.Entity]*CinematicOverlayComponent
}

// NewCinematicOverlaySystem creates a new CinematicOverlaySystem.
func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	self := &CinematicOverlaySystem{
		indices:      []uint16{0, 1, 2, 0, 2, 3},
		vertices:     make([]ebiten.Vertex, 4),
		spotlightSeg: 64,
	}
	self.spotlightV = make([]ebiten.Vertex, self.spotlightSeg+2)
	self.spotlightI = make([]uint16, self.spotlightSeg*3)
	for i := 0; i < self.spotlightSeg; i++ {
		self.spotlightI[i*3] = 0
		self.spotlightI[i*3+1] = uint16(i + 1)
		self.spotlightI[i*3+2] = uint16(i + 2)
	}
	return self
}

// Update updates all cinematic overlays in the world.
func (self *CinematicOverlaySystem) Update(world *lazyecs.World, dt float64) {
	self.toRemove = self.toRemove[:0]
	query := world.Query(CTCinematicOverlay)
	for query.Next() {
		for _, entity := range query.Entities() {
			cinematic, _ := lazyecs.GetComponent[CinematicOverlayComponent](world, entity)
			self.overlays[entity] = cinematic

			if cinematic.Finished {
				if cinematic.EndType == CinematicTransitionIn {
					self.toRemove = append(self.toRemove, entity)
				}
				continue
			}

			self.overlayEntities = append(self.overlayEntities, entity)

			// Update timing
			cinematic.Delayer.Update(dt)

			// Trigger delay if cinematic is finished and auto-finish is enabled
			if cinematic.CinematicFinished && cinematic.AutoFinish && !cinematic.DelayFinished {
				cinematic.Delayer.Activate("cinematic_delay")
			}

			// Handle start phase
			if !cinematic.CinematicFinished {
				value, finished := cinematic.Tween.Update(float32(dt))
				cinematic.CinematicFinished = finished
				cinematic.TransitionValue = float64(value)
			} else if cinematic.DelayFinished {
				// Transition to end phase once
				cinematic.lastStepOnce.Do(cinematic.setupLastStep)
				value, finished := cinematic.Tween.Update(float32(dt))
				cinematic.TransitionValue = float64(value)
				cinematic.Finished = finished
				if cinematic.Finished && cinematic.Callback != nil {
					cinematic.Callback()
				}
			} else {
				continue
			}

			// Update alpha during start phase
			if !cinematic.CinematicFinished && cinematic.StartFade {
				if cinematic.CinematicType == CinematicMovie {
					cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, cinematic.TransitionValue)
				} else {
					cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, cinematic.TransitionValue/cinematic.SpotlightMaxValue)
				}
			}

			// Update alpha during end phase
			if cinematic.DelayFinished && cinematic.EndFade {
				if !cinematic.StartFade {
					if cinematic.CinematicType == CinematicMovie {
						cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(cinematic.Radius/float64(cinematic.Height), 1.0, cinematic.TransitionValue)
					} else {
						cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(cinematic.Radius/cinematic.SpotlightMaxValue, 1.0, cinematic.TransitionValue/cinematic.SpotlightMaxValue)
					}
				} else {
					if cinematic.CinematicType == CinematicMovie {
						cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, cinematic.TransitionValue)
					} else {
						cinematic.CinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, cinematic.TransitionValue/cinematic.SpotlightMaxValue)
					}
				}
			}
		}
	}
	for _, entity := range self.toRemove {
		delete(self.overlays, entity)
		world.RemoveEntity(entity)
	}
}

// Draw renders all cinematic overlays to the screen.
func (self *CinematicOverlaySystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	for _, entity := range self.overlayEntities {
		cinematic := self.overlays[entity]

		if cinematic.EndType == CinematicTransitionIn && cinematic.Finished {
			continue
		}

		cinematic.RenderTarget.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0})
		width, height := renderer.screen.Bounds().Dx(), renderer.screen.Bounds().Dy()

		// Update vertices for the current frame
		self.updateOverlayVertices(width, height)

		// Apply alpha to overlay color and draw
		overlayColor := cinematic.OverlayColor
		overlayColor.A = uint8(255 * cinematic.CinematicAlphaValue)
		cinematic.Overlay.Fill(overlayColor)

		cinematic.RenderTarget.DrawTriangles(self.vertices, self.indices, cinematic.Overlay, nil)

		drawOptions := &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendClear}

		if cinematic.CinematicType == CinematicMovie {
			self.drawLetterbox(cinematic, width, height, drawOptions)
		} else {
			self.drawSpotlight(cinematic, drawOptions)
		}

		renderer.Flush()
		renderer.screen.DrawImage(cinematic.RenderTarget, nil)
	}
}

func (self *CinematicOverlaySystem) updateOverlayVertices(width, height int) {
	self.vertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[1] = ebiten.Vertex{DstX: float32(width), DstY: 0, SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[2] = ebiten.Vertex{DstX: float32(width), DstY: float32(height), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	self.vertices[3] = ebiten.Vertex{DstX: 0, DstY: float32(height), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
}

func (self *CinematicOverlaySystem) drawLetterbox(cinematic *CinematicOverlayComponent, width, height int, drawOptions *ebiten.DrawTrianglesOptions) {
	cinematic.Placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	middle := float64(height) / 2
	halfScalingSize := (float64(height) * cinematic.TransitionValue) / 2

	p0 := ebimath.V(0, middle-halfScalingSize)
	p1 := ebimath.V(float64(width), middle-halfScalingSize)
	p2 := ebimath.V(float64(width), middle+halfScalingSize)
	p3 := ebimath.V(0, middle+halfScalingSize)

	placeholderVertices := []ebiten.Vertex{
		{DstX: utils.AdjustDestinationPixel(float32(p0.X)), DstY: utils.AdjustDestinationPixel(float32(p0.Y)), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: utils.AdjustDestinationPixel(float32(p1.X)), DstY: utils.AdjustDestinationPixel(float32(p1.Y)), SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: utils.AdjustDestinationPixel(float32(p2.X)), DstY: utils.AdjustDestinationPixel(float32(p2.Y)), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: utils.AdjustDestinationPixel(float32(p3.X)), DstY: utils.AdjustDestinationPixel(float32(p3.Y)), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}

	overlayIndices := []uint16{0, 1, 2, 0, 2, 3}
	cinematic.RenderTarget.DrawTriangles(placeholderVertices, overlayIndices, cinematic.Placeholder, drawOptions)
}

func (self *CinematicOverlaySystem) drawSpotlight(cinematic *CinematicOverlayComponent, drawOptions *ebiten.DrawTrianglesOptions) {
	centerX := cinematic.Offset.X
	centerY := cinematic.Offset.Y
	radius := cinematic.TransitionValue

	self.spotlightV[0] = ebiten.Vertex{
		DstX:   float32(centerX),
		DstY:   float32(centerY),
		SrcX:   0.5,
		SrcY:   0.5,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	}

	for i := 0; i <= self.spotlightSeg; i++ {
		angle := 2 * math.Pi * float64(i) / float64(self.spotlightSeg)
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		self.spotlightV[i+1] = ebiten.Vertex{
			DstX:   utils.AdjustDestinationPixel(float32(x)),
			DstY:   utils.AdjustDestinationPixel(float32(y)),
			SrcX:   0.5,
			SrcY:   0.5,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		}
	}

	cinematic.RenderTarget.DrawTriangles(self.spotlightV, self.spotlightI, cinematic.Placeholder, drawOptions)
}
