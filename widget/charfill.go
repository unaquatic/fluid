package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// CharFill fills its layout area with one character and style.
// A one-column CharFill forms a vertical line; a one-row CharFill forms a horizontal line.
type CharFill struct {
	*render.BaseRenderable
	char rune
	fg   style.Color
	bg   style.Color
}

// CharFillOption configures a CharFill during construction.
type CharFillOption func(*CharFill)

// NewCharFill returns a CharFill that uses char, fg, and bg for every cell.
func NewCharFill(char rune, fg, bg style.Color, opts ...CharFillOption) *CharFill {
	c := &CharFill{
		BaseRenderable: render.NewBaseRenderable(),
		char:           char,
		fg:             fg,
		bg:             bg,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewVLine returns a one-column vertical line that does not shrink.
func NewVLine(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithWidth(layout.Points(1)),
		CharFillWithFlexShrink(0),
	)
}

// NewHLine returns a one-row horizontal line that grows along its flex axis.
func NewHLine(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithHeight(layout.Points(1)),
		CharFillWithFlexGrow(1),
	)
}

// NewCharCell returns a fixed one-cell widget that does not shrink.
func NewCharCell(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithWidth(layout.Points(1)),
		CharFillWithHeight(layout.Points(1)),
		CharFillWithFlexShrink(0),
	)
}

// RenderSelf fills the widget's layout area with its configured character and colors.
func (c *CharFill) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	x := c.X()
	y := c.Y()
	w := c.Width()
	h := c.Height()
	for row := range h {
		for col := range w {
			buf.SetCell(x+col, y+row, c.char, c.fg, c.bg, 0)
		}
	}
}

// CharFill options.

// CharFillWithWidth sets the width.
func CharFillWithWidth(s layout.Scalar) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().Width = s }
}

// CharFillWithHeight sets the height.
func CharFillWithHeight(s layout.Scalar) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().Height = s }
}

// CharFillWithFlexGrow sets the flex-grow factor.
func CharFillWithFlexGrow(v int) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().FlexGrow = v }
}

// CharFillWithFlexShrink sets the flex-shrink factor.
func CharFillWithFlexShrink(v int) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().FlexShrink = v }
}
