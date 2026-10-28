package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// fencedCodeBlockParser parses fenced code blocks.
type fencedCodeBlockParser struct{}

// NewFencedCodeBlockParser returns the parser of fenced code blocks, which
// start with three or more "`" or "~" characters.
func NewFencedCodeBlockParser() BlockParser { return fencedCodeBlockParser{} }

// Trigger implements [BlockParser].
func (fencedCodeBlockParser) Trigger() []byte { return []byte{'`', '~'} }

// Open implements [BlockParser]. The rest of the fence line is the block's
// info string.
func (fencedCodeBlockParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	rest := s.Rest()
	if !strings.HasPrefix(rest, "```") && !strings.HasPrefix(rest, "~~~") {
		return nil, None
	}
	c := rest[0]
	n := runLen(rest, 0, c)
	// The info string after a backtick fence holds no backticks.
	if c == '`' && strings.IndexByte(rest[n:], '`') >= 0 {
		return nil, None
	}
	b := &markdown.FencedCodeBlock{FenceChar: c, FenceLength: n, FenceOffset: s.indent}
	s.advanceNextNonspace()
	s.advanceOffset(n, false)
	return b, NoChildren
}

// Continue implements [BlockParser]. A closing fence closes the block and
// consumes the line it is on.
func (fencedCodeBlockParser) Continue(n markdown.Node, s *BlockState) State {
	b := n.(*markdown.FencedCodeBlock)
	ln := s.line
	if s.indent <= 3 && peek(ln, s.nextNonspace) == int(b.FenceChar) {
		f := runLen(ln, s.nextNonspace, b.FenceChar)
		if f >= b.FenceLength && isBlank(ln[s.nextNonspace+f:]) {
			s.Close(n, s.lineNumber)
			sp := n.SourceSpan()
			sp.End = s.lineOffset + s.nextNonspace + f // the closing fence's end
			n.SetSourceSpan(sp)
			return Close
		}
	}
	// Skip as much indentation as the opening fence had.
	for i := b.FenceOffset; i > 0 && isSpaceOrTab(peek(ln, s.offset)); i-- {
		s.advanceOffset(1, true)
	}
	return Continue | NoChildren
}

// AddLine implements [LineAdder].
func (fencedCodeBlockParser) AddLine(n markdown.Node, s *BlockState) {
	n.SetLines(append(n.Lines(), s.Remaining()))
}

// Close implements [BlockParser]. The block's first line is its info
// string; its lines are those after it.
func (fencedCodeBlockParser) Close(n markdown.Node, s *BlockState) {
	b := n.(*markdown.FencedCodeBlock)
	lines := n.Lines()
	b.Info = replaceNUL(unescape(strings.Trim(lines[0].Text, " \t")))
	b.Literal = joinLines(lines[1:])
	n.SetLines(lines[1:])
}

// CanInterruptParagraph implements [BlockParser].
func (fencedCodeBlockParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (fencedCodeBlockParser) CanAcceptIndentedLine() bool { return false }

// codeBlockParser parses indented code blocks.
type codeBlockParser struct{}

// NewCodeBlockParser returns the parser of indented code blocks, at lines
// indented 4 columns or more.
func NewCodeBlockParser() BlockParser { return codeBlockParser{} }

// Trigger implements [BlockParser].
func (codeBlockParser) Trigger() []byte { return nil }

// Open implements [BlockParser]. The block starts after its indentation.
func (codeBlockParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	if !s.Indented() {
		return nil, None
	}
	s.advanceOffset(4, true)
	s.StartAt(s.lineOffset + s.offset)
	return &markdown.CodeBlock{}, NoChildren
}

// Continue implements [BlockParser].
func (codeBlockParser) Continue(n markdown.Node, s *BlockState) State {
	switch {
	case s.indent >= 4:
		s.advanceOffset(4, true)
	case s.blank:
		s.advanceNextNonspace()
	default:
		return Close
	}
	return Continue | NoChildren
}

// AddLine implements [LineAdder].
func (codeBlockParser) AddLine(n markdown.Node, s *BlockState) {
	n.SetLines(append(n.Lines(), s.Remaining()))
}

// Close implements [BlockParser]. The block ends before its trailing blank
// lines, so that a blank line after it parts it from the next block.
func (codeBlockParser) Close(n markdown.Node, s *BlockState) {
	lines := n.Lines()
	end := len(lines)
	for end > 0 && isBlank(lines[end-1].Text) {
		end--
	}
	n.(*markdown.CodeBlock).Literal = joinLines(lines[:end])
	n.SetLines(lines[:end])
	// The block took each of its lines, so its last nonblank line is as
	// many lines before its last as it has trailing blank lines.
	if blanks := len(lines) - end; blanks > 0 && end > 0 {
		sp := n.SourceSpan()
		sp.End = s.lineEnds[s.lineOf(sp.End)-blanks]
		n.SetSourceSpan(sp)
	}
}

// CanInterruptParagraph implements [BlockParser]; indented lines continue
// a paragraph.
func (codeBlockParser) CanInterruptParagraph() bool { return false }

// CanAcceptIndentedLine implements [BlockParser].
func (codeBlockParser) CanAcceptIndentedLine() bool { return true }

// joinLines returns the text of lines, each followed by a newline.
func joinLines(lines []markdown.Line) string {
	if len(lines) == 0 {
		return ""
	}
	return replaceNUL(joinText(lines, "\n"))
}
