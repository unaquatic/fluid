package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.UseMouse {
		t.Error("expected UseMouse=false")
	}
	if cfg.EnableMouseMovement {
		t.Error("expected EnableMouseMovement=false")
	}
	if cfg.InlineMode {
		t.Error("expected InlineMode=false")
	}
	if cfg.EnableAutoFocus {
		t.Error("expected EnableAutoFocus=false")
	}
	if cfg.HandleCtrlC {
		t.Error("expected HandleCtrlC=false")
	}
	if !cfg.BackgroundColor.Equals(style.ColorBlack) {
		t.Error("expected BackgroundColor=Black")
	}
	if cfg.UseKittyKeyboard {
		t.Error("expected UseKittyKeyboard=false")
	}
}

func TestMergeConfig_Defaults(t *testing.T) {
	cfg := mergeConfig(nil)

	if !cfg.BackgroundColor.Equals(style.ColorBlack) {
		t.Errorf("expected default BackgroundColor=Black, got %v", cfg.BackgroundColor)
	}
}

func TestMergeConfig_Override(t *testing.T) {
	cfg := mergeConfig([]Config{
		{
			BackgroundColor:    style.ColorRed,
			KittyKeyboardFlags: KittyFlagDisambiguate,
		},
	})

	if !cfg.BackgroundColor.Equals(style.ColorRed) {
		t.Errorf("expected BackgroundColor=Red, got %v", cfg.BackgroundColor)
	}
	if cfg.KittyKeyboardFlags != KittyFlagDisambiguate {
		t.Errorf("expected KittyKeyboardFlags=%d, got %d", KittyFlagDisambiguate, cfg.KittyKeyboardFlags)
	}
}

func TestRendererControlState_String(t *testing.T) {
	tests := []struct {
		state    ControlState
		expected string
	}{
		{StateIdle, "idle"},
		{StateAutoStarted, "auto"},
		{StateExplicitStarted, "started"},
		{StateExplicitPaused, "paused"},
		{StateExplicitSuspended, "suspended"},
		{StateExplicitStopped, "stopped"},
		{ControlState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("state %d: expected %q, got %q", tt.state, tt.expected, got)
		}
	}
}

func TestNewRenderer_Properties(t *testing.T) {
	r := newInputTestRenderer()

	if r.Root() == nil {
		t.Fatal("expected non-nil root")
	}
	if r.Root().ID() != "root" {
		t.Errorf("expected root ID='root', got %q", r.Root().ID())
	}
	if r.IsRunning() {
		t.Error("expected not running before start")
	}
	if r.IsDestroyed() {
		t.Error("expected not destroyed")
	}
	if r.ControlState() != StateIdle {
		t.Errorf("expected state=idle, got %s", r.ControlState())
	}
	if r.EventBus() == nil {
		t.Error("expected non-nil EventBus")
	}
	if r.FocusedRenderable() != nil {
		t.Error("expected no focused renderable initially")
	}
}

func TestNewRenderer_Buffers(t *testing.T) {
	r := newInputTestRenderer()

	if r.CurrentBuffer() == nil {
		t.Fatal("expected non-nil CurrentBuffer")
	}
	if r.NextBuffer() == nil {
		t.Fatal("expected non-nil NextBuffer")
	}
	if r.CurrentBuffer().Width() != 40 || r.CurrentBuffer().Height() != 20 {
		t.Error("expected buffer dimensions to match terminal size")
	}
}

func TestRenderer_PauseResume(t *testing.T) {
	r := newInputTestRenderer()

	r.mu.Lock()
	r.controlState = StateExplicitStarted
	r.mu.Unlock()

	r.Pause()
	if r.ControlState() != StateExplicitPaused {
		t.Errorf("expected paused, got %s", r.ControlState())
	}

	r.Resume()
	if r.ControlState() != StateExplicitStarted {
		t.Errorf("expected started after resume, got %s", r.ControlState())
	}
}

func TestRenderer_Stop(t *testing.T) {
	r := newInputTestRenderer()

	r.mu.Lock()
	r.controlState = StateExplicitStarted
	r.mu.Unlock()

	r.Stop()
	if r.ControlState() != StateExplicitStopped {
		t.Errorf("expected stopped, got %s", r.ControlState())
	}
}

func TestRenderer_Destroy(t *testing.T) {
	r := newInputTestRenderer()

	var destroyCalled bool
	r.config.OnDestroy = func() {
		destroyCalled = true
	}

	r.mu.Lock()
	r.controlState = StateExplicitStarted
	r.mu.Unlock()

	r.Destroy()

	if !r.IsDestroyed() {
		t.Error("expected destroyed after Destroy()")
	}
	if !destroyCalled {
		t.Error("expected OnDestroy callback to be called")
	}
}

func TestRenderer_DestroyIdempotent(t *testing.T) {
	r := newInputTestRenderer()

	callCount := 0
	r.config.OnDestroy = func() {
		callCount++
	}

	r.Destroy()
	r.Destroy()
	r.Destroy()

	if callCount != 1 {
		t.Errorf("expected OnDestroy called once, got %d", callCount)
	}
}

func TestRenderer_FocusRenderable(t *testing.T) {
	r := newInputTestRenderer()

	a := render.NewBaseRenderable(render.WithID("a"), render.WithFocusable(true))
	b := render.NewBaseRenderable(render.WithID("b"), render.WithFocusable(true))
	r.root.Add(a)
	r.root.Add(b)

	r.FocusRenderable(a)
	if r.FocusedRenderable() != a {
		t.Error("expected a to be focused")
	}
	if !a.Focused() {
		t.Error("expected a.Focused() = true")
	}

	r.FocusRenderable(b)
	if r.FocusedRenderable() != b {
		t.Error("expected b to be focused")
	}
	if a.Focused() {
		t.Error("expected a.Focused() = false after blur")
	}
	if !b.Focused() {
		t.Error("expected b.Focused() = true")
	}
}

func TestRenderer_RequestRender(t *testing.T) {
	r := newInputTestRenderer()

	r.RequestRender()

	if !r.dirty.Load() {
		t.Error("expected dirty to be true after RequestRender")
	}
	select {
	case <-r.wake:
	default:
		t.Error("expected RequestRender to wake the event loop")
	}
	select {
	case <-r.pendingWork:
		t.Fatal("RequestRender must not queue work")
	default:
	}
}

func TestRenderer_RequestRenderIdempotent(t *testing.T) {
	r := newInputTestRenderer()

	// Repeated requests retain one wake token and one dirty frame.
	r.RequestRender()
	r.RequestRender()
	r.RequestRender()

	if !r.dirty.Load() {
		t.Error("expected dirty to be true")
	}
	select {
	case <-r.wake:
	default:
		t.Fatal("expected a wake-up")
	}
	select {
	case <-r.wake:
		t.Error("expected wake-ups to coalesce into one")
	default:
	}
}

func TestRenderer_RenderFrame(t *testing.T) {
	var outBuf bytes.Buffer
	w, h := 10, 5

	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)

	r := &Renderer{
		root:          root,
		hitGrid:       hitGrid,
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		config:        DefaultConfig(),
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		width:         w,
		height:        h,
		controlState:  StateExplicitStarted,
		parser:        input.NewParser(),
		eventBus:      input.NewEventBus(),
		pendingWork:   make(chan func()),
		wake:          make(chan struct{}, 1),
		stopChan:      make(chan struct{}, 1),
		done:          make(chan struct{}),
		sel:           render.NewSelection(),
		linkRegistry:  NewLinkRegistry(),
		stats:         NewStats(60),
		lastFrameTime: time.Now(),
	}

	root.SetContext(r)

	// Render one child to exercise a complete frame.
	child := render.NewBaseRenderable(render.WithID("test-child"))
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		buf.SetCell(0, 0, 'T', style.ColorWhite, style.ColorBlack, 0)
	}
	root.Add(child)

	r.config.GatherStats = true
	r.renderFrame()

	// Enabled statistics count the rendered frame.
	stats := r.Stats()
	if stats.FrameCount != 1 {
		t.Errorf("expected FrameCount=1, got %d", stats.FrameCount)
	}
}

func TestRenderer_SetBackgroundColor(t *testing.T) {
	r := newInputTestRenderer()

	r.SetBackgroundColor(style.ColorString("#ff0000"))

	r.mu.Lock()
	bg := r.config.BackgroundColor
	r.mu.Unlock()

	if !bg.Equals(style.ColorRed) {
		t.Errorf("expected red background, got %v", bg)
	}
}

func TestRenderer_SetTitle(t *testing.T) {
	var outBuf bytes.Buffer
	r := newInputTestRenderer()
	r.backend = NewBackend(strings.NewReader(""), &outBuf)

	r.SetTitle("Test Title")

	if !strings.Contains(outBuf.String(), "\x1b]0;Test Title\x07") {
		t.Errorf("expected title escape, got: %q", outBuf.String())
	}
}

func TestRenderer_StatsToggle(t *testing.T) {
	r := newInputTestRenderer()

	r.SetGatherStats(true)
	if !r.config.GatherStats {
		t.Error("expected GatherStats=true")
	}

	r.SetGatherStats(false)
	if r.config.GatherStats {
		t.Error("expected GatherStats=false")
	}
}

func TestRenderer_ResetStats(t *testing.T) {
	r := newInputTestRenderer()
	r.config.GatherStats = true

	r.mu.Lock()
	r.stats.record(0.016)
	r.stats.record(0.016)
	r.mu.Unlock()

	r.ResetStats()

	stats := r.Stats()
	if stats.FrameCount != 0 {
		t.Errorf("expected FrameCount=0 after reset, got %d", stats.FrameCount)
	}
}

func TestRenderer_SetCursorPosition(t *testing.T) {
	r := newInputTestRenderer()

	r.SetCursorPosition(10, 20, true)

	r.mu.Lock()
	x, y, vis := r.cursorX, r.cursorY, r.cursorVisible
	r.mu.Unlock()

	if x != 10 || y != 20 {
		t.Errorf("expected cursor at (10,20), got (%d,%d)", x, y)
	}
	if !vis {
		t.Error("expected cursor visible")
	}
}

func TestRenderer_SetCursorStyle(t *testing.T) {
	r := newInputTestRenderer()

	r.SetCursorStyle(CursorUnderline)

	r.mu.Lock()
	cs := r.cursorStyle
	r.mu.Unlock()

	if cs != CursorUnderline {
		t.Errorf("expected cursor style underline, got %v", cs)
	}
}

func TestRenderer_SetCursorColor(t *testing.T) {
	var outBuf bytes.Buffer
	r := newInputTestRenderer()
	r.backend = NewBackend(strings.NewReader(""), &outBuf)

	r.SetCursorColor(style.ColorRed)

	r.mu.Lock()
	cc := r.cursorColor
	r.mu.Unlock()

	if !cc.Equals(style.ColorRed) {
		t.Errorf("expected cursor color red, got %v", cc)
	}

	// Setting color writes an OSC 12 sequence immediately.
	output := outBuf.String()
	if !strings.Contains(output, "\x1b]12;rgb:") {
		t.Errorf("expected OSC 12 escape in output, got: %q", output)
	}
}

func TestRenderer_GetCursorState(t *testing.T) {
	r := newInputTestRenderer()

	r.SetCursorPosition(5, 10, true)
	r.SetCursorStyle(CursorBlock)
	r.SetCursorColor(style.ColorGreen)

	state := r.CursorState()

	if state.X != 5 || state.Y != 10 {
		t.Errorf("expected (5,10), got (%d,%d)", state.X, state.Y)
	}
	if !state.Visible {
		t.Error("expected visible=true")
	}
	if state.Style != CursorBlock {
		t.Errorf("expected style block, got %v", state.Style)
	}
	if !state.Color.Equals(style.ColorGreen) {
		t.Errorf("expected color green, got %v", state.Color)
	}
	if state.Blinking {
		t.Error("expected blinking=false by default")
	}
}

func TestRenderer_SetCursorStyleOptions(t *testing.T) {
	var outBuf bytes.Buffer
	r := newInputTestRenderer()
	r.backend = NewBackend(strings.NewReader(""), &outBuf)

	cursorStyle := CursorLine
	blinking := true
	cursorColor := style.ColorBlue
	pointer := MousePointerText

	r.SetCursorStyleOptions(CursorStyleOptions{
		Style:    &cursorStyle,
		Blinking: &blinking,
		Color:    &cursorColor,
		Cursor:   &pointer,
	})

	state := r.CursorState()

	if state.Style != CursorLine {
		t.Errorf("expected style line, got %v", state.Style)
	}
	if !state.Blinking {
		t.Error("expected blinking=true")
	}
	if !state.Color.Equals(style.ColorBlue) {
		t.Errorf("expected color blue, got %v", state.Color)
	}

	r.mu.Lock()
	mp := r.mousePointer
	r.mu.Unlock()
	if mp != MousePointerText {
		t.Errorf("expected mousePointer='text', got %q", mp)
	}

	// Each non-nil option writes its corresponding terminal sequence.
	output := outBuf.String()
	if !strings.Contains(output, "\x1b]12;rgb:") {
		t.Error("expected OSC 12 cursor color escape")
	}
	if !strings.Contains(output, "\x1b]22;text\x1b\\") {
		t.Error("expected OSC 22 mouse pointer escape")
	}
}

func TestRenderer_SetCursorStyleOptions_Partial(t *testing.T) {
	r := newInputTestRenderer()

	// Establish cursor state before a partial update.
	r.SetCursorStyle(CursorBlock)
	r.SetCursorPosition(1, 2, true)

	// Update only the style field.
	newStyle := CursorUnderline
	r.SetCursorStyleOptions(CursorStyleOptions{
		Style: &newStyle,
	})

	state := r.CursorState()
	if state.Style != CursorUnderline {
		t.Errorf("expected style underline, got %v", state.Style)
	}
	// Unspecified position and visibility remain unchanged.
	if state.X != 1 || state.Y != 2 {
		t.Errorf("expected position (1,2), got (%d,%d)", state.X, state.Y)
	}
}

func TestRenderer_SetMousePointer(t *testing.T) {
	var outBuf bytes.Buffer
	r := newInputTestRenderer()
	r.backend = NewBackend(strings.NewReader(""), &outBuf)

	r.SetMousePointer(MousePointerCrosshair)

	r.mu.Lock()
	mp := r.mousePointer
	r.mu.Unlock()

	if mp != MousePointerCrosshair {
		t.Errorf("expected 'crosshair', got %q", mp)
	}

	output := outBuf.String()
	if !strings.Contains(output, "\x1b]22;crosshair\x1b\\") {
		t.Errorf("expected OSC 22 escape, got: %q", output)
	}
}
