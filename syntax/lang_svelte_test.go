package syntax

import "testing"

func TestSvelte(t *testing.T) {
	testLanguage(t, "svelte", []string{"App.svelte"}, svelteTests)
}

var svelteTests = []kindCase{
	{name: "scripts", src: "<script>\n  export let name = 'x';\n  $: doubled = n * 2;\n</script>", want: []Token{
		tok(Tag, "script"), tok(Keyword, "export"), tok(Keyword, "let"), tok(Variable, "name"), tok(String, "'x'"),
		tok(Variable, "doubled"), tok(Operator, "*"), tok(Tag, "script"),
	}},
	{name: "expressions", src: "<h1>Hello {name}!</h1>\n<p>{a + b} and {fn(c)}</p>", want: []Token{
		tok(Tag, "h1"), tok(Text, "Hello "), tok(Punctuation, "{"), tok(Variable, "name"), tok(Punctuation, "}"),
		tok(Text, "!"), tok(Punctuation, "{"), tok(Variable, "a"), tok(Operator, "+"), tok(Punctuation, "}"),
		tok(Function, "fn"),
	}},
	{name: "blocks", src: "{#if count > 10}\n  <p>big</p>\n{:else if count > 5}\n  <p>mid</p>\n{:else}\n{/if}\n{#each items as item}{item}{/each}\n{#await p then v}{v}{/await}\n{@html raw}", want: []Token{
		tok(Keyword, "#if"), tok(Variable, "count"), tok(Operator, ">"), tok(Number, "10"), tok(Keyword, ":else"),
		tok(Keyword, "if"), tok(Keyword, ":else"), tok(Keyword, "/if"), tok(Keyword, "#each"), tok(Variable, "items"),
		tok(Keyword, "as"), tok(Variable, "item"), tok(Keyword, "/each"), tok(Keyword, "#await"), tok(Keyword, "/await"),
		tok(Keyword, "@html"), tok(Variable, "raw"),
	}},
	{name: "attribute expressions and directives", src: "<input bind:value={text} on:input={handle} class:active={on} {disabled} />", want: []Token{
		tok(Attribute, "bind:value"), tok(Punctuation, "{"), tok(Variable, "text"), tok(Attribute, "on:input"),
		tok(Variable, "handle"), tok(Attribute, "class:active"), tok(Variable, "on"), tok(Variable, "disabled"),
		tok(Punctuation, "/>"),
	}},
	{name: "handlers across lines", src: "<button on:click={() => {\n  n++\n}} class=\"x\">go</button>", want: []Token{
		tok(Attribute, "on:click"), tok(Variable, "n"), tok(Operator, "++"), tok(Attribute, "class"), tok(String, `"x"`),
		tok(Text, "go"), tok(Tag, "button"),
	}},
	{name: "braces inside strings", src: "<p>{'}' + x}</p>", want: []Token{
		tok(String, "'}'"), tok(Variable, "x"), tok(Punctuation, "}"), tok(Tag, "p"),
	}},
	{name: "typescript and styles", src: "<script lang=\"ts\">\n  let n: number = 1;\n</script>\n<style>\n  h1 { color: purple; }\n</style>", want: []Token{
		tok(Type, "number"), tok(Tag, "h1"), tok(Property, "color"),
	}},
}
