// Package extension adds the syntax of GitHub Flavored Markdown, as its
// specification, version 0.29-gfm, defines it, to a
// [github.com/unaquatic/fluid/internal/markdown/md.Markdown], or on its own
// to a [github.com/unaquatic/fluid/internal/markdown/parser.Parser]: tables,
// strikethrough, extended autolinks, and task list items.
//
// [GFM] adds all of them, and [Table], [Strikethrough], [Linkify], and
// [TaskList] each add one. [NewTable] and [NewLinkify] make them with
// options.
//
// It also adds syntax that GFM does not have, as PHP Markdown Extra defines
// it: [Footnote] adds footnotes, and [NewFootnote] makes it with options,
// and [DefinitionList] adds definition lists. [Typographer] writes quotation
// marks, dashes, and ellipses as typographers do, as text, such as “ and —,
// and [NewTypographer] makes it with other replacements. [CJK] suits
// Chinese, Japanese, and Korean, and [NewCJK] makes it with options.
//
//	m := md.New(md.WithExtensions(extension.GFM))
//	p := parser.NewParser(parser.WithExtensions(extension.GFM))
package extension

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// An Extension extends a [md.Markdown], its parser and its renderer, or a
// [parser.Parser] on its own.
type Extension interface {
	md.Extender
	parser.Extension
}

var (
	// GFM adds tables, strikethrough, extended autolinks, and task list
	// items.
	GFM Extension = gfm{}

	// Table adds tables, as NewTable makes it with no options.
	Table = NewTable()

	// Strikethrough adds struck-through text: the delimiter rule that
	// [NewStrikethroughDelimiterRule] returns, and the renderer that
	// [NewStrikethroughHTMLRenderer] returns, with priority 500.
	Strikethrough Extension = strikethrough{}

	// Linkify adds extended autolinks, as NewLinkify makes it with no
	// options.
	Linkify = NewLinkify()

	// TaskList adds task list items: the transformer of paragraphs that
	// [NewTaskListTransformer] returns and the transformer of documents
	// that [NewTaskListCleanup] returns, both with priority 100. The
	// renderer of CommonMark renders the items' checkboxes.
	TaskList Extension = taskList{}

	// Footnote adds footnotes, as NewFootnote makes it with no options.
	Footnote = NewFootnote()

	// DefinitionList adds definition lists: the parsers that
	// [NewDefinitionListParser] and [NewDefinitionDescriptionParser]
	// return, with priorities 101 and 102, and the renderer that
	// [NewDefinitionListHTMLRenderer] returns, with priority 500.
	DefinitionList Extension = definitionList{}

	// Typographer writes punctuation as typographers do, as NewTypographer
	// makes it with no options.
	Typographer = NewTypographer()

	// CJK suits Chinese, Japanese, and Korean: soft line breaks between
	// wide characters render as nothing, as [html.EastAsianLineBreaksSimple]
	// renders them, and a backslash before a space escapes it, to stand for
	// nothing.
	CJK = NewCJK(WithEastAsianLineBreaks(html.EastAsianLineBreaksSimple), WithEscapedSpace())
)

// gfm is the extension GFM.
type gfm struct{}

// Extend implements [md.Extender].
func (gfm) Extend(m md.Markdown) {
	for _, e := range []Extension{Table, Strikethrough, Linkify, TaskList} {
		e.Extend(m)
	}
}

// ExtendParser implements [parser.Extension].
func (gfm) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithExtensions(Table, Strikethrough, Linkify, TaskList))
}

// strikethrough is the extension Strikethrough.
type strikethrough struct{}

// Extend implements [md.Extender].
func (e strikethrough) Extend(m md.Markdown) {
	m.Parser().AddOptions(parser.WithExtensions(e))
	m.Renderer().AddOptions(html.WithNodeRenderer(NewStrikethroughHTMLRenderer(), 500))
}

// ExtendParser implements [parser.Extension].
func (strikethrough) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithDelimiterRule(NewStrikethroughDelimiterRule()))
}

// taskList is the extension TaskList.
type taskList struct{}

// Extend implements [md.Extender].
func (e taskList) Extend(m md.Markdown) { m.Parser().AddOptions(parser.WithExtensions(e)) }

// ExtendParser implements [parser.Extension].
func (taskList) ExtendParser(c *parser.Config) {
	c.Apply(
		parser.WithParagraphTransformer(NewTaskListTransformer(), 100),
		parser.WithASTTransformer(NewTaskListCleanup(), 100),
	)
}

// isBlank reports whether s holds only spaces, tabs, and line endings.
func isBlank(s string) bool {
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '\v', '\f':
		default:
			return false
		}
	}
	return true
}

// runLen returns the number of consecutive bytes c at offset i of s.
func runLen(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
}

// isUnicodeSpace reports whether r is Unicode whitespace, as CommonMark
// defines it.
func isUnicodeSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\f', '\r':
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

// isPunct reports whether r is a Unicode punctuation character, as
// CommonMark defines it, which includes symbols.
func isPunct(r rune) bool {
	if r < utf8.RuneSelf {
		return isASCIIPunct(byte(r))
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// isASCIIPunct reports whether c is an ASCII punctuation character.
func isASCIIPunct(c byte) bool {
	return '!' <= c && c <= '/' || ':' <= c && c <= '@' || '[' <= c && c <= '`' || '{' <= c && c <= '~'
}

// replaceNUL replaces the NUL characters of s with U+FFFD, as CommonMark
// replaces them.
func replaceNUL(s string) string { return strings.ReplaceAll(s, "\x00", "\uFFFD") }

func isDigit(c byte) bool  { return '0' <= c && c <= '9' }
func isLetter(c byte) bool { return 'a' <= c|0x20 && c|0x20 <= 'z' }
func isAlnum(c byte) bool  { return isDigit(c) || isLetter(c) }
