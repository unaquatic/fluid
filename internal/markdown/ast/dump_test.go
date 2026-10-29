package ast_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// dump returns what Dump writes for n and src.
func dump(t *testing.T, n ast.Node, src string) string {
	t.Helper()
	var b strings.Builder
	if err := ast.Dump(&b, n, src); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestDump checks the lines that Dump writes: kinds, fields that are not
// zero, attributes, spans, the source of blocks, and line breaks.
func TestDump(t *testing.T) {
	const src = "# Title {#t .big}\n\nSome *emph*\ntext  \nend\n\n- [x] a"
	doc := parser.NewParser(parser.WithAttribute()).Parse(src)
	const want = `Document [0:50]
  Heading Level=1 {id="t" class="big"} [0:17] "# Title {#t .big}"
    Text "Title"
  Paragraph [19:41] "Some *emph*\ntext  \nend"
    Text "Some "
    Emphasis Level=1
      Text "emph"
    Text "" softbreak
    Text "text" hardbreak
    Text "end"
  List Marker='-' IsTight=true [43:50] "- [x] a"
    ListItem Offset=2 [43:50] "- [x] a"
      Paragraph [45:50] "[x] a"
        Text "[x] a"
`
	if got := dump(t, doc, src); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	// Without the source, blocks show only their spans.
	if got, want := dump(t, doc.LastChild(), ""), "List Marker='-' IsTight=true [43:50]\n  ListItem Offset=2 [43:50]\n    Paragraph [45:50]\n      Text \"[x] a\"\n"; got != want {
		t.Errorf("without source:\n%s\nwant:\n%s", got, want)
	}
}

// TestDumpValues checks how Dump writes the values of fields and
// attributes, and the span of inline content that has one.
func TestDumpValues(t *testing.T) {
	table := &ast.Table{Alignments: []ast.Alignment{ast.AlignLeft, ast.AlignNone}}
	link := &ast.AutoLink{AutoLinkType: ast.AutoLinkEmail, URL: "mailto:a@b.c", Label: "a@b.c"}
	link.SetSourceSpan(ast.Span{Start: 1, End: 6})
	link.SetAttribute("data-n", 2.5)
	link.SetAttribute("data-l", []any{"x", true, nil})
	item := &ast.ListItem{Task: ast.Unchecked}
	doc := &ast.Document{Meta: map[string]any{"b": 2, "a": "x"}}
	for _, tt := range []struct {
		n    ast.Node
		want string
	}{
		{table, "Table Alignments=[left none] [0:0] \"\"\n"},
		{link, `AutoLink AutoLinkType=email URL="mailto:a@b.c" Label="a@b.c" {data-n=2.5 data-l=["x" true nil]} [1:6]` + "\n"},
		{item, "ListItem Task=unchecked [0:0] \"\"\n"},
		{doc, "Document Meta=map[a:x b:2] [0:0]\n"},
	} {
		if got := dump(t, tt.n, "src"); got != tt.want {
			t.Errorf("got  %q\nwant %q", got, tt.want)
		}
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("full") }

// TestDumpError checks that Dump returns the error of a write.
func TestDumpError(t *testing.T) {
	if err := ast.Dump(failWriter{}, parser.NewParser().Parse("a"), "a"); err == nil || err.Error() != "full" {
		t.Errorf("error %v, want full", err)
	}
}
