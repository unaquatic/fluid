package widget

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/unaquatic/fluid/style"
)

func TestChromaStyleFromTheme_CorePaletteMapsToTokens(t *testing.T) {
	th := style.DefaultDark()
	// Unique role colors reveal exactly which fallback each token uses.
	th.Code.Keyword = style.Entry{Fg: style.ColorFromInts(10, 10, 10, 255)}
	th.Code.Type = style.Entry{Fg: style.ColorFromInts(20, 20, 20, 255)}
	th.Code.Function = style.Entry{Fg: style.ColorFromInts(30, 30, 30, 255)}
	th.Code.String = style.Entry{Fg: style.ColorFromInts(40, 40, 40, 255)}
	th.Code.Number = style.Entry{Fg: style.ColorFromInts(50, 50, 50, 255)}
	th.Code.Comment = style.Entry{Fg: style.ColorFromInts(60, 60, 60, 255)}
	th.Code.Operator = style.Entry{Fg: style.ColorFromInts(70, 70, 70, 255)}
	th.Code.Constant = style.Entry{Fg: style.ColorFromInts(80, 80, 80, 255)}
	th.Code.Decorator = style.Entry{Fg: style.ColorFromInts(90, 90, 90, 255)}

	s := ChromaStyleFromTheme(th)

	cases := []struct {
		token  chroma.TokenType
		expect style.Color
	}{
		{chroma.Keyword, th.Code.Keyword.Fg},
		{chroma.KeywordType, th.Code.Type.Fg},
		{chroma.KeywordConstant, th.Code.Constant.Fg},
		{chroma.NameFunction, th.Code.Function.Fg},
		{chroma.NameClass, th.Code.Type.Fg},
		{chroma.NameConstant, th.Code.Constant.Fg},
		{chroma.NameDecorator, th.Code.Decorator.Fg},
		{chroma.LiteralString, th.Code.String.Fg},
		{chroma.LiteralNumber, th.Code.Number.Fg},
		{chroma.Comment, th.Code.Comment.Fg},
		{chroma.Operator, th.Code.Operator.Fg},
		{chroma.OperatorWord, th.Code.Keyword.Fg},
	}
	for _, c := range cases {
		entry := s.Get(c.token)
		if !entry.Colour.IsSet() {
			t.Errorf("token %s: chroma Colour not set", c.token)
			continue
		}
		gotHex := "#" + entry.Colour.String()[1:] // chroma colour renders as #RRGGBB already
		wantHex := c.expect.ToHex()
		if gotHex != wantHex {
			t.Errorf("token %s: got %s want %s", c.token, gotHex, wantHex)
		}
	}
}

func TestChromaStyleFromTheme_CommentItalicFromAttrs(t *testing.T) {
	th := style.DefaultDark()
	th.Code.Comment = style.Entry{Fg: style.ColorFromInts(99, 99, 99, 255)}
	th.Code.Comment.Attrs = style.AttrItalic

	s := ChromaStyleFromTheme(th)
	entry := s.Get(chroma.Comment)
	if entry.Italic != chroma.Yes {
		t.Errorf("expected chroma italic=Yes for Comment with AttrItalic, got %v", entry.Italic)
	}
}

func TestChromaStyleFromTheme_InheritanceFallbacks(t *testing.T) {
	// Leave selected roles unset to exercise their documented fallbacks.
	th := style.DefaultDark()
	th.Code.DiffAdded = style.Entry{}
	th.Code.DiffRemoved = style.Entry{}
	th.Code.Tag = style.Entry{}
	th.Code.Attribute = style.Entry{}
	th.Code.Preproc = style.Entry{}
	th.Code.StringEscape = style.Entry{}

	s := ChromaStyleFromTheme(th)

	cases := []struct {
		name   string
		token  chroma.TokenType
		expect style.Color
	}{
		{"DiffAdded → Success", chroma.GenericInserted, th.Success},
		{"DiffRemoved → Danger", chroma.GenericDeleted, th.Danger},
		{"Tag → Keyword", chroma.NameTag, th.Code.Keyword.Fg},
		{"Attribute → Variable", chroma.NameAttribute, th.Code.Variable.Or(style.Entry{Fg: th.TextStrong}).Fg},
		{"Preproc → Comment", chroma.CommentPreproc, th.Code.Comment.Fg},
		{"StringEscape → Number", chroma.LiteralStringEscape, th.Code.Number.Fg},
	}
	for _, c := range cases {
		entry := s.Get(c.token)
		if !entry.Colour.IsSet() {
			t.Errorf("%s: chroma Colour not set", c.name)
			continue
		}
		got := "#" + entry.Colour.String()[1:]
		want := c.expect.ToHex()
		if got != want {
			t.Errorf("%s: got %s want %s", c.name, got, want)
		}
	}
}

func TestChromaHighlighter_SvelteErrorTokenRendersAsPlainText(t *testing.T) {
	// Chroma's Svelte lexer can classify valid template syntax as Error.
	// Give Error a unique bold-underlined style so neutralization is observable.
	defaultFg := style.ColorFromInts(200, 200, 200, 255)
	defaultBg := style.ColorFromInts(30, 30, 30, 255)
	th := style.DefaultDark()
	th.Code.Error.Attrs = style.AttrBold | style.AttrUnderline
	h := NewChromaHighlighterWithStyle(ChromaStyleFromTheme(th))

	src := "<button on:click={() => count++}>\n  Count: {count}\n</button>\n"
	lines, err := h.Highlight(src, "svelte", defaultFg, defaultBg)
	if err != nil {
		t.Fatalf("Highlight returned error: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("Highlight returned no lines")
	}

	// No output span may retain the unique Error attribute combination.
	errMask := style.AttrBold | style.AttrUnderline
	for y, line := range lines {
		for x, sp := range line.Spans {
			if sp.Attributes&errMask == errMask {
				t.Errorf("line %d span %d (%q): carries bold+underline; Error token was not neutralised", y, x, sp.Text)
			}
		}
	}
}

func TestMarkdownWithTheme_InstallsDerivedHighlighter(t *testing.T) {
	th := style.DefaultDark()
	th.Code.Keyword = style.Entry{Fg: style.ColorFromInts(11, 22, 33, 255)}

	m := NewMarkdown("```go\nfunc f() {}\n```", MarkdownWithTheme(th))
	ch, ok := m.highlighter.(*ChromaHighlighter)
	if !ok {
		t.Fatalf("expected ChromaHighlighter, got %T", m.highlighter)
	}
	entry := ch.chromaStyle.Get(chroma.Keyword)
	if !entry.Colour.IsSet() {
		t.Fatal("derived style Keyword colour not set")
	}
	got := "#" + entry.Colour.String()[1:]
	want := th.Code.Keyword.Fg.ToHex()
	if got != want {
		t.Errorf("derived highlighter not using Theme.Code.Keyword: got %s want %s", got, want)
	}
}
