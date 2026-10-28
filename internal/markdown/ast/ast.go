// Package ast defines the tree of a parsed Markdown document, its abstract
// syntax tree, which package
// [github.com/unaquatic/fluid/internal/markdown/parser] builds.
//
// The tree is made of [Node] values: blocks, such as [*Paragraph] and
// [*List], which hold blocks or inline content, and inline content, such as
// [*Text] and [*Emphasis]. Its root is a [*Document]. Each node links to its
// parent, its siblings, and its children, and [Walk] visits them. Text
// holds the characters it stands for, with escapes and entities resolved.
//
// An extension of the parser adds a kind of node with a type that embeds
// [BaseBlock] or [BaseInline] and a [NodeKind] that [NewNodeKind] makes.
//
// Every block records, in a [Span], the range of the source it comes from,
// in bytes; a [LineIndex] finds the lines and columns of those offsets.
// [Dump] writes a tree as text, for debugging and tests.
package ast

import (
	"slices"
	"strconv"
)

// A Span is the range of a document's source that a node comes from: the
// bytes src[Start:End].
type Span struct {
	Start, End int
}

// A LineIndex finds the lines and columns of byte offsets in a document's
// source.
type LineIndex struct {
	// Starts holds the offsets at which the source's lines start, in order.
	// Lines end with "\n", "\r\n", or "\r".
	Starts []int
}

// NewLineIndex returns the line index of src.
func NewLineIndex(src string) *LineIndex {
	x := &LineIndex{Starts: []int{0}}
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				i++
			}
			fallthrough
		case '\n':
			x.Starts = append(x.Starts, i+1)
		}
	}
	return x
}

// Position returns the line and column of offset, both counted from 1. The
// column counts bytes, so that the first byte of a line is at column 1.
func (x *LineIndex) Position(offset int) (line, col int) {
	i, found := slices.BinarySearch(x.Starts, offset)
	if !found {
		i--
	}
	return i + 1, offset - x.Starts[i] + 1
}

// A Line is a line of a block's source text, as a block took it.
type Line struct {
	// Text is the line from where the block took it, without its line
	// ending. The columns of a tab that is only partly the block's are
	// spaces.
	Text string

	// Offset is the offset in the source of the start of Text, or of the
	// tab whose spaces start it.
	Offset int
}

// A LinkDef is a link reference definition.
type LinkDef struct {
	Span         // from the "[" through the destination or title
	Label string // the label as written, without its brackets
	URL   string // the destination, with escapes and entities resolved
	Title string // the title, with escapes and entities resolved
}

// A Task is the task list marker of a [ListItem].
type Task uint8

const (
	NotTask   Task = iota // the item has no task list marker
	Unchecked             // the marker "[ ]"
	Checked               // the marker "[x]" or "[X]"
)

// String returns the name of t: "none", "unchecked", or "checked".
func (t Task) String() string {
	switch t {
	case NotTask:
		return "none"
	case Unchecked:
		return "unchecked"
	case Checked:
		return "checked"
	}
	return "Task(" + strconv.Itoa(int(t)) + ")"
}

// An Alignment is the alignment of a [Table] column.
type Alignment uint8

const (
	AlignNone   Alignment = iota // the delimiter row has no colon, as in "---"
	AlignLeft                    // a colon before the hyphens, as in ":--"
	AlignCenter                  // a colon at each end, as in ":-:"
	AlignRight                   // a colon after the hyphens, as in "--:"
)

// String returns the name of a: "none", "left", "center", or "right".
func (a Alignment) String() string {
	switch a {
	case AlignNone:
		return "none"
	case AlignLeft:
		return "left"
	case AlignCenter:
		return "center"
	case AlignRight:
		return "right"
	}
	return "Alignment(" + strconv.Itoa(int(a)) + ")"
}

// A Prioritized is a value with the priority that orders it among the
// others of its kind, such as a parser or a transformer: lower priorities
// come first, and equal ones in the order in which they were added.
type Prioritized[T any] struct {
	Value    T
	Priority int
}
