package terminal

import (
	"io"
	"os"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// ControlState
// ---------------------------------------------------------------------------

// ControlState represents the current lifecycle state of the terminal.
// The terminal progresses through these states as it is started, paused,
// suspended, and stopped. Widgets and external code can inspect the state
// to decide whether rendering is active.
type ControlState int

const (
	// StateIdle is the initial state before the terminal has been started.
	StateIdle ControlState = iota
	// StateAutoStarted means the terminal is in auto mode: it renders only
	// when the scene is marked dirty via RequestRender.
	StateAutoStarted
	// StateExplicitStarted means the terminal is running a continuous
	// render loop at the configured frame rate.
	StateExplicitStarted
	// StateExplicitPaused means the render loop is still alive but
	// skipping frames. Resume with Resume().
	StateExplicitPaused
	// StateExplicitSuspended means the terminal has been restored to its
	// original state and the render loop is paused. Used for shelling out
	// to external commands.
	StateExplicitSuspended
	// StateExplicitStopped means the render loop has exited. The terminal
	// cannot be restarted.
	StateExplicitStopped
)

// String returns the name of the state.
func (s ControlState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateAutoStarted:
		return "auto"
	case StateExplicitStarted:
		return "started"
	case StateExplicitPaused:
		return "paused"
	case StateExplicitSuspended:
		return "suspended"
	case StateExplicitStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// ---------------------------------------------------------------------------
// Kitty keyboard flag constants (re-exported for convenience)
// ---------------------------------------------------------------------------

const (
	KittyFlagDisambiguate  = input.KittyFlagDisambiguate
	KittyFlagAlternateKeys = input.KittyFlagAlternateKeys
)

// ---------------------------------------------------------------------------
// CursorStyle
// ---------------------------------------------------------------------------

// CursorStyle represents the terminal cursor shape. The terminal translates
// these into DECSCUSR escape sequences when flushing each frame.
type CursorStyle = string

const (
	// CursorBlock renders the cursor as a solid filled rectangle.
	CursorBlock CursorStyle = "block"
	// CursorLine renders the cursor as a thin vertical bar (I-beam).
	CursorLine CursorStyle = "line"
	// CursorUnderline renders the cursor as a horizontal line under the
	// character.
	CursorUnderline CursorStyle = "underline"
)

// ---------------------------------------------------------------------------
// MousePointerStyle
// ---------------------------------------------------------------------------

// MousePointerStyle represents the mouse pointer shape. Terminals that
// support the OSC 22 sequence will change the pointer icon accordingly.
type MousePointerStyle = string

const (
	// MousePointerDefault is the standard arrow pointer.
	MousePointerDefault MousePointerStyle = "default"
	// MousePointerPointer is the hand/link pointer (for clickable elements).
	MousePointerPointer MousePointerStyle = "pointer"
	// MousePointerText is the I-beam pointer (for text selection areas).
	MousePointerText MousePointerStyle = "text"
	// MousePointerCrosshair is the crosshair pointer (for precision targeting).
	MousePointerCrosshair MousePointerStyle = "crosshair"
	// MousePointerMove is the four-way arrow pointer (for draggable elements).
	MousePointerMove MousePointerStyle = "move"
	// MousePointerNotAllowed is the circle-slash pointer (for disabled areas).
	MousePointerNotAllowed MousePointerStyle = "not-allowed"
)

// ---------------------------------------------------------------------------
// CursorStyleOptions
// ---------------------------------------------------------------------------

// CursorStyleOptions configures cursor appearance. All fields are optional
// pointers; nil means "keep the current value".
type CursorStyleOptions struct {
	// Style overrides the cursor shape (block, line, or underline).
	Style *CursorStyle
	// Blinking overrides the cursor blink state.
	Blinking *bool
	// Color overrides the cursor color.
	Color *style.Color
	// Cursor overrides the mouse pointer style (e.g. pointer, text, move).
	Cursor *MousePointerStyle
}

// ---------------------------------------------------------------------------
// CursorState
// ---------------------------------------------------------------------------

// CursorState represents the full state of the terminal cursor at a given
// frame. The terminal computes this from SetCursorPosition and
// SetCursorStyleOptions calls made by the focused widget, then applies the
// corresponding escape sequences during flush.
type CursorState struct {
	// X, Y is the cursor position (0-indexed, absolute terminal coords).
	X, Y int
	// Visible controls whether the cursor is shown. Widgets without text
	// input typically set this to false.
	Visible bool
	// Style is the cursor shape (block, line, or underline).
	Style CursorStyle
	// Blinking controls whether the cursor blinks.
	Blinking bool
	// Color is the cursor's color. A zero value uses the terminal default.
	Color style.Color
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config configures the terminal.
type Config struct {
	// Stdin is the input reader used for keyboard and mouse events.
	// Default: os.Stdin.
	Stdin io.Reader

	// Stdout is the output writer used for rendering terminal escape
	// sequences and content. Default: os.Stdout.
	Stdout io.Writer

	// TargetFPS is the desired frame rate for the render loop. The terminal
	// sleeps between frames to approximate this rate. Actual FPS may be
	// lower under heavy load. Default: 30.
	TargetFPS int

	// MaxFPS caps the maximum frame rate. Even when frames are requested
	// rapidly (e.g. via RequestRender), the terminal will not exceed this
	// rate. Default: 60.
	MaxFPS int

	// UseMouse enables mouse click and scroll event tracking via terminal
	// escape sequences. When false, no mouse events are delivered to
	// widgets. Default: true.
	UseMouse bool

	// EnableMouseMovement enables mouse motion tracking in addition to
	// clicks and scrolls. Generates events on every cursor move, which
	// increases terminal I/O. Only takes effect when UseMouse is true.
	// Default: false.
	EnableMouseMovement bool

	// Fullscreen uses the alternate screen buffer with a fixed viewport
	// that fills the entire terminal. The original terminal content is
	// restored when the terminal exits. Mutually exclusive with InlineMode.
	// Default: true.
	Fullscreen bool

	// AutoFocus automatically focuses the widget under the mouse pointer
	// when a click is detected. When false, focus must be managed manually
	// via FocusRenderable. Default: true.
	AutoFocus bool

	// ExitOnCtrlC intercepts Ctrl+C and triggers a clean shutdown of the
	// terminal. When false, Ctrl+C is delivered as a normal key event.
	// Default: true.
	ExitOnCtrlC bool

	// InlineMode enables inline rendering mode. Instead of using the
	// alternate screen and a fixed-size viewport, content is rendered
	// inline in the normal terminal buffer. The height is determined by
	// the widget tree, while the width tracks the terminal width (minus
	// a 1-column right margin to prevent auto-wrap).
	//
	// When content fits on screen, previous output is erased line-by-line
	// and rewritten (like Ink's log-update). When content exceeds the
	// terminal height, the terminal is cleared and all content is rewritten
	// from scratch (like Ink's clearTerminal).
	//
	// When InlineMode is true, Fullscreen is forced to false.
	// Default: false.
	InlineMode bool

	// ClearScreenOnStart clears the terminal screen and scrollback before
	// the first render. When false, previous terminal output is preserved
	// and the inline content appears below the shell prompt, matching the
	// behavior of tools like Claude Code.
	// Only applies when InlineMode is true. Default: false.
	ClearScreenOnStart bool

	// BackgroundColor fills the entire buffer each frame before widgets
	// render. Transparent regions of the widget tree will show this color.
	// Default: style.ColorBlack.
	BackgroundColor style.Color

	// UseKittyKeyboard enables the Kitty keyboard protocol, which provides
	// unambiguous key reporting including modifier keys, key release events,
	// and distinguishing between e.g. Enter vs Ctrl+M. Requires a
	// Kitty-compatible terminal (Kitty, WezTerm, Ghostty, etc.).
	// Default: false.
	UseKittyKeyboard bool

	// KittyKeyboardFlags controls which Kitty keyboard protocol features
	// are enabled. Flags are bitwise-OR'd together.
	// Default: KittyFlagDisambiguate | KittyFlagAlternateKeys.
	KittyKeyboardFlags uint

	// AltSendsEscPrefix tells the input parser that the terminal sends
	// Alt/Option as an ESC prefix rather than setting the high bit. Enable
	// this when the terminal uses emacs-style ESC b/f for Alt+Arrow word
	// navigation (e.g. macOS Terminal.app, xterm altSendsEscape). Allows
	// the parser to infer Shift from double-ESC + unmodified nav keys
	// (ESC ESC [ D → Shift+Alt+Left). Default: false.
	AltSendsEscPrefix bool

	// PostProcessFns are functions called after each render pass, receiving
	// the fully rendered buffer and the frame's delta time. Use these for
	// effects like scanlines, vignettes, or debug overlays.
	PostProcessFns []func(buf *buffer.Buffer, deltaTime float64)

	// OnDestroy is called once when the terminal is destroyed, after the
	// terminal has been restored to its original state. Use it for final
	// cleanup (e.g. flushing logs, closing connections).
	OnDestroy func()

	// GatherStats enables per-frame collection of terminal statistics
	// (frame time, layout time, render time, flush time). Access the
	// collected data via Terminal.Stats(). Default: false.
	GatherStats bool

	// WidthProbe controls whether the renderer measures the terminal's real
	// cell widths during setup. Default: WidthProbeOff.
	WidthProbe WidthProbe

	// WidthProbeSamples are the strings measured when WidthProbe is
	// WidthProbeAuto. Results are read back with Renderer.ProbedWidths.
	WidthProbeSamples []string

	// DebugLog, when non-empty, enables per-frame diagnostic logging to
	// the given file path (e.g. "logs/fluid.log"). The directory must
	// exist. Each line is a tab-separated record:
	//
	//   frame  <n>  <total_ms>ms
	//   render  full|partial  buf=<w>x<h>  dirty=<n>  clear=<µs>  exec=<µs>
	//   flush   full|diff|overflow  rows=<h>  bytes=<n>  build=<µs>  write=<µs>
	//
	// Only meaningful in inline mode. Default: "" (disabled).
	DebugLog string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Stdin:               os.Stdin,
		Stdout:              os.Stdout,
		TargetFPS:           30,
		MaxFPS:              60,
		UseMouse:            true,
		EnableMouseMovement: false,
		Fullscreen:          true,
		AutoFocus:           true,
		ExitOnCtrlC:         true,
		BackgroundColor:     style.ColorBlack,
		UseKittyKeyboard:    false,
		KittyKeyboardFlags:  KittyFlagDisambiguate | KittyFlagAlternateKeys,
	}
}

// mergeConfig merges user config over defaults.
func mergeConfig(configs []Config) Config {
	cfg := DefaultConfig()

	if len(configs) == 0 {
		return cfg
	}

	user := configs[0]

	if user.Stdin != nil {
		cfg.Stdin = user.Stdin
	}
	if user.Stdout != nil {
		cfg.Stdout = user.Stdout
	}
	if user.TargetFPS > 0 {
		cfg.TargetFPS = user.TargetFPS
	}
	if user.MaxFPS > 0 {
		cfg.MaxFPS = user.MaxFPS
	}

	cfg.UseMouse = user.UseMouse
	cfg.EnableMouseMovement = user.EnableMouseMovement
	cfg.Fullscreen = user.Fullscreen
	cfg.AutoFocus = user.AutoFocus
	cfg.ExitOnCtrlC = user.ExitOnCtrlC
	cfg.InlineMode = user.InlineMode
	cfg.ClearScreenOnStart = user.ClearScreenOnStart
	cfg.UseKittyKeyboard = user.UseKittyKeyboard
	cfg.AltSendsEscPrefix = user.AltSendsEscPrefix
	cfg.GatherStats = user.GatherStats
	if user.WidthProbe.Valid() {
		cfg.WidthProbe = user.WidthProbe
	}
	cfg.WidthProbeSamples = user.WidthProbeSamples

	if cfg.InlineMode {
		cfg.Fullscreen = false
	}

	if user.BackgroundColor != (style.Color{}) {
		cfg.BackgroundColor = user.BackgroundColor
	}
	if user.KittyKeyboardFlags != 0 {
		cfg.KittyKeyboardFlags = user.KittyKeyboardFlags
	}
	if len(user.PostProcessFns) > 0 {
		cfg.PostProcessFns = user.PostProcessFns
	}
	if user.OnDestroy != nil {
		cfg.OnDestroy = user.OnDestroy
	}
	if user.DebugLog != "" {
		cfg.DebugLog = user.DebugLog
	}

	return cfg
}
