package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestDrawGrid_2x2_FullBorder(t *testing.T) {
	// A 2×2 grid with outer style. Layout:
	// columnOffsets: [0, 4, 8]  → 2 columns
	// rowOffsets:    [0, 3, 6]  → 2 rows
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     true,
		DrawOuter:     true,
	})

	// Top-left corner
	c := b.Cell(0, 0)
	if c.Char != style.SingleChars.TopLeft {
		t.Errorf("(0,0) = %q (%U), want TopLeft %q", c.Char, c.Char, style.SingleChars.TopLeft)
	}

	// Top-right corner
	c = b.Cell(8, 0)
	if c.Char != style.SingleChars.TopRight {
		t.Errorf("(8,0) = %q, want TopRight", c.Char)
	}

	// Bottom-left corner
	c = b.Cell(0, 6)
	if c.Char != style.SingleChars.BottomLeft {
		t.Errorf("(0,6) = %q, want BottomLeft", c.Char)
	}

	// Bottom-right corner
	c = b.Cell(8, 6)
	if c.Char != style.SingleChars.BottomRight {
		t.Errorf("(8,6) = %q, want BottomRight", c.Char)
	}

	// Center cross intersection
	c = b.Cell(4, 3)
	if c.Char != style.SingleChars.Cross {
		t.Errorf("center (4,3) = %q (%U), want Cross %q", c.Char, c.Char, style.SingleChars.Cross)
	}

	// Top T-junction
	c = b.Cell(4, 0)
	if c.Char != style.SingleChars.TopT {
		t.Errorf("top-T (4,0) = %q, want TopT", c.Char)
	}

	// Bottom T-junction
	c = b.Cell(4, 6)
	if c.Char != style.SingleChars.BottomT {
		t.Errorf("bottom-T (4,6) = %q, want BottomT", c.Char)
	}

	// Left T-junction
	c = b.Cell(0, 3)
	if c.Char != style.SingleChars.LeftT {
		t.Errorf("left-T (0,3) = %q, want LeftT", c.Char)
	}

	// Right T-junction
	c = b.Cell(8, 3)
	if c.Char != style.SingleChars.RightT {
		t.Errorf("right-T (8,3) = %q, want RightT", c.Char)
	}

	// Horizontal border (top row)
	c = b.Cell(2, 0)
	if c.Char != style.SingleChars.Horizontal {
		t.Errorf("horizontal (2,0) = %q, want Horizontal", c.Char)
	}

	// Vertical border (left col)
	c = b.Cell(0, 1)
	if c.Char != style.SingleChars.Vertical {
		t.Errorf("vertical (0,1) = %q, want Vertical", c.Char)
	}
}

func TestDrawGrid_InnerOnly(t *testing.T) {
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     true,
		DrawOuter:     false,
	})

	// Outer corners should NOT be drawn
	c := b.Cell(0, 0)
	if c.Char == style.SingleChars.TopLeft {
		t.Error("outer=false: (0,0) should not be TopLeft corner")
	}

	// Inner cross should still be drawn
	c = b.Cell(4, 3)
	if c.Char != style.SingleChars.Cross {
		t.Errorf("inner cross (4,3) = %q, want Cross", c.Char)
	}
}

func TestDrawGrid_OuterOnly(t *testing.T) {
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     false,
		DrawOuter:     true,
	})

	// Outer corner should be drawn
	c := b.Cell(0, 0)
	if c.Char != style.SingleChars.TopLeft {
		t.Errorf("outer=true: (0,0) = %q, want TopLeft", c.Char)
	}

	// Inner cross should NOT be drawn (no inner lines)
	c = b.Cell(4, 3)
	if c.Char == style.SingleChars.Cross {
		t.Error("inner=false: center should not be Cross")
	}
}

func TestDrawGrid_NeitherInnerNorOuter(t *testing.T) {
	b := New(9, 7)
	b.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)

	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     false,
		DrawOuter:     false,
	})

	// Nothing should have changed
	c := b.Cell(0, 0)
	if c.Char != 'X' {
		t.Error("no inner/outer: buffer should be unchanged")
	}
}

func TestDrawGrid_SingleCell(t *testing.T) {
	// 1x1 grid (just a box)
	b := New(5, 5)
	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4},
		RowOffsets:    []int{0, 4},
		DrawInner:     true,
		DrawOuter:     true,
	})

	// Should be a simple box
	c := b.Cell(0, 0)
	if c.Char != style.SingleChars.TopLeft {
		t.Errorf("1x1 grid TL = %q, want TopLeft", c.Char)
	}
	c = b.Cell(4, 4)
	if c.Char != style.SingleChars.BottomRight {
		t.Errorf("1x1 grid BR = %q, want BottomRight", c.Char)
	}
}

func TestDrawGrid_EmptyOffsets(t *testing.T) {
	b := New(5, 5)
	b.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0)

	// columnCount = 0 → should be no-op
	b.DrawGrid(GridDrawOptions{
		Chars:         style.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0},
		RowOffsets:    []int{0, 3},
		DrawInner:     true,
		DrawOuter:     true,
	})

	c := b.Cell(0, 0)
	if c.Char != 'Z' {
		t.Error("empty offsets should be no-op")
	}
}

func TestDrawGrid_DoubleChars(t *testing.T) {
	b := New(5, 5)
	b.DrawGrid(GridDrawOptions{
		Chars:         style.DoubleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4},
		RowOffsets:    []int{0, 4},
		DrawInner:     true,
		DrawOuter:     true,
	})

	c := b.Cell(0, 0)
	if c.Char != style.DoubleChars.TopLeft {
		t.Errorf("double grid TL = %q, want %q", c.Char, style.DoubleChars.TopLeft)
	}
}

func TestGridIntersection(t *testing.T) {
	chars := style.SingleChars

	tests := []struct {
		up, down, left, right bool
		want                  rune
	}{
		{true, true, true, true, chars.Cross},
		{false, true, false, true, chars.TopLeft},
		{false, true, true, false, chars.TopRight},
		{true, false, false, true, chars.BottomLeft},
		{true, false, true, false, chars.BottomRight},
		{true, true, false, true, chars.LeftT},
		{true, true, true, false, chars.RightT},
		{false, true, true, true, chars.TopT},
		{true, false, true, true, chars.BottomT},
		{false, false, true, true, chars.Horizontal},
		{true, true, false, false, chars.Vertical},
	}

	for _, tt := range tests {
		got := gridIntersection(chars, tt.up, tt.down, tt.left, tt.right)
		if got != tt.want {
			t.Errorf("gridIntersection(up=%v,down=%v,left=%v,right=%v) = %q, want %q",
				tt.up, tt.down, tt.left, tt.right, got, tt.want)
		}
	}
}
