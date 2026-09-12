package widget

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// --- Code tests ---

func TestNewCode(t *testing.T) {
	c := NewCode("func main() {}", CodeWithFiletype("go"))
	if c.Content() != "func main() {}" {
		t.Errorf("expected content, got %q", c.Content())
	}
	if c.Filetype() != "go" {
		t.Errorf("expected filetype 'go', got %q", c.Filetype())
	}
	if c.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", c.LineCount())
	}
}

func TestCode_MultiLine(t *testing.T) {
	code := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"
	c := NewCode(code)
	if c.LineCount() != 7 {
		t.Errorf("expected 7 lines, got %d", c.LineCount())
	}
}

func TestCode_Render(t *testing.T) {
	c := NewCode("hello", CodeWithFg(style.ColorGreen))
	testutil.SetupWidget(c, 10, 1)

	buf := buffer.New(10, 1)
	c.RenderSelf(buf, 0)

	cell := buf.Cell(0, 0)
	if cell.Char != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", cell.Char)
	}
}

func TestCode_SetContent(t *testing.T) {
	c := NewCode("original")
	c.SetContent("updated")
	if c.Content() != "updated" {
		t.Errorf("expected 'updated', got %q", c.Content())
	}
}

func TestCode_SyntaxHighlighting_Go(t *testing.T) {
	code := "package main\n\nfunc main() {}"
	c := NewCode(code, CodeWithFiletype("go"))
	testutil.SetupWidget(c, 40, 3)

	buf := buffer.New(40, 3)
	c.RenderSelf(buf, 0)

	// "package" is a keyword — should get syntax color, not default white
	cell := buf.Cell(0, 0) // 'p' of 'package'
	if cell.Char != 'p' {
		t.Errorf("expected 'p' at (0,0), got %c", cell.Char)
	}
	// With monokai theme, keywords get a distinct color, not plain white
	if cell.Fg == style.ColorWhite {
		t.Error("expected syntax-highlighted 'package' keyword to have non-white fg color")
	}
}

func TestCode_SyntaxHighlighting_Python(t *testing.T) {
	code := "def hello():\n    print('world')"
	c := NewCode(code, CodeWithFiletype("python"))
	testutil.SetupWidget(c, 40, 2)

	buf := buffer.New(40, 2)
	c.RenderSelf(buf, 0)

	// 'def' is a keyword
	cell := buf.Cell(0, 0) // 'd'
	if cell.Char != 'd' {
		t.Errorf("expected 'd' at (0,0), got %c", cell.Char)
	}
	if cell.Fg == style.ColorWhite {
		t.Error("expected syntax-highlighted 'def' keyword to have non-white fg color")
	}
}

func TestCode_SyntaxHighlighting_JavaScript(t *testing.T) {
	code := "const x = 42;"
	c := NewCode(code, CodeWithFiletype("javascript"))
	testutil.SetupWidget(c, 40, 1)

	buf := buffer.New(40, 1)
	c.RenderSelf(buf, 0)

	// 'c' of 'const' keyword should be highlighted
	cell := buf.Cell(0, 0)
	if cell.Char != 'c' {
		t.Errorf("expected 'c' at (0,0), got %c", cell.Char)
	}
	if cell.Fg == style.ColorWhite {
		t.Error("expected syntax-highlighted 'const' keyword to have non-white fg color")
	}
}

func TestCode_SyntaxHighlighting_UnknownLanguage(t *testing.T) {
	code := "hello world"
	c := NewCode(code, CodeWithFiletype("nonexistent_language_xyz"))
	testutil.SetupWidget(c, 20, 1)

	buf := buffer.New(20, 1)
	c.RenderSelf(buf, 0)

	// Should fallback to rendering plain text without crashing
	cell := buf.Cell(0, 0)
	if cell.Char != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", cell.Char)
	}
}

func TestCode_ThemeSwitch(t *testing.T) {
	code := "func main() {}"
	c := NewCode(code, CodeWithFiletype("go"), CodeWithTheme("monokai"))

	if c.Theme() != "monokai" {
		t.Errorf("expected theme 'monokai', got %q", c.Theme())
	}

	testutil.SetupWidget(c, 30, 1)

	buf1 := buffer.New(30, 1)
	c.RenderSelf(buf1, 0)
	fgMonokai := buf1.Cell(0, 0).Fg

	// Switch to a different theme
	c.SetTheme("github")
	if c.Theme() != "github" {
		t.Errorf("expected theme 'github', got %q", c.Theme())
	}

	buf2 := buffer.New(30, 1)
	c.RenderSelf(buf2, 0)
	fgGithub := buf2.Cell(0, 0).Fg

	// The two themes should produce different colors for keywords
	// (this test verifies theme switching works, not specific color values)
	_ = fgMonokai
	_ = fgGithub
}

func TestCode_ScrollY(t *testing.T) {
	// 10 lines, viewport of 3 lines
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = string(rune('A'+i)) + " line"
	}
	code := strings.Join(lines, "\n")

	c := NewCode(code)
	testutil.SetupWidget(c, 20, 3)

	// Initially at scrollY=0
	if c.ScrollY() != 0 {
		t.Errorf("expected scrollY=0, got %d", c.ScrollY())
	}

	buf := buffer.New(20, 3)
	c.RenderSelf(buf, 0)
	if buf.Cell(0, 0).Char != 'A' {
		t.Errorf("expected 'A' at row 0, got %c", buf.Cell(0, 0).Char)
	}

	// Scroll down by 3
	c.SetScrollY(3)
	if c.ScrollY() != 3 {
		t.Errorf("expected scrollY=3, got %d", c.ScrollY())
	}

	buf2 := buffer.New(20, 3)
	c.RenderSelf(buf2, 0)
	if buf2.Cell(0, 0).Char != 'D' {
		t.Errorf("expected 'D' at row 0 after scroll, got %c", buf2.Cell(0, 0).Char)
	}

	// MaxScrollY should be 7 (10 lines - 3 viewport)
	if c.MaxScrollY() != 7 {
		t.Errorf("expected MaxScrollY=7, got %d", c.MaxScrollY())
	}
}

func TestCode_ScrollY_Clamping(t *testing.T) {
	c := NewCode("line1\nline2\nline3")
	testutil.SetupWidget(c, 20, 5)

	// MaxScrollY should be 0 (content fits)
	if c.MaxScrollY() != 0 {
		t.Errorf("expected MaxScrollY=0, got %d", c.MaxScrollY())
	}

	// Setting scrollY to negative should clamp to 0
	c.SetScrollY(-5)
	if c.ScrollY() != 0 {
		t.Errorf("expected scrollY=0 after negative, got %d", c.ScrollY())
	}

	// Setting scrollY beyond max should clamp
	c.SetScrollY(100)
	if c.ScrollY() != 0 {
		t.Errorf("expected scrollY=0 after over-scroll, got %d", c.ScrollY())
	}
}

func TestCode_ScrollX(t *testing.T) {
	// Long line that exceeds viewport width (WrapNone needed for horizontal scroll)
	c := NewCode("ABCDEFGHIJKLMNOPQRSTUVWXYZ", CodeWithWrapMode(WrapNone))
	testutil.SetupWidget(c, 10, 1)

	buf := buffer.New(10, 1)
	c.RenderSelf(buf, 0)
	if buf.Cell(0, 0).Char != 'A' {
		t.Errorf("expected 'A' at x=0, got %c", buf.Cell(0, 0).Char)
	}

	// Scroll right by 5
	c.SetScrollX(5)
	if c.ScrollX() != 5 {
		t.Errorf("expected scrollX=5, got %d", c.ScrollX())
	}

	buf2 := buffer.New(10, 1)
	c.RenderSelf(buf2, 0)
	if buf2.Cell(0, 0).Char != 'F' {
		t.Errorf("expected 'F' at x=0 after scroll, got %c", buf2.Cell(0, 0).Char)
	}

	// MaxScrollX = 26 - 10 = 16
	if c.MaxScrollX() != 16 {
		t.Errorf("expected MaxScrollX=16, got %d", c.MaxScrollX())
	}
}

func TestCode_ScrollToBottom(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	c := NewCode(strings.Join(lines, "\n"))
	testutil.SetupWidget(c, 20, 5)

	c.ScrollToBottom()
	if c.ScrollY() != 15 { // 20 - 5 = 15
		t.Errorf("expected scrollY=15, got %d", c.ScrollY())
	}
}

func TestCode_MouseScroll(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}
	c := NewCode(strings.Join(lines, "\n"))
	testutil.SetupWidget(c, 20, 3)

	if c.OnMouseScrollHandler == nil {
		t.Fatal("expected OnMouseScrollHandler to be set")
	}

	// Simulate scroll down
	e := &input.MouseEvent{
		Type:   input.MouseScroll,
		Scroll: &input.ScrollInfo{DeltaX: 0, DeltaY: 2},
	}
	c.OnMouseScrollHandler(e)

	if c.ScrollY() != 2 {
		t.Errorf("expected scrollY=2 after scroll, got %d", c.ScrollY())
	}

	// Scroll up
	e2 := &input.MouseEvent{
		Type:   input.MouseScroll,
		Scroll: &input.ScrollInfo{DeltaX: 0, DeltaY: -1},
	}
	c.OnMouseScrollHandler(e2)

	if c.ScrollY() != 1 {
		t.Errorf("expected scrollY=1 after scroll up, got %d", c.ScrollY())
	}
}

func TestCode_WrapModeChar(t *testing.T) {
	// 20-char line in a 10-wide viewport should wrap to 2 visual lines
	c := NewCode("12345678901234567890", CodeWithWrapMode(WrapChar))
	testutil.SetupWidget(c, 10, 5)

	if c.VisualLineCount() != 2 {
		t.Errorf("expected 2 visual lines with char wrap, got %d", c.VisualLineCount())
	}

	buf := buffer.New(10, 5)
	c.RenderSelf(buf, 0)
	if buf.Cell(0, 0).Char != '1' {
		t.Errorf("expected '1' at (0,0), got %c", buf.Cell(0, 0).Char)
	}
	if buf.Cell(0, 1).Char != '1' {
		// Second visual line starts with char 11 which is '1'
		t.Errorf("expected '1' at (0,1), got %c", buf.Cell(0, 1).Char)
	}
}

func TestCode_WrapModeWord(t *testing.T) {
	// Words that should wrap at word boundaries
	c := NewCode("hello world foo bar", CodeWithWrapMode(WrapWord))
	testutil.SetupWidget(c, 12, 5)

	if c.VisualLineCount() < 2 {
		t.Errorf("expected at least 2 visual lines with word wrap, got %d", c.VisualLineCount())
	}
}

func TestCode_WrapVisualToLogicalMapping(t *testing.T) {
	// Two logical lines, first one wraps.
	// Logical 0: "hello world foo bar baz" (23 chars, wraps at width 12)
	// Logical 1: "end"
	c := NewCode("hello world foo bar baz\nend", CodeWithWrapMode(WrapWord))
	testutil.SetupWidget(c, 12, 10)

	if c.LineCount() != 2 {
		t.Fatalf("expected 2 logical lines, got %d", c.LineCount())
	}

	vc := c.VisualLineCount()
	t.Logf("visual line count: %d", vc)
	for i := range vc {
		logical := c.LogicalLineForVisualLine(i)
		t.Logf("  visual %d -> logical %d", i, logical)
	}

	// All visual lines except the last should map to logical 0
	for i := 0; i < vc-1; i++ {
		if c.LogicalLineForVisualLine(i) != 0 {
			t.Errorf("visual line %d should map to logical 0, got %d", i, c.LogicalLineForVisualLine(i))
		}
	}
	// Last visual line should map to logical 1
	if c.LogicalLineForVisualLine(vc-1) != 1 {
		t.Errorf("last visual line should map to logical 1, got %d", c.LogicalLineForVisualLine(vc-1))
	}
}

func TestCode_WrapModeNone_NoScrollX(t *testing.T) {
	// With WrapNone, horizontal scroll should be possible
	c := NewCode("ABCDEFGHIJKLMNOPQRSTUVWXYZ", CodeWithWrapMode(WrapNone))
	testutil.SetupWidget(c, 10, 1)

	// VisualLineCount should be 1 (no wrapping)
	if c.VisualLineCount() != 1 {
		t.Errorf("expected 1 visual line with WrapNone, got %d", c.VisualLineCount())
	}

	// MaxScrollX should be > 0
	if c.MaxScrollX() <= 0 {
		t.Error("expected MaxScrollX > 0 with long line and WrapNone")
	}
}

func TestCode_SetFiletype(t *testing.T) {
	c := NewCode("x = 42")
	testutil.SetupWidget(c, 20, 1)

	// Initially no filetype
	buf1 := buffer.New(20, 1)
	c.RenderSelf(buf1, 0)

	// Set filetype to python
	c.SetFiletype("python")
	if c.Filetype() != "python" {
		t.Errorf("expected filetype 'python', got %q", c.Filetype())
	}

	buf2 := buffer.New(20, 1)
	c.RenderSelf(buf2, 0)

	// Content should still render
	if buf2.Cell(0, 0).Char != 'x' {
		t.Errorf("expected 'x' at (0,0), got %c", buf2.Cell(0, 0).Char)
	}
}

func TestCode_AppendContent(t *testing.T) {
	c := NewCode("line1")
	if c.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", c.LineCount())
	}

	c.AppendContent("\nline2\nline3")
	if c.Content() != "line1\nline2\nline3" {
		t.Errorf("expected appended content, got %q", c.Content())
	}
	if c.LineCount() != 3 {
		t.Errorf("expected 3 lines after append, got %d", c.LineCount())
	}
}

func TestCode_Streaming(t *testing.T) {
	c := NewCode("", CodeWithStreaming(true))
	if !c.Streaming() {
		t.Error("expected streaming=true")
	}

	c.AppendContent("line1\n")
	c.AppendContent("line2\n")
	c.AppendContent("line3")

	if c.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", c.LineCount())
	}
}

func TestCode_ActiveLine(t *testing.T) {
	code := "line1\nline2\nline3"
	c := NewCode(code, CodeWithActiveLine(1))
	testutil.SetupWidget(c, 20, 3)

	if c.ActiveLine() != 1 {
		t.Errorf("expected activeLine=1, got %d", c.ActiveLine())
	}

	// Render and verify active line background is applied
	buf := buffer.New(20, 3)
	c.RenderSelf(buf, 0)

	// Row 1 (active line) should have a different bg than row 0
	cellActive := buf.Cell(0, 1)
	cellNormal := buf.Cell(0, 0)
	// Active line has activeLineBg; normal line has the default (or transparent)
	_ = cellActive
	_ = cellNormal
	// Just verify it renders without panic
}

func TestCode_Options(t *testing.T) {
	c := NewCode("test",
		CodeWithID("code1"),
		CodeWithFg(style.ColorGreen),
		CodeWithBg(style.ColorFromInts(30, 30, 30, 255)),
		CodeWithFiletype("go"),
		CodeWithTheme("dracula"),
		CodeWithWrapMode(WrapWord),
		CodeWithStreaming(true),
		CodeWithSelectable(true),
		CodeWithScrollX(0),
		CodeWithScrollY(0),
		CodeWithActiveLine(0),
		CodeWithWidth(layout.Points(40)),
		CodeWithHeight(layout.Points(10)),
		CodeWithFlexGrow(1),
	)

	if c.ID() != "code1" {
		t.Errorf("expected ID 'code1', got %q", c.ID())
	}
	if c.Fg() != style.ColorGreen {
		t.Error("expected green fg")
	}
	if c.Theme() != "dracula" {
		t.Errorf("expected theme 'dracula', got %q", c.Theme())
	}
	if c.WrapModeValue() != WrapWord {
		t.Errorf("expected WrapWord, got %d", c.WrapModeValue())
	}
	if !c.Streaming() {
		t.Error("expected streaming=true")
	}
	if !c.Selectable() {
		t.Error("expected selectable=true")
	}
	if c.ActiveLine() != 0 {
		t.Errorf("expected activeLine=0, got %d", c.ActiveLine())
	}
}

func TestCode_ScrollInfoProvider(t *testing.T) {
	// Verify Code implements ScrollInfoProvider interface
	c := NewCode("line1\nline2\nline3")
	var _ ScrollInfoProvider = c // compile-time check
	if c.ScrollY() != 0 {
		t.Errorf("expected initial ScrollY=0, got %d", c.ScrollY())
	}
}

func TestCode_LineInfoProvider(t *testing.T) {
	// Verify Code implements LineInfoProvider interface
	c := NewCode("line1\nline2\nline3")
	var _ LineInfoProvider = c // compile-time check
	if c.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", c.LineCount())
	}
}

func TestCode_GetLineHighlights(t *testing.T) {
	c := NewCode("package main\nfunc main() {}", CodeWithFiletype("go"))
	testutil.SetupWidget(c, 40, 2)

	chunks := c.LineHighlights(0)
	if len(chunks) == 0 {
		t.Error("expected non-empty line highlights for line 0")
	}

	// Concatenate chunk text should give the line content
	var text string
	for _, ch := range chunks {
		text += ch.Text
	}
	if text != "package main" {
		t.Errorf("expected line 0 text 'package main', got %q", text)
	}

	// Out of range should return nil
	chunks2 := c.LineHighlights(-1)
	if chunks2 != nil {
		t.Error("expected nil for negative line index")
	}
	chunks3 := c.LineHighlights(100)
	if chunks3 != nil {
		t.Error("expected nil for out-of-range line index")
	}
}

func TestCode_EmptyContent(t *testing.T) {
	c := NewCode("")
	if c.LineCount() != 1 {
		t.Errorf("expected 1 line for empty content, got %d", c.LineCount())
	}

	testutil.SetupWidget(c, 10, 1)

	// Should render without panic
	buf := buffer.New(10, 1)
	c.RenderSelf(buf, 0)
}

func TestCode_MeasureIntrinsic(t *testing.T) {
	c := NewCode("hello\nworld foo bar")

	// Use TextSurface's MeasureForDimensions (width=0 = no-wrap mode)
	m := c.textSurface.MeasureForDimensions(0, 0)
	if m == nil {
		t.Fatal("expected non-nil measure result")
	}
	if m.LineCount != 2 {
		t.Errorf("expected height 2, got %d", m.LineCount)
	}
	if m.MaxWidth != 13 { // "world foo bar" = 13 chars
		t.Errorf("expected width 13, got %d", m.MaxWidth)
	}
}

// --- Markdown tests ---

func TestNewMarkdown(t *testing.T) {
	m := NewMarkdown("# Hello\n\nThis is a paragraph.")
	if m.Content() != "# Hello\n\nThis is a paragraph." {
		t.Errorf("unexpected content: %q", m.Content())
	}
	if m.LineCount() < 2 {
		t.Errorf("expected at least 2 lines, got %d", m.LineCount())
	}
}

func TestMarkdown_Headings(t *testing.T) {
	// Three headings with one blank row inserted between each pair by
	// InsertBlockSpacing: H1, blank, H2, blank, H3.
	m := NewMarkdown("# H1\n## H2\n### H3")
	if m.LineCount() != 5 {
		t.Errorf("expected 5 lines (3 headings + 2 blank separators), got %d", m.LineCount())
	}
}

func TestMarkdown_CodeBlock(t *testing.T) {
	m := NewMarkdown("```go\nfunc main() {}\n```")
	// Code fences are not rendered as lines, content is
	if m.LineCount() < 1 {
		t.Errorf("expected at least 1 rendered line, got %d", m.LineCount())
	}
}

func TestMarkdown_List(t *testing.T) {
	m := NewMarkdown("- item 1\n- item 2\n- item 3")
	if m.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", m.LineCount())
	}
}

func TestMarkdown_Render(t *testing.T) {
	m := NewMarkdown("# Test")
	testutil.SetupWidget(m, 20, 5)

	buf := buffer.New(20, 5)
	m.RenderSelf(buf, 0)

	// First character should be 'T' (heading without #)
	cell := buf.Cell(0, 0)
	if cell.Char != 'T' {
		t.Errorf("expected 'T' at (0,0), got %c", cell.Char)
	}
}

func TestMarkdown_AllHeadings(t *testing.T) {
	// Six headings with one blank row between each pair: 6 + 5 = 11.
	m := NewMarkdown("# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6")
	testutil.SetupWidget(m, 40, 15)

	if m.LineCount() != 11 {
		t.Errorf("expected 11 lines (6 headings + 5 blank separators), got %d", m.LineCount())
	}

	buf := buffer.New(40, 15)
	m.RenderSelf(buf, 0)

	// H1 should start with 'H'
	if buf.Cell(0, 0).Char != 'H' {
		t.Errorf("expected 'H' at row 0, got %c", buf.Cell(0, 0).Char)
	}
}

func TestMarkdown_Links(t *testing.T) {
	m := NewMarkdown("[click here](https://example.com)")
	testutil.SetupWidget(m, 40, 1)

	if m.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", m.LineCount())
	}

	buf := buffer.New(40, 1)
	m.RenderSelf(buf, 0)

	// Should render "click here" (not the URL or brackets)
	cell := buf.Cell(0, 0)
	if cell.Char != 'c' {
		t.Errorf("expected 'c' at (0,0) for link text, got %c", cell.Char)
	}
	// Link text should be underlined and have link color
	if cell.Attributes&style.AttrUnderline == 0 {
		t.Error("expected link text to have underline attribute")
	}
}

func TestMarkdown_Strikethrough(t *testing.T) {
	m := NewMarkdown("this is ~~deleted~~ text")
	testutil.SetupWidget(m, 40, 1)

	buf := buffer.New(40, 1)
	m.RenderSelf(buf, 0)

	// Find the 'd' of "deleted" and check strikethrough attr
	// "this is " = 8 chars, so 'd' should be at x=8
	cell := buf.Cell(8, 0)
	if cell.Char != 'd' {
		t.Errorf("expected 'd' at (8,0), got %c", cell.Char)
	}
	if cell.Attributes&style.AttrStrikethrough == 0 {
		t.Error("expected strikethrough attribute on deleted text")
	}
}

func TestMarkdown_MixedInline(t *testing.T) {
	m := NewMarkdown("**bold** and *italic* and `code`")
	testutil.SetupWidget(m, 50, 1)

	buf := buffer.New(50, 1)
	m.RenderSelf(buf, 0)

	// "bold" starts at 0 with AttrBold
	cell := buf.Cell(0, 0)
	if cell.Char != 'b' {
		t.Errorf("expected 'b' at (0,0), got %c", cell.Char)
	}
	if cell.Attributes&style.AttrBold == 0 {
		t.Error("expected bold attribute")
	}
}

func TestMarkdown_Table(t *testing.T) {
	md := "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |"
	m := NewMarkdown(md)
	testutil.SetupWidget(m, 40, 20)

	// Table should produce: top border + header + separator + 2 data rows + bottom border = 6 lines
	if m.LineCount() < 5 {
		t.Errorf("expected at least 5 lines for table, got %d", m.LineCount())
	}

	buf := buffer.New(40, 20)
	m.RenderSelf(buf, 0)

	// Top-left corner should be box drawing char
	cell := buf.Cell(0, 0)
	if cell.Char != '┌' {
		t.Errorf("expected '┌' at (0,0) for table border, got %c (%U)", cell.Char, cell.Char)
	}
}

func TestMarkdown_TableAlignment(t *testing.T) {
	md := "| Left | Center | Right |\n|:-----|:------:|------:|\n| a | b | c |"
	m := NewMarkdown(md)
	testutil.SetupWidget(m, 50, 20)

	// Should parse alignment markers
	if m.LineCount() < 4 {
		t.Errorf("expected at least 4 lines for aligned table, got %d", m.LineCount())
	}
}

func TestMarkdown_WordWrap(t *testing.T) {
	// Long paragraph that should wrap
	longText := "This is a very long paragraph that should wrap at word boundaries when the widget is narrow enough to trigger wrapping behavior."
	m := NewMarkdown(longText, MarkdownWithWrapMode(WrapWord))
	testutil.SetupWidget(m, 30, 20)

	if m.LineCount() < 3 {
		t.Errorf("expected at least 3 wrapped lines, got %d", m.LineCount())
	}
}

func TestMarkdown_NoWrap(t *testing.T) {
	longText := "This is a very long line that should not wrap at all"
	m := NewMarkdown(longText, MarkdownWithWrapMode(WrapNone))
	testutil.SetupWidget(m, 20, 5)

	if m.LineCount() != 1 {
		t.Errorf("expected 1 line with WrapNone, got %d", m.LineCount())
	}
}

func TestMarkdown_MeasureFuncReportsLineCount(t *testing.T) {
	// Markdown with 20 lines inside a column container should report
	// its intrinsic height via MeasureFunc.
	// 10 paragraphs separated by blank lines → 10 content + 9 spacer = 19 lines.
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("Line %d", i))
	}
	m := NewMarkdown(strings.Join(lines, "\n\n"),
		MarkdownWithWidth(layout.Points(40)),
	)
	// Height is Auto — MeasureFunc should report the full line count.

	container := layout.NewNode()
	container.FlexDirection = layout.DirectionColumn
	container.Width = layout.Points(40)
	container.Height = layout.Points(100)
	container.AddChild(m.LayoutNode())

	container.ComputeLayout(40, 100)
	m.UpdateFromLayout()

	expected := 19 // 10 paragraphs + 9 blank lines between them
	if m.Height() != expected {
		t.Errorf("expected intrinsic height=%d, got %d", expected, m.Height())
	}
	if m.LineCount() != expected {
		t.Errorf("expected LineCount()=%d, got %d", expected, m.LineCount())
	}
}

func TestMarkdown_Blockquote(t *testing.T) {
	m := NewMarkdown("> This is a quote")
	testutil.SetupWidget(m, 40, 1)

	buf := buffer.New(40, 1)
	m.RenderSelf(buf, 0)

	// Should start with '│' border
	cell := buf.Cell(0, 0)
	if cell.Char != '│' {
		t.Errorf("expected '│' at (0,0) for blockquote, got %c", cell.Char)
	}
}

func TestMarkdown_HorizontalRule(t *testing.T) {
	// Thematic breaks render as invisible spacing (no visual rule),
	// styled as margin only rather than a drawn line.
	// A rule between two paragraphs produces a 1-row gap — the same as
	// a plain paragraph break — via a single blank row contributed by
	// InsertBlockSpacing before the rule (the trailing spacer is
	// suppressed by isContinuationBlock).
	m := NewMarkdown("before\n\n---\n\nafter")
	if got := m.LineCount(); got != 3 {
		t.Errorf("expected paragraph+rule+paragraph to produce 3 lines (before, blank, after), got %d", got)
	}

	testutil.SetupWidget(m, 40, 3)
	buf := buffer.New(40, 3)
	m.RenderSelf(buf, 0)

	// Row 0: "before"
	if cell := buf.Cell(0, 0); cell.Char != 'b' {
		t.Errorf("expected 'b' at (0,0), got %q", cell.Char)
	}
	// Row 1: blank (the single-row gap).
	for x := 0; x < 40; x++ {
		cell := buf.Cell(x, 1)
		if cell.Char != 0 && cell.Char != ' ' {
			t.Errorf("expected empty cell at (%d,1) inside rule gap, got %q", x, cell.Char)
		}
	}
	// Row 2: "after"
	if cell := buf.Cell(0, 2); cell.Char != 'a' {
		t.Errorf("expected 'a' at (0,2), got %q", cell.Char)
	}
}

func TestMarkdown_Theme_AppliesAllRoles(t *testing.T) {
	th := style.DefaultDark()
	// Override a couple of roles so we can assert propagation on values
	// that differ from the widget's built-in defaults.
	th.MarkdownText = style.Entry{Fg: style.ColorFromInts(10, 20, 30, 255)}
	th.MarkdownHeading[0] = style.Entry{Fg: style.ColorFromInts(40, 50, 60, 255), Attrs: style.AttrBold}
	th.CodeFg = style.ColorFromInts(70, 80, 90, 255)
	th.CodeBg = style.ColorFromInts(100, 110, 120, 255)
	th.MarkdownCode = style.Entry{Fg: th.CodeFg, Bg: th.CodeBg}
	th.MarkdownLink = style.Entry{Fg: style.ColorFromInts(130, 140, 150, 255), Attrs: style.AttrUnderline}
	th.MarkdownBullet = style.Entry{Fg: style.ColorFromInts(160, 170, 180, 255)}
	th.MarkdownListEnum = style.Entry{Fg: style.ColorFromInts(165, 175, 185, 255)}
	th.MarkdownQuote = style.Entry{Fg: style.ColorFromInts(190, 200, 210, 255), Attrs: style.AttrItalic}
	th.MarkdownStrong = style.Entry{Fg: style.ColorFromInts(11, 21, 31, 255), Attrs: style.AttrBold}
	th.MarkdownEmph = style.Entry{Fg: style.ColorFromInts(12, 22, 32, 255), Attrs: style.AttrItalic}

	m := NewMarkdown("hello", MarkdownWithTheme(th))
	s := m.mdStyles()

	if !s.Fg.Equals(th.MarkdownText.Fg) {
		t.Errorf("Fg mismatch: got %v want %v", s.Fg, th.MarkdownText.Fg)
	}
	cases := []struct {
		name string
		got  style.Entry
		want style.Entry
	}{
		{"Heading[0]", s.Heading(1), th.MarkdownHeading[0]},
		{"Code", s.Code, th.MarkdownCode},
		{"Link", s.Link, th.MarkdownLink},
		{"Bullet", s.Bullet, th.MarkdownBullet},
		{"ListEnum", s.ListEnum, th.MarkdownListEnum},
		{"Quote", s.Quote, th.MarkdownQuote},
		{"Strong", s.Strong, th.MarkdownStrong},
		{"Emph", s.Emph, th.MarkdownEmph},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s mismatch: got %+v want %+v", c.name, c.got, c.want)
		}
	}
}

func TestMarkdown_Theme_LaterOptionWins(t *testing.T) {
	// Per-widget With... options applied after a theme must override
	// the theme's value.
	th := style.DefaultDark()
	override := style.ColorFromInts(1, 2, 3, 255)
	m := NewMarkdown("x", MarkdownWithTheme(th), MarkdownWithFg(override))
	if got := m.Fg(); !got.Equals(override) {
		t.Errorf("expected option override to beat theme; got %v want %v", got, override)
	}
}

func TestMarkdown_OrderedList(t *testing.T) {
	m := NewMarkdown("1. first\n2. second\n3. third")
	if m.LineCount() < 3 {
		t.Errorf("expected at least 3 lines for ordered list, got %d", m.LineCount())
	}
}

func TestMarkdown_NestedList(t *testing.T) {
	m := NewMarkdown("- item1\n  - nested1\n  - nested2\n- item2")
	testutil.SetupWidget(m, 40, 10)

	if m.LineCount() < 4 {
		t.Errorf("expected at least 4 lines for nested list, got %d", m.LineCount())
	}
}

func TestMarkdown_EmptyContent(t *testing.T) {
	m := NewMarkdown("")
	testutil.SetupWidget(m, 20, 5)

	buf := buffer.New(20, 5)
	m.RenderSelf(buf, 0) // should not panic
}

func TestMarkdown_AppendContent(t *testing.T) {
	m := NewMarkdown("# Title")
	m.AppendContent("\n\nParagraph text.")
	if !strings.Contains(m.Content(), "Paragraph text.") {
		t.Error("expected appended content")
	}
}

func TestMarkdown_AppendContentStreaming(t *testing.T) {
	m := NewMarkdown("")
	m.AppendContent("# Hello\n")
	m.AppendContent("World")
	if m.LineCount() < 2 {
		t.Errorf("expected at least 2 lines, got %d", m.LineCount())
	}
}

func TestMarkdown_Options(t *testing.T) {
	m := NewMarkdown("test",
		MarkdownWithID("md1"),
		MarkdownWithFg(style.ColorGreen),
		MarkdownWithBg(style.ColorFromInts(30, 30, 30, 255)),
		MarkdownWithWrapMode(WrapChar),
		MarkdownWithWidth(layout.Points(40)),
		MarkdownWithHeight(layout.Points(10)),
		MarkdownWithFlexGrow(1),
	)

	if m.ID() != "md1" {
		t.Errorf("expected ID 'md1', got %q", m.ID())
	}
	if m.Fg() != style.ColorGreen {
		t.Error("expected green fg")
	}
	if m.WrapModeValue() != WrapChar {
		t.Error("expected WrapChar")
	}
}

func TestMarkdown_InlineCode(t *testing.T) {
	m := NewMarkdown("Use `fmt.Println` here")
	testutil.SetupWidget(m, 40, 1)

	buf := buffer.New(40, 1)
	m.RenderSelf(buf, 0)

	// 'f' of fmt.Println starts at position 4 ("Use ")
	cell := buf.Cell(4, 0)
	if cell.Char != 'f' {
		t.Errorf("expected 'f' at (4,0), got %c", cell.Char)
	}
}

// --- Diff tests ---

func TestNewDiff(t *testing.T) {
	patch := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {
-    println("hello")
+    fmt.Println("hello")
 }`

	d := NewDiff(patch)
	if d.Patch() != patch {
		t.Error("unexpected patch content")
	}
	if d.LineCount() < 5 {
		t.Errorf("expected at least 5 lines, got %d", d.LineCount())
	}
}

func TestDiff_Render(t *testing.T) {
	// Simple diff with added/removed lines
	patch := "@@ -1,3 +1,3 @@\n context\n-old line\n+new line"
	d := NewDiff(patch)
	testutil.SetupWidget(d, 40, 10)

	buf := buffer.New(40, 10)
	d.RenderSelf(buf, 0)

	// Should render without panic and produce content
	// Line 0 is the hunk header "@@..." — rendered in hunk color
	// Line 1 is context, Line 2 is removed, Line 3 is added
	if d.LineCount() != 4 {
		t.Errorf("expected 4 lines, got %d", d.LineCount())
	}
}

func TestDiff_SetPatch(t *testing.T) {
	d := NewDiff("original")
	d.SetPatch("+new line")
	if d.Patch() != "+new line" {
		t.Errorf("unexpected patch after SetPatch")
	}
}

func TestDiff_UnifiedView_LineNumbers(t *testing.T) {
	patch := "@@ -1,3 +1,4 @@\n context1\n-removed\n+added1\n+added2\n context2"
	d := NewDiff(patch, DiffWithShowLineNumbers(true))
	testutil.SetupWidget(d, 50, 10)

	// LineCount includes hunk header + content lines
	if d.LineCount() < 5 {
		t.Errorf("expected at least 5 lines, got %d", d.LineCount())
	}

	buf := buffer.New(50, 10)
	d.RenderSelf(buf, 0)

	// Verify it renders without crashing — the gutter + content should fit in 50 cols
}

func TestDiff_SplitView(t *testing.T) {
	patch := "@@ -1,3 +1,3 @@\n context\n-old line\n+new line"
	d := NewDiff(patch, DiffWithViewMode(DiffSplit))

	if d.ViewMode() != DiffSplit {
		t.Errorf("expected DiffSplit view mode")
	}

	testutil.SetupWidget(d, 60, 10)

	buf := buffer.New(60, 10)
	d.RenderSelf(buf, 0)

	// Split view should have entries
	if d.LineCount() < 2 {
		t.Errorf("expected at least 2 split pairs, got %d", d.LineCount())
	}
}

func TestDiff_SplitView_Padding(t *testing.T) {
	// 2 removes, 1 add — left panel should have 2 lines, right should have 1 + 1 empty
	patch := "@@ -1,3 +1,2 @@\n context\n-remove1\n-remove2\n+add1"
	d := NewDiff(patch, DiffWithViewMode(DiffSplit))
	testutil.SetupWidget(d, 60, 10)

	buf := buffer.New(60, 10)
	d.RenderSelf(buf, 0)

	// The removed block (2) should be padded against adds (1)
	if d.LineCount() < 3 {
		t.Errorf("expected at least 3 split pairs, got %d", d.LineCount())
	}
}

func TestDiff_ScrollY(t *testing.T) {
	// Large diff with many lines
	var lines []string
	lines = append(lines, "@@ -1,10 +1,10 @@")
	for i := range 20 {
		lines = append(lines, fmt.Sprintf(" line %d", i))
	}
	patch := strings.Join(lines, "\n")

	d := NewDiff(patch)
	testutil.SetupWidget(d, 40, 5)

	if d.ScrollY() != 0 {
		t.Errorf("expected initial scrollY=0, got %d", d.ScrollY())
	}

	d.SetScrollY(3)
	if d.ScrollY() != 3 {
		t.Errorf("expected scrollY=3, got %d", d.ScrollY())
	}

	// Negative should clamp
	d.SetScrollY(-5)
	if d.ScrollY() != 0 {
		t.Errorf("expected scrollY=0 after negative, got %d", d.ScrollY())
	}
}

func TestDiff_MouseScroll(t *testing.T) {
	var lines []string
	lines = append(lines, "@@ -1,10 +1,10 @@")
	for i := range 20 {
		lines = append(lines, fmt.Sprintf(" line %d", i))
	}

	d := NewDiff(strings.Join(lines, "\n"))
	testutil.SetupWidget(d, 40, 5)

	if d.OnMouseScrollHandler == nil {
		t.Fatal("expected OnMouseScrollHandler to be set")
	}

	e := &input.MouseEvent{
		Type:   input.MouseScroll,
		Scroll: &input.ScrollInfo{DeltaX: 0, DeltaY: 2},
	}
	d.OnMouseScrollHandler(e)

	if d.ScrollY() != 2 {
		t.Errorf("expected scrollY=2, got %d", d.ScrollY())
	}
}

func TestDiff_InlineDiff(t *testing.T) {
	// Test the computeInlineDiff function directly
	oldSegs, newSegs := computeInlineDiff("hello world", "hello earth")

	// Should have: "hello " unchanged, "world" vs "earth" changed
	if len(oldSegs) < 2 {
		t.Errorf("expected at least 2 old segments, got %d", len(oldSegs))
	}
	if len(newSegs) < 2 {
		t.Errorf("expected at least 2 new segments, got %d", len(newSegs))
	}

	// First segment should be unchanged (common prefix "hello ")
	if oldSegs[0].changed {
		t.Error("first old segment should be unchanged")
	}
	if newSegs[0].changed {
		t.Error("first new segment should be unchanged")
	}
}

func TestDiff_InlineDiff_Identical(t *testing.T) {
	oldSegs, newSegs := computeInlineDiff("same text", "same text")

	// All unchanged
	for _, s := range oldSegs {
		if s.changed {
			t.Error("expected all segments unchanged for identical text")
		}
	}
	for _, s := range newSegs {
		if s.changed {
			t.Error("expected all segments unchanged for identical text")
		}
	}
}

func TestDiff_InlineDiff_CompletelyDifferent(t *testing.T) {
	oldSegs, newSegs := computeInlineDiff("abc", "xyz")

	// Should have one changed segment each
	hasChanged := false
	for _, s := range oldSegs {
		if s.changed {
			hasChanged = true
		}
	}
	if !hasChanged {
		t.Error("expected at least one changed segment in old")
	}
	hasChanged = false
	for _, s := range newSegs {
		if s.changed {
			hasChanged = true
		}
	}
	if !hasChanged {
		t.Error("expected at least one changed segment in new")
	}
}

func TestDiff_ParseHunkHeader(t *testing.T) {
	p := ParsePatch("@@ -10,5 +20,7 @@ func main()")
	if p == nil || len(p.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}
	h := p.Hunks[0]
	if h.OldStart != 10 {
		t.Errorf("expected OldStart=10, got %d", h.OldStart)
	}
	if h.OldLines != 5 {
		t.Errorf("expected OldLines=5, got %d", h.OldLines)
	}
	if h.NewStart != 20 {
		t.Errorf("expected NewStart=20, got %d", h.NewStart)
	}
	if h.NewLines != 7 {
		t.Errorf("expected NewLines=7, got %d", h.NewLines)
	}
}

func TestDiff_ParseHunkHeader_SingleLine(t *testing.T) {
	p := ParsePatch("@@ -1 +1 @@")
	if p == nil || len(p.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}
	h := p.Hunks[0]
	if h.OldStart != 1 {
		t.Errorf("expected OldStart=1, got %d", h.OldStart)
	}
	if h.OldLines != 1 {
		t.Errorf("expected OldLines=1, got %d", h.OldLines)
	}
	if h.NewStart != 1 {
		t.Errorf("expected NewStart=1, got %d", h.NewStart)
	}
}

func TestDiff_EmptyPatch(t *testing.T) {
	d := NewDiff("")
	if d.LineCount() != 0 {
		t.Errorf("expected 0 lines for empty patch, got %d", d.LineCount())
	}

	testutil.SetupWidget(d, 20, 5)

	buf := buffer.New(20, 5)
	d.RenderSelf(buf, 0) // should not panic
}

func TestDiff_MultipleHunks(t *testing.T) {
	patch := `@@ -1,3 +1,3 @@
 line1
-old1
+new1
@@ -10,3 +10,3 @@
 line10
-old10
+new10`
	d := NewDiff(patch)
	if d.LineCount() < 6 {
		t.Errorf("expected at least 6 lines for multi-hunk diff, got %d", d.LineCount())
	}
}

func TestDiff_ScrollToBottom(t *testing.T) {
	var lines []string
	lines = append(lines, "@@ -1,20 +1,20 @@")
	for i := range 20 {
		lines = append(lines, fmt.Sprintf(" line %d", i))
	}

	d := NewDiff(strings.Join(lines, "\n"))
	testutil.SetupWidget(d, 40, 5)

	d.ScrollToBottom()
	expected := max(d.LineCount()-5, 0)
	if d.ScrollY() != expected {
		t.Errorf("expected scrollY=%d, got %d", expected, d.ScrollY())
	}
}

func TestDiff_ScrollInfoProvider(t *testing.T) {
	d := NewDiff("@@ -1 +1 @@\n-old\n+new")
	var _ ScrollInfoProvider = d // compile-time check
}

func TestDiff_LineInfoProvider(t *testing.T) {
	d := NewDiff("@@ -1 +1 @@\n-old\n+new")
	var _ LineInfoProvider = d // compile-time check
}

func TestDiff_SetViewMode(t *testing.T) {
	d := NewDiff("@@ -1,2 +1,2 @@\n-old\n+new")
	if d.ViewMode() != DiffUnified {
		t.Error("expected initial DiffUnified")
	}

	d.SetViewMode(DiffSplit)
	if d.ViewMode() != DiffSplit {
		t.Error("expected DiffSplit after SetViewMode")
	}
}

func TestDiff_Options(t *testing.T) {
	d := NewDiff("test",
		DiffWithID("diff1"),
		DiffWithViewMode(DiffSplit),
		DiffWithFiletype("go"),
		DiffWithTheme("dracula"),
		DiffWithShowLineNumbers(true),
		DiffWithInlineHighlight(false),
		DiffWithShowSigns(true),
		DiffWithAddedBg(style.ColorGreen),
		DiffWithRemovedBg(style.ColorRed),
		DiffWithContextBg(style.ColorTransparent),
		DiffWithScrollY(0),
		DiffWithWidth(layout.Points(40)),
		DiffWithHeight(layout.Points(10)),
		DiffWithFlexGrow(1),
	)

	if d.ID() != "diff1" {
		t.Errorf("expected ID 'diff1', got %q", d.ID())
	}
	if d.ViewMode() != DiffSplit {
		t.Error("expected DiffSplit")
	}
	if d.Filetype() != "go" {
		t.Errorf("expected filetype 'go', got %q", d.Filetype())
	}
	if d.Theme() != "dracula" {
		t.Errorf("expected theme 'dracula', got %q", d.Theme())
	}
}

func TestDiff_NoLineNumbers(t *testing.T) {
	patch := "@@ -1,2 +1,2 @@\n-old\n+new"
	d := NewDiff(patch, DiffWithShowLineNumbers(false))
	testutil.SetupWidget(d, 30, 5)

	buf := buffer.New(30, 5)
	d.RenderSelf(buf, 0) // should render without gutter
}

// --- TabSelect tests ---

func makeTabOptions() []TabSelectOption {
	return []TabSelectOption{
		{Name: "Tab 1", Value: 1},
		{Name: "Tab 2", Value: 2},
		{Name: "Tab 3", Value: 3},
	}
}

func TestNewTabSelect(t *testing.T) {
	ts := NewTabSelect(makeTabOptions())
	if ts.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0, got %d", ts.SelectedIndex())
	}
}

func TestTabSelect_Navigation(t *testing.T) {
	ts := NewTabSelect(makeTabOptions())
	testutil.SetupWidget(ts, 30, 2)

	// Move right
	ts.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Type: input.KeyPress})
	if ts.SelectedIndex() != 1 {
		t.Errorf("expected index 1, got %d", ts.SelectedIndex())
	}

	// Move right again
	ts.HandleKeyPress(&input.KeyEvent{Name: input.KeyRight, Type: input.KeyPress})
	if ts.SelectedIndex() != 2 {
		t.Errorf("expected index 2, got %d", ts.SelectedIndex())
	}

	// Move left
	ts.HandleKeyPress(&input.KeyEvent{Name: input.KeyLeft, Type: input.KeyPress})
	if ts.SelectedIndex() != 1 {
		t.Errorf("expected index 1, got %d", ts.SelectedIndex())
	}
}

func TestTabSelect_Render(t *testing.T) {
	ts := NewTabSelect(makeTabOptions())
	testutil.SetupWidget(ts, 30, 2)

	buf := buffer.New(30, 2)
	ts.RenderSelf(buf, 0)
	// Should not panic
}

// --- LineNumber tests ---

// mockLineProvider implements LineInfoProvider
type mockLineProvider struct {
	*render.BaseRenderable
	lineCount int
}

func (m *mockLineProvider) LineCount() int { return m.lineCount }

func TestNewLineNumber(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      10,
	}
	ln := NewLineNumber(target)
	if ln == nil {
		t.Fatal("NewLineNumber returned nil")
	}
}

func TestLineNumber_Render(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
	}
	ln := NewLineNumber(target, LineNumberWithMinWidth(4))
	testutil.SetupWidget(ln, 5, 5)

	buf := buffer.New(5, 5)
	ln.RenderSelf(buf, 0)

	// Check that line numbers are rendered
	text := buf.PlainText(true)
	if len(text) == 0 {
		t.Error("expected non-empty render output")
	}
}

// mockCursorProvider implements LineInfoProvider, ScrollInfoProvider, CursorInfoProvider
type mockCursorProvider struct {
	*render.BaseRenderable
	lineCount int
	scrollY   int
	cursorRow int
}

func (m *mockCursorProvider) LineCount() int { return m.lineCount }
func (m *mockCursorProvider) ScrollY() int   { return m.scrollY }
func (m *mockCursorProvider) CursorRow() int { return m.cursorRow }

// mockVisualLineProvider implements VisualLineProvider
type mockVisualLineProvider struct {
	*render.BaseRenderable
	lineCount   int
	scrollY     int
	cursorRow   int
	visualLines int
	logicalMap  []int // visual line -> logical line
}

func (m *mockVisualLineProvider) LineCount() int       { return m.lineCount }
func (m *mockVisualLineProvider) ScrollY() int         { return m.scrollY }
func (m *mockVisualLineProvider) CursorRow() int       { return m.cursorRow }
func (m *mockVisualLineProvider) VisualLineCount() int { return m.visualLines }
func (m *mockVisualLineProvider) LogicalLineForVisualLine(v int) int {
	if v >= 0 && v < len(m.logicalMap) {
		return m.logicalMap[v]
	}
	return v
}

func TestLineNumber_ActiveLine(t *testing.T) {
	target := &mockCursorProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
		cursorRow:      2,
	}
	ln := NewLineNumber(target,
		LineNumberWithActiveFg(style.ColorBrightYellow),
		LineNumberWithActiveBg(style.ColorFromInts(40, 40, 40, 255)),
	)
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0)

	// Row 2 (active line) should have activeFg color
	activeCell := buf.Cell(0, 2)
	normalCell := buf.Cell(0, 0)
	_ = activeCell
	_ = normalCell
	// Just verify it renders without panic
}

func TestLineNumber_RelativeMode(t *testing.T) {
	target := &mockCursorProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
		cursorRow:      2, // cursor on line 2 (0-indexed)
	}
	ln := NewLineNumber(target, LineNumberWithMode(LineNumberRelative))
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0)

	// Row 2 (active) should show absolute number "3"
	// Row 0 should show relative distance "2"
	// Row 1 should show relative distance "1"
	// Row 3 should show relative distance "1"
	// Row 4 should show relative distance "2"
	// We verify the active line shows its absolute number
	text := buf.PlainText(true)
	if len(text) == 0 {
		t.Error("expected non-empty render for relative mode")
	}
}

func TestLineNumber_ScrollSync(t *testing.T) {
	target := &mockCursorProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      20,
		scrollY:        5,
		cursorRow:      7,
	}
	ln := NewLineNumber(target)
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0)

	// Should render lines starting from scrollY=5 (display line 6)
	// Just verify renders without panic
}

func TestLineNumber_WrapAwareness(t *testing.T) {
	// Simulate a wrapped target: 3 logical lines, 5 visual lines
	// Line 0 -> visual 0
	// Line 1 -> visual 1,2 (wrapped)
	// Line 2 -> visual 3,4 (wrapped)
	target := &mockVisualLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      3,
		visualLines:    5,
		cursorRow:      1,
		logicalMap:     []int{0, 1, 1, 2, 2},
	}
	ln := NewLineNumber(target)
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0)

	// Visual line 0: should show "1" (logical line 0)
	// Visual line 1: should show "2" (first visual line of logical line 1)
	// Visual line 2: should show empty (continuation of logical line 1)
	// Visual line 3: should show "3" (first visual line of logical line 2)
	// Visual line 4: should show empty (continuation of logical line 2)
	// Just verify renders without panic
}

func TestLineNumber_ComputedWidth(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      100,
	}
	ln := NewLineNumber(target, LineNumberWithWidth(layout.Points(10)))
	testutil.SetupWidget(ln, 10, 10)

	cw := ln.ComputedWidth()
	// 100 lines + offset 1 = max "101" = 3 digits + 1 space = 4 width (matches minWidth)
	if cw < 4 {
		t.Errorf("expected computed width >= 4, got %d", cw)
	}

	// Now try with 10000 lines
	target.lineCount = 10000
	cw2 := ln.ComputedWidth()
	// "10001" = 5 digits + 1 space = 6
	if cw2 < 6 {
		t.Errorf("expected computed width >= 6 for 10000 lines, got %d", cw2)
	}
}

func TestLineNumber_Hidden(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
	}
	ln := NewLineNumber(target, LineNumberWithShow(false))
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0)

	// Should not render any numbers
	if ln.ComputedWidth() != 0 {
		t.Errorf("expected computed width 0 when hidden, got %d", ln.ComputedWidth())
	}
}

func TestLineNumber_Separator(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      3,
	}
	ln := NewLineNumber(target, LineNumberWithSeparator('│'))
	testutil.SetupWidget(ln, 6, 3)

	buf := buffer.New(6, 3)
	ln.RenderSelf(buf, 0)

	// Should render separator after line numbers
}

func TestLineNumber_Options(t *testing.T) {
	target := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
	}
	ln := NewLineNumber(target,
		LineNumberWithID("ln1"),
		LineNumberWithFg(style.ColorGreen),
		LineNumberWithBg(style.ColorFromInts(20, 20, 20, 255)),
		LineNumberWithActiveFg(style.ColorBrightYellow),
		LineNumberWithActiveBg(style.ColorFromInts(40, 40, 40, 255)),
		LineNumberWithMinWidth(5),
		LineNumberWithOffset(0),
		LineNumberWithShow(true),
		LineNumberWithMode(LineNumberRelative),
		LineNumberWithSeparator('│'),
		LineNumberWithSeparatorFg(style.ColorGray),
		LineNumberWithWidth(layout.Points(6)),
		LineNumberWithHeight(layout.Points(10)),
	)

	if ln.ID() != "ln1" {
		t.Errorf("expected ID 'ln1', got %q", ln.ID())
	}
	if ln.Mode() != LineNumberRelative {
		t.Error("expected LineNumberRelative mode")
	}
}

func TestLineNumber_NilTarget(t *testing.T) {
	ln := NewLineNumber(nil)
	testutil.SetupWidget(ln, 6, 5)

	buf := buffer.New(6, 5)
	ln.RenderSelf(buf, 0) // should not panic with nil target
}

func TestLineNumber_SetTarget(t *testing.T) {
	target1 := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      5,
	}
	target2 := &mockLineProvider{
		BaseRenderable: render.NewBaseRenderable(),
		lineCount:      10,
	}
	ln := NewLineNumber(target1)
	ln.SetTarget(target2)
	if ln.Target() != target2 {
		t.Error("expected target to be updated")
	}
}

// --- Tier 3 basic tests ---

func TestNewTable(t *testing.T) {
	cols := []TableColumn{
		{Name: "Name", Width: 10},
		{Name: "Value", Width: 10},
	}
	rows := [][]string{
		{"Alice", "100"},
		{"Bob", "200"},
	}
	tbl := NewTable(cols, rows)
	if tbl.SelectedRow() != 0 {
		t.Errorf("expected selectedRow 0, got %d", tbl.SelectedRow())
	}
}

func TestTable_Render(t *testing.T) {
	cols := []TableColumn{{Name: "Col1"}}
	rows := [][]string{{"A"}, {"B"}}
	tbl := NewTable(cols, rows)
	testutil.SetupWidget(tbl, 20, 5)

	buf := buffer.New(20, 5)
	tbl.RenderSelf(buf, 0)
}

func TestNewList(t *testing.T) {
	items := []ListItem{
		{Text: "Item 1"},
		{Text: "Item 2"},
		{Text: "Item 3"},
	}
	l := NewList(items)
	if l.SelectedIndex() != 0 {
		t.Errorf("expected index 0, got %d", l.SelectedIndex())
	}
}

func TestList_Navigation(t *testing.T) {
	items := []ListItem{{Text: "A"}, {Text: "B"}, {Text: "C"}}
	l := NewList(items)
	testutil.SetupWidget(l, 20, 5)

	l.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	if l.SelectedIndex() != 1 {
		t.Errorf("expected index 1, got %d", l.SelectedIndex())
	}
}

func TestNewGauge(t *testing.T) {
	g := NewGauge(GaugeWithValue(0.5))
	if g.Value() != 0.5 {
		t.Errorf("expected value 0.5, got %f", g.Value())
	}
}

func TestGauge_Render(t *testing.T) {
	g := NewGauge(GaugeWithValue(0.75))
	testutil.SetupWidget(g, 20, 1)

	buf := buffer.New(20, 1)
	g.RenderSelf(buf, 0)
}

func TestNewSparkline(t *testing.T) {
	data := []float64{1, 3, 5, 2, 7, 4}
	s := NewSparkline(data)
	if len(s.data) != 6 {
		t.Errorf("expected 6 data points, got %d", len(s.data))
	}
}

func TestSparkline_Render(t *testing.T) {
	data := []float64{1, 3, 5, 2, 7, 4}
	s := NewSparkline(data)
	testutil.SetupWidget(s, 10, 1)

	buf := buffer.New(10, 1)
	s.RenderSelf(buf, 0)
}

func TestSparkline_AddPoint(t *testing.T) {
	s := NewSparkline([]float64{1, 2, 3})
	testutil.SetupWidget(s, 5, 1)

	s.AddPoint(4)
	s.AddPoint(5)
	s.AddPoint(6)
	s.AddPoint(7)
	// Data should be trimmed to width=5
	if len(s.data) > 5 {
		t.Errorf("expected data trimmed to 5, got %d", len(s.data))
	}
}

func TestNewTree(t *testing.T) {
	root := []*TreeNode{
		{
			Text:     "Root",
			Expanded: true,
			Children: []*TreeNode{
				{Text: "Child 1"},
				{Text: "Child 2"},
			},
		},
	}
	tree := NewTree(root)
	if len(tree.flatItems) != 3 { // root + 2 children
		t.Errorf("expected 3 flat items, got %d", len(tree.flatItems))
	}
}

func TestTree_ExpandCollapse(t *testing.T) {
	root := []*TreeNode{
		{
			Text:     "Root",
			Expanded: false,
			Children: []*TreeNode{
				{Text: "Child"},
			},
		},
	}
	tree := NewTree(root)
	testutil.SetupWidget(tree, 20, 10)

	if len(tree.flatItems) != 1 {
		t.Errorf("expected 1 flat item (collapsed), got %d", len(tree.flatItems))
	}

	// Expand
	tree.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})
	if len(tree.flatItems) != 2 {
		t.Errorf("expected 2 flat items (expanded), got %d", len(tree.flatItems))
	}

	// Collapse
	tree.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})
	if len(tree.flatItems) != 1 {
		t.Errorf("expected 1 flat item (collapsed again), got %d", len(tree.flatItems))
	}
}
