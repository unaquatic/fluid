package syntax

import "testing"

func TestYAML(t *testing.T) {
	testLanguage(t, "yaml", []string{"yml", "config.yaml", "ci.yml", ".clang-format"}, yamlTests)
}

var yamlTests = []kindCase{
	{name: "mappings and scalars", src: "name: app\nversion: 1.2.3\nport: 8080\nratio: 0.5\nenabled: true\nmode: off\nnothing: null\ntilde: ~\nquoted: \"a: b\"\nsingle: 'x'", want: []Token{
		tok(Property, "name"), tok(Punctuation, ":"), tok(Text, "app"), tok(Property, "version"), tok(Text, "1.2.3"),
		tok(Property, "port"), tok(Number, "8080"), tok(Property, "ratio"), tok(Number, "0.5"), tok(Property, "enabled"),
		tok(Primitive, "true"), tok(Property, "mode"), tok(Primitive, "off"), tok(Property, "nothing"),
		tok(Primitive, "null"), tok(Property, "tilde"), tok(Primitive, "~"), tok(Property, "quoted"),
		tok(String, `"a: b"`), tok(Property, "single"), tok(String, "'x'"),
	}},
	{name: "sequences and flow collections", src: "items:\n  - one\n  - two: 2\n  - [a, b]\n  - {k: v, n: 1}", want: []Token{
		tok(Property, "items"), tok(Punctuation, "-"), tok(Text, "one"), tok(Punctuation, "-"), tok(Property, "two"),
		tok(Number, "2"), tok(Punctuation, "["), tok(Text, "a"), tok(Punctuation, ","), tok(Text, "b"),
		tok(Punctuation, "]"), tok(Punctuation, "{"), tok(Property, "k"), tok(Text, "v"), tok(Property, "n"),
		tok(Number, "1"), tok(Punctuation, "}"),
	}},
	{name: "comments", src: "# top\nkey: value # trailing\nurl: http://x#frag", want: []Token{
		tok(Comment, "# top"), tok(Property, "key"), tok(Text, "value"), tok(Comment, "# trailing"), tok(Property, "url"),
		tok(Text, "http://x#frag"),
	}},
	{name: "block scalars", src: "script: |\n  echo one\n  echo two\nfolded: >-\n  long text\n\n  more\nafter: 1", want: []Token{
		tok(Property, "script"), tok(Punctuation, "|"), tok(String, "  echo one"), tok(String, "  echo two"),
		tok(Property, "folded"), tok(Punctuation, ">-"), tok(String, "  long text"), tok(String, "  more"),
		tok(Property, "after"), tok(Number, "1"),
	}},
	{name: "block scalars in sequences", src: "steps:\n  - run: |\n      make\n      make test\n  - name: x", want: []Token{
		tok(Property, "run"), tok(Punctuation, "|"), tok(String, "      make"), tok(String, "      make test"),
		tok(Property, "name"), tok(Text, "x"),
	}},
	{name: "block scalar headers with comments", src: "a: |+2 # keep\n    text\nb: 1", want: []Token{
		tok(Punctuation, "|+2"), tok(Comment, "# keep"), tok(String, "    text"), tok(Property, "b"),
	}},
	{name: "anchors aliases and tags", src: "base: &base\n  a: 1\nderived:\n  <<: *base\n  b: !!str 2\n  c: !custom val", want: []Token{
		tok(Property, "base"), tok(Variable, "&base"), tok(Property, "a"), tok(Property, "derived"), tok(Property, "<<"),
		tok(Variable, "*base"), tok(Property, "b"), tok(Type, "!!str"), tok(Property, "c"), tok(Type, "!custom"),
		tok(Text, "val"),
	}},
	{name: "documents", src: "---\na: 1\n...\n--- # second\nb: 2", want: []Token{
		tok(Punctuation, "---"), tok(Property, "a"), tok(Punctuation, "..."), tok(Punctuation, "---"),
		tok(Comment, "# second"), tok(Property, "b"),
	}},
	{name: "keys with spaces and quotes", src: "\"quoted key\": 1\nsimple key: 2\n'single': 3", want: []Token{
		tok(Property, `"quoted key"`), tok(Number, "1"), tok(Property, "simple key"), tok(Number, "2"),
		tok(Property, "'single'"), tok(Number, "3"),
	}},
	{name: "numbers", src: "a: -1\nb: +2.5\nc: 1e3\nd: 0x1F\nf: .5\ng: 8080/tcp", want: []Token{
		tok(Number, "-1"), tok(Number, "+2.5"), tok(Number, "1e3"), tok(Number, "0x1F"), tok(Number, ".5"),
		tok(Text, "8080/tcp"),
	}},
	{name: "scalars that start with digits", src: "e: 12abc\nmemory: 512Mi\nversion: 1.0a\nhex: 0x1Fg\n- 3rd", want: []Token{
		tok(Text, "12abc"), tok(Text, "512Mi"), tok(Text, "1.0a"), tok(Text, "0x1Fg"), tok(Text, "3rd"),
	}},
	{name: "unclosed block scalar", src: "text: |\n  still inside", open: true, want: []Token{
		tok(Punctuation, "|"), tok(String, "  still inside"),
	}},
}
