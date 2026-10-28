package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// codeSpanParser parses code spans.
type codeSpanParser struct{}

// NewCodeSpanParser returns the parser of code spans: text between two
// strings of backticks of the same length. A string of backticks that none
// closes is text.
func NewCodeSpanParser() InlineParser { return codeSpanParser{} }

// Trigger implements [InlineParser].
func (codeSpanParser) Trigger() []byte { return []byte{'`'} }

// noTicksKey keys the set of lengths of backtick strings that no string of
// the same length follows past the position parsed, so that none closes
// them, in the values of an [InlineState].
type noTicksKey struct{}

// Parse implements [InlineParser].
func (codeSpanParser) Parse(parent markdown.Node, s *InlineState) (markdown.Node, int) {
	src, start := s.Source(), s.Pos()
	n := runLen(src, start, '`')
	after := start + n
	noTicks, _ := s.Value(noTicksKey{}).(map[int]bool)
	if !noTicks[n] {
		for i := after; ; {
			j := strings.IndexByte(src[i:], '`')
			if j < 0 {
				break
			}
			j += i
			k := runLen(src, j, '`')
			if k == n {
				// Line endings become spaces, and one space is stripped from
				// each end of code that has spaces at both ends and not only
				// spaces.
				code := strings.ReplaceAll(src[after:j], "\n", " ")
				if code != "" && code[0] == ' ' && code[len(code)-1] == ' ' && strings.Trim(code, " ") != "" {
					code = code[1 : len(code)-1]
				}
				cs := &markdown.CodeSpan{}
				cs.AppendChild(cs, markdown.NewText(code))
				return cs, j + k - start
			}
			i = j + k
		}
		if noTicks == nil {
			noTicks = make(map[int]bool)
			s.SetValue(noTicksKey{}, noTicks)
		}
		noTicks[n] = true
	}
	return markdown.NewText(src[start:after]), n
}
