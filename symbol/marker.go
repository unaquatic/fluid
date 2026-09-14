package symbol

// Filled, open, and small bullet runes.
const (
	BulletFilled = '●' // U+25CF
	BulletOpen   = '○' // U+25CB
	BulletSmall  = '•' // U+2022
)

// Directional triangle and arrow runes.
const (
	TriangleUp    = '▲' // U+25B2
	TriangleDown  = '▼' // U+25BC
	TriangleLeft  = '◀' // U+25C0
	TriangleRight = '▶' // U+25B6

	ArrowUp    = '↑' // U+2191
	ArrowDown  = '↓' // U+2193
	ArrowLeft  = '←' // U+2190
	ArrowRight = '→' // U+2192
)

const (
	TreeExpanded  = string(TriangleDown)  // Marks an expanded tree node.
	TreeCollapsed = string(TriangleRight) // Marks a collapsed tree node.
)

// Miscellaneous status and truncation markers.
const (
	Ellipsis  = '…' // U+2026
	CheckMark = '✓' // U+2713
	CrossMark = '✗' // U+2717
)
