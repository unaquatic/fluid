package widget

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/parser"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
	"github.com/unaquatic/fluid/textbuf"
)

// defaultParser implements markdownParser with the markdown parser's
// CommonMark syntax, the GitHub Flavored Markdown extensions, and escaped
// spaces for Chinese, Japanese, and Korean.
//
// It leaves out footnotes, definition lists, and link reference
// definitions, which show as the text they are written in. Text written
// after any of them changes the blocks before it: a footnote's or a link
// reference's definition makes the references before it links, and a term
// with a description joins the definition list above it. A StreamMarkdown
// would then change blocks it has already shown, which inline output cannot
// rewrite once they have scrolled into the terminal's scrollback.
type defaultParser struct{}

// defaultMarkdown parses the syntax of defaultParser.
var defaultMarkdown = parser.NewParser(
	parser.WithExtensions(extension.GFM, extension.CJK),
	parser.WithoutLinkReferences(),
)

// Parse converts Markdown source into spaced mdBlock trees.
func (defaultParser) Parse(content string, styles mdStyles) []mdBlock {
	return insertBlockSpacing(convertBlocks(defaultMarkdown.Parse(content), styles))
}

// softBreaks decides which soft line breaks show as a space: all but those
// between East Asian wide characters, which their scripts do not part.
const softBreaks = extension.EastAsianLineBreaksSimple

// mdHardBreak is the text of a chunk that a hard line break leaves in a
// block's body, where the next row of the body starts.
const mdHardBreak = "\n"

// isHardBreak reports whether c is a hard line break.
func isHardBreak(c style.TextChunk) bool { return c.Text == mdHardBreak }

// Block-level AST conversion.

// convertBlocks converts the blocks in parent into mdBlock values.
func convertBlocks(parent ast.Node, s mdStyles) []mdBlock {
	var out []mdBlock
	for b := range ast.Children(parent) {
		if blk, ok := convertBlock(b, s); ok {
			out = append(out, blk)
		}
	}
	return out
}

// convertBlock converts one parsed block.
func convertBlock(b ast.Node, s mdStyles) (mdBlock, bool) {
	switch b := b.(type) {
	case *ast.Heading:
		// A heading is one row, on which a hard line break is a space, after
		// its marker if it has one.
		text := headingMarker(b) + strings.ReplaceAll(chunksText(inlineChunks(b, style.Entry{}, s)), mdHardBreak, " ")
		return mdBlock{
			Kind:  mdBlockHeading,
			Level: b.Level,
			Body:  []style.TextChunk{s.Heading(b.Level).Chunk(text)},
		}, true

	case *ast.Paragraph:
		return mdBlock{
			Kind: mdBlockParagraph,
			Body: inlineChunks(b, style.Entry{}, s),
		}, true

	case *ast.HTMLBlock:
		// Raw HTML is plain text, with each of its lines on a row.
		var body []style.TextChunk
		for i, line := range strings.Split(b.Literal, "\n") {
			if i > 0 {
				body = append(body, style.Entry{}.Chunk(mdHardBreak))
			}
			if line != "" {
				body = append(body, style.Entry{}.Chunk(line))
			}
		}
		return mdBlock{Kind: mdBlockParagraph, Body: body}, true

	case *ast.FencedCodeBlock:
		// Use only the language of an info string such as "go title=foo".
		return mdBlock{
			Kind:      mdBlockCode,
			Lang:      b.Language(),
			CodeLines: codeLines(b.Literal),
		}, true

	case *ast.CodeBlock:
		return mdBlock{Kind: mdBlockCode, CodeLines: codeLines(b.Literal)}, true

	case *ast.List:
		kind := mdBlockList
		if b.IsOrdered() {
			kind = mdBlockOrderedList
		}
		var items []mdBlock
		itemNum := b.Start
		for item := range ast.Children(b) {
			items = append(items, convertListItem(item.(*ast.ListItem), s, b, itemNum))
			if b.IsOrdered() {
				itemNum++
			}
		}
		return mdBlock{
			Kind:     kind,
			Level:    b.Start,
			Children: items,
			Loose:    !b.IsTight,
		}, true

	case *ast.Blockquote:
		// Preserve nested block structure within a blockquote.
		var children []mdBlock
		for child := range ast.Children(b) {
			if blk, ok := convertBlock(child, s); ok {
				// Quote paragraph text inherits the quote style.
				if blk.Kind == mdBlockParagraph {
					for i := range blk.Body {
						blk.Body[i] = s.Quote.Apply(blk.Body[i])
					}
				}
				children = append(children, blk)
			}
		}
		return mdBlock{
			Kind:        mdBlockQuote,
			Prefix:      []style.TextChunk{s.QuoteBorder.Chunk(string(symbol.Vertical) + " ")},
			PrefixWidth: 2,
			Children:    children,
		}, true

	case *ast.ThematicBreak:
		return mdBlock{Kind: mdBlockRule}, true

	case *ast.Table:
		var rows [][]mdTableCell
		for row := range ast.Children(b) {
			rows = append(rows, convertCells(row, s))
		}
		aligns := make([]int, len(b.Alignments))
		for i, a := range b.Alignments {
			switch a {
			case ast.AlignCenter:
				aligns[i] = 0
			case ast.AlignRight:
				aligns[i] = 1
			default:
				aligns[i] = -1
			}
		}
		return mdBlock{
			Kind:        mdBlockTable,
			TableCells:  rows,
			TableAligns: aligns,
		}, true
	}
	return mdBlock{}, false
}

// convertCells converts the cells of a table row.
func convertCells(row ast.Node, s mdStyles) []mdTableCell {
	cells := make([]mdTableCell, 0, row.ChildCount())
	for c := range ast.Children(row) {
		cells = append(cells, mdTableCell{Chunks: inlineChunks(c, style.Entry{}, s)})
	}
	return cells
}

// convertListItem returns a list item of list l, numbered itemNum if l is
// ordered, with its marker, first body, and nested blocks.
func convertListItem(li *ast.ListItem, s mdStyles, l *ast.List, itemNum int) mdBlock {
	// The marker shows as written: a bullet in the bullet style, or a number
	// and its delimiter in the list number style, and then a task item's
	// box.
	p := []style.TextChunk{s.Bullet.Chunk(string(l.Marker) + " ")}
	if l.IsOrdered() {
		p = []style.TextChunk{s.ListEnum.Chunk(intToStr(itemNum) + string(l.Marker) + " ")}
	}
	switch li.Task {
	case ast.Unchecked:
		p = append(p, s.Bullet.Chunk("[ ] "))
	case ast.Checked:
		p = append(p, s.Bullet.Chunk("[x] "))
	}
	body, children := itemContent(li, s)
	return mdBlock{
		Kind:        mdBlockListItem,
		Prefix:      p,
		PrefixWidth: textbuf.StringWidth(chunksText(p)),
		Body:        body,
		Children:    children,
	}
}

// headingMarker returns the marker that heading h starts with, as written,
// and the space after it: "#" for each level for an ATX heading, and
// nothing for a setext heading, whose text starts where it does and whose
// underline follows it.
func headingMarker(h *ast.Heading) string {
	if lines := h.Lines(); len(lines) > 0 && lines[0].Offset == h.SourceSpan().Start {
		return ""
	}
	return strings.Repeat("#", h.Level) + " "
}

// itemContent returns the inline content of the first paragraph of n, a
// list item, which shows beside its marker, and n's other blocks.
func itemContent(n ast.Node, s mdStyles) ([]style.TextChunk, []mdBlock) {
	var body []style.TextChunk
	var children []mdBlock
	first := true
	for child := range ast.Children(n) {
		if p, ok := child.(*ast.Paragraph); ok && first {
			body = inlineChunks(p, style.Entry{}, s)
			first = false
			continue
		}
		if blk, ok := convertBlock(child, s); ok {
			children = append(children, blk)
		}
	}
	return body, children
}

// codeTabWidth matches TextBuffer's default so Markdown and Code align tabs equally.
const codeTabWidth = 4

// codeLines returns code-block lines with tabs expanded for direct chunk rendering.
func codeLines(text string) []string {
	var out []string
	for line := range strings.Lines(text) {
		out = append(out, expandCodeTabs(strings.TrimSuffix(line, "\n")))
	}
	return out
}

// expandCodeTabs replaces each tab with codeTabWidth spaces.
func expandCodeTabs(line string) string {
	if !strings.ContainsRune(line, '\t') {
		return line
	}
	return strings.ReplaceAll(line, "\t", strings.Repeat(" ", codeTabWidth))
}

// Inline AST conversion.

// inlineChunks flattens the inline content of parent while merging inherited styles.
// Nested styles combine, so bold inline code retains code colors and gains bold.
func inlineChunks(parent ast.Node, in style.Entry, s mdStyles) []style.TextChunk {
	var chunks []style.TextChunk
	for t := range ast.Children(parent) {
		switch t := t.(type) {
		case *ast.Text:
			if t.Value != "" {
				chunks = append(chunks, in.Chunk(t.Value))
			}
			switch {
			case t.HardLineBreak():
				chunks = append(chunks, in.Chunk(mdHardBreak))
			case t.SoftLineBreak() && softBreaks.KeepsSoftLineBreak(t):
				chunks = append(chunks, in.Chunk(" "))
			}

		case *ast.CodeSpan:
			chunks = append(chunks, in.Merge(s.Code).Chunk(chunksText(inlineChunks(t, style.Entry{}, s))))

		case *ast.RawHTML:
			chunks = append(chunks, in.Chunk(t.Value))

		case *ast.Emphasis:
			e := s.Strong
			if t.Level == 1 {
				e = s.Emph
			}
			chunks = append(chunks, inlineChunks(t, in.Merge(e), s)...)

		case *ast.Strikethrough:
			struck := in
			struck.Attrs |= style.AttrStrikethrough
			chunks = append(chunks, inlineChunks(t, struck, s)...)

		case *ast.Link:
			linkChunks := inlineChunks(t, in.Merge(s.LinkText), s)
			for i := range linkChunks {
				linkChunks[i].Link = t.Destination
			}
			chunks = append(chunks, linkChunks...)

		case *ast.AutoLink:
			c := in.Merge(s.Link).Chunk(t.Label)
			c.Link = t.URL
			chunks = append(chunks, c)

		case *ast.Image:
			// An image shows its description.
			chunks = append(chunks, inlineChunks(t, in, s)...)
		}
	}
	return chunks
}

func chunksText(chunks []style.TextChunk) string {
	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
