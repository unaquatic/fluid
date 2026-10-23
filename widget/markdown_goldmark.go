package widget

import (
	"strings"

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

// GoldmarkParser implements MarkdownParser using goldmark (CommonMark).
// It supports GFM extensions: tables and strikethrough.
type GoldmarkParser struct {
	parser goldmarkparser.Parser
}

// NewGoldmarkParser creates a GoldmarkParser with GFM extensions.
func NewGoldmarkParser() *GoldmarkParser {
	gm := goldmark.New(goldmark.WithExtensions(goldmarkext.GFM))
	return &GoldmarkParser{parser: gm.Parser()}
}

// Parse converts markdown content into a tree of MdBlocks.
func (g *GoldmarkParser) Parse(content string, styles MdStyles) []MdBlock {
	src := []byte(content)
	tree := g.parser.Parse(goldmarktext.NewReader(src))
	blocks := g.walkChildren(tree, src, styles)
	return InsertBlockSpacing(blocks)
}

// ---------------------------------------------------------------------------
// AST Walking — Block Level
// ---------------------------------------------------------------------------

// walkChildren converts all direct children of node into MdBlocks.
func (g *GoldmarkParser) walkChildren(node goldmarkast.Node, src []byte, s MdStyles) []MdBlock {
	var blocks []MdBlock
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if blk, ok := g.convertBlock(child, src, s); ok {
			blocks = append(blocks, blk)
		}
	}
	return blocks
}

// convertBlock converts a single goldmark AST node into an MdBlock.
func (g *GoldmarkParser) convertBlock(node goldmarkast.Node, src []byte, s MdStyles) (MdBlock, bool) {
	switch n := node.(type) {
	case *goldmarkast.Heading:
		fg := s.HeadingColor(n.Level)
		chunks := g.inlineChunks(n, src, 0, s)
		return MdBlock{
			Kind:  MdBlockHeading,
			Level: n.Level,
			Body:  []style.TextChunk{{Text: chunksText(chunks), Fg: &fg, Attributes: style.AttrBold}},
		}, true

	case *goldmarkast.Paragraph:
		return MdBlock{
			Kind: MdBlockParagraph,
			Body: g.inlineChunks(n, src, 0, s),
		}, true

	case *goldmarkast.FencedCodeBlock:
		lang := ""
		if n.Info != nil {
			lang = strings.TrimSpace(string(n.Info.Segment.Value(src)))
			// Strip anything after a space (e.g. "go title=foo")
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
		// Recurse into blockquote children to build a tree.
		var children []MdBlock
		for bqChild := n.FirstChild(); bqChild != nil; bqChild = bqChild.NextSibling() {
			if blk, ok := g.convertBlock(bqChild, src, s); ok {
				// Apply quote styling to paragraph body chunks.
				if blk.Kind == MdBlockParagraph {
					for i := range blk.Body {
						blk.Body[i].Attributes |= style.AttrItalic
						if blk.Body[i].Fg == nil {
							blk.Body[i].Fg = &s.QuoteFg
						}
					}
				}
				children = append(children, blk)
			}
		}
		return MdBlock{
			Kind:        MdBlockQuote,
			Prefix:      []style.TextChunk{{Text: string(symbol.Vertical) + " ", Fg: &s.QuoteBorder}},
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
						Chunks: g.inlineChunks(c, src, 0, s),
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
		// Unknown block — try to recurse into children and return first match.
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if blk, ok := g.convertBlock(child, src, s); ok {
				return blk, true
			}
		}
		return MdBlock{}, false
	}
}

// convertListItem builds an MdBlockListItem with its prefix and child blocks.
func (g *GoldmarkParser) convertListItem(li *goldmarkast.ListItem, src []byte, s MdStyles, listKind MdBlockKind, itemNum int) MdBlock {
	// Build prefix.
	var prefix []style.TextChunk
	var prefixWidth int
	if listKind == MdBlockOrderedList {
		p := "  " + intToStr(itemNum) + ". "
		prefix = []style.TextChunk{{Text: p, Fg: &s.BulletFg}}
		prefixWidth = textbuf.StringWidth(p)
	} else {
		p := string(symbol.BulletFilled) + " "
		prefix = []style.TextChunk{{Text: p, Fg: &s.BulletFg}}
		prefixWidth = len(p)
	}

	// First paragraph becomes the Body (rendered on the same line as prefix).
	// Remaining children (more paragraphs, code blocks, nested lists) go into Children.
	var body []style.TextChunk
	var children []MdBlock
	first := true

	for liChild := li.FirstChild(); liChild != nil; liChild = liChild.NextSibling() {
		if first {
			if p, ok := liChild.(*goldmarkast.Paragraph); ok {
				body = g.inlineChunks(p, src, 0, s)
				first = false
				continue
			}
			if tb, ok := liChild.(*goldmarkast.TextBlock); ok {
				body = g.inlineChunks(tb, src, 0, s)
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

// extractLines reads the text lines from a code block node.
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
		out = append(out, strings.TrimRight(string(seg.Value(src)), "\n"))
	}
	return out
}

// ---------------------------------------------------------------------------
// AST Walking — Inline Level
// ---------------------------------------------------------------------------

func (g *GoldmarkParser) inlineChunks(parent goldmarkast.Node, src []byte, attrs uint32, s MdStyles) []style.TextChunk {
	var chunks []style.TextChunk
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *goldmarkast.Text:
			chunks = append(chunks, style.TextChunk{
				Text: string(n.Segment.Value(src)), Attributes: attrs,
			})
			if n.SoftLineBreak() {
				chunks = append(chunks, style.TextChunk{Text: " ", Attributes: attrs})
			}

		case *goldmarkast.String:
			chunks = append(chunks, style.TextChunk{
				Text: string(n.Value), Attributes: attrs,
			})

		case *goldmarkast.CodeSpan:
			chunks = append(chunks, style.TextChunk{
				Text: string(n.Text(src)), Fg: &s.CodeFg, Bg: &s.CodeBg, Attributes: attrs,
			})

		case *goldmarkast.Emphasis:
			childAttrs := attrs
			if n.Level == 1 {
				childAttrs |= style.AttrItalic
			} else {
				childAttrs |= style.AttrBold
			}
			chunks = append(chunks, g.inlineChunks(n, src, childAttrs, s)...)

		case *goldmarkast.Link:
			linkChunks := g.inlineChunks(n, src, attrs|style.AttrUnderline, s)
			url := string(n.Destination)
			for i := range linkChunks {
				linkChunks[i].Fg = &s.LinkFg
				linkChunks[i].Link = url
			}
			chunks = append(chunks, linkChunks...)

		case *goldmarkextast.Strikethrough:
			chunks = append(chunks, g.inlineChunks(n, src, attrs|style.AttrStrikethrough, s)...)

		case *goldmarkast.AutoLink:
			url := string(n.URL(src))
			chunks = append(chunks, style.TextChunk{
				Text: url, Fg: &s.LinkFg, Attributes: attrs | style.AttrUnderline, Link: url,
			})

		case *goldmarkast.RawHTML:
			segs := n.Segments
			for i := 0; i < segs.Len(); i++ {
				seg := segs.At(i)
				chunks = append(chunks, style.TextChunk{
					Text: string(seg.Value(src)), Attributes: attrs,
				})
			}

		default:
			chunks = append(chunks, g.inlineChunks(child, src, attrs, s)...)
		}
	}
	return chunks
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
