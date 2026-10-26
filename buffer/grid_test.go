package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

func TestDrawGrid_2x2_FullBorder(t *testing.T) {
	// Three boundaries along each axis define a two-column, two-row grid.
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     true,
		DrawOuter:     true,
	})

	c := b.Cell(0, 0)
	if c.Char != symbol.SingleChars.TopLeft {
		t.Errorf("(0,0) = %q (%U), want TopLeft %q", c.Char, c.Char, symbol.SingleChars.TopLeft)
	}

	c = b.Cell(8, 0)
	if c.Char != symbol.SingleChars.TopRight {
		t.Errorf("(8,0) = %q, want TopRight", c.Char)
	}

	c = b.Cell(0, 6)
	if c.Char != symbol.SingleChars.BottomLeft {
		t.Errorf("(0,6) = %q, want BottomLeft", c.Char)
	}

	c = b.Cell(8, 6)
	if c.Char != symbol.SingleChars.BottomRight {
		t.Errorf("(8,6) = %q, want BottomRight", c.Char)
	}

	c = b.Cell(4, 3)
	if c.Char != symbol.SingleChars.Cross {
		t.Errorf("center (4,3) = %q (%U), want Cross %q", c.Char, c.Char, symbol.SingleChars.Cross)
	}

	c = b.Cell(4, 0)
	if c.Char != symbol.SingleChars.TopT {
		t.Errorf("top-T (4,0) = %q, want TopT", c.Char)
	}

	c = b.Cell(4, 6)
	if c.Char != symbol.SingleChars.BottomT {
		t.Errorf("bottom-T (4,6) = %q, want BottomT", c.Char)
	}

	c = b.Cell(0, 3)
	if c.Char != symbol.SingleChars.LeftT {
		t.Errorf("left-T (0,3) = %q, want LeftT", c.Char)
	}

	c = b.Cell(8, 3)
	if c.Char != symbol.SingleChars.RightT {
		t.Errorf("right-T (8,3) = %q, want RightT", c.Char)
	}

	c = b.Cell(2, 0)
	if c.Char != symbol.SingleChars.Horizontal {
		t.Errorf("horizontal (2,0) = %q, want Horizontal", c.Char)
	}

	c = b.Cell(0, 1)
	if c.Char != symbol.SingleChars.Vertical {
		t.Errorf("vertical (0,1) = %q, want Vertical", c.Char)
	}
}

func TestDrawGrid_InnerOnly(t *testing.T) {
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     true,
		DrawOuter:     false,
	})

	// Disabling the outer border leaves its corner cells unchanged.
	c := b.Cell(0, 0)
	if c.Char == symbol.SingleChars.TopLeft {
		t.Error("outer=false: (0,0) should not be TopLeft corner")
	}

	// The enabled inner lines still meet at the center.
	c = b.Cell(4, 3)
	if c.Char != symbol.SingleChars.Cross {
		t.Errorf("inner cross (4,3) = %q, want Cross", c.Char)
	}
}

func TestDrawGrid_OuterOnly(t *testing.T) {
	b := New(9, 7)
	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     false,
		DrawOuter:     true,
	})

	// The enabled outer border draws its corner.
	c := b.Cell(0, 0)
	if c.Char != symbol.SingleChars.TopLeft {
		t.Errorf("outer=true: (0,0) = %q, want TopLeft", c.Char)
	}

	// Disabling inner lines leaves the center unchanged.
	c = b.Cell(4, 3)
	if c.Char == symbol.SingleChars.Cross {
		t.Error("inner=false: center should not be Cross")
	}
}

func TestDrawGrid_NeitherInnerNorOuter(t *testing.T) {
	b := New(9, 7)
	b.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)

	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4, 8},
		RowOffsets:    []int{0, 3, 6},
		DrawInner:     false,
		DrawOuter:     false,
	})

	// With both options disabled, DrawGrid leaves the buffer unchanged.
	c := b.Cell(0, 0)
	if c.Char != 'X' {
		t.Error("no inner/outer: buffer should be unchanged")
	}
}

func TestDrawGrid_SingleCell(t *testing.T) {
	// A grid with one row and one column is a box.
	b := New(5, 5)
	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4},
		RowOffsets:    []int{0, 4},
		DrawInner:     true,
		DrawOuter:     true,
	})

	c := b.Cell(0, 0)
	if c.Char != symbol.SingleChars.TopLeft {
		t.Errorf("1x1 grid TL = %q, want TopLeft", c.Char)
	}
	c = b.Cell(4, 4)
	if c.Char != symbol.SingleChars.BottomRight {
		t.Errorf("1x1 grid BR = %q, want BottomRight", c.Char)
	}
}

func TestDrawGrid_EmptyOffsets(t *testing.T) {
	b := New(5, 5)
	b.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0)

	// One column boundary defines no columns, so drawing is a no-op.
	b.DrawGrid(GridDrawOptions{
		Chars:         symbol.SingleChars,
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
		Chars:         symbol.DoubleChars,
		BorderFg:      style.ColorWhite,
		BorderBg:      style.ColorBlack,
		ColumnOffsets: []int{0, 4},
		RowOffsets:    []int{0, 4},
		DrawInner:     true,
		DrawOuter:     true,
	})

	c := b.Cell(0, 0)
	if c.Char != symbol.DoubleChars.TopLeft {
		t.Errorf("double grid TL = %q, want %q", c.Char, symbol.DoubleChars.TopLeft)
	}
}

func TestGridIntersection(t *testing.T) {
	chars := symbol.SingleChars

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
