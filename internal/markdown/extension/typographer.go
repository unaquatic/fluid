package extension

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// A TypographicPunctuation is a kind of punctuation that the typographer
// writes as typographers do, such as a quotation mark that opens a quote.
type TypographicPunctuation int

// Kinds of typographic punctuation.
const (
	LeftSingleQuote  TypographicPunctuation = iota + 1 // ' that opens a quote, as ‘
	RightSingleQuote                                   // ' that closes a quote, as ’
	LeftDoubleQuote                                    // " that opens a quote, as “
	RightDoubleQuote                                   // " that closes a quote, as ”
	EnDash                                             // --, as –
	EmDash                                             // ---, as —
	Ellipsis                                           // ..., as …
	LeftAngleQuote                                     // <<, as «
	RightAngleQuote                                    // >>, as »
	Apostrophe                                         // ', as ’

	typographicPunctuationEnd
)

// TypographicSubstitutions maps kinds of punctuation to the text that
// replaces them.
type TypographicSubstitutions map[TypographicPunctuation]string

// defaultSubstitutions returns the text that replaces each kind of
// punctuation by default.
func defaultSubstitutions() TypographicSubstitutions {
	return TypographicSubstitutions{
		LeftSingleQuote:  "‘",
		RightSingleQuote: "’",
		LeftDoubleQuote:  "“",
		RightDoubleQuote: "”",
		EnDash:           "–",
		EmDash:           "—",
		Ellipsis:         "…",
		LeftAngleQuote:   "«",
		RightAngleQuote:  "»",
		Apostrophe:       "’",
	}
}

// A TypographerConfig holds the text that replaces each kind of
// punctuation, which is text, not markup. A kind that it replaces with ""
// stays as written.
type TypographerConfig struct {
	Substitutions TypographicSubstitutions
}

// A TypographerOption configures the typographer.
type TypographerOption func(c *TypographerConfig)

// WithTypographicSubstitutions sets the text that replaces the kinds of
// punctuation in subs, in place of their defaults; "" keeps a kind as
// written.
func WithTypographicSubstitutions(subs TypographicSubstitutions) TypographerOption {
	return func(c *TypographerConfig) {
		for k, v := range subs {
			c.Substitutions[k] = v
		}
	}
}

// typographer is an extension of typographic punctuation.
type typographer struct {
	options []TypographerOption
}

// NewTypographer returns an extension that writes punctuation as
// typographers do, configured by opts: the parser that
// [NewTypographerParser] returns, with priority 9999, after the other
// inline parsers.
func NewTypographer(opts ...TypographerOption) parser.Extension {
	return &typographer{options: opts}
}

// ExtendParser implements [parser.Extension].
func (e *typographer) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithInlineParser(NewTypographerParser(e.options...), 9999))
}

// typographerParser parses typographic punctuation.
type typographerParser struct {
	subs [typographicPunctuationEnd]string
}

// NewTypographerParser returns the parser of typographic punctuation,
// configured by opts, as goldmark's typographer parses it: "--", "---",
// "...", "<<", and ">>", and quotation marks, which open or close quotes,
// as the characters around them decide, or are apostrophes, as in "don't"
// and "'90s". A double quotation mark after a digit and before another is
// an inch mark, as in "Monitor 21"", which stays as written. The text of
// the punctuation that the parser replaces is the replacement's.
//
// Each inline content, such as a paragraph, a heading, or a table cell,
// counts the quotes that it opens and has not closed, which a quotation
// mark that could open or close a quote closes. goldmark means to count
// them so too, but never resets its counts, and so counts them across the
// document.
func NewTypographerParser(opts ...TypographerOption) parser.InlineParser {
	c := TypographerConfig{Substitutions: defaultSubstitutions()}
	for _, opt := range opts {
		opt(&c)
	}
	p := &typographerParser{}
	for k, v := range c.Substitutions {
		if 0 < k && k < typographicPunctuationEnd {
			p.subs[k] = v
		}
	}
	return p
}

// Trigger implements [parser.InlineParser].
func (*typographerParser) Trigger() []byte { return []byte{'\'', '"', '-', '.', '<', '>'} }

// typographerState is the state of the typographer as an
// [parser.InlineState] parses inline content.
type typographerState struct {
	// single and double count the quotes that the content has opened and
	// not closed.
	single, double int

	// lineStart and lineEnd are the range of the line last found, with its
	// line ending, and runStart and runEnd that of the run of quotation
	// marks last found, so that the marks of a long line or run do not each
	// scan it.
	lineStart, lineEnd int
	runStart, runEnd   int
}

// typographerStateOf returns the state of the typographer for the content
// that s parses.
func typographerStateOf(s *parser.InlineState) *typographerState {
	st, _ := s.Value(typographerState{}).(*typographerState)
	if st == nil {
		st = &typographerState{}
		s.SetValue(typographerState{}, st)
	}
	return st
}

// line returns the content that s parses from its position through the end
// of the line, with the line ending, if any.
func (st *typographerState) line(s *parser.InlineState) string {
	src, pos := s.Source(), s.Pos()
	if pos < st.lineStart || pos >= st.lineEnd {
		st.lineStart, st.lineEnd = pos, len(src)
		if i := strings.IndexByte(src[pos:], '\n'); i >= 0 {
			st.lineEnd = pos + i + 1
		}
	}
	return src[pos:st.lineEnd]
}

// runLen returns the length of the run of quotation marks at the position
// of s, the start of line.
func (st *typographerState) runLen(s *parser.InlineState, line string) int {
	pos := s.Pos()
	if pos < st.runStart || pos >= st.runEnd {
		st.runStart, st.runEnd = pos, pos+runLen(line, 0, line[0])
	}
	return st.runEnd - pos
}

// Parse implements [parser.InlineParser].
func (p *typographerParser) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
	st := typographerStateOf(s)
	line := st.line(s)
	c := line[0]
	if len(line) > 2 {
		switch {
		case c == '-' && p.has(EmDash) && line[1] == '-' && line[2] == '-':
			return p.replace(EmDash, 3)
		case c == '.' && p.has(Ellipsis) && line[1] == '.' && line[2] == '.':
			return p.replace(Ellipsis, 3)
		case c == '.':
			return nil, 0
		}
	}
	if len(line) > 1 {
		switch {
		case c == '<' && p.has(LeftAngleQuote) && line[1] == '<':
			return p.replace(LeftAngleQuote, 2)
		case c == '>' && p.has(RightAngleQuote) && line[1] == '>':
			return p.replace(RightAngleQuote, 2)
		case c == '<' || c == '>':
			return nil, 0
		case c == '-' && p.has(EnDash) && line[1] == '-':
			return p.replace(EnDash, 2)
		}
	}
	if c != '\'' && c != '"' {
		return nil, 0
	}

	before := precedingRune(s)
	canOpen, canClose := quoteFlanking(line, st.runLen(s, line), before)
	if c == '\'' {
		return p.single(line, before, canOpen, canClose, st)
	}
	if p.has(LeftDoubleQuote) && canOpen && !canClose {
		st.double++
		return p.replace(LeftDoubleQuote, 1)
	}
	if p.has(RightDoubleQuote) && st.double > 0 && closesQuote(line, canOpen, canClose) {
		if len(line) > 1 && line[1] == '"' && unicode.IsDigit(before) {
			return nil, 0 // an inch mark
		}
		st.double--
		return p.replace(RightDoubleQuote, 1)
	}
	return nil, 0
}

// single parses the single quotation mark at the start of line, after
// before, which the flanking of its run lets open or close quotes.
func (p *typographerParser) single(line string, before rune, canOpen, canClose bool, st *typographerState) (ast.Node, int) {
	if p.has(Apostrophe) {
		// A decade, such as '90s.
		if canOpen && !canClose && len(line) > 3 && isDigit(line[1]) && isDigit(line[2]) && line[3] == 's' {
			after := ' '
			if len(line) > 4 {
				after, _ = utf8.DecodeRuneInString(line[4:])
			}
			if isSpaceRune(after) || isPunctRune(after) {
				return p.replace(Apostrophe, 1)
			}
		}
		// 'twas, 'em, 'net, and 'll.
		if len(line) > 1 && (unicode.IsPunct(before) || unicode.IsSpace(before)) && strings.IndexByte("tenl", line[1]) >= 0 {
			return p.replace(Apostrophe, 1)
		}
		// Between a letter or digit and a letter.
		if len(line) > 1 && (unicode.IsDigit(before) || unicode.IsLetter(before)) && unicode.IsLetter(runeAt(line, 1)) {
			return p.replace(Apostrophe, 1)
		}
	}
	if p.has(LeftSingleQuote) && canOpen && !canClose {
		k := LeftSingleQuote
		// Alice's, I'm, Don't, and You'd; I've, I'll, and You're.
		if len(line) > 1 && strings.IndexByte("smtd", line[1]) >= 0 && (len(line) < 3 || endsContraction(line[2])) ||
			len(line) > 2 && (line[1:3] == "ve" || line[1:3] == "ll" || line[1:3] == "re") && (len(line) < 4 || endsContraction(line[3])) {
			k = RightSingleQuote
		}
		if k == LeftSingleQuote {
			st.single++
		}
		return p.replace(k, 1)
	}
	// A plural possessive or a dropped letter, such as Smiths' and doin'.
	if p.has(RightSingleQuote) && len(line) > 2 && !unicode.IsDigit(runeAt(line, 1)) {
		return p.replace(RightSingleQuote, 1)
	}
	if p.has(RightSingleQuote) && st.single > 0 && closesQuote(line, canOpen, canClose) {
		st.single--
		return p.replace(RightSingleQuote, 1)
	}
	return nil, 0
}

// has reports whether p replaces punctuation of kind k.
func (p *typographerParser) has(k TypographicPunctuation) bool { return p.subs[k] != "" }

// replace returns the text that replaces n bytes of punctuation of kind k,
// or nil if p keeps it.
func (p *typographerParser) replace(k TypographicPunctuation, n int) (ast.Node, int) {
	if !p.has(k) {
		return nil, 0
	}
	return ast.NewText(p.subs[k]), n
}

// closesQuote reports whether the quotation mark at the start of line
// closes a quote: if its run can only close one, or can open or close one
// and punctuation, and then the end of the line, punctuation, or a space,
// follows it.
func closesQuote(line string, canOpen, canClose bool) bool {
	if canClose && !canOpen {
		return true
	}
	return canClose && canOpen && len(line) > 1 && unicode.IsPunct(runeAt(line, 1)) &&
		(len(line) == 2 || isASCIIPunct(line[2]) || isLineSpace(line[2]))
}

// endsContraction reports whether c, after the letters of a contraction,
// ends it.
func endsContraction(c byte) bool { return isASCIIPunct(c) || isLineSpace(c) }

// precedingRune returns the character before the position of s, or a line
// ending at the start of the content.
func precedingRune(s *parser.InlineState) rune {
	if s.Pos() == 0 {
		return '\n'
	}
	r, _ := utf8.DecodeLastRuneInString(s.Source()[:s.Pos()])
	return r
}

// quoteFlanking reports whether the run of n quotation marks at the start
// of line, after before, can open and close quotes, as CommonMark decides
// whether a run of "*" can open and close emphasis.
func quoteFlanking(line string, n int, before rune) (canOpen, canClose bool) {
	after := ' '
	if n < len(line) {
		after, _ = utf8.DecodeRuneInString(line[n:])
	}
	beforeSpace, beforePunct := isSpaceRune(before), isPunctRune(before)
	afterSpace, afterPunct := isSpaceRune(after), isPunctRune(after)
	canOpen = !afterSpace && (!afterPunct || beforeSpace || beforePunct)
	canClose = !beforeSpace && (!beforePunct || afterSpace || afterPunct)
	return canOpen, canClose
}

// runeAt returns the character at offset i of s.
func runeAt(s string, i int) rune {
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
}

// isLineSpace reports whether c is a space, tab, or line ending.
func isLineSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// isSpaceRune reports whether r is a space, tab, line ending, or Unicode
// whitespace.
func isSpaceRune(r rune) bool { return r < utf8.RuneSelf && isLineSpace(byte(r)) || unicode.IsSpace(r) }

// isPunctRune reports whether r is Unicode punctuation or a symbol.
func isPunctRune(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }
