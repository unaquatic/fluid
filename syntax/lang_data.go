package syntax

import "strings"

// JSON lexes JSON, and the comments of JSON with comments: an object's keys
// are Properties.
type JSON struct{}

// jsonLang returns JSON.
func jsonLang() *Language {
	return &Language{
		Name:      "json",
		Aliases:   []string{"jsonc", "json5", "jsonl", "ndjson", "geojson", "webmanifest", "avsc", "har"},
		Filenames: []string{".babelrc", ".eslintrc", ".prettierrc", "composer.lock", "flake.lock"},
		Lexer:     JSON{},
	}
}

// AppendLine implements Lexer.
func (JSON) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	var buf [1]Frame
	fs := AppendFrames(Frames(buf[:0]), st)
	for !s.Done() {
		if f := fs.Top(); f != nil && f.Mode == modeComment {
			if i := strings.Index(s.Rest(), "*/"); i >= 0 {
				s.Emit(Comment, i+2)
				fs = fs[:0]
			} else {
				s.EmitRest(Comment)
			}
			continue
		}
		switch b := s.Peek(0); {
		case IsSpace(b):
			s.Space()
		case b == '"' || b == '\'':
			end := quotedEnd(s.Line, s.Pos)
			k := String
			if NextNonSpace(s.Line, end) == ':' {
				k = Property
			}
			s.EmitTo(k, end)
		case b == '-' || b == '+' || IsDigit(b) || (b == '.' && IsDigit(s.Peek(1))):
			s.EmitTo(Number, NumberEnd(s.Line, s.Pos+boolInt(!IsDigit(b) && b != '.')))
		case s.HasPrefix("//"):
			s.EmitRest(Comment)
		case s.HasPrefix("/*"):
			s.Emit(Comment, 2)
			fs = append(fs, Frame{Mode: modeComment, N: 1})
		case IdentStart(s.Line, s.Pos, "$"):
			end := IdentEnd(s.Line, s.Pos, "$")
			k := Variable
			switch s.Line[s.Pos:end] {
			case "true", "false", "null", "NaN", "Infinity":
				k = Primitive
			default:
				if NextNonSpace(s.Line, end) == ':' {
					k = Property
				}
			}
			s.EmitTo(k, end)
		case strings.IndexByte("{}[],:", b) >= 0:
			s.Emit(Punctuation, 1)
		default:
			s.EmitRune(Text)
		}
	}
	return s.Toks, s.Finish(fs)
}

// quotedEnd returns the offset in line just past the string at i, which
// starts with its quote and may escape characters with backslashes, or
// len(line) if the line does not close it.
func quotedEnd(line string, i int) int {
	q := line[i]
	for j := i + 1; j < len(line); j++ {
		switch line[j] {
		case '\\':
			j++
		case q:
			return j + 1
		}
	}
	return len(line)
}

// YAML lexes YAML: keys are Properties, plain scalars are Text, and the
// lines of a block scalar, such as one after "|", are Strings.
type YAML struct{}

// yamlLang returns YAML.
func yamlLang() *Language {
	return &Language{
		Name:      "yaml",
		Aliases:   []string{"yml"},
		Filenames: []string{".clang-format", ".clang-tidy", ".gemrc", "CITATION.cff", "glide.lock"},
		Lexer:     YAML{},
	}
}

// AppendLine implements Lexer.
func (YAML) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	indent := SpaceEnd(line, 0)
	// A block scalar runs over the lines indented further than the line
	// that opens it, and over blank lines.
	if f, _, ok := st.Top(); ok && f.Mode == modeBlock {
		if indent == len(line) || indent > int(f.N) {
			s.EmitRest(String)
			return s.Toks, st
		}
	}
	s.Space()
	if s.HasPrefix("---") || s.HasPrefix("...") {
		if rest := s.Rest()[3:]; rest == "" || IsSpace(rest[0]) {
			s.Emit(Punctuation, 3)
		}
	}
	key := true
	for !s.Done() {
		switch b := s.Peek(0); {
		case IsSpace(b):
			s.Space()
		case b == '#' && (s.Pos == 0 || IsSpace(s.Line[s.Pos-1])):
			s.EmitRest(Comment)
		case b == '-' && (s.Peek(1) == 0 || IsSpace(s.Peek(1))):
			s.Emit(Punctuation, 1)
			key = true
		case b == '"' || b == '\'':
			end := quotedEnd(s.Line, s.Pos)
			k := String
			if key && yamlKeyEnd(s.Line, end) {
				k = Property
			}
			s.EmitTo(k, end)
		case strings.IndexByte("{}[],", b) >= 0:
			s.Emit(Punctuation, 1)
			key = b == '{' || b == ','
		case b == ':' && (s.Peek(1) == 0 || IsSpace(s.Peek(1))):
			s.Emit(Punctuation, 1)
			key = false
		case (b == '|' || b == '>') && yamlBlockHeader(s.Line, s.Pos) > 0:
			// A block scalar header, such as |- or >2, ends its line.
			s.EmitTo(Punctuation, yamlBlockHeader(s.Line, s.Pos))
			s.Space()
			s.EmitRest(Comment)
			return s.Toks, State("").Push(Frame{Mode: modeBlock, N: uint8(min(indent, 255))})
		case (b == '&' || b == '*') && IdentStart(s.Line, s.Pos+1, "-"):
			s.EmitTo(Variable, IdentEnd(s.Line, s.Pos+1, "-"))
		case b == '!':
			end := s.Pos + 1
			for end < len(s.Line) && !IsSpace(s.Line[end]) {
				end++
			}
			s.EmitTo(Type, end)
		default:
			end := yamlPlainEnd(s.Line, s.Pos)
			if key && yamlKeyEnd(s.Line, end) {
				s.EmitTo(Property, trimRightSpace(s.Line, s.Pos, end))
				continue
			}
			s.EmitTo(yamlScalarKind(s.Line[s.Pos:end]), end)
			key = false
		}
	}
	return s.Toks, ""
}

// yamlBlockHeader returns the offset in line just past the block scalar
// header at i, such as | or >-2, or 0 if no header that ends the line,
// but for a comment, is there.
func yamlBlockHeader(line string, i int) int {
	end := i + 1
	for end < len(line) && (line[end] == '-' || line[end] == '+' || IsDigit(line[end])) {
		end++
	}
	if rest := strings.TrimLeft(line[end:], " \t"); rest != "" && rest[0] != '#' {
		return 0
	}
	return end
}

// yamlKeyEnd reports whether a key's ':' comes at offset i in line, after
// spaces.
func yamlKeyEnd(line string, i int) bool {
	i = SpaceEnd(line, i)
	return i < len(line) && line[i] == ':' && (i+1 == len(line) || IsSpace(line[i+1]))
}

// yamlPlainEnd returns the offset in line of the end of the plain scalar
// at i: before ": ", " #", or, in a flow collection, a bracket or comma.
func yamlPlainEnd(line string, i int) int {
	j := i
	for j < len(line) {
		switch b := line[j]; {
		case b == ':' && (j+1 == len(line) || IsSpace(line[j+1])):
			return j
		case b == '#' && j > i && IsSpace(line[j-1]):
			return trimRightSpace(line, i, j)
		case strings.IndexByte(",[]{}", b) >= 0 && j > i:
			return j
		}
		j++
	}
	return trimRightSpace(line, i, j)
}

// yamlScalarKind returns the kind of the plain scalar v.
func yamlScalarKind(v string) Kind {
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return Primitive
	}
	if v != "" && (IsDigit(v[0]) || (len(v) > 1 && (v[0] == '-' || v[0] == '+' || v[0] == '.') && IsDigit(v[1]))) {
		// A YAML number has no suffix, as C's 10u8 has, so that a scalar
		// such as 512Mi is Text.
		start := boolInt(v[0] == '-' || v[0] == '+')
		if numberBody(v, start) == len(v) {
			return Number
		}
	}
	return Text
}

// TOML lexes TOML: tables are Keywords and keys Properties.
type TOML struct{}

// The quotes of TOML, as the N of a modeQuote frame.
const (
	tomlBasic   = iota // """...""", with escapes
	tomlLiteral        // '''...''', without
)

// tomlLang returns TOML.
func tomlLang() *Language {
	return &Language{
		Name:      "toml",
		Filenames: []string{"Cargo.lock", "Pipfile", "poetry.lock", "uv.lock"},
		Lexer:     TOML{},
	}
}

// AppendLine implements Lexer.
func (TOML) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	if f, _, ok := st.Top(); ok && f.Mode == modeQuote {
		if !tomlMultiline(&s, f.N) {
			return s.Toks, st
		}
	} else {
		s.Space()
		if s.Peek(0) == '[' {
			end := strings.LastIndexByte(s.Line, ']')
			if end < s.Pos {
				end = len(s.Line) - 1
			}
			s.EmitTo(Keyword, end+1)
		}
	}
	// A line without '=' continues an array, and holds no key.
	key := st == "" && strings.Contains(s.Rest(), "=")
	// open holds the brackets open on the line, innermost last: a key
	// follows a comma in an inline table, {...}, but not in an array.
	var buf [8]byte
	open := buf[:0]
	for !s.Done() {
		switch b := s.Peek(0); {
		case IsSpace(b):
			s.Space()
		case b == '#':
			s.EmitRest(Comment)
		case s.HasPrefix(`"""`) || s.HasPrefix(`'''`):
			n := uint8(tomlBasic)
			if b == '\'' {
				n = tomlLiteral
			}
			s.Emit(String, 3)
			if !tomlMultiline(&s, n) {
				return s.Toks, State("").Push(Frame{Mode: modeQuote, N: n})
			}
		case b == '"' || b == '\'':
			end := quotedEnd(s.Line, s.Pos)
			if b == '\'' {
				if i := strings.IndexByte(s.Rest()[1:], '\''); i >= 0 {
					end = s.Pos + i + 2
				}
			}
			k := String
			if key {
				k = Property
			}
			s.EmitTo(k, end)
		case b == '=':
			s.Emit(Operator, 1)
			key = false
		case b == '{' || b == '[':
			s.Emit(Punctuation, 1)
			open = append(open, b)
			key = b == '{'
		case b == '}' || b == ']':
			s.Emit(Punctuation, 1)
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
		case b == ',':
			s.Emit(Punctuation, 1)
			key = len(open) > 0 && open[len(open)-1] == '{'
		case b == '.' && key:
			s.Emit(Punctuation, 1)
		default:
			end := s.Pos
			for end < len(s.Line) && !IsSpace(s.Line[end]) && strings.IndexByte("=,[]{}#", s.Line[end]) < 0 && !(key && s.Line[end] == '.') {
				end++
			}
			end = max(end, s.Pos+runeLen(s.Line, s.Pos))
			word := s.Line[s.Pos:end]
			switch {
			case key:
				s.EmitTo(Property, end)
			case word == "true" || word == "false":
				s.EmitTo(Primitive, end)
			case IsDigit(b) || b == '-' || b == '+' || word == "inf" || word == "nan":
				s.EmitTo(Number, end)
			default:
				s.EmitTo(Text, end)
			}
		}
	}
	return s.Toks, ""
}

// tomlMultiline lexes the body of a multiline string with quote n and
// reports whether it closed on the line.
func tomlMultiline(s *Scanner, n uint8) bool {
	close := `"""`
	if n == tomlLiteral {
		close = `'''`
	}
	for !s.Done() {
		rest := s.Rest()
		i := strings.Index(rest, close)
		e := -1
		if n == tomlBasic {
			// Search for an escape only up to the close.
			upto := rest
			if i >= 0 {
				upto = rest[:i]
			}
			e = strings.IndexByte(upto, '\\')
		}
		switch {
		case e >= 0:
			s.Emit(String, e)
			s.Emit(StringEscape, 1+runeLen(s.Line, s.Pos+1))
		case i >= 0:
			s.Emit(String, i+3)
			return true
		default:
			s.EmitRest(String)
		}
	}
	return false
}

// INI lexes INI files, Java properties, and .env files: sections are
// Keywords, keys Properties, and values Strings.
type INI struct{}

// iniLang returns INI.
func iniLang() *Language {
	return &Language{
		Name: "ini",
		Aliases: []string{
			"cfg", "conf", "dosini", "env", "dotenv", "properties", "editorconfig",
			"gitconfig", "inf", "service", "desktop", "systemd",
		},
		Filenames: []string{".env", ".editorconfig", ".gitconfig", ".npmrc", ".gitmodules", "setup.cfg"},
		Lexer:     INI{},
	}
}

// AppendLine implements Lexer.
func (INI) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	s.Space()
	switch b := s.Peek(0); {
	case b == ';' || b == '#':
		s.EmitRest(Comment)
		return s.Toks, ""
	case b == '[':
		s.EmitTo(Keyword, strings.LastIndexByte(s.Line, ']')+1)
		s.EmitRest(Text)
		return s.Toks, ""
	}
	if s.HasPrefix("export ") {
		s.Emit(Keyword, 6)
		s.Space()
	}
	i := strings.IndexAny(s.Rest(), "=:")
	if i < 0 {
		s.EmitRest(Text)
		return s.Toks, ""
	}
	s.EmitTo(Property, trimRightSpace(s.Line, s.Pos, s.Pos+i))
	s.Space()
	s.Emit(Operator, 1)
	s.Space()
	s.EmitRest(String)
	return s.Toks, ""
}

// Diff lexes unified and context diffs: added lines are Inserted, removed
// lines Deleted, file headers Heading, and hunk headers Subheading.
type Diff struct{}

// diffLang returns diff.
func diffLang() *Language {
	return &Language{
		Name:    "diff",
		Aliases: []string{"patch", "udiff"},
		Lexer:   Diff{},
	}
}

// AppendLine implements Lexer.
func (Diff) AppendLine(dst []Token, line string, _ State) ([]Token, State) {
	if line == "" {
		return dst, ""
	}
	k := Text
	switch {
	case line[0] == ' ':
	case line == "---" || line[0] == '!':
		k = Strong
	case line[0] == '+' || strings.HasPrefix(line, "> "):
		k = Inserted
	case line[0] == '-' || strings.HasPrefix(line, "< "):
		k = Deleted
	case line[0] == '@' || isNormalDiffCommand(line):
		k = Subheading
	case line[0] == '=' || strings.HasPrefix(line, "diff") || strings.HasPrefix(line, "index") || strings.HasPrefix(line, "Index"):
		k = Heading
	}
	return append(dst, Token{Kind: k, Text: line}), ""
}

// isNormalDiffCommand reports whether line is a command of a normal diff,
// such as 5,7c8,10: a range, a letter, and a range.
func isNormalDiffCommand(line string) bool {
	i := diffRangeEnd(line, 0)
	if i == 0 || i >= len(line) || strings.IndexByte("acd", line[i]) < 0 {
		return false
	}
	return diffRangeEnd(line, i+1) == len(line) && i+1 < len(line)
}

// diffRangeEnd returns the offset in line of the end of the range, such as
// 5 or 5,7, at i, or i if no range is there.
func diffRangeEnd(line string, i int) int {
	j := i
	for j < len(line) && IsDigit(line[j]) {
		j++
	}
	if j > i && j+1 < len(line) && line[j] == ',' && IsDigit(line[j+1]) {
		j++
		for j < len(line) && IsDigit(line[j]) {
			j++
		}
	}
	return j
}
