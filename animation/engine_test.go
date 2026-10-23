package animation

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Mock renderer
// ---------------------------------------------------------------------------

type mockRenderer struct {
	liveCount int
}

func (m *mockRenderer) RequestLive() { m.liveCount++ }
func (m *mockRenderer) DropLive()    { m.liveCount-- }

// ---------------------------------------------------------------------------
// Setup / Teardown
// ---------------------------------------------------------------------------

func resetForTest() {
	ResetEngine()
}

// ---------------------------------------------------------------------------
// Attach / Detach
// ---------------------------------------------------------------------------

func TestAttachDetach(t *testing.T) {
	resetForTest()

	if IsAttached() {
		t.Error("should not be attached initially")
	}

	r := &mockRenderer{}
	Attach(r)
	if !IsAttached() {
		t.Error("should be attached after Attach()")
	}

	Detach()
	if IsAttached() {
		t.Error("should not be attached after Detach()")
	}
}

// ---------------------------------------------------------------------------
// Register / Unregister
// ---------------------------------------------------------------------------

func TestRegisterUnregister(t *testing.T) {
	resetForTest()

	tl := NewTimeline()
	Register(tl)
	if ActiveCount() != 1 {
		t.Errorf("expected 1 timeline, got %d", ActiveCount())
	}

	// Duplicate registration should not add twice
	Register(tl)
	if ActiveCount() != 1 {
		t.Errorf("expected 1 timeline after duplicate, got %d", ActiveCount())
	}

	Unregister(tl)
	if ActiveCount() != 0 {
		t.Errorf("expected 0 timelines, got %d", ActiveCount())
	}
}

// ---------------------------------------------------------------------------
// Update drives timelines
// ---------------------------------------------------------------------------

func TestUpdateDrivesTimelines(t *testing.T) {
	resetForTest()

	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline()
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"x": 100.0},
	})
	Register(tl)

	Update(0.5)
	got := target.Property("x")
	if got < 40.0 || got > 60.0 {
		t.Errorf("after Update(0.5): x = %f, want ≈ 50.0", got)
	}
}

// ---------------------------------------------------------------------------
// Completed timelines are auto-removed
// ---------------------------------------------------------------------------

func TestCompletedAutoRemove(t *testing.T) {
	resetForTest()

	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})
	Register(tl)

	if ActiveCount() != 1 {
		t.Errorf("expected 1, got %d", ActiveCount())
	}

	Update(0.6) // complete the timeline
	if ActiveCount() != 0 {
		t.Errorf("expected 0 after completion, got %d", ActiveCount())
	}
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

func TestClear(t *testing.T) {
	resetForTest()

	Register(NewTimeline())
	Register(NewTimeline())
	Register(NewTimeline())

	if ActiveCount() != 3 {
		t.Errorf("expected 3, got %d", ActiveCount())
	}

	Clear()
	if ActiveCount() != 0 {
		t.Errorf("expected 0 after Clear, got %d", ActiveCount())
	}
}

// ---------------------------------------------------------------------------
// Synced timelines are not updated by engine
// ---------------------------------------------------------------------------

func TestSyncedNotUpdatedByEngine(t *testing.T) {
	resetForTest()

	target := newMockTarget(map[string]float64{"x": 0.0})

	tl := NewTimeline()
	tl.synced = true
	tl.Add(target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"x": 100.0},
	})
	Register(tl)

	Update(0.5)
	got := target.Property("x")
	if got != 0.0 {
		t.Errorf("synced timeline should not be updated by engine: x = %f", got)
	}
}

// ---------------------------------------------------------------------------
// CreateTimeline convenience
// ---------------------------------------------------------------------------

func TestCreateTimeline(t *testing.T) {
	resetForTest()

	tl := CreateTimeline(TimelineOptions{Duration: 1.0, Autoplay: true})
	if ActiveCount() != 1 {
		t.Errorf("expected 1 after CreateTimeline, got %d", ActiveCount())
	}
	if !tl.IsPlaying() {
		t.Error("expected playing after CreateTimeline with Autoplay")
	}
}

// ---------------------------------------------------------------------------
// Live rendering integration
// ---------------------------------------------------------------------------

func TestLiveRendering(t *testing.T) {
	resetForTest()

	r := &mockRenderer{}
	Attach(r)

	target := newMockTarget(map[string]float64{"x": 0.0})
	tl := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl.Add(target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"x": 10.0},
	})
	Register(tl)

	if r.liveCount != 1 {
		t.Errorf("expected liveCount=1 after registering playing timeline, got %d", r.liveCount)
	}

	// Complete
	Update(0.6)

	if r.liveCount != 0 {
		t.Errorf("expected liveCount=0 after all timelines complete, got %d", r.liveCount)
	}
}

// ---------------------------------------------------------------------------
// Multiple timelines
// ---------------------------------------------------------------------------

func TestMultipleTimelines(t *testing.T) {
	resetForTest()

	t1Target := newMockTarget(map[string]float64{"a": 0.0})
	t2Target := newMockTarget(map[string]float64{"b": 0.0})

	tl1 := NewTimeline(TimelineOptions{Duration: 1.0, Autoplay: true})
	tl1.Add(t1Target, AnimationOptions{
		Duration:   1.0,
		Ease:       Linear,
		Properties: map[string]float64{"a": 100.0},
	})

	tl2 := NewTimeline(TimelineOptions{Duration: 0.5, Autoplay: true})
	tl2.Add(t2Target, AnimationOptions{
		Duration:   0.5,
		Ease:       Linear,
		Properties: map[string]float64{"b": 50.0},
	})

	Register(tl1)
	Register(tl2)

	Update(0.5)

	gotA := t1Target.Property("a")
	gotB := t2Target.Property("b")

	if gotA < 40.0 || gotA > 60.0 {
		t.Errorf("a = %f, want ≈ 50.0", gotA)
	}
	if gotB < 40.0 || gotB > 60.0 {
		t.Errorf("b = %f, want ≈ 50.0", gotB)
	}

	// After tl2 completes, only tl1 remains
	Update(0.1) // tl2 should be auto-removed
	if ActiveCount() != 1 {
		t.Errorf("expected 1 after tl2 completes, got %d", ActiveCount())
	}
}
