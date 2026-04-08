package animation

import (
	"fmt"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// AnimationTarget
// ---------------------------------------------------------------------------

// AnimationTarget is any object whose float64 properties can be animated.
// Widgets expose animatable properties via this interface.
type AnimationTarget interface {
	Property(name string) float64
	SetProperty(name string, value float64)
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

// TimelineOptions configures a Timeline at construction time.
type TimelineOptions struct {
	Duration   float64 // total duration in seconds; default: 0 (auto-calculated)
	Loop       bool    // loop when complete; default: false
	Autoplay   bool    // start playing immediately; default: true
	OnComplete func()
	OnPause    func()
}

// LoopConfig controls how an animation loops.
// The zero value means no looping.
type LoopConfig struct {
	infinite bool
	count    int
}

// LoopInfinite returns a LoopConfig that loops forever.
func LoopInfinite() LoopConfig { return LoopConfig{infinite: true} }

// LoopCount returns a LoopConfig that loops n times.
func LoopCount(n int) LoopConfig { return LoopConfig{count: n} }

// AnimationOptions configures a single animation added to a Timeline.
type AnimationOptions struct {
	Duration  float64    // seconds
	Ease      EasingName // default: "linear"
	Loop      LoopConfig // default: no looping
	LoopDelay float64    // seconds between loops
	Alternate bool       // reverse on alternate loops
	Once      bool       // remove after completion

	OnUpdate   func(progress float64)
	OnComplete func()
	OnStart    func()
	OnLoop     func()

	// Target property values (end values).
	Properties map[string]float64
}

// ---------------------------------------------------------------------------
// Internal items
// ---------------------------------------------------------------------------

type propertyAnimation struct {
	startValue float64
	endValue   float64
	easingFunc EasingFunc
}

type timelineItem struct {
	target     AnimationTarget
	properties map[string]*propertyAnimation
	options    AnimationOptions
	startTime  float64
	started    bool
	complete   bool
	once       bool
	loopCount  int
}

type callbackItem struct {
	callback  func()
	startTime float64
	fired     bool
}

type subTimelineItem struct {
	timeline  *Timeline
	startTime float64
}

// ---------------------------------------------------------------------------
// Timeline
// ---------------------------------------------------------------------------

// Timeline drives property-based animations and timed callbacks.
type Timeline struct {
	items        []timelineItem
	callbacks    []callbackItem
	subTimelines []subTimelineItem
	currentTime  float64 // seconds
	duration     float64 // total duration in seconds (explicit or auto)
	autoDuration bool    // true if duration was not set explicitly
	isPlaying    bool
	isComplete   bool
	loop         bool
	synced       bool // if true, driven by parent timeline, not engine

	onComplete func()
	onPause    func()

	stateListeners []func(*Timeline)
	mu             sync.Mutex
}

// NewTimeline creates a Timeline with the given options.
func NewTimeline(opts ...TimelineOptions) *Timeline {
	tl := &Timeline{
		autoDuration: true,
	}

	if len(opts) > 0 {
		o := opts[0]
		if o.Duration > 0 {
			tl.duration = o.Duration
			tl.autoDuration = false
		}
		tl.loop = o.Loop
		tl.onComplete = o.OnComplete
		tl.onPause = o.OnPause

		// Default Autoplay = true (zero-value bool is false, so we check
		// whether the caller explicitly passed false vs. not setting it).
		// Convention: if TimelineOptions is provided, Autoplay defaults to
		// true unless explicitly set to false.
		if !o.Autoplay {
			// Autoplay was explicitly false — start paused.
			tl.isPlaying = false
		} else {
			tl.isPlaying = true
		}
	} else {
		// No options: autoplay by default.
		tl.isPlaying = true
	}

	return tl
}

// ---------------------------------------------------------------------------
// Adding animations
// ---------------------------------------------------------------------------

// Add adds an animation for the given target at an optional start time
// (default: 0). Returns the Timeline for chaining.
func (tl *Timeline) Add(target AnimationTarget, opts AnimationOptions, startTime ...float64) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	st := 0.0
	if len(startTime) > 0 {
		st = startTime[0]
	}

	item := tl.makeItem(target, opts, st)
	tl.items = append(tl.items, item)
	tl.recalcDuration()
	return tl
}

// AddAt adds an animation with a string start time. Supports:
//   - "+=0.5" — relative offset from current timeline end
//   - "-=0.2" — relative offset backward from end
//   - "0.3"   — absolute time in seconds
func (tl *Timeline) AddAt(target AnimationTarget, opts AnimationOptions, startTimeStr string) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	st := tl.parseStartTime(startTimeStr)
	item := tl.makeItem(target, opts, st)
	tl.items = append(tl.items, item)
	tl.recalcDuration()
	return tl
}

// Once adds a one-shot animation at the current time. The animation is
// removed after it completes.
func (tl *Timeline) Once(target AnimationTarget, opts AnimationOptions) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	opts.Once = true
	item := tl.makeItem(target, opts, tl.currentTime)
	item.once = true
	tl.items = append(tl.items, item)
	tl.recalcDuration()
	return tl
}

// Call schedules a callback at the given time (default: 0).
func (tl *Timeline) Call(callback func(), startTime ...float64) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	st := 0.0
	if len(startTime) > 0 {
		st = startTime[0]
	}

	tl.callbacks = append(tl.callbacks, callbackItem{
		callback:  callback,
		startTime: st,
	})
	tl.recalcDuration()
	return tl
}

// Sync nests a sub-timeline at the given start time (default: 0).
// The sub-timeline is driven by this timeline, not the engine.
func (tl *Timeline) Sync(sub *Timeline, startTime ...float64) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	st := 0.0
	if len(startTime) > 0 {
		st = startTime[0]
	}

	sub.synced = true
	sub.isPlaying = true
	tl.subTimelines = append(tl.subTimelines, subTimelineItem{
		timeline:  sub,
		startTime: st,
	})
	tl.recalcDuration()
	return tl
}

// ---------------------------------------------------------------------------
// Playback control
// ---------------------------------------------------------------------------

// Play starts or resumes playback. Returns the Timeline for chaining.
func (tl *Timeline) Play() *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.isComplete {
		// Reset if trying to play after completion
		tl.resetItemsLocked()
		tl.currentTime = 0
		tl.isComplete = false
	}
	tl.isPlaying = true
	tl.notifyStateListenersLocked()
	return tl
}

// Pause pauses playback. Returns the Timeline for chaining.
func (tl *Timeline) Pause() *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.isPlaying = false
	if tl.onPause != nil {
		tl.onPause()
	}
	tl.notifyStateListenersLocked()
	return tl
}

// Restart resets the timeline to the beginning and starts playback.
// Returns the Timeline for chaining.
func (tl *Timeline) Restart() *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.currentTime = 0
	tl.isComplete = false
	tl.isPlaying = true
	tl.resetItemsLocked()
	tl.notifyStateListenersLocked()
	return tl
}

// ---------------------------------------------------------------------------
// State queries
// ---------------------------------------------------------------------------

// IsPlaying returns whether the timeline is currently playing.
func (tl *Timeline) IsPlaying() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.isPlaying
}

// IsComplete returns whether the timeline has finished.
func (tl *Timeline) IsComplete() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.isComplete
}

// CurrentTime returns the current playback time in seconds.
func (tl *Timeline) CurrentTime() float64 {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.currentTime
}

// Duration returns the total duration in seconds.
func (tl *Timeline) Duration() float64 {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.duration
}

// IsSynced returns whether this timeline is driven by a parent timeline.
func (tl *Timeline) IsSynced() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.synced
}

// ---------------------------------------------------------------------------
// State change listeners
// ---------------------------------------------------------------------------

// AddStateChangeListener subscribes to state changes (play/pause/complete).
func (tl *Timeline) AddStateChangeListener(listener func(*Timeline)) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.stateListeners = append(tl.stateListeners, listener)
}

// RemoveStateChangeListener unsubscribes from state changes.
// Uses function pointer comparison (caller should keep a reference).
func (tl *Timeline) RemoveStateChangeListener(listener func(*Timeline)) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	// Function comparison in Go is limited; we compare the formatted pointer.
	target := fmt.Sprintf("%p", listener)
	for i, l := range tl.stateListeners {
		if fmt.Sprintf("%p", l) == target {
			tl.stateListeners = append(tl.stateListeners[:i], tl.stateListeners[i+1:]...)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Update (called each frame)
// ---------------------------------------------------------------------------

// Update advances the timeline by deltaTime seconds.
// Called by the engine each frame (or by a parent timeline if synced).
func (tl *Timeline) Update(deltaTime float64) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.isPlaying || tl.isComplete {
		return
	}

	tl.currentTime += deltaTime

	// Process animation items
	tl.processItems()

	// Process callbacks
	tl.processCallbacks()

	// Process sub-timelines
	tl.processSubTimelines(deltaTime)

	// Remove completed once-items
	tl.removeOnceItems()

	// Check overall completion
	if tl.duration > 0 && tl.currentTime >= tl.duration {
		if tl.loop {
			tl.currentTime = 0
			tl.resetItemsLocked()
		} else {
			tl.isComplete = true
			tl.isPlaying = false
			if tl.onComplete != nil {
				tl.onComplete()
			}
		}
		tl.notifyStateListenersLocked()
	}
}

// ResetItems resets all items to their initial state.
func (tl *Timeline) ResetItems() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.resetItemsLocked()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (tl *Timeline) makeItem(target AnimationTarget, opts AnimationOptions, startTime float64) timelineItem {
	// Build property animation map
	props := make(map[string]*propertyAnimation, len(opts.Properties))
	for name, endVal := range opts.Properties {
		props[name] = &propertyAnimation{
			endValue: endVal,
		}
	}

	if opts.Ease == "" {
		opts.Ease = Linear
	}
	if opts.Duration <= 0 {
		opts.Duration = 0.001 // Avoid division by zero; near-instant
	}

	return timelineItem{
		target:     target,
		properties: props,
		options:    opts,
		startTime:  startTime,
		once:       opts.Once,
	}
}

func (tl *Timeline) parseStartTime(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "+=") {
		var offset float64
		fmt.Sscanf(s[2:], "%f", &offset)
		return tl.calculatedEnd() + offset
	}
	if strings.HasPrefix(s, "-=") {
		var offset float64
		fmt.Sscanf(s[2:], "%f", &offset)
		end := tl.calculatedEnd()
		result := end - offset
		if result < 0 {
			result = 0
		}
		return result
	}
	var t float64
	fmt.Sscanf(s, "%f", &t)
	return t
}

// calculatedEnd returns the latest end time across all items/callbacks/subs.
func (tl *Timeline) calculatedEnd() float64 {
	end := 0.0
	for _, item := range tl.items {
		e := item.startTime + item.options.Duration
		if e > end {
			end = e
		}
	}
	for _, cb := range tl.callbacks {
		if cb.startTime > end {
			end = cb.startTime
		}
	}
	for _, sub := range tl.subTimelines {
		e := sub.startTime + sub.timeline.Duration()
		if e > end {
			end = e
		}
	}
	return end
}

// recalcDuration updates the duration if it was auto-calculated.
func (tl *Timeline) recalcDuration() {
	if tl.autoDuration {
		tl.duration = tl.calculatedEnd()
	}
}

func (tl *Timeline) processItems() {
	for i := range tl.items {
		item := &tl.items[i]
		if tl.currentTime < item.startTime {
			continue
		}
		if item.complete {
			continue
		}

		// Initialize start values on first frame
		if !item.started {
			for name, pa := range item.properties {
				pa.startValue = item.target.Property(name)
				pa.easingFunc = GetEasingFunc(item.options.Ease)
				_ = name // iterate
			}
			item.started = true
			if item.options.OnStart != nil {
				item.options.OnStart()
			}
		}

		// Calculate progress
		elapsed := tl.currentTime - item.startTime
		rawProgress := elapsed / item.options.Duration
		if rawProgress > 1.0 {
			rawProgress = 1.0
		}

		// Handle alternating loops
		effectiveProgress := rawProgress
		if item.options.Alternate && item.loopCount%2 == 1 {
			effectiveProgress = 1.0 - rawProgress
		}

		// Apply easing and update properties
		for _, pa := range item.properties {
			easedProgress := pa.easingFunc(effectiveProgress)
			value := pa.startValue + (pa.endValue-pa.startValue)*easedProgress
			item.target.SetProperty(findPropertyName(item.properties, pa), value)
		}

		if item.options.OnUpdate != nil {
			// Use the first property's easing for the callback progress
			var progress float64
			for _, pa := range item.properties {
				progress = pa.easingFunc(effectiveProgress)
				break
			}
			item.options.OnUpdate(progress)
		}

		// Handle completion
		if rawProgress >= 1.0 {
			if tl.shouldLoop(item) {
				item.loopCount++
				if item.options.OnLoop != nil {
					item.options.OnLoop()
				}
				// Reset start time for next loop iteration
				item.startTime = tl.currentTime + item.options.LoopDelay
				item.started = false
			} else {
				item.complete = true
				if item.options.OnComplete != nil {
					item.options.OnComplete()
				}
			}
		}
	}
}

// findPropertyName returns the map key for a given *propertyAnimation value.
func findPropertyName(props map[string]*propertyAnimation, target *propertyAnimation) string {
	for name, pa := range props {
		if pa == target {
			return name
		}
	}
	return ""
}

func (tl *Timeline) shouldLoop(item *timelineItem) bool {
	loop := item.options.Loop
	if loop.infinite {
		return true
	}
	if loop.count > 0 {
		return item.loopCount < loop.count-1
	}
	return false
}

func (tl *Timeline) processCallbacks() {
	for i := range tl.callbacks {
		cb := &tl.callbacks[i]
		if cb.fired {
			continue
		}
		if tl.currentTime >= cb.startTime {
			cb.fired = true
			if cb.callback != nil {
				cb.callback()
			}
		}
	}
}

func (tl *Timeline) processSubTimelines(deltaTime float64) {
	for _, sub := range tl.subTimelines {
		if tl.currentTime < sub.startTime {
			continue
		}
		// Drive the sub-timeline with the same deltaTime
		sub.timeline.Update(deltaTime)
	}
}

func (tl *Timeline) removeOnceItems() {
	n := 0
	for _, item := range tl.items {
		if item.once && item.complete {
			continue // skip completed once-items
		}
		tl.items[n] = item
		n++
	}
	tl.items = tl.items[:n]
}

func (tl *Timeline) resetItemsLocked() {
	for i := range tl.items {
		tl.items[i].started = false
		tl.items[i].complete = false
		tl.items[i].loopCount = 0
	}
	for i := range tl.callbacks {
		tl.callbacks[i].fired = false
	}
	for _, sub := range tl.subTimelines {
		sub.timeline.Restart()
	}
}

func (tl *Timeline) notifyStateListenersLocked() {
	for _, l := range tl.stateListeners {
		l(tl)
	}
}
