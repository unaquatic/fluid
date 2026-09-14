package symbol

// Thin horizontal and vertical line runes.
const (
	Horizontal = '─' // U+2500
	Vertical   = '│' // U+2502
)

// Heavy horizontal and vertical line runes.
const (
	HeavyHorizontal = '━' // U+2501
	HeavyVertical   = '┃' // U+2503
)

// Double horizontal and vertical line runes.
const (
	DoubleHorizontal = '═' // U+2550
	DoubleVertical   = '║' // U+2551
)

// Thin dashed line runes grouped by dash count.
const (
	LightDoubleDashHorizontal    = '╌' // U+254C
	LightDoubleDashVertical      = '╎' // U+254E
	LightTripleDashHorizontal    = '┄' // U+2504
	LightTripleDashVertical      = '┆' // U+2506
	LightQuadrupleDashHorizontal = '┈' // U+2508
	LightQuadrupleDashVertical   = '┊' // U+250A
)

// Heavy dashed line runes grouped by dash count.
const (
	HeavyDoubleDashHorizontal    = '╍' // U+254D
	HeavyDoubleDashVertical      = '╏' // U+254F
	HeavyTripleDashHorizontal    = '┅' // U+2505
	HeavyTripleDashVertical      = '┇' // U+2507
	HeavyQuadrupleDashHorizontal = '┉' // U+2509
	HeavyQuadrupleDashVertical   = '┋' // U+250B
)
