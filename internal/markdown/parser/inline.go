package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// An InlineParser parses one kind of inline element.
type InlineParser interface {
	// Trigger returns the bytes that can start the element.
	Trigger() []byte

	// Parse parses the element at the current position of s, in block
	// parent, if one starts there, and returns it and its length, or nil
	// and 0. A [*ast.Text] that it returns joins the text around it.
	Parse(parent ast.Node, s *InlineState) (ast.Node, int)
}

// An InlineState is the state of a [Parser] as it parses the inline content
// of a block: the content, and the position that it is parsed to.
type InlineState struct {
	syn   *syntax
	refs  map[string]*ast.LinkDef
	block *BlockState // the state of the document's blocks, which holds its values

	src        string
	pos        int
	delimStack *Delimiter
	brackets   *bracket

	// bottoms holds processEmphasis's bounds of the search for openers,
	// for each class of delimiters.
	bottoms [][2][3]*Delimiter

	// lastIndex holds the offsets of the last occurrences of strings in
	// the content, as HasAfter finds them.
	lastIndex map[string]int

	values map[any]any
}

// newInlineState returns the state of a Parser that parses the inline
// content of the document whose blocks b parsed.
func newInlineState(b *BlockState) *InlineState {
	return &InlineState{syn: b.syntax, refs: b.refs, block: b, bottoms: make([][2][3]*Delimiter, len(b.syntax.delims))}
}

// Source returns the inline content being parsed, without the spaces, tabs,
// and line endings around it, and with its NUL characters replaced with
// U+FFFD.
func (s *InlineState) Source() string { return s.src }

// Pos returns the offset in Source of the element to parse.
func (s *InlineState) Pos() int { return s.pos }

// Rest returns Source from Pos.
func (s *InlineState) Rest() string { return s.src[s.pos:] }

// InLinkText reports whether Pos may be in the text of a link or image,
// after an opening bracket that no bracket has closed.
func (s *InlineState) InLinkText() bool { return s.brackets != nil }

// LinkDef returns the document's link reference definition for label,
// which is matched as CommonMark matches labels.
func (s *InlineState) LinkDef(label string) (*ast.LinkDef, bool) {
	d, ok := s.refs[normalizeLabel(label)]
	return d, ok
}

// HasAfter reports whether sub occurs in Source at offset i or after it. It
// finds the last occurrence of each sub once for each inline content, so
// that each of many unterminated elements need not scan the rest of it.
func (s *InlineState) HasAfter(sub string, i int) bool {
	last, ok := s.lastIndex[sub]
	if !ok {
		if s.lastIndex == nil {
			s.lastIndex = make(map[string]int)
		}
		last = strings.LastIndex(s.src, sub)
		s.lastIndex[sub] = last
	}
	return last >= i
}

// Value returns the value that SetValue stored for key while parsing the
// current inline content, or nil.
func (s *InlineState) Value(key any) any { return s.values[key] }

// SetValue stores value for key while parsing the current inline content.
func (s *InlineState) SetValue(key, value any) {
	if s.values == nil {
		s.values = make(map[any]any)
	}
	s.values[key] = value
}

// DocumentValue returns the value that SetDocumentValue, or the
// [BlockState.SetValue] of the document's blocks, stored for key while
// parsing the current document, or nil. Values that the block parsers
// store, such as the definitions they find, last through the parsing of
// every inline content of the document.
func (s *InlineState) DocumentValue(key any) any { return s.block.Value(key) }

// SetDocumentValue stores value for key while parsing the current
// document.
func (s *InlineState) SetDocumentValue(key, value any) { s.block.SetValue(key, value) }

// parse parses src as the inline content of block b, whose children it
// becomes.
func (s *InlineState) parse(b ast.Node, src string) {
	s.src = replaceNUL(strings.Trim(src, " \t\n"))
	s.pos = 0
	s.delimStack, s.brackets = nil, nil
	clear(s.lastIndex)
	clear(s.values)
	for s.pos < len(s.src) {
		s.parseInline(b)
	}
	s.processEmphasis(nil)
	joinTexts(b)
}

// parseInline parses the inline element at pos and adds it to b.
func (s *InlineState) parseInline(b ast.Node) {
	c := s.src[s.pos]
	for _, ip := range s.syn.triggers[c] {
		if n, l := ip.Parse(b, s); l > 0 {
			b.AppendChild(b, n)
			s.pos += l
			return
		}
	}
	switch {
	case c == '\n':
		s.parseNewline(b)
	case c == '[':
		t := ast.NewText("[")
		b.AppendChild(b, t)
		s.addBracket(t, s.pos, false)
		s.pos++
	case c == '!':
		s.parseBang(b)
	case c == ']':
		s.parseCloseBracket(b)
	case s.syn.delimIdx[c] > 0:
		s.handleDelim(c, b)
	case s.syn.special[c]:
		b.AppendChild(b, ast.NewText(s.src[s.pos:s.pos+1]))
		s.pos++
	default:
		s.parseString(b)
	}
}

// parseString adds the text from pos to the next special byte to b, or the
// text before the first element that a word parser parses in it, and the
// element.
func (s *InlineState) parseString(b ast.Node) {
	start := s.pos
	i := start
	for {
		if s.syn.wordStart[s.src[i]] && (i == 0 || !isAlnum(s.src[i-1])) && !s.afterEscapedSpace(i) {
			s.pos = i
			for _, w := range s.syn.words {
				if n, l := w.Parse(b, s); l > 0 {
					if i > start {
						b.AppendChild(b, ast.NewText(s.src[start:i]))
					}
					b.AppendChild(b, n)
					s.pos = i + l
					return
				}
			}
		}
		if i++; i == len(s.src) || s.syn.special[s.src[i]] {
			break
		}
	}
	b.AppendChild(b, ast.NewText(s.src[start:i]))
	s.pos = i
}

// afterEscapedSpace reports whether offset i of the content follows a space
// that a backslash escapes, with WithEscapedSpace, after which a word does
// not start.
func (s *InlineState) afterEscapedSpace(i int) bool {
	if !s.syn.escapedSpace || i < 2 || s.src[i-1] != ' ' {
		return false
	}
	n := 0 // backslashes before the space
	for j := i - 2; j >= 0 && s.src[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

// parseNewline adds a soft or hard line break to b for the line ending at
// pos, removing the spaces around it. The break is an empty text, which
// joins the text before it.
func (s *InlineState) parseNewline(b ast.Node) {
	s.pos++
	br := ast.NewText("")
	if last, ok := b.LastChild().(*ast.Text); ok && strings.HasSuffix(last.Value, " ") {
		br.SetHardLineBreak(strings.HasSuffix(last.Value, "  "))
		last.Value = strings.TrimRight(last.Value, " ")
	}
	br.SetSoftLineBreak(!br.HardLineBreak())
	b.AppendChild(b, br)
	for s.pos < len(s.src) && s.src[s.pos] == ' ' {
		s.pos++
	}
}

// joinTexts joins the adjacent texts among the descendants of n, of which
// only the last can have a line break after it, and removes empty texts
// with none.
func joinTexts(n ast.Node) {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; {
		t, ok := c.(*ast.Text)
		if !ok {
			joinTexts(c)
			c = c.NextSibling()
			continue
		}
		next := c.NextSibling()
		if nt, ok := next.(*ast.Text); ok && !hasBreak(t) {
			b.WriteString(t.Value)
			for ok && !hasBreak(t) {
				b.WriteString(nt.Value)
				t.SetSoftLineBreak(nt.SoftLineBreak())
				t.SetHardLineBreak(nt.HardLineBreak())
				next = nt.NextSibling()
				n.RemoveChild(n, nt)
				nt, ok = next.(*ast.Text)
			}
			t.Value = b.String()
			b.Reset()
		}
		if t.Value == "" && !hasBreak(t) {
			n.RemoveChild(n, t)
		}
		c = next
	}
}

// hasBreak reports whether a line break follows t.
func hasBreak(t *ast.Text) bool { return t.SoftLineBreak() || t.HardLineBreak() }
