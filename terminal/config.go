package terminal

import (
	"io"
	"os"
	"strconv"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/style"
)

// ControlState identifies the renderer's lifecycle state.
type ControlState int

const (
	// StateIdle means terminal setup has not started.
	StateIdle ControlState = iota
	// StateAutoStarted renders only while a frame is requested.
	StateAutoStarted
	// StateExplicitStarted renders whenever the event loop wakes, regardless of
	// whether the scene was dirty.
	StateExplicitStarted
	// StateExplicitPaused keeps the event loop alive without rendering frames.
	StateExplicitPaused
	// StateExplicitSuspended means terminal modes are restored and rendering is
	// suspended.
	StateExplicitSuspended
	// StateExplicitStopped means the render loop has exited.
	StateExplicitStopped
)

// String returns s's name.
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

// Selected Kitty keyboard flags re-exported from package input.
const (
	KittyFlagDisambiguate  = input.KittyFlagDisambiguate
	KittyFlagAlternateKeys = input.KittyFlagAlternateKeys
)

// CursorStyle identifies a terminal cursor shape. The zero value,
// CursorDefault, leaves the shape to the terminal.
type CursorStyle int

const (
	// CursorDefault is the terminal's own cursor shape.
	CursorDefault CursorStyle = iota
	// CursorBlock is a rectangular cursor.
	CursorBlock
	// CursorUnderline is a horizontal cursor below the cell content.
	CursorUnderline
	// CursorLine is a thin vertical cursor.
	CursorLine
)

// String returns s's name, such as "block".
func (s CursorStyle) String() string {
	switch s {
	case CursorDefault:
		return "default"
	case CursorBlock:
		return "block"
	case CursorUnderline:
		return "underline"
	case CursorLine:
		return "line"
	}
	return "CursorStyle(" + strconv.Itoa(int(s)) + ")"
}

// MousePointerStyle identifies a pointer shape for terminals supporting OSC
// 22, which takes a CSS cursor name, such as "pointer".
type MousePointerStyle string

const (
	// MousePointerDefault is the standard arrow pointer.
	MousePointerDefault MousePointerStyle = "default"
	// MousePointerPointer is a hand pointer for clickable elements.
	MousePointerPointer MousePointerStyle = "pointer"
	// MousePointerText is an I-beam pointer for text.
	MousePointerText MousePointerStyle = "text"
	// MousePointerCrosshair is a crosshair pointer.
	MousePointerCrosshair MousePointerStyle = "crosshair"
	// MousePointerMove is a four-way pointer for draggable elements.
	MousePointerMove MousePointerStyle = "move"
	// MousePointerNotAllowed marks a disabled interaction.
	MousePointerNotAllowed MousePointerStyle = "not-allowed"
)

// CursorStyleOptions contains optional cursor and pointer changes. A nil field
// preserves its current value.
type CursorStyleOptions struct {
	// Style overrides the cursor shape.
	Style *CursorStyle
	// Blinking overrides the cursor blink state.
	Blinking *bool
	// Color overrides the cursor color.
	Color *style.Color
	// Cursor overrides the mouse pointer style.
	Cursor *MousePointerStyle
}

// CursorState contains the terminal cursor state for a frame.
type CursorState struct {
	// X and Y are zero-based absolute terminal coordinates.
	X, Y int
	// Visible controls whether the cursor is shown.
	Visible bool
	// Style is the cursor shape.
	Style CursorStyle
	// Blinking controls whether the cursor blinks.
	Blinking bool
	// Color is the cursor color. A zero value selects the terminal default.
	Color style.Color
}

// Config controls terminal setup, rendering, input, and diagnostics.
type Config struct {
	// Stdin receives terminal input. The default is os.Stdin.
	Stdin io.Reader

	// Stdout receives terminal output. The default is os.Stdout.
	Stdout io.Writer

	// TargetFPS records the desired render-loop rate. It is currently not used by
	// the event loop. The default is 30.
	TargetFPS int

	// MaxFPS records the desired render-rate limit. It is currently not enforced.
	// The default is 60.
	MaxFPS int

	// UseMouse enables click, drag, and scroll reporting. The default is true.
	UseMouse bool

	// EnableMouseMovement enables reporting for motion without a pressed button.
	// It applies only when UseMouse is true and defaults to false.
	EnableMouseMovement bool

	// Fullscreen uses the alternate screen. InlineMode overrides it to false. The
	// default is true.
	Fullscreen bool

	// AutoFocus focuses a focusable renderable on mouse down. The default is true.
	AutoFocus bool

	// ExitOnCtrlC destroys the renderer after an unprevented Control-C event. The
	// default is true.
	ExitOnCtrlC bool

	// InlineMode renders in the normal screen and scrollback instead of the
	// alternate screen. Content determines height; terminal width minus a
	// one-column wrap margin determines width.
	//
	// Changed rows are updated in place while content fits. Overflow triggers a
	// complete screen and scrollback rewrite. The default is false.
	InlineMode bool

	// ClearScreenOnStart clears the screen and scrollback before the first inline
	// render. The default is false.
	ClearScreenOnStart bool

	// ClearOnExit erases inline output when the renderer is destroyed, so the
	// terminal looks as it did before the renderer drew it. Output that reached
	// the scrollback, or that a resize may have moved, can only be erased with
	// the whole screen and scrollback.
	//
	// The default is false, which leaves inline output in place with the cursor
	// on a clean line below it. Quitting from the alternate screen erases the
	// inline output it covered either way.
	ClearOnExit bool

	// BackgroundColor fills the frame before widgets render. The default is
	// style.ColorBlack.
	BackgroundColor style.Color

	// UseKittyKeyboard enables Kitty keyboard protocol parsing and terminal mode.
	// The default is false.
	UseKittyKeyboard bool

	// KittyKeyboardFlags is the bitwise union of enabled Kitty features.
	// Default: KittyFlagDisambiguate | KittyFlagAlternateKeys.
	KittyKeyboardFlags uint

	// AltSendsEscPrefix enables the parser heuristic for terminals that encode
	// Alt navigation with ESC prefixes. The default is false.
	AltSendsEscPrefix bool

	// OnDestroy runs after terminal restoration and scene destruction.
	OnDestroy func()

	// GatherStats enables per-frame timing statistics. The default is false.
	GatherStats bool

	// WidthProbe controls startup measurement of terminal cell widths. The
	// default is WidthProbeOff.
	WidthProbe WidthProbe

	// WidthProbeSamples contains strings measured by WidthProbeAuto.
	WidthProbeSamples []string

	// DebugLog enables diagnostics at the given path, whose parent directory
	// must exist. Each record is a line of log/slog text, its time, level, and
	// message followed by these attributes:
	//
	//   msg=flush kind=full|diff bytes=<n> top=<row> cursor=<row>   each inline flush
	//   msg=frame n=<count> width=<w> height=<h>                    each full fullscreen repaint
	//   msg=resize width=<w> height=<h>
	//   msg="probe measured" widths=<widths>, or msg="probe unavailable" error=<err>
	//   msg="debug log opened" pid=<pid>, and msg="debug log closed" frames=<count>
	//
	// Debug logging is disabled by default.
	DebugLog string
}

// DefaultConfig returns the default terminal configuration.
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

// mergeConfig overlays the first supplied configuration on DefaultConfig.
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
	cfg.ClearOnExit = user.ClearOnExit
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
	if user.OnDestroy != nil {
		cfg.OnDestroy = user.OnDestroy
	}
	if user.DebugLog != "" {
		cfg.DebugLog = user.DebugLog
	}

	return cfg
}
