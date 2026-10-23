// Package layout is a flexbox layout engine that sizes and places nodes in
// terminal cells.
//
// Each [Node] carries CSS-like sizing, flex, alignment, padding, margin,
// gap, positioning, overflow, and visibility properties. Computing the layout
// of a tree gives every node its integer position and size in
// [ComputedLayout], and a leaf that sizes itself, such as a block of text,
// reports its size through a [MeasureFunc].
package layout
