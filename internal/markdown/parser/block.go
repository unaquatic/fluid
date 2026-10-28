package parser

import (
	"iter"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// A BlockParser parses blocks of one kind.
type BlockParser interface {
	// Trigger returns the bytes that a block can start with, after its
	// indentation, or nil if it can start with any.
	Trigger() []byte

	// Open opens a block at the current position of the line of s, within
	// parent, the deepest open block that the line continues, if one starts
	// there. It consumes the block's markers and returns the block, with
	// [HasChildren] if more blocks can start in it on the line or
	// [NoChildren] if it takes the rest of the line, or nil and [None].
	// The [Parser] adds the block to the tree, unless Open adds it with
	// [BlockState.Add] or opens one that is in it with [BlockState.Reopen].
	// Open may close blocks, such as the paragraph that parent is; if it
	// then opens none, the block parsers try the line again in the tip.
	Open(parent ast.Node, s *BlockState) (ast.Node, State)

	// Continue reports whether the current line of s continues block n,
	// consuming the markers of n that the line starts with. It returns
	// [Continue] with [HasChildren] for a block in which more blocks can
	// start, Continue with [NoChildren] for a block that takes the rest of
	// the line, Continue alone for a block that takes the rest of the line
	// unless another block starts there, as a paragraph does, or [Close]. A
	// block that the line ends, such as fenced code at its closing fence,
	// closes itself with [BlockState.Close], which consumes the line.
	Continue(n ast.Node, s *BlockState) State

	// Close finishes block n, to which no more lines can be added. It runs
	// while n is the deepest open block, after [BlockState.Close] ends n's
	// span at the end of its last line.
	Close(n ast.Node, s *BlockState)

	// CanInterruptParagraph reports whether a block can start on a line
	// that would otherwise continue a paragraph.
	CanInterruptParagraph() bool

	// CanAcceptIndentedLine reports whether a block can start after
	// indentation of 4 columns or more.
	CanAcceptIndentedLine() bool
}

// A LineAdder is a [BlockParser] whose blocks take lines of text: the rest
// of the line that opens one, and of each line that continues one and
// starts no other block.
type LineAdder interface {
	// AddLine adds the rest of the current line of s to block n.
	AddLine(n ast.Node, s *BlockState)
}

// A State reports what [BlockParser.Open] opened, or how a line continues a
// block, as a set of flags.
type State int

const (
	None        State = 1 << iota // nothing opened
	Continue                      // the line continues the block
	Close                         // the line does not continue the block, which closes
	HasChildren                   // more blocks can start in the block on the line
	NoChildren                    // the block takes the rest of the line
)

// openBlock is a block that is open: an entry of the stack of open blocks,
// from the document to the tip.
type openBlock struct {
	node      ast.Node
	parser    BlockParser // nil for the document
	container bool        // opened with HasChildren
	state     State       // what Continue returned for the current line

	// childEnd is the number of the last line of the block's last child.
	childEnd int

	// loosens is the list that a blank line before the block makes loose,
	// if the block stays in the tree as it closes.
	loosens *ast.List

	closed bool // whether the block has closed
}

// A BlockState is the state of a [Parser] as it builds the tree of blocks:
// the line that it parses, how much of the line is consumed, and the open
// blocks.
type BlockState struct {
	syntax *syntax
	src    string
	doc    *ast.Document

	open        []*openBlock // root first
	lastMatched int          // index in open of the deepest block the line continues
	allClosed   bool         // whether blocks the line does not continue are closed

	line       string
	lineNumber int
	lineOffset int   // offset in the source of the line's start
	lineEnds   []int // offset in the source of each line's end, by number

	// offset is the byte offset, and column the column, that the line is
	// consumed to.
	offset, column int

	// nextNonspace is the offset, and nextNonspaceColumn the column, of the
	// first character at or after offset that is not a space or tab.
	nextNonspace, nextNonspaceColumn int

	indent               int  // columns from column to nextNonspaceColumn
	indented             bool // indent >= 4
	blank                bool // the line has only spaces and tabs after offset
	partiallyConsumedTab bool // offset is in the middle of a tab

	// ruleFail is the offset at which a thematic break failed on the
	// current line, and before which none can start.
	ruleFail int

	// startAt is the offset in the source at which the next block that Add
	// adds starts, or -1 for blockStart, the offset of NextNonspace when the
	// block parsers started to look for one.
	startAt, blockStart int

	refs   map[string]*ast.LinkDef // link reference definitions by normalized label
	ids    IDs                     // made when first used
	values map[any]any
}

// newBlockState returns the state of a Parser with syntax at the start of
// document src.
func newBlockState(syntax *syntax, src string) *BlockState {
	doc := &ast.Document{}
	doc.SetSourceSpan(ast.Span{End: len(src)})
	return &BlockState{
		syntax:   syntax,
		src:      src,
		doc:      doc,
		open:     []*openBlock{{node: doc, container: true}},
		lineEnds: make([]int, 1, strings.Count(src, "\n")+2),
		startAt:  -1,
		refs:     make(map[string]*ast.LinkDef),
	}
}

// Source returns the source of the document, in which the spans of nodes
// and the offsets of lines are.
func (s *BlockState) Source() string { return s.src }

// Line returns the current line, without its line ending. It holds the NUL
// characters of the source, which inline parsing replaces with U+FFFD, as
// CommonMark replaces them.
func (s *BlockState) Line() string { return s.line }

// LineNumber returns the number of the current line, counted from 1.
func (s *BlockState) LineNumber() int { return s.lineNumber }

// LineOffset returns the offset in the source at which the current line
// starts. Offsets in the line, such as Offset, are relative to it.
func (s *BlockState) LineOffset() int { return s.lineOffset }

// Offset returns the byte offset in the line that it is consumed to.
func (s *BlockState) Offset() int { return s.offset }

// Column returns the column that the line is consumed to, with tabs
// stopping at multiples of 4.
func (s *BlockState) Column() int { return s.column }

// NextNonspace returns the offset of the first character of the line at or
// after Offset that is not a space or tab, or the line's length.
func (s *BlockState) NextNonspace() int {
	s.findNextNonspace()
	return s.nextNonspace
}

// Indent returns the columns from Column to NextNonspace.
func (s *BlockState) Indent() int {
	s.findNextNonspace()
	return s.indent
}

// Indented reports whether Indent is 4 or more, which indents code.
func (s *BlockState) Indented() bool {
	s.findNextNonspace()
	return s.indented
}

// Blank reports whether only spaces and tabs follow Offset.
func (s *BlockState) Blank() bool {
	s.findNextNonspace()
	return s.blank
}

// Rest returns the line from NextNonspace.
func (s *BlockState) Rest() string { return s.line[s.NextNonspace():] }

// Remaining returns the line from Offset, with the columns of a tab that is
// partly consumed as spaces, for a block to take.
func (s *BlockState) Remaining() ast.Line {
	l := ast.Line{Text: s.line[s.offset:], Offset: s.lineOffset + s.offset}
	if s.partiallyConsumedTab {
		l.Text = strings.Repeat(" ", 4-s.column%4) + s.line[s.offset+1:]
	}
	return l
}

// Advance consumes n bytes of the line.
func (s *BlockState) Advance(n int) { s.advanceOffset(n, false) }

// AdvanceColumns consumes n columns of the line, of which a tab can supply
// only some.
func (s *BlockState) AdvanceColumns(n int) { s.advanceOffset(n, true) }

// AdvanceNextNonspace consumes the line up to NextNonspace.
func (s *BlockState) AdvanceNextNonspace() {
	s.findNextNonspace()
	s.advanceNextNonspace()
}

// Tip returns the deepest open block.
func (s *BlockState) Tip() ast.Node { return s.open[len(s.open)-1].node }

// Lazy reports whether the current line, if no block starts on it, would
// continue the paragraph that is the tip only lazily, as a paragraph
// continuation line that does not continue the blocks around it.
func (s *BlockState) Lazy() bool {
	_, para := s.Tip().(*ast.Paragraph)
	return !s.allClosed && !s.Blank() && para
}

// StartAt makes the span of the next block that the parser adds start at
// offset in the source, rather than at NextNonspace. It suits blocks that
// start at their indentation, or on the line before.
func (s *BlockState) StartAt(offset int) { s.startAt = offset }

// CloseUnmatched closes the blocks that the current line does not continue.
func (s *BlockState) CloseUnmatched() {
	if s.allClosed {
		return
	}
	if s.lastMatched+1 < len(s.open) {
		s.Close(s.open[s.lastMatched+1].node, s.lineNumber-1)
	}
	s.allClosed = true
}

// Add closes the blocks that the current line does not continue, and those
// that cannot hold n, and adds n, which p parses, as the last child of the
// tip, which n becomes, with state [HasChildren] if blocks can start in it.
// A list can hold only list items, which only lists can hold; a leaf can
// hold no blocks. The span of n starts where the line's NextNonspace was as
// the block parsers started to look for a block, unless StartAt sets it.
func (s *BlockState) Add(n ast.Node, p BlockParser, state State) {
	s.CloseUnmatched()
	for !s.canHold(s.open[len(s.open)-1], n) {
		s.Close(s.Tip(), s.lineNumber-1)
	}
	start := s.blockStart
	if s.startAt >= 0 {
		start, s.startAt = s.startAt, -1
	}
	n.SetSourceSpan(ast.Span{Start: start, End: start})

	// A blank line between the blocks of a list item, or between items,
	// makes their list loose, unless the block after it leaves its parent
	// as it closes, as one that is removed or moved elsewhere does.
	b := &openBlock{node: n, parser: p, container: state&HasChildren != 0}
	parent := s.open[len(s.open)-1]
	if parent.node.LastChild() != nil && parent.childEnd != s.lineOf(start)-1 {
		b.loosens = s.listOf(len(s.open) - 1)
	}
	parent.node.AppendChild(parent.node, n)
	s.open = append(s.open, b)
}

// Reopen opens block n, which p parses and which is in the tree already,
// as the tip, with [HasChildren], so that blocks can start in it on the
// current line and on those after it: a block that closed, such as a
// definition list that a definition after it continues, or one that p put
// in the place of another. It first closes the blocks that the current line
// does not continue, and reports whether n is then the last child of the
// tip, which it must be to open. The span of n ends again as n closes.
func (s *BlockState) Reopen(n ast.Node, p BlockParser) bool {
	s.CloseUnmatched()
	tip := s.Tip()
	if n.Parent() != tip || tip.LastChild() != n {
		return false
	}
	b := &openBlock{node: n, parser: p, container: true}
	if c := n.LastChild(); c != nil {
		b.childEnd = s.lineOf(c.SourceSpan().End)
	}
	s.open = append(s.open, b)
	return true
}

// canHold reports whether open block b can hold block n.
func (s *BlockState) canHold(b *openBlock, n ast.Node) bool {
	if !b.container {
		return false
	}
	_, list := b.node.(*ast.List)
	_, item := n.(*ast.ListItem)
	return list == item
}

// listOf returns the list that the open block at index i of the stack is,
// or is an item of, or nil.
func (s *BlockState) listOf(i int) *ast.List {
	switch n := s.open[i].node.(type) {
	case *ast.List:
		return n
	case *ast.ListItem:
		l, _ := s.open[i-1].node.(*ast.List)
		return l
	}
	return nil
}

// Close closes open block n, whose last line is line, and makes its parent
// the tip. It first closes the open blocks in n, as blocks that the current
// line does not continue, at the line before it or line if that is
// earlier. The span of each block it closes ends at the end of its last
// line before its parser closes it.
func (s *BlockState) Close(n ast.Node, line int) {
	// n is usually the tip, so search from it.
	i := len(s.open) - 1
	for i >= 0 && s.open[i].node != n {
		i--
	}
	if i < 0 {
		return
	}
	for len(s.open) > i {
		b := s.open[len(s.open)-1]
		s.open = s.open[:len(s.open)-1]
		b.closed = true
		end := line
		if b.node != n {
			end = min(line, s.lineNumber-1)
		}
		sp := b.node.SourceSpan()
		if end >= 1 && end < len(s.lineEnds) {
			sp.End = max(s.lineEnds[end], sp.Start)
		}
		b.node.SetSourceSpan(sp)
		if b.parser != nil {
			b.parser.Close(b.node, s)
		}
		// A block that moves elsewhere, such as a footnote definition to
		// the list of footnotes, still ends where the next block of its
		// parent may follow it.
		if b.node.Parent() != nil && len(s.open) > 0 {
			parent := s.open[len(s.open)-1]
			parent.childEnd = s.lineOf(b.node.SourceSpan().End)
			if b.loosens != nil && b.node.Parent() == parent.node {
				b.loosens.IsTight = false
			}
		}
	}
	s.lastMatched = min(s.lastMatched, len(s.open)-1)
}

// lineOf returns the number of the line that offset in the source is on.
func (s *BlockState) lineOf(offset int) int {
	i, _ := slices.BinarySearch(s.lineEnds[1:], offset)
	return i + 1
}

// IDs returns the IDs of the elements of the current document, which
// [WithIDs] sets how to make.
func (s *BlockState) IDs() IDs {
	if s.ids == nil {
		s.ids = s.syntax.newIDs()
	}
	return s.ids
}

// Value returns the value that SetValue stored for key while parsing the
// current document, or nil.
func (s *BlockState) Value(key any) any { return s.values[key] }

// SetValue stores value for key while parsing the current document.
func (s *BlockState) SetValue(key, value any) {
	if s.values == nil {
		s.values = make(map[any]any)
	}
	s.values[key] = value
}

// incorporateLine adds line ln, at offset off in the source, to the block
// tree.
func (s *BlockState) incorporateLine(off int, ln string) {
	s.offset, s.column = 0, 0
	s.nextNonspace, s.nextNonspaceColumn = 0, 0
	s.ruleFail = 0
	s.blank, s.partiallyConsumedTab = false, false
	s.lineNumber++
	s.line = ln
	s.lineOffset = off
	s.lineEnds = append(s.lineEnds, off+len(ln))

	// Continue the open blocks that the line matches.
	s.lastMatched = 0
	for i := 1; i < len(s.open); i++ {
		b := s.open[i]
		s.findNextNonspace()
		st := b.parser.Continue(b.node, s)
		if i >= len(s.open) || s.open[i] != b {
			return // the block closed itself, consuming the line
		}
		if st&Continue == 0 {
			break
		}
		b.state = st
		s.lastMatched = i
	}
	s.allClosed = s.lastMatched == len(s.open)-1
	container := s.open[s.lastMatched]

	// Open new blocks until a leaf takes the rest of the line. A parser
	// that closes the container and opens nothing leaves the line to the
	// tip.
	if container.container || container.state&NoChildren == 0 {
		for {
			s.findNextNonspace()
			opened, state := s.openBlock(container)
			if opened == nil && container.closed {
				container = s.open[len(s.open)-1]
				continue
			}
			if opened == nil {
				s.advanceNextNonspace()
				break
			}
			container = s.open[len(s.open)-1]
			if state&HasChildren == 0 {
				break
			}
		}
	}

	// Add what remains of the line to the appropriate block.
	if s.Lazy() {
		tip := s.open[len(s.open)-1]
		tip.parser.(LineAdder).AddLine(tip.node, s)
		return
	}
	s.CloseUnmatched()
	if la, ok := container.parser.(LineAdder); ok && !container.container {
		la.AddLine(container.node, s)
	} else if s.offset < len(ln) && !s.blank {
		s.blockStart = s.lineOffset + s.nextNonspace
		p := &ast.Paragraph{}
		s.Add(p, s.syntax.paragraph, NoChildren)
		s.advanceNextNonspace()
		s.syntax.paragraph.AddLine(p, s)
	}
}

// openBlock opens the block that starts at the current position, if any,
// within container, and returns it and its state.
func (s *BlockState) openBlock(container *openBlock) (ast.Node, State) {
	if s.blank {
		return nil, None
	}
	_, inPara := s.Tip().(*ast.Paragraph)
	s.blockStart = s.lineOffset + s.nextNonspace
	for _, bp := range s.syntax.starts[s.line[s.nextNonspace]] {
		if s.indented && !bp.CanAcceptIndentedLine() || inPara && !bp.CanInterruptParagraph() {
			continue
		}
		n, state := bp.Open(container.node, s)
		if n == nil {
			if container.closed {
				return nil, None // the parser closed the container
			}
			continue
		}
		if n.Parent() == nil {
			s.Add(n, bp, state)
		}
		return n, state
	}
	return nil, None
}

// findNextNonspace finds the first character at or after offset that is not
// a space or tab, and how far it is indented. While offset is before the
// character last found, that character is still the first.
func (s *BlockState) findNextNonspace() {
	if s.nextNonspace <= s.offset {
		i, cols := s.offset, s.column
		for i < len(s.line) {
			if c := s.line[i]; c == ' ' {
				cols++
			} else if c == '\t' {
				cols += 4 - cols%4
			} else {
				break
			}
			i++
		}
		s.nextNonspace, s.nextNonspaceColumn = i, cols
	}
	s.blank = s.nextNonspace == len(s.line)
	s.indent = s.nextNonspaceColumn - s.column
	s.indented = s.indent >= 4
}

// advanceNextNonspace consumes the line up to nextNonspace, as
// findNextNonspace last found it.
func (s *BlockState) advanceNextNonspace() {
	s.offset, s.column = s.nextNonspace, s.nextNonspaceColumn
	s.partiallyConsumedTab = false
}

// advanceOffset consumes count characters of the line or, if columns is
// set, count columns, of which a tab may supply only some.
func (s *BlockState) advanceOffset(count int, columns bool) {
	for count > 0 && s.offset < len(s.line) {
		if s.line[s.offset] != '\t' {
			s.partiallyConsumedTab = false
			s.offset++
			s.column++
			count--
			continue
		}
		toTab := 4 - s.column%4
		if !columns {
			s.partiallyConsumedTab = false
			s.column += toTab
			s.offset++
			count--
			continue
		}
		s.partiallyConsumedTab = toTab > count
		n := min(toTab, count)
		s.column += n
		if !s.partiallyConsumedTab {
			s.offset++
		}
		count -= n
	}
}

// lines returns the offsets in src and the text of src's lines, without
// their line endings, which are "\n", "\r\n", or "\r". A line ending at the
// end of src starts no line.
func lines(src string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for off := 0; off < len(src); {
			rest := src[off:]
			i := strings.IndexAny(rest, "\r\n")
			if i < 0 {
				yield(off, rest)
				return
			}
			n := 1
			if rest[i] == '\r' && i+1 < len(rest) && rest[i+1] == '\n' {
				n = 2
			}
			if !yield(off, rest[:i]) {
				return
			}
			off += i + n
		}
	}
}

// replaceNUL replaces the NUL characters of s with U+FFFD, as CommonMark
// replaces them.
func replaceNUL(s string) string {
	if strings.IndexByte(s, 0) < 0 {
		return s
	}
	return strings.ReplaceAll(s, "\x00", "\uFFFD")
}

// joinText returns the text of lines, parted by newlines and followed by
// end.
func joinText(lines []ast.Line, end string) string {
	if len(lines) == 1 && end == "" {
		return lines[0].Text
	}
	n := len(end)
	for _, l := range lines {
		n += len(l.Text) + 1
	}
	var b strings.Builder
	b.Grow(n)
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(l.Text)
	}
	b.WriteString(end)
	return b.String()
}

// sourceOffset returns the offset in the source of offset k in the text of
// lines, as joinText joins them, which holds no spaces of tabs.
func sourceOffset(lines []ast.Line, k int) int {
	for _, l := range lines {
		if k <= len(l.Text) {
			return l.Offset + k
		}
		k -= len(l.Text) + 1
	}
	if len(lines) == 0 {
		return 0
	}
	last := lines[len(lines)-1]
	return last.Offset + len(last.Text)
}

// peek returns the byte at offset i of s, or -1 past its end.
func peek(s string, i int) int {
	if i >= len(s) {
		return -1
	}
	return int(s[i])
}

// runLen returns the number of consecutive bytes c at offset i of s.
func runLen(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
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

func isSpaceOrTab[T byte | int](c T) bool { return c == ' ' || c == '\t' }
func isDigit(c byte) bool                 { return '0' <= c && c <= '9' }
func isLetter(c byte) bool                { return 'a' <= c|0x20 && c|0x20 <= 'z' }
func isAlnum(c byte) bool                 { return isDigit(c) || isLetter(c) }
