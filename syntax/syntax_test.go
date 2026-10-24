package syntax

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

// tok returns a Token for tables.
func tok(k Kind, text string) Token { return Token{Kind: k, Text: text} }

// kindCase is a source in a language and the kinds of texts in it.
type kindCase struct {
	// name names the case's subtest.
	name string
	// src is the source, of one or more lines.
	src string
	// want holds texts of src, in the order they appear in it, each with
	// the Kind every byte of it must have, whichever tokens hold it.
	want []Token
	// open reports that src ends inside a construct, such as an unclosed
	// comment, so that its last line ends in a State that is not the zero
	// State.
	open bool
	// bug, if not empty, describes a known bug that keeps the lexer from
	// lexing src as want says; the case is skipped until it is fixed.
	bug string
}

// languageTests holds the cases of each language, by its name. Every
// language that Lookup finds has cases.
var languageTests = map[string][]kindCase{
	"bash":       bashTests,
	"c":          cTests,
	"cpp":        cppTests,
	"csharp":     csharpTests,
	"css":        cssTests,
	"dart":       dartTests,
	"diff":       diffTests,
	"dockerfile": dockerfileTests,
	"go":         goTests,
	"graphql":    graphqlTests,
	"hcl":        hclTests,
	"html":       htmlTests,
	"ini":        iniTests,
	"java":       javaTests,
	"javascript": javascriptTests,
	"json":       jsonTests,
	"kotlin":     kotlinTests,
	"lua":        luaTests,
	"makefile":   makefileTests,
	"markdown":   markdownTests,
	"php":        phpTests,
	"powershell": powershellTests,
	"protobuf":   protobufTests,
	"python":     pythonTests,
	"ruby":       rubyTests,
	"rust":       rustTests,
	"scala":      scalaTests,
	"scss":       scssTests,
	"sql":        sqlTests,
	"svelte":     svelteTests,
	"swift":      swiftTests,
	"toml":       tomlTests,
	"tsx":        tsxTests,
	"typescript": typescriptTests,
	"vue":        vueTests,
	"xml":        xmlTests,
	"yaml":       yamlTests,
	"zig":        zigTests,
}

// testLanguage tests the language named name: that Lookup finds it by each
// of names, and that it lexes each case as the case wants.
func testLanguage(t *testing.T, name string, names []string, cases []kindCase) {
	t.Helper()
	l := Lookup(name)
	if l == nil || l.Name != name {
		t.Fatalf("Lookup(%q) = %v", name, l)
	}
	for _, n := range names {
		if got := Lookup(n); got != l {
			t.Errorf("Lookup(%q) is not %s", n, name)
		}
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.bug != "" {
				t.Skip("known bug: " + c.bug)
			}
			kinds, end := byteKinds(l, c.src)
			at := 0
			for _, w := range c.want {
				i := strings.Index(c.src[at:], w.Text)
				if i < 0 {
					t.Fatalf("%q is not in the source after offset %d\n%s", w.Text, at, describe(l, c.src))
				}
				i += at
				for j := i; j < i+len(w.Text); j++ {
					if kinds[j] != w.Kind {
						t.Fatalf("%q at offset %d is %v at byte %d, want %v\n%s", w.Text, i, kinds[j], j-i, w.Kind, describe(l, c.src))
					}
				}
				at = i + len(w.Text)
			}
			if (end != "") != c.open {
				t.Errorf("the source ends in State %q, want open = %v\n%s", end, c.open, describe(l, c.src))
			}
		})
	}
}

// byteKinds returns the Kind of each byte of src, which l lexes, newlines
// being Text, and the State its last line ends in.
func byteKinds(l *Language, src string) ([]Kind, State) {
	kinds := make([]Kind, 0, len(src))
	var (
		st   State
		toks []Token
	)
	for i, line := range strings.Split(src, "\n") {
		if i > 0 {
			kinds = append(kinds, Text)
		}
		toks, st = l.AppendLine(toks[:0], line, st)
		for _, tk := range toks {
			for range len(tk.Text) {
				kinds = append(kinds, tk.Kind)
			}
		}
	}
	return kinds, st
}

// describe returns the tokens of each line of src, which l lexes, for a
// failure's message.
func describe(l *Language, src string) string {
	var b strings.Builder
	var (
		st   State
		toks []Token
	)
	for i, line := range strings.Split(src, "\n") {
		toks, st = l.AppendLine(toks[:0], line, st)
		fmt.Fprintf(&b, "%3d |", i+1)
		for _, tk := range toks {
			fmt.Fprintf(&b, " %v(%q)", tk.Kind, tk.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// stress holds sources that stress lexers with constructs left open.
var stress = []string{
	"", "\n", "\n\n", "\"", "'", "`", "/*", "*/", "<!--", "${", "#{", "{{", "}}", "<", "</",
	"<a", "<a b=\"", "```", "~~~", "\\", "r#\"", "R\"x(", "[==[", "--[[", "<<EOF", "@\"",
	"$'", "#!", "/re", "é ü 中文 😀", "\t\r\x00\xff", "a\\\nb", "'''\"\"\"", "{%", "\"\\",
	"$((", "<script>", "<style>", "a {", "key: |", "[[", "{{#if", "=begin", "<<<EOT", "@'",
}

// corpus returns the sources of every language's cases and the stress
// sources.
func corpus() []string {
	srcs := slices.Clone(stress)
	for _, name := range slices.Sorted(maps.Keys(languageTests)) {
		for _, c := range languageTests[name] {
			srcs = append(srcs, c.src)
		}
	}
	return srcs
}

func TestEveryLanguageHasTests(t *testing.T) {
	for l := range Languages() {
		if len(languageTests[l.Name]) == 0 {
			t.Errorf("%s has no test cases", l.Name)
		}
	}
	for name := range languageTests {
		if l := Lookup(name); l == nil || l.Name != name {
			t.Errorf("test cases for %q, which Lookup does not find by its name", name)
		}
	}
}

func TestTokensConcatenate(t *testing.T) {
	// Every language lexes every source, its own or not, into tokens that
	// are not empty and that concatenate to the source.
	srcs := corpus()
	for l := range Languages() {
		for _, src := range srcs {
			var b strings.Builder
			for tk := range l.Tokens(src) {
				if tk.Text == "" {
					t.Errorf("%s: empty %v token lexing %q", l.Name, tk.Kind, src)
				}
				if tk.Kind >= NumKinds {
					t.Errorf("%s: kind %v out of range lexing %q", l.Name, tk.Kind, src)
				}
				b.WriteString(tk.Text)
			}
			if b.String() != src {
				t.Errorf("%s: tokens of %q concatenate to %q", l.Name, src, b.String())
			}
		}
	}
}

// appendLines appends to dst the tokens of lines, which l lexes starting in
// st, with a newline token between lines, and returns the extended slice
// and the State the lines end in.
func appendLines(dst []Token, l *Language, lines []string, st State) ([]Token, State) {
	for i, line := range lines {
		if i > 0 {
			dst = append(dst, tok(Text, "\n"))
		}
		dst, st = l.AppendLine(dst, line, st)
	}
	return dst, st
}

func TestResumeAfterAppend(t *testing.T) {
	// A document that grows, as one streamed in does, is lexed in parts: its
	// first lines, then, from the State they end in, the lines appended to
	// them, with other text lexed in between. Its tokens are those of
	// lexing it in one pass, so the State holds all that a lexer carries
	// from one line to the next.
	other := strings.Split(strings.Join(stress, "\n"), "\n")
	for l := range Languages() {
		srcs := slices.Clone(stress)
		for _, c := range languageTests[l.Name] {
			srcs = append(srcs, c.src)
		}
		for _, src := range srcs {
			lines := strings.Split(src, "\n")
			want := slices.Collect(l.Tokens(src))
			for cut := range len(lines) {
				if cut == 0 {
					continue
				}
				got, st := appendLines(nil, l, lines[:cut], "")
				appendLines(nil, l, other, "")
				got, _ = appendLines(append(got, tok(Text, "\n")), l, lines[cut:], st)
				if !slices.Equal(got, want) {
					t.Errorf("%s: %q resumed at line %d lexes to\n%v\nin parts, and to\n%v\nin one pass", l.Name, src, cut, got, want)
				}
			}
		}
	}
}

func TestCLikeIgnoresIncompleteQuotes(t *testing.T) {
	c := &CLike{Quotes: []Quote{{Open: `"`}, {Close: `'`}}}
	toks, _ := c.AppendLine(nil, `a "b" 'c'`, "")
	var b strings.Builder
	for _, tk := range toks {
		if tk.Kind == String {
			t.Errorf("a quote without both delimiters opened: %v", toks)
		}
		b.WriteString(tk.Text)
	}
	if b.String() != `a "b" 'c'` {
		t.Errorf("tokens concatenate to %q", b.String())
	}
}

func TestAppendLineKeepsDst(t *testing.T) {
	dst := []Token{tok(Keyword, "kept")}
	for l := range Languages() {
		got, _ := l.AppendLine(slices.Clone(dst), "x = 1", "")
		if len(got) == 0 || got[0] != dst[0] {
			t.Errorf("%s: AppendLine changed dst: %v", l.Name, got)
		}
	}
}

func TestLookup(t *testing.T) {
	// The names of each language are tested with its cases; these test how
	// Lookup compares and splits names.
	tests := []struct{ name, want string }{
		{"Go", "go"}, {"GO", "go"}, {" golang ", "go"}, {"main.go", "go"}, {"cmd/x/main.go", "go"},
		{`C:\src\main.go`, "go"}, {"src/GNUmakefile", "makefile"}, {"app/Dockerfile", "dockerfile"},
		{"README.MD", "markdown"}, {"a.b.c.json", "json"}, {"text", ""}, {"plaintext", ""}, {"", ""},
		{"nonexistent", ""}, {"a.unknown", ""}, {".", ""}, {"/", ""},
	}
	for _, tt := range tests {
		got := ""
		if l := Lookup(tt.name); l != nil {
			got = l.Name
		}
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLanguageNamesUnique(t *testing.T) {
	seen := map[string]string{}
	for l := range Languages() {
		names := append([]string{l.Name}, l.Aliases...)
		for _, f := range l.Filenames {
			names = append(names, strings.ToLower(f))
		}
		for _, n := range names {
			if other, ok := seen[n]; ok && other != l.Name {
				t.Errorf("name %q belongs to both %s and %s", n, other, l.Name)
			}
			seen[n] = l.Name
			if n != strings.ToLower(n) {
				t.Errorf("%s: name %q is not lower case", l.Name, n)
			}
		}
	}
}

func TestStatePushTop(t *testing.T) {
	frames := []Frame{
		{Mode: 1, N: 2},
		{Mode: 3, N: 4, Delim: "EOF"},
		{Mode: 5, Delim: strings.Repeat("x", 300)},
	}
	var st State
	for _, f := range frames {
		st = st.Push(f)
	}
	got := AppendFrames(nil, st)
	want := Frames{frames[0], frames[1], {Mode: 5, Delim: strings.Repeat("x", 255)}}
	if !slices.Equal(got, want) {
		t.Errorf("AppendFrames = %v, want %v", got, want)
	}
	if got.State() != st {
		t.Errorf("Frames.State() does not round-trip")
	}
	for _, bad := range []State{"", "a", "ab", "\xff\x00\x01"} {
		if _, _, ok := bad.Top(); ok {
			t.Errorf("State(%q).Top() reports a frame", bad)
		}
	}
}

func TestKindString(t *testing.T) {
	if got := Keyword.String(); got != "Keyword" {
		t.Errorf("Keyword.String() = %q", got)
	}
	if got := Kind(200).String(); got != "Kind(200)" {
		t.Errorf("Kind(200).String() = %q", got)
	}
	for k := range NumKinds {
		if k.String() == "" {
			t.Errorf("Kind %d has no name", k)
		}
	}
}

func FuzzLexers(f *testing.F) {
	for _, src := range corpus() {
		f.Add(src)
	}
	f.Fuzz(func(t *testing.T, src string) {
		for l := range Languages() {
			var st State
			var toks []Token
			for line := range strings.SplitSeq(src, "\n") {
				toks, st = l.AppendLine(toks[:0], line, st)
				var b strings.Builder
				for _, tk := range toks {
					b.WriteString(tk.Text)
				}
				if b.String() != line {
					t.Fatalf("%s: tokens of %q concatenate to %q", l.Name, line, b.String())
				}
			}
		}
	})
}

func BenchmarkTokens(b *testing.B) {
	src, err := os.ReadFile("clike.go")
	if err != nil {
		b.Fatal(err)
	}
	l := Lookup("go")
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		for range l.Tokens(string(src)) {
		}
	}
}
