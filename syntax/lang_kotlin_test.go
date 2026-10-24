package syntax

import "testing"

func TestKotlin(t *testing.T) {
	testLanguage(t, "kotlin", []string{"kt", "kts", "build.gradle.kts", "Main.kt"}, kotlinTests)
}

var kotlinTests = []kindCase{
	{name: "declarations", src: "package app\nimport kotlin.math.max\nclass User(val name: String) : Base(), Named\nobject Single\ninterface Repo\ntypealias Id = Long\nfun greet(u: User): String = \"hi\"", want: []Token{
		tok(Keyword, "package"), tok(Variable, "app"), tok(Keyword, "import"), tok(Keyword, "class"), tok(Type, "User"),
		tok(Keyword, "val"), tok(Variable, "name"), tok(Type, "String"), tok(Function, "Base"), tok(Type, "Named"),
		tok(Keyword, "object"), tok(Type, "Single"), tok(Keyword, "interface"), tok(Type, "Repo"),
		tok(Keyword, "typealias"), tok(Type, "Id"), tok(Type, "Long"), tok(Keyword, "fun"), tok(Function, "greet"),
		tok(Type, "User"), tok(String, `"hi"`),
	}},
	{name: "modifiers", src: "open class A {\n  override fun f() {}\n  private val x = 1\n  internal var y: Int? = null\n  lateinit var z: String\n  companion object {}\n}\nsealed interface S\nenum class E { A }\ninline fun i() {}\nsuspend fun s() {}", want: []Token{
		tok(Keyword, "open"), tok(Keyword, "class"), tok(Keyword, "override"), tok(Keyword, "fun"), tok(Function, "f"),
		tok(Keyword, "private"), tok(Keyword, "val"), tok(Keyword, "internal"), tok(Keyword, "var"), tok(Type, "Int"),
		tok(Operator, "?"), tok(Primitive, "null"), tok(Keyword, "lateinit"), tok(Keyword, "companion"),
		tok(Keyword, "object"), tok(Keyword, "sealed"), tok(Keyword, "interface"), tok(Keyword, "enum"),
		tok(Keyword, "class"), tok(Keyword, "inline"), tok(Keyword, "suspend"),
	}},
	{name: "control flow", src: "when (x) {\n  is String -> 1\n  in 0..9 -> 2\n  else -> 3\n}\nfor (i in 1 until 10) {}\nif (a) b else c\ntry { throw e } catch (e: Exception) {} finally {}\nwhile (true) { break }", want: []Token{
		tok(Keyword, "when"), tok(Keyword, "is"), tok(Type, "String"), tok(Operator, "->"), tok(Keyword, "in"),
		tok(Number, "0"), tok(Operator, ".."), tok(Number, "9"), tok(Keyword, "else"), tok(Keyword, "for"),
		tok(Keyword, "in"), tok(Variable, "until"), tok(Keyword, "if"), tok(Keyword, "else"), tok(Keyword, "try"),
		tok(Keyword, "throw"), tok(Keyword, "catch"), tok(Type, "Exception"), tok(Keyword, "finally"),
		tok(Keyword, "while"), tok(Primitive, "true"), tok(Keyword, "break"),
	}},
	{name: "string templates", src: "val s = \"Hi $name, ${user.age + 1}\\n\"\nval c = 'x'", want: []Token{
		tok(String, `"Hi `), tok(StringEscape, "$name"), tok(String, ", "), tok(StringEscape, "${"), tok(Variable, "user"),
		tok(Property, "age"), tok(Operator, "+"), tok(Number, "1"), tok(StringEscape, "}"), tok(StringEscape, `\n`),
		tok(String, `"`), tok(String, "'x'"),
	}},
	{name: "raw strings across lines", src: "val q = \"\"\"\n  SELECT $col\n  FROM t\n\"\"\".trimIndent()", want: []Token{
		tok(String, `"""`), tok(String, "  SELECT "), tok(StringEscape, "$col"), tok(String, "  FROM t"),
		tok(String, `"""`), tok(Function, "trimIndent"),
	}},
	{name: "nested comments", src: "/* a /* b */ c */ val x = 1\n// line", want: []Token{
		tok(Comment, "/* a /* b */ c */"), tok(Keyword, "val"), tok(Comment, "// line"),
	}},
	{name: "annotations and labels", src: "@JvmStatic fun m() {}\n@Suppress(\"x\") class C\nlist.forEach { return@forEach }", want: []Token{
		tok(Decorator, "@JvmStatic"), tok(Decorator, "@Suppress"), tok(String, `"x"`), tok(Keyword, "return"),
		tok(Decorator, "@forEach"),
	}},
	{name: "null safety", src: "a?.b ?: c; d!!.e; f as? G", want: []Token{
		tok(Variable, "a"), tok(Property, "b"), tok(Operator, "?"), tok(Operator, "!!"), tok(Property, "e"),
		tok(Keyword, "as"), tok(Operator, "?"), tok(Type, "G"),
	}},
	{name: "extension functions", src: "fun String.shout() = uppercase()", want: []Token{
		tok(Keyword, "fun"), tok(Type, "String"), tok(Function, "shout"), tok(Function, "uppercase"),
	}},
	{name: "contextual words are names", src: "val data = value; val actual = get(); field = file", want: []Token{
		tok(Variable, "data"), tok(Variable, "value"), tok(Variable, "actual"), tok(Function, "get"),
		tok(Variable, "field"), tok(Variable, "file"),
	}},
	{name: "numbers", src: "val n = 1_000L + 0xFF + 0b1 + 1.5f + 2e3", want: []Token{
		tok(Number, "1_000L"), tok(Number, "0xFF"), tok(Number, "0b1"), tok(Number, "1.5f"), tok(Number, "2e3"),
	}},
	{name: "elvis operator", src: "val n = x ?: 0", want: []Token{
		tok(Keyword, "val"), tok(Variable, "n"), tok(Variable, "x"), tok(Operator, "?:"), tok(Number, "0"),
	}},
}
