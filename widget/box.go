package widget

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// Box lays out child renderables with optional borders and background fill.
// BoxWithLayout sets its layout, such as the gaps between its children.
type Box struct {
	*render.BaseRenderable

	backgroundColor    style.Color
	borderStyle        style.BorderStyle
	border             style.Border
	borderColor        style.Color
	shouldFill         bool
	title              string
	titleAlignment     layout.Alignment
	focusedBorderColor style.Color
}

// BoxOption configures a Box during construction.
type BoxOption func(*Box)

// NewBox returns a column-oriented Box with optional configuration.
func NewBox(opts ...BoxOption) *Box {
	b := &Box{
		BaseRenderable:     render.NewBaseRenderable(),
		backgroundColor:    style.ColorTransparent,
		borderStyle:        style.BorderNone,
		border:             style.BorderNoBorder(),
		borderColor:        style.ColorWhite,
		focusedBorderColor: style.ColorTransparent,
		titleAlignment:     layout.TextAlignLeft,
	}

	// Containers stack children vertically unless an option changes the direction.
	b.LayoutNode().FlexDirection = layout.DirectionColumn

	for _, opt := range opts {
		opt(b)
	}

	// Keep children inside every active border side.
	b.applyBorderPadding()

	return b
}

// applyBorderPadding reserves at least one cell inside each active border side.
func (b *Box) applyBorderPadding() {
	sides := b.border
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

// RenderSelf draws the box background, border, and title.
func (b *Box) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	w := b.Width()
	h := b.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := b.X()
	y := b.Y()

	// A partial border is active when any side is enabled.
	sides := b.border
	hasBorder := sides.Top || sides.Right || sides.Bottom || sides.Left

	// Fill only the content area when a border is present.
	if b.shouldFill && !b.backgroundColor.IsTransparent() {
		if hasBorder {
			inner := buffer.InnerRect(x, y, w, h, sides)
			if inner.Width > 0 && inner.Height > 0 {
				buf.FillRect(inner.X, inner.Y, inner.Width, inner.Height, b.backgroundColor)
			}
		} else {
			buf.FillRect(x, y, w, h, b.backgroundColor)
		}
	}

	// Focus color overrides the normal border color when configured.
	if hasBorder {
		borderColor := b.borderColor
		if b.Focused() && !b.focusedBorderColor.IsTransparent() {
			borderColor = b.focusedBorderColor
		}

		buf.DrawBorder(buffer.BorderDrawOptions{
			X:               x,
			Y:               y,
			Width:           w,
			Height:          h,
			Style:           b.borderStyle,
			Border:          b.border,
			Color:           borderColor,
			BackgroundColor: b.backgroundColor,
			ShouldFill:      false, // The content area was filled above.
			Title:           b.title,
			TitleAlignment:  b.titleAlignment,
		})
	}
}

// InnerRect returns the drawable content area inside active border sides.
func (b *Box) InnerRect() buffer.Rect {
	sides := b.border
	return buffer.InnerRect(b.X(), b.Y(), b.Width(), b.Height(), sides)
}

// Bg returns the color that fills the box, inside any border, while
// ShouldFill reports true.
func (b *Box) Bg() style.Color { return b.backgroundColor }

// ShouldFill reports whether the box fills itself with Bg.
func (b *Box) ShouldFill() bool { return b.shouldFill }

// BorderStyle returns the border's line style, such as style.BorderSingle.
func (b *Box) BorderStyle() style.BorderStyle { return b.borderStyle }

// Border returns the sides the border draws.
func (b *Box) Border() style.Border { return b.border }

// BorderColor returns the color of the border.
func (b *Box) BorderColor() style.Color { return b.borderColor }

// FocusedBorderColor returns the border color used while the box has focus,
// or a transparent color when the box keeps BorderColor.
func (b *Box) FocusedBorderColor() style.Color { return b.focusedBorderColor }

// Title returns the title drawn in the top border.
func (b *Box) Title() string { return b.title }

// TitleAlignment returns where Title sits along the top border.
func (b *Box) TitleAlignment() layout.Alignment { return b.titleAlignment }

// SetBg fills the box with c, as BoxWithBg does, and requests a frame.
func (b *Box) SetBg(c style.Color) {
	b.backgroundColor = c
	b.shouldFill = true
	b.RequestRender()
}

// SetShouldFill turns the background fill on or off and requests a frame.
func (b *Box) SetShouldFill(v bool) {
	b.shouldFill = v
	b.RequestRender()
}

// SetBorderStyle sets the border's line style and requests a frame.
func (b *Box) SetBorderStyle(s style.BorderStyle) {
	b.borderStyle = s
	b.RequestRender()
}

// SetBorder sets which border sides to draw, reserves a cell of padding inside
// each active side, as NewBox does, and lays the box out again. The padding
// stays when a side is turned off.
func (b *Box) SetBorder(brd style.Border) {
	b.border = brd
	b.applyBorderPadding()
	b.LayoutNode().MarkDirty()
	b.RequestRender()
}

// SetBorderColor sets the border color and requests a frame.
func (b *Box) SetBorderColor(c style.Color) {
	b.borderColor = c
	b.RequestRender()
}

// SetFocusedBorderColor sets the border color used while the box has focus
// and requests a frame.
func (b *Box) SetFocusedBorderColor(c style.Color) {
	b.focusedBorderColor = c
	b.RequestRender()
}

// SetTitle sets the title drawn in the top border and requests a frame.
func (b *Box) SetTitle(t string) {
	b.title = t
	b.RequestRender()
}

// SetTitleAlignment sets where the title sits along the top border and
// requests a frame.
func (b *Box) SetTitleAlignment(a layout.Alignment) {
	b.titleAlignment = a
	b.RequestRender()
}

// Box options.

// BoxWithBg sets the background color.
func BoxWithBg(c style.Color) BoxOption {
	return func(b *Box) {
		b.backgroundColor = c
		b.shouldFill = true
	}
}

// BoxWithBorderStyle sets the border style.
func BoxWithBorderStyle(s style.BorderStyle) BoxOption {
	return func(b *Box) {
		b.borderStyle = s
	}
}

// BoxWithBorder sets which border sides to draw.
func BoxWithBorder(brd style.Border) BoxOption {
	return func(b *Box) {
		b.border = brd
	}
}

// BoxWithBorderColor sets the border color.
func BoxWithBorderColor(c style.Color) BoxOption {
	return func(b *Box) {
		b.borderColor = c
	}
}

// BoxWithTitle sets the title drawn in the top border.
func BoxWithTitle(t string) BoxOption {
	return func(b *Box) {
		b.title = t
	}
}

// BoxWithTitleAlignment sets the title alignment.
func BoxWithTitleAlignment(a layout.Alignment) BoxOption {
	return func(b *Box) {
		b.titleAlignment = a
	}
}

// BoxWithFocusedBorderColor sets the border color when focused.
func BoxWithFocusedBorderColor(c style.Color) BoxOption {
	return func(b *Box) {
		b.focusedBorderColor = c
	}
}

// BoxWithShouldFill enables or disables background fill.
func BoxWithShouldFill(v bool) BoxOption {
	return func(b *Box) {
		b.shouldFill = v
	}
}

// Box layout options.

// BoxWithID sets the widget ID.
func BoxWithID(id string) BoxOption {
	return func(b *Box) { b.SetID(id) }
}

// BoxWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func BoxWithLayout(opts ...layout.Option) BoxOption {
	return func(w *Box) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}

// Children adds non-nil child renderables to a Box during construction.
//
//	widget.NewBox(
//	    widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionRow)),
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
