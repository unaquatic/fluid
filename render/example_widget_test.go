package render_test

import (
	"fmt"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/textbuf"
	"github.com/unaquatic/fluid/widget"
)

// Counter shows a count, which the space bar raises while it has focus.
type Counter struct {
	*render.BaseRenderable
	Count int
}

// NewCounter returns a focusable Counter as wide as its label.
func NewCounter() *Counter {
	c := &Counter{BaseRenderable: render.NewBaseRenderable(render.WithFocusable(true))}
	c.LayoutNode().Measure = func(int, layout.MeasureMode, int, layout.MeasureMode) layout.MeasureResult {
		return layout.MeasureResult{Width: textbuf.StringWidth(c.Label()), Height: 1}
	}
	return c
}

// Label returns the text c shows.
func (c *Counter) Label() string { return fmt.Sprintf("count: %d", c.Count) }

// RenderSelf draws c's label at its position.
func (c *Counter) RenderSelf(buf *buffer.Buffer, _ time.Duration) {
	buf.DrawText(c.Label(), c.X(), c.Y(), style.ColorWhite)
}

// HandleKeyPress raises c's count on the space bar.
func (c *Counter) HandleKeyPress(e *input.KeyEvent) bool {
	if !e.Matches(input.KeySpace, 0) {
		return false
	}
	c.Count++
	c.LayoutNode().MarkDirty() // The label can need another column.
	c.RequestRender()
	return true
}

// This example writes a widget, Counter, and draws it in a row beside a Text
// widget, which starts where Counter's measured width ends.
func Example_widget() {
	c := NewCounter()
	root := terminal.NewRootRenderable(30, 1)
	root.Add(widget.NewBox(
		widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionRow), layout.WithGap(1)),
		widget.Children(c, widget.NewText("(space adds one)")),
	))

	// A renderer sends keys to the focused widget; send two directly.
	for range 2 {
		c.HandleKeyPress(&input.KeyEvent{Name: input.KeySpace})
	}

	buf := buffer.New(30, 1)
	root.Render(buf, 0)
	fmt.Println(testutil.RowText(buf, 0))
	// Output: count: 2 (space adds one)
}
