package parser_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

type attrs = []ast.Attribute

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

// TestHeadingAttributes checks which attributes headings get, and what they
// leave of the heading's text, with WithAttribute and without it.
func TestHeadingAttributes(t *testing.T) {
	p := parser.NewParser(parser.WithAttribute())
	for _, tt := range []struct {
		name, src, want string
	}{
		{"ATX", "# Title {#top .big}", `Document [0:19]
  Heading Level=1 {id="top" class="big"} [0:19] "# Title {#top .big}"
    Text "Title"
`},
		{"closing sequence", "## Title ## {#t}", `Document [0:16]
  Heading Level=2 {id="t"} [0:16] "## Title ## {#t}"
    Text "Title"
`},
		{"closing sequence after", "## Title {#t} ##", `Document [0:16]
  Heading Level=2 [0:16] "## Title {#t} ##"
    Text "Title {#t}"
`},
		{"only attributes", "# {#x}", `Document [0:6]
  Heading Level=1 {id="x"} [0:6] "# {#x}"
`},
		{"setext", "Some\nTitle {.x}\n===", `Document [0:19]
  Heading Level=1 {class="x"} [0:19] "Some\nTitle {.x}\n==="
    Text "Some" softbreak
    Text "Title"
`},
		{"not at the end", "# A {#a} b", `Document [0:10]
  Heading Level=1 [0:10] "# A {#a} b"
    Text "A {#a} b"
`},
		{"escaped brace", `# A \{#a}`, `Document [0:9]
  Heading Level=1 [0:9] "# A \\{#a}"
    Text "A {#a}"
`},
		{"invalid", "# T {class=1}", `Document [0:13]
  Heading Level=1 [0:13] "# T {class=1}"
    Text "T {class=1}"
`},
		{"last list", "# T {#a} {#b}", `Document [0:13]
  Heading Level=1 {id="b"} [0:13] "# T {#a} {#b}"
    Text "T {#a}"
`},
		{"any names", "# T {data-n=1 onclick=x lang=en}", `Document [0:32]
  Heading Level=1 {data-n=1 onclick="x" lang="en"} [0:32] "# T {data-n=1 onclick=x lang=en}"
    Text "T"
`},
		{"in code span", "# `x {#a}`", `Document [0:10]
  Heading Level=1 [0:10] "# ` + "`x {#a}`" + `"
    CodeSpan
      Text "x {#a}"
`},
		{"paragraph", "text {#a}", `Document [0:9]
  Paragraph [0:9] "text {#a}"
    Text "text {#a}"
`},
	} {
		t.Run(tt.name, func(t *testing.T) { checkTree(t, p, tt.src, tt.want) })
	}
	checkTree(t, parser.NewParser(), "# T {#x}", `Document [0:8]
  Heading Level=1 [0:8] "# T {#x}"
    Text "T {#x}"
`)
}

// TestAutoHeadingID checks the IDs that headings get, from their source,
// unique in their document, after the IDs of their attributes.
func TestAutoHeadingID(t *testing.T) {
	p := parser.NewParser(parser.WithAutoHeadingID(), parser.WithAttribute())
	for _, tt := range []struct {
		name, src, want string
	}{
		{"words", "# Hello, World!", `Document [0:15]
  Heading Level=1 {id="hello-world"} [0:15] "# Hello, World!"
    Text "Hello, World!"
`},
		{"source", "## *Em* `code` &amp; x_y-z", `Document [0:26]
  Heading Level=2 {id="em-code-amp-x-y-z"} [0:26] "## *Em* ` + "`code`" + ` &amp; x_y-z"
    Emphasis Level=1
      Text "Em"
    Text " "
    CodeSpan
      Text "code"
    Text " & x_y-z"
`},
		{"non-ASCII", "# héllo wörld", `Document [0:15]
  Heading Level=1 {id="hllo-wrld"} [0:15] "# héllo wörld"
    Text "héllo wörld"
`},
		{"empty", "#", `Document [0:1]
  Heading Level=1 {id="heading"} [0:1] "#"
`},
		{"repeated", "# A\n# A\n# a-1", `Document [0:13]
  Heading Level=1 {id="a"} [0:3] "# A"
    Text "A"
  Heading Level=1 {id="a-1"} [4:7] "# A"
    Text "A"
  Heading Level=1 {id="a-1-1"} [8:13] "# a-1"
    Text "a-1"
`},
		{"attribute first", "# B {#a}\n# A", `Document [0:12]
  Heading Level=1 {id="a"} [0:8] "# B {#a}"
    Text "B"
  Heading Level=1 {id="a-1"} [9:12] "# A"
    Text "A"
`},
		{"after other attributes", "# A {data-x=1}", `Document [0:14]
  Heading Level=1 {data-x=1 id="a"} [0:14] "# A {data-x=1}"
    Text "A"
`},
		{"setext last line", "One\nTwo\n---", `Document [0:11]
  Heading Level=2 {id="two"} [0:11] "One\nTwo\n---"
    Text "One" softbreak
    Text "Two"
`},
	} {
		t.Run(tt.name, func(t *testing.T) { checkTree(t, p, tt.src, tt.want) })
	}
	// Each document has IDs of its own.
	for range 2 {
		checkTree(t, p, "# A", `Document [0:3]
  Heading Level=1 {id="a"} [0:3] "# A"
    Text "A"
`)
	}
}

// numbered generates the IDs "h1", "h2", and so on.
type numbered struct{ n int }

func (s *numbered) Generate(value string, kind ast.NodeKind) string {
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
		for h := range ast.Children(p.Parse("# a\n# b")) {
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
