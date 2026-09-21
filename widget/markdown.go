package widget

import (
	"math"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
	"github.com/unaquatic/fluid/textbuf"
)

// Markdown renders markdown content with headings, lists, code blocks, links,
// tables, and word wrapping. Scrolling is handled by a parent ScrollBox.
type Markdown struct {
	*render.BaseRenderable

	mu       sync.Mutex // Protects content and contentDirty.
	content  string
	wrapMode WrapMode

	// Parsing produces width-independent blocks; wrapping converts them to lines.
	blocks        []MdBlock
	renderedLines []mdLine
	contentDirty  bool // Requires parsing and wrapping.
	wrapDirty     bool // Requires wrapping without parsing.
	lastWrapWidth int  // Width used to produce renderedLines.

	parser MarkdownParser // Defaults to GoldmarkParser.

	fg style.Color
	bg style.Color

	// headings stores styles for levels one through six.
	headings [6]style.Entry

	// highlighter styles fenced code; nil selects the monochrome fallback.
	highlighter SyntaxHighlighter

	// codeFg and codeBg style fenced blocks when highlighting is unavailable.
	// Inline code uses code instead, while a highlighter supplies token styles.
	codeFg style.Color
	codeBg style.Color

	// codeBlockPad is inside the fenced-code background; codeBlockMargin is outside it.
	codeBlockPad    int
	codeBlockMargin int

	// prosePad insets prose but not top-level fenced-code backgrounds.
	prosePad int

	// Inline element styles.
	code        style.Entry
	link        style.Entry
	linkText    style.Entry
	strong      style.Entry
	emph        style.Entry
	quote       style.Entry
	quoteBorder style.Entry
	bullet      style.Entry
	listEnum    style.Entry
	rule        style.Entry
}

// Markdown parsing.

// MarkdownParser converts Markdown source into width-independent blocks.
type MarkdownParser interface {
	// Parse converts content to blocks and applies styles to inline elements.
	Parse(content string, styles MdStyles) []MdBlock
}

// MdStyles contains the inline styles applied while parsing Markdown.
//
// Each entry carries both colors and attributes, keeping style policy outside the parser.
type MdStyles struct {
	// Fg is the fallback foreground for chunks without an explicit style.
	Fg style.Color

	Headings [6]style.Entry

	Code        style.Entry
	Link        style.Entry
	LinkText    style.Entry
	Strong      style.Entry
	Emph        style.Entry
	Quote       style.Entry
	QuoteBorder style.Entry
	Bullet      style.Entry
	ListEnum    style.Entry
	Rule        style.Entry
}

// Heading returns the style for levels one through six and uses level six otherwise.
func (s MdStyles) Heading(level int) style.Entry {
	if level < 1 || level > len(s.Headings) {
		return s.Headings[len(s.Headings)-1]
	}
	return s.Headings[level-1]
}

// Width-independent block representation.

// MdBlockKind identifies a markdown block type.
type MdBlockKind int

const (
	MdBlockParagraph MdBlockKind = iota
	MdBlockHeading
	MdBlockCode
	MdBlockList        // Unordered-list container.
	MdBlockOrderedList // Ordered-list container.
	MdBlockListItem    // Item nested under a list container.
	MdBlockQuote
	MdBlockRule
	MdBlockBlank
	MdBlockTable
)

// MdBlock is a parsed markdown block. Blocks form a tree: container blocks
// (List, OrderedList, Quote, ListItem) hold child blocks in Children.
type MdBlock struct {
	Kind        MdBlockKind
	Level       int               // Heading level or list indentation level.
	Prefix      []style.TextChunk // Bullet, number, or quote marker.
	PrefixWidth int               // Terminal width of Prefix.
	Body        []style.TextChunk // Parsed inline content before wrapping.
	Children    []MdBlock         // Nested list items, quote contents, or other blocks.
	Lang        string            // Fenced-code language, such as "go" or "python".
	CodeLines   []string          // Raw code-block lines.
	TableCells  [][]MdTableCell   // Table rows with inline cell formatting.
	TableAligns []int             // Column alignment: -1 left, 0 center, or 1 right.
}

// MdTableCell holds a table cell's inline content.
type MdTableCell struct {
	Chunks []style.TextChunk
}

// mdLine contains one width-dependent rendered Markdown line.
type mdLine struct {
	chunks []style.TextChunk
}

// mdLineWidth returns the terminal width of line.
func mdLineWidth(line mdLine) int {
	w := 0
	for _, c := range line.chunks {
		w += textbuf.StringWidth(c.Text)
	}
	return w
}

// MarkdownOption configures a Markdown widget.
type MarkdownOption func(*Markdown)

// NewMarkdown returns a Markdown widget containing content.
func NewMarkdown(content string, opts ...MarkdownOption) *Markdown {
	m := &Markdown{
		BaseRenderable: render.NewBaseRenderable(),
		content:        content,
		wrapMode:       WrapWord,
		fg:             style.ColorWhite,
		bg:             style.ColorTransparent,
		contentDirty:   true,

		headings: [6]style.Entry{
			{Fg: style.ColorBrightCyan, Attrs: style.AttrBold},
			{Fg: style.ColorBrightGreen, Attrs: style.AttrBold},
			{Fg: style.ColorBrightYellow, Attrs: style.AttrBold},
			{Fg: style.ColorBrightMagenta, Attrs: style.AttrBold},
			{Fg: style.ColorBrightBlue, Attrs: style.AttrBold},
			{Fg: style.ColorBrightRed, Attrs: style.AttrBold},
		},

		codeFg: style.ColorFromInts(200, 200, 200, 255),
		codeBg: style.ColorFromInts(30, 30, 30, 255),

		code: style.Entry{
			Fg: style.ColorFromInts(200, 200, 200, 255),
			Bg: style.ColorFromInts(30, 30, 30, 255),
		},
		link:        style.Entry{Fg: style.ColorBrightBlue, Attrs: style.AttrUnderline},
		linkText:    style.Entry{Fg: style.ColorBrightBlue, Attrs: style.AttrUnderline},
		strong:      style.Entry{Attrs: style.AttrBold},
		emph:        style.Entry{Attrs: style.AttrItalic},
		quote:       style.Entry{Fg: style.ColorGray, Attrs: style.AttrItalic},
		quoteBorder: style.Entry{Fg: style.ColorBrightBlack},
		bullet:      style.Entry{Fg: style.ColorBrightBlue},
		listEnum:    style.Entry{Fg: style.ColorBrightBlue},
		rule:        style.Entry{Fg: style.ColorGray},
	}

	for _, opt := range opts {
		opt(m)
	}

	// Use Goldmark and Chroma unless options supplied replacements.
	if m.parser == nil {
		m.parser = sharedGoldmarkParser()
	}
	if m.highlighter == nil {
		m.highlighter = NewChromaHighlighter("monokai")
	}

	m.blocks = m.parser.Parse(m.content, m.mdStyles())

	m.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		w := width
		if widthMode == layout.MeasureUndefined {
			w = 80
		}

		// The flex engine probes intrinsic size with a width near MaxInt16.
		// Return the cached height without treating that probe as a real wrap width.
		//
		// A real terminal viewport cannot approach half of MaxInt16 columns.
		if w >= math.MaxInt16/2 && (m.lastWrapWidth > 0 || len(m.blocks) == 0) {
			h := len(m.renderedLines)
			if heightMode == layout.MeasureExactly {
				h = height
			} else if heightMode == layout.MeasureAtMost && h > height {
				h = height
			}
			return layout.MeasureResult{Width: w, Height: h}
		}

		// Reuse cached lines only at the exact width that produced them. An exact
		// comparison lets a terminal resize trigger wrapping in this layout pass.
		//
		// Empty block lists always have zero height and need no wrapping.
		//
		// SetBlocks marks only wrapDirty, so both dirty flags must be clear before
		// renderedLines can supply an intrinsic height.
		if !m.contentDirty && !m.wrapDirty && (len(m.blocks) == 0 || (m.lastWrapWidth > 0 && w == m.lastWrapWidth)) {
			h := len(m.renderedLines)
			if heightMode == layout.MeasureExactly {
				h = height
			} else if heightMode == layout.MeasureAtMost && h > height {
				h = height
			}
			return layout.MeasureResult{Width: w, Height: h}
		}

		// Wrap at the proposed layout width before reporting height.
		m.wrapToWidth(w)
		// Do not impose a minimum height. Blocks such as thematic breaks can emit
		// no rows, and StreamMarkdown relies on those children measuring zero.
		h := len(m.renderedLines)
		if heightMode == layout.MeasureExactly {
			h = height
		} else if heightMode == layout.MeasureAtMost && h > height {
			h = height
		}
		return layout.MeasureResult{Width: w, Height: h}
	}

	return m
}

// Content management.

// SetContent replaces the Markdown source.
// It is safe to call from any goroutine because parsing is deferred until measurement or rendering.
func (m *Markdown) SetContent(content string) {
	m.mu.Lock()
	if m.content == content {
		m.mu.Unlock()
		return
	}
	m.content = content
	m.contentDirty = true
	m.mu.Unlock()
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// Content returns the current content.
func (m *Markdown) Content() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.content
}

// AppendContent appends source text and defers parsing until measurement or rendering.
// Safe to call from any goroutine.
func (m *Markdown) AppendContent(text string) {
	m.mu.Lock()
	m.content += text
	m.contentDirty = true
	m.mu.Unlock()
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// SetBlocks replaces parsed blocks directly and schedules wrapping without parsing.
// StreamMarkdown uses it to distribute already parsed blocks to children.
//
// Equal blocks leave wrapped output and dirty state unchanged.
func (m *Markdown) SetBlocks(blocks []MdBlock) {
	m.mu.Lock()
	m.contentDirty = false // The caller supplied parsed blocks.
	m.mu.Unlock()

	if reflect.DeepEqual(m.blocks, blocks) {
		return
	}

	m.blocks = blocks
	m.wrapDirty = true
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// LineCount returns the number of rendered (post-wrap) lines.
func (m *Markdown) LineCount() int {
	m.ensureWrapped()
	return len(m.renderedLines)
}

// Markdown wrapping.

// mdStyles returns the parser styles configured on the widget.
func (m *Markdown) mdStyles() MdStyles {
	return MdStyles{
		Fg:       m.fg,
		Headings: m.headings,
		Code:     m.code,
		Link:     m.link, LinkText: m.linkText,
		Strong: m.strong, Emph: m.emph,
		Quote: m.quote, QuoteBorder: m.quoteBorder,
		Bullet: m.bullet, ListEnum: m.listEnum,
		Rule: m.rule,
	}
}

// ensureParsed reparses dirty content. It must run on the main goroutine.
func (m *Markdown) ensureParsed() {
	m.mu.Lock()
	dirty := m.contentDirty
	content := m.content
	if dirty {
		m.contentDirty = false
	}
	m.mu.Unlock()
	if !dirty {
		return
	}
	m.blocks = m.parser.Parse(content, m.mdStyles())
	m.wrapDirty = true // Parsed blocks require new width-dependent lines.
}

// ensureWrapped parses when needed and wraps at the current widget width.
// It must run on the main goroutine.
func (m *Markdown) ensureWrapped() {
	m.ensureParsed()
	w := m.Width()
	if !m.wrapDirty && w == m.lastWrapWidth {
		return
	}
	m.wrapBlocks(w)
}

// wrapToWidth parses when needed and wraps at w for intrinsic measurement.
// It must run on the main goroutine.
func (m *Markdown) wrapToWidth(w int) {
	m.ensureParsed()
	if !m.wrapDirty && w == m.lastWrapWidth {
		return
	}
	m.wrapBlocks(w)
}

// Markdown rendering.

// RenderSelf draws wrapped lines as styled grapheme clusters.
// An enclosing ScrollBox is responsible for clipping and scrolling.
func (m *Markdown) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := m.Width()
	if w <= 0 {
		return
	}

	m.ensureWrapped()

	x0 := m.X()
	y := m.Y()

	for _, line := range m.renderedLines {
		cx := x0
		for _, chunk := range line.chunks {
			fg := m.fg
			bg := m.bg
			if chunk.Fg != nil {
				fg = *chunk.Fg
			}
			if chunk.Bg != nil {
				bg = *chunk.Bg
			}
			attrs := chunk.Attributes
			remaining := chunk.Text
			for len(remaining) > 0 {
				cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
				remaining = rest
				if cx+rw > x0+w {
					break // The remaining chunk lies outside the widget.
				}
				buf.SetGraphemeCluster(cx, y, cluster, fg, bg, attrs)
				cx += rw
			}
		}
		y++
	}
}

// Block rendering helpers.

// renderCodeBlock renders a fenced or indented code block at width w.
// It uses syntax highlighting when both a highlighter and language are available.
//
// Each line fills the block width with codeBg. codeBlockPad inserts blank rows
// above and below and moves source text inward by the same number of columns.
func (m *Markdown) renderCodeBlock(blk MdBlock, indent string, w int) {
	lead := indent + strings.Repeat(" ", m.codeBlockPad)

	for range m.codeBlockPad {
		m.addCodeLine(nil, 0, w)
	}
	if !m.renderHighlighted(blk, lead, w) {
		for _, line := range blk.CodeLines {
			text := lead + line
			m.addCodeLine(
				[]style.TextChunk{{Text: text, Fg: &m.codeFg, Bg: &m.codeBg}},
				textbuf.StringWidth(text), w,
			)
		}
	}
	for range m.codeBlockPad {
		m.addCodeLine(nil, 0, w)
	}
}

// renderHighlighted appends highlighted code lines and reports whether it succeeded.
func (m *Markdown) renderHighlighted(blk MdBlock, lead string, w int) bool {
	if m.highlighter == nil || blk.Lang == "" {
		return false
	}
	content := strings.Join(blk.CodeLines, "\n")
	highlighted, err := m.highlighter.Highlight(content, blk.Lang, m.codeFg, m.codeBg)
	if err != nil || len(highlighted) == 0 {
		return false
	}
	for _, line := range highlighted {
		chunks := []style.TextChunk{{Text: lead, Bg: &m.codeBg}}
		used := textbuf.StringWidth(lead)
		for _, span := range line.Spans {
			fg, bg := span.Fg, span.Bg
			chunks = append(chunks, style.TextChunk{
				Text:       span.Text,
				Fg:         &fg,
				Bg:         &bg,
				Attributes: span.Attributes,
			})
			used += textbuf.StringWidth(span.Text)
		}
		m.addCodeLine(chunks, used, w)
	}
	return true
}

// addCodeLine appends one code line whose background spans the configured margins.
func (m *Markdown) addCodeLine(chunks []style.TextChunk, used, w int) {
	margin := m.codeBlockMargin
	if margin > 0 {
		gutter := style.TextChunk{Text: strings.Repeat(" ", margin)}
		chunks = append([]style.TextChunk{gutter}, chunks...)
		used += margin
	}
	if end := w - margin; used < end {
		chunks = append(chunks, style.TextChunk{
			Text: strings.Repeat(" ", end-used),
			Bg:   &m.codeBg,
		})
	}
	m.renderedLines = append(m.renderedLines, mdLine{chunks: chunks})
}

// isContinuationBlock reports whether adjacent blocks should have no inserted spacer.
func isContinuationBlock(prev, cur MdBlockKind) bool {
	listKinds := func(k MdBlockKind) bool {
		return k == MdBlockList || k == MdBlockOrderedList || k == MdBlockListItem
	}
	if listKinds(prev) && listKinds(cur) {
		return true
	}
	if prev == MdBlockCode && cur == MdBlockCode {
		return true
	}
	// A thematic break uses only its preceding spacer, producing one blank row.
	if prev == MdBlockRule {
		return true
	}
	return false
}

// InsertBlockSpacing inserts blank blocks between unrelated adjacent blocks.
func InsertBlockSpacing(blocks []MdBlock) []MdBlock {
	if len(blocks) <= 1 {
		return blocks
	}
	spaced := make([]MdBlock, 0, len(blocks)*2)
	for i, blk := range blocks {
		if i > 0 {
			prev := blocks[i-1]
			needsSpace := blk.Kind != MdBlockBlank && prev.Kind != MdBlockBlank &&
				!isContinuationBlock(prev.Kind, blk.Kind)
			if needsSpace {
				spaced = append(spaced, MdBlock{Kind: MdBlockBlank})
			}
		}
		spaced = append(spaced, blk)
	}
	return spaced
}

// wrapBlocks converts parsed blocks to rendered lines at width w.
func (m *Markdown) wrapBlocks(w int) {
	m.renderedLines = nil
	m.lastWrapWidth = w
	m.wrapDirty = false
	m.wrapBlockList(m.blocks, w, m.prosePad)
}

// wrapBlockList recursively wraps blocks with indent columns of left indentation.
func (m *Markdown) wrapBlockList(blocks []MdBlock, w int, indent int) {
	avail := max(w-indent, 4)
	indentStr := strings.Repeat(" ", indent)
	// Apply prosePad on both sides while preserving non-positive no-wrap widths.
	proseW := w
	if m.prosePad > 0 {
		proseW = max(w-m.prosePad, 4)
	}

	for _, blk := range blocks {
		switch blk.Kind {
		case MdBlockParagraph:
			m.addWrappedLine(
				[]style.TextChunk{{Text: indentStr}}, indent, proseW,
				blk.Body...,
			)
		case MdBlockHeading:
			m.addWrappedLine(
				append([]style.TextChunk{{Text: indentStr}}, blk.Body...),
				indent, proseW,
			)
		case MdBlockCode:
			// Top-level code ignores prosePad; nested code retains its parent indent.
			codeIndent := indentStr
			if indent == m.prosePad {
				codeIndent = ""
			}
			m.renderCodeBlock(blk, codeIndent, w)
		case MdBlockList, MdBlockOrderedList:
			// List containers render their child items at the current indentation.
			m.wrapBlockList(blk.Children, w, indent)

		case MdBlockListItem:
			// Place the item prefix and body on the first line.
			prefix := make([]style.TextChunk, 0, len(blk.Prefix)+1)
			if indent > 0 {
				prefix = append(prefix, style.TextChunk{Text: indentStr})
			}
			prefix = append(prefix, blk.Prefix...)
			m.addWrappedLine(prefix, indent+blk.PrefixWidth, proseW, blk.Body...)
			// Align nested blocks after the item prefix.
			if len(blk.Children) > 0 {
				m.wrapBlockList(blk.Children, w, indent+blk.PrefixWidth)
			}

		case MdBlockQuote:
			// Place the quote prefix and first paragraph on the same line.
			prefix := make([]style.TextChunk, 0, len(blk.Prefix)+1)
			if indent > 0 {
				prefix = append(prefix, style.TextChunk{Text: indentStr})
			}
			prefix = append(prefix, blk.Prefix...)
			if len(blk.Children) > 0 && blk.Children[0].Kind == MdBlockParagraph {
				m.addWrappedLine(prefix, indent+blk.PrefixWidth, proseW, blk.Children[0].Body...)
				if len(blk.Children) > 1 {
					m.wrapBlockList(blk.Children[1:], w, indent+blk.PrefixWidth)
				}
			} else {
				// Preserve the prefix even when the quote starts with another block type.
				m.renderedLines = append(m.renderedLines, mdLine{chunks: prefix})
				m.wrapBlockList(blk.Children, w, indent+blk.PrefixWidth)
			}

		case MdBlockRule:
			// A thematic break emits no cells. Its preceding MdBlockBlank supplies
			// the same one-row separation as an ordinary paragraph boundary.
		case MdBlockBlank:
			m.renderedLines = append(m.renderedLines, mdLine{})
		case MdBlockTable:
			m.renderTable(blk.TableCells, blk.TableAligns, avail)
		}
	}
}

// addWrappedLine appends prefix and body, wrapping body around the prefix width.
func (m *Markdown) addWrappedLine(prefix []style.TextChunk, prefixWidth int, maxWidth int, body ...style.TextChunk) {
	if m.wrapMode == WrapNone || maxWidth <= 0 {
		// Preserve all chunks on one line when wrapping is disabled.
		all := make([]style.TextChunk, 0, len(prefix)+len(body))
		all = append(all, prefix...)
		all = append(all, body...)
		m.renderedLines = append(m.renderedLines, mdLine{chunks: all})
		return
	}

	availWidth := max(maxWidth-prefixWidth, 4)

	// Avoid cluster conversion when the body already fits.
	totalWidth := 0
	for _, c := range body {
		totalWidth += m.StringWidth(c.Text)
	}

	if totalWidth <= availWidth {
		// Preserve the original chunks and their style boundaries.
		all := make([]style.TextChunk, 0, len(prefix)+len(body))
		all = append(all, prefix...)
		all = append(all, body...)
		m.renderedLines = append(m.renderedLines, mdLine{chunks: all})
		return
	}

	// Flatten styled chunks into indivisible grapheme clusters for wrapping.
	type styledCluster struct {
		text string
		base rune // First rune used to detect whitespace boundaries.
		w    int
		fg   *style.Color
		bg   *style.Color
		attr uint32
		link string
	}

	var allClusters []styledCluster
	for _, c := range body {
		remaining := c.Text
		for len(remaining) > 0 {
			cluster, rest, cw := textbuf.NextGraphemeCluster(remaining)
			remaining = rest
			var base rune
			for _, r := range cluster {
				base = r
				break
			}
			allClusters = append(allClusters, styledCluster{
				text: cluster, base: base, w: cw,
				fg: c.Fg, bg: c.Bg, attr: c.Attributes, link: c.Link,
			})
		}
	}

	firstLine := true
	clusterIdx := 0

	for clusterIdx < len(allClusters) {
		lineWidth := 0
		lineAvail := availWidth
		if !firstLine {
			// Continuations omit the prefix but retain prose padding.
			lineAvail = maxWidth
			if m.prosePad > 0 {
				lineAvail = max(maxWidth-m.prosePad, 4)
			}
		}

		lineStart := clusterIdx

		if m.wrapMode == WrapWord {
			// Track the last whitespace boundary that fits.
			lastBreak := -1
			for clusterIdx < len(allClusters) {
				rw := allClusters[clusterIdx].w
				if lineWidth+rw > lineAvail {
					break
				}
				lineWidth += rw
				if unicode.IsSpace(allClusters[clusterIdx].base) {
					lastBreak = clusterIdx
				}
				clusterIdx++
			}

			// Prefer the last complete word when more clusters remain.
			if clusterIdx < len(allClusters) && lastBreak > lineStart {
				clusterIdx = lastBreak + 1
			}
		} else {
			// Character wrapping fills each line by grapheme width.
			for clusterIdx < len(allClusters) {
				rw := allClusters[clusterIdx].w
				if lineWidth+rw > lineAvail {
					break
				}
				lineWidth += rw
				clusterIdx++
			}
		}

		// Reassemble this line while preserving style boundaries.
		var lineChunks []style.TextChunk
		if firstLine {
			lineChunks = append(lineChunks, prefix...)
		} else if m.prosePad > 0 {
			lineChunks = append(lineChunks, style.TextChunk{
				Text: strings.Repeat(" ", m.prosePad),
			})
		}

		// Merge adjacent clusters with identical styles.
		var sb strings.Builder
		var curFg *style.Color
		var curBg *style.Color
		var curAttr uint32
		var curLink string
		started := false

		flushChunk := func() {
			if sb.Len() > 0 {
				lineChunks = append(lineChunks, style.TextChunk{
					Text:       sb.String(),
					Fg:         curFg,
					Bg:         curBg,
					Attributes: curAttr,
					Link:       curLink,
				})
				sb.Reset()
			}
		}

		end := clusterIdx
		for j := lineStart; j < end; j++ {
			sc := allClusters[j]
			if !started {
				curFg = sc.fg
				curBg = sc.bg
				curAttr = sc.attr
				curLink = sc.link
				started = true
			} else if sc.fg != curFg || sc.bg != curBg || sc.attr != curAttr || sc.link != curLink {
				flushChunk()
				curFg = sc.fg
				curBg = sc.bg
				curAttr = sc.attr
				curLink = sc.link
			}
			sb.WriteString(sc.text)
		}
		flushChunk()

		m.renderedLines = append(m.renderedLines, mdLine{chunks: lineChunks})
		firstLine = false

		// Drop whitespace that would begin the next continuation line.
		for clusterIdx < len(allClusters) && unicode.IsSpace(allClusters[clusterIdx].base) {
			clusterIdx++
		}
	}
}

// cellText returns a table cell's text without styling.
func cellText(cell MdTableCell) string {
	return chunksText(cell.Chunks)
}

// renderTable appends a bordered table to renderedLines.
func (m *Markdown) renderTable(rows [][]MdTableCell, aligns []int, maxWidth int) {
	if len(rows) == 0 {
		return
	}

	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return
	}

	// Size each column to its widest cell.
	colWidths := make([]int, numCols)
	for _, row := range rows {
		for c := 0; c < numCols && c < len(row); c++ {
			cw := m.StringWidth(cellText(row[c]))
			if cw > colWidths[c] {
				colWidths[c] = cw
			}
		}
	}

	// Reserve one padding column on each side of a cell.
	for c := range colWidths {
		colWidths[c] += 2
	}

	borderFg := m.rule.Fg
	bc := symbol.SingleChars
	m.renderedLines = append(m.renderedLines, m.buildTableBorder(colWidths, bc.TopLeft, bc.TopT, bc.TopRight, bc.Horizontal, borderFg))

	for rowIdx, row := range rows {
		var chunks []style.TextChunk
		for c := 0; c < numCols; c++ {
			chunks = append(chunks, style.TextChunk{Text: string(bc.Vertical), Fg: &borderFg})

			var cell MdTableCell
			if c < len(row) {
				cell = row[c]
			}
			text := cellText(cell)

			cellWidth := colWidths[c]
			textWidth := m.StringWidth(text)
			padding := cellWidth - textWidth
			if padding < 0 {
				text = truncateStringToWidth(text, cellWidth)
				padding = 0
			}

			align := -1
			if c < len(aligns) {
				align = aligns[c]
			}

			var leftPad, rightPad int
			switch align {
			case 0:
				leftPad = padding / 2
				rightPad = padding - leftPad
			case 1:
				leftPad = padding
				rightPad = 0
			default:
				leftPad = 1
				rightPad = max(padding-1, 0)
			}

			if rowIdx == 0 {
				// Render the header in bold with the body foreground.
				fg := m.fg
				chunks = append(chunks, style.TextChunk{
					Text:       strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad),
					Fg:         &fg,
					Attributes: style.AttrBold,
				})
			} else if len(cell.Chunks) > 0 {
				// Preserve inline formatting between the alignment padding.
				if leftPad > 0 {
					chunks = append(chunks, style.TextChunk{Text: strings.Repeat(" ", leftPad)})
				}
				chunks = append(chunks, cell.Chunks...)
				if rightPad > 0 {
					chunks = append(chunks, style.TextChunk{Text: strings.Repeat(" ", rightPad)})
				}
			} else {
				// Fill an empty cell to its complete column width.
				chunks = append(chunks, style.TextChunk{Text: strings.Repeat(" ", cellWidth)})
			}
		}
		chunks = append(chunks, style.TextChunk{Text: string(bc.Vertical), Fg: &borderFg})
		m.renderedLines = append(m.renderedLines, mdLine{chunks: chunks})

		if rowIdx == 0 {
			m.renderedLines = append(m.renderedLines, m.buildTableBorder(colWidths, bc.LeftT, bc.Cross, bc.RightT, bc.Horizontal, borderFg))
		}
	}

	m.renderedLines = append(m.renderedLines, m.buildTableBorder(colWidths, bc.BottomLeft, bc.BottomT, bc.BottomRight, bc.Horizontal, borderFg))
}

// buildTableBorder builds a horizontal table border line.
func (m *Markdown) buildTableBorder(colWidths []int, left, mid, right, fill rune, fg style.Color) mdLine {
	var sb strings.Builder
	sb.WriteRune(left)
	for i, w := range colWidths {
		for range w {
			sb.WriteRune(fill)
		}
		if i < len(colWidths)-1 {
			sb.WriteRune(mid)
		}
	}
	sb.WriteRune(right)
	return mdLine{chunks: []style.TextChunk{{Text: sb.String(), Fg: &fg}}}
}

// truncateStringToWidth returns the longest prefix that fits in maxWidth columns.
// It preserves complete grapheme clusters, including multi-codepoint emoji.
func truncateStringToWidth(s string, maxWidth int) string {
	var sb strings.Builder
	width := 0
	remaining := s
	for len(remaining) > 0 {
		cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
		remaining = rest
		if width+rw > maxWidth {
			break
		}
		sb.WriteString(cluster)
		width += rw
	}
	return sb.String()
}

// Markdown appearance and wrapping.

// SetFg sets the default foreground color.
func (m *Markdown) SetFg(c style.Color) {
	m.fg = c
	m.contentDirty = true // Inline styles are applied during parsing.
	m.RequestRender()
}

// SetBg sets the default background color.
func (m *Markdown) SetBg(c style.Color) {
	m.bg = c
	m.RequestRender()
}

// SetWrapMode sets the wrap mode.
func (m *Markdown) SetWrapMode(wm WrapMode) {
	if m.wrapMode != wm {
		m.wrapMode = wm
		m.wrapDirty = true
		m.RequestRender()
	}
}

// Fg returns the foreground color.
func (m *Markdown) Fg() style.Color { return m.fg }

// Bg returns the background color.
func (m *Markdown) Bg() style.Color { return m.bg }

// WrapModeValue returns the configured wrap mode.
func (m *Markdown) WrapModeValue() WrapMode { return m.wrapMode }

// SetHeight sets the markdown widget height dynamically after construction.
func (m *Markdown) SetHeight(s layout.Scalar) {
	m.LayoutNode().Height = s
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// Markdown options.

// MarkdownWithFg sets the foreground color.
func MarkdownWithFg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.fg = c }
}

// MarkdownWithBg sets the background color.
func MarkdownWithBg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.bg = c }
}

// MarkdownWithCodeFg sets the inline-code foreground and fenced-code fallback foreground.
// Syntax-highlighted tokens retain their highlighter colors.
func MarkdownWithCodeFg(c style.Color) MarkdownOption {
	return func(m *Markdown) {
		m.codeFg = c
		m.code = m.code.WithFg(c)
	}
}

// MarkdownWithCodeBg sets the background for inline and fenced code.
// Highlighted tokens use it when their syntax style has no background.
func MarkdownWithCodeBg(c style.Color) MarkdownOption {
	return func(m *Markdown) {
		m.codeBg = c
		m.code = m.code.WithBg(c)
	}
}

// MarkdownWithCode sets the complete inline-code style without changing fenced-code fallbacks.
func MarkdownWithCode(e style.Entry) MarkdownOption {
	return func(m *Markdown) { m.code = e }
}

// MarkdownWithCodeBlockBg sets the fenced-code background without changing inline code.
func MarkdownWithCodeBlockBg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.codeBg = c }
}

// MarkdownWithCodeBlockPadding adds n rows above and below fenced code and
// n columns before its text. Negative values become zero.
func MarkdownWithCodeBlockPadding(n int) MarkdownOption {
	return func(m *Markdown) { m.codeBlockPad = max(n, 0) }
}

// MarkdownWithCodeBlockMargin leaves n columns outside each side of a fenced-code
// background. Negative values become zero.
func MarkdownWithCodeBlockMargin(n int) MarkdownOption {
	return func(m *Markdown) { m.codeBlockMargin = max(n, 0) }
}

// MarkdownWithProsePadding insets paragraphs, headings, lists, and quotes by n columns.
// Top-level fenced code remains full width. Negative values become zero.
func MarkdownWithProsePadding(n int) MarkdownOption {
	return func(m *Markdown) { m.prosePad = max(n, 0) }
}

// MarkdownWithStrong sets the style for **strong** spans.
func MarkdownWithStrong(e style.Entry) MarkdownOption {
	return func(m *Markdown) { m.strong = e }
}

// MarkdownWithEmph sets the style for *emphasis* spans.
func MarkdownWithEmph(e style.Entry) MarkdownOption {
	return func(m *Markdown) { m.emph = e }
}

// MarkdownWithWrapMode sets the wrap mode.
func MarkdownWithWrapMode(wm WrapMode) MarkdownOption {
	return func(m *Markdown) { m.wrapMode = wm }
}

// MarkdownWithID sets the widget ID.
func MarkdownWithID(id string) MarkdownOption {
	return func(m *Markdown) { m.SetID(id) }
}

// MarkdownWithWidth sets the widget width.
func MarkdownWithWidth(s layout.Scalar) MarkdownOption {
	return func(m *Markdown) { m.LayoutNode().Width = s }
}

// MarkdownWithHeight sets the widget height.
func MarkdownWithHeight(s layout.Scalar) MarkdownOption {
	return func(m *Markdown) { m.LayoutNode().Height = s }
}

// MarkdownWithFlexGrow sets the flex-grow factor.
func MarkdownWithFlexGrow(v int) MarkdownOption {
	return func(m *Markdown) { m.LayoutNode().FlexGrow = v }
}

// MarkdownWithParser sets a custom MarkdownParser implementation.
// When nil (default), a GoldmarkParser is used.
func MarkdownWithParser(p MarkdownParser) MarkdownOption {
	return func(m *Markdown) { m.parser = p }
}

// MarkdownWithHighlighter sets a custom SyntaxHighlighter for fenced code
// blocks. When nil, a ChromaHighlighter with the "monokai" theme is used.
func MarkdownWithHighlighter(h SyntaxHighlighter) MarkdownOption {
	return func(m *Markdown) { m.highlighter = h }
}

// MarkdownWithTheme applies Markdown and code roles from t.
// Later options override individual roles. A nil theme leaves defaults unchanged.
//
// It also installs a Chroma style derived from t.Code. Apply MarkdownWithHighlighter
// afterward to use a different highlighter.
func MarkdownWithTheme(t *style.Theme) MarkdownOption {
	return func(m *Markdown) {
		if t == nil {
			return
		}
		m.fg = t.ResolveMarkdownText().Fg
		m.bg = t.Surface
		for i := range m.headings {
			m.headings[i] = t.ResolveMarkdownHeading(i + 1)
		}
		m.codeFg = t.CodeFg
		m.codeBg = t.CodeBg
		m.code = t.ResolveMarkdownCode()
		m.link = t.ResolveMarkdownLink()
		m.linkText = t.ResolveMarkdownLinkText()
		m.strong = t.ResolveMarkdownStrong()
		m.emph = t.ResolveMarkdownEmph()
		m.quote = t.ResolveMarkdownQuote()
		m.quoteBorder = t.ResolveMarkdownQuoteBorder()
		m.bullet = t.ResolveMarkdownBullet()
		m.listEnum = t.ResolveMarkdownListEnum()
		m.rule = t.ResolveMarkdownRule()
		// Keep fenced-code token colors consistent with the selected UI theme.
		m.highlighter = NewChromaHighlighterWithStyle(ChromaStyleForTheme(t))
	}
}
