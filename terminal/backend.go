package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/unaquatic/fluid/style"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// trueColorPrograms holds lowercase TERM_PROGRAM values of 24-bit color terminals.
var trueColorPrograms = []string{
	"alacritty", "ghostty", "hyper", "iterm.app", "kitty",
	"vscode", "warpterminal", "wezterm", "zed",
}

// trueColorMarkers holds environment variables set by 24-bit color terminals.
var trueColorMarkers = []string{
	"GHOSTTY_RESOURCES_DIR", "ITERM_SESSION_ID", "KITTY_WINDOW_ID",
	"WARP_SESSION_ID", "WARP_TERMINAL_SESSION_UUID", "WEZTERM_PANE", "WT_SESSION",
}

// DetectColorMode infers the terminal color mode from the environment.
// FLUID_COLOR set to "truecolor", "24bit", "256", or "16" overrides it.
func DetectColorMode() style.ColorMode {
	switch strings.ToLower(os.Getenv("FLUID_COLOR")) {
	case "truecolor", "24bit":
		return style.ColorTrueColor
	case "256":
		return style.Color256
	case "16":
		return style.Color16
	}
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	advertised := colorTerm == "truecolor" || colorTerm == "24bit" || strings.HasSuffix(term, "-direct")
	multiplexed := os.Getenv("TMUX") != "" || strings.HasPrefix(term, "tmux") || strings.HasPrefix(term, "screen")
	knownTerminal := slices.Contains(trueColorPrograms, program) ||
		slices.ContainsFunc(trueColorMarkers, func(k string) bool { return os.Getenv(k) != "" }) ||
		strings.Contains(term, "ghostty") ||
		strings.EqualFold(os.Getenv("TERMINAL_EMULATOR"), "jetbrains-jediterm")

	switch {
	case advertised:
		return style.ColorTrueColor
	case multiplexed:
		return style.Color256
	case knownTerminal:
		return style.ColorTrueColor
	case strings.Contains(term, "256color"):
		return style.Color256
	case term == "" || term == "dumb":
		return style.Color16
	default:
		return style.ColorTrueColor
	}
}

// Backend controls terminal modes and writes ANSI and xterm escape sequences.
// Its output methods are safe for concurrent use.
type Backend struct {
	stdin  io.Reader
	stdout io.Writer

	oldState  *term.State // The terminal state saved before raw mode.
	colorMode style.ColorMode
	indexers  map[style.ColorMode]style.Indexers
	width     int
	height    int

	mouseEnabled         bool
	kittyKeyboardEnabled bool
	kittyFlags           uint
	alternateScreen      bool

	cursorStyle    CursorStyle
	cursorBlinking bool

	mu sync.Mutex // Protects terminal output writes.
}

// NewBackend returns a backend using stdin and stdout. It detects color support
// from the process environment.
func NewBackend(stdin io.Reader, stdout io.Writer) *Backend {
	return &Backend{
		stdin:     stdin,
		stdout:    stdout,
		colorMode: DetectColorMode(),
	}
}

// ColorMode returns the configured terminal color capability.
func (t *Backend) ColorMode() style.ColorMode {
	return t.colorMode
}

// SetColorMode overrides the auto-detected color mode.
func (t *Backend) SetColorMode(m style.ColorMode) {
	t.colorMode = m
}

// Indexers returns the Indexers of the configured color mode, which
// SetIndexers set, or the zero Indexers, which convert colors as the mode
// does by default.
func (t *Backend) Indexers() style.Indexers {
	return t.indexers[t.colorMode]
}

// SetIndexers sets the Indexers that convert colors to palette entries while
// the color mode is m. The zero Indexers convert them as m does by default.
func (t *Backend) SetIndexers(m style.ColorMode, x style.Indexers) {
	if t.indexers == nil {
		t.indexers = make(map[style.ColorMode]style.Indexers)
	}
	t.indexers[m] = x
}

// Setup enters raw mode when stdin is a terminal, records its size, optionally
// enters the alternate screen, enables bracketed paste, and hides the cursor.
func (t *Backend) Setup(useAlternateScreen bool) error {
	// Raw mode is available only for an os.File terminal.
	if f, ok := t.stdin.(*os.File); ok {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return fmt.Errorf("terminal: failed to enter raw mode: %w", err)
		}
		t.oldState = oldState
	}

	// A failed size query leaves the cached dimensions unchanged.
	w, h, err := t.Size()
	if err == nil {
		t.width = w
		t.height = h
	}

	if useAlternateScreen {
		t.EnterAlternateScreen()
	}

	t.EnableBracketedPaste()

	t.HideCursor()

	return nil
}

// Restore shows the cursor, disables enabled input modes, leaves the alternate
// screen, and restores the saved terminal state.
func (t *Backend) Restore() error {
	t.ShowCursor()

	t.DisableBracketedPaste()

	if t.mouseEnabled {
		t.DisableMouse()
	}

	if t.kittyKeyboardEnabled {
		t.DisableKittyKeyboard()
	}

	if t.alternateScreen {
		t.LeaveAlternateScreen()
	}

	if t.oldState != nil {
		if f, ok := t.stdin.(*os.File); ok {
			return term.Restore(int(f.Fd()), t.oldState)
		}
	}

	return nil
}

// Size returns terminal dimensions from stdout or stdin using TIOCGWINSZ.
func (t *Backend) Size() (width, height int, err error) {
	// Prefer stdout because it identifies the rendered terminal.
	if f, ok := t.stdout.(*os.File); ok {
		ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
		if err == nil {
			return int(ws.Col), int(ws.Row), nil
		}
	}
	if f, ok := t.stdin.(*os.File); ok {
		ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
		if err == nil {
			return int(ws.Col), int(ws.Row), nil
		}
	}
	return 0, 0, errors.New("terminal: unable to get terminal size")
}

// EnableMouse enables button, drag, and SGR mouse reporting. enableMovement
// also enables motion reporting without a pressed button.
func (t *Backend) EnableMouse(enableMovement bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Report button presses and releases.
	t.writeRaw("\x1b[?1000h")
	// Report motion while a button is pressed.
	t.writeRaw("\x1b[?1002h")
	// Optionally report all pointer motion.
	if enableMovement {
		t.writeRaw("\x1b[?1003h")
	}
	// Encode reports with SGR coordinates.
	t.writeRaw("\x1b[?1006h")

	t.mouseEnabled = true
}

// DisableMouse disables all mouse tracking.
func (t *Backend) DisableMouse() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l")
	t.mouseEnabled = false
}

// EnableKittyKeyboard enables Kitty keyboard reporting with flags.
func (t *Backend) EnableKittyKeyboard(flags uint) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b[>%du", flags))
	t.kittyKeyboardEnabled = true
	t.kittyFlags = flags
}

// DisableKittyKeyboard disables the Kitty keyboard protocol.
func (t *Backend) DisableKittyKeyboard() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[<u")
	t.kittyKeyboardEnabled = false
	t.kittyFlags = 0
}

// SetKittyKeyboardFlags writes new Kitty flags without disabling the protocol.
func (t *Backend) SetKittyKeyboardFlags(flags uint) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b[>%du", flags))
	t.kittyFlags = flags
}

// KittyKeyboardFlags returns the current Kitty keyboard flags.
func (t *Backend) KittyKeyboardFlags() uint {
	return t.kittyFlags
}

// SetCursorPosition moves the hardware cursor to zero-based (x, y) and sets its
// visibility.
func (t *Backend) SetCursorPosition(x, y int, visible bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
	if visible {
		t.writeRaw("\x1b[?25h")
	} else {
		t.writeRaw("\x1b[?25l")
	}
}

// SetCursorStyle updates the cursor shape while preserving its blinking state.
func (t *Backend) SetCursorStyle(cursorStyle CursorStyle) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorStyle = cursorStyle
	t.writeCursorShape()
}

// SetCursorBlinking updates cursor blinking while preserving its shape.
func (t *Backend) SetCursorBlinking(blinking bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorBlinking = blinking
	t.writeCursorShape()
}

// CursorStyle returns the current cursor shape.
func (t *Backend) CursorStyle() CursorStyle {
	return t.cursorStyle
}

// CursorBlinking returns whether the cursor is blinking.
func (t *Backend) CursorBlinking() bool {
	return t.cursorBlinking
}

// writeCursorShape writes the DECSCUSR sequence for the current shape and blink
// state. The caller must hold t.mu.
func (t *Backend) writeCursorShape() {
	// DECSCUSR uses adjacent odd and even values for blinking and steady
	// shapes, 2 for a block, 4 for an underline, and 6 for a line, and 0 for
	// the terminal's default.
	var base int
	if t.cursorStyle >= CursorBlock && t.cursorStyle <= CursorLine {
		base = 2 * int(t.cursorStyle)
	}

	if base != 0 && t.cursorBlinking {
		base-- // The odd code selects the blinking variant.
	}

	t.writeRaw(fmt.Sprintf("\x1b[%d q", base))
}

// SetCursorColor sets the cursor color via OSC 12.
func (t *Backend) SetCursorColor(r, g, b uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b]12;rgb:%02x/%02x/%02x\x1b\\", r, g, b))
}

// ResetCursorColor resets the cursor color to the terminal default via OSC 112.
func (t *Backend) ResetCursorColor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b]112\x1b\\")
}

// SetMousePointer writes pointerStyle using OSC 22 without validating it.
func (t *Backend) SetMousePointer(pointerStyle MousePointerStyle) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b]22;%s\x1b\\", pointerStyle))
}

// ShowCursor makes the cursor visible.
func (t *Backend) ShowCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?25h")
}

// HideCursor makes the cursor invisible.
func (t *Backend) HideCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?25l")
}

// SetTitle sets the terminal window and tab title via OSC 0.
func (t *Backend) SetTitle(title string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b]0;%s\x07", title))
}

// SetProgress shows an indeterminate progress indicator via OSC 9;4 when
// active is true, and clears it when active is false.
func (t *Backend) SetProgress(active bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if active {
		t.writeRaw("\x1b]9;4;3\x07")
	} else {
		t.writeRaw("\x1b]9;4;0\x07")
	}
}

// Clear clears the terminal screen.
func (t *Backend) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[2J\x1b[H")
}

// WriteOut writes data to stdout while holding the output lock.
func (t *Backend) WriteOut(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.stdout.Write(data)
}

// EnableBracketedPaste enables bracketed paste mode.
func (t *Backend) EnableBracketedPaste() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?2004h")
}

// DisableBracketedPaste disables bracketed paste mode.
func (t *Backend) DisableBracketedPaste() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?2004l")
}

// AlternateScreen reports whether the alternate screen buffer is active.
func (t *Backend) AlternateScreen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.alternateScreen
}

// EnterAlternateScreen switches to the alternate screen buffer.
func (t *Backend) EnterAlternateScreen() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?1049h")
	t.alternateScreen = true
}

// LeaveAlternateScreen returns to the main screen buffer.
func (t *Backend) LeaveAlternateScreen() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[?1049l")
	t.alternateScreen = false
}

// writeRaw writes s to stdout and discards any write error. The caller must hold
// t.mu.
func (t *Backend) writeRaw(s string) {
	_, _ = io.WriteString(t.stdout, s)
}
