// Package symbol provides Unicode character constants and symbol sets
// used throughout fluid for drawing borders, bars, blocks, scrollbars,
// shades, tree guides, and other visual elements.
//
// This centralises the scattered rune literals that were previously
// hardcoded in individual widget and buffer files.
//
// Sub-groups are organised by file:
//
//   - border.go    — Box-drawing border character sets (single, double, rounded, heavy)
//   - bar.go       — Vertical/horizontal bar fill characters (eighths)
//   - block.go     — Block elements (quadrants, half-blocks, full block)
//   - scrollbar.go — Scrollbar track/thumb character sets
//   - shade.go     — Shade characters and gauge fill/empty
//   - line.go      — Standalone line-drawing characters and the DOT constant
//   - marker.go    — Indicator/marker runes (bullets, arrows, tree guides)
package symbol

// DOT is a middle dot, useful as a general-purpose small marker.
const DOT = '·'
