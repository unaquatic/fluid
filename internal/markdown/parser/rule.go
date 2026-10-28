package parser

import "github.com/unaquatic/fluid/internal/markdown"

// thematicBreakParser parses thematic breaks.
type thematicBreakParser struct{}

// NewThematicBreakParser returns the parser of thematic breaks: lines of
// three or more "*", "-", or "_" characters, and spaces and tabs.
func NewThematicBreakParser() BlockParser { return thematicBreakParser{} }

// Trigger implements [BlockParser].
func (thematicBreakParser) Trigger() []byte { return []byte{'*', '-', '_'} }

// Open implements [BlockParser].
func (thematicBreakParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	if s.nextNonspace < s.ruleFail {
		return nil, None
	}
	n, ok := thematicBreak(s.Rest())
	if !ok {
		if n > 0 {
			s.ruleFail = s.nextNonspace + n
		}
		return nil, None
	}
	s.advanceOffset(len(s.line)-s.offset, false)
	return &markdown.ThematicBreak{}, NoChildren
}

// Continue implements [BlockParser]; a thematic break is one line.
func (thematicBreakParser) Continue(markdown.Node, *BlockState) State { return Close }

// Close implements [BlockParser].
func (thematicBreakParser) Close(markdown.Node, *BlockState) {}

// CanInterruptParagraph implements [BlockParser].
func (thematicBreakParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (thematicBreakParser) CanAcceptIndentedLine() bool { return false }

// thematicBreak reports whether line s, from its first character that is
// not a space or tab, is a thematic break. If s starts with a character of
// one but is not one, it also returns the offset of the character that ends
// it, which also ends any that starts between.
func thematicBreak(s string) (int, bool) {
	if s == "" || s[0] != '*' && s[0] != '-' && s[0] != '_' {
		return 0, false
	}
	n := 0
	for i := range len(s) {
		switch s[i] {
		case s[0]:
			n++
		case ' ', '\t':
		default:
			return i, false
		}
	}
	return len(s), n >= 3
}
