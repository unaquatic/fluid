package md_test

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestConvert checks the package's Convert, and a Markdown with a parser
// and a renderer of its own.
func TestConvert(t *testing.T) {
	var b strings.Builder
	if err := md.Convert("# a\n\n*b*", &b); err != nil || b.String() != "<h1>a</h1>\n<p><em>b</em></p>\n" {
		t.Errorf("Convert: %q, %v", b.String(), err)
	}

	m := md.New(md.WithParser(parser.NewParser()), md.WithRenderer(html.NewRenderer(html.WithXHTML())))
	b.Reset()
	if err := m.Convert("***", &b); err != nil || b.String() != "<hr />\n" {
		t.Errorf("Convert with XHTML renderer: %q, %v", b.String(), err)
	}
}
