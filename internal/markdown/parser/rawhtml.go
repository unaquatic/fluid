package parser

import "github.com/unaquatic/fluid/internal/markdown"

// rawHTMLParser parses inline HTML.
type rawHTMLParser struct{}

// NewRawHTMLParser returns the parser of inline HTML: open and closing
// tags, comments, processing instructions, declarations, and CDATA
// sections.
func NewRawHTMLParser() InlineParser { return rawHTMLParser{} }

// Trigger implements [InlineParser].
func (rawHTMLParser) Trigger() []byte { return []byte{'<'} }

// Parse implements [InlineParser]. It checks that the string that ends a
// comment, processing instruction, declaration, or CDATA section follows
// before it scans for it, so that unterminated ones do not each scan to the
// end of the text.
func (rawHTMLParser) Parse(parent markdown.Node, s *InlineState) (markdown.Node, int) {
	r := s.Rest()
	if end := htmlTagEnd(r); end != "" && !s.HasAfter(end, s.pos+2) {
		return nil, 0
	}
	n := htmlTag(r)
	if n == 0 {
		return nil, 0
	}
	return &markdown.RawHTML{Value: r[:n]}, n
}
