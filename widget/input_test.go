package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

func setupInput(value string) *Input {
	inp := NewInput(InputWithValue(value))
	testutil.SetupWidget(inp, 20, 1)
	return inp
}

func TestNewInput_Defaults(t *testing.T) {
	inp := NewInput()
	if inp.Value() != "" {
		t.Errorf("expected empty value, got %q", inp.Value())
	}
	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
	}
	if !inp.Focusable() {
		t.Error("expected focusable true")
	}
}

func TestInput_WithValue(t *testing.T) {
	inp := NewInput(InputWithValue("hello"))
	if inp.Value() != "hello" {
		t.Errorf("expected 'hello', got %q", inp.Value())
	}
	if inp.CursorPos() != 5 {
		t.Errorf("expected cursor at 5, got %d", inp.CursorPos())
	}
}

func TestInput_SetValue(t *testing.T) {
	inp := NewInput()
	inp.SetValue("test")
	if inp.Value() != "test" {
		t.Errorf("expected 'test', got %q", inp.Value())
	}
}

func TestInput_InsertChar(t *testing.T) {
	inp := setupInput("hllo")
	inp.SetCursorPos(1)

	// Insert 'e' at position 1
	e := &input.KeyEvent{Name: "e", Type: input.KeyPress}
	inp.HandleKeyPress(e)

	if inp.Value() != "hello" {
		t.Errorf("expected 'hello', got %q", inp.Value())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestInput_Backspace(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(5)

	e := &input.KeyEvent{Name: input.KeyBackspace, Type: input.KeyPress}
	inp.HandleKeyPress(e)

	if inp.Value() != "hell" {
		t.Errorf("expected 'hell', got %q", inp.Value())
	}
}

func TestInput_Delete(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(0)

	e := &input.KeyEvent{Name: input.KeyDelete, Type: input.KeyPress}
	inp.HandleKeyPress(e)

	if inp.Value() != "ello" {
		t.Errorf("expected 'ello', got %q", inp.Value())
	}
}

func TestInput_CursorMovement(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(3)

	// Move left
	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Type: input.KeyPress})
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}

	// Move right
	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Type: input.KeyPress})
	if inp.CursorPos() != 3 {
		t.Errorf("expected cursor at 3, got %d", inp.CursorPos())
	}

	// Line home (Ctrl+A)
	inp.HandleKeyPress(&input.KeyEvent{Name: "a", Ctrl: true, Type: input.KeyPress})
	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
	}

	// Line end (Ctrl+E)
	inp.HandleKeyPress(&input.KeyEvent{Name: "e", Ctrl: true, Type: input.KeyPress})
	if inp.CursorPos() != 5 {
		t.Errorf("expected cursor at 5, got %d", inp.CursorPos())
	}
}

func TestInput_MaxLength(t *testing.T) {
	inp := NewInput(InputWithMaxLength(5))
	testutil.SetupWidget(inp, 20, 1)

	// Type 7 characters
	for _, ch := range "abcdefg" {
		inp.HandleKeyPress(&input.KeyEvent{Name: string(ch), Type: input.KeyPress})
	}

	if inp.Value() != "abcde" {
		t.Errorf("expected 'abcde' (max 5), got %q", inp.Value())
	}
}

func TestInput_OnSubmit(t *testing.T) {
	var submitted string
	inp := NewInput(InputWithOnSubmit(func(v string) { submitted = v }))
	inp.SetValue("test")

	testutil.SetupWidget(inp, 20, 1)

	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})

	if submitted != "test" {
		t.Errorf("expected submitted 'test', got %q", submitted)
	}
}

func TestInput_DeleteToEnd(t *testing.T) {
	inp := setupInput("hello world")
	inp.SetCursorPos(5)

	inp.HandleKeyPress(&input.KeyEvent{Name: "k", Ctrl: true, Type: input.KeyPress})
	if inp.Value() != "hello" {
		t.Errorf("expected 'hello', got %q", inp.Value())
	}
}

func TestInput_DeleteToStart(t *testing.T) {
	inp := setupInput("hello world")
	inp.SetCursorPos(6)

	inp.HandleKeyPress(&input.KeyEvent{Name: "u", Ctrl: true, Type: input.KeyPress})
	if inp.Value() != "world" {
		t.Errorf("expected 'world', got %q", inp.Value())
	}
}

func TestInput_RenderPlaceholder(t *testing.T) {
	inp := NewInput(InputWithPlaceholder("Type here..."))
	testutil.SetupWidget(inp, 20, 1)

	buf := buffer.New(20, 1)
	inp.RenderSelf(buf, 0)

	text := buf.PlainText()
	if len(text) == 0 {
		t.Error("expected placeholder to be rendered")
	}
}

func TestInput_HandlePaste(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(5)

	inp.HandlePaste(&input.PasteEvent{Text: " world\nnewline"})
	// Newlines should be stripped
	if inp.Value() != "hello worldnewline" {
		t.Errorf("expected 'hello worldnewline', got %q", inp.Value())
	}
}

func TestInput_Options(t *testing.T) {
	inp := NewInput(
		InputWithID("my-input"),
		InputWithPlaceholder("type here"),
		InputWithTextColor(style.ColorRed),
		InputWithBackgroundColor(style.ColorBlue),
		InputWithCursorColor(style.ColorGreen),
		InputWithWidth(layout.Points(30)),
	)

	if inp.ID() != "my-input" {
		t.Errorf("expected id 'my-input', got %q", inp.ID())
	}
}

// ---------------------------------------------------------------------------
// New tests: Input embeds Textarea
// ---------------------------------------------------------------------------

func TestInput_EnterDoesNotInsertNewline(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(3)

	// Enter should NOT insert a newline in Input
	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})
	if inp.LineCount() != 1 {
		t.Errorf("expected 1 line (no newline in input), got %d", inp.LineCount())
	}
}

func TestInput_UpDownIgnored(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(3)

	// Up/Down should be ignored in single-line input
	handled := inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	if handled {
		t.Error("expected Up to be unhandled in Input")
	}
	handled = inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	if handled {
		t.Error("expected Down to be unhandled in Input")
	}
}

func TestInput_TabIgnored(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(3)

	// Tab should be ignored in Input
	handled := inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyTab, Type: input.KeyPress})
	if handled {
		t.Error("expected Tab to be unhandled in Input")
	}
}

func TestInput_SelectAll(t *testing.T) {
	inp := setupInput("hello world")

	// Meta+A (Cmd+A, Kitty protocol) triggers select-all
	inp.HandleKeyPress(&input.KeyEvent{Name: "a", Meta: true, Type: input.KeyPress})

	selected := inp.SelectedText()
	if selected != "hello world" {
		t.Errorf("expected 'hello world' selected, got %q", selected)
	}
}

func TestInput_MaxLengthOnPaste(t *testing.T) {
	inp := NewInput(InputWithValue("ab"), InputWithMaxLength(5))
	testutil.SetupWidget(inp, 20, 1)

	inp.SetCursorPos(2)
	inp.HandlePaste(&input.PasteEvent{Text: "cdefgh"})

	// Only "cde" should be inserted (3 remaining slots)
	if inp.Value() != "abcde" {
		t.Errorf("expected 'abcde' (max 5), got %q", inp.Value())
	}
}

func TestInput_SetValueStripsNewlines(t *testing.T) {
	inp := NewInput()
	inp.SetValue("hello\nworld")
	if inp.Value() != "helloworld" {
		t.Errorf("expected newlines stripped, got %q", inp.Value())
	}
}

func TestInput_WordNavigation(t *testing.T) {
	inp := setupInput("hello world foo")
	inp.SetCursorPos(0)

	// Ctrl+Right to move to next word
	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Ctrl: true, Type: input.KeyPress})
	pos := inp.CursorPos()
	if pos == 0 {
		t.Error("expected cursor to advance past first word")
	}

	// Ctrl+Left to move back
	inp.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Ctrl: true, Type: input.KeyPress})
	pos = inp.CursorPos()
	if pos != 0 {
		t.Errorf("expected cursor at 0, got %d", pos)
	}
}

func TestInput_OnInputCallback(t *testing.T) {
	var lastInput string
	inp := NewInput(InputWithOnInput(func(v string) { lastInput = v }))
	testutil.SetupWidget(inp, 20, 1)

	inp.HandleKeyPress(&input.KeyEvent{Name: "x", Type: input.KeyPress})
	if lastInput != "x" {
		t.Errorf("expected OnInput with 'x', got %q", lastInput)
	}
}

func TestInput_OnChangeCallback(t *testing.T) {
	var lastChange string
	inp := NewInput(InputWithOnChange(func(v string) { lastChange = v }))
	testutil.SetupWidget(inp, 20, 1)

	inp.HandleKeyPress(&input.KeyEvent{Name: "y", Type: input.KeyPress})
	if lastChange != "y" {
		t.Errorf("expected OnChange with 'y', got %q", lastChange)
	}
}

func TestInput_CustomKeyBindings(t *testing.T) {
	inp := setupInput("hello")
	inp.SetCursorPos(5)

	// Map Ctrl+B to backspace
	inp.Textarea.SetKeyBindings([]input.KeyBinding{
		{Name: "b", Ctrl: true, Action: ActionBackspace},
	})

	inp.HandleKeyPress(&input.KeyEvent{Name: "b", Ctrl: true, Type: input.KeyPress})
	if inp.Value() != "hell" {
		t.Errorf("expected 'hell' after custom Ctrl+B, got %q", inp.Value())
	}
}

func TestInput_LineCount(t *testing.T) {
	inp := setupInput("hello")
	if inp.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", inp.LineCount())
	}
}
