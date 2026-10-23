package layout

import "fmt"

// ScalarUnit identifies how a layout scalar is resolved.
type ScalarUnit int

const (
	UnitPoints    ScalarUnit = iota // An absolute number of cells.
	UnitPercent                     // A percentage of the parent dimension.
	UnitAuto                        // A size determined from content or stretching.
	UnitUndefined                   // No value is specified.
)

// Scalar is an integer layout value paired with its unit.
type Scalar struct {
	// Value is the number of cells, or the percentage with UnitPercent.
	Value int
	// Unit says how Value is read.
	Unit ScalarUnit
}

// Points returns an absolute size of v cells.
func Points(v int) Scalar {
	return Scalar{Value: v, Unit: UnitPoints}
}

// Percent returns v percent of a parent dimension.
func Percent(v int) Scalar {
	return Scalar{Value: v, Unit: UnitPercent}
}

// Auto returns a scalar whose size is determined during layout.
func Auto() Scalar {
	return Scalar{Value: 0, Unit: UnitAuto}
}

// Undefined returns a scalar with no specified value.
func Undefined() Scalar {
	return Scalar{Value: 0, Unit: UnitUndefined}
}

// IsUndefined reports whether s has no specified value.
func (s Scalar) IsUndefined() bool {
	return s.Unit == UnitUndefined
}

// IsAuto reports whether s is sized during layout.
func (s Scalar) IsAuto() bool {
	return s.Unit == UnitAuto
}

// String returns a readable representation such as "103pt", "50%", or
// "auto".
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

// Resolve converts s to cells relative to parentDimension. It reports false
// for automatic and undefined values.
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

// Visibility controls a node's participation in layout and rendering.
type Visibility int

const (
	Visible Visibility = iota // The node participates in layout and rendering.
	Hidden                    // The node participates in layout but is not rendered.
	Gone                      // The node participates in neither layout nor rendering.
)

// FlexDirection identifies the main axis and item order of a flex container.
type FlexDirection int

const (
	DirectionRow           FlexDirection = iota // Lay out items from left to right.
	DirectionColumn                             // Lay out items from top to bottom.
	DirectionRowReverse                         // Lay out items from right to left.
	DirectionColumnReverse                      // Lay out items from bottom to top.
)

// IsRow reports whether d uses the horizontal axis.
func (d FlexDirection) IsRow() bool {
	return d == DirectionRow || d == DirectionRowReverse
}

// IsColumn reports whether d uses the vertical axis.
func (d FlexDirection) IsColumn() bool {
	return d == DirectionColumn || d == DirectionColumnReverse
}

// IsReverse reports whether d reverses item order.
func (d FlexDirection) IsReverse() bool {
	return d == DirectionRowReverse || d == DirectionColumnReverse
}

// JustifyContent controls item distribution along the main axis.
type JustifyContent int

const (
	JustifyStart        JustifyContent = iota // Pack items at the start.
	JustifyCenter                             // Center items.
	JustifyEnd                                // Pack items at the end.
	JustifySpaceBetween                       // Distribute space between items.
	JustifySpaceAround                        // Give each item equal surrounding space.
	JustifySpaceEvenly                        // Make all outer and inner spaces equal.
)

// Align controls item alignment along the cross axis.
type Align int

const (
	AlignStart   Align = iota // Align at the cross-axis start.
	AlignCenter               // Center on the cross axis.
	AlignEnd                  // Align at the cross-axis end.
	AlignStretch              // Stretch to fill the cross axis.
	AlignAuto                 // Use the parent's AlignItems value for AlignSelf.
)

// FlexWrap controls whether and in which direction items form additional
// lines.
type FlexWrap int

const (
	WrapNone    FlexWrap = iota // Keep all items on one line.
	WrapWrap                    // Add lines in the cross-axis direction.
	WrapReverse                 // Add lines in the reverse cross-axis direction.
)

// Position controls whether a node participates in flex flow.
type Position int

const (
	PositionRelative Position = iota // Participate in flex layout.
	PositionAbsolute                 // Use offsets outside flex flow.
)

// Sticky controls whether a node remains pinned to a viewport edge while its
// container scrolls. Sticky nodes still participate in normal flex layout.
type Sticky int

const (
	StickyNone   Sticky = iota // Scroll normally.
	StickyTop                  // Pin to the viewport's top edge.
	StickyBottom               // Pin to the viewport's bottom edge.
	StickyLeft                 // Pin to the viewport's left edge.
	StickyRight                // Pin to the viewport's right edge.
)

// Overflow controls content that extends beyond a node's bounds.
type Overflow int

const (
	OverflowVisible Overflow = iota // Allow content outside the bounds.
	OverflowHidden                  // Clip content to the bounds.
	OverflowScroll                  // Clip content and allow scrolling.
)

// Alignment controls horizontal text alignment. It is distinct from [Align],
// which positions flex items along a container's cross axis.
type Alignment int

const (
	TextAlignLeft   Alignment = iota // Align text to the left.
	TextAlignCenter                  // Center text horizontally.
	TextAlignRight                   // Align text to the right.
)

// ComputedLayout contains a node's resolved absolute position and size in
// terminal cells.
type ComputedLayout struct {
	// X is the absolute x-coordinate relative to the root.
	X int
	// Y is the absolute y-coordinate relative to the root.
	Y int
	// Width is the resolved width in cells.
	Width int
	// Height is the resolved height in cells.
	Height int
}

// Cache contains the inputs from a node's most recent layout. A clean
// node can reuse its subtree layout when the available dimensions match.
type Cache struct {
	// AvailWidth is the available width used for the cached layout.
	AvailWidth int
	// AvailHeight is the available height used for the cached layout.
	AvailHeight int
	// ParentX is the cached absolute x-coordinate of the node.
	ParentX int
	// ParentY is the cached absolute y-coordinate of the node.
	ParentY int
	// Valid reports whether the other fields hold a layout to compare against.
	Valid bool
}
