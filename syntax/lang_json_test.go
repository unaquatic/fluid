package syntax

import "testing"

func TestJSON(t *testing.T) {
	testLanguage(t, "json", []string{"jsonc", "json5", "jsonl", "ndjson", "package.json", ".babelrc", "flake.lock"}, jsonTests)
}

var jsonTests = []kindCase{
	{name: "objects", src: `{"name": "x", "version": "1.0", "private": true, "count": 3, "none": null}`, want: []Token{
		tok(Punctuation, "{"), tok(Property, `"name"`), tok(Punctuation, ":"), tok(String, `"x"`), tok(Punctuation, ","),
		tok(Property, `"version"`), tok(String, `"1.0"`), tok(Property, `"private"`), tok(Primitive, "true"),
		tok(Property, `"count"`), tok(Number, "3"), tok(Property, `"none"`), tok(Primitive, "null"), tok(Punctuation, "}"),
	}},
	{name: "nested across lines", src: "{\n  \"a\": [1, 2.5, -3e2, null],\n  \"b\": {\"c\": false}\n}", want: []Token{
		tok(Property, `"a"`), tok(Punctuation, "["), tok(Number, "1"), tok(Number, "2.5"), tok(Number, "-3e2"),
		tok(Primitive, "null"), tok(Punctuation, "]"), tok(Property, `"b"`), tok(Property, `"c"`), tok(Primitive, "false"),
	}},
	{name: "escapes", src: `{"s": "line\n\"quoted\" \u00e9", "a\"b": 1}`, want: []Token{
		tok(Property, `"s"`), tok(String, `"line\n\"quoted\" \u00e9"`), tok(Property, `"a\"b"`), tok(Number, "1"),
	}},
	{name: "colons inside strings", src: `["a:b", "c" , "d"]`, want: []Token{
		tok(String, `"a:b"`), tok(String, `"c"`), tok(String, `"d"`),
	}},
	{name: "keys spaced from colons", src: `{"k" : "v"}`, want: []Token{
		tok(Property, `"k"`), tok(Punctuation, ":"), tok(String, `"v"`),
	}},
	{name: "comments", src: "{\n  // line\n  \"a\": 1, /* block\n  comment */ \"b\": 2\n}", want: []Token{
		tok(Comment, "// line"), tok(Property, `"a"`), tok(Comment, "/* block"), tok(Comment, "  comment */"),
		tok(Property, `"b"`), tok(Number, "2"),
	}},
	{name: "json5", src: "{unquoted: 'single', trailing: +1, hex: 0x1F, inf: Infinity, nan: NaN, frac: .5,}", want: []Token{
		tok(Property, "unquoted"), tok(String, "'single'"), tok(Property, "trailing"), tok(Number, "+1"),
		tok(Property, "hex"), tok(Number, "0x1F"), tok(Property, "inf"), tok(Primitive, "Infinity"), tok(Property, "nan"),
		tok(Primitive, "NaN"), tok(Number, ".5"),
	}},
	{name: "json lines", src: "{\"a\": 1}\n{\"a\": 2}", want: []Token{
		tok(Property, `"a"`), tok(Number, "1"), tok(Property, `"a"`), tok(Number, "2"),
	}},
	{name: "unclosed block comment", src: "/* open\nstill", open: true, want: []Token{
		tok(Comment, "/* open"), tok(Comment, "still"),
	}},
}
