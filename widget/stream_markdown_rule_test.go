package widget

import (
	"testing"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
)

// TestStreamMarkdown_RuleNoPhantomRow guards against a regression where
// thematic breaks (`---`) inside StreamMarkdown content produced a ghost
// blank row in layout. StreamMarkdown puts each parsed block into its
// own Markdown child; a rule emits zero rendered lines, but the
// Markdown widget's MeasureFunc previously floored the reported height
// to 1, which made every rule allocate a phantom row. After the fix
// the sum of child measure heights matches the standalone Markdown
// line count for the same source, so a previous response containing
// "\n\n---\n\n" no longer leaves blank rows above the debug footer.
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
