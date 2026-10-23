package widget_test

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// rerender draws an already laid-out widget into a fresh buffer.
func rerender(w testutil.SelfRenderer, width, height int) *buffer.Buffer {
	buf := buffer.New(width, height)
	w.RenderSelf(buf, 0)
	return buf
}

// Text scrolling sequences.

// TestGoldenSeq_TextScrollY captures vertical offsets in a shorter viewport.
func TestGoldenSeq_TextScrollY(t *testing.T) {
	txt := widget.NewText("line0\nline1\nline2\nline3\nline4")
	testutil.SetupWidget(txt, 10, 3)

	seq := testutil.NewGoldenSequence()

	// Capture the initial top viewport.
	seq.Add("initial", rerender(txt, 10, 3))

	// Capture a one-row offset.
	txt.SetScrollY(1)
	seq.Add("scroll_y=1", rerender(txt, 10, 3))

	// Capture a two-row offset.
	txt.SetScrollY(2)
	seq.Add("scroll_y=2", rerender(txt, 10, 3))

	testutil.AssertGoldenSequence(t, "text_scroll_y", seq)
}

// TestGoldenSeq_TextScrollX captures horizontal scrolling of a long line.
func TestGoldenSeq_TextScrollX(t *testing.T) {
	txt := widget.NewText("ABCDEFGHIJKLMNO")
	testutil.SetupWidget(txt, 8, 1)

	seq := testutil.NewGoldenSequence()

	// Capture the initial left edge.
	seq.Add("initial", rerender(txt, 8, 1))

	// Capture a three-column offset.
	txt.SetScrollX(3)
	seq.Add("scroll_x=3", rerender(txt, 8, 1))

	// Capture a seven-column offset.
	txt.SetScrollX(7)
	seq.Add("scroll_x=7", rerender(txt, 8, 1))

	testutil.AssertGoldenSequence(t, "text_scroll_x", seq)
}

// Text mutation sequences.

// TestGoldenSeq_TextContentChange captures text content changing over time.
func TestGoldenSeq_TextContentChange(t *testing.T) {
	txt := widget.NewText("Hello")
	testutil.SetupWidget(txt, 20, 1)

	seq := testutil.NewGoldenSequence()

	seq.Add("initial", rerender(txt, 20, 1))

	// Replace the initial plain content.
	txt.SetContent("World")
	seq.Add("after_set_content", rerender(txt, 20, 1))

	// Replace it with styled content.
	txt.SetStyledContent(style.NewStyledText(
		style.FgRed("Red"),
		style.Plain(" "),
		style.FgBlue("Blue"),
	))
	seq.Add("after_styled_content", rerender(txt, 20, 1))

	testutil.AssertGoldenSequence(t, "text_content_change", seq)
}

// TestGoldenSeq_TextNodeMutation captures TextNode tree mutations.
func TestGoldenSeq_TextNodeMutation(t *testing.T) {
	txt := widget.NewText("")
	testutil.SetupWidget(txt, 20, 1)

	seq := testutil.NewGoldenSequence()

	// Capture an empty node tree.
	seq.Add("empty", rerender(txt, 20, 1))

	// Add plain node content.
	txt.AddText("Hello")
	seq.Add("add_text", rerender(txt, 20, 1))

	// Append bold node content.
	txt.AddNode(widget.BoldText(" Bold"))
	seq.AddStyled("add_bold_node", rerender(txt, 20, 1))

	// Replace the complete node tree.
	txt.ClearNodes()
	txt.AddNode(widget.ItalicText("Replaced"))
	seq.AddStyled("clear_and_replace", rerender(txt, 20, 1))

	testutil.AssertGoldenSequence(t, "text_node_mutation", seq)
}

// TestGoldenSeq_TextNodeColorChange captures a node's fg color changing.
func TestGoldenSeq_TextNodeColorChange(t *testing.T) {
	txt := widget.NewText("")
	span := widget.SpanText("color", widget.TextNodeWithFg(style.ColorRed))
	txt.AddNode(span)
	testutil.SetupWidget(txt, 10, 1)

	seq := testutil.NewGoldenSequence()

	// Capture the initial red foreground.
	seq.AddStyled("red", rerender(txt, 10, 1))

	// Change the node foreground to green.
	green := style.ColorGreen
	span.SetFg(&green)
	seq.AddStyled("green", rerender(txt, 10, 1))

	// Change the node foreground to blue.
	blue := style.ColorBlue
	span.SetFg(&blue)
	seq.AddStyled("blue", rerender(txt, 10, 1))

	testutil.AssertGoldenSequence(t, "text_node_color_change", seq)
}

// Input focus and typing sequences.

// TestGoldenSeq_InputFocusUnfocus captures placeholder, text, and focus states.
func TestGoldenSeq_InputFocusUnfocus(t *testing.T) {
	inp := widget.NewInput(
		widget.InputWithPlaceholder("Type here"),
		widget.InputWithPlaceholderColor(style.ColorFromInts(128, 128, 128, 255)),
	)
	testutil.SetupWidget(inp, 15, 1)

	seq := testutil.NewGoldenSequence()

	// Capture the unfocused placeholder.
	seq.AddStyled("unfocused_empty", rerender(inp, 15, 1))

	// Capture the focused placeholder.
	inp.Focus()
	seq.AddStyled("focused_empty", rerender(inp, 15, 1))

	// Capture focused input after typing.
	pressKey(inp, "H")
	pressKey(inp, "i")
	seq.AddStyled("focused_with_text", rerender(inp, 15, 1))

	// Capture the entered text after blur.
	inp.Blur()
	seq.AddStyled("unfocused_with_text", rerender(inp, 15, 1))

	testutil.AssertGoldenSequence(t, "input_focus_unfocus", seq)
}

// TestGoldenSeq_InputTypingSequence captures character-by-character typing.
func TestGoldenSeq_InputTypingSequence(t *testing.T) {
	inp := widget.NewInput()
	testutil.SetupWidget(inp, 10, 1)

	seq := testutil.NewGoldenSequence()

	seq.Add("empty", rerender(inp, 10, 1))

	for _, ch := range "Hello" {
		pressKey(inp, string(ch))
		seq.Add("typed_"+string(ch), rerender(inp, 10, 1))
	}

	// Capture two deletions after the typing sequence.
	pressKey(inp, input.KeyBackspace)
	seq.Add("backspace_1", rerender(inp, 10, 1))

	pressKey(inp, input.KeyBackspace)
	seq.Add("backspace_2", rerender(inp, 10, 1))

	testutil.AssertGoldenSequence(t, "input_typing_sequence", seq)
}

// Wrapping sequences.

// TestGoldenSeq_TextWrapGrowingContent captures wrapping as content grows.
func TestGoldenSeq_TextWrapGrowingContent(t *testing.T) {
	txt := widget.NewText("", widget.TextWithWrapMode(widget.WrapWord))
	testutil.SetupWidget(txt, 10, 4)

	seq := testutil.NewGoldenSequence()

	// Capture content that fits on one row.
	txt.ClearNodes()
	txt.AddText("Hi")
	seq.Add("short", rerender(txt, 10, 4))

	// Capture content that wraps once.
	txt.ClearNodes()
	txt.AddText("Hello World")
	seq.Add("medium", rerender(txt, 10, 4))

	// Capture content that wraps across several rows.
	txt.ClearNodes()
	txt.AddText("The quick brown fox jumps")
	seq.Add("long", rerender(txt, 10, 4))

	testutil.AssertGoldenSequence(t, "text_wrap_growing", seq)
}

// Box focus sequence.

// TestGoldenSeq_BoxFocusBorderColor captures normal and focused border colors.
func TestGoldenSeq_BoxFocusBorderColor(t *testing.T) {
	box := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithBorderColor(style.ColorWhite),
		widget.BoxWithFocusedBorderColor(style.ColorGreen),
		widget.BoxWithWidth(layout.Points(10)),
		widget.BoxWithHeight(layout.Points(3)),
	)
	box.SetFocusable(true)
	testutil.SetupWidget(box, 10, 3)

	seq := testutil.NewGoldenSequence()

	// Capture the normal white border.
	seq.AddStyled("unfocused", rerender(box, 10, 3))

	// Capture the focused green border.
	box.Focus()
	seq.AddStyled("focused", rerender(box, 10, 3))

	// Capture the restored normal border after blur.
	box.Blur()
	seq.AddStyled("blurred", rerender(box, 10, 3))

	testutil.AssertGoldenSequence(t, "box_focus_border_color", seq)
}

// Wrapped-text scrolling.

// TestGoldenSeq_TextWrapThenScroll tests wrapped text that is scrolled.
func TestGoldenSeq_TextWrapThenScroll(t *testing.T) {
	txt := widget.NewText("Line 1 content\nLine 2 content\nLine 3 content\nLine 4 content",
		widget.TextWithWrapMode(widget.WrapWord))
	testutil.SetupWidget(txt, 10, 3)

	seq := testutil.NewGoldenSequence()

	// Capture the first wrapped viewport.
	seq.Add("initial", rerender(txt, 10, 3))

	// Capture the wrapped viewport after scrolling.
	txt.SetScrollY(2)
	seq.Add("scrolled", rerender(txt, 10, 3))

	testutil.AssertGoldenSequence(t, "text_wrap_then_scroll", seq)
}
