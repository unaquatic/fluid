package parser_test

import (
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestSpans checks the ranges of the source that blocks come from.
func TestSpans(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{
			name: "leaves",
			src:  "# Title #  \n\npara\ngraph  \n\n***\n\n```go\ncode\n```  \nSetext\n---\n\n  <div>\nx\n\n    indented\n\n\nend",
			want: `Document [0:90]
  Heading Level=1 [0:11] "# Title #  "
    Text "Title"
  Paragraph [13:25] "para\ngraph  "
    Text "para" softbreak
    Text "graph"
  ThematicBreak [27:30] "***"
  FencedCodeBlock Info="go" Literal="code\n" FenceChar='` + "`" + `' FenceLength=3 [32:46] "` + "```go\\ncode\\n```" + `"
  Heading Level=2 [49:59] "Setext\n---"
    Text "Setext"
  HTMLBlock HTMLBlockType=6 Literal="  <div>\nx" [61:70] "  <div>\nx"
  CodeBlock Literal="indented\n" [76:84] "indented"
  Paragraph [87:90] "end"
    Text "end"
`,
		},
		{
			name: "containers",
			src:  "> a\n> > b\nlazy\n\n1. one\n   - two\n\n     more\n2.\n",
			want: `Document [0:45]
  Blockquote [0:14] "> a\n> > b\nlazy"
    Paragraph [2:3] "a"
      Text "a"
    Blockquote [6:14] "> b\nlazy"
      Paragraph [8:14] "b\nlazy"
        Text "b" softbreak
        Text "lazy"
  List Marker='.' IsTight=true Start=1 [16:45] "1. one\n   - two\n\n     more\n2."
    ListItem Offset=3 [16:42] "1. one\n   - two\n\n     more"
      Paragraph [19:22] "one"
        Text "one"
      List Marker='-' [26:42] "- two\n\n     more"
        ListItem Offset=2 [26:42] "- two\n\n     more"
          Paragraph [28:31] "two"
            Text "two"
          Paragraph [38:42] "more"
            Text "more"
    ListItem Offset=3 [43:45] "2."
`,
		},
		{
			name: "after link definitions",
			src:  "[a]: /a\n[b]: /b\ntext\n\n[c]: /c\nSetext\n===",
			want: `Document Links=[{Start=0 End=7 Label="a" URL="/a" Title=""} {Start=8 End=15 Label="b" URL="/b" Title=""} {Start=22 End=29 Label="c" URL="/c" Title=""}] [0:40]
  Paragraph [16:20] "text"
    Text "text"
  Heading Level=1 [30:40] "Setext\n==="
    Text "Setext"
`,
		},
		{
			name: "line endings and NUL",
			src:  "a\x00b\r\n\r\n> c\rd",
			want: `Document [0:12]
  Paragraph [0:3] "a\x00b"
    Text "a�b"
  Blockquote [7:12] "> c\rd"
    Paragraph [9:12] "c\rd"
      Text "c" softbreak
      Text "d"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkTree(t, parser.NewParser(), tt.src, tt.want)
		})
	}
}

// TestIndentedCodeEnd checks that an indented code block ends at its last
// nonblank line, so that a blank line after it parts it from the next list
// item and loosens the list, as commonmark.js parses it.
func TestIndentedCodeEnd(t *testing.T) {
	l := parser.NewParser().Parse("- x\n-     code\n\n- y\n").FirstChild().(*ast.List)
	if l.IsTight {
		t.Error("list is tight, want loose")
	}
}

// TestLineIndex checks the lines and columns of offsets after each kind of
// line ending.
func TestLineIndex(t *testing.T) {
	x := ast.NewLineIndex("ab\ncd\r\nef\rg")
	for _, tt := range []struct{ offset, line, col int }{
		{0, 1, 1}, {2, 1, 3}, {3, 2, 1}, {5, 2, 3}, {7, 3, 1}, {10, 4, 1}, {11, 4, 2},
	} {
		if line, col := x.Position(tt.offset); line != tt.line || col != tt.col {
			t.Errorf("Position(%d) = %d, %d, want %d, %d", tt.offset, line, col, tt.line, tt.col)
		}
	}
}
