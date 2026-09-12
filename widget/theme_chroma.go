package widget

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/unaquatic/fluid/style"
)

// ChromaStyleFromTheme builds a *chroma.Style whose colors come from the
// given Theme's CodePalette. The result can be plugged into a
// ChromaHighlighter via NewChromaHighlighterWithStyle so syntax
// highlighting stays in harmony with the rest of the UI palette.
//
// When a CodePalette field is unset (zero-alpha Color), a sensible parent
// role is used (see the OrInherit calls below). That way callers can
// specify just a handful of roles and still get full coverage.
//
// Chroma's inheritance tree means we only need to paint ~15 top-level
// token types; the library propagates them to every child (e.g.
// KeywordDeclaration inherits from Keyword, LiteralStringDouble from
// LiteralString, CommentMultiline from Comment, etc.).
func ChromaStyleFromTheme(t *style.Theme) *chroma.Style {
	c := t.Code

	variable := c.Variable.Or(style.Entry{Fg: t.TextStrong})

	entries := chroma.StyleEntries{
		// Base — sets the fallback fg/bg for tokens that don't inherit
		// a more specific color. The Background entry's bg feeds into
		// what Chroma reports as the theme bg, but we also override
		// per-token bg to t.CodeBg in resolveTokenStyle, so this is
		// mostly cosmetic.
		chroma.Background: spec(variable.WithBg(t.CodeBg)),
		chroma.Text:       spec(variable),
		chroma.Other:      spec(variable),

		// Errors — leaf token types, no inheritance subtree.
		chroma.Error: spec(c.Error),

		// Keywords — all KeywordXxx inherit from Keyword.
		chroma.Keyword:         spec(c.Keyword),
		chroma.KeywordConstant: spec(c.Constant),
		chroma.KeywordType:     spec(c.Type),
		chroma.KeywordPseudo:   spec(c.Primitive.Or(c.Constant)),

		// Names — Name is the parent of most identifiers.
		chroma.Name:                 spec(variable),
		chroma.NameBuiltin:          spec(c.Builtin),
		chroma.NameBuiltinPseudo:    spec(c.Builtin),
		chroma.NameFunction:         spec(c.Function),
		chroma.NameClass:            spec(c.Type),
		chroma.NameConstant:         spec(c.Constant),
		chroma.NameDecorator:        spec(c.Decorator),
		chroma.NameException:        spec(c.Error),
		chroma.NameNamespace:        spec(c.Type),
		chroma.NameOther:            spec(c.Object.Or(variable)),
		chroma.NameProperty:         spec(c.Property.Or(variable)),
		chroma.NameTag:              spec(c.Tag.Or(c.Keyword)),
		chroma.NameAttribute:        spec(c.Attribute.Or(variable)),
		chroma.NameLabel:            spec(c.Decorator),
		chroma.NameVariable:         spec(variable),
		chroma.NameVariableClass:    spec(variable),
		chroma.NameVariableGlobal:   spec(variable),
		chroma.NameVariableInstance: spec(variable),

		// Literals — String and Number have deep subtrees that inherit.
		chroma.LiteralString:       spec(c.String),
		chroma.LiteralStringEscape: spec(c.StringEscape.Or(c.Number)),
		chroma.LiteralStringRegex:  spec(c.Regex.Or(c.String)),
		chroma.LiteralNumber:       spec(c.Number),
		chroma.LiteralDate:         spec(c.Number),

		// Operators
		chroma.Operator:     spec(c.Operator),
		chroma.OperatorWord: spec(c.Keyword),

		// Punctuation
		chroma.Punctuation: spec(c.Punctuation),

		// Comments
		chroma.Comment:        spec(c.Comment),
		chroma.CommentPreproc: spec(c.Preproc.Or(c.Comment)),

		// Generic (mostly diff output and markdown-as-code)
		chroma.GenericInserted:  spec(c.DiffAdded.Or(style.Entry{Fg: t.Success})),
		chroma.GenericDeleted:   spec(c.DiffRemoved.Or(style.Entry{Fg: t.Danger})),
		chroma.GenericEmph:      spec(t.ResolveMarkdownEmph()),
		chroma.GenericStrong:    spec(t.ResolveMarkdownStrong()),
		chroma.GenericHeading:   spec(t.ResolveMarkdownHeading(1)),
		chroma.GenericError:     spec(c.Error),
		chroma.GenericOutput:    spec(style.Entry{Fg: t.TextMuted}),
		chroma.GenericPrompt:    spec(style.Entry{Fg: t.TextWeak}),
		chroma.GenericTraceback: spec(c.Error),
	}

	// chroma.MustNewStyle panics on parse errors; because all inputs are
	// hex-from-color (well-formed) or fixed literals, this won't happen
	// in practice.
	return chroma.MustNewStyle("fluid-theme", entries)
}

// NewChromaHighlighterWithStyle returns a ChromaHighlighter that uses the
// given pre-built chroma.Style, bypassing the name-based theme lookup.
// Used when deriving a syntax style from a Theme palette.
func NewChromaHighlighterWithStyle(s *chroma.Style) *ChromaHighlighter {
	h := &ChromaHighlighter{theme: "fluid-theme"}
	h.chromaStyle = s
	return h
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// spec renders an Entry in the space-separated word format Chroma's
// StyleEntries parser expects, e.g. `#cc7832 bg:#1c1c1c bold`. Unset
// colors are omitted so Chroma's own inheritance fills them in.
func spec(e style.Entry) string {
	var parts []string
	if e.Fg.A != 0 {
		parts = append(parts, e.Fg.ToHex())
	}
	if e.Bg.A != 0 {
		parts = append(parts, "bg:"+e.Bg.ToHex())
	}
	if e.Attrs&style.AttrBold != 0 {
		parts = append(parts, "bold")
	}
	if e.Attrs&style.AttrItalic != 0 {
		parts = append(parts, "italic")
	}
	if e.Attrs&style.AttrUnderline != 0 {
		parts = append(parts, "underline")
	}
	return strings.Join(parts, " ")
}
