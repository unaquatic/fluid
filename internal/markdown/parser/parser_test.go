package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/internal/spectest"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestCommonMark checks the trees of the examples of the CommonMark
// specification, version 0.31.2, by their HTML, with raw HTML and XHTML.
func TestCommonMark(t *testing.T) {
	p := parser.NewParser()
	spectest.Run(t, "testdata/spec.json", func(src string) string {
		return spectest.HTML(p.Parse(src), spectest.Options{XHTML: true, Unsafe: true})
	})
}

// TestLinks checks that a document holds the first definition of each link
// label, in order, with its label as written and its range in the source.
func TestLinks(t *testing.T) {
	const src = "[Foo]: /a \"t\"  \n[foo]: /b\n [Bar  baz]:\n  <c d>\n\n[foo] [bar baz]\n"
	d := parser.NewParser().Parse(src)
	want := []*ast.LinkDef{
		{Span: span(src, `[Foo]: /a "t"`), Label: "Foo", URL: "/a", Title: "t"},
		{Span: span(src, "[Bar  baz]:\n  <c d>"), Label: "Bar  baz", URL: "c d"},
	}
	if !reflect.DeepEqual(d.Links, want) {
		t.Errorf("Links = %+v, want %+v", d.Links, want)
	}
}

// TestEscapedSpace checks that, with WithEscapedSpace, a backslash before a
// space escapes it, unless the space ends a line, and that it lets emphasis
// close before text that no space parts from it.
func TestEscapedSpace(t *testing.T) {
	p := parser.NewParser(parser.WithEscapedSpace())
	for _, tt := range []struct {
		name, src, want string
	}{
		{"escaped", `a\ b`, "ab"},
		{"two", `a\ \ b`, "ab"},
		{"escaped backslash", `a\\ b`, `a\ b`},
		{"emphasis", `**“日本”**\ 語`, "<strong>“日本”</strong>語"},
		{"before a soft line break", "a\\ \nb", "a\\\nb"},
		{"before a hard line break", "a\\  \nb", "a\\<br>\nb"},
		{"at the end", `a\ `, `a\`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := spectest.HTML(p.Parse(tt.src), spectest.Options{}), "<p>"+tt.want+"</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
	if got := spectest.HTML(parser.NewParser().Parse(`a\ b`), spectest.Options{}); got != "<p>a\\ b</p>\n" {
		t.Errorf("without WithEscapedSpace: %q", got)
	}
}

// TestWithoutLinkReferences checks that, with WithoutLinkReferences, a link
// reference definition stays the text of its paragraph, also under a setext
// heading's underline, and that no reference link finds it.
func TestWithoutLinkReferences(t *testing.T) {
	p := parser.NewParser(parser.WithoutLinkReferences())
	for _, tt := range []struct {
		name, src, want string
	}{
		{"definition", "[a]: /url\n\n[a]", "<p>[a]: /url</p>\n<p>[a]</p>\n"},
		{"under a heading", "[a]: /url\n===", "<h1>[a]: /url</h1>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := p.Parse(tt.src)
			if got := spectest.HTML(d, spectest.Options{}); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if len(d.Links) > 0 {
				t.Errorf("Links = %+v, want none", d.Links)
			}
		})
	}
}

// span returns the range of the first occurrence of sub in src.
func span(src, sub string) ast.Span {
	i := strings.Index(src, sub)
	if i < 0 {
		panic("no " + sub)
	}
	return ast.Span{Start: i, End: i + len(sub)}
}
