package parser_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// spans returns the kinds of the blocks in n, in the order they start,
// each with the source that its span covers.
func spans(src string, n markdown.Node) []string {
	var out []string
	_ = markdown.Walk(n, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if entering && n.Type() == markdown.TypeBlock {
			s := n.SourceSpan()
			out = append(out, fmt.Sprintf("%s %q", n.Kind(), src[s.Start:s.End]))
		}
		return markdown.WalkContinue, nil
	})
	return out
}

// TestSpans checks the ranges of the source that blocks come from.
func TestSpans(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string
	}{
		{
			name: "leaves",
			src:  "# Title #  \n\npara\ngraph  \n\n***\n\n```go\ncode\n```  \nSetext\n---\n\n  <div>\nx\n\n    indented\n\n\nend",
			want: []string{
				`Heading "# Title #  "`,
				`Paragraph "para\ngraph  "`,
				`ThematicBreak "***"`,
				`FencedCodeBlock "` + "```go\\ncode\\n```" + `"`,
				`Heading "Setext\n---"`,
				`HTMLBlock "  <div>\nx"`,
				`CodeBlock "indented"`,
				`Paragraph "end"`,
			},
		},
		{
			name: "containers",
			src:  "> a\n> > b\nlazy\n\n1. one\n   - two\n\n     more\n2.\n",
			want: []string{
				`Blockquote "> a\n> > b\nlazy"`,
				`Paragraph "a"`,
				`Blockquote "> b\nlazy"`,
				`Paragraph "b\nlazy"`,
				`List "1. one\n   - two\n\n     more\n2."`,
				`ListItem "1. one\n   - two\n\n     more"`,
				`Paragraph "one"`,
				`List "- two\n\n     more"`,
				`ListItem "- two\n\n     more"`,
				`Paragraph "two"`,
				`Paragraph "more"`,
				`ListItem "2."`,
			},
		},
		{
			name: "after link definitions",
			src:  "[a]: /a\n[b]: /b\ntext\n\n[c]: /c\nSetext\n===",
			want: []string{
				`Paragraph "text"`,
				`Heading "Setext\n==="`,
			},
		},
		{
			name: "line endings and NUL",
			src:  "a\x00b\r\n\r\n> c\rd",
			want: []string{
				`Paragraph "a\x00b"`,
				`Blockquote "> c\rd"`,
				`Paragraph "c\rd"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := spans(tt.src, parser.NewParser().Parse(tt.src)); !slices.Equal(got, tt.want) {
				t.Errorf("spans of %q:\ngot  %q\nwant %q", tt.src, got, tt.want)
			}
		})
	}
}

// TestIndentedCodeEnd checks that an indented code block ends at its last
// nonblank line, so that a blank line after it parts it from the next list
// item and loosens the list, as commonmark.js parses it.
func TestIndentedCodeEnd(t *testing.T) {
	l := parser.NewParser().Parse("- x\n-     code\n\n- y\n").FirstChild().(*markdown.List)
	if l.IsTight {
		t.Error("list is tight, want loose")
	}
}

// TestLineIndex checks the lines and columns of offsets after each kind of
// line ending.
func TestLineIndex(t *testing.T) {
	x := markdown.NewLineIndex("ab\ncd\r\nef\rg")
	for _, tt := range []struct{ offset, line, col int }{
		{0, 1, 1}, {2, 1, 3}, {3, 2, 1}, {5, 2, 3}, {7, 3, 1}, {10, 4, 1}, {11, 4, 2},
	} {
		if line, col := x.Position(tt.offset); line != tt.line || col != tt.col {
			t.Errorf("Position(%d) = %d, %d, want %d, %d", tt.offset, line, col, tt.line, tt.col)
		}
	}
}
