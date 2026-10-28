package parser_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

type attrs = []markdown.Attribute

// TestParseAttributes checks the attributes that each list parses to, and
// the length that it takes, or that it is no list.
func TestParseAttributes(t *testing.T) {
	for _, tt := range []struct {
		name, src string
		want      attrs
		n         int // the length taken; -1 for all of src
	}{
		{"shorthands", "{#top .a .b}", attrs{{Name: "id", Value: "top"}, {Name: "class", Value: "a b"}}, -1},
		{"spaces before", "  {#x} after", attrs{{Name: "id", Value: "x"}}, 6},
		{"shorthand characters", "{#a-b_c:d.e√}", attrs{{Name: "id", Value: "a-b_c:d.e√"}}, -1},
		{"words", "{key=value data-x=a:b.c-d}", attrs{{Name: "key", Value: "value"}, {Name: "data-x", Value: "a:b.c-d"}}, -1},
		{"literals", "{t=true f=false z=null}", attrs{{Name: "t", Value: true}, {Name: "f", Value: false}, {Name: "z", Value: nil}}, -1},
		{"numbers", "{a=1 b=-2.5 c=+3e2 d=1.}", attrs{{Name: "a", Value: 1.0}, {Name: "b", Value: -2.5}, {Name: "c", Value: 300.0}, {Name: "d", Value: 1.0}}, -1},
		{"quoted", `{s="a \"b\" \\ \/ \t \x"}`, attrs{{Name: "s", Value: "a \"b\" \\ / \t \\x"}}, -1},
		{"array", `{l=[1, "two" [true]]}`, attrs{{Name: "l", Value: []any{1.0, "two", []any{true}}}}, -1},
		{"nested", "{n={a=b}}", attrs{{Name: "n", Value: attrs{{Name: "a", Value: "b"}}}}, -1},
		{"class joins", `{class="x y" .z}`, attrs{{Name: "class", Value: "x y z"}}, -1},
		{"commas", "{a=1, b=2,c=3}", attrs{{Name: "a", Value: 1.0}, {Name: "b", Value: 2.0}, {Name: "c", Value: 3.0}}, -1},
		{"spaces around equals", "{a = 1 }", attrs{{Name: "a", Value: 1.0}}, -1},
		{"empty", "{}", attrs{}, -1},

		{"no braces", "#x", nil, 0},
		{"space only", "{ }", nil, 0},
		{"unclosed", "{#x", nil, 0},
		{"name only", "{a}", nil, 0},
		{"no value", "{a=}", nil, 0},
		{"name starts with digit", "{1=2}", nil, 0},
		{"class not string", "{class=0}", nil, 0},
		{"unclosed quote", `{s="x}`, nil, 0},
		{"quote across lines", "{s=\"x\ny\"}", nil, 0},
		{"no exponent digits", "{a=1e}", nil, 0},
		{"leading comma", "{a=[,1]}", nil, 0},
		{"trailing comma", "{a=[1,]}", nil, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, n := parser.ParseAttributes(tt.src)
			if tt.n < 0 {
				tt.n = len(tt.src)
			}
			if !reflect.DeepEqual(got, tt.want) || n != tt.n {
				t.Errorf("ParseAttributes(%q) = %#v, %d, want %#v, %d", tt.src, got, n, tt.want, tt.n)
			}
		})
	}
}

// TestParseAttributesDepth checks that brackets and braces nest 32 deep,
// and no deeper.
func TestParseAttributesDepth(t *testing.T) {
	for _, tt := range []struct {
		depth int
		ok    bool
	}{{32, true}, {33, false}} {
		// The outer braces and depth-1 brackets.
		src := "{a=" + strings.Repeat("[", tt.depth-1) + strings.Repeat("]", tt.depth-1) + "}"
		if _, n := parser.ParseAttributes(src); n > 0 != tt.ok {
			t.Errorf("depth %d: length %d, want a list: %v", tt.depth, n, tt.ok)
		}
	}
}

// convert returns the HTML that m converts src to.
func convert(t *testing.T, m md.Markdown, src string) string {
	t.Helper()
	var b strings.Builder
	if err := m.Convert(src, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestHeadingAttributes checks which attributes headings get, and what they
// leave of the heading's text, with WithAttribute and without it.
func TestHeadingAttributes(t *testing.T) {
	m := md.New(md.WithParserOptions(parser.WithAttribute()))
	for _, tt := range []struct {
		name, src, want string
	}{
		{"ATX", "# Title {#top .big}", `<h1 id="top" class="big">Title</h1>`},
		{"closing sequence", "## Title ## {#t}", `<h2 id="t">Title</h2>`},
		{"closing sequence after", "## Title {#t} ##", `<h2>Title {#t}</h2>`},
		{"only attributes", "# {#x}", `<h1 id="x"></h1>`},
		{"setext", "Some\nTitle {.x}\n===", "<h1 class=\"x\">Some\nTitle</h1>"},
		{"not at the end", "# A {#a} b", `<h1>A {#a} b</h1>`},
		{"escaped brace", `# A \{#a}`, `<h1>A {#a}</h1>`},
		{"invalid", "# T {class=1}", `<h1>T {class=1}</h1>`},
		{"last list", "# T {#a} {#b}", `<h1 id="b">T {#a}</h1>`},
		{"filtered", "# T {data-n=1 onclick=x lang=en}", `<h1 data-n="1" lang="en">T</h1>`},
		{"in code span", "# `x {#a}`", "<h1><code>x {#a}</code></h1>"},
		{"paragraph", "text {#a}", "<p>text {#a}</p>"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := convert(t, m, tt.src); got != tt.want+"\n" {
				t.Errorf("got %q, want %q", got, tt.want+"\n")
			}
		})
	}
	if got := convert(t, md.New(), "# T {#x}"); got != "<h1>T {#x}</h1>\n" {
		t.Errorf("without WithAttribute: %q", got)
	}
}

// TestAutoHeadingID checks the IDs that headings get, from their source,
// unique in their document, after the IDs of their attributes.
func TestAutoHeadingID(t *testing.T) {
	m := md.New(md.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()))
	for _, tt := range []struct {
		name, src, want string
	}{
		{"words", "# Hello, World!", `<h1 id="hello-world">Hello, World!</h1>`},
		{"source", "## *Em* `code` &amp; x_y-z", `<h2 id="em-code-amp-x-y-z"><em>Em</em> <code>code</code> &amp; x_y-z</h2>`},
		{"non-ASCII", "# héllo wörld", `<h1 id="hllo-wrld">héllo wörld</h1>`},
		{"empty", "#", `<h1 id="heading"></h1>`},
		{"repeated", "# A\n# A\n# a-1", "<h1 id=\"a\">A</h1>\n<h1 id=\"a-1\">A</h1>\n<h1 id=\"a-1-1\">a-1</h1>"},
		{"attribute first", "# B {#a}\n# A", "<h1 id=\"a\">B</h1>\n<h1 id=\"a-1\">A</h1>"},
		{"after other attributes", "# A {data-x=1}", `<h1 data-x="1" id="a">A</h1>`},
		{"setext last line", "One\nTwo\n---", "<h2 id=\"two\">One\nTwo</h2>"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := convert(t, m, tt.src); got != tt.want+"\n" {
				t.Errorf("got %q, want %q", got, tt.want+"\n")
			}
		})
	}
	// Each document has IDs of its own.
	for range 2 {
		if got := convert(t, m, "# A"); got != "<h1 id=\"a\">A</h1>\n" {
			t.Errorf("again: %q", got)
		}
	}
}

// numbered generates the IDs "h1", "h2", and so on.
type numbered struct{ n int }

func (s *numbered) Generate(value string, kind markdown.NodeKind) string {
	s.n++
	return "h" + strconv.Itoa(s.n)
}

func (s *numbered) Put(id string) {}

// TestWithIDs checks that the parser makes the IDs of each document with
// the function that WithIDs sets.
func TestWithIDs(t *testing.T) {
	made := 0
	p := parser.NewParser(parser.WithAutoHeadingID(), parser.WithIDs(func() parser.IDs {
		made++
		return &numbered{}
	}))
	for range 2 {
		var got []any
		for h := range markdown.Children(p.Parse("# a\n# b")) {
			id, _ := h.Attribute("id")
			got = append(got, id)
		}
		if want := []any{"h1", "h2"}; !reflect.DeepEqual(got, want) {
			t.Errorf("IDs %q, want %q", got, want)
		}
	}
	if made != 2 {
		t.Errorf("made %d IDs, want 2", made)
	}
}
