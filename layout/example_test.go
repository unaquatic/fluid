package layout_test

import (
	"fmt"

	"github.com/unaquatic/fluid/layout"
)

// This example lays out a row of a fixed-width sidebar and a content area
// that grows to fill the rest of an 80-column terminal.
func Example() {
	row := layout.NewNode()
	row.FlexDirection = layout.DirectionRow
	row.Gap = 1

	sidebar := layout.NewNode()
	sidebar.Width = layout.Points(24)
	row.AddChild(sidebar)

	content := layout.NewNode()
	content.FlexGrow = 1
	row.AddChild(content)

	row.ComputeLayout(80, 24)
	for _, n := range row.Children {
		fmt.Printf("%+v\n", n.Computed)
	}
	// Output:
	// {X:0 Y:0 Width:24 Height:24}
	// {X:25 Y:0 Width:55 Height:24}
}

// This example sizes a leaf node from its content, as text widgets do.
func ExampleMeasureFunc() {
	const label = "hello, fluid"

	leaf := layout.NewNode()
	leaf.Measure = func(width int, widthMode layout.MeasureMode, height int, heightMode layout.MeasureMode) layout.MeasureResult {
		return layout.MeasureResult{Width: len(label), Height: 1}
	}

	root := layout.NewNode()
	root.FlexDirection = layout.DirectionRow
	root.AlignItems = layout.AlignStart // Keep the leaf from stretching to the row's height.
	root.AddChild(leaf)
	root.ComputeLayout(80, 24)
	fmt.Printf("%dx%d\n", leaf.Computed.Width, leaf.Computed.Height)
	// Output: 12x1
}
