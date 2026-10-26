package syntax

import "testing"

func TestTOML(t *testing.T) {
	testLanguage(t, "toml", []string{"Cargo.toml", "pyproject.toml", "Cargo.lock", "Pipfile", "poetry.lock"}, tomlTests)
}

var tomlTests = []kindCase{
	{name: "tables and keys", src: "[package]\nname = \"app\"\nversion = \"0.1.0\"\nedition = 2021\n\n[dependencies.serde]\nversion = \"1\"\n[[bin]]\nname = 'tool'", want: []Token{
		tok(Keyword, "[package]"), tok(Property, "name"), tok(Operator, "="), tok(String, `"app"`), tok(Property, "version"),
		tok(String, `"0.1.0"`), tok(Property, "edition"), tok(Number, "2021"), tok(Keyword, "[dependencies.serde]"),
		tok(Property, "version"), tok(Keyword, "[[bin]]"), tok(Property, "name"), tok(String, "'tool'"),
	}},
	{name: "values", src: "s = \"esc \\t\"\nl = 'C:\\path'\ni = 1_000\nh = 0xDEAD\nf = 3.14\ne = 6.02e23\nb = true\nd = 1979-05-27T07:32:00Z\nn = inf\nm = -nan", want: []Token{
		tok(Property, "s"), tok(String, `"esc \t"`), tok(Property, "l"), tok(String, `'C:\path'`), tok(Number, "1_000"),
		tok(Number, "0xDEAD"), tok(Number, "3.14"), tok(Number, "6.02e23"), tok(Primitive, "true"),
		tok(Number, "1979-05-27T07:32:00Z"), tok(Number, "inf"), tok(Number, "-nan"),
	}},
	{name: "arrays and inline tables", src: "f = [\"a\", \"b\"]\nn = [1, 2]\nt = { a = 1, b = [2, 3] }\nm = [[1, 2], [\"x\"]]", want: []Token{
		tok(Property, "f"), tok(String, `"a"`), tok(String, `"b"`), tok(Property, "n"), tok(Number, "2"),
		tok(Property, "t"), tok(Property, "a"), tok(Property, "b"), tok(Number, "2"), tok(Number, "3"),
		tok(Property, "m"), tok(Number, "1"), tok(Number, "2"), tok(String, `"x"`),
	}},
	{name: "arrays across lines", src: "deps = [\n  \"one\",\n  { name = \"two\", v = 2 },\n]\nnext = 1", want: []Token{
		tok(Property, "deps"), tok(String, `"one"`), tok(Property, "name"), tok(String, `"two"`), tok(Property, "v"),
		tok(Number, "2"), tok(Punctuation, "]"), tok(Property, "next"),
	}},
	{name: "multiline strings", src: "desc = \"\"\"\nline \\\"one\\\"\n\"\"\"\nraw = '''\nC:\\no\\escape\n'''\nafter = 1", want: []Token{
		tok(String, `"""`), tok(String, "line "), tok(StringEscape, `\"`), tok(String, "one"), tok(StringEscape, `\"`),
		tok(String, `"""`), tok(Property, "raw"), tok(String, "'''"), tok(String, `C:\no\escape`), tok(String, "'''"),
		tok(Property, "after"),
	}},
	{name: "dotted and quoted keys", src: "a.b.c = 1\n\"quoted key\" = 2\nsite.\"google.com\" = true", want: []Token{
		tok(Property, "a"), tok(Punctuation, "."), tok(Property, "b"), tok(Property, "c"), tok(Property, `"quoted key"`),
		tok(Property, "site"), tok(Property, `"google.com"`), tok(Primitive, "true"),
	}},
	{name: "comments", src: "# comment\nkey = \"value\" # trailing\n[table] # after header", want: []Token{
		tok(Comment, "# comment"), tok(Property, "key"), tok(Comment, "# trailing"), tok(Keyword, "[table]"),
		tok(Comment, "# after header"),
	}},
	{name: "unclosed multiline string", src: "s = \"\"\"\nopen", open: true, want: []Token{
		tok(String, `"""`), tok(String, "open"),
	}},
}
