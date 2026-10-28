package extension_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

func Example_footnote() {
	const src = "Text.[^1]\n\n[^1]: A note."
	p := parser.NewParser(parser.WithExtensions(extension.Footnote))
	if err := ast.Dump(os.Stdout, p.Parse(src), src); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Document [0:24]
	//   Paragraph [0:9] "Text.[^1]"
	//     Text "Text."
	//     FootnoteLink Ref="1" Index=1 RefCount=1
	//   FootnoteList Count=1 [24:24] ""
	//     Footnote Ref="1" Index=1 [11:24] "[^1]: A note."
	//       Paragraph [17:24] "A note."
	//         Text "A note."
	//         FootnoteBacklink Index=1 RefCount=1
}

func Example_definitionList() {
	const src = "Term\n: A description.\n: Another."
	p := parser.NewParser(parser.WithExtensions(extension.DefinitionList))
	if err := ast.Dump(os.Stdout, p.Parse(src), src); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Document [0:32]
	//   DefinitionList Offset=2 [0:32] "Term\n: A description.\n: Another."
	//     DefinitionTerm [0:4] "Term"
	//       Text "Term"
	//     DefinitionDescription IsTight=true [5:21] ": A description."
	//       Paragraph [7:21] "A description."
	//         Text "A description."
	//     DefinitionDescription IsTight=true [22:32] ": Another."
	//       Paragraph [24:32] "Another."
	//         Text "Another."
}

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
