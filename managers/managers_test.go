package managers

import (
	"math"
	"testing"
)

func TestCooldownManager(t *testing.T) {
	t.Run("SetHasProgressRatioOverride", func(t *testing.T) {
		m := NewCooldownManager(16)

		// Duration <= 0 is ignored.
		m.Set("zero", 0, nil)
		m.Set("neg", -1.0, nil)
		if m.Has("zero") || m.Has("neg") {
			t.Fatal("expected non-positive durations to be ignored")
		}

		called := 0
		m.Set("dash", 2.0, func() { called++ })
		if !m.Has("dash") {
			t.Fatal("expected dash cooldown to be active")
		}

		// Duplicate Set on active ID is ignored.
		m.Set("dash", 10.0, func() { called += 100 })
		if ratio := m.GetRatio("dash"); math.Abs(ratio-1.0) > 1e-9 {
			t.Fatalf("expected initial ratio 1.0, got %f", ratio)
		}
		if prog := m.GetProgress("dash"); math.Abs(prog-0.0) > 1e-9 {
			t.Fatalf("expected initial progress 0.0, got %f", prog)
		}

		m.Update(0.5) // remaining 1.5 / 2.0 = 0.75, progress = 0.25
		if ratio := m.GetRatio("dash"); math.Abs(ratio-0.75) > 1e-9 {
			t.Fatalf("expected ratio 0.75, got %f", ratio)
		}
		if prog := m.GetProgress("dash"); math.Abs(prog-0.25) > 1e-9 {
			t.Fatalf("expected progress 0.25, got %f", prog)
		}

		// Override resets remaining and initial duration.
		m.Override("dash", 4.0)
		if ratio := m.GetRatio("dash"); math.Abs(ratio-1.0) > 1e-9 {
			t.Fatalf("expected ratio 1.0 after Override, got %f", ratio)
		}
		if prog := m.GetProgress("dash"); math.Abs(prog-0.0) > 1e-9 {
			t.Fatalf("expected progress 0.0 after Override, got %f", prog)
		}

		m.Update(2.0)
		if ratio := m.GetRatio("dash"); math.Abs(ratio-0.5) > 1e-9 {
			t.Fatalf("expected ratio 0.5 halfway through overridden duration, got %f", ratio)
		}

		m.Update(2.0)
		if m.Has("dash") {
			t.Fatal("expected dash cooldown to be removed after expiry")
		}
		if called != 1 {
			t.Fatalf("expected original callback called once, got %d", called)
		}
	})

	t.Run("SimultaneousExpirations", func(t *testing.T) {
		m := NewCooldownManager(0) // tests default limit fallback
		fired := make([]string, 0, 5)

		m.Set("a", 0.10, func() { fired = append(fired, "a") })
		m.Set("b", 0.20, func() { fired = append(fired, "b") })
		m.Set("c", 0.20, func() { fired = append(fired, "c") })
		m.Set("d", 0.15, func() { fired = append(fired, "d") })
		m.Set("e", 0.50, func() { fired = append(fired, "e") })
		m.Set("nilcb", 0.10, nil)

		m.Update(0.25)

		if len(fired) != 4 {
			t.Fatalf("expected 4 simultaneous callbacks fired without skips, got %v", fired)
		}
		for _, id := range []string{"a", "b", "c", "d", "nilcb"} {
			if m.Has(id) {
				t.Fatalf("expected expired cooldown %q to be removed", id)
			}
		}
		if !m.Has("e") {
			t.Fatal("expected cooldown 'e' to remain active")
		}
	})

	t.Run("ResetAndReuse", func(t *testing.T) {
		m := NewCooldownManager(8)
		m.Set("x", 1.0, func() {})
		m.Set("y", 2.0, func() {})
		m.Reset()

		if m.Has("x") || m.Has("y") {
			t.Fatal("expected Reset to clear all active cooldowns")
		}

		// Re-setting and updating after Reset must not panic on nil elements.
		called := 0
		m.Set("z", 0.5, func() { called++ })
		m.Update(0.25)
		if !m.Has("z") {
			t.Fatal("expected 'z' active after Reset + Set")
		}
		m.Update(0.3)
		if m.Has("z") || called != 1 {
			t.Fatalf("expected 'z' completed with 1 callback, has=%v called=%d", m.Has("z"), called)
		}
	})
}

func TestDelayManager(t *testing.T) {
	t.Run("AddActivateUpdateAndCleanup", func(t *testing.T) {
		m := NewDelayManager()

		idleCalls := 0
		activeCalls := 0

		m.Add("idle_task", 0.1, func() { idleCalls++ })
		m.Add("active_task", 0.3, func() { activeCalls++ })
		m.Add("nil_task", 0.2, nil)

		if ok, idx := m.HasId("active_task"); !ok || idx < 0 {
			t.Fatalf("expected active_task to exist, got ok=%v idx=%d", ok, idx)
		}

		m.Activate("active_task")
		m.Activate("nil_task")

		// Idle task must not count down; nil_task completes cleanly at t=0.2.
		m.Update(0.25)
		if ok, _ := m.HasId("nil_task"); ok {
			t.Fatal("expected nil_task to be removed after expiry")
		}
		if activeCalls != 0 {
			t.Fatalf("expected active_task not finished yet, got calls=%d", activeCalls)
		}
		if ok, _ := m.HasId("idle_task"); !ok || idleCalls != 0 {
			t.Fatalf("expected idle_task untouched, ok=%v calls=%d", ok, idleCalls)
		}

		// Finish active_task.
		m.Update(0.1)
		if activeCalls != 1 {
			t.Fatalf("expected active_task callback once, got %d", activeCalls)
		}
		if ok, _ := m.HasId("active_task"); ok {
			t.Fatal("expected completed active_task to be cleaned up")
		}

		// Re-updating must not re-fire completed task.
		m.Update(0.5)
		if activeCalls != 1 || idleCalls != 0 {
			t.Fatalf("unexpected callback counts after cleanup: active=%d idle=%d", activeCalls, idleCalls)
		}
	})

	t.Run("ReplaceExistingAndSimultaneousCleanup", func(t *testing.T) {
		m := NewDelayManager()

		oldCalled := false
		newCalled := false
		m.Add("dup", 0.1, func() { oldCalled = true })
		m.Add("dup", 0.2, func() { newCalled = true })
		m.Activate("dup")

		secondCalled := false
		m.Add("second", 0.2, func() { secondCalled = true })
		m.Activate("second")

		m.Update(0.25)
		if oldCalled {
			t.Fatal("replaced callback should not fire")
		}
		if !newCalled || !secondCalled {
			t.Fatalf("expected both active tasks to fire, newCalled=%v secondCalled=%v", newCalled, secondCalled)
		}
		if len(m.delays) != 0 {
			t.Fatalf("expected all completed tasks cleaned up, got len=%d", len(m.delays))
		}
	})

	t.Run("ActivateMissingPanics", func(t *testing.T) {
		m := NewDelayManager()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected Activate on missing task id to panic")
			}
		}()
		m.Activate("missing")
	})
}
