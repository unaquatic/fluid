package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// CharFill is a widget that fills its area with a single character.
// With Width=1, it acts as a vertical line; with Height=1, a horizontal line.
type CharFill struct {
	*render.BaseRenderable
	char rune
	fg   style.Color
	bg   style.Color
}

// CharFillOption configures a CharFill during construction.
type CharFillOption func(*CharFill)

// NewCharFill creates a CharFill with the given character and colors.
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

// NewVLine creates a 1-column vertical line widget.
func NewVLine(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithWidth(layout.Points(1)),
		CharFillWithFlexShrink(0),
	)
}

// NewHLine creates a 1-row horizontal line widget with flex-grow.
func NewHLine(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithHeight(layout.Points(1)),
		CharFillWithFlexGrow(1),
	)
}

// NewCharCell creates a fixed 1x1 cell widget that won't shrink.
func NewCharCell(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithWidth(layout.Points(1)),
		CharFillWithHeight(layout.Points(1)),
		CharFillWithFlexShrink(0),
	)
}

// RenderSelf fills every cell in the widget's area with the character.
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

// ---------------------------------------------------------------------------
// CharFill Options
// ---------------------------------------------------------------------------

// CharFillWithWidth sets the width.
func CharFillWithWidth(s layout.Scalar) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().Width = s }
}

// CharFillWithHeight sets the height.
func CharFillWithHeight(s layout.Scalar) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().Height = s }
}

// CharFillWithFlexGrow sets flex grow.
func CharFillWithFlexGrow(v int) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().FlexGrow = v }
}

// CharFillWithFlexShrink sets flex shrink.
func CharFillWithFlexShrink(v int) CharFillOption {
	return func(c *CharFill) { c.LayoutNode().FlexShrink = v }
}
