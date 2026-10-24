package syntax

import "testing"

func TestSCSS(t *testing.T) {
	testLanguage(t, "scss", []string{"sass", "less", "theme.scss"}, scssTests)
}

var scssTests = []kindCase{
	{name: "variables", src: "$primary: #333 !default;\n$map: (key: 1, other: 2);\na { color: $primary; }", want: []Token{
		tok(Variable, "$primary"), tok(Punctuation, ":"), tok(Number, "#333"), tok(Keyword, "!default"), tok(Variable, "$map"),
		tok(Property, "color"), tok(Variable, "$primary"),
	}},
	{name: "nesting and the parent selector", src: ".card {\n  &__title { font-weight: bold; }\n  &:hover, &.active { color: red; }\n  .nested & { x: y }\n}", want: []Token{
		tok(Type, ".card"), tok(Keyword, "&"), tok(Property, "font-weight"), tok(Keyword, "&"), tok(Decorator, ":hover"),
		tok(Keyword, "&"), tok(Type, ".active"), tok(Property, "color"), tok(Type, ".nested"), tok(Keyword, "&"),
		tok(Property, "x"),
	}},
	{name: "mixins and functions", src: "@mixin flex($dir: row) { display: flex; }\n.a { @include flex(column); }\n@function double($n) { @return $n * 2; }", want: []Token{
		tok(Keyword, "@mixin"), tok(Function, "flex"), tok(Variable, "$dir"), tok(Property, "display"), tok(Keyword, "@include"),
		tok(Function, "flex"), tok(Keyword, "@function"), tok(Function, "double"), tok(Variable, "$n"), tok(Keyword, "@return"),
		tok(Variable, "$n"), tok(Operator, "*"), tok(Number, "2"),
	}},
	{name: "interpolation in selectors", src: ".icon-#{$name} {}", want: []Token{
		tok(Type, ".icon-"), tok(Variable, "#{$name}"), tok(Punctuation, "{"),
	}},
	{name: "interpolation in declarations", src: ".icon-#{$name} { width: #{$size}px; }", want: []Token{
		tok(Type, ".icon-"), tok(Variable, "#{$name}"), tok(Property, "width"), tok(Punctuation, ":"),
		tok(Variable, "#{$size}"), tok(Punctuation, ";"),
	}},
	{name: "interpolation in nested declarations", src: ".a {\n  margin: #{$y} #{$x};\n  #{$prop}: red;\n  padding-#{$side}: 0;\n  .b-#{$c}:hover { color: blue }\n}", want: []Token{
		tok(Property, "margin"), tok(Variable, "#{$y}"), tok(Variable, "#{$x}"), tok(Variable, "#{$prop}"),
		tok(Punctuation, ":"), tok(Text, " red"), tok(Property, "padding-"), tok(Variable, "#{$side}"),
		tok(Punctuation, ":"), tok(Number, "0"), tok(Type, ".b-"), tok(Variable, "#{$c}"), tok(Decorator, ":hover"),
		tok(Property, "color"),
	}},
	{name: "comments", src: "// line comment\n/* block */\na { b: c; } // trailing", want: []Token{
		tok(Comment, "// line comment"), tok(Comment, "/* block */"), tok(Tag, "a"), tok(Comment, "// trailing"),
	}},
	{name: "placeholders and extends", src: "%btn { padding: 0 }\n.b { @extend %btn; }", want: []Token{
		tok(Type, "%btn"), tok(Property, "padding"), tok(Keyword, "@extend"),
	}},
	{name: "control directives", src: "@each $c in red, blue { .t-#{$c} { color: $c } }\n@if $a == 1 { } @else { }", want: []Token{
		tok(Keyword, "@each"), tok(Variable, "$c"), tok(Type, ".t-"), tok(Variable, "#{$c}"), tok(Keyword, "@if"),
		tok(Variable, "$a"), tok(Operator, "=="), tok(Keyword, "@else"),
	}},
}
