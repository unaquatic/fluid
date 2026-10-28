package ast

import "strings"

// Kinds of the blocks of CommonMark, GitHub Flavored Markdown, and the other
// extensions of the parser.
var (
	KindDocument        = NewNodeKind("Document")
	KindParagraph       = NewNodeKind("Paragraph")
	KindHeading         = NewNodeKind("Heading")
	KindThematicBreak   = NewNodeKind("ThematicBreak")
	KindCodeBlock       = NewNodeKind("CodeBlock")
	KindFencedCodeBlock = NewNodeKind("FencedCodeBlock")
	KindBlockquote      = NewNodeKind("Blockquote")
	KindList            = NewNodeKind("List")
	KindListItem        = NewNodeKind("ListItem")
	KindHTMLBlock       = NewNodeKind("HTMLBlock")
	KindTable           = NewNodeKind("Table")
	KindTableHeader     = NewNodeKind("TableHeader")
	KindTableRow        = NewNodeKind("TableRow")
	KindTableCell       = NewNodeKind("TableCell")
	KindFootnoteList    = NewNodeKind("FootnoteList")
	KindFootnote        = NewNodeKind("Footnote")

	KindDefinitionList        = NewNodeKind("DefinitionList")
	KindDefinitionTerm        = NewNodeKind("DefinitionTerm")
	KindDefinitionDescription = NewNodeKind("DefinitionDescription")
)

// A Document is the root of a document's tree.
type Document struct {
	BaseBlock

	// Links holds the document's link reference definitions, in the order
	// in which they appear. Of several definitions of one label, it holds
	// only the first, which is the one that links use.
	Links []*LinkDef

	// Meta holds the document's metadata, such as the fields of its front
	// matter, by name. The parser sets none; extensions set it, and make
	// it first, as it is nil until one does.
	Meta map[string]any
}

// Type implements [Node]; it returns [TypeDocument].
func (*Document) Type() NodeType { return TypeDocument }

// Kind implements [Node].
func (*Document) Kind() NodeKind { return KindDocument }

// A Paragraph is a paragraph, whose children are its inline content.
type Paragraph struct {
	BaseBlock
}

// Kind implements [Node].
func (*Paragraph) Kind() NodeKind { return KindParagraph }

// A Heading is an ATX or setext heading, whose children are its inline
// content.
type Heading struct {
	BaseBlock
	Level int // 1 through 6
}

// Kind implements [Node].
func (*Heading) Kind() NodeKind { return KindHeading }

// A ThematicBreak is a thematic break.
type ThematicBreak struct {
	BaseBlock
}

// Kind implements [Node].
func (*ThematicBreak) Kind() NodeKind { return KindThematicBreak }

// A CodeBlock is an indented code block.
type CodeBlock struct {
	BaseBlock

	// Literal is the code, each line of which ends with a newline.
	Literal string
}

// Kind implements [Node].
func (*CodeBlock) Kind() NodeKind { return KindCodeBlock }

// A FencedCodeBlock is a fenced code block.
type FencedCodeBlock struct {
	BaseBlock

	// Info is the block's info string, with escapes and entities resolved.
	// Its first word, which Language returns, usually names the code's
	// language.
	Info string

	// Literal is the code, each line of which ends with a newline. It is
	// empty for a block with no lines.
	Literal string

	FenceChar   byte // '`' or '~'
	FenceLength int  // length of the opening fence
	FenceOffset int  // columns of indentation before the opening fence
}

// Kind implements [Node].
func (*FencedCodeBlock) Kind() NodeKind { return KindFencedCodeBlock }

// Language returns the first word of the block's info string.
func (b *FencedCodeBlock) Language() string {
	lang, _, _ := strings.Cut(b.Info, " ")
	return lang
}

// A Blockquote is a block quote.
type Blockquote struct {
	BaseBlock
}

// Kind implements [Node].
func (*Blockquote) Kind() NodeKind { return KindBlockquote }

// A List is a bullet or ordered list, whose children are [*ListItem]
// nodes.
type List struct {
	BaseBlock

	// Marker is the character of a bullet list's markers, '-', '+', or '*',
	// or the delimiter after an ordered list's numbers, '.' or ')'.
	Marker byte

	// IsTight reports whether no blank line parts the list's items, or
	// blocks within one of them; the paragraphs of a tight list are not
	// wrapped in paragraph tags.
	IsTight bool

	// Start is the number of an ordered list's first item, and 0 for a
	// bullet list.
	Start int
}

// Kind implements [Node].
func (*List) Kind() NodeKind { return KindList }

// IsOrdered reports whether l is an ordered list.
func (l *List) IsOrdered() bool { return l.Marker == '.' || l.Marker == ')' }

// A ListItem is an item of a [List]. Its span runs from its marker through
// the end of its last block, or the end of the marker if it has none.
type ListItem struct {
	BaseBlock

	// Offset is the column at which the item's content starts, to which
	// lines after its first are indented.
	Offset int

	// Task is the item's task list marker, which GFM recognizes before the
	// text of the item's first paragraph and removes from it.
	Task Task
}

// Kind implements [Node].
func (*ListItem) Kind() NodeKind { return KindListItem }

// An HTMLBlock is a block of raw HTML.
type HTMLBlock struct {
	BaseBlock

	// HTMLBlockType is the number, 1 through 7, of the condition that
	// started the block, as CommonMark numbers them, which decides how it
	// ends.
	HTMLBlockType int

	// Literal is the HTML as written, with no newline after its last line.
	Literal string
}

// Kind implements [Node].
func (*HTMLBlock) Kind() NodeKind { return KindHTMLBlock }

// A Table is a GFM table, whose children are a [*TableHeader] and
// [*TableRow] nodes, each with a [*TableCell] for each column.
type Table struct {
	BaseBlock
	Alignments []Alignment // each column's alignment
}

// Kind implements [Node].
func (*Table) Kind() NodeKind { return KindTable }

// A TableHeader is the header row of a [Table].
type TableHeader struct {
	BaseBlock
}

// Kind implements [Node].
func (*TableHeader) Kind() NodeKind { return KindTableHeader }

// A TableRow is a body row of a [Table].
type TableRow struct {
	BaseBlock
}

// Kind implements [Node].
func (*TableRow) Kind() NodeKind { return KindTableRow }

// A TableCell is a cell of a table row, whose children are its inline
// content. Its span is the range of its content, without the spaces around
// it; a cell that its row lacks has an empty span at the row's end.
type TableCell struct {
	BaseBlock
	Alignment Alignment
}

// Kind implements [Node].
func (*TableCell) Kind() NodeKind { return KindTableCell }

// A FootnoteList holds the footnotes of a document, as its last block, in
// the order of their numbers: a [*Footnote] for each definition that the
// document refers to. Its span is empty, at the end of the document, as
// the footnotes keep the spans of their definitions.
type FootnoteList struct {
	BaseBlock

	// Count is the number of footnotes that the document refers to.
	Count int
}

// Kind implements [Node].
func (*FootnoteList) Kind() NodeKind { return KindFootnoteList }

// A Footnote is the definition of a footnote, such as "[^1]: text", whose
// children are its blocks. Its span runs from its label through the end of
// its last block, or the end of the label if it has none.
type Footnote struct {
	BaseBlock

	// Ref is the footnote's label, without its brackets and caret, which
	// the references to it match exactly.
	Ref string

	// Index is the footnote's number, from 1, in the order in which the
	// document first refers to its footnotes, or 0 if it refers to none.
	Index int
}

// Kind implements [Node].
func (*Footnote) Kind() NodeKind { return KindFootnote }

// A DefinitionList is a list of terms and their descriptions, as PHP
// Markdown Extra defines it, whose children are [*DefinitionTerm] and
// [*DefinitionDescription] nodes: terms, then the descriptions of them, and
// so on. Its span runs from its first term through the end of its last
// description.
type DefinitionList struct {
	BaseBlock

	// Offset is the column, from the list's, at which the content of the
	// list's last description starts, to which lines after the first of the
	// description are indented.
	Offset int
}

// Kind implements [Node].
func (*DefinitionList) Kind() NodeKind { return KindDefinitionList }

// A DefinitionTerm is a term of a [DefinitionList], whose children are its
// inline content.
type DefinitionTerm struct {
	BaseBlock
}

// Kind implements [Node].
func (*DefinitionTerm) Kind() NodeKind { return KindDefinitionTerm }

// A DefinitionDescription is a description of the terms before it in a
// [DefinitionList], whose children are its blocks. Its span runs from its
// ":" through the end of its last block.
type DefinitionDescription struct {
	BaseBlock

	// IsTight reports whether no blank line comes before the description,
	// or between its blocks; the paragraphs of a tight description are not
	// wrapped in paragraph tags.
	IsTight bool
}

// Kind implements [Node].
func (*DefinitionDescription) Kind() NodeKind { return KindDefinitionDescription }
