package parser_test

import (
	"fmt"
	"os"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

func ExampleNewParser() {
	p := parser.NewParser(parser.WithAttribute(), parser.WithAutoHeadingID())
	doc := p.Parse("# Install {.step}\n\n## Install")
	for h := range ast.Children(doc) {
		fmt.Println(h.Attributes())
	}
	// Output:
	// [{class step} {id install}]
	// [{id install-1}]
}

// issueRef parses "#" and digits as a link to the issue they number.
type issueRef struct{}

func (issueRef) Trigger() []byte { return []byte{'#'} }

func (issueRef) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
	r := s.Rest()
	n := 1
	for n < len(r) && '0' <= r[n] && r[n] <= '9' {
		n++
	}
	if n == 1 {
		return nil, 0
	}
	l := &ast.Link{Destination: "/issues/" + r[1:n]}
	l.AppendChild(l, ast.NewText(r[:n]))
	return l, n
}

func ExampleWithInlineParser() {
	const src = "Fixed in #12, not in # 3."
	p := parser.NewParser(parser.WithInlineParser(issueRef{}, 600))
	if err := ast.Dump(os.Stdout, p.Parse(src), src); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Document [0:25]
	//   Paragraph [0:25] "Fixed in #12, not in # 3."
	//     Text "Fixed in "
	//     Link Destination="/issues/12"
	//       Text "#12"
	//     Text ", not in # 3."
}
