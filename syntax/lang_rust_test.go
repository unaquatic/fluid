package syntax

import "testing"

func TestRust(t *testing.T) {
	testLanguage(t, "rust", []string{"rs", "main.rs"}, rustTests)
}

var rustTests = []kindCase{
	{name: "items", src: "pub fn add(a: i32, b: i32) -> i32 { a + b }\nstruct Point { x: f64 }\nenum E { A, B(u8) }\ntrait T { fn m(&self); }\nimpl T for Point {}\ntype Alias = Vec<String>;\nunion U { i: u32 }", want: []Token{
		tok(Keyword, "pub"), tok(Keyword, "fn"), tok(Function, "add"), tok(Type, "i32"), tok(Operator, "->"),
		tok(Keyword, "struct"), tok(Type, "Point"), tok(Type, "f64"), tok(Keyword, "enum"), tok(Type, "E"),
		tok(Type, "A"), tok(Function, "B"), tok(Type, "u8"), tok(Keyword, "trait"), tok(Type, "T"), tok(Keyword, "fn"),
		tok(Function, "m"), tok(Operator, "&"), tok(Keyword, "self"), tok(Keyword, "impl"), tok(Type, "T"),
		tok(Keyword, "for"), tok(Type, "Point"), tok(Keyword, "type"), tok(Type, "Alias"), tok(Type, "Vec"),
		tok(Type, "String"), tok(Keyword, "union"), tok(Type, "U"),
	}},
	{name: "control flow", src: "if let Some(x) = o { } else { loop { break } }\nmatch e { Ok(v) => v, Err(_) => return }\nwhile cond {} for i in 0..10 {}\nlet v = 'outer: loop { continue 'outer };", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "let"), tok(Builtin, "Some"), tok(Keyword, "else"), tok(Keyword, "loop"),
		tok(Keyword, "break"), tok(Keyword, "match"), tok(Builtin, "Ok"), tok(Operator, "=>"), tok(Builtin, "Err"),
		tok(Keyword, "return"), tok(Keyword, "while"), tok(Keyword, "for"), tok(Keyword, "in"), tok(Number, "0"),
		tok(Operator, ".."), tok(Number, "10"), tok(Attribute, "'outer"), tok(Keyword, "loop"), tok(Keyword, "continue"),
		tok(Attribute, "'outer"),
	}},
	{name: "modules and constants", src: "use std::io::{self, Read};\nmod net;\nconst MAX_LEN: usize = 10;\nstatic GLOBAL: &str = \"g\";\nextern crate alloc;\npub(crate) unsafe fn f() {}", want: []Token{
		tok(Keyword, "use"), tok(Variable, "std"), tok(Punctuation, "::"), tok(Variable, "io"), tok(Keyword, "self"),
		tok(Type, "Read"), tok(Keyword, "mod"), tok(Keyword, "const"), tok(Constant, "MAX_LEN"), tok(Type, "usize"),
		tok(Keyword, "static"), tok(Constant, "GLOBAL"), tok(Type, "str"), tok(Keyword, "extern"), tok(Keyword, "crate"),
		tok(Keyword, "pub"), tok(Keyword, "crate"), tok(Keyword, "unsafe"), tok(Keyword, "fn"),
	}},
	{name: "macros", src: "println!(\"{}\", x);\nassert_eq!(a, b);\nlet v = vec![1, 2];\nmacro_rules! m { () => {} }\nlet ne = a != b;", want: []Token{
		tok(Function, "println!"), tok(String, `"{}"`), tok(Function, "assert_eq!"), tok(Function, "vec!"),
		tok(Keyword, "macro_rules"), tok(Variable, "a"), tok(Operator, "!="),
	}},
	{name: "lifetimes and characters", src: "fn f<'a>(s: &'a str) -> &'static str {\n  let c = 'x'; let n = '\\n'; let q = '\\''; let u = '\\u{1F600}'; let e = 'é';\n  s\n}", want: []Token{
		tok(Attribute, "'a"), tok(Attribute, "'a"), tok(Type, "str"), tok(Attribute, "'static"), tok(String, "'x'"),
		tok(String, `'\n'`), tok(String, `'\''`), tok(String, `'\u{1F600}'`), tok(String, "'é'"),
	}},
	{name: "byte, raw, and C strings", src: "let a = b'x'; let s = b\"bytes\"; let r = r\"raw\\n\"; let h = r#\"has \"quotes\"\"#; let c = c\"cstr\"; let br = br#\"b\"#;", want: []Token{
		tok(String, "b'x'"), tok(String, `b"bytes"`), tok(String, `r"raw\n"`), tok(String, `r#"has "quotes""#`),
		tok(String, `c"cstr"`), tok(String, `br#"b"#`),
	}},
	{name: "strings span lines", src: "let s = \"a\\tb\nc\";\nlet r = r##\"x\n\"# still\"##; done()", want: []Token{
		tok(String, `"a`), tok(StringEscape, `\t`), tok(String, "b"), tok(String, `c"`), tok(String, `r##"x`),
		tok(String, `"# still"##`), tok(Function, "done"),
	}},
	{name: "attributes", src: "#[derive(Debug, Clone)]\n#![allow(dead_code)]\n#[cfg(test)] mod tests {}\n#[serde(rename = \"x]\")]\nstruct S;", want: []Token{
		tok(Decorator, "#[derive(Debug, Clone)]"), tok(Decorator, "#![allow(dead_code)]"), tok(Decorator, "#[cfg(test)]"),
		tok(Keyword, "mod"), tok(Decorator, `#[serde(rename = "x]")]`), tok(Keyword, "struct"),
	}},
	{name: "comments", src: "// line\n/// doc\n//! inner doc\n/* a /* nested */ still */ x\n/* open\n close */ y", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/// doc"), tok(Comment, "//! inner doc"),
		tok(Comment, "/* a /* nested */ still */"), tok(Variable, "x"), tok(Comment, "/* open"), tok(Comment, " close */"),
		tok(Variable, "y"),
	}},
	{name: "numbers", src: "let n = [1, 1_000, 0xff, 0o7, 0b1, 1.5, 2e10, 1u8, 2.0f32, 3usize];", want: []Token{
		tok(Number, "1"), tok(Number, "1_000"), tok(Number, "0xff"), tok(Number, "0o7"), tok(Number, "0b1"),
		tok(Number, "1.5"), tok(Number, "2e10"), tok(Number, "1u8"), tok(Number, "2.0f32"), tok(Number, "3usize"),
	}},
	{name: "members and await", src: "fut.await; s.len(); p.x; t.0; x.match_y", want: []Token{
		tok(Keyword, "await"), tok(Function, "len"), tok(Property, "x"), tok(Number, "0"), tok(Property, "match_y"),
	}},
	{name: "self and primitives", src: "impl Foo { fn new() -> Self { Self { ok: true, no: false } } }", want: []Token{
		tok(Keyword, "impl"), tok(Type, "Foo"), tok(Function, "new"), tok(Keyword, "Self"), tok(Primitive, "true"),
		tok(Primitive, "false"),
	}},
	{name: "operators", src: "a += 1; let r = b..=c; x as u64; &mut y; *ptr; a == b && c != d || !e; f()?;", want: []Token{
		tok(Operator, "+="), tok(Operator, "..="), tok(Keyword, "as"), tok(Type, "u64"), tok(Operator, "&"),
		tok(Keyword, "mut"), tok(Operator, "*"), tok(Operator, "=="), tok(Operator, "&&"), tok(Operator, "!="),
		tok(Operator, "||"), tok(Operator, "!"), tok(Operator, "?"),
	}},
	{name: "unclosed nested comment", src: "/* a /* b */\nstill", open: true, want: []Token{
		tok(Comment, "/* a /* b */"), tok(Comment, "still"),
	}},
}
