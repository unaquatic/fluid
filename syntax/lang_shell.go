package syntax

import (
	"slices"
	"strings"
)

// Shell lexes the Bourne shell language of sh, bash, and zsh.
//
// The first word of a command is a Keyword or a Builtin when Words names it
// so; other commands and their arguments are Text.
type Shell struct {
	// Words maps keywords and builtin commands to their kinds.
	Words map[string]Kind
}

// The quotes of the shell, as the N of a modeQuote frame.
const (
	shDouble   = iota // "..."
	shSingle          // '...'
	shANSI            // $'...'
	shBacktick        // `...`
)

// shellLang returns the shell language.
func shellLang() *Language {
	return &Language{
		Name: "bash",
		Aliases: []string{
			"sh", "shell", "zsh", "ksh", "ash", "dash", "shellscript", "bashrc", "zshrc",
			"profile", "ebuild", "eclass", "exheres-0", "exlib",
		},
		Filenames: []string{".bashrc", ".bash_profile", ".zshrc", ".profile", "PKGBUILD", "APKBUILD"},
		Lexer:     newShell(),
	}
}

// newShell returns a Shell with the keywords and builtins of bash.
func newShell() *Shell {
	return &Shell{Words: wordKinds(
		Keyword, `if then else elif fi case esac for select while until do done
			function time coproc [[ ]]`,
		Builtin, `alias bg bind break builtin caller cd command compgen complete
			compopt continue declare dirs disown echo enable eval exec exit export
			false fc fg getopts hash help history jobs kill let local logout
			mapfile popd printf pushd pwd read readarray readonly return set shift
			shopt source suspend test times trap true type typeset ulimit umask
			unalias unset wait . [`,
	)}
}

// AppendLine implements Lexer.
func (sh *Shell) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	var buf [4]Frame
	fs := AppendFrames(Frames(buf[:0]), st)
	cmd := true
	// A line after a backslash continues the command of the line before,
	// whose frame tells whether a command may come next.
	if f := fs.Top(); f != nil && f.Mode == modeBody {
		cmd = f.N != 0
		fs = fs[:len(fs)-1]
	}
	// expectIn counts the words before the "in" of a for loop, and arith
	// the arithmetic expressions open, in which << shifts.
	expectIn, arith := 0, 0
	for !s.Done() {
		var ok bool
		if fs, ok = s.Continue(fs); ok {
			continue
		}
		if f := fs.Top(); f != nil && f.Mode == modeQuote {
			fs = sh.quote(&s, fs)
			cmd = false
			continue
		}
		fs, cmd = sh.word(&s, fs, cmd, &expectIn, &arith)
	}
	if s.Continued {
		fs = append(fs, Frame{Mode: modeBody, N: uint8(boolInt(cmd))})
	}
	return s.Toks, s.Finish(fs)
}

// word lexes the word or operator at the scanner's position and returns
// whether a command may come after it. expectIn counts the words left
// before a for loop's "in", and arith the arithmetic expressions open.
func (sh *Shell) word(s *Scanner, fs Frames, cmd bool, expectIn, arith *int) (Frames, bool) {
	b := s.Peek(0)
	prev := byte(' ')
	if s.Pos > 0 {
		prev = s.Line[s.Pos-1]
	}
	switch {
	case IsSpace(b):
		s.Space()
		return fs, cmd
	case b == '#' && (IsSpace(prev) || strings.IndexByte(";|&(", prev) >= 0 || s.Pos == 0):
		s.EmitRest(Comment)
		return fs, cmd
	case b == '"':
		s.Emit(String, 1)
		fs = append(fs, Frame{Mode: modeQuote, N: shDouble})
		return fs, false
	case b == '\'':
		s.Emit(String, 1)
		fs = append(fs, Frame{Mode: modeQuote, N: shSingle})
		return fs, false
	case b == '`':
		s.Emit(String, 1)
		fs = append(fs, Frame{Mode: modeQuote, N: shBacktick})
		return fs, false
	case b == '$' && s.HasPrefix("$(("):
		s.Emit(Operator, 3)
		*arith++
		return fs, false
	case b == '(' && s.Peek(1) == '(' && cmd:
		// An arithmetic command, such as (( i++ )).
		s.Emit(Punctuation, 2)
		*arith++
		return fs, false
	case b == ')' && s.Peek(1) == ')' && *arith > 0:
		s.Emit(Punctuation, 2)
		*arith--
		return fs, false
	case b == '$' && s.Peek(1) == '\'':
		s.Emit(String, 2)
		fs = append(fs, Frame{Mode: modeQuote, N: shANSI})
		return fs, false
	case b == '$':
		return fs, sh.dollar(s, Variable)
	case b == '\\':
		if s.Pos+1 == len(s.Line) {
			s.Continued = true
			s.Emit(StringEscape, 1)
			return fs, cmd
		}
		s.Emit(StringEscape, 1+runeLen(s.Line, s.Pos+1))
		return fs, false
	case b == '<' && s.Peek(1) == '<' && s.Peek(2) != '<' && *arith == 0:
		if shellHeredoc(s) {
			return fs, cmd
		}
		s.Emit(Operator, 2)
		return fs, cmd
	case b == '&' && s.Peek(1) == '>':
		s.Emit(Operator, 2+boolInt(s.Peek(2) == '>'))
		return fs, cmd
	case b == ';' || b == '|' || b == '&':
		n := 1
		if next := s.Peek(1); next == b || (b == '|' && next == '&') {
			n = 2
		}
		s.Emit(Operator, n)
		return fs, true
	case b == '<' || b == '>':
		n := 1
		for n < 3 && (s.Peek(n) == '>' || s.Peek(n) == '&' || s.Peek(n) == '<') {
			n++
		}
		s.Emit(Operator, n)
		return fs, cmd
	case b == '(' || b == '{':
		s.Emit(Punctuation, 1)
		return fs, true
	case b == ')' || b == '}':
		s.Emit(Punctuation, 1)
		return fs, false
	case b == '=':
		s.Emit(Operator, 1)
		return fs, cmd
	}
	end := shellWordEnd(s.Line, s.Pos)
	word := s.Line[s.Pos:end]
	// An assignment, NAME=value, before a command.
	if cmd {
		if n := IdentEnd(word, 0, ""); n > 0 && IdentStart(word, 0, "") && n < len(word) && (word[n] == '=' || strings.HasPrefix(word[n:], "+=")) {
			s.Emit(Variable, n)
			s.Emit(Operator, 1+boolInt(word[n] == '+'))
			s.EmitTo(Text, end)
			return fs, true
		}
	}
	// A digit before a redirection names a file descriptor.
	if isDigits(word) {
		s.EmitTo(Number, end)
		return fs, false
	}
	if word == "in" && *expectIn > 0 {
		s.EmitTo(Keyword, end)
		*expectIn = 0
		return fs, false
	}
	if *expectIn > 0 {
		*expectIn--
	}
	if word == "]]" {
		s.EmitTo(Keyword, end)
		return fs, false
	}
	if !cmd {
		s.EmitTo(Text, end)
		return fs, false
	}
	k, ok := sh.Words[word]
	switch {
	case ok && k == Keyword:
		s.EmitTo(Keyword, end)
		switch word {
		case "for", "select", "case":
			*expectIn = 2
			return fs, false
		case "function":
			s.Space()
			if !s.Done() {
				s.EmitTo(Function, shellWordEnd(s.Line, s.Pos))
			}
			return fs, false
		case "fi", "done", "esac", "]]":
			return fs, false
		}
		return fs, true
	case ok:
		s.EmitTo(k, end)
	case strings.HasPrefix(s.Line[end:], "()"):
		s.EmitTo(Function, end)
	default:
		s.EmitTo(Text, end)
	}
	return fs, false
}

// isDigits reports whether w is one or more ASCII digits.
func isDigits(w string) bool {
	for i := range len(w) {
		if !IsDigit(w[i]) {
			return false
		}
	}
	return w != ""
}

// shellWordEnd returns the offset in line of the end of the word at i,
// which runs to a space, a quote, or a character the shell treats apart.
func shellWordEnd(line string, i int) int {
	j := i
	for {
		for j < len(line) && !IsSpace(line[j]) && strings.IndexByte(";|&()<>\"'`$\\=", line[j]) < 0 {
			j++
		}
		// An '=' inside a word, as in --flag=value, belongs to it.
		if j < len(line) && line[j] == '=' && j > i {
			j++
			continue
		}
		break
	}
	if j == i {
		return i + runeLen(line, i)
	}
	return j
}

// dollar lexes the expansion at the scanner's position, which starts with
// '$', as k: a variable, a braced parameter expansion, or the opening of a
// command substitution, which inside a string is lexed whole. It returns
// whether a command may come after it.
func (sh *Shell) dollar(s *Scanner, k Kind) bool {
	rest := s.Rest()
	if len(rest) < 2 {
		s.Emit(Text, 1)
		return false
	}
	switch b := rest[1]; {
	case b == '{':
		s.Emit(k, matchingEnd(rest, 1, '{', '}'))
	case b == '(':
		if k != Variable {
			s.Emit(k, matchingEnd(rest, 1, '(', ')'))
			return false
		}
		s.Emit(Operator, 2)
		return true
	case IdentStart(rest, 1, ""):
		s.Emit(k, IdentEnd(rest, 1, ""))
	case IsDigit(b) || strings.IndexByte("@#?$!*-", b) >= 0:
		s.Emit(k, 2)
	default:
		s.Emit(Text, 1)
	}
	return false
}

// matchingEnd returns the offset in s just past the close that matches the
// open at s[i], or len(s) if the line does not close it.
func matchingEnd(s string, i int, open, close byte) int {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case open:
			depth++
		case close:
			if depth--; depth == 0 {
				return j + 1
			}
		}
	}
	return len(s)
}

// quote lexes the body of a quoted string.
func (sh *Shell) quote(s *Scanner, fs Frames) Frames {
	f := fs.Top()
	close, escapes := byte('"'), true
	switch f.N {
	case shSingle:
		close, escapes = '\'', false
	case shANSI:
		close = '\''
	case shBacktick:
		close = '`'
	}
	for !s.Done() {
		rest := s.Rest()
		i := 0
		for i < len(rest) && rest[i] != close && !(escapes && rest[i] == '\\') && !(f.N == shDouble && rest[i] == '$') {
			i++
		}
		s.Emit(String, i)
		switch {
		case s.Done():
		case s.Peek(0) == close:
			s.Emit(String, 1)
			return fs[:len(fs)-1]
		case s.Peek(0) == '\\':
			s.Emit(StringEscape, 1+runeLen(s.Line, s.Pos+1))
		case s.Peek(1) == '(':
			sh.dollar(s, StringEscape)
		default:
			sh.dollar(s, Variable)
		}
	}
	return fs
}

// shellHeredoc lexes the start of a heredoc, such as <<EOF, <<-EOF, or
// << 'EOF', and reports whether one starts.
func shellHeredoc(s *Scanner) bool {
	rest := s.Rest()
	i := 2
	indented := i < len(rest) && rest[i] == '-'
	if indented {
		i++
	}
	j := SpaceEnd(rest, i)
	var word string
	end := j
	switch {
	case j < len(rest) && (rest[j] == '\'' || rest[j] == '"'):
		k := strings.IndexByte(rest[j+1:], rest[j])
		if k <= 0 {
			return false
		}
		word, end = rest[j+1:j+1+k], j+2+k
	case j < len(rest) && rest[j] == '\\':
		end = IdentEnd(rest, j+1, "")
		word = rest[j+1 : end]
	default:
		end = IdentEnd(rest, j, "")
		word = rest[j:end]
	}
	// A word that starts with a digit is a shift's operand, as in
	// $[1 << 2], rather than a heredoc's.
	if word == "" || IsDigit(word[0]) {
		return false
	}
	s.Emit(Operator, i)
	s.EmitTo(Text, s.Pos+j-i)
	s.Emit(String, end-j)
	s.Heredoc(String, word, indented, false)
	return true
}

// Dockerfile lexes Dockerfiles: comments and instructions, whose arguments
// Shell lexes.
type Dockerfile struct {
	// Shell lexes the arguments of instructions.
	Shell *Shell
}

// dockerInstructions are the instructions of a Dockerfile, in lower case.
var dockerInstructions = []string{
	"add", "arg", "cmd", "copy", "entrypoint", "env", "expose", "from", "healthcheck", "label",
	"maintainer", "onbuild", "run", "shell", "stopsignal", "user", "volume", "workdir",
}

// dockerCommands are the instructions whose arguments start with a command
// or an assignment.
var dockerCommands = []string{"arg", "cmd", "entrypoint", "env", "healthcheck", "label", "onbuild", "run", "shell"}

// dockerfileLang returns the Dockerfile language.
func dockerfileLang(sh *Shell) *Language {
	return &Language{
		Name:      "dockerfile",
		Aliases:   []string{"docker", "containerfile"},
		Filenames: []string{"Dockerfile", "Containerfile"},
		Lexer:     &Dockerfile{Shell: sh},
	}
}

// AppendLine implements Lexer.
func (d *Dockerfile) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	inner := st
	if f, below, ok := st.Top(); ok && f.Mode == modeEmbed {
		// The line continues an instruction.
		return d.args(s, line, below)
	}
	s.Space()
	if s.Peek(0) == '#' {
		s.EmitRest(Comment)
		return s.Toks, ""
	}
	for !s.Done() {
		end := IdentEnd(s.Line, s.Pos, "")
		word := strings.ToLower(s.Line[s.Pos:end])
		if end == s.Pos || !slices.Contains(dockerInstructions, word) {
			break
		}
		s.EmitTo(Keyword, end)
		s.Space()
		// Only some instructions take a command; the arguments of the
		// others start as a continued command does, past its name.
		if !slices.Contains(dockerCommands, word) {
			inner = inner.Push(Frame{Mode: modeBody})
		}
		if word == "from" {
			for !s.Done() {
				end := shellWordEnd(s.Line, s.Pos)
				k := Text
				if strings.EqualFold(s.Line[s.Pos:end], "as") {
					k = Keyword
				}
				s.EmitTo(k, end)
				s.Space()
			}
			return s.Toks, ""
		}
		if word != "onbuild" {
			break
		}
	}
	// The exec form of a command is a JSON array.
	if s.Peek(0) == '[' {
		toks, _ := JSON{}.AppendLine(s.Toks, s.Rest(), "")
		return toks, ""
	}
	return d.args(s, s.Line[s.Pos:], inner)
}

// args lexes text, the arguments of an instruction that end s's line, with
// the shell, starting in st, and returns the tokens and the State for the
// next line.
func (d *Dockerfile) args(s Scanner, text string, st State) ([]Token, State) {
	toks, next := d.Shell.AppendLine(s.Toks, text, st)
	if next == "" {
		return toks, ""
	}
	return toks, next.Push(Frame{Mode: modeEmbed})
}

// Makefile lexes makefiles: rules, variable assignments, directives, and
// comments, with recipes lexed by Shell.
type Makefile struct {
	// Shell lexes recipes.
	Shell *Shell
}

// makefileLang returns the makefile language.
func makefileLang(sh *Shell) *Language {
	return &Language{
		Name:      "makefile",
		Aliases:   []string{"make", "mk", "mak", "bsdmake"},
		Filenames: []string{"Makefile", "GNUmakefile", "makefile"},
		Lexer:     &Makefile{Shell: sh},
	}
}

// makeDirectives are the directives of GNU make.
var makeDirectives = []string{
	"include", "-include", "sinclude", "ifeq", "ifneq", "ifdef", "ifndef", "else", "endif",
	"define", "endef", "export", "unexport", "override", "vpath", "undefine", "private",
}

// AppendLine implements Lexer.
func (m *Makefile) AppendLine(dst []Token, line string, st State) ([]Token, State) {
	s := NewScanner(dst, line)
	if f, below, ok := st.Top(); (ok && f.Mode == modeEmbed) || strings.HasPrefix(line, "\t") {
		return m.recipe(s, below)
	}
	s.Space()
	if s.Peek(0) == '#' {
		s.EmitRest(Comment)
		return s.Toks, ""
	}
	end := s.Pos
	for end < len(line) && !IsSpace(line[end]) && line[end] != ':' && line[end] != '=' && line[end] != '$' {
		end++
	}
	if slices.Contains(makeDirectives, line[s.Pos:end]) {
		s.EmitTo(Keyword, end)
		m.text(&s)
		return s.Toks, ""
	}
	// An assignment's operator, or a rule's colon, after the first words.
	if i := strings.IndexAny(line[s.Pos:], ":=!?+"); i >= 0 {
		i += s.Pos
		rest := line[i:]
		for _, op := range []string{"::=", ":=", "?=", "+=", "!=", "="} {
			if strings.HasPrefix(rest, op) {
				s.EmitTo(Variable, trimRightSpace(line, s.Pos, i))
				s.EmitTo(Text, i)
				s.Emit(Operator, len(op))
				m.text(&s)
				return s.Toks, ""
			}
		}
		if rest[0] == ':' {
			m.targets(&s, i)
			s.Emit(Operator, 1+boolInt(strings.HasPrefix(rest, "::")))
			m.text(&s)
			return s.Toks, ""
		}
	}
	m.text(&s)
	return s.Toks, ""
}

// trimRightSpace returns the offset in line before the spaces that end
// line[i:j].
func trimRightSpace(line string, i, j int) int {
	for j > i && IsSpace(line[j-1]) {
		j--
	}
	return j
}

// targets lexes the targets of a rule, which run to end, as Functions and
// the variables among them as Variables.
func (m *Makefile) targets(s *Scanner, end int) {
	for s.Pos < end {
		switch b := s.Peek(0); {
		case IsSpace(b):
			s.Space()
		case b == '$':
			makeRef(s)
		default:
			j := s.Pos
			for j < end && !IsSpace(s.Line[j]) && s.Line[j] != '$' {
				j++
			}
			s.EmitTo(Function, j)
		}
	}
}

// text lexes the rest of a line outside of recipes: variable references
// and a comment among text.
func (m *Makefile) text(s *Scanner) {
	for !s.Done() {
		i := strings.IndexAny(s.Rest(), "$#")
		if i < 0 {
			s.EmitRest(Text)
			return
		}
		s.Emit(Text, i)
		if s.Peek(0) == '#' {
			s.EmitRest(Comment)
			return
		}
		makeRef(s)
	}
}

// makeRef lexes the variable reference at the scanner's position, such as
// $(CC), ${SRC}, or $@, as a Variable.
func makeRef(s *Scanner) {
	switch s.Peek(1) {
	case '(':
		s.Emit(Variable, matchingEnd(s.Rest(), 1, '(', ')'))
	case '{':
		s.Emit(Variable, matchingEnd(s.Rest(), 1, '{', '}'))
	case 0:
		s.Emit(Text, 1)
	case '$':
		s.Emit(Variable, IdentEnd(s.Line, s.Pos+2, "")-s.Pos)
	default:
		s.Emit(Variable, 1+runeLen(s.Line, s.Pos+1))
	}
}

// recipe lexes a line of a recipe, starting in the shell State st: the
// shell's words between make's variable references. It returns the tokens
// and the State for the next line.
func (m *Makefile) recipe(s Scanner, st State) ([]Token, State) {
	s.Space()
	for s.Peek(0) == '@' || s.Peek(0) == '-' || s.Peek(0) == '+' {
		s.Emit(Operator, 1)
	}
	for !s.Done() {
		i := strings.IndexByte(s.Rest(), '$')
		if i < 0 {
			i = len(s.Rest())
		}
		if i > 0 {
			var toks []Token
			toks, st = m.Shell.AppendLine(s.Toks, s.Line[s.Pos:s.Pos+i], st)
			s.Toks, s.Pos = toks, s.Pos+i
			s.Start = len(s.Toks)
		}
		if !s.Done() {
			makeRef(&s)
			s.Start = len(s.Toks)
		}
	}
	// Each line of a recipe is a command of its own, unless a backslash
	// continues it.
	if !strings.HasSuffix(s.Line, `\`) {
		return s.Toks, ""
	}
	return s.Toks, st.Push(Frame{Mode: modeEmbed})
}
