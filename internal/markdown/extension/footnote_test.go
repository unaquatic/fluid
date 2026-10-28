package extension_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// ref returns the HTML of reference refIndex to footnote index, with the
// default options.
func ref(index, refIndex int) string {
	return fmt.Sprintf(`<sup id="fnref%s:%d"><a href="#fn:%d" class="footnote-ref" role="doc-noteref">%d</a></sup>`, refNumber(refIndex), index, index, index)
}

// back returns the HTML of the link back to reference refIndex to footnote
// index, with the default options.
func back(index, refIndex int) string {
	return fmt.Sprintf(`&#160;<a href="#fnref%s:%d" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a>`, refNumber(refIndex), index)
}

// refNumber returns the number in the ID of reference refIndex: none for
// the first.
func refNumber(refIndex int) string {
	if refIndex == 0 {
		return ""
	}
	return strconv.Itoa(refIndex)
}

// notes returns the HTML of a list of footnotes, which holds items.
func notes(items ...string) string {
	return "<div class=\"footnotes\" role=\"doc-endnotes\">\n<hr>\n<ol>\n" + strings.Join(items, "") + "</ol>\n</div>\n"
}

// note returns the HTML of footnote index, whose blocks render body.
func note(index int, body string) string {
	return fmt.Sprintf("<li id=\"fn:%d\">\n%s</li>\n", index, body)
}

// TestFootnote checks how footnotes and the references to them render.
func TestFootnote(t *testing.T) {
	m := md.New(md.WithExtensions(extension.Footnote))
	for _, tt := range []struct {
		name, src, want string
	}{
		{"paragraphs", "Text.[^1]\n\n[^1]: First.\n\n    Second.", `<p>Text.<sup id="fnref:1"><a href="#fn:1" class="footnote-ref" role="doc-noteref">1</a></sup></p>
<div class="footnotes" role="doc-endnotes">
<hr>
<ol>
<li id="fn:1">
<p>First.</p>
<p>Second.&#160;<a href="#fnref:1" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a></p>
</li>
</ol>
</div>
`},
		{"numbered by first reference", "a[^3] b[^1] c[^2]\n\n[^1]: one\n[^2]: two\n[^3]: three",
			"<p>a" + ref(1, 0) + " b" + ref(2, 0) + " c" + ref(3, 0) + "</p>\n" +
				notes(note(1, "<p>three"+back(1, 0)+"</p>\n"), note(2, "<p>one"+back(2, 0)+"</p>\n"), note(3, "<p>two"+back(3, 0)+"</p>\n"))},
		{"references repeated", "a[^x] b[^x]\n\n[^x]: note",
			"<p>a" + ref(1, 0) + " b" + ref(1, 1) + "</p>\n" + notes(note(1, "<p>note"+back(1, 0)+back(1, 1)+"</p>\n"))},
		{"unreferenced", "a[^1]\n\n[^1]: one\n[^2]: unused", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"no references", "a\n\n[^1]: unused", "<p>a</p>\n"},
		{"undefined", "a[^1] [^2]\n\n[^1]: one", "<p>a" + ref(1, 0) + " [^2]</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"label case", "a[^A]\n\n[^a]: case", "<p>a[^A]</p>\n"},
		{"after bang", "a![^1]\n\n[^1]: one", "<p>a!" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"before parentheses", "a [^1](/u)\n\n[^1]: one", "<p>a " + ref(1, 0) + "(/u)</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"escaped bracket", "a[^x\\]y]\n\n[^x\\]y]: one", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"blank label", "a[^ ]\n\n[^ ]:", "<p>a[^ ]</p>\n<p>[^ ]:</p>\n"},
		{"indented", "a[^1]\n\n    [^1]: x", "<p>a[^1]</p>\n<pre><code>[^1]: x\n</code></pre>\n"},
		{"interrupts a paragraph", "a[^1]\n[^1]: one", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"lazy line", "a[^1]\n\n[^1]: one\ntwo", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>one\ntwo"+back(1, 0)+"</p>\n"))},
		{"after blank lines", "a[^1]\n\n[^1]: one\n\n\nb", "<p>a" + ref(1, 0) + "</p>\n<p>b</p>\n" + notes(note(1, "<p>one"+back(1, 0)+"</p>\n"))},
		{"empty", "a[^1]\n\n[^1]:", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, back(1, 0)))},
		{"code last", "a[^1]\n\n[^1]: text\n\n        code",
			"<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>text</p>\n<pre><code>code\n</code></pre>\n"+back(1, 0)))},
		{"defined twice", "a[^1]\n\n[^1]: first\n[^1]: second", "<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>first"+back(1, 0)+"</p>\n"))},
		{"referred to in a footnote", "a[^1]\n\n[^1]: see[^2]\n[^2]: two",
			"<p>a" + ref(1, 0) + "</p>\n" + notes(note(1, "<p>see"+ref(2, 0)+back(1, 0)+"</p>\n"), note(2, "<p>two"+back(2, 0)+"</p>\n"))},
		{"in a block quote", "> a[^1]\n>\n> [^1]: quoted\n\nb",
			"<blockquote>\n<p>a" + ref(1, 0) + "</p>\n</blockquote>\n<p>b</p>\n" + notes(note(1, "<p>quoted"+back(1, 0)+"</p>\n"))},
		{"in a tight list", "- a[^1]\n\n  [^1]: note\n- b",
			"<ul>\n<li>a" + ref(1, 0) + "</li>\n<li>b</li>\n</ul>\n" + notes(note(1, "<p>note"+back(1, 0)+"</p>\n"))},

		// Unlike goldmark, which loses the text of definitions in
		// footnotes, a definition in a footnote is none.
		{"definition in a footnote", "a[^1] [^2]\n\n[^1]: [^2]: x y",
			"<p>a" + ref(1, 0) + " [^2]</p>\n" + notes(note(1, "<p>[^2]: x y"+back(1, 0)+"</p>\n"))},
		{"definition in a footnote's quote", "a[^1] [^2]\n\n[^1]:\n    > [^2]: x y",
			"<p>a" + ref(1, 0) + " [^2]</p>\n" + notes(note(1, "<blockquote>\n<p>[^2]: x y</p>\n</blockquote>\n"+back(1, 0)))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := convert(t, m, tt.src); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestFootnoteXHTML checks that the list of footnotes renders its rule as
// XHTML does.
func TestFootnoteXHTML(t *testing.T) {
	m := md.New(md.WithExtensions(extension.Footnote), md.WithRendererOptions(html.WithXHTML()))
	got := convert(t, m, "a[^1]\n\n[^1]: one")
	if want := "<hr />\n<ol>"; !strings.Contains(got, want) {
		t.Errorf("got:\n%s\nwant %q in it", got, want)
	}
}

// TestFootnoteOptions checks what each option changes, and the numbers that
// "^^" and "%%" stand for.
func TestFootnoteOptions(t *testing.T) {
	const src = "a[^x] b[^x]\n\n[^x]: n"
	for _, tt := range []struct {
		name string
		opts []extension.FootnoteOption
		want string
	}{
		{"all", []extension.FootnoteOption{
			extension.WithFootnoteIDPrefix("p-"),
			extension.WithFootnoteLinkClass("c^^"),
			extension.WithFootnoteLinkTitle(`see "^^"`),
			extension.WithFootnoteBacklinkClass("b%%"),
			extension.WithFootnoteBacklinkTitle("back to ^^"),
			extension.WithFootnoteBacklinkHTML("<b>^^</b>"),
		}, `<p>a<sup id="p-fnref:1"><a href="#p-fn:1" class="c1" title="see &quot;1&quot;" role="doc-noteref">1</a></sup> b<sup id="p-fnref1:1"><a href="#p-fn:1" class="c1" title="see &quot;1&quot;" role="doc-noteref">1</a></sup></p>
<div class="footnotes" role="doc-endnotes">
<hr>
<ol>
<li id="p-fn:1">
<p>n&#160;<a href="#p-fnref:1" class="b2" title="back to 1" role="doc-backlink"><b>1</b></a>&#160;<a href="#p-fnref1:1" class="b2" title="back to 1" role="doc-backlink"><b>1</b></a></p>
</li>
</ol>
</div>
`},
		{"prefix function", []extension.FootnoteOption{
			extension.WithFootnoteIDPrefixFunction(func(n markdown.Node) string { return n.Kind().String() + "-" }),
		}, `<p>a<sup id="FootnoteLink-fnref:1"><a href="#FootnoteLink-fn:1" class="footnote-ref" role="doc-noteref">1</a></sup> b<sup id="FootnoteLink-fnref1:1"><a href="#FootnoteLink-fn:1" class="footnote-ref" role="doc-noteref">1</a></sup></p>
<div class="footnotes" role="doc-endnotes">
<hr>
<ol>
<li id="Footnote-fn:1">
<p>n&#160;<a href="#FootnoteBacklink-fnref:1" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a>&#160;<a href="#FootnoteBacklink-fnref1:1" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a></p>
</li>
</ol>
</div>
`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := md.New(md.WithExtensions(extension.NewFootnote(tt.opts...)))
			if got := convert(t, m, src); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}

	// A prefix comes before the function.
	prefixed := convert(t, md.New(md.WithExtensions(extension.NewFootnote(extension.WithFootnoteIDPrefix("p-")))), src)
	both := md.New(md.WithExtensions(extension.NewFootnote(
		extension.WithFootnoteIDPrefixFunction(func(markdown.Node) string { return "f-" }),
		extension.WithFootnoteIDPrefix("p-"),
	)))
	if got := convert(t, both, src); got != prefixed || !strings.Contains(got, `"p-fn:1"`) {
		t.Errorf("prefix and function:\n%s\nwant:\n%s", got, prefixed)
	}
}

// TestFootnoteTree checks the nodes of footnotes, and their spans, with the
// extension of a parser on its own.
func TestFootnoteTree(t *testing.T) {
	const src = "a[^n] b[^n]\n\n[^n]: one\n\n    two\n\nc[^e]\n\n[^e]:\n\n[^u]: unused"
	p := parser.NewParser(parser.WithExtensions(extension.Footnote))
	doc := p.Parse(src)

	var got []string
	_ = markdown.Walk(doc, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if !entering {
			return markdown.WalkContinue, nil
		}
		s := n.SourceSpan()
		switch n := n.(type) {
		case *markdown.FootnoteList:
			got = append(got, fmt.Sprintf("FootnoteList %d %v", n.Count, s))
		case *markdown.Footnote:
			got = append(got, fmt.Sprintf("Footnote %q %d %q", n.Ref, n.Index, src[s.Start:s.End]))
		case *markdown.FootnoteLink:
			got = append(got, fmt.Sprintf("FootnoteLink %d %d %d", n.Index, n.RefCount, n.RefIndex))
		case *markdown.FootnoteBacklink:
			got = append(got, fmt.Sprintf("FootnoteBacklink %d %d %d", n.Index, n.RefCount, n.RefIndex))
		case *markdown.Paragraph:
			got = append(got, fmt.Sprintf("Paragraph %q", src[s.Start:s.End]))
		}
		return markdown.WalkContinue, nil
	})
	end := markdown.Span{Start: len(src), End: len(src)}
	want := []string{
		`Paragraph "a[^n] b[^n]"`,
		"FootnoteLink 1 2 0",
		"FootnoteLink 1 2 1",
		`Paragraph "c[^e]"`,
		"FootnoteLink 2 1 0",
		fmt.Sprintf("FootnoteList 2 %v", end),
		`Footnote "n" 1 "[^n]: one\n\n    two"`,
		`Paragraph "one"`,
		`Paragraph "two"`,
		"FootnoteBacklink 1 2 0",
		"FootnoteBacklink 1 2 1",
		`Footnote "e" 2 "[^e]:"`,
		"FootnoteBacklink 2 1 0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
	if _, ok := doc.LastChild().(*markdown.FootnoteList); !ok {
		t.Errorf("last block %s, want the footnotes", doc.LastChild().Kind())
	}
}
