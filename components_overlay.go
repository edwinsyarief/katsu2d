package katsu2d

import (
	"image/color"
	"sync"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SpotlightOvershootFactor = 0.25
)

type FadeType int

const (
	FadeTypeOut FadeType = iota
	FadeTypeIn
)

type FadeOverlayComponent struct {
	FadeType    FadeType
	FadeColor   color.RGBA
	Overlay     *ebiten.Image
	Tween       *tween.Tween
	CurrentFade float32
	Finished    bool
	Callback    func()
}

func NewFadeOverlayComponent(fadeType FadeType, fadeColor color.RGBA, duration float64, callback func()) *FadeOverlayComponent {
	begin, end := float32(0.0), float32(1.0)
	easing := ease.CubicInOut
	if fadeType == FadeTypeIn {
		begin, end = 1.0, 0.0
		easing = ease.QuadInOut
	}
	overlay := ebiten.NewImage(1, 1)
	overlay.Fill(fadeColor)
	return &FadeOverlayComponent{
		FadeType:  fadeType,
		FadeColor: fadeColor,
		Overlay:   overlay,
		Tween:     tween.New(begin, end, float32(duration), easing),
		Callback:  callback,
	}
}

func (self *FadeOverlayComponent) SetDelay(delay float64) {
	self.Tween.SetDelay(float32(delay))
}

type CinematicType int

const (
	CinematicMovie CinematicType = iota
	CinematicSpotlight
)

type CinematicOverlayType int

const (
	CinematicTransitionIn CinematicOverlayType = iota
	CinematicTransitionOut
)

type CinematicOverlayComponent struct {
	CinematicType                                          CinematicType
	StartType, EndType                                     CinematicOverlayType
	StartFade, EndFade                                     bool
	AutoFinish, CinematicFinished, DelayFinished, Finished bool
	Radius, StartSpeed, EndSpeed, CinematicDelay           float64
	CinematicAlphaValue                                    float64
	TransitionValue                                        float64
	Width, Height                                          int
	SpotlightMaxValue                                      float64
	Offset                                                 ebimath.Vector
	OverlayColor                                           color.RGBA
	OverlayOpacity                                         float64
	Overlay, Placeholder                                   *ebiten.Image
	RenderTarget                                           *ebiten.Image
	Tween                                                  *tween.Sequence
	Delayer                                                *managers.DelayManager
	lastStepOnce                                           sync.Once
	Callback                                               func()
}

func NewCinematicOverlayComponent(width, height int, col color.RGBA, opacity float64, cinematicType CinematicType,
	startType, endType CinematicOverlayType, startFade, endFade, autoFinish bool,
	cinematicDelay, radius, startSpeed, endSpeed float64, offset ebimath.Vector, callback func()) *CinematicOverlayComponent {
	result := &CinematicOverlayComponent{
		OverlayColor:   col,
		OverlayOpacity: opacity,
		Width:          width,
		Height:         height,
		CinematicType:  cinematicType,
		StartType:      startType,
		EndType:        endType,
		StartFade:      startFade,
		EndFade:        endFade,
		AutoFinish:     autoFinish,
		CinematicDelay: cinematicDelay,
		Radius:         radius,
		StartSpeed:     startSpeed,
		EndSpeed:       endSpeed,
		Offset:         offset,
		Delayer:        managers.NewDelayManager(),
		Callback:       callback,
	}
	result.Delayer.Add("cinematic_delay", cinematicDelay, func() {
		result.DelayFinished = true
	})
	result.RenderTarget = ebiten.NewImage(width, height)
	result.Overlay = ebiten.NewImage(1, 1)
	result.Placeholder = ebiten.NewImage(1, 1)
	result.Overlay.Fill(result.OverlayColor)
	result.Placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	result.CinematicAlphaValue = 1.0
	if result.CinematicType == CinematicSpotlight {
		result.SpotlightMaxValue = float64(height) * (1 + SpotlightOvershootFactor)
	}
	result.Tween = result.createStartTween()
	return result
}

func (self *CinematicOverlayComponent) createStartTween() *tween.Sequence {
	if self.CinematicType == CinematicMovie {
		if self.StartType == CinematicTransitionIn {
			self.TransitionValue = 0.0
			maxValue := self.Radius / float64(self.Height)
			return tween.NewSequence(
				tween.New(0.0, float32(maxValue), float32(self.StartSpeed), ease.QuadInOut),
			)
		}
		self.TransitionValue = 1.0
		return tween.NewSequence(
			tween.New(1.0, float32(self.Radius/float64(self.Height)), float32(self.StartSpeed), ease.QuadInOut),
		)
	}
	if self.StartType == CinematicTransitionIn {
		self.TransitionValue = 0.0
		return tween.NewSequence(
			tween.New(0.0, float32(self.Radius), float32(self.StartSpeed), ease.QuadInOut),
		)
	}
	self.TransitionValue = self.SpotlightMaxValue
	return tween.NewSequence(
		tween.New(float32(self.SpotlightMaxValue), float32(self.Radius), float32(self.StartSpeed), ease.QuadInOut),
	)
}

func (self *CinematicOverlayComponent) createEndTween() *tween.Tween {
	if self.CinematicType == CinematicMovie {
		if self.EndType == CinematicTransitionIn {
			return tween.New(float32(self.Radius)/float32(self.Height), 1.0, float32(self.EndSpeed), ease.QuadIn)
		}
		return tween.New(float32(self.Radius)/float32(self.Height), 0.0, float32(self.EndSpeed), ease.QuadOut)
	}
	if self.EndType == CinematicTransitionIn {
		return tween.New(float32(self.Radius), float32(self.SpotlightMaxValue), float32(self.EndSpeed), ease.BackIn)
	}
	return tween.New(float32(self.Radius), 0.0, float32(self.EndSpeed), ease.BackOut)
}

func (self *CinematicOverlayComponent) setupLastStep() {
	self.Tween.Remove(0)
	self.Tween.Add(self.createEndTween())
	self.Tween.Reset()
}

func (self *CinematicOverlayComponent) EndCinematic() {
	if self.AutoFinish || !self.CinematicFinished {
		return
	}
	self.Delayer.Activate("cinematic_delay")
}
