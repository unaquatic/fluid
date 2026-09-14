package widget

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/unaquatic/fluid/style"
)

// ChromaStyleFromTheme returns a Chroma style derived from t's code palette.
// Use the result with NewChromaHighlighterWithStyle to keep syntax colors
// consistent with the rest of the UI theme.
//
// Unset code colors inherit from related theme roles, so a partially specified
// code palette still produces a complete syntax style.
//
// Chroma propagates these entries through its token hierarchy. For example,
// KeywordDeclaration inherits Keyword and LiteralStringDouble inherits LiteralString.
func ChromaStyleFromTheme(t *style.Theme) *chroma.Style {
	c := t.Code

	variable := c.Variable.Or(style.Entry{Fg: t.TextStrong})

	entries := chroma.StyleEntries{
		// Base entries provide fallbacks for tokens without a more specific style.
		// resolveTokenStyle still applies t.CodeBg as each token's background.
		chroma.Background: spec(variable.WithBg(t.CodeBg)),
		chroma.Text:       spec(variable),
		chroma.Other:      spec(variable),

		// Error is a leaf token type.
		chroma.Error: spec(c.Error),

		// Specific keyword token types inherit Keyword unless overridden here.
		chroma.Keyword:         spec(c.Keyword),
		chroma.KeywordConstant: spec(c.Constant),
		chroma.KeywordType:     spec(c.Type),
		chroma.KeywordPseudo:   spec(c.Primitive.Or(c.Constant)),

		// Name is the parent style for most identifiers.
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

		// String and Number styles propagate to their literal subtypes.
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

		// Generic token types primarily style diff and Markdown-like source.
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

	// Every specification below is generated from a Color or fixed keyword,
	// so MustNewStyle cannot receive malformed style syntax.
	return chroma.MustNewStyle("fluid-theme", entries)
}

// NewChromaHighlighterWithStyle returns a highlighter that uses s directly
// instead of resolving a Chroma theme by name.
func NewChromaHighlighterWithStyle(s *chroma.Style) *ChromaHighlighter {
	h := &ChromaHighlighter{theme: "fluid-theme"}
	h.chromaStyle = s
	return h
}

// spec formats an Entry for Chroma's StyleEntries parser, for example
// `#cc7832 bg:#1c1c1c bold`. Unset
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
