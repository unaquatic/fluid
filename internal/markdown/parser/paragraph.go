package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// paragraphParser parses paragraphs, which [NewParagraphParser] returns.
type paragraphParser struct{}

// NewParagraphParser returns the parser of paragraphs, which start where a
// line has text that no other block takes. A [Parser] opens paragraphs
// itself, so the parser's Open opens none. As a paragraph closes, the link
// reference definitions at its start become the document's, and the
// paragraph starts after them, or goes if nothing else is left.
func NewParagraphParser() BlockParser { return paragraphParser{} }

// Trigger implements [BlockParser].
func (paragraphParser) Trigger() []byte { return nil }

// Open implements [BlockParser]; it opens nothing.
func (paragraphParser) Open(markdown.Node, *BlockState) (markdown.Node, State) { return nil, None }

// Continue implements [BlockParser]; a blank line ends a paragraph, and any
// other line continues it unless another block starts on it.
func (paragraphParser) Continue(n markdown.Node, s *BlockState) State {
	if s.Blank() {
		return Close
	}
	return Continue
}

// AddLine implements [LineAdder].
func (paragraphParser) AddLine(n markdown.Node, s *BlockState) {
	n.SetLines(append(n.Lines(), s.Remaining()))
}

// Close implements [BlockParser].
func (paragraphParser) Close(n markdown.Node, s *BlockState) {
	lines, start := s.takeRefs(n.Lines())
	n.SetLines(lines)
	parent := n.Parent()
	if len(lines) == 0 || isBlank(joinText(lines, "")) {
		parent.RemoveChild(parent, n)
		return
	}
	sp := n.SourceSpan()
	sp.Start = start
	n.SetSourceSpan(sp)
	p := n.(*markdown.Paragraph)
	for _, t := range s.syntax.paragraphTransformers {
		if t.Transform(p, s); p.Parent() == nil {
			return
		}
	}
}

// CanInterruptParagraph implements [BlockParser].
func (paragraphParser) CanInterruptParagraph() bool { return false }

// CanAcceptIndentedLine implements [BlockParser].
func (paragraphParser) CanAcceptIndentedLine() bool { return false }

// takeRefs adds the link reference definitions at the start of lines, the
// lines of a paragraph, to the document, and returns the lines after them
// and the offset in the source at which those start.
func (s *BlockState) takeRefs(lines []markdown.Line) ([]markdown.Line, int) {
	if len(lines) == 0 || !strings.HasPrefix(lines[0].Text, "[") {
		return lines, sourceOffset(lines, 0)
	}
	text := joinText(lines, "")
	pos := 0
	for strings.HasPrefix(text[pos:], "[") {
		n := s.parseReference(text[pos:], func(i int) int { return sourceOffset(lines, pos+i) })
		if n == 0 {
			break
		}
		pos += n
	}
	// A definition ends at the end of a line, so the rest starts at a line.
	k := 0
	for i := 0; k < len(lines) && i < pos; k++ {
		i += len(lines[k].Text) + 1
	}
	return lines[k:], sourceOffset(lines, pos)
}

// parseReference adds the link reference definition at the start of src,
// if any, to the document, and returns its length, including the line
// ending after it, or 0. The definition's span is in the source, at which
// at finds the offsets in src.
func (s *BlockState) parseReference(src string, at func(int) int) int {
	n := linkLabel(src)
	if n == 0 || n >= len(src) || src[n] != ':' {
		return 0
	}
	label := src[1 : n-1]
	i := spnl(src, n+1)
	url, dn, ok := linkDestination(src[i:])
	if !ok || dn == 0 && !strings.HasPrefix(src[i:], "<") {
		return 0
	}
	i += dn

	beforeTitle := i
	i = spnl(src, i)
	title, tn, hasTitle := "", 0, false
	if i > beforeTitle {
		title, tn, hasTitle = linkTitle(src[i:])
	}
	if hasTitle {
		i += tn
	} else {
		i = beforeTitle
	}
	end, ok := lineEnd(src, i)
	if !ok && hasTitle {
		// The title is not at the line's end, but the destination may be,
		// which makes a definition without the title.
		title = ""
		i = beforeTitle
		end, ok = lineEnd(src, i)
	}
	if !ok {
		return 0
	}

	key := normalizeLabel(label)
	if key == "" {
		return 0
	}
	if _, dup := s.refs[key]; !dup {
		d := &markdown.LinkDef{
			Span:  markdown.Span{Start: at(0), End: at(i)},
			Label: replaceNUL(label),
			URL:   replaceNUL(url),
			Title: replaceNUL(title),
		}
		s.refs[key] = d
		s.doc.Links = append(s.doc.Links, d)
	}
	return end
}

// lineEnd returns the offset after the spaces, tabs, and line ending at
// offset i of s, and reports whether only those, or the end of s, follow i.
func lineEnd(s string, i int) (int, bool) {
	for i < len(s) && isSpaceOrTab(s[i]) {
		i++
	}
	switch {
	case i == len(s):
		return i, true
	case s[i] == '\n':
		return i + 1, true
	}
	return 0, false
}
