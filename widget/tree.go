package widget

import (
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// TreeNode represents a node in the tree.
type TreeNode struct {
	Text     string
	Value    any
	Children []*TreeNode
	Expanded bool
}

// flatTreeItem is a flattened tree node for rendering.
type flatTreeItem struct {
	node  *TreeNode
	depth int
}

// Tree is an expandable/collapsible tree view with key navigation.
type Tree struct {
	*render.BaseRenderable

	root          []*TreeNode
	selectedIndex int // index in flattened list
	scrollOffset  int

	textColor       style.Color
	selectedFg      style.Color
	selectedBg      style.Color
	branchColor     style.Color
	backgroundColor style.Color

	indentSize int

	// Flattened view
	flatItems []flatTreeItem

	OnNodeSelected func(node *TreeNode)
}

// TreeOption configures a Tree.
type TreeOption func(*Tree)

// NewTree creates a Tree widget.
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

// flatten builds the flat list from the tree.
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
func (t *Tree) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
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

		// Build prefix
		var indent strings.Builder
		for d := 0; d < item.depth*t.indentSize; d++ {
			indent.WriteString(" ")
		}

		// Expand indicator
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

		// Draw branch indicator in a different color
		if len(item.node.Children) > 0 {
			indentWidth := item.depth * t.indentSize
			buf.DrawText(indicator, x+indentWidth, screenY, t.branchColor)
		}
	}
}

// HandleKeyPress processes keyboard input.
func (t *Tree) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(t.flatItems) == 0 {
		return false
	}

	handled := true
	switch {
	case e.Matches(input.KeyUp) || e.Matches("k"):
		if t.selectedIndex > 0 {
			t.selectedIndex--
			t.ensureVisible()
			t.RequestRender()
		}

	case e.Matches(input.KeyDown) || e.Matches("j"):
		if t.selectedIndex < len(t.flatItems)-1 {
			t.selectedIndex++
			t.ensureVisible()
			t.RequestRender()
		}

	case e.Matches(input.KeyReturn) || e.Matches(input.KeySpace):
		t.toggleSelected()

	case e.Matches(input.KeyLeft):
		t.collapseSelected()

	case e.Matches(input.KeyRight):
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

// --- Tree Options ---

// TreeWithTextColor sets the text color.
func TreeWithTextColor(c style.Color) TreeOption {
	return func(t *Tree) { t.textColor = c }
}

// TreeWithSelectedBg sets the selected background.
func TreeWithSelectedBg(c style.Color) TreeOption {
	return func(t *Tree) { t.selectedBg = c }
}

// TreeWithBranchColor sets the branch indicator color.
func TreeWithBranchColor(c style.Color) TreeOption {
	return func(t *Tree) { t.branchColor = c }
}

// TreeWithIndentSize sets the indent size per depth level.
func TreeWithIndentSize(n int) TreeOption {
	return func(t *Tree) { t.indentSize = n }
}

// TreeWithOnNodeSelected sets the node selected callback.
func TreeWithOnNodeSelected(fn func(*TreeNode)) TreeOption {
	return func(t *Tree) { t.OnNodeSelected = fn }
}

// TreeWithID sets the widget ID.
func TreeWithID(id string) TreeOption {
	return func(t *Tree) { t.SetID(id) }
}

// TreeWithWidth sets the widget width.
func TreeWithWidth(s layout.Scalar) TreeOption {
	return func(t *Tree) { t.LayoutNode().Width = s }
}

// TreeWithHeight sets the widget height.
func TreeWithHeight(s layout.Scalar) TreeOption {
	return func(t *Tree) { t.LayoutNode().Height = s }
}
