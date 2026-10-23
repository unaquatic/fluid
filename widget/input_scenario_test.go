package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// Input event scenarios.

// pressKey dispatches one key press to inp.
func pressKey(inp *widget.Input, name string) {
	e := &input.KeyEvent{Name: name, Type: input.KeyPress}
	inp.HandleKeyPress(e)
}

// renderInput renders an already laid-out Input into a fresh buffer.
func renderInput(inp *widget.Input, width, height int) *buffer.Buffer {
	buf := buffer.New(width, height)
	inp.RenderSelf(buf, 0)
	return buf
}

// TestInput_TypeAndRender verifies typing characters one at a time updates
// both the value and the rendered buffer.
func TestInput_TypeAndRender(t *testing.T) {
	inp := widget.NewInput()
	testutil.SetupWidget(inp, 20, 1)

	pressKey(inp, "H")
	pressKey(inp, "i")

	if inp.Value() != "Hi" {
		t.Fatalf("expected 'Hi', got %q", inp.Value())
	}

	buf := renderInput(inp, 20, 1)
	testutil.AssertCell(t, buf, 0, 0, 'H')
	testutil.AssertCell(t, buf, 1, 0, 'i')
}

// TestInput_TypeAndBackspace verifies backspace removes the last character.
func TestInput_TypeAndBackspace(t *testing.T) {
	inp := widget.NewInput(widget.InputWithValue("Hello"))
	testutil.SetupWidget(inp, 20, 1)

	pressKey(inp, input.KeyBackspace)
	pressKey(inp, input.KeyBackspace)

	if inp.Value() != "Hel" {
		t.Fatalf("expected 'Hel', got %q", inp.Value())
	}

	buf := renderInput(inp, 20, 1)
	testutil.AssertRow(t, buf, 0, "Hel")
}

// TestInput_CursorMoveThenInsert verifies cursor movement and mid-string insertion.
func TestInput_CursorMoveThenInsert(t *testing.T) {
	inp := widget.NewInput(widget.InputWithValue("Hllo"))
	testutil.SetupWidget(inp, 20, 1)

	// Insert between the first and second characters.
	inp.SetCursorPos(1)
	pressKey(inp, "e")

	if inp.Value() != "Hello" {
		t.Fatalf("expected 'Hello', got %q", inp.Value())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

// TestInput_SequentialEdits verifies a mixed sequence of insertion, movement, and deletion.
func TestInput_SequentialEdits(t *testing.T) {
	inp := widget.NewInput()
	testutil.SetupWidget(inp, 20, 1)

	// Build the initial value one event at a time.
	pressKey(inp, "a")
	pressKey(inp, "b")
	pressKey(inp, "c")
	if inp.Value() != "abc" {
		t.Fatalf("after typing: expected 'abc', got %q", inp.Value())
	}

	// Remove the final character.
	pressKey(inp, input.KeyBackspace)
	if inp.Value() != "ab" {
		t.Fatalf("after backspace: expected 'ab', got %q", inp.Value())
	}

	// Replace it with an uppercase character.
	pressKey(inp, "C")
	if inp.Value() != "abC" {
		t.Fatalf("after re-type: expected 'abC', got %q", inp.Value())
	}

	buf := renderInput(inp, 20, 1)
	testutil.AssertCell(t, buf, 0, 0, 'a')
	testutil.AssertCell(t, buf, 1, 0, 'b')
	testutil.AssertCell(t, buf, 2, 0, 'C')
}

// TestInput_OnSubmitCallback verifies that Enter invokes OnSubmit.
func TestInput_OnSubmitCallback(t *testing.T) {
	var submitted string
	inp := widget.NewInput(
		widget.InputWithValue("test"),
		widget.InputWithOnSubmit(func(v string) { submitted = v }),
	)
	testutil.SetupWidget(inp, 20, 1)

	pressKey(inp, input.KeyReturn)

	if submitted != "test" {
		t.Errorf("expected 'test', got %q", submitted)
	}
}

// TestInput_OnInputCallback verifies that each inserted rune invokes OnInput.
func TestInput_OnInputCallback(t *testing.T) {
	var inputValues []string
	inp := widget.NewInput(
		widget.InputWithOnInput(func(v string) { inputValues = append(inputValues, v) }),
	)
	testutil.SetupWidget(inp, 20, 1)

	pressKey(inp, "a")
	pressKey(inp, "b")

	if len(inputValues) < 2 {
		t.Fatalf("expected at least 2 OnInput calls, got %d", len(inputValues))
	}
	if inputValues[0] != "a" {
		t.Errorf("first OnInput: expected 'a', got %q", inputValues[0])
	}
	if inputValues[1] != "ab" {
		t.Errorf("second OnInput: expected 'ab', got %q", inputValues[1])
	}
}

// TestInput_MaxLength verifies the max length constraint.
func TestInput_MaxLength(t *testing.T) {
	inp := widget.NewInput(widget.InputWithMaxLength(3))
	testutil.SetupWidget(inp, 20, 1)

	pressKey(inp, "a")
	pressKey(inp, "b")
	pressKey(inp, "c")
	pressKey(inp, "d") // Exceeds the configured maximum length.

	if inp.Value() != "abc" {
		t.Errorf("expected 'abc', got %q", inp.Value())
	}
}

// TestInput_Paste verifies paste event handling.
func TestInput_Paste(t *testing.T) {
	inp := widget.NewInput()
	testutil.SetupWidget(inp, 20, 1)

	// Paste should use the same content-change path as typing.
	pe := &input.PasteEvent{Text: "pasted"}
	inp.HandlePaste(pe)

	if inp.Value() != "pasted" {
		t.Errorf("expected 'pasted', got %q", inp.Value())
	}

	buf := renderInput(inp, 20, 1)
	testutil.AssertRow(t, buf, 0, "pasted")
}

// TestInput_DeleteToEnd verifies Ctrl+K (delete to end of line).
func TestInput_DeleteToEnd(t *testing.T) {
	inp := widget.NewInput(widget.InputWithValue("Hello World"))
	testutil.SetupWidget(inp, 20, 1)

	inp.SetCursorPos(5) // after "Hello"
	e := &input.KeyEvent{Name: "k", Ctrl: true, Type: input.KeyPress}
	inp.HandleKeyPress(e)

	if inp.Value() != "Hello" {
		t.Errorf("expected 'Hello', got %q", inp.Value())
	}
}

// TestInput_RenderAfterMultipleOps verifies the buffer is correct after
// a complex sequence of operations.
func TestInput_RenderAfterMultipleOps(t *testing.T) {
	inp := widget.NewInput()
	testutil.SetupWidget(inp, 20, 1)

	// Build a five-character value.
	for _, ch := range "abcde" {
		pressKey(inp, string(ch))
	}

	// Remove its final two characters.
	pressKey(inp, input.KeyBackspace)
	pressKey(inp, input.KeyBackspace)

	// Append replacement characters.
	pressKey(inp, "X")
	pressKey(inp, "Y")

	expected := "abcXY"
	if inp.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, inp.Value())
	}

	buf := renderInput(inp, 20, 1)
	testutil.AssertRow(t, buf, 0, expected)
}
