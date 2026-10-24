package syntax

import "strings"

// Markdown lexes Markdown: headings, quotes, list markers, emphasis, code
// spans, links, and HTML comments, and fenced code blocks in the language
// their info string names.
type Markdown struct {
	// Lookup finds the language of a fenced code block by the first word of
	// its info string, such as "go". Nil finds it with the package's
	// Lookup. Fenced code in a language it does not find is Text.
	Lookup func(name string) *Language
}

// markdownLang returns Markdown.
func markdownLang() *Language {
	return &Language{
		Name:    "markdown",
		Aliases: []string{"md", "mkd", "mdx", "markdn", "mdown", "rmd"},
		Lexer:   &Markdown{},
	}
}

// fenceTilde marks, in the N of a fence frame, a fence of tildes rather
// than backticks; the bits below it hold the fence's length.
const fenceTilde = 1 << 7

// AppendLine implements Lexer.
func (m *Markdown) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	if f, below, ok := st.Top(); ok {
		switch f.Mode {
		case modeEmbed:
			return m.fenced(s, f, below, st)
		case modeDelimited:
			if fs, _ := s.Continue(Frames{f}); len(fs) > 0 {
				return s.Toks, st
			}
			m.inline(&s)
			return s.Toks, s.Finish(nil)
		}
	}
	indent := SpaceEnd(line, 0)
	rest := line[indent:]
	if indent <= 3 {
		if f, ok := fenceOpen(rest); ok {
			s.EmitRest(String)
			return s.Toks, State("").Push(f)
		}
		if n := len(rest) - len(strings.TrimLeft(rest, "#")); n >= 1 && n <= 6 && (n == len(rest) || IsSpace(rest[n])) {
			k := Subheading
			if n == 1 {
				k = Heading
			}
			s.EmitRest(k)
			return s.Toks, ""
		}
	}
	switch {
	case strings.HasPrefix(rest, ">"):
		s.EmitTo(Keyword, SpaceEnd(line, indent+1))
		s.EmitRest(Emph)
		return s.Toks, ""
	case listMarkerEnd(rest) > 0:
		s.EmitTo(Text, indent)
		s.Emit(Keyword, listMarkerEnd(rest))
		if r := s.Rest(); len(r) >= 3 && r[0] == '[' && r[2] == ']' && strings.IndexByte(" xX", r[1]) >= 0 {
			s.Emit(Keyword, 3)
		}
	}
	m.inline(&s)
	return s.Toks, s.Finish(nil)
}

// fenceOpen reports whether line, without its indentation, opens a fenced
// code block, and returns the block's frame.
func fenceOpen(line string) (Frame, bool) {
	if !strings.HasPrefix(line, "```") && !strings.HasPrefix(line, "~~~") {
		return Frame{}, false
	}
	c := line[0]
	n := len(line) - len(strings.TrimLeft(line, line[:1]))
	info := strings.TrimSpace(line[n:])
	if c == '`' && strings.IndexByte(info, '`') >= 0 {
		return Frame{}, false
	}
	lang, _, _ := strings.Cut(info, " ")
	lang = strings.Trim(lang, "{}.")
	f := Frame{Mode: modeEmbed, N: uint8(min(n, fenceTilde-1)), Delim: lang}
	if c == '~' {
		f.N |= fenceTilde
	}
	return f, true
}

// fenced lexes a line of the fenced code block f, whose language's State
// is below, as its language does, unless it closes the block. st is the
// State the line starts in.
func (m *Markdown) fenced(s Scanner, f Frame, below, st State) ([]Token, State) {
	c, n := byte('`'), int(f.N&^fenceTilde)
	if f.N&fenceTilde != 0 {
		c = '~'
	}
	if rest := strings.TrimLeft(s.Line, " "); len(s.Line)-len(rest) <= 3 {
		if run := len(rest) - len(strings.TrimLeft(rest, string(c))); run >= n && strings.TrimSpace(rest[run:]) == "" {
			s.EmitRest(String)
			return s.Toks, ""
		}
	}
	lookup := m.Lookup
	if lookup == nil {
		lookup = Lookup
	}
	var lang *Language
	if f.Delim != "" {
		lang = lookup(f.Delim)
	}
	if lang == nil {
		s.EmitRest(Text)
		return s.Toks, st
	}
	toks, next := lang.AppendLine(s.Toks, s.Line, below)
	if next == below {
		return toks, st
	}
	return toks, next.Push(f)
}

// listMarkerEnd returns the length of the list marker, such as "- " or
// "1. ", that line starts with, or 0 if it starts with none.
func listMarkerEnd(line string) int {
	if len(line) >= 2 && strings.IndexByte("-*+", line[0]) >= 0 && IsSpace(line[1]) {
		return 2
	}
	i := 0
	for i < len(line) && i < 9 && IsDigit(line[i]) {
		i++
	}
	if i > 0 && i+1 < len(line) && (line[i] == '.' || line[i] == ')') && IsSpace(line[i+1]) {
		return i + 2
	}
	return 0
}

// inline lexes the rest of a line's inline Markdown.
func (m *Markdown) inline(s *Scanner) {
	closers := linkClosers{Bracket: -1, Paren: -1}
	for !s.Done() {
		rest := s.Rest()
		switch b := rest[0]; {
		case b == '\\' && len(rest) > 1:
			s.Emit(Text, 1+runeLen(rest, 1))
		case b == '`':
			n := len(rest) - len(strings.TrimLeft(rest, "`"))
			if end := codeSpanEnd(rest, n); end > 0 {
				s.Emit(String, end)
			} else {
				s.Emit(Text, n)
			}
		case b == '*' || b == '_' || b == '~':
			if end, k := emphasisEnd(s); end > 0 {
				s.Emit(k, end)
			} else {
				s.Emit(Text, len(rest)-len(strings.TrimLeft(rest, rest[:1])))
			}
		case b == '[' || (b == '!' && strings.HasPrefix(rest, "![")):
			if !m.link(s, &closers) {
				s.Emit(Text, 1)
			}
		case strings.HasPrefix(rest, "<!--"):
			s.Emit(Comment, 4)
			s.Delimit(Comment, "-->")
		default:
			i := strings.IndexAny(rest[1:], "\\`*_~[!<")
			if i < 0 {
				s.EmitRest(Text)
				return
			}
			s.Emit(Text, i+1)
		}
	}
}

// codeSpanEnd returns the length of the code span at the start of s, which
// opens with n backticks, or 0 if the line does not close it.
func codeSpanEnd(s string, n int) int {
	for i := n; i < len(s); {
		j := strings.IndexByte(s[i:], '`')
		if j < 0 {
			return 0
		}
		i += j
		run := len(s[i:]) - len(strings.TrimLeft(s[i:], "`"))
		if run == n {
			return i + n
		}
		i += run
	}
	return 0
}

// emphasisEnd returns the length and kind of the emphasis, strong
// emphasis, or strikethrough at the scanner's position, or 0 if none is
// there.
func emphasisEnd(s *Scanner) (int, Kind) {
	rest := s.Rest()
	c := rest[0]
	n := min(len(rest)-len(strings.TrimLeft(rest, rest[:1])), 2)
	if c == '~' && n < 2 {
		return 0, Text
	}
	// An opener needs text after it, and an underscore one that is not
	// inside a word.
	if n >= len(rest) || IsSpace(rest[n]) || (c == '_' && s.Pos > 0 && IsWordByte(s.Line[s.Pos-1])) {
		return 0, Text
	}
	delim := rest[:n]
	i := strings.Index(rest[n:], delim)
	if i <= 0 || IsSpace(rest[n+i-1]) {
		return 0, Text
	}
	switch {
	case c == '~':
		return n + i + n, Deleted
	case n == 2:
		return n + i + n, Strong
	}
	return n + i + n, Emph
}

// linkClosers holds the offsets in a line of the next ']' and ')', which
// close links, for NextByte.
type linkClosers struct {
	// Bracket is the offset of the next ']'.
	Bracket int
	// Paren is the offset of the next ')'.
	Paren int
}

// link lexes the link or image at the scanner's position, such as
// [text](url), and reports whether one is there. It finds the brackets
// that close it with the offsets in closers.
func (m *Markdown) link(s *Scanner, closers *linkClosers) bool {
	rest := s.Rest()
	open := 1
	if rest[0] == '!' {
		open = 2
	}
	close := NextByte(s.Line, s.Pos+open, ']', &closers.Bracket) - s.Pos
	if close+1 >= len(rest) || rest[close+1] != '(' {
		return false
	}
	end := NextByte(s.Line, s.Pos+close+2, ')', &closers.Paren) - s.Pos
	if end >= len(rest) {
		return false
	}
	s.Emit(Text, open)
	s.Emit(Tag, close-open)
	s.Emit(Text, 2)
	s.Emit(Attribute, end-close-2)
	s.Emit(Text, 1)
	return true
}
