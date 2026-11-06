package extension_test

import (
	"fmt"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

func ExampleNewTypographer() {
	p := parser.NewParser(parser.WithExtensions(extension.NewTypographer(
		extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{
			extension.LeftDoubleQuote:  "„",
			extension.RightDoubleQuote: "“",
		}),
	)))
	doc := p.Parse(`"Don't," she said -- twice...`)
	fmt.Println(doc.FirstChild().FirstChild().(*ast.Text).Value)
	// Output:
	// „Don’t,“ she said – twice…
}

func ExampleEastAsianLineBreaks_KeepsSoftLineBreak() {
	doc := parser.NewParser(parser.WithExtensions(extension.CJK)).Parse("日本語の\n文章です。\nLatin\nwords **“強調”**\\ です。")
	// Write the text as a renderer that shows soft line breaks as spaces
	// does.
	var b strings.Builder
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(*ast.Text); ok && entering {
			b.WriteString(t.Value)
			if t.SoftLineBreak() && extension.EastAsianLineBreaksSimple.KeepsSoftLineBreak(t) {
				b.WriteString(" ")
			}
		}
		return ast.WalkContinue, nil
	})
	fmt.Println(b.String())
	// Output:
	// 日本語の文章です。 Latin words “強調”です。
}
