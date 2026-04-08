package widget

import (
	"strings"

	"github.com/unaquatic/fluid/textbuf"
)

// displayWidth returns the display column width of a string.
func displayWidth(s string) int {
	width := 0
	remaining := s
	for len(remaining) > 0 {
		_, rest, rw := textbuf.NextGraphemeCluster(remaining)
		remaining = rest
		width += rw
	}
	return width
}

// truncateToWidth truncates a string to fit within maxWidth display columns.
// Uses grapheme cluster iteration so multi-codepoint emoji are kept intact.
func truncateToWidth(s string, maxWidth int) string {
	var sb strings.Builder
	width := 0
	remaining := s
	for len(remaining) > 0 {
		cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
		remaining = rest
		if width+rw > maxWidth {
			break
		}
		sb.WriteString(cluster)
		width += rw
	}
	return sb.String()
}
