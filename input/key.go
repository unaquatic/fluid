package input

import (
	"strconv"
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

// Mods is a set of modifier keys held during an input event. Combine them
// with |, as in ModCtrl|ModShift.
type Mods uint8

const (
	// ModCtrl is Control.
	ModCtrl Mods = 1 << iota
	// ModShift is Shift.
	ModShift
	// ModAlt is Alt or Option.
	ModAlt
	// ModMeta is a distinct Meta, Super, or Command key.
	ModMeta
)

// Has reports whether m includes every key in mod.
func (m Mods) Has(mod Mods) bool {
	return m&mod == mod
}

// String returns the keys of m in canonical order, joined by "+", such as
// "ctrl+alt+meta+shift", or "" when m is empty.
func (m Mods) String() string {
	var parts []string
	if m.Has(ModCtrl) {
		parts = append(parts, "ctrl")
	}
	if m.Has(ModAlt) {
		parts = append(parts, "alt")
	}
	if m.Has(ModMeta) {
		parts = append(parts, "meta")
	}
	if m.Has(ModShift) {
		parts = append(parts, "shift")
	}
	return strings.Join(parts, "+")
}

// comboString returns name with the modifiers mods before it in canonical
// order, such as "ctrl+shift+a".
func comboString(name string, mods Mods) string {
	if mods == 0 {
		return name
	}
	return mods.String() + "+" + name
}

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

// KeySource identifies the encoding a key event was read from.
type KeySource int

const (
	// KeySourceRaw is a legacy terminal key sequence or a printable byte.
	KeySourceRaw KeySource = iota
	// KeySourceKitty is a Kitty keyboard protocol sequence.
	KeySourceKitty
)

// String returns s's name, such as "kitty".
func (s KeySource) String() string {
	switch s {
	case KeySourceRaw:
		return "raw"
	case KeySourceKitty:
		return "kitty"
	}
	return "KeySource(" + strconv.Itoa(int(s)) + ")"
}

// KeyEvent describes one keyboard input event.
type KeyEvent struct {
	// Name is the canonical key name, such as "a", "return", "up", or "f1".
	Name string
	// Text is the printable input text, or empty for a non-printable key.
	Text string
	// Mods are the modifier keys held.
	Mods Mods
	// Raw is the original terminal input sequence.
	Raw string
	// Type is the key's press, release, or repeat state.
	Type KeyEventType
	// Code is the Kitty keyboard protocol code point, when available.
	Code string
	// Source is the encoding the parser read the key from.
	Source KeySource

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
	return comboString(e.Name, e.Mods)
}

// Matches reports whether the event has name and exactly the modifiers mods.
func (e *KeyEvent) Matches(name string, mods Mods) bool {
	return e.Name == name && e.Mods == mods
}

// IsRune reports whether the event represents printable text without Control,
// Alt, or Meta.
//
// A non-empty Text field identifies printable input. For events constructed
// without Text, IsRune falls back to a Name containing exactly one rune.
func (e *KeyEvent) IsRune() bool {
	if e.Mods&(ModCtrl|ModAlt|ModMeta) != 0 {
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
