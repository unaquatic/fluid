package layout

// MeasureMode tells a MeasureFunc how to interpret the constraint on each axis.
type MeasureMode int

const (
	// MeasureExactly requires the supplied dimension.
	MeasureExactly MeasureMode = iota
	// MeasureAtMost permits any dimension up to the supplied limit.
	MeasureAtMost
	// MeasureUndefined requests the unconstrained intrinsic dimension.
	MeasureUndefined
)

// MeasureResult contains an intrinsic width and height in terminal cells.
type MeasureResult struct {
	// Width is the measured width in cells.
	Width int
	// Height is the measured height in cells.
	Height int
}

// MeasureFunc returns a node's intrinsic size under width and height
// constraints expressed in terminal cells.
//
// Layout calls the function when at least one axis is automatic. Leaf nodes
// and composite widgets that measure their own content can both provide it.
type MeasureFunc func(
	width int, widthMode MeasureMode,
	height int, heightMode MeasureMode,
) MeasureResult

// Node is one node in a flex layout tree.
type Node struct {
	// Container properties.

	// FlexDirection is the main axis and item order. The default is
	// DirectionRow.
	FlexDirection FlexDirection
	// FlexWrap is the line-wrapping mode. The default is WrapNone.
	FlexWrap FlexWrap
	// JustifyContent is the main-axis distribution. The default is
	// JustifyStart.
	JustifyContent JustifyContent
	// AlignItems is the cross-axis alignment. The default is AlignStretch.
	AlignItems Align
	// Gap is the fallback gap between rows and columns.
	Gap int
	// RowGap is the row gap when positive, overriding Gap.
	RowGap int
	// ColumnGap is the column gap when positive, overriding Gap.
	ColumnGap int

	// Item properties.

	// FlexGrow is the growth factor. The default is zero.
	FlexGrow int
	// FlexShrink is the shrink factor. The default is one.
	FlexShrink int
	// FlexBasis is the initial main-axis size. The default is Auto.
	FlexBasis Scalar
	// AlignSelf overrides AlignItems for this item. The default is AlignAuto.
	AlignSelf Align

	// Dimensions.

	// Width is the preferred width. The default is Auto.
	Width Scalar
	// Height is the preferred height. The default is Auto.
	Height Scalar
	// MinWidth is the minimum width. The default is Undefined.
	MinWidth Scalar
	// MaxWidth is the maximum width. The default is Undefined.
	MaxWidth Scalar
	// MinHeight is the minimum height. The default is Undefined.
	MinHeight Scalar
	// MaxHeight is the maximum height. The default is Undefined.
	MaxHeight Scalar

	// Intrinsic measurement.

	// Measure is an optional function that measures the node's content.
	Measure MeasureFunc

	// Box model.

	// MarginTop is the top margin. The default is zero points.
	MarginTop Scalar
	// MarginRight is the right margin. The default is zero points.
	MarginRight Scalar
	// MarginBottom is the bottom margin. The default is zero points.
	MarginBottom Scalar
	// MarginLeft is the left margin. The default is zero points.
	MarginLeft Scalar
	// PaddingTop is the top padding. The default is zero points.
	PaddingTop Scalar
	// PaddingRight is the right padding. The default is zero points.
	PaddingRight Scalar
	// PaddingBottom is the bottom padding. The default is zero points.
	PaddingBottom Scalar
	// PaddingLeft is the left padding. The default is zero points.
	PaddingLeft Scalar

	// Positioning.

	// Position is the positioning mode. The default is PositionRelative.
	Position Position
	// Top is the top offset. The default is Undefined.
	Top Scalar
	// Right is the right offset. The default is Undefined.
	Right Scalar
	// Bottom is the bottom offset. The default is Undefined.
	Bottom Scalar
	// Left is the left offset. The default is Undefined.
	Left Scalar
	// Overflow is the overflow mode. The default is OverflowVisible.
	Overflow Overflow
	// Sticky is the viewport edge the node pins to. The default is StickyNone.
	Sticky Sticky
	// Visibility controls layout and rendering visibility. The default is
	// Visible.
	Visibility Visibility

	// Resolved output.

	// Computed is the position and size from the last layout.
	Computed ComputedLayout

	// Tree links.

	// Parent is the parent node, or nil for a root.
	Parent *Node
	// Children are the child nodes, in layout order.
	Children []*Node

	// Layout state.
	dirty bool
	// Cache holds the inputs used to skip an unchanged subtree.
	Cache Cache
}

// NewNode returns a dirty layout node with flexbox defaults.
func NewNode() *Node {
	return &Node{
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

// AddChild appends child to n, removing it from any previous parent first.
func (n *Node) AddChild(child *Node) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n
	n.Children = append(n.Children, child)
	n.MarkDirty()
}

// AddChildAt inserts child at index after removing it from any previous parent.
// It appends child when index is outside the existing child range.
func (n *Node) AddChildAt(child *Node, index int) {
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

// RemoveChild removes child from n and clears its parent link. It does nothing
// when child is not a direct child of n.
func (n *Node) RemoveChild(child *Node) {
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
func (n *Node) ChildCount() int {
	return len(n.Children)
}

// MarkDirty marks n and every ancestor for layout recomputation.
func (n *Node) MarkDirty() {
	// Walk the full chain because a skipped Gone node can outlive a clean ancestor.
	for node := n; node != nil; node = node.Parent {
		node.dirty = true
	}
}

// IsDirty reports whether n requires layout recomputation.
func (n *Node) IsDirty() bool {
	return n.dirty
}

// ShiftDescendantPositions moves every descendant's absolute position and
// cached parent position by (dx, dy). It preserves internal layout when a
// clean subtree moves as a unit.
func (n *Node) ShiftDescendantPositions(dx, dy int) {
	for _, child := range n.Children {
		child.Computed.X += dx
		child.Computed.Y += dy
		child.Cache.ParentX += dx
		child.Cache.ParentY += dy
		child.ShiftDescendantPositions(dx, dy)
	}
}

// SetMargin sets all four margins to all.
func (n *Node) SetMargin(all Scalar) {
	n.MarginTop = all
	n.MarginRight = all
	n.MarginBottom = all
	n.MarginLeft = all
	n.MarkDirty()
}

// SetMarginXY sets both horizontal margins to x and both vertical margins to y.
func (n *Node) SetMarginXY(x, y Scalar) {
	n.MarginLeft = x
	n.MarginRight = x
	n.MarginTop = y
	n.MarginBottom = y
	n.MarkDirty()
}

// SetPadding sets padding on all four sides to all.
func (n *Node) SetPadding(all Scalar) {
	n.PaddingTop = all
	n.PaddingRight = all
	n.PaddingBottom = all
	n.PaddingLeft = all
	n.MarkDirty()
}

// SetPaddingXY sets both horizontal paddings to x and both vertical paddings to y.
func (n *Node) SetPaddingXY(x, y Scalar) {
	n.PaddingLeft = x
	n.PaddingRight = x
	n.PaddingTop = y
	n.PaddingBottom = y
	n.MarkDirty()
}

// SetSize sets the preferred width and height.
func (n *Node) SetSize(w, h Scalar) {
	n.Width = w
	n.Height = h
	n.MarkDirty()
}
