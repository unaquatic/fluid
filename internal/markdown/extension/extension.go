// Package extension adds syntax to a
// [github.com/unaquatic/fluid/internal/markdown/parser.Parser]: that of
// GitHub Flavored Markdown, as its specification, version 0.29-gfm,
// defines it, and that of other extensions.
//
// [GFM] adds tables, strikethrough, extended autolinks, and task list items,
// and [Table], [Strikethrough], [Linkify], and [TaskList] each add one of
// them. [NewLinkify] makes Linkify with options.
//
// [Footnote] adds footnotes, and [DefinitionList] definition lists, as PHP
// Markdown Extra defines them. [Typographer] writes quotation marks,
// dashes, and ellipses as typographers do, as text, such as “ and —, and
// [NewTypographer] makes it with other replacements. [CJK] suits Chinese,
// Japanese, and Korean, and [EastAsianLineBreaks] decides which of their
// soft line breaks a renderer shows.
//
//	p := parser.NewParser(parser.WithExtensions(extension.GFM, extension.Footnote))
//
// # Differences from goldmark
//
// The extensions parse as goldmark's do, but where goldmark's result is a
// bug, or depends on how it splits text into nodes:
//
//   - A footnote definition in a footnote is none, where goldmark opens one
//     and loses its text.
//   - The typographer counts the quotes that each inline content opens,
//     where goldmark, whose reset never runs, counts them across the
//     document. A quotation mark at the start of a line, also of one in a
//     block quote, follows a line ending, where goldmark sees the ">".
//   - A blank line between the blocks of a description makes it loose,
//     where goldmark leaves only its first paragraph without paragraph
//     tags.
//   - A link reference definition between a list and a ":" parts them, as
//     it parts a paragraph from a ":" in goldmark, which makes it a term of
//     the list instead.
//   - By [EastAsianLineBreaks], a soft line break before content other than
//     text, such as emphasis, stays between letters, where goldmark leaves
//     it out, and the characters on each side are those of the text, with
//     escapes and entities resolved.
package extension

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/parser"
)

var (
	// GFM adds tables, strikethrough, extended autolinks, and task list
	// items.
	GFM parser.Extension = gfm{}

	// Table adds tables: the parser that [NewTableParser] returns, with
	// priority 900, after the block parsers of CommonMark.
	Table parser.Extension = table{}

	// Strikethrough adds struck-through text: the delimiter rule that
	// [NewStrikethroughDelimiterRule] returns.
	Strikethrough parser.Extension = strikethrough{}

	// Linkify adds extended autolinks, as NewLinkify makes it with no
	// options.
	Linkify = NewLinkify()

	// TaskList adds task list items: the transformer of paragraphs that
	// [NewTaskListTransformer] returns and the transformer of documents
	// that [NewTaskListCleanup] returns, both with priority 100.
	TaskList parser.Extension = taskList{}

	// Footnote adds footnotes: the parser of definitions that
	// [NewFootnoteBlockParser] returns, with priority 999, the parser of
	// references that [NewFootnoteParser] returns, with priority 101, and
	// the transformer of documents that [NewFootnoteASTTransformer]
	// returns, with priority 999.
	Footnote parser.Extension = footnote{}

	// DefinitionList adds definition lists: the parsers that
	// [NewDefinitionListParser] and [NewDefinitionDescriptionParser]
	// return, with priorities 101 and 102.
	DefinitionList parser.Extension = definitionList{}

	// Typographer writes punctuation as typographers do, as NewTypographer
	// makes it with no options.
	Typographer = NewTypographer()

	// CJK suits Chinese, Japanese, and Korean: a backslash before a space
	// escapes it, to stand for nothing, as [parser.WithEscapedSpace] lets
	// it. A renderer leaves out the soft line breaks between wide
	// characters, which these scripts do not part, as
	// [EastAsianLineBreaksSimple] decides.
	CJK parser.Extension = cjk{}
)

// gfm is the extension GFM.
type gfm struct{}

// ExtendParser implements [parser.Extension].
func (gfm) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithExtensions(Table, Strikethrough, Linkify, TaskList))
}

// strikethrough is the extension Strikethrough.
type strikethrough struct{}

// ExtendParser implements [parser.Extension].
func (strikethrough) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithDelimiterRule(NewStrikethroughDelimiterRule()))
}

// taskList is the extension TaskList.
type taskList struct{}

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
