package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// ---------------------------------------------------------------------------
// Box with border -> correct characters at edges
// ---------------------------------------------------------------------------

// TestBox_SingleBorder verifies that a Box with a single border renders
// correct box-drawing characters at the corners and edges.
func TestBox_SingleBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 5, 3)

	// Corners
	testutil.AssertCell(t, buf, 0, 0, '┌') // top-left
	testutil.AssertCell(t, buf, 4, 0, '┐') // top-right
	testutil.AssertCell(t, buf, 0, 2, '└') // bottom-left
	testutil.AssertCell(t, buf, 4, 2, '┘') // bottom-right

	// Horizontal edges
	testutil.AssertCell(t, buf, 1, 0, '─')
	testutil.AssertCell(t, buf, 2, 0, '─')
	testutil.AssertCell(t, buf, 3, 0, '─')

	// Vertical edges
	testutil.AssertCell(t, buf, 0, 1, '│')
	testutil.AssertCell(t, buf, 4, 1, '│')
}

// TestBox_RoundedBorder verifies rounded corner characters.
func TestBox_RoundedBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
	)
	buf := testutil.RenderWidget(box, 5, 3)

	testutil.AssertCell(t, buf, 0, 0, '╭') // rounded top-left
	testutil.AssertCell(t, buf, 4, 0, '╮') // rounded top-right
	testutil.AssertCell(t, buf, 0, 2, '╰') // rounded bottom-left
	testutil.AssertCell(t, buf, 4, 2, '╯') // rounded bottom-right
}

// TestBox_BackgroundFill verifies background color fills the interior.
func TestBox_BackgroundFill(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBackgroundColor(style.ColorRed),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 5, 3)

	// Interior cells (1,1) to (3,1) should have red bg
	for x := 1; x <= 3; x++ {
		cell := buf.Cell(x, 1)
		if !cell.Bg.Equals(style.ColorRed) {
			t.Errorf("cell(%d,1): expected red bg, got %v", x, cell.Bg)
		}
	}
}

// TestBox_NoBorder_BackgroundFillsAll verifies that without border,
// background fills the entire area.
func TestBox_NoBorder_BackgroundFillsAll(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBackgroundColor(style.ColorBlue),
		widget.BoxWithShouldFill(true),
	)
	buf := testutil.RenderWidget(box, 4, 2)

	for y := range 2 {
		for x := range 4 {
			cell := buf.Cell(x, y)
			if !cell.Bg.Equals(style.ColorBlue) {
				t.Errorf("cell(%d,%d): expected blue bg, got %v", x, y, cell.Bg)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Box + children layout positioning
// ---------------------------------------------------------------------------

// TestBox_ChildTextPositioning verifies that a Text widget inside a Box
// renders at the correct position based on layout.
func TestBox_ChildTextPositioning(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	txt := widget.NewText("Hello",
		widget.TextWithWidth(layout.Points(20)),
		widget.TextWithHeight(layout.Points(1)),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 20, 3, txt)

	buf := buffer.New(20, 3)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertRow(t, buf, 0, "Hello")
}

// TestBox_MultipleChildrenStack verifies that multiple Text children in a
// column-direction Box render on successive rows.
func TestBox_MultipleChildrenStack(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)

	txt1 := widget.NewText("First",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(0),
	)
	txt2 := widget.NewText("Second",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(0),
	)
	txt3 := widget.NewText("Third",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(0),
	)
	box.Add(txt1)
	box.Add(txt2)
	box.Add(txt3)

	testutil.SetupWidgetTree(box, 10, 3, txt1, txt2, txt3)

	buf := buffer.New(10, 3)
	box.RenderSelf(buf, 0)
	txt1.RenderSelf(buf, 0)
	txt2.RenderSelf(buf, 0)
	txt3.RenderSelf(buf, 0)

	testutil.AssertRow(t, buf, 0, "First")
	testutil.AssertRow(t, buf, 1, "Second")
	testutil.AssertRow(t, buf, 2, "Third")
}

// TestBox_RowDirection verifies children in a row-direction Box render
// side by side horizontally.
func TestBox_RowDirection(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)

	txt1 := widget.NewText("AB",
		widget.TextWithWidth(layout.Points(2)),
		widget.TextWithHeight(layout.Points(1)),
	)
	txt2 := widget.NewText("CD",
		widget.TextWithWidth(layout.Points(2)),
		widget.TextWithHeight(layout.Points(1)),
	)
	box.Add(txt1)
	box.Add(txt2)

	testutil.SetupWidgetTree(box, 10, 1, txt1, txt2)

	buf := buffer.New(10, 1)
	box.RenderSelf(buf, 0)
	txt1.RenderSelf(buf, 0)
	txt2.RenderSelf(buf, 0)

	testutil.AssertCell(t, buf, 0, 0, 'A')
	testutil.AssertCell(t, buf, 1, 0, 'B')
	testutil.AssertCell(t, buf, 2, 0, 'C')
	testutil.AssertCell(t, buf, 3, 0, 'D')
}

// TestBox_BorderWithChild verifies that a child renders inside the border area.
// Note: Box borders are purely visual — you must set padding to offset children
// from the style. This test uses padding(1) to match the 1-cell-wide style.
func TestBox_BorderWithChild(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(12)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithPadding(layout.Points(1)), // offset children from border
	)

	txt := widget.NewText("inner",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(1),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 12, 3, txt)

	buf := buffer.New(12, 3)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	// Border characters
	testutil.AssertCell(t, buf, 0, 0, '┌')
	testutil.AssertCell(t, buf, 11, 0, '┐')

	// Text should be inside the border at y=1, starting at x=1
	testutil.AssertCell(t, buf, 1, 1, 'i')
	testutil.AssertCell(t, buf, 2, 1, 'n')
	testutil.AssertCell(t, buf, 3, 1, 'n')
	testutil.AssertCell(t, buf, 4, 1, 'e')
	testutil.AssertCell(t, buf, 5, 1, 'r')
}

// TestBox_PaddingOffsetsChild verifies that padding pushes children inward.
func TestBox_PaddingOffsetsChild(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithPadding(layout.Points(1)),
	)

	txt := widget.NewText("padded",
		widget.TextWithHeight(layout.Points(1)),
		widget.TextWithFlexGrow(1),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 20, 5, txt)

	buf := buffer.New(20, 5)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	// Nothing at (0,0) — that's the padding area
	cell00 := buf.Cell(0, 0)
	if cell00.Char == 'p' {
		t.Error("text should not render at (0,0) with 1px padding")
	}

	// Text should start at (1,1) due to padding
	testutil.AssertCell(t, buf, 1, 1, 'p')
	testutil.AssertCell(t, buf, 2, 1, 'a')
}

// TestBox_FlexGrow verifies that children with FlexGrow expand to fill space.
func TestBox_FlexGrow(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(3)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)

	// Fixed height child
	fixed := widget.NewText("F",
		widget.TextWithHeight(layout.Points(1)),
	)
	// Flex-grow child
	grow := widget.NewText("G",
		widget.TextWithFlexGrow(1),
	)
	box.Add(fixed)
	box.Add(grow)

	testutil.SetupWidgetTree(box, 10, 3, fixed, grow)

	// Fixed should get 1 row, grow should get 2 rows
	if fixed.Height() != 1 {
		t.Errorf("fixed height: expected 1, got %d", fixed.Height())
	}
	if grow.Height() != 2 {
		t.Errorf("grow height: expected 2, got %d", grow.Height())
	}
}

// ---------------------------------------------------------------------------
// Nested boxes
// ---------------------------------------------------------------------------

// TestBox_NestedBoxes verifies correct rendering with nested boxes.
func TestBox_NestedBoxes(t *testing.T) {
	outer := widget.NewBox(
		widget.BoxWithWidth(layout.Points(20)),
		widget.BoxWithHeight(layout.Points(5)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)

	header := widget.NewBox(
		widget.BoxWithHeight(layout.Points(1)),
		widget.BoxWithBackgroundColor(style.ColorRed),
	)
	body := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithBackgroundColor(style.ColorBlue),
	)

	outer.Add(header)
	outer.Add(body)

	testutil.SetupWidgetTree(outer, 20, 5, header, body)

	buf := buffer.New(20, 5)
	outer.RenderSelf(buf, 0)
	header.RenderSelf(buf, 0)
	body.RenderSelf(buf, 0)

	// Header row (y=0) should have red bg
	c0 := buf.Cell(0, 0)
	if !c0.Bg.Equals(style.ColorRed) {
		t.Errorf("header bg: expected red, got %v", c0.Bg)
	}

	// Body area (y=1..4) should have blue bg
	c1 := buf.Cell(0, 1)
	if !c1.Bg.Equals(style.ColorBlue) {
		t.Errorf("body bg: expected blue, got %v", c1.Bg)
	}

	// Header = 1 row, body = 4 rows (flex grow fills remaining)
	if header.Height() != 1 {
		t.Errorf("header height: expected 1, got %d", header.Height())
	}
	if body.Height() != 4 {
		t.Errorf("body height: expected 4, got %d", body.Height())
	}
}
