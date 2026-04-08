package terminal

import (
	"context"
	"log"
	"os"
	"os/signal"
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

// ---------------------------------------------------------------------------
// Renderer
// ---------------------------------------------------------------------------

// Renderer is the top-level terminal renderer. It owns the render loop,
// terminal setup/teardown, input parsing, focus management, mouse dispatch,
// text selection, and the widget tree rooted at a RootRenderable. Create
// one with NewRenderer, attach widgets via Root().Add(), then call Start()
// or Auto() to begin rendering.
type Renderer struct {
	root    *RootRenderable
	hitGrid *HitGrid
	backend *Backend
	config  Config

	currentBuffer *buffer.Buffer // currently displayed frame
	nextBuffer    *buffer.Buffer // being drawn into

	width, height int

	controlState ControlState
	destroyed    bool

	// Input
	parser        *input.Parser
	eventBus      *input.EventBus
	inputHandlers []func(sequence string) bool // pre-processors

	// Focus
	focused     render.Renderable
	lastHovered render.Renderable // for over/out tracking

	// Drag tracking
	dragSource render.Renderable
	dragButton input.MouseButton
	dragging   bool

	// Render loop
	renderRequested chan struct{} // buffered channel, size 1
	stopChan        chan struct{}
	renderTicker    *time.Ticker

	// Post-processing
	postProcessFns []func(*buffer.Buffer, float64)

	// Stats
	stats         Stats
	frameCount    uint64
	lastFrameTime time.Time

	// Cursor
	cursorX        int
	cursorY        int
	cursorVisible  bool
	cursorStyle    CursorStyle
	cursorBlinking bool
	cursorColor    style.Color
	mousePointer   MousePointerStyle

	// Selection
	sel *render.Selection

	// Hyperlinks (OSC 8)
	linkRegistry *LinkRegistry

	// Live rendering
	liveCount int32 // atomic

	// Inline mode state
	lastInlineHeight      int            // erase count for previous frame (N+1: content + trailing)
	lastInlineLayoutWidth int            // tracks layout width to avoid forced relayout every frame
	inlineCurrentBuf      *buffer.Buffer // what's currently displayed; enables row-level diff

	// Debug logger. Non-nil only when Config.DebugLog is set.
	// Writes to a file in the logs directory; safe to call from the render loop.
	dbg     *log.Logger
	dbgFile *os.File

	// Done channel — closed when Destroy() completes.
	done chan struct{}

	mu          sync.Mutex
	resizeCh    chan resizeEvent // serializes resize onto the render loop
	invalidated bool             // when true, next frame uses FullFlush instead of DiffAndFlush
}

// resizeEvent carries a pending terminal resize.
type resizeEvent struct {
	width, height int
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewRenderer creates and initializes a new terminal renderer.
// Call Start() or Auto() to begin rendering.
func NewRenderer(config ...Config) (*Renderer, error) {
	cfg := mergeConfig(config)

	// Create backend
	backend := NewBackend(cfg.Stdin, cfg.Stdout)

	// Query terminal size
	w, h, err := backend.Size()
	if err != nil {
		// Fallback to reasonable defaults
		w, h = 80, 24
	}

	// Create double buffers.
	currentBuffer := buffer.New(w, h)
	nextBuffer := buffer.New(w, h)

	// Create components
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	parser := input.NewParser()
	eventBus := input.NewEventBus()

	r := &Renderer{
		root:            root,
		hitGrid:         hitGrid,
		backend:         backend,
		config:          cfg,
		currentBuffer:   currentBuffer,
		nextBuffer:      nextBuffer,
		width:           w,
		height:          h,
		controlState:    StateIdle,
		parser:          parser,
		eventBus:        eventBus,
		renderRequested: make(chan struct{}, 1),
		stopChan:        make(chan struct{}),
		done:            make(chan struct{}),
		resizeCh:        make(chan resizeEvent, 1),
		invalidated:     true,
		postProcessFns:  cfg.PostProcessFns,
		stats:           NewStats(60),
		lastFrameTime:   time.Now(),
		sel:             render.NewSelection(),
		linkRegistry:    NewLinkRegistry(),
	}

	// Open debug log file if requested.
	if cfg.DebugLog != "" {
		f, err := os.OpenFile(cfg.DebugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			r.dbgFile = f
			r.dbg = log.New(f, "", log.Ltime|log.Lmicroseconds)
			r.dbg.Printf("=== fluid debug log opened  pid=%d ===", os.Getpid())
		}
		// If the file can't be opened we silently skip logging so the app still runs.
	}

	// The Renderer satisfies Context directly — no closure wiring needed.
	root.SetContext(r)

	// Configure Kitty keyboard parsing
	if cfg.UseKittyKeyboard {
		parser.SetKittyEnabled(true)
	}

	// Configure Alt ESC-prefix heuristic
	if cfg.AltSendsEscPrefix {
		parser.SetAltSendsEscPrefix(true)
	}

	return r, nil
}

// ---------------------------------------------------------------------------
// Properties
// ---------------------------------------------------------------------------

// Root returns the component tree root.
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

// IsRunning returns whether the render loop is active.
func (r *Renderer) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.controlState == StateAutoStarted || r.controlState == StateExplicitStarted
}

// IsDestroyed returns whether the renderer has been destroyed.
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

// CurrentBuffer returns the currently displayed buffer.
func (r *Renderer) CurrentBuffer() *buffer.Buffer { return r.currentBuffer }

// NextBuffer returns the buffer being drawn into.
func (r *Renderer) NextBuffer() *buffer.Buffer { return r.nextBuffer }

// EventBus returns the event bus for external subscriptions.
func (r *Renderer) EventBus() *input.EventBus { return r.eventBus }

// LinkRegistry returns the hyperlink registry for OSC 8 link allocation.
func (r *Renderer) LinkRegistry() *LinkRegistry { return r.linkRegistry }

// FocusedRenderable returns the currently focused renderable.
func (r *Renderer) FocusedRenderable() render.Renderable {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.focused
}

// Done returns a channel that is closed when Destroy completes.
// Use this to detect shutdown in select statements.
func (r *Renderer) Done() <-chan struct{} { return r.done }

// Backend returns the underlying terminal backend for advanced use.
func (r *Renderer) Backend() *Backend { return r.backend }

// ColorMode returns the terminal's color capability.
func (r *Renderer) ColorMode() style.ColorMode { return r.backend.ColorMode() }

// SetColorMode overrides the auto-detected color mode.
func (r *Renderer) SetColorMode(m style.ColorMode) { r.backend.SetColorMode(m) }

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Start sets up the terminal and starts a continuous render loop.
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

	go r.renderLoop()
	go r.inputLoop()
	go r.watchResize()

	return nil
}

// Auto sets up the terminal and renders on demand when dirty.
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

	go r.renderLoop()
	go r.inputLoop()
	go r.watchResize()

	return nil
}

// Run starts the renderer in auto (render-on-dirty) mode and blocks until
// the context is cancelled, a SIGINT/SIGTERM is received, or Destroy is
// called externally (e.g. via ExitOnCtrlC). It always calls Destroy before
// returning.
//
// This is the recommended way to run a fluid application:
//
//	return r.Run(ctx)
func (r *Renderer) Run(ctx context.Context) error {
	if err := r.Auto(); err != nil {
		return err
	}
	return r.Wait(ctx)
}

// RunContinuous starts the renderer in continuous (every-tick) mode and
// blocks until shutdown. See Run for details.
func (r *Renderer) RunContinuous(ctx context.Context) error {
	if err := r.Start(); err != nil {
		return err
	}
	return r.Wait(ctx)
}

// Wait blocks until the context is cancelled, a SIGINT/SIGTERM is received,
// or Destroy is called externally. It always calls Destroy (idempotent)
// before returning.
//
// Use Wait when you need to perform work between Start/Auto and blocking:
//
//	r.Start()
//	r.FocusRenderable(myWidget)
//	return r.Wait(ctx)
func (r *Renderer) Wait(ctx context.Context) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	select {
	case <-ctx.Done():
	case <-sig:
	case <-r.done:
		return nil // already destroyed
	}

	r.Destroy()
	return ctx.Err()
}

// Pause pauses the render loop.
func (r *Renderer) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.controlState == StateExplicitStarted || r.controlState == StateAutoStarted {
		r.controlState = StateExplicitPaused
	}
}

// Resume resumes from pause.
func (r *Renderer) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.controlState == StateExplicitPaused {
		r.controlState = StateExplicitStarted
	}
}

// Suspend suspends: restores terminal, disables mouse, pauses loop.
func (r *Renderer) Suspend() {
	r.mu.Lock()
	prevState := r.controlState
	r.controlState = StateExplicitSuspended
	r.mu.Unlock()

	_ = prevState

	// Restore terminal state
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

// Stop stops the render loop.
func (r *Renderer) Stop() {
	r.mu.Lock()
	if r.controlState == StateExplicitStopped || r.controlState == StateIdle {
		r.mu.Unlock()
		return
	}
	r.controlState = StateExplicitStopped
	r.mu.Unlock()

	// Signal the render loop to stop
	select {
	case r.stopChan <- struct{}{}:
	default:
	}

	if r.renderTicker != nil {
		r.renderTicker.Stop()
	}
}

// Destroy performs full cleanup: stop, restore terminal, destroy tree.
func (r *Renderer) Destroy() {
	r.mu.Lock()
	if r.destroyed {
		r.mu.Unlock()
		return
	}
	r.destroyed = true
	r.mu.Unlock()

	// Stop the render loop
	r.Stop()

	// In inline mode, move cursor past the rendered content so the
	// shell prompt appears on a clean line below.
	if r.config.InlineMode && r.lastInlineHeight > 0 {
		r.backend.WriteOut([]byte("\r\n"))
	}

	// Restore terminal
	_ = r.backend.Restore()

	// Destroy the component tree
	r.root.DestroyRecursively()

	// Clear event bus
	r.eventBus.Clear()

	// Signal waiters that shutdown is complete.
	close(r.done)

	// Call OnDestroy callback
	if r.config.OnDestroy != nil {
		r.config.OnDestroy()
	}

	// Flush and close debug log.
	if r.dbg != nil {
		r.dbg.Printf("=== fluid debug log closed  frames=%d ===", atomic.LoadUint64(&r.frameCount))
		r.dbgFile.Close()
		r.dbg = nil
		r.dbgFile = nil
	}
}

// RequestRender signals that a render frame is needed.
func (r *Renderer) RequestRender() {
	select {
	case r.renderRequested <- struct{}{}:
	default:
		// Already requested
	}
}

// ---------------------------------------------------------------------------
// Context interface methods
// ---------------------------------------------------------------------------

// AddToHitGrid registers a renderable area in the hit grid.
func (r *Renderer) AddToHitGrid(x, y, w, h int, id uint64) {
	r.hitGrid.AddClipped(x, y, w, h, id)
}

// PushHitGridScissorRect pushes a scissor rect onto the hit grid.
func (r *Renderer) PushHitGridScissorRect(x, y, w, h int) {
	r.hitGrid.PushScissorRect(x, y, w, h)
}

// PopHitGridScissorRect pops the current hit grid scissor rect.
func (r *Renderer) PopHitGridScissorRect() {
	r.hitGrid.PopScissorRect()
}

// ClearHitGrid clears the hit grid and all scissor rects.
func (r *Renderer) ClearHitGrid() {
	r.hitGrid.Clear()
	r.hitGrid.ClearScissorRects()
}

// Selection returns the active selection, or nil.
func (r *Renderer) Selection() *render.Selection {
	if r.sel.IsActive() {
		return r.sel
	}
	return nil
}

// StartSelection initiates a selection from the given coordinates.
func (r *Renderer) StartSelection(_ render.Renderable, x, y int) {
	r.sel.Start(x, y)
}

// UpdateSelection updates the focus point of the active selection.
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

// RequestLive increments the live rendering reference count.
func (r *Renderer) RequestLive() {
	atomic.AddInt32(&r.liveCount, 1)
}

// DropLive decrements the live rendering reference count.
func (r *Renderer) DropLive() {
	atomic.AddInt32(&r.liveCount, -1)
}

// ---------------------------------------------------------------------------
// Render Loop
// ---------------------------------------------------------------------------

// renderLoop runs the main render loop as a goroutine.
func (r *Renderer) renderLoop() {
	r.mu.Lock()
	fps := r.config.TargetFPS
	if fps <= 0 {
		fps = 30
	}
	r.mu.Unlock()

	r.renderTicker = time.NewTicker(time.Second / time.Duration(fps))
	defer r.renderTicker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-r.renderTicker.C:
			// Process any pending resizes on the render loop (no race).
			r.drainResizes()

			r.mu.Lock()
			state := r.controlState
			r.mu.Unlock()

			switch state {
			case StateAutoStarted:
				// Auto mode: only render if dirty or live
				hasLive := atomic.LoadInt32(&r.liveCount) > 0
				select {
				case <-r.renderRequested:
					r.renderFrame()
				default:
					if hasLive {
						r.renderFrame()
					}
				}

			case StateExplicitStarted:
				r.renderFrame()

			case StateExplicitPaused, StateExplicitSuspended:
				// Skip rendering

			case StateExplicitStopped:
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Resize (processed on the render loop)
// ---------------------------------------------------------------------------

// drainResizes processes any pending resize events before rendering.
// Runs on the render loop goroutine so there is no race with renderFrame.
func (r *Renderer) drainResizes() {
	for {
		select {
		case ev := <-r.resizeCh:
			r.processResize(ev.width, ev.height)
		default:
			return
		}
	}
}

// processResize applies a terminal resize. Must be called from the render loop.
func (r *Renderer) processResize(w, h int) {
	r.mu.Lock()
	r.width = w
	r.height = h
	r.mu.Unlock()

	if r.dbg != nil {
		r.dbg.Printf("[resize] %dx%d", w, h)
	}

	if r.config.InlineMode {
		// In inline mode, only width matters. Height is content-driven
		// and computed each frame.
		r.invalidated = true
		r.hitGrid.Resize(w, h)
		r.root.LayoutNode().Width = layout.Points(w)
		r.root.LayoutNode().MarkDirty()
		// Force clearTerminal on the next render — but only if we've
		// already rendered or ClearScreenOnStart is set. Without a
		// previous render, triggering clearTerminal would nuke the
		// user's previous terminal output.
		if r.lastInlineHeight > 0 || r.config.ClearScreenOnStart {
			r.lastInlineHeight = h + 1
		}
	} else {
		// Fullscreen mode: resize both buffers and mark invalidated
		// so the next frame uses FullFlush instead of DiffAndFlush.
		r.currentBuffer.Resize(w, h)
		r.nextBuffer.Resize(w, h)
		r.nextBuffer.Clear(r.config.BackgroundColor)
		r.invalidated = true

		r.hitGrid.Resize(w, h)
		r.root.Resize(w, h)
	}

	// Emit resize event.
	r.eventBus.Emit(input.TerminalEvent{
		Type: input.EventResize,
		Resize: &input.ResizeEvent{
			Width:  w,
			Height: h,
		},
	})
}

// ---------------------------------------------------------------------------
// Render Frame
// ---------------------------------------------------------------------------

// renderFrame executes a single frame render.
//
//  1. Render the component tree into nextBuffer
//  2. DiffAndFlush: diff next vs current, write changes to terminal,
//     copy changed cells next → current (so current always reflects terminal)
//  3. Clear nextBuffer for the next frame
//
// No buffer swap. currentBuffer accumulates the terminal state.
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

	// Render the component tree into nextBuffer
	r.root.Render(r.nextBuffer, deltaTime)

	if r.dbg != nil && r.invalidated {
		fc := atomic.LoadUint64(&r.frameCount)
		r.dbg.Printf("[frame] #%d  buf=%dx%d", fc, r.nextBuffer.Width(), r.nextBuffer.Height())
	}

	// Run post-process functions
	r.mu.Lock()
	fns := make([]func(*buffer.Buffer, float64), len(r.postProcessFns))
	copy(fns, r.postProcessFns)
	r.mu.Unlock()

	for _, fn := range fns {
		fn(r.nextBuffer, deltaTime)
	}

	// Flush to terminal. After a resize (or first frame), use FullFlush to
	// repaint everything; otherwise diff for efficiency.
	tFlush0 := time.Now()
	if r.invalidated {
		// Clear the screen before a full repaint. Some terminal emulators
		// don't properly clear/resize the alternate screen buffer on
		// SIGWINCH, leaving stale content that corrupts the display.
		// An explicit clear ensures a clean slate.
		if !r.config.InlineMode {
			r.backend.Clear()
		}
		r.FullFlush(r.nextBuffer)
		r.invalidated = false
	} else {
		r.DiffAndFlush(r.currentBuffer, r.nextBuffer)
	}

	// Populate frame-level stats
	r.root.LastFrameStats.Flush = time.Since(tFlush0)
	r.root.LastFrameStats.Total = time.Since(frameStart)

	// Sync currentBuffer to match what the terminal now shows, then
	// clear nextBuffer for the next frame.
	r.currentBuffer.CopyFrom(r.nextBuffer)
	r.nextBuffer.Clear(bgColor)

	// Handle cursor position after render
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

	// Increment frame counter.
	atomic.AddUint64(&r.frameCount, 1)

	// Update stats
	if r.config.GatherStats {
		renderTime := time.Since(frameStart).Seconds()
		r.mu.Lock()
		r.stats.record(renderTime)
		r.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Input & Focus
// ---------------------------------------------------------------------------

// FocusRenderable sets keyboard focus to the given renderable.
//
// The caller may pass any Renderable — including the embedded *BaseRenderable
// that Focus() passes via ctx.FocusRenderable(b). We resolve to the correct
// outer widget type via LookupRenderable(num), which returns whatever was
// registered in Add() (the outer widget, e.g. *widget.Input).
//
// Num() equality (not pointer equality) prevents infinite recursion when
// Focus() -> ctx.FocusRenderable(embeddedBase) re-enters this method.
func (r *Renderer) FocusRenderable(renderable render.Renderable) {
	if renderable == nil {
		return
	}

	// Resolve to the correct outer widget type from the global registry.
	// Add() re-registers each child with its outer type, so this returns
	// e.g. *widget.Input instead of the embedded *BaseRenderable.
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

	// Blur the previous focused renderable
	if prev != nil && !prev.IsDestroyed() {
		prev.Blur()
	}

	// Call Focus() which sets the internal focused flag on the embedded
	// BaseRenderable. Focus() will call ctx.FocusRenderable(b) with the
	// embedded *BaseRenderable, but the Num() guard above catches the
	// re-entrant call and prevents infinite recursion.
	if actual != nil && !actual.IsDestroyed() && actual.Focusable() {
		actual.Focus()
		// Re-assert r.focused to the resolved outer type. Focus() may have
		// triggered ctx.FocusRenderable with the embedded *BaseRenderable,
		// which the Num() guard caught, but be defensive.
		r.mu.Lock()
		r.focused = actual
		r.mu.Unlock()
	}

	r.RequestRender()
}

// AddInputHandler adds an input pre-processor.
// Handlers are called in order; if one returns true, processing stops.
func (r *Renderer) AddInputHandler(handler func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputHandlers = append(r.inputHandlers, handler)
}

// PrependInputHandler adds an input pre-processor at the front.
func (r *Renderer) PrependInputHandler(handler func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputHandlers = append([]func(string) bool{handler}, r.inputHandlers...)
}

// RemoveInputHandler removes the first matching input pre-processor.
func (r *Renderer) RemoveInputHandler(handler func(string) bool) {
	// Note: function comparison in Go is limited; this uses address comparison.
	// In practice, callers should keep a reference to the handler they added.
	r.mu.Lock()
	defer r.mu.Unlock()
	// We can't reliably compare function pointers in Go, so this is a no-op
	// placeholder. Callers should manage their own handler lifecycle.
}

// SetCursorPosition sets the cursor position and visibility.
func (r *Renderer) SetCursorPosition(x, y int, visible bool) {
	r.mu.Lock()
	r.cursorX = x
	r.cursorY = y
	r.cursorVisible = visible
	r.mu.Unlock()
}

// SetCursorStyle sets the cursor style.
func (r *Renderer) SetCursorStyle(cursorStyle string) {
	r.mu.Lock()
	r.cursorStyle = cursorStyle
	r.mu.Unlock()

	r.backend.SetCursorStyle(cursorStyle)
}

// SetCursorColor sets the cursor color via OSC 12.
func (r *Renderer) SetCursorColor(color style.Color) {
	r.mu.Lock()
	r.cursorColor = color
	r.mu.Unlock()

	r.backend.SetCursorColor(color.R, color.G, color.B)
}

// CursorState returns the full cursor state.
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

// SetCursorStyleOptions sets cursor style, blinking, color, and mouse pointer
// in one call. Only non-nil fields are applied.
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

// SetMousePointer sets the mouse pointer style via OSC 22.
func (r *Renderer) SetMousePointer(pointerStyle MousePointerStyle) {
	r.mu.Lock()
	r.mousePointer = pointerStyle
	r.mu.Unlock()

	r.backend.SetMousePointer(pointerStyle)
}

// ---------------------------------------------------------------------------
// Rendering configuration
// ---------------------------------------------------------------------------

// SetBackgroundColor sets the renderer background color.
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

// AddPostProcessFn adds a post-process function.
func (r *Renderer) AddPostProcessFn(fn func(*buffer.Buffer, float64)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postProcessFns = append(r.postProcessFns, fn)
}

// RemovePostProcessFn is a placeholder for removing post-process functions.
// In Go, function comparison is limited; callers should manage their own lifecycle.
func (r *Renderer) RemovePostProcessFn(fn func(*buffer.Buffer, float64)) {
	// Placeholder: cannot reliably compare function values in Go
}

// ClearPostProcessFns removes all post-process functions.
func (r *Renderer) ClearPostProcessFns() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postProcessFns = nil
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

// Stats returns current rendering statistics.
func (r *Renderer) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// ResetStats resets frame timing stats.
func (r *Renderer) ResetStats() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.reset()
}

// SetGatherStats enables or disables stats gathering.
func (r *Renderer) SetGatherStats(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config.GatherStats = enabled
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// setup performs terminal initialization common to Start() and Auto().
func (r *Renderer) setup() error {
	if err := r.backend.Setup(r.config.Fullscreen); err != nil {
		return err
	}

	// Enable mouse if configured
	if r.config.UseMouse {
		r.backend.EnableMouse(r.config.EnableMouseMovement)
	}

	// Enable Kitty keyboard if configured
	if r.config.UseKittyKeyboard {
		r.backend.EnableKittyKeyboard(r.config.KittyKeyboardFlags)
	}

	if r.config.InlineMode {
		// In inline mode: don't clear screen, hide hardware cursor.
		// The cursor is drawn visually (inverted cell) in the buffer.
		// The terminal is in raw mode (for input) but content goes
		// into the normal scrollback buffer.
		r.backend.HideCursor()
	} else {
		// Fullscreen mode: clear screen.
		r.backend.Clear()
	}

	// Re-query size after setup (render loop not started yet, safe to call directly)
	if w, h, err := r.backend.Size(); err == nil {
		r.processResize(w, h)
	}

	return nil
}
