package widget

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/style"
)

// MarkdownText returns the text that the Markdown src shows, without its
// syntax and its styles, as Markdown parses it: each block's text on lines
// of its own, in order. A heading drops its marker, a list item its bullet or
// number, a quote its border, a code span its backticks, and a link its
// destination, keeping its text. A code block keeps its lines as written,
// with tabs expanded, a table puts each row on a line with its cells a space
// apart, and a hard line break starts a new line. A thematic break shows
// nothing.
func MarkdownText(src string) string {
	var lines []string
	for b := range ast.Children(defaultMarkdown.Parse(src)) {
		lines = appendBlockText(lines, b)
	}
	return strings.Join(lines, "\n")
}

// appendBlockText appends the lines of text that block b shows to lines, as
// MarkdownText gives them, and returns the result.
func appendBlockText(lines []string, b ast.Node) []string {
	switch b := b.(type) {
	case *ast.Heading, *ast.Paragraph:
		return append(lines, strings.Split(inlineText(b), mdHardBreak)...)
	case *ast.HTMLBlock:
		return append(lines, strings.Split(strings.TrimSuffix(b.Literal, "\n"), "\n")...)
	case *ast.FencedCodeBlock:
		return append(lines, codeLines(b.Literal)...)
	case *ast.CodeBlock:
		return append(lines, codeLines(b.Literal)...)
	case *ast.Table:
		for row := range ast.Children(b) {
			cells := make([]string, 0, row.ChildCount())
			for c := range ast.Children(row) {
				cells = append(cells, inlineText(c))
			}
			lines = append(lines, strings.Join(cells, " "))
		}
		return lines
	case *ast.ThematicBreak:
		return lines
	}
	// A list, a list item, and a quote show the text of the blocks they hold.
	for child := range ast.Children(b) {
		lines = appendBlockText(lines, child)
	}
	return lines
}

// inlineText returns the text of the inline content of n, without its syntax.
func inlineText(n ast.Node) string {
	return chunksText(inlineChunks(n, style.Entry{}, mdStyles{}))
}
