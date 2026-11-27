package katsu2d

import (
	"image"
	"image/color"
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// FadeOverlaySystem updates and draws full-screen fade overlays.
type FadeOverlaySystem struct {
	filter           *teishoku.Filter2[FadeOverlayComponent, TweenComponent]
	tmHandle         teishoku.ResourceHandle[TextureManager]
	indices          Indices
	vertices         Vertices
	toRemove         []teishoku.Entity
	pendingCallbacks []func()
	initialized      bool
}

// NewFadeOverlaySystem creates a new FadeOverlaySystem with pre-allocated buffers.
func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{
		indices:          Indices{0, 1, 2, 0, 2, 3},
		vertices:         make(Vertices, 4),
		toRemove:         make([]teishoku.Entity, 0, 8),
		pendingCallbacks: make([]func(), 0, 4),
	}
}

// Initialize binds the system's ECS filter and resource handles once.
func (s *FadeOverlaySystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}

	s.filter = teishoku.NewFilter2[FadeOverlayComponent, TweenComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.initialized = true
}

// Update synchronizes fade progress from TweenComponent, fires completion callbacks,
// and removes completed FadeTypeIn entities after iteration.
func (s *FadeOverlaySystem) Update(w *teishoku.World, dt float64) {
	if s.filter == nil {
		return
	}

	s.toRemove = s.toRemove[:0]
	s.pendingCallbacks = s.pendingCallbacks[:0]

	chunks := s.filter.Chunks()
	for chunks.Next() {
		ents := chunks.Entities()
		fades, tweens := chunks.Get()
		for i := range fades {
			fade := &fades[i]
			tween := &tweens[i]

			if fade.FadeType == FadeTypeIn && fade.Finished {
				s.toRemove = append(s.toRemove, ents[i])
				continue
			}

			fade.CurrentFade = tween.Current
			if tween.Finished && !fade.Finished {
				fade.Finished = true
				if fade.Callback != nil {
					s.pendingCallbacks = append(s.pendingCallbacks, fade.Callback)
				}
				if fade.FadeType == FadeTypeIn {
					s.toRemove = append(s.toRemove, ents[i])
				}
			}
		}
	}

	for _, cb := range s.pendingCallbacks {
		cb()
	}
	clear(s.pendingCallbacks)
	s.pendingCallbacks = s.pendingCallbacks[:0]

	if len(s.toRemove) > 0 {
		w.RemoveEntities(s.toRemove)
		s.toRemove = s.toRemove[:0]
	}
}

// Draw renders active fade overlays using the 1x1 white texture from TextureManager.
func (s *FadeOverlaySystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if s.filter == nil || rdr == nil {
		return
	}
	screen := rdr.GetScreen()
	if screen == nil {
		return
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return
	}

	tm := s.tmHandle.Get()
	if tm == nil {
		tm = GetTextureManager(w)
		if tm == nil {
			return
		}
		s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	}
	img := tm.Get(0)
	if img == nil {
		return
	}
	srcBounds := img.Bounds()

	flushed := false
	chunks := s.filter.Chunks()
	for chunks.Next() {
		fades, _ := chunks.Get()
		for i := range fades {
			fade := &fades[i]
			if fade.FadeType == FadeTypeIn && fade.Finished {
				continue
			}
			if !flushed {
				rdr.Flush()
				flushed = true
			}
			updateOverlayVertices(s.vertices, width, height, fade.FadeColor, fade.CurrentFade, srcBounds)
			rdr.AddCustomMeshes(s.vertices, s.indices, img)
		}
	}
}

func updateOverlayVertices(vertices []ebiten.Vertex, width, height int, col color.RGBA, alpha float64, srcBounds image.Rectangle) {
	a := float32(Clamp(alpha, 0.0, 1.0))
	const inv255 = float32(1.0 / 255.0)
	sR := float32(col.R) * inv255 * a
	sG := float32(col.G) * inv255 * a
	sB := float32(col.B) * inv255 * a
	sA := float32(col.A) * inv255 * a
	w := float32(width)
	h := float32(height)
	sx0 := float32(srcBounds.Min.X)
	sy0 := float32(srcBounds.Min.Y)
	sx1 := float32(srcBounds.Max.X)
	sy1 := float32(srcBounds.Max.Y)
	vertices[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: sx0, SrcY: sy0, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[1] = ebiten.Vertex{DstX: w, DstY: 0, SrcX: sx1, SrcY: sy0, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[2] = ebiten.Vertex{DstX: w, DstY: h, SrcX: sx1, SrcY: sy1, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
	vertices[3] = ebiten.Vertex{DstX: 0, DstY: h, SrcX: sx0, SrcY: sy1, ColorR: sR, ColorG: sG, ColorB: sB, ColorA: sA}
}

// CinematicOverlaySystem updates and draws letterbox and spotlight cinematic overlays.
type CinematicOverlaySystem struct {
	filter       *teishoku.Filter3[CinematicOverlayComponent, TweenComponent, TimerComponent]
	builder      *teishoku.Builder3[CinematicOverlayComponent, TweenComponent, TimerComponent]
	tmHandle     teishoku.ResourceHandle[TextureManager]
	indices      []uint16
	vertices     []ebiten.Vertex
	letterboxV   [4]ebiten.Vertex
	spotlightV   []ebiten.Vertex
	spotlightI   []uint16
	spotlightSeg int
	toRemove     []teishoku.Entity
	clearOpts    ebiten.DrawTrianglesOptions
	target       *ebiten.Image
	initialized  bool
}

// NewCinematicOverlaySystem creates a new CinematicOverlaySystem with pre-allocated geometry buffers.
func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	const segs = 64
	sys := &CinematicOverlaySystem{
		indices:      []uint16{0, 1, 2, 0, 2, 3},
		vertices:     make([]ebiten.Vertex, 4),
		spotlightSeg: segs,
		spotlightV:   make([]ebiten.Vertex, segs+2),
		spotlightI:   make([]uint16, segs*3),
		toRemove:     make([]teishoku.Entity, 0, 8),
		clearOpts:    ebiten.DrawTrianglesOptions{Blend: ebiten.BlendClear},
	}
	for i := range segs {
		sys.spotlightI[i*3] = 0
		sys.spotlightI[i*3+1] = uint16(i + 1)
		sys.spotlightI[i*3+2] = uint16(i + 2)
	}
	return sys
}

// Initialize binds the system's ECS filter, builder, resource handle, and event subscriptions once.
func (s *CinematicOverlaySystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}

	s.filter = teishoku.NewFilter3[CinematicOverlayComponent, TweenComponent, TimerComponent](w)
	s.builder = teishoku.NewBuilder3[CinematicOverlayComponent, TweenComponent, TimerComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())

	if info := GetHiResDisplayInfo(w); info != nil && info.Width > 0 && info.Height > 0 {
		s.ensureTarget(info.Width, info.Height)
	}

	Subscribe(w, s.onEngineLayoutChanged)
	Subscribe(w, s.onTimerFinished)
	Subscribe(w, s.onTweenFinished)

	s.initialized = true
}

// Dispose releases the offscreen render target held by the system.
func (s *CinematicOverlaySystem) Dispose() {
	if s.target != nil {
		s.target.Deallocate()
		s.target = nil
	}
}

func (s *CinematicOverlaySystem) ensureTarget(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if s.target != nil {
		bounds := s.target.Bounds()
		if bounds.Dx() == width && bounds.Dy() == height {
			return
		}
		s.target.Deallocate()
		s.target = nil
	}
	s.target = ebiten.NewImage(width, height)
}

func (s *CinematicOverlaySystem) onEngineLayoutChanged(evt EngineLayoutChangedEvent) {
	s.ensureTarget(evt.Width, evt.Height)
}

func (s *CinematicOverlaySystem) onTimerFinished(evt TimerFinishedEvent) {
	if s.builder == nil {
		return
	}
	cin, tw, timer := s.builder.Get(evt.Entity)
	if cin == nil || tw == nil || timer == nil {
		return
	}
	if !cin.CinematicFinished || cin.DelayFinished {
		return
	}
	s.startEndPhase(cin, tw)
}

func (s *CinematicOverlaySystem) onTweenFinished(evt TweenFinishedEvent) {
	if s.builder == nil {
		return
	}
	cin, tw, timer := s.builder.Get(evt.Entity)
	if cin == nil || tw == nil || timer == nil {
		return
	}

	if !cin.CinematicFinished {
		cin.TransitionValue = tw.Current
		if cin.StartFade {
			updateStartFadeAlpha(cin)
		}
		cin.CinematicFinished = true
		cin.Phase = CinematicPhaseHold

		if cin.AutoFinish {
			if cin.CinematicDelay <= 0 {
				s.startEndPhase(cin, tw)
			} else {
				timer.Time = cin.CinematicDelay
				timer.State = TimerStateActive
			}
		}
		return
	}

	if cin.DelayFinished && !cin.Finished {
		cin.TransitionValue = tw.Current
		if cin.EndFade {
			updateEndFadeAlpha(cin)
		}
		cin.Finished = true
		cin.Phase = CinematicPhaseDone
		if cin.Callback != nil {
			cin.Callback()
		}
	}
}

func (s *CinematicOverlaySystem) startEndPhase(cin *CinematicOverlayComponent, tw *TweenComponent) {
	cin.DelayFinished = true
	cin.Phase = CinematicPhaseEnd

	var startVal, endVal float64
	var easeType EaseType
	if cin.CinematicType == CinematicMovie {
		if cin.Height > 0 {
			startVal = cin.Radius / float64(cin.Height)
		}
		if cin.EndType == CinematicTransitionIn {
			endVal = 1.0
			easeType = QuadIn
		} else {
			endVal = 0.0
			easeType = QuadOut
		}
	} else {
		startVal = cin.Radius
		if cin.EndType == CinematicTransitionIn {
			endVal = cin.SpotlightMaxValue
			easeType = BackIn
		} else {
			endVal = 0.0
			easeType = BackOut
		}
	}

	tw.ID = "cinematic_end"
	tw.Delay = 0
	tw.Start = startVal
	tw.End = endVal
	tw.Duration = cin.EndSpeed
	tw.Time = 0
	tw.EaseType = easeType
	tw.Current = startVal
	tw.Finished = false
}

// Update syncs cinematic transition progress from TweenComponent, updates fade alpha,
// and removes completed CinematicTransitionIn entities after iteration.
func (s *CinematicOverlaySystem) Update(w *teishoku.World, dt float64) {
	if s.filter == nil {
		return
	}

	s.toRemove = s.toRemove[:0]
	chunks := s.filter.Chunks()
	for chunks.Next() {
		ents := chunks.Entities()
		cins, tweens, _ := chunks.Get()
		for i := range cins {
			cin := &cins[i]
			tw := &tweens[i]

			if cin.Finished {
				if cin.EndType == CinematicTransitionIn {
					s.toRemove = append(s.toRemove, ents[i])
				}
				continue
			}

			cin.TransitionValue = tw.Current
			if !cin.CinematicFinished && cin.StartFade {
				updateStartFadeAlpha(cin)
			} else if cin.DelayFinished && cin.EndFade {
				updateEndFadeAlpha(cin)
			}
		}
	}

	if len(s.toRemove) > 0 {
		w.RemoveEntities(s.toRemove)
		s.toRemove = s.toRemove[:0]
	}
}

func updateStartFadeAlpha(cin *CinematicOverlayComponent) {
	if cin.CinematicType == CinematicMovie {
		cin.CinematicAlphaValue = CalculateProgressRatio(0.0, 1.0, cin.TransitionValue)
		return
	}
	if cin.SpotlightMaxValue > 0 {
		cin.CinematicAlphaValue = CalculateProgressRatio(0.0, 1.0, cin.TransitionValue/cin.SpotlightMaxValue)
	}
}

func updateEndFadeAlpha(cin *CinematicOverlayComponent) {
	if !cin.StartFade {
		if cin.CinematicType == CinematicMovie {
			var startRatio float64
			if cin.Height > 0 {
				startRatio = cin.Radius / float64(cin.Height)
			}
			cin.CinematicAlphaValue = CalculateProgressRatio(startRatio, 1.0, cin.TransitionValue)
			return
		}
		if cin.SpotlightMaxValue > 0 {
			cin.CinematicAlphaValue = CalculateProgressRatio(cin.Radius/cin.SpotlightMaxValue, 1.0, cin.TransitionValue/cin.SpotlightMaxValue)
		}
		return
	}

	if cin.CinematicType == CinematicMovie {
		cin.CinematicAlphaValue = CalculateProgressRatio(0.0, 1.0, cin.TransitionValue)
		return
	}
	if cin.SpotlightMaxValue > 0 {
		cin.CinematicAlphaValue = CalculateProgressRatio(0.0, 1.0, cin.TransitionValue/cin.SpotlightMaxValue)
	}
}

// Draw composites active cinematic overlays onto the screen using an offscreen target
// and BlendClear cutout geometry.
func (s *CinematicOverlaySystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if s.filter == nil || rdr == nil {
		return
	}
	screen := rdr.GetScreen()
	if screen == nil {
		return
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return
	}

	tm := s.tmHandle.Get()
	if tm == nil {
		tm = GetTextureManager(w)
		if tm == nil {
			return
		}
		s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	}
	whiteImg := tm.Get(0)
	if whiteImg == nil {
		return
	}
	srcBounds := whiteImg.Bounds()

	flushed := false
	chunks := s.filter.Chunks()
	for chunks.Next() {
		cins, _, _ := chunks.Get()
		for i := range cins {
			cin := &cins[i]
			if cin.EndType == CinematicTransitionIn && cin.Finished {
				continue
			}
			if !flushed {
				s.ensureTarget(width, height)
				if s.target == nil {
					return
				}
				rdr.Flush()
				flushed = true
			}

			s.target.Fill(color.RGBA{})
			alpha := Clamp(cin.CinematicAlphaValue*cin.OverlayOpacity, 0.0, 1.0)
			updateOverlayVertices(s.vertices, width, height, cin.OverlayColor, alpha, srcBounds)
			s.target.DrawTriangles(s.vertices, s.indices, whiteImg, nil)

			if cin.CinematicType == CinematicMovie {
				s.drawLetterbox(cin, width, height, whiteImg, srcBounds)
			} else {
				s.drawSpotlight(cin, whiteImg, srcBounds)
			}

			screen.DrawImage(s.target, nil)
		}
	}
}

func (s *CinematicOverlaySystem) drawLetterbox(cin *CinematicOverlayComponent, width, height int, whiteImg *ebiten.Image, srcBounds image.Rectangle) {
	if cin.TransitionValue <= 0 {
		return
	}
	middle := float64(height) * 0.5
	halfScalingSize := (float64(height) * cin.TransitionValue) * 0.5
	if halfScalingSize <= 0 {
		return
	}

	top := AdjustDestinationPixel(float32(middle - halfScalingSize))
	bottom := AdjustDestinationPixel(float32(middle + halfScalingSize))
	left := AdjustDestinationPixel(0)
	right := AdjustDestinationPixel(float32(width))

	sx0 := float32(srcBounds.Min.X)
	sy0 := float32(srcBounds.Min.Y)
	sx1 := float32(srcBounds.Max.X)
	sy1 := float32(srcBounds.Max.Y)

	s.letterboxV[0] = ebiten.Vertex{DstX: left, DstY: top, SrcX: sx0, SrcY: sy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	s.letterboxV[1] = ebiten.Vertex{DstX: right, DstY: top, SrcX: sx1, SrcY: sy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	s.letterboxV[2] = ebiten.Vertex{DstX: right, DstY: bottom, SrcX: sx1, SrcY: sy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	s.letterboxV[3] = ebiten.Vertex{DstX: left, DstY: bottom, SrcX: sx0, SrcY: sy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}

	s.target.DrawTriangles(s.letterboxV[:], s.indices, whiteImg, &s.clearOpts)
}

func (s *CinematicOverlaySystem) drawSpotlight(cin *CinematicOverlayComponent, whiteImg *ebiten.Image, srcBounds image.Rectangle) {
	radius := cin.TransitionValue
	if radius <= 0 {
		return
	}
	centerX := cin.Offset.X
	centerY := cin.Offset.Y
	srcX := float32(srcBounds.Min.X+srcBounds.Max.X) * 0.5
	srcY := float32(srcBounds.Min.Y+srcBounds.Max.Y) * 0.5

	s.spotlightV[0] = ebiten.Vertex{
		DstX:   AdjustDestinationPixel(float32(centerX)),
		DstY:   AdjustDestinationPixel(float32(centerY)),
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	}

	step := (2 * math.Pi) / float64(s.spotlightSeg)
	for i := 0; i <= s.spotlightSeg; i++ {
		sin, cos := math.Sincos(float64(i) * step)
		x := centerX + radius*cos
		y := centerY + radius*sin
		s.spotlightV[i+1] = ebiten.Vertex{
			DstX:   AdjustDestinationPixel(float32(x)),
			DstY:   AdjustDestinationPixel(float32(y)),
			SrcX:   srcX,
			SrcY:   srcY,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		}
	}

	s.target.DrawTriangles(s.spotlightV, s.spotlightI, whiteImg, &s.clearOpts)
}
