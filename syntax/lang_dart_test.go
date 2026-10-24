package syntax

import "testing"

func TestDart(t *testing.T) {
	testLanguage(t, "dart", []string{"main.dart"}, dartTests)
}

var dartTests = []kindCase{
	{name: "declarations", src: "import 'package:flutter/material.dart';\nclass MyApp extends StatelessWidget {\n  const MyApp({super.key});\n  @override\n  Widget build(BuildContext context) => const Text('hi');\n}\nmixin M on Base {}\nenum E { a, b }\nextension X on String {}\ntypedef F = void Function(int);", want: []Token{
		tok(Keyword, "import"), tok(String, "'package:flutter/material.dart'"), tok(Keyword, "class"), tok(Type, "MyApp"),
		tok(Keyword, "extends"), tok(Type, "StatelessWidget"), tok(Keyword, "const"), tok(Function, "MyApp"),
		tok(Keyword, "super"), tok(Property, "key"), tok(Decorator, "@override"), tok(Type, "Widget"),
		tok(Function, "build"), tok(Type, "BuildContext"), tok(Operator, "=>"), tok(Keyword, "const"),
		tok(Function, "Text"), tok(Keyword, "mixin"), tok(Type, "M"), tok(Keyword, "on"), tok(Type, "Base"),
		tok(Keyword, "enum"), tok(Type, "E"), tok(Keyword, "extension"), tok(Type, "X"), tok(Keyword, "on"),
		tok(Type, "String"), tok(Keyword, "typedef"), tok(Type, "F"), tok(BuiltinType, "void"),
	}},
	{name: "keywords", src: "final x = 1; var y; late String z; dynamic d;\nFuture<void> f({required int r}) async { await g(); }\nStream<int> s() async* { yield 1; }\nif (a is! B) {} else {}\nfor (var i in xs) {}\nswitch (v) { case 1: break; default: }", want: []Token{
		tok(Keyword, "final"), tok(Keyword, "var"), tok(Keyword, "late"), tok(Type, "String"), tok(Keyword, "dynamic"),
		tok(Type, "Future"), tok(BuiltinType, "void"), tok(Keyword, "required"), tok(BuiltinType, "int"), tok(Keyword, "async"),
		tok(Keyword, "await"), tok(Type, "Stream"), tok(Keyword, "async"), tok(Operator, "*"), tok(Keyword, "yield"),
		tok(Keyword, "if"), tok(Keyword, "is"), tok(Operator, "!"), tok(Type, "B"), tok(Keyword, "else"),
		tok(Keyword, "for"), tok(Keyword, "in"), tok(Keyword, "switch"), tok(Keyword, "case"), tok(Keyword, "break"),
		tok(Keyword, "default"),
	}},
	{name: "string interpolation", src: "var s = 'Hi $name, ${a.b}!';\nvar t = \"esc\\n\";\nvar r = r'raw $x \\n';", want: []Token{
		tok(String, "'Hi "), tok(StringEscape, "$name"), tok(String, ", "), tok(StringEscape, "${"), tok(Variable, "a"),
		tok(Property, "b"), tok(StringEscape, "}"), tok(String, "!'"), tok(String, `"esc`), tok(StringEscape, `\n`),
		tok(String, `r'raw $x \n'`),
	}},
	{name: "multiline strings", src: "var m = '''\nline $x\n''';\nvar p = r\"\"\"\na $b\n\"\"\";\nvar n = 1;", want: []Token{
		tok(String, "'''"), tok(String, "line "), tok(StringEscape, "$x"), tok(String, "'''"), tok(String, `r"""`),
		tok(String, "a $b"), tok(String, `"""`), tok(Keyword, "var"),
	}},
	{name: "types and primitives", src: "bool b = true; int? n = null; double d = 1.5; num x = 0x1F; List<String> l = [];", want: []Token{
		tok(BuiltinType, "bool"), tok(Primitive, "true"), tok(BuiltinType, "int"), tok(Operator, "?"), tok(Primitive, "null"),
		tok(BuiltinType, "double"), tok(Number, "1.5"), tok(BuiltinType, "num"), tok(Number, "0x1F"), tok(Type, "List"),
	}},
	{name: "cascades and null awareness", src: "obj..a = 1..b(); x ??= y; z?.w;", want: []Token{
		tok(Operator, ".."), tok(Operator, "??="), tok(Operator, "?"), tok(Property, "w"),
	}},
	{name: "comments", src: "/* a /* b */ c */ var v;\n/// doc\n// line", want: []Token{
		tok(Comment, "/* a /* b */ c */"), tok(Keyword, "var"), tok(Comment, "/// doc"), tok(Comment, "// line"),
	}},
}
