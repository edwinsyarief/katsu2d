package katsu2d

import (
	"image"
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// TransitionKind identifies the visual style of a scene transition.
type TransitionKind uint8

const (
	// TransitionKindFade fades the screen out to a solid color and back in.
	TransitionKindFade TransitionKind = iota
	// TransitionKindSpotlight closes a circular spotlight cutout onto a point and reopens it.
	TransitionKindSpotlight
	// TransitionKindLetterbox closes top and bottom letterbox bars toward the center and reopens them.
	TransitionKindLetterbox
	// TransitionKindWipe sweeps a solid color curtain across the screen in a direction.
	TransitionKindWipe
	// TransitionKindCrossFade blends a snapshot of the outgoing scene into the live incoming scene.
	TransitionKindCrossFade
	// TransitionKindSlide slides a snapshot of the outgoing scene off-screen to reveal the incoming scene.
	TransitionKindSlide
)

// TransitionDirection specifies the movement direction for directional transitions
// such as Wipe and Slide.
type TransitionDirection uint8

const (
	// TransitionDirLeft moves from right to left.
	TransitionDirLeft TransitionDirection = iota
	// TransitionDirRight moves from left to right.
	TransitionDirRight
	// TransitionDirUp moves from bottom to top.
	TransitionDirUp
	// TransitionDirDown moves from top to bottom.
	TransitionDirDown

	// TransitionWipeLeft is an alias for TransitionDirLeft.
	TransitionWipeLeft = TransitionDirLeft
	// TransitionWipeRight is an alias for TransitionDirRight.
	TransitionWipeRight = TransitionDirRight
	// TransitionWipeUp is an alias for TransitionDirUp.
	TransitionWipeUp = TransitionDirUp
	// TransitionWipeDown is an alias for TransitionDirDown.
	TransitionWipeDown = TransitionDirDown

	// TransitionSlideLeft is an alias for TransitionDirLeft.
	TransitionSlideLeft = TransitionDirLeft
	// TransitionSlideRight is an alias for TransitionDirRight.
	TransitionSlideRight = TransitionDirRight
	// TransitionSlideUp is an alias for TransitionDirUp.
	TransitionSlideUp = TransitionDirUp
	// TransitionSlideDown is an alias for TransitionDirDown.
	TransitionSlideDown = TransitionDirDown
)

// TransitionConfig configures a scene transition's visual style, timing, easing, and callbacks.
type TransitionConfig struct {
	Kind         TransitionKind
	Direction    TransitionDirection
	Color        color.Color
	Duration     float64
	HoldDuration float64
	OutEase      EaseType
	InEase       EaseType
	CenterX      float64
	CenterY      float64
	BlockInput   bool
	UnscaledTime bool
	BelowOverlay bool
	OnMidpoint   func()
	OnComplete   func()
}

// TransitionOption configures optional parameters on a TransitionConfig.
type TransitionOption func(*TransitionConfig)

// WithTransitionEase sets the easing functions used for the out and in phases.
// Single-phase transitions (CrossFade, Slide) use outEase.
func WithTransitionEase(outEase, inEase EaseType) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.OutEase = outEase
		cfg.InEase = inEase
	}
}

// WithTransitionHold sets a pause duration in seconds at the midpoint of a
// two-phase transition while the screen is fully covered.
func WithTransitionHold(seconds float64) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.HoldDuration = max(0.0, seconds)
	}
}

// WithTransitionCenter sets the normalized [0, 1] viewport focal point used by
// Spotlight and Letterbox transitions. Defaults to (0.5, 0.5).
func WithTransitionCenter(normX, normY float64) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.CenterX = normX
		cfg.CenterY = normY
	}
}

// WithTransitionBlockInput configures whether InputSystem actions and mouse wheel
// deltas are zeroed while the transition is active. Defaults to true.
func WithTransitionBlockInput(block bool) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.BlockInput = block
	}
}

// WithTransitionUnscaledTime configures whether the transition advances using
// unscaled frame time instead of Engine.TimeScale(). Defaults to true.
func WithTransitionUnscaledTime(unscaled bool) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.UnscaledTime = unscaled
	}
}

// WithTransitionOnMidpoint registers a callback invoked immediately after the
// scene switch occurs at the transition midpoint (or initial capture for single-phase transitions).
func WithTransitionOnMidpoint(fn func()) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.OnMidpoint = fn
	}
}

// WithTransitionOnComplete registers a callback invoked once the transition finishes.
func WithTransitionOnComplete(fn func()) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.OnComplete = fn
	}
}

// WithTransitionBelowOverlay configures whether the transition renders beneath
// Engine overlay draw systems instead of on top of them. Defaults to false.
func WithTransitionBelowOverlay(below bool) TransitionOption {
	return func(cfg *TransitionConfig) {
		cfg.BelowOverlay = below
	}
}

func defaultTransitionConfig(kind TransitionKind, duration float64, col color.Color, opts ...TransitionOption) TransitionConfig {
	if col == nil {
		col = color.Black
	}
	cfg := TransitionConfig{
		Kind:         kind,
		Color:        col,
		Duration:     max(0.0, duration),
		OutEase:      QuadInOut,
		InEase:       QuadInOut,
		CenterX:      0.5,
		CenterY:      0.5,
		BlockInput:   true,
		UnscaledTime: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// FadeTransition creates a two-phase full-screen color fade transition.
func FadeTransition(duration float64, col color.Color, opts ...TransitionOption) TransitionConfig {
	return defaultTransitionConfig(TransitionKindFade, duration, col, opts...)
}

// SpotlightTransition creates a two-phase circular spotlight cutout transition.
func SpotlightTransition(duration float64, col color.Color, opts ...TransitionOption) TransitionConfig {
	return defaultTransitionConfig(TransitionKindSpotlight, duration, col, opts...)
}

// LetterboxTransition creates a two-phase horizontal letterbox shutter transition.
func LetterboxTransition(duration float64, col color.Color, opts ...TransitionOption) TransitionConfig {
	return defaultTransitionConfig(TransitionKindLetterbox, duration, col, opts...)
}

// WipeTransition creates a two-phase directional color curtain wipe transition.
func WipeTransition(duration float64, dir TransitionDirection, col color.Color, opts ...TransitionOption) TransitionConfig {
	cfg := defaultTransitionConfig(TransitionKindWipe, duration, col, opts...)
	cfg.Direction = dir
	return cfg
}

// CrossFadeTransition creates a single-phase cross-fade transition between the
// outgoing snapshot and the live incoming scene.
func CrossFadeTransition(duration float64, opts ...TransitionOption) TransitionConfig {
	return defaultTransitionConfig(TransitionKindCrossFade, duration, color.Black, opts...)
}

// SlideTransition creates a single-phase slide transition that moves the outgoing
// snapshot off-screen in the given direction to reveal the live incoming scene.
func SlideTransition(duration float64, dir TransitionDirection, opts ...TransitionOption) TransitionConfig {
	cfg := defaultTransitionConfig(TransitionKindSlide, duration, color.Black, opts...)
	cfg.Direction = dir
	return cfg
}

type inputBlockState struct {
	blocked bool
}

func setWorldInputBlocked(w *teishoku.World, blocked bool) {
	if w == nil {
		return
	}
	res := w.Resources()
	if ok, _ := teishoku.HasResource[inputBlockState](res); !ok {
		if !blocked {
			return
		}
		teishoku.AddResource(res, &inputBlockState{blocked: true})
		return
	}
	state, _ := teishoku.GetResource[inputBlockState](res)
	if state != nil {
		state.blocked = blocked
	}
}

type transitionPhase uint8

const (
	transitionPhaseIdle transitionPhase = iota
	transitionPhaseOut
	transitionPhaseHold
	transitionPhaseIn
	transitionPhaseSingle
)

const transitionSpotlightSegments = 64

type sceneTransitionController struct {
	phase        transitionPhase
	target       string
	cfg          TransitionConfig
	elapsed      float64
	outDuration  float64
	inDuration   float64
	coverage     float64
	colR         float32
	colG         float32
	colB         float32
	colA         float32
	needsCapture bool
	switched     bool

	// Pre-allocated geometry and draw option buffers (zero hot-path allocations).
	quadVerts      [4]ebiten.Vertex
	quadInds       [6]uint16
	letterboxVerts [8]ebiten.Vertex
	letterboxInds  [12]uint16
	spotVerts      []ebiten.Vertex
	spotInds       []uint16
	clearOpts      ebiten.DrawTrianglesOptions
	copyOpts       ebiten.DrawTrianglesOptions
	drawImgOpts    ebiten.DrawImageOptions

	// Lazily allocated reusable offscreen targets (released via dispose / Deallocate).
	maskBuf    *ebiten.Image
	captureBuf *ebiten.Image
}

func newSceneTransitionController() *sceneTransitionController {
	c := &sceneTransitionController{
		quadInds:       [6]uint16{0, 1, 2, 0, 2, 3},
		letterboxInds:  [12]uint16{0, 1, 2, 0, 2, 3, 4, 5, 6, 4, 6, 7},
		spotVerts:      make([]ebiten.Vertex, transitionSpotlightSegments+2),
		spotInds:       make([]uint16, transitionSpotlightSegments*3),
		clearOpts:      ebiten.DrawTrianglesOptions{Blend: ebiten.BlendClear},
		copyOpts:       ebiten.DrawTrianglesOptions{Blend: ebiten.BlendCopy},
	}
	for i := range transitionSpotlightSegments {
		c.spotInds[i*3] = 0
		c.spotInds[i*3+1] = uint16(i + 1)
		c.spotInds[i*3+2] = uint16(i + 2)
	}
	return c
}

func (c *sceneTransitionController) isTransitioning() bool {
	return c.phase != transitionPhaseIdle
}

func (c *sceneTransitionController) isInputBlocked() bool {
	return c.phase != transitionPhaseIdle && c.cfg.BlockInput
}

func (c *sceneTransitionController) belowOverlay() bool {
	return c.cfg.BelowOverlay
}

func (c *sceneTransitionController) usesUnscaledTime() bool {
	return c.cfg.UnscaledTime
}

func (c *sceneTransitionController) isSinglePhase() bool {
	return c.cfg.Kind == TransitionKindCrossFade || c.cfg.Kind == TransitionKindSlide
}

func (c *sceneTransitionController) start(sm *SceneManager, target string, cfg TransitionConfig) bool {
	if c.isTransitioning() || sm == nil {
		return false
	}
	if _, ok := sm.scenes[target]; !ok {
		return false
	}

	cfg.Duration = max(0.0, cfg.Duration)
	cfg.HoldDuration = max(0.0, cfg.HoldDuration)
	if cfg.Color == nil {
		cfg.Color = color.Black
	}

	c.target = target
	c.cfg = cfg
	c.colR, c.colG, c.colB, c.colA = colorToPremulFloat32(cfg.Color)
	c.elapsed = 0
	c.switched = false
	c.needsCapture = false

	if c.isSinglePhase() {
		if cfg.Duration <= 0 {
			c.executeMidpoint(sm)
			onComplete := c.cfg.OnComplete
			c.resetState()
			if onComplete != nil {
				onComplete()
			}
			return true
		}
		c.phase = transitionPhaseSingle
		c.needsCapture = true
		c.coverage = 0
		return true
	}

	half := cfg.Duration * 0.5
	c.outDuration = half
	c.inDuration = half
	if half > 0 {
		c.phase = transitionPhaseOut
		c.coverage = 0
		return true
	}
	if cfg.HoldDuration > 0 {
		c.phase = transitionPhaseHold
		c.coverage = 1.0
		return true
	}

	c.executeMidpoint(sm)
	onComplete := c.cfg.OnComplete
	c.resetState()
	if onComplete != nil {
		onComplete()
	}
	return true
}

func (c *sceneTransitionController) cancel() {
	if !c.isTransitioning() {
		return
	}
	c.resetState()
}

func (c *sceneTransitionController) resetState() {
	c.phase = transitionPhaseIdle
	c.target = ""
	c.elapsed = 0
	c.outDuration = 0
	c.inDuration = 0
	c.coverage = 0
	c.needsCapture = false
	c.switched = false
	c.cfg.OnMidpoint = nil
	c.cfg.OnComplete = nil
}

func (c *sceneTransitionController) executeMidpoint(sm *SceneManager) {
	if c.switched {
		return
	}
	c.switched = true
	sm.switchToInternal(c.target)
	sm.syncInputBlockState()
	if fn := c.cfg.OnMidpoint; fn != nil {
		c.cfg.OnMidpoint = nil
		fn()
	}
}

func (c *sceneTransitionController) complete() {
	onComplete := c.cfg.OnComplete
	c.resetState()
	if onComplete != nil {
		onComplete()
	}
}

func (c *sceneTransitionController) update(sm *SceneManager, dt float64) {
	if c.phase == transitionPhaseIdle {
		return
	}
	if dt < 0 {
		dt = 0
	}

	const timeEps = 1e-9
	rem := dt
	for c.phase != transitionPhaseIdle {
		switch c.phase {
		case transitionPhaseOut:
			if c.outDuration <= 0 {
				c.coverage = 1.0
				if c.cfg.HoldDuration > 0 {
					c.phase = transitionPhaseHold
					c.elapsed = 0
					continue
				}
				c.executeMidpoint(sm)
				if c.phase == transitionPhaseIdle {
					return
				}
				c.phase = transitionPhaseIn
				c.elapsed = 0
				continue
			}

			need := c.outDuration - c.elapsed
			if rem == 0 || rem+timeEps < need {
				c.elapsed += rem
				t := c.elapsed / c.outDuration
				c.coverage = Clamp(evalTransitionEaseOut(c.cfg.OutEase, t), 0.0, 1.0)
				return
			}
			rem = max(0.0, rem-need)
			c.elapsed = c.outDuration
			c.coverage = 1.0
			if c.cfg.HoldDuration > 0 {
				c.phase = transitionPhaseHold
				c.elapsed = 0
				continue
			}
			c.executeMidpoint(sm)
			if c.phase == transitionPhaseIdle {
				return
			}
			c.phase = transitionPhaseIn
			c.elapsed = 0

		case transitionPhaseHold:
			c.coverage = 1.0
			need := c.cfg.HoldDuration - c.elapsed
			if c.cfg.HoldDuration > 0 && (rem == 0 || rem+timeEps < need) {
				c.elapsed += rem
				return
			}
			rem = max(0.0, rem-need)
			c.executeMidpoint(sm)
			if c.phase == transitionPhaseIdle {
				return
			}
			c.phase = transitionPhaseIn
			c.elapsed = 0

		case transitionPhaseIn:
			if c.inDuration <= 0 {
				c.coverage = 0.0
				c.complete()
				return
			}
			need := c.inDuration - c.elapsed
			if rem == 0 || rem+timeEps < need {
				c.elapsed += rem
				t := c.elapsed / c.inDuration
				c.coverage = Clamp(evalTransitionEaseIn(c.cfg.InEase, t), 0.0, 1.0)
				return
			}
			c.coverage = 0.0
			c.complete()
			return

		case transitionPhaseSingle:
			if c.needsCapture {
				return
			}
			if c.cfg.Duration <= 0 {
				c.coverage = 1.0
				c.complete()
				return
			}
			need := c.cfg.Duration - c.elapsed
			if rem == 0 || rem+timeEps < need {
				c.elapsed += rem
				t := c.elapsed / c.cfg.Duration
				c.coverage = Clamp(evalTransitionEaseOut(c.cfg.OutEase, t), 0.0, 1.0)
				return
			}
			c.coverage = 1.0
			c.complete()
			return
		}
	}
}

func (c *sceneTransitionController) ensureMaskBuf(w, h int) *ebiten.Image {
	c.maskBuf = ensureOffscreenTarget(c.maskBuf, w, h)
	return c.maskBuf
}

func (c *sceneTransitionController) ensureCaptureBuf(w, h int) *ebiten.Image {
	c.captureBuf = ensureOffscreenTarget(c.captureBuf, w, h)
	return c.captureBuf
}

func (c *sceneTransitionController) dispose() {
	if c.maskBuf != nil {
		c.maskBuf.Deallocate()
		c.maskBuf = nil
	}
	if c.captureBuf != nil {
		c.captureBuf.Deallocate()
		c.captureBuf = nil
	}
}

func ensureOffscreenTarget(img *ebiten.Image, w, h int) *ebiten.Image {
	if w <= 0 || h <= 0 {
		return img
	}
	if img != nil {
		b := img.Bounds()
		if b.Dx() == w && b.Dy() == h {
			return img
		}
		img.Deallocate()
	}
	return ebiten.NewImage(w, h)
}

func (c *sceneTransitionController) fillOpaqueBlack(dst *ebiten.Image) {
	if dst == nil {
		return
	}
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	whiteImg := defaultWhiteImage()
	setQuadVertices(&c.quadVerts, 0, 0, float32(w), float32(h), whiteImg.Bounds(), 0, 0, 0, 1)
	dst.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, &c.copyOpts)
}

func (c *sceneTransitionController) draw(screen *ebiten.Image) {
	if c.phase == transitionPhaseIdle || screen == nil {
		return
	}
	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	switch c.cfg.Kind {
	case TransitionKindFade:
		c.drawFade(screen, w, h)
	case TransitionKindSpotlight:
		c.drawSpotlight(screen, w, h)
	case TransitionKindLetterbox:
		c.drawLetterbox(screen, w, h)
	case TransitionKindWipe:
		c.drawWipe(screen, w, h)
	case TransitionKindCrossFade:
		c.drawCrossFade(screen)
	case TransitionKindSlide:
		c.drawSlide(screen, w, h)
	}
}

func (c *sceneTransitionController) drawFade(screen *ebiten.Image, w, h int) {
	alpha := float32(Clamp(c.coverage, 0.0, 1.0))
	if alpha <= 0 {
		return
	}
	whiteImg := defaultWhiteImage()
	setQuadVertices(
		&c.quadVerts,
		0, 0, float32(w), float32(h),
		whiteImg.Bounds(),
		c.colR*alpha, c.colG*alpha, c.colB*alpha, c.colA*alpha,
	)
	screen.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, nil)
}

func (c *sceneTransitionController) drawLetterbox(screen *ebiten.Image, w, h int) {
	cov := Clamp(c.coverage, 0.0, 1.0)
	if cov <= 0 {
		return
	}
	whiteImg := defaultWhiteImage()
	srcBounds := whiteImg.Bounds()
	if cov >= 1.0 {
		setQuadVertices(&c.quadVerts, 0, 0, float32(w), float32(h), srcBounds, c.colR, c.colG, c.colB, c.colA)
		screen.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, nil)
		return
	}

	centerY := float64(h) * Clamp(c.cfg.CenterY, 0.0, 1.0)
	topY := AdjustDestinationPixel(float32(centerY * cov))
	botY := AdjustDestinationPixel(float32(float64(h) - (float64(h)-centerY)*cov))
	fw := float32(w)
	fh := float32(h)

	sx0 := float32(srcBounds.Min.X)
	sy0 := float32(srcBounds.Min.Y)
	sx1 := float32(srcBounds.Max.X)
	sy1 := float32(srcBounds.Max.Y)

	// Top bar
	c.letterboxVerts[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: sx0, SrcY: sy0, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[1] = ebiten.Vertex{DstX: fw, DstY: 0, SrcX: sx1, SrcY: sy0, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[2] = ebiten.Vertex{DstX: fw, DstY: topY, SrcX: sx1, SrcY: sy1, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[3] = ebiten.Vertex{DstX: 0, DstY: topY, SrcX: sx0, SrcY: sy1, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}

	// Bottom bar
	c.letterboxVerts[4] = ebiten.Vertex{DstX: 0, DstY: botY, SrcX: sx0, SrcY: sy0, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[5] = ebiten.Vertex{DstX: fw, DstY: botY, SrcX: sx1, SrcY: sy0, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[6] = ebiten.Vertex{DstX: fw, DstY: fh, SrcX: sx1, SrcY: sy1, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}
	c.letterboxVerts[7] = ebiten.Vertex{DstX: 0, DstY: fh, SrcX: sx0, SrcY: sy1, ColorR: c.colR, ColorG: c.colG, ColorB: c.colB, ColorA: c.colA}

	screen.DrawTriangles(c.letterboxVerts[:], c.letterboxInds[:], whiteImg, nil)
}

func (c *sceneTransitionController) drawWipe(screen *ebiten.Image, w, h int) {
	cov := Clamp(c.coverage, 0.0, 1.0)
	if cov <= 0 {
		return
	}

	fw := float64(w)
	fh := float64(h)
	var x0, y0, x1, y1 float32
	isOut := c.phase != transitionPhaseIn

	switch c.cfg.Direction {
	case TransitionDirRight:
		y0, y1 = 0, float32(h)
		if isOut {
			x0 = 0
			x1 = AdjustDestinationPixel(float32(fw * cov))
		} else {
			x0 = AdjustDestinationPixel(float32(fw * (1.0 - cov)))
			x1 = float32(w)
		}
	case TransitionDirLeft:
		y0, y1 = 0, float32(h)
		if isOut {
			x0 = AdjustDestinationPixel(float32(fw * (1.0 - cov)))
			x1 = float32(w)
		} else {
			x0 = 0
			x1 = AdjustDestinationPixel(float32(fw * cov))
		}
	case TransitionDirDown:
		x0, x1 = 0, float32(w)
		if isOut {
			y0 = 0
			y1 = AdjustDestinationPixel(float32(fh * cov))
		} else {
			y0 = AdjustDestinationPixel(float32(fh * (1.0 - cov)))
			y1 = float32(h)
		}
	case TransitionDirUp:
		x0, x1 = 0, float32(w)
		if isOut {
			y0 = AdjustDestinationPixel(float32(fh * (1.0 - cov)))
			y1 = float32(h)
		} else {
			y0 = 0
			y1 = AdjustDestinationPixel(float32(fh * cov))
		}
	}

	if x1 <= x0 || y1 <= y0 {
		return
	}
	whiteImg := defaultWhiteImage()
	setQuadVertices(&c.quadVerts, x0, y0, x1, y1, whiteImg.Bounds(), c.colR, c.colG, c.colB, c.colA)
	screen.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, nil)
}

func (c *sceneTransitionController) drawSpotlight(screen *ebiten.Image, w, h int) {
	cov := Clamp(c.coverage, 0.0, 1.0)
	if cov <= 0 {
		return
	}

	whiteImg := defaultWhiteImage()
	srcBounds := whiteImg.Bounds()
	if cov >= 1.0 {
		setQuadVertices(&c.quadVerts, 0, 0, float32(w), float32(h), srcBounds, c.colR, c.colG, c.colB, c.colA)
		screen.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, nil)
		return
	}

	cx := float64(w) * c.cfg.CenterX
	cy := float64(h) * c.cfg.CenterY
	maxDX := max(math.Abs(cx), math.Abs(float64(w)-cx))
	maxDY := max(math.Abs(cy), math.Abs(float64(h)-cy))
	maxRadius := math.Hypot(maxDX, maxDY) + 2.0
	radius := maxRadius * (1.0 - cov)
	if radius <= 0 {
		setQuadVertices(&c.quadVerts, 0, 0, float32(w), float32(h), srcBounds, c.colR, c.colG, c.colB, c.colA)
		screen.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, nil)
		return
	}

	mask := c.ensureMaskBuf(w, h)
	if mask == nil {
		return
	}

	setQuadVertices(&c.quadVerts, 0, 0, float32(w), float32(h), srcBounds, c.colR, c.colG, c.colB, c.colA)
	mask.DrawTriangles(c.quadVerts[:], c.quadInds[:], whiteImg, &c.copyOpts)

	srcX := float32(srcBounds.Min.X+srcBounds.Max.X) * 0.5
	srcY := float32(srcBounds.Min.Y+srcBounds.Max.Y) * 0.5
	c.spotVerts[0] = ebiten.Vertex{
		DstX:   AdjustDestinationPixel(float32(cx)),
		DstY:   AdjustDestinationPixel(float32(cy)),
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
	}

	const step = (2 * math.Pi) / float64(transitionSpotlightSegments)
	for i := 0; i <= transitionSpotlightSegments; i++ {
		sin, cos := math.Sincos(float64(i) * step)
		c.spotVerts[i+1] = ebiten.Vertex{
			DstX:   AdjustDestinationPixel(float32(cx + radius*cos)),
			DstY:   AdjustDestinationPixel(float32(cy + radius*sin)),
			SrcX:   srcX,
			SrcY:   srcY,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
	}

	mask.DrawTriangles(c.spotVerts, c.spotInds, whiteImg, &c.clearOpts)
	screen.DrawImage(mask, nil)
}

func (c *sceneTransitionController) drawCrossFade(screen *ebiten.Image) {
	if c.captureBuf == nil {
		return
	}
	alpha := float32(1.0 - Clamp(c.coverage, 0.0, 1.0))
	if alpha <= 0 {
		return
	}
	c.drawImgOpts.GeoM.Reset()
	c.drawImgOpts.ColorScale.Reset()
	c.drawImgOpts.ColorScale.ScaleAlpha(alpha)
	screen.DrawImage(c.captureBuf, &c.drawImgOpts)
}

func (c *sceneTransitionController) drawSlide(screen *ebiten.Image, w, h int) {
	if c.captureBuf == nil {
		return
	}
	cov := Clamp(c.coverage, 0.0, 1.0)
	if cov >= 1.0 {
		return
	}
	var dx, dy float64
	switch c.cfg.Direction {
	case TransitionDirLeft:
		dx = -float64(w) * cov
	case TransitionDirRight:
		dx = float64(w) * cov
	case TransitionDirUp:
		dy = -float64(h) * cov
	case TransitionDirDown:
		dy = float64(h) * cov
	}
	c.drawImgOpts.GeoM.Reset()
	c.drawImgOpts.GeoM.Translate(dx, dy)
	c.drawImgOpts.ColorScale.Reset()
	screen.DrawImage(c.captureBuf, &c.drawImgOpts)
}

func setQuadVertices(verts *[4]ebiten.Vertex, x0, y0, x1, y1 float32, srcBounds image.Rectangle, r, g, b, a float32) {
	sx0 := float32(srcBounds.Min.X)
	sy0 := float32(srcBounds.Min.Y)
	sx1 := float32(srcBounds.Max.X)
	sy1 := float32(srcBounds.Max.Y)
	verts[0] = ebiten.Vertex{DstX: x0, DstY: y0, SrcX: sx0, SrcY: sy0, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	verts[1] = ebiten.Vertex{DstX: x1, DstY: y0, SrcX: sx1, SrcY: sy0, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	verts[2] = ebiten.Vertex{DstX: x1, DstY: y1, SrcX: sx1, SrcY: sy1, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	verts[3] = ebiten.Vertex{DstX: x0, DstY: y1, SrcX: sx0, SrcY: sy1, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
}

func colorToPremulFloat32(col color.Color) (r, g, b, a float32) {
	if col == nil {
		return 0, 0, 0, 1
	}
	cr, cg, cb, ca := col.RGBA()
	const inv65535 = float32(1.0 / 65535.0)
	return float32(cr) * inv65535, float32(cg) * inv65535, float32(cb) * inv65535, float32(ca) * inv65535
}

func evalTransitionEaseOut(et EaseType, t float64) float64 {
	return EaseTypes[float64](et)(Clamp(t, 0.0, 1.0), 0.0, 1.0, 1.0)
}

func evalTransitionEaseIn(et EaseType, t float64) float64 {
	return EaseTypes[float64](et)(Clamp(t, 0.0, 1.0), 1.0, -1.0, 1.0)
}
