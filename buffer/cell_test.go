package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestNewCell_NormalChar(t *testing.T) {
	c := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	if c.Char != 'A' {
		t.Errorf("expected Char='A', got %q", c.Char)
	}
	if c.Width != 1 {
		t.Errorf("expected Width=1, got %d", c.Width)
	}
	if !c.Fg.Equals(style.ColorWhite) {
		t.Errorf("expected Fg=White, got %v", c.Fg)
	}
	if !c.Bg.Equals(style.ColorBlack) {
		t.Errorf("expected Bg=Black, got %v", c.Bg)
	}
}

func TestNewCell_WideChar(t *testing.T) {
	// CJK character (wide)
	c := NewCell('\u4e16', style.ColorWhite, style.ColorBlack, 0) // '世'
	if c.Width != 2 {
		t.Errorf("expected Width=2 for CJK char, got %d", c.Width)
	}
	if !c.IsWide() {
		t.Error("expected IsWide() to be true")
	}
}

func TestNewCell_ZeroChar(t *testing.T) {
	c := NewCell(0, style.ColorWhite, style.ColorBlack, 0)
	if c.Width != 1 {
		t.Errorf("expected Width=1 for zero rune, got %d", c.Width)
	}
}

func TestCell_IsEmpty(t *testing.T) {
	tests := []struct {
		char rune
		want bool
	}{
		{0, true},
		{' ', true},
		{'A', false},
		{'\u4e16', false},
	}
	for _, tt := range tests {
		c := Cell{Char: tt.char, Width: 1}
		if got := c.IsEmpty(); got != tt.want {
			t.Errorf("Cell{Char=%q}.IsEmpty() = %v, want %v", tt.char, got, tt.want)
		}
	}
}

func TestCell_IsContinuation(t *testing.T) {
	c := Cell{Width: -1}
	if !c.IsContinuation() {
		t.Error("expected IsContinuation() to be true for Width=-1")
	}
	c2 := Cell{Width: 1}
	if c2.IsContinuation() {
		t.Error("expected IsContinuation() to be false for Width=1")
	}
}

func TestCell_Equals(t *testing.T) {
	a := NewCell('X', style.ColorRed, style.ColorBlue, style.AttrBold)
	b := NewCell('X', style.ColorRed, style.ColorBlue, style.AttrBold)
	if !a.Equals(b) {
		t.Error("equal cells should be equal")
	}

	c := NewCell('Y', style.ColorRed, style.ColorBlue, style.AttrBold)
	if a.Equals(c) {
		t.Error("cells with different chars should not be equal")
	}

	d := NewCell('X', style.ColorGreen, style.ColorBlue, style.AttrBold)
	if a.Equals(d) {
		t.Error("cells with different fg should not be equal")
	}
}

func TestCell_WithFg(t *testing.T) {
	c := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	c2 := c.WithFg(style.ColorRed)
	if !c2.Fg.Equals(style.ColorRed) {
		t.Errorf("WithFg: expected Red, got %v", c2.Fg)
	}
	// Original unchanged
	if !c.Fg.Equals(style.ColorWhite) {
		t.Error("WithFg should not modify original")
	}
}

func TestCell_WithBg(t *testing.T) {
	c := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	c2 := c.WithBg(style.ColorGreen)
	if !c2.Bg.Equals(style.ColorGreen) {
		t.Errorf("WithBg: expected Green, got %v", c2.Bg)
	}
	if !c.Bg.Equals(style.ColorBlack) {
		t.Error("WithBg should not modify original")
	}
}

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		{'A', 1},
		{'z', 1},
		{' ', 1},
		{'\u4e16', 2}, // CJK '世'
		{'\uff21', 2}, // Fullwidth 'Ａ'
		{0, 1},        // null rune treated as width 1
	}
	for _, tt := range tests {
		if got := RuneWidth(tt.r); got != tt.want {
			t.Errorf("RuneWidth(%q) = %d, want %d", tt.r, got, tt.want)
		}
	}
}

func TestCellClear(t *testing.T) {
	if CellClear.Char != ' ' {
		t.Errorf("CellClear.Char = %q, want ' '", CellClear.Char)
	}
	if CellClear.Width != 1 {
		t.Errorf("CellClear.Width = %d, want 1", CellClear.Width)
	}
	if !CellClear.Fg.Equals(style.ColorWhite) {
		t.Error("CellClear.Fg should be White")
	}
	if !CellClear.Bg.Equals(style.ColorTransparent) {
		t.Error("CellClear.Bg should be Transparent")
	}
}
