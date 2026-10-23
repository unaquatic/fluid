package widget

import (
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	goldmarkast "github.com/yuin/goldmark/ast"
	goldmarkext "github.com/yuin/goldmark/extension"
	goldmarkextast "github.com/yuin/goldmark/extension/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	goldmarktext "github.com/yuin/goldmark/text"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
	"github.com/unaquatic/fluid/textbuf"
)

// GoldmarkParser implements MarkdownParser with Goldmark's CommonMark parser
// and GitHub Flavored Markdown extensions.
type GoldmarkParser struct {
	parser goldmarkparser.Parser
}

// NewGoldmarkParser returns a parser with GitHub Flavored Markdown extensions.
func NewGoldmarkParser() *GoldmarkParser {
	gm := goldmark.New(goldmark.WithExtensions(goldmarkext.GFM))
	return &GoldmarkParser{parser: gm.Parser()}
}

// sharedGoldmarkParser returns the parser used by every Markdown widget that
// does not supply its own. A Goldmark parser holds fixed trigger tables of
// about thirteen kilobytes, so one per widget dominated the memory of a long
// document. Goldmark guards its own initialization with a sync.Once and only
// reads those tables while parsing, so a single instance serves all of them.
var sharedGoldmarkParser = sync.OnceValue(NewGoldmarkParser)

// Parse converts Markdown source into spaced MdBlock trees.
func (g *GoldmarkParser) Parse(content string, styles MdStyles) []MdBlock {
	src := []byte(content)
	tree := g.parser.Parse(goldmarktext.NewReader(src))
	blocks := g.walkChildren(tree, src, styles)
	return InsertBlockSpacing(blocks)
}

// Block-level AST conversion.

// walkChildren converts node's direct children into MdBlock values.
func (g *GoldmarkParser) walkChildren(node goldmarkast.Node, src []byte, s MdStyles) []MdBlock {
	var blocks []MdBlock
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if blk, ok := g.convertBlock(child, src, s); ok {
			blocks = append(blocks, blk)
		}
	}
	return blocks
}

// convertBlock converts one supported Goldmark block node.
func (g *GoldmarkParser) convertBlock(node goldmarkast.Node, src []byte, s MdStyles) (MdBlock, bool) {
	switch n := node.(type) {
	case *goldmarkast.Heading:
		chunks := g.inlineChunks(n, src, style.Entry{}, s)
		return MdBlock{
			Kind:  MdBlockHeading,
			Level: n.Level,
			Body:  []style.TextChunk{s.Heading(n.Level).Chunk(chunksText(chunks))},
		}, true

	case *goldmarkast.Paragraph:
		return MdBlock{
			Kind: MdBlockParagraph,
			Body: g.inlineChunks(n, src, style.Entry{}, s),
		}, true

	case *goldmarkast.FencedCodeBlock:
		lang := ""
		if n.Info != nil {
			lang = strings.TrimSpace(string(n.Info.Segment.Value(src)))
			// Use only the language portion of an info string such as "go title=foo".
			if idx := strings.IndexByte(lang, ' '); idx > 0 {
				lang = lang[:idx]
			}
		}
		return MdBlock{
			Kind:      MdBlockCode,
			Lang:      lang,
			CodeLines: g.extractLines(n, src),
		}, true

	case *goldmarkast.CodeBlock:
		return MdBlock{
			Kind:      MdBlockCode,
			CodeLines: g.extractLines(n, src),
		}, true

	case *goldmarkast.List:
		kind := MdBlockList
		if n.IsOrdered() {
			kind = MdBlockOrderedList
		}
		var items []MdBlock
		itemNum := n.Start
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			li, ok := item.(*goldmarkast.ListItem)
			if !ok {
				continue
			}
			liBlock := g.convertListItem(li, src, s, kind, itemNum)
			items = append(items, liBlock)
			if n.IsOrdered() {
				itemNum++
			}
		}
		return MdBlock{
			Kind:     kind,
			Level:    n.Start,
			Children: items,
		}, true

	case *goldmarkast.Blockquote:
		// Preserve nested block structure within a blockquote.
		var children []MdBlock
		for bqChild := n.FirstChild(); bqChild != nil; bqChild = bqChild.NextSibling() {
			if blk, ok := g.convertBlock(bqChild, src, s); ok {
				// Quote paragraph text inherits the quote style.
				if blk.Kind == MdBlockParagraph {
					for i := range blk.Body {
						blk.Body[i] = s.Quote.Apply(blk.Body[i])
					}
				}
				children = append(children, blk)
			}
		}
		return MdBlock{
			Kind:        MdBlockQuote,
			Prefix:      []style.TextChunk{s.QuoteBorder.Chunk(string(symbol.Vertical) + " ")},
			PrefixWidth: 2,
			Children:    children,
		}, true

	case *goldmarkast.ThematicBreak:
		return MdBlock{Kind: MdBlockRule}, true

	case *goldmarkextast.Table:
		var rows [][]MdTableCell
		var aligns []int
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			var cells []MdTableCell
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if c, ok := cell.(*goldmarkextast.TableCell); ok {
					cells = append(cells, MdTableCell{
						Chunks: g.inlineChunks(c, src, style.Entry{}, s),
					})
					if len(aligns) < len(cells) {
						a := -1
						switch c.Alignment {
						case goldmarkextast.AlignCenter:
							a = 0
						case goldmarkextast.AlignRight:
							a = 1
						}
						aligns = append(aligns, a)
					}
				}
			}
			rows = append(rows, cells)
		}
		return MdBlock{
			Kind:        MdBlockTable,
			TableCells:  rows,
			TableAligns: aligns,
		}, true

	default:
		// Recover the first supported descendant of an unhandled wrapper node.
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if blk, ok := g.convertBlock(child, src, s); ok {
				return blk, true
			}
		}
		return MdBlock{}, false
	}
}

// convertListItem returns a list item with its marker, first body, and nested blocks.
func (g *GoldmarkParser) convertListItem(li *goldmarkast.ListItem, src []byte, s MdStyles, listKind MdBlockKind, itemNum int) MdBlock {
	// Build the ordered number or unordered bullet marker.
	var prefix []style.TextChunk
	var prefixWidth int
	if listKind == MdBlockOrderedList {
		p := "  " + intToStr(itemNum) + ". "
		prefix = []style.TextChunk{s.ListEnum.Chunk(p)}
		prefixWidth = textbuf.StringWidth(p)
	} else {
		p := string(symbol.BulletFilled) + " "
		prefix = []style.TextChunk{s.Bullet.Chunk(p)}
		prefixWidth = len(p)
	}

	// Render the first text block beside the marker and retain later blocks as children.
	var body []style.TextChunk
	var children []MdBlock
	first := true

	for liChild := li.FirstChild(); liChild != nil; liChild = liChild.NextSibling() {
		if first {
			if p, ok := liChild.(*goldmarkast.Paragraph); ok {
				body = g.inlineChunks(p, src, style.Entry{}, s)
				first = false
				continue
			}
			if tb, ok := liChild.(*goldmarkast.TextBlock); ok {
				body = g.inlineChunks(tb, src, style.Entry{}, s)
				first = false
				continue
			}
		}
		if blk, ok := g.convertBlock(liChild, src, s); ok {
			children = append(children, blk)
		}
	}

	return MdBlock{
		Kind:        MdBlockListItem,
		Prefix:      prefix,
		PrefixWidth: prefixWidth,
		Body:        body,
		Children:    children,
	}
}

// codeTabWidth matches TextBuffer's default so Markdown and Code align tabs equally.
const codeTabWidth = 4

// extractLines returns code-block lines with tabs expanded for direct chunk rendering.
func (g *GoldmarkParser) extractLines(n goldmarkast.Node, src []byte) []string {
	type hasLines interface {
		Lines() *goldmarktext.Segments
	}
	hl, ok := n.(hasLines)
	if !ok {
		return nil
	}
	lines := hl.Lines()
	var out []string
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		line := strings.TrimRight(string(seg.Value(src)), "\n")
		out = append(out, expandCodeTabs(line))
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

// inlineChunks flattens an inline subtree while merging inherited styles.
// Nested styles combine, so bold inline code retains code colors and gains bold.
func (g *GoldmarkParser) inlineChunks(parent goldmarkast.Node, src []byte, in style.Entry, s MdStyles) []style.TextChunk {
	var chunks []style.TextChunk
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *goldmarkast.Text:
			chunks = append(chunks, in.Chunk(string(n.Segment.Value(src))))
			if n.SoftLineBreak() {
				chunks = append(chunks, in.Chunk(" "))
			}

		case *goldmarkast.String:
			chunks = append(chunks, in.Chunk(string(n.Value)))

		case *goldmarkast.CodeSpan:
			chunks = append(chunks, in.Merge(s.Code).Chunk(string(n.Text(src))))

		case *goldmarkast.Emphasis:
			e := s.Strong
			if n.Level == 1 {
				e = s.Emph
			}
			chunks = append(chunks, g.inlineChunks(n, src, in.Merge(e), s)...)

		case *goldmarkast.Link:
			url := string(n.Destination)
			linkChunks := g.inlineChunks(n, src, in.Merge(s.LinkText), s)
			for i := range linkChunks {
				linkChunks[i].Link = url
			}
			chunks = append(chunks, linkChunks...)

		case *goldmarkextast.Strikethrough:
			struck := in
			struck.Attrs |= style.AttrStrikethrough
			chunks = append(chunks, g.inlineChunks(n, src, struck, s)...)

		case *goldmarkast.AutoLink:
			url := string(n.URL(src))
			c := in.Merge(s.Link).Chunk(url)
			c.Link = url
			chunks = append(chunks, c)

		case *goldmarkast.RawHTML:
			segs := n.Segments
			for i := range segs.Len() {
				seg := segs.At(i)
				chunks = append(chunks, in.Chunk(string(seg.Value(src))))
			}

		default:
			chunks = append(chunks, g.inlineChunks(child, src, in, s)...)
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
