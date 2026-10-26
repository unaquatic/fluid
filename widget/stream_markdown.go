package widget

import (
	"reflect"
	"slices"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// StreamMarkdown renders incrementally changing Markdown through stable child widgets.
// It parses the full content once per update and assigns each block to one Markdown
// child. Unchanged children retain their wrapped output, so only changed blocks rewrap.
type StreamMarkdown struct {
	*render.BaseRenderable

	// bg fills the entire widget behind its content, as SetBg sets it.
	bg style.Color

	// fullContent is the source most recently supplied to SetContent.
	fullContent string

	// parser and styles match the options used to construct child widgets.
	parser markdownParser
	styles mdStyles

	// allBlocks is reparsed whenever fullContent changes.
	allBlocks []mdBlock

	// leader leads the content, as SetLeader sets it, and trailer follows
	// it, as SetTrailer sets it.
	leader, trailer []style.TextChunk

	// children contains one stable Markdown widget per block.
	children []*Markdown

	// mdOpts is applied to every child created after construction.
	mdOpts []MarkdownOption
}

// StreamMarkdownOption configures a StreamMarkdown widget.
type StreamMarkdownOption func(*StreamMarkdown)

// NewStreamMarkdown returns a column container for incrementally updated Markdown.
func NewStreamMarkdown(opts ...StreamMarkdownOption) *StreamMarkdown {
	sm := &StreamMarkdown{
		BaseRenderable: render.NewBaseRenderable(),
	}

	// mdBlockBlank values provide spacing, so the child layout needs no gap.
	sm.LayoutNode().FlexDirection = layout.DirectionColumn
	sm.LayoutNode().AlignItems = layout.AlignStretch

	for _, opt := range opts {
		opt(sm)
	}

	// Derive parsing state from the same options that each child receives.
	tmpl := NewMarkdown("", sm.mdOpts...)
	sm.parser = tmpl.parser
	sm.styles = tmpl.mdStyles()

	return sm
}

// StreamMarkdown content.

// SetContent parses and distributes the complete Markdown source.
func (sm *StreamMarkdown) SetContent(content string) {
	if sm.fullContent == content && len(sm.children) > 0 {
		return
	}
	sm.fullContent = content
	sm.allBlocks = sm.parser.Parse(content, sm.styles)
	sm.distributeBlocks()
}

// Content returns the full content string.
func (sm *StreamMarkdown) Content() string {
	return sm.fullContent
}

// Bg returns the color that fills the widget behind its content.
func (sm *StreamMarkdown) Bg() style.Color { return sm.bg }

// SetBg fills the widget and its Markdown blocks with c, as
// StreamMarkdownWithBg does, and requests a frame.
func (sm *StreamMarkdown) SetBg(c style.Color) {
	sm.bg = c
	sm.mdOpts = append(sm.mdOpts, MarkdownWithBg(c))
	for _, child := range sm.children {
		child.SetBg(c)
	}
	sm.RequestRender()
}

// SetLeader sets text that leads the content, on its first line, as
// withLeader places it, through later content updates too. Empty text
// removes it. Only the block that the text leads is wrapped again.
func (sm *StreamMarkdown) SetLeader(t style.StyledText) {
	if setChunks(&sm.leader, t) {
		sm.distributeBlocks()
	}
}

// Leader returns the text that leads the content.
func (sm *StreamMarkdown) Leader() style.StyledText {
	return style.StyledText{Chunks: sm.leader}
}

// SetTrailer sets text that follows the content, on its last line when it
// fits there, as withTrailer places it, through later content updates too.
// Empty text removes it. Only the block that the text follows is wrapped
// again.
func (sm *StreamMarkdown) SetTrailer(t style.StyledText) {
	if setChunks(&sm.trailer, t) {
		sm.distributeBlocks()
	}
}

// Trailer returns the text that follows the content.
func (sm *StreamMarkdown) Trailer() style.StyledText {
	return style.StyledText{Chunks: sm.trailer}
}

// setChunks sets *dst to a copy of the chunks of t, nil when there are none,
// and reports whether that changed it.
func setChunks(dst *[]style.TextChunk, t style.StyledText) bool {
	chunks := t.Chunks
	if len(chunks) == 0 {
		chunks = nil
	}
	if reflect.DeepEqual(*dst, chunks) {
		return false
	}
	*dst = slices.Clone(chunks)
	return true
}

// withLeader returns blocks with leader before their first inline text,
// parted from it by a space: at the start of the first paragraph, heading,
// or list item, also of one in a list or a quote. Blocks that start with a
// code block, table, or rule take leader as a paragraph of its own before
// them, parted as insertBlockSpacing parts blocks, and no blocks take it
// alone. withLeader leaves blocks as they are, copying the blocks it
// changes.
func withLeader(blocks []mdBlock, leader []style.TextChunk) []mdBlock {
	if len(leader) == 0 {
		return blocks
	}
	if out, ok := prependLeader(blocks, leader); ok {
		return out
	}
	para := mdBlock{Kind: mdBlockParagraph, Body: slices.Clone(leader)}
	if len(blocks) == 0 {
		return []mdBlock{para}
	}
	out := []mdBlock{para}
	if needsSpace(para, blocks[0]) {
		out = append(out, mdBlock{Kind: mdBlockBlank})
	}
	return append(out, blocks...)
}

// prependLeader adds leader before the first inline text of blocks, as
// withLeader does, and reports whether blocks start with such text.
func prependLeader(blocks []mdBlock, leader []style.TextChunk) ([]mdBlock, bool) {
	for i, b := range blocks {
		switch b.Kind {
		case mdBlockBlank:
			continue
		case mdBlockParagraph, mdBlockHeading, mdBlockListItem:
			// A list item's text is on its marker's line, before its nested
			// blocks.
			b.Body = leadBody(leader, b.Body)
		case mdBlockList, mdBlockOrderedList, mdBlockQuote:
			children, ok := prependLeader(b.Children, leader)
			if !ok {
				return blocks, false
			}
			b.Children = children
		default:
			return blocks, false
		}
		out := slices.Clone(blocks)
		out[i] = b
		return out, true
	}
	return blocks, false
}

// leadBody returns leader followed by a space and body, or leader alone when
// body is empty.
func leadBody(leader, body []style.TextChunk) []style.TextChunk {
	if len(body) == 0 {
		return slices.Clone(leader)
	}
	return slices.Concat(leader, []style.TextChunk{{Text: " "}}, body)
}

// withTrailer returns blocks with trailer after their last inline text,
// parted from it by a space: at the end of the last paragraph, heading, or
// list item, also of one in a list or a quote. Blocks that end with a code
// block, table, or rule take trailer as a paragraph of its own after them,
// parted as insertBlockSpacing parts blocks, and no blocks take it alone.
// withTrailer leaves blocks as they are, copying the blocks it changes.
func withTrailer(blocks []mdBlock, trailer []style.TextChunk) []mdBlock {
	if len(trailer) == 0 {
		return blocks
	}
	if out, ok := appendTrailer(blocks, trailer); ok {
		return out
	}
	para := mdBlock{Kind: mdBlockParagraph, Body: slices.Clone(trailer)}
	if len(blocks) == 0 {
		return []mdBlock{para}
	}
	out := slices.Clip(blocks)
	if needsSpace(blocks[len(blocks)-1], para) {
		out = append(out, mdBlock{Kind: mdBlockBlank})
	}
	return append(out, para)
}

// appendTrailer adds trailer after the last inline text of blocks, as
// withTrailer does, and reports whether blocks end with such text.
func appendTrailer(blocks []mdBlock, trailer []style.TextChunk) ([]mdBlock, bool) {
	for i, b := range slices.Backward(blocks) {
		switch b.Kind {
		case mdBlockBlank:
			continue
		case mdBlockParagraph, mdBlockHeading:
			b.Body = trailBody(b.Body, trailer)
		case mdBlockListItem:
			// The item's text comes before its nested blocks.
			children, ok := appendTrailer(b.Children, trailer)
			switch {
			case ok:
				b.Children = children
			case slices.ContainsFunc(b.Children, func(c mdBlock) bool { return c.Kind != mdBlockBlank }):
				return blocks, false
			default:
				b.Body = trailBody(b.Body, trailer)
			}
		case mdBlockList, mdBlockOrderedList, mdBlockQuote:
			children, ok := appendTrailer(b.Children, trailer)
			if !ok {
				return blocks, false
			}
			b.Children = children
		default:
			return blocks, false
		}
		out := slices.Clone(blocks)
		out[i] = b
		return out, true
	}
	return blocks, false
}

// trailBody returns body followed by a space and trailer, or trailer alone
// when body is empty.
func trailBody(body, trailer []style.TextChunk) []style.TextChunk {
	if len(body) == 0 {
		return slices.Clone(trailer)
	}
	return slices.Concat(body, []style.TextChunk{{Text: " "}}, trailer)
}

// RenderSelf fills the background across the full content area.
func (sm *StreamMarkdown) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	if !sm.bg.IsTransparent() {
		buf.FillRect(sm.X(), sm.Y(), sm.Width(), sm.Height(), sm.bg)
	}
}

// distributeBlocks assigns one block to each child, creating children as needed,
// with the leader before the first of them and the trailer after the last.
// Children are retained when content shrinks so future updates can reuse them.
func (sm *StreamMarkdown) distributeBlocks() {
	allBlocks := withTrailer(withLeader(sm.allBlocks, sm.leader), sm.trailer)

	numChildren := len(allBlocks)
	if numChildren == 0 {
		numChildren = 1
	}

	// Grow the stable child pool to fit the current block count.
	for len(sm.children) < numChildren {
		sm.addChild(sm.newChild())
	}

	// Clear retained children that have no corresponding block in this update.
	for i := 0; i < numChildren; i++ {
		if i < len(allBlocks) {
			sm.children[i].setBlocks(allBlocks[i : i+1])
		} else {
			sm.children[i].setBlocks(nil)
		}
	}
}

// newChild returns a full-width Markdown configured with the shared child options.
func (sm *StreamMarkdown) newChild() *Markdown {
	opts := make([]MarkdownOption, len(sm.mdOpts))
	copy(opts, sm.mdOpts)
	opts = append(opts, MarkdownWithLayout(layout.WithWidth(layout.Percent(100))))
	return NewMarkdown("", opts...)
}

// addChild records child and adds it to the render tree.
func (sm *StreamMarkdown) addChild(child *Markdown) {
	sm.children = append(sm.children, child)
	sm.Add(child)
}

// StreamMarkdown options.

// StreamMarkdownWithOpts sets the MarkdownOption list applied to each child.
func StreamMarkdownWithOpts(opts ...MarkdownOption) StreamMarkdownOption {
	return func(sm *StreamMarkdown) { sm.mdOpts = opts }
}

// StreamMarkdownWithID sets the widget ID.
func StreamMarkdownWithID(id string) StreamMarkdownOption {
	return func(sm *StreamMarkdown) { sm.SetID(id) }
}

// StreamMarkdownWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func StreamMarkdownWithLayout(opts ...layout.Option) StreamMarkdownOption {
	return func(w *StreamMarkdown) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}

// StreamMarkdownWithFg sets the foreground color on each child Markdown widget.
func StreamMarkdownWithFg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithFg(c))
	}
}

// StreamMarkdownWithBg sets the widget and child Markdown backgrounds.
func StreamMarkdownWithBg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.bg = c
		sm.mdOpts = append(sm.mdOpts, MarkdownWithBg(c))
	}
}

// StreamMarkdownWithCodeFg sets the inline and fallback code foreground on each child.
func StreamMarkdownWithCodeFg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeFg(c))
	}
}

// StreamMarkdownWithCodeBg sets the inline and fenced code background on each child.
func StreamMarkdownWithCodeBg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBg(c))
	}
}

// StreamMarkdownWithCodeBlockBg sets only the fenced-block background on each child.
func StreamMarkdownWithCodeBlockBg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBlockBg(c))
	}
}

// StreamMarkdownWithCodeBlockPadding sets fenced-block padding on each child.
func StreamMarkdownWithCodeBlockPadding(n int) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBlockPadding(n))
	}
}

// StreamMarkdownWithCodeBlockPaddingY sets the blank rows above and below
// fenced blocks on each child, in place of those the padding sets.
func StreamMarkdownWithCodeBlockPaddingY(n int) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBlockPaddingY(n))
	}
}

// StreamMarkdownWithCodeBlockMargin sets fenced-block margins on each child.
func StreamMarkdownWithCodeBlockMargin(n int) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBlockMargin(n))
	}
}

// StreamMarkdownWithCodeBlockFit sets whether each child fits fenced-block
// backgrounds to their widest line.
func StreamMarkdownWithCodeBlockFit(v bool) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithCodeBlockFit(v))
	}
}

// StreamMarkdownWithProsePadding sets prose padding on each child.
func StreamMarkdownWithProsePadding(n int) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithProsePadding(n))
	}
}

// StreamMarkdownWithTheme applies t to children created by NewStreamMarkdown.
// Options applied after it can override individual theme values.
func StreamMarkdownWithTheme(t *style.Theme) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithTheme(t))
	}
}
