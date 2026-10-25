package widget

import (
	"strings"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// TreeNode contains one displayed value and its child nodes.
type TreeNode struct {
	// Text is the node's displayed text.
	Text string
	// Value is caller-defined data for the node.
	Value any
	// Children are the nodes nested under it, in order.
	Children []*TreeNode
	// Expanded shows Children; toggling the node flips it.
	Expanded bool
}

// flatTreeItem pairs a visible node with its depth in the expanded tree.
type flatTreeItem struct {
	node  *TreeNode
	depth int
}

// Tree is a keyboard-navigable view of expandable TreeNode values.
type Tree struct {
	*render.BaseRenderable

	root          []*TreeNode
	selectedIndex int // Index in flatItems.
	scrollOffset  int

	textColor       style.Color
	selectedFg      style.Color
	selectedBg      style.Color
	branchColor     style.Color
	backgroundColor style.Color

	indentSize int

	// flatItems contains only nodes visible under the current expansion state.
	flatItems []flatTreeItem

	// OnNodeSelected is called when the user activates a leaf node.
	OnNodeSelected func(node *TreeNode)
}

// TreeOption configures a Tree.
type TreeOption func(*Tree)

// NewTree returns a focusable Tree rooted at root.
func NewTree(root []*TreeNode, opts ...TreeOption) *Tree {
	t := &Tree{
		BaseRenderable:  render.NewBaseRenderable(render.WithFocusable(true)),
		root:            root,
		textColor:       style.ColorWhite,
		selectedFg:      style.ColorWhite,
		selectedBg:      style.ColorFromInts(51, 153, 255, 255),
		branchColor:     style.ColorGray,
		backgroundColor: style.ColorTransparent,
		indentSize:      2,
	}

	for _, opt := range opts {
		opt(t)
	}

	t.flatten()

	return t
}

// SetRoot replaces the tree root nodes.
func (t *Tree) SetRoot(root []*TreeNode) {
	t.root = root
	t.selectedIndex = 0
	t.scrollOffset = 0
	t.flatten()
	t.RequestRender()
}

// flatten rebuilds the visible, depth-annotated node list.
func (t *Tree) flatten() {
	t.flatItems = nil
	for _, node := range t.root {
		t.flattenNode(node, 0)
	}
}

func (t *Tree) flattenNode(node *TreeNode, depth int) {
	t.flatItems = append(t.flatItems, flatTreeItem{node: node, depth: depth})
	if node.Expanded {
		for _, child := range node.Children {
			t.flattenNode(child, depth+1)
		}
	}
}

// RenderSelf draws the tree.
func (t *Tree) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	w := t.Width()
	h := t.Height()
	if w <= 0 || h <= 0 || len(t.flatItems) == 0 {
		return
	}

	x := t.X()
	y := t.Y()

	if !t.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, t.backgroundColor)
	}

	for i := t.scrollOffset; i < len(t.flatItems) && i-t.scrollOffset < h; i++ {
		item := t.flatItems[i]
		screenY := y + (i - t.scrollOffset)
		isSelected := i == t.selectedIndex

		fg := t.textColor
		bg := t.backgroundColor

		if isSelected && t.Focused() {
			fg = t.selectedFg
			bg = t.selectedBg
		}

		if !bg.IsTransparent() {
			buf.FillRect(x, screenY, w, 1, bg)
		}

		// Indent by depth before the expand/collapse indicator.
		var indent strings.Builder
		for d := 0; d < item.depth*t.indentSize; d++ {
			indent.WriteString(" ")
		}

		// Branches show their current expansion state; leaves reserve one cell.
		indicator := " "
		if len(item.node.Children) > 0 {
			if item.node.Expanded {
				indicator = symbol.TreeExpanded
			} else {
				indicator = symbol.TreeCollapsed
			}
		}

		text := truncateToWidth(indent.String()+indicator+" "+item.node.Text, w)
		buf.DrawText(text, x, screenY, fg)

		// Repaint branch indicators with their dedicated color.
		if len(item.node.Children) > 0 {
			indentWidth := item.depth * t.indentSize
			buf.DrawText(indicator, x+indentWidth, screenY, t.branchColor)
		}
	}
}

// HandleKeyPress moves, expands, collapses, or activates the selected node.
func (t *Tree) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(t.flatItems) == 0 {
		return false
	}

	handled := true
	switch {
	case e.Matches(input.KeyUp, 0) || e.Matches("k", 0):
		if t.selectedIndex > 0 {
			t.selectedIndex--
			t.ensureVisible()
			t.RequestRender()
		}

	case e.Matches(input.KeyDown, 0) || e.Matches("j", 0):
		if t.selectedIndex < len(t.flatItems)-1 {
			t.selectedIndex++
			t.ensureVisible()
			t.RequestRender()
		}

	case e.Matches(input.KeyReturn, 0) || e.Matches(input.KeySpace, 0):
		t.toggleSelected()

	case e.Matches(input.KeyLeft, 0):
		t.collapseSelected()

	case e.Matches(input.KeyRight, 0):
		t.expandSelected()

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

func (t *Tree) toggleSelected() {
	if t.selectedIndex < 0 || t.selectedIndex >= len(t.flatItems) {
		return
	}
	item := t.flatItems[t.selectedIndex]
	if len(item.node.Children) > 0 {
		item.node.Expanded = !item.node.Expanded
		t.flatten()
		t.RequestRender()
	} else if t.OnNodeSelected != nil {
		t.OnNodeSelected(item.node)
	}
}

func (t *Tree) expandSelected() {
	if t.selectedIndex < 0 || t.selectedIndex >= len(t.flatItems) {
		return
	}
	item := t.flatItems[t.selectedIndex]
	if len(item.node.Children) > 0 && !item.node.Expanded {
		item.node.Expanded = true
		t.flatten()
		t.RequestRender()
	}
}

func (t *Tree) collapseSelected() {
	if t.selectedIndex < 0 || t.selectedIndex >= len(t.flatItems) {
		return
	}
	item := t.flatItems[t.selectedIndex]
	if item.node.Expanded {
		item.node.Expanded = false
		t.flatten()
		t.RequestRender()
	}
}

func (t *Tree) ensureVisible() {
	h := t.Height()
	if t.selectedIndex < t.scrollOffset {
		t.scrollOffset = t.selectedIndex
	}
	if t.selectedIndex >= t.scrollOffset+h {
		t.scrollOffset = t.selectedIndex - h + 1
	}
}

// Tree options.

// TreeWithFg sets the text color.
func TreeWithFg(c style.Color) TreeOption {
	return func(t *Tree) { t.textColor = c }
}

// TreeWithSelectedBg sets the selected row's background color.
func TreeWithSelectedBg(c style.Color) TreeOption {
	return func(t *Tree) { t.selectedBg = c }
}

// TreeWithBranchColor sets the branch indicator color.
func TreeWithBranchColor(c style.Color) TreeOption {
	return func(t *Tree) { t.branchColor = c }
}

// TreeWithIndentSize sets the number of columns added for each depth level.
func TreeWithIndentSize(n int) TreeOption {
	return func(t *Tree) { t.indentSize = n }
}

// TreeWithOnNodeSelected sets the callback invoked when the user activates a leaf.
func TreeWithOnNodeSelected(fn func(*TreeNode)) TreeOption {
	return func(t *Tree) { t.OnNodeSelected = fn }
}

// TreeWithID sets the widget ID.
func TreeWithID(id string) TreeOption {
	return func(t *Tree) { t.SetID(id) }
}

// TreeWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func TreeWithLayout(opts ...layout.Option) TreeOption {
	return func(w *Tree) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}
