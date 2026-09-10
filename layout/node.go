package layout

// --- MeasureFunc ---

// MeasureMode tells a MeasureFunc how to interpret the constraint on each axis.
type MeasureMode int

const (
	// MeasureExactly means the dimension is fixed; the node must use it.
	MeasureExactly MeasureMode = iota
	// MeasureAtMost means the dimension is an upper bound; the node may be smaller.
	MeasureAtMost
	// MeasureUndefined means no constraint; return the intrinsic size.
	MeasureUndefined
)

// MeasureResult holds the intrinsic size returned by a MeasureFunc.
// Dimensions are in integer cells — the natural unit for a TUI.
type MeasureResult struct {
	Width  int
	Height int
}

// MeasureFunc is a callback that returns the intrinsic size of a node.
// It receives the available width/height (in cells) and the mode for each axis.
// This is analogous to Yoga's YGMeasureFunc, but uses integer cells since
// terminal UIs operate on a discrete character grid.
//
// The function is consulted when the node has Auto sizing on at least one
// axis. It works for both leaf nodes and composite widgets that manage their
// own children (e.g. Diff, which knows its total line count).
type MeasureFunc func(
	width int, widthMode MeasureMode,
	height int, heightMode MeasureMode,
) MeasureResult

// LayoutNode represents a single node in the flex layout tree.
type LayoutNode struct {
	// --- Container properties ---
	FlexDirection  FlexDirection  // default: DirectionRow
	FlexWrap       FlexWrap       // default: WrapNone
	JustifyContent JustifyContent // default: JustifyStart
	AlignItems     Align          // default: AlignStretch
	Gap            int            // shorthand for RowGap + ColumnGap
	RowGap         int            // gap between rows (cross-axis for row direction)
	ColumnGap      int            // gap between columns (main-axis for row direction)

	// --- Item properties ---
	FlexGrow   int    // default: 0
	FlexShrink int    // default: 1
	FlexBasis  Scalar // default: Auto
	AlignSelf  Align  // default: AlignAuto (inherit from parent)

	// --- Dimensions ---
	Width     Scalar // default: Auto
	Height    Scalar // default: Auto
	MinWidth  Scalar // default: Undefined
	MaxWidth  Scalar // default: Undefined
	MinHeight Scalar // default: Undefined
	MaxHeight Scalar // default: Undefined

	// --- Intrinsic measurement ---
	Measure MeasureFunc // optional: called for leaf nodes with Auto sizing

	// --- Box model ---
	MarginTop     Scalar // default: Points(0)
	MarginRight   Scalar
	MarginBottom  Scalar
	MarginLeft    Scalar
	PaddingTop    Scalar // default: Points(0)
	PaddingRight  Scalar
	PaddingBottom Scalar
	PaddingLeft   Scalar

	// --- Positioning ---
	Position   Position   // default: PositionRelative
	Top        Scalar     // default: Undefined
	Right      Scalar     // default: Undefined
	Bottom     Scalar     // default: Undefined
	Left       Scalar     // default: Undefined
	Overflow   Overflow   // default: OverflowVisible
	Sticky     Sticky     // default: StickyNone
	Visibility Visibility // default: Visible

	// --- Output ---
	Computed ComputedLayout

	// --- Tree ---
	Parent   *LayoutNode
	Children []*LayoutNode

	// --- Internal ---
	dirty bool
	Cache LayoutCache // Yoga-style layout cache for subtree skip
}

// NewNode creates a LayoutNode with sensible defaults matching CSS flexbox.
func NewNode() *LayoutNode {
	return &LayoutNode{
		FlexDirection:  DirectionRow,
		FlexWrap:       WrapNone,
		JustifyContent: JustifyStart,
		AlignItems:     AlignStretch,

		FlexGrow:   0,
		FlexShrink: 1,
		FlexBasis:  Auto(),
		AlignSelf:  AlignAuto,

		Width:     Auto(),
		Height:    Auto(),
		MinWidth:  Undefined(),
		MaxWidth:  Undefined(),
		MinHeight: Undefined(),
		MaxHeight: Undefined(),

		MarginTop:     Points(0),
		MarginRight:   Points(0),
		MarginBottom:  Points(0),
		MarginLeft:    Points(0),
		PaddingTop:    Points(0),
		PaddingRight:  Points(0),
		PaddingBottom: Points(0),
		PaddingLeft:   Points(0),

		Position: PositionRelative,
		Top:      Undefined(),
		Right:    Undefined(),
		Bottom:   Undefined(),
		Left:     Undefined(),
		Overflow: OverflowVisible,

		dirty: true,
	}
}

// --- Tree Manipulation ---

// AddChild appends a child to this node and sets its parent.
func (n *LayoutNode) AddChild(child *LayoutNode) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n
	n.Children = append(n.Children, child)
	n.MarkDirty()
}

// AddChildAt inserts a child at the given index.
// If index is out of range, the child is appended.
func (n *LayoutNode) AddChildAt(child *LayoutNode, index int) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n

	if index < 0 || index >= len(n.Children) {
		n.Children = append(n.Children, child)
	} else {
		n.Children = append(n.Children, nil)
		copy(n.Children[index+1:], n.Children[index:])
		n.Children[index] = child
	}
	n.MarkDirty()
}

// RemoveChild removes a child from this node and clears its parent.
func (n *LayoutNode) RemoveChild(child *LayoutNode) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.Parent = nil
			n.MarkDirty()
			return
		}
	}
}

// ChildCount returns the number of direct children.
func (n *LayoutNode) ChildCount() int {
	return len(n.Children)
}

// --- Dirty Tracking ---

// MarkDirty marks this node and all ancestors as needing layout.
func (n *LayoutNode) MarkDirty() {
	// Skipped Gone nodes can remain dirty after their ancestors become clean.
	for node := n; node != nil; node = node.Parent {
		node.dirty = true
	}
}

// IsDirty returns true if this node needs layout recomputation.
func (n *LayoutNode) IsDirty() bool {
	return n.dirty
}

// ShiftDescendantPositions adjusts the absolute Computed.X/Y of all
// descendants by (dx, dy). Used when a clean subtree moved because a
// sibling changed size — the internal layout is identical, only the
// absolute offset changed. Much cheaper than full re-layout (no scalar
// resolution, no flex math, no MeasureFunc calls).
func (n *LayoutNode) ShiftDescendantPositions(dx, dy int) {
	for _, child := range n.Children {
		child.Computed.X += dx
		child.Computed.Y += dy
		child.Cache.ParentX += dx
		child.Cache.ParentY += dy
		child.ShiftDescendantPositions(dx, dy)
	}
}

// --- Shorthand Setters ---

// SetMargin sets all four margins to the same value.
func (n *LayoutNode) SetMargin(all Scalar) {
	n.MarginTop = all
	n.MarginRight = all
	n.MarginBottom = all
	n.MarginLeft = all
	n.MarkDirty()
}

// SetMarginXY sets horizontal (x) and vertical (y) margins.
func (n *LayoutNode) SetMarginXY(x, y Scalar) {
	n.MarginLeft = x
	n.MarginRight = x
	n.MarginTop = y
	n.MarginBottom = y
	n.MarkDirty()
}

// SetPadding sets all four paddings to the same value.
func (n *LayoutNode) SetPadding(all Scalar) {
	n.PaddingTop = all
	n.PaddingRight = all
	n.PaddingBottom = all
	n.PaddingLeft = all
	n.MarkDirty()
}

// SetPaddingXY sets horizontal (x) and vertical (y) paddings.
func (n *LayoutNode) SetPaddingXY(x, y Scalar) {
	n.PaddingLeft = x
	n.PaddingRight = x
	n.PaddingTop = y
	n.PaddingBottom = y
	n.MarkDirty()
}

// SetSize sets width and height.
func (n *LayoutNode) SetSize(w, h Scalar) {
	n.Width = w
	n.Height = h
	n.MarkDirty()
}
