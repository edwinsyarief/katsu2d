package overlays

import (
	"image/color"
	"math"
	"sync"

	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/katsu2d/utils"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// Pre-allocated buffers for performance
var (
	cinematicOverlayIndices = []uint16{0, 1, 2, 0, 2, 3}
	cinematicVertices       = make([]ebiten.Vertex, 4)
	spotlightVertices       = make([]ebiten.Vertex, 66) // segments + 2
	spotlightIndices        = make([]uint16, 64*3)      // Pre-calculated for 64 segments
)

const (
	SpotlightSegments = 64
	// SpotlightOvershootFactor is the additional scaling factor for the spotlight effect.
	SpotlightOvershootFactor = 0.25
)

// Initialize spotlight indices once
func init() {
	for i := 0; i < SpotlightSegments; i++ {
		spotlightIndices[i*3] = 0
		spotlightIndices[i*3+1] = uint16(i + 1)
		spotlightIndices[i*3+2] = uint16(i + 2)
	}
}

// CinematicType defines the type of cinematic effect.
type CinematicType int

const (
	CinematicMovie     CinematicType = iota // Letterbox-style effect
	CinematicSpotlight                      // Circular spotlight effect
)

// CinematicOverlayType defines the transition direction.
type CinematicOverlayType int

const (
	CinematicTransitionIn  CinematicOverlayType = iota // Fade in
	CinematicTransitionOut                             // Fade out
)

// CinematicOverlay manages a visual overlay for cinematic effects in a game.
type CinematicOverlay struct {
	cinematicType                                          CinematicType
	startType, endType                                     CinematicOverlayType
	startFade, endFade                                     bool
	autoFinish, cinematicFinished, delayFinished, finished bool
	radius, startSpeed, endSpeed, cinematicDelay           float64
	cinematicAlphaValue                                    float64
	transitionValue                                        float64
	width, height                                          int
	spotlightMaxValue                                      float64 // Precomputed max value for spotlight

	offset               ebimath.Vector
	overlayColor         color.RGBA
	overlayOpacity       float64
	overlay, placeholder *ebiten.Image
	renderTarget         *ebiten.Image // Renamed from temp for clarity

	tween   *tween.Sequence
	delayer *managers.Delayer

	lastStepOnce sync.Once

	callback func()
}

// NewCinematicOverlay creates a new cinematic overlay with the specified parameters.
func NewCinematicOverlay(width, height int, col color.RGBA, opacity float64, cinematicType CinematicType,
	startType, endType CinematicOverlayType, startFade, endFade, autoFinish bool,
	cinematicDelay, radius, startSpeed, endSpeed float64, offset ebimath.Vector, callback func()) *CinematicOverlay {
	result := &CinematicOverlay{
		overlayColor:   col,
		overlayOpacity: opacity,
		width:          width,
		height:         height,
		cinematicType:  cinematicType,
		startType:      startType,
		endType:        endType,
		startFade:      startFade,
		endFade:        endFade,
		autoFinish:     autoFinish,
		cinematicDelay: cinematicDelay,
		radius:         radius,
		startSpeed:     startSpeed,
		endSpeed:       endSpeed,
		offset:         offset,
		delayer:        managers.NewDelayer(),
		callback:       callback,
	}

	// Configure delayer for cinematic delay
	result.delayer.Add("cinematic_delay", cinematicDelay, func() {
		result.delayFinished = true
	})

	// Initialize image buffers
	result.renderTarget = ebiten.NewImage(width, height)
	result.overlay = ebiten.NewImage(1, 1)
	result.placeholder = ebiten.NewImage(1, 1)
	result.overlay.Fill(result.overlayColor)
	result.placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	result.cinematicAlphaValue = 1.0

	// Precompute spotlight max value if applicable
	if result.cinematicType == CinematicSpotlight {
		result.spotlightMaxValue = float64(height) * (1 + SpotlightOvershootFactor)
	}

	// Initialize the starting tween
	result.tween = result.createStartTween()

	return result
}

// createStartTween generates the initial tween sequence based on cinematic type and start type.
func (self *CinematicOverlay) createStartTween() *tween.Sequence {
	if self.cinematicType == CinematicMovie {
		if self.startType == CinematicTransitionIn {
			self.transitionValue = 0.0
			maxValue := self.radius / float64(self.height)
			return tween.NewSequence(
				tween.New(0.0, float32(maxValue), float32(self.startSpeed), ease.QuadInOut),
			)
		}
		self.transitionValue = 1.0
		return tween.NewSequence(
			tween.New(1.0, float32(self.radius/float64(self.height)), float32(self.startSpeed), ease.QuadInOut),
		)
	}

	// Spotlight type
	if self.startType == CinematicTransitionIn {
		self.transitionValue = 0.0
		return tween.NewSequence(
			tween.New(0.0, float32(self.radius), float32(self.startSpeed), ease.QuadInOut),
		)
	}
	self.transitionValue = self.spotlightMaxValue
	return tween.NewSequence(
		tween.New(float32(self.spotlightMaxValue), float32(self.radius), float32(self.startSpeed), ease.QuadInOut),
	)
}

// Update advances the state of the cinematic overlay.
func (self *CinematicOverlay) Update(delta float64) {
	if self.finished {
		return
	}

	// Update timing
	self.delayer.Update(delta)

	// Trigger delay if cinematic is finished and auto-finish is enabled
	if self.cinematicFinished && self.autoFinish && !self.delayFinished {
		self.delayer.Activate("cinematic_delay")
	}

	// Handle start phase
	if !self.cinematicFinished {
		value, finished := self.tween.Update(float32(delta))
		self.cinematicFinished = finished
		self.transitionValue = float64(value)
	} else if self.delayFinished {
		// Transition to end phase once
		self.lastStepOnce.Do(self.setupLastStep)
		value, finished := self.tween.Update(float32(delta))
		self.transitionValue = float64(value)
		self.finished = finished
		if self.finished && self.callback != nil {
			self.callback()
		}
	} else {
		return
	}

	// Update alpha during start phase
	if !self.cinematicFinished && self.startFade {
		if self.cinematicType == CinematicMovie {
			self.cinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, self.transitionValue)
		} else {
			self.cinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, self.transitionValue/self.spotlightMaxValue)
		}
	}

	// Update alpha during end phase
	if self.delayFinished && self.endFade {
		if !self.startFade {
			if self.cinematicType == CinematicMovie {
				self.cinematicAlphaValue = utils.CalculateProgressRatio(self.radius/float64(self.height), 1.0, self.transitionValue)
			} else {
				self.cinematicAlphaValue = utils.CalculateProgressRatio(self.radius/self.spotlightMaxValue, 1.0, self.transitionValue/self.spotlightMaxValue)
			}
		} else {
			if self.cinematicType == CinematicMovie {
				self.cinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, self.transitionValue)
			} else {
				self.cinematicAlphaValue = utils.CalculateProgressRatio(0.0, 1.0, self.transitionValue/self.spotlightMaxValue)
			}
		}
	}
}

// EndCinematic manually triggers the end phase if not auto-finished.
func (self *CinematicOverlay) EndCinematic() {
	if self.autoFinish || !self.cinematicFinished {
		return
	}
	self.delayer.Activate("cinematic_delay")
}

// IsFinished checks if the overlay has completed.
func (self *CinematicOverlay) IsFinished() bool {
	return self.finished
}

// GetEndType returns the end transition type.
func (self *CinematicOverlay) GetEndType() CinematicOverlayType {
	return self.endType
}

func (self *CinematicOverlay) Draw(screen *ebiten.Image) {
	if self.endType == CinematicTransitionIn && self.finished {
		return
	}

	self.renderTarget.Clear()
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Update vertices for the current frame
	self.updateOverlayVertices(width, height)

	// Apply alpha to overlay color and draw
	overlayColor := self.overlayColor
	overlayColor.A = uint8(255 * self.cinematicAlphaValue)
	self.overlay.Fill(overlayColor)

	self.renderTarget.DrawTriangles(cinematicVertices, cinematicOverlayIndices, self.overlay, nil)

	drawOptions := &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendClear}

	if self.cinematicType == CinematicMovie {
		self.drawLetterbox(width, height, drawOptions)
	} else {
		self.drawSpotlight(drawOptions)
	}

	screen.DrawImage(self.renderTarget, nil)
}

// Helper methods to break down the Draw function
func (self *CinematicOverlay) updateOverlayVertices(width, height int) {
	cinematicVertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	cinematicVertices[1] = ebiten.Vertex{DstX: float32(width), DstY: 0, SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	cinematicVertices[2] = ebiten.Vertex{DstX: float32(width), DstY: float32(height), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	cinematicVertices[3] = ebiten.Vertex{DstX: 0, DstY: float32(height), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
}

// createEndTween generates the tween for the end phase.
func (self *CinematicOverlay) createEndTween() *tween.Tween {
	if self.cinematicType == CinematicMovie {
		if self.endType == CinematicTransitionIn {
			return tween.New(float32(self.radius)/float32(self.height), 1.0, float32(self.endSpeed), ease.QuadIn)
		}
		return tween.New(float32(self.radius)/float32(self.height), 0.0, float32(self.endSpeed), ease.QuadOut)
	}
	// Spotlight type
	if self.endType == CinematicTransitionIn {
		return tween.New(float32(self.radius), float32(self.spotlightMaxValue), float32(self.endSpeed), ease.BackIn)
	}
	return tween.New(float32(self.radius), 0.0, float32(self.endSpeed), ease.BackOut)
}

// setupLastStep configures the end phase tween.
func (self *CinematicOverlay) setupLastStep() {
	self.tween.Remove(0)
	self.tween.Add(self.createEndTween())
	self.tween.Reset()
}

func (self *CinematicOverlay) drawLetterbox(width, height int, drawOptions *ebiten.DrawTrianglesOptions) {
	self.placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	middle := float64(height) / 2
	halfScalingSize := (float64(height) * self.transitionValue) / 2

	p0 := ebimath.V(0, middle-halfScalingSize)
	p1 := ebimath.V(float64(width), middle-halfScalingSize)
	p2 := ebimath.V(float64(width), middle+halfScalingSize)
	p3 := ebimath.V(0, middle+halfScalingSize)

	placeholderVertices := []ebiten.Vertex{
		{DstX: float32(p0.X), DstY: float32(p0.Y), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(p1.X), DstY: float32(p1.Y), SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(p2.X), DstY: float32(p2.Y), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(p3.X), DstY: float32(p3.Y), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}

	overlayIndices := []uint16{0, 1, 2, 0, 2, 3}
	self.renderTarget.DrawTriangles(placeholderVertices, overlayIndices, self.placeholder, drawOptions)
}

func (self *CinematicOverlay) drawSpotlight(drawOptions *ebiten.DrawTrianglesOptions) {
	const segments = 64
	centerX := self.offset.X
	centerY := self.offset.Y
	radius := self.transitionValue

	spotlightVertices[0] = ebiten.Vertex{
		DstX:   float32(centerX),
		DstY:   float32(centerY),
		SrcX:   0.5,
		SrcY:   0.5,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	}

	for i := 0; i <= segments; i++ {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		spotlightVertices[i+1] = ebiten.Vertex{
			DstX:   float32(x),
			DstY:   float32(y),
			SrcX:   0.5,
			SrcY:   0.5,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		}
	}

	self.renderTarget.DrawTriangles(spotlightVertices, spotlightIndices, self.placeholder, drawOptions)
}
