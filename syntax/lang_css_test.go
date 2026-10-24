package syntax

import "testing"

func TestCSS(t *testing.T) {
	testLanguage(t, "css", []string{"style.css"}, cssTests)
}

var cssTests = []kindCase{
	{name: "selectors", src: "h1, .title > a:hover::before, #main ~ p + span[data-x=\"1\"], * {}", want: []Token{
		tok(Tag, "h1"), tok(Punctuation, ","), tok(Type, ".title"), tok(Operator, ">"), tok(Tag, "a"), tok(Decorator, ":hover"),
		tok(Decorator, "::before"), tok(Type, "#main"), tok(Operator, "~"), tok(Tag, "p"), tok(Operator, "+"), tok(Tag, "span"),
		tok(Punctuation, "["), tok(Attribute, "data-x"), tok(Operator, "="), tok(String, `"1"`), tok(Punctuation, "]"),
		tok(Operator, "*"),
	}},
	{name: "attribute selectors", src: "a[href^=\"http\"], a[target], input[type=text] {}", want: []Token{
		tok(Attribute, "href"), tok(Operator, "^="), tok(String, `"http"`), tok(Attribute, "target"), tok(Attribute, "type"),
		tok(Operator, "="), tok(String, "text"),
	}},
	{name: "declarations", src: "a {\n  color: #fff;\n  margin: 0 auto 1.5em -2px;\n  width: calc(100% - 2rem);\n  font: 12px/1.5 \"Inter\", sans-serif;\n  --gap: 4px;\n  gap: var(--gap) !important;\n}", want: []Token{
		tok(Property, "color"), tok(Punctuation, ":"), tok(Number, "#fff"), tok(Punctuation, ";"), tok(Property, "margin"),
		tok(Number, "0"), tok(Text, "auto"), tok(Number, "1.5em"), tok(Number, "-2px"), tok(Property, "width"),
		tok(Function, "calc"), tok(Number, "100%"), tok(Operator, "-"), tok(Number, "2rem"), tok(Property, "font"),
		tok(Number, "12px"), tok(Operator, "/"), tok(Number, "1.5"), tok(String, `"Inter"`), tok(Text, "sans-serif"),
		tok(Variable, "--gap"), tok(Number, "4px"), tok(Property, "gap"), tok(Function, "var"), tok(Variable, "--gap"),
		tok(Keyword, "!important"),
	}},
	{name: "at-rules", src: "@import url(\"a.css\");\n@media (max-width: 600px) and (prefers-color-scheme: dark) {\n  .x { display: none }\n}\n@font-face { font-family: X; }", want: []Token{
		tok(Keyword, "@import"), tok(Function, "url"), tok(String, `"a.css"`), tok(Keyword, "@media"), tok(Number, "600px"),
		tok(Type, ".x"), tok(Property, "display"), tok(Text, "none"), tok(Keyword, "@font-face"), tok(Property, "font-family"),
	}},
	{name: "keyframes", src: "@keyframes spin {\n  from { opacity: 0 }\n  50% { opacity: .5 }\n  to { opacity: 1 }\n}", want: []Token{
		tok(Keyword, "@keyframes"), tok(Tag, "from"), tok(Property, "opacity"), tok(Number, "0"), tok(Number, "50%"),
		tok(Number, ".5"), tok(Tag, "to"),
	}},
	{name: "functions and urls", src: "a { background: url(img/x.png) no-repeat, linear-gradient(to right, #000 0%, rgba(0,0,0,.5) 100%); }", want: []Token{
		tok(Function, "url"), tok(String, "img/x.png"), tok(Text, "no-repeat"), tok(Function, "linear-gradient"),
		tok(Number, "#000"), tok(Number, "0%"), tok(Function, "rgba"), tok(Number, ".5"), tok(Number, "100%"),
	}},
	{name: "comments", src: "/* header */\na { /* inline */ color: red; }\n/* multi\nline */ b {}", want: []Token{
		tok(Comment, "/* header */"), tok(Comment, "/* inline */"), tok(Property, "color"), tok(Comment, "/* multi"),
		tok(Comment, "line */"), tok(Tag, "b"),
	}},
	{name: "minified", src: "a{color:red}b{x:y;z:1px}", want: []Token{
		tok(Tag, "a"), tok(Property, "color"), tok(Text, "red"), tok(Tag, "b"), tok(Property, "x"), tok(Text, "y"),
		tok(Property, "z"), tok(Number, "1px"),
	}},
	{name: "nesting", src: "a {\n  &:hover { color: red; }\n  .child { margin: 0 }\n}", want: []Token{
		tok(Keyword, "&"), tok(Decorator, ":hover"), tok(Property, "color"), tok(Type, ".child"), tok(Property, "margin"),
	}},
	{name: "values across lines", src: "a {\n  grid-template-areas:\n    \"head head\"\n    \"main side\";\n}", want: []Token{
		tok(Property, "grid-template-areas"), tok(String, `"head head"`), tok(String, `"main side"`),
	}},
	{name: "vendor prefixes", src: "a { -webkit-transition: all 1s; }", want: []Token{
		tok(Property, "-webkit-transition"), tok(Text, "all"), tok(Number, "1s"),
	}},
	{name: "dots and signs without digits", src: "a { b: .. -. +.5; }", want: []Token{
		tok(Property, "b"), tok(Number, "."), tok(Number, "-"), tok(Number, "+.5"), tok(Punctuation, ";"),
	}},
	{name: "unclosed block", src: "a {\n  color: red;", open: true, want: []Token{
		tok(Tag, "a"), tok(Property, "color"),
	}},
	{name: "unclosed comment", src: "/* open\nstill", open: true, want: []Token{
		tok(Comment, "/* open"), tok(Comment, "still"),
	}},
}
