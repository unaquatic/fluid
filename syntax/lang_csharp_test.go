package syntax

import "testing"

func TestCSharp(t *testing.T) {
	testLanguage(t, "csharp", []string{"c#", "cs", "Program.cs"}, csharpTests)
}

var csharpTests = []kindCase{
	{name: "types and members", src: "namespace App;\npublic sealed class Foo : IBar {\n  private readonly int _n;\n  public string Name { get; set; } = \"x\";\n  protected internal static void Run() {}\n}", want: []Token{
		tok(Keyword, "namespace"), tok(Type, "App"), tok(Keyword, "public"), tok(Keyword, "sealed"), tok(Keyword, "class"),
		tok(Type, "Foo"), tok(Type, "IBar"), tok(Keyword, "private"), tok(Keyword, "readonly"), tok(BuiltinType, "int"),
		tok(Variable, "_n"), tok(Keyword, "public"), tok(BuiltinType, "string"), tok(Variable, "get"), tok(Variable, "set"),
		tok(String, `"x"`), tok(Keyword, "protected"), tok(Keyword, "internal"), tok(Keyword, "static"), tok(BuiltinType, "void"),
		tok(Function, "Run"),
	}},
	{name: "records structs interfaces enums", src: "public record Point(int X, int Y);\nstruct S {}\ninterface I {}\nenum E { A }", want: []Token{
		tok(Keyword, "record"), tok(Type, "Point"), tok(BuiltinType, "int"), tok(Keyword, "struct"), tok(Type, "S"),
		tok(Keyword, "interface"), tok(Type, "I"), tok(Keyword, "enum"), tok(Type, "E"),
	}},
	{name: "statements", src: "async Task<int> F() {\n  await Task.Delay(1);\n  var x = y is null ? default : (int)z;\n  foreach (var i in xs) yield return i;\n  lock (o) {}\n  using var r = R();\n  throw new InvalidOperationException();\n}", want: []Token{
		tok(Keyword, "async"), tok(Type, "Task"), tok(BuiltinType, "int"), tok(Function, "F"), tok(Keyword, "await"),
		tok(Type, "Task"), tok(Function, "Delay"), tok(Keyword, "var"), tok(Keyword, "is"), tok(Primitive, "null"),
		tok(Keyword, "default"), tok(Keyword, "foreach"), tok(Keyword, "in"), tok(Keyword, "yield"),
		tok(Keyword, "return"), tok(Keyword, "lock"), tok(Keyword, "using"), tok(Keyword, "throw"), tok(Keyword, "new"),
	}},
	{name: "strings", src: "var a = \"esc\\n\";\nvar b = @\"C:\\dir\"\"q\";\nvar c = $\"x={x}, {{lit}}\";\nvar d = $@\"p\\{n}\";\nvar e = \"\"\"raw \"quoted\" \"\"\";", want: []Token{
		tok(String, `"esc`), tok(StringEscape, `\n`), tok(String, `@"C:\dir`), tok(StringEscape, `""`), tok(String, `q"`),
		tok(String, `$"x=`), tok(StringEscape, "{"), tok(Variable, "x"), tok(StringEscape, "}"), tok(String, `, {{lit}}"`),
		tok(String, `$@"p\`), tok(StringEscape, "{"), tok(Variable, "n"), tok(StringEscape, "}"), tok(String, `"`),
		tok(String, `"""raw "quoted" """`),
	}},
	{name: "verbatim string across lines", src: "var s = @\"line1\nline2\";\nvar x = 1;", want: []Token{
		tok(String, `@"line1`), tok(String, `line2"`), tok(Keyword, "var"),
	}},
	{name: "raw interpolated string across lines", src: "var j = $\"\"\"\n  {\"id\": {id}}\n  \"\"\";", want: []Token{
		tok(String, `$"""`), tok(String, `  `), tok(StringEscape, "{"), tok(String, `"id"`), tok(Variable, "id"),
		tok(String, `  """`),
	}},
	{name: "preprocessor and attributes", src: "#region Setup\n#if DEBUG\n[Serializable]\nclass C {}\n#endif\n#endregion", want: []Token{
		tok(Preproc, "#region"), tok(Type, "Setup"), tok(Preproc, "#if"), tok(Constant, "DEBUG"), tok(Punctuation, "["),
		tok(Type, "Serializable"), tok(Preproc, "#endif"), tok(Preproc, "#endregion"),
	}},
	{name: "contextual words are names", src: "var value = file.Select(from);\nvar get = set + add;", want: []Token{
		tok(Keyword, "var"), tok(Variable, "value"), tok(Variable, "file"), tok(Function, "Select"), tok(Variable, "from"),
		tok(Variable, "get"), tok(Variable, "set"), tok(Variable, "add"),
	}},
	{name: "numbers", src: "var n = 1_000 + 0xFF + 0b1010 + 1.5m + 2.0f + 3d + 10L;", want: []Token{
		tok(Number, "1_000"), tok(Number, "0xFF"), tok(Number, "0b1010"), tok(Number, "1.5m"), tok(Number, "2.0f"),
		tok(Number, "3d"), tok(Number, "10L"),
	}},
	{name: "operators", src: "Func<int, int> f = x => x * 2; a ??= b; c?.D(); e!.f; g = h switch { _ => 0 };", want: []Token{
		tok(Type, "Func"), tok(Operator, "=>"), tok(Operator, "??="), tok(Operator, "?"), tok(Function, "D"),
		tok(Operator, "!"), tok(Property, "f"), tok(Keyword, "switch"), tok(Operator, "=>"),
	}},
	{name: "comments", src: "// line\n/// <summary>doc</summary>\n/* block */ int x;", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/// <summary>doc</summary>"), tok(Comment, "/* block */"), tok(BuiltinType, "int"),
	}},
	{name: "capitalized members", src: "var n = user.Name; user.Save(); Console.WriteLine(DateTime.Now, Color.RED);", want: []Token{
		tok(Keyword, "var"), tok(Variable, "n"), tok(Variable, "user"), tok(Property, "Name"), tok(Variable, "user"),
		tok(Function, "Save"), tok(Type, "Console"), tok(Function, "WriteLine"), tok(Type, "DateTime"), tok(Property, "Now"),
		tok(Type, "Color"), tok(Constant, "RED"),
	}},
}
