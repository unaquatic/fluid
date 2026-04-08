package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/unaquatic/fluid/style"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// DetectColorMode checks environment variables to determine color support.
func DetectColorMode() style.ColorMode {
	// COLORTERM=truecolor or COLORTERM=24bit → true color
	ct := os.Getenv("COLORTERM")
	if ct == "truecolor" || ct == "24bit" {
		return style.ColorTrueColor
	}

	// Many modern terminals set COLORTERM but some don't.
	// Check TERM for 256-color indicators.
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		return style.Color256
	}

	// Common terminals that support true color but may not set COLORTERM:
	// iTerm, kitty, WezTerm, Alacritty, Windows Terminal, etc.
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app", "WezTerm", "Hyper":
		return style.ColorTrueColor
	}

	// WT_SESSION is set by Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		return style.ColorTrueColor
	}

	// KITTY_WINDOW_ID for kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return style.ColorTrueColor
	}

	// Default: if TERM is set and not "dumb", assume 256
	if term != "" && term != "dumb" {
		return style.Color256
	}

	return style.Color16
}

// ---------------------------------------------------------------------------
// Backend
// ---------------------------------------------------------------------------

// Backend provides low-level terminal control: raw mode setup/teardown,
// alternate screen management, mouse protocol, Kitty keyboard protocol,
// cursor styling, and escape sequence output. It is a pure Go
// implementation using golang.org/x/term and ANSI/xterm escape sequences.
// The Renderer owns a Backend and calls its methods during setup, flush,
// and teardown.
type Backend struct {
	stdin  io.Reader
	stdout io.Writer

	oldState  *term.State // saved terminal state for restore
	colorMode style.ColorMode
	width     int
	height    int

	mouseEnabled         bool
	kittyKeyboardEnabled bool
	kittyFlags           uint
	alternateScreen      bool

	cursorStyle    string // "block", "line", "underline"
	cursorBlinking bool

	mu sync.Mutex // protects stdout writes
}

// NewBackend creates a new Backend backed by the given reader and writer.
// The terminal's color mode is auto-detected from environment variables.
func NewBackend(stdin io.Reader, stdout io.Writer) *Backend {
	return &Backend{
		stdin:     stdin,
		stdout:    stdout,
		colorMode: DetectColorMode(),
	}
}

// ColorMode returns the terminal's color capability.
func (t *Backend) ColorMode() style.ColorMode {
	return t.colorMode
}

// SetColorMode overrides the auto-detected color mode.
func (t *Backend) SetColorMode(m style.ColorMode) {
	t.colorMode = m
}

// ---------------------------------------------------------------------------
// Setup & Teardown
// ---------------------------------------------------------------------------

// Setup enters raw mode, optionally enters alternate screen, and enables
// bracketed paste.
func (t *Backend) Setup(useAlternateScreen bool) error {
	// Enter raw mode on stdin if it's a real terminal
	if f, ok := t.stdin.(*os.File); ok {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return fmt.Errorf("terminal: failed to enter raw mode: %w", err)
		}
		t.oldState = oldState
	}

	// Query size
	w, h, err := t.Size()
	if err == nil {
		t.width = w
		t.height = h
	}

	// Alternate screen
	if useAlternateScreen {
		t.EnterAlternateScreen()
	}

	// Enable bracketed paste
	t.EnableBracketedPaste()

	// Hide cursor during rendering
	t.HideCursor()

	return nil
}

// Restore restores the terminal to its original state, leaves alternate screen,
// and disables mouse tracking.
func (t *Backend) Restore() error {
	// Show cursor
	t.ShowCursor()

	// Disable bracketed paste
	t.DisableBracketedPaste()

	// Disable mouse if enabled
	if t.mouseEnabled {
		t.DisableMouse()
	}

	// Disable Kitty keyboard if enabled
	if t.kittyKeyboardEnabled {
		t.DisableKittyKeyboard()
	}

	// Leave alternate screen
	if t.alternateScreen {
		t.LeaveAlternateScreen()
	}

	// Restore terminal state
	if t.oldState != nil {
		if f, ok := t.stdin.(*os.File); ok {
			return term.Restore(int(f.Fd()), t.oldState)
		}
	}

	return nil
}

// Size queries the terminal dimensions via TIOCGWINSZ.
func (t *Backend) Size() (width, height int, err error) {
	// Try stdout first, then stdin
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

// ---------------------------------------------------------------------------
// Mouse Control
// ---------------------------------------------------------------------------

// EnableMouse enables SGR extended mouse mode.
// If enableMovement is true, also tracks mouse motion.
func (t *Backend) EnableMouse(enableMovement bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Button events
	t.writeRaw("\x1b[?1000h")
	// Button + drag
	t.writeRaw("\x1b[?1002h")
	// All motion (if requested)
	if enableMovement {
		t.writeRaw("\x1b[?1003h")
	}
	// SGR extended mode
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

// ---------------------------------------------------------------------------
// Keyboard Protocol
// ---------------------------------------------------------------------------

// EnableKittyKeyboard enables the Kitty keyboard protocol with the given flags.
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

// SetKittyKeyboardFlags updates the Kitty keyboard flags without toggling.
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

// ---------------------------------------------------------------------------
// Cursor Control
// ---------------------------------------------------------------------------

// SetCursorPosition moves the cursor to (x, y) and shows/hides it.
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

// SetCursorStyle sets the cursor shape: "block", "line", "underline".
// Respects the current blinking state.
func (t *Backend) SetCursorStyle(cursorStyle string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorStyle = cursorStyle
	t.writeCursorShape()
}

// SetCursorBlinking sets whether the cursor blinks.
// Respects the current cursor style.
func (t *Backend) SetCursorBlinking(blinking bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursorBlinking = blinking
	t.writeCursorShape()
}

// CursorStyle returns the current cursor style name.
func (t *Backend) CursorStyle() string {
	return t.cursorStyle
}

// CursorBlinking returns whether the cursor is blinking.
func (t *Backend) CursorBlinking() bool {
	return t.cursorBlinking
}

// writeCursorShape writes the DECSCUSR escape for the current style+blinking.
// Caller must hold t.mu.
func (t *Backend) writeCursorShape() {
	// DECSCUSR values:
	//   0 = default
	//   1 = blinking block,   2 = steady block
	//   3 = blinking underline, 4 = steady underline
	//   5 = blinking bar,     6 = steady bar
	var base int
	switch t.cursorStyle {
	case "block":
		base = 2
	case "underline":
		base = 4
	case "line", "bar":
		base = 6
	default:
		base = 0
	}

	if base != 0 && t.cursorBlinking {
		base-- // odd = blinking variant
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

// SetMousePointer sets the mouse pointer shape via OSC 22.
// Supported styles: "default", "pointer", "text", "crosshair", "move", "not-allowed".
func (t *Backend) SetMousePointer(pointerStyle string) {
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

// ---------------------------------------------------------------------------
// Output
// ---------------------------------------------------------------------------

// SetTitle sets the terminal window title via OSC 2.
func (t *Backend) SetTitle(title string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw(fmt.Sprintf("\x1b]2;%s\x07", title))
}

// Clear clears the terminal screen.
func (t *Backend) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.writeRaw("\x1b[2J\x1b[H")
}

// WriteOut writes raw bytes to stdout (mutex-protected).
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

// ---------------------------------------------------------------------------
// Alternate Screen
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

// writeRaw writes a raw string to stdout without locking.
// Caller must hold t.mu.
func (t *Backend) writeRaw(s string) {
	_, _ = io.WriteString(t.stdout, s)
}
