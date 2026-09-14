package widget

import (
	"testing"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
)

// TestStreamMarkdown_RuleNoPhantomRow verifies that a zero-height thematic-break
// child does not add a row. Streamed child heights must sum to the same height
// as a standalone Markdown widget for the same source.
func TestStreamMarkdown_RuleNoPhantomRow(t *testing.T) {
	const src = "before\n\n---\n\nafter"

	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent(src)

	measureTotal := 0
	for _, child := range sm.children {
		mr := child.LayoutNode().Measure(80, layout.MeasureAtMost, 0, layout.MeasureUndefined)
		measureTotal += mr.Height
	}

	md := NewMarkdown(src, MarkdownWithFg(style.ColorWhite))
	want := md.LineCount()

	if measureTotal != want {
		t.Errorf("StreamMarkdown layout height = %d, want %d (matches standalone Markdown.LineCount); a thematic break is leaking a phantom row", measureTotal, want)
	}
}
