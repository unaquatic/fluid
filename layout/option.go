package layout

// Option sets a layout property of a Node and marks the node dirty, so that
// applying it to a node already laid out takes effect at the next layout.
// Widgets take Options through their WithLayout options, such as
// widget.TextWithLayout, which apply them in order.
type Option func(*Node)

// set returns an Option that applies f to a node and marks the node dirty.
func set(f func(*Node)) Option {
	return func(n *Node) {
		f(n)
		n.MarkDirty()
	}
}

// WithWidth sets the node's width.
func WithWidth(s Scalar) Option { return set(func(n *Node) { n.Width = s }) }

// WithHeight sets the node's height.
func WithHeight(s Scalar) Option { return set(func(n *Node) { n.Height = s }) }

// WithMinWidth sets the node's minimum width.
func WithMinWidth(s Scalar) Option { return set(func(n *Node) { n.MinWidth = s }) }

// WithMaxWidth sets the node's maximum width.
func WithMaxWidth(s Scalar) Option { return set(func(n *Node) { n.MaxWidth = s }) }

// WithMinHeight sets the node's minimum height.
func WithMinHeight(s Scalar) Option { return set(func(n *Node) { n.MinHeight = s }) }

// WithMaxHeight sets the node's maximum height.
func WithMaxHeight(s Scalar) Option { return set(func(n *Node) { n.MaxHeight = s }) }

// WithFlexGrow sets the share of free space the node grows by.
func WithFlexGrow(v int) Option { return set(func(n *Node) { n.FlexGrow = v }) }

// WithFlexShrink sets the share of overflow the node shrinks by.
func WithFlexShrink(v int) Option { return set(func(n *Node) { n.FlexShrink = v }) }

// WithFlexBasis sets the node's main-axis size before growing and shrinking.
func WithFlexBasis(s Scalar) Option { return set(func(n *Node) { n.FlexBasis = s }) }

// WithFlexDirection sets the axis the node lays out its children along.
func WithFlexDirection(d FlexDirection) Option {
	return set(func(n *Node) { n.FlexDirection = d })
}

// WithFlexWrap sets whether the node wraps its children onto more lines.
func WithFlexWrap(w FlexWrap) Option { return set(func(n *Node) { n.FlexWrap = w }) }

// WithJustifyContent sets how the node places its children along the main
// axis.
func WithJustifyContent(j JustifyContent) Option {
	return set(func(n *Node) { n.JustifyContent = j })
}

// WithAlignItems sets how the node aligns its children along the cross axis.
func WithAlignItems(a Align) Option { return set(func(n *Node) { n.AlignItems = a }) }

// WithAlignSelf sets how the node aligns itself along its parent's cross
// axis, in place of the parent's AlignItems.
func WithAlignSelf(a Align) Option { return set(func(n *Node) { n.AlignSelf = a }) }

// WithGap sets the gap between the node's rows and between its columns,
// unless WithRowGap or WithColumnGap sets one apart.
func WithGap(v int) Option { return set(func(n *Node) { n.Gap = v }) }

// WithRowGap sets the gap between the node's rows when positive, in place of
// the Gap.
func WithRowGap(v int) Option { return set(func(n *Node) { n.RowGap = v }) }

// WithColumnGap sets the gap between the node's columns when positive, in
// place of the Gap.
func WithColumnGap(v int) Option { return set(func(n *Node) { n.ColumnGap = v }) }

// WithOverflow sets how the node treats content outside its bounds.
func WithOverflow(o Overflow) Option { return set(func(n *Node) { n.Overflow = o }) }

// WithPosition sets whether the node takes part in its parent's flex layout
// or, with PositionAbsolute, sits at the offsets WithTop, WithRight,
// WithBottom, and WithLeft set.
func WithPosition(p Position) Option { return set(func(n *Node) { n.Position = p }) }

// WithTop sets the node's offset from the top of its parent.
func WithTop(s Scalar) Option { return set(func(n *Node) { n.Top = s }) }

// WithRight sets the node's offset from the right of its parent.
func WithRight(s Scalar) Option { return set(func(n *Node) { n.Right = s }) }

// WithBottom sets the node's offset from the bottom of its parent.
func WithBottom(s Scalar) Option { return set(func(n *Node) { n.Bottom = s }) }

// WithLeft sets the node's offset from the left of its parent.
func WithLeft(s Scalar) Option { return set(func(n *Node) { n.Left = s }) }

// WithMargin sets the node's margin on all four sides.
func WithMargin(s Scalar) Option { return set(func(n *Node) { n.SetMargin(s) }) }

// WithMarginTop sets the node's top margin.
func WithMarginTop(s Scalar) Option { return set(func(n *Node) { n.MarginTop = s }) }

// WithMarginRight sets the node's right margin.
func WithMarginRight(s Scalar) Option { return set(func(n *Node) { n.MarginRight = s }) }

// WithMarginBottom sets the node's bottom margin.
func WithMarginBottom(s Scalar) Option { return set(func(n *Node) { n.MarginBottom = s }) }

// WithMarginLeft sets the node's left margin.
func WithMarginLeft(s Scalar) Option { return set(func(n *Node) { n.MarginLeft = s }) }

// WithPadding sets the node's padding on all four sides.
func WithPadding(s Scalar) Option { return set(func(n *Node) { n.SetPadding(s) }) }

// WithPaddingTop sets the node's top padding.
func WithPaddingTop(s Scalar) Option { return set(func(n *Node) { n.PaddingTop = s }) }

// WithPaddingRight sets the node's right padding.
func WithPaddingRight(s Scalar) Option { return set(func(n *Node) { n.PaddingRight = s }) }

// WithPaddingBottom sets the node's bottom padding.
func WithPaddingBottom(s Scalar) Option { return set(func(n *Node) { n.PaddingBottom = s }) }

// WithPaddingLeft sets the node's left padding.
func WithPaddingLeft(s Scalar) Option { return set(func(n *Node) { n.PaddingLeft = s }) }
