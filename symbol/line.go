package symbol

// Standalone line-drawing characters — individual runes for horizontal
// and vertical lines, used outside of full border sets (e.g. separators,
// horizontal rules, blockquote bars).

// ---------------------------------------------------------------------------
// Single-weight lines
// ---------------------------------------------------------------------------

const (
	Horizontal = '─' // U+2500
	Vertical   = '│' // U+2502
)

// ---------------------------------------------------------------------------
// Heavy-weight lines
// ---------------------------------------------------------------------------

const (
	HeavyHorizontal = '━' // U+2501
	HeavyVertical   = '┃' // U+2503
)

// ---------------------------------------------------------------------------
// Double lines
// ---------------------------------------------------------------------------

const (
	DoubleHorizontal = '═' // U+2550
	DoubleVertical   = '║' // U+2551
)

// ---------------------------------------------------------------------------
// Dashed lines (light)
// ---------------------------------------------------------------------------

const (
	LightDoubleDashHorizontal    = '╌' // U+254C
	LightDoubleDashVertical      = '╎' // U+254E
	LightTripleDashHorizontal    = '┄' // U+2504
	LightTripleDashVertical      = '┆' // U+2506
	LightQuadrupleDashHorizontal = '┈' // U+2508
	LightQuadrupleDashVertical   = '┊' // U+250A
)

// ---------------------------------------------------------------------------
// Dashed lines (heavy)
// ---------------------------------------------------------------------------

const (
	HeavyDoubleDashHorizontal    = '╍' // U+254D
	HeavyDoubleDashVertical      = '╏' // U+254F
	HeavyTripleDashHorizontal    = '┅' // U+2505
	HeavyTripleDashVertical      = '┇' // U+2507
	HeavyQuadrupleDashHorizontal = '┉' // U+2509
	HeavyQuadrupleDashVertical   = '┋' // U+250B
)
