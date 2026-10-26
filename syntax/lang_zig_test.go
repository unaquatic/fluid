package syntax

import "testing"

func TestZig(t *testing.T) {
	testLanguage(t, "zig", []string{"main.zig"}, zigTests)
}

var zigTests = []kindCase{
	{name: "declarations", src: "const std = @import(\"std\");\npub fn main() !void {\n    var x: u32 = 0;\n    const Point = struct { x: i32, y: i32 };\n    _ = x;\n}", want: []Token{
		tok(Keyword, "const"), tok(Variable, "std"), tok(BuiltinFunction, "@import"), tok(String, `"std"`), tok(Keyword, "pub"),
		tok(Keyword, "fn"), tok(Function, "main"), tok(Operator, "!"), tok(BuiltinType, "void"), tok(Keyword, "var"),
		tok(Variable, "x"), tok(BuiltinType, "u32"), tok(Number, "0"), tok(Keyword, "const"), tok(Type, "Point"),
		tok(Keyword, "struct"), tok(BuiltinType, "i32"),
	}},
	{name: "control flow", src: "if (a) b else c;\nwhile (i < n) : (i += 1) {}\nfor (items) |item| {}\nswitch (v) { 1 => {}, else => {} }\ndefer f(); errdefer g();\nreturn try h() orelse unreachable;\nconst r = x catch |err| err;", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "else"), tok(Keyword, "while"), tok(Operator, "+="), tok(Keyword, "for"),
		tok(Operator, "|"), tok(Variable, "item"), tok(Keyword, "switch"), tok(Operator, "=>"), tok(Keyword, "else"),
		tok(Keyword, "defer"), tok(Keyword, "errdefer"), tok(Keyword, "return"), tok(Keyword, "try"),
		tok(Keyword, "orelse"), tok(Keyword, "unreachable"), tok(Keyword, "catch"),
	}},
	{name: "builtins and integer types", src: "const a = @intCast(b);\nconst c: u7 = 1;\nconst d: i128 = 2;\nconst e: usize = @sizeOf(T);\nconst f: c_int = 0;", want: []Token{
		tok(BuiltinFunction, "@intCast"), tok(BuiltinType, "u7"), tok(BuiltinType, "i128"), tok(BuiltinType, "usize"), tok(BuiltinFunction, "@sizeOf"),
		tok(Type, "T"), tok(BuiltinType, "c_int"),
	}},
	{name: "names that only look like integer types", src: "const u8x = i2c;", want: []Token{
		tok(Variable, "u8x"), tok(Variable, "i2c"),
	}},
	{name: "multiline strings", src: "const s =\n    \\\\line one\n    \\\\line two\n;", want: []Token{
		tok(String, `\\line one`), tok(String, `\\line two`), tok(Punctuation, ";"),
	}},
	{name: "strings and characters", src: "const s = \"a\\n\";\nconst c = 'x';", want: []Token{
		tok(String, `"a`), tok(StringEscape, `\n`), tok(String, `"`), tok(String, "'x'"),
	}},
	{name: "primitives", src: "const t = true;\nconst n = null;\nvar u: u8 = undefined;\nconst f = false;", want: []Token{
		tok(Primitive, "true"), tok(Primitive, "null"), tok(Primitive, "undefined"), tok(Primitive, "false"),
	}},
	{name: "comptime and types", src: "fn max(comptime T: type, a: T, b: T) T {\n    return if (a > b) a else b;\n}", want: []Token{
		tok(Keyword, "fn"), tok(Function, "max"), tok(Keyword, "comptime"), tok(Type, "T"), tok(BuiltinType, "type"),
		tok(Keyword, "return"), tok(Keyword, "if"), tok(Keyword, "else"),
	}},
	{name: "comments", src: "// comment\n/// doc\n//! top-level doc\nconst x = 1; // trailing", want: []Token{
		tok(Comment, "// comment"), tok(Comment, "/// doc"), tok(Comment, "//! top-level doc"), tok(Keyword, "const"),
		tok(Comment, "// trailing"),
	}},
	{name: "numbers", src: "const n = 0x1F + 0o17 + 0b11 + 1_000 + 1.5e3;", want: []Token{
		tok(Number, "0x1F"), tok(Number, "0o17"), tok(Number, "0b11"), tok(Number, "1_000"), tok(Number, "1.5e3"),
	}},
}
