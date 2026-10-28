package extension_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestDefinitionList checks how definition lists render.
func TestDefinitionList(t *testing.T) {
	m := md.New(md.WithExtensions(extension.DefinitionList))
	for _, tt := range []struct {
		name, src, want string
	}{
		{"terms and descriptions", "Apple\n:   Pomaceous fruit\nof plants.\n\nOrange\n:   Citrus.",
			"<dl>\n<dt>Apple</dt>\n<dd>Pomaceous fruit\nof plants.</dd>\n<dt>Orange</dt>\n<dd>Citrus.</dd>\n</dl>\n"},
		{"several descriptions", "Apple\n:   Fruit.\n:   A company.", "<dl>\n<dt>Apple</dt>\n<dd>Fruit.</dd>\n<dd>A company.</dd>\n</dl>\n"},
		{"several terms", "Term 1\nTerm 2\n:   Definition", "<dl>\n<dt>Term 1</dt>\n<dt>Term 2</dt>\n<dd>Definition</dd>\n</dl>\n"},
		{"inline content", "T *a* `b`\n: d", "<dl>\n<dt>T <em>a</em> <code>b</code></dt>\n<dd>d</dd>\n</dl>\n"},
		{"loose after a blank line", "Apple\n\n:   Fruit.", "<dl>\n<dt>Apple</dt>\n<dd>\n<p>Fruit.</p>\n</dd>\n</dl>\n"},
		{"blocks", "T\n\n:   One.\n\n    Two.\n\n:   Three.",
			"<dl>\n<dt>T</dt>\n<dd>\n<p>One.</p>\n<p>Two.</p>\n</dd>\n<dd>\n<p>Three.</p>\n</dd>\n</dl>\n"},
		{"tight and loose", "T\n: a\n\n: b\n: c", "<dl>\n<dt>T</dt>\n<dd>a</dd>\n<dd>\n<p>b</p>\n</dd>\n<dd>c</dd>\n</dl>\n"},
		{"several blank lines", "T\n\n: a\n\n\n:   b", "<dl>\n<dt>T</dt>\n<dd>\n<p>a</p>\n</dd>\n<dd>\n<p>b</p>\n</dd>\n</dl>\n"},
		{"lazy line", "T\n: a\nlazy\n: b", "<dl>\n<dt>T</dt>\n<dd>a\nlazy</dd>\n<dd>b</dd>\n</dl>\n"},
		{"code after a paragraph", "T\n: a\n    ```\n    x\n    ```", "<dl>\n<dt>T</dt>\n<dd>a\n<pre><code>x\n</code></pre>\n</dd>\n</dl>\n"},
		{"8 spaces", "T\n:        eight", "<dl>\n<dt>T</dt>\n<dd>eight</dd>\n</dl>\n"},
		{"empty", "T\n: ", "<dl>\n<dt>T</dt>\n<dd></dd>\n</dl>\n"},
		{"in a block quote", "> Apple\n> :   red\n>     more", "<blockquote>\n<dl>\n<dt>Apple</dt>\n<dd>red\nmore</dd>\n</dl>\n</blockquote>\n"},
		{"in a tight list", "- T\n  : d\n- e", "<ul>\n<li>\n<dl>\n<dt>T</dt>\n<dd>d</dd>\n</dl>\n</li>\n<li>e</li>\n</ul>\n"},
		{"no space", "T\n:x", "<p>T\n:x</p>\n"},
		{"no content", "T\n:", "<p>T\n:</p>\n"},
		{"indented", "T\n : x", "<p>T\n: x</p>\n"},
		{"after a heading", "# H\n: d", "<h1>H</h1>\n<p>: d</p>\n"},
		{"terms only link definitions", "[a]: /u\n: x", "<p>: x</p>\n"},
		{"link definition between", "T\n\n[x]: /u\n\n: d", "<p>T</p>\n<p>: d</p>\n"},

		// Unlike goldmark's, a tight description starts a block that is not
		// a paragraph on a new line, as an item of a tight list does, and a
		// blank line between the blocks of a description makes it loose.
		{"code first", "T\n: ```\n  x\n  ```", "<dl>\n<dt>T</dt>\n<dd>\n<pre><code>x\n</code></pre>\n</dd>\n</dl>\n"},
		{"indented code first", "T\n:         code", "<dl>\n<dt>T</dt>\n<dd>\n<pre><code>code\n</code></pre>\n</dd>\n</dl>\n"},
		{"nested", "T\n: a\n  : b", "<dl>\n<dt>T</dt>\n<dd>\n<dl>\n<dt>a</dt>\n<dd>b</dd>\n</dl>\n</dd>\n</dl>\n"},
		{"blank line between blocks", "T\n: a\n\n    b", "<dl>\n<dt>T</dt>\n<dd>\n<p>a</p>\n<p>b</p>\n</dd>\n</dl>\n"},
		{"blank line in code", "T\n: ```\n  x\n\n  y\n  ```", "<dl>\n<dt>T</dt>\n<dd>\n<pre><code>x\n\ny\n</code></pre>\n</dd>\n</dl>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := convert(t, m, tt.src); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestDefinitionListTree checks the nodes of definition lists, and their
// spans, with the extension of a parser on its own.
func TestDefinitionListTree(t *testing.T) {
	const src = "> One  \n> Two\n> :   a\n>\n> :\tb\n>     c\n>\n> Three\n> : \n\nafter"
	p := parser.NewParser(parser.WithExtensions(extension.DefinitionList))
	var got []string
	_ = markdown.Walk(p.Parse(src), func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if !entering || n.Type() != markdown.TypeBlock {
			return markdown.WalkContinue, nil
		}
		s := n.SourceSpan()
		text := fmt.Sprintf("%s %q", n.Kind(), src[s.Start:s.End])
		switch n := n.(type) {
		case *markdown.DefinitionList:
			text += fmt.Sprintf(" offset %d", n.Offset)
		case *markdown.DefinitionDescription:
			text += fmt.Sprintf(" tight %v", n.IsTight)
		}
		got = append(got, text)
		return markdown.WalkContinue, nil
	})
	want := []string{
		`Blockquote "> One  \n> Two\n> :   a\n>\n> :\tb\n>     c\n>\n> Three\n> : "`,
		`DefinitionList "One  \n> Two\n> :   a\n>\n> :\tb\n>     c\n>\n> Three\n> :" offset 2`,
		`DefinitionTerm "One"`,
		`DefinitionTerm "Two"`,
		`DefinitionDescription ":   a" tight true`,
		`Paragraph "a"`,
		`DefinitionDescription ":\tb\n>     c" tight false`,
		`Paragraph "b\n>     c"`,
		`DefinitionTerm "Three"`,
		`DefinitionDescription ":" tight true`,
		`Paragraph "after"`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
}
