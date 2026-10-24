package syntax

import "testing"

func TestC(t *testing.T) {
	testLanguage(t, "c", []string{"h", "main.c", "include/x.h"}, cTests)
}

var cTests = []kindCase{
	{name: "preprocessor", src: "#include <stdio.h>\n#include \"local.h\"\n#define BUF 64\n#ifdef DEBUG\n#endif\n  #pragma once\n# define SPACED 1", want: []Token{
		tok(Preproc, "#include"), tok(String, "<stdio.h>"), tok(Preproc, "#include"), tok(String, `"local.h"`),
		tok(Preproc, "#define"), tok(Constant, "BUF"), tok(Number, "64"), tok(Preproc, "#ifdef"),
		tok(Constant, "DEBUG"), tok(Preproc, "#endif"), tok(Preproc, "#pragma"), tok(Variable, "once"),
		tok(Preproc, "# define"), tok(Constant, "SPACED"),
	}},
	{name: "function-like macros", src: "#define MAX(a, b) ((a) > (b) ? (a) : (b))", want: []Token{
		tok(Preproc, "#define"), tok(Function, "MAX"), tok(Variable, "a"), tok(Operator, ">"), tok(Operator, "?"),
		tok(Punctuation, ":"),
	}},
	{name: "declarations", src: "static const unsigned long n = sizeof(int);\ntypedef struct node {\n\tstruct node *next;\n} node_t;\nenum color { RED, GREEN };\nunion u { int i; float f; };\nextern void f(void);", want: []Token{
		tok(Keyword, "static"), tok(Keyword, "const"), tok(BuiltinType, "unsigned"), tok(BuiltinType, "long"), tok(Variable, "n"),
		tok(Keyword, "sizeof"), tok(BuiltinType, "int"), tok(Keyword, "typedef"), tok(Keyword, "struct"), tok(Type, "node"),
		tok(Keyword, "struct"), tok(Type, "node"), tok(Operator, "*"), tok(Variable, "next"), tok(Variable, "node_t"),
		tok(Keyword, "enum"), tok(Type, "color"), tok(Constant, "RED"), tok(Constant, "GREEN"), tok(Keyword, "union"),
		tok(Type, "u"), tok(BuiltinType, "float"), tok(Keyword, "extern"), tok(BuiltinType, "void"), tok(Function, "f"), tok(BuiltinType, "void"),
	}},
	{name: "standard types", src: "size_t s; int64_t i; uint8_t b; bool ok; FILE *fp; volatile register char c;", want: []Token{
		tok(BuiltinType, "size_t"), tok(BuiltinType, "int64_t"), tok(BuiltinType, "uint8_t"), tok(BuiltinType, "bool"), tok(BuiltinType, "FILE"),
		tok(Keyword, "volatile"), tok(Keyword, "register"), tok(BuiltinType, "char"),
	}},
	{name: "control flow", src: "for (int i = 0; i < n; i++) { if (x) continue; else break; }\nwhile (1) {} do {} while (0);\nswitch (c) { case 'a': default: goto end; }\nend: return 0;", want: []Token{
		tok(Keyword, "for"), tok(BuiltinType, "int"), tok(Operator, "<"), tok(Operator, "++"), tok(Keyword, "if"),
		tok(Keyword, "continue"), tok(Keyword, "else"), tok(Keyword, "break"), tok(Keyword, "while"), tok(Keyword, "do"),
		tok(Keyword, "while"), tok(Keyword, "switch"), tok(Keyword, "case"), tok(String, "'a'"), tok(Keyword, "default"),
		tok(Keyword, "goto"), tok(Variable, "end"), tok(Keyword, "return"),
	}},
	{name: "functions and calls", src: "int main(int argc, char **argv) {\n\tprintf(\"%d\\n\", argc);\n\treturn EXIT_SUCCESS;\n}", want: []Token{
		tok(BuiltinType, "int"), tok(Function, "main"), tok(Variable, "argc"), tok(Operator, "**"), tok(Variable, "argv"),
		tok(Function, "printf"), tok(String, `"%d`), tok(StringEscape, `\n`), tok(String, `"`), tok(Constant, "EXIT_SUCCESS"),
	}},
	{name: "primitives", src: "p = NULL; ok = true || false; q = nullptr;", want: []Token{
		tok(Primitive, "NULL"), tok(Primitive, "true"), tok(Primitive, "false"), tok(Primitive, "nullptr"),
	}},
	{name: "strings and characters", src: "char *s = \"a\\tb\\\"c\"; char c = '\\n'; char q = '\\''; wchar_t *w = L\"wide\"; char *u = u8\"utf\";", want: []Token{
		tok(String, `"a`), tok(StringEscape, `\t`), tok(String, "b"), tok(StringEscape, `\"`), tok(String, `c"`),
		tok(String, "'"), tok(StringEscape, `\n`), tok(String, "'"), tok(StringEscape, `\'`), tok(BuiltinType, "wchar_t"),
		tok(String, `L"wide"`), tok(String, `u8"utf"`),
	}},
	{name: "string continued by a backslash", src: "char *s = \"line one \\\ncontinued\";\nint x;", want: []Token{
		tok(String, `"line one `), tok(StringEscape, `\`), tok(String, `continued"`), tok(BuiltinType, "int"),
	}},
	{name: "comments", src: "// line\n/* block\n   more */ int x; /* inline */ int y;", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/* block"), tok(Comment, "   more */"), tok(BuiltinType, "int"),
		tok(Comment, "/* inline */"), tok(BuiltinType, "int"),
	}},
	{name: "numbers", src: "x = 42 + 0x1Fu + 017 + 3.14f + 1e-5 + 10UL + 0b101 + .5;", want: []Token{
		tok(Number, "42"), tok(Number, "0x1Fu"), tok(Number, "017"), tok(Number, "3.14f"), tok(Number, "1e-5"),
		tok(Number, "10UL"), tok(Number, "0b101"), tok(Number, ".5"),
	}},
	{name: "operators and members", src: "a->b; c.d; e <<= 2; ~f; g != h; i &= ~j; k = l ? m : n;", want: []Token{
		tok(Variable, "a"), tok(Operator, "->"), tok(Property, "b"), tok(Variable, "c"), tok(Punctuation, "."),
		tok(Property, "d"), tok(Operator, "<<="), tok(Operator, "~"), tok(Operator, "!="), tok(Operator, "&="),
		tok(Operator, "?"), tok(Punctuation, ":"),
	}},
	{name: "unclosed block comment", src: "/* open\nstill", open: true, want: []Token{
		tok(Comment, "/* open"), tok(Comment, "still"),
	}},
}
