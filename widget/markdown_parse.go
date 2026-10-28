package widget

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/parser"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
	"github.com/unaquatic/fluid/textbuf"
)

// gfmParser implements markdownParser with the markdown parser's
// CommonMark syntax and GitHub Flavored Markdown extensions.
type gfmParser struct{}

// gfmMarkdown parses CommonMark with the GitHub Flavored Markdown extensions.
var gfmMarkdown = parser.NewParser(parser.WithExtensions(extension.GFM))

// Parse converts Markdown source into spaced mdBlock trees.
func (gfmParser) Parse(content string, styles mdStyles) []mdBlock {
	return insertBlockSpacing(convertBlocks(gfmMarkdown.Parse(content), styles))
}

// mdHardBreak is the text of a chunk that a hard line break leaves in a
// block's body, where the next row of the body starts.
const mdHardBreak = "\n"

// isHardBreak reports whether c is a hard line break.
func isHardBreak(c style.TextChunk) bool { return c.Text == mdHardBreak }

// Block-level AST conversion.

// convertBlocks converts the blocks in parent into mdBlock values.
func convertBlocks(parent markdown.Node, s mdStyles) []mdBlock {
	var out []mdBlock
	for b := range markdown.Children(parent) {
		if blk, ok := convertBlock(b, s); ok {
			out = append(out, blk)
		}
	}
	return out
}

// convertBlock converts one parsed block.
func convertBlock(b markdown.Node, s mdStyles) (mdBlock, bool) {
	switch b := b.(type) {
	case *markdown.Heading:
		// A heading is one row, on which a hard line break is a space.
		text := strings.ReplaceAll(chunksText(inlineChunks(b, style.Entry{}, s)), mdHardBreak, " ")
		return mdBlock{
			Kind:  mdBlockHeading,
			Level: b.Level,
			Body:  []style.TextChunk{s.Heading(b.Level).Chunk(text)},
		}, true

	case *markdown.Paragraph:
		return mdBlock{
			Kind: mdBlockParagraph,
			Body: inlineChunks(b, style.Entry{}, s),
		}, true

	case *markdown.HTMLBlock:
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

	case *markdown.FencedCodeBlock:
		// Use only the language of an info string such as "go title=foo".
		return mdBlock{
			Kind:      mdBlockCode,
			Lang:      b.Language(),
			CodeLines: codeLines(b.Literal),
		}, true

	case *markdown.CodeBlock:
		return mdBlock{Kind: mdBlockCode, CodeLines: codeLines(b.Literal)}, true

	case *markdown.List:
		kind := mdBlockList
		if b.IsOrdered() {
			kind = mdBlockOrderedList
		}
		var items []mdBlock
		itemNum := b.Start
		for item := range markdown.Children(b) {
			items = append(items, convertListItem(item.(*markdown.ListItem), s, kind, itemNum))
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

	case *markdown.Blockquote:
		// Preserve nested block structure within a blockquote.
		var children []mdBlock
		for child := range markdown.Children(b) {
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

	case *markdown.ThematicBreak:
		return mdBlock{Kind: mdBlockRule}, true

	case *markdown.Table:
		var rows [][]mdTableCell
		for row := range markdown.Children(b) {
			rows = append(rows, convertCells(row, s))
		}
		aligns := make([]int, len(b.Alignments))
		for i, a := range b.Alignments {
			switch a {
			case markdown.AlignCenter:
				aligns[i] = 0
			case markdown.AlignRight:
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
func convertCells(row markdown.Node, s mdStyles) []mdTableCell {
	cells := make([]mdTableCell, 0, row.ChildCount())
	for c := range markdown.Children(row) {
		cells = append(cells, mdTableCell{Chunks: inlineChunks(c, style.Entry{}, s)})
	}
	return cells
}

// convertListItem returns a list item with its marker, first body, and nested blocks.
func convertListItem(li *markdown.ListItem, s mdStyles, listKind mdBlockKind, itemNum int) mdBlock {
	// Build the task box, ordered number, or unordered bullet marker.
	var prefix []style.TextChunk
	var prefixWidth int
	switch {
	case li.Task != markdown.NotTask:
		p := "[ ] "
		if li.Task == markdown.Checked {
			p = "[" + string(symbol.CheckMark) + "] "
		}
		prefix = []style.TextChunk{s.Bullet.Chunk(p)}
		prefixWidth = textbuf.StringWidth(p)
	case listKind == mdBlockOrderedList:
		p := intToStr(itemNum) + ". "
		prefix = []style.TextChunk{s.ListEnum.Chunk(p)}
		prefixWidth = textbuf.StringWidth(p)
	default:
		p := string(symbol.BulletFilled) + " "
		prefix = []style.TextChunk{s.Bullet.Chunk(p)}
		prefixWidth = len(p)
	}

	// Render the first paragraph beside the marker and retain other blocks as children.
	var body []style.TextChunk
	var children []mdBlock
	first := true

	for child := range markdown.Children(li) {
		if p, ok := child.(*markdown.Paragraph); ok && first {
			body = inlineChunks(p, style.Entry{}, s)
			first = false
			continue
		}
		if blk, ok := convertBlock(child, s); ok {
			children = append(children, blk)
		}
	}

	return mdBlock{
		Kind:        mdBlockListItem,
		Prefix:      prefix,
		PrefixWidth: prefixWidth,
		Body:        body,
		Children:    children,
	}
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
func inlineChunks(parent markdown.Node, in style.Entry, s mdStyles) []style.TextChunk {
	var chunks []style.TextChunk
	for t := range markdown.Children(parent) {
		switch t := t.(type) {
		case *markdown.Text:
			if t.Value != "" {
				chunks = append(chunks, in.Chunk(t.Value))
			}
			switch {
			case t.HardLineBreak():
				chunks = append(chunks, in.Chunk(mdHardBreak))
			case t.SoftLineBreak():
				chunks = append(chunks, in.Chunk(" "))
			}

		case *markdown.CodeSpan:
			chunks = append(chunks, in.Merge(s.Code).Chunk(chunksText(inlineChunks(t, style.Entry{}, s))))

		case *markdown.RawHTML:
			chunks = append(chunks, in.Chunk(t.Value))

		case *markdown.Emphasis:
			e := s.Strong
			if t.Level == 1 {
				e = s.Emph
			}
			chunks = append(chunks, inlineChunks(t, in.Merge(e), s)...)

		case *markdown.Strikethrough:
			struck := in
			struck.Attrs |= style.AttrStrikethrough
			chunks = append(chunks, inlineChunks(t, struck, s)...)

		case *markdown.Link:
			linkChunks := inlineChunks(t, in.Merge(s.LinkText), s)
			for i := range linkChunks {
				linkChunks[i].Link = t.Destination
			}
			chunks = append(chunks, linkChunks...)

		case *markdown.AutoLink:
			c := in.Merge(s.Link).Chunk(t.Label)
			c.Link = t.URL
			chunks = append(chunks, c)

		case *markdown.Image:
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
