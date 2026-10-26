package widget

import (
	"strings"
	"sync"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/syntax"
	"github.com/unaquatic/fluid/textbuf"
)

// SyntaxHighlighter converts source code into styled logical lines.
// Implementations may cache parser and style state. Configure themes through
// the concrete implementation's constructor or methods.
type SyntaxHighlighter interface {
	// Highlight returns one styled Line per logical line in content.
	// filetype identifies the source language, such as "go" or "python".
	// defaultFg and defaultBg apply when a token has no explicit color.
	Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error)
}

// defaultTheme returns the theme that widgets color code with when they are
// given none. It is shared, so it must not be changed.
var defaultTheme = sync.OnceValue(style.DefaultDark)

// Highlighter implements SyntaxHighlighter with the lexers of package
// syntax, styling each kind of token with a role of a theme's code palette.
type Highlighter struct {
	// Styles holds the style of each kind of token, indexed by kind. A token
	// whose style has no foreground takes the default foreground Highlight
	// is given, and every token takes the default background.
	Styles [syntax.NumKinds]style.Entry
}

// NewHighlighter returns a Highlighter that styles tokens with the code
// palette of t, as TokenStyles does. A nil t selects style.DefaultDark.
func NewHighlighter(t *style.Theme) *Highlighter {
	h := &Highlighter{}
	h.SetTheme(t)
	return h
}

// SetTheme styles tokens with the code palette of t, as TokenStyles does. A
// nil t selects style.DefaultDark.
func (h *Highlighter) SetTheme(t *style.Theme) {
	if t == nil {
		t = defaultTheme()
	}
	h.Styles = TokenStyles(t)
}

// TokenStyles returns the style of each kind of token under the code
// palette of t. An optional role that t leaves unset inherits the role its
// documentation names, and a role without a foreground inherits that of
// Variable, or of t.TextStrong if Variable has none, so that every token
// of a theme has a color.
func TokenStyles(t *style.Theme) [syntax.NumKinds]style.Entry {
	c := t.Code
	variable := c.Variable.Or(style.Entry{Fg: t.TextStrong})
	var s [syntax.NumKinds]style.Entry
	s[syntax.Text] = variable
	s[syntax.Keyword] = c.Keyword
	s[syntax.Type] = c.Type
	s[syntax.Function] = c.Function
	s[syntax.BuiltinType] = c.BuiltinType.Or(c.Type)
	s[syntax.BuiltinFunction] = c.BuiltinFunction.Or(c.Function)
	s[syntax.BuiltinVariable] = c.BuiltinVariable.Or(variable)
	s[syntax.Variable] = variable
	s[syntax.Constant] = c.Constant
	s[syntax.Primitive] = c.Primitive.Or(c.Constant)
	s[syntax.Property] = c.Property.Or(variable)
	s[syntax.Attribute] = c.Attribute.Or(variable)
	s[syntax.Tag] = c.Tag.Or(c.Keyword)
	s[syntax.Decorator] = c.Decorator
	s[syntax.String] = c.String
	s[syntax.StringEscape] = c.StringEscape.Or(c.Number)
	s[syntax.Regex] = c.Regex.Or(c.String)
	s[syntax.Number] = c.Number
	s[syntax.Comment] = c.Comment
	s[syntax.Preproc] = c.Preproc.Or(c.Comment)
	s[syntax.Operator] = c.Operator
	s[syntax.Punctuation] = c.Punctuation
	s[syntax.Heading] = t.ResolveMarkdownHeading(1)
	s[syntax.Subheading] = variable
	s[syntax.Emph] = t.ResolveMarkdownEmph()
	s[syntax.Strong] = t.ResolveMarkdownStrong()
	s[syntax.Inserted] = c.DiffAdded.Or(style.Entry{Fg: t.Success})
	s[syntax.Deleted] = c.DiffRemoved.Or(style.Entry{Fg: t.Danger})
	for k := range s {
		s[k].Fg = style.OrInherit(s[k].Fg, variable.Fg)
	}
	return s
}

// Highlight tokenizes content in the language filetype names, as
// syntax.Lookup finds it, and returns one styled Line per logical line. A
// language Lookup does not find leaves content Text. It never returns an
// error.
func (h *Highlighter) Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error) {
	lang := syntax.Lookup(filetype)
	lines := make([]textbuf.Line, 0, strings.Count(content, "\n")+1)
	var (
		st   syntax.State
		toks []syntax.Token
	)
	for text := range strings.SplitSeq(content, "\n") {
		toks = toks[:0]
		switch {
		case lang != nil:
			toks, st = lang.AppendLine(toks, text, st)
		case text != "":
			toks = append(toks, syntax.Token{Kind: syntax.Text, Text: text})
		}
		lines = append(lines, h.Line(text, toks, defaultFg, defaultBg))
	}
	return lines, nil
}

// Line returns text as a styled Line, with adjacent tokens of the same
// style in one span. toks must be the tokens a syntax.Lexer appends for
// text, whose texts concatenate to it; the spans' texts are cut from text by
// the tokens' lengths.
func (h *Highlighter) Line(text string, toks []syntax.Token, defaultFg, defaultBg style.Color) textbuf.Line {
	spans := make([]textbuf.Span, 0, len(toks))
	start, pos := 0, 0
	for _, t := range toks {
		e := h.Styles[t.Kind]
		fg := style.OrInherit(e.Fg, defaultFg)
		end := min(pos+len(t.Text), len(text))
		if n := len(spans); n > 0 && spans[n-1].Fg == fg && spans[n-1].Attributes == e.Attrs {
			spans[n-1].Text = text[start:end]
		} else {
			start = pos
			spans = append(spans, textbuf.Span{Text: text[pos:end], Fg: fg, Bg: defaultBg, Attributes: e.Attrs})
		}
		pos = end
	}
	return textbuf.Line{Spans: spans}
}
