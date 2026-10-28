package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// atxHeadingParser parses ATX headings.
type atxHeadingParser struct{}

// NewATXHeadingParser returns the parser of ATX headings, which start with
// one to six "#" characters.
func NewATXHeadingParser() BlockParser { return atxHeadingParser{} }

// Trigger implements [BlockParser].
func (atxHeadingParser) Trigger() []byte { return []byte{'#'} }

// Open implements [BlockParser].
func (atxHeadingParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	rest := s.Rest()
	level := runLen(rest, 0, '#')
	if level == 0 || level > 6 || level < len(rest) && !isSpaceOrTab(rest[level]) {
		return nil, None
	}
	n := level
	for n < len(rest) && isSpaceOrTab(rest[n]) {
		n++
	}
	s.advanceNextNonspace()
	s.advanceOffset(n, false)
	h := &markdown.Heading{Level: level}
	text := s.headingAttributes(h, s.line[s.offset:])
	h.SetLines([]markdown.Line{{Text: trimClosingSequence(text), Offset: s.lineOffset + s.offset}})
	s.advanceOffset(len(s.line)-s.offset, false)
	return h, NoChildren
}

// Continue implements [BlockParser]; a heading is one line.
func (atxHeadingParser) Continue(markdown.Node, *BlockState) State { return Close }

// Close implements [BlockParser].
func (atxHeadingParser) Close(n markdown.Node, s *BlockState) { s.headingID(n.(*markdown.Heading)) }

// CanInterruptParagraph implements [BlockParser].
func (atxHeadingParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (atxHeadingParser) CanAcceptIndentedLine() bool { return false }

// trimClosingSequence removes the optional closing sequence of '#'
// characters from the content of an ATX heading.
func trimClosingSequence(s string) string {
	t := strings.TrimRight(s, " \t")
	u := strings.TrimRight(t, "#")
	switch {
	case u == "":
		return ""
	case len(u) < len(t) && isSpaceOrTab(u[len(u)-1]):
		return strings.TrimRight(u, " \t")
	}
	return s
}

// setextHeadingParser parses setext headings.
type setextHeadingParser struct{}

// NewSetextHeadingParser returns the parser of setext headings, which turn
// a paragraph into a heading at a line of "=" or "-" characters under it.
func NewSetextHeadingParser() BlockParser { return setextHeadingParser{} }

// Trigger implements [BlockParser].
func (setextHeadingParser) Trigger() []byte { return []byte{'=', '-'} }

// Open implements [BlockParser]. It puts the heading in the paragraph's
// place, with its lines after the link reference definitions at its start.
func (p setextHeadingParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	para, ok := parent.(*markdown.Paragraph)
	if !ok {
		return nil, None
	}
	level := setextLevel(s.Rest())
	if level == 0 {
		return nil, None
	}
	s.CloseUnmatched()
	lines, start := s.takeRefs(para.Lines())
	para.SetLines(nil)
	if len(lines) == 0 || isBlank(joinText(lines, "")) {
		// The paragraph held only link reference definitions, which leave
		// no heading text.
		return nil, None
	}
	h := &markdown.Heading{Level: level}
	last := &lines[len(lines)-1]
	last.Text = s.headingAttributes(h, last.Text)
	h.SetLines(lines)
	h.SetSourceSpan(markdown.Span{Start: start, End: start})
	grand := para.Parent()
	grand.ReplaceChild(grand, para, h)
	tip := s.open[len(s.open)-1]
	tip.node, tip.parser = h, p
	s.advanceOffset(len(s.line)-s.offset, false)
	return h, NoChildren
}

// Continue implements [BlockParser]; a setext heading ends at its
// underline.
func (setextHeadingParser) Continue(markdown.Node, *BlockState) State { return Close }

// Close implements [BlockParser].
func (setextHeadingParser) Close(n markdown.Node, s *BlockState) { s.headingID(n.(*markdown.Heading)) }

// CanInterruptParagraph implements [BlockParser].
func (setextHeadingParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (setextHeadingParser) CanAcceptIndentedLine() bool { return false }

// headingAttributes sets the attributes at the end of text, a line of the
// content of heading h, on h, if the parser lets headings end with them, and
// returns text without them.
func (s *BlockState) headingAttributes(h *markdown.Heading, text string) string {
	if !s.syntax.attribute {
		return text
	}
	attrs, rest := trailingAttributes(text)
	for _, a := range attrs {
		h.SetAttribute(a.Name, a.Value)
	}
	return rest
}

// headingID gives heading h an ID, if the parser gives headings them,
// unless it has an attribute "id", which the document's IDs record as used.
// The IDs generate it from the source of the heading's last line.
func (s *BlockState) headingID(h *markdown.Heading) {
	if !s.syntax.autoHeadingID {
		return
	}
	if id, ok := h.Attribute("id"); ok {
		if id, ok := id.(string); ok {
			s.IDs().Put(id)
		}
		return
	}
	var text string
	if lines := h.Lines(); len(lines) > 0 {
		text = lines[len(lines)-1].Text
	}
	h.SetAttribute("id", s.IDs().Generate(text, markdown.KindHeading))
}

// setextLevel returns the level of the setext heading that line s, from its
// first character that is not a space or tab, underlines, or 0 if s is not
// an underline.
func setextLevel(s string) int {
	if s == "" || s[0] != '=' && s[0] != '-' {
		return 0
	}
	n := runLen(s, 0, s[0])
	if !isBlank(s[n:]) {
		return 0
	}
	if s[0] == '=' {
		return 1
	}
	return 2
}
