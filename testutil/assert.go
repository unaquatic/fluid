package testutil

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// Cell assertions.

// AssertCell reports a test error unless the cell at (x, y) contains char.
func AssertCell(t *testing.T, buf *buffer.Buffer, x, y int, char rune) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected %q, got %q", x, y, char, cell.Char)
	}
}

// AssertCellFg reports test errors unless the cell at (x, y) has char and fg.
func AssertCellFg(t *testing.T, buf *buffer.Buffer, x, y int, char rune, fg style.Color) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected char %q, got %q", x, y, char, cell.Char)
	}
	if !cell.Fg.Equals(fg) {
		t.Errorf("cell(%d,%d): expected fg %v, got %v", x, y, fg, cell.Fg)
	}
}

// AssertCellStyle reports test errors unless the cell at (x, y) matches
// char, fg, bg, and the exact attribute bitmask.
func AssertCellStyle(t *testing.T, buf *buffer.Buffer, x, y int, char rune, fg, bg style.Color, attrs style.Attrs) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected char %q, got %q", x, y, char, cell.Char)
	}
	if !cell.Fg.Equals(fg) {
		t.Errorf("cell(%d,%d): expected fg %v, got %v", x, y, fg, cell.Fg)
	}
	if !cell.Bg.Equals(bg) {
		t.Errorf("cell(%d,%d): expected bg %v, got %v", x, y, bg, cell.Bg)
	}
	if cell.Attributes != attrs {
		t.Errorf("cell(%d,%d): expected attrs %d, got %d", x, y, attrs, cell.Attributes)
	}
}

// AssertCellAttrs reports test errors unless the cell at (x, y) contains char
// and every bit in attrs. Additional attribute bits are allowed.
func AssertCellAttrs(t *testing.T, buf *buffer.Buffer, x, y int, char rune, attrs style.Attrs) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected char %q, got %q", x, y, char, cell.Char)
	}
	if cell.Attributes&attrs != attrs {
		t.Errorf("cell(%d,%d): expected attrs %d to be set, got %d", x, y, attrs, cell.Attributes)
	}
}

// Row text and assertions.

// RowText returns row y as plain text with trailing spaces removed.
// It treats zero-value cells as spaces.
func RowText(buf *buffer.Buffer, y int) string {
	var sb strings.Builder
	w := buf.Width()
	for x := range w {
		cell := buf.Cell(x, y)
		if cell.Char == 0 {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(cell.Char)
		}
	}
	return strings.TrimRight(sb.String(), " ")
}

// AssertRow reports a test error unless RowText for y equals expected.
func AssertRow(t *testing.T, buf *buffer.Buffer, y int, expected string) {
	t.Helper()
	got := RowText(buf, y)
	if got != expected {
		t.Errorf("row %d: expected %q, got %q", y, expected, got)
	}
}

// AssertRowContains reports a test error unless RowText for y contains substr.
func AssertRowContains(t *testing.T, buf *buffer.Buffer, y int, substr string) {
	t.Helper()
	got := RowText(buf, y)
	if !strings.Contains(got, substr) {
		t.Errorf("row %d: expected to contain %q, got %q", y, substr, got)
	}
}
