package syntax

import "testing"

func TestJava(t *testing.T) {
	testLanguage(t, "java", []string{"Main.java"}, javaTests)
}

var javaTests = []kindCase{
	{name: "declarations", src: "package com.example;\nimport java.util.List;\npublic final class Main extends Base implements Runnable {}\ninterface I {}\nenum E { A }\nrecord R(int x) {}", want: []Token{
		tok(Keyword, "package"), tok(Variable, "com"), tok(Property, "example"), tok(Keyword, "import"),
		tok(Variable, "java"), tok(Property, "util"), tok(Type, "List"), tok(Keyword, "public"), tok(Keyword, "final"),
		tok(Keyword, "class"), tok(Type, "Main"), tok(Keyword, "extends"), tok(Type, "Base"), tok(Keyword, "implements"),
		tok(Type, "Runnable"), tok(Keyword, "interface"), tok(Type, "I"), tok(Keyword, "enum"), tok(Type, "E"),
		tok(Type, "A"), tok(Keyword, "record"), tok(Type, "R"), tok(BuiltinType, "int"),
	}},
	{name: "methods and annotations", src: "@Override\npublic static void main(String[] args) throws Exception {\n  System.out.println(args.length);\n}\n@SuppressWarnings(\"unchecked\")", want: []Token{
		tok(Decorator, "@Override"), tok(Keyword, "public"), tok(Keyword, "static"), tok(BuiltinType, "void"),
		tok(Function, "main"), tok(Type, "String"), tok(Variable, "args"), tok(Keyword, "throws"), tok(Type, "Exception"),
		tok(Type, "System"), tok(Property, "out"), tok(Function, "println"), tok(Variable, "args"), tok(Property, "length"),
		tok(Decorator, "@SuppressWarnings"), tok(String, `"unchecked"`),
	}},
	{name: "statements", src: "for (int i = 0; i < n; i++) { if (o instanceof String s) break; }\ntry (var r = open()) {} catch (IOException e) {} finally {}\nsynchronized (lock) { assert x; }\nswitch (v) { case 1 -> { yield 2; } default -> {} }\nthrow new IllegalStateException();", want: []Token{
		tok(Keyword, "for"), tok(BuiltinType, "int"), tok(Keyword, "if"), tok(Keyword, "instanceof"), tok(Type, "String"),
		tok(Keyword, "break"), tok(Keyword, "try"), tok(Keyword, "var"), tok(Function, "open"), tok(Keyword, "catch"),
		tok(Type, "IOException"), tok(Keyword, "finally"), tok(Keyword, "synchronized"), tok(Keyword, "assert"),
		tok(Keyword, "switch"), tok(Keyword, "case"), tok(Operator, "->"), tok(Keyword, "yield"), tok(Keyword, "default"),
		tok(Keyword, "throw"), tok(Keyword, "new"),
	}},
	{name: "primitive types and literals", src: "boolean b = true; char c = 'c'; double d = 1.0; Object o = null; long l = 1L; float f = 1.5f; byte y; short s;", want: []Token{
		tok(BuiltinType, "boolean"), tok(Primitive, "true"), tok(BuiltinType, "char"), tok(String, "'c'"), tok(BuiltinType, "double"),
		tok(Number, "1.0"), tok(Type, "Object"), tok(Primitive, "null"), tok(BuiltinType, "long"), tok(Number, "1L"),
		tok(BuiltinType, "float"), tok(Number, "1.5f"), tok(BuiltinType, "byte"), tok(BuiltinType, "short"),
	}},
	{name: "text blocks", src: "String s = \"\"\"\n    multi \"line\"\n    \\t text\"\"\";\nint x;", want: []Token{
		tok(String, `"""`), tok(String, `    multi "line"`), tok(StringEscape, `\t`), tok(String, ` text"""`), tok(BuiltinType, "int"),
	}},
	{name: "constants and member keywords", src: "static final int MAX_SIZE = 10;\nClass<?> c = Foo.class;\nOuter.this.run();\nsuper.close();", want: []Token{
		tok(Constant, "MAX_SIZE"), tok(Type, "Class"), tok(Keyword, "class"), tok(Keyword, "this"), tok(Function, "run"),
		tok(Keyword, "super"), tok(Function, "close"),
	}},
	{name: "generics and lambdas", src: "List<String> xs = new ArrayList<>();\nxs.forEach(x -> System.out.println(x));\nRunnable r = Main::run;", want: []Token{
		tok(Type, "List"), tok(Operator, "<"), tok(Type, "String"), tok(Keyword, "new"), tok(Type, "ArrayList"),
		tok(Function, "forEach"), tok(Operator, "->"), tok(Punctuation, "::"),
	}},
	{name: "contextual words are names", src: "int to = 5; b.with(x); var module = open; requires(uses);", want: []Token{
		tok(Variable, "to"), tok(Function, "with"), tok(Variable, "module"), tok(Variable, "open"), tok(Function, "requires"),
		tok(Variable, "uses"),
	}},
	{name: "comments", src: "/** Javadoc {@link Foo} */\n// line\nint x; /* block */", want: []Token{
		tok(Comment, "/** Javadoc {@link Foo} */"), tok(Comment, "// line"), tok(BuiltinType, "int"), tok(Comment, "/* block */"),
	}},
	{name: "numbers", src: "int n = 0x1F + 0b101 + 1_000 + 017; double d = 1e-3 + .5;", want: []Token{
		tok(Number, "0x1F"), tok(Number, "0b101"), tok(Number, "1_000"), tok(Number, "017"), tok(Number, "1e-3"),
		tok(Number, ".5"),
	}},
	{name: "identifiers with dollar signs", src: "int $x = a$b;", want: []Token{
		tok(Variable, "$x"), tok(Variable, "a$b"),
	}},
	{name: "capitalized members stay types", src: "Map.Entry<K, V> e = null;", want: []Token{
		tok(Type, "Map"), tok(Type, "Entry"), tok(Type, "K"), tok(Type, "V"), tok(Variable, "e"), tok(Primitive, "null"),
	}},
}
