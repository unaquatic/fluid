package syntax

import "testing"

func TestMarkdown(t *testing.T) {
	testLanguage(t, "markdown", []string{"md", "mdx", "mkd", "README.md", "docs/guide.markdown"}, markdownTests)
}

var markdownTests = []kindCase{
	{name: "headings", src: "# Title\n## Section\n###### Deep\n####### seven\n#hashtag\n   # indented", want: []Token{
		tok(Heading, "# Title"), tok(Subheading, "## Section"), tok(Subheading, "###### Deep"), tok(Text, "####### seven"),
		tok(Text, "#hashtag"), tok(Heading, "   # indented"),
	}},
	{name: "emphasis", src: "*em* _em_ **strong** __strong__ ~~del~~ snake_case_word 2 * 3 * 4", want: []Token{
		tok(Emph, "*em*"), tok(Emph, "_em_"), tok(Strong, "**strong**"), tok(Strong, "__strong__"), tok(Deleted, "~~del~~"),
		tok(Text, "snake_case_word"), tok(Text, "2 * 3 * 4"),
	}},
	{name: "code spans", src: "use `fmt.Println` and ``a `b` c`` but not `unclosed", want: []Token{
		tok(String, "`fmt.Println`"), tok(String, "``a `b` c``"), tok(Text, "`unclosed"),
	}},
	{name: "links and images", src: "[text](http://x.y) and ![alt](img.png) and [ref][1] and [bare]", want: []Token{
		tok(Text, "["), tok(Tag, "text"), tok(Text, "]("), tok(Attribute, "http://x.y"), tok(Text, ")"), tok(Text, "!["),
		tok(Tag, "alt"), tok(Attribute, "img.png"), tok(Text, "[ref][1]"), tok(Text, "[bare]"),
	}},
	{name: "lists", src: "- item\n* star\n+ plus\n1. one\n10) ten\n- [ ] todo\n- [x] done\n  - nested", want: []Token{
		tok(Keyword, "- "), tok(Text, "item"), tok(Keyword, "* "), tok(Keyword, "+ "), tok(Keyword, "1. "),
		tok(Keyword, "10) "), tok(Keyword, "- [ ]"), tok(Text, " todo"), tok(Keyword, "- [x]"), tok(Keyword, "- "),
		tok(Text, "nested"),
	}},
	{name: "inline markup in list items", src: "- **bold** item with `code`", want: []Token{
		tok(Keyword, "- "), tok(Strong, "**bold**"), tok(String, "`code`"),
	}},
	{name: "quotes", src: "> quoted *text*\n>no space", want: []Token{
		tok(Keyword, "> "), tok(Emph, "quoted *text*"), tok(Keyword, ">"), tok(Emph, "no space"),
	}},
	{name: "fenced code", src: "```go\nfunc main() {}\n```\n~~~python\ndef f(): pass\n~~~\nafter", want: []Token{
		tok(String, "```go"), tok(Keyword, "func"), tok(Function, "main"), tok(String, "```"), tok(String, "~~~python"),
		tok(Keyword, "def"), tok(Function, "f"), tok(String, "~~~"), tok(Text, "after"),
	}},
	{name: "fenced code keeps its language's state", src: "```js\nconst t = `a\nb`;\n```\n# H", want: []Token{
		tok(Keyword, "const"), tok(String, "`a"), tok(String, "b`"), tok(String, "```"), tok(Heading, "# H"),
	}},
	{name: "fence lengths and info strings", src: "````md title=x\n```inner\n````\n```unknownlang\n<b>\n```\n```{.python}\nx\n```", want: []Token{
		tok(String, "````md title=x"), tok(String, "```inner"), tok(String, "````"), tok(String, "```unknownlang"),
		tok(Text, "<b>"), tok(String, "```"), tok(String, "```{.python}"), tok(Variable, "x"), tok(String, "```"),
	}},
	{name: "a shorter fence does not close a longer one", src: "````\n```\nstill code\n````", want: []Token{
		tok(String, "````"), tok(Text, "```"), tok(Text, "still code"), tok(String, "````"),
	}},
	{name: "html comments", src: "text <!-- inline --> more\n<!-- multi\nline -->\nafter", want: []Token{
		tok(Text, "text "), tok(Comment, "<!-- inline -->"), tok(Text, " more"), tok(Comment, "<!-- multi"),
		tok(Comment, "line -->"), tok(Text, "after"),
	}},
	{name: "escapes", src: "\\*not em\\* and \\`not code\\`", want: []Token{
		tok(Text, "\\*not em\\* and \\`not code\\`"),
	}},
	{name: "unclosed fence", src: "```go\nfunc f() {", open: true, want: []Token{
		tok(String, "```go"), tok(Keyword, "func"),
	}},
}
