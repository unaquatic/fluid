package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/textbuf"
)

// Text test helpers.

func setupText(content string, w, h int, opts ...TextOption) *Text {
	txt := NewText(content, opts...)
	testutil.SetupWidget(txt, w, h)
	return txt
}

func setupStyledText(content style.StyledText, w, h int, opts ...TextOption) *Text {
	txt := NewStyledText(content, opts...)
	testutil.SetupWidget(txt, w, h)
	return txt
}

// Core Text behavior.

func TestNewText_String(t *testing.T) {
	txt := NewText("Hello, World!")
	if txt.PlainText() != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", txt.PlainText())
	}
}

func TestNewText_StyledText(t *testing.T) {
	st := style.NewStyledText(style.FgRed("red"), style.Plain(" and "), style.FgBlue("blue"))
	txt := NewStyledText(st)
	if txt.PlainText() != "red and blue" {
		t.Errorf("expected 'red and blue', got %q", txt.PlainText())
	}
}

func TestNewText_Nil(t *testing.T) {
	txt := NewText("")
	if txt.PlainText() != "" {
		t.Errorf("expected empty, got %q", txt.PlainText())
	}
	if txt.hasManualContent {
		t.Error("expected hasManualContent=false for nil")
	}
}

func TestText_SetContent(t *testing.T) {
	txt := NewText("initial")
	txt.SetContent("updated")
	if txt.PlainText() != "updated" {
		t.Errorf("expected 'updated', got %q", txt.PlainText())
	}
}

func TestText_WrapNone(t *testing.T) {
	txt := setupText("Hello World", 5, 1, TextWithWrapMode(textbuf.WrapNone))

	if txt.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", txt.LineCount())
	}
}

func TestText_WrapChar(t *testing.T) {
	txt := setupText("HelloWorld", 5, 3, TextWithWrapMode(textbuf.WrapChar))

	if txt.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", txt.LineCount())
	}
}

func TestText_WrapWord(t *testing.T) {
	txt := setupText("Hello World Foo", 7, 5, TextWithWrapMode(textbuf.WrapWord))

	if txt.LineCount() < 2 {
		t.Errorf("expected at least 2 lines, got %d", txt.LineCount())
	}
}

func TestText_RenderSelf(t *testing.T) {
	txt := setupText("Hi", 5, 1, TextWithFg(style.ColorRed))

	buf := buffer.New(5, 1)
	txt.RenderSelf(buf, 0)

	cell0 := buf.Cell(0, 0)
	if cell0.Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %c", cell0.Char)
	}
	cell1 := buf.Cell(1, 0)
	if cell1.Char != 'i' {
		t.Errorf("expected 'i' at (1,0), got %c", cell1.Char)
	}
}

func TestText_EmptyContent(t *testing.T) {
	txt := setupText("", 5, 1)

	buf := buffer.New(5, 1)
	txt.RenderSelf(buf, 0)

	if txt.LineCount() != 0 {
		t.Errorf("expected 0 lines for empty content, got %d", txt.LineCount())
	}
}

func TestText_Multiline(t *testing.T) {
	txt := setupText("line1\nline2\nline3", 10, 5)

	if txt.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", txt.LineCount())
	}
}

func TestText_Options(t *testing.T) {
	txt := NewText("test",
		TextWithID("my-text"),
		TextWithFg(style.ColorRed),
		TextWithBg(style.ColorBlue),
		TextWithAttributes(style.AttrBold),
		TextWithWrapMode(textbuf.WrapWord),
		TextWithTruncate(true),
		TextWithSelectable(true),
		TextWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithHeight(layout.Points(5)),
			layout.WithFlexGrow(1),
		),
	)

	if txt.ID() != "my-text" {
		t.Errorf("expected id 'my-text', got %q", txt.ID())
	}
	if !txt.Selectable() {
		t.Error("expected selectable true")
	}
}

// Multi-chunk styled rendering.

func TestText_MultiChunkRendering(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("red"),
		style.Plain(" "),
		style.FgBlue("blue"),
	)
	txt := setupStyledText(st, 20, 1)

	buf := buffer.New(20, 1)
	txt.RenderSelf(buf, 0)

	// The first chunk uses red.
	cell0 := buf.Cell(0, 0)
	if cell0.Char != 'r' {
		t.Errorf("expected 'r' at (0,0), got %c", cell0.Char)
	}
	if cell0.Fg != style.ColorRed {
		t.Errorf("expected Red fg for 'r', got %v", cell0.Fg)
	}

	// The second chunk uses blue.
	cell4 := buf.Cell(4, 0)
	if cell4.Char != 'b' {
		t.Errorf("expected 'b' at (4,0), got %c", cell4.Char)
	}
	if cell4.Fg != style.ColorBlue {
		t.Errorf("expected Blue fg for 'b', got %v", cell4.Fg)
	}
}

func TestText_MultiChunkMultiline(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("line1\nline2"),
	)
	txt := setupStyledText(st, 20, 5)

	if txt.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", txt.LineCount())
	}

	buf := buffer.New(20, 5)
	txt.RenderSelf(buf, 0)

	c := buf.Cell(0, 0)
	if c.Fg != style.ColorRed {
		t.Errorf("line 1 should be Red, got %v", c.Fg)
	}
	c = buf.Cell(0, 1)
	if c.Fg != style.ColorRed {
		t.Errorf("line 2 should be Red, got %v", c.Fg)
	}
}

func TestText_MultiChunkWrapChar(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("AABB"),
		style.FgBlue("CCDD"),
	)
	txt := setupStyledText(st, 4, 4, TextWithWrapMode(textbuf.WrapChar))

	if txt.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", txt.LineCount())
	}

	buf := buffer.New(4, 4)
	txt.RenderSelf(buf, 0)

	if buf.Cell(0, 0).Fg != style.ColorRed {
		t.Error("expected Red at (0,0)")
	}
	if buf.Cell(0, 1).Fg != style.ColorBlue {
		t.Error("expected Blue at (0,1)")
	}
}

func TestText_MultiChunkWrapWord(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("Hello "),
		style.FgBlue("World"),
	)
	txt := setupStyledText(st, 6, 4, TextWithWrapMode(textbuf.WrapWord))

	if txt.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", txt.LineCount())
	}

	buf := buffer.New(6, 4)
	txt.RenderSelf(buf, 0)

	if buf.Cell(0, 0).Fg != style.ColorRed {
		t.Error("expected Red at (0,0)")
	}
	if buf.Cell(0, 1).Fg != style.ColorBlue {
		t.Error("expected Blue at (0,1)")
	}
}

func TestText_AttributesMerge(t *testing.T) {
	st := style.NewStyledText(style.TextItalic("hi"))
	txt := setupStyledText(st, 10, 1, TextWithAttributes(style.AttrBold))

	buf := buffer.New(10, 1)
	txt.RenderSelf(buf, 0)

	cell := buf.Cell(0, 0)
	if cell.Attributes&style.AttrBold == 0 {
		t.Error("expected bold attribute")
	}
	if cell.Attributes&style.AttrItalic == 0 {
		t.Error("expected italic attribute")
	}
}

// Truncation.

func TestText_Truncate(t *testing.T) {
	txt := setupText("Hello World", 6, 1, TextWithTruncate(true))

	if txt.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", txt.LineCount())
	}

	buf := buffer.New(6, 1)
	txt.RenderSelf(buf, 0)

	cell5 := buf.Cell(5, 0)
	if cell5.Char != '…' {
		t.Errorf("expected ellipsis at (5,0), got %c (%d)", cell5.Char, cell5.Char)
	}
}

func TestText_TruncateShortText(t *testing.T) {
	txt := setupText("Hi", 10, 1, TextWithTruncate(true))

	buf := buffer.New(10, 1)
	txt.RenderSelf(buf, 0)

	if buf.Cell(0, 0).Char != 'H' {
		t.Error("expected 'H' at (0,0)")
	}
	if buf.Cell(1, 0).Char != 'i' {
		t.Error("expected 'i' at (1,0)")
	}
}

func TestText_TruncateMultiChunk(t *testing.T) {
	st := style.NewStyledText(
		style.FgRed("Red"),
		style.FgBlue("Blue"),
	)
	txt := setupStyledText(st, 5, 1, TextWithTruncate(true))

	buf := buffer.New(5, 1)
	txt.RenderSelf(buf, 0)

	cell4 := buf.Cell(4, 0)
	if cell4.Char != '…' {
		t.Errorf("expected ellipsis at (4,0), got %c", cell4.Char)
	}
}

// Scrolling.

func TestText_ScrollY(t *testing.T) {
	txt := setupText("line1\nline2\nline3\nline4", 10, 2)
	txt.SetScrollY(1)

	buf := buffer.New(10, 2)
	txt.RenderSelf(buf, 0)

	cell := buf.Cell(4, 0)
	if cell.Char != '2' {
		t.Errorf("expected '2' at (4,0), got %c", cell.Char)
	}
}

func TestText_ScrollX(t *testing.T) {
	txt := setupText("Hello World", 5, 1)
	txt.SetScrollX(6)

	buf := buffer.New(5, 1)
	txt.RenderSelf(buf, 0)

	cell := buf.Cell(0, 0)
	if cell.Char != 'W' {
		t.Errorf("expected 'W' at (0,0), got %c", cell.Char)
	}
}

func TestText_ScrollClamp(t *testing.T) {
	txt := setupText("AB", 10, 1)

	txt.SetScrollX(100)
	if txt.ScrollX() != 0 {
		t.Errorf("scrollX should be 0, got %d", txt.ScrollX())
	}

	txt.SetScrollY(-5)
	if txt.ScrollY() != 0 {
		t.Errorf("scrollY should be 0, got %d", txt.ScrollY())
	}
}

func TestText_MaxScroll(t *testing.T) {
	txt := setupText("line1\nline2\nline3\nline4\nline5", 10, 2)

	maxY := txt.MaxScrollY()
	if maxY != 3 {
		t.Errorf("expected maxScrollY=3, got %d", maxY)
	}
}

func TestText_ScrollWidth(t *testing.T) {
	txt := setupText("Hello World!", 5, 1)

	sw := txt.ScrollWidth()
	if sw != 12 {
		t.Errorf("expected scrollWidth=12, got %d", sw)
	}
}

// Intrinsic measurement.

func TestText_MeasureIntrinsic_NoWrap(t *testing.T) {
	txt := NewText("Hello World")
	m := txt.textSurface.MeasureForDimensions(0, 0)
	if m == nil {
		t.Fatal("expected non-nil measure result")
	}
	if m.MaxWidth != 11 {
		t.Errorf("expected width=11, got %d", m.MaxWidth)
	}
	if m.LineCount != 1 {
		t.Errorf("expected height=1, got %d", m.LineCount)
	}
}

func TestText_MeasureIntrinsic_Multiline(t *testing.T) {
	txt := NewText("Hello\nWorld!")
	m := txt.textSurface.MeasureForDimensions(0, 0)
	if m == nil {
		t.Fatal("expected non-nil measure result")
	}
	if m.MaxWidth != 6 {
		t.Errorf("expected width=6, got %d", m.MaxWidth)
	}
	if m.LineCount != 2 {
		t.Errorf("expected height=2, got %d", m.LineCount)
	}
}

func TestText_MeasureIntrinsic_Wrapped(t *testing.T) {
	txt := NewText("Hello World", TextWithWrapMode(textbuf.WrapWord))
	m := txt.textSurface.MeasureForDimensions(6, 0)
	if m == nil {
		t.Fatal("expected non-nil measure result")
	}
	if m.LineCount < 2 {
		t.Errorf("expected height>=2, got %d", m.LineCount)
	}
	// MaxWidth describes source width; the layout engine applies the constraint.
}

// Configuration setters.

func TestText_Setters(t *testing.T) {
	txt := NewText("test")

	txt.SetFg(style.ColorRed)
	if txt.Fg() != style.ColorRed {
		t.Error("SetFg failed")
	}

	txt.SetBg(style.ColorBlue)
	if txt.Bg() != style.ColorBlue {
		t.Error("SetBg failed")
	}

	txt.SetAttributes(style.AttrBold)
	if txt.Attributes() != style.AttrBold {
		t.Error("SetAttributes failed")
	}

	txt.SetWrapMode(textbuf.WrapWord)
	if txt.WrapModeValue() != textbuf.WrapWord {
		t.Error("SetWrapMode failed")
	}

	txt.SetTruncate(true)
	if !txt.Truncate() {
		t.Error("SetTruncate failed")
	}
}

// TextNode integration.

func TestText_WithTextNodes(t *testing.T) {
	txt := NewText("")
	txt.AddText("Hello ")
	txt.AddNode(NewBoldNode(StringChild("world")))

	testutil.SetupWidget(txt, 20, 1)

	buf := buffer.New(20, 1)
	txt.RenderSelf(buf, 0)

	cell0 := buf.Cell(0, 0)
	if cell0.Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %c", cell0.Char)
	}
	cell6 := buf.Cell(6, 0)
	if cell6.Char != 'w' {
		t.Errorf("expected 'w' at (6,0), got %c", cell6.Char)
	}
}

func TestText_ManualContentOverridesNodes(t *testing.T) {
	txt := NewText("manual content")
	txt.AddNode(NewBoldNode(StringChild("node content")))

	if txt.PlainText() != "manual content" {
		t.Errorf("expected 'manual content', got %q", txt.PlainText())
	}
}

func TestText_NodeChildrenProducePlainText(t *testing.T) {
	txt := NewText("")
	txt.AddText("Hello ")
	txt.AddNode(NewBoldNode(StringChild("world")))

	testutil.SetupWidget(txt, 20, 1)

	_ = txt.LineCount()

	if txt.PlainText() != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", txt.PlainText())
	}
}

func TestText_NodeRendersStyles(t *testing.T) {
	txt := NewText("")
	txt.AddNode(SpanText("R", TextNodeWithFg(style.ColorRed)))
	txt.AddNode(SpanText("B", TextNodeWithFg(style.ColorBlue)))

	testutil.SetupWidget(txt, 10, 1)

	buf := buffer.New(10, 1)
	txt.RenderSelf(buf, 0)

	c0 := buf.Cell(0, 0)
	if c0.Char != 'R' || c0.Fg != style.ColorRed {
		t.Errorf("expected Red 'R' at (0,0), got char=%c fg=%v", c0.Char, c0.Fg)
	}

	c1 := buf.Cell(1, 0)
	if c1.Char != 'B' || c1.Fg != style.ColorBlue {
		t.Errorf("expected Blue 'B' at (1,0), got char=%c fg=%v", c1.Char, c1.Fg)
	}
}
