package textbuf

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestEditBufferView_BasicSetup(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld\nfoo")

	ev := NewEditSurface(eb, 80, 24)

	vlines := ev.VirtualLines()
	if len(vlines) != 3 {
		t.Fatalf("expected 3 virtual lines, got %d", len(vlines))
	}
}

func TestEditBufferView_Viewport(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	// Create 20 lines
	var text strings.Builder
	for i := range 20 {
		if i > 0 {
			text.WriteString("\n")
		}
		text.WriteString("line")
	}
	eb.SetText(text.String())

	ev := NewEditSurface(eb, 80, 5)

	vlines := ev.VirtualLines()
	if len(vlines) != 5 {
		t.Fatalf("expected 5 visible lines (viewport height), got %d", len(vlines))
	}
}

func TestEditBufferView_CursorScroll(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	var text strings.Builder
	for i := range 20 {
		if i > 0 {
			text.WriteString("\n")
		}
		text.WriteString("line")
	}
	eb.SetText(text.String())

	ev := NewEditSurface(eb, 80, 5)

	// Move cursor to line 19 -- should scroll viewport
	eb.SetCursor(19, 0)
	vp := ev.Viewport()
	if vp == nil {
		t.Fatal("viewport should not be nil")
	}
	if vp.Y == 0 {
		t.Error("viewport should have scrolled when cursor moved to end")
	}

	// The cursor line should be visible
	vlines := ev.VirtualLines()
	found := false
	for _, vl := range vlines {
		if vl.SourceLine == 19 {
			found = true
			break
		}
	}
	if !found {
		t.Error("cursor line 19 should be visible in viewport")
	}
}

func TestEditBufferView_VisualCursor(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")
	eb.SetCursor(1, 3)

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.VisualCursor()
	if vc.LogicalRow != 1 || vc.LogicalCol != 3 {
		t.Errorf("logical cursor: expected (1,3), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}
	if vc.VisualRow != 1 {
		t.Errorf("visual row: expected 1, got %d", vc.VisualRow)
	}
	if vc.VisualCol != 3 {
		t.Errorf("visual col: expected 3, got %d", vc.VisualCol)
	}
}

func TestEditBufferView_DrawableView(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("test")

	ev := NewEditSurface(eb, 80, 24)

	// Test that EditSurface satisfies Surface
	var dv Surface = ev
	vlines := dv.VirtualLines()
	if len(vlines) != 1 {
		t.Fatalf("expected 1 virtual line, got %d", len(vlines))
	}
}

func TestEditBufferView_Empty(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	if !ev.IsEmpty() {
		t.Error("should be empty initially")
	}

	eb.InsertChar('x')
	if ev.IsEmpty() {
		t.Error("should not be empty after insert")
	}
}

func TestEditBufferView_WrapMode(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abcdefghij") // 10 chars

	ev := NewEditSurface(eb, 4, 24)
	ev.SetWrapMode(WrapChar)

	vlines := ev.VirtualLines()
	// 10 chars at width 4: "abcd", "efgh", "ij" = 3 virtual lines
	if len(vlines) != 3 {
		t.Fatalf("expected 3 virtual lines with wrap, got %d", len(vlines))
	}
}

// ---------------------------------------------------------------------------
// Tests for new EditSurface methods
// ---------------------------------------------------------------------------

func TestEditBufferView_SetCursorByOffset(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")

	ev := NewEditSurface(eb, 80, 24)

	// Offset 0 = (0,0)
	ev.SetCursorByOffset(0)
	r, c := ev.Cursor()
	if r != 0 || c != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", r, c)
	}

	// Offset 3 = (0,3) - "hel|lo"
	ev.SetCursorByOffset(3)
	r, c = ev.Cursor()
	if r != 0 || c != 3 {
		t.Errorf("expected (0,3), got (%d,%d)", r, c)
	}

	// Offset 6 = (1,0) - after "hello\n"
	ev.SetCursorByOffset(6)
	r, c = ev.Cursor()
	if r != 1 || c != 0 {
		t.Errorf("expected (1,0), got (%d,%d)", r, c)
	}

	// Offset 9 = (1,3) - "wor|ld"
	ev.SetCursorByOffset(9)
	r, c = ev.Cursor()
	if r != 1 || c != 3 {
		t.Errorf("expected (1,3), got (%d,%d)", r, c)
	}
}

func TestEditBufferView_GetNextWordBoundary(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world foo")
	eb.SetCursor(0, 0)

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.NextWordBoundary()
	// From start of "hello", should jump to start of "world"
	if vc.LogicalRow != 0 || vc.LogicalCol != 6 {
		t.Errorf("expected word boundary at (0,6), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}
	// Should also have valid visual coords
	if vc.VisualRow != 0 {
		t.Errorf("expected visual row 0, got %d", vc.VisualRow)
	}
}

func TestEditBufferView_GetPrevWordBoundary(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world foo")
	eb.SetCursor(0, 8) // middle of "world"

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.PrevWordBoundary()
	// From middle of "world", should jump to start of "world"
	if vc.LogicalRow != 0 || vc.LogicalCol != 6 {
		t.Errorf("expected word boundary at (0,6), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}
}

func TestEditBufferView_GetPrevWordBoundary_LineStart(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")
	eb.SetCursor(1, 0) // start of "world"

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.PrevWordBoundary()
	// From start of line 1, should go to end of line 0
	if vc.LogicalRow != 0 {
		t.Errorf("expected row 0, got %d", vc.LogicalRow)
	}
	if vc.LogicalCol != 5 {
		t.Errorf("expected col 5 (end of 'hello'), got %d", vc.LogicalCol)
	}
}

func TestEditBufferView_GetEOL(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld!")
	eb.SetCursor(0, 2) // middle of "hello"

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.EOL()
	if vc.LogicalRow != 0 || vc.LogicalCol != 5 {
		t.Errorf("expected EOL at (0,5), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}

	eb.SetCursor(1, 0)
	vc = ev.EOL()
	if vc.LogicalRow != 1 || vc.LogicalCol != 6 {
		t.Errorf("expected EOL at (1,6), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}
}

func TestEditBufferView_GetVisualSOL(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abcdefghij") // 10 chars

	ev := NewEditSurface(eb, 4, 24)
	ev.SetWrapMode(WrapChar)
	// Virtual lines: "abcd", "efgh", "ij"

	// Cursor at col 6 (in "efgh" wrapped line, col offset 4)
	eb.SetCursor(0, 6)

	vc := ev.VisualSOL()
	// Visual SOL of second wrapped line should be col 4
	if vc.LogicalCol != 4 {
		t.Errorf("expected visual SOL col 4, got %d", vc.LogicalCol)
	}
	if vc.LogicalRow != 0 {
		t.Errorf("expected row 0, got %d", vc.LogicalRow)
	}
}

func TestEditBufferView_GetVisualEOL(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abcdefghij") // 10 chars

	ev := NewEditSurface(eb, 4, 24)
	ev.SetWrapMode(WrapChar)
	// Virtual lines: "abcd", "efgh", "ij"

	// Cursor at col 5 (in "efgh" wrapped line)
	eb.SetCursor(0, 5)

	vc := ev.VisualEOL()
	// Visual EOL of second wrapped line should be col 7 (width-1 to stay
	// on the current visual line).
	if vc.LogicalCol != 7 {
		t.Errorf("expected visual EOL col 7, got %d", vc.LogicalCol)
	}
	// Must stay on the same visual row (row 1)
	if vc.VisualRow != 1 {
		t.Errorf("expected visual row 1, got %d", vc.VisualRow)
	}

	// Cursor on last virtual line ("ij") should return actual end of line
	eb.SetCursor(0, 8)
	vc = ev.VisualEOL()
	if vc.LogicalCol != 10 {
		t.Errorf("expected visual EOL col 10 for last vline, got %d", vc.LogicalCol)
	}

	// Cursor on first virtual line ("abcd") — also a continuation
	eb.SetCursor(0, 2)
	vc = ev.VisualEOL()
	if vc.LogicalCol != 3 {
		t.Errorf("expected visual EOL col 3 for first wrapped vline, got %d", vc.LogicalCol)
	}
	if vc.VisualRow != 0 {
		t.Errorf("expected visual row 0, got %d", vc.VisualRow)
	}
}

func TestEditBufferView_GetVisualSOL_NoWrap(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")
	eb.SetCursor(1, 3)

	ev := NewEditSurface(eb, 80, 24)

	vc := ev.VisualSOL()
	if vc.LogicalRow != 1 || vc.LogicalCol != 0 {
		t.Errorf("expected visual SOL at (1,0), got (%d,%d)", vc.LogicalRow, vc.LogicalCol)
	}
}

func TestEditBufferView_GetText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nworld")

	ev := NewEditSurface(eb, 80, 24)

	got := ev.Text()
	if got != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", got)
	}
}

func TestEditBufferView_DeleteSelectedText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")

	ev := NewEditSurface(eb, 80, 24)

	// Select "lo wo" (offsets 3-8)
	bg := style.ColorFromInts(0, 0, 255, 255)
	ev.SetSelection(&TextSelection{Start: 3, End: 8, BgColor: &bg})

	ev.DeleteSelectedText()

	got := ev.Text()
	if got != "helrld" {
		t.Errorf("expected 'helrld', got %q", got)
	}

	// Selection should be cleared
	if ev.Selection() != nil {
		t.Error("expected selection to be cleared after delete")
	}
}

func TestEditBufferView_DeleteSelectedText_NoSelection(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")

	ev := NewEditSurface(eb, 80, 24)

	// Should be a no-op
	ev.DeleteSelectedText()
	if ev.Text() != "hello" {
		t.Errorf("expected text unchanged, got %q", ev.Text())
	}
}

func TestEditBufferView_SetLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")

	ev := NewEditSurface(eb, 80, 24)

	bg := style.ColorFromInts(0, 0, 255, 255)

	// Select with updateCursor=true, followCursor=false
	ok := ev.SetLocalSelection(2, 0, 7, 0, &bg, nil, true, false)
	if !ok {
		t.Fatal("expected SetLocalSelection to return true")
	}

	sel := ev.Selection()
	if sel == nil {
		t.Fatal("expected selection to be set")
	}

	// Cursor should have moved to focus position (7,0)
	r, c := ev.Cursor()
	if r != 0 || c != 7 {
		t.Errorf("expected cursor at (0,7), got (%d,%d)", r, c)
	}
}

func TestEditBufferView_SetLocalSelection_NoCursorUpdate(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")
	eb.SetCursor(0, 0) // at start

	ev := NewEditSurface(eb, 80, 24)

	// Select without moving cursor
	ok := ev.SetLocalSelection(2, 0, 7, 0, nil, nil, false, false)
	if !ok {
		t.Fatal("expected SetLocalSelection to return true")
	}

	// Cursor should NOT have moved
	r, c := ev.Cursor()
	if r != 0 || c != 0 {
		t.Errorf("expected cursor to stay at (0,0), got (%d,%d)", r, c)
	}
}

func TestEditBufferView_UpdateSelection(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")

	ev := NewEditSurface(eb, 80, 24)

	bg := style.ColorFromInts(0, 0, 255, 255)
	ev.UpdateSelection(5, &bg, nil)

	sel := ev.Selection()
	if sel == nil {
		t.Fatal("expected selection")
	}
	if sel.End != 5 {
		t.Errorf("expected end=5, got %d", sel.End)
	}
}

func TestEditBufferView_UpdateLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")

	ev := NewEditSurface(eb, 80, 24)

	ok := ev.UpdateLocalSelection(1, 0, 4, 0, nil, nil)
	if !ok {
		t.Fatal("expected UpdateLocalSelection to return true")
	}
	if ev.Selection() == nil {
		t.Error("expected selection to be set")
	}
}

func TestEditBufferView_ResetLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello")

	ev := NewEditSurface(eb, 80, 24)
	ev.SetLocalSelection(0, 0, 3, 0, nil, nil, false, false)

	ev.ResetLocalSelection()
	if ev.Selection() != nil {
		t.Error("expected selection to be nil after ResetLocalSelection")
	}
}

func TestEditBufferView_GetSelectedText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello world")

	ev := NewEditSurface(eb, 80, 24)

	// No selection
	if ev.SelectedText() != "" {
		t.Error("expected empty string with no selection")
	}

	ev.SetSelection(&TextSelection{Start: 6, End: 11})
	got := ev.SelectedText()
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestEditBufferView_GetLogicalLineInfo(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("hello\nab\nworld!")

	ev := NewEditSurface(eb, 80, 24)
	ev.SetWrapMode(WrapChar)
	ev.SetViewportSize(3, 20)

	info := ev.LogicalLineInfo()
	// Should still be 3 logical lines regardless of wrapping
	if len(info.Widths) != 3 {
		t.Fatalf("expected 3 logical lines, got %d", len(info.Widths))
	}
	if info.Widths[0] != 5 {
		t.Errorf("line 0 width: expected 5, got %d", info.Widths[0])
	}
	if info.Widths[1] != 2 {
		t.Errorf("line 1 width: expected 2, got %d", info.Widths[1])
	}
	if info.Widths[2] != 6 {
		t.Errorf("line 2 width: expected 6, got %d", info.Widths[2])
	}
}

func TestEditBufferView_SetPlaceholderSpans(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	red := style.ColorFromInts(255, 0, 0, 255)
	blue := style.ColorFromInts(0, 0, 255, 255)
	spans := []Span{
		{Text: "Type ", Fg: red},
		{Text: "here", Fg: blue},
	}
	ev.SetPlaceholderSpans(spans)

	got := ev.PlaceholderSpans()
	if len(got) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(got))
	}
	if got[0].Text != "Type " || got[0].Fg != red {
		t.Errorf("span 0 mismatch: %+v", got[0])
	}
	if got[1].Text != "here" || got[1].Fg != blue {
		t.Errorf("span 1 mismatch: %+v", got[1])
	}

	// Setting span placeholder should clear plain text
	plainText, _ := ev.PlaceholderText()
	if plainText != "" {
		t.Errorf("expected plain placeholder to be cleared, got %q", plainText)
	}
}

func TestEditBufferView_PlaceholderText(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	gray := style.ColorFromInts(128, 128, 128, 255)
	ev.SetPlaceholderText("Enter text...", gray)

	text, fg := ev.PlaceholderText()
	if text != "Enter text..." {
		t.Errorf("expected 'Enter text...', got %q", text)
	}
	if fg != gray {
		t.Errorf("expected gray color, got %+v", fg)
	}

	// Setting plain text should clear span placeholder
	if ev.PlaceholderSpans() != nil {
		t.Error("expected placeholder spans to be nil after setting plain text")
	}
}

func TestEditBufferView_SetTabIndicator(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	ev.SetTabIndicator('→')
	if ev.tabIndicator != '→' {
		t.Errorf("expected tab indicator '→', got %c", ev.tabIndicator)
	}
	// Should also set on the underlying surface
	if ev.surface.tabIndicator != '→' {
		t.Error("expected underlying surface tab indicator to be set")
	}
}

func TestEditBufferView_SetTabIndicatorColor(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	c := style.ColorFromInts(77, 77, 77, 255)
	ev.SetTabIndicatorColor(c)
	if ev.tabIndicatorColor == nil || *ev.tabIndicatorColor != c {
		t.Errorf("expected tab indicator color %v", c)
	}
	// Should also set on the underlying surface
	if ev.surface.tabIndicatorColor == nil || *ev.surface.tabIndicatorColor != c {
		t.Error("expected underlying surface tab indicator color to be set")
	}
}

func TestEditBufferView_SetViewport_MoveCursor(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	var text strings.Builder
	for i := range 20 {
		if i > 0 {
			text.WriteString("\n")
		}
		text.WriteString("line")
	}
	eb.SetText(text.String())

	ev := NewEditSurface(eb, 80, 5)

	// Set cursor to end
	eb.SetCursor(19, 0)

	// SetViewport with moveCursor=true
	ev.SetViewport(&Viewport{X: 0, Y: 0, Width: 80, Height: 5}, true)
	// After ensureCursorVisible, viewport should have scrolled
	vp := ev.Viewport()
	if vp.Y == 0 {
		// Cursor is at line 19, viewport height 5 — should scroll
		t.Error("expected viewport to scroll to make cursor visible")
	}
}

func TestEditBufferView_GetVirtualLineCount(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abcdef\nxy")

	ev := NewEditSurface(eb, 3, 20)
	ev.SetWrapMode(WrapChar)

	// "abcdef" → 2 vlines, "xy" → 1 vline = 3 total
	if ev.VirtualLineCount() != 3 {
		t.Errorf("expected 3, got %d", ev.VirtualLineCount())
	}
	if ev.TotalVirtualLineCount() != 3 {
		t.Errorf("expected 3, got %d", ev.TotalVirtualLineCount())
	}
}

func TestEditBufferView_IsEmpty(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)

	ev := NewEditSurface(eb, 80, 24)

	if !ev.IsEmpty() {
		t.Error("expected empty")
	}

	eb.InsertChar('x')
	if ev.IsEmpty() {
		t.Error("expected not empty after insert")
	}

	eb.DeleteCharBackward()
	if !ev.IsEmpty() {
		t.Error("expected empty after deleting the only char")
	}
}

func TestEditBufferView_MoveUpDownVisual(t *testing.T) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	eb.SetText("abcdefgh") // 8 chars

	ev := NewEditSurface(eb, 4, 24)
	ev.SetWrapMode(WrapChar)
	// Virtual lines: "abcd" (0-3), "efgh" (4-7)

	// Start at (0,2)
	eb.SetCursor(0, 2)
	vc := ev.VisualCursor()
	if vc.VisualRow != 0 {
		t.Errorf("expected visual row 0, got %d", vc.VisualRow)
	}

	// Move down: should go to second virtual line, same column
	ev.MoveDownVisual()
	vc = ev.VisualCursor()
	if vc.VisualRow != 1 {
		t.Errorf("expected visual row 1 after move down, got %d", vc.VisualRow)
	}

	// Move up: should go back
	ev.MoveUpVisual()
	vc = ev.VisualCursor()
	if vc.VisualRow != 0 {
		t.Errorf("expected visual row 0 after move up, got %d", vc.VisualRow)
	}
}
