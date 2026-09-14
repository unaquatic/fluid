package widget

import (
	"strings"

	"github.com/unaquatic/fluid/textbuf"
)

// displayWidth returns the terminal column width of s.
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

// truncateToWidth returns the longest prefix of s that fits in maxWidth columns.
// It preserves complete grapheme clusters, including multi-codepoint emoji.
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
