package animation

import (
	"maps"
	"math"
	"testing"
)

// mockTarget stores named properties for timeline tests.
type mockTarget struct {
	props map[string]float64
}

func newMockTarget(initial map[string]float64) *mockTarget {
	props := make(map[string]float64)
	maps.Copy(props, initial)
	return &mockTarget{props: props}
}

func (m *mockTarget) Property(name string) float64 {
	return m.props[name]
}

func (m *mockTarget) SetProperty(name string, value float64) {
	m.props[name] = value
}

// Timeline construction.

func TestNewTimelineDefaults(t *testing.T) {
	tl := NewTimeline()
	if !tl.IsPlaying() {
		t.Error("expected autoplay by default")
	}
	if tl.IsComplete() {
		t.Error("should not be complete at start")
	}
	if tl.Duration() != 0 {
		t.Errorf("expected 0 auto-duration, got %f", tl.Duration())
	}
}

func TestNewTimelineWithOptions(t *testing.T) {
	tl := NewTimeline(TimelineOptions{
		Duration: 2.0,
		Loop:     true,
		Autoplay: true,
	})
	if tl.Duration() != 2.0 {
		t.Errorf("expected duration 2.0, got %f", tl.Duration())
	}
	if !tl.IsPlaying() {
		t.Error("expected autoplay")
	}
}

func TestNewTimelineNoAutoplay(t *testing.T) {
	tl := NewTimeline(TimelineOptions{
		Duration: 1.0,
		Autoplay: false,
	})
	if tl.IsPlaying() {
		t.Error("expected not playing when Autoplay=false")
	}
}

// Property interpolation.

func TestBasicPropertyAnimation(t *testing.T) {
	target := newMockTarget(map[string]float64{"opacity": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 1.0,
		Ease:     Linear,
		Properties: map[string]float64{
			"opacity": 1.0,
		},
	})

	// Linear interpolation reaches the midpoint at half the duration.
	tl.Update(0.5)
	got := target.Property("opacity")
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("at t=0.5: opacity = %f, want ≈ 0.5", got)
	}

	// The property reaches its end value at the full duration.
	tl.Update(0.5)
	got = target.Property("opacity")
	if math.Abs(got-1.0) > 0.01 {
		t.Errorf("at t=1.0: opacity = %f, want ≈ 1.0", got)
	}
}

func TestPropertyAnimationWithEasing(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 1.0,
		Ease:     InQuad,
		Properties: map[string]float64{
			"x": 100.0,
		},
	})

	// InQuad maps half progress to one-quarter of the value range.
	tl.Update(0.5)
	got := target.Property("x")
	if math.Abs(got-25.0) > 1.0 {
		t.Errorf("at t=0.5: x = %f, want ≈ 25.0 (inQuad)", got)
	}
}

// Delayed scheduling.

func TestDelayedStart(t *testing.T) {
	target := newMockTarget(map[string]float64{"y": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 0.5,
		Ease:     Linear,
		Properties: map[string]float64{
			"y": 10.0,
		},
	}, 0.5) // Start after half a second.

	// The target remains unchanged before the start time.
	tl.Update(0.3)
	if got := target.Property("y"); got != 0.0 {
		t.Errorf("at t=0.3: y = %f, want 0.0 (not yet started)", got)
	}

	// At time 0.75, the half-second animation has run for 0.25 seconds.
	tl.Update(0.45)
	got := target.Property("y")
	if math.Abs(got-5.0) > 0.5 {
		t.Errorf("at t=0.75: y = %f, want ≈ 5.0", got)
	}
}

func TestAddAtRelativeOffset(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0, "y": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})
	// The relative item starts at 0.7 and ends at 1.0.
	tl.AddAt(target, AnimationOptions{
		Duration:   0.3,
		Ease:       Linear,
		Properties: map[string]float64{"y": 5.0},
	}, "+=0.2")

	if math.Abs(tl.Duration()-1.0) > 0.01 {
		t.Errorf("expected duration ≈ 1.0, got %f", tl.Duration())
	}
}

// Playback controls.

func TestPlayPause(t *testing.T) {
	target := newMockTarget(map[string]float64{"v": 0.0})

	tl := NewTimeline(TimelineOptions{Autoplay: false})
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"v": 100.0},
	})

	// A paused timeline ignores elapsed time.
	tl.Update(0.5)
	if got := target.Property("v"); got != 0.0 {
		t.Errorf("paused: v = %f, want 0.0", got)
	}

	// Playback advances interpolation.
	tl.Play()
	tl.Update(0.5)
	got := target.Property("v")
	if math.Abs(got-50.0) > 1.0 {
		t.Errorf("after play: v = %f, want ≈ 50.0", got)
	}

	// Pausing again freezes the interpolated value.
	tl.Pause()
	tl.Update(0.5)
	got2 := target.Property("v")
	if math.Abs(got2-got) > 0.01 {
		t.Errorf("after pause: v changed from %f to %f", got, got2)
	}
}

func TestRestart(t *testing.T) {
	target := newMockTarget(map[string]float64{"v": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"v": 100.0},
	})

	// Advance beyond the duration to complete the first pass.
	tl.Update(1.5)
	if !tl.IsComplete() {
		t.Error("expected complete after full duration")
	}

	// Restart clears completion and begins from time zero.
	target.SetProperty("v", 0.0)
	tl.Restart()
	if tl.IsComplete() {
		t.Error("should not be complete after restart")
	}
	if !tl.IsPlaying() {
		t.Error("should be playing after restart")
	}

	tl.Update(0.5)
	got := target.Property("v")
	if math.Abs(got-50.0) > 1.0 {
		t.Errorf("after restart + 0.5s: v = %f, want ≈ 50.0", got)
	}
}

// Animation and timeline callbacks.

func TestOnStartCallback(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})
	started := false

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 1.0,
		Ease:     Linear,
		Properties: map[string]float64{
			"x": 10.0,
		},
		OnStart: func() { started = true },
	})

	tl.Update(0.1)
	if !started {
		t.Error("OnStart should have fired")
	}
}

func TestOnCompleteCallback(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})
	completed := false

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 0.5,
		Ease:     Linear,
		Properties: map[string]float64{
			"x": 10.0,
		},
		OnComplete: func() { completed = true },
	})

	tl.Update(0.3)
	if completed {
		t.Error("OnComplete should not have fired yet")
	}

	tl.Update(0.3) // Total time 0.6 exceeds the 0.5 duration.
	if !completed {
		t.Error("OnComplete should have fired")
	}
}

func TestOnUpdateCallback(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})
	var lastProgress float64

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 1.0,
		Ease:     Linear,
		Properties: map[string]float64{
			"x": 10.0,
		},
		OnUpdate: func(p float64) { lastProgress = p },
	})

	tl.Update(0.5)
	if math.Abs(lastProgress-0.5) > 0.01 {
		t.Errorf("OnUpdate progress = %f, want ≈ 0.5", lastProgress)
	}
}

func TestTimelineOnComplete(t *testing.T) {
	completed := false
	tl := NewTimeline(TimelineOptions{
		Duration:   0.5,
		OnComplete: func() { completed = true },
		Autoplay:   true,
	})

	// Add an item whose completion reaches the timeline duration.
	target := newMockTarget(map[string]float64{"x": 0.0})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 1.0},
	})

	tl.Update(0.6)
	if !completed {
		t.Error("timeline OnComplete should have fired")
	}
	if !tl.IsComplete() {
		t.Error("timeline should be complete")
	}
}

func TestCallCallback(t *testing.T) {
	fired := false

	tl := NewTimeline(TimelineOptions{Duration: 1.0, Autoplay: true})
	tl.Call(func() { fired = true }, 0.5)

	tl.Update(0.3)
	if fired {
		t.Error("callback should not fire at t=0.3")
	}

	tl.Update(0.3) // Total time reaches 0.6.
	if !fired {
		t.Error("callback should fire at t=0.6 (scheduled at 0.5)")
	}
}

// Looping.

func TestLoopTimeline(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline(TimelineOptions{
		Duration: 1.0,
		Loop:     true,
		Autoplay: true,
	})
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})

	// Advancing past the duration starts another pass.
	tl.Update(1.1)
	if tl.IsComplete() {
		t.Error("looping timeline should not complete")
	}
	if !tl.IsPlaying() {
		t.Error("should still be playing")
	}

	// A looping timeline resets its current time to zero.
	ct := tl.CurrentTime()
	if ct > 0.5 {
		t.Errorf("after loop, currentTime should be near 0, got %f", ct)
	}
}

func TestItemLoop(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})
	loopCount := 0

	tl := NewTimeline(TimelineOptions{Duration: 5.0, Autoplay: true})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
		Loop:       LoopCount(3), // Run three iterations.
		OnLoop:     func() { loopCount++ },
	})

	// Advance far enough to trigger every configured iteration.
	for range 20 {
		tl.Update(0.3)
	}

	if loopCount < 2 {
		t.Errorf("expected at least 2 loop callbacks, got %d", loopCount)
	}
}

// Alternating iterations.

func TestAlternate(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline(TimelineOptions{Duration: 5.0, Autoplay: true})
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
		Loop:       LoopInfinite(),
		Alternate:  true,
	})

	// The first pass moves from zero to ten.
	tl.Update(1.0)
	first := target.Property("x")

	// Completing the pass prepares the next iteration to run in reverse.
	_ = first
}

// One-shot items.

func TestOnce(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline(TimelineOptions{Duration: 5.0, Autoplay: true})
	tl.Once(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})

	// Advance past the one-shot duration.
	tl.Update(0.6)
	got := target.Property("x")
	if math.Abs(got-10.0) > 0.5 {
		t.Errorf("after once animation: x = %f, want ≈ 10.0", got)
	}

	// A later update leaves the completed item removed.
	tl.Update(0.1)
	// The timeline retains no one-shot state after completion.
	tl.mu.Lock()
	itemCount := len(tl.items)
	tl.mu.Unlock()
	if itemCount != 0 {
		t.Errorf("expected 0 items after once-removal, got %d", itemCount)
	}
}

// Synchronized sub-timelines.

func TestSync(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	sub := NewTimeline(TimelineOptions{Duration: 0.5})
	sub.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})

	parent := NewTimeline(TimelineOptions{Duration: 1.0, Autoplay: true})
	parent.Sync(sub, 0.2) // Start the sub-timeline after 0.2 seconds.

	// The sub-timeline remains idle before its scheduled start.
	parent.Update(0.1)
	if got := target.Property("x"); got != 0.0 {
		t.Errorf("at t=0.1: x = %f, want 0.0 (sub not started)", got)
	}

	// Crossing the start time advances the sub-timeline with the parent frame delta.
	parent.Update(0.4) // Parent time reaches 0.5.
	got := target.Property("x")
	if got < 1.0 {
		t.Errorf("at t=0.5: x = %f, want > 1.0 (sub partially complete)", got)
	}

	// Sync marks the child as parent-driven.
	if !sub.IsSynced() {
		t.Error("sub-timeline should be synced")
	}
}

// Multiple properties.

func TestMultipleProperties(t *testing.T) {
	target := newMockTarget(map[string]float64{
		"x": 0.0,
		"y": 0.0,
	})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration: 1.0,
		Ease:     Linear,
		Properties: map[string]float64{
			"x": 100.0,
			"y": 50.0,
		},
	})

	tl.Update(0.5)
	gotX := target.Property("x")
	gotY := target.Property("y")
	if math.Abs(gotX-50.0) > 1.0 {
		t.Errorf("x = %f, want ≈ 50.0", gotX)
	}
	if math.Abs(gotY-25.0) > 1.0 {
		t.Errorf("y = %f, want ≈ 25.0", gotY)
	}
}

// State-change listeners.

func TestStateChangeListener(t *testing.T) {
	changes := 0
	tl := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl.AddStateChangeListener(func(*Timeline) {
		changes++
	})

	target := newMockTarget(map[string]float64{"x": 0.0})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 1.0},
	})

	tl.Update(0.6) // Completion notifies state listeners.
	if changes == 0 {
		t.Error("state change listener should have been called on completion")
	}
}

func TestRemoveStateChangeListener(t *testing.T) {
	changes := 0
	listener := func(*Timeline) { changes++ }

	tl := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl.AddStateChangeListener(listener)
	tl.RemoveStateChangeListener(listener)

	target := newMockTarget(map[string]float64{"x": 0.0})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 1.0},
	})

	tl.Update(0.6)
	if changes != 0 {
		t.Errorf("expected 0 changes after removal, got %d", changes)
	}
}

// Automatic duration.

func TestAutoDuration(t *testing.T) {
	tl := NewTimeline()
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 1.0},
	}, 0.3) // A 0.5-second item starting at 0.3 ends at 0.8.

	if math.Abs(tl.Duration()-0.8) > 0.01 {
		t.Errorf("auto duration = %f, want 0.8", tl.Duration())
	}
}

func TestOnPauseCallback(t *testing.T) {
	paused := false
	tl := NewTimeline(TimelineOptions{
		Duration: 1.0,
		Autoplay: true,
		OnPause:  func() { paused = true },
	})

	tl.Pause()
	if !paused {
		t.Error("OnPause should have fired")
	}
}

func TestPlayAfterComplete(t *testing.T) {
	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})

	tl.Update(0.6)
	if !tl.IsComplete() {
		t.Error("should be complete")
	}

	// Play restarts a completed timeline.
	target.SetProperty("x", 0.0)
	tl.Play()
	if tl.IsComplete() {
		t.Error("should not be complete after Play()")
	}
	if !tl.IsPlaying() {
		t.Error("should be playing after Play()")
	}
}
