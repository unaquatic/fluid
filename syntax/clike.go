package syntax

import (
	"slices"
	"strings"
)

// CLike lexes a language whose lexical syntax resembles C's: identifiers
// and keywords, numbers, quoted literals, operators, punctuation, and line
// and block comments.
//
// An identifier is classified by the first rule that holds: its kind in
// Words, unless a member access, such as ".", "?.", or "->", comes right
// before it; the kind a Declarators keyword before it gives; Function when
// it is a macro call or followed by '('; Property when AssignProperties
// and it is followed by '='; Constant when CapsConstants and it is all
// capitals; Type when CapitalTypes and it is capitalized; Property after a
// member access; and Variable otherwise.
type CLike struct {
	// Words maps keywords and predeclared names to their kinds, such as
	// "return" to Keyword and "int" to Type.
	Words map[string]Kind
	// MemberWords are the Words that keep their kind after a member access,
	// such as "await" in Rust's fut.await.
	MemberWords []string
	// FoldCase looks words up in lower case, for a case-insensitive
	// language such as SQL, whose Words and Declarators keys are then in
	// lower case.
	FoldCase bool
	// Declarators map keywords to the kind of the identifier right after
	// them, such as "func" to Function and "class" to Type.
	Declarators map[string]Kind
	// LineComments start comments that run to the end of the line, such as
	// "//".
	LineComments []string
	// Directives start line comments that are compiler directives instead,
	// lexed as Preproc, such as "//go:".
	Directives []string
	// BlockComment holds the delimiters of block comments, such as "/*"
	// and "*/", or two empty strings for a language without them.
	BlockComment [2]string
	// NestedComments reports whether block comments nest, as in Rust.
	NestedComments bool
	// Quotes are the quoted literals of the language. The first that
	// matches opens, so a longer delimiter, such as `"""`, must come before
	// a shorter one it starts with. A Quote without an Open and a Close is
	// ignored.
	Quotes []Quote
	// IdentChars are the bytes besides ASCII letters, digits, and '_' that
	// identifiers may hold, such as "$" in JavaScript.
	IdentChars string
	// IdentSuffixes are bytes an identifier may end with, unless '=' comes
	// next, such as "?" and "!" in Ruby.
	IdentSuffixes string
	// Sigils are bytes that mark the identifier right after them as a
	// Variable, such as "$" in PHP.
	Sigils string
	// Decorators reports whether '@' and a dotted name right after it are a
	// Decorator, as in Java and Python.
	Decorators bool
	// Preproc reports whether '#' and a word after it, first on a line, are
	// a preprocessor directive, as in C.
	Preproc bool
	// CapitalTypes reports whether capitalized identifiers name types, as
	// in Java.
	CapitalTypes bool
	// CapsConstants reports whether all-capitals identifiers, such as
	// MAX_SIZE, name constants.
	CapsConstants bool
	// AssignProperties reports whether an identifier followed by '=' is a
	// Property, as are HCL's attributes.
	AssignProperties bool
	// Regex reports whether '/' starts a regular expression literal where
	// an operand is expected, as in JavaScript.
	Regex bool
	// JSX reports whether '<' starts a JSX element where an operand is
	// expected.
	JSX bool
	// Macros reports whether an identifier followed by '!' is a macro call,
	// lexed as a Function, as in Rust.
	Macros bool
	// Special, if not nil, lexes a construct of the language that CLike
	// does not know, such as a Rust raw string, at the scanner's position.
	// It reports whether it classified any text. A construct that runs past
	// the end of the line opens a frame with Scanner.Delimit or
	// Scanner.Heredoc. CLike calls Special before it tries its own rules.
	Special func(s *Scanner) bool
}

// Quote describes a quoted literal, such as a string.
type Quote struct {
	// Open and Close delimit the literal.
	Open, Close string
	// Prefixes are the words that may come right before Open, compared
	// without regard to case, such as "r" in Python. A Quote with Prefixes
	// opens only after one of them, and one without only after none.
	Prefixes []string
	// Kind is the kind of the literal. The zero Kind, Text, stands for
	// String.
	Kind Kind
	// Escapes reports whether a backslash escapes the character after it.
	Escapes bool
	// Doubled reports whether a doubled Close stands for one Close
	// character, as '' does in SQL.
	Doubled bool
	// Multiline reports whether the literal may run past the end of a line.
	// One that is not ends at the end of its line, unless a backslash
	// escapes the newline.
	Multiline bool
	// Interp, if not empty, opens an expression interpolated in the
	// literal, which runs to the InterpClose that matches it, such as "${"
	// and "}" in JavaScript.
	Interp, InterpClose string
	// Sigil, if not 0, makes it and the identifier right after it an
	// interpolated variable, such as '$' in Kotlin.
	Sigil byte
}

// kind returns the kind of q's text.
func (q *Quote) kind() Kind {
	if q.Kind == Text {
		return String
	}
	return q.Kind
}

// AppendLine implements Lexer.
func (c *CLike) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	var buf [8]Frame
	fs := AppendFrames(Frames(buf[:0]), st)
	for !s.Done() {
		var ok bool
		if fs, ok = s.Continue(fs); ok {
			continue
		}
		if f := fs.Top(); f != nil {
			switch f.Mode {
			case modeComment:
				fs = c.comment(&s, fs)
				continue
			case modeQuote:
				fs = c.quote(&s, fs)
				continue
			case modeJSX:
				fs = c.jsxChildren(&s, fs)
				continue
			case modeJSXTag:
				fs = c.jsxTag(&s, fs)
				continue
			case modeInterp:
				if fs, ok = c.interp(&s, fs); ok {
					continue
				}
			}
		}
		fs = c.code(&s, fs)
		fs = append(fs, s.Open...)
		s.Open = s.Open[:0]
	}
	// A literal that cannot span lines ends with the line, and with it
	// everything interpolated in it.
	if !s.Continued {
		for i, f := range fs {
			if f.Mode == modeQuote && int(f.N) < len(c.Quotes) && !c.Quotes[f.N].Multiline {
				fs = fs[:i]
				break
			}
		}
	}
	return s.Toks, s.Finish(fs)
}

// comment lexes the body of a block comment.
func (c *CLike) comment(s *Scanner, fs Frames) Frames {
	open, close := c.BlockComment[0], c.BlockComment[1]
	f := fs.Top()
	// closeAt is the offset of the next close, or the line's length if
	// none is left; searching for it again only once Pos has passed it
	// keeps a line of many nested opens linear to lex.
	closeAt := -1
	for !s.Done() {
		if closeAt < s.Pos {
			closeAt = len(s.Line)
			if i := strings.Index(s.Rest(), close); i >= 0 {
				closeAt = s.Pos + i
			}
		}
		if c.NestedComments {
			if i := strings.Index(s.Line[s.Pos:closeAt], open); i >= 0 {
				s.Emit(Comment, i+len(open))
				f.N++
				continue
			}
		}
		if closeAt == len(s.Line) {
			s.EmitRest(Comment)
			return fs
		}
		s.EmitTo(Comment, closeAt+len(close))
		if f.N--; f.N == 0 {
			fs = fs[:len(fs)-1]
			return fs
		}
	}
	return fs
}

// quote lexes the body of a quoted literal.
func (c *CLike) quote(s *Scanner, fs Frames) Frames {
	f := fs.Top()
	if int(f.N) >= len(c.Quotes) {
		fs = fs[:len(fs)-1]
		return fs
	}
	q := &c.Quotes[f.N]
	k := q.kind()
	for !s.Done() {
		rest := s.Rest()
		i := 0
		for i < len(rest) {
			b := rest[i]
			if (q.Escapes && b == '\\') || b == q.Close[0] || (q.Interp != "" && b == q.Interp[0]) || (q.Sigil != 0 && b == q.Sigil) {
				break
			}
			i++
		}
		s.Emit(k, i)
		if s.Done() {
			return fs
		}
		// The opener of an interpolation comes before escapes, since Swift's
		// \( starts with a backslash.
		switch {
		case q.Interp != "" && s.HasPrefix(q.Interp):
			if len(q.Interp) == 1 && strings.HasPrefix(s.Rest()[1:], q.Interp) {
				// A doubled one-byte opener, as {{ in a Python f-string,
				// stands for itself.
				s.Emit(k, 2)
				continue
			}
			s.Emit(StringEscape, len(q.Interp))
			fs = append(fs, Frame{Mode: modeInterp, Delim: q.InterpClose})
			return fs
		case q.Escapes && s.Peek(0) == '\\':
			if s.Pos+1 == len(s.Line) {
				s.Continued = true
				s.Emit(StringEscape, 1)
				return fs
			}
			s.Emit(StringEscape, 1+runeLen(s.Line, s.Pos+1))
		case s.HasPrefix(q.Close):
			if q.Doubled && strings.HasPrefix(s.Rest()[len(q.Close):], q.Close) {
				s.Emit(StringEscape, 2*len(q.Close))
				continue
			}
			s.Emit(k, len(q.Close))
			fs = fs[:len(fs)-1]
			return fs
		case q.Sigil != 0 && s.Peek(0) == q.Sigil && IdentStart(s.Line, s.Pos+1, ""):
			s.EmitTo(StringEscape, IdentEnd(s.Line, s.Pos+1, ""))
		default:
			s.Emit(k, 1)
		}
	}
	return fs
}

// interp lexes the brackets of an interpolated expression and reports
// whether it classified any text. The bracket that closes the expression
// pops its frame.
func (c *CLike) interp(s *Scanner, fs Frames) (Frames, bool) {
	f := fs.Top()
	close := f.Delim
	if close == "" {
		close = "}"
	}
	open := byte('{')
	if close == ")" {
		open = '('
	}
	switch b := s.Peek(0); {
	case b == open:
		f.N++
		s.Emit(Punctuation, 1)
		return fs, true
	case strings.HasPrefix(s.Rest(), close) && f.N == 0:
		fs = fs[:len(fs)-1]
		k := Punctuation
		if top := fs.Top(); top != nil && top.Mode == modeQuote {
			k = StringEscape
		}
		s.Emit(k, len(close))
		return fs, true
	case b == close[0]:
		f.N--
		s.Emit(Punctuation, 1)
		return fs, true
	}
	return fs, false
}

// code lexes the token at the scanner's position outside of literals and
// comments.
func (c *CLike) code(s *Scanner, fs Frames) Frames {
	b := s.Peek(0)
	if IsSpace(b) {
		s.Space()
		return fs
	}
	if c.Special != nil && c.Special(s) {
		return fs
	}
	if open := c.BlockComment[0]; open != "" && s.HasPrefix(open) {
		s.Emit(Comment, len(open))
		fs = append(fs, Frame{Mode: modeComment, N: 1})
		return fs
	}
	for _, lc := range c.LineComments {
		if s.HasPrefix(lc) {
			k := Comment
			for _, d := range c.Directives {
				if s.HasPrefix(d) {
					k = Preproc
				}
			}
			s.EmitRest(k)
			return fs
		}
	}
	if c.Preproc && b == '#' && s.AtLineStart() {
		c.preproc(s)
		return fs
	}
	if i := c.matchQuote(s.Line, s.Pos, ""); i >= 0 {
		fs = c.openQuote(s, fs, i, s.Pos)
		return fs
	}
	if IsDigit(b) || (b == '.' && IsDigit(s.Peek(1))) {
		s.EmitTo(Number, NumberEnd(s.Line, s.Pos))
		return fs
	}
	if IdentStart(s.Line, s.Pos, c.IdentChars) {
		fs = c.ident(s, fs)
		return fs
	}
	if strings.IndexByte(c.Sigils, b) >= 0 && IdentStart(s.Line, s.Pos+1, "") {
		s.EmitTo(Variable, IdentEnd(s.Line, s.Pos+1, ""))
		return fs
	}
	if c.Decorators && b == '@' && IdentStart(s.Line, s.Pos+1, "") {
		end := IdentEnd(s.Line, s.Pos+1, "")
		for end+1 < len(s.Line) && s.Line[end] == '.' && IdentStart(s.Line, end+1, "") {
			end = IdentEnd(s.Line, end+1, "")
		}
		s.EmitTo(Decorator, end)
		return fs
	}
	if c.Regex && b == '/' && OperandExpected(s) {
		if end := RegexEnd(s.Line, s.Pos); end > 0 {
			s.EmitTo(Regex, end)
			return fs
		}
	}
	if c.JSX && b == '<' && OperandExpected(s) && jsxStart(s.Line, s.Pos) {
		fs = c.jsxOpen(s, fs)
		return fs
	}
	switch b {
	case '(', ')', '[', ']', '{', '}', ',', ';':
		s.Emit(Punctuation, 1)
	case '.':
		if s.HasPrefix("..") {
			s.Emit(Operator, 2+boolInt(s.Peek(2) == '.' || s.Peek(2) == '='))
		} else {
			s.Emit(Punctuation, 1)
		}
	case ':':
		if s.Peek(1) == '=' {
			s.Emit(Operator, 2)
		} else {
			s.Emit(Punctuation, 1)
		}
	case '+', '-', '*', '/', '%', '=', '&', '|', '^', '!', '<', '>', '~', '?', '@':
		s.Emit(Operator, 1)
	default:
		s.EmitRune(Text)
	}
	return fs
}

// boolInt returns 1 for true and 0 for false.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// preproc lexes a preprocessor directive: '#', its name, and for an
// include, the <file> after it.
func (c *CLike) preproc(s *Scanner) {
	end := SpaceEnd(s.Line, s.Pos+1)
	nameEnd := IdentEnd(s.Line, end, "")
	name := s.Line[end:nameEnd]
	s.EmitTo(Preproc, nameEnd)
	if name == "include" || name == "import" {
		s.Space()
		if s.Peek(0) == '<' {
			if i := strings.IndexByte(s.Rest(), '>'); i > 0 {
				s.Emit(String, i+1)
			}
		}
	}
}

// matchQuote returns the index of the quote that opens at line[at] after
// prefix, or -1 if none does. A quote without both delimiters never opens.
func (c *CLike) matchQuote(line string, at int, prefix string) int {
	rest := line[at:]
	for i := range c.Quotes {
		q := &c.Quotes[i]
		if q.Open == "" || q.Close == "" {
			continue
		}
		if (len(q.Prefixes) > 0) != (prefix != "") || !strings.HasPrefix(rest, q.Open) {
			continue
		}
		if prefix == "" || slices.ContainsFunc(q.Prefixes, func(p string) bool { return strings.EqualFold(p, prefix) }) {
			return i
		}
	}
	return -1
}

// openQuote lexes the text from Pos through the opening delimiter of quote
// i, which starts at offset at, and pushes the quote's frame.
func (c *CLike) openQuote(s *Scanner, fs Frames, i, at int) Frames {
	q := &c.Quotes[i]
	s.EmitTo(q.kind(), at+len(q.Open))
	fs = append(fs, Frame{Mode: modeQuote, N: uint8(i)})
	return fs
}

// lookup returns the kind of word in m, folding its case if c.FoldCase.
func (c *CLike) lookup(m map[string]Kind, word string) (Kind, bool) {
	if !c.FoldCase {
		k, ok := m[word]
		return k, ok
	}
	var buf [32]byte
	if len(word) > len(buf) {
		return 0, false
	}
	for i := range len(word) {
		b := word[i]
		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}
		buf[i] = b
	}
	k, ok := m[string(buf[:len(word)])]
	return k, ok
}

// ident lexes the identifier at the scanner's position, or the quoted
// literal it prefixes.
func (c *CLike) ident(s *Scanner, fs Frames) Frames {
	line := s.Line
	end := c.identEnd(line, s.Pos)
	word := line[s.Pos:end]
	if end < len(line) {
		if i := c.matchQuote(line, end, word); i >= 0 {
			fs = c.openQuote(s, fs, i, end)
			return fs
		}
	}
	// A member's name is not a keyword or predeclared name, as from in
	// Array.from, but for MemberWords.
	member := AfterMember(line, s.Pos)
	var (
		k  Kind
		ok bool
	)
	if !member || slices.Contains(c.MemberWords, word) {
		k, ok = c.lookup(c.Words, word)
	}
	if !ok {
		k, end = c.classify(s, word, end, member)
	}
	s.EmitTo(k, end)
	if k != Keyword || c.Declarators == nil {
		return fs
	}
	// The identifier after a declarator is the name it declares.
	decl, ok := c.lookup(c.Declarators, word)
	if !ok {
		return fs
	}
	start := SpaceEnd(line, s.Pos)
	if start == s.Pos || !IdentStart(line, start, c.IdentChars) {
		return fs
	}
	nameEnd := c.identEnd(line, start)
	if _, isWord := c.lookup(c.Words, line[start:nameEnd]); isWord {
		return fs
	}
	// A name before a dot qualifies the declared one, as the receiver
	// type of Kotlin's fun String.ext() does.
	if nameEnd < len(line) && line[nameEnd] == '.' {
		return fs
	}
	s.EmitTo(Text, start)
	s.EmitTo(decl, nameEnd)
	return fs
}

// identEnd returns the offset in line of the end of the identifier at i,
// with one of IdentSuffixes, unless '=' comes after it.
func (c *CLike) identEnd(line string, i int) int {
	end := IdentEnd(line, i, c.IdentChars)
	if end < len(line) && strings.IndexByte(c.IdentSuffixes, line[end]) >= 0 && (end+1 >= len(line) || line[end+1] != '=') {
		end++
	}
	return end
}

// classify returns the kind of word, an identifier that ends at end and is
// not in Words, and the end of its token, which takes a macro's '!'.
// member reports whether a member access comes right before word.
func (c *CLike) classify(s *Scanner, word string, end int, member bool) (Kind, int) {
	line := s.Line
	next := NextNonSpace(line, end)
	switch {
	case c.Macros && end+1 < len(line) && line[end] == '!' && line[end+1] != '=':
		return Function, end + 1
	case next == '(':
		return Function, end
	case c.AssignProperties && next == '=' && !assignedOperator(line, end):
		return Property, end
	case c.CapsConstants && IsCaps(word):
		return Constant, end
	case c.CapitalTypes && IsCapitalized(word):
		return Type, end
	case member:
		return Property, end
	}
	return Variable, end
}

// AfterMember reports whether a member access ends right before offset i
// in line: a '.', as in a.b or a?.b, but not the dots of a range or a
// spread, such as a..b or ...b, or a "->", as in a->b.
func AfterMember(line string, i int) bool {
	switch {
	case i >= 2 && line[i-2:i] == "->":
		return true
	case i >= 1 && line[i-1] == '.':
		return i < 2 || line[i-2] != '.'
	}
	return false
}

// assignedOperator reports whether the '=' after the identifier that ends
// at end in line starts a longer operator, such as == or =>.
func assignedOperator(line string, end int) bool {
	i := SpaceEnd(line, end) + 1
	return i < len(line) && (line[i] == '=' || line[i] == '>' || line[i] == '~')
}

// OperandExpected reports whether an operand, rather than an operator, may
// come at the scanner's position, judged by the token before it: as at
// the start of a line, or after an operator, an opening bracket, or a
// keyword.
func OperandExpected(s *Scanner) bool {
	p, ok := s.Prev()
	if !ok {
		return true
	}
	switch p.Kind {
	case Operator, Keyword, Comment:
		return true
	case Punctuation:
		last := p.Text[len(p.Text)-1]
		return last != ')' && last != ']' && last != '}'
	}
	return false
}

// maxRegex bounds the length of a regular expression literal, so that a
// long line of slashes that close none stays linear to lex.
const maxRegex = 1024

// RegexEnd returns the offset in line just past the regular expression
// literal at i, which starts with '/', and its flags, or -1 if the line
// does not close it within maxRegex bytes.
func RegexEnd(line string, i int) int {
	inClass := false
	for j := i + 1; j < min(len(line), i+maxRegex); j++ {
		switch line[j] {
		case '\\':
			j++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				if j == i+1 {
					return -1
				}
				return IdentEnd(line, j+1, "")
			}
		}
	}
	return -1
}

// jsxStart reports whether a JSX element opens at the '<' at line[i]: a
// fragment, <>, or a tag name, but not the type parameters of a generic
// arrow function, such as <T,> or <T extends U>.
func jsxStart(line string, i int) bool {
	if i+1 < len(line) && line[i+1] == '>' {
		return true
	}
	if !IdentStart(line, i+1, "") {
		return false
	}
	j := SpaceEnd(line, IdentEnd(line, i+1, ".-:"))
	return j >= len(line) || (line[j] != ',' && !strings.HasPrefix(line[j:], "extends "))
}

// jsxOpen lexes '<' and the name of a JSX tag, or the '<>' of a fragment,
// and pushes the frame that lexes what follows.
func (c *CLike) jsxOpen(s *Scanner, fs Frames) Frames {
	if s.Peek(1) == '>' {
		s.Emit(Punctuation, 2)
		fs = c.enterChildren(fs)
		return fs
	}
	s.Emit(Punctuation, 1)
	s.EmitTo(Tag, IdentEnd(s.Line, s.Pos, ".-:"))
	fs = append(fs, Frame{Mode: modeJSXTag})
	return fs
}

// enterChildren opens the children of a JSX element whose tag has closed:
// one level deeper in the children frame on top of fs, or a new one.
func (c *CLike) enterChildren(fs Frames) Frames {
	if f := fs.Top(); f != nil && f.Mode == modeJSX {
		f.N++
		return fs
	}
	fs = append(fs, Frame{Mode: modeJSX, N: 1})
	return fs
}

// jsxTag lexes the attributes of a JSX tag through the '>' that closes it.
func (c *CLike) jsxTag(s *Scanner, fs Frames) Frames {
	for !s.Done() {
		switch b := s.Peek(0); {
		case IsSpace(b):
			s.Space()
		case b == '/' && s.Peek(1) == '>':
			s.Emit(Punctuation, 2)
			fs = fs[:len(fs)-1]
			return fs
		case b == '>':
			s.Emit(Punctuation, 1)
			fs = fs[:len(fs)-1]
			fs = c.enterChildren(fs)
			return fs
		case b == '{':
			s.Emit(Punctuation, 1)
			fs = append(fs, Frame{Mode: modeInterp, Delim: "}"})
			return fs
		case b == '"' || b == '\'':
			if i := strings.IndexByte(s.Rest()[1:], b); i >= 0 {
				s.Emit(String, i+2)
			} else {
				s.EmitRest(String)
			}
		case b == '=':
			s.Emit(Operator, 1)
		case IdentStart(s.Line, s.Pos, ""):
			s.EmitTo(Attribute, IdentEnd(s.Line, s.Pos, "-:"))
		default:
			s.EmitRune(Text)
		}
	}
	return fs
}

// jsxChildren lexes the text, nested elements, and interpolated
// expressions among the children of a JSX element.
func (c *CLike) jsxChildren(s *Scanner, fs Frames) Frames {
	f := fs.Top()
	for !s.Done() {
		switch b := s.Peek(0); {
		case b == '{':
			s.Emit(Punctuation, 1)
			fs = append(fs, Frame{Mode: modeInterp, Delim: "}"})
			return fs
		case b == '<' && s.Peek(1) == '/':
			end := strings.IndexByte(s.Rest(), '>')
			if end < 0 {
				s.EmitRest(Tag)
				return fs
			}
			s.Emit(Punctuation, 2)
			nameEnd := IdentEnd(s.Line, SpaceEnd(s.Line, s.Pos), ".-:")
			s.EmitTo(Tag, nameEnd)
			s.EmitTo(Punctuation, s.Pos+strings.IndexByte(s.Rest(), '>')+1)
			if f.N--; f.N == 0 {
				fs = fs[:len(fs)-1]
				return fs
			}
		case b == '<' && (IdentStart(s.Line, s.Pos+1, "") || s.Peek(1) == '>'):
			fs = c.jsxOpen(s, fs)
			return fs
		default:
			i := strings.IndexAny(s.Rest(), "{<")
			if i < 0 {
				s.EmitRest(Text)
				return fs
			}
			s.Emit(Text, max(i, 1))
		}
	}
	return fs
}
