package widget

import (
	"time"

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
		CharFillWithLayout(layout.WithWidth(layout.Points(1)), layout.WithFlexShrink(0)),
	)
}

// NewHLine returns a one-row horizontal line that grows along its flex axis.
func NewHLine(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithLayout(layout.WithHeight(layout.Points(1)), layout.WithFlexGrow(1)),
	)
}

// NewCharCell returns a fixed one-cell widget that does not shrink.
func NewCharCell(char rune, fg, bg style.Color) *CharFill {
	return NewCharFill(char, fg, bg,
		CharFillWithLayout(
			layout.WithWidth(layout.Points(1)),
			layout.WithHeight(layout.Points(1)),
			layout.WithFlexShrink(0),
		),
	)
}

// RenderSelf fills the widget's layout area with its configured character and colors.
func (c *CharFill) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
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

// CharFillWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func CharFillWithLayout(opts ...layout.Option) CharFillOption {
	return func(w *CharFill) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}
