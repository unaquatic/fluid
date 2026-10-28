package parser

import (
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown"
)

// A DelimiterProcessor processes the runs of delimiter characters, such as
// "*", that delimit spans of inline text. A [Parser] scans the runs, and
// pairs those that open spans with those that close them by the rules
// CommonMark gives for emphasis, as the processor allows.
type DelimiterProcessor interface {
	// IsDelimiter reports whether c is a delimiter character that the
	// processor processes.
	IsDelimiter(c byte) bool

	// CanOpenCloser reports whether opener can open the span that closer
	// closes.
	CanOpenCloser(opener, closer *Delimiter) bool

	// OnMatch returns the node of a span between runs that consume
	// characters at each end, whose children the parser adds.
	OnMatch(consumes int) markdown.Node
}

// A Delimiter is a run of delimiter characters, as a [DelimiterProcessor]
// sees it.
type Delimiter struct {
	Char           byte
	Length         int // characters left, which spans have not used
	OriginalLength int
	CanOpen        bool
	CanClose       bool
	Processor      DelimiterProcessor

	node       *markdown.Text // the run's text
	class      int            // index of the run's class in the syntax
	prev, next *Delimiter
}

// A DelimiterRule is a [DelimiterProcessor] of one character, whose runs
// pair with runs of the same character, as "*" delimits emphasis.
type DelimiterRule struct {
	Char byte

	// MaxRun is the length of the longest run that can open or close a
	// span. A longer run is text. Zero means no limit.
	MaxRun int

	// Intraword reports whether runs inside a word can open and close
	// spans, as those of "*" can and those of "_" cannot.
	Intraword bool

	// New returns the node of a span between runs that consume 1 or 2
	// characters at each end.
	New func(consumes int) markdown.Node
}

// IsDelimiter implements [DelimiterProcessor].
func (r DelimiterRule) IsDelimiter(c byte) bool { return c == r.Char }

// CanOpenCloser implements [DelimiterProcessor]; runs of the same character
// pair.
func (r DelimiterRule) CanOpenCloser(opener, closer *Delimiter) bool {
	return opener.Char == closer.Char
}

// OnMatch implements [DelimiterProcessor].
func (r DelimiterRule) OnMatch(consumes int) markdown.Node { return r.New(consumes) }

// handleDelim adds the run of delimiter c at pos to b as text and, if it
// can open or close a span, to the delimiter stack.
func (s *InlineState) handleDelim(c byte, b markdown.Node) {
	class := s.syn.delimIdx[c] - 1
	dc := s.syn.delims[class]
	n, canOpen, canClose := s.scanDelims(c, dc.intraword)
	node := markdown.NewText(s.src[s.pos : s.pos+n])
	b.AppendChild(b, node)
	s.pos += n
	if dc.maxRun > 0 && n > dc.maxRun || !canOpen && !canClose {
		return
	}
	d := &Delimiter{
		Char: c, Length: n, OriginalLength: n, CanOpen: canOpen, CanClose: canClose, Processor: dc.processor,
		node: node, class: class, prev: s.delimStack,
	}
	if s.delimStack != nil {
		s.delimStack.next = d
	}
	s.delimStack = d
}

// scanDelims returns the length of the run of delimiter c at pos and
// whether it can open and close spans, which a run inside a word can only
// if intraword is set.
func (s *InlineState) scanDelims(c byte, intraword bool) (n int, canOpen, canClose bool) {
	n = runLen(s.src, s.pos, c)
	before, after := '\n', '\n'
	if s.pos > 0 {
		before, _ = utf8.DecodeLastRuneInString(s.src[:s.pos])
	}
	if s.pos+n < len(s.src) {
		after, _ = utf8.DecodeRuneInString(s.src[s.pos+n:])
	}
	beforeSpace, beforePunct := isUnicodeSpace(before), isPunct(before)
	afterSpace, afterPunct := isUnicodeSpace(after), isPunct(after)
	left := !afterSpace && (!afterPunct || beforeSpace || beforePunct)
	right := !beforeSpace && (!beforePunct || afterSpace || afterPunct)
	if !intraword {
		return n, left && (!right || beforePunct), right && (!left || afterPunct)
	}
	return n, left, right
}

// removeDelim removes d from the delimiter stack.
func (s *InlineState) removeDelim(d *Delimiter) {
	if d.prev != nil {
		d.prev.next = d.next
	}
	if d.next == nil {
		s.delimStack = d.prev
	} else {
		d.next.prev = d.prev
	}
}

// processEmphasis matches the openers and closers above bottom in the
// delimiter stack, makes spans of them, and removes them from the stack.
func (s *InlineState) processEmphasis(bottom *Delimiter) {
	// bottoms bounds the search for an opener for closers by their class,
	// whether they can open, and their length modulo 3.
	for i := range s.bottoms {
		for j := range s.bottoms[i] {
			for k := range s.bottoms[i][j] {
				s.bottoms[i][j][k] = bottom
			}
		}
	}

	closer := s.delimStack
	for closer != nil && closer.prev != bottom {
		closer = closer.prev
	}
	for closer != nil {
		if !closer.CanClose {
			closer = closer.next
			continue
		}
		oi := 0
		if closer.CanOpen {
			oi = 1
		}
		limit := &s.bottoms[closer.class][oi][closer.OriginalLength%3]

		opener := closer.prev
		found := false
		for opener != nil && opener != bottom && opener != *limit {
			odd := (closer.CanOpen || opener.CanClose) && closer.OriginalLength%3 != 0 &&
				(opener.OriginalLength+closer.OriginalLength)%3 == 0
			if opener.CanOpen && !odd && opener.Processor.CanOpenCloser(opener, closer) {
				found = true
				break
			}
			opener = opener.prev
		}

		if !found {
			*limit = closer.prev
			next := closer.next
			if !closer.CanOpen {
				s.removeDelim(closer)
			}
			closer = next
			continue
		}

		use := 1
		if closer.Length >= 2 && opener.Length >= 2 {
			use = 2
		}
		on, cn := opener.node, closer.node
		opener.Length -= use
		closer.Length -= use
		on.Value = on.Value[:len(on.Value)-use]
		cn.Value = cn.Value[:len(cn.Value)-use]

		span := opener.Processor.OnMatch(use)
		for t := on.NextSibling(); t != nil && t != markdown.Node(cn); {
			next := t.NextSibling()
			span.AppendChild(span, t)
			t = next
		}
		parent := on.Parent()
		parent.InsertAfter(parent, on, span)

		// Remove the delimiters between the opener and the closer.
		opener.next, closer.prev = closer, opener

		if opener.Length == 0 {
			parent.RemoveChild(parent, on)
			s.removeDelim(opener)
		}
		if closer.Length == 0 {
			if p := cn.Parent(); p != nil {
				p.RemoveChild(p, cn)
			}
			next := closer.next
			s.removeDelim(closer)
			closer = next
		}
	}

	for s.delimStack != nil && s.delimStack != bottom {
		s.removeDelim(s.delimStack)
	}
}

// isUnicodeSpace reports whether r is Unicode whitespace, as CommonMark
// defines it.
func isUnicodeSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\f', '\r':
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

// isPunct reports whether r is a Unicode punctuation character, as
// CommonMark defines it, which includes symbols.
func isPunct(r rune) bool {
	if r < utf8.RuneSelf {
		return isASCIIPunct(byte(r))
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// isASCIIPunct reports whether c is an ASCII punctuation character.
func isASCIIPunct(c byte) bool {
	return '!' <= c && c <= '/' || ':' <= c && c <= '@' || '[' <= c && c <= '`' || '{' <= c && c <= '~'
}
