package input

import (
	"strings"
	"unicode/utf8"
)

// KeyEventType identifies a key press, release, or repeat event.
type KeyEventType int

const (
	KeyPress KeyEventType = iota
	KeyRelease
	KeyRepeat
)

// String returns the key event type's name.
func (t KeyEventType) String() string {
	switch t {
	case KeyPress:
		return "press"
	case KeyRelease:
		return "release"
	case KeyRepeat:
		return "repeat"
	default:
		return "unknown"
	}
}

// Modifier identifies a keyboard modifier.
type Modifier int

const (
	ModCtrl Modifier = iota
	ModShift
	ModAlt
	ModMeta
)

const (
	KeyReturn    = "return"
	KeyEnter     = "return" // KeyEnter is an alias for KeyReturn.
	KeyEscape    = "escape"
	KeyTab       = "tab"
	KeyBackspace = "backspace"
	KeyDelete    = "delete"
	KeyInsert    = "insert"
	KeyUp        = "up"
	KeyDown      = "down"
	KeyLeft      = "left"
	KeyRight     = "right"
	KeyHome      = "home"
	KeyEnd       = "end"
	KeyPageUp    = "pageup"
	KeyPageDown  = "pagedown"
	KeySpace     = "space"
	KeyF1        = "f1"
	KeyF2        = "f2"
	KeyF3        = "f3"
	KeyF4        = "f4"
	KeyF5        = "f5"
	KeyF6        = "f6"
	KeyF7        = "f7"
	KeyF8        = "f8"
	KeyF9        = "f9"
	KeyF10       = "f10"
	KeyF11       = "f11"
	KeyF12       = "f12"
)

// KeyEvent describes one keyboard input event.
type KeyEvent struct {
	Name     string       // The canonical key name, such as "a", "return", "up", or "f1".
	Text     string       // Printable input text, or empty for a non-printable key.
	Ctrl     bool         // Whether Control was held.
	Shift    bool         // Whether Shift was held.
	Alt      bool         // Whether Alt was held.
	Meta     bool         // Whether a distinct Meta, Super, or Command key was held.
	Raw      string       // The original terminal input sequence.
	Type     KeyEventType // The key's press, release, or repeat state.
	Code     string       // The Kitty keyboard protocol code point, when available.
	Source   string       // The parser source, either "raw" or "kitty".
	Repeated bool         // Whether this is an automatic key repeat.

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation prevents the event from reaching later propagation targets.
func (e *KeyEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents the event's default renderable behavior.
func (e *KeyEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped reports whether StopPropagation was called.
func (e *KeyEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented reports whether PreventDefault was called.
func (e *KeyEvent) DefaultPrevented() bool { return e.defaultPrevented }

// String returns the key combination in canonical modifier order, such as
// "ctrl+shift+a".
func (e *KeyEvent) String() string {
	var parts []string
	if e.Ctrl {
		parts = append(parts, "ctrl")
	}
	if e.Alt {
		parts = append(parts, "alt")
	}
	if e.Meta {
		parts = append(parts, "meta")
	}
	if e.Shift {
		parts = append(parts, "shift")
	}
	parts = append(parts, e.Name)
	return strings.Join(parts, "+")
}

// Matches reports whether the event has name and exactly the supplied
// modifiers.
func (e *KeyEvent) Matches(name string, modifiers ...Modifier) bool {
	if e.Name != name {
		return false
	}

	wantCtrl, wantShift, wantAlt, wantMeta := false, false, false, false
	for _, m := range modifiers {
		switch m {
		case ModCtrl:
			wantCtrl = true
		case ModShift:
			wantShift = true
		case ModAlt:
			wantAlt = true
		case ModMeta:
			wantMeta = true
		}
	}

	return e.Ctrl == wantCtrl &&
		e.Shift == wantShift &&
		e.Alt == wantAlt &&
		e.Meta == wantMeta
}

// IsRune reports whether the event represents printable text without Control,
// Alt, or Meta.
//
// A non-empty Text field identifies printable input. For events constructed
// without Text, IsRune falls back to a Name containing exactly one rune.
func (e *KeyEvent) IsRune() bool {
	if e.Ctrl || e.Alt || e.Meta {
		return false
	}
	if len(e.Text) > 0 {
		return true
	}
	// Support manually constructed events that predate the Text field.
	return utf8.RuneCountInString(e.Name) == 1
}

// Rune returns the first rune in Text, or in Name when Text is empty. It
// returns zero when IsRune reports false.
func (e *KeyEvent) Rune() rune {
	if !e.IsRune() {
		return 0
	}
	if len(e.Text) > 0 {
		r, _ := utf8.DecodeRuneInString(e.Text)
		return r
	}
	r, _ := utf8.DecodeRuneInString(e.Name)
	return r
}
