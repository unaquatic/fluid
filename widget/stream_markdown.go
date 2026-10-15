package widget

import (
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

	// BackgroundColor fills the entire widget behind its content.
	BackgroundColor style.Color

	// fullContent is the source most recently supplied to SetContent.
	fullContent string

	// parser and styles match the options used to construct child widgets.
	parser MarkdownParser
	styles MdStyles

	// allBlocks is reparsed whenever fullContent changes.
	allBlocks []MdBlock

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

	// MdBlockBlank values provide spacing, so the child layout needs no gap.
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

// RenderSelf fills the background across the full content area.
func (sm *StreamMarkdown) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	if !sm.BackgroundColor.IsTransparent() {
		buf.FillRect(sm.X(), sm.Y(), sm.Width(), sm.Height(), sm.BackgroundColor)
	}
}

// distributeBlocks assigns one block to each child, creating children as needed.
// Children are retained when content shrinks so future updates can reuse them.
func (sm *StreamMarkdown) distributeBlocks() {
	allBlocks := sm.allBlocks

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
			sm.children[i].SetBlocks(allBlocks[i : i+1])
		} else {
			sm.children[i].SetBlocks(nil)
		}
	}
}

// newChild returns a full-width Markdown configured with the shared child options.
func (sm *StreamMarkdown) newChild() *Markdown {
	opts := make([]MarkdownOption, len(sm.mdOpts))
	copy(opts, sm.mdOpts)
	opts = append(opts, MarkdownWithWidth(layout.Percent(100)))
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

// StreamMarkdownWithWidth sets the widget width.
func StreamMarkdownWithWidth(s layout.Scalar) StreamMarkdownOption {
	return func(sm *StreamMarkdown) { sm.LayoutNode().Width = s }
}

// StreamMarkdownWithFlexGrow sets flex grow.
func StreamMarkdownWithFlexGrow(v int) StreamMarkdownOption {
	return func(sm *StreamMarkdown) { sm.LayoutNode().FlexGrow = v }
}

// StreamMarkdownWithID sets the widget ID.
func StreamMarkdownWithID(id string) StreamMarkdownOption {
	return func(sm *StreamMarkdown) { sm.SetID(id) }
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
		sm.BackgroundColor = c
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
