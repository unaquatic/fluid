package widget

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/unaquatic/fluid/style"
)

func TestChromaStyleFromTheme_CorePaletteMapsToTokens(t *testing.T) {
	th := style.DefaultDark()
	// DefaultDark populates Code; make every role unique so we can
	// assert the exact field each token type uses.
	th.Code.Keyword = style.ColorFromInts(10, 10, 10, 255)
	th.Code.Type = style.ColorFromInts(20, 20, 20, 255)
	th.Code.Function = style.ColorFromInts(30, 30, 30, 255)
	th.Code.String = style.ColorFromInts(40, 40, 40, 255)
	th.Code.Number = style.ColorFromInts(50, 50, 50, 255)
	th.Code.Comment = style.ColorFromInts(60, 60, 60, 255)
	th.Code.Operator = style.ColorFromInts(70, 70, 70, 255)
	th.Code.Constant = style.ColorFromInts(80, 80, 80, 255)
	th.Code.Decorator = style.ColorFromInts(90, 90, 90, 255)

	s := ChromaStyleFromTheme(th)

	cases := []struct {
		token  chroma.TokenType
		expect style.Color
	}{
		{chroma.Keyword, th.Code.Keyword},
		{chroma.KeywordType, th.Code.Type},
		{chroma.KeywordConstant, th.Code.Constant},
		{chroma.NameFunction, th.Code.Function},
		{chroma.NameClass, th.Code.Type},
		{chroma.NameConstant, th.Code.Constant},
		{chroma.NameDecorator, th.Code.Decorator},
		{chroma.LiteralString, th.Code.String},
		{chroma.LiteralNumber, th.Code.Number},
		{chroma.Comment, th.Code.Comment},
		{chroma.Operator, th.Code.Operator},
		{chroma.OperatorWord, th.Code.Keyword},
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
	th.Code.Comment = style.ColorFromInts(99, 99, 99, 255)
	th.Code.CommentAttrs = style.AttrItalic

	s := ChromaStyleFromTheme(th)
	entry := s.Get(chroma.Comment)
	if entry.Italic != chroma.Yes {
		t.Errorf("expected chroma italic=Yes for Comment with AttrItalic, got %v", entry.Italic)
	}
}

func TestChromaStyleFromTheme_InheritanceFallbacks(t *testing.T) {
	// Leave DiffAdded / DiffRemoved / Tag / Attribute / Preproc /
	// StringEscape zero to verify they fall back to parent roles.
	th := style.DefaultDark()
	th.Code.DiffAdded = style.Color{}
	th.Code.DiffRemoved = style.Color{}
	th.Code.Tag = style.Color{}
	th.Code.Attribute = style.Color{}
	th.Code.Preproc = style.Color{}
	th.Code.StringEscape = style.Color{}

	s := ChromaStyleFromTheme(th)

	cases := []struct {
		name   string
		token  chroma.TokenType
		expect style.Color
	}{
		{"DiffAdded → Success", chroma.GenericInserted, th.Success},
		{"DiffRemoved → Danger", chroma.GenericDeleted, th.Danger},
		{"Tag → Keyword", chroma.NameTag, th.Code.Keyword},
		{"Attribute → Variable", chroma.NameAttribute, style.OrInherit(th.Code.Variable, th.TextStrong)},
		{"Preproc → Comment", chroma.CommentPreproc, th.Code.Comment},
		{"StringEscape → Number", chroma.LiteralStringEscape, th.Code.Number},
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
	// Chroma's Svelte lexer (github.com/alecthomas/chroma v2.23.1) emits
	// chroma.Error for the '>' that closes an opening tag containing a
	// template expression (e.g. `<button on:click={() => count++}>`).
	// The renderer must neutralise Error tokens so malformed-looking
	// highlighting (underline, etc.) doesn't leak into well-formed source.
	//
	// Build a highlighter whose style actively marks chroma.Error with
	// bold+underline so we can observe the neutralisation working.
	defaultFg := style.ColorFromInts(200, 200, 200, 255)
	defaultBg := style.ColorFromInts(30, 30, 30, 255)
	th := style.DefaultDark()
	th.Code.ErrorAttrs = style.AttrBold | style.AttrUnderline
	h := NewChromaHighlighterWithStyle(ChromaStyleFromTheme(th))

	src := "<button on:click={() => count++}>\n  Count: {count}\n</button>\n"
	lines, err := h.Highlight(src, "svelte", defaultFg, defaultBg)
	if err != nil {
		t.Fatalf("Highlight returned error: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("Highlight returned no lines")
	}

	// Walk every span on every line; none may carry the Error signature
	// (bold+underline together is unique to ErrorAttrs in this theme).
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
	th.Code.Keyword = style.ColorFromInts(11, 22, 33, 255)

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
	want := th.Code.Keyword.ToHex()
	if got != want {
		t.Errorf("derived highlighter not using Theme.Code.Keyword: got %s want %s", got, want)
	}
}
