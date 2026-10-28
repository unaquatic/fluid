package parser

import "github.com/unaquatic/fluid/internal/markdown"

// listParser parses lists and list items.
type listParser struct{}

// NewListParser returns the parser of list items, and of the lists that
// hold them, which start with a bullet list marker ("-", "+", or "*") or an
// ordered list marker (up to 9 digits and "." or ")").
func NewListParser() BlockParser { return listParser{} }

// Trigger implements [BlockParser].
func (listParser) Trigger() []byte { return []byte("-+*0123456789") }

// Open implements [BlockParser]. It adds a list for the item unless the
// item continues one.
func (p listParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	m, ok := s.parseListMarker(parent)
	if !ok {
		return nil, None
	}
	s.CloseUnmatched()
	if l, ok := s.Tip().(*markdown.List); !ok || l.Marker != m.marker {
		s.Add(&markdown.List{Marker: m.marker, IsTight: true, Start: m.start}, p, HasChildren)
	}
	return &markdown.ListItem{Offset: m.offset + m.padding}, HasChildren
}

// listMarker is the marker of a list item.
type listMarker struct {
	marker  byte // the bullet, or the delimiter after the number
	start   int  // the number of an ordered item
	offset  int  // columns of indentation before the marker
	padding int  // columns from the start of the marker to the item's content
}

// parseListMarker parses the list item marker at the current position, if
// any, within container, and consumes it and the spaces after it that
// belong to the item's indentation.
func (s *BlockState) parseListMarker(container markdown.Node) (listMarker, bool) {
	if s.indent >= 4 {
		return listMarker{}, false
	}
	ln := s.line
	rest := ln[s.nextNonspace:]
	m := listMarker{offset: s.indent}
	_, inPara := container.(*markdown.Paragraph)
	var n int // length of the marker
	switch {
	case rest == "":
		return m, false
	case rest[0] == '*' || rest[0] == '+' || rest[0] == '-':
		m.marker = rest[0]
		n = 1
	default:
		for n < len(rest) && n < 10 && isDigit(rest[n]) {
			n++
		}
		if n == 0 || n > 9 || n >= len(rest) || rest[n] != '.' && rest[n] != ')' {
			return m, false
		}
		for _, c := range []byte(rest[:n]) {
			m.start = m.start*10 + int(c-'0')
		}
		// Only an item numbered 1 can interrupt a paragraph.
		if inPara && m.start != 1 {
			return m, false
		}
		m.marker = rest[n]
		n++
	}

	// The marker must be followed by a space, a tab, or the end of the line.
	if c := peek(ln, s.nextNonspace+n); c != -1 && !isSpaceOrTab(c) {
		return m, false
	}
	// An item that interrupts a paragraph cannot start with a blank line.
	if inPara && isBlank(ln[s.nextNonspace+n:]) {
		return m, false
	}

	s.advanceNextNonspace()
	s.advanceOffset(n, true)
	spacesStartCol, spacesStartOffset := s.column, s.offset
	for {
		s.advanceOffset(1, true)
		c := peek(ln, s.offset)
		if s.column-spacesStartCol >= 5 || c == -1 || !isSpaceOrTab(c) {
			break
		}
	}
	blankItem := peek(ln, s.offset) == -1
	spaces := s.column - spacesStartCol
	if spaces >= 5 || spaces < 1 || blankItem {
		// The content starts one space after the marker; further spaces
		// are part of it, as for indented code.
		m.padding = n + 1
		s.column, s.offset = spacesStartCol, spacesStartOffset
		if isSpaceOrTab(peek(ln, s.offset)) {
			s.advanceOffset(1, true)
		}
	} else {
		m.padding = n + spaces
	}
	return m, true
}

// Continue implements [BlockParser]. A list continues while its items do,
// and a line continues an item if it is indented to the item's content, or
// is blank after the item's first line.
func (listParser) Continue(n markdown.Node, s *BlockState) State {
	item, ok := n.(*markdown.ListItem)
	if !ok {
		return Continue | HasChildren
	}
	switch {
	case s.blank:
		if !n.HasChildren() {
			return Close // a blank line after an empty item ends it
		}
		s.advanceNextNonspace()
	case s.indent >= item.Offset:
		s.advanceOffset(item.Offset, true)
	default:
		return Close
	}
	return Continue | HasChildren
}

// Close implements [BlockParser]. A list ends with its last item, and an
// item with its last block or, if it has none, its marker.
func (listParser) Close(n markdown.Node, s *BlockState) {
	sp := n.SourceSpan()
	switch last := n.LastChild(); {
	case last != nil:
		sp.End = last.SourceSpan().End
	default:
		sp.End = sp.Start + markerWidth(s.src[sp.Start:])
	}
	n.SetSourceSpan(sp)
}

// markerWidth returns the length of the list item marker at the start of
// s.
func markerWidth(s string) int {
	n := 0
	for n < len(s) && isDigit(s[n]) {
		n++
	}
	return n + 1
}

// CanInterruptParagraph implements [BlockParser].
func (listParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (listParser) CanAcceptIndentedLine() bool { return false }
