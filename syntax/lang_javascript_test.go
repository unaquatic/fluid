package syntax

import "testing"

func TestJavaScript(t *testing.T) {
	testLanguage(t, "javascript", []string{"js", "jsx", "mjs", "cjs", "node", "react", "app.js"}, javascriptTests)
}

var javascriptTests = []kindCase{
	{name: "declarations", src: "const a = 1; let b; var c;\nfunction f(x) { return x }\nclass A extends B { static m() {} }\nexport default async function* g() { yield await h() }", want: []Token{
		tok(Keyword, "const"), tok(Variable, "a"), tok(Keyword, "let"), tok(Keyword, "var"), tok(Keyword, "function"),
		tok(Function, "f"), tok(Keyword, "return"), tok(Keyword, "class"), tok(Type, "A"), tok(Keyword, "extends"),
		tok(Type, "B"), tok(Keyword, "static"), tok(Function, "m"), tok(Keyword, "export"), tok(Keyword, "default"),
		tok(Keyword, "async"), tok(Keyword, "function"), tok(Operator, "*"), tok(Function, "g"), tok(Keyword, "yield"),
		tok(Keyword, "await"), tok(Function, "h"),
	}},
	{name: "control flow", src: "if (a) {} else if (b) {}\nfor (const k of o) {} for (k in o) {}\nwhile (x) {} do {} while (y);\nswitch (v) { case 1: break; default: }\ntry {} catch (e) {} finally {}\nthrow new Error('x');", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "else"), tok(Keyword, "if"), tok(Keyword, "for"), tok(Keyword, "const"),
		tok(Keyword, "of"), tok(Keyword, "for"), tok(Keyword, "in"), tok(Keyword, "while"), tok(Keyword, "do"),
		tok(Keyword, "while"), tok(Keyword, "switch"), tok(Keyword, "case"), tok(Keyword, "break"), tok(Keyword, "default"),
		tok(Keyword, "try"), tok(Keyword, "catch"), tok(Keyword, "finally"), tok(Keyword, "throw"), tok(Keyword, "new"),
		tok(BuiltinType, "Error"), tok(String, "'x'"),
	}},
	{name: "modules", src: "import React, { useState as use } from 'react';\nexport { a, b as c };\nconst m = require('m');", want: []Token{
		tok(Keyword, "import"), tok(Type, "React"), tok(Variable, "useState"), tok(Keyword, "as"), tok(Variable, "use"),
		tok(Keyword, "from"), tok(String, "'react'"), tok(Keyword, "export"), tok(Keyword, "as"), tok(BuiltinFunction, "require"),
	}},
	{name: "primitives and builtins", src: "x = [true, false, null, undefined, NaN, Infinity, this];\nconsole.log(JSON.stringify(o), Math.PI, Object.keys(o));", want: []Token{
		tok(Primitive, "true"), tok(Primitive, "false"), tok(Primitive, "null"), tok(Primitive, "undefined"),
		tok(Primitive, "NaN"), tok(Primitive, "Infinity"), tok(Keyword, "this"), tok(BuiltinVariable, "console"),
		tok(Function, "log"), tok(BuiltinVariable, "JSON"), tok(Function, "stringify"), tok(BuiltinVariable, "Math"),
		tok(Constant, "PI"), tok(BuiltinType, "Object"), tok(Function, "keys"),
	}},
	{name: "member names are not keywords", src: "Array.from(x); p.catch(e).finally(f); m.delete(k); o.default; a?.b?.c(); this.#priv", want: []Token{
		tok(BuiltinType, "Array"), tok(Function, "from"), tok(Function, "catch"), tok(Function, "finally"),
		tok(Function, "delete"), tok(Property, "default"), tok(Variable, "a"), tok(Property, "b"), tok(Function, "c"),
		tok(Keyword, "this"), tok(Property, "#priv"),
	}},
	{name: "strings", src: "s = 'it\\'s' + \"line\\n\" + \"\\u00e9\";", want: []Token{
		tok(String, "'it"), tok(StringEscape, `\'`), tok(String, "s'"), tok(String, `"line`), tok(StringEscape, `\n`),
		tok(String, `"`), tok(StringEscape, `\u`),
	}},
	{name: "template literals", src: "t = `a ${b + c} d ${`nested ${e}`}`;", want: []Token{
		tok(String, "`a "), tok(StringEscape, "${"), tok(Variable, "b"), tok(Operator, "+"), tok(Variable, "c"),
		tok(StringEscape, "}"), tok(String, " d "), tok(StringEscape, "${"), tok(String, "`nested "),
		tok(StringEscape, "${"), tok(Variable, "e"), tok(StringEscape, "}"), tok(String, "`"), tok(StringEscape, "}"),
		tok(String, "`"), tok(Punctuation, ";"),
	}},
	{name: "template literals across lines", src: "html = `\n  <div>${x}</div>\n`; y", want: []Token{
		tok(String, "`"), tok(String, "  <div>"), tok(StringEscape, "${"), tok(Variable, "x"), tok(StringEscape, "}"),
		tok(String, "</div>"), tok(String, "`"), tok(Variable, "y"),
	}},
	{name: "tagged templates", src: "q = sql`SELECT ${id}`;", want: []Token{
		tok(Variable, "sql"), tok(String, "`SELECT "), tok(StringEscape, "${"), tok(Variable, "id"),
	}},
	{name: "regular expressions and division", src: "re = /ab+c/gi; x = a / b / c; y = (1) / 2; z = s.replace(/[/]/g, '');\nif (/^\\d+$/.test(v)) {}", want: []Token{
		tok(Regex, "/ab+c/gi"), tok(Variable, "a"), tok(Operator, "/"), tok(Variable, "b"), tok(Operator, "/"),
		tok(Punctuation, ")"), tok(Operator, "/"), tok(Function, "replace"), tok(Regex, "/[/]/g"),
		tok(Regex, `/^\d+$/`), tok(Function, "test"),
	}},
	{name: "comments", src: "// line\n/* block\n */ x /* inline */ = 1;\n/** JSDoc @param x */", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/* block"), tok(Comment, " */"), tok(Variable, "x"),
		tok(Comment, "/* inline */"), tok(Comment, "/** JSDoc @param x */"),
	}},
	{name: "shebang", src: "#!/usr/bin/env node\nconsole.log(1);", want: []Token{
		tok(Comment, "#!/usr/bin/env node"), tok(BuiltinVariable, "console"),
	}},
	{name: "numbers", src: "n = [0, 42, 0xFF, 0o17, 0b101, 1_000, 3.14, .5, 1e3, 10n];", want: []Token{
		tok(Number, "0"), tok(Number, "42"), tok(Number, "0xFF"), tok(Number, "0o17"), tok(Number, "0b101"),
		tok(Number, "1_000"), tok(Number, "3.14"), tok(Number, ".5"), tok(Number, "1e3"), tok(Number, "10n"),
	}},
	{name: "operators", src: "a ??= b; c ||= d; e **= 2; f === g; h !== i; j => j; [...xs]; k?.l; m >>>= 1;", want: []Token{
		tok(Operator, "??="), tok(Operator, "||="), tok(Operator, "**="), tok(Operator, "==="), tok(Operator, "!=="),
		tok(Operator, "=>"), tok(Operator, "..."), tok(Operator, "?"), tok(Property, "l"), tok(Operator, ">>>="),
	}},
	{name: "constants and classes", src: "const MAX_RETRIES = 3; new Map(); class Foo {}; Foo.bar;", want: []Token{
		tok(Constant, "MAX_RETRIES"), tok(Keyword, "new"), tok(BuiltinType, "Map"), tok(Type, "Foo"), tok(Type, "Foo"),
		tok(Property, "bar"),
	}},
	{name: "decorators", src: "@Component({ selector: 'x' })\nclass C { @Input() name; }", want: []Token{
		tok(Decorator, "@Component"), tok(Keyword, "class"), tok(Decorator, "@Input"),
	}},
	{name: "jsx elements", src: "const el = <div className=\"app\" onClick={() => go(1)}>Hi {name}, don't <b>x</b></div>;", want: []Token{
		tok(Punctuation, "<"), tok(Tag, "div"), tok(Attribute, "className"), tok(Operator, "="), tok(String, `"app"`),
		tok(Attribute, "onClick"), tok(Punctuation, "{"), tok(Function, "go"), tok(Text, "Hi "), tok(Variable, "name"),
		tok(Text, ", don't "), tok(Tag, "b"), tok(Text, "x"), tok(Tag, "b"), tok(Tag, "div"), tok(Punctuation, ";"),
	}},
	{name: "jsx across lines", src: "return (\n  <>\n    <Comp a={1} />\n    {items.map(i => <li key={i}>{i}</li>)}\n  </>\n);", want: []Token{
		tok(Punctuation, "<>"), tok(Tag, "Comp"), tok(Attribute, "a"), tok(Number, "1"), tok(Punctuation, "/>"),
		tok(Variable, "items"), tok(Function, "map"), tok(Tag, "li"), tok(Attribute, "key"), tok(Tag, "li"),
		tok(Punctuation, "</>"), tok(Punctuation, ");"),
	}},
	{name: "jsx attributes across lines", src: "<input\n  type=\"text\"\n  value={v}\n/>", want: []Token{
		tok(Tag, "input"), tok(Attribute, "type"), tok(String, `"text"`), tok(Attribute, "value"), tok(Variable, "v"),
		tok(Punctuation, "/>"),
	}},
	{name: "comparisons are not jsx", src: "if (a < b && c > d) {}", want: []Token{
		tok(Variable, "a"), tok(Operator, "<"), tok(Variable, "b"), tok(Operator, ">"),
	}},
	{name: "unclosed template literal", src: "s = `open\nstill", open: true, want: []Token{
		tok(String, "`open"), tok(String, "still"),
	}},
	{name: "quoted string ends with its line", src: "s = 'abc\nx = 1", want: []Token{
		tok(String, "'abc"), tok(Variable, "x"),
	}},
	{name: "builtin variables", src: "module.exports = exports = { window, document, globalThis, process, Reflect, Intl }", want: []Token{
		tok(BuiltinVariable, "module"), tok(Property, "exports"), tok(BuiltinVariable, "exports"),
		tok(BuiltinVariable, "window"), tok(BuiltinVariable, "document"), tok(BuiltinVariable, "globalThis"),
		tok(BuiltinVariable, "process"), tok(BuiltinVariable, "Reflect"), tok(BuiltinVariable, "Intl"),
	}},
}
