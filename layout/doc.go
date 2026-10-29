// Package layout is a flexbox layout engine that sizes and places nodes in
// terminal cells.
//
// Each [Node] carries CSS-like sizing, flex, alignment, padding, margin,
// gap, positioning, overflow, and visibility properties. Computing the layout
// of a tree gives every node its integer position and size in
// [ComputedLayout], and a leaf that sizes itself, such as a block of text,
// reports its size through a [MeasureFunc].
//
// Lengths are [Scalar] values: [Points] counts cells, [Percent] takes a share
// of the parent, and [Auto] leaves the size to layout. A Node lays its
// children out in a row, as CSS does, but the renderer's root and the Box
// widget lay theirs out in a column.
//
// Every widget returns its node from LayoutNode and takes the [Option] values
// of this package through an option of its own, such as widget.BoxWithLayout,
// whose example lays out a row of boxes.
package layout
