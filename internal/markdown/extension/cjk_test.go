package extension_test

import (
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/internal/spectest"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestEastAsianLineBreaks checks which soft line breaks each style leaves
// out, as the HTML of the tests renders them.
func TestEastAsianLineBreaks(t *testing.T) {
	const (
		simple = extension.EastAsianLineBreaksSimple
		css3   = extension.EastAsianLineBreaksCSS3Draft
		none   = extension.EastAsianLineBreaksNone
	)
	p := parser.NewParser()
	for _, tt := range []struct {
		name  string
		style extension.EastAsianLineBreaks
		src   string
		want  string
	}{
		{"none", none, "日本\n語", "日本\n語"},
		{"wide", simple, "日本\n語", "日本語"},
		{"wide and narrow", simple, "日本\nabc", "日本\nabc"},
		{"narrow and wide", simple, "abc\n日本", "abc\n日本"},
		{"Hangul", simple, "한\n국", "한국"},
		{"CJK punctuation", simple, "。\n語", "。語"},
		{"fullwidth Latin", simple, "ａ\nｂ", "ａ\nｂ"},
		{"hard line break", simple, "日本  \n語", "日本<br>\n語"},
		{"before emphasis", simple, "日本\n*語*", "日本<em>語</em>"},
		{"after emphasis", simple, "*日本*\n語", "<em>日本</em>\n語"},
		{"before code", simple, "日本\n`語`", "日本<code>語</code>"},
		{"end of link text", simple, "[日本\n](/u)語", `<a href="/u">日本</a>語`},
		{"alt text", simple, "![日本\n語](/i)", `<img src="/i" alt="日本語">`},
		{"CSS3 wide", css3, "日本\n語", "日本語"},
		{"CSS3 wide and narrow", css3, "日本\nabc", "日本abc"},
		{"CSS3 narrow and wide", css3, "abc\n日本", "abc日本"},
		{"CSS3 Hangul", css3, "한\n국", "한\n국"},
		{"CSS3 fullwidth", css3, "ａ\nｂ", "ａｂ"},
		{"CSS3 zero width space", css3, "a\u200b\nb", "a\u200bb"},
		{"CSS3 punctuation", css3, "abc\n!", "abc!"},
		{"CSS3 narrow", css3, "abc\ndef", "abc\ndef"},

		// Unlike goldmark, which leaves out every soft line break before
		// inline content that is not text, the characters around it
		// decide.
		{"narrow before emphasis", simple, "a\n*b*", "a\n<em>b</em>"},
		{"before raw HTML", simple, "日本\n<b>語", "日本\n<!-- raw HTML omitted -->語"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := spectest.HTML(p.Parse(tt.src), spectest.Options{KeepsSoftLineBreak: tt.style.KeepsSoftLineBreak})
			if want := "<p>" + tt.want + "</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestCJK checks that CJK lets a backslash escape a space, with the soft
// line breaks that EastAsianLineBreaksSimple leaves out and without.
func TestCJK(t *testing.T) {
	const src = "日本\n語 **“日本”**\\ 語"
	for _, tt := range []struct {
		name  string
		exts  []parser.Extension
		style extension.EastAsianLineBreaks
		want  string
	}{
		{"CJK, simple line breaks", []parser.Extension{extension.CJK}, extension.EastAsianLineBreaksSimple, "日本語 <strong>“日本”</strong>語"},
		{"CJK", []parser.Extension{extension.CJK}, extension.EastAsianLineBreaksNone, "日本\n語 <strong>“日本”</strong>語"},
		{"neither", nil, extension.EastAsianLineBreaksNone, "日本\n語 <strong>“日本”</strong>\\ 語"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := parser.NewParser(parser.WithExtensions(tt.exts...)).Parse(src)
			got := spectest.HTML(doc, spectest.Options{KeepsSoftLineBreak: tt.style.KeepsSoftLineBreak})
			if want := "<p>" + tt.want + "</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestCJKLinkify checks that an extended autolink does not start after an
// escaped space, as no word does.
func TestCJKLinkify(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.GFM, extension.CJK))
	for _, tt := range []struct{ src, want string }{
		{`a\ www.x.org`, "<p>awww.x.org</p>\n"},
		{`a\\ www.x.org`, `<p>a\ <a href="http://www.x.org">www.x.org</a></p>` + "\n"},
		{"a www.x.org", `<p>a <a href="http://www.x.org">www.x.org</a></p>` + "\n"},
	} {
		if got := convert(t, p, tt.src); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.src, got, tt.want)
		}
	}
}

// TestCJKParserOnly checks that the extension of a parser adds escaped
// spaces.
func TestCJKParserOnly(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.CJK))
	if got := p.Parse(`a\ b`).FirstChild().FirstChild(); got.NextSibling() != nil {
		t.Errorf("paragraph of %q holds more than one text", `a\ b`)
	}
}
