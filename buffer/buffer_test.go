package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

func TestNew(t *testing.T) {
	b := New(10, 5)
	if b.Width() != 10 {
		t.Errorf("Width() = %d, want 10", b.Width())
	}
	if b.Height() != 5 {
		t.Errorf("Height() = %d, want 5", b.Height())
	}

	// All cells should be CellClear
	for y := range 5 {
		for x := range 10 {
			c := b.Cell(x, y)
			if !c.Equals(CellClear) {
				t.Errorf("cell (%d,%d) should be CellClear, got %+v", x, y, c)
			}
		}
	}
}

func TestNewWithID(t *testing.T) {
	b := NewWithID(5, 5, "test-buf")
	if b.ID() != "test-buf" {
		t.Errorf("ID() = %q, want %q", b.ID(), "test-buf")
	}
}

func TestNew_NegativeDimensions(t *testing.T) {
	b := New(-1, -1)
	if b.Width() != 0 || b.Height() != 0 {
		t.Errorf("negative dims should clamp to 0, got %d x %d", b.Width(), b.Height())
	}
}

func TestBounds(t *testing.T) {
	b := New(10, 5)
	r := b.Bounds()
	if r.X != 0 || r.Y != 0 || r.Width != 10 || r.Height != 5 {
		t.Errorf("Bounds() = %+v, want {0 0 10 5}", r)
	}
}

func TestSetCell_GetCell(t *testing.T) {
	b := New(10, 10)
	b.SetCell(3, 4, 'X', style.ColorRed, style.ColorBlue, style.AttrBold)

	c := b.Cell(3, 4)
	if c.Char != 'X' {
		t.Errorf("Char = %q, want 'X'", c.Char)
	}
	if !c.Fg.Equals(style.ColorRed) {
		t.Errorf("Fg = %v, want Red", c.Fg)
	}
	if !c.Bg.Equals(style.ColorBlue) {
		t.Errorf("Bg = %v, want Blue", c.Bg)
	}
	if c.Attributes != style.AttrBold {
		t.Errorf("Attributes = %d, want AttrBold", c.Attributes)
	}
}

func TestSetCell_WideChar(t *testing.T) {
	b := New(10, 1)
	b.SetCell(2, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // '世'

	c := b.Cell(2, 0)
	if c.Char != '\u4e16' {
		t.Errorf("wide char cell Char = %q, want '世'", c.Char)
	}
	if c.Width != 2 {
		t.Errorf("wide char cell Width = %d, want 2", c.Width)
	}

	// Continuation cell
	cont := b.Cell(3, 0)
	if !cont.IsContinuation() {
		t.Error("cell after wide char should be continuation")
	}
}

// TestSetCell_WideChar_LastColumn verifies that SetCell silently drops a wide
// character whose continuation cell would fall outside the buffer. Before the
// fix, SetCell stored Width=2 at the last column without a continuation cell;
// InlineFlush then emitted a 2-column rune making the row 1 column wider than
// the buffer and triggering terminal auto-wrap.
func TestSetCell_WideChar_LastColumn(t *testing.T) {
	b := New(5, 1) // columns 0-4; last valid column is 4

	// Place a wide char at column 3 (fits — continuation at 4).
	b.SetCell(3, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // '世'
	if b.Cell(3, 0).Width != 2 {
		t.Errorf("col 3: expected Width=2, got %d", b.Cell(3, 0).Width)
	}
	if !b.Cell(4, 0).IsContinuation() {
		t.Error("col 4 should be continuation of the wide char at col 3")
	}

	// Now try placing a wide char at column 4 (last column — no room for continuation).
	b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
	// The continuation cell at col 5 is out of bounds; the write must be dropped.
	// Column 4 should still be the continuation cell from the previous write.
	if !b.Cell(4, 0).IsContinuation() {
		t.Errorf("col 4: wide char at last column should be dropped; cell should "+
			"remain a continuation cell (Width=%d)", b.Cell(4, 0).Width)
	}
}

// TestSetCellBlended_WideChar_LastColumn mirrors TestSetCell_WideChar_LastColumn
// for the blended variant.
func TestSetCellBlended_WideChar_LastColumn(t *testing.T) {
	b := New(3, 1) // columns 0-2; last valid column is 2

	// Wide char at col 2 must be dropped.
	b.SetCellBlended(2, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
	// Column 2 should remain CellClear (default after New).
	c := b.Cell(2, 0)
	if c.Width == 2 {
		t.Error("SetCellBlended: wide char at last column should be dropped, got Width=2")
	}
}

// TestSetCellDirect_WideCell_LastColumn mirrors the above for SetCellDirect.
func TestSetCellDirect_WideCell_LastColumn(t *testing.T) {
	b := New(3, 1)

	wideCell := Cell{Char: '\u4e16', Width: 2, Fg: style.ColorWhite, Bg: style.ColorBlack}
	b.SetCellDirect(2, 0, wideCell)
	c := b.Cell(2, 0)
	if c.Width == 2 {
		t.Error("SetCellDirect: wide cell at last column should be dropped, got Width=2")
	}
}

func TestGetCell_OOB(t *testing.T) {
	b := New(5, 5)
	c := b.Cell(-1, 0)
	if !c.Equals(CellClear) {
		t.Error("OOB cell should be CellClear")
	}
	c = b.Cell(5, 0)
	if !c.Equals(CellClear) {
		t.Error("OOB cell should be CellClear")
	}
	c = b.Cell(0, 5)
	if !c.Equals(CellClear) {
		t.Error("OOB cell should be CellClear")
	}
}

func TestSetCell_OOB(t *testing.T) {
	b := New(5, 5)
	// Should be a no-op, not panic
	b.SetCell(-1, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
	b.SetCell(5, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
	b.SetCell(0, 5, 'X', style.ColorWhite, style.ColorBlack, 0)
}

func TestClear(t *testing.T) {
	b := New(5, 5)
	b.SetCell(2, 2, 'A', style.ColorWhite, style.ColorBlack, 0)

	b.Clear(style.ColorRed)
	c := b.Cell(2, 2)
	if c.Char != ' ' {
		t.Errorf("after Clear, Char = %q, want ' '", c.Char)
	}
	if !c.Bg.Equals(style.ColorRed) {
		t.Errorf("after Clear(Red), Bg = %v, want Red", c.Bg)
	}
}

func TestDrawText(t *testing.T) {
	b := New(20, 1)
	b.DrawText("Hello", 2, 0, style.ColorWhite)

	expected := "Hello"
	for i, ch := range expected {
		c := b.Cell(2+i, 0)
		if c.Char != ch {
			t.Errorf("cell (%d,0).Char = %q, want %q", 2+i, c.Char, ch)
		}
	}
}

func TestDrawText_WithWideChars(t *testing.T) {
	b := New(20, 1)
	b.DrawText("\u4e16\u754c", 0, 0, style.ColorWhite) // "世界"

	c0 := b.Cell(0, 0)
	if c0.Char != '\u4e16' || c0.Width != 2 {
		t.Errorf("cell 0: Char=%q, Width=%d, want '世', 2", c0.Char, c0.Width)
	}
	c1 := b.Cell(1, 0)
	if !c1.IsContinuation() {
		t.Error("cell 1 should be continuation")
	}
	c2 := b.Cell(2, 0)
	if c2.Char != '\u754c' || c2.Width != 2 {
		t.Errorf("cell 2: Char=%q, Width=%d, want '界', 2", c2.Char, c2.Width)
	}
}

func TestDrawText_WithBg(t *testing.T) {
	b := New(20, 1)
	b.DrawText("Hi", 0, 0, style.ColorWhite, WithBg(style.ColorRed))

	c := b.Cell(0, 0)
	if !c.Bg.Equals(style.ColorRed) {
		t.Errorf("cell Bg = %v, want Red", c.Bg)
	}
}

func TestDrawText_WithMaxWidth(t *testing.T) {
	b := New(20, 1)
	b.DrawText("Hello World", 0, 0, style.ColorWhite, WithMaxWidth(5))

	c := b.Cell(5, 0)
	if c.Char != ' ' || !c.Equals(CellClear) {
		t.Error("text should stop at maxWidth")
	}
}

func TestDrawText_WithSelection(t *testing.T) {
	b := New(20, 1)
	blue := style.ColorBlue
	sel := textbuf.TextSelection{
		Start:   1,
		End:     3,
		BgColor: &blue,
	}
	b.DrawText("Hello", 0, 0, style.ColorWhite, WithSelection(sel))

	// Char at index 0 should not be selected
	c0 := b.Cell(0, 0)
	if c0.Bg.Equals(style.ColorBlue) {
		t.Error("char 0 should not be selected")
	}

	// Chars at index 1,2 should be selected
	c1 := b.Cell(1, 0)
	if !c1.Bg.Equals(style.ColorBlue) {
		t.Errorf("char 1 should be selected, Bg = %v", c1.Bg)
	}
	c2 := b.Cell(2, 0)
	if !c2.Bg.Equals(style.ColorBlue) {
		t.Errorf("char 2 should be selected, Bg = %v", c2.Bg)
	}

	// Char at index 3 should not be selected
	c3 := b.Cell(3, 0)
	if c3.Bg.Equals(style.ColorBlue) {
		t.Error("char 3 should not be selected")
	}
}

func TestDrawChar(t *testing.T) {
	b := New(10, 10)
	b.DrawChar('Z', 5, 5, style.ColorWhite, style.ColorBlack, style.AttrItalic)

	c := b.Cell(5, 5)
	if c.Char != 'Z' {
		t.Errorf("Char = %q, want 'Z'", c.Char)
	}
	if c.Attributes != style.AttrItalic {
		t.Error("expected AttrItalic")
	}
}

func TestFillRect(t *testing.T) {
	b := New(10, 10)
	b.FillRect(2, 2, 3, 3, style.ColorGreen)

	// Inside rect
	c := b.Cell(3, 3)
	if !c.Bg.Equals(style.ColorGreen) {
		t.Errorf("inside FillRect Bg = %v, want Green", c.Bg)
	}

	// Outside rect
	c2 := b.Cell(0, 0)
	if c2.Bg.Equals(style.ColorGreen) {
		t.Error("outside FillRect should not be Green")
	}
}

func TestDrawHLine(t *testing.T) {
	b := New(10, 1)
	b.DrawHLine(2, 0, 5, '-', style.ColorWhite, style.ColorBlack)

	for x := 2; x < 7; x++ {
		c := b.Cell(x, 0)
		if c.Char != '-' {
			t.Errorf("cell (%d,0) = %q, want '-'", x, c.Char)
		}
	}
	// Before
	c := b.Cell(1, 0)
	if c.Char == '-' {
		t.Error("cell before line should not be '-'")
	}
}

func TestDrawVLine(t *testing.T) {
	b := New(1, 10)
	b.DrawVLine(0, 2, 5, '|', style.ColorWhite, style.ColorBlack)

	for y := 2; y < 7; y++ {
		c := b.Cell(0, y)
		if c.Char != '|' {
			t.Errorf("cell (0,%d) = %q, want '|'", y, c.Char)
		}
	}
}

func TestDrawBox(t *testing.T) {
	b := New(10, 5)
	b.DrawBox(BoxDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderStyle:     style.BorderSingle,
		Border:          style.BorderAll(),
		BorderColor:     style.ColorWhite,
		BackgroundColor: style.ColorBlack,
		ShouldFill:      true,
	})

	// Top-left corner
	c := b.Cell(0, 0)
	if c.Char != style.SingleChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c.Char, style.SingleChars.TopLeft)
	}

	// Top-right corner
	c = b.Cell(9, 0)
	if c.Char != style.SingleChars.TopRight {
		t.Errorf("top-right = %q, want %q", c.Char, style.SingleChars.TopRight)
	}

	// Bottom-left corner
	c = b.Cell(0, 4)
	if c.Char != style.SingleChars.BottomLeft {
		t.Errorf("bottom-left = %q, want %q", c.Char, style.SingleChars.BottomLeft)
	}

	// Bottom-right corner
	c = b.Cell(9, 4)
	if c.Char != style.SingleChars.BottomRight {
		t.Errorf("bottom-right = %q, want %q", c.Char, style.SingleChars.BottomRight)
	}

	// Top edge (horizontal)
	c = b.Cell(5, 0)
	if c.Char != style.SingleChars.Horizontal {
		t.Errorf("top edge = %q, want %q", c.Char, style.SingleChars.Horizontal)
	}

	// Left edge (vertical)
	c = b.Cell(0, 2)
	if c.Char != style.SingleChars.Vertical {
		t.Errorf("left edge = %q, want %q", c.Char, style.SingleChars.Vertical)
	}
}

func TestDrawBox_WithTitle(t *testing.T) {
	b := New(20, 5)
	b.DrawBox(BoxDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		BorderStyle:     style.BorderSingle,
		Border:          style.BorderAll(),
		BorderColor:     style.ColorWhite,
		BackgroundColor: style.ColorBlack,
		ShouldFill:      true,
		Title:           "Test",
		TitleAlignment:  "left",
	})

	// Title should start at x=2 (padding=2)
	c := b.Cell(2, 0)
	if c.Char != 'T' {
		t.Errorf("title first char = %q, want 'T'", c.Char)
	}
}

func TestResize(t *testing.T) {
	b := New(5, 5)
	b.SetCell(2, 2, 'X', style.ColorWhite, style.ColorBlack, 0)
	b.PushScissorRect(0, 0, 5, 5)
	b.PushOpacity(0.5)

	b.Resize(10, 10)

	if b.Width() != 10 || b.Height() != 10 {
		t.Errorf("after Resize: %d x %d, want 10 x 10", b.Width(), b.Height())
	}

	// Old data should be gone
	c := b.Cell(2, 2)
	if c.Char != ' ' {
		t.Error("after Resize cells should be cleared")
	}

	// Stacks should be cleared
	if b.CurrentOpacity() != 1.0 {
		t.Error("after Resize opacity should be 1.0")
	}
}

func TestPlainText(t *testing.T) {
	b := New(5, 2)
	b.DrawText("Hello", 0, 0, style.ColorWhite)
	b.DrawText("World", 0, 1, style.ColorWhite)

	got := b.PlainText(true)
	want := "Hello\nWorld"
	if got != want {
		t.Errorf("PlainText(true) = %q, want %q", got, want)
	}

	got = b.PlainText()
	want = "HelloWorld"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}

func TestPlainText_WithWideChars(t *testing.T) {
	b := New(10, 1)
	b.DrawText("A\u4e16B", 0, 0, style.ColorWhite) // "A世B"

	got := b.PlainText()
	// Wide char occupies 2 cells. "A" + "世" (2 cells) + "B" = 4 display cells used
	// Remaining cells are spaces
	if len(got) < 3 {
		t.Errorf("PlainText should contain at least 'A世B', got %q", got)
	}
	// Check first few chars
	runes := []rune(got)
	if runes[0] != 'A' || runes[1] != '\u4e16' || runes[2] != 'B' {
		t.Errorf("PlainText runes = %v, want [A 世 B ...]", runes[:3])
	}
}

func TestLines(t *testing.T) {
	b := New(5, 1)
	b.DrawText("AB", 0, 0, style.ColorRed)
	b.DrawText("CD", 2, 0, style.ColorBlue)

	lines := b.Lines()
	if len(lines) != 1 {
		t.Fatalf("Lines() returned %d lines, want 1", len(lines))
	}

	// Should have at least 2 spans (different fg colors)
	// plus possibly a trailing space span
	if len(lines[0].Spans) < 2 {
		t.Errorf("expected at least 2 spans, got %d", len(lines[0].Spans))
	}
}

func TestScissorRect_BlocksWrite(t *testing.T) {
	b := New(10, 10)
	b.PushScissorRect(2, 2, 3, 3)

	// Inside clip region: should work
	b.SetCell(3, 3, 'Y', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(3, 3).Char != 'Y' {
		t.Error("cell inside scissor should be written")
	}

	// Outside clip region: should be blocked
	b.SetCell(0, 0, 'N', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(0, 0).Char == 'N' {
		t.Error("cell outside scissor should be blocked")
	}

	b.PopScissorRect()

	// After pop, should work everywhere
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(0, 0).Char != 'A' {
		t.Error("after PopScissorRect, write should work")
	}
}

func TestOpacity_AffectsCells(t *testing.T) {
	b := New(10, 1)

	b.PushOpacity(0.5)
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)

	c := b.Cell(0, 0)
	// White fg has A=255, with opacity 0.5 it should be ~128
	if c.Fg.A < 125 || c.Fg.A > 130 {
		t.Errorf("Fg.A = %d, want ~128", c.Fg.A)
	}
	if c.Bg.A < 125 || c.Bg.A > 130 {
		t.Errorf("Bg.A = %d, want ~128 (Black.A=255 * 0.5)", c.Bg.A)
	}

	b.PopOpacity()
}

func TestCopyFrom(t *testing.T) {
	a := New(5, 3)
	a.SetCell(1, 1, 'Q', style.ColorRed, style.ColorBlue, 0)

	b := New(5, 3)
	b.CopyFrom(a)

	c := b.Cell(1, 1)
	if c.Char != 'Q' {
		t.Errorf("CopyFrom: Char = %q, want 'Q'", c.Char)
	}
	if !c.Fg.Equals(style.ColorRed) {
		t.Error("CopyFrom: Fg should be Red")
	}
}

func TestCopyFrom_DifferentSize(t *testing.T) {
	a := New(5, 3)
	b := New(10, 10)
	b.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0)

	b.CopyFrom(a) // different size, should be no-op
	c := b.Cell(0, 0)
	if c.Char != 'Z' {
		t.Error("CopyFrom with mismatched sizes should be no-op")
	}
}

func TestSetRespectAlpha(t *testing.T) {
	b := New(5, 5)
	if b.RespectAlpha() {
		t.Error("default RespectAlpha should be false")
	}
	b.SetRespectAlpha(true)
	if !b.RespectAlpha() {
		t.Error("after SetRespectAlpha(true), should be true")
	}
}

func TestRect_Contains(t *testing.T) {
	r := Rect{X: 5, Y: 5, Width: 10, Height: 10}

	if !r.Contains(5, 5) {
		t.Error("should contain top-left")
	}
	if !r.Contains(14, 14) {
		t.Error("should contain bottom-right edge")
	}
	if r.Contains(4, 5) {
		t.Error("should not contain point left of rect")
	}
	if r.Contains(15, 5) {
		t.Error("should not contain point right of rect")
	}
}

func TestRect_Intersect(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	b := Rect{X: 5, Y: 5, Width: 10, Height: 10}

	r := a.Intersect(b)
	if r.X != 5 || r.Y != 5 || r.Width != 5 || r.Height != 5 {
		t.Errorf("Intersect = %+v, want {5 5 5 5}", r)
	}
}

func TestRect_Intersect_NoOverlap(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 5, Height: 5}
	b := Rect{X: 10, Y: 10, Width: 5, Height: 5}

	r := a.Intersect(b)
	if !r.IsEmpty() {
		t.Errorf("non-overlapping Intersect should be empty, got %+v", r)
	}
}

func TestRect_IsEmpty(t *testing.T) {
	if !(Rect{Width: 0, Height: 5}).IsEmpty() {
		t.Error("zero width should be empty")
	}
	if !(Rect{Width: 5, Height: 0}).IsEmpty() {
		t.Error("zero height should be empty")
	}
	if (Rect{Width: 5, Height: 5}).IsEmpty() {
		t.Error("5x5 should not be empty")
	}
}

func TestRect_RightBottom(t *testing.T) {
	r := Rect{X: 3, Y: 4, Width: 10, Height: 20}
	if r.Right() != 13 {
		t.Errorf("Right() = %d, want 13", r.Right())
	}
	if r.Bottom() != 24 {
		t.Errorf("Bottom() = %d, want 24", r.Bottom())
	}
}

func floatEq2(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}

// ---------------------------------------------------------------------------
// Border draw test helpers
// ---------------------------------------------------------------------------

func getChar(b *Buffer, x, y int) rune {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return 0
	}
	return b.cells[y*b.width+x].Char
}

func getBg(b *Buffer, x, y int) style.Color {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return style.ColorTransparent
	}
	return b.cells[y*b.width+x].Bg
}

// ---------------------------------------------------------------------------
// DrawBorder tests
// ---------------------------------------------------------------------------

func TestDrawBorder_AllSides_Single(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		BackgroundColor: style.ColorBlack,
		ShouldFill:      true,
	})

	// Corners
	if c := getChar(buf, 0, 0); c != style.SingleChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c, style.SingleChars.TopLeft)
	}
	if c := getChar(buf, 9, 0); c != style.SingleChars.TopRight {
		t.Errorf("top-right = %q, want %q", c, style.SingleChars.TopRight)
	}
	if c := getChar(buf, 0, 4); c != style.SingleChars.BottomLeft {
		t.Errorf("bottom-left = %q, want %q", c, style.SingleChars.BottomLeft)
	}
	if c := getChar(buf, 9, 4); c != style.SingleChars.BottomRight {
		t.Errorf("bottom-right = %q, want %q", c, style.SingleChars.BottomRight)
	}

	// Top edge
	for x := 1; x < 9; x++ {
		if c := getChar(buf, x, 0); c != style.SingleChars.Horizontal {
			t.Errorf("top edge (%d,0) = %q, want %q", x, c, style.SingleChars.Horizontal)
		}
	}

	// Bottom edge
	for x := 1; x < 9; x++ {
		if c := getChar(buf, x, 4); c != style.SingleChars.Horizontal {
			t.Errorf("bottom edge (%d,4) = %q, want %q", x, c, style.SingleChars.Horizontal)
		}
	}

	// Left edge
	for y := 1; y < 4; y++ {
		if c := getChar(buf, 0, y); c != style.SingleChars.Vertical {
			t.Errorf("left edge (0,%d) = %q, want %q", y, c, style.SingleChars.Vertical)
		}
	}

	// Right edge
	for y := 1; y < 4; y++ {
		if c := getChar(buf, 9, y); c != style.SingleChars.Vertical {
			t.Errorf("right edge (9,%d) = %q, want %q", y, c, style.SingleChars.Vertical)
		}
	}

	// Interior should be filled
	bg := getBg(buf, 5, 2)
	if !bg.Equals(style.ColorBlack) {
		t.Errorf("interior bg = %v, want Black", bg)
	}
}

func TestDrawBorder_AllSides_Double(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderDouble,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		BackgroundColor: style.ColorBlack,
		ShouldFill:      false,
	})

	if c := getChar(buf, 0, 0); c != style.DoubleChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c, style.DoubleChars.TopLeft)
	}
	if c := getChar(buf, 5, 0); c != style.DoubleChars.Horizontal {
		t.Errorf("top edge = %q, want %q", c, style.DoubleChars.Horizontal)
	}
	if c := getChar(buf, 0, 2); c != style.DoubleChars.Vertical {
		t.Errorf("left edge = %q, want %q", c, style.DoubleChars.Vertical)
	}
}

func TestDrawBorder_AllSides_Rounded(t *testing.T) {
	buf := New(6, 4)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 6, Height: 4,
		BorderConfig: BorderConfig{
			Style:  style.BorderRounded,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
	})

	if c := getChar(buf, 0, 0); c != style.RoundedChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c, style.RoundedChars.TopLeft)
	}
	if c := getChar(buf, 5, 3); c != style.RoundedChars.BottomRight {
		t.Errorf("bottom-right = %q, want %q", c, style.RoundedChars.BottomRight)
	}
}

func TestDrawBorder_AllSides_Heavy(t *testing.T) {
	buf := New(6, 4)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 6, Height: 4,
		BorderConfig: BorderConfig{
			Style:  style.BorderHeavy,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
	})

	if c := getChar(buf, 0, 0); c != style.HeavyChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c, style.HeavyChars.TopLeft)
	}
	if c := getChar(buf, 3, 0); c != style.HeavyChars.Horizontal {
		t.Errorf("top edge = %q, want %q", c, style.HeavyChars.Horizontal)
	}
}

func TestDrawBorder_TopOnly(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderSides(style.SideTop),
			Color:  style.ColorWhite,
		},
	})

	// Top edge should be horizontal (no corners since left/right are off)
	for x := range 10 {
		c := getChar(buf, x, 0)
		if c != style.SingleChars.Horizontal {
			t.Errorf("top-only: (%d,0) = %q, want horizontal", x, c)
		}
	}

	// No left/right/bottom edges (should remain default space)
	if c := getChar(buf, 0, 2); c != ' ' {
		t.Errorf("no left edge expected, got %q", c)
	}
	if c := getChar(buf, 9, 2); c != ' ' {
		t.Errorf("no right edge expected, got %q", c)
	}
}

func TestDrawBorder_LeftRight(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderSides(style.SideLeft, style.SideRight),
			Color:  style.ColorWhite,
		},
	})

	// Left edge (full height since no top/bottom)
	for y := range 5 {
		c := getChar(buf, 0, y)
		if c != style.SingleChars.Vertical {
			t.Errorf("left edge (0,%d) = %q, want vertical", y, c)
		}
	}

	// Right edge
	for y := range 5 {
		c := getChar(buf, 9, y)
		if c != style.SingleChars.Vertical {
			t.Errorf("right edge (9,%d) = %q, want vertical", y, c)
		}
	}

	// No top/bottom (should remain default space)
	if c := getChar(buf, 5, 0); c != ' ' {
		t.Errorf("no top edge expected at (5,0), got %q", c)
	}
}

func TestDrawBorder_NoSides(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderNoBorder(),
			Color:  style.ColorWhite,
		},
	})

	// Nothing should be drawn (should remain default space)
	if c := getChar(buf, 0, 0); c != ' ' {
		t.Errorf("no border: (0,0) should be empty, got %q", c)
	}
}

func TestDrawBorder_WithFill(t *testing.T) {
	buf := New(10, 5)
	bgColor := style.ColorFromInts(128, 128, 128, 255)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		BackgroundColor: bgColor,
		ShouldFill:      true,
	})

	// Interior (1,1) to (8,3) should have bg color
	bg := getBg(buf, 5, 2)
	if !bg.Equals(bgColor) {
		t.Errorf("interior bg = %v, want %v", bg, bgColor)
	}
}

func TestDrawBorder_Offset(t *testing.T) {
	buf := New(20, 10)
	buf.DrawBorder(BorderDrawOptions{
		X: 5, Y: 3, Width: 8, Height: 4,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
	})

	// Top-left corner at (5,3)
	if c := getChar(buf, 5, 3); c != style.SingleChars.TopLeft {
		t.Errorf("offset top-left = %q, want %q", c, style.SingleChars.TopLeft)
	}
	// Bottom-right corner at (12,6)
	if c := getChar(buf, 12, 6); c != style.SingleChars.BottomRight {
		t.Errorf("offset bottom-right = %q, want %q", c, style.SingleChars.BottomRight)
	}
}

func TestDrawBorder_ZeroDimensions(t *testing.T) {
	buf := New(10, 10)
	// Should not panic
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 0, Height: 0,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
	})
}

// ---------------------------------------------------------------------------
// Title tests
// ---------------------------------------------------------------------------

func TestDrawBorder_TitleLeft(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		Title:          "Test",
		TitleAlignment: layout.TextAlignLeft,
	})

	// Title should start at x=1 (just after left corner)
	if c := getChar(buf, 1, 0); c != 'T' {
		t.Errorf("title left: (1,0) = %q, want 'T'", c)
	}
	if c := getChar(buf, 2, 0); c != 'e' {
		t.Errorf("title left: (2,0) = %q, want 'e'", c)
	}
	if c := getChar(buf, 3, 0); c != 's' {
		t.Errorf("title left: (3,0) = %q, want 's'", c)
	}
	if c := getChar(buf, 4, 0); c != 't' {
		t.Errorf("title left: (4,0) = %q, want 't'", c)
	}
}

func TestDrawBorder_TitleCenter(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		Title:          "Hi",
		TitleAlignment: layout.TextAlignCenter,
	})

	// maxWidth = 20-2 = 18, titleLen = 2, startX = 1 + (18-2)/2 = 1+8 = 9
	if c := getChar(buf, 9, 0); c != 'H' {
		t.Errorf("title center: (9,0) = %q, want 'H'", c)
	}
	if c := getChar(buf, 10, 0); c != 'i' {
		t.Errorf("title center: (10,0) = %q, want 'i'", c)
	}
}

func TestDrawBorder_TitleRight(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		Title:          "OK",
		TitleAlignment: layout.TextAlignRight,
	})

	// maxWidth = 18, titleLen = 2, startX = 1 + 18 - 2 = 17
	if c := getChar(buf, 17, 0); c != 'O' {
		t.Errorf("title right: (17,0) = %q, want 'O'", c)
	}
	if c := getChar(buf, 18, 0); c != 'K' {
		t.Errorf("title right: (18,0) = %q, want 'K'", c)
	}
}

func TestDrawBorder_TitleTruncation(t *testing.T) {
	buf := New(6, 4) // inner title width = 6-2 = 4
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 6, Height: 4,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderAll(),
			Color:  style.ColorWhite,
		},
		Title:          "TooLong",
		TitleAlignment: layout.TextAlignLeft,
	})

	// Only first 4 chars should fit
	expected := "TooL"
	for i, ch := range expected {
		c := getChar(buf, 1+i, 0)
		if c != ch {
			t.Errorf("truncated title: (%d,0) = %q, want %q", 1+i, c, ch)
		}
	}
	// Next cell should be the border, not 'o'
	if c := getChar(buf, 5, 0); c == 'o' {
		t.Error("title should have been truncated")
	}
}

func TestDrawBorder_TitleNoTopBorder(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		BorderConfig: BorderConfig{
			Style:  style.BorderSingle,
			Border: style.BorderSides(style.SideLeft, style.SideRight, style.SideBottom), // no top
			Color:  style.ColorWhite,
		},
		Title:          "Skip",
		TitleAlignment: layout.TextAlignLeft,
	})

	// Title should NOT be drawn since there's no top border
	if c := getChar(buf, 1, 0); c == 'S' {
		t.Error("title should not be drawn without top border")
	}
}

// ---------------------------------------------------------------------------
// InnerRect tests
// ---------------------------------------------------------------------------

func TestInnerRect_AllSides(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.AllSides())
	if r.X != 1 || r.Y != 1 || r.Width != 8 || r.Height != 6 {
		t.Errorf("InnerRect(AllSides) = %+v, want {1 1 8 6}", r)
	}
}

func TestInnerRect_NoSides(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.NoSides())
	if r.X != 0 || r.Y != 0 || r.Width != 10 || r.Height != 8 {
		t.Errorf("InnerRect(NoSides) = %+v, want {0 0 10 8}", r)
	}
}

func TestInnerRect_TopBottom(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.BorderSidesConfig{Top: true, Bottom: true})
	if r.X != 0 || r.Y != 1 || r.Width != 10 || r.Height != 6 {
		t.Errorf("InnerRect(Top,Bottom) = %+v, want {0 1 10 6}", r)
	}
}

func TestInnerRect_LeftRight(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.BorderSidesConfig{Left: true, Right: true})
	if r.X != 1 || r.Y != 0 || r.Width != 8 || r.Height != 8 {
		t.Errorf("InnerRect(Left,Right) = %+v, want {1 0 8 8}", r)
	}
}

func TestInnerRect_Offset(t *testing.T) {
	r := InnerRect(5, 3, 10, 8, style.AllSides())
	if r.X != 6 || r.Y != 4 || r.Width != 8 || r.Height != 6 {
		t.Errorf("InnerRect with offset = %+v, want {6 4 8 6}", r)
	}
}

func TestInnerRect_TooSmall(t *testing.T) {
	// Width=2, with left+right borders leaves 0 inner width
	r := InnerRect(0, 0, 2, 2, style.AllSides())
	if r.Width != 0 || r.Height != 0 {
		t.Errorf("InnerRect too small = %+v, want Width=0 Height=0", r)
	}
}

func TestInnerRect_TooSmallClamps(t *testing.T) {
	// Width=1 with left+right gives -1 which should clamp to 0
	r := InnerRect(0, 0, 1, 1, style.AllSides())
	if r.Width < 0 || r.Height < 0 {
		t.Errorf("InnerRect should not have negative dimensions: %+v", r)
	}
}

func TestInnerRect_SingleSide(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.BorderSidesConfig{Top: true})
	if r.X != 0 || r.Y != 1 || r.Width != 10 || r.Height != 7 {
		t.Errorf("InnerRect(Top only) = %+v, want {0 1 10 7}", r)
	}
}
