package animation

import (
	"slices"
	"sync"
)

// RendererLike provides the live-rendering reference count used by TimelineEngine.
// This narrow interface avoids importing the render package.
type RendererLike interface {
	// RequestLive increments the renderer's live-rendering reference count.
	RequestLive()
	// DropLive decrements the renderer's live-rendering reference count.
	DropLive()
}

// TimelineEngine stores registered timelines and advances them on renderer frames.
type TimelineEngine struct {
	timelines []*Timeline
	renderer  RendererLike
	attached  bool
	liveHeld  bool // Whether this engine currently holds one live-rendering reference.
	mu        sync.Mutex
}

// engine is the package-wide timeline registry used by the package functions.
var engine = &TimelineEngine{}

// Attach binds the package engine to renderer.
// The renderer must call Update with each frame's elapsed time.
func Attach(renderer RendererLike) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.renderer = renderer
	engine.attached = true
}

// Detach releases any live-rendering reference and unbinds the renderer.
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

// Register adds tl to the package engine unless it is already registered.
func Register(tl *Timeline) {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	// A timeline receives at most one update per engine frame.
	if slices.Contains(engine.timelines, tl) {
		return
	}

	engine.timelines = append(engine.timelines, tl)
	engine.updateLiveState()
}

// Unregister removes the first registration of tl, if present.
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

// Update advances each registered, unsynchronized timeline by deltaTime seconds.
// It then removes completed timelines and updates live-rendering state.
func Update(deltaTime float64) {
	engine.mu.Lock()
	// Release the engine lock before timeline callbacks can register or unregister timelines.
	tls := make([]*Timeline, len(engine.timelines))
	copy(tls, engine.timelines)
	engine.mu.Unlock()

	for _, tl := range tls {
		if tl.IsSynced() {
			continue
		}
		tl.Update(deltaTime)
	}

	// Compact the registry in place after callbacks and property updates finish.
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

// CreateTimeline returns a new Timeline registered with the package engine.
// Playback state follows NewTimeline's autoplay rules.
func CreateTimeline(opts ...TimelineOptions) *Timeline {
	tl := NewTimeline(opts...)
	Register(tl)
	return tl
}

// ActiveCount returns the number of registered timelines, including paused timelines.
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

// updateLiveState holds one live-rendering reference while any timeline is playing.
// The caller must hold e.mu.
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

// ResetEngine releases live rendering and clears all package engine state.
// It is intended for tests.
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
