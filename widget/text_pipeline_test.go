// These external-package tests verify the public Text-to-buffer rendering pipeline.
package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// Text and TextNode rendering.

// TestTextPipeline_PlainString verifies plain Text content in the output buffer.
func TestTextPipeline_PlainString(t *testing.T) {
	buf := testutil.RenderWidget(widget.NewText("Hello"), 10, 1)
	testutil.AssertRow(t, buf, 0, "Hello")
}

// TestTextPipeline_StyledText verifies chunk foregrounds through the full pipeline.
func TestTextPipeline_StyledText(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("R"),
		style.FgBlue("B"),
	)
	buf := testutil.RenderWidget(widget.NewStyledText(st), 10, 1)

	testutil.AssertCellFg(t, buf, 0, 0, 'R', style.ColorRed)
	testutil.AssertCellFg(t, buf, 1, 0, 'B', style.ColorBlue)
}

// TestTextPipeline_BoldNode verifies that a bold node sets the buffer attribute.
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

// TestTextPipeline_MixedNodes verifies ordered text with independent attributes.
func TestTextPipeline_MixedNodes(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("A")
	txt.AddNode(widget.BoldText("B"))
	txt.AddNode(widget.ItalicText("C"))

	buf := testutil.RenderWidget(txt, 10, 1)

	// A remains plain.
	testutil.AssertCellStyle(t, buf, 0, 0, 'A', style.ColorWhite, style.ColorTransparent, 0)
	// B inherits bold.
	testutil.AssertCellAttrs(t, buf, 1, 0, 'B', style.AttrBold)
	// C inherits italic.
	testutil.AssertCellAttrs(t, buf, 2, 0, 'C', style.AttrItalic)
}

// TestTextPipeline_NestedBoldItalic verifies additive attributes in nested nodes.
func TestTextPipeline_NestedBoldItalic(t *testing.T) {
	txt := widget.NewText("")

	bold := widget.NewBoldNode()
	italic := widget.NewItalicNode(widget.StringChild("X"))
	bold.AddNode(italic)
	txt.AddNode(bold)

	buf := testutil.RenderWidget(txt, 10, 1)

	testutil.AssertCellAttrs(t, buf, 0, 0, 'X', style.AttrBold|style.AttrItalic)
}

// TestTextPipeline_ColoredSpans verifies independent foregrounds on adjacent spans.
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

// TestTextPipeline_LineBreakNode verifies that a line-break node starts a new row.
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
	// Truncation ends with the ellipsis glyph.
	lastCell := buf.Cell(5, 0)
	if lastCell.Char != '…' {
		t.Errorf("expected ellipsis at position 5, got %q (row: %q)", lastCell.Char, row)
	}
}

// TestTextPipeline_MultiChunkWrapPreservesStyle verifies styles across wrapped chunks.
func TestTextPipeline_MultiChunkWrapPreservesStyle(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("AAAA"),
		style.FgBlue("BBBB"),
	)
	txt := widget.NewStyledText(st, widget.TextWithWrapMode(widget.WrapChar))

	// At width five, the blue chunk starts on the first row and continues on the second.
	buf := testutil.RenderWidget(txt, 5, 3)

	// The first row contains four red cells and one blue cell.
	for x := range 4 {
		testutil.AssertCellFg(t, buf, x, 0, 'A', style.ColorRed)
	}
	testutil.AssertCellFg(t, buf, 4, 0, 'B', style.ColorBlue)

	// The second row contains the remaining blue cells.
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

// TextNode mutation and rerendering.

// TestTextMutation_AddNode verifies that an added node appears after rerendering.
func TestTextMutation_AddNode(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("Hello")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "Hello")

	// Add content after the initial render.
	txt.AddText(" World")

	// Render the dirty tree again.
	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "Hello World")
}

// TestTextMutation_ClearAndRebuild verifies replacement after clearing all nodes.
func TestTextMutation_ClearAndRebuild(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("original")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "original")

	// Replace the complete node tree.
	txt.ClearNodes()
	txt.AddNode(widget.BoldText("replaced"))

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "replaced")
	testutil.AssertCellAttrs(t, buf2, 0, 0, 'r', style.AttrBold)
}

// TestTextMutation_ModifyChildNode verifies style changes after the initial render.
func TestTextMutation_ModifyChildNode(t *testing.T) {
	txt := widget.NewText("")
	span := widget.SpanText("hello", widget.TextNodeWithFg(style.ColorRed))
	txt.AddNode(span)

	buf := testutil.RenderWidget(txt, 10, 1)
	testutil.AssertCellFg(t, buf, 0, 0, 'h', style.ColorRed)

	// Mutate the child foreground before rerendering.
	blue := style.ColorBlue
	span.SetFg(&blue)

	buf2 := buffer.New(10, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertCellFg(t, buf2, 0, 0, 'h', style.ColorBlue)
}

// TestTextMutation_ReplaceChildren verifies replacement through the public node API.
func TestTextMutation_ReplaceChildren(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("before")

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "before")

	// Replace all root children in one mutation.
	txt.ClearNodes()
	txt.AddText("after")

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "after")
}

// TestTextMutation_SetContent_OverridesNodes verifies direct-content precedence.
func TestTextMutation_SetContent_OverridesNodes(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.BoldText("from-nodes"))

	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertRow(t, buf, 0, "from-nodes")

	// Switch from node-backed content to direct content.
	txt.SetContent("manual")

	buf2 := buffer.New(20, 1)
	txt.RenderSelf(buf2, 0)
	testutil.AssertRow(t, buf2, 0, "manual")
}
