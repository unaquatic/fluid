package syntax

import "testing"

func TestGo(t *testing.T) {
	testLanguage(t, "go", []string{"golang", "main.go"}, goTests)
}

var goTests = []kindCase{
	{name: "keywords", src: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfor i := range 10 {\n\t\tif i > 0 {\n\t\t\tcontinue\n\t\t}\n\t\tdefer f()\n\t\tgo g()\n\t}\n}", want: []Token{
		tok(Keyword, "package"), tok(Variable, "main"), tok(Keyword, "import"), tok(String, `"fmt"`),
		tok(Keyword, "func"), tok(Function, "main"), tok(Keyword, "for"), tok(Variable, "i"),
		tok(Operator, ":="), tok(Keyword, "range"), tok(Number, "10"), tok(Keyword, "if"),
		tok(Keyword, "continue"), tok(Keyword, "defer"), tok(Function, "f"), tok(Keyword, "go"), tok(Function, "g"),
	}},
	{name: "control flow", src: "switch x := v.(type) {\ncase int:\n\tfallthrough\ndefault:\n\tbreak\n}\nselect {\ncase <-ch:\n}\ngoto end\nreturn", want: []Token{
		tok(Keyword, "switch"), tok(Keyword, "type"), tok(Keyword, "case"), tok(BuiltinType, "int"),
		tok(Keyword, "fallthrough"), tok(Keyword, "default"), tok(Keyword, "break"), tok(Keyword, "select"),
		tok(Keyword, "case"), tok(Operator, "<-"), tok(Keyword, "goto"), tok(Variable, "end"), tok(Keyword, "return"),
	}},
	{name: "types and builtins", src: "var m map[string]int = make(map[string]int, len(s))\nvar e error\nvar a any\nch := make(chan<- rune, cap(b))\nx := append(xs, min(1, 2))\npanic(new(T))", want: []Token{
		tok(Keyword, "var"), tok(Variable, "m"), tok(Keyword, "map"), tok(Punctuation, "["), tok(BuiltinType, "string"),
		tok(BuiltinType, "int"), tok(BuiltinFunction, "make"), tok(BuiltinFunction, "len"), tok(BuiltinType, "error"), tok(BuiltinType, "any"),
		tok(Keyword, "chan"), tok(BuiltinType, "rune"), tok(BuiltinFunction, "cap"), tok(BuiltinFunction, "append"), tok(BuiltinFunction, "min"),
		tok(BuiltinFunction, "panic"), tok(BuiltinFunction, "new"),
	}},
	{name: "type declarations", src: "type Point struct{ X, Y int }\ntype Reader interface {\n\tRead(p []byte) (n int, err error)\n}\ntype ID = string", want: []Token{
		tok(Keyword, "type"), tok(Type, "Point"), tok(Keyword, "struct"), tok(Variable, "X"), tok(Variable, "Y"),
		tok(BuiltinType, "int"), tok(Keyword, "type"), tok(Type, "Reader"), tok(Keyword, "interface"), tok(Function, "Read"),
		tok(Variable, "p"), tok(BuiltinType, "byte"), tok(BuiltinType, "error"), tok(Type, "ID"), tok(BuiltinType, "string"),
	}},
	{name: "methods and member access", src: "func (p *Point) Dist(q Point) float64 {\n\treturn math.Sqrt(p.X*q.X + p.Y*q.Y)\n}", want: []Token{
		tok(Keyword, "func"), tok(Variable, "p"), tok(Operator, "*"), tok(Function, "Dist"), tok(Variable, "q"),
		tok(BuiltinType, "float64"), tok(Keyword, "return"), tok(Variable, "math"), tok(Punctuation, "."),
		tok(Function, "Sqrt"), tok(Variable, "p"), tok(Property, "X"), tok(Operator, "*"), tok(Variable, "q"),
		tok(Property, "X"), tok(Operator, "+"),
	}},
	{name: "member names are not keywords", src: "x.type; y.range; z.len(); w.string", want: []Token{
		tok(Property, "type"), tok(Property, "range"), tok(Function, "len"), tok(Property, "string"),
	}},
	{name: "generics", src: "func Map[T, U any](xs []T, f func(T) U) []U {\n\treturn nil\n}", want: []Token{
		tok(Function, "Map"), tok(Punctuation, "["), tok(Variable, "T"), tok(Variable, "U"), tok(BuiltinType, "any"),
		tok(Variable, "xs"), tok(Keyword, "func"), tok(Primitive, "nil"),
	}},
	{name: "primitives and iota", src: "const (\n\tA = iota\n\tB\n)\nok := true || false && p != nil", want: []Token{
		tok(Keyword, "const"), tok(Variable, "A"), tok(Constant, "iota"), tok(Primitive, "true"),
		tok(Operator, "||"), tok(Primitive, "false"), tok(Operator, "&&"), tok(Operator, "!="), tok(Primitive, "nil"),
	}},
	{name: "numbers", src: "n := []float64{42, 0x1F, 0b1010, 0o17, 1_000_000, 3.14, 1e-9, .5, 2i, 0x1p-2}", want: []Token{
		tok(Number, "42"), tok(Number, "0x1F"), tok(Number, "0b1010"), tok(Number, "0o17"), tok(Number, "1_000_000"),
		tok(Number, "3.14"), tok(Number, "1e-9"), tok(Number, ".5"), tok(Number, "2i"), tok(Number, "0x1p-2"),
	}},
	{name: "strings and runes", src: "s := \"tab\\tquote\\\"\" + `raw\\n` + string('x') + string('\\n')", want: []Token{
		tok(String, `"tab`), tok(StringEscape, `\t`), tok(String, "quote"), tok(StringEscape, `\"`), tok(String, `"`),
		tok(String, "`raw\\n`"), tok(BuiltinType, "string"), tok(String, "'x'"), tok(String, "'"), tok(StringEscape, `\n`),
		tok(String, "'"),
	}},
	{name: "raw string across lines", src: "q := `SELECT *\nFROM t`\nnext()", want: []Token{
		tok(String, "`SELECT *"), tok(String, "FROM t`"), tok(Function, "next"),
	}},
	{name: "interpreted string ends with its line", src: "s := \"abc\nx := 1", want: []Token{
		tok(String, `"abc`), tok(Variable, "x"), tok(Number, "1"),
	}},
	{name: "comments", src: "// line comment\n/* block\n   comment */ x := 1 // trailing", want: []Token{
		tok(Comment, "// line comment"), tok(Comment, "/* block"), tok(Comment, "   comment */"),
		tok(Variable, "x"), tok(Comment, "// trailing"),
	}},
	{name: "directives", src: "//go:build linux && amd64\n//go:generate stringer -type=Kind\n//line foo.go:10\n// go:not-a-directive", want: []Token{
		tok(Preproc, "//go:build linux && amd64"), tok(Preproc, "//go:generate stringer -type=Kind"),
		tok(Preproc, "//line foo.go:10"), tok(Comment, "// go:not-a-directive"),
	}},
	{name: "operators", src: "a <<= 2; b &^= c; ch <- v; x := <-ch; f(xs...); p := &q; *p++", want: []Token{
		tok(Operator, "<<="), tok(Operator, "&^="), tok(Operator, "<-"), tok(Operator, ":="), tok(Operator, "<-"),
		tok(Operator, "..."), tok(Operator, "&"), tok(Operator, "*"), tok(Operator, "++"),
	}},
	{name: "labels", src: "outer:\n\tfor {\n\t\tbreak outer\n\t}", want: []Token{
		tok(Variable, "outer"), tok(Punctuation, ":"), tok(Keyword, "for"), tok(Keyword, "break"), tok(Variable, "outer"),
	}},
	{name: "unclosed block comment", src: "/* unterminated\nstill a comment", open: true, want: []Token{
		tok(Comment, "/* unterminated"), tok(Comment, "still a comment"),
	}},
	{name: "unclosed raw string", src: "s := `open\nstill", open: true, want: []Token{
		tok(String, "`open"), tok(String, "still"),
	}},
}
