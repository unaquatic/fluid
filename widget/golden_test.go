package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// Box border golden tests.

func TestGolden_BoxSingleBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 10, 5)
	testutil.AssertGoldenFrames(t, "box_single_border", buf, nil)
}

func TestGolden_BoxRoundedBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
	)
	buf := testutil.RenderWidget(box, 10, 5)
	testutil.AssertGoldenFrames(t, "box_rounded_border", buf, nil)
}

func TestGolden_BoxDoubleBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderDouble),
	)
	buf := testutil.RenderWidget(box, 10, 5)
	testutil.AssertGoldenFrames(t, "box_double_border", buf, nil)
}

func TestGolden_BoxHeavyBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderHeavy),
	)
	buf := testutil.RenderWidget(box, 10, 5)
	testutil.AssertGoldenFrames(t, "box_heavy_border", buf, nil)
}

func TestGolden_BoxBorderWithTitle(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Title"),
	)
	buf := testutil.RenderWidget(box, 14, 5)
	testutil.AssertGoldenFrames(t, "box_border_with_title", buf, nil)
}

func TestGolden_BoxBorderWithCenteredTitle(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
		widget.BoxWithTitle("Center"),
		widget.BoxWithTitleAlignment(layout.TextAlignCenter),
	)
	buf := testutil.RenderWidget(box, 20, 4)
	testutil.AssertGoldenFrames(t, "box_border_centered_title", buf, nil)
}

func TestGolden_BoxPartialBorder_TopBottom(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderSides(style.SideTop, style.SideBottom)),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 10, 4)
	testutil.AssertGoldenFrames(t, "box_partial_border_top_bottom", buf, nil)
}

// Box background golden tests.

func TestGolden_BoxFilled(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBackgroundColor(style.ColorRed),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 8, 4)
	// Plain output captures border geometry but not cell backgrounds.
	testutil.AssertGoldenFrames(t, "box_filled_with_border", buf, nil)
}

// Text wrapping golden tests.

func TestGolden_TextWrapNone(t *testing.T) {
	txt := widget.NewText("Hello World, this is a long line of text")
	buf := testutil.RenderWidget(txt, 15, 1)
	testutil.AssertGoldenFrames(t, "text_wrap_none", buf, nil)
}

func TestGolden_TextWrapChar(t *testing.T) {
	txt := widget.NewText("ABCDEFGHIJKLMNO", widget.TextWithWrapMode(widget.WrapChar))
	buf := testutil.RenderWidget(txt, 5, 4)
	testutil.AssertGoldenFrames(t, "text_wrap_char", buf, nil)
}

func TestGolden_TextWrapWord(t *testing.T) {
	txt := widget.NewText("The quick brown fox jumps over",
		widget.TextWithWrapMode(widget.WrapWord))
	buf := testutil.RenderWidget(txt, 12, 5)
	testutil.AssertGoldenFrames(t, "text_wrap_word", buf, nil)
}

func TestGolden_TextTruncated(t *testing.T) {
	txt := widget.NewText("Hello World", widget.TextWithTruncate(true))
	buf := testutil.RenderWidget(txt, 8, 1)
	testutil.AssertGoldenFrames(t, "text_truncated", buf, nil)
}

func TestGolden_TextHardLineBreaks(t *testing.T) {
	txt := widget.NewText("Line 1\nLine 2\nLine 3")
	buf := testutil.RenderWidget(txt, 10, 4)
	testutil.AssertGoldenFrames(t, "text_hard_line_breaks", buf, nil)
}

func TestGolden_TextWrapWordLongWord(t *testing.T) {
	txt := widget.NewText("A superlongword here",
		widget.TextWithWrapMode(widget.WrapWord))
	buf := testutil.RenderWidget(txt, 8, 4)
	testutil.AssertGoldenFrames(t, "text_wrap_word_long", buf, nil)
}

// Styled TextNode golden tests.

func TestGolden_StyledBoldText(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("Normal ")
	txt.AddNode(widget.BoldText("Bold"))
	buf := testutil.RenderWidget(txt, 15, 1)
	testutil.AssertGoldenFrames(t, "styled_bold_text", buf, nil)
}

func TestGolden_StyledMixedAttributes(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.BoldText("B"))
	txt.AddNode(widget.ItalicText("I"))
	txt.AddNode(widget.UnderlineText("U"))
	txt.AddText("N")
	buf := testutil.RenderWidget(txt, 10, 1)
	testutil.AssertGoldenFrames(t, "styled_mixed_attrs", buf, nil)
}

func TestGolden_StyledColoredSpans(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.SpanText("Red", widget.TextNodeWithFg(style.ColorRed)))
	txt.AddText(" ")
	txt.AddNode(widget.SpanText("Green", widget.TextNodeWithFg(style.ColorGreen)))
	txt.AddText(" ")
	txt.AddNode(widget.SpanText("Blue", widget.TextNodeWithFg(style.ColorBlue)))
	buf := testutil.RenderWidget(txt, 20, 1)
	testutil.AssertGoldenFrames(t, "styled_colored_spans", buf, nil)
}

func TestGolden_StyledNestedBoldItalic(t *testing.T) {
	txt := widget.NewText("")
	txt.AddText("A")
	bold := widget.NewBoldNode()
	bold.AddString("B")
	italic := widget.NewItalicNode(widget.StringChild("C"))
	bold.AddNode(italic)
	txt.AddNode(bold)
	txt.AddText("D")
	buf := testutil.RenderWidget(txt, 10, 1)
	testutil.AssertGoldenFrames(t, "styled_nested_bold_italic", buf, nil)
}

func TestGolden_StyledTextWithLineBreak(t *testing.T) {
	txt := widget.NewText("")
	txt.AddNode(widget.SpanText("top", widget.TextNodeWithFg(style.ColorRed)))
	txt.AddNode(widget.NewLineBreakNode())
	txt.AddNode(widget.SpanText("bottom", widget.TextNodeWithFg(style.ColorBlue)))
	buf := testutil.RenderWidget(txt, 10, 3)
	testutil.AssertGoldenFrames(t, "styled_text_line_break", buf, nil)
}

// Box margin golden tests.

// TestGolden_BoxMarginRight_RowChildren verifies spacing between row children.
// A four-cell child with a three-cell right margin precedes another four-cell child.
func TestGolden_BoxMarginRight_RowChildren(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)
	childA := widget.NewBox(
		widget.BoxWithWidth(layout.Points(4)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithMarginRight(layout.Points(3)),
		widget.BoxWithBackgroundColor(style.ColorRed),
	)
	txtA := widget.NewText("AAAA",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	childA.Add(txtA)

	childB := widget.NewBox(
		widget.BoxWithWidth(layout.Points(4)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithBackgroundColor(style.ColorBlue),
	)
	txtB := widget.NewText("BBBB",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	childB.Add(txtB)

	container.Add(childA)
	container.Add(childB)

	testutil.SetupWidgetTree(container, 20, 1, childA, txtA, childB, txtB)

	buf := buffer.New(20, 1)
	container.RenderSelf(buf, 0)
	childA.RenderSelf(buf, 0)
	txtA.RenderSelf(buf, 0)
	childB.RenderSelf(buf, 0)
	txtB.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_margin_right_row", buf, nil)
}

// TestGolden_BoxMarginLeft_RowChildren verifies a child's offset from the left edge.
func TestGolden_BoxMarginLeft_RowChildren(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)
	child := widget.NewBox(
		widget.BoxWithWidth(layout.Points(4)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithMarginLeft(layout.Points(3)),
	)
	txt := widget.NewText("CCCC",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	child.Add(txt)
	container.Add(child)

	testutil.SetupWidgetTree(container, 20, 1, child, txt)

	buf := buffer.New(20, 1)
	container.RenderSelf(buf, 0)
	child.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_margin_left_row", buf, nil)
}

// TestGolden_BoxMarginTop_ColumnChildren verifies a two-row top margin.
func TestGolden_BoxMarginTop_ColumnChildren(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	child := widget.NewBox(
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithMarginTop(layout.Points(2)),
	)
	txt := widget.NewText("Hello",
		widget.TextWithHeight(layout.Points(1)),
	)
	child.Add(txt)
	container.Add(child)

	testutil.SetupWidgetTree(container, 10, 5, child, txt)

	buf := buffer.New(10, 5)
	container.RenderSelf(buf, 0)
	child.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_margin_top_column", buf, nil)
}

// TestGolden_BoxMarginBottom_ColumnChildren verifies spacing between column children.
func TestGolden_BoxMarginBottom_ColumnChildren(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	txt2 := widget.NewText("Second",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(0),
	)

	child1 := widget.NewBox(
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexShrink(0),
		widget.BoxWithMarginBottom(layout.Points(2)),
	)
	child1Txt := widget.NewText("First",
		widget.TextWithHeight(layout.Points(1)),
	)
	child1.Add(child1Txt)

	container.Add(child1)
	container.Add(txt2)

	testutil.SetupWidgetTree(container, 10, 5, child1, child1Txt, txt2)

	buf := buffer.New(10, 5)
	container.RenderSelf(buf, 0)
	child1.RenderSelf(buf, 0)
	child1Txt.RenderSelf(buf, 0)
	txt2.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_margin_bottom_column", buf, nil)
}

// TestGolden_BoxMarginRight_ColumnStretch verifies that a right margin reduces stretched width.
func TestGolden_BoxMarginRight_ColumnStretch(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithAlignItems(layout.AlignStretch),
	)
	child := widget.NewBox(
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithMarginRight(layout.Points(5)),
		widget.BoxWithBackgroundColor(style.ColorFromInts(40, 40, 40, 255)),
	)
	txt := widget.NewText("Hello World Here!",
		widget.TextWithHeight(layout.Points(1)),
	)
	child.Add(txt)
	container.Add(child)

	testutil.SetupWidgetTree(container, 20, 3, child, txt)

	buf := buffer.New(20, 3)
	container.RenderSelf(buf, 0)
	child.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_margin_right_column_stretch", buf, nil)
}

// TestGolden_BoxMarginRight_MatchesLayoutNode compares the option with direct layout configuration.
func TestGolden_BoxMarginRight_MatchesLayoutNode(t *testing.T) {
	// Build one tree through the Box option.
	container1 := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)
	childA1 := widget.NewBox(
		widget.BoxWithWidth(layout.Points(4)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithMarginRight(layout.Points(2)),
	)
	txtA1 := widget.NewText("AAAA",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	childA1.Add(txtA1)
	childB1 := widget.NewText("BBBB",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	container1.Add(childA1)
	container1.Add(childB1)
	testutil.SetupWidgetTree(container1, 20, 1, childA1, txtA1, childB1)

	buf1 := buffer.New(20, 1)
	container1.RenderSelf(buf1, 0)
	childA1.RenderSelf(buf1, 0)
	txtA1.RenderSelf(buf1, 0)
	childB1.RenderSelf(buf1, 0)

	// Build the equivalent tree through its LayoutNode.
	container2 := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)
	childA2 := widget.NewBox(
		widget.BoxWithWidth(layout.Points(4)),
		widget.BoxWithHeight(layout.Points(1)),
	)
	childA2.LayoutNode().MarginRight = layout.Points(2) // old way
	txtA2 := widget.NewText("AAAA",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	childA2.Add(txtA2)
	childB2 := widget.NewText("BBBB",
		widget.TextWithWidth(layout.Points(4)),
		widget.TextWithHeight(layout.Points(1)),
	)
	container2.Add(childA2)
	container2.Add(childB2)
	testutil.SetupWidgetTree(container2, 20, 1, childA2, txtA2, childB2)

	buf2 := buffer.New(20, 1)
	container2.RenderSelf(buf2, 0)
	childA2.RenderSelf(buf2, 0)
	txtA2.RenderSelf(buf2, 0)
	childB2.RenderSelf(buf2, 0)

	// Both configuration paths must render identically.
	golden1 := testutil.GoldenText(buf1)
	golden2 := testutil.GoldenText(buf2)
	if golden1 != golden2 {
		t.Errorf("BoxWithMarginRight and LayoutNode().MarginRight produce different results:\n"+
			"--- BoxWithMarginRight ---\n%s\n"+
			"--- LayoutNode().MarginRight ---\n%s",
			golden1, golden2)
	}

	// Their computed child positions must also match.
	if childB1.X() != childB2.X() {
		t.Errorf("childB X position differs: BoxWithMarginRight=%d, LayoutNode=%d",
			childB1.X(), childB2.X())
	}

	testutil.AssertGoldenFrames(t, "box_margin_right_matches_layout_node", buf1, nil)
}

// TestGolden_BoxCombinedMargins verifies independent margins on three sides.
func TestGolden_BoxCombinedMargins(t *testing.T) {
	container := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	child := widget.NewBox(
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexShrink(0),
		widget.BoxWithMarginLeft(layout.Points(2)),
		widget.BoxWithMarginRight(layout.Points(3)),
		widget.BoxWithMarginTop(layout.Points(1)),
		widget.BoxWithBackgroundColor(style.ColorFromInts(60, 60, 60, 255)),
	)
	txt := widget.NewText("Margined",
		widget.TextWithHeight(layout.Points(1)),
	)
	child.Add(txt)
	container.Add(child)

	testutil.SetupWidgetTree(container, 20, 5, child, txt)

	buf := buffer.New(20, 5)
	container.RenderSelf(buf, 0)
	child.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_combined_margins", buf, nil)
}

// Layout composition golden tests.

func TestGolden_BoxWithTextChildren(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(15)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	t1 := widget.NewText("First", widget.TextWithHeight(layout.Points(1)))
	t2 := widget.NewText("Second", widget.TextWithHeight(layout.Points(1)))
	t3 := widget.NewText("Third", widget.TextWithHeight(layout.Points(1)))
	box.Add(t1)
	box.Add(t2)
	box.Add(t3)

	testutil.SetupWidgetTree(box, 15, 3, t1, t2, t3)

	buf := buffer.New(15, 3)
	box.RenderSelf(buf, 0)
	t1.RenderSelf(buf, 0)
	t2.RenderSelf(buf, 0)
	t3.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_text_children_column", buf, nil)
}

func TestGolden_BoxRowWithChildren(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(12)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)
	t1 := widget.NewText("AB",
		widget.TextWithWidth(layout.Points(3)),
		widget.TextWithHeight(layout.Points(1)),
	)
	t2 := widget.NewText("CD",
		widget.TextWithWidth(layout.Points(3)),
		widget.TextWithHeight(layout.Points(1)),
	)
	t3 := widget.NewText("EF",
		widget.TextWithWidth(layout.Points(3)),
		widget.TextWithHeight(layout.Points(1)),
	)
	box.Add(t1)
	box.Add(t2)
	box.Add(t3)

	testutil.SetupWidgetTree(box, 12, 1, t1, t2, t3)

	buf := buffer.New(12, 1)
	box.RenderSelf(buf, 0)
	t1.RenderSelf(buf, 0)
	t2.RenderSelf(buf, 0)
	t3.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_row_children", buf, nil)
}

func TestGolden_BoxBorderPaddingChild(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(16)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
		widget.BoxWithPadding(layout.Points(1)),
	)
	txt := widget.NewText("Content",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(1),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 16, 5, txt)

	buf := buffer.New(16, 5)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "box_border_padding_child", buf, nil)
}

func TestGolden_NestedBoxesWithBorders(t *testing.T) {
	outer := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(7)),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)

	txt := widget.NewText("Hello",
		widget.TextWithHeight(layout.Points(1)),
	)
	inner := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
	)

	outer.Add(txt)
	outer.Add(inner)

	testutil.SetupWidgetTree(outer, 20, 7, txt, inner)

	buf := buffer.New(20, 7)
	outer.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)
	inner.RenderSelf(buf, 0)

	testutil.AssertGoldenFrames(t, "nested_boxes_with_borders", buf, nil)
}

// Input golden tests.

func TestGolden_InputEmpty(t *testing.T) {
	inp := widget.NewInput()
	buf := testutil.RenderWidget(inp, 15, 1)
	testutil.AssertGoldenFrames(t, "input_empty", buf, nil)
}

func TestGolden_InputWithValue(t *testing.T) {
	inp := widget.NewInput(widget.InputWithValue("Hello World"))
	buf := testutil.RenderWidget(inp, 15, 1)
	testutil.AssertGoldenFrames(t, "input_with_value", buf, nil)
}

func TestGolden_InputWithPlaceholder(t *testing.T) {
	inp := widget.NewInput(widget.InputWithPlaceholder("Type here..."))
	buf := testutil.RenderWidget(inp, 15, 1)
	testutil.AssertGoldenFrames(t, "input_placeholder", buf, nil)
}

// CharFill golden tests.

func TestGolden_CharFillBlock(t *testing.T) {
	cf := widget.NewCharFill('#', style.ColorWhite, style.ColorTransparent)
	buf := testutil.RenderWidget(cf, 6, 3)
	testutil.AssertGoldenFrames(t, "charfill_block", buf, nil)
}

func TestGolden_CharFillVLine(t *testing.T) {
	vl := widget.NewVLine('|', style.ColorWhite, style.ColorTransparent)
	buf := testutil.RenderWidget(vl, 1, 5)
	testutil.AssertGoldenFrames(t, "charfill_vline", buf, nil)
}

func TestGolden_CharFillHLine(t *testing.T) {
	hl := widget.NewHLine('-', style.ColorWhite, style.ColorTransparent)
	buf := testutil.RenderWidget(hl, 10, 1)
	testutil.AssertGoldenFrames(t, "charfill_hline", buf, nil)
}

// Gauge golden tests.

func TestGolden_GaugeEmpty(t *testing.T) {
	g := widget.NewGauge(widget.GaugeWithValue(0), widget.GaugeWithShowLabel(false))
	buf := testutil.RenderWidget(g, 20, 1)
	testutil.AssertGoldenFrames(t, "gauge_empty", buf, nil)
}

func TestGolden_GaugeHalf(t *testing.T) {
	g := widget.NewGauge(widget.GaugeWithValue(0.5), widget.GaugeWithShowLabel(false))
	buf := testutil.RenderWidget(g, 20, 1)
	testutil.AssertGoldenFrames(t, "gauge_half", buf, nil)
}

func TestGolden_GaugeFull(t *testing.T) {
	g := widget.NewGauge(widget.GaugeWithValue(1.0), widget.GaugeWithShowLabel(false))
	buf := testutil.RenderWidget(g, 20, 1)
	testutil.AssertGoldenFrames(t, "gauge_full", buf, nil)
}

func TestGolden_GaugeWithLabel(t *testing.T) {
	g := widget.NewGauge(widget.GaugeWithValue(0.75))
	buf := testutil.RenderWidget(g, 20, 1)
	testutil.AssertGoldenFrames(t, "gauge_with_label", buf, nil)
}

// Sparkline golden tests.

func TestGolden_SparklineBasic(t *testing.T) {
	data := []float64{1, 3, 7, 2, 5, 8, 4, 6}
	sp := widget.NewSparkline(data)
	buf := testutil.RenderWidget(sp, 10, 1)
	testutil.AssertGoldenFrames(t, "sparkline_basic", buf, nil)
}

func TestGolden_SparklineTall(t *testing.T) {
	data := []float64{0, 2, 4, 8, 4, 2, 0}
	sp := widget.NewSparkline(data)
	buf := testutil.RenderWidget(sp, 10, 3)
	testutil.AssertGoldenFrames(t, "sparkline_tall", buf, nil)
}

// List golden tests.

func TestGolden_ListBasic(t *testing.T) {
	items := []widget.ListItem{
		{Text: "Apple"},
		{Text: "Banana"},
		{Text: "Cherry"},
	}
	l := widget.NewList(items)
	buf := testutil.RenderWidget(l, 15, 3)
	testutil.AssertGoldenFrames(t, "list_basic", buf, nil)
}

func TestGolden_ListScrolled(t *testing.T) {
	items := []widget.ListItem{
		{Text: "Item 1"},
		{Text: "Item 2"},
		{Text: "Item 3"},
		{Text: "Item 4"},
		{Text: "Item 5"},
	}
	l := widget.NewList(items)
	buf := testutil.RenderWidget(l, 15, 3)
	testutil.AssertGoldenFrames(t, "list_scrolled", buf, nil)
}

// Select golden tests.

func TestGolden_SelectBasic(t *testing.T) {
	options := []widget.SelectOption{
		{Name: "Option A"},
		{Name: "Option B"},
		{Name: "Option C"},
	}
	s := widget.NewSelect(options)
	buf := testutil.RenderWidget(s, 15, 3)
	testutil.AssertGoldenFrames(t, "select_basic", buf, nil)
}

func TestGolden_SelectWithDescription(t *testing.T) {
	options := []widget.SelectOption{
		{Name: "Save", Description: "Save the file"},
		{Name: "Quit", Description: "Exit the app"},
	}
	s := widget.NewSelect(options)
	buf := testutil.RenderWidget(s, 25, 4)
	testutil.AssertGoldenFrames(t, "select_with_description", buf, nil)
}

// TabSelect golden tests.

func TestGolden_TabSelectBasic(t *testing.T) {
	options := []widget.TabSelectOption{
		{Name: "Tab 1"},
		{Name: "Tab 2"},
		{Name: "Tab 3"},
	}
	ts := widget.NewTabSelect(options)
	buf := testutil.RenderWidget(ts, 25, 2)
	testutil.AssertGoldenFrames(t, "tabselect_basic", buf, nil)
}

// Table golden tests.

func TestGolden_TableBasic(t *testing.T) {
	columns := []widget.TableColumn{
		{Name: "Name", Width: 8},
		{Name: "Age", Width: 4},
	}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}
	tbl := widget.NewTable(columns, rows)
	buf := testutil.RenderWidget(tbl, 14, 4)
	testutil.AssertGoldenFrames(t, "table_basic", buf, nil)
}

func TestGolden_TableStriped(t *testing.T) {
	columns := []widget.TableColumn{
		{Name: "Item", Width: 10},
		{Name: "Qty", Width: 5},
	}
	rows := [][]string{
		{"Apples", "5"},
		{"Bananas", "3"},
		{"Cherries", "8"},
	}
	tbl := widget.NewTable(columns, rows, widget.TableWithStripedRows(true))
	buf := testutil.RenderWidget(tbl, 17, 5)
	testutil.AssertGoldenFrames(t, "table_striped", buf, nil)
}

// Tree golden tests.

func TestGolden_TreeBasic(t *testing.T) {
	root := []*widget.TreeNode{
		{Text: "Root", Expanded: true, Children: []*widget.TreeNode{
			{Text: "Child 1"},
			{Text: "Child 2", Expanded: true, Children: []*widget.TreeNode{
				{Text: "Grandchild"},
			}},
		}},
	}
	tr := widget.NewTree(root)
	buf := testutil.RenderWidget(tr, 20, 5)
	testutil.AssertGoldenFrames(t, "tree_basic", buf, nil)
}

func TestGolden_TreeCollapsed(t *testing.T) {
	root := []*widget.TreeNode{
		{Text: "Folder A", Children: []*widget.TreeNode{
			{Text: "Hidden"},
		}},
		{Text: "Folder B"},
	}
	tr := widget.NewTree(root)
	buf := testutil.RenderWidget(tr, 20, 3)
	testutil.AssertGoldenFrames(t, "tree_collapsed", buf, nil)
}

// ScrollBar golden tests.

func TestGolden_ScrollBarVertical(t *testing.T) {
	sb := widget.NewScrollBar()
	sb.SetScrollState(100, 20, 0) // content=100, viewport=20, pos=0
	buf := testutil.RenderWidget(sb, 1, 10)
	testutil.AssertGoldenFrames(t, "scrollbar_vertical_top", buf, nil)
}

func TestGolden_ScrollBarVerticalMiddle(t *testing.T) {
	sb := widget.NewScrollBar()
	sb.SetScrollState(100, 20, 40) // scrolled to middle
	buf := testutil.RenderWidget(sb, 1, 10)
	testutil.AssertGoldenFrames(t, "scrollbar_vertical_middle", buf, nil)
}

func TestGolden_ScrollBarHorizontal(t *testing.T) {
	sb := widget.NewScrollBar(widget.ScrollBarWithOrientation(widget.ScrollBarHorizontal))
	sb.SetScrollState(80, 20, 0)
	buf := testutil.RenderWidget(sb, 10, 1)
	testutil.AssertGoldenFrames(t, "scrollbar_horizontal", buf, nil)
}

// Textarea golden tests.

func TestGolden_TextareaEmpty(t *testing.T) {
	ta := widget.NewTextarea()
	buf := testutil.RenderWidget(ta, 20, 3)
	testutil.AssertGoldenFrames(t, "textarea_empty", buf, nil)
}

func TestGolden_TextareaWithContent(t *testing.T) {
	ta := widget.NewTextarea()
	ta.SetText("Hello\nWorld")
	buf := testutil.RenderWidget(ta, 20, 3)
	testutil.AssertGoldenFrames(t, "textarea_with_content", buf, nil)
}

// Code golden tests.

func TestGolden_CodeBasic(t *testing.T) {
	c := widget.NewCode("func main() {\n\tfmt.Println(\"hello\")\n}",
		widget.CodeWithWrapMode(widget.WrapNone))
	buf := testutil.RenderWidget(c, 30, 4)
	testutil.AssertGoldenFrames(t, "code_basic", buf, nil)
}

// Markdown golden tests.

func TestGolden_MarkdownHeading(t *testing.T) {
	md := widget.NewMarkdown("# Hello\n\nSome text here.")
	buf := testutil.RenderWidget(md, 20, 4)
	testutil.AssertGoldenFrames(t, "markdown_heading", buf, nil)
}

func TestGolden_MarkdownList(t *testing.T) {
	md := widget.NewMarkdown("- Item 1\n- Item 2\n- Item 3")
	buf := testutil.RenderWidget(md, 15, 3)
	testutil.AssertGoldenFrames(t, "markdown_list", buf, nil)
}

// ScrollBox golden tests.

func TestGolden_ScrollBoxWithChild(t *testing.T) {
	txt := widget.NewText("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")
	sbox := widget.NewScrollBox(widget.ScrollBoxWithChildren(txt))

	// Lay out the parent and child before rendering the container.
	testutil.SetupWidgetTree(sbox, 15, 3, txt)
	buf := buffer.New(15, 3)
	txt.RenderSelf(buf, 0)
	testutil.AssertGoldenFrames(t, "scrollbox_with_child", buf, nil)
}
