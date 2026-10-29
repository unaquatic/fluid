package extension_test

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/internal/spectest"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// dump returns the tree of n, whose document's source is src, as
// ast.Dump writes it.
func dump(t testing.TB, n ast.Node, src string) string {
	t.Helper()
	var b strings.Builder
	if err := ast.Dump(&b, n, src); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// checkTree checks that src parses with p to the tree want, as dump writes
// it.
func checkTree(t *testing.T, p parser.Parser, src, want string) {
	t.Helper()
	if got := dump(t, p.Parse(src), src); got != want {
		t.Errorf("tree of %q:\n%s\nwant:\n%s", src, got, want)
	}
}

// convert returns the HTML of src, which p parses, as the tests render it.
func convert(t testing.TB, p parser.Parser, src string) string {
	t.Helper()
	return spectest.HTML(p.Parse(src), spectest.Options{})
}

// TestGFM checks the trees of the examples of the extensions in the GitHub
// Flavored Markdown specification, version 0.29-gfm, by their HTML, whose
// tables align cells with attributes.
func TestGFM(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.GFM))
	spectest.Run(t, "testdata/gfm.json", func(src string) string {
		return spectest.HTML(p.Parse(src), spectest.Options{Unsafe: true, TableAlignAttribute: true})
	})
}

// TestParserOnly checks that the extensions extend a parser on its own.
func TestParserOnly(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.GFM))
	got := p.Parse("| a |\n| - |\n\n~x~ www.x.org\n\n- [x] done").FirstChild()
	if _, ok := got.(*ast.Table); !ok {
		t.Errorf("first block %s, want a table", got.Kind())
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
			_ = ast.Walk(p.Parse(src), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if l, ok := n.(*ast.AutoLink); ok && entering {
					got = append(got, l.Label)
				}
				return ast.WalkContinue, nil
			})
			if !slices.Equal(got, tt.want) {
				t.Errorf("autolinks %q, want %q", got, tt.want)
			}
		})
	}
}

// FuzzParse checks that parsing any input, with GFM or without, with
// heading attributes and IDs, and with footnotes, definition lists, the
// typographer, and the extension for CJK, completes without panicking, as
// dumping the tree and deciding its soft line breaks do, and that the spans
// of blocks nest and follow each other in order.
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
	ps := []parser.Parser{
		parser.NewParser(),
		parser.NewParser(parser.WithExtensions(extension.GFM)),
		parser.NewParser(parser.WithAttribute(), parser.WithAutoHeadingID()),
		parser.NewParser(parser.WithExtensions(extension.GFM, extension.Footnote)),
		parser.NewParser(parser.WithExtensions(extension.DefinitionList)),
		parser.NewParser(parser.WithExtensions(extension.GFM, extension.Footnote, extension.DefinitionList, extension.Typographer, extension.CJK)),
	}
	f.Fuzz(func(t *testing.T, src string) {
		for _, p := range ps {
			d := p.Parse(src)
			if err := ast.Dump(io.Discard, d, src); err != nil {
				t.Fatal(err)
			}
			_ = ast.Walk(d, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if t, ok := n.(*ast.Text); ok && entering && t.SoftLineBreak() {
					extension.EastAsianLineBreaksSimple.KeepsSoftLineBreak(t)
					extension.EastAsianLineBreaksCSS3Draft.KeepsSoftLineBreak(t)
				}
				return ast.WalkContinue, nil
			})
			checkSpans(t, src, d)
			for _, l := range d.Links {
				if !within(l.Span, ast.Span{End: len(src)}) || src[l.Start] != '[' {
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
func checkSpans(t *testing.T, src string, parent ast.Node) {
	t.Helper()
	outer := parent.SourceSpan()
	_, footnotes := parent.(*ast.FootnoteList)
	if footnotes {
		outer = ast.Span{End: len(src)}
	}
	prev := outer.Start
	for c := range ast.Children(parent) {
		if c.Type() != ast.TypeBlock {
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
func within(s, outer ast.Span) bool {
	return outer.Start <= s.Start && s.Start <= s.End && s.End <= outer.End
}

// TestTableSpans checks the ranges of the source that a table, its rows,
// and its cells come from, after a paragraph line, with a cell that its row
// lacks.
func TestTableSpans(t *testing.T) {
	const src = "intro\n | a | b\\|c |\n|---|:-:|\n| 1 |\n\nafter"
	checkTree(t, parser.NewParser(parser.WithExtensions(extension.GFM)), src, `Document [0:42]
  Paragraph [0:5] "intro"
    Text "intro"
  Table Alignments=[none center] [7:35] "| a | b\\|c |\n|---|:-:|\n| 1 |"
    TableHeader [7:19] "| a | b\\|c |"
      TableCell [9:10] "a"
        Text "a"
      TableCell Alignment=center [13:17] "b\\|c"
        Text "b|c"
    TableRow [30:35] "| 1 |"
      TableCell [32:33] "1"
        Text "1"
      TableCell Alignment=center [35:35] ""
  Paragraph [37:42] "after"
    Text "after"
`)
}
