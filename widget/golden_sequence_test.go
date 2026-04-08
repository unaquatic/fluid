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

// rerender renders a widget to a fresh buffer of the same size.
func rerender(w testutil.SelfRenderer, width, height int) *buffer.Buffer {
	buf := buffer.New(width, height)
	w.RenderSelf(buf, 0)
	return buf
}

// ---------------------------------------------------------------------------
// Text scroll: before and after
// ---------------------------------------------------------------------------

// TestGoldenSeq_TextScrollY captures the text widget before and after
// vertical scrolling through 5 lines of content in a 3-line viewport.
func TestGoldenSeq_TextScrollY(t *testing.T) {
	txt := widget.NewText("line0\nline1\nline2\nline3\nline4")
	testutil.SetupWidget(txt, 10, 3)

	seq := testutil.NewGoldenSequence()

	// Step 1: initial (no scroll)
	seq.Add("initial", rerender(txt, 10, 3))

	// Step 2: scroll down by 1
	txt.SetScrollY(1)
	seq.Add("scroll_y=1", rerender(txt, 10, 3))

	// Step 3: scroll down by 2
	txt.SetScrollY(2)
	seq.Add("scroll_y=2", rerender(txt, 10, 3))

	testutil.AssertGoldenSequence(t, "text_scroll_y", seq)
}

// TestGoldenSeq_TextScrollX captures horizontal scrolling of a long line.
func TestGoldenSeq_TextScrollX(t *testing.T) {
	txt := widget.NewText("ABCDEFGHIJKLMNO")
	testutil.SetupWidget(txt, 8, 1)

	seq := testutil.NewGoldenSequence()

	// Step 1: initial
	seq.Add("initial", rerender(txt, 8, 1))

	// Step 2: scroll right by 3
	txt.SetScrollX(3)
	seq.Add("scroll_x=3", rerender(txt, 8, 1))

	// Step 3: scroll right by 7
	txt.SetScrollX(7)
	seq.Add("scroll_x=7", rerender(txt, 8, 1))

	testutil.AssertGoldenSequence(t, "text_scroll_x", seq)
}

// ---------------------------------------------------------------------------
// Text content mutation: before and after
// ---------------------------------------------------------------------------

// TestGoldenSeq_TextContentChange captures text content changing over time.
func TestGoldenSeq_TextContentChange(t *testing.T) {
	txt := widget.NewText("Hello")
	testutil.SetupWidget(txt, 20, 1)

	seq := testutil.NewGoldenSequence()

	seq.Add("initial", rerender(txt, 20, 1))

	// Change content
	txt.SetContent("World")
	seq.Add("after_set_content", rerender(txt, 20, 1))

	// Change again with styled text
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

	// Step 1: empty
	seq.Add("empty", rerender(txt, 20, 1))

	// Step 2: add plain text
	txt.AddText("Hello")
	seq.Add("add_text", rerender(txt, 20, 1))

	// Step 3: add bold node
	txt.AddNode(widget.BoldText(" Bold"))
	seq.AddStyled("add_bold_node", rerender(txt, 20, 1))

	// Step 4: clear and replace
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

	// Step 1: red
	seq.AddStyled("red", rerender(txt, 10, 1))

	// Step 2: change to green
	green := style.ColorGreen
	span.SetFg(&green)
	seq.AddStyled("green", rerender(txt, 10, 1))

	// Step 3: change to blue
	blue := style.ColorBlue
	span.SetFg(&blue)
	seq.AddStyled("blue", rerender(txt, 10, 1))

	testutil.AssertGoldenSequence(t, "text_node_color_change", seq)
}

// ---------------------------------------------------------------------------
// Input: focus / unfocus state changes
// ---------------------------------------------------------------------------

// TestGoldenSeq_InputFocusUnfocus captures the Input widget rendering
// in unfocused and focused states.
func TestGoldenSeq_InputFocusUnfocus(t *testing.T) {
	inp := widget.NewInput(
		widget.InputWithPlaceholder("Type here"),
		widget.InputWithPlaceholderColor(style.ColorFromInts(128, 128, 128, 255)),
	)
	testutil.SetupWidget(inp, 15, 1)

	seq := testutil.NewGoldenSequence()

	// Step 1: unfocused, empty — shows placeholder
	seq.AddStyled("unfocused_empty", rerender(inp, 15, 1))

	// Step 2: focused, empty — shows placeholder dimly
	inp.Focus()
	seq.AddStyled("focused_empty", rerender(inp, 15, 1))

	// Step 3: focused, type text
	pressKey(inp, "H")
	pressKey(inp, "i")
	seq.AddStyled("focused_with_text", rerender(inp, 15, 1))

	// Step 4: unfocused, with text
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

	// Backspace twice
	pressKey(inp, input.KeyBackspace)
	seq.Add("backspace_1", rerender(inp, 10, 1))

	pressKey(inp, input.KeyBackspace)
	seq.Add("backspace_2", rerender(inp, 10, 1))

	testutil.AssertGoldenSequence(t, "input_typing_sequence", seq)
}

// ---------------------------------------------------------------------------
// Text wrapping: content that grows and needs re-wrap
// ---------------------------------------------------------------------------

// TestGoldenSeq_TextWrapGrowingContent captures text that grows beyond
// the viewport width, triggering wrapping.
func TestGoldenSeq_TextWrapGrowingContent(t *testing.T) {
	txt := widget.NewText("", widget.TextWithWrapMode(widget.WrapWord))
	testutil.SetupWidget(txt, 10, 4)

	seq := testutil.NewGoldenSequence()

	// Step 1: short text, no wrap needed
	txt.ClearNodes()
	txt.AddText("Hi")
	seq.Add("short", rerender(txt, 10, 4))

	// Step 2: medium text, single wrap
	txt.ClearNodes()
	txt.AddText("Hello World")
	seq.Add("medium", rerender(txt, 10, 4))

	// Step 3: long text, multiple wraps
	txt.ClearNodes()
	txt.AddText("The quick brown fox jumps")
	seq.Add("long", rerender(txt, 10, 4))

	testutil.AssertGoldenSequence(t, "text_wrap_growing", seq)
}

// ---------------------------------------------------------------------------
// Box focus: border color change
// ---------------------------------------------------------------------------

// TestGoldenSeq_BoxFocusBorderColor captures the box border color
// changing when the box gains/loses focus.
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

	// Step 1: unfocused — white border
	seq.AddStyled("unfocused", rerender(box, 10, 3))

	// Step 2: focused — green border
	box.Focus()
	seq.AddStyled("focused", rerender(box, 10, 3))

	// Step 3: blurred — back to white
	box.Blur()
	seq.AddStyled("blurred", rerender(box, 10, 3))

	testutil.AssertGoldenSequence(t, "box_focus_border_color", seq)
}

// ---------------------------------------------------------------------------
// Text scroll + wrap interaction
// ---------------------------------------------------------------------------

// TestGoldenSeq_TextWrapThenScroll tests wrapped text that is scrolled.
func TestGoldenSeq_TextWrapThenScroll(t *testing.T) {
	txt := widget.NewText("Line 1 content\nLine 2 content\nLine 3 content\nLine 4 content",
		widget.TextWithWrapMode(widget.WrapWord))
	testutil.SetupWidget(txt, 10, 3)

	seq := testutil.NewGoldenSequence()

	// Step 1: initial view
	seq.Add("initial", rerender(txt, 10, 3))

	// Step 2: scroll down
	txt.SetScrollY(2)
	seq.Add("scrolled", rerender(txt, 10, 3))

	testutil.AssertGoldenSequence(t, "text_wrap_then_scroll", seq)
}
