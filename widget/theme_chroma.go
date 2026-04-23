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

	variable := style.OrInherit(c.Variable, t.TextStrong)

	entries := chroma.StyleEntries{
		// Base — sets the fallback fg/bg for tokens that don't inherit
		// a more specific color. The Background entry's bg feeds into
		// what Chroma reports as the theme bg, but we also override
		// per-token bg to t.CodeBg in resolveTokenStyle, so this is
		// mostly cosmetic.
		chroma.Background: bgSpec(variable, t.CodeBg),
		chroma.Text:       variable.ToHex(),
		chroma.Other:      variable.ToHex(),

		// Errors — leaf token types, no inheritance subtree.
		chroma.Error: join(c.Error.ToHex(), attrsToChromaSpec(c.ErrorAttrs)),

		// Keywords — all KeywordXxx inherit from Keyword.
		chroma.Keyword:         join(c.Keyword.ToHex(), attrsToChromaSpec(c.KeywordAttrs)),
		chroma.KeywordConstant: join(c.Constant.ToHex(), attrsToChromaSpec(c.ConstantAttrs)),
		chroma.KeywordType:     join(c.Type.ToHex(), attrsToChromaSpec(c.TypeAttrs)),

		// Names — Name is the parent of most identifiers.
		chroma.Name:                variable.ToHex(),
		chroma.NameBuiltin:         c.Builtin.ToHex(),
		chroma.NameBuiltinPseudo:   c.Builtin.ToHex(),
		chroma.NameFunction:        join(c.Function.ToHex(), attrsToChromaSpec(c.FunctionAttrs)),
		chroma.NameClass:           join(c.Type.ToHex(), attrsToChromaSpec(c.TypeAttrs)),
		chroma.NameConstant:        join(c.Constant.ToHex(), attrsToChromaSpec(c.ConstantAttrs)),
		chroma.NameDecorator:       c.Decorator.ToHex(),
		chroma.NameException:       c.Error.ToHex(),
		chroma.NameNamespace:       c.Type.ToHex(),
		chroma.NameOther:           variable.ToHex(),
		chroma.NameTag:             style.OrInherit(c.Tag, c.Keyword).ToHex(),
		chroma.NameAttribute:       style.OrInherit(c.Attribute, variable).ToHex(),
		chroma.NameLabel:           c.Decorator.ToHex(),
		chroma.NameVariable:        variable.ToHex(),
		chroma.NameVariableClass:   variable.ToHex(),
		chroma.NameVariableGlobal:  variable.ToHex(),
		chroma.NameVariableInstance: variable.ToHex(),

		// Literals — String and Number have deep subtrees that inherit.
		chroma.LiteralString:       join(c.String.ToHex(), attrsToChromaSpec(c.StringAttrs)),
		chroma.LiteralStringEscape: style.OrInherit(c.StringEscape, c.Number).ToHex(),
		chroma.LiteralNumber:       c.Number.ToHex(),
		chroma.LiteralDate:         c.Number.ToHex(),

		// Operators
		chroma.Operator:     join(c.Operator.ToHex(), attrsToChromaSpec(c.OperatorAttrs)),
		chroma.OperatorWord: join(c.Keyword.ToHex(), attrsToChromaSpec(c.KeywordAttrs)),

		// Punctuation
		chroma.Punctuation: join(c.Punctuation.ToHex(), attrsToChromaSpec(c.PunctuationAttrs)),

		// Comments
		chroma.Comment:        join(c.Comment.ToHex(), attrsToChromaSpec(c.CommentAttrs)),
		chroma.CommentPreproc: style.OrInherit(c.Preproc, c.Comment).ToHex(),

		// Generic (mostly diff output and markdown-as-code)
		chroma.GenericInserted: style.OrInherit(c.DiffAdded, t.Success).ToHex(),
		chroma.GenericDeleted:  style.OrInherit(c.DiffRemoved, t.Danger).ToHex(),
		chroma.GenericEmph:     "italic",
		chroma.GenericStrong:   "bold",
		chroma.GenericHeading:  t.MarkdownHeading[0].ToHex() + " bold",
		chroma.GenericError:    c.Error.ToHex(),
		chroma.GenericOutput:   t.TextMuted.ToHex(),
		chroma.GenericPrompt:   t.TextWeak.ToHex(),
		chroma.GenericTraceback: c.Error.ToHex(),
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

// bgSpec returns a Chroma StyleEntries spec that sets fg and bg.
func bgSpec(fg, bg style.Color) string {
	return fg.ToHex() + " bg:" + bg.ToHex()
}

// join concatenates non-empty parts with a single space, the format
// Chroma's StyleEntries parser expects.
func join(parts ...string) string {
	nonEmpty := parts[:0]
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " ")
}

// attrsToChromaSpec converts a bitmask of AttrBold / AttrItalic /
// AttrUnderline into the corresponding Chroma spec words.
func attrsToChromaSpec(attrs uint32) string {
	var parts []string
	if attrs&style.AttrBold != 0 {
		parts = append(parts, "bold")
	}
	if attrs&style.AttrItalic != 0 {
		parts = append(parts, "italic")
	}
	if attrs&style.AttrUnderline != 0 {
		parts = append(parts, "underline")
	}
	return strings.Join(parts, " ")
}
