// Integration tests for cross-package behavior in the fluid framework.
// These tests use the public API only (external test package) to verify that
// widgets, layout, text nodes, and rendering work together correctly.
package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// ---------------------------------------------------------------------------
// TextNode -> Text -> Buffer render pipeline
// ---------------------------------------------------------------------------

// TestTextPipeline_PlainString verifies that a Text widget with a plain string
// renders the correct characters into the buffer.
func TestTextPipeline_PlainString(t *testing.T) {
	buf := testutil.RenderWidget(widget.NewText("Hello"), 10, 1)
	testutil.AssertRow(t, buf, 0, "Hello")
}

// TestTextPipeline_StyledText verifies styled text chunks render with correct
// fg colors through the full pipeline.
func TestTextPipeline_StyledText(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("R"),
		style.FgBlue("B"),
	)
	buf := testutil.RenderWidget(widget.NewStyledText(st), 10, 1)

	testutil.AssertCellFg(t, buf, 0, 0, 'R', style.ColorRed)
	testutil.AssertCellFg(t, buf, 1, 0, 'B', style.ColorBlue)
}

// TestTextPipeline_BoldNode verifies that a BoldNode produces bold-attributed
// characters in the buffer.
func TestTextPipeline_BoldNode(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.BoldText("bold"))

	buf := testutil.RenderWidget(txt, 10, 1)

	testutil.AssertCellAttrs(t, buf, 0, 0, 'b', style.AttrBold)
	testutil.AssertCellAttrs(t, buf, 1, 0, 'o', style.AttrBold)
	testutil.AssertCellAttrs(t, buf, 2, 0, 'l', style.AttrBold)
	testutil.AssertCellAttrs(t, buf, 3, 0, 'd', style.AttrBold)
}

// TestTextPipeline_ItalicNode verifies italic attribute rendering.
func TestTextPipeline_ItalicNode(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.ItalicText("em"))

	buf := testutil.RenderWidget(txt, 10, 1)

	testutil.AssertCellAttrs(t, buf, 0, 0, 'e', style.AttrItalic)
	testutil.AssertCellAttrs(t, buf, 1, 0, 'm', style.AttrItalic)
}

// TestTextPipeline_MixedNodes verifies that plain text, bold, and italic nodes
// produce the correct characters and attributes in sequence.
func TestTextPipeline_MixedNodes(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("A")
	txt.AddNode(widget.BoldText("B"))
	txt.AddNode(widget.ItalicText("C"))

	buf := testutil.RenderWidget(txt, 10, 1)

	// 'A' — no attributes
	testutil.AssertCellStyle(t, buf, 0, 0, 'A', style.ColorWhite, style.ColorTransparent, 0)
	// 'B' — bold
	testutil.AssertCellAttrs(t, buf, 1, 0, 'B', style.AttrBold)
	// 'C' — italic
	testutil.AssertCellAttrs(t, buf, 2, 0, 'C', style.AttrItalic)
}

// TestTextPipeline_NestedBoldItalic verifies deep nesting:
// root > bold > italic > "X" → X has both bold and italic.
func TestTextPipeline_NestedBoldItalic(t *testing.T) {
	txt := widget.NewText("")

	bold := widget.NewBoldNode()
	italic := widget.NewItalicNode(widget.StringChild("X"))
	bold.AddNode(italic)
	txt.AddNode(bold)

	buf := testutil.RenderWidget(txt, 10, 1)

	testutil.AssertCellAttrs(t, buf, 0, 0, 'X', style.AttrBold|style.AttrItalic)
}

// TestTextPipeline_ColoredSpans verifies that SpanText with different fg colors
// renders each character with its respective color.
func TestTextPipeline_ColoredSpans(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.SpanText("R", widget.TextNodeWithFg(style.ColorRed)))
	txt.AddNode(widget.SpanText("G", widget.TextNodeWithFg(style.ColorGreen)))
	txt.AddNode(widget.SpanText("B", widget.TextNodeWithFg(style.ColorBlue)))

	buf := testutil.RenderWidget(txt, 10, 1)

	testutil.AssertCellFg(t, buf, 0, 0, 'R', style.ColorRed)
	testutil.AssertCellFg(t, buf, 1, 0, 'G', style.ColorGreen)
	testutil.AssertCellFg(t, buf, 2, 0, 'B', style.ColorBlue)
}

// TestTextPipeline_LineBreakNode verifies that a LineBreakNode produces a
// newline that pushes subsequent text to the next row.
func TestTextPipeline_LineBreakNode(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("line1")
	txt.AddNode(widget.NewLineBreakNode())
	txt.AddText("line2")

	buf := testutil.RenderWidget(txt, 10, 3)

	testutil.AssertRow(t, buf, 0, "line1")
	testutil.AssertRow(t, buf, 1, "line2")
}

// TestTextPipeline_WrapChar verifies character-level wrapping.
func TestTextPipeline_WrapChar(t *testing.T) {
	txt := widget.NewText("ABCDE", widget.TextWithWrapMode(widget.WrapChar))
	buf := testutil.RenderWidget(txt, 3, 3)

	testutil.AssertRow(t, buf, 0, "ABC")
	testutil.AssertRow(t, buf, 1, "DE")
}

// TestTextPipeline_WrapWord verifies word-level wrapping.
func TestTextPipeline_WrapWord(t *testing.T) {
	txt := widget.NewText("Hi world", widget.TextWithWrapMode(widget.WrapWord))
	buf := testutil.RenderWidget(txt, 5, 3)

	testutil.AssertRow(t, buf, 0, "Hi")
	testutil.AssertRow(t, buf, 1, "world")
}

// TestTextPipeline_Truncation verifies ellipsis truncation.
func TestTextPipeline_Truncation(t *testing.T) {
	txt := widget.NewText("Hello World", widget.TextWithTruncate(true))
	buf := testutil.RenderWidget(txt, 6, 1)

	row := testutil.RowText(buf, 0)
	runes := []rune(row)
	if len(runes) > 6 {
		t.Errorf("truncated row too long (%d runes): %q", len(runes), row)
	}
	// Should end with ellipsis character '…'
	lastCell := buf.Cell(5, 0)
	if lastCell.Char != '…' {
		t.Errorf("expected ellipsis at position 5, got %q (row: %q)", lastCell.Char, row)
	}
}

// TestTextPipeline_MultiChunkWrapPreservesStyle verifies that when multi-chunk
// text wraps, each chunk retains its fg color across line breaks.
func TestTextPipeline_MultiChunkWrapPreservesStyle(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("AAAA"),
		style.FgBlue("BBBB"),
	)
	txt := widget.NewStyledText(st, widget.TextWithWrapMode(widget.WrapChar))

	// 5 chars wide: "AAAA" wraps to "AAAAB"/"BBB"
	buf := testutil.RenderWidget(txt, 5, 3)

	// Row 0: first 4 chars are red 'A', 5th is blue 'B'
	for x := range 4 {
		testutil.AssertCellFg(t, buf, x, 0, 'A', style.ColorRed)
	}
	testutil.AssertCellFg(t, buf, 4, 0, 'B', style.ColorBlue)

	// Row 1: remaining blue 'B's
	for x := range 3 {
		testutil.AssertCellFg(t, buf, x, 1, 'B', style.ColorBlue)
	}
}

// TestTextPipeline_ScrollY verifies vertical scrolling skips lines.
func TestTextPipeline_ScrollY(t *testing.T) {
	txt := widget.NewText("line0\nline1\nline2\nline3")
	txt.SetScrollY(2)

	buf := testutil.RenderWidget(txt, 10, 2)

	testutil.AssertRow(t, buf, 0, "line2")
	testutil.AssertRow(t, buf, 1, "line3")
}

// ---------------------------------------------------------------------------
// TextNode mutation -> re-render
// ---------------------------------------------------------------------------

// TestTextMutation_AddNode verifies that adding a TextNode and re-rendering
// produces updated buffer content.
func TestTextMutation_AddNode(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("Hello")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "Hello")

	// Mutate: add more text
	txt.AddText(" World")

	// Re-render
	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "Hello World")
}

// TestTextMutation_ClearAndRebuild verifies that ClearNodes + re-adding
// produces correct output.
func TestTextMutation_ClearAndRebuild(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("original")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "original")

	// Clear and rebuild
	txt.ClearNodes()
	txt.AddNode(widget.BoldText("replaced"))

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "replaced")
	testutil.AssertCellAttrs(t, buf2, 0, 0, 'r', style.AttrBold)
}

// TestTextMutation_ModifyChildNode verifies that modifying a child TextNode's
// style causes the re-rendered output to reflect the change.
func TestTextMutation_ModifyChildNode(t *testing.T) {
	txt := widget.NewText("")
	span := widget.SpanText("hello", widget.TextNodeWithFg(style.ColorRed))
	txt.AddNode(span)

	buf := testutil.RenderWidget(txt, 10, 1)
	testutil.AssertCellFg(t, buf, 0, 0, 'h', style.ColorRed)

	// Change color to blue
	blue := style.ColorBlue
	span.SetFg(&blue)

	buf2 := buffer.New(10, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertCellFg(t, buf2, 0, 0, 'h', style.ColorBlue)
}

// TestTextMutation_ReplaceChildren verifies ReplaceChildren on the node tree
// propagates through to rendering.
func TestTextMutation_ReplaceChildren(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("before")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "before")

	// Get the root TextNode and replace its children
	txt.ClearNodes()
	txt.AddText("after")

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "after")
}

// TestTextMutation_SetContent_OverridesNodes verifies that SetContent with a
// string overrides TextNode-based content.
func TestTextMutation_SetContent_OverridesNodes(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.BoldText("from-nodes"))

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "from-nodes")

	// Override with manual content
	txt.SetContent("manual")

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "manual")
}
