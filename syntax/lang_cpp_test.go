package syntax

import "testing"

func TestCPP(t *testing.T) {
	testLanguage(t, "cpp", []string{"c++", "cc", "cxx", "hpp", "hh", "ino", "main.cpp"}, cppTests)
}

var cppTests = []kindCase{
	{name: "classes and namespaces", src: "namespace app {\nclass Foo : public Bar {\npublic:\n  virtual ~Foo() override;\n  explicit Foo(int x) noexcept;\nprivate:\n  mutable int n_;\n  friend class Baz;\n};\n}", want: []Token{
		tok(Keyword, "namespace"), tok(Type, "app"), tok(Keyword, "class"), tok(Type, "Foo"), tok(Punctuation, ":"),
		tok(Keyword, "public"), tok(Variable, "Bar"), tok(Keyword, "public"), tok(Keyword, "virtual"), tok(Operator, "~"),
		tok(Function, "Foo"), tok(Keyword, "override"), tok(Keyword, "explicit"), tok(Function, "Foo"),
		tok(Keyword, "noexcept"), tok(Keyword, "private"), tok(Keyword, "mutable"), tok(Type, "int"),
		tok(Keyword, "friend"), tok(Keyword, "class"), tok(Type, "Baz"),
	}},
	{name: "templates and casts", src: "template <typename T, class U>\nconcept Small = sizeof(T) < 4;\nauto p = static_cast<int*>(q);\nauto r = reinterpret_cast<char*>(&s);\nstd::vector<std::string> v;\nusing std::cout;", want: []Token{
		tok(Keyword, "template"), tok(Keyword, "typename"), tok(Keyword, "class"), tok(Keyword, "concept"),
		tok(Type, "Small"), tok(Keyword, "sizeof"), tok(Keyword, "auto"), tok(Keyword, "static_cast"), tok(Operator, "<"),
		tok(Type, "int"), tok(Keyword, "reinterpret_cast"), tok(Type, "char"), tok(Variable, "std"),
		tok(Punctuation, "::"), tok(Variable, "vector"), tok(Keyword, "using"),
	}},
	{name: "raw strings", src: "auto s = R\"(raw \"text\")\";\nauto d = R\"x(a)\"b)x\";\nauto m = u8R\"(m)\";\nauto w = LR\"(w)\";", want: []Token{
		tok(String, `R"(raw "text")"`), tok(String, `R"x(a)"b)x"`), tok(String, `u8R"(m)"`), tok(String, `LR"(w)"`),
	}},
	{name: "raw string across lines", src: "auto q = R\"sql(\nSELECT \"a\"\n)sql\";\ndone();", want: []Token{
		tok(String, `R"sql(`), tok(String, `SELECT "a"`), tok(String, `)sql"`), tok(Function, "done"),
	}},
	{name: "exceptions and keywords", src: "try { throw std::runtime_error(\"x\"); } catch (const std::exception& e) {}\nif constexpr (N > 4) {}\nco_await f(); co_return 1;\ndelete p; auto t = new T();\nthis->x = 1; decltype(a) b;", want: []Token{
		tok(Keyword, "try"), tok(Keyword, "throw"), tok(Function, "runtime_error"), tok(Keyword, "catch"),
		tok(Keyword, "const"), tok(Operator, "&"), tok(Keyword, "if"), tok(Keyword, "constexpr"), tok(Keyword, "co_await"),
		tok(Keyword, "co_return"), tok(Keyword, "delete"), tok(Keyword, "new"), tok(Keyword, "this"),
		tok(Operator, "->"), tok(Property, "x"), tok(Keyword, "decltype"),
	}},
	{name: "alternative operators and primitives", src: "if (a and b or not c) {}\nbool ok = true && !false;\nint* p = nullptr;", want: []Token{
		tok(Keyword, "and"), tok(Keyword, "or"), tok(Keyword, "not"), tok(Type, "bool"), tok(Primitive, "true"),
		tok(Primitive, "false"), tok(Primitive, "nullptr"),
	}},
	{name: "character types and literals", src: "char8_t a = u8'a'; char16_t b = u'b'; char32_t c = U'c'; wchar_t d = L'd';", want: []Token{
		tok(Type, "char8_t"), tok(String, "u8'a'"), tok(Type, "char16_t"), tok(String, "u'b'"), tok(Type, "char32_t"),
		tok(String, "U'c'"), tok(Type, "wchar_t"), tok(String, "L'd'"),
	}},
	{name: "preprocessor", src: "#include <vector>\n#if defined(X) && X > 1\n#endif", want: []Token{
		tok(Preproc, "#include"), tok(String, "<vector>"), tok(Preproc, "#if"), tok(Function, "defined"),
		tok(Preproc, "#endif"),
	}},
	{name: "lambdas and members", src: "auto f = [&](int x) -> int { return x * 2; };\nobj.method(); ptr->field;", want: []Token{
		tok(Keyword, "auto"), tok(Operator, "&"), tok(Operator, "->"), tok(Type, "int"), tok(Keyword, "return"),
		tok(Function, "method"), tok(Property, "field"),
	}},
	{name: "comments", src: "// line\n/* block */ int x;", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/* block */"), tok(Type, "int"),
	}},
}
