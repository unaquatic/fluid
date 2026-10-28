package html_test

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// render returns the HTML of src with renderer options opts.
func render(t *testing.T, src string, opts ...html.Option) string {
	t.Helper()
	var b strings.Builder
	if err := md.New(md.WithRendererOptions(opts...)).Convert(src, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestOptions checks what each option changes.
func TestOptions(t *testing.T) {
	for _, tt := range []struct {
		name, src string
		opts      []html.Option
		want      string
	}{
		{"raw HTML omitted", "<div>x</div>\n\na <b>c</b>", nil, "<!-- raw HTML omitted -->\n<p>a <!-- raw HTML omitted -->c<!-- raw HTML omitted --></p>\n"},
		{"raw HTML unsafe", "<div>x</div>\n\na <b>c</b>", []html.Option{html.WithUnsafe()}, "<div>x</div>\n<p>a <b>c</b></p>\n"},
		{"dangerous URLs blanked", "[a](javascript:alert(1)) ![b](data:text/html,x) ![c](data:image/png;base64,x)", nil,
			`<p><a href="">a</a> <img src="" alt="b"> <img src="data:image/png;base64,x" alt="c"></p>` + "\n"},
		{"dangerous URLs unsafe", "[a](javascript:x)", []html.Option{html.WithUnsafe()}, `<p><a href="javascript:x">a</a></p>` + "\n"},
		{"HTML5", "a  \nb\n\n***", nil, "<p>a<br>\nb</p>\n<hr>\n"},
		{"XHTML", "a  \nb\n\n***", []html.Option{html.WithXHTML()}, "<p>a<br />\nb</p>\n<hr />\n"},
		{"hard wraps", "a\nb", []html.Option{html.WithHardWraps()}, "<p>a<br>\nb</p>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := render(t, tt.src, tt.opts...); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// setAttributes sets its attributes on every node but documents and text.
type setAttributes []markdown.Attribute

func (attrs setAttributes) Transform(doc *markdown.Document) {
	_ = markdown.Walk(doc, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		switch n.(type) {
		case *markdown.Document, *markdown.Text:
		default:
			if entering {
				for _, a := range attrs {
					n.SetAttribute(a.Name, a.Value)
				}
			}
		}
		return markdown.WalkContinue, nil
	})
}

// renderWith returns the HTML of src, of whose nodes setAttributes sets
// attrs.
func renderWith(t *testing.T, src string, attrs setAttributes) string {
	t.Helper()
	var b strings.Builder
	m := md.New(md.WithParserOptions(parser.WithASTTransformer(attrs, 100)))
	if err := m.Convert(src, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestAttributes checks which elements get the attributes of their nodes,
// which their filters allow.
func TestAttributes(t *testing.T) {
	attrs := setAttributes{{Name: "class", Value: "c"}, {Name: "onclick", Value: "x"}, {Name: "data-n", Value: 2.0}}
	const a = ` class="c" data-n="2"`
	for _, tt := range []struct {
		name, src, want string
	}{
		{"heading", "# h", "<h1" + a + ">h</h1>\n"},
		{"block quote", "> q", "<blockquote" + a + ">\n<p" + a + ">q</p>\n</blockquote>\n"},
		{"list", "3. x\n4. y", "<ol start=\"3\"" + a + ">\n<li" + a + ">x</li>\n<li" + a + ">y</li>\n</ol>\n"},
		{"thematic break", "***", "<hr" + a + ">\n"},
		{"inline", "*e* **s** `c`", "<p" + a + "><em" + a + ">e</em> <strong" + a + ">s</strong> <code" + a + ">c</code></p>\n"},
		{"links", `[l](/u "t") ![i](/s) <http://h>`,
			`<p` + a + `><a href="/u" title="t"` + a + `>l</a> <img src="/s" alt="i"` + a + `> <a href="http://h"` + a + `>http://h</a></p>` + "\n"},
		{"none", "    code\n\n```\nf\n```\n\n<div></div>\n\nx <b>",
			"<pre><code>code\n</code></pre>\n<pre><code>f\n</code></pre>\n<!-- raw HTML omitted -->\n<p" + a + ">x <!-- raw HTML omitted --></p>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderWith(t, tt.src, attrs); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAttributeValues checks how each type of value renders, and that
// values of other types, and names that HTML does not allow, do not.
func TestAttributeValues(t *testing.T) {
	attrs := setAttributes{
		{Name: "data-b", Value: true},
		{Name: "data-f", Value: 1.5},
		{Name: "data-i", Value: 7},
		{Name: "data-y", Value: []byte("y")},
		{Name: "data-q", Value: `a"<&`},
		{Name: "data-l", Value: []any{1}},
		{Name: "data-z", Value: nil},
		{Name: `data-x"y`, Value: "z"},
		{Name: "data-s p", Value: "z"},
	}
	want := `<h1 data-b="true" data-f="1.5" data-i="7" data-y="y" data-q="a&quot;&lt;&amp;">h</h1>` + "\n"
	if got := renderWith(t, "# h", attrs); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestAttributeFilter checks the names that filters allow.
func TestAttributeFilter(t *testing.T) {
	base := html.NewAttributeFilter("a")
	for _, tt := range []struct {
		name   string
		filter html.AttributeFilter
		allows map[string]bool
	}{
		{"new", base, map[string]bool{"a": true, "b": false, "data-b": true, "": false}},
		{"extended", base.Extend("b"), map[string]bool{"a": true, "b": true, "c": false}},
		{"none", html.NewAttributeFilter(), map[string]bool{"a": false, "data-a": true}},
		{"zero", html.AttributeFilter{}, map[string]bool{"a": true, "onclick": true}},
		{"zero extended", html.AttributeFilter{}.Extend("a"), map[string]bool{"b": true}},
	} {
		for name, want := range tt.allows {
			if got := tt.filter.Allows(name); got != want {
				t.Errorf("%s: Allows(%q) = %v, want %v", tt.name, name, got, want)
			}
		}
	}
}

// TestEastAsianLineBreaks checks which soft line breaks each style leaves
// out.
func TestEastAsianLineBreaks(t *testing.T) {
	const (
		simple = html.EastAsianLineBreaksSimple
		css3   = html.EastAsianLineBreaksCSS3Draft
		none   = html.EastAsianLineBreaksNone
	)
	for _, tt := range []struct {
		name  string
		style html.EastAsianLineBreaks
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
			if got, want := render(t, tt.src, html.WithEastAsianLineBreaks(tt.style)), "<p>"+tt.want+"</p>\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// emphasisAsSpan renders emphasis as a span element.
type emphasisAsSpan struct{}

func (emphasisAsSpan) RegisterFuncs(reg html.NodeRendererFuncRegisterer) {
	reg.Register(markdown.KindEmphasis, func(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if entering {
			w.WriteString("<span>")
		} else {
			w.WriteString("</span>")
		}
		return markdown.WalkContinue, nil
	})
}

// TestNodeRenderer checks that a node renderer of a lower priority than
// CommonMark's renders the kinds it registers, and one of a higher priority
// does not.
func TestNodeRenderer(t *testing.T) {
	if got := render(t, "*a*", html.WithNodeRenderer(emphasisAsSpan{}, 500)); got != "<p><span>a</span></p>\n" {
		t.Errorf("priority 500: %q", got)
	}
	if got := render(t, "*a*", html.WithNodeRenderer(emphasisAsSpan{}, 2000)); got != "<p><em>a</em></p>\n" {
		t.Errorf("priority 2000: %q", got)
	}
}
