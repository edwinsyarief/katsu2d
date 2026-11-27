package katsu2d

import (
	"image/color"
	"math"
	"testing"

	"github.com/edwinsyarief/teishoku"
)

func TestAnimationSystem(t *testing.T) {
	frames := []Bound{
		{Min: Point{0, 0}, Max: Point{16, 16}},
		{Min: Point{16, 0}, Max: Point{32, 16}},
		{Min: Point{32, 0}, Max: Point{48, 16}},
	}

	t.Run("AnimLoop", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewAnimationSystem()
		sys.Initialize(w)
		sys.Initialize(w) // idempotent guard

		e := teishoku.NewBuilder2[AnimationComponent, SpriteComponent](w).NewEntityWith(
			AnimationComponent{
				Frames: frames,
				Speed:  0.1,
				Mode:   AnimLoop,
				Active: true,
			},
			NewSpriteComponent(0, 16, 16),
		)

		sys.Update(w, 0.05)
		anim := teishoku.GetComponent[AnimationComponent](w, e)
		if anim.Current != 0 {
			t.Fatalf("expected Current=0 before speed threshold, got %d", anim.Current)
		}

		sys.Update(w, 0.05)
		anim = teishoku.GetComponent[AnimationComponent](w, e)
		spr := teishoku.GetComponent[SpriteComponent](w, e)
		if anim.Current != 1 || spr.Bound != frames[1] {
			t.Fatalf("expected frame 1, got Current=%d Bound=%+v", anim.Current, spr.Bound)
		}

		sys.Update(w, 0.1)
		anim = teishoku.GetComponent[AnimationComponent](w, e)
		if anim.Current != 2 {
			t.Fatalf("expected frame 2, got %d", anim.Current)
		}

		sys.Update(w, 0.1)
		anim = teishoku.GetComponent[AnimationComponent](w, e)
		spr = teishoku.GetComponent[SpriteComponent](w, e)
		if anim.Current != 0 || !anim.Active || spr.Bound != frames[0] {
			t.Fatalf("expected loop back to frame 0 and stay active, got Current=%d Active=%v Bound=%+v", anim.Current, anim.Active, spr.Bound)
		}
	})

	t.Run("AnimOnce", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewAnimationSystem()
		sys.Initialize(w)

		e := teishoku.NewBuilder2[AnimationComponent, SpriteComponent](w).NewEntityWith(
			AnimationComponent{
				Frames: frames,
				Speed:  0.1,
				Mode:   AnimOnce,
				Active: true,
			},
			NewSpriteComponent(0, 16, 16),
		)

		sys.Update(w, 0.1) // -> 1
		sys.Update(w, 0.1) // -> 2 (last frame)
		anim := teishoku.GetComponent[AnimationComponent](w, e)
		if anim.Current != 2 {
			t.Fatalf("expected Current=2 at last frame, got %d", anim.Current)
		}

		sys.Update(w, 0.1) // deactivates at last frame
		anim = teishoku.GetComponent[AnimationComponent](w, e)
		spr := teishoku.GetComponent[SpriteComponent](w, e)
		if anim.Active || anim.Current != 2 || spr.Bound != frames[2] {
			t.Fatalf("expected AnimOnce to deactivate at last frame 2, got Active=%v Current=%d", anim.Active, anim.Current)
		}

		sys.Update(w, 0.5)
		anim = teishoku.GetComponent[AnimationComponent](w, e)
		if anim.Active || anim.Current != 2 {
			t.Fatalf("expected AnimOnce to stay inactive at frame 2, got Active=%v Current=%d", anim.Active, anim.Current)
		}
	})

	t.Run("AnimBoomerang", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewAnimationSystem()
		sys.Initialize(w)

		e := teishoku.NewBuilder2[AnimationComponent, SpriteComponent](w).NewEntityWith(
			AnimationComponent{
				Frames:    frames,
				Speed:     0.1,
				Mode:      AnimBoomerang,
				Direction: true,
				Active:    true,
			},
			NewSpriteComponent(0, 16, 16),
		)

		expected := []int{1, 2, 1, 0, 0, 1}
		for step, want := range expected {
			sys.Update(w, 0.1)
			anim := teishoku.GetComponent[AnimationComponent](w, e)
			if anim.Current != want {
				t.Fatalf("step %d: expected Current=%d, got %d", step, want, anim.Current)
			}
			if !anim.Active {
				t.Fatalf("step %d: expected boomerang animation to stay active", step)
			}
		}

		// Single-frame boomerang deactivates cleanly.
		eSingle := teishoku.NewBuilder2[AnimationComponent, SpriteComponent](w).NewEntityWith(
			AnimationComponent{
				Frames: frames[:1],
				Speed:  0.1,
				Mode:   AnimBoomerang,
				Active: true,
			},
			NewSpriteComponent(0, 16, 16),
		)
		sys.Update(w, 0.1)
		singleAnim := teishoku.GetComponent[AnimationComponent](w, eSingle)
		if singleAnim.Active || singleAnim.Current != 0 {
			t.Fatalf("expected single-frame boomerang to deactivate, got Active=%v Current=%d", singleAnim.Active, singleAnim.Current)
		}
	})

	t.Run("SpeedGuard", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewAnimationSystem()
		sys.Initialize(w)

		b := teishoku.NewBuilder2[AnimationComponent, SpriteComponent](w)
		eZero := b.NewEntityWith(
			AnimationComponent{Frames: frames, Speed: 0, Mode: AnimLoop, Active: true},
			NewSpriteComponent(0, 16, 16),
		)
		eNeg := b.NewEntityWith(
			AnimationComponent{Frames: frames, Speed: -0.2, Mode: AnimLoop, Active: true},
			NewSpriteComponent(0, 16, 16),
		)

		sys.Update(w, 1.0)
		for _, e := range []teishoku.Entity{eZero, eNeg} {
			anim := teishoku.GetComponent[AnimationComponent](w, e)
			if anim.Current != 0 || anim.Elapsed != 0 {
				t.Fatalf("expected Speed <= 0 to be skipped, got Current=%d Elapsed=%f", anim.Current, anim.Elapsed)
			}
		}
	})
}

func TestTimerSystem(t *testing.T) {
	w := teishoku.NewWorld(16)
	sys := NewTimerSystem()
	sys.Initialize(w)
	sys.Initialize(w) // idempotent Initialize must not duplicate subscriptions or reset state

	var events []TimerFinishedEvent
	Subscribe(w, func(evt TimerFinishedEvent) {
		events = append(events, evt)
	})

	b := teishoku.NewBuilder[TimerComponent](w)
	eActive := b.NewEntityWith(TimerComponent{
		ID:    "cooldown",
		Time:  0.5,
		State: TimerStateActive,
	})
	eIdle := b.NewEntityWith(TimerComponent{
		ID:    "idle",
		Time:  0.1,
		State: TimerStateIdle,
	})

	sys.Update(w, 0.2)
	if len(events) != 0 {
		t.Fatalf("expected 0 events at t=0.2, got %d", len(events))
	}
	tc := teishoku.GetComponent[TimerComponent](w, eActive)
	if math.Abs(tc.Time-0.3) > 1e-9 || tc.State != TimerStateActive {
		t.Fatalf("expected Time=0.3 Active, got Time=%f State=%v", tc.Time, tc.State)
	}
	idleTC := teishoku.GetComponent[TimerComponent](w, eIdle)
	if idleTC.Time != 0.1 || idleTC.State != TimerStateIdle {
		t.Fatalf("idle timer should remain unchanged, got %+v", *idleTC)
	}

	sys.Update(w, 0.35)
	if len(events) != 1 {
		t.Fatalf("expected 1 TimerFinishedEvent, got %d", len(events))
	}
	if events[0].Entity != eActive || events[0].ID != "cooldown" {
		t.Fatalf("unexpected event payload: %+v", events[0])
	}
	tc = teishoku.GetComponent[TimerComponent](w, eActive)
	if tc.Time != 0 || tc.State != TimerStateDone {
		t.Fatalf("expected Time=0 Done, got Time=%f State=%v", tc.Time, tc.State)
	}

	// Further updates must not re-fire the completed timer.
	sys.Update(w, 0.5)
	if len(events) != 1 {
		t.Fatalf("expected still 1 event after completion, got %d", len(events))
	}
}

func TestTweenSystem(t *testing.T) {
	w := teishoku.NewWorld(16)
	sys := NewTweenSystem()
	sys.Initialize(w)
	sys.Initialize(w) // idempotent Initialize

	var events []TweenFinishedEvent
	Subscribe(w, func(evt TweenFinishedEvent) {
		events = append(events, evt)
	})

	b := teishoku.NewBuilder[TweenComponent](w)
	eTween := b.NewEntityWith(TweenComponent{
		ID:       "move_x",
		Delay:    0.5,
		Start:    10,
		End:      110,
		Duration: 1.0,
		EaseType: Linear,
		Current:  10,
	})

	// During delay phase.
	sys.Update(w, 0.25)
	tw := teishoku.GetComponent[TweenComponent](w, eTween)
	if tw.Current != 10 || tw.Finished || len(events) != 0 {
		t.Fatalf("expected tween to remain at Start during delay, got Current=%f Finished=%v events=%d", tw.Current, tw.Finished, len(events))
	}

	// Halfway through active duration (total time = 0.25 + 0.75 = 1.0, elapsed past delay = 0.5).
	sys.Update(w, 0.75)
	tw = teishoku.GetComponent[TweenComponent](w, eTween)
	if math.Abs(tw.Current-60.0) > 1e-9 || tw.Finished || len(events) != 0 {
		t.Fatalf("expected midpoint Current=60, got Current=%f Finished=%v events=%d", tw.Current, tw.Finished, len(events))
	}

	// Completion.
	sys.Update(w, 0.5)
	tw = teishoku.GetComponent[TweenComponent](w, eTween)
	if tw.Current != 110 || !tw.Finished {
		t.Fatalf("expected finished tween at End=110, got Current=%f Finished=%v", tw.Current, tw.Finished)
	}
	if len(events) != 1 || events[0].Entity != eTween || events[0].ID != "move_x" {
		t.Fatalf("expected 1 TweenFinishedEvent for eTween, got %+v", events)
	}

	// Duration <= 0 guard completes immediately without NaN.
	eZero := b.NewEntityWith(TweenComponent{
		ID:       "instant_zero",
		Start:    0,
		End:      50,
		Duration: 0,
	})
	eNeg := b.NewEntityWith(TweenComponent{
		ID:       "instant_neg",
		Start:    5,
		End:      25,
		Duration: -1,
	})

	sys.Update(w, 0.016)
	twZero := teishoku.GetComponent[TweenComponent](w, eZero)
	twNeg := teishoku.GetComponent[TweenComponent](w, eNeg)
	if !twZero.Finished || twZero.Current != 50 || math.IsNaN(twZero.Current) {
		t.Fatalf("expected zero-duration tween to snap to End=50, got %+v", *twZero)
	}
	if !twNeg.Finished || twNeg.Current != 25 || math.IsNaN(twNeg.Current) {
		t.Fatalf("expected negative-duration tween to snap to End=25, got %+v", *twNeg)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 total TweenFinishedEvents, got %d", len(events))
	}
}

func TestFadeOverlaySystem(t *testing.T) {
	w := teishoku.NewWorld(16)
	tweenSys := NewTweenSystem()
	fadeSys := NewFadeOverlaySystem()
	tweenSys.Initialize(w)
	fadeSys.Initialize(w)
	fadeSys.Initialize(w) // idempotent guard

	outCallbacks := 0
	eOut := SpawnFadeOverlay(w, FadeTypeOut, color.RGBA{0, 0, 0, 255}, 0.4, 0.1, func() {
		outCallbacks++
	})
	if w.Len() != 1 {
		t.Fatalf("expected world Len=1, got %d", w.Len())
	}

	step := func(dt float64) {
		tweenSys.Update(w, dt)
		fadeSys.Update(w, dt)
	}

	step(0.1) // end of delay
	fadeOut := teishoku.GetComponent[FadeOverlayComponent](w, eOut)
	if fadeOut.Finished || fadeOut.CurrentFade != 0.0 {
		t.Fatalf("expected FadeTypeOut at 0.0 after delay, got CurrentFade=%f Finished=%v", fadeOut.CurrentFade, fadeOut.Finished)
	}

	step(0.4) // complete FadeTypeOut
	fadeOut = teishoku.GetComponent[FadeOverlayComponent](w, eOut)
	if !fadeOut.Finished || fadeOut.CurrentFade != 1.0 {
		t.Fatalf("expected FadeTypeOut persisted at 1.0, got CurrentFade=%f Finished=%v", fadeOut.CurrentFade, fadeOut.Finished)
	}
	if outCallbacks != 1 {
		t.Fatalf("expected 1 FadeTypeOut callback, got %d", outCallbacks)
	}
	if !w.IsValid(eOut) || w.Len() != 1 {
		t.Fatalf("expected FadeTypeOut entity to persist, valid=%v len=%d", w.IsValid(eOut), w.Len())
	}

	// Clean up FadeTypeOut before testing FadeTypeIn so w.Len() == 0 can be verified cleanly.
	w.RemoveEntity(eOut)
	if w.Len() != 0 {
		t.Fatalf("expected world Len=0 after removing FadeTypeOut, got %d", w.Len())
	}

	inCallbacks := 0
	eIn := SpawnFadeOverlay(w, FadeTypeIn, color.RGBA{0, 0, 0, 255}, 0.4, 0.0, func() {
		inCallbacks++
	})
	if w.Len() != 1 {
		t.Fatalf("expected world Len=1 after spawning FadeTypeIn, got %d", w.Len())
	}

	step(0.2)
	if !w.IsValid(eIn) {
		t.Fatal("FadeTypeIn should still be alive mid-transition")
	}
	step(0.2)

	if inCallbacks != 1 {
		t.Fatalf("expected 1 FadeTypeIn callback, got %d", inCallbacks)
	}
	if w.IsValid(eIn) || w.Len() != 0 {
		t.Fatalf("expected FadeTypeIn entity to auto-remove, valid=%v len=%d", w.IsValid(eIn), w.Len())
	}
}

func TestCinematicOverlaySystem(t *testing.T) {
	t.Run("MovieAutoFinishTransitionIn", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		tweenSys := NewTweenSystem()
		timerSys := NewTimerSystem()
		cinSys := NewCinematicOverlaySystem()
		tweenSys.Initialize(w)
		timerSys.Initialize(w)
		cinSys.Initialize(w)
		cinSys.Initialize(w) // idempotent guard
		defer cinSys.Dispose()

		cbCount := 0
		e := SpawnCinematicOverlay(
			w,
			800, 600,
			color.RGBA{0, 0, 0, 255},
			1.0,
			CinematicMovie,
			CinematicTransitionIn,
			CinematicTransitionIn,
			true, true, true,
			0.3, // cinematicDelay
			300, // radius
			0.5, // startSpeed
			0.5, // endSpeed
			V(400, 300),
			func() { cbCount++ },
		)

		step := func(dt float64) {
			tweenSys.Update(w, dt)
			timerSys.Update(w, dt)
			cinSys.Update(w, dt)
		}

		// Start phase -> Hold phase (step in 0.25s increments so 0.3s delay does not expire on the same tick).
		step(0.25)
		step(0.25)
		cin := teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin == nil || cin.Phase != CinematicPhaseHold || !cin.CinematicFinished {
			t.Fatalf("expected CinematicPhaseHold after start phase, got %+v", cin)
		}

		// Delay phase -> End phase (0.05s remaining after the 0.25s tick that activated the timer).
		step(0.1)
		cin = teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin == nil || cin.Phase != CinematicPhaseEnd || !cin.DelayFinished {
			t.Fatalf("expected CinematicPhaseEnd after delay phase, got %+v", cin)
		}

		// End phase -> Done -> auto-removed on CinematicTransitionIn.
		step(0.25)
		step(0.25)
		if cbCount != 1 {
			t.Fatalf("expected callback fired once, got %d", cbCount)
		}
		if w.IsValid(e) || w.Len() != 0 {
			t.Fatalf("expected CinematicTransitionIn entity auto-removed, valid=%v len=%d", w.IsValid(e), w.Len())
		}
	})

	t.Run("SpotlightManualEndTransitionOut", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		tweenSys := NewTweenSystem()
		timerSys := NewTimerSystem()
		cinSys := NewCinematicOverlaySystem()
		tweenSys.Initialize(w)
		timerSys.Initialize(w)
		cinSys.Initialize(w)

		cbCount := 0
		e := SpawnCinematicOverlay(
			w,
			800, 600,
			color.RGBA{0, 0, 0, 255},
			0.9,
			CinematicSpotlight,
			CinematicTransitionOut,
			CinematicTransitionOut,
			true, true, false,
			0.2, // cinematicDelay
			150, // radius
			0.4, // startSpeed
			0.4, // endSpeed
			V(400, 300),
			func() { cbCount++ },
		)

		step := func(dt float64) {
			tweenSys.Update(w, dt)
			timerSys.Update(w, dt)
			cinSys.Update(w, dt)
		}

		// Complete start phase.
		step(0.4)
		cin := teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin.Phase != CinematicPhaseHold || !cin.CinematicFinished || cin.DelayFinished {
			t.Fatalf("expected Hold phase waiting for manual EndCinematic, got %+v", *cin)
		}

		// Stepping without EndCinematic must remain in Hold phase.
		step(1.0)
		cin = teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin.Phase != CinematicPhaseHold {
			t.Fatalf("expected manual spotlight to stay in Hold phase, got %v", cin.Phase)
		}

		EndCinematic(w, e)
		step(0.2) // delay finishes
		cin = teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin.Phase != CinematicPhaseEnd || !cin.DelayFinished {
			t.Fatalf("expected End phase after EndCinematic + delay, got %+v", *cin)
		}

		step(0.4) // end phase finishes
		cin = teishoku.GetComponent[CinematicOverlayComponent](w, e)
		if cin.Phase != CinematicPhaseDone || !cin.Finished {
			t.Fatalf("expected Done phase, got %+v", *cin)
		}
		if cbCount != 1 {
			t.Fatalf("expected 1 callback, got %d", cbCount)
		}

		// Subsequent update keeps CinematicTransitionOut entity alive.
		step(0.1)
		if !w.IsValid(e) || w.Len() != 1 {
			t.Fatalf("expected CinematicTransitionOut entity to remain alive, valid=%v len=%d", w.IsValid(e), w.Len())
		}

		// Dispose is safe to call repeatedly.
		cinSys.Dispose()
		cinSys.Dispose()
	})
}

func TestSpriteAndOrderedSpriteSystemSorting(t *testing.T) {
	t.Run("SpriteSystemDynamicZ", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewSpriteSystem()
		sys.Initialize(w)
		sys.Initialize(w) // idempotent guard

		b := teishoku.NewBuilder2[TransformComponent, SpriteComponent](w)
		spawn := func(z float64) teishoku.Entity {
			tr := NewTransformComponent()
			tr.Z = z
			return b.NewEntityWith(tr, NewSpriteComponent(0, 16, 16))
		}

		e30 := spawn(30)
		e10 := spawn(10)
		e20 := spawn(20)

		sys.Update(w, 1.0/60.0)
		if len(sys.entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(sys.entries))
		}
		if sys.entries[0].entity != e10 || sys.entries[1].entity != e20 || sys.entries[2].entity != e30 {
			t.Fatalf("unexpected initial Z sort order: %+v", sys.entries)
		}

		// Mutate Z on existing entities and verify re-sorting.
		teishoku.GetComponent[TransformComponent](w, e30).Z = 5
		teishoku.GetComponent[TransformComponent](w, e10).Z = 40

		sys.Update(w, 1.0/60.0)
		if sys.entries[0].entity != e30 || sys.entries[1].entity != e20 || sys.entries[2].entity != e10 {
			t.Fatalf("unexpected dynamic Z sort order after mutation: %+v", sys.entries)
		}
	})

	t.Run("OrderedSpriteSystemAutoSortY", func(t *testing.T) {
		w := teishoku.NewWorld(16)
		sys := NewOrderedSpriteSystem()
		sys.Initialize(w)
		sys.Initialize(w) // idempotent guard

		b := teishoku.NewBuilder3[TransformComponent, SpriteComponent, OrderableComponent](w)
		spawn := func(y, yOffset float64) teishoku.Entity {
			tr := NewTransformComponent()
			tr.Position = Point{X: 0, Y: y}
			return b.NewEntityWith(tr, NewSpriteComponent(0, 16, 16), NewYOrderableComponent(yOffset))
		}

		eA := spawn(100, 0)
		eB := spawn(50, 0)
		eC := spawn(75, 0)

		sys.Update(w, 1.0/60.0)
		if len(sys.entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(sys.entries))
		}
		if sys.entries[0].entity != eB || sys.entries[1].entity != eC || sys.entries[2].entity != eA {
			t.Fatalf("unexpected initial Y sort order: %+v", sys.entries)
		}

		// Move entities past each other in Y.
		teishoku.GetComponent[TransformComponent](w, eA).Position.Y = 10
		teishoku.GetComponent[TransformComponent](w, eB).Position.Y = 200

		sys.Update(w, 1.0/60.0)
		if sys.entries[0].entity != eA || sys.entries[1].entity != eC || sys.entries[2].entity != eB {
			t.Fatalf("unexpected dynamic Y sort order after crossing: %+v", sys.entries)
		}
	})
}

func TestLineAndRibbonNoLeak(t *testing.T) {
	const limit = 20

	line := NewLine()
	line.SetPointLimit(limit)
	var lineCapAt25 int
	for i := range 500 {
		line.AddPoint(V(float64(i)*10, float64(i%7)*5))
		if i == 25 {
			lineCapAt25 = cap(line.points)
		}
	}
	if len(line.points) != limit {
		t.Fatalf("expected Line len=%d, got %d", limit, len(line.points))
	}
	if cap(line.points) != lineCapAt25 || cap(line.points) > limit*4 {
		t.Fatalf("Line backing array grew unexpectedly: capAt25=%d finalCap=%d", lineCapAt25, cap(line.points))
	}

	rt := NewRibbonTrails().SetLimit(limit)
	var rtCapAt25 int
	for i := range 500 {
		rt.AddPoint(V(float64(i)*10, float64(i%5)*8))
		if i == 25 {
			rtCapAt25 = cap(rt.points)
		}
	}
	if len(rt.points) != limit {
		t.Fatalf("expected RibbonTrails len=%d, got %d", limit, len(rt.points))
	}
	if cap(rt.points) != rtCapAt25 || cap(rt.points) > limit*4 {
		t.Fatalf("RibbonTrails backing array grew unexpectedly: capAt25=%d finalCap=%d", rtCapAt25, cap(rt.points))
	}

	// Closed line BuildMesh with interpolated colors and widths.
	closed := NewLine()
	closed.AddPoint(V(0, 0))
	closed.AddPoint(V(100, 0))
	closed.AddPoint(V(100, 100))
	closed.AddPoint(V(0, 100))
	closed.SetIsClosed(true)
	closed.SetJointMode(LineJointRound)
	closed.SetWidth(12)
	closed.SetInterpolatedColors(
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 255, 0, 255},
		color.RGBA{0, 0, 255, 255},
	)
	closed.SetInterpolatedWidths(1.0, 2.0, 0.5)
	closed.BuildMesh()

	verts, indices := closed.GetMesh()
	if len(verts) == 0 || len(indices) == 0 || len(indices)%3 != 0 {
		t.Fatalf("expected valid closed line mesh, got verts=%d indices=%d", len(verts), len(indices))
	}
	for i, v := range verts {
		if math.IsNaN(float64(v.DstX)) || math.IsNaN(float64(v.DstY)) {
			t.Fatalf("vertex %d has NaN coordinate: %+v", i, v)
		}
	}
}

func TestColorAndMathUtils(t *testing.T) {
	t.Run("LerpPremultipliedRGBA", func(t *testing.T) {
		zero := color.RGBA{0, 0, 0, 0}
		if got := LerpPremultipliedRGBA(zero, zero, 0.5); got != (color.RGBA{}) {
			t.Fatalf("expected zero color on zero alpha, got %+v", got)
		}
		redZeroA := color.RGBA{255, 0, 0, 0}
		if got := LerpPremultipliedRGBA(redZeroA, zero, 0.5); got != (color.RGBA{}) {
			t.Fatalf("expected zero color when interpolated alpha <= 0, got %+v", got)
		}

		c1 := color.RGBA{0, 100, 200, 255}
		c2 := color.RGBA{200, 200, 100, 255}
		if got := LerpPremultipliedRGBA(c1, c2, -1); got != c1 {
			t.Fatalf("expected clamped t=0 -> %+v, got %+v", c1, got)
		}
		if got := LerpPremultipliedRGBA(c1, c2, 2); got != c2 {
			t.Fatalf("expected clamped t=1 -> %+v, got %+v", c2, got)
		}
		mid := LerpPremultipliedRGBA(c1, c2, 0.5)
		if mid.R != 100 || mid.G != 150 || mid.B != 150 || mid.A != 255 {
			t.Fatalf("unexpected midpoint color: %+v", mid)
		}
	})

	t.Run("HexStringToRGBA", func(t *testing.T) {
		if got := HexStringToRGBA("#FF0080"); got != (color.RGBA{255, 0, 128, 255}) {
			t.Fatalf("unexpected #FF0080 parse: %+v", got)
		}
		if got := HexStringToRGBA("#f0a"); got != (color.RGBA{255, 0, 170, 255}) {
			t.Fatalf("unexpected #f0a parse: %+v", got)
		}
		if got := HexStringToRGBA(""); got != (color.RGBA{0, 0, 0, 255}) {
			t.Fatalf("unexpected empty string parse: %+v", got)
		}
		if got := HexStringToRGBA("bad"); got != (color.RGBA{0, 0, 0, 255}) {
			t.Fatalf("unexpected invalid string parse: %+v", got)
		}
	})

	t.Run("InterpolateColorWithRand", func(t *testing.T) {
		rng := RandomWithSeed(42, 99)
		c1 := color.RGBA{200, 150, 100, 255}
		c2 := color.RGBA{50, 150, 220, 80} // R and A have min > max
		for range 100 {
			got := InterpolateColorWithRand(rng, c1, c2)
			if got.R < 50 || got.R > 200 {
				t.Fatalf("R out of [50,200]: %d", got.R)
			}
			if got.G != 150 {
				t.Fatalf("G expected 150, got %d", got.G)
			}
			if got.B < 100 || got.B > 220 {
				t.Fatalf("B out of [100,220]: %d", got.B)
			}
			if got.A < 80 || got.A > 255 {
				t.Fatalf("A out of [80,255]: %d", got.A)
			}
		}
	})

	t.Run("RectangleIntersects", func(t *testing.T) {
		a := NewRectangle(0, 0, 10, 10)
		b := NewRectangle(5, 5, 15, 15)
		c := NewRectangle(20, 20, 30, 30)
		empty := NewRectangle(5, 5, 5, 5)

		if !a.Intersects(b) || !b.Intersects(a) {
			t.Fatal("expected overlapping axis-aligned rectangles to intersect")
		}
		if a.Intersects(c) {
			t.Fatal("expected separated axis-aligned rectangles not to intersect")
		}
		if a.Intersects(empty) {
			t.Fatal("expected empty rectangle not to intersect")
		}

		// Rotated rectangles via SAT.
		r1 := NewRectangle(0, 0, 20, 4)
		r1.SetAngle(math.Pi / 4)
		r2 := NewRectangle(8, -2, 28, 2)
		r2.SetAngle(-math.Pi / 4)
		if !r1.Intersects(r2) {
			t.Fatal("expected crossing rotated rectangles to intersect")
		}

		rFar := NewRectangle(50, 50, 70, 54)
		rFar.SetAngle(math.Pi / 3)
		if r1.Intersects(rFar) {
			t.Fatal("expected separated rotated rectangles not to intersect")
		}
	})
}
