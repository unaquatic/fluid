// Package testutil provides helpers for laying out, rendering, inspecting,
// and comparing fluid widgets in tests.
//
// [RenderWidget] lays out a widget at an exact size and draws it, without its
// children, into a new buffer, which needs no terminal. [AssertGoldenFrames]
// compares a buffer with the golden file testdata/<name>.frames.golden, in
// the form [GoldenFrames] returns: the text of the cells in one frame, their
// foreground colors, background colors, and attributes in one frame each,
// omitting a frame that no cell sets, and a legend of the colors.
// [AssertGoldenSequence] compares a [GoldenSequence] of such snapshots, one
// for each state of a widget, with testdata/<name>.seq.golden.
//
// The -update test flag creates or replaces golden files, and
// "fluid design html" draws them in color for review.
package testutil
