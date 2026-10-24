package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/textbuf"
)

func setupTextarea(text string) *Textarea {
	ta := NewTextarea(TextareaWithText(text))
	testutil.SetupWidget(ta, 40, 10)
	return ta
}

// Core Textarea behavior.

func TestNewTextarea_Defaults(t *testing.T) {
	ta := NewTextarea()
	if ta.Text() != "" {
		t.Errorf("expected empty text, got %q", ta.Text())
	}
	if ta.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", ta.LineCount())
	}
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestTextarea_SetText(t *testing.T) {
	ta := NewTextarea()
	ta.SetText("line1\nline2\nline3")
	if ta.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", ta.LineCount())
	}
	if ta.Text() != "line1\nline2\nline3" {
		t.Errorf("unexpected text: %q", ta.Text())
	}
}

func TestTextarea_InsertText(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)
	ta.InsertText(",")
	if ta.Text() != "hello, world" {
		t.Errorf("expected 'hello, world', got %q", ta.Text())
	}
}

func TestTextarea_InsertNewline(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)

	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})

	if ta.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", ta.LineCount())
	}
	if ta.Text() != "hello\n world" {
		t.Errorf("expected 'hello\\n world', got %q", ta.Text())
	}
}

func TestTextarea_Backspace(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 5)

	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyBackspace, Type: input.KeyPress})
	if ta.Text() != "hell" {
		t.Errorf("expected 'hell', got %q", ta.Text())
	}
}

func TestTextarea_BackspaceMergeLines(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(1, 0)

	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyBackspace, Type: input.KeyPress})
	if ta.LineCount() != 1 {
		t.Errorf("expected 1 line after merge, got %d", ta.LineCount())
	}
	if ta.Text() != "helloworld" {
		t.Errorf("expected 'helloworld', got %q", ta.Text())
	}
}

func TestTextarea_DeleteForward(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 0)

	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDelete, Type: input.KeyPress})
	if ta.Text() != "ello" {
		t.Errorf("expected 'ello', got %q", ta.Text())
	}
}

func TestTextarea_CursorMovement(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(0, 2)

	// Down moves to the same column on the next line.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 2 {
		t.Errorf("expected cursor at (1,2), got (%d,%d)", row, col)
	}

	// Up returns to the original line and column.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col = ta.CursorPosition()
	if row != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveUpAtFirstRow(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(0, 3) // middle of first line

	// Up at the first row clamps to that line's start.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveDownAtLastRow(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(1, 2) // middle of last line

	// Down at the last row clamps to that line's end.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 5 {
		t.Errorf("expected cursor at (1,5), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveUpAtFirstRow_AlreadyAtStart(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(0, 0)

	// Up at the buffer origin has no effect.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveDownAtLastRow_AlreadyAtEnd(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(1, 5)

	// Down at the buffer end has no effect.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 5 {
		t.Errorf("expected cursor at (1,5), got (%d,%d)", row, col)
	}
}

func TestTextarea_UndoRedo(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 5)

	// Create one undoable edit.
	ta.InsertText(" world")
	if ta.Text() != "hello world" {
		t.Errorf("expected 'hello world', got %q", ta.Text())
	}

	// Undo restores the original content.
	if !ta.Undo() {
		t.Error("expected Undo to return true")
	}
	if ta.Text() != "hello" {
		t.Errorf("expected 'hello' after undo, got %q", ta.Text())
	}

	// Redo reapplies the insertion.
	if !ta.Redo() {
		t.Error("expected Redo to return true")
	}
	if ta.Text() != "hello world" {
		t.Errorf("expected 'hello world' after redo, got %q", ta.Text())
	}
}

func TestTextarea_SelectAll(t *testing.T) {
	ta := setupTextarea("hello\nworld")

	// The Kitty protocol reports Command+A as Meta+A.
	ta.HandleKeyPress(&input.KeyEvent{Name: "a", Mods: input.ModMeta, Type: input.KeyPress})

	selected := ta.SelectedText()
	if selected != "hello\nworld" {
		t.Errorf("expected all text selected, got %q", selected)
	}
}

func TestTextarea_DeleteLine(t *testing.T) {
	ta := setupTextarea("line1\nline2\nline3")
	ta.SetCursor(1, 0)

	ta.HandleKeyPress(&input.KeyEvent{Name: "d", Mods: input.ModCtrl | input.ModShift, Type: input.KeyPress})
	if ta.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", ta.LineCount())
	}
	if ta.Text() != "line1\nline3" {
		t.Errorf("expected 'line1\\nline3', got %q", ta.Text())
	}
}

func TestTextarea_ReadOnly(t *testing.T) {
	ta := NewTextarea(TextareaWithText("readonly"), TextareaWithReadOnly(true))
	testutil.SetupWidget(ta, 20, 5)

	// Read-only mode rejects printable input.
	ta.HandleKeyPress(&input.KeyEvent{Name: "x", Type: input.KeyPress})
	if ta.Text() != "readonly" {
		t.Errorf("expected 'readonly' (read-only), got %q", ta.Text())
	}
}

func TestTextarea_Render(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)

	// The first content cell contains the initial text.
	cell := buf.Cell(0, 0)
	if cell.Char != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", cell.Char)
	}
}

func TestTextarea_HandlePaste(t *testing.T) {
	ta := setupTextarea("")
	ta.HandlePaste(&input.PasteEvent{Text: "pasted\ntext"})
	if ta.Text() != "pasted\ntext" {
		t.Errorf("expected 'pasted\\ntext', got %q", ta.Text())
	}
}

func TestTextarea_ScrollTo(t *testing.T) {
	ta := setupTextarea("line1\nline2\nline3\nline4\nline5")
	ta.ScrollTo(2)
	// Empty content must render safely.
}

// Word wrapping.

func TestTextarea_WrapMode(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello world foo bar"),
		TextareaWithWrapMode(WrapWord),
	)
	testutil.SetupWidget(ta, 10, 5)

	// A ten-column viewport wraps the sentence onto multiple visual lines.
	buf := buffer.New(10, 5)
	ta.RenderSelf(buf, 0)

	// Wrapping preserves the first source character.
	cell := buf.Cell(0, 0)
	if cell.Char != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", cell.Char)
	}
}

func TestTextarea_SetWrapMode(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetWrapMode(WrapChar)

	// Character wrapping must render safely.
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)
}

// Configurable keybindings.

func TestTextarea_CustomKeyBindings(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 5)

	// Replace the default Control+B movement with backspace.
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "b", Mods: input.ModCtrl, Action: ActionBackspace},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "b", Mods: input.ModCtrl, Type: input.KeyPress})
	if ta.Text() != "hell" {
		t.Errorf("expected custom Ctrl+B to backspace, got %q", ta.Text())
	}
}

func TestTextarea_KeyAliases(t *testing.T) {
	ta := setupTextarea("hello")

	// Map the custom "enter" name to the standard return key.
	ta.SetKeyAliases(input.KeyAliasMap{
		"enter": "return",
	})

	// Alias resolution must reach the newline action.
	ta.SetCursor(0, 3)
	ta.HandleKeyPress(&input.KeyEvent{Name: "return", Type: input.KeyPress})
	if ta.LineCount() != 2 {
		t.Errorf("expected 2 lines after enter, got %d", ta.LineCount())
	}
}

// Keyboard selection.

func TestTextarea_ShiftArrowSelection(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 0)

	// Extend the selection across the first five characters.
	for range 5 {
		ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Mods: input.ModShift, Type: input.KeyPress})
	}

	if !ta.HasSelection() {
		t.Error("expected selection to be active")
	}
	selected := ta.SelectedText()
	if selected != "hello" {
		t.Errorf("expected 'hello' selected, got %q", selected)
	}
}

func TestTextarea_ShiftCtrlWordSelection(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 0)

	// Extend to the next word boundary.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Mods: input.ModCtrl | input.ModShift, Type: input.KeyPress})

	if !ta.HasSelection() {
		t.Error("expected selection to be active")
	}
	selected := ta.SelectedText()
	// Word-boundary movement includes the trailing separator space.
	if len(selected) == 0 {
		t.Error("expected non-empty selection")
	}
}

func TestTextarea_SelectionClearedOnNavigation(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SelectAll()

	if !ta.HasSelection() {
		t.Error("expected selection after SelectAll")
	}

	// Movement without Shift clears the selection.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Type: input.KeyPress})

	if ta.HasSelection() {
		t.Error("expected selection to be cleared after navigation")
	}
}

func TestTextarea_DeleteSelection(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 0)

	// Select the first five characters.
	for range 5 {
		ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Mods: input.ModShift, Type: input.KeyPress})
	}

	// Printable input replaces the selected range.
	ta.HandleKeyPress(&input.KeyEvent{Name: "X", Type: input.KeyPress})
	if ta.Text() != "X world" {
		t.Errorf("expected 'X world', got %q", ta.Text())
	}
}

// Word navigation.

func TestTextarea_CtrlLeftRight(t *testing.T) {
	ta := setupTextarea("hello world foo")
	ta.SetCursor(0, 0)

	// Control+Right moves to the next word boundary.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Mods: input.ModCtrl, Type: input.KeyPress})
	_, col := ta.CursorPosition()
	if col == 0 {
		t.Error("expected cursor to move past first word")
	}

	// Control+Left returns to the previous word boundary.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Mods: input.ModCtrl, Type: input.KeyPress})
	_, col = ta.CursorPosition()
	if col != 0 {
		t.Errorf("expected cursor at 0 after Ctrl+Left, got %d", col)
	}
}

// Line and document boundary movement.

func TestTextarea_HomeEnd(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)

	// Logical Home moves to column zero.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyHome, Type: input.KeyPress})
	_, col := ta.CursorPosition()
	if col != 0 {
		t.Errorf("expected col 0 after Home, got %d", col)
	}

	// Logical End moves to the line's final column.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyEnd, Type: input.KeyPress})
	_, col = ta.CursorPosition()
	if col != 11 { // "hello world" = 11 chars
		t.Errorf("expected col 11 after End, got %d", col)
	}
}

func TestTextarea_DocStartEnd(t *testing.T) {
	ta := setupTextarea("line1\nline2\nline3")
	ta.SetCursor(1, 2)

	// Document Home moves to the buffer origin.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyHome, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected (0,0) after Home, got (%d,%d)", row, col)
	}

	// Document End moves to the final line and column.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyEnd, Type: input.KeyPress})
	row, col = ta.CursorPosition()
	if row != 2 {
		t.Errorf("expected row 2 after Ctrl+End, got %d", row)
	}
}

// Page movement.

func TestTextarea_PageUpDown(t *testing.T) {
	// Use more logical lines than fit in the viewport.
	lines := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no"
	ta := NewTextarea(TextareaWithText(lines))
	testutil.SetupWidget(ta, 10, 3)

	ta.SetCursor(0, 0)

	// PageDown advances by the three-row viewport height.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyPageDown, Type: input.KeyPress})
	row, _ := ta.CursorPosition()
	if row != 3 {
		t.Errorf("expected row 3 after PageDown, got %d", row)
	}

	// PageUp returns by the same distance.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyPageUp, Type: input.KeyPress})
	row, _ = ta.CursorPosition()
	if row != 0 {
		t.Errorf("expected row 0 after PageUp, got %d", row)
	}
}

// Submit action.

func TestTextarea_OnSubmit(t *testing.T) {
	var submitted string
	ta := NewTextarea(
		TextareaWithText("hello"),
		TextareaWithOnSubmit(func(text string) { submitted = text }),
	)
	testutil.SetupWidget(ta, 20, 5)

	// Textarea maps Enter to newline by default, so bind submit explicitly.
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "s", Mods: input.ModCtrl, Action: ActionSubmit},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "s", Mods: input.ModCtrl, Type: input.KeyPress})
	if submitted != "hello" {
		t.Errorf("expected submitted 'hello', got %q", submitted)
	}
}

// Content-change callback.

func TestTextarea_OnContentChange(t *testing.T) {
	var lastContent string
	ta := NewTextarea(TextareaWithOnContentChange(func(text string) {
		lastContent = text
	}))
	testutil.SetupWidget(ta, 20, 5)

	ta.HandleKeyPress(&input.KeyEvent{Name: "a", Type: input.KeyPress})
	if lastContent != "a" {
		t.Errorf("expected content 'a', got %q", lastContent)
	}

	ta.HandleKeyPress(&input.KeyEvent{Name: "b", Type: input.KeyPress})
	if lastContent != "ab" {
		t.Errorf("expected content 'ab', got %q", lastContent)
	}
}

// Selection foreground.

func TestTextarea_SelectionFg(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello"),
		TextareaWithSelectionFg(style.ColorBlack),
		TextareaWithSelectionBg(style.ColorWhite),
	)
	testutil.SetupWidget(ta, 20, 5)

	if ta.selectionFg.IsTransparent() {
		t.Error("expected selectionFg to be set")
	}
}

// Styled placeholders.

func TestTextarea_PlaceholderSpans(t *testing.T) {
	ta := NewTextarea()
	ta.SetPlaceholderSpans([]textbuf.Span{
		{Text: "Type ", Fg: style.ColorGray},
		{Text: "here", Fg: style.ColorWhite},
	})
	testutil.SetupWidget(ta, 20, 5)

	// A styled placeholder must render safely.
	buf := buffer.New(20, 5)
	ta.RenderSelf(buf, 0)
}

// EditSurface access.

func TestTextarea_EditBufferView(t *testing.T) {
	ta := setupTextarea("hello")
	ev := ta.EditSurface()
	if ev == nil {
		t.Fatal("expected non-nil EditSurface")
	}
	if ev.Text() != "hello" {
		t.Errorf("expected EditSurface text 'hello', got %q", ev.Text())
	}
}

// Read-only navigation.

func TestTextarea_ReadOnlyAllowsNavigation(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello\nworld"),
		TextareaWithReadOnly(true),
	)
	testutil.SetupWidget(ta, 20, 5)

	ta.SetCursor(0, 0)

	// Read-only mode still permits cursor movement.
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, _ := ta.CursorPosition()
	if row != 1 {
		t.Errorf("expected cursor to move down in read-only, got row %d", row)
	}
}

// Deletion to line boundaries.

func TestTextarea_DeleteToEnd(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)

	// Bind Control+K to deletion through line end.
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "k", Mods: input.ModCtrl, Action: ActionDeleteToEnd},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "k", Mods: input.ModCtrl, Type: input.KeyPress})
	if ta.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", ta.Text())
	}
}

func TestTextarea_DeleteToStart(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 6)

	// Bind Control+U to deletion through line start.
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "u", Mods: input.ModCtrl, Action: ActionDeleteToStart},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "u", Mods: input.ModCtrl, Type: input.KeyPress})
	if ta.Text() != "world" {
		t.Errorf("expected 'world', got %q", ta.Text())
	}
}

// Multiline rendering.

func TestTextarea_RenderMultiLine(t *testing.T) {
	ta := setupTextarea("abc\ndef\nghi")
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)

	// The second logical line starts on the second row.
	cell := buf.Cell(0, 1)
	if cell.Char != 'd' {
		t.Errorf("expected 'd' at (0,1), got %c", cell.Char)
	}

	// The third logical line starts on the third row.
	cell = buf.Cell(0, 2)
	if cell.Char != 'g' {
		t.Errorf("expected 'g' at (0,2), got %c", cell.Char)
	}
}

// Empty undo history.

func TestTextarea_UndoEmpty(t *testing.T) {
	ta := NewTextarea()
	if ta.Undo() {
		t.Error("expected Undo to return false on empty history")
	}
	if ta.Redo() {
		t.Error("expected Redo to return false on empty history")
	}
}
