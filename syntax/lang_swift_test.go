package syntax

import "testing"

func TestSwift(t *testing.T) {
	testLanguage(t, "swift", []string{"main.swift"}, swiftTests)
}

var swiftTests = []kindCase{
	{name: "declarations", src: "import UIKit\nstruct Point { var x: Double; let y: Int }\nclass View: UIView {}\nenum Dir { case up, down }\nprotocol P {}\nextension Point: P {}\nactor Counter {}\ntypealias Fn = () -> Void\nfunc add(_ a: Int, to b: Int) -> Int { a + b }", want: []Token{
		tok(Keyword, "import"), tok(Type, "UIKit"), tok(Keyword, "struct"), tok(Type, "Point"), tok(Keyword, "var"),
		tok(Variable, "x"), tok(Type, "Double"), tok(Keyword, "let"), tok(Type, "Int"), tok(Keyword, "class"),
		tok(Type, "View"), tok(Type, "UIView"), tok(Keyword, "enum"), tok(Type, "Dir"), tok(Keyword, "case"),
		tok(Variable, "up"), tok(Keyword, "protocol"), tok(Type, "P"), tok(Keyword, "extension"), tok(Type, "Point"),
		tok(Keyword, "actor"), tok(Type, "Counter"), tok(Keyword, "typealias"), tok(Type, "Fn"), tok(Operator, "->"),
		tok(Type, "Void"), tok(Keyword, "func"), tok(Function, "add"), tok(Variable, "to"),
	}},
	{name: "control flow", src: "guard let v = opt else { return }\nif case .some(let x) = y {}\nfor i in 0..<10 where i > 2 {}\nswitch k { case .a: fallthrough; default: break }\ndefer {}\nrepeat {} while x\ndo { try f() } catch { throw e }", want: []Token{
		tok(Keyword, "guard"), tok(Keyword, "let"), tok(Keyword, "else"), tok(Keyword, "return"), tok(Keyword, "if"),
		tok(Keyword, "case"), tok(Function, "some"), tok(Keyword, "let"), tok(Keyword, "for"), tok(Keyword, "in"),
		tok(Number, "0"), tok(Operator, "..<"), tok(Number, "10"), tok(Keyword, "where"), tok(Keyword, "switch"),
		tok(Keyword, "case"), tok(Property, "a"), tok(Keyword, "fallthrough"), tok(Keyword, "default"),
		tok(Keyword, "break"), tok(Keyword, "defer"), tok(Keyword, "repeat"), tok(Keyword, "while"), tok(Keyword, "do"),
		tok(Keyword, "try"), tok(Keyword, "catch"), tok(Keyword, "throw"),
	}},
	{name: "attributes and modifiers", src: "@MainActor final class VM: ObservableObject {\n  @Published private(set) var n = 0\n  static let shared = VM()\n  lazy var z = 1\n  weak var d: D?\n  override init() {}\n}", want: []Token{
		tok(Decorator, "@MainActor"), tok(Keyword, "final"), tok(Keyword, "class"), tok(Type, "VM"),
		tok(Type, "ObservableObject"), tok(Decorator, "@Published"), tok(Keyword, "private"), tok(Keyword, "set"),
		tok(Keyword, "static"), tok(Keyword, "let"), tok(Function, "VM"), tok(Keyword, "lazy"), tok(Keyword, "weak"),
		tok(Operator, "?"), tok(Keyword, "override"), tok(Keyword, "init"),
	}},
	{name: "string interpolation", src: "let s = \"Hi \\(name), \\(f(a) + b)!\\n\"", want: []Token{
		tok(String, `"Hi `), tok(StringEscape, `\(`), tok(Variable, "name"), tok(StringEscape, ")"), tok(String, ", "),
		tok(StringEscape, `\(`), tok(Function, "f"), tok(Punctuation, "("), tok(Variable, "a"), tok(Punctuation, ")"),
		tok(Operator, "+"), tok(Variable, "b"), tok(StringEscape, ")"), tok(String, "!"), tok(StringEscape, `\n`),
		tok(String, `"`),
	}},
	{name: "multiline strings", src: "let m = \"\"\"\n    line \\(x)\n    \"\"\"\nlet y = 1", want: []Token{
		tok(String, `"""`), tok(String, "    line "), tok(StringEscape, `\(`), tok(Variable, "x"), tok(StringEscape, ")"),
		tok(String, `    """`), tok(Keyword, "let"),
	}},
	{name: "raw strings", src: "let r = #\"raw \\n \"quoted\"\"#\nlet rr = ##\"a\"#b\"##", want: []Token{
		tok(String, `#"raw \n "quoted""#`), tok(String, `##"a"#b"##`),
	}},
	{name: "directives", src: "#if DEBUG\nprint(1)\n#elseif os(iOS)\n#endif\nif #available(iOS 15, *) {}", want: []Token{
		tok(Preproc, "#if"), tok(Type, "DEBUG"), tok(Function, "print"), tok(Preproc, "#elseif"), tok(Function, "os"),
		tok(Preproc, "#endif"), tok(Keyword, "if"), tok(Preproc, "#available"),
	}},
	{name: "member keywords", src: "let t = Foo.self; let v = Bar.init(); x.let", want: []Token{
		tok(Keyword, "self"), tok(Keyword, "init"), tok(Property, "let"),
	}},
	{name: "nested comments and primitives", src: "/* a /* b */ c */ let x: Int? = nil\nlet ok = true || false", want: []Token{
		tok(Comment, "/* a /* b */ c */"), tok(Keyword, "let"), tok(Primitive, "nil"), tok(Primitive, "true"),
		tok(Primitive, "false"),
	}},
	{name: "operators and optionals", src: "a ?? b; c?.d; e!.f; g === h; i ... j", want: []Token{
		tok(Operator, "??"), tok(Operator, "?"), tok(Property, "d"), tok(Operator, "!"), tok(Property, "f"),
		tok(Operator, "==="), tok(Operator, "..."),
	}},
	{name: "unclosed multiline string", src: "let s = \"\"\"\nopen", open: true, want: []Token{
		tok(String, `"""`), tok(String, "open"),
	}},
}
