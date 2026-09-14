package animation

import (
	"fmt"
	"strings"
	"sync"
)

// AnimationTarget exposes named floating-point properties that a Timeline can animate.
type AnimationTarget interface {
	Property(name string) float64
	SetProperty(name string, value float64)
}

// TimelineOptions configures timeline-wide duration, playback, and callbacks.
type TimelineOptions struct {
	Duration   float64 // Total seconds; non-positive values use the latest item end.
	Loop       bool    // Restarts the timeline when its duration elapses.
	Autoplay   bool    // If true, a configured timeline starts playing immediately.
	OnComplete func()
	OnPause    func()
}

// LoopConfig controls how an animation loops.
// The zero value means no looping.
type LoopConfig struct {
	infinite bool
	count    int
}

// LoopInfinite returns a configuration with no iteration limit.
func LoopInfinite() LoopConfig { return LoopConfig{infinite: true} }

// LoopCount limits an animation to n iterations when n is positive.
// A non-positive value disables looping.
func LoopCount(n int) LoopConfig { return LoopConfig{count: n} }

// AnimationOptions configures one property animation on a Timeline.
type AnimationOptions struct {
	Duration  float64    // Seconds; non-positive values become 0.001.
	Ease      EasingName // Empty selects Linear.
	Loop      LoopConfig // Zero runs one iteration.
	LoopDelay float64    // Seconds between iterations.
	Alternate bool       // Reverses progress on every second iteration.
	Once      bool       // Removes the item after its final iteration.

	OnUpdate   func(progress float64)
	OnComplete func()
	OnStart    func()
	OnLoop     func()

	// Properties maps target property names to their final values.
	Properties map[string]float64
}

// Timeline item state.

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

// Timeline coordinates property animations, timed callbacks, and nested timelines.
type Timeline struct {
	items        []timelineItem
	callbacks    []callbackItem
	subTimelines []subTimelineItem
	currentTime  float64 // Elapsed playback seconds.
	duration     float64 // Explicit duration or latest scheduled end time, in seconds.
	autoDuration bool    // Whether additions recalculate duration.
	isPlaying    bool
	isComplete   bool
	loop         bool
	synced       bool // Whether a parent Timeline drives updates instead of TimelineEngine.

	onComplete func()
	onPause    func()

	stateListeners []func(*Timeline)
	mu             sync.Mutex
}

// NewTimeline returns a Timeline configured by the first options value.
// With no options it starts playing; with options it plays only when Autoplay is true.
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

		// A supplied options value uses its Autoplay field directly.
		if !o.Autoplay {
			// A false Autoplay value starts the configured timeline paused.
			tl.isPlaying = false
		} else {
			tl.isPlaying = true
		}
	} else {
		// The no-options constructor starts immediately.
		tl.isPlaying = true
	}

	return tl
}

// Add schedules an animation for target at an optional absolute start time.
// The default start time is zero. Add returns tl for chaining.
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

// AddAt schedules an animation using a string start time and returns tl.
// A "+=0.5" or "-=0.2" value is relative to the current timeline end;
// "0.3" is an absolute time in seconds.
func (tl *Timeline) AddAt(target AnimationTarget, opts AnimationOptions, startTimeStr string) *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	st := tl.parseStartTime(startTimeStr)
	item := tl.makeItem(target, opts, st)
	tl.items = append(tl.items, item)
	tl.recalcDuration()
	return tl
}

// Once schedules an animation at the current time and removes it after completion.
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

// Call schedules callback at an optional absolute time and returns tl.
// The default time is zero.
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

// Sync schedules sub at an optional absolute time and returns tl.
// The parent timeline drives sub instead of TimelineEngine.
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

// Playback control.

// Play starts or resumes playback and returns tl.
// A completed timeline restarts from time zero.
func (tl *Timeline) Play() *Timeline {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.isComplete {
		// Playing a completed timeline starts a fresh pass.
		tl.resetItemsLocked()
		tl.currentTime = 0
		tl.isComplete = false
	}
	tl.isPlaying = true
	tl.notifyStateListenersLocked()
	return tl
}

// Pause stops time advancement, invokes OnPause, and returns tl.
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

// Restart resets all items and callbacks, starts at time zero, and returns tl.
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

// Playback state.

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

// State-change listeners.

// AddStateChangeListener adds a callback for play, pause, restart, loop, and completion changes.
func (tl *Timeline) AddStateChangeListener(listener func(*Timeline)) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.stateListeners = append(tl.stateListeners, listener)
}

// RemoveStateChangeListener removes the first listener with the same function pointer.
func (tl *Timeline) RemoveStateChangeListener(listener func(*Timeline)) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	// Functions are not directly comparable, so compare their formatted code pointers.
	target := fmt.Sprintf("%p", listener)
	for i, l := range tl.stateListeners {
		if fmt.Sprintf("%p", l) == target {
			tl.stateListeners = append(tl.stateListeners[:i], tl.stateListeners[i+1:]...)
			return
		}
	}
}

// Update advances a playing timeline by deltaTime seconds.
// TimelineEngine calls it for root timelines; a parent calls it for synchronized timelines.
func (tl *Timeline) Update(deltaTime float64) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.isPlaying || tl.isComplete {
		return
	}

	tl.currentTime += deltaTime

	// Apply due property animations before callbacks observe this frame.
	tl.processItems()

	// Fire callbacks whose scheduled times have elapsed.
	tl.processCallbacks()

	// Advance nested timelines after their start times.
	tl.processSubTimelines(deltaTime)

	// Discard completed one-shot items.
	tl.removeOnceItems()

	// Complete or restart after the timeline-wide duration elapses.
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

// ResetItems clears item, callback, and nested-timeline progress without changing current time.
func (tl *Timeline) ResetItems() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.resetItemsLocked()
}

// Timeline scheduling and update helpers.

func (tl *Timeline) makeItem(target AnimationTarget, opts AnimationOptions, startTime float64) timelineItem {
	// Store end values now and capture start values when the item begins.
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
		opts.Duration = 0.001 // Keep progress finite while approximating an instant update.
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

// calculatedEnd returns the latest scheduled end across items, callbacks, and sub-timelines.
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

// recalcDuration extends an automatically sized timeline to its latest scheduled end.
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

		// Capture current target values when the item first becomes due.
		if !item.started {
			for name, pa := range item.properties {
				pa.startValue = item.target.Property(name)
				pa.easingFunc = GetEasingFunc(item.options.Ease)
				_ = name
			}
			item.started = true
			if item.options.OnStart != nil {
				item.options.OnStart()
			}
		}

		// Clamp elapsed progress at completion.
		elapsed := tl.currentTime - item.startTime
		rawProgress := elapsed / item.options.Duration
		if rawProgress > 1.0 {
			rawProgress = 1.0
		}

		// Odd alternate iterations run from end to start.
		effectiveProgress := rawProgress
		if item.options.Alternate && item.loopCount%2 == 1 {
			effectiveProgress = 1.0 - rawProgress
		}

		// Interpolate every target property with the selected easing function.
		for _, pa := range item.properties {
			easedProgress := pa.easingFunc(effectiveProgress)
			value := pa.startValue + (pa.endValue-pa.startValue)*easedProgress
			item.target.SetProperty(findPropertyName(item.properties, pa), value)
		}

		if item.options.OnUpdate != nil {
			// Report eased progress from the first configured property.
			var progress float64
			for _, pa := range item.properties {
				progress = pa.easingFunc(effectiveProgress)
				break
			}
			item.options.OnUpdate(progress)
		}

		// Either schedule the next iteration or finalize this item.
		if rawProgress >= 1.0 {
			if tl.shouldLoop(item) {
				item.loopCount++
				if item.options.OnLoop != nil {
					item.options.OnLoop()
				}
				// LoopDelay begins after the frame that completed this iteration.
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

// findPropertyName returns the property name associated with target.
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
		// Nested timelines receive the parent's complete frame delta.
		sub.timeline.Update(deltaTime)
	}
}

func (tl *Timeline) removeOnceItems() {
	n := 0
	for _, item := range tl.items {
		if item.once && item.complete {
			continue // Omit completed one-shot items during in-place compaction.
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
