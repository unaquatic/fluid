package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// Box lays out child renderables with optional borders, background fill, and gaps.
type Box struct {
	*render.BaseRenderable

	BackgroundColor    style.Color
	BorderStyle        style.BorderStyle
	Border             style.Border
	BorderColor        style.Color
	ShouldFill         bool
	Title              string
	TitleAlignment     layout.Alignment
	FocusedBorderColor style.Color
	Gap                layout.Scalar
	RowGap             layout.Scalar
	ColumnGap          layout.Scalar
}

// BoxOption configures a Box during construction.
type BoxOption func(*Box)

// NewBox returns a column-oriented Box with optional configuration.
func NewBox(opts ...BoxOption) *Box {
	b := &Box{
		BaseRenderable:     render.NewBaseRenderable(),
		BackgroundColor:    style.ColorTransparent,
		BorderStyle:        style.BorderNone,
		Border:             style.BorderNoBorder(),
		BorderColor:        style.ColorWhite,
		FocusedBorderColor: style.ColorTransparent,
		TitleAlignment:     layout.TextAlignLeft,
	}

	// Containers stack children vertically unless an option changes the direction.
	b.LayoutNode().FlexDirection = layout.DirectionColumn

	for _, opt := range opts {
		opt(b)
	}

	// Keep children inside every active border side.
	b.applyBorderPadding()

	// Resolve the general and axis-specific gap options after all options run.
	b.applyGap()

	return b
}

// applyBorderPadding reserves at least one cell inside each active border side.
func (b *Box) applyBorderPadding() {
	sides := style.GetBorderSides(b.Border)
	ln := b.LayoutNode()

	if sides.Top {
		if pt, ok := ln.PaddingTop.Resolve(0); !ok || pt < 1 {
			ln.PaddingTop = layout.Points(1)
		}
	}
	if sides.Bottom {
		if pb, ok := ln.PaddingBottom.Resolve(0); !ok || pb < 1 {
			ln.PaddingBottom = layout.Points(1)
		}
	}
	if sides.Left {
		if pl, ok := ln.PaddingLeft.Resolve(0); !ok || pl < 1 {
			ln.PaddingLeft = layout.Points(1)
		}
	}
	if sides.Right {
		if pr, ok := ln.PaddingRight.Resolve(0); !ok || pr < 1 {
			ln.PaddingRight = layout.Points(1)
		}
	}
}

// applyGap copies resolved gap options to the layout node.
func (b *Box) applyGap() {
	ln := b.LayoutNode()
	if g, ok := b.Gap.Resolve(0); ok {
		ln.Gap = g
		ln.RowGap = g
		ln.ColumnGap = g
	}
	if g, ok := b.RowGap.Resolve(0); ok {
		ln.RowGap = g
	}
	if g, ok := b.ColumnGap.Resolve(0); ok {
		ln.ColumnGap = g
	}
}

// RenderSelf draws the box background, border, and title.
func (b *Box) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := b.Width()
	h := b.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := b.X()
	y := b.Y()

	// A partial border is active when any side is enabled.
	sides := style.GetBorderSides(b.Border)
	hasBorder := sides.Top || sides.Right || sides.Bottom || sides.Left

	// Fill only the content area when a border is present.
	if b.ShouldFill && !b.BackgroundColor.IsTransparent() {
		if hasBorder {
			inner := buffer.InnerRect(x, y, w, h, sides)
			if inner.Width > 0 && inner.Height > 0 {
				buf.FillRect(inner.X, inner.Y, inner.Width, inner.Height, b.BackgroundColor)
			}
		} else {
			buf.FillRect(x, y, w, h, b.BackgroundColor)
		}
	}

	// Focus color overrides the normal border color when configured.
	if hasBorder {
		borderColor := b.BorderColor
		if b.Focused() && !b.FocusedBorderColor.IsTransparent() {
			borderColor = b.FocusedBorderColor
		}

		buf.DrawBorder(buffer.BorderDrawOptions{
			X:               x,
			Y:               y,
			Width:           w,
			Height:          h,
			Style:           b.BorderStyle,
			Border:          b.Border,
			Color:           borderColor,
			BackgroundColor: b.BackgroundColor,
			ShouldFill:      false, // The content area was filled above.
			Title:           b.Title,
			TitleAlignment:  b.TitleAlignment,
		})
	}
}

// InnerRect returns the drawable content area inside active border sides.
func (b *Box) InnerRect() buffer.Rect {
	sides := style.GetBorderSides(b.Border)
	return buffer.InnerRect(b.X(), b.Y(), b.Width(), b.Height(), sides)
}

// Box options.

// BoxWithBackgroundColor sets the background color.
func BoxWithBackgroundColor(c style.Color) BoxOption {
	return func(b *Box) {
		b.BackgroundColor = c
		b.ShouldFill = true
	}
}

// BoxWithBorderStyle sets the border style.
func BoxWithBorderStyle(s style.BorderStyle) BoxOption {
	return func(b *Box) {
		b.BorderStyle = s
	}
}

// BoxWithBorder sets which border sides to draw.
func BoxWithBorder(brd style.Border) BoxOption {
	return func(b *Box) {
		b.Border = brd
	}
}

// BoxWithBorderColor sets the border color.
func BoxWithBorderColor(c style.Color) BoxOption {
	return func(b *Box) {
		b.BorderColor = c
	}
}

// BoxWithTitle sets the title drawn in the top border.
func BoxWithTitle(t string) BoxOption {
	return func(b *Box) {
		b.Title = t
	}
}

// BoxWithTitleAlignment sets the title alignment.
func BoxWithTitleAlignment(a layout.Alignment) BoxOption {
	return func(b *Box) {
		b.TitleAlignment = a
	}
}

// BoxWithFocusedBorderColor sets the border color when focused.
func BoxWithFocusedBorderColor(c style.Color) BoxOption {
	return func(b *Box) {
		b.FocusedBorderColor = c
	}
}

// BoxWithShouldFill enables or disables background fill.
func BoxWithShouldFill(v bool) BoxOption {
	return func(b *Box) {
		b.ShouldFill = v
	}
}

// BoxWithGap sets both row and column gaps between children.
func BoxWithGap(g layout.Scalar) BoxOption {
	return func(b *Box) {
		b.Gap = g
	}
}

// BoxWithRowGap sets the vertical gap between children.
func BoxWithRowGap(g layout.Scalar) BoxOption {
	return func(b *Box) {
		b.RowGap = g
	}
}

// BoxWithColumnGap sets the horizontal gap between children.
func BoxWithColumnGap(g layout.Scalar) BoxOption {
	return func(b *Box) {
		b.ColumnGap = g
	}
}

// Box layout options.

// BoxWithID sets the widget ID.
func BoxWithID(id string) BoxOption {
	return func(b *Box) { b.SetID(id) }
}

// BoxWithWidth sets the widget width.
func BoxWithWidth(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().Width = s }
}

// BoxWithHeight sets the widget height.
func BoxWithHeight(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().Height = s }
}

// BoxWithMinHeight sets the minimum height.
func BoxWithMinHeight(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().MinHeight = s }
}

// BoxWithFlexGrow sets the flex-grow factor.
func BoxWithFlexGrow(v int) BoxOption {
	return func(b *Box) { b.LayoutNode().FlexGrow = v }
}

// BoxWithFlexShrink sets the flex-shrink factor.
func BoxWithFlexShrink(v int) BoxOption {
	return func(b *Box) { b.LayoutNode().FlexShrink = v }
}

// BoxWithFlexDirection sets the flex direction.
func BoxWithFlexDirection(d layout.FlexDirection) BoxOption {
	return func(b *Box) { b.LayoutNode().FlexDirection = d }
}

// BoxWithJustifyContent sets how children are distributed along the main axis.
func BoxWithJustifyContent(j layout.JustifyContent) BoxOption {
	return func(b *Box) { b.LayoutNode().JustifyContent = j }
}

// BoxWithAlignItems sets how children align along the cross axis.
func BoxWithAlignItems(a layout.Align) BoxOption {
	return func(b *Box) { b.LayoutNode().AlignItems = a }
}

// BoxWithPadding sets padding on all sides.
func BoxWithPadding(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().SetPadding(s) }
}

// BoxWithMargin sets the margin on all sides.
func BoxWithMargin(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().SetMargin(s) }
}

// BoxWithMarginTop sets the top margin.
func BoxWithMarginTop(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().MarginTop = s }
}

// BoxWithMarginRight sets the right margin.
func BoxWithMarginRight(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().MarginRight = s }
}

// BoxWithMarginBottom sets the bottom margin.
func BoxWithMarginBottom(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().MarginBottom = s }
}

// BoxWithMarginLeft sets the left margin.
func BoxWithMarginLeft(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().MarginLeft = s }
}

// BoxWithFlexBasis sets the flex basis.
func BoxWithFlexBasis(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().FlexBasis = s }
}

// BoxWithPaddingTop sets the top padding.
func BoxWithPaddingTop(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().PaddingTop = s }
}

// BoxWithPaddingRight sets the right padding.
func BoxWithPaddingRight(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().PaddingRight = s }
}

// BoxWithPaddingBottom sets the bottom padding.
func BoxWithPaddingBottom(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().PaddingBottom = s }
}

// BoxWithPaddingLeft sets the left padding.
func BoxWithPaddingLeft(s layout.Scalar) BoxOption {
	return func(b *Box) { b.LayoutNode().PaddingLeft = s }
}

// BoxWithOverflow sets the overflow mode.
func BoxWithOverflow(o layout.Overflow) BoxOption {
	return func(b *Box) { b.LayoutNode().Overflow = o }
}

// Children adds non-nil child renderables to a Box during construction.
//
//	widget.NewBox(
//	    widget.BoxWithFlexDirection(layout.DirectionRow),
//	    widget.Children(
//	        widget.NewText("Hello"),
//	        widget.NewGauge(),
//	    ),
//	)
func Children(children ...render.Renderable) BoxOption {
	return func(b *Box) {
		for _, child := range children {
			if child != nil {
				b.Add(child)
			}
		}
	}
}

// SetWidth sets the box width dynamically after construction.
func (b *Box) SetWidth(s layout.Scalar) {
	b.LayoutNode().Width = s
	b.LayoutNode().MarkDirty()
	b.RequestRender()
}

// SetHeight sets the box height dynamically after construction.
func (b *Box) SetHeight(s layout.Scalar) {
	b.LayoutNode().Height = s
	b.LayoutNode().MarkDirty()
	b.RequestRender()
}
