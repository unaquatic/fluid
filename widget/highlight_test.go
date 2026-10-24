package widget

import (
	"testing"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/syntax"
	"github.com/unaquatic/fluid/textbuf"
)

func TestTokenStyles_CorePaletteMapsToKinds(t *testing.T) {
	th := style.DefaultDark()
	// Unique role colors reveal exactly which role each kind uses.
	th.Code.Keyword = style.Entry{Fg: style.ColorFromInts(10, 10, 10, 255)}
	th.Code.Type = style.Entry{Fg: style.ColorFromInts(20, 20, 20, 255)}
	th.Code.Function = style.Entry{Fg: style.ColorFromInts(30, 30, 30, 255)}
	th.Code.String = style.Entry{Fg: style.ColorFromInts(40, 40, 40, 255)}
	th.Code.Number = style.Entry{Fg: style.ColorFromInts(50, 50, 50, 255)}
	th.Code.Comment = style.Entry{Fg: style.ColorFromInts(60, 60, 60, 255), Attrs: style.AttrItalic}
	th.Code.Operator = style.Entry{Fg: style.ColorFromInts(70, 70, 70, 255)}
	th.Code.Constant = style.Entry{Fg: style.ColorFromInts(80, 80, 80, 255)}
	th.Code.Decorator = style.Entry{Fg: style.ColorFromInts(90, 90, 90, 255)}

	s := TokenStyles(th)
	tests := []struct {
		kind syntax.Kind
		want style.Entry
	}{
		{syntax.Keyword, th.Code.Keyword},
		{syntax.Type, th.Code.Type},
		{syntax.Function, th.Code.Function},
		{syntax.String, th.Code.String},
		{syntax.Number, th.Code.Number},
		{syntax.Comment, th.Code.Comment},
		{syntax.Operator, th.Code.Operator},
		{syntax.Constant, th.Code.Constant},
		{syntax.Decorator, th.Code.Decorator},
	}
	for _, tt := range tests {
		if got := s[tt.kind]; got != tt.want {
			t.Errorf("%v: got %+v, want %+v", tt.kind, got, tt.want)
		}
	}
}

func TestTokenStyles_InheritanceFallbacks(t *testing.T) {
	// Leave optional roles unset to exercise their documented fallbacks.
	th := style.DefaultDark()
	th.Code.DiffAdded = style.Entry{}
	th.Code.DiffRemoved = style.Entry{}
	th.Code.Tag = style.Entry{}
	th.Code.Attribute = style.Entry{}
	th.Code.Property = style.Entry{}
	th.Code.Preproc = style.Entry{}
	th.Code.StringEscape = style.Entry{}
	th.Code.Regex = style.Entry{}
	th.Code.Primitive = style.Entry{}
	variable := th.Code.Variable.Or(style.Entry{Fg: th.TextStrong})

	s := TokenStyles(th)
	tests := []struct {
		name string
		kind syntax.Kind
		want style.Color
	}{
		{"Inserted → Success", syntax.Inserted, th.Success},
		{"Deleted → Danger", syntax.Deleted, th.Danger},
		{"Tag → Keyword", syntax.Tag, th.Code.Keyword.Fg},
		{"Attribute → Variable", syntax.Attribute, variable.Fg},
		{"Property → Variable", syntax.Property, variable.Fg},
		{"Preproc → Comment", syntax.Preproc, th.Code.Comment.Fg},
		{"StringEscape → Number", syntax.StringEscape, th.Code.Number.Fg},
		{"Regex → String", syntax.Regex, th.Code.String.Fg},
		{"Primitive → Constant", syntax.Primitive, th.Code.Constant.Fg},
		{"Text → Variable", syntax.Text, variable.Fg},
	}
	for _, tt := range tests {
		if got := s[tt.kind].Fg; got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got.ToHex(), tt.want.ToHex())
		}
	}
}

func TestTokenStyles_UnsetRoleTakesVariableColor(t *testing.T) {
	th := style.DefaultDark()
	th.Code.Keyword = style.Entry{Attrs: style.AttrBold}

	s := TokenStyles(th)
	if got, want := s[syntax.Keyword], (style.Entry{Fg: th.Code.Variable.Fg, Attrs: style.AttrBold}); got != want {
		t.Errorf("Keyword: got %+v, want %+v", got, want)
	}
}

func TestHighlighter_Highlight(t *testing.T) {
	th := style.DefaultDark()
	fg, bg := style.ColorFromInts(1, 2, 3, 255), style.ColorFromInts(4, 5, 6, 255)
	h := NewHighlighter(th)

	lines, err := h.Highlight("func main() {}\n\nx := 1", "go", fg, bg)
	if err != nil {
		t.Fatalf("Highlight: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	first := lines[0].Spans[0]
	if first.Text != "func" || first.Fg != th.Code.Keyword.Fg || first.Bg != bg {
		t.Errorf("first span = %+v, want func in the keyword color on bg", first)
	}
	if lines[1].Spans == nil || len(lines[1].Spans) != 0 {
		t.Errorf("empty line spans = %#v, want empty and not nil", lines[1].Spans)
	}
	for i, line := range lines {
		for j := 1; j < len(line.Spans); j++ {
			a, b := line.Spans[j-1], line.Spans[j]
			if a.Fg == b.Fg && a.Attributes == b.Attributes {
				t.Errorf("line %d: spans %q and %q share a style but were not merged", i, a.Text, b.Text)
			}
		}
	}
}

func TestHighlighter_UnknownLanguageIsText(t *testing.T) {
	th := style.DefaultDark()
	fg, bg := style.ColorWhite, style.ColorBlack
	lines, _ := NewHighlighter(th).Highlight("a b\nc", "nonexistent_language_xyz", fg, bg)
	want := []textbuf.Line{
		{Spans: []textbuf.Span{{Text: "a b", Fg: th.Code.Variable.Fg, Bg: bg}}},
		{Spans: []textbuf.Span{{Text: "c", Fg: th.Code.Variable.Fg, Bg: bg}}},
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i := range want {
		if len(lines[i].Spans) != 1 || lines[i].Spans[0] != want[i].Spans[0] {
			t.Errorf("line %d = %+v, want %+v", i, lines[i].Spans, want[i].Spans)
		}
	}
}

func TestHighlighter_ZeroStylesUseDefaults(t *testing.T) {
	fg, bg := style.ColorFromInts(9, 9, 9, 255), style.ColorFromInts(8, 8, 8, 255)
	lines, _ := (&Highlighter{}).Highlight("if x {}", "go", fg, bg)
	if len(lines) != 1 || len(lines[0].Spans) != 1 {
		t.Fatalf("got %+v, want one span in the defaults", lines)
	}
	if sp := lines[0].Spans[0]; sp.Text != "if x {}" || sp.Fg != fg || sp.Bg != bg {
		t.Errorf("span = %+v", sp)
	}
}

func TestMarkdownWithTheme_InstallsThemeHighlighter(t *testing.T) {
	th := style.DefaultDark()
	th.Code.Keyword = style.Entry{Fg: style.ColorFromInts(11, 22, 33, 255)}

	m := NewMarkdown("```go\nfunc f() {}\n```", MarkdownWithTheme(th))
	h, ok := m.highlighter.(*Highlighter)
	if !ok {
		t.Fatalf("expected *Highlighter, got %T", m.highlighter)
	}
	if got := h.Styles[syntax.Keyword].Fg; got != th.Code.Keyword.Fg {
		t.Errorf("highlighter keyword color = %s, want %s", got.ToHex(), th.Code.Keyword.Fg.ToHex())
	}
}

func TestCode_SetThemeRestylesTokens(t *testing.T) {
	monokai, darcula := style.DefaultDark(), style.DarculaSolid()
	c := NewCode("func main() {}", CodeWithFiletype("go"), CodeWithTheme(monokai))
	if got := c.LineHighlights(0)[0]; *got.Fg != monokai.Code.Keyword.Fg {
		t.Fatalf("keyword fg = %s, want %s", got.Fg.ToHex(), monokai.Code.Keyword.Fg.ToHex())
	}
	c.SetTheme(darcula)
	if got := c.LineHighlights(0)[0]; *got.Fg != darcula.Code.Keyword.Fg {
		t.Errorf("keyword fg after SetTheme = %s, want %s", got.Fg.ToHex(), darcula.Code.Keyword.Fg.ToHex())
	}
}
