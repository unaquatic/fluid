package syntax

import "testing"

func TestHTML(t *testing.T) {
	testLanguage(t, "html", []string{"htm", "xhtml", "html5", "index.html"}, htmlTests)
}

var htmlTests = []kindCase{
	{name: "documents", src: "<!DOCTYPE html>\n<html lang=\"en\">\n<head><title>T</title></head>\n</html>", want: []Token{
		tok(Preproc, "<!DOCTYPE html>"), tok(Punctuation, "<"), tok(Tag, "html"), tok(Attribute, "lang"),
		tok(Operator, "="), tok(String, `"en"`), tok(Punctuation, ">"), tok(Tag, "head"), tok(Tag, "title"), tok(Text, "T"),
		tok(Punctuation, "</"), tok(Tag, "title"), tok(Tag, "head"), tok(Tag, "html"),
	}},
	{name: "attributes", src: "<input type=\"text\" disabled value='x' data-id=\"1\" aria-label=\"a b\"/>", want: []Token{
		tok(Tag, "input"), tok(Attribute, "type"), tok(String, `"text"`), tok(Attribute, "disabled"), tok(Attribute, "value"),
		tok(String, "'x'"), tok(Attribute, "data-id"), tok(String, `"1"`), tok(Attribute, "aria-label"), tok(String, `"a b"`),
		tok(Punctuation, "/>"),
	}},
	{name: "entities and text", src: "<p>a &amp; b &lt; c &#169; &#x1F600; & d</p>", want: []Token{
		tok(Text, "a "), tok(StringEscape, "&amp;"), tok(Text, " b "), tok(StringEscape, "&lt;"),
		tok(StringEscape, "&#169;"), tok(StringEscape, "&#x1F600;"), tok(Text, " & d"),
	}},
	{name: "text that only looks like markup", src: "<p>1 < 2 and x < y</p>", want: []Token{
		tok(Text, "1 < 2 and x < y"), tok(Tag, "p"),
	}},
	{name: "comments", src: "<!-- one -->\n<!-- multi\nline -->\n<p>", want: []Token{
		tok(Comment, "<!-- one -->"), tok(Comment, "<!-- multi"), tok(Comment, "line -->"), tok(Tag, "p"),
	}},
	{name: "tags across lines", src: "<a href=\"/x\"\n   class=\"btn\"\n   target=_blank>link</a>", want: []Token{
		tok(Tag, "a"), tok(Attribute, "href"), tok(String, `"/x"`), tok(Attribute, "class"), tok(String, `"btn"`),
		tok(Attribute, "target"), tok(Text, "link"), tok(Tag, "a"),
	}},
	{name: "attribute values across lines", src: "<div class=\"a\n  b\" id=\"x\">t</div>", want: []Token{
		tok(String, `"a`), tok(String, `  b"`), tok(Attribute, "id"), tok(String, `"x"`), tok(Text, "t"), tok(Tag, "div"),
	}},
	{name: "scripts", src: "<script>\nconst x = 1; // js\nif (a < b) {}\n</script>\n<p>", want: []Token{
		tok(Tag, "script"), tok(Keyword, "const"), tok(Comment, "// js"), tok(Keyword, "if"), tok(Operator, "<"),
		tok(Punctuation, "</"), tok(Tag, "script"), tok(Tag, "p"),
	}},
	{name: "script types", src: "<script type=\"module\">let a</script>\n<script type=\"application/json\">{\"a\": 1}</script>\n<script lang=\"ts\">let b: string</script>", want: []Token{
		tok(Keyword, "let"), tok(Tag, "script"), tok(Text, `{"a": 1}`), tok(Tag, "script"), tok(Keyword, "let"),
		tok(BuiltinType, "string"),
	}},
	{name: "script state across lines", src: "<script>\nconst s = `multi\nline`;\n</script>", want: []Token{
		tok(String, "`multi"), tok(String, "line`"), tok(Tag, "script"),
	}},
	{name: "styles", src: "<style>\nbody { color: red; }\n.x:hover { margin: 0 auto }\n</style>", want: []Token{
		tok(Tag, "style"), tok(Tag, "body"), tok(Property, "color"), tok(Type, ".x"), tok(Decorator, ":hover"),
		tok(Property, "margin"), tok(Number, "0"), tok(Tag, "style"),
	}},
	{name: "closing tags ignore case", src: "<SCRIPT>var x</SCRIPT><p>", want: []Token{
		tok(Tag, "SCRIPT"), tok(Keyword, "var"), tok(Tag, "SCRIPT"), tok(Tag, "p"),
	}},
	{name: "unclosed comment", src: "<!-- open\nstill", open: true, want: []Token{
		tok(Comment, "<!-- open"), tok(Comment, "still"),
	}},
	{name: "unclosed script", src: "<script>\nlet x", open: true, want: []Token{
		tok(Tag, "script"), tok(Keyword, "let"),
	}},
	{name: "unquoted attribute values", src: "<input type=text value = 42 disabled><br class=x/>\n<script type=module>let a = 1</script>", want: []Token{
		tok(Punctuation, "<"), tok(Tag, "input"), tok(Attribute, "type"), tok(Operator, "="), tok(String, "text"),
		tok(Attribute, "value"), tok(Operator, "="), tok(String, "42"), tok(Attribute, "disabled"), tok(Punctuation, ">"),
		tok(Punctuation, "<"), tok(Tag, "br"), tok(Attribute, "class"), tok(Operator, "="), tok(String, "x"),
		tok(Punctuation, "/>"), tok(Punctuation, "<"), tok(Tag, "script"), tok(Attribute, "type"), tok(Operator, "="),
		tok(String, "module"), tok(Punctuation, ">"), tok(Keyword, "let"), tok(Variable, "a"), tok(Number, "1"),
	}},
}
