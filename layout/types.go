package layout

import "fmt"

// ScalarUnit indicates how a scalar value should be interpreted.
type ScalarUnit int

const (
	UnitPoints    ScalarUnit = iota // Absolute cell count
	UnitPercent                     // Percentage of parent dimension
	UnitAuto                        // Auto-sized (content or stretch)
	UnitUndefined                   // Not set
)

// Scalar represents a dimension value with a unit.
type Scalar struct {
	Value int
	Unit  ScalarUnit
}

// --- Scalar Constructors ---

// Points creates a Scalar with absolute cell count.
func Points(v int) Scalar {
	return Scalar{Value: v, Unit: UnitPoints}
}

// Percent creates a Scalar with percentage of parent dimension.
func Percent(v int) Scalar {
	return Scalar{Value: v, Unit: UnitPercent}
}

// Auto creates an auto-sized Scalar.
func Auto() Scalar {
	return Scalar{Value: 0, Unit: UnitAuto}
}

// Undefined creates an undefined Scalar.
func Undefined() Scalar {
	return Scalar{Value: 0, Unit: UnitUndefined}
}

// --- Scalar Methods ---

// IsUndefined returns true if the scalar's unit is UnitUndefined.
func (s Scalar) IsUndefined() bool {
	return s.Unit == UnitUndefined
}

// IsAuto returns true if the scalar's unit is UnitAuto.
func (s Scalar) IsAuto() bool {
	return s.Unit == UnitAuto
}

// String returns a human-readable representation of the scalar (e.g. "103pt", "50%", "auto").
func (s Scalar) String() string {
	switch s.Unit {
	case UnitPoints:
		return fmt.Sprintf("%dpt", s.Value)
	case UnitPercent:
		return fmt.Sprintf("%d%%", s.Value)
	case UnitAuto:
		return "auto"
	case UnitUndefined:
		return "undefined"
	default:
		return fmt.Sprintf("Scalar(%d,%d)", s.Value, s.Unit)
	}
}

// Resolve resolves the scalar to an absolute integer value given the parent dimension.
// Points returns Value directly, Percent returns Value*parentDimension/100.
// Auto and Undefined return (0, false).
func (s Scalar) Resolve(parentDimension int) (int, bool) {
	switch s.Unit {
	case UnitPoints:
		return s.Value, true
	case UnitPercent:
		return s.Value * parentDimension / 100, true
	default:
		return 0, false
	}
}

// --- Visibility ---

// Visibility controls how a node participates in layout and rendering.
type Visibility int

const (
	Visible Visibility = iota // normal: participates in layout and renders
	Hidden                    // invisible: participates in layout but does not render
	Gone                      // removed: does not participate in layout or render
)

// --- FlexDirection ---

// FlexDirection controls the primary axis of layout.
type FlexDirection int

const (
	DirectionRow           FlexDirection = iota // left to right
	DirectionColumn                             // top to bottom
	DirectionRowReverse                         // right to left
	DirectionColumnReverse                      // bottom to top
)

// IsRow returns true for Row or RowReverse.
func (d FlexDirection) IsRow() bool {
	return d == DirectionRow || d == DirectionRowReverse
}

// IsColumn returns true for Column or ColumnReverse.
func (d FlexDirection) IsColumn() bool {
	return d == DirectionColumn || d == DirectionColumnReverse
}

// IsReverse returns true for RowReverse or ColumnReverse.
func (d FlexDirection) IsReverse() bool {
	return d == DirectionRowReverse || d == DirectionColumnReverse
}

// --- JustifyContent ---

// JustifyContent controls distribution of items along the main axis.
type JustifyContent int

const (
	JustifyStart        JustifyContent = iota // pack items at start
	JustifyCenter                             // center items
	JustifyEnd                                // pack items at end
	JustifySpaceBetween                       // even space between items
	JustifySpaceAround                        // even space around items
	JustifySpaceEvenly                        // equal space between and around
)

// --- Align ---

// Align controls alignment on the cross axis (used for AlignItems and AlignSelf).
type Align int

const (
	AlignStart   Align = iota // align to cross-axis start
	AlignCenter               // center on cross-axis
	AlignEnd                  // align to cross-axis end
	AlignStretch              // stretch to fill cross-axis (default for items)
	AlignAuto                 // inherit from parent's AlignItems (for self)
)

// --- FlexWrap ---

// FlexWrap controls whether items wrap to new lines.
type FlexWrap int

const (
	WrapNone    FlexWrap = iota // no wrapping (single line)
	WrapWrap                    // wrap to next line
	WrapReverse                 // wrap in reverse direction
)

// --- Position ---

// Position controls whether a node participates in flex flow.
type Position int

const (
	PositionRelative Position = iota // participates in flex layout
	PositionAbsolute                 // removed from flex flow, positioned via offsets
)

// --- Sticky ---

// Sticky controls whether a node pins to a viewport edge when
// scrolled inside a ScrollBox. The node participates in normal flex layout
// but receives a post-layout offset that keeps it visible.
type Sticky int

const (
	StickyNone   Sticky = iota // normal scrolling
	StickyTop                  // pins to the top edge of the scroll viewport
	StickyBottom               // pins to the bottom edge of the scroll viewport
	StickyLeft                 // pins to the left edge of the scroll viewport
	StickyRight                // pins to the right edge of the scroll viewport
)

// --- Overflow ---

// Overflow controls how content that exceeds bounds is handled.
type Overflow int

const (
	OverflowVisible Overflow = iota // content can extend beyond bounds
	OverflowHidden                  // content is clipped (triggers scissor rect)
	OverflowScroll                  // content is clipped + scrollable
)

// --- ComputedLayout ---

// --- Alignment (text) ---

// Alignment controls horizontal text alignment (e.g. for titles, labels).
// This is distinct from [Align], which controls cross-axis positioning in
// a flexbox container.
type Alignment int

const (
	TextAlignLeft   Alignment = iota // align text to the left (default)
	TextAlignCenter                  // center text horizontally
	TextAlignRight                   // align text to the right
)

// ComputedLayout holds the resolved position and size of a node.
// All values are in integer cells — the natural unit for a TUI.
type ComputedLayout struct {
	X      int // Absolute x position (relative to root)
	Y      int // Absolute y position (relative to root)
	Width  int // Resolved width in cells
	Height int // Resolved height in cells
}

// LayoutCache stores the inputs from the last layoutFlexContainer call.
// When a clean node receives the same available dimensions, its layout
// result is identical — the entire subtree can be skipped.
// This is the same approach Yoga uses (YGCachedMeasurement).
type LayoutCache struct {
	AvailWidth  int // available width passed to layoutFlexContainer
	AvailHeight int // available height passed to layoutFlexContainer
	ParentX     int // absolute X from parent (for shift detection)
	ParentY     int // absolute Y from parent (for shift detection)
	Valid       bool
}
