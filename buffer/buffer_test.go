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

	// New initializes every cell to CellClear.
	for y := range 5 {
		for x := range 10 {
			c := b.Cell(x, y)
			if !c.Equals(CellClear) {
				t.Errorf("cell (%d,%d) should be CellClear, got %+v", x, y, c)
			}
		}
	}
}

func TestNew_WithID(t *testing.T) {
	b := New(5, 5, WithID("test-buf"))
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
	b.SetCell(2, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // 世

	c := b.Cell(2, 0)
	if c.Char != '\u4e16' {
		t.Errorf("wide char cell Char = %q, want '世'", c.Char)
	}
	if c.Width != 2 {
		t.Errorf("wide char cell Width = %d, want 2", c.Width)
	}

	// The wide character occupies the following cell as a continuation.
	cont := b.Cell(3, 0)
	if !cont.IsContinuation() {
		t.Error("cell after wide char should be continuation")
	}
}

// TestSetCell_WideChar_LastColumn verifies that SetCell discards a wide
// character whose continuation would lie outside the buffer. A stranded head
// would make the terminal advance beyond the row and trigger automatic wrap.
func TestSetCell_WideChar_LastColumn(t *testing.T) {
	b := New(5, 1) // Valid columns are 0 through 4.

	// A wide character at column 3 fits because its continuation occupies column 4.
	b.SetCell(3, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // 世
	if b.Cell(3, 0).Width != 2 {
		t.Errorf("col 3: expected Width=2, got %d", b.Cell(3, 0).Width)
	}
	if !b.Cell(4, 0).IsContinuation() {
		t.Error("col 4 should be continuation of the wide char at col 3")
	}

	// A wide character at column 4 has no room for its continuation.
	b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
	// Discarding the write preserves the existing continuation in column 4.
	if !b.Cell(4, 0).IsContinuation() {
		t.Errorf("col 4: wide char at last column should be dropped; cell should "+
			"remain a continuation cell (Width=%d)", b.Cell(4, 0).Width)
	}
}

// TestSetCellBlended_WideChar_LastColumn verifies the same boundary behavior
// for blended writes.
func TestSetCellBlended_WideChar_LastColumn(t *testing.T) {
	b := New(3, 1) // Valid columns are 0 through 2.

	// Column 2 has no room for a continuation cell.
	b.SetCellBlended(2, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
	// The discarded write leaves column 2 clear.
	c := b.Cell(2, 0)
	if c.Width == 2 {
		t.Error("SetCellBlended: wide char at last column should be dropped, got Width=2")
	}
}

// TestSetCellDirect_WideCell_LastColumn verifies the same boundary behavior
// for direct writes.
func TestSetCellDirect_WideCell_LastColumn(t *testing.T) {
	b := New(3, 1)

	wideCell := Cell{Char: '\u4e16', Width: 2, Fg: style.ColorWhite, Bg: style.ColorBlack}
	b.SetCellDirect(2, 0, wideCell)
	c := b.Cell(2, 0)
	if c.Width == 2 {
		t.Error("SetCellDirect: wide cell at last column should be dropped, got Width=2")
	}
}

// TestSetGraphemeCluster_SpacingMark verifies that a cluster containing a
// spacing mark claims every column it renders. If its width came only from the
// first rune, the terminal would advance farther than the buffer records.
func TestSetGraphemeCluster_SpacingMark(t *testing.T) {
	b := New(10, 1)
	b.SetGraphemeCluster(2, 0, "\u0E01\u0E33", style.ColorWhite, style.ColorBlack, 0) // กำ

	c := b.Cell(2, 0)
	if c.Char != '\u0E01' {
		t.Errorf("Char = %q, want '\u0E01'", c.Char)
	}
	if c.Combining() != "\u0E33" {
		t.Errorf("Combining = %q, want %q", c.Combining(), "\u0E33")
	}
	if c.Width != 2 {
		t.Errorf("Width = %d, want 2", c.Width)
	}
	if !b.Cell(3, 0).IsContinuation() {
		t.Error("cell after spacing-mark cluster should be a continuation")
	}
}

// TestDrawClusters_ColumnAccounting verifies that the columns claimed by a row
// equal the columns the terminal advances while flushing it. A mismatch would
// desynchronize the renderer's cursor position.
//
// The terminal advances by each emitted code point's width, so the test sums
// those widths independently of the stored cluster width.
func TestDrawClusters_ColumnAccounting(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"ascii", "the quick brown fox"},
		{"cjk", "\u4e16\u754c mixed \u4e2d\u6587"},
		{"thai spacing marks", "\u0E23\u0E32\u0E04\u0E32 100 \u0E1A\u0E32\u0E17 \u0E04\u0E33\u0E19\u0E27\u0E13"},
		{"thai nonspacing marks", "\u0E17\u0E35\u0E48\u0E19\u0E35\u0E48 here"},
		{"bom", "\ufeffleading BOM"},
		{"emoji", "ok \U0001F680 done"},
	}

	const width = 40
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(width, 1)
			b.Clear(style.ColorTransparent)
			b.DrawClusters(tt.text, 0, 0, width, style.ColorWhite, style.ColorTransparent, 0, nil)

			claimed, advanced := 0, 0
			for x := range width {
				c := b.Cell(x, 0)
				if c.IsContinuation() {
					continue
				}
				claimed += int(c.Width)
				ch := c.Char
				if ch == 0 {
					ch = ' '
				}
				advanced += textbuf.RuneWidth(ch)
				for _, r := range c.Combining() {
					advanced += textbuf.RuneWidth(r)
				}
			}
			if claimed != width {
				t.Errorf("cells claim %d columns, want %d", claimed, width)
			}
			if advanced != width {
				t.Errorf("flushed row advances %d columns, want %d", advanced, width)
			}
		})
	}
}

// TestDetachWide_OverdrawKeepsRowWidth covers drawing over half of a wide
// character. A stranded continuation makes the flushed row one column short
// and a stranded head makes it one column long; either desynchronises the
// renderer's cursor and strands garbage in rows the diff never repaints.
func TestDetachWide_OverdrawKeepsRowWidth(t *testing.T) {
	const width = 10
	fg, bg := style.ColorWhite, style.ColorBlack

	// Continuation cells emit nothing because their head already advanced over them.
	advanced := func(b *Buffer) int {
		n := 0
		for x := range width {
			c := b.Cell(x, 0)
			if c.Width == -1 {
				continue
			}
			n += textbuf.RuneWidth(c.Char)
			for _, r := range c.Combining() {
				n += textbuf.RuneWidth(r)
			}
		}
		return n
	}

	tests := []struct {
		name      string
		overdraw  func(b *Buffer)
		wantAt4   rune
		wantAt5   rune
		wantWidth int
	}{
		{
			name:      "overwrite head",
			overdraw:  func(b *Buffer) { b.SetCell(4, 0, 'X', fg, bg, 0) },
			wantAt4:   'X',
			wantAt5:   ' ',
			wantWidth: 1,
		},
		{
			name:      "overwrite continuation",
			overdraw:  func(b *Buffer) { b.SetCell(5, 0, 'X', fg, bg, 0) },
			wantAt4:   ' ',
			wantAt5:   'X',
			wantWidth: 1,
		},
		{
			name:      "fill over head",
			overdraw:  func(b *Buffer) { b.FillRect(0, 0, 5, 1, bg) },
			wantAt4:   ' ',
			wantAt5:   ' ',
			wantWidth: 1,
		},
		{
			name:      "blend over continuation",
			overdraw:  func(b *Buffer) { b.SetCellBlended(5, 0, 'X', fg, bg, 0) },
			wantAt4:   ' ',
			wantAt5:   'X',
			wantWidth: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(width, 1)
			b.Clear(style.ColorTransparent)
			for x := range width {
				b.SetCell(x, 0, ' ', fg, bg, 0)
			}
			b.SetCell(4, 0, '\u4e16', fg, bg, 0) // The head is at 4 and its continuation at 5.
			tt.overdraw(b)

			if got := b.Cell(4, 0).Char; got != tt.wantAt4 {
				t.Errorf("cell 4 Char = %q, want %q", got, tt.wantAt4)
			}
			if got := b.Cell(5, 0).Char; got != tt.wantAt5 {
				t.Errorf("cell 5 Char = %q, want %q", got, tt.wantAt5)
			}
			if got := int(b.Cell(4, 0).Width); got != tt.wantWidth {
				t.Errorf("cell 4 Width = %d, want %d", got, tt.wantWidth)
			}
			if got := advanced(b); got != width {
				t.Errorf("flushed row advances %d columns, want %d", got, width)
			}
		})
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
	// Out-of-bounds writes must be safe no-ops.
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
	b.DrawText("\u4e16\u754c", 0, 0, style.ColorWhite) // 世界

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
		BgColor: blue,
	}
	b.DrawText("Hello", 0, 0, style.ColorWhite, WithSelection(sel))

	c0 := b.Cell(0, 0)
	if c0.Bg.Equals(style.ColorBlue) {
		t.Error("char 0 should not be selected")
	}

	c1 := b.Cell(1, 0)
	if !c1.Bg.Equals(style.ColorBlue) {
		t.Errorf("char 1 should be selected, Bg = %v", c1.Bg)
	}
	c2 := b.Cell(2, 0)
	if !c2.Bg.Equals(style.ColorBlue) {
		t.Errorf("char 2 should be selected, Bg = %v", c2.Bg)
	}

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

	c := b.Cell(3, 3)
	if !c.Bg.Equals(style.ColorGreen) {
		t.Errorf("inside FillRect Bg = %v, want Green", c.Bg)
	}

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
	// The line does not affect the preceding cell.
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

	c := b.Cell(0, 0)
	if c.Char != style.SingleChars.TopLeft {
		t.Errorf("top-left = %q, want %q", c.Char, style.SingleChars.TopLeft)
	}

	c = b.Cell(9, 0)
	if c.Char != style.SingleChars.TopRight {
		t.Errorf("top-right = %q, want %q", c.Char, style.SingleChars.TopRight)
	}

	c = b.Cell(0, 4)
	if c.Char != style.SingleChars.BottomLeft {
		t.Errorf("bottom-left = %q, want %q", c.Char, style.SingleChars.BottomLeft)
	}

	c = b.Cell(9, 4)
	if c.Char != style.SingleChars.BottomRight {
		t.Errorf("bottom-right = %q, want %q", c.Char, style.SingleChars.BottomRight)
	}

	c = b.Cell(5, 0)
	if c.Char != style.SingleChars.Horizontal {
		t.Errorf("top edge = %q, want %q", c.Char, style.SingleChars.Horizontal)
	}

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
		TitleAlignment:  layout.TextAlignLeft,
	})

	// DrawBox reserves two columns between the left edge and the title.
	c := b.Cell(2, 0)
	if c.Char != 'T' {
		t.Errorf("title first char = %q, want 'T'", c.Char)
	}
}

func TestResize(t *testing.T) {
	b := New(5, 5)
	b.SetCell(2, 2, 'X', style.ColorWhite, style.ColorBlack, 0)
	b.PushScissorRect(0, 0, 5, 5)

	b.Resize(10, 10)

	if b.Width() != 10 || b.Height() != 10 {
		t.Errorf("after Resize: %d x %d, want 10 x 10", b.Width(), b.Height())
	}

	// Resize discards the previous cells.
	c := b.Cell(2, 2)
	if c.Char != ' ' {
		t.Error("after Resize cells should be cleared")
	}
}

func TestPlainText(t *testing.T) {
	b := New(5, 2)
	b.DrawText("Hello", 0, 0, style.ColorWhite)
	b.DrawText("World", 0, 1, style.ColorWhite)

	got := b.String()
	want := "Hello\nWorld"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	got = b.PlainText()
	want = "HelloWorld"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}

func TestPlainText_WithWideChars(t *testing.T) {
	b := New(10, 1)
	b.DrawText("A\u4e16B", 0, 0, style.ColorWhite) // A世B

	got := b.PlainText()
	// PlainText omits the continuation cell but retains trailing empty cells.
	if len(got) < 3 {
		t.Errorf("PlainText should contain at least 'A世B', got %q", got)
	}
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

	// The foreground color change creates a second span; trailing spaces can
	// create another span.
	if len(lines[0].Spans) < 2 {
		t.Errorf("expected at least 2 spans, got %d", len(lines[0].Spans))
	}
}

func TestScissorRect_BlocksWrite(t *testing.T) {
	b := New(10, 10)
	b.PushScissorRect(2, 2, 3, 3)

	// The clipping region permits the first write.
	b.SetCell(3, 3, 'Y', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(3, 3).Char != 'Y' {
		t.Error("cell inside scissor should be written")
	}

	// The clipping region rejects the second write.
	b.SetCell(0, 0, 'N', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(0, 0).Char == 'N' {
		t.Error("cell outside scissor should be blocked")
	}

	b.PopScissorRect()

	// Popping the only clipping region restores unrestricted drawing.
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	if b.Cell(0, 0).Char != 'A' {
		t.Error("after PopScissorRect, write should work")
	}
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

	b.CopyFrom(a) // Mismatched dimensions make CopyFrom a no-op.
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

// TestRectEmptyResultsAreZero pins the property callers rely on: an empty
// result carries no position, so reading X or Y without rechecking IsEmpty
// cannot pick up a stale coordinate.
func TestRectEmptyResultsAreZero(t *testing.T) {
	tests := []struct {
		name string
		got  Rect
	}{
		{
			name: "intersect misses entirely",
			got:  Rect{Width: 39, Height: 2}.Intersect(Rect{Y: 8, Width: 39, Height: 2}),
		},
		{
			name: "intersect touches along an edge",
			got:  Rect{Y: 4, Width: 39, Height: 2}.Intersect(Rect{Width: 39, Height: 4}),
		},
		{
			name: "union of two flat operands",
			got:  Rect{}.Union(Rect{X: 39, Y: 7, Width: 0, Height: 2}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != (Rect{}) {
				t.Errorf("got %+v, want the zero rectangle", tt.got)
			}
		})
	}
}

// TestCopyRowsFromClampsToBothBuffers covers a shorter source. Copying the rows
// both buffers hold beats refusing the call, which leaves the destination
// silently describing something the source never held.
func TestCopyRowsFromClampsToBothBuffers(t *testing.T) {
	dst := New(4, 6)
	dst.Clear(style.ColorBlack)
	src := New(4, 3)
	for y := range 3 {
		src.SetCell(0, y, 'x', style.ColorWhite, style.ColorBlack, 0)
	}

	dst.CopyRowsFrom(src, 1, 4) // Rows 1-4 requested, rows 1-2 available.

	for y := 1; y < 3; y++ {
		if got := dst.Cell(0, y).Char; got != 'x' {
			t.Errorf("row %d holds %q, want %q", y, got, 'x')
		}
	}
	if got := dst.Cell(0, 3).Char; got == 'x' {
		t.Error("row 3 was copied from beyond the source")
	}
}

// TestSetHeightKeepsRowsInPlace covers the reason SetHeight exists. A frame as
// tall as a whole transcript must not be copied to gain or lose a row.
func TestSetHeightKeepsRowsInPlace(t *testing.T) {
	b := New(4, 4)
	for y := range 4 {
		b.SetCell(0, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
	}

	b.SetHeight(6)

	if b.Height() != 6 {
		t.Fatalf("height is %d, want 6", b.Height())
	}
	for y := range 4 {
		if got := b.Cell(0, y).Char; got != rune('a'+y) {
			t.Errorf("row %d holds %q, want %q", y, got, rune('a'+y))
		}
	}
	// Added rows read as clear, exactly as a fresh buffer does.
	for y := 4; y < 6; y++ {
		if got := b.Cell(0, y); got != CellClear {
			t.Errorf("added row %d holds %+v, want CellClear", y, got)
		}
	}
}

func TestSetHeightShrinkThenGrowClearsReusedRows(t *testing.T) {
	b := New(4, 6)
	for y := range 6 {
		b.SetCell(0, y, 'x', style.ColorWhite, style.ColorBlack, 0)
	}

	b.SetHeight(2)
	if b.Height() != 2 {
		t.Fatalf("height is %d, want 2", b.Height())
	}
	b.SetHeight(5)

	// Rows reclaimed from the backing array must not resurface as stale content.
	for y := 2; y < 5; y++ {
		if got := b.Cell(0, y); got != CellClear {
			t.Errorf("reused row %d holds %+v, want CellClear", y, got)
		}
	}
	if got := b.Cell(0, 0).Char; got != 'x' {
		t.Errorf("kept row holds %q, want %q", got, 'x')
	}
}

func TestSetHeightZeroAndNegative(t *testing.T) {
	for _, h := range []int{0, -3} {
		b := New(4, 4)
		b.SetHeight(h)
		if b.Height() != 0 {
			t.Errorf("SetHeight(%d) gave height %d, want 0", h, b.Height())
		}
		if got := b.Cell(0, 0); got != CellClear {
			t.Errorf("SetHeight(%d) left %+v readable", h, got)
		}
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

func TestDrawBorder_AllSides_Single(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		Style:           style.BorderSingle,
		Border:          style.BorderAll(),
		Color:           style.ColorWhite,
		BackgroundColor: style.ColorBlack,
		ShouldFill:      true,
	})

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

	for x := 1; x < 9; x++ {
		if c := getChar(buf, x, 0); c != style.SingleChars.Horizontal {
			t.Errorf("top edge (%d,0) = %q, want %q", x, c, style.SingleChars.Horizontal)
		}
	}

	for x := 1; x < 9; x++ {
		if c := getChar(buf, x, 4); c != style.SingleChars.Horizontal {
			t.Errorf("bottom edge (%d,4) = %q, want %q", x, c, style.SingleChars.Horizontal)
		}
	}

	for y := 1; y < 4; y++ {
		if c := getChar(buf, 0, y); c != style.SingleChars.Vertical {
			t.Errorf("left edge (0,%d) = %q, want %q", y, c, style.SingleChars.Vertical)
		}
	}

	for y := 1; y < 4; y++ {
		if c := getChar(buf, 9, y); c != style.SingleChars.Vertical {
			t.Errorf("right edge (9,%d) = %q, want %q", y, c, style.SingleChars.Vertical)
		}
	}

	// The configured background fills the interior.
	bg := getBg(buf, 5, 2)
	if !bg.Equals(style.ColorBlack) {
		t.Errorf("interior bg = %v, want Black", bg)
	}
}

func TestDrawBorder_AllSides_Double(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		Style:           style.BorderDouble,
		Border:          style.BorderAll(),
		Color:           style.ColorWhite,
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
		Style:  style.BorderRounded,
		Border: style.BorderAll(),
		Color:  style.ColorWhite,
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
		Style:  style.BorderHeavy,
		Border: style.BorderAll(),
		Color:  style.ColorWhite,
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
		Style:  style.BorderSingle,
		Border: style.BorderSides(style.SideTop),
		Color:  style.ColorWhite,
	})

	// Without vertical sides, the top border contains no corner runes.
	for x := range 10 {
		c := getChar(buf, x, 0)
		if c != style.SingleChars.Horizontal {
			t.Errorf("top-only: (%d,0) = %q, want horizontal", x, c)
		}
	}

	// Disabled sides remain untouched.
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
		Style:  style.BorderSingle,
		Border: style.BorderSides(style.SideLeft, style.SideRight),
		Color:  style.ColorWhite,
	})

	// With no horizontal sides, both vertical sides span the full height.
	for y := range 5 {
		c := getChar(buf, 0, y)
		if c != style.SingleChars.Vertical {
			t.Errorf("left edge (0,%d) = %q, want vertical", y, c)
		}
	}

	for y := range 5 {
		c := getChar(buf, 9, y)
		if c != style.SingleChars.Vertical {
			t.Errorf("right edge (9,%d) = %q, want vertical", y, c)
		}
	}

	// Cells between the vertical sides remain untouched.
	if c := getChar(buf, 5, 0); c != ' ' {
		t.Errorf("no top edge expected at (5,0), got %q", c)
	}
}

func TestDrawBorder_NoSides(t *testing.T) {
	buf := New(10, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		Style:  style.BorderSingle,
		Border: style.BorderNoBorder(),
		Color:  style.ColorWhite,
	})

	// With every side disabled, DrawBorder leaves the buffer untouched.
	if c := getChar(buf, 0, 0); c != ' ' {
		t.Errorf("no border: (0,0) should be empty, got %q", c)
	}
}

func TestDrawBorder_WithFill(t *testing.T) {
	buf := New(10, 5)
	bgColor := style.ColorFromInts(128, 128, 128, 255)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 10, Height: 5,
		Style:           style.BorderSingle,
		Border:          style.BorderAll(),
		Color:           style.ColorWhite,
		BackgroundColor: bgColor,
		ShouldFill:      true,
	})

	// The enabled borders leave an interior from (1,1) through (8,3).
	bg := getBg(buf, 5, 2)
	if !bg.Equals(bgColor) {
		t.Errorf("interior bg = %v, want %v", bg, bgColor)
	}
}

func TestDrawBorder_Offset(t *testing.T) {
	buf := New(20, 10)
	buf.DrawBorder(BorderDrawOptions{
		X: 5, Y: 3, Width: 8, Height: 4,
		Style:  style.BorderSingle,
		Border: style.BorderAll(),
		Color:  style.ColorWhite,
	})

	if c := getChar(buf, 5, 3); c != style.SingleChars.TopLeft {
		t.Errorf("offset top-left = %q, want %q", c, style.SingleChars.TopLeft)
	}
	if c := getChar(buf, 12, 6); c != style.SingleChars.BottomRight {
		t.Errorf("offset bottom-right = %q, want %q", c, style.SingleChars.BottomRight)
	}
}

func TestDrawBorder_ZeroDimensions(t *testing.T) {
	buf := New(10, 10)
	// Zero dimensions make DrawBorder a safe no-op.
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 0, Height: 0,
		Style:  style.BorderSingle,
		Border: style.BorderAll(),
		Color:  style.ColorWhite,
	})
}

func TestDrawBorder_TitleLeft(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		Style:          style.BorderSingle,
		Border:         style.BorderAll(),
		Color:          style.ColorWhite,
		Title:          "Test",
		TitleAlignment: layout.TextAlignLeft,
	})

	// A left-aligned title begins immediately after the left corner.
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
		Style:          style.BorderSingle,
		Border:         style.BorderAll(),
		Color:          style.ColorWhite,
		Title:          "Hi",
		TitleAlignment: layout.TextAlignCenter,
	})

	// The 18-column interior centers the two-column title at x=9.
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
		Style:          style.BorderSingle,
		Border:         style.BorderAll(),
		Color:          style.ColorWhite,
		Title:          "OK",
		TitleAlignment: layout.TextAlignRight,
	})

	// The 18-column interior right-aligns the two-column title at x=17.
	if c := getChar(buf, 17, 0); c != 'O' {
		t.Errorf("title right: (17,0) = %q, want 'O'", c)
	}
	if c := getChar(buf, 18, 0); c != 'K' {
		t.Errorf("title right: (18,0) = %q, want 'K'", c)
	}
}

func TestDrawBorder_TitleTruncation(t *testing.T) {
	buf := New(6, 4) // The title has four columns between the corners.
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 6, Height: 4,
		Style:          style.BorderSingle,
		Border:         style.BorderAll(),
		Color:          style.ColorWhite,
		Title:          "TooLong",
		TitleAlignment: layout.TextAlignLeft,
	})

	// Only the first four title runes fit between the corners.
	expected := "TooL"
	for i, ch := range expected {
		c := getChar(buf, 1+i, 0)
		if c != ch {
			t.Errorf("truncated title: (%d,0) = %q, want %q", 1+i, c, ch)
		}
	}
	// The right corner must not be replaced by the truncated title.
	if c := getChar(buf, 5, 0); c == 'o' {
		t.Error("title should have been truncated")
	}
}

func TestDrawBorder_TitleNoTopBorder(t *testing.T) {
	buf := New(20, 5)
	buf.DrawBorder(BorderDrawOptions{
		X: 0, Y: 0, Width: 20, Height: 5,
		Style:          style.BorderSingle,
		Border:         style.BorderSides(style.SideLeft, style.SideRight, style.SideBottom), // The top side is disabled.
		Color:          style.ColorWhite,
		Title:          "Skip",
		TitleAlignment: layout.TextAlignLeft,
	})

	// A title requires an enabled top border.
	if c := getChar(buf, 1, 0); c == 'S' {
		t.Error("title should not be drawn without top border")
	}
}

func TestInnerRect_AllSides(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.BorderAll())
	if r.X != 1 || r.Y != 1 || r.Width != 8 || r.Height != 6 {
		t.Errorf("InnerRect(AllSides) = %+v, want {1 1 8 6}", r)
	}
}

func TestInnerRect_NoSides(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.BorderNoBorder())
	if r.X != 0 || r.Y != 0 || r.Width != 10 || r.Height != 8 {
		t.Errorf("InnerRect(NoSides) = %+v, want {0 0 10 8}", r)
	}
}

func TestInnerRect_TopBottom(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.Border{Top: true, Bottom: true})
	if r.X != 0 || r.Y != 1 || r.Width != 10 || r.Height != 6 {
		t.Errorf("InnerRect(Top,Bottom) = %+v, want {0 1 10 6}", r)
	}
}

func TestInnerRect_LeftRight(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.Border{Left: true, Right: true})
	if r.X != 1 || r.Y != 0 || r.Width != 8 || r.Height != 8 {
		t.Errorf("InnerRect(Left,Right) = %+v, want {1 0 8 8}", r)
	}
}

func TestInnerRect_Offset(t *testing.T) {
	r := InnerRect(5, 3, 10, 8, style.BorderAll())
	if r.X != 6 || r.Y != 4 || r.Width != 8 || r.Height != 6 {
		t.Errorf("InnerRect with offset = %+v, want {6 4 8 6}", r)
	}
}

func TestInnerRect_TooSmall(t *testing.T) {
	// Opposing borders consume both available rows and columns.
	r := InnerRect(0, 0, 2, 2, style.BorderAll())
	if r.Width != 0 || r.Height != 0 {
		t.Errorf("InnerRect too small = %+v, want Width=0 Height=0", r)
	}
}

func TestInnerRect_TooSmallClamps(t *testing.T) {
	// Borders larger than the box clamp the interior dimensions to zero.
	r := InnerRect(0, 0, 1, 1, style.BorderAll())
	if r.Width < 0 || r.Height < 0 {
		t.Errorf("InnerRect should not have negative dimensions: %+v", r)
	}
}

func TestInnerRect_SingleSide(t *testing.T) {
	r := InnerRect(0, 0, 10, 8, style.Border{Top: true})
	if r.X != 0 || r.Y != 1 || r.Width != 10 || r.Height != 7 {
		t.Errorf("InnerRect(Top only) = %+v, want {0 1 10 7}", r)
	}
}

// TestDetachWide_RespectsScissor verifies that repairing a wide pair does not
// modify a cell outside the clipping region, where it could erase an adjacent
// widget's glyph.
//
// A stranded head or continuation is safe because the terminal renderer emits
// it as a space and preserves the row's recorded width.
func TestDetachWide_RespectsScissor(t *testing.T) {
	tests := []struct {
		name    string
		scissor []int // x, y, w, h
		drawAt  int
		checkAt int
	}{
		{
			// The clip contains the continuation at 5 but not its head at 4.
			name:    "head outside the clip",
			scissor: []int{5, 0, 5, 1},
			drawAt:  5,
			checkAt: 4,
		},
		{
			// The clip contains the head at 4 but not its continuation at 5.
			name:    "continuation outside the clip",
			scissor: []int{0, 0, 5, 1},
			drawAt:  4,
			checkAt: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(10, 1)
			b.Clear(style.ColorTransparent)
			b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
			before := b.Cell(tt.checkAt, 0)

			b.PushScissorRect(tt.scissor[0], tt.scissor[1], tt.scissor[2], tt.scissor[3])
			b.SetCell(tt.drawAt, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
			b.PopScissorRect()

			if got := b.Cell(tt.checkAt, 0); got != before {
				t.Errorf("cell %d lies outside the scissor but was modified: Char=%q Width=%d",
					tt.checkAt, got.Char, got.Width)
			}
		})
	}
}

// TestDetachWide_UnclippedStillRepairs verifies that an unrestricted write
// repairs the other half of a wide pair.
func TestDetachWide_UnclippedStillRepairs(t *testing.T) {
	b := New(10, 1)
	b.Clear(style.ColorTransparent)
	b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)

	b.SetCell(4, 0, 'X', style.ColorWhite, style.ColorBlack, 0)

	if b.Cell(5, 0).IsContinuation() {
		t.Error("continuation should have been blanked when nothing clips it")
	}
}
