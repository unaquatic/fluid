package textbuf

import "testing"

func TestEditBuffer_SetText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")

	if eb.Text() != "hello\nworld" {
		t.Errorf("expected %q, got %q", "hello\nworld", eb.Text())
	}
	if eb.CursorRow() != 0 || eb.CursorCol() != 0 {
		t.Errorf("cursor should be at (0,0), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_InsertChar(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5) // end of "hello"
	eb.InsertChar('!')

	if eb.Text() != "hello!" {
		t.Errorf("expected %q, got %q", "hello!", eb.Text())
	}
	if eb.CursorCol() != 6 {
		t.Errorf("cursor col should be 6, got %d", eb.CursorCol())
	}
}

func TestEditBuffer_InsertText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hd")
	eb.SetCursor(0, 1) // after 'h'
	eb.InsertText("ello worl")

	if eb.Text() != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", eb.Text())
	}
}

func TestEditBuffer_InsertNewline(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5)
	eb.InsertText("\nworld")

	if eb.Text() != "hello\nworld" {
		t.Errorf("expected %q, got %q", "hello\nworld", eb.Text())
	}
	if eb.CursorRow() != 1 || eb.CursorCol() != 5 {
		t.Errorf("cursor should be at (1,5), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_DeleteCharBackward(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5)
	eb.DeleteCharBackward()

	if eb.Text() != "hell" {
		t.Errorf("expected %q, got %q", "hell", eb.Text())
	}
	if eb.CursorCol() != 4 {
		t.Errorf("cursor col should be 4, got %d", eb.CursorCol())
	}
}

func TestEditBuffer_DeleteCharBackward_MergeLines(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")
	eb.SetCursor(1, 0) // start of "world"
	eb.DeleteCharBackward()

	if eb.Text() != "helloworld" {
		t.Errorf("expected %q, got %q", "helloworld", eb.Text())
	}
	if eb.CursorRow() != 0 || eb.CursorCol() != 5 {
		t.Errorf("cursor should be at (0,5), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_DeleteChar(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 0)
	eb.DeleteChar()

	if eb.Text() != "ello" {
		t.Errorf("expected %q, got %q", "ello", eb.Text())
	}
}

func TestEditBuffer_DeleteChar_MergeLines(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")
	eb.SetCursor(0, 5) // end of "hello"
	eb.DeleteChar()

	if eb.Text() != "helloworld" {
		t.Errorf("expected %q, got %q", "helloworld", eb.Text())
	}
}

func TestEditBuffer_NewLine(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("helloworld")
	eb.SetCursor(0, 5)
	eb.NewLine()

	if eb.Text() != "hello\nworld" {
		t.Errorf("expected %q, got %q", "hello\nworld", eb.Text())
	}
	if eb.CursorRow() != 1 || eb.CursorCol() != 0 {
		t.Errorf("cursor should be at (1,0), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_CursorMovement(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abc\ndef\nghi")

	eb.SetCursor(0, 1) // 'b' in "abc"
	eb.MoveCursorRight()
	if eb.CursorCol() != 2 {
		t.Errorf("after right: expected col 2, got %d", eb.CursorCol())
	}

	eb.MoveCursorLeft()
	if eb.CursorCol() != 1 {
		t.Errorf("after left: expected col 1, got %d", eb.CursorCol())
	}

	eb.MoveCursorDown()
	if eb.CursorRow() != 1 || eb.CursorCol() != 1 {
		t.Errorf("after down: expected (1,1), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}

	eb.MoveCursorUp()
	if eb.CursorRow() != 0 || eb.CursorCol() != 1 {
		t.Errorf("after up: expected (0,1), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_CursorMovement_LineWrap(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abc\ndef")

	// Moving right at line end crosses the newline.
	eb.SetCursor(0, 3)
	eb.MoveCursorRight()
	if eb.CursorRow() != 1 || eb.CursorCol() != 0 {
		t.Errorf("expected (1,0), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}

	// Moving left at line start crosses the preceding newline.
	eb.MoveCursorLeft()
	if eb.CursorRow() != 0 || eb.CursorCol() != 3 {
		t.Errorf("expected (0,3), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_Undo(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5)
	eb.InsertChar('!')

	if eb.Text() != "hello!" {
		t.Fatalf("after insert: expected %q, got %q", "hello!", eb.Text())
	}

	ok := eb.Undo()
	if !ok {
		t.Fatal("undo should succeed")
	}
	if eb.Text() != "hello" {
		t.Errorf("after undo: expected %q, got %q", "hello", eb.Text())
	}
}

func TestEditBuffer_Redo(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5)
	eb.InsertChar('!')

	eb.Undo()
	ok := eb.Redo()
	if !ok {
		t.Fatal("redo should succeed")
	}
	if eb.Text() != "hello!" {
		t.Errorf("after redo: expected %q, got %q", "hello!", eb.Text())
	}
}

func TestEditBuffer_DeleteLine(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("aaa\nbbb\nccc")
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	if eb.Text() != "aaa\nccc" {
		t.Errorf("expected %q, got %q", "aaa\nccc", eb.Text())
	}
}

func TestEditBuffer_DeleteRange(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")
	eb.DeleteRange(0, 5, 0, 11) // delete " world"

	if eb.Text() != "hello" {
		t.Errorf("expected %q, got %q", "hello", eb.Text())
	}
}

func TestEditBuffer_WordBoundary(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world foo")
	eb.SetCursor(0, 0)

	next := eb.NextWordBoundary()
	if next.Col != 6 { // past "hello "
		t.Errorf("next word boundary: expected col 6, got %d", next.Col)
	}

	eb.SetCursor(0, 6) // at "world"
	prev := eb.PrevWordBoundary()
	if prev.Col != 0 {
		t.Errorf("prev word boundary: expected col 0, got %d", prev.Col)
	}
}

func TestEditBuffer_GotoLine(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("aaa\nbbb\nccc")
	eb.GotoLine(2)

	if eb.CursorRow() != 2 || eb.CursorCol() != 0 {
		t.Errorf("expected (2,0), got (%d,%d)", eb.CursorRow(), eb.CursorCol())
	}
}

func TestEditBuffer_GetEOL(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")
	eb.SetCursor(0, 3)

	eol := eb.EOL()
	if eol.Col != 11 {
		t.Errorf("EOL: expected col 11, got %d", eol.Col)
	}
}

func TestEditBuffer_ID(t *testing.T) {
	tb1 := NewTextBuffer()
	tb2 := NewTextBuffer()
	eb1 := NewEditBuffer(tb1)
	eb2 := NewEditBuffer(tb2)

	if eb1.ID() == 0 {
		t.Error("ID should not be 0")
	}
	if eb1.ID() == eb2.ID() {
		t.Error("IDs should be unique")
	}
}

func TestEditBuffer_ReplaceText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")
	eb.SetCursor(0, 5)
	eb.InsertChar('!')

	// ReplaceText adds an undoable replacement without clearing prior history.
	eb.ReplaceText("new content")
	if eb.Text() != "new content" {
		t.Errorf("expected %q, got %q", "new content", eb.Text())
	}

	// Undo restores the content immediately before replacement.
	ok := eb.Undo()
	if !ok {
		t.Fatal("undo should succeed after ReplaceText")
	}
	if eb.Text() != "hello!" {
		t.Errorf("after undo: expected %q, got %q", "hello!", eb.Text())
	}
}

func TestEditBuffer_GetTextRangeByCoords(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")

	got := eb.TextRangeByCoords(0, 0, 0, 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestEditBuffer_SetCursorByByteOffset(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")

	// Three ASCII bytes occupy three display columns.
	eb.SetCursorByByteOffset(0, 3)
	if eb.CursorCol() != 3 {
		t.Errorf("expected col=3, got %d", eb.CursorCol())
	}
}

func TestEditBuffer_SetCursorByByteOffset_UTF8(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("日本語") // 3 chars, each 3 bytes, each 2 display cols

	// Three UTF-8 bytes for 日 occupy two display columns.
	eb.SetCursorByByteOffset(0, 3)
	if eb.CursorCol() != 2 {
		t.Errorf("expected col=2 after first CJK char, got %d", eb.CursorCol())
	}

	// Six UTF-8 bytes for 日本 occupy four display columns.
	eb.SetCursorByByteOffset(0, 6)
	if eb.CursorCol() != 4 {
		t.Errorf("expected col=4 after two CJK chars, got %d", eb.CursorCol())
	}
}
