package widget

import (
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

	mu       sync.Mutex // protects content + contentDirty from concurrent access
	content  string
	wrapMode WrapMode

	// Two-phase pipeline:
	// Phase 1 (parse): content → blocks (width-independent)
	// Phase 2 (wrap):  blocks + width → renderedLines
	blocks        []MdBlock
	renderedLines []mdLine
	contentDirty  bool // content changed → need re-parse + re-wrap
	wrapDirty     bool // width changed → need re-wrap only
	lastWrapWidth int  // width used for the last wrap

	parser MarkdownParser // pluggable parser (default: GoldmarkParser)

	fg style.Color
	bg style.Color

	// Heading colors
	h1Fg style.Color
	h2Fg style.Color
	h3Fg style.Color
	h4Fg style.Color
	h5Fg style.Color
	h6Fg style.Color

	// Syntax highlighting for fenced code blocks.
	highlighter SyntaxHighlighter // nil = monochrome fallback

	// Other element colors
	codeFg      style.Color
	codeBg      style.Color
	linkFg      style.Color
	quoteFg     style.Color
	quoteBorder style.Color
	bulletFg    style.Color
	ruleFg      style.Color
}

// ---------------------------------------------------------------------------
// Markdown Parser Interface
// ---------------------------------------------------------------------------

// MarkdownParser parses a markdown string into block-level IR.
// Implementations may use any parsing strategy (goldmark, regex, etc.)
// as long as they produce the same []MdBlock output.
type MarkdownParser interface {
	// Parse converts markdown content into a slice of blocks.
	// The styles parameter provides colors for styling inline elements.
	Parse(content string, styles MdStyles) []MdBlock
}

// MdStyles provides colors and styles for the markdown parser to apply
// to inline elements during parsing. This decouples the parser from the
// Markdown widget's internal color fields.
type MdStyles struct {
	Fg                                 style.Color
	CodeFg                             style.Color
	CodeBg                             style.Color
	LinkFg                             style.Color
	QuoteFg                            style.Color
	QuoteBorder                        style.Color
	BulletFg                           style.Color
	H1Fg, H2Fg, H3Fg, H4Fg, H5Fg, H6Fg style.Color
}

// HeadingColor returns the color for a heading level.
func (s MdStyles) HeadingColor(level int) style.Color {
	switch level {
	case 1:
		return s.H1Fg
	case 2:
		return s.H2Fg
	case 3:
		return s.H3Fg
	case 4:
		return s.H4Fg
	case 5:
		return s.H5Fg
	case 6:
		return s.H6Fg
	default:
		return s.Fg
	}
}

// ---------------------------------------------------------------------------
// Block-level intermediate representation (width-independent)
// ---------------------------------------------------------------------------

// MdBlockKind identifies a markdown block type.
type MdBlockKind int

const (
	MdBlockParagraph MdBlockKind = iota
	MdBlockHeading
	MdBlockCode
	MdBlockList        // unordered list container
	MdBlockOrderedList // ordered list container
	MdBlockListItem    // single list item (child of List/OrderedList)
	MdBlockQuote
	MdBlockRule
	MdBlockBlank
	MdBlockTable
)

// MdBlock is a parsed markdown block. Blocks form a tree: container blocks
// (List, OrderedList, Quote, ListItem) hold child blocks in Children.
type MdBlock struct {
	Kind        MdBlockKind
	Level       int               // heading level (1-6), list indent
	Prefix      []style.TextChunk // rendered prefix (bullet, number, quote marker)
	PrefixWidth int               // display width of prefix
	Body        []style.TextChunk // inline-parsed body chunks (not yet wrapped)
	Children    []MdBlock         // nested blocks (list items, quote contents, etc.)
	Lang        string            // fenced code block language (e.g. "go", "python")
	CodeLines   []string          // for code blocks: raw lines
	TableCells  [][]MdTableCell   // for tables: rows of cells with inline formatting
	TableAligns []int             // for tables: column alignment (-1=left, 0=center, 1=right)
}

// MdTableCell holds a table cell's inline content.
type MdTableCell struct {
	Chunks []style.TextChunk
}

// ---------------------------------------------------------------------------
// Wrapped output line (width-dependent)
// ---------------------------------------------------------------------------

// mdLine is a rendered markdown line (post-wrapping).
type mdLine struct {
	chunks []style.TextChunk
}

// mdLineWidth returns the display width of an mdLine.
func mdLineWidth(line mdLine) int {
	w := 0
	for _, c := range line.chunks {
		w += textbuf.StringWidth(c.Text)
	}
	return w
}

// MarkdownOption configures a Markdown widget.
type MarkdownOption func(*Markdown)

// NewMarkdown creates a Markdown widget.
func NewMarkdown(content string, opts ...MarkdownOption) *Markdown {
	m := &Markdown{
		BaseRenderable: render.NewBaseRenderable(),
		content:        content,
		wrapMode:       WrapWord,
		fg:             style.ColorWhite,
		bg:             style.ColorTransparent,
		contentDirty:   true,

		h1Fg: style.ColorBrightCyan,
		h2Fg: style.ColorBrightGreen,
		h3Fg: style.ColorBrightYellow,
		h4Fg: style.ColorBrightMagenta,
		h5Fg: style.ColorBrightBlue,
		h6Fg: style.ColorBrightRed,

		codeFg:      style.ColorFromInts(200, 200, 200, 255),
		codeBg:      style.ColorFromInts(30, 30, 30, 255),
		linkFg:      style.ColorBrightBlue,
		quoteFg:     style.ColorGray,
		quoteBorder: style.ColorBrightBlack,
		bulletFg:    style.ColorBrightBlue,
		ruleFg:      style.ColorGray,
	}

	for _, opt := range opts {
		opt(m)
	}

	// Default to goldmark parser if none provided.
	if m.parser == nil {
		m.parser = NewGoldmarkParser()
	}
	// Default to Chroma syntax highlighter for fenced code blocks.
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

		// Fast path: if we already have rendered lines and the proposed
		// width matches (or is larger than) the last wrap width, reuse
		// the cached line count. This avoids expensive re-wraps when
		// intrinsicMainSize probes with large/varying widths during
		// ancestor layout passes.
		//
		// Must also check wrapDirty: SetBlocks marks wrapDirty=true
		// without bumping contentDirty, so renderedLines may be stale
		// relative to the current m.blocks. Skipping this check leads
		// to the parent sizing the content box to the pre-swap line
		// count, clipping any new lines on render.
		if len(m.renderedLines) > 0 && !m.contentDirty && !m.wrapDirty && m.lastWrapWidth > 0 && w >= m.lastWrapWidth {
			h := len(m.renderedLines)
			if heightMode == layout.MeasureExactly {
				h = height
			} else if heightMode == layout.MeasureAtMost && h > height {
				h = height
			}
			return layout.MeasureResult{Width: w, Height: h}
		}

		// Re-wrap to the proposed width so line count is accurate.
		m.wrapToWidth(w)
		// Height is the number of rendered lines, with no floor: a block
		// that emits nothing (e.g. MdBlockRule under the current
		// thematic-break-as-spacing rendering) must report Height=0 so
		// the parent doesn't allocate an empty row for it. Without this,
		// StreamMarkdown — which puts each block in its own Markdown
		// child — would leave a phantom blank row wherever a previous
		// streamed message contained "\n\n---\n\n", visible as ghost
		// rows above the message's debug footer.
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

// ---------------------------------------------------------------------------
// Content Management
// ---------------------------------------------------------------------------

// SetContent sets the markdown content.
// Safe to call from any goroutine — parsing is deferred to the render
// loop via the lazy ensureWrapped / wrapToWidth path.
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

// AppendContent appends text (optimized for streaming).
// Safe to call from any goroutine.
func (m *Markdown) AppendContent(text string) {
	m.mu.Lock()
	m.content += text
	m.contentDirty = true
	m.mu.Unlock()
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// SetBlocks replaces the parsed block list directly, bypassing the
// content→parse phase. Only the wrap phase runs on the next render.
// Used by StreamMarkdown to distribute pre-parsed blocks to children.
//
// If the new blocks are deeply equal to the current ones, this is a
// no-op (keeps cached renderedLines and wrapDirty state intact). This
// lets streaming updates cheaply skip unchanged earlier children.
func (m *Markdown) SetBlocks(blocks []MdBlock) {
	m.mu.Lock()
	m.contentDirty = false // blocks are already parsed
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

// ---------------------------------------------------------------------------
// Wrapping
// ---------------------------------------------------------------------------

// mdStyles builds the MdStyles from the widget's current color fields.
func (m *Markdown) mdStyles() MdStyles {
	return MdStyles{
		Fg: m.fg, CodeFg: m.codeFg, CodeBg: m.codeBg,
		LinkFg: m.linkFg, QuoteFg: m.quoteFg, QuoteBorder: m.quoteBorder,
		BulletFg: m.bulletFg,
		H1Fg:     m.h1Fg, H2Fg: m.h2Fg, H3Fg: m.h3Fg,
		H4Fg: m.h4Fg, H5Fg: m.h5Fg, H6Fg: m.h6Fg,
	}
}

// ensureParsed re-parses content into blocks if contentDirty.
// Must be called from the main goroutine.
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
	m.wrapDirty = true // blocks changed → need re-wrap
}

// ensureWrapped runs both phases if needed and wraps to m.Width().
// Must be called from the main goroutine (RenderSelf).
func (m *Markdown) ensureWrapped() {
	m.ensureParsed()
	w := m.Width()
	if !m.wrapDirty && w == m.lastWrapWidth {
		return
	}
	m.wrapBlocks(w)
}

// wrapToWidth runs both phases if needed and wraps to the given width.
// Used by MeasureFunc to wrap to the proposed layout width.
// Must be called from the main goroutine.
func (m *Markdown) wrapToWidth(w int) {
	m.ensureParsed()
	if !m.wrapDirty && w == m.lastWrapWidth {
		return
	}
	m.wrapBlocks(w)
}

// OnResize triggers re-wrap when size changes.
func (m *Markdown) OnResize(width, height int) {
	m.wrapDirty = true
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// RenderSelf draws the markdown content directly from renderedLines.
// Each mdLine's chunks are rendered as styled grapheme clusters.
// A parent ScrollBox handles clipping via scissor rect.
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
					break // past widget width
				}
				buf.SetGraphemeCluster(cx, y, cluster, fg, bg, attrs)
				cx += rw
			}
		}
		y++
	}
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// renderCodeBlock renders a fenced/indented code block. Uses the
// SyntaxHighlighter when available and a language is specified;
// otherwise falls back to monochrome codeFg/codeBg.
func (m *Markdown) renderCodeBlock(blk MdBlock, indent string) {
	content := strings.Join(blk.CodeLines, "\n")

	// Try syntax highlighting.
	if m.highlighter != nil && blk.Lang != "" {
		highlighted, err := m.highlighter.Highlight(content, blk.Lang, m.codeFg, m.codeBg)
		if err == nil && len(highlighted) > 0 {
			for _, line := range highlighted {
				var chunks []style.TextChunk
				chunks = append(chunks, style.TextChunk{Text: indent, Bg: &m.codeBg})
				for _, span := range line.Spans {
					fg := span.Fg
					bg := span.Bg
					chunks = append(chunks, style.TextChunk{
						Text:       span.Text,
						Fg:         &fg,
						Bg:         &bg,
						Attributes: span.Attributes,
					})
				}
				m.renderedLines = append(m.renderedLines, mdLine{chunks: chunks})
			}
			return
		}
	}

	// Monochrome fallback.
	for _, line := range blk.CodeLines {
		m.renderedLines = append(m.renderedLines, mdLine{
			chunks: []style.TextChunk{{Text: indent + line, Fg: &m.codeFg, Bg: &m.codeBg}},
		})
	}
}

// isContinuationBlock returns true when two adjacent blocks belong to the
// same logical group and should NOT have a blank line inserted between them.
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
	// A thematic break already contributes its own leading blank via the
	// spacer inserted before it; skip the trailing spacer so the rule plus
	// its surrounding gap totals a single blank row between neighbours.
	if prev == MdBlockRule {
		return true
	}
	return false
}

// InsertBlockSpacing adds blank lines between blocks that need visual spacing.
// This is a shared utility used by MarkdownParser implementations.
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

// ---------------------------------------------------------------------------
// Phase 2: Wrap — []MdBlock + width → []mdLine (width-dependent)
// ---------------------------------------------------------------------------

// wrapBlocks converts blocks to renderedLines at the given width.
func (m *Markdown) wrapBlocks(w int) {
	m.renderedLines = nil
	m.lastWrapWidth = w
	m.wrapDirty = false
	m.wrapBlockList(m.blocks, w, 0)
}

// wrapBlockList recursively wraps a slice of blocks with the given indent.
func (m *Markdown) wrapBlockList(blocks []MdBlock, w int, indent int) {
	avail := w - indent
	if avail < 4 {
		avail = 4
	}
	indentStr := strings.Repeat(" ", indent)

	for _, blk := range blocks {
		switch blk.Kind {
		case MdBlockParagraph:
			m.addWrappedLine(
				[]style.TextChunk{{Text: indentStr}}, indent, w,
				blk.Body...,
			)
		case MdBlockHeading:
			m.addWrappedLine(
				append([]style.TextChunk{{Text: indentStr}}, blk.Body...),
				indent, w,
			)
		case MdBlockCode:
			// Top-level code blocks sit flush with the container edge.
			// Nested code (inside lists or blockquotes) keeps its parent
			// indent so it aligns with sibling content.
			m.renderCodeBlock(blk, indentStr)
		case MdBlockList, MdBlockOrderedList:
			// Container: render children (ListItems).
			m.wrapBlockList(blk.Children, w, indent)

		case MdBlockListItem:
			// Prefix + body on the first line.
			prefix := make([]style.TextChunk, 0, len(blk.Prefix)+1)
			if indent > 0 {
				prefix = append(prefix, style.TextChunk{Text: indentStr})
			}
			prefix = append(prefix, blk.Prefix...)
			m.addWrappedLine(prefix, indent+blk.PrefixWidth, w, blk.Body...)
			// Nested children (more paragraphs, code blocks, nested lists)
			// are indented by the prefix width.
			if len(blk.Children) > 0 {
				m.wrapBlockList(blk.Children, w, indent+blk.PrefixWidth)
			}

		case MdBlockQuote:
			// Prefix + first child paragraph body on the first line.
			prefix := make([]style.TextChunk, 0, len(blk.Prefix)+1)
			if indent > 0 {
				prefix = append(prefix, style.TextChunk{Text: indentStr})
			}
			prefix = append(prefix, blk.Prefix...)
			if len(blk.Children) > 0 && blk.Children[0].Kind == MdBlockParagraph {
				m.addWrappedLine(prefix, indent+blk.PrefixWidth, w, blk.Children[0].Body...)
				if len(blk.Children) > 1 {
					m.wrapBlockList(blk.Children[1:], w, indent+blk.PrefixWidth)
				}
			} else {
				// Quote with no paragraph body — just render prefix.
				m.renderedLines = append(m.renderedLines, mdLine{chunks: prefix})
				m.wrapBlockList(blk.Children, w, indent+blk.PrefixWidth)
			}

		case MdBlockRule:
			// Render thematic breaks as invisible vertical spacing rather
			// than a drawn line: the rule border is dropped and
			// replaced with a tall margin.
			// The rule itself emits nothing; InsertBlockSpacing contributes
			// a single blank line before the rule (and suppresses the one
			// after via isContinuationBlock) so surrounding paragraphs are
			// separated by exactly one blank row — the same gap as a plain
			// paragraph break.
		case MdBlockBlank:
			m.renderedLines = append(m.renderedLines, mdLine{})
		case MdBlockTable:
			m.renderTable(blk.TableCells, blk.TableAligns, avail)
		}
	}
}

// addWrappedLine adds chunks to renderedLines, optionally wrapping.
// prefix chunks are prepended, prefixWidth is their total display width.
// The remaining chunks form the body that gets wrapped.
func (m *Markdown) addWrappedLine(prefix []style.TextChunk, prefixWidth int, maxWidth int, body ...style.TextChunk) {
	if m.wrapMode == WrapNone || maxWidth <= 0 {
		// No wrapping — single line
		all := make([]style.TextChunk, 0, len(prefix)+len(body))
		all = append(all, prefix...)
		all = append(all, body...)
		m.renderedLines = append(m.renderedLines, mdLine{chunks: all})
		return
	}

	availWidth := max(maxWidth-prefixWidth, 4)

	// Calculate total body width
	totalWidth := 0
	for _, c := range body {
		totalWidth += m.StringWidth(c.Text)
	}

	if totalWidth <= availWidth {
		// Fits on one line
		all := make([]style.TextChunk, 0, len(prefix)+len(body))
		all = append(all, prefix...)
		all = append(all, body...)
		m.renderedLines = append(m.renderedLines, mdLine{chunks: all})
		return
	}

	// Need to wrap: flatten body chunks into grapheme clusters with style
	type styledCluster struct {
		text string
		base rune // first rune — used for word-break detection
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
			lineAvail = maxWidth // continuation lines use full width
		}

		lineStart := clusterIdx

		if m.wrapMode == WrapWord {
			// Find word boundary
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

			// If we found a word boundary and didn't consume all, break there
			if clusterIdx < len(allClusters) && lastBreak > lineStart {
				clusterIdx = lastBreak + 1
			}
		} else {
			// Char wrap
			for clusterIdx < len(allClusters) {
				rw := allClusters[clusterIdx].w
				if lineWidth+rw > lineAvail {
					break
				}
				lineWidth += rw
				clusterIdx++
			}
		}

		// Build chunks for this line
		var lineChunks []style.TextChunk
		if firstLine {
			lineChunks = append(lineChunks, prefix...)
		}

		// Group consecutive clusters with same style
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

		// Skip leading whitespace on continuation lines
		for clusterIdx < len(allClusters) && unicode.IsSpace(allClusters[clusterIdx].base) {
			clusterIdx++
		}
	}
}

// cellText extracts plain text from an MdTableCell.
func cellText(cell MdTableCell) string {
	return chunksText(cell.Chunks)
}

// renderTable renders a parsed table into mdLines.
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

	// Calculate column widths from cell content.
	colWidths := make([]int, numCols)
	for _, row := range rows {
		for c := 0; c < numCols && c < len(row); c++ {
			cw := m.StringWidth(cellText(row[c]))
			if cw > colWidths[c] {
				colWidths[c] = cw
			}
		}
	}

	// Add padding (1 space each side).
	for c := range colWidths {
		colWidths[c] += 2
	}

	borderFg := m.ruleFg
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
				// Header row: bold, use fg color.
				fg := m.fg
				chunks = append(chunks, style.TextChunk{
					Text:       strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad),
					Fg:         &fg,
					Attributes: style.AttrBold,
				})
			} else if len(cell.Chunks) > 0 {
				// Data row with inline formatting: pad left, emit styled chunks, pad right.
				if leftPad > 0 {
					chunks = append(chunks, style.TextChunk{Text: strings.Repeat(" ", leftPad)})
				}
				chunks = append(chunks, cell.Chunks...)
				if rightPad > 0 {
					chunks = append(chunks, style.TextChunk{Text: strings.Repeat(" ", rightPad)})
				}
			} else {
				// Empty cell.
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

// truncateStringToWidth truncates a string to fit within maxWidth display columns.
// Uses grapheme cluster iteration so multi-codepoint emoji are kept intact.
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

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

// SetFg sets the default foreground color.
func (m *Markdown) SetFg(c style.Color) {
	m.fg = c
	m.contentDirty = true // colors affect inline parsing
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

// WrapModeValue returns the current wrap mode.
func (m *Markdown) WrapModeValue() WrapMode { return m.wrapMode }

// SetHeight sets the markdown widget height dynamically after construction.
func (m *Markdown) SetHeight(s layout.Scalar) {
	m.LayoutNode().Height = s
	m.LayoutNode().MarkDirty()
	m.RequestRender()
}

// ---------------------------------------------------------------------------
// Markdown Options
// ---------------------------------------------------------------------------

// MarkdownWithFg sets the foreground color.
func MarkdownWithFg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.fg = c }
}

// MarkdownWithBg sets the background color.
func MarkdownWithBg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.bg = c }
}

// MarkdownWithCodeFg sets the default foreground color for inline code spans
// and monochrome fenced code blocks. Syntax-highlighted code still uses its
// theme's per-token colors.
func MarkdownWithCodeFg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.codeFg = c }
}

// MarkdownWithCodeBg sets the background color for inline code spans and
// fenced code blocks. Syntax-highlighted code uses this as the default bg
// for tokens that don't specify their own. Pass the surrounding container
// color when you want code to blend in with the message body.
func MarkdownWithCodeBg(c style.Color) MarkdownOption {
	return func(m *Markdown) { m.codeBg = c }
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

// MarkdownWithFlexGrow sets flex grow.
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

// MarkdownWithTheme applies every color role the Markdown widget uses from
// the given Theme. Later MarkdownWith... options (e.g. MarkdownWithFg) take
// precedence, so callers can override individual roles on top of a theme.
// If t is nil the widget's built-in defaults are retained.
//
// A Chroma highlighter is installed using a style derived from Theme.Code
// so syntax-highlighted code harmonises with the rest of the palette.
// Callers that need a specific pre-built Chroma theme can chain
// MarkdownWithHighlighter after MarkdownWithTheme.
func MarkdownWithTheme(t *style.Theme) MarkdownOption {
	return func(m *Markdown) {
		if t == nil {
			return
		}
		m.fg = t.TextStrong
		m.bg = t.Surface
		m.h1Fg = t.MarkdownHeading[0]
		m.h2Fg = t.MarkdownHeading[1]
		m.h3Fg = t.MarkdownHeading[2]
		m.h4Fg = t.MarkdownHeading[3]
		m.h5Fg = t.MarkdownHeading[4]
		m.h6Fg = t.MarkdownHeading[5]
		m.codeFg = t.CodeFg
		m.codeBg = t.CodeBg
		m.linkFg = t.MarkdownLink
		m.quoteFg = t.MarkdownQuoteFg
		m.quoteBorder = t.MarkdownQuoteBd
		m.bulletFg = t.MarkdownBullet
		m.ruleFg = t.MarkdownRule
		// Always derive the Chroma style from the theme's palette so
		// syntax colors harmonise with the rest of the UI. Callers
		// who want a specific pre-built Chroma theme can still install
		// one manually via MarkdownWithHighlighter after the theme.
		m.highlighter = NewChromaHighlighterWithStyle(ChromaStyleFromTheme(t))
	}
}
