package parser

import "github.com/unaquatic/fluid/internal/markdown"

// blockquoteParser parses block quotes.
type blockquoteParser struct{}

// NewBlockquoteParser returns the parser of block quotes, whose lines start
// with ">".
func NewBlockquoteParser() BlockParser { return blockquoteParser{} }

// Trigger implements [BlockParser].
func (blockquoteParser) Trigger() []byte { return []byte{'>'} }

// Open implements [BlockParser].
func (blockquoteParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	s.consumeQuoteMarker()
	return &markdown.Blockquote{}, HasChildren
}

// Continue implements [BlockParser]; a line continues a block quote if it
// starts with its marker.
func (blockquoteParser) Continue(n markdown.Node, s *BlockState) State {
	if s.Indented() || peek(s.line, s.NextNonspace()) != '>' {
		return Close
	}
	s.consumeQuoteMarker()
	return Continue | HasChildren
}

// Close implements [BlockParser].
func (blockquoteParser) Close(markdown.Node, *BlockState) {}

// CanInterruptParagraph implements [BlockParser].
func (blockquoteParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (blockquoteParser) CanAcceptIndentedLine() bool { return false }

// consumeQuoteMarker consumes the ">" at nextNonspace and the space or tab,
// or a column of the tab, after it.
func (s *BlockState) consumeQuoteMarker() {
	s.advanceNextNonspace()
	s.advanceOffset(1, false)
	if isSpaceOrTab(peek(s.line, s.offset)) {
		s.advanceOffset(1, true)
	}
}
