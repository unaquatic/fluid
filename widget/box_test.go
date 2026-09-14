package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

func TestNewBox_Defaults(t *testing.T) {
	b := NewBox()
	if b == nil {
		t.Fatal("NewBox returned nil")
	}
	if b.BackgroundColor != style.ColorTransparent {
		t.Error("expected transparent background by default")
	}
	if b.BorderStyle != style.BorderNone {
		t.Error("expected no border style by default")
	}
	if b.ShouldFill {
		t.Error("expected ShouldFill false by default")
	}
	if b.LayoutNode().FlexDirection != layout.DirectionColumn {
		t.Error("expected DirectionColumn by default for Box")
	}
}

func TestNewBox_WithOptions(t *testing.T) {
	b := NewBox(
		BoxWithID("test-box"),
		BoxWithBackgroundColor(style.ColorRed),
		BoxWithBorderStyle(style.BorderRounded),
		BoxWithBorder(style.BorderAll()),
		BoxWithBorderColor(style.ColorBlue),
		BoxWithTitle("My Box"),
		BoxWithTitleAlignment(layout.TextAlignCenter),
		BoxWithFocusedBorderColor(style.ColorGreen),
		BoxWithGap(layout.Points(2)),
		BoxWithWidth(layout.Points(20)),
		BoxWithHeight(layout.Points(10)),
	)

	if b.ID() != "test-box" {
		t.Errorf("expected id 'test-box', got %q", b.ID())
	}
	if !b.BackgroundColor.Equals(style.ColorRed) {
		t.Error("expected red background")
	}
	if b.BorderStyle != style.BorderRounded {
		t.Error("expected rounded border")
	}
	if !b.Border.All {
		t.Error("expected all borders")
	}
	if !b.BorderColor.Equals(style.ColorBlue) {
		t.Error("expected blue border color")
	}
	if b.Title != "My Box" {
		t.Errorf("expected title 'My Box', got %q", b.Title)
	}
	if b.TitleAlignment != layout.TextAlignCenter {
		t.Error("expected center title alignment")
	}
	if !b.ShouldFill {
		t.Error("expected ShouldFill true when background color is set")
	}
}

func TestBox_RenderSelf_FillBackground(t *testing.T) {
	b := NewBox(
		BoxWithBackgroundColor(style.ColorRed),
		BoxWithShouldFill(true),
	)
	// RenderSelf needs a computed layout rectangle.
	testutil.SetupWidget(b, 5, 3)

	buf := buffer.New(5, 3)
	b.RenderSelf(buf, 0)

	// An unbordered fill covers the complete rectangle.
	for y := range 3 {
		for x := range 5 {
			cell := buf.Cell(x, y)
			if !cell.Bg.Equals(style.ColorRed) {
				t.Errorf("cell (%d,%d) bg expected red, got %v", x, y, cell.Bg)
			}
		}
	}
}

func TestBox_RenderSelf_WithBorder(t *testing.T) {
	b := NewBox(
		BoxWithBorder(style.BorderAll()),
		BoxWithBorderStyle(style.BorderSingle),
		BoxWithBorderColor(style.ColorWhite),
	)
	testutil.SetupWidget(b, 5, 3)

	buf := buffer.New(5, 3)
	b.RenderSelf(buf, 0)

	// Verify each corner character.
	topLeft := buf.Cell(0, 0)
	if topLeft.Char != '┌' {
		t.Errorf("expected '┌' at (0,0), got %c", topLeft.Char)
	}
	topRight := buf.Cell(4, 0)
	if topRight.Char != '┐' {
		t.Errorf("expected '┐' at (4,0), got %c", topRight.Char)
	}
	bottomLeft := buf.Cell(0, 2)
	if bottomLeft.Char != '└' {
		t.Errorf("expected '└' at (0,2), got %c", bottomLeft.Char)
	}
	bottomRight := buf.Cell(4, 2)
	if bottomRight.Char != '┘' {
		t.Errorf("expected '┘' at (4,2), got %c", bottomRight.Char)
	}
}

func TestBox_InnerRect(t *testing.T) {
	b := NewBox(
		BoxWithBorder(style.BorderAll()),
		BoxWithBorderStyle(style.BorderSingle),
	)
	testutil.SetupWidget(b, 10, 5)

	inner := b.InnerRect()
	if inner.X != 1 || inner.Y != 1 || inner.Width != 8 || inner.Height != 3 {
		t.Errorf("expected inner rect {1,1,8,3}, got {%d,%d,%d,%d}", inner.X, inner.Y, inner.Width, inner.Height)
	}
}

func TestBox_ZeroSize(t *testing.T) {
	b := NewBox()
	buf := buffer.New(0, 0)
	// A zero-sized box must render safely.
	b.RenderSelf(buf, 0)
}

// Children option behavior.

func TestBox_Children_AddsChildren(t *testing.T) {
	txt1 := NewText("hello")
	txt2 := NewText("world")

	b := NewBox(
		Children(txt1, txt2),
	)

	if b.ChildrenCount() != 2 {
		t.Fatalf("expected 2 children, got %d", b.ChildrenCount())
	}
	kids := b.Children()
	if kids[0].ID() != txt1.ID() {
		t.Errorf("expected first child %q, got %q", txt1.ID(), kids[0].ID())
	}
	if kids[1].ID() != txt2.ID() {
		t.Errorf("expected second child %q, got %q", txt2.ID(), kids[1].ID())
	}
}

func TestBox_Children_SetsParent(t *testing.T) {
	txt := NewText("child")
	b := NewBox(Children(txt))

	if txt.Parent() == nil {
		t.Fatal("expected child to have parent set")
	}
	if txt.Parent().ID() != b.ID() {
		t.Errorf("expected parent %q, got %q", b.ID(), txt.Parent().ID())
	}
}

func TestBox_Children_SkipsNil(t *testing.T) {
	txt := NewText("valid")
	b := NewBox(Children(nil, txt, nil))

	if b.ChildrenCount() != 1 {
		t.Fatalf("expected 1 child (nils skipped), got %d", b.ChildrenCount())
	}
}

func TestBox_Children_Empty(t *testing.T) {
	b := NewBox(Children())
	if b.ChildrenCount() != 0 {
		t.Errorf("expected 0 children, got %d", b.ChildrenCount())
	}
}

func TestBox_Children_Nested(t *testing.T) {
	inner := NewBox(
		BoxWithID("inner"),
		Children(
			NewText("leaf"),
		),
	)
	outer := NewBox(
		BoxWithID("outer"),
		Children(inner),
	)

	if outer.ChildrenCount() != 1 {
		t.Fatalf("expected 1 child on outer, got %d", outer.ChildrenCount())
	}
	if inner.ChildrenCount() != 1 {
		t.Fatalf("expected 1 child on inner, got %d", inner.ChildrenCount())
	}
	if inner.Parent().ID() != "outer" {
		t.Errorf("expected inner parent 'outer', got %q", inner.Parent().ID())
	}
}

func TestBox_Children_WithOtherOptions(t *testing.T) {
	txt := NewText("hello")
	b := NewBox(
		BoxWithID("combo"),
		BoxWithBackgroundColor(style.ColorRed),
		BoxWithFlexDirection(layout.DirectionRow),
		Children(txt),
	)

	if b.ID() != "combo" {
		t.Errorf("expected id 'combo', got %q", b.ID())
	}
	if !b.BackgroundColor.Equals(style.ColorRed) {
		t.Error("expected red background")
	}
	if b.ChildrenCount() != 1 {
		t.Errorf("expected 1 child, got %d", b.ChildrenCount())
	}
}

func TestBox_Children_RendersCorrectly(t *testing.T) {
	txt := NewText("Hi",
		TextWithWidth(layout.Points(10)),
		TextWithHeight(layout.Points(1)),
	)
	b := NewBox(
		BoxWithWidth(layout.Points(10)),
		BoxWithHeight(layout.Points(1)),
		Children(txt),
	)

	testutil.SetupWidgetTree(b, 10, 1, txt)

	buf := buffer.New(10, 1)
	txt.RenderSelf(buf, 0)

	cell := buf.Cell(0, 0)
	if cell.Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %c", cell.Char)
	}
}
