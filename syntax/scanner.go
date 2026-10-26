package syntax

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// The frame modes the lexers of this package share. Each lexer reads only
// the modes it pushes.
const (
	// modeComment is a block comment; N is its nesting depth.
	modeComment uint8 = iota + 1
	// modeQuote is a quoted literal; N indexes the lexer's quotes.
	modeQuote
	// modeInterp is an expression interpolated in a literal or markup; N is
	// the depth of the brackets open in it, and Delim the text that closes
	// it.
	modeInterp
	// modeDelimited is a literal or comment that Delim ends; N is its Kind.
	modeDelimited
	// modeHeredoc is a heredoc, which a line that is Delim ends; N holds
	// its Kind and heredoc flags.
	modeHeredoc
	// modeJSX is the children of a JSX element; N is the depth of the
	// elements open.
	modeJSX
	// modeJSXTag is the attributes of a JSX tag.
	modeJSXTag
	// modeEmbed is a region in an embedded language, whose State lies below
	// the frame; N identifies the language and Delim holds what ends it.
	modeEmbed
	// modeTag is the attributes of a markup tag; Delim is its name.
	modeTag
	// modeBlock is an indented block, as of a YAML block scalar; N is the
	// indentation of the line that opened it.
	modeBlock
	// modeBody marks a line, or the rest of one, that continues a command
	// past its name, as after a shell's escaped newline; for the shell, N
	// reports whether a command may come next.
	modeBody
)

// Heredoc flags, in the bits of a modeHeredoc frame's N above its Kind.
const (
	// heredocIndented lets the closing word follow spaces and tabs.
	heredocIndented uint8 = 1 << 5
	// heredocPrefix closes the heredoc at a line that starts with the word.
	heredocPrefix uint8 = 1 << 6
	// heredocKindMask selects the Kind.
	heredocKindMask uint8 = heredocIndented - 1
)

// Scanner walks a line, appending tokens as a Lexer classifies its text.
type Scanner struct {
	// Line is the line being lexed.
	Line string
	// Pos is the byte offset in Line of the next text to classify.
	Pos int
	// Toks holds the tokens of Line[:Pos], after those of the slice the
	// lexer was given.
	Toks []Token
	// Start is the index in Toks of the first token of Line.
	Start int
	// Open holds the frames of constructs that Delimit has left open at the
	// end of the line, innermost last, for the lexer to push.
	Open Frames
	// Deferred holds the frames of heredocs that Heredoc has opened, whose
	// bodies start on the next line, in the order they open.
	Deferred Frames
	// Continued reports that the line ends with a backslash that escapes
	// the newline.
	Continued bool
}

// NewScanner returns a Scanner that appends the tokens of line to dst.
func NewScanner(dst []Token, line string) Scanner {
	return Scanner{Line: line, Toks: dst, Start: len(dst)}
}

// Emit classifies the next n bytes as k. A token of the kind of the token
// before it extends that token.
func (s *Scanner) Emit(k Kind, n int) {
	end := min(s.Pos+max(n, 0), len(s.Line))
	if end == s.Pos {
		return
	}
	if i := len(s.Toks) - 1; i >= s.Start && s.Toks[i].Kind == k {
		t := &s.Toks[i]
		t.Text = s.Line[s.Pos-len(t.Text) : end]
	} else {
		s.Toks = append(s.Toks, Token{Kind: k, Text: s.Line[s.Pos:end]})
	}
	s.Pos = end
}

// EmitTo classifies the text up to byte offset end as k.
func (s *Scanner) EmitTo(k Kind, end int) { s.Emit(k, end-s.Pos) }

// EmitRest classifies the rest of the line as k.
func (s *Scanner) EmitRest(k Kind) { s.EmitTo(k, len(s.Line)) }

// EmitRune classifies the next character as k.
func (s *Scanner) EmitRune(k Kind) { s.Emit(k, runeLen(s.Line, s.Pos)) }

// runeLen returns the length in bytes of the character at s[i], or 0 at
// the end of s.
func runeLen(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	if s[i] < utf8.RuneSelf {
		return 1
	}
	_, n := utf8.DecodeRuneInString(s[i:])
	return n
}

// Done reports whether no text is left to classify.
func (s *Scanner) Done() bool { return s.Pos >= len(s.Line) }

// Rest returns the text left to classify.
func (s *Scanner) Rest() string { return s.Line[s.Pos:] }

// Peek returns the byte i bytes after Pos, or 0 past the end of the line.
func (s *Scanner) Peek(i int) byte {
	if j := s.Pos + i; j >= 0 && j < len(s.Line) {
		return s.Line[j]
	}
	return 0
}

// HasPrefix reports whether the text left to classify starts with p.
func (s *Scanner) HasPrefix(p string) bool { return strings.HasPrefix(s.Line[s.Pos:], p) }

// Space classifies the spaces at Pos as Text and reports whether there
// were any.
func (s *Scanner) Space() bool {
	end := SpaceEnd(s.Line, s.Pos)
	s.EmitTo(Text, end)
	return end > s.Pos
}

// AtLineStart reports whether only spaces come before Pos.
func (s *Scanner) AtLineStart() bool {
	return strings.TrimLeft(s.Line[:s.Pos], " \t") == ""
}

// Prev returns the last token of the line that is not all spaces, and
// false if there is none.
func (s *Scanner) Prev() (Token, bool) {
	for i := len(s.Toks) - 1; i >= s.Start; i-- {
		if t := s.Toks[i]; t.Kind != Text || strings.TrimSpace(t.Text) != "" {
			return t, true
		}
	}
	return Token{}, false
}

// Delimit classifies the text through the next close as k. Without a close
// on the line, it classifies the rest of the line and adds a frame to Open
// that continues the construct on the next line. It reports whether the
// construct closed.
func (s *Scanner) Delimit(k Kind, close string) bool {
	if i := strings.Index(s.Rest(), close); i >= 0 {
		s.Emit(k, i+len(close))
		return true
	}
	s.EmitRest(k)
	s.Open = append(s.Open, Frame{Mode: modeDelimited, N: uint8(k), Delim: close})
	return false
}

// Heredoc adds a frame to Deferred for a heredoc whose body starts on the
// next line and is classified as k, up to and including a line that is
// word. If indented, the word may follow spaces and tabs; if prefix, the
// line need only start with it.
func (s *Scanner) Heredoc(k Kind, word string, indented, prefix bool) {
	n := uint8(k) & heredocKindMask
	if indented {
		n |= heredocIndented
	}
	if prefix {
		n |= heredocPrefix
	}
	s.Deferred = append(s.Deferred, Frame{Mode: modeHeredoc, N: n, Delim: word})
}

// HeredocLine classifies the whole line, the body of the heredoc f, and
// reports whether it is the line that closes the heredoc.
func (s *Scanner) HeredocLine(f Frame) bool {
	text := s.Line
	if f.N&heredocIndented != 0 {
		text = strings.TrimLeft(text, " \t")
	}
	var closed bool
	if f.N&heredocPrefix != 0 {
		closed = strings.HasPrefix(text, f.Delim)
	} else {
		closed = strings.TrimRight(text, " \t\r") == f.Delim
	}
	s.EmitRest(Kind(f.N & heredocKindMask))
	return closed
}

// Continue lexes the line in the construct on top of fs, if it is one of
// the shared modes Delimit and Heredoc open, and reports whether it was.
// It returns fs without the construct's frame when the construct closes.
func (s *Scanner) Continue(fs Frames) (Frames, bool) {
	f := fs.Top()
	if f == nil {
		return fs, false
	}
	switch f.Mode {
	case modeDelimited:
		if i := strings.Index(s.Rest(), f.Delim); i >= 0 {
			s.Emit(Kind(f.N), i+len(f.Delim))
			fs = fs[:len(fs)-1]
		} else {
			s.EmitRest(Kind(f.N))
		}
		return fs, true
	case modeHeredoc:
		if s.Pos == 0 {
			if s.HeredocLine(*f) {
				fs = fs[:len(fs)-1]
			}
			return fs, true
		}
	}
	return fs, false
}

// Finish returns the State for the next line: fs with the frames of Open
// and then those of Deferred on top, the first heredoc innermost so that
// its body comes first.
func (s *Scanner) Finish(fs Frames) State {
	fs = append(fs, s.Open...)
	for i := len(s.Deferred) - 1; i >= 0; i-- {
		fs = append(fs, s.Deferred[i])
	}
	return fs.State()
}

// SpaceEnd returns the offset in s of the first byte at or after i that is
// not a space, tab, carriage return, form feed, or vertical tab.
func SpaceEnd(s string, i int) int {
	for i < len(s) && IsSpace(s[i]) {
		i++
	}
	return i
}

// IsSpace reports whether b is a space, tab, carriage return, form feed, or
// vertical tab.
func IsSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\f' || b == '\v'
}

// IsDigit reports whether b is an ASCII digit.
func IsDigit(b byte) bool { return '0' <= b && b <= '9' }

// IsLetter reports whether b is an ASCII letter.
func IsLetter(b byte) bool { return 'a' <= b|0x20 && b|0x20 <= 'z' }

// IsWordByte reports whether b is an ASCII letter, digit, or underscore.
func IsWordByte(b byte) bool { return IsLetter(b) || IsDigit(b) || b == '_' }

// IdentStart reports whether an identifier starts at s[i]: a letter, an
// underscore, or one of extra.
func IdentStart(s string, i int, extra string) bool {
	if i >= len(s) {
		return false
	}
	b := s[i]
	if b < utf8.RuneSelf {
		return IsLetter(b) || b == '_' || strings.IndexByte(extra, b) >= 0
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return unicode.IsLetter(r)
}

// IdentEnd returns the offset in s of the end of the identifier at i, made
// of letters, digits, underscores, and the bytes of extra.
func IdentEnd(s string, i int, extra string) int {
	for i < len(s) {
		b := s[i]
		if b < utf8.RuneSelf {
			if !IsWordByte(b) && strings.IndexByte(extra, b) < 0 {
				break
			}
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.Is(unicode.Mn, r) {
			break
		}
		i += n
	}
	return i
}

// NumberEnd returns the offset in s of the end of the number at i: a
// decimal, hexadecimal, octal, or binary integer or float, with
// underscores between digits and a suffix of letters and digits, such as
// 0x1F, 1_000, 2.5e-3, 10u8, or 3.0f.
func NumberEnd(s string, i int) int {
	j := numberBody(s, i)
	for j < len(s) && IsWordByte(s[j]) {
		j++
	}
	return j
}

// numberBody returns the offset in s of the end of the number at i, as
// NumberEnd does, but before any suffix.
func numberBody(s string, i int) int {
	j := i
	digits, exp := IsDigit, byte('e')
	if j+1 < len(s) && s[j] == '0' {
		switch s[j+1] | 0x20 {
		case 'x':
			digits, exp = isHexDigit, 'p'
			j += 2
		case 'b', 'o':
			j += 2
		}
	}
	for j < len(s) && (digits(s[j]) || s[j] == '_') {
		j++
	}
	// A fraction needs a digit after the point, so that 1..2 and 1.max()
	// keep their dots.
	if j+1 < len(s) && s[j] == '.' && digits(s[j+1]) {
		j++
		for j < len(s) && (digits(s[j]) || s[j] == '_') {
			j++
		}
	}
	// An exponent: e, or p for hexadecimal floats, then an optional sign.
	if j < len(s) {
		if s[j]|0x20 == exp {
			k := j + 1
			if k < len(s) && (s[k] == '+' || s[k] == '-') {
				k++
			}
			if k < len(s) && IsDigit(s[k]) {
				j = k
				for j < len(s) && (IsDigit(s[j]) || s[j] == '_') {
					j++
				}
			}
		}
	}
	return j
}

// isHexDigit reports whether b is a hexadecimal digit.
func isHexDigit(b byte) bool { return IsDigit(b) || 'a' <= b|0x20 && b|0x20 <= 'f' }

// NextByte returns the offset in line of the first c at or after from, or
// len(line) if there is none. It keeps the offset in *next, which must
// start below zero, and reuses it while from has not passed it, so that
// searching from each of many offsets that move forward stays linear.
func NextByte(line string, from int, c byte, next *int) int {
	if *next >= from {
		return *next
	}
	if i := strings.IndexByte(line[from:], c); i >= 0 {
		*next = from + i
	} else {
		*next = len(line)
	}
	return *next
}

// IsCaps reports whether w is an all-capitals identifier of two or more
// bytes, such as MAX_SIZE.
func IsCaps(w string) bool {
	letters := 0
	for i := range len(w) {
		switch b := w[i]; {
		case 'A' <= b && b <= 'Z':
			letters++
		case IsDigit(b) || b == '_':
		default:
			return false
		}
	}
	return letters > 0 && len(w) > 1
}

// IsCapitalized reports whether w starts with an upper-case letter.
func IsCapitalized(w string) bool {
	r, _ := utf8.DecodeRuneInString(w)
	return unicode.IsUpper(r)
}

// NextNonSpace returns the byte at the first offset at or after i in s
// that is not a space, or 0 if there is none.
func NextNonSpace(s string, i int) byte {
	if i = SpaceEnd(s, i); i < len(s) {
		return s[i]
	}
	return 0
}

// words adds each space-separated word of list to m as k.
func words(m map[string]Kind, k Kind, list string) {
	for w := range strings.FieldsSeq(list) {
		m[w] = k
	}
}
