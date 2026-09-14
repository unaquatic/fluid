package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func newTestBackend() (*Backend, *bytes.Buffer) {
	var buf bytes.Buffer
	t := NewBackend(strings.NewReader(""), &buf)
	return t, &buf
}

func TestTerminal_EnableMouse(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableMouse(false)

	output := buf.String()

	// Mouse mode includes button press and release reporting.
	if !strings.Contains(output, "\x1b[?1000h") {
		t.Error("expected button events escape")
	}
	// Mouse mode includes drag reporting.
	if !strings.Contains(output, "\x1b[?1002h") {
		t.Error("expected button+drag escape")
	}
	// Disabled movement omits motion without a pressed button.
	if strings.Contains(output, "\x1b[?1003h") {
		t.Error("should not enable all motion without enableMovement")
	}
	// Mouse reports use SGR encoding.
	if !strings.Contains(output, "\x1b[?1006h") {
		t.Error("expected SGR extended mode escape")
	}

	if !term.mouseEnabled {
		t.Error("expected mouseEnabled to be true")
	}
}

func TestTerminal_EnableMouseMovement(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableMouse(true)

	output := buf.String()

	if !strings.Contains(output, "\x1b[?1003h") {
		t.Error("expected all motion escape when enableMovement is true")
	}

	if !term.mouseEnabled {
		t.Error("expected mouseEnabled to be true")
	}
}

func TestTerminal_DisableMouse(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableMouse(false)
	buf.Reset()

	term.DisableMouse()

	output := buf.String()
	if !strings.Contains(output, "\x1b[?1000l") {
		t.Error("expected mouse disable escape")
	}
	if !strings.Contains(output, "\x1b[?1006l") {
		t.Error("expected SGR disable escape")
	}

	if term.mouseEnabled {
		t.Error("expected mouseEnabled to be false")
	}
}

func TestTerminal_EnableKittyKeyboard(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableKittyKeyboard(5) // Enable protocol flags with bitmask 5.

	output := buf.String()
	if !strings.Contains(output, "\x1b[>5u") {
		t.Errorf("expected kitty enable escape, got: %q", output)
	}

	if !term.kittyKeyboardEnabled {
		t.Error("expected kittyKeyboardEnabled to be true")
	}
	if term.kittyFlags != 5 {
		t.Errorf("expected kittyFlags=5, got %d", term.kittyFlags)
	}
}

func TestTerminal_DisableKittyKeyboard(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableKittyKeyboard(5)
	buf.Reset()

	term.DisableKittyKeyboard()

	output := buf.String()
	if !strings.Contains(output, "\x1b[<u") {
		t.Errorf("expected kitty disable escape, got: %q", output)
	}

	if term.kittyKeyboardEnabled {
		t.Error("expected kittyKeyboardEnabled to be false")
	}
	if term.kittyFlags != 0 {
		t.Errorf("expected kittyFlags=0, got %d", term.kittyFlags)
	}
}

func TestTerminal_SetKittyKeyboardFlags(t *testing.T) {
	term, buf := newTestBackend()
	term.EnableKittyKeyboard(1)
	buf.Reset()

	term.SetKittyKeyboardFlags(7)

	output := buf.String()
	if !strings.Contains(output, "\x1b[>7u") {
		t.Errorf("expected kitty flags escape, got: %q", output)
	}
	if term.kittyFlags != 7 {
		t.Errorf("expected kittyFlags=7, got %d", term.kittyFlags)
	}
}

func TestTerminal_KittyKeyboardFlags(t *testing.T) {
	term, _ := newTestBackend()
	term.EnableKittyKeyboard(3)

	if term.KittyKeyboardFlags() != 3 {
		t.Errorf("expected KittyKeyboardFlags()=3, got %d", term.KittyKeyboardFlags())
	}
}

func TestTerminal_SetCursorPosition(t *testing.T) {
	term, buf := newTestBackend()
	term.SetCursorPosition(5, 10, true)

	output := buf.String()
	// Zero-based (5,10) becomes the one-based cursor position ESC[11;6H.
	if !strings.Contains(output, "\x1b[11;6H") {
		t.Errorf("expected cursor position escape, got: %q", output)
	}
	// The position sequence is followed by cursor visibility.
	if !strings.Contains(output, "\x1b[?25h") {
		t.Error("expected cursor show escape")
	}
}

func TestTerminal_SetCursorPosition_Hidden(t *testing.T) {
	term, buf := newTestBackend()
	term.SetCursorPosition(0, 0, false)

	output := buf.String()
	if !strings.Contains(output, "\x1b[?25l") {
		t.Error("expected cursor hide escape")
	}
}

func TestTerminal_SetCursorStyle(t *testing.T) {
	tests := []struct {
		style    string
		expected string
	}{
		{"block", "\x1b[2 q"},
		{"underline", "\x1b[4 q"},
		{"line", "\x1b[6 q"},
		{"bar", "\x1b[6 q"},
		{"unknown", "\x1b[0 q"},
	}

	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			term, buf := newTestBackend()
			term.SetCursorStyle(tt.style)
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q for style %q, got: %q", tt.expected, tt.style, output)
			}
		})
	}
}

func TestTerminal_ShowHideCursor(t *testing.T) {
	term, buf := newTestBackend()

	term.ShowCursor()
	if !strings.Contains(buf.String(), "\x1b[?25h") {
		t.Error("expected show cursor escape")
	}

	buf.Reset()
	term.HideCursor()
	if !strings.Contains(buf.String(), "\x1b[?25l") {
		t.Error("expected hide cursor escape")
	}
}

func TestTerminal_SetCursorBlinking(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		blinking bool
		expected string
	}{
		{"block steady", "block", false, "\x1b[2 q"},
		{"block blinking", "block", true, "\x1b[1 q"},
		{"underline steady", "underline", false, "\x1b[4 q"},
		{"underline blinking", "underline", true, "\x1b[3 q"},
		{"line steady", "line", false, "\x1b[6 q"},
		{"line blinking", "line", true, "\x1b[5 q"},
		{"bar steady", "bar", false, "\x1b[6 q"},
		{"bar blinking", "bar", true, "\x1b[5 q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term, buf := newTestBackend()
			term.SetCursorStyle(tt.style)
			buf.Reset()

			term.SetCursorBlinking(tt.blinking)
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q, got: %q", tt.expected, output)
			}
		})
	}
}

func TestTerminal_CursorStyleAccessors(t *testing.T) {
	term, _ := newTestBackend()

	term.SetCursorStyle("underline")
	if term.CursorStyle() != "underline" {
		t.Errorf("expected 'underline', got %q", term.CursorStyle())
	}

	term.SetCursorBlinking(true)
	if !term.CursorBlinking() {
		t.Error("expected CursorBlinking()=true")
	}

	term.SetCursorBlinking(false)
	if term.CursorBlinking() {
		t.Error("expected CursorBlinking()=false")
	}
}

func TestTerminal_SetCursorColor(t *testing.T) {
	term, buf := newTestBackend()
	term.SetCursorColor(0xFF, 0x00, 0xAB)

	output := buf.String()
	expected := "\x1b]12;rgb:ff/00/ab\x1b\\"
	if !strings.Contains(output, expected) {
		t.Errorf("expected %q, got: %q", expected, output)
	}
}

func TestTerminal_ResetCursorColor(t *testing.T) {
	term, buf := newTestBackend()
	term.ResetCursorColor()

	output := buf.String()
	expected := "\x1b]112\x1b\\"
	if !strings.Contains(output, expected) {
		t.Errorf("expected %q, got: %q", expected, output)
	}
}

func TestTerminal_SetMousePointer(t *testing.T) {
	tests := []struct {
		style    string
		expected string
	}{
		{"default", "\x1b]22;default\x1b\\"},
		{"pointer", "\x1b]22;pointer\x1b\\"},
		{"text", "\x1b]22;text\x1b\\"},
		{"crosshair", "\x1b]22;crosshair\x1b\\"},
		{"move", "\x1b]22;move\x1b\\"},
		{"not-allowed", "\x1b]22;not-allowed\x1b\\"},
	}

	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			term, buf := newTestBackend()
			term.SetMousePointer(tt.style)
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q, got: %q", tt.expected, output)
			}
		})
	}
}

func TestTerminal_SetTitle(t *testing.T) {
	term, buf := newTestBackend()
	term.SetTitle("My App")

	output := buf.String()
	if !strings.Contains(output, "\x1b]2;My App\x07") {
		t.Errorf("expected title escape, got: %q", output)
	}
}

func TestTerminal_Clear(t *testing.T) {
	term, buf := newTestBackend()
	term.Clear()

	output := buf.String()
	if !strings.Contains(output, "\x1b[2J\x1b[H") {
		t.Errorf("expected clear escape, got: %q", output)
	}
}

func TestTerminal_WriteOut(t *testing.T) {
	term, buf := newTestBackend()
	data := []byte("hello world")
	n, err := term.WriteOut(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}
	if buf.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}
}

func TestTerminal_BracketedPaste(t *testing.T) {
	term, buf := newTestBackend()

	term.EnableBracketedPaste()
	if !strings.Contains(buf.String(), "\x1b[?2004h") {
		t.Error("expected bracketed paste enable escape")
	}

	buf.Reset()
	term.DisableBracketedPaste()
	if !strings.Contains(buf.String(), "\x1b[?2004l") {
		t.Error("expected bracketed paste disable escape")
	}
}

func TestTerminal_AlternateScreen(t *testing.T) {
	term, buf := newTestBackend()

	term.EnterAlternateScreen()
	if !strings.Contains(buf.String(), "\x1b[?1049h") {
		t.Error("expected alternate screen enter escape")
	}
	if !term.alternateScreen {
		t.Error("expected alternateScreen to be true")
	}

	buf.Reset()
	term.LeaveAlternateScreen()
	if !strings.Contains(buf.String(), "\x1b[?1049l") {
		t.Error("expected alternate screen leave escape")
	}
	if term.alternateScreen {
		t.Error("expected alternateScreen to be false")
	}
}
