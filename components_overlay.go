package katsu2d

import (
	"image/color"

	"github.com/edwinsyarief/teishoku"
)

const (
	// SpotlightOvershootFactor scales the maximum spotlight radius beyond the viewport height.
	SpotlightOvershootFactor = 0.25
)

// FadeType defines the direction of a screen fade overlay.
type FadeType int

const (
	// FadeTypeOut fades from transparent to the target fade color.
	FadeTypeOut FadeType = iota
	// FadeTypeIn fades from the target fade color to transparent and auto-removes when finished.
	FadeTypeIn
)

// FadeOverlayComponent stores state for a full-screen color fade transition.
type FadeOverlayComponent struct {
	FadeType    FadeType
	FadeColor   color.RGBA
	CurrentFade float64
	Finished    bool
	Callback    func()
}

// SpawnFadeOverlay creates a fade overlay entity with FadeOverlayComponent and TweenComponent.
func SpawnFadeOverlay(w *teishoku.World, fadeType FadeType, col color.RGBA, duration, delay float64, callback func()) teishoku.Entity {
	if w == nil {
		return teishoku.Entity{}
	}

	start, end := 0.0, 1.0
	easing := CubicInOut
	if fadeType == FadeTypeIn {
		start, end = 1.0, 0.0
		easing = QuadInOut
	}

	fade := FadeOverlayComponent{
		FadeType:    fadeType,
		FadeColor:   col,
		CurrentFade: start,
		Callback:    callback,
	}
	tw := TweenComponent{
		Delay:    delay,
		Start:    start,
		End:      end,
		Duration: duration,
		EaseType: easing,
		Current:  start,
	}

	return teishoku.NewBuilder2[FadeOverlayComponent, TweenComponent](w).NewEntityWith(fade, tw)
}

// CinematicType selects the cutout geometry used by a cinematic overlay.
type CinematicType int

const (
	// CinematicMovie renders horizontal letterbox bars with a centered viewport cutout.
	CinematicMovie CinematicType = iota
	// CinematicSpotlight renders a circular spotlight cutout around Offset.
	CinematicSpotlight
)

// CinematicOverlayType defines whether a transition phase opens inward or closes outward.
type CinematicOverlayType int

const (
	// CinematicTransitionIn transitions toward an open/cleared screen.
	CinematicTransitionIn CinematicOverlayType = iota
	// CinematicTransitionOut transitions toward a closed/covered screen.
	CinematicTransitionOut
)

// CinematicPhase tracks the lifecycle stage of a CinematicOverlayComponent.
type CinematicPhase int

const (
	// CinematicPhaseStart is the initial transition phase.
	CinematicPhaseStart CinematicPhase = iota
	// CinematicPhaseHold is the idle/delay phase between start and end transitions.
	CinematicPhaseHold
	// CinematicPhaseEnd is the closing transition phase.
	CinematicPhaseEnd
	// CinematicPhaseDone indicates all cinematic phases have finished.
	CinematicPhaseDone
)

// CinematicOverlayComponent stores state for letterbox and spotlight cinematic transitions.
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
	Offset                                                 Vector
	OverlayColor                                           color.RGBA
	OverlayOpacity                                         float64
	Phase                                                  CinematicPhase
	Callback                                               func()
}

// SpawnCinematicOverlay creates a cinematic overlay entity configured with
// CinematicOverlayComponent, TweenComponent, and TimerComponent.
func SpawnCinematicOverlay(
	w *teishoku.World,
	width, height int,
	col color.RGBA,
	opacity float64,
	cinematicType CinematicType,
	startType, endType CinematicOverlayType,
	startFade, endFade, autoFinish bool,
	cinematicDelay, radius, startSpeed, endSpeed float64,
	offset Vector,
	callback func(),
) teishoku.Entity {
	if w == nil {
		return teishoku.Entity{}
	}

	var spotlightMax float64
	if cinematicType == CinematicSpotlight {
		spotlightMax = float64(height) * (1 + SpotlightOvershootFactor)
	}

	var startVal, endVal float64
	if cinematicType == CinematicMovie {
		var targetRatio float64
		if height > 0 {
			targetRatio = radius / float64(height)
		}
		if startType == CinematicTransitionIn {
			startVal = 0.0
		} else {
			startVal = 1.0
		}
		endVal = targetRatio
	} else {
		if startType == CinematicTransitionIn {
			startVal = 0.0
		} else {
			startVal = spotlightMax
		}
		endVal = radius
	}

	alphaVal := 1.0
	if startFade {
		if cinematicType == CinematicMovie {
			alphaVal = CalculateProgressRatio(0.0, 1.0, startVal)
		} else if spotlightMax > 0 {
			alphaVal = CalculateProgressRatio(0.0, 1.0, startVal/spotlightMax)
		}
	}

	cin := CinematicOverlayComponent{
		CinematicType:       cinematicType,
		StartType:           startType,
		EndType:             endType,
		StartFade:           startFade,
		EndFade:             endFade,
		AutoFinish:          autoFinish,
		Radius:              radius,
		StartSpeed:          startSpeed,
		EndSpeed:            endSpeed,
		CinematicDelay:      cinematicDelay,
		CinematicAlphaValue: alphaVal,
		TransitionValue:     startVal,
		Width:               width,
		Height:              height,
		SpotlightMaxValue:   spotlightMax,
		Offset:              offset,
		OverlayColor:        col,
		OverlayOpacity:      opacity,
		Phase:               CinematicPhaseStart,
		Callback:            callback,
	}

	tw := TweenComponent{
		ID:       "cinematic_start",
		Start:    startVal,
		End:      endVal,
		Duration: startSpeed,
		EaseType: QuadInOut,
		Current:  startVal,
	}

	timer := TimerComponent{
		ID:    "cinematic_delay",
		Time:  cinematicDelay,
		State: TimerStateIdle,
	}

	return teishoku.NewBuilder3[CinematicOverlayComponent, TweenComponent, TimerComponent](w).NewEntityWith(cin, tw, timer)
}

// EndCinematic triggers the delay timer to close a manual cinematic overlay
// when AutoFinish is false, the start phase has finished, and the delay has not yet finished.
func EndCinematic(w *teishoku.World, e teishoku.Entity) {
	if w == nil || !w.IsValid(e) {
		return
	}
	cin := teishoku.GetComponent[CinematicOverlayComponent](w, e)
	timer := teishoku.GetComponent[TimerComponent](w, e)
	if cin == nil || timer == nil {
		return
	}
	if cin.AutoFinish || !cin.CinematicFinished || cin.DelayFinished || timer.State == TimerStateActive {
		return
	}
	timer.Time = cin.CinematicDelay
	timer.State = TimerStateActive
}
