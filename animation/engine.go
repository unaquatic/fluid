package animation

import (
	"slices"
	"sync"
)

// ---------------------------------------------------------------------------
// RendererLike is the minimal interface used by the engine to request live
// rendering. We use an interface to avoid a circular import on render.
// ---------------------------------------------------------------------------

// RendererLike is the minimal interface the engine needs from a renderer.
// It allows requesting continuous rendering while animations are active.
type RendererLike interface {
	// RequestLive increments the renderer's live-rendering reference count.
	RequestLive()
	// DropLive decrements the renderer's live-rendering reference count.
	DropLive()
}

// ---------------------------------------------------------------------------
// TimelineEngine
// ---------------------------------------------------------------------------

// TimelineEngine manages all active timelines and drives them from the
// renderer's frame callback.
type TimelineEngine struct {
	timelines []*Timeline
	renderer  RendererLike
	attached  bool
	liveHeld  bool // true if we've called RequestLive
	mu        sync.Mutex
}

// Global engine instance.
var engine = &TimelineEngine{}

// ---------------------------------------------------------------------------
// Attach / Detach
// ---------------------------------------------------------------------------

// Attach binds the engine to a renderer. The renderer calls
// engine.Update(deltaTime) each frame.
func Attach(renderer RendererLike) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.renderer = renderer
	engine.attached = true
}

// Detach unbinds the engine from the renderer.
func Detach() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.liveHeld && engine.renderer != nil {
		engine.renderer.DropLive()
		engine.liveHeld = false
	}
	engine.renderer = nil
	engine.attached = false
}

// ---------------------------------------------------------------------------
// Register / Unregister
// ---------------------------------------------------------------------------

// Register adds a timeline to the engine for automatic updates.
// Synced timelines (driven by a parent) are not registered.
func Register(tl *Timeline) {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	// Avoid duplicates
	if slices.Contains(engine.timelines, tl) {
		return
	}

	engine.timelines = append(engine.timelines, tl)
	engine.updateLiveState()
}

// Unregister removes a timeline from the engine.
func Unregister(tl *Timeline) {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	for i, existing := range engine.timelines {
		if existing == tl {
			engine.timelines = append(engine.timelines[:i], engine.timelines[i+1:]...)
			break
		}
	}
	engine.updateLiveState()
}

// Clear removes all timelines from the engine.
func Clear() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.timelines = nil
	engine.updateLiveState()
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update advances all registered, non-synced timelines by deltaTime seconds.
// Called by the renderer each frame.
func Update(deltaTime float64) {
	engine.mu.Lock()
	// Copy the slice so timelines can call Register/Unregister during Update
	// without deadlocking.
	tls := make([]*Timeline, len(engine.timelines))
	copy(tls, engine.timelines)
	engine.mu.Unlock()

	for _, tl := range tls {
		if tl.IsSynced() {
			continue
		}
		tl.Update(deltaTime)
	}

	// Remove completed timelines
	engine.mu.Lock()
	n := 0
	for _, tl := range engine.timelines {
		if !tl.IsComplete() {
			engine.timelines[n] = tl
			n++
		}
	}
	engine.timelines = engine.timelines[:n]
	engine.updateLiveState()
	engine.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Convenience constructor
// ---------------------------------------------------------------------------

// CreateTimeline creates a Timeline, registers it with the global engine,
// and auto-plays unless Autoplay is explicitly false.
func CreateTimeline(opts ...TimelineOptions) *Timeline {
	tl := NewTimeline(opts...)
	Register(tl)
	return tl
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// ActiveCount returns the number of registered timelines.
func ActiveCount() int {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	return len(engine.timelines)
}

// IsAttached returns whether the engine is attached to a renderer.
func IsAttached() bool {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	return engine.attached
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// updateLiveState requests or drops live rendering based on whether there
// are active playing timelines. Must be called with engine.mu held.
func (e *TimelineEngine) updateLiveState() {
	if e.renderer == nil {
		return
	}

	hasPlaying := false
	for _, tl := range e.timelines {
		if tl.IsPlaying() {
			hasPlaying = true
			break
		}
	}

	if hasPlaying && !e.liveHeld {
		e.renderer.RequestLive()
		e.liveHeld = true
	} else if !hasPlaying && e.liveHeld {
		e.renderer.DropLive()
		e.liveHeld = false
	}
}

// ResetEngine resets the global engine state. Intended for tests only.
func ResetEngine() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.liveHeld && engine.renderer != nil {
		engine.renderer.DropLive()
	}
	engine.timelines = nil
	engine.renderer = nil
	engine.attached = false
	engine.liveHeld = false
}
