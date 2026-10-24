package syntax

import "testing"

func TestVue(t *testing.T) {
	testLanguage(t, "vue", []string{"App.vue"}, vueTests)
}

var vueTests = []kindCase{
	{name: "templates", src: "<template>\n  <div :class=\"{ active: isActive }\" @click=\"toggle()\" v-if=\"show\">{{ msg.toUpperCase() }}</div>\n</template>", want: []Token{
		tok(Tag, "template"), tok(Tag, "div"), tok(Attribute, ":class"), tok(Punctuation, "{"), tok(Variable, "active"),
		tok(Variable, "isActive"), tok(Attribute, "@click"), tok(Function, "toggle"), tok(Attribute, "v-if"),
		tok(Variable, "show"), tok(Punctuation, "{{"), tok(Variable, "msg"), tok(Function, "toUpperCase"),
		tok(Punctuation, "}}"), tok(Tag, "div"), tok(Tag, "template"),
	}},
	{name: "loops and slots", src: "<li v-for=\"item in items\" :key=\"item.id\" #default=\"{ row }\">", want: []Token{
		tok(Attribute, "v-for"), tok(Variable, "item"), tok(Keyword, "in"), tok(Variable, "items"), tok(Attribute, ":key"),
		tok(Property, "id"), tok(Attribute, "#default"), tok(Variable, "row"),
	}},
	{name: "plain attributes", src: "<p class=\"a b\" title=\"{{ x }}\">", want: []Token{
		tok(Attribute, "class"), tok(String, `"a b"`), tok(Attribute, "title"), tok(String, `"{{ x }}"`),
	}},
	{name: "interpolation across lines", src: "<p>{{\n  a + b\n}}</p>", want: []Token{
		tok(Punctuation, "{{"), tok(Variable, "a"), tok(Operator, "+"), tok(Variable, "b"), tok(Punctuation, "}}"),
		tok(Tag, "p"),
	}},
	{name: "single braces are text", src: "<p>{ not interpolated }</p>", want: []Token{
		tok(Text, "{ not interpolated }"),
	}},
	{name: "scripts", src: "<script setup lang=\"ts\">\nimport { ref } from 'vue'\nconst n: number = 1\n</script>", want: []Token{
		tok(Tag, "script"), tok(Attribute, "setup"), tok(Attribute, "lang"), tok(String, `"ts"`), tok(Keyword, "import"),
		tok(Keyword, "from"), tok(Keyword, "const"), tok(BuiltinType, "number"), tok(Tag, "script"),
	}},
	{name: "styles", src: "<style scoped lang=\"scss\">\n$c: red;\n.a { &:hover { color: $c; } }\n</style>", want: []Token{
		tok(Attribute, "scoped"), tok(Variable, "$c"), tok(Type, ".a"), tok(Keyword, "&"), tok(Decorator, ":hover"),
		tok(Property, "color"), tok(Variable, "$c"), tok(Tag, "style"),
	}},
}
