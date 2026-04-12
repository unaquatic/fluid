package widget

import (
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// StreamMarkdown is a drop-in replacement for Markdown that splits streamed
// content across multiple child Markdown widgets. The full content is parsed
// centrally into []MdBlock, then each block is assigned to its own child
// Markdown widget via SetBlocks. Children whose block hasn't changed skip
// re-wrap, keeping per-frame cost proportional to the actively-growing
// tail block.
type StreamMarkdown struct {
	*render.BaseRenderable

	// Content tracking
	fullContent string

	// Central parser + styles (derived from mdOpts at construction)
	parser MarkdownParser
	styles MdStyles

	// Parsed blocks for the full content (re-parsed each SetContent)
	allBlocks []MdBlock

	// Child widgets
	children []*Markdown // one per block chunk

	// Options to apply to each child Markdown widget
	mdOpts []MarkdownOption
}

// StreamMarkdownOption configures a StreamMarkdown widget.
type StreamMarkdownOption func(*StreamMarkdown)

// NewStreamMarkdown creates a StreamMarkdown widget.
func NewStreamMarkdown(opts ...StreamMarkdownOption) *StreamMarkdown {
	sm := &StreamMarkdown{
		BaseRenderable: render.NewBaseRenderable(),
	}

	// Column layout — no gap because MdBlockBlank entries in the block
	// list already provide inter-block spacing.
	sm.LayoutNode().FlexDirection = layout.DirectionColumn
	sm.LayoutNode().AlignItems = layout.AlignStretch

	for _, opt := range opts {
		opt(sm)
	}

	// Derive parser and styles from a template Markdown widget so we
	// parse with the same colors/options the children will render with.
	tmpl := NewMarkdown("", sm.mdOpts...)
	sm.parser = tmpl.parser
	sm.styles = tmpl.mdStyles()

	return sm
}

// ---------------------------------------------------------------------------
// Public API (matches Markdown's interface used by chat.go)
// ---------------------------------------------------------------------------

// SetContent sets the full markdown content. Parses it into blocks
// centrally, splits the block list, and distributes slices to children.
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

// ---------------------------------------------------------------------------
// Block distribution
// ---------------------------------------------------------------------------

// distributeBlocks assigns one block per child Markdown widget, adding
// children as needed (never removed). Each child's SetBlocks short-circuits
// if its block is unchanged, so only the tail child re-wraps during
// streaming updates.
func (sm *StreamMarkdown) distributeBlocks() {
	allBlocks := sm.allBlocks

	numChildren := len(allBlocks)
	if numChildren == 0 {
		numChildren = 1
	}

	// Add children as needed.
	for len(sm.children) < numChildren {
		sm.addChild(sm.newChild())
	}

	// One block per child.
	for i := 0; i < numChildren; i++ {
		if i < len(allBlocks) {
			sm.children[i].SetBlocks(allBlocks[i : i+1])
		} else {
			sm.children[i].SetBlocks(nil)
		}
	}
}

// newChild creates a new Markdown widget with the configured options.
func (sm *StreamMarkdown) newChild() *Markdown {
	opts := make([]MarkdownOption, len(sm.mdOpts))
	copy(opts, sm.mdOpts)
	opts = append(opts, MarkdownWithWidth(layout.Percent(100)))
	return NewMarkdown("", opts...)
}

// addChild adds a Markdown widget as a child of this StreamMarkdown.
func (sm *StreamMarkdown) addChild(child *Markdown) {
	sm.children = append(sm.children, child)
	sm.Add(child)
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

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

// StreamMarkdownWithFg is a convenience that adds MarkdownWithFg to child opts.
func StreamMarkdownWithFg(c style.Color) StreamMarkdownOption {
	return func(sm *StreamMarkdown) {
		sm.mdOpts = append(sm.mdOpts, MarkdownWithFg(c))
	}
}
