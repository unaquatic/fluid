package input

import "testing"

func TestKeyEventType_String(t *testing.T) {
	tests := []struct {
		t    KeyEventType
		want string
	}{
		{KeyPress, "press"},
		{KeyRelease, "release"},
		{KeyRepeat, "repeat"},
	}
	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("KeyEventType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestKeyEvent_String(t *testing.T) {
	tests := []struct {
		ev   KeyEvent
		want string
	}{
		{KeyEvent{Name: "a"}, "a"},
		{KeyEvent{Name: "a", Ctrl: true}, "ctrl+a"},
		{KeyEvent{Name: "a", Alt: true}, "alt+a"},
		{KeyEvent{Name: "a", Shift: true}, "shift+a"},
		{KeyEvent{Name: "a", Ctrl: true, Shift: true}, "ctrl+shift+a"},
		{KeyEvent{Name: "a", Ctrl: true, Alt: true, Shift: true}, "ctrl+alt+shift+a"},
		{KeyEvent{Name: "up"}, "up"},
		{KeyEvent{Name: "return", Meta: true}, "meta+return"},
	}
	for _, tt := range tests {
		if got := tt.ev.String(); got != tt.want {
			t.Errorf("KeyEvent.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestKeyEvent_Matches(t *testing.T) {
	ev := KeyEvent{Name: "a", Ctrl: true}

	if !ev.Matches("a", ModCtrl) {
		t.Error("should match ctrl+a")
	}
	if ev.Matches("a") {
		t.Error("should not match plain 'a' (has ctrl)")
	}
	if ev.Matches("b", ModCtrl) {
		t.Error("should not match ctrl+b")
	}
	if ev.Matches("a", ModCtrl, ModShift) {
		t.Error("should not match ctrl+shift+a (no shift)")
	}
}

func TestKeyEvent_Matches_NoModifiers(t *testing.T) {
	ev := KeyEvent{Name: "return"}
	if !ev.Matches("return") {
		t.Error("should match plain return")
	}
	if ev.Matches("return", ModCtrl) {
		t.Error("should not match ctrl+return")
	}
}

func TestKeyEvent_Matches_MultipleModifiers(t *testing.T) {
	ev := KeyEvent{Name: "x", Ctrl: true, Shift: true, Alt: true}

	if !ev.Matches("x", ModCtrl, ModShift, ModAlt) {
		t.Error("should match ctrl+shift+alt+x")
	}
	if ev.Matches("x", ModCtrl, ModShift) {
		t.Error("should not match without alt")
	}
}

func TestKeyEvent_IsRune(t *testing.T) {
	tests := []struct {
		ev   KeyEvent
		want bool
		desc string
	}{
		// Backwards-compatible: Name-only (no Text field)
		{KeyEvent{Name: "a"}, true, "plain 'a' (fallback, no Text)"},
		{KeyEvent{Name: "Z"}, true, "plain 'Z' (fallback, no Text)"},
		{KeyEvent{Name: "\u4e16"}, true, "CJK (fallback, no Text)"},
		{KeyEvent{Name: "return"}, false, "return (multi-char Name, no Text)"},
		{KeyEvent{Name: "a", Ctrl: true}, false, "ctrl+a"},
		{KeyEvent{Name: "a", Alt: true}, false, "alt+a"},
		{KeyEvent{Name: "a", Meta: true}, false, "meta+a"},
		// Text field (as produced by the parser)
		{KeyEvent{Name: "a", Text: "a"}, true, "'a' with Text"},
		{KeyEvent{Name: KeySpace, Text: " "}, true, "space with Text"},
		{KeyEvent{Name: KeySpace}, false, "space without Text (fallback: multi-char Name)"},
		{KeyEvent{Name: KeySpace, Text: " ", Ctrl: true}, false, "ctrl+space"},
		{KeyEvent{Name: KeyReturn}, false, "return key"},
		{KeyEvent{Name: KeyTab}, false, "tab key"},
	}
	for _, tt := range tests {
		if got := tt.ev.IsRune(); got != tt.want {
			t.Errorf("%s: IsRune() = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestKeyEvent_Rune(t *testing.T) {
	// Backwards-compatible: Name-only (no Text field)
	ev := KeyEvent{Name: "x"}
	if r := ev.Rune(); r != 'x' {
		t.Errorf("Rune() = %q, want 'x'", r)
	}

	ev2 := KeyEvent{Name: "return"}
	if r := ev2.Rune(); r != 0 {
		t.Errorf("Rune() for non-rune = %q, want 0", r)
	}

	ev3 := KeyEvent{Name: "a", Ctrl: true}
	if r := ev3.Rune(); r != 0 {
		t.Errorf("Rune() for ctrl = %q, want 0", r)
	}

	// Text field (as produced by the parser)
	ev4 := KeyEvent{Name: KeySpace, Text: " "}
	if r := ev4.Rune(); r != ' ' {
		t.Errorf("Rune() for space with Text = %q, want ' '", r)
	}

	ev5 := KeyEvent{Name: KeySpace, Text: " ", Ctrl: true}
	if r := ev5.Rune(); r != 0 {
		t.Errorf("Rune() for ctrl+space = %q, want 0", r)
	}

	ev6 := KeyEvent{Name: "a", Text: "a"}
	if r := ev6.Rune(); r != 'a' {
		t.Errorf("Rune() for 'a' with Text = %q, want 'a'", r)
	}
}

func TestKeyEvent_PropagationControl(t *testing.T) {
	ev := KeyEvent{Name: "a"}

	if ev.PropagationStopped() {
		t.Error("should not be stopped initially")
	}
	if ev.DefaultPrevented() {
		t.Error("should not be prevented initially")
	}

	ev.StopPropagation()
	if !ev.PropagationStopped() {
		t.Error("should be stopped after StopPropagation")
	}

	ev.PreventDefault()
	if !ev.DefaultPrevented() {
		t.Error("should be prevented after PreventDefault")
	}
}

func TestKeyConstants(t *testing.T) {
	// Verify aliases
	if KeyEnter != KeyReturn {
		t.Error("KeyEnter should equal KeyReturn")
	}
	if KeyReturn != "return" {
		t.Error("KeyReturn should be 'return'")
	}
	if KeyEscape != "escape" {
		t.Error("KeyEscape should be 'escape'")
	}
	if KeyTab != "tab" {
		t.Error("KeyTab should be 'tab'")
	}
	if KeyBackspace != "backspace" {
		t.Error("KeyBackspace should be 'backspace'")
	}
	if KeySpace != "space" {
		t.Error("KeySpace should be 'space'")
	}
	if KeyF12 != "f12" {
		t.Error("KeyF12 should be 'f12'")
	}
}
