package textbuf

import (
	"github.com/unaquatic/fluid/style"
)

// WidthMethod selects code-point or grapheme-cluster terminal width calculation.
//
// WcWidth sums each code point's East Asian width. Unicode segments text using
// UAX #29 before calculating each cluster's width.
type WidthMethod int

const (
	// WcWidth sums per-code-point widths.
	WcWidth WidthMethod = iota

	// Unicode calculates widths for extended grapheme clusters.
	Unicode
)

// String returns w's name.
func (w WidthMethod) String() string {
	switch w {
	case WcWidth:
		return "wcwidth"
	case Unicode:
		return "unicode"
	default:
		return "unknown"
	}
}

// ParseWidthMethod returns WcWidth for "wcwidth" and Unicode for every other
// value.
func ParseWidthMethod(s string) WidthMethod {
	switch s {
	case "wcwidth":
		return WcWidth
	default:
		return Unicode
	}
}

// DefaultWidthMethod returns Unicode.
func DefaultWidthMethod() WidthMethod {
	return Unicode
}

// StringWidthMethod returns s's terminal width using m.
func StringWidthMethod(s string, m WidthMethod) int {
	if m == WcWidth {
		w := 0
		for _, r := range s {
			w += RuneWidth(r)
		}
		return w
	}
	return StringWidth(s)
}

// ClusterWidth returns at least one column for a grapheme cluster. This lets
// cursor movement advance past isolated zero-width clusters.
func ClusterWidth(cw int) int {
	if cw == 0 {
		return 1
	}
	return cw
}

// WrapMode controls how long lines are wrapped in a TextSurface.
type WrapMode int

const (
	// WrapNone disables wrapping. Lines extend beyond the viewport.
	WrapNone WrapMode = iota
	// WrapChar wraps at the nearest column boundary.
	WrapChar
	// WrapWord wraps at word boundaries when possible.
	WrapWord
)

// Viewport defines a rectangular window in virtual-line coordinates.
type Viewport struct {
	X, Y          int
	Width, Height int
}

// TextSelection describes a highlighted half-open range in global display-width
// coordinates.
type TextSelection struct {
	Start, End int // Inclusive start and exclusive end offsets.
	BgColor    *style.Color
	FgColor    *style.Color
}

// Span is a run of text with uniform styling.
type Span struct {
	Text       string
	Fg         style.Color
	Bg         style.Color
	Attributes uint32
	Width      int // Cached display width; zero can mean not yet measured.
}

// VirtualChunk is an equally styled fragment of a wrapped visual line.
type VirtualChunk struct {
	Text          string      // Resolved text content.
	GraphemeStart int         // Display-column offset within the virtual line.
	Width         int         // Display width.
	Fg            style.Color // Foreground color.
	Bg            style.Color // Background color.
	Attributes    uint32      // Text attributes.
}

// VirtualLine is one visual row produced by wrapping a logical source line.
type VirtualLine struct {
	Chunks          []VirtualChunk
	Width           int // Total display width.
	CharOffset      int // Global display-width offset at the line start.
	SourceLine      int // Logical source-line index.
	SourceColOffset int // Display-column offset within the source line.
}

// LineInfo contains parallel layout metadata for virtual lines.
type LineInfo struct {
	Starts   []int // Global display-width offset of each virtual line.
	Widths   []int // Display width of each virtual line.
	Sources  []int // Logical source-line index of each virtual line.
	MaxWidth int   // Maximum virtual-line width.
}

// Surface exposes viewport-adjusted text and styling to a buffer renderer.
type Surface interface {
	// VirtualLines returns virtual lines within the vertical viewport.
	VirtualLines() []VirtualLine
	// Viewport returns the current viewport, or nil if none.
	Viewport() *Viewport
	// CachedLineInfo returns metadata for the visible virtual lines.
	CachedLineInfo() LineInfo
	// Selection returns the current text selection, or nil when absent.
	Selection() *TextSelection
	// DefaultFg returns the default foreground color.
	DefaultFg() style.Color
	// DefaultBg returns the default background color.
	DefaultBg() style.Color
	// LineHighlights returns styling ranges for a logical source line.
	LineHighlights(lineIdx int) []Highlight
	// LineBackground returns the background for a logical source line.
	LineBackground(lineIdx int) style.Color
	// Truncate reports whether rendering replaces overflow with an ellipsis.
	Truncate() bool
}
