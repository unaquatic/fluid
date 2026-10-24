package syntax

import "testing"

func TestPython(t *testing.T) {
	testLanguage(t, "python", []string{"py", "py3", "python3", "pyi", "BUILD", "BUILD.bazel", "main.py"}, pythonTests)
}

var pythonTests = []kindCase{
	{name: "imports", src: "import os\nfrom typing import List as L\nfrom . import sibling\nfrom .pkg import mod", want: []Token{
		tok(Keyword, "import"), tok(Variable, "os"), tok(Keyword, "from"), tok(Variable, "typing"), tok(Keyword, "import"),
		tok(Variable, "List"), tok(Keyword, "as"), tok(Keyword, "from"), tok(Punctuation, "."), tok(Keyword, "import"),
		tok(Keyword, "from"), tok(Property, "pkg"), tok(Keyword, "import"),
	}},
	{name: "control flow", src: "for x in xs:\n    if x is not None and x or not y:\n        pass\n    elif x:\n        break\n    else:\n        continue\nwhile True:\n    return\ntry:\n    raise ValueError from e\nexcept E as err:\n    del err\nfinally:\n    assert ok", want: []Token{
		tok(Keyword, "for"), tok(Keyword, "in"), tok(Keyword, "if"), tok(Keyword, "is"), tok(Keyword, "not"),
		tok(Primitive, "None"), tok(Keyword, "and"), tok(Keyword, "or"), tok(Keyword, "not"), tok(Keyword, "pass"),
		tok(Keyword, "elif"), tok(Keyword, "break"), tok(Keyword, "else"), tok(Keyword, "continue"), tok(Keyword, "while"),
		tok(Primitive, "True"), tok(Keyword, "return"), tok(Keyword, "try"), tok(Keyword, "raise"), tok(Builtin, "ValueError"),
		tok(Keyword, "from"), tok(Keyword, "except"), tok(Keyword, "as"), tok(Keyword, "del"), tok(Keyword, "finally"),
		tok(Keyword, "assert"),
	}},
	{name: "definitions", src: "def f(a, *args, **kw) -> int:\n    return a\nclass Point(Base, metaclass=Meta):\n    pass\nasync def g():\n    await h()\n    yield\nlam = lambda a: a", want: []Token{
		tok(Keyword, "def"), tok(Function, "f"), tok(Variable, "a"), tok(Operator, "*"), tok(Variable, "args"),
		tok(Operator, "**"), tok(Variable, "kw"), tok(Operator, "->"), tok(Builtin, "int"), tok(Keyword, "class"),
		tok(Type, "Point"), tok(Variable, "Base"), tok(Variable, "metaclass"), tok(Operator, "="), tok(Variable, "Meta"),
		tok(Keyword, "async"), tok(Keyword, "def"), tok(Function, "g"), tok(Keyword, "await"), tok(Function, "h"),
		tok(Keyword, "yield"), tok(Keyword, "lambda"),
	}},
	{name: "scopes and context managers", src: "with open(p) as f, lock:\n    global g\n    nonlocal n", want: []Token{
		tok(Keyword, "with"), tok(Builtin, "open"), tok(Keyword, "as"), tok(Keyword, "global"), tok(Keyword, "nonlocal"),
	}},
	{name: "builtins and primitives", src: "print(len(xs), isinstance(x, int), sorted(d), None, True, False, self, cls, Exception)", want: []Token{
		tok(Builtin, "print"), tok(Builtin, "len"), tok(Builtin, "isinstance"), tok(Builtin, "int"), tok(Builtin, "sorted"),
		tok(Primitive, "None"), tok(Primitive, "True"), tok(Primitive, "False"), tok(Builtin, "self"), tok(Builtin, "cls"),
		tok(Builtin, "Exception"),
	}},
	{name: "member names are not builtins", src: "self.type = obj.id\nx.format(); os.path.join(a, b)", want: []Token{
		tok(Builtin, "self"), tok(Property, "type"), tok(Variable, "obj"), tok(Property, "id"), tok(Variable, "x"),
		tok(Function, "format"), tok(Variable, "os"), tok(Property, "path"), tok(Function, "join"),
	}},
	{name: "soft keywords are names", src: "match = re.match(p, s)\ncase = 2", want: []Token{
		tok(Variable, "match"), tok(Variable, "re"), tok(Function, "match"), tok(Variable, "case"),
	}},
	{name: "decorators and matrix multiplication", src: "@property\n@app.route('/x', methods=['GET'])\ndef v(): pass\nm = a @ b", want: []Token{
		tok(Decorator, "@property"), tok(Decorator, "@app.route"), tok(String, "'/x'"), tok(Variable, "methods"),
		tok(String, "'GET'"), tok(Function, "v"), tok(Keyword, "pass"), tok(Variable, "m"), tok(Variable, "a"), tok(Operator, "@"), tok(Variable, "b"),
	}},
	{name: "strings and prefixes", src: "s = 'a\\'b' + \"c\\n\" + r'\\d+' + b\"\\x00\" + u\"x\" + rb'\\w' + Rb'y'", want: []Token{
		tok(String, "'a"), tok(StringEscape, `\'`), tok(String, "b'"), tok(String, `"c`), tok(StringEscape, `\n`),
		tok(String, `"`), tok(String, `r'\d+'`), tok(String, `b"`), tok(StringEscape, `\x`), tok(String, `00"`),
		tok(String, `u"x"`), tok(String, `rb'\w'`), tok(String, "Rb'y'"),
	}},
	{name: "f-strings", src: "f\"{x!r:>10} {{lit}} {d['k']}\"\nrf\"\\d{n}\"", want: []Token{
		tok(String, `f"`), tok(StringEscape, "{"), tok(Variable, "x"), tok(Operator, "!"), tok(Variable, "r"),
		tok(Punctuation, ":"), tok(Operator, ">"), tok(Number, "10"), tok(StringEscape, "}"), tok(String, " {{lit}} "),
		tok(StringEscape, "{"), tok(Variable, "d"), tok(String, "'k'"), tok(StringEscape, "}"), tok(String, `"`),
		tok(String, `rf"\d`), tok(StringEscape, "{"), tok(Variable, "n"), tok(StringEscape, "}"), tok(String, `"`),
	}},
	{name: "triple-quoted strings", src: "\"\"\"Doc\nmore\n\"\"\"\nx = '''a\nb'''\ny = f\"\"\"\n{v}\n\"\"\"", want: []Token{
		tok(String, `"""Doc`), tok(String, "more"), tok(String, `"""`), tok(Variable, "x"), tok(String, "'''a"),
		tok(String, "b'''"), tok(Variable, "y"), tok(String, `f"""`), tok(StringEscape, "{"), tok(Variable, "v"),
		tok(StringEscape, "}"), tok(String, `"""`),
	}},
	{name: "string continued by a backslash", src: "s = 'abc\\\ndef'\nx = 1", want: []Token{
		tok(String, "'abc"), tok(StringEscape, `\`), tok(String, "def'"), tok(Variable, "x"),
	}},
	{name: "string ends with its line", src: "s = 'abc\nx = 1", want: []Token{
		tok(String, "'abc"), tok(Variable, "x"),
	}},
	{name: "numbers", src: "n = [0, 42, 0x1F, 0o17, 0b11, 1_000, 3.14, 1e10, 2.5e-3, 3j, .5]", want: []Token{
		tok(Number, "0"), tok(Number, "42"), tok(Number, "0x1F"), tok(Number, "0o17"), tok(Number, "0b11"),
		tok(Number, "1_000"), tok(Number, "3.14"), tok(Number, "1e10"), tok(Number, "2.5e-3"), tok(Number, "3j"),
		tok(Number, ".5"),
	}},
	{name: "operators", src: "a //= 2; b **= 3; (c := d); e != f; g <<= 1; h ^= ~i", want: []Token{
		tok(Operator, "//="), tok(Operator, "**="), tok(Operator, ":="), tok(Operator, "!="), tok(Operator, "<<="),
		tok(Operator, "^="), tok(Operator, "~"),
	}},
	{name: "comments", src: "# comment\nx = 1  # trailing\ns = '#not a comment'", want: []Token{
		tok(Comment, "# comment"), tok(Variable, "x"), tok(Comment, "# trailing"), tok(String, "'#not a comment'"),
	}},
	{name: "constants", src: "MAX_SIZE = 10\nDEBUG = False\nX = 1", want: []Token{
		tok(Constant, "MAX_SIZE"), tok(Constant, "DEBUG"), tok(Primitive, "False"), tok(Variable, "X"),
	}},
	{name: "unclosed docstring", src: "\"\"\"never closed\nstill", open: true, want: []Token{
		tok(String, `"""never closed`), tok(String, "still"),
	}},
}
