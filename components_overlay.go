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

func NewFadeOverlayComponent(
	fadeType FadeType, fadeColor color.RGBA, duration float64, callback func()) *FadeOverlayComponent {
	res := &FadeOverlayComponent{}

	begin, end := float32(0.0), float32(1.0)
	easing := ease.CubicInOut
	if fadeType == FadeTypeIn {
		begin, end = 1.0, 0.0
		easing = ease.QuadInOut
	}
	overlay := ebiten.NewImage(1, 1)
	overlay.Fill(fadeColor)

	res.FadeType = fadeType
	res.FadeColor = fadeColor
	res.Overlay = overlay
	res.Tween = tween.New(begin, end, float32(duration), easing)
	res.Callback = callback

	return res
}

func (self *FadeOverlayComponent) SetDelay(delay float64) *FadeOverlayComponent {
	self.Tween.SetDelay(float32(delay))
	return self
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
	res := &CinematicOverlayComponent{}

	res.OverlayColor = col
	res.OverlayOpacity = opacity
	res.Width = width
	res.Height = height
	res.CinematicType = cinematicType
	res.StartType = startType
	res.EndType = endType
	res.StartFade = startFade
	res.EndFade = endFade
	res.AutoFinish = autoFinish
	res.CinematicDelay = cinematicDelay
	res.Radius = radius
	res.StartSpeed = startSpeed
	res.EndSpeed = endSpeed
	res.Offset = offset
	res.Delayer = managers.NewDelayManager()
	res.Callback = callback

	res.Delayer.Add("cinematic_delay", cinematicDelay, func() {
		res.DelayFinished = true
	})
	res.RenderTarget = ebiten.NewImage(width, height)
	res.Overlay = ebiten.NewImage(1, 1)
	res.Placeholder = ebiten.NewImage(1, 1)
	res.Overlay.Fill(res.OverlayColor)
	res.Placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	res.CinematicAlphaValue = 1.0
	if res.CinematicType == CinematicSpotlight {
		res.SpotlightMaxValue = float64(height) * (1 + SpotlightOvershootFactor)
	}
	res.Tween = res.createStartTween()

	return res
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
