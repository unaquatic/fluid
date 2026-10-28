package ast_test

import (
	"fmt"
	"os"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

func ExampleDump() {
	const src = "# Notes\n\n- *one*\n- two"
	doc := parser.NewParser().Parse(src)
	if err := ast.Dump(os.Stdout, doc, src); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Document [0:22]
	//   Heading Level=1 [0:7] "# Notes"
	//     Text "Notes"
	//   List Marker='-' IsTight=true [9:22] "- *one*\n- two"
	//     ListItem Offset=2 [9:16] "- *one*"
	//       Paragraph [11:16] "*one*"
	//         Emphasis Level=1
	//           Text "one"
	//     ListItem Offset=2 [17:22] "- two"
	//       Paragraph [19:22] "two"
	//         Text "two"
}

func ExampleWalk() {
	doc := parser.NewParser().Parse("See [the spec](https://spec.commonmark.org) and [the tests](/tests).")
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if l, ok := n.(*ast.Link); ok && entering {
			fmt.Println(l.Destination)
		}
		return ast.WalkContinue, nil
	})
	// Output:
	// https://spec.commonmark.org
	// /tests
}

func ExampleLineIndex() {
	const src = "# Title\n\nA paragraph\nover two lines."
	doc := parser.NewParser().Parse(src)
	x := ast.NewLineIndex(src)
	for b := range ast.Children(doc) {
		line, col := x.Position(b.SourceSpan().Start)
		fmt.Printf("%s at %d:%d\n", b.Kind(), line, col)
	}
	// Output:
	// Heading at 1:1
	// Paragraph at 3:1
}
