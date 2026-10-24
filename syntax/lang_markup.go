package syntax

import "strings"

// Markup lexes HTML-like markup: tags, attributes, comments, entities, and
// declarations, and the scripts and styles that HTML elements and the
// template syntaxes of Vue and Svelte embed.
type Markup struct {
	// Script lexes script elements and interpolated expressions; nil leaves
	// them Text.
	Script Lexer
	// TypeScript lexes script elements whose lang is "ts", or Script if
	// nil.
	TypeScript Lexer
	// Style lexes style elements; nil leaves them Text.
	Style Lexer
	// SCSS lexes style elements whose lang is "scss", or Style if nil.
	SCSS Lexer
	// Vue lexes Vue's {{ }} interpolations and the expressions in its
	// directive attributes, such as v-if, :value, and @click.
	Vue bool
	// Svelte lexes Svelte's { } expressions and blocks, such as {#if}.
	Svelte bool
}

// The languages a Markup embeds, as the N of a modeEmbed frame.
const (
	embedScript = iota + 1
	embedTypeScript
	embedStyle
	embedSCSS
)

// tagClosing marks, in the N of a modeTag frame, a closing tag; the bits
// above it hold the embedded language its lang attribute names.
const tagClosing = 1

// embedded returns the lexer of the embedded language id, or nil.
func (m *Markup) embedded(id uint8) Lexer {
	switch id {
	case embedScript:
		return m.Script
	case embedTypeScript:
		if m.TypeScript != nil {
			return m.TypeScript
		}
		return m.Script
	case embedStyle:
		return m.Style
	case embedSCSS:
		if m.SCSS != nil {
			return m.SCSS
		}
		return m.Style
	}
	return nil
}

// AppendLine implements Lexer.
func (m *Markup) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	// Below an embedded region lies the State of its language; below any
	// other frame lie Markup's own, such as the tag around an attribute
	// value that runs past the end of a line.
	var buf [4]Frame
	fs := Frames(buf[:0])
	var inner State
	if f, below, ok := st.Top(); ok && f.Mode == modeEmbed {
		fs, inner = append(fs, f), below
	} else {
		fs = AppendFrames(fs, st)
	}
	for !s.Done() {
		f := fs.Top()
		switch {
		case f == nil:
			fs = m.content(&s, fs)
		case f.Mode == modeDelimited:
			fs, _ = s.Continue(fs)
		case f.Mode == modeTag:
			fs = m.attrs(&s, fs)
		case f.Mode == modeInterp:
			fs = m.interp(&s, fs)
		case f.Mode == modeEmbed:
			fs, inner = m.embed(&s, fs, inner)
		default:
			fs = fs[:0]
		}
		fs = append(fs, s.Open...)
		s.Open = s.Open[:0]
	}
	if f := fs.Top(); f != nil && f.Mode == modeEmbed {
		if next := inner.Push(*f); next != st {
			return s.Toks, next
		}
		return s.Toks, st
	}
	return s.Toks, s.Finish(fs)
}

// content lexes text and the markup that opens at the scanner's position.
func (m *Markup) content(s *Scanner, fs Frames) Frames {
	rest := s.Rest()
	switch b := rest[0]; {
	case strings.HasPrefix(rest, "<!--"):
		s.Emit(Comment, 4)
		s.Delimit(Comment, "-->")
	case strings.HasPrefix(rest, "<![CDATA["):
		s.Emit(Preproc, 9)
		s.Delimit(Preproc, "]]>")
	case strings.HasPrefix(rest, "<!"):
		s.Emit(Preproc, 2)
		s.Delimit(Preproc, ">")
	case strings.HasPrefix(rest, "<?"):
		s.Emit(Preproc, 2)
		s.Delimit(Preproc, "?>")
	case b == '<' && (IdentStart(rest, 1, "") || (len(rest) > 2 && rest[1] == '/' && IdentStart(rest, 2, ""))):
		n := uint8(0)
		if rest[1] == '/' {
			n = tagClosing
		}
		s.Emit(Punctuation, 1+int(n))
		end := IdentEnd(s.Line, s.Pos, "-:.")
		name := strings.ToLower(s.Line[s.Pos:end])
		s.EmitTo(Tag, end)
		if n == 0 {
			switch name {
			case "script":
				n = embedScript << 1
			case "style":
				n = embedStyle << 1
			}
		}
		fs = append(fs, Frame{Mode: modeTag, N: n, Delim: name})
	case b == '&':
		end := 1
		for end < len(rest) && end < 32 && (IsWordByte(rest[end]) || rest[end] == '#') {
			end++
		}
		if end > 1 && end < len(rest) && rest[end] == ';' {
			s.Emit(StringEscape, end+1)
		} else {
			s.Emit(Text, 1)
		}
	case m.Vue && strings.HasPrefix(rest, "{{"):
		s.Emit(Punctuation, 2)
		fs = append(fs, Frame{Mode: modeInterp, Delim: "}}"})
	case m.Svelte && b == '{':
		s.Emit(Punctuation, 1)
		// A block tag, such as {#if}, {:else}, {/if}, or {@html}.
		if len(rest) > 1 && strings.IndexByte("#:/@", rest[1]) >= 0 {
			s.EmitTo(Keyword, IdentEnd(s.Line, s.Pos+1, ""))
		}
		fs = append(fs, Frame{Mode: modeInterp, Delim: "}"})
	default:
		stops := "<&"
		if m.Vue || m.Svelte {
			stops = "<&{"
		}
		i := strings.IndexAny(rest[1:], stops)
		if i < 0 {
			s.EmitRest(Text)
			return fs
		}
		s.Emit(Text, i+1)
	}
	return fs
}

// attrs lexes the attributes of the tag on top of fs through the '>' that
// closes it.
func (m *Markup) attrs(s *Scanner, fs Frames) Frames {
	f := fs.Top()
	attr := ""
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
			name, id := f.Delim, f.N>>1
			fs = fs[:len(fs)-1]
			if f.N&tagClosing == 0 && id != 0 && m.embedded(id) != nil {
				fs = append(fs, Frame{Mode: modeEmbed, N: id, Delim: name})
			}
			return fs
		case b == '=':
			s.Emit(Operator, 1)
			s.Space()
			if v := unquotedValue(s.Rest(), m.Svelte); v != "" {
				if attr == "lang" || attr == "type" {
					m.setLang(f, `"`+v+`"`)
				}
				s.Emit(String, len(v))
				attr = ""
			}
		case b == '"' || b == '\'':
			if attr == "lang" || attr == "type" {
				m.setLang(f, s.Rest())
			}
			m.value(s, attr)
			attr = ""
		case m.Svelte && b == '{':
			s.Emit(Punctuation, 1)
			fs = append(fs, Frame{Mode: modeInterp, Delim: "}"})
			return fs
		case IdentStart(s.Line, s.Pos, ":@#"):
			end := IdentEnd(s.Line, s.Pos, "-:.@#")
			attr = strings.ToLower(s.Line[s.Pos:end])
			s.EmitTo(Attribute, end)
		default:
			s.EmitRune(Text)
		}
	}
	return fs
}

// unquotedValue returns the attribute value without quotes that rest
// starts with, as in class=btn, up to a space or the tag's '>' or "/>", or
// "" if a quoted value or, in Svelte, an expression starts there instead.
func unquotedValue(rest string, svelte bool) string {
	if rest == "" || rest[0] == '"' || rest[0] == '\'' || rest[0] == '>' || svelte && rest[0] == '{' {
		return ""
	}
	end := strings.IndexAny(rest, " \t>")
	if end < 0 {
		end = len(rest)
	}
	if end > 0 && end < len(rest) && rest[end] == '>' && rest[end-1] == '/' {
		end--
	}
	return rest[:end]
}

// setLang chooses the language of a script or style element from the value
// of its lang or type attribute, which starts value, quote included.
func (m *Markup) setLang(f *Frame, value string) {
	v, _, _ := strings.Cut(strings.ToLower(value[1:]), value[:1])
	id := f.N >> 1
	switch {
	case id == embedScript && (v == "ts" || v == "typescript" || strings.HasSuffix(v, "/typescript")):
		id = embedTypeScript
	case id == embedScript && v != "" && v != "module" && !strings.Contains(v, "javascript") && v != "js":
		// A script of another type, such as JSON or a template, stays Text.
		id = 0
	case id == embedStyle && (v == "scss" || v == "sass"):
		id = embedSCSS
	}
	f.N = f.N&tagClosing | id<<1
}

// value lexes the quoted value of the attribute attr. The value of one of
// Vue's directives is lexed as Script.
func (m *Markup) value(s *Scanner, attr string) {
	q := s.Rest()[:1]
	end := strings.Index(s.Rest()[1:], q)
	if end < 0 {
		s.Emit(String, 1)
		s.Delimit(String, q)
		return
	}
	if m.Vue && m.Script != nil && attr != "" && (strings.HasPrefix(attr, "v-") || strings.IndexByte(":@#", attr[0]) >= 0) {
		s.Emit(String, 1)
		s.Toks, _ = m.Script.AppendLine(s.Toks, s.Line[s.Pos:s.Pos+end], "")
		s.Pos += end
		s.Start = len(s.Toks)
		s.Emit(String, 1)
		return
	}
	s.Emit(String, end+2)
}

// interp lexes an interpolated expression as Script through the text that
// closes it: "}}" for Vue, or for Svelte the '}' that matches its '{'.
func (m *Markup) interp(s *Scanner, fs Frames) Frames {
	f := fs.Top()
	rest := s.Rest()
	end, closed := len(rest), false
	if f.Delim == "}}" {
		if i := strings.Index(rest, "}}"); i >= 0 {
			end, closed = i, true
		}
	} else {
		depth := int(f.N)
		for i := 0; i < len(rest); i++ {
			switch rest[i] {
			case '"', '\'', '`':
				if j := strings.IndexByte(rest[i+1:], rest[i]); j >= 0 {
					i += j + 1
				}
			case '{':
				depth++
			case '}':
				if depth == 0 {
					end, closed = i, true
				}
				depth--
			}
			if closed {
				break
			}
		}
		f.N = uint8(max(depth, 0))
	}
	if m.Script != nil {
		s.Toks, _ = m.Script.AppendLine(s.Toks, rest[:end], "")
		s.Pos += end
		s.Start = len(s.Toks)
	} else {
		s.Emit(Text, end)
	}
	if closed {
		s.Emit(Punctuation, len(f.Delim))
		fs = fs[:len(fs)-1]
	}
	return fs
}

// embed lexes the content of a script or style element, in its language
// and starting in inner, up to the tag that closes the element, and
// returns the State the language ends in.
func (m *Markup) embed(s *Scanner, fs Frames, inner State) (Frames, State) {
	f := fs.Top()
	rest := s.Rest()
	end := indexFold(rest, "</"+f.Delim)
	if end < 0 {
		end = len(rest)
	}
	lexer := m.embedded(f.N)
	if lexer == nil {
		s.Emit(Text, end)
	} else {
		s.Toks, inner = lexer.AppendLine(s.Toks, rest[:end], inner)
		s.Pos += end
		s.Start = len(s.Toks)
	}
	if end < len(rest) {
		return fs[:len(fs)-1], ""
	}
	return fs, inner
}

// indexFold returns the index of the first instance of sub in s, ignoring
// the case of ASCII letters, or -1 if sub is not in s.
func indexFold(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

// CSS lexes CSS, and SCSS's additions to it when SCSS: selectors, at-rules,
// properties, and values.
type CSS struct {
	// SCSS adds // comments, $variables, placeholders, and interpolation.
	SCSS bool
}

// AppendLine implements Lexer.
func (c *CSS) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	var buf [2]Frame
	fs := AppendFrames(Frames(buf[:0]), st)
	depth := 0
	if len(fs) > 0 && fs[0].Mode == modeBlock {
		depth = int(fs[0].N)
		fs = fs[1:]
	}
	comment := len(fs) > 0 && fs[0].Mode == modeComment
	value := false
	// A line inside a block that has no colon continues a value.
	if depth > 0 && !strings.ContainsAny(line, ":{}") {
		value = true
	}
	// stmt reports that a statement starts at the scanner's position, where
	// a declaration may, and decl that the statement is a declaration.
	stmt, decl := true, false
	// bracket is the offset of the next ']', for NextByte.
	bracket := -1
	for !s.Done() {
		if comment {
			if i := strings.Index(s.Rest(), "*/"); i >= 0 {
				s.Emit(Comment, i+2)
				comment = false
			} else {
				s.EmitRest(Comment)
			}
			continue
		}
		rest := s.Rest()
		atStmt := stmt
		stmt = false
		// Whether a statement is a declaration is judged at its first token.
		if atStmt && !IsSpace(rest[0]) && !strings.HasPrefix(rest, "/*") {
			decl = depth > 0 && !value && cssDeclaration(rest)
		}
		switch b := rest[0]; {
		case IsSpace(b):
			s.Space()
			stmt = atStmt
		case strings.HasPrefix(rest, "/*"):
			s.Emit(Comment, 2)
			comment, stmt = true, atStmt
		case c.SCSS && strings.HasPrefix(rest, "//"):
			s.EmitRest(Comment)
		case b == '{':
			s.Emit(Punctuation, 1)
			depth++
			value, stmt, decl = false, true, false
		case b == '}':
			s.Emit(Punctuation, 1)
			depth = max(depth-1, 0)
			value, stmt, decl = false, true, false
		case b == ';':
			s.Emit(Punctuation, 1)
			value, stmt, decl = false, true, false
		case b == '"' || b == '\'':
			s.EmitTo(String, quotedEnd(s.Line, s.Pos))
		case b == '@' && IdentStart(rest, 1, ""):
			s.Emit(Keyword, IdentEnd(rest, 1, "-"))
			value = true
		case c.SCSS && b == '$' && IdentStart(rest, 1, ""):
			s.Emit(Variable, IdentEnd(rest, 1, "-"))
			// A variable's declaration, such as $c: red.
			if !value && NextNonSpace(s.Line, s.Pos) == ':' {
				s.Space()
				s.Emit(Punctuation, 1)
				value = true
			}
		case c.SCSS && strings.HasPrefix(rest, "#{"):
			s.Emit(Variable, matchingEnd(rest, 1, '{', '}'))
		case value:
			c.valueToken(&s)
		case decl && atStmt:
			end := IdentEnd(rest, 0, "-")
			if end == 0 {
				end = 1
			}
			k := Property
			if strings.HasPrefix(rest, "--") {
				k = Variable
				end = IdentEnd(rest, 2, "-")
			}
			s.Emit(k, end)
			s.Space()
			if s.Peek(0) == ':' {
				s.Emit(Punctuation, 1)
				value = true
			}
		case decl && b == ':':
			// The colon after a property name that interpolation builds, as
			// in #{$prop}: red or margin-#{$side}: 0.
			s.Emit(Punctuation, 1)
			value = true
		default:
			c.selectorToken(&s, &bracket)
		}
	}
	var next Frames
	if depth > 0 {
		next = append(next, Frame{Mode: modeBlock, N: uint8(min(depth, 255))})
	}
	if comment {
		next = append(next, Frame{Mode: modeComment})
	}
	if len(next) == 0 {
		return s.Toks, ""
	}
	if ns := next.State(); ns != st {
		return s.Toks, ns
	}
	return s.Toks, st
}

// cssDeclaration reports whether a declaration, such as "color: red",
// starts s: a colon comes before a ';', '{', or '}' does, and the first of
// them is not a '{', as it is after a selector such as "a:hover {". The
// braces of an SCSS interpolation, such as #{$size}, are skipped.
func cssDeclaration(s string) bool {
	end := len(s)
scan:
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '#':
			if i+1 < len(s) && s[i+1] == '{' {
				i = matchingEnd(s, i+1, '{', '}') - 1
			}
		case '{':
			return false
		case ';', '}':
			end = i
			break scan
		}
	}
	return strings.IndexByte(s[:end], ':') > 0
}

// valueToken lexes the token at the scanner's position in a property's
// value or an at-rule's prelude.
func (c *CSS) valueToken(s *Scanner) {
	rest := s.Rest()
	switch b := rest[0]; {
	case IsDigit(b) || ((b == '.' || b == '-' || b == '+') && len(rest) > 1 && (IsDigit(rest[1]) || rest[1] == '.')):
		// A dot or sign before no digit, as in .. or -., is a number of
		// its own byte, so that the scanner moves on.
		end := max(NumberEnd(rest, boolInt(b == '-' || b == '+')), 1)
		if end < len(rest) && rest[end] == '%' {
			end++
		}
		s.Emit(Number, end)
	case b == '#' && len(rest) > 1 && isHexDigit(rest[1]):
		end := 1
		for end < len(rest) && IsWordByte(rest[end]) {
			end++
		}
		s.Emit(Number, end)
	case b == '!':
		s.Emit(Keyword, IdentEnd(rest, 1, ""))
	case strings.HasPrefix(rest, "--") && IdentStart(rest, 2, ""):
		s.Emit(Variable, IdentEnd(rest, 2, "-"))
	case IdentStart(rest, 0, "") || (b == '-' && IdentStart(rest, 1, "")):
		end := IdentEnd(rest, 0, "-")
		if end < len(rest) && rest[end] == '(' {
			s.Emit(Function, end)
			// An unquoted url() argument is a String.
			if strings.EqualFold(rest[:end], "url") {
				s.Emit(Punctuation, 1)
				if r := s.Rest(); r != "" && r[0] != '"' && r[0] != '\'' {
					if i := strings.IndexByte(r, ')'); i > 0 {
						s.Emit(String, i)
					}
				}
			}
			return
		}
		s.Emit(Text, end)
	case strings.IndexByte("(),", b) >= 0:
		s.Emit(Punctuation, 1)
	case strings.IndexByte("+-*/=<>:", b) >= 0:
		s.Emit(Operator, 1)
	default:
		s.EmitRune(Text)
	}
}

// selectorToken lexes the token at the scanner's position in a selector,
// finding the ']' of an attribute selector with NextByte and bracket.
func (c *CSS) selectorToken(s *Scanner, bracket *int) {
	rest := s.Rest()
	switch b := rest[0]; {
	case (b == '.' || b == '#' || (c.SCSS && b == '%')) && IdentStart(rest, 1, "-"):
		s.Emit(Type, IdentEnd(rest, 1, "-"))
	case b == ':':
		n := 1
		if strings.HasPrefix(rest, "::") {
			n = 2
		}
		s.Emit(Decorator, IdentEnd(rest, n, "-"))
	case b == '[':
		end := NextByte(s.Line, s.Pos, ']', bracket) - s.Pos
		if end >= len(rest) {
			s.Emit(Punctuation, 1)
			return
		}
		s.Emit(Punctuation, 1)
		inside := rest[1:end]
		name := IdentEnd(inside, 0, "-")
		s.Emit(Attribute, name)
		if op := strings.IndexByte(inside, '='); op >= name {
			// The operator runs from the name through '=', as in ~= or ^=.
			s.Emit(Operator, op-name+1)
			s.Emit(String, len(inside)-op-1)
		} else {
			s.Emit(Text, len(inside)-name)
		}
		s.Emit(Punctuation, 1)
	case b == '&':
		s.Emit(Keyword, 1)
	case IsDigit(b):
		end := NumberEnd(rest, 0)
		if end < len(rest) && rest[end] == '%' {
			end++
		}
		s.Emit(Number, end)
	case IdentStart(rest, 0, "-"):
		s.Emit(Tag, IdentEnd(rest, 0, "-"))
	case strings.IndexByte(">+~*", b) >= 0:
		s.Emit(Operator, 1)
	case b == ',' || b == '(' || b == ')':
		s.Emit(Punctuation, 1)
	default:
		s.EmitRune(Text)
	}
}
