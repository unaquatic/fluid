package widget_test

import (
	"fmt"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// This example places two bordered boxes in a row: a sidebar 10 columns wide
// and a content box that grows to fill the rest.
func ExampleBoxWithLayout() {
	sidebar := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithLayout(layout.WithWidth(layout.Points(10))),
		widget.Children(widget.NewText("files")),
	)
	content := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderRounded),
		widget.BoxWithTitle("main.go"),
		widget.BoxWithLayout(layout.WithFlexGrow(1)),
		widget.Children(widget.NewText("package main")),
	)
	row := widget.NewBox(
		widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionRow), layout.WithGap(1)),
		widget.Children(sidebar, content),
	)

	root := terminal.NewRootRenderable(30, 3)
	root.Add(row)
	buf := buffer.New(30, 3)
	root.Render(buf, 0)
	fmt.Print(testutil.GoldenText(buf))
	// Output:
	// |┌────────┐ ╭main.go──────────╮|
	// |│files   │ │package main     │|
	// |└────────┘ ╰─────────────────╯|
}
