package testutil

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Cell assertion helpers
// ---------------------------------------------------------------------------

// AssertCell asserts that the cell at (x, y) has the expected character.
func AssertCell(t *testing.T, buf *buffer.Buffer, x, y int, char rune) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected %q, got %q", x, y, char, cell.Char)
	}
}

// AssertCellFg asserts the cell character and foreground color.
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

// AssertCellStyle asserts the cell character, fg, bg, and attributes.
func AssertCellStyle(t *testing.T, buf *buffer.Buffer, x, y int, char rune, fg, bg style.Color, attrs uint32) {
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

// AssertCellAttrs asserts the cell character and that specific attribute bits are set.
func AssertCellAttrs(t *testing.T, buf *buffer.Buffer, x, y int, char rune, attrs uint32) {
	t.Helper()
	cell := buf.Cell(x, y)
	if cell.Char != char {
		t.Errorf("cell(%d,%d): expected char %q, got %q", x, y, char, cell.Char)
	}
	if cell.Attributes&attrs != attrs {
		t.Errorf("cell(%d,%d): expected attrs %d to be set, got %d", x, y, attrs, cell.Attributes)
	}
}

// ---------------------------------------------------------------------------
// Row / text assertion helpers
// ---------------------------------------------------------------------------

// RowText extracts the plain text characters from row y (0-indexed).
// Trailing spaces are trimmed.
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

// AssertRow asserts that row y contains the expected text (trailing spaces trimmed).
func AssertRow(t *testing.T, buf *buffer.Buffer, y int, expected string) {
	t.Helper()
	got := RowText(buf, y)
	if got != expected {
		t.Errorf("row %d: expected %q, got %q", y, expected, got)
	}
}

// AssertRowContains asserts that row y contains the expected substring.
func AssertRowContains(t *testing.T, buf *buffer.Buffer, y int, substr string) {
	t.Helper()
	got := RowText(buf, y)
	if !strings.Contains(got, substr) {
		t.Errorf("row %d: expected to contain %q, got %q", y, substr, got)
	}
}
