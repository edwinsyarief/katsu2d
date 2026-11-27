package katsu2d

import (
	"image/color"
	"math"
	"sync"
	"testing"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	testAudioOnce sync.Once
	testAudioMgr  *AudioManager
)

func getTestAudioManager() *AudioManager {
	testAudioOnce.Do(func() {
		testAudioMgr = NewAudioManager(44100)
	})
	return testAudioMgr
}

func newTestEngine() *Engine {
	e := &Engine{
		world:                teishoku.NewWorld(64),
		fm:                   NewFontManager(),
		am:                   getTestAudioManager(),
		shm:                  NewShaderManager(),
		tm:                   NewTextureManager(),
		renderer:             NewBatchRenderer(),
		timeScale:            1.0,
		windowWidth:          800,
		windowHeight:         600,
		clearScreenEachFrame: true,
	}
	e.scm = NewSceneManager(e)
	initializeAssetManagers(e.world, e.tm, e.fm, e.am, e.shm, e.scm)
	e.layoutEventCh = teishoku.NewEventChannel[EngineLayoutChangedEvent](getEventBus(e.world))
	return e
}

func TestTransitionConstructorsAndOptions(t *testing.T) {
	// Direction alias equivalence
	if TransitionWipeLeft != TransitionDirLeft ||
		TransitionWipeRight != TransitionDirRight ||
		TransitionWipeUp != TransitionDirUp ||
		TransitionWipeDown != TransitionDirDown {
		t.Fatal("wipe direction aliases do not match TransitionDir constants")
	}
	if TransitionSlideLeft != TransitionDirLeft ||
		TransitionSlideRight != TransitionDirRight ||
		TransitionSlideUp != TransitionDirUp ||
		TransitionSlideDown != TransitionDirDown {
		t.Fatal("slide direction aliases do not match TransitionDir constants")
	}

	var midpointCalled, completeCalled bool
	cfg := FadeTransition(
		1.2,
		color.NRGBA{R: 255, G: 0, B: 0, A: 128},
		WithTransitionEase(CubicIn, CubicOut),
		WithTransitionHold(0.25),
		WithTransitionCenter(0.25, 0.75),
		WithTransitionBlockInput(false),
		WithTransitionUnscaledTime(false),
		WithTransitionBelowOverlay(true),
		WithTransitionOnMidpoint(func() { midpointCalled = true }),
		WithTransitionOnComplete(func() { completeCalled = true }),
	)

	if cfg.Kind != TransitionKindFade {
		t.Fatalf("expected Fade kind, got %v", cfg.Kind)
	}
	if cfg.Duration != 1.2 || cfg.HoldDuration != 0.25 {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
	if cfg.OutEase != CubicIn || cfg.InEase != CubicOut {
		t.Fatalf("unexpected eases: out=%v in=%v", cfg.OutEase, cfg.InEase)
	}
	if cfg.CenterX != 0.25 || cfg.CenterY != 0.75 {
		t.Fatalf("unexpected center: (%v, %v)", cfg.CenterX, cfg.CenterY)
	}
	if cfg.BlockInput || cfg.UnscaledTime || !cfg.BelowOverlay {
		t.Fatalf("unexpected flags: block=%v unscaled=%v below=%v", cfg.BlockInput, cfg.UnscaledTime, cfg.BelowOverlay)
	}
	cfg.OnMidpoint()
	cfg.OnComplete()
	if !midpointCalled || !completeCalled {
		t.Fatal("expected midpoint and complete callbacks to be wired")
	}

	// Verify defaults on other presets
	spot := SpotlightTransition(0.8, nil)
	if spot.Kind != TransitionKindSpotlight || spot.Color == nil || !spot.BlockInput || !spot.UnscaledTime || spot.BelowOverlay {
		t.Fatalf("unexpected spotlight defaults: %+v", spot)
	}
	if spot.CenterX != 0.5 || spot.CenterY != 0.5 {
		t.Fatalf("expected default center (0.5, 0.5), got (%v, %v)", spot.CenterX, spot.CenterY)
	}

	lb := LetterboxTransition(0.6, color.Black)
	if lb.Kind != TransitionKindLetterbox {
		t.Fatalf("expected Letterbox kind, got %v", lb.Kind)
	}

	wipe := WipeTransition(0.5, TransitionWipeRight, color.White)
	if wipe.Kind != TransitionKindWipe || wipe.Direction != TransitionDirRight {
		t.Fatalf("unexpected wipe config: %+v", wipe)
	}

	cf := CrossFadeTransition(0.4)
	if cf.Kind != TransitionKindCrossFade || !cf.BlockInput || !cf.UnscaledTime {
		t.Fatalf("unexpected crossfade config: %+v", cf)
	}

	sl := SlideTransition(0.4, TransitionSlideUp)
	if sl.Kind != TransitionKindSlide || sl.Direction != TransitionDirUp {
		t.Fatalf("unexpected slide config: %+v", sl)
	}
}

func TestTwoPhaseTransitionLifecycle(t *testing.T) {
	eng := newTestEngine()
	defer eng.Dispose()

	var events []string
	sceneA := NewScene()
	sceneA.OnExit = func(*Engine) { events = append(events, "A:exit") }

	sceneB := NewScene()
	sceneB.OnEnter = func(*Engine) { events = append(events, "B:enter") }

	eng.AddScene("A", sceneA)
	eng.AddScene("B", sceneB)
	eng.SwitchScene("A")

	// Unknown target must be rejected
	if eng.TransitionTo("missing", FadeTransition(0.5, color.Black)) {
		t.Fatal("expected TransitionTo unknown scene to return false")
	}
	if eng.IsTransitioning() {
		t.Fatal("expected IsTransitioning to be false after rejected transition")
	}

	// 1.0s total (0.5s out + 0.5s in) + 0.2s hold
	ok := eng.TransitionTo("B", FadeTransition(
		1.0,
		color.Black,
		WithTransitionEase(Linear, Linear),
		WithTransitionHold(0.2),
		WithTransitionOnMidpoint(func() { events = append(events, "midpoint") }),
		WithTransitionOnComplete(func() { events = append(events, "complete") }),
	))
	if !ok {
		t.Fatal("expected TransitionTo B to succeed")
	}
	if !eng.IsTransitioning() {
		t.Fatal("expected IsTransitioning to be true")
	}

	// Re-entrancy guard
	if eng.TransitionTo("A", FadeTransition(0.5, color.Black)) {
		t.Fatal("expected duplicate TransitionTo while transitioning to return false")
	}

	// Advance 15 frames = 0.25s (halfway through Out phase)
	for range 15 {
		if err := eng.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if eng.SceneManager().CurrentScene() != sceneA {
		t.Fatal("expected scene A to remain active during Out phase")
	}
	if math.Abs(eng.SceneManager().transitions.coverage-0.5) > 0.02 {
		t.Fatalf("expected coverage ~0.5 at mid-Out, got %v", eng.SceneManager().transitions.coverage)
	}

	// Advance another 18 frames = 0.30s (total 0.55s: inside 0.2s Hold phase)
	for range 18 {
		if err := eng.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if eng.SceneManager().transitions.phase != transitionPhaseHold {
		t.Fatalf("expected Hold phase, got %v", eng.SceneManager().transitions.phase)
	}
	if eng.SceneManager().CurrentScene() != sceneA {
		t.Fatal("expected scene switch to wait until end of Hold phase")
	}

	// Advance 15 frames = 0.25s (total 0.80s: past Hold, into In phase)
	for range 15 {
		if err := eng.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if eng.SceneManager().CurrentScene() != sceneB {
		t.Fatal("expected scene B to be active after Midpoint")
	}
	if eng.SceneManager().transitions.phase != transitionPhaseIn {
		t.Fatalf("expected In phase, got %v", eng.SceneManager().transitions.phase)
	}
	if len(events) != 3 || events[0] != "A:exit" || events[1] != "B:enter" || events[2] != "midpoint" {
		t.Fatalf("unexpected midpoint event order: %v", events)
	}

	// Advance 30 frames = 0.50s (total 1.30s >= 1.20s): completed
	for range 30 {
		if err := eng.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if eng.IsTransitioning() {
		t.Fatal("expected transition to finish")
	}
	if len(events) != 4 || events[3] != "complete" {
		t.Fatalf("expected complete event at end, got %v", events)
	}
}

func TestSinglePhaseCaptureAndDraw(t *testing.T) {
	for _, kind := range []TransitionKind{TransitionKindCrossFade, TransitionKindSlide} {
		eng := newTestEngine()
		sceneA := NewScene()
		sceneB := NewScene()

		var enterCount int
		sceneB.OnEnter = func(*Engine) { enterCount++ }

		eng.AddScene("A", sceneA)
		eng.AddScene("B", sceneB)
		eng.SwitchScene("A")

		var midpoint, complete bool
		var cfg TransitionConfig
		if kind == TransitionKindCrossFade {
			cfg = CrossFadeTransition(0.5,
				WithTransitionEase(Linear, Linear),
				WithTransitionOnMidpoint(func() { midpoint = true }),
				WithTransitionOnComplete(func() { complete = true }),
			)
		} else {
			cfg = SlideTransition(0.5, TransitionSlideLeft,
				WithTransitionEase(Linear, Linear),
				WithTransitionOnMidpoint(func() { midpoint = true }),
				WithTransitionOnComplete(func() { complete = true }),
			)
		}

		if !eng.TransitionTo("B", cfg) {
			t.Fatalf("kind %v: TransitionTo failed", kind)
		}

		// Update before first Draw must wait for capture
		for range 5 {
			_ = eng.Update()
		}
		if eng.SceneManager().CurrentScene() != sceneA || !eng.SceneManager().transitions.needsCapture {
			t.Fatalf("kind %v: expected capture to wait for first Draw", kind)
		}

		screen := ebiten.NewImage(64, 64)
		eng.Draw(screen)

		if eng.SceneManager().transitions.needsCapture {
			t.Fatalf("kind %v: expected needsCapture=false after first Draw", kind)
		}
		if eng.SceneManager().CurrentScene() != sceneB || enterCount != 1 || !midpoint {
			t.Fatalf("kind %v: expected immediate switch + midpoint on first Draw", kind)
		}
		if eng.SceneManager().transitions.captureBuf == nil {
			t.Fatalf("kind %v: expected captureBuf to be allocated", kind)
		}

		// Advance 0.5s (30 frames)
		for range 30 {
			_ = eng.Update()
			eng.Draw(screen)
		}
		if eng.IsTransitioning() || !complete {
			t.Fatalf("kind %v: expected single-phase transition to complete", kind)
		}

		eng.Dispose()
		screen.Deallocate()
	}
}

func TestTwoPhaseKindsDrawAndResize(t *testing.T) {
	configs := []TransitionConfig{
		FadeTransition(0.2, color.Black),
		SpotlightTransition(0.2, color.Black, WithTransitionCenter(0.3, 0.7)),
		LetterboxTransition(0.2, color.Black),
		WipeTransition(0.2, TransitionWipeDown, color.RGBA{R: 10, G: 20, B: 30, A: 255}),
	}

	for _, cfg := range configs {
		eng := newTestEngine()
		eng.AddScene("A", NewScene())
		eng.AddScene("B", NewScene())
		eng.SwitchScene("A")

		if !eng.TransitionTo("B", cfg) {
			t.Fatalf("kind %v: TransitionTo failed", cfg.Kind)
		}

		small := ebiten.NewImage(64, 64)
		large := ebiten.NewImage(128, 96)
		for frame := range 16 {
			_ = eng.Update()
			if frame < 8 {
				eng.Draw(small)
			} else {
				eng.Draw(large)
			}
		}
		if eng.IsTransitioning() {
			t.Fatalf("kind %v: expected transition to complete", cfg.Kind)
		}

		eng.Dispose()
		if eng.SceneManager().transitions.maskBuf != nil || eng.SceneManager().transitions.captureBuf != nil {
			t.Fatalf("kind %v: expected Dispose to clear offscreen buffers", cfg.Kind)
		}
		small.Deallocate()
		large.Deallocate()
	}
}

func TestTransitionInputBlocking(t *testing.T) {
	eng := newTestEngine()
	defer eng.Dispose()

	globalInputSys := NewInputSystem()
	eng.AddUpdateSystem(globalInputSys)

	sceneInputSys := NewInputSystem()
	sceneA := NewScene()
	sceneA.AddSystem(sceneInputSys)
	sceneB := NewScene()
	sceneB.AddSystem(NewInputSystem())

	eng.AddScene("A", sceneA)
	eng.AddScene("B", sceneB)
	eng.SwitchScene("A")

	comp := NewInputComponent(map[Action][]KeyConfig{
		"jump": {NewKeyConfig(ebiten.KeySpace)},
	})
	comp.JustPressed["jump"] = true
	comp.Pressed["jump"] = true
	comp.JustReleased["jump"] = true
	comp.MouseWheelX = 2.5
	comp.MouseWheelY = -1.5

	globalEnt := teishoku.NewBuilder[InputComponent](eng.World()).NewEntityWith(comp)
	sceneEnt := teishoku.NewBuilder[InputComponent](sceneA.World()).NewEntityWith(NewInputComponent(map[Action][]KeyConfig{
		"fire": {NewKeyConfig(ebiten.KeyZ)},
	}))
	sComp := teishoku.GetComponent[InputComponent](sceneA.World(), sceneEnt)
	sComp.JustPressed["fire"] = true
	sComp.Pressed["fire"] = true
	sComp.MouseWheelY = 4.0

	if !eng.TransitionTo("B", FadeTransition(0.4, color.Black, WithTransitionBlockInput(true))) {
		t.Fatal("TransitionTo failed")
	}

	_ = eng.Update()

	gAfter := teishoku.GetComponent[InputComponent](eng.World(), globalEnt)
	if gAfter.IsJustPressed("jump") || gAfter.IsPressed("jump") || gAfter.IsJustReleased("jump") || gAfter.MouseWheelX != 0 || gAfter.MouseWheelY != 0 {
		t.Fatalf("expected global input to be zeroed while transition blocks input, got %+v", gAfter)
	}
	sAfter := teishoku.GetComponent[InputComponent](sceneA.World(), sceneEnt)
	if sAfter.IsJustPressed("fire") || sAfter.IsPressed("fire") || sAfter.MouseWheelY != 0 {
		t.Fatalf("expected scene input to be zeroed while transition blocks input, got %+v", sAfter)
	}

	eng.CancelTransition()
	if eng.SceneManager().isTransitionInputBlocked() {
		t.Fatal("expected CancelTransition to release input block")
	}
	if state, _ := teishoku.GetResource[inputBlockState](eng.World().Resources()); state == nil || state.blocked {
		t.Fatal("expected global inputBlockState.blocked to be false after cancel")
	}
}

func TestTransitionTimeScalingAndChaining(t *testing.T) {
	eng := newTestEngine()
	defer eng.Dispose()

	eng.AddScene("A", NewScene())
	eng.AddScene("B", NewScene())
	eng.SwitchScene("A")
	eng.SetTimeScale(0.0)

	// Scaled transition must pause when TimeScale == 0
	if !eng.TransitionTo("B", FadeTransition(0.2, color.Black, WithTransitionUnscaledTime(false))) {
		t.Fatal("TransitionTo failed")
	}
	for range 20 {
		_ = eng.Update()
	}
	if !eng.IsTransitioning() {
		t.Fatal("expected scaled transition not to advance while TimeScale == 0")
	}

	eng.CancelTransition()

	// Unscaled transition must advance even when TimeScale == 0, and allow chaining in OnComplete
	chained := false
	if !eng.TransitionTo("B", FadeTransition(0.1, color.Black,
		WithTransitionUnscaledTime(true),
		WithTransitionOnComplete(func() {
			chained = eng.TransitionTo("A", FadeTransition(0.1, color.Black))
		}),
	)) {
		t.Fatal("TransitionTo failed")
	}
	for range 10 {
		_ = eng.Update()
	}
	if !chained || !eng.IsTransitioning() {
		t.Fatalf("expected OnComplete chaining to start second transition, chained=%v active=%v", chained, eng.IsTransitioning())
	}

	// Zero-duration transitions complete immediately
	eng.CancelTransition()
	var mid, done bool
	if !eng.TransitionTo("A", FadeTransition(0, color.Black,
		WithTransitionOnMidpoint(func() { mid = true }),
		WithTransitionOnComplete(func() { done = true }),
	)) {
		t.Fatal("zero-duration TransitionTo failed")
	}
	if eng.IsTransitioning() || !mid || !done {
		t.Fatalf("expected zero-duration transition to finish immediately: active=%v mid=%v done=%v", eng.IsTransitioning(), mid, done)
	}
}

func BenchmarkTransitionControllerUpdate(b *testing.B) {
	eng := newTestEngine()
	defer eng.Dispose()
	eng.AddScene("A", NewScene())
	eng.AddScene("B", NewScene())
	eng.SwitchScene("A")

	sm := eng.SceneManager()
	ctrl := sm.transitions
	cfg := FadeTransition(1000.0, color.Black)
	ctrl.start(sm, "B", cfg)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ctrl.update(sm, 1.0/60.0)
		if !ctrl.isTransitioning() {
			ctrl.start(sm, "B", cfg)
		}
	}
}
