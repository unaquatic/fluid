package extension_test

import (
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestCJK checks what the extension for Chinese, Japanese, and Korean adds,
// with each option and with none.
func TestCJK(t *testing.T) {
	const src = "日本\n語 **“日本”**\\ 語"
	for _, tt := range []struct {
		name string
		ext  extension.Extension
		want string
	}{
		{"CJK", extension.CJK, "日本語 <strong>“日本”</strong>語"},
		{"line breaks", extension.NewCJK(extension.WithEastAsianLineBreaks(html.EastAsianLineBreaksSimple)), "日本語 <strong>“日本”</strong>\\ 語"},
		{"escaped space", extension.NewCJK(extension.WithEscapedSpace()), "日本\n語 <strong>“日本”</strong>語"},
		{"nothing", extension.NewCJK(), "日本\n語 <strong>“日本”</strong>\\ 語"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := convert(t, md.New(md.WithExtensions(tt.ext)), src), "<p>"+tt.want+"</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}

	// A style of none leaves the renderer's own.
	m := md.New(
		md.WithRendererOptions(html.WithEastAsianLineBreaks(html.EastAsianLineBreaksSimple)),
		md.WithExtensions(extension.NewCJK(extension.WithEscapedSpace())),
	)
	if got := convert(t, m, "日本\n語"); got != "<p>日本語</p>\n" {
		t.Errorf("with the renderer's style: %q", got)
	}
}

// TestCJKLinkify checks that an extended autolink does not start after an
// escaped space, as no word does.
func TestCJKLinkify(t *testing.T) {
	m := md.New(md.WithExtensions(extension.GFM, extension.CJK))
	for _, tt := range []struct{ src, want string }{
		{`a\ www.x.org`, "<p>awww.x.org</p>\n"},
		{`a\\ www.x.org`, `<p>a\ <a href="http://www.x.org">www.x.org</a></p>` + "\n"},
		{"a www.x.org", `<p>a <a href="http://www.x.org">www.x.org</a></p>` + "\n"},
	} {
		if got := convert(t, m, tt.src); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.src, got, tt.want)
		}
	}
}

// TestCJKParserOnly checks that the extension of a parser on its own adds
// escaped spaces.
func TestCJKParserOnly(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.CJK))
	if got := p.Parse(`a\ b`).FirstChild().FirstChild(); got.NextSibling() != nil {
		t.Errorf("paragraph of %q holds more than one text", `a\ b`)
	}
}
