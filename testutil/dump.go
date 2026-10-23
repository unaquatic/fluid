package testutil

import (
	"fmt"
	"strings"

	"github.com/unaquatic/fluid/buffer"
)

// DumpBuffer returns a multi-line string representation of the buffer,
// useful for debug output in test failures.
func DumpBuffer(buf *buffer.Buffer) string {
	var sb strings.Builder
	w := buf.Width()
	h := buf.Height()
	sb.WriteString(fmt.Sprintf("Buffer %dx%d:\n", w, h))
	for y := range h {
		sb.WriteString(fmt.Sprintf("  %2d: |", y))
		for x := range w {
			cell := buf.Cell(x, y)
			if cell.Char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cell.Char)
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}
