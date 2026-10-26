package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// Border rendering.

// TestBox_SingleBorder verifies the characters used for each border position.
func TestBox_SingleBorder(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 5, 3)

	// Check all four corners independently.
	testutil.AssertCell(t, buf, 0, 0, '┌') // top-left
	testutil.AssertCell(t, buf, 4, 0, '┐') // top-right
	testutil.AssertCell(t, buf, 0, 2, '└') // bottom-left
	testutil.AssertCell(t, buf, 4, 2, '┘') // bottom-right

	// Check a non-corner cell on each horizontal edge.
	testutil.AssertCell(t, buf, 1, 0, '─')
	testutil.AssertCell(t, buf, 2, 0, '─')
	testutil.AssertCell(t, buf, 3, 0, '─')

	// Check a non-corner cell on each vertical edge.
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
		widget.BoxWithBg(style.ColorRed),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)
	buf := testutil.RenderWidget(box, 5, 3)

	// The border encloses three red background cells.
	for x := 1; x <= 3; x++ {
		cell := buf.Cell(x, 1)
		if !cell.Bg.Equals(style.ColorRed) {
			t.Errorf("cell(%d,1): expected red bg, got %v", x, cell.Bg)
		}
	}
}

// TestBox_NoBorder_BackgroundFillsAll verifies that an unbordered fill covers every cell.
func TestBox_NoBorder_BackgroundFillsAll(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBg(style.ColorBlue),
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

// Child layout.

// TestBox_ChildTextPositioning verifies that layout places child text in its parent.
func TestBox_ChildTextPositioning(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithHeight(layout.Points(3)),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
	)
	txt := widget.NewText("Hello",
		widget.TextWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithHeight(layout.Points(1)),
		),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 20, 3, txt)

	buf := buffer.New(20, 3)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	testutil.AssertRow(t, buf, 0, "Hello")
}

// TestBox_MultipleChildrenStack verifies that a column Box stacks text children.
func TestBox_MultipleChildrenStack(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(10)),
			layout.WithHeight(layout.Points(3)),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
	)

	txt1 := widget.NewText("First",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(0)),
	)
	txt2 := widget.NewText("Second",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(0)),
	)
	txt3 := widget.NewText("Third",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(0)),
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

// TestBox_RowDirection verifies that a row Box places children side by side.
func TestBox_RowDirection(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(10)),
			layout.WithHeight(layout.Points(1)),
			layout.WithFlexDirection(layout.DirectionRow),
		),
	)

	txt1 := widget.NewText("AB",
		widget.TextWithLayout(
			layout.WithWidth(layout.Points(2)),
			layout.WithHeight(layout.Points(1)),
		),
	)
	txt2 := widget.NewText("CD",
		widget.TextWithLayout(
			layout.WithWidth(layout.Points(2)),
			layout.WithHeight(layout.Points(1)),
		),
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

// TestBox_BorderWithChild verifies that border padding keeps child content inside the border.
func TestBox_BorderWithChild(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(12)),
			layout.WithHeight(layout.Points(3)),
			layout.WithPadding(layout.Points(1)), // offset children from border
		),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
	)

	txt := widget.NewText("inner",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(1)),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 12, 3, txt)

	buf := buffer.New(12, 3)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	// The surrounding border must remain intact.
	testutil.AssertCell(t, buf, 0, 0, '┌')
	testutil.AssertCell(t, buf, 11, 0, '┐')

	// One cell of border padding places text at (1,1).
	testutil.AssertCell(t, buf, 1, 1, 'i')
	testutil.AssertCell(t, buf, 2, 1, 'n')
	testutil.AssertCell(t, buf, 3, 1, 'n')
	testutil.AssertCell(t, buf, 4, 1, 'e')
	testutil.AssertCell(t, buf, 5, 1, 'r')
}

// TestBox_PaddingOffsetsChild verifies that padding pushes children inward.
func TestBox_PaddingOffsetsChild(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithHeight(layout.Points(5)),
			layout.WithPadding(layout.Points(1)),
		),
	)

	txt := widget.NewText("padded",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(1)),
	)
	box.Add(txt)

	testutil.SetupWidgetTree(box, 20, 5, txt)

	buf := buffer.New(20, 5)
	box.RenderSelf(buf, 0)
	txt.RenderSelf(buf, 0)

	// Padding leaves the first cell empty.
	cell00 := buf.Cell(0, 0)
	if cell00.Char == 'p' {
		t.Error("text should not render at (0,0) with 1px padding")
	}

	// Content starts after one cell of padding on each axis.
	testutil.AssertCell(t, buf, 1, 1, 'p')
	testutil.AssertCell(t, buf, 2, 1, 'a')
}

// TestBox_FlexGrow verifies that children with FlexGrow expand to fill space.
func TestBox_FlexGrow(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(10)),
			layout.WithHeight(layout.Points(3)),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
	)

	// The first child reserves one row.
	fixed := widget.NewText("F",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
	)
	// The second child receives the remaining rows.
	grow := widget.NewText("G",
		widget.TextWithLayout(layout.WithFlexGrow(1)),
	)
	box.Add(fixed)
	box.Add(grow)

	testutil.SetupWidgetTree(box, 10, 3, fixed, grow)

	// The three-row parent allocates one fixed row and two flexible rows.
	if fixed.Height() != 1 {
		t.Errorf("fixed height: expected 1, got %d", fixed.Height())
	}
	if grow.Height() != 2 {
		t.Errorf("grow height: expected 2, got %d", grow.Height())
	}
}

// Nested boxes.

// TestBox_NestedBoxes verifies correct rendering with nested boxes.
func TestBox_NestedBoxes(t *testing.T) {
	outer := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithHeight(layout.Points(5)),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
	)

	header := widget.NewBox(
		widget.BoxWithLayout(layout.WithHeight(layout.Points(1))),
		widget.BoxWithBg(style.ColorRed),
	)
	body := widget.NewBox(
		widget.BoxWithLayout(layout.WithFlexGrow(1)),
		widget.BoxWithBg(style.ColorBlue),
	)

	outer.Add(header)
	outer.Add(body)

	testutil.SetupWidgetTree(outer, 20, 5, header, body)

	buf := buffer.New(20, 5)
	outer.RenderSelf(buf, 0)
	header.RenderSelf(buf, 0)
	body.RenderSelf(buf, 0)

	// The fixed header occupies the first row.
	c0 := buf.Cell(0, 0)
	if !c0.Bg.Equals(style.ColorRed) {
		t.Errorf("header bg: expected red, got %v", c0.Bg)
	}

	// The flexible body fills the remaining four rows.
	c1 := buf.Cell(0, 1)
	if !c1.Bg.Equals(style.ColorBlue) {
		t.Errorf("body bg: expected blue, got %v", c1.Bg)
	}

	// Computed heights match the fixed and remaining allocations.
	if header.Height() != 1 {
		t.Errorf("header height: expected 1, got %d", header.Height())
	}
	if body.Height() != 4 {
		t.Errorf("body height: expected 4, got %d", body.Height())
	}
}
