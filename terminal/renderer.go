package terminal

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// Renderer owns terminal modes, event dispatch, frame rendering, and the scene
// graph rooted at RootRenderable.
type Renderer struct {
	root    *RootRenderable
	hitGrid *HitGrid
	backend *Backend
	config  Config

	currentBuffer *buffer.Buffer // The frame represented on screen.
	nextBuffer    *buffer.Buffer // The frame currently being built.

	width, height int

	controlState ControlState
	destroyed    bool

	// Input state.
	parser        *input.Parser
	eventBus      *input.EventBus
	inputHandlers []func(sequence string) bool // Raw-sequence preprocessors.

	// Focus and hover state.
	focused     render.Renderable
	lastHovered render.Renderable // The target of the previous mouse event.

	// Drag state.
	dragSource render.Renderable
	dragButton input.MouseButton
	dragging   bool

	// pendingWork provides back pressure for mutations. The one-slot wake channel
	// coalesces render requests, while dirty records that a frame is owed.
	pendingWork chan func()
	wake        chan struct{}
	stopChan    chan struct{}
	dirty       atomic.Bool

	// Post-processing callbacks.
	postProcessFns []func(*buffer.Buffer, float64)

	// Frame statistics.
	stats         Stats
	frameCount    uint64
	lastFrameTime time.Time

	// Cursor and pointer state.
	cursorX        int
	cursorY        int
	cursorVisible  bool
	cursorStyle    CursorStyle
	cursorBlinking bool
	cursorColor    style.Color
	mousePointer   MousePointerStyle

	// Text selection.
	sel *render.Selection

	// OSC 8 hyperlinks.
	linkRegistry *LinkRegistry

	// Live-render references.
	liveCount int32 // Accessed atomically.

	// Inline rendering state. The terminal holds every row of inlineCurrentBuf,
	// but only rows inlineViewportTop and below remain addressable; earlier rows
	// have scrolled into the scrollback.
	lastInlineLayoutWidth int            // The width used for the previous inline layout.
	inlineCurrentBuf      *buffer.Buffer // The frame represented by current inline output.
	inlineViewportTop     int            // The frame row shown on the terminal's first row.
	inlineCursorRow       int            // The frame row holding the terminal cursor.
	inlineTermHeight      int            // The terminal height used for the previous flush.
	inlineEraseNext       bool           // Whether the next flush erases the screen first.

	// Optional diagnostic log.
	dbg     *log.Logger
	dbgFile *os.File

	// Closed when Destroy completes.
	done chan struct{}

	mu          sync.Mutex
	invalidated bool // Whether the next fullscreen frame requires FullFlush.

	// Width-probe result from setup.
	probedWidths []int
	probeErr     error
}

// NewRenderer returns an idle renderer using the first supplied configuration.
func NewRenderer(config ...Config) (*Renderer, error) {
	cfg := mergeConfig(config)

	backend := NewBackend(cfg.Stdin, cfg.Stdout)

	// Use a conventional viewport when terminal dimensions are unavailable.
	w, h, err := backend.Size()
	if err != nil {
		w, h = 80, 24
	}

	// Fullscreen rendering compares the next frame with the current frame.
	currentBuffer := buffer.New(w, h)
	nextBuffer := buffer.New(w, h)

	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	parser := input.NewParser()
	eventBus := input.NewEventBus()

	r := &Renderer{
		root:           root,
		hitGrid:        hitGrid,
		backend:        backend,
		config:         cfg,
		currentBuffer:  currentBuffer,
		nextBuffer:     nextBuffer,
		width:          w,
		height:         h,
		controlState:   StateIdle,
		parser:         parser,
		eventBus:       eventBus,
		pendingWork:    make(chan func()),
		wake:           make(chan struct{}, 1),
		stopChan:       make(chan struct{}, 1),
		done:           make(chan struct{}),
		invalidated:    true,
		postProcessFns: cfg.PostProcessFns,
		stats:          NewStats(60),
		lastFrameTime:  time.Now(),
		sel:            render.NewSelection(),
		linkRegistry:   NewLinkRegistry(),
	}

	// Failure to open an optional debug log does not prevent renderer creation.
	if cfg.DebugLog != "" {
		f, err := os.OpenFile(cfg.DebugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			r.dbgFile = f
			r.dbg = log.New(f, "", log.Ltime|log.Lmicroseconds)
			r.dbg.Printf("=== fluid debug log opened  pid=%d ===", os.Getpid())
		}
	}

	// Widgets communicate with the renderer through the render.Context interface.
	root.SetContext(r)

	if cfg.UseKittyKeyboard {
		parser.SetKittyEnabled(true)
	}

	if cfg.AltSendsEscPrefix {
		parser.SetAltSendsEscPrefix(true)
	}

	return r, nil
}

// Root returns the scene graph root.
func (r *Renderer) Root() *RootRenderable { return r.root }

// Width returns the current terminal width in cells.
func (r *Renderer) Width() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.width
}

// Height returns the current terminal height in cells.
func (r *Renderer) Height() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.height
}

// IsRunning reports whether the renderer is in an active auto or explicit state.
func (r *Renderer) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.controlState == StateAutoStarted || r.controlState == StateExplicitStarted
}

// IsDestroyed reports whether Destroy was called.
func (r *Renderer) IsDestroyed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.destroyed
}

// ControlState returns the current lifecycle state.
func (r *Renderer) ControlState() ControlState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.controlState
}

// CurrentBuffer returns the fullscreen buffer representing terminal state.
func (r *Renderer) CurrentBuffer() *buffer.Buffer { return r.currentBuffer }

// NextBuffer returns the frame buffer used for the next draw.
func (r *Renderer) NextBuffer() *buffer.Buffer { return r.nextBuffer }

// EventBus returns the renderer's terminal event bus.
func (r *Renderer) EventBus() *input.EventBus { return r.eventBus }

// LinkRegistry returns the OSC 8 hyperlink registry.
func (r *Renderer) LinkRegistry() *LinkRegistry { return r.linkRegistry }

// FocusedRenderable returns the current keyboard-focus target.
func (r *Renderer) FocusedRenderable() render.Renderable {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.focused
}

// Done returns a channel closed when Destroy completes.
func (r *Renderer) Done() <-chan struct{} { return r.done }

// Backend returns the renderer's terminal backend.
func (r *Renderer) Backend() *Backend { return r.backend }

// ColorMode returns the terminal's color capability.
func (r *Renderer) ColorMode() style.ColorMode { return r.backend.ColorMode() }

// SetColorMode overrides the auto-detected color mode.
func (r *Renderer) SetColorMode(m style.ColorMode) { r.backend.SetColorMode(m) }

// Start sets up terminal modes and starts explicit rendering. In explicit mode,
// every event-loop wake renders a frame regardless of dirty state.
func (r *Renderer) Start() error {
	r.mu.Lock()
	if r.controlState != StateIdle && r.controlState != StateExplicitStopped {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	if err := r.setup(); err != nil {
		return err
	}

	r.mu.Lock()
	r.controlState = StateExplicitStarted
	r.mu.Unlock()

	go r.eventLoop()
	go r.inputLoop()
	go r.watchResize()

	return nil
}

// Auto sets up terminal modes and renders when RequestRender marks a frame due.
func (r *Renderer) Auto() error {
	r.mu.Lock()
	if r.controlState != StateIdle && r.controlState != StateExplicitStopped {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	if err := r.setup(); err != nil {
		return err
	}

	r.mu.Lock()
	r.controlState = StateAutoStarted
	r.mu.Unlock()

	go r.eventLoop()
	go r.inputLoop()
	go r.watchResize()

	return nil
}

// Run starts auto mode and waits for context cancellation, SIGINT, SIGTERM, or
// external destruction.
func (r *Renderer) Run(ctx context.Context) error {
	if err := r.Auto(); err != nil {
		return err
	}
	return r.Wait(ctx)
}

// RunContinuous starts explicit mode and waits for shutdown. Explicit mode
// renders on every event-loop wake.
func (r *Renderer) RunContinuous(ctx context.Context) error {
	if err := r.Start(); err != nil {
		return err
	}
	return r.Wait(ctx)
}

// Wait blocks until context cancellation, SIGINT, SIGTERM, or external
// destruction. It destroys a still-running renderer before returning.
func (r *Renderer) Wait(ctx context.Context) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	select {
	case <-ctx.Done():
	case <-sig:
	case <-r.done:
		return nil // Destroy already completed.
	}

	r.Destroy()
	return ctx.Err()
}

// Pause changes an active renderer to StateExplicitPaused.
func (r *Renderer) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.controlState == StateExplicitStarted || r.controlState == StateAutoStarted {
		r.controlState = StateExplicitPaused
	}
}

// Resume changes StateExplicitPaused to StateExplicitStarted.
func (r *Renderer) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.controlState == StateExplicitPaused {
		r.controlState = StateExplicitStarted
	}
}

// Suspend restores terminal modes and enters StateExplicitSuspended.
func (r *Renderer) Suspend() {
	r.mu.Lock()
	prevState := r.controlState
	r.controlState = StateExplicitSuspended
	r.mu.Unlock()

	_ = prevState

	// Restore each mode that setup may have enabled.
	r.backend.ShowCursor()
	if r.backend.mouseEnabled {
		r.backend.DisableMouse()
	}
	if r.backend.kittyKeyboardEnabled {
		r.backend.DisableKittyKeyboard()
	}
	if r.backend.alternateScreen {
		r.backend.LeaveAlternateScreen()
	}
	if r.backend.oldState != nil {
		_ = r.backend.Restore()
	}
}

// Stop marks the event loop stopped and sends its stop signal.
func (r *Renderer) Stop() {
	r.mu.Lock()
	if r.controlState == StateExplicitStopped || r.controlState == StateIdle {
		r.mu.Unlock()
		return
	}
	r.controlState = StateExplicitStopped
	r.mu.Unlock()

	// Do not block if a stop signal is already pending.
	select {
	case r.stopChan <- struct{}{}:
	default:
	}
}

// Destroy idempotently stops rendering, restores terminal modes, destroys the
// scene, clears subscriptions, and closes Done.
func (r *Renderer) Destroy() {
	r.mu.Lock()
	if r.destroyed {
		r.mu.Unlock()
		return
	}
	r.destroyed = true
	r.mu.Unlock()

	r.Stop()

	// Leave the cursor on a clean line below inline output.
	if r.config.InlineMode && r.inlineCurrentBuf != nil {
		var out bytes.Buffer
		inlineMoveRows(&out, r.inlineCurrentBuf.Height()-1-r.inlineCursorRow)
		out.WriteString("\r\n")
		r.backend.WriteOut(out.Bytes())
	}

	_ = r.backend.Restore()

	r.root.DestroyRecursively()

	r.eventBus.Clear()

	close(r.done)

	if r.config.OnDestroy != nil {
		r.config.OnDestroy()
	}

	// Record the final frame count before closing diagnostics.
	if r.dbg != nil {
		r.dbg.Printf("=== fluid debug log closed  frames=%d ===", atomic.LoadUint64(&r.frameCount))
		r.dbgFile.Close()
		r.dbg = nil
		r.dbgFile = nil
	}
}

// RequestRender marks one frame due and wakes the event loop. Repeated requests
// coalesce until the loop clears the dirty flag.
func (r *Renderer) RequestRender() {
	r.dirty.Store(true)
	r.signal()
}

// signal performs a nonblocking wake. An existing token already guarantees
// another event-loop iteration.
func (r *Renderer) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// HitTesting reports whether pointer targets are resolved against the scene.
//
// Inline mode does not hit test. Its frame is addressed in content rows, of
// which the terminal shows only the rows below the viewport, while a mouse
// report names a viewport row. Leaving the mouse to the terminal also keeps its
// own selection and scrollback bindings working.
func (r *Renderer) HitTesting() bool {
	return r.hitGrid != nil && !r.InlineMode()
}

// AddToHitGrid registers a clipped renderable area in the hit grid.
func (r *Renderer) AddToHitGrid(x, y, w, h int, id uint64) {
	if !r.HitTesting() {
		return
	}
	r.hitGrid.AddClipped(x, y, w, h, id)
}

// PushHitGridScissorRect restricts subsequent hit-grid registrations.
func (r *Renderer) PushHitGridScissorRect(x, y, w, h int) {
	if !r.HitTesting() {
		return
	}
	r.hitGrid.PushScissorRect(x, y, w, h)
}

// PopHitGridScissorRect restores the preceding hit-grid clipping region.
func (r *Renderer) PopHitGridScissorRect() {
	if !r.HitTesting() {
		return
	}
	r.hitGrid.PopScissorRect()
}

// ClearHitGrid clears the hit grid and all scissor rects.
func (r *Renderer) ClearHitGrid() {
	if !r.HitTesting() {
		return
	}
	r.hitGrid.Clear()
	r.hitGrid.ClearScissorRects()
}

// Selection returns the active text selection, or nil when inactive.
func (r *Renderer) Selection() *render.Selection {
	if r.sel.IsActive() {
		return r.sel
	}
	return nil
}

// StartSelection begins a selection at absolute terminal coordinates (x, y).
func (r *Renderer) StartSelection(_ render.Renderable, x, y int) {
	r.sel.Start(x, y)
}

// UpdateSelection moves the active focus to absolute terminal coordinates
// (x, y) and applies each option.
func (r *Renderer) UpdateSelection(_ render.Renderable, x, y int, opts ...render.SelectionUpdateOption) {
	r.sel.Update(x, y)
	for _, opt := range opts {
		if opt.FinishDragging {
			r.sel.SetDragging(false)
			r.sel.SetStart(false)
		}
	}
}

// ClearSelection clears the active selection.
func (r *Renderer) ClearSelection() {
	r.sel.Clear()
}

// RequestLive increments the live-render reference count and requests a frame.
func (r *Renderer) RequestLive() {
	atomic.AddInt32(&r.liveCount, 1)
	r.RequestRender()
}

// DropLive decrements the live-render reference count.
func (r *Renderer) DropLive() {
	atomic.AddInt32(&r.liveCount, -1)
}

// Enqueue sends fn to the event-loop goroutine before its next frame.
//
// Enqueue blocks until the loop receives fn or the renderer is destroyed. Do
// not call it from the event-loop goroutine, because that goroutine is the only
// receiver.
func (r *Renderer) Enqueue(fn func()) {
	select {
	case r.pendingWork <- fn:
	case <-r.done:
	}
}

// eventLoop serializes enqueued mutations, input dispatch, resize processing,
// and rendering.
func (r *Renderer) eventLoop() {
	for {
		// Wait for mutation work, a render wake, or shutdown.
		select {
		case fn := <-r.pendingWork:
			fn()
		case <-r.wake:
		case <-r.stopChan:
			return
		}

		// Apply all immediately available mutations before rendering.
		r.drainWork()

		r.mu.Lock()
		state := r.controlState
		r.mu.Unlock()

		switch state {
		case StateExplicitPaused, StateExplicitSuspended:
			continue
		case StateExplicitStopped:
			return
		}

		// Explicit mode renders after every wake, even without a dirty scene.
		if state == StateExplicitStarted {
			r.dirty.Store(true)
		}

		// Clear before drawing so a request made during the frame remains pending.
		if r.dirty.Swap(false) {
			r.renderFrame()
		}
	}
}

// drainWork runs queued callbacks until no callback is immediately available.
func (r *Renderer) drainWork() {
	for {
		select {
		case fn := <-r.pendingWork:
			fn()
		default:
			return
		}
	}
}

// processResize applies new terminal dimensions and emits EventResize. During
// normal operation it runs on the event-loop goroutine.
func (r *Renderer) processResize(w, h int) {
	r.mu.Lock()
	r.width = w
	r.height = h
	r.mu.Unlock()

	if r.dbg != nil {
		r.dbg.Printf("[resize] %dx%d", w, h)
	}

	if r.config.InlineMode {
		// Inline layout uses terminal width and content-derived height. A height
		// change is detected by the flush, which then rewrites every row.
		r.invalidated = true
		r.hitGrid.Resize(w, h)
		r.root.LayoutNode().Width = layout.Points(w)
		r.root.LayoutNode().MarkDirty()
	} else {
		// Fullscreen resize invalidates both frame buffers for a complete repaint.
		r.currentBuffer.Resize(w, h)
		r.nextBuffer.Resize(w, h)
		r.nextBuffer.Clear(r.config.BackgroundColor)
		r.invalidated = true

		r.hitGrid.Resize(w, h)
		r.root.Resize(w, h)
	}

	// Publish the applied dimensions after renderer state is current.
	r.eventBus.Emit(input.TerminalEvent{
		Type: input.EventResize,
		Resize: &input.ResizeEvent{
			Width:  w,
			Height: h,
		},
	})
}

// renderFrame renders and flushes one inline or fullscreen frame.
//
// Fullscreen rendering draws into nextBuffer, flushes it, copies it into
// currentBuffer, and clears nextBuffer. The buffers are not swapped.
func (r *Renderer) renderFrame() {
	if r.config.InlineMode {
		r.renderInlineFrame()
		return
	}

	frameStart := time.Now()

	r.mu.Lock()
	deltaTime := frameStart.Sub(r.lastFrameTime).Seconds()
	r.lastFrameTime = frameStart
	bgColor := r.config.BackgroundColor
	r.mu.Unlock()

	// Draw the scene into the next fullscreen frame.
	r.root.Render(r.nextBuffer, deltaTime)

	if r.dbg != nil && r.invalidated {
		fc := atomic.LoadUint64(&r.frameCount)
		r.dbg.Printf("[frame] #%d  buf=%dx%d", fc, r.nextBuffer.Width(), r.nextBuffer.Height())
	}

	// Post-processing modifies the complete frame before flushing.
	r.mu.Lock()
	fns := make([]func(*buffer.Buffer, float64), len(r.postProcessFns))
	copy(fns, r.postProcessFns)
	r.mu.Unlock()

	for _, fn := range fns {
		fn(r.nextBuffer, deltaTime)
	}

	// The first frame and resized frames require a complete repaint.
	tFlush0 := time.Now()
	if r.invalidated {
		// Clear stale alternate-screen cells before a complete repaint.
		if !r.config.InlineMode {
			r.backend.Clear()
		}
		r.FullFlush(r.nextBuffer)
		r.invalidated = false
	} else {
		r.DiffAndFlush(r.currentBuffer, r.nextBuffer)
	}

	// Record flush and total frame durations.
	r.root.LastFrameStats.Flush = time.Since(tFlush0)
	r.root.LastFrameStats.Total = time.Since(frameStart)

	// Preserve terminal state for the next diff, then prepare the draw buffer.
	r.currentBuffer.CopyFrom(r.nextBuffer)
	r.nextBuffer.Clear(bgColor)

	// Apply hardware cursor state after cell output.
	r.mu.Lock()
	cursorX := r.cursorX
	cursorY := r.cursorY
	cursorVisible := r.cursorVisible
	r.mu.Unlock()

	if cursorVisible {
		r.backend.SetCursorPosition(cursorX, cursorY, true)
	} else {
		r.backend.HideCursor()
	}

	atomic.AddUint64(&r.frameCount, 1)

	// Aggregate frame duration only when configured.
	if r.config.GatherStats {
		renderTime := time.Since(frameStart).Seconds()
		r.mu.Lock()
		r.stats.record(renderTime)
		r.mu.Unlock()
	}
}

// FocusRenderable makes renderable the keyboard dispatch target and blurs the
// previous target. It also invokes Focus when the new target is focusable.
//
// An embedded BaseRenderable is resolved to its registered outer widget.
// Numeric ID equality prevents the reentrant Focus call from transferring focus
// again.
func (r *Renderer) FocusRenderable(renderable render.Renderable) {
	if renderable == nil {
		return
	}

	// Resolve embedded values before storing or invoking widget methods.
	actual := render.LookupRenderable(renderable.Num())
	if actual == nil {
		actual = renderable
	}

	r.mu.Lock()
	prev := r.focused
	if prev != nil && prev.Num() == actual.Num() {
		r.mu.Unlock()
		return
	}
	r.focused = actual
	r.mu.Unlock()

	// Blur the previous target after publishing the new target under the lock.
	if prev != nil && !prev.IsDestroyed() {
		prev.Blur()
	}

	// Focus updates the widget's internal state and reenters this method; the
	// numeric-ID guard handles that call.
	if actual != nil && !actual.IsDestroyed() && actual.Focusable() {
		actual.Focus()
		// Retain the resolved outer widget after the reentrant call.
		r.mu.Lock()
		r.focused = actual
		r.mu.Unlock()
	}

	r.RequestRender()
}

// AddInputHandler appends a raw-input preprocessor. A preprocessor returning
// true prevents later input handling.
func (r *Renderer) AddInputHandler(handler func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputHandlers = append(r.inputHandlers, handler)
}

// PrependInputHandler inserts a raw-input preprocessor before existing handlers.
func (r *Renderer) PrependInputHandler(handler func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputHandlers = append([]func(string) bool{handler}, r.inputHandlers...)
}

// RemoveInputHandler currently does nothing because Go function values cannot
// be compared for identity.
func (r *Renderer) RemoveInputHandler(handler func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
}

// SetCursorPosition records zero-based hardware cursor coordinates and
// visibility for the next frame.
func (r *Renderer) SetCursorPosition(x, y int, visible bool) {
	r.mu.Lock()
	r.cursorX = x
	r.cursorY = y
	r.cursorVisible = visible
	r.mu.Unlock()
}

// SetCursorStyle records and immediately writes the cursor shape.
func (r *Renderer) SetCursorStyle(cursorStyle string) {
	r.mu.Lock()
	r.cursorStyle = cursorStyle
	r.mu.Unlock()

	r.backend.SetCursorStyle(cursorStyle)
}

// SetCursorColor records color and writes its RGB channels using OSC 12.
func (r *Renderer) SetCursorColor(color style.Color) {
	r.mu.Lock()
	r.cursorColor = color
	r.mu.Unlock()

	r.backend.SetCursorColor(color.R, color.G, color.B)
}

// CursorState returns a snapshot of the configured cursor state.
func (r *Renderer) CursorState() CursorState {
	r.mu.Lock()
	defer r.mu.Unlock()

	return CursorState{
		X:        r.cursorX,
		Y:        r.cursorY,
		Visible:  r.cursorVisible,
		Style:    r.cursorStyle,
		Blinking: r.cursorBlinking,
		Color:    r.cursorColor,
	}
}

// SetCursorStyleOptions applies each non-nil cursor or pointer option.
func (r *Renderer) SetCursorStyleOptions(options CursorStyleOptions) {
	if options.Style != nil {
		r.mu.Lock()
		r.cursorStyle = *options.Style
		r.mu.Unlock()
		r.backend.SetCursorStyle(*options.Style)
	}

	if options.Blinking != nil {
		r.mu.Lock()
		r.cursorBlinking = *options.Blinking
		r.mu.Unlock()
		r.backend.SetCursorBlinking(*options.Blinking)
	}

	if options.Color != nil {
		r.SetCursorColor(*options.Color)
	}

	if options.Cursor != nil {
		r.SetMousePointer(*options.Cursor)
	}
}

// SetMousePointer records and writes pointerStyle using OSC 22.
func (r *Renderer) SetMousePointer(pointerStyle MousePointerStyle) {
	r.mu.Lock()
	r.mousePointer = pointerStyle
	r.mu.Unlock()

	r.backend.SetMousePointer(pointerStyle)
}

// SetBackgroundColor resolves ci, sets the frame background, and requests a
// repaint.
func (r *Renderer) SetBackgroundColor(ci style.ColorInput) {
	c := style.ParseColor(ci)
	r.mu.Lock()
	r.config.BackgroundColor = c
	r.mu.Unlock()
	r.RequestRender()
}

// SetTitle sets the terminal window title.
func (r *Renderer) SetTitle(title string) {
	r.backend.SetTitle(title)
}

// AddPostProcessFn appends a callback run after scene rendering and before
// flushing.
func (r *Renderer) AddPostProcessFn(fn func(*buffer.Buffer, float64)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postProcessFns = append(r.postProcessFns, fn)
}

// RemovePostProcessFn currently does nothing because Go function values cannot
// be compared for identity.
func (r *Renderer) RemovePostProcessFn(fn func(*buffer.Buffer, float64)) {
}

// ClearPostProcessFns removes all post-process functions.
func (r *Renderer) ClearPostProcessFns() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postProcessFns = nil
}

// Stats returns a snapshot of aggregate rendering statistics.
func (r *Renderer) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// ResetStats clears aggregate frame timings and counts.
func (r *Renderer) ResetStats() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.reset()
}

// SetGatherStats controls aggregate frame-duration collection.
func (r *Renderer) SetGatherStats(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config.GatherStats = enabled
}

// setup initializes terminal modes, optional protocols and probing, and current
// terminal dimensions.
func (r *Renderer) setup() error {
	if err := r.backend.Setup(r.config.Fullscreen); err != nil {
		return err
	}

	// Inline mode does not hit test, so mouse reports would be discarded while
	// suppressing the terminal's own selection and scrolling.
	if r.config.UseMouse && !r.config.InlineMode {
		r.backend.EnableMouse(r.config.EnableMouseMovement)
	}

	if r.config.UseKittyKeyboard {
		r.backend.EnableKittyKeyboard(r.config.KittyKeyboardFlags)
	}

	if r.config.InlineMode {
		// Inline mode retains normal scrollback and draws a visual cursor in cells.
		r.backend.HideCursor()
		r.inlineEraseNext = r.config.ClearScreenOnStart
	} else {
		// Non-inline rendering begins from a cleared screen.
		r.backend.Clear()
	}

	// Probe before inputLoop starts so cursor reports have a single reader.
	r.probeWidths()

	// Apply dimensions after terminal modes and probing are configured.
	if w, h, err := r.backend.Size(); err == nil {
		r.processResize(w, h)
	}

	return nil
}

// probeWidths records measurements or the reason measurement was unavailable.
func (r *Renderer) probeWidths() {
	if r.config.WidthProbe != WidthProbeAuto || len(r.config.WidthProbeSamples) == 0 {
		return
	}
	widths, err := r.backend.ProbeWidths(r.config.WidthProbeSamples)

	r.mu.Lock()
	r.probedWidths, r.probeErr = widths, err
	r.mu.Unlock()

	if r.dbg != nil {
		if err != nil {
			r.dbg.Printf("[probe] unavailable: %v", err)
		} else {
			r.dbg.Printf("[probe] measured %v", widths)
		}
	}
}

// ProbedWidths returns measured columns in sample order. A non-nil error
// explains a failed requested probe; two nil results mean no probe was requested.
func (r *Renderer) ProbedWidths() (widths []int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.probedWidths), r.probeErr
}
