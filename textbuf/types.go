package textbuf

import (
	"github.com/unaquatic/fluid/style"
)

// WidthMethod controls how character/grapheme widths are calculated.
//
//   - WcWidth: uses per-codepoint East Asian Width (EAW) tables, matching
//     the POSIX wcwidth() function. Compatible with tmux and older terminals.
//
//   - Unicode: grapheme-cluster-aware width via UAX #29.
//     Handles ZWJ sequences, skin tone modifiers, flag pairs, etc.
//     Matches modern terminals (iTerm2, WezTerm, Kitty, Windows Terminal).
type WidthMethod int

const (
	// WcWidth uses per-codepoint East Asian Width lookup.
	WcWidth WidthMethod = iota

	// Unicode uses grapheme-cluster-aware width calculation.
	Unicode
)

// String returns the string representation.
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

// ParseWidthMethod parses a string into a WidthMethod.
// Returns Unicode for unrecognised values.
func ParseWidthMethod(s string) WidthMethod {
	switch s {
	case "wcwidth":
		return WcWidth
	default:
		return Unicode
	}
}

// DefaultWidthMethod returns the default WidthMethod.
func DefaultWidthMethod() WidthMethod {
	return Unicode
}

// StringWidthMethod returns the display width of s using the specified method.
// WcWidth sums per-codepoint widths; Unicode uses grapheme-cluster-aware width.
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

// ClusterWidth returns cw unchanged unless it is zero, in which case it
// returns 1.  Zero-width grapheme clusters (lone ZWJ, bare combining marks)
// still need to occupy one cursor column so the cursor can advance past them.
func ClusterWidth(cw int) int {
	if cw == 0 {
		return 1
	}
	return cw
}

// ---------------------------------------------------------------------------
// WrapMode
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Viewport
// ---------------------------------------------------------------------------

// Viewport defines a rectangular window into the virtual line space.
type Viewport struct {
	X, Y          int
	Width, Height int
}

// ---------------------------------------------------------------------------
// TextSelection
// ---------------------------------------------------------------------------

// TextSelection describes a highlighted character range.
type TextSelection struct {
	Start, End int // character offset range (display-width based)
	BgColor    *style.Color
	FgColor    *style.Color
}

// ---------------------------------------------------------------------------
// Span
// ---------------------------------------------------------------------------

// Span is a run of text with uniform styling.
type Span struct {
	Text       string
	Fg         style.Color
	Bg         style.Color
	Attributes uint32
	Width      int // display width (0 = not yet measured)
}

// ---------------------------------------------------------------------------
// VirtualChunk
// ---------------------------------------------------------------------------

// VirtualChunk is a fragment of a VirtualLine that maps back to source text.
type VirtualChunk struct {
	Text          string      // resolved text content
	GraphemeStart int         // column offset within the source span
	Width         int         // display width of this chunk
	Fg            style.Color // foreground color
	Bg            style.Color // background color
	Attributes    uint32      // text attributes
}

// ---------------------------------------------------------------------------
// VirtualLine
// ---------------------------------------------------------------------------

// VirtualLine is one visual row after wrapping / viewport slicing.
type VirtualLine struct {
	Chunks          []VirtualChunk
	Width           int // total display width
	CharOffset      int // global char offset where this vline starts
	SourceLine      int // logical line index
	SourceColOffset int // column offset within the source line (>0 for wrapped continuations)
}

// ---------------------------------------------------------------------------
// LineInfo
// ---------------------------------------------------------------------------

// LineInfo holds cached layout metadata for virtual lines.
type LineInfo struct {
	Starts   []int // char offset where each vline starts
	Widths   []int // display width of each vline
	Sources  []int // source logical line index for each vline
	MaxWidth int   // maximum width across all lines
}

// ---------------------------------------------------------------------------
// Surface interface
// ---------------------------------------------------------------------------

// Surface is the interface that Buffer.DrawTextSurface accepts.
// Both TextSurface and EditSurface implement this interface.
type Surface interface {
	// VirtualLines returns the visible virtual lines (viewport-sliced).
	VirtualLines() []VirtualLine
	// Viewport returns the current viewport, or nil if none.
	Viewport() *Viewport
	// CachedLineInfo returns layout metadata for the visible virtual lines.
	CachedLineInfo() LineInfo
	// Selection returns the current text selection, or nil.
	Selection() *TextSelection
	// DefaultFg returns the default foreground color.
	DefaultFg() style.Color
	// DefaultBg returns the default background color.
	DefaultBg() style.Color
	// LineHighlights returns the highlights for a source (physical) line.
	// Used by the renderer to apply highlight colors at render time.
	LineHighlights(lineIdx int) []Highlight
	// LineBackground returns the per-line background color for a source
	// (physical) line. The renderer calls FillRect before painting text when
	// the returned color is non-transparent. Used for active-line highlighting
	// and diff hunk coloring.
	LineBackground(lineIdx int) style.Color
	// Truncate returns whether lines wider than the viewport should be
	// truncated with an ellipsis in the last column.
	Truncate() bool
}
