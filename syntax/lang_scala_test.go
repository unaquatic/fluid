package syntax

import "testing"

func TestScala(t *testing.T) {
	testLanguage(t, "scala", []string{"sc", "sbt", "Main.scala"}, scalaTests)
}

var scalaTests = []kindCase{
	{name: "declarations", src: "package app\nimport scala.util.Try\nobject Main extends App {\n  def f(x: Int): Int = x\n  val a = 1; var b = 2; lazy val c = 3\n}\ntrait T\ncase class P(x: Int)\nsealed abstract class A\nenum Color { case Red }\ntype Id = Long", want: []Token{
		tok(Keyword, "package"), tok(Keyword, "import"), tok(Variable, "scala"), tok(Property, "util"), tok(Type, "Try"),
		tok(Keyword, "object"), tok(Type, "Main"), tok(Keyword, "extends"), tok(Type, "App"), tok(Keyword, "def"),
		tok(Function, "f"), tok(Type, "Int"), tok(Keyword, "val"), tok(Keyword, "var"), tok(Keyword, "lazy"),
		tok(Keyword, "trait"), tok(Type, "T"), tok(Keyword, "case"), tok(Keyword, "class"), tok(Type, "P"),
		tok(Keyword, "sealed"), tok(Keyword, "abstract"), tok(Keyword, "class"), tok(Type, "A"), tok(Keyword, "enum"),
		tok(Type, "Color"), tok(Keyword, "case"), tok(Type, "Red"), tok(Keyword, "type"), tok(Type, "Id"), tok(Type, "Long"),
	}},
	{name: "control flow", src: "x match {\n  case Some(v) if v > 0 => v\n  case _ => 0\n}\nfor (i <- 1 to 10 if i % 2 == 0) yield i\ntry f() catch { case e: Exception => } finally g()\nif (a) b else c\nwhile (d) e\nthrow new E", want: []Token{
		tok(Keyword, "match"), tok(Keyword, "case"), tok(Function, "Some"), tok(Keyword, "if"), tok(Operator, "=>"),
		tok(Keyword, "case"), tok(Keyword, "for"), tok(Operator, "<-"), tok(Variable, "to"), tok(Keyword, "if"),
		tok(Keyword, "yield"), tok(Keyword, "try"), tok(Function, "f"), tok(Keyword, "catch"), tok(Keyword, "case"),
		tok(Type, "Exception"), tok(Keyword, "finally"), tok(Keyword, "if"), tok(Keyword, "else"), tok(Keyword, "while"),
		tok(Keyword, "throw"), tok(Keyword, "new"), tok(Type, "E"),
	}},
	{name: "string interpolation", src: "val s = s\"Hi $name ${a + b}\"\nval f = f\"$x%.2f\"\nval plain = \"no $interp\"", want: []Token{
		tok(String, `s"Hi `), tok(StringEscape, "$name"), tok(String, " "), tok(StringEscape, "${"), tok(Variable, "a"),
		tok(Operator, "+"), tok(Variable, "b"), tok(StringEscape, "}"), tok(String, `"`), tok(String, `f"`),
		tok(StringEscape, "$x"), tok(String, `%.2f"`), tok(String, `"no $interp"`),
	}},
	{name: "multiline strings", src: "val q = \"\"\"a\n  |b $x\"\"\".stripMargin\nval r = s\"\"\"\n  $y\n\"\"\"", want: []Token{
		tok(String, `"""a`), tok(String, "  |b $x"), tok(String, `"""`), tok(Property, "stripMargin"),
		tok(String, `s"""`), tok(StringEscape, "$y"), tok(String, `"""`),
	}},
	{name: "scala 3", src: "given Ordering[Int] = Ordering.Int\ndef f(using ctx: Ctx) = ctx\nextension (s: String) def shout = s\nopaque type Meters = Double\ninline def g = 1", want: []Token{
		tok(Keyword, "given"), tok(Type, "Ordering"), tok(Keyword, "using"), tok(Keyword, "extension"), tok(Keyword, "def"),
		tok(Function, "shout"), tok(Keyword, "opaque"), tok(Keyword, "type"), tok(Type, "Meters"), tok(Keyword, "inline"),
	}},
	{name: "annotations", src: "@tailrec def loop(n: Int): Int = loop(n - 1)\n@deprecated(\"x\") val v = 1", want: []Token{
		tok(Decorator, "@tailrec"), tok(Keyword, "def"), tok(Function, "loop"), tok(Decorator, "@deprecated"),
	}},
	{name: "comments and primitives", src: "/* a /* b */ c */ val n = null\n// line\nval t = true; val c = 'x'", want: []Token{
		tok(Comment, "/* a /* b */ c */"), tok(Primitive, "null"), tok(Comment, "// line"), tok(Primitive, "true"),
		tok(String, "'x'"),
	}},
	{name: "numbers", src: "val n = 1_000L + 0xFF + 1.5 + 2e3d", want: []Token{
		tok(Number, "1_000L"), tok(Number, "0xFF"), tok(Number, "1.5"), tok(Number, "2e3d"),
	}},
}
