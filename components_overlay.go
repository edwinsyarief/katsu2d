package katsu2d

import "image/color"

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
	CurrentFade float64
	Finished    bool
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
	Offset                                                 Vector
	OverlayColor                                           color.RGBA
	OverlayOpacity                                         float64
}
