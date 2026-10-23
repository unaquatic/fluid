package symbol

// Markers and indicators — bullets, arrows, tree-view guides, and other
// small decorative characters used by List, Tree, Select, and Markdown widgets.

// ---------------------------------------------------------------------------
// Bullets
// ---------------------------------------------------------------------------

const (
	BulletFilled = '●' // U+25CF
	BulletOpen   = '○' // U+25CB
	BulletSmall  = '•' // U+2022
)

// ---------------------------------------------------------------------------
// Arrows / triangles
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Tree-view guides
// ---------------------------------------------------------------------------

const (
	TreeExpanded  = string(TriangleDown)  // shown when a tree node is open
	TreeCollapsed = string(TriangleRight) // shown when a tree node is closed
)

// ---------------------------------------------------------------------------
// Misc
// ---------------------------------------------------------------------------

const (
	Ellipsis  = '…' // U+2026
	CheckMark = '✓' // U+2713
	CrossMark = '✗' // U+2717
)
