package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/internal/spectest"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestCommonMark checks the examples of the CommonMark specification,
// version 0.31.2, which render raw HTML and XHTML.
func TestCommonMark(t *testing.T) {
	m := md.New(md.WithRendererOptions(html.WithXHTML(), html.WithUnsafe()))
	spectest.Run(t, "testdata/spec.json", func(src string) string {
		var b strings.Builder
		if err := m.Convert(src, &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	})
}

// TestLinks checks that a document holds the first definition of each link
// label, in order, with its label as written and its range in the source.
func TestLinks(t *testing.T) {
	const src = "[Foo]: /a \"t\"  \n[foo]: /b\n [Bar  baz]:\n  <c d>\n\n[foo] [bar baz]\n"
	d := parser.NewParser().Parse(src)
	want := []*markdown.LinkDef{
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
	m := md.New(md.WithParserOptions(parser.WithEscapedSpace()))
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
			var b strings.Builder
			if err := m.Convert(tt.src, &b); err != nil {
				t.Fatal(err)
			}
			if got, want := b.String(), "<p>"+tt.want+"</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
	var b strings.Builder
	if err := md.Convert(`a\ b`, &b); err != nil || b.String() != "<p>a\\ b</p>\n" {
		t.Errorf("without WithEscapedSpace: %q, %v", b.String(), err)
	}
}

// span returns the range of the first occurrence of sub in src.
func span(src, sub string) markdown.Span {
	i := strings.Index(src, sub)
	if i < 0 {
		panic("no " + sub)
	}
	return markdown.Span{Start: i, End: i + len(sub)}
}
