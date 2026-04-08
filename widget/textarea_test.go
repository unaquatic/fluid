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

// ---------------------------------------------------------------------------
// Original tests (preserved)
// ---------------------------------------------------------------------------

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

	// Move down
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 2 {
		t.Errorf("expected cursor at (1,2), got (%d,%d)", row, col)
	}

	// Move up
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col = ta.CursorPosition()
	if row != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveUpAtFirstRow(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(0, 3) // middle of first line

	// Up at first row should move to start of line
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveDownAtLastRow(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(1, 2) // middle of last line

	// Down at last row should move to end of line
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 5 {
		t.Errorf("expected cursor at (1,5), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveUpAtFirstRow_AlreadyAtStart(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(0, 0)

	// Up at (0,0) should stay at (0,0)
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestTextarea_MoveDownAtLastRow_AlreadyAtEnd(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	ta.SetCursor(1, 5)

	// Down at end of last line should stay put
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 1 || col != 5 {
		t.Errorf("expected cursor at (1,5), got (%d,%d)", row, col)
	}
}

func TestTextarea_UndoRedo(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 5)

	// Insert text
	ta.InsertText(" world")
	if ta.Text() != "hello world" {
		t.Errorf("expected 'hello world', got %q", ta.Text())
	}

	// Undo
	if !ta.Undo() {
		t.Error("expected Undo to return true")
	}
	if ta.Text() != "hello" {
		t.Errorf("expected 'hello' after undo, got %q", ta.Text())
	}

	// Redo
	if !ta.Redo() {
		t.Error("expected Redo to return true")
	}
	if ta.Text() != "hello world" {
		t.Errorf("expected 'hello world' after redo, got %q", ta.Text())
	}
}

func TestTextarea_SelectAll(t *testing.T) {
	ta := setupTextarea("hello\nworld")

	// Meta+A (Cmd+A, Kitty protocol) triggers select-all
	ta.HandleKeyPress(&input.KeyEvent{Name: "a", Meta: true, Type: input.KeyPress})

	selected := ta.SelectedText()
	if selected != "hello\nworld" {
		t.Errorf("expected all text selected, got %q", selected)
	}
}

func TestTextarea_DeleteLine(t *testing.T) {
	ta := setupTextarea("line1\nline2\nline3")
	ta.SetCursor(1, 0)

	ta.HandleKeyPress(&input.KeyEvent{Name: "d", Ctrl: true, Shift: true, Type: input.KeyPress})
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

	// Try to type - should be ignored
	ta.HandleKeyPress(&input.KeyEvent{Name: "x", Type: input.KeyPress})
	if ta.Text() != "readonly" {
		t.Errorf("expected 'readonly' (read-only), got %q", ta.Text())
	}
}

func TestTextarea_Render(t *testing.T) {
	ta := setupTextarea("hello\nworld")
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)

	// Check first character
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
	// Verify no panic
}

// ---------------------------------------------------------------------------
// New tests: Word wrap
// ---------------------------------------------------------------------------

func TestTextarea_WrapMode(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello world foo bar"),
		TextareaWithWrapMode(WrapWord),
	)
	testutil.SetupWidget(ta, 10, 5)

	// With wrap word and width=10, "hello world foo bar" wraps to multiple visual lines
	buf := buffer.New(10, 5)
	ta.RenderSelf(buf, 0)

	// First line should start with 'h'
	cell := buf.Cell(0, 0)
	if cell.Char != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", cell.Char)
	}
}

func TestTextarea_SetWrapMode(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetWrapMode(WrapChar)

	// Should not panic and should accept the mode
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)
}

// ---------------------------------------------------------------------------
// New tests: Configurable keybindings
// ---------------------------------------------------------------------------

func TestTextarea_CustomKeyBindings(t *testing.T) {
	ta := setupTextarea("hello")
	ta.SetCursor(0, 5)

	// Override: map Ctrl+B to backspace
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "b", Ctrl: true, Action: ActionBackspace},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "b", Ctrl: true, Type: input.KeyPress})
	if ta.Text() != "hell" {
		t.Errorf("expected custom Ctrl+B to backspace, got %q", ta.Text())
	}
}

func TestTextarea_KeyAliases(t *testing.T) {
	ta := setupTextarea("hello")

	// Set an alias mapping "enter" -> "return"
	ta.SetKeyAliases(input.KeyAliasMap{
		"enter": "return",
	})

	// This should resolve via the alias
	ta.SetCursor(0, 3)
	ta.HandleKeyPress(&input.KeyEvent{Name: "return", Type: input.KeyPress})
	if ta.LineCount() != 2 {
		t.Errorf("expected 2 lines after enter, got %d", ta.LineCount())
	}
}

// ---------------------------------------------------------------------------
// New tests: Selection
// ---------------------------------------------------------------------------

func TestTextarea_ShiftArrowSelection(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 0)

	// Shift+Right 5 times to select "hello"
	for range 5 {
		ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Shift: true, Type: input.KeyPress})
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

	// Shift+Ctrl+Right to select first word
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Ctrl: true, Shift: true, Type: input.KeyPress})

	if !ta.HasSelection() {
		t.Error("expected selection to be active")
	}
	selected := ta.SelectedText()
	// Should select "hello " (word + trailing space) depending on word boundary impl
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

	// Arrow key should clear selection
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Type: input.KeyPress})

	if ta.HasSelection() {
		t.Error("expected selection to be cleared after navigation")
	}
}

func TestTextarea_DeleteSelection(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 0)

	// Select "hello" with Shift+Right
	for range 5 {
		ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Shift: true, Type: input.KeyPress})
	}

	// Type a character to replace selection
	ta.HandleKeyPress(&input.KeyEvent{Name: "X", Type: input.KeyPress})
	if ta.Text() != "X world" {
		t.Errorf("expected 'X world', got %q", ta.Text())
	}
}

// ---------------------------------------------------------------------------
// New tests: Word navigation
// ---------------------------------------------------------------------------

func TestTextarea_CtrlLeftRight(t *testing.T) {
	ta := setupTextarea("hello world foo")
	ta.SetCursor(0, 0)

	// Ctrl+Right: move to next word boundary
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Ctrl: true, Type: input.KeyPress})
	_, col := ta.CursorPosition()
	if col == 0 {
		t.Error("expected cursor to move past first word")
	}

	// Ctrl+Left: move back
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Ctrl: true, Type: input.KeyPress})
	_, col = ta.CursorPosition()
	if col != 0 {
		t.Errorf("expected cursor at 0 after Ctrl+Left, got %d", col)
	}
}

// ---------------------------------------------------------------------------
// New tests: Home/End/DocStart/DocEnd
// ---------------------------------------------------------------------------

func TestTextarea_HomeEnd(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)

	// Home -> col 0
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyHome, Type: input.KeyPress})
	_, col := ta.CursorPosition()
	if col != 0 {
		t.Errorf("expected col 0 after Home, got %d", col)
	}

	// End -> end of line
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyEnd, Type: input.KeyPress})
	_, col = ta.CursorPosition()
	if col != 11 { // "hello world" = 11 chars
		t.Errorf("expected col 11 after End, got %d", col)
	}
}

func TestTextarea_DocStartEnd(t *testing.T) {
	ta := setupTextarea("line1\nline2\nline3")
	ta.SetCursor(1, 2)

	// Home -> start of document
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyHome, Type: input.KeyPress})
	row, col := ta.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("expected (0,0) after Home, got (%d,%d)", row, col)
	}

	// End -> end of document
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyEnd, Type: input.KeyPress})
	row, col = ta.CursorPosition()
	if row != 2 {
		t.Errorf("expected row 2 after Ctrl+End, got %d", row)
	}
}

// ---------------------------------------------------------------------------
// New tests: PageUp/PageDown
// ---------------------------------------------------------------------------

func TestTextarea_PageUpDown(t *testing.T) {
	// Create textarea with many lines but small viewport
	lines := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no"
	ta := NewTextarea(TextareaWithText(lines))
	testutil.SetupWidget(ta, 10, 3)

	ta.SetCursor(0, 0)

	// PageDown should move by viewport height (3)
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyPageDown, Type: input.KeyPress})
	row, _ := ta.CursorPosition()
	if row != 3 {
		t.Errorf("expected row 3 after PageDown, got %d", row)
	}

	// PageUp should move back
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyPageUp, Type: input.KeyPress})
	row, _ = ta.CursorPosition()
	if row != 0 {
		t.Errorf("expected row 0 after PageUp, got %d", row)
	}
}

// ---------------------------------------------------------------------------
// New tests: Submit action
// ---------------------------------------------------------------------------

func TestTextarea_OnSubmit(t *testing.T) {
	var submitted string
	ta := NewTextarea(
		TextareaWithText("hello"),
		TextareaWithOnSubmit(func(text string) { submitted = text }),
	)
	testutil.SetupWidget(ta, 20, 5)

	// Submit is not a default textarea binding (Enter = newline),
	// but we can add it via custom keybindings
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "s", Ctrl: true, Action: ActionSubmit},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "s", Ctrl: true, Type: input.KeyPress})
	if submitted != "hello" {
		t.Errorf("expected submitted 'hello', got %q", submitted)
	}
}

// ---------------------------------------------------------------------------
// New tests: ContentChange callback
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// New tests: Selection foreground color
// ---------------------------------------------------------------------------

func TestTextarea_SelectionFg(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello"),
		TextareaWithSelectionFg(style.ColorBlack),
		TextareaWithSelectionBg(style.ColorWhite),
	)
	testutil.SetupWidget(ta, 20, 5)

	if ta.selectionFg == nil {
		t.Error("expected selectionFg to be set")
	}
}

// ---------------------------------------------------------------------------
// New tests: Placeholder spans
// ---------------------------------------------------------------------------

func TestTextarea_PlaceholderSpans(t *testing.T) {
	ta := NewTextarea()
	ta.SetPlaceholderSpans([]textbuf.Span{
		{Text: "Type ", Fg: style.ColorGray},
		{Text: "here", Fg: style.ColorWhite},
	})
	testutil.SetupWidget(ta, 20, 5)

	// Should render without panic
	buf := buffer.New(20, 5)
	ta.RenderSelf(buf, 0)
}

// ---------------------------------------------------------------------------
// New tests: EditSurface access
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// New tests: ReadOnly allows navigation
// ---------------------------------------------------------------------------

func TestTextarea_ReadOnlyAllowsNavigation(t *testing.T) {
	ta := NewTextarea(
		TextareaWithText("hello\nworld"),
		TextareaWithReadOnly(true),
	)
	testutil.SetupWidget(ta, 20, 5)

	ta.SetCursor(0, 0)

	// Navigation should work in read-only mode
	ta.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	row, _ := ta.CursorPosition()
	if row != 1 {
		t.Errorf("expected cursor to move down in read-only, got row %d", row)
	}
}

// ---------------------------------------------------------------------------
// New tests: DeleteToEnd / DeleteToStart (Ctrl+K / Ctrl+U)
// ---------------------------------------------------------------------------

func TestTextarea_DeleteToEnd(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 5)

	// Add Ctrl+K binding to textarea
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "k", Ctrl: true, Action: ActionDeleteToEnd},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "k", Ctrl: true, Type: input.KeyPress})
	if ta.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", ta.Text())
	}
}

func TestTextarea_DeleteToStart(t *testing.T) {
	ta := setupTextarea("hello world")
	ta.SetCursor(0, 6)

	// Add Ctrl+U binding to textarea
	ta.SetKeyBindings([]input.KeyBinding{
		{Name: "u", Ctrl: true, Action: ActionDeleteToStart},
	})

	ta.HandleKeyPress(&input.KeyEvent{Name: "u", Ctrl: true, Type: input.KeyPress})
	if ta.Text() != "world" {
		t.Errorf("expected 'world', got %q", ta.Text())
	}
}

// ---------------------------------------------------------------------------
// New tests: Multi-line render
// ---------------------------------------------------------------------------

func TestTextarea_RenderMultiLine(t *testing.T) {
	ta := setupTextarea("abc\ndef\nghi")
	buf := buffer.New(40, 10)
	ta.RenderSelf(buf, 0)

	// Check second line first char
	cell := buf.Cell(0, 1)
	if cell.Char != 'd' {
		t.Errorf("expected 'd' at (0,1), got %c", cell.Char)
	}

	// Check third line first char
	cell = buf.Cell(0, 2)
	if cell.Char != 'g' {
		t.Errorf("expected 'g' at (0,2), got %c", cell.Char)
	}
}

// ---------------------------------------------------------------------------
// New tests: Undo with no history
// ---------------------------------------------------------------------------

func TestTextarea_UndoEmpty(t *testing.T) {
	ta := NewTextarea()
	if ta.Undo() {
		t.Error("expected Undo to return false on empty history")
	}
	if ta.Redo() {
		t.Error("expected Redo to return false on empty history")
	}
}
