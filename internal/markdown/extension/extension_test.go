package extension_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/internal/spectest"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// convert returns the HTML that m converts src to.
func convert(t testing.TB, m md.Markdown, src string) string {
	t.Helper()
	var b strings.Builder
	if err := m.Convert(src, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestGFM checks the examples of the extensions in the GitHub Flavored
// Markdown specification, version 0.29-gfm, whose tables align cells with
// attributes.
func TestGFM(t *testing.T) {
	m := md.New(
		md.WithExtensions(
			extension.NewTable(extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute)),
			extension.Strikethrough, extension.Linkify, extension.TaskList,
		),
		md.WithRendererOptions(html.WithUnsafe()),
	)
	spectest.Run(t, "testdata/gfm.json", func(src string) string { return convert(t, m, src) })
}

// TestParserOnly checks that the extensions extend a parser on its own.
func TestParserOnly(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.GFM))
	got := p.Parse("| a |\n| - |\n\n~x~ www.x.org\n\n- [x] done").FirstChild()
	if _, ok := got.(*markdown.Table); !ok {
		t.Errorf("first block %s, want a table", got.Kind())
	}
}

// TestTableCellAlignMethod checks how each method renders a right-aligned
// cell, without XHTML and with it.
func TestTableCellAlignMethod(t *testing.T) {
	const src = "| a |\n| -: |"
	for _, tt := range []struct {
		method extension.TableCellAlignMethod
		xhtml  bool
		want   string
	}{
		{extension.TableCellAlignDefault, false, `<th style="text-align:right">a</th>`},
		{extension.TableCellAlignDefault, true, `<th align="right">a</th>`},
		{extension.TableCellAlignAttribute, false, `<th align="right">a</th>`},
		{extension.TableCellAlignStyle, true, `<th style="text-align:right">a</th>`},
		{extension.TableCellAlignNone, false, `<th>a</th>`},
	} {
		opts := []md.Option{md.WithExtensions(extension.NewTable(extension.WithTableCellAlignMethod(tt.method)))}
		if tt.xhtml {
			opts = append(opts, md.WithRendererOptions(html.WithXHTML()))
		}
		if got := convert(t, md.New(opts...), src); !strings.Contains(got, tt.want) {
			t.Errorf("method %d, XHTML %v: %q, want %q in it", tt.method, tt.xhtml, got, tt.want)
		}
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

// TestTableAttributes checks the attributes of the elements of tables: a
// cell's style comes before the alignment that the style method adds, and
// its align replaces the alignment that the attribute method writes.
func TestTableAttributes(t *testing.T) {
	const src = "| a | b |\n| -: | :- |\n| 1 | 2 |"
	for _, tt := range []struct {
		name   string
		method extension.TableCellAlignMethod
		attrs  setAttributes
		want   string
	}{
		{"style", extension.TableCellAlignStyle, setAttributes{{Name: "class", Value: "c"}, {Name: "style", Value: "color:red"}, {Name: "onclick", Value: "x"}}, `<table class="c" style="color:red">
<thead class="c" style="color:red">
<tr>
<th style="color:red;text-align:right" class="c">a</th>
<th style="color:red;text-align:left" class="c">b</th>
</tr>
</thead>
<tbody>
<tr class="c" style="color:red">
<td style="color:red;text-align:right" class="c">1</td>
<td style="color:red;text-align:left" class="c">2</td>
</tr>
</tbody>
</table>
`},
		{"attribute", extension.TableCellAlignAttribute, setAttributes{{Name: "align", Value: "center"}}, `<table align="center">
<thead align="center">
<tr>
<th align="center">a</th>
<th align="center">b</th>
</tr>
</thead>
<tbody>
<tr align="center">
<td align="center">1</td>
<td align="center">2</td>
</tr>
</tbody>
</table>
`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := md.New(
				md.WithExtensions(extension.NewTable(extension.WithTableCellAlignMethod(tt.method))),
				md.WithParserOptions(parser.WithASTTransformer(tt.attrs, 100)),
			)
			if got := convert(t, m, src); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestStrikethroughAttributes checks the attributes of del elements.
func TestStrikethroughAttributes(t *testing.T) {
	m := md.New(
		md.WithExtensions(extension.Strikethrough),
		md.WithParserOptions(parser.WithASTTransformer(setAttributes{{Name: "class", Value: "c"}, {Name: "onclick", Value: "x"}}, 100)),
	)
	const want = `<p class="c"><del class="c">x</del></p>` + "\n"
	if got := convert(t, m, "~~x~~"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestLinkifyOptions checks which extended autolinks each option allows.
func TestLinkifyOptions(t *testing.T) {
	const src = "www.a.org http://b.org ftp://c.org ssh://d.org e@f.org"
	for _, tt := range []struct {
		name string
		opts []extension.LinkifyOption
		want []string
	}{
		{"default", nil, []string{"www.a.org", "http://b.org", "ftp://c.org", "e@f.org"}},
		{"protocols", []extension.LinkifyOption{extension.WithLinkifyAllowedProtocols("https", "ssh:")}, []string{"www.a.org", "ssh://d.org", "e@f.org"}},
		{"no www", []extension.LinkifyOption{extension.WithLinkifyWWW(false)}, []string{"http://b.org", "ftp://c.org", "e@f.org"}},
		{"no email", []extension.LinkifyOption{extension.WithLinkifyEmail(false)}, []string{"www.a.org", "http://b.org", "ftp://c.org"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewParser(parser.WithExtensions(extension.NewLinkify(tt.opts...)))
			var got []string
			_ = markdown.Walk(p.Parse(src), func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
				if l, ok := n.(*markdown.AutoLink); ok && entering {
					got = append(got, l.Label)
				}
				return markdown.WalkContinue, nil
			})
			if !slices.Equal(got, tt.want) {
				t.Errorf("autolinks %q, want %q", got, tt.want)
			}
		})
	}
}

// FuzzParse checks that parsing and rendering any input, with GFM or
// without, with heading attributes and IDs, and with footnotes, definition
// lists, the typographer, and the extension for CJK, completes without
// panicking, and that the spans of blocks nest and follow each other in
// order.
func FuzzParse(f *testing.F) {
	for _, file := range []string{"../parser/testdata/spec.json", "testdata/gfm.json"} {
		for _, ex := range spectest.Examples(f, file) {
			f.Add(ex.Markdown)
		}
	}
	f.Add("# a {#x .y z=[1, \"2\", {w=null}]}\nb {.c}\n===\n# a")
	f.Add("a[^1] ![^x]\n\n> [^x]: q[^1]\n- [^1]: one\n\n      two\n  [^2]:\n\n[^1]: dup")
	f.Add("'a' \"b\" c's '90s -- --- ... << >> \"21\"\" *\"e\"*")
	f.Add("日本\n語 한\n국 a\u200b\nb **“日本”**\\ 語 [日\n](/u)本 ![日\n本](/i) a\\ \nb")
	f.Add("T\nU\n:   a\nlazy\n\n    b\n:\tc\n\n[x]: /u\nV\n: ```\n  d\n\n> W\n> :         e\n: f")
	ms := []md.Markdown{
		md.New(),
		md.New(md.WithExtensions(extension.GFM), md.WithRendererOptions(html.WithUnsafe())),
		md.New(md.WithParserOptions(parser.WithAttribute(), parser.WithAutoHeadingID())),
		md.New(md.WithExtensions(extension.GFM, extension.Footnote)),
		md.New(md.WithExtensions(extension.DefinitionList)),
		md.New(md.WithExtensions(extension.GFM, extension.Footnote, extension.DefinitionList, extension.Typographer, extension.CJK)),
		md.New(md.WithExtensions(extension.NewCJK(extension.WithEastAsianLineBreaks(html.EastAsianLineBreaksCSS3Draft), extension.WithEscapedSpace()))),
	}
	f.Fuzz(func(t *testing.T, src string) {
		for _, m := range ms {
			d := m.Parser().Parse(src)
			var b strings.Builder
			if err := m.Renderer().Render(&b, d); err != nil {
				t.Fatal(err)
			}
			checkSpans(t, src, d)
			for _, l := range d.Links {
				if !within(l.Span, markdown.Span{End: len(src)}) || src[l.Start] != '[' {
					t.Fatalf("%q: link definition %v", src, l.Span)
				}
			}
		}
	})
}

// checkSpans checks that the spans of the blocks in parent lie within
// parent's and follow each other in order. The footnotes of a list of
// footnotes keep the spans of their definitions, which lie anywhere in the
// document, in any order, and start with "[^".
func checkSpans(t *testing.T, src string, parent markdown.Node) {
	t.Helper()
	outer := parent.SourceSpan()
	_, footnotes := parent.(*markdown.FootnoteList)
	if footnotes {
		outer = markdown.Span{End: len(src)}
	}
	prev := outer.Start
	for c := range markdown.Children(parent) {
		if c.Type() != markdown.TypeBlock {
			continue
		}
		s := c.SourceSpan()
		if !within(s, outer) || s.Start < prev && !footnotes {
			t.Fatalf("%q: %s %v not after %d within %v", src, c.Kind(), s, prev, outer)
		}
		if footnotes && !strings.HasPrefix(src[s.Start:], "[^") {
			t.Fatalf("%q: footnote %v does not start with [^", src, s)
		}
		prev = s.End
		checkSpans(t, src, c)
	}
}

// within reports whether s lies within outer.
func within(s, outer markdown.Span) bool {
	return outer.Start <= s.Start && s.Start <= s.End && s.End <= outer.End
}

// TestTableSpans checks the ranges of the source that a table, its rows,
// and its cells come from, after a paragraph line, with a cell that its row
// lacks.
func TestTableSpans(t *testing.T) {
	const src = "intro\n | a | b\\|c |\n|---|:-:|\n| 1 |\n\nafter"
	p := parser.NewParser(parser.WithExtensions(extension.GFM))
	var got []string
	_ = markdown.Walk(p.Parse(src), func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if entering && n.Type() == markdown.TypeBlock {
			s := n.SourceSpan()
			got = append(got, fmt.Sprintf("%s %q", n.Kind(), src[s.Start:s.End]))
		}
		return markdown.WalkContinue, nil
	})
	want := []string{
		`Paragraph "intro"`,
		`Table "| a | b\\|c |\n|---|:-:|\n| 1 |"`,
		`TableHeader "| a | b\\|c |"`,
		`TableCell "a"`,
		`TableCell "b\\|c"`,
		`TableRow "| 1 |"`,
		`TableCell "1"`,
		`TableCell ""`,
		`Paragraph "after"`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("spans:\ngot  %q\nwant %q", got, want)
	}
}
