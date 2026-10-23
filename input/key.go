package input

import (
	"strings"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// KeyEventType
// ---------------------------------------------------------------------------

// KeyEventType distinguishes press, release, and repeat.
type KeyEventType int

const (
	KeyPress KeyEventType = iota
	KeyRelease
	KeyRepeat
)

// String returns a human-readable name for the key event type.
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

// ---------------------------------------------------------------------------
// Modifier
// ---------------------------------------------------------------------------

// Modifier represents a keyboard modifier key.
type Modifier int

const (
	ModCtrl Modifier = iota
	ModShift
	ModAlt
	ModMeta
)

// ---------------------------------------------------------------------------
// Key Name Constants
// ---------------------------------------------------------------------------

const (
	KeyReturn    = "return"
	KeyEnter     = "return" // alias
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

// ---------------------------------------------------------------------------
// KeyEvent
// ---------------------------------------------------------------------------

// KeyEvent represents a keyboard input event.
type KeyEvent struct {
	Name     string       // Key name: "a", "space", "return", "escape", "up", "f1", "tab", etc.
	Text     string       // Printable text: "a", " ", etc. Empty for non-printable keys (arrows, function keys, etc.)
	Ctrl     bool         // Ctrl modifier
	Shift    bool         // Shift modifier
	Alt      bool         // Alt/Meta modifier
	Meta     bool         // Meta modifier (distinct from Alt on some systems)
	Raw      string       // Raw input escape sequence
	Type     KeyEventType // press, release, repeat
	Code     string       // Key code (for Kitty protocol)
	Source   string       // "raw" or "kitty"
	Repeated bool         // Whether key is auto-repeated

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation stops the event from reaching further handlers.
func (e *KeyEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents renderable handlers from processing.
func (e *KeyEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped returns true if propagation was stopped.
func (e *KeyEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented returns true if default was prevented.
func (e *KeyEvent) DefaultPrevented() bool { return e.defaultPrevented }

// String returns a human-readable form, e.g. "ctrl+shift+a".
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

// Matches checks if the event matches a key name and optional modifiers.
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

// IsRune returns true if this event represents a printable character
// without Ctrl, Alt, or Meta modifiers.
//
// When the Text field is populated (as done by the parser), it is the
// authoritative signal. When Text is empty (e.g. manually constructed
// events in tests), the method falls back to checking whether Name is
// a single rune — preserving backwards compatibility.
func (e *KeyEvent) IsRune() bool {
	if e.Ctrl || e.Alt || e.Meta {
		return false
	}
	if len(e.Text) > 0 {
		return true
	}
	// Fallback for events constructed without Text.
	return utf8.RuneCountInString(e.Name) == 1
}

// Rune returns the rune if IsRune(), else 0.
// Prefers the Text field when available, falling back to Name.
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
