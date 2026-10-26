package buffer

import (
	"unique"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// Cell stores one terminal cell or one part of a multi-column grapheme cluster.
//
// The fields are ordered and sized to keep a cell at 32 bytes. A buffer holds
// one cell per terminal column per row, and inline rendering sizes a buffer to
// the whole transcript, so the width of this struct sets the memory cost of the
// renderer.
type Cell struct {
	// Char is the first code point; zero represents an empty cell.
	Char rune
	// Fg is the foreground color.
	Fg style.Color
	// Bg is the background color.
	Bg style.Color
	// Attributes holds style flags in its low 8 bits and a link ID in the
	// upper bits.
	Attributes style.Attrs
	// Width is a head's column width, or a continuation's negative offset to
	// its head.
	Width int16

	// combining holds the interned code points that follow Char in the same
	// grapheme cluster. Handles are comparable and globally unique, so cells
	// copied or compared between buffers keep their meaning.
	combining unique.Handle[string]
}

// noCombining marks a cell holding a single code point.
//
// It is the zero Handle rather than a handle to the empty string, because a
// zeroed buffer must already mean "no continuation". Handle.Value dereferences
// an interior pointer, so the zero Handle must never reach it.
var noCombining unique.Handle[string]

// combiningHandle interns s, or returns noCombining when s is empty.
func combiningHandle(s string) unique.Handle[string] {
	if s == "" {
		return noCombining
	}
	return unique.Make(s)
}

// CellClear is the default empty cell.
var CellClear = Cell{
	Char:  ' ',
	Fg:    style.ColorWhite,
	Bg:    style.ColorTransparent,
	Width: 1,
}

// NewCell returns a cell whose width is derived from char.
func NewCell(char rune, fg, bg style.Color, attrs style.Attrs) Cell {
	return Cell{
		Char:       char,
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      int16(RuneWidth(char)),
	}
}

// Combining returns the code points following Char in this cell's grapheme
// cluster, or the empty string when the cell holds a single code point.
func (c Cell) Combining() string {
	if c.combining == noCombining {
		return ""
	}
	return c.combining.Value()
}

// HasCombining reports whether the cell holds more than one code point.
func (c Cell) HasCombining() bool { return c.combining != noCombining }

// WithCombining returns a copy of c carrying the given cluster continuation.
// An empty continuation clears it.
func (c Cell) WithCombining(s string) Cell {
	c.combining = combiningHandle(s)
	return c
}

// IsWide reports whether c is the head of a multi-column character.
func (c Cell) IsWide() bool {
	return c.Width > 1
}

// IsContinuation reports whether c continues a multi-column character.
func (c Cell) IsContinuation() bool {
	return c.Width < 0
}

// IsEmpty reports whether c contains no visible character.
func (c Cell) IsEmpty() bool {
	return c.Char == 0 || c.Char == ' '
}

// Equals reports whether c and other have equal content and style.
func (c Cell) Equals(other Cell) bool {
	return c.Char == other.Char &&
		c.combining == other.combining &&
		c.Fg.Equals(other.Fg) &&
		c.Bg.Equals(other.Bg) &&
		c.Attributes == other.Attributes &&
		c.Width == other.Width
}

// WithFg returns a copy with a new foreground color.
func (c Cell) WithFg(fg style.Color) Cell {
	c.Fg = fg
	return c
}

// WithBg returns a copy with a new background color.
func (c Cell) WithBg(bg style.Color) Cell {
	c.Bg = bg
	return c
}

// RuneWidth returns the number of terminal columns occupied by r. It treats a
// zero rune as a one-column empty cell.
func RuneWidth(r rune) int {
	if r == 0 {
		return 1
	}
	return textbuf.RuneWidth(r)
}

// isColorWithAlpha reports whether c is not fully opaque.
func isColorWithAlpha(c style.Color) bool {
	return c.A < 255
}

// BlendCell composites src over dst using perceptual alpha blending.
func BlendCell(dst, src Cell) Cell {
	hasBgAlpha := isColorWithAlpha(src.Bg)
	hasFgAlpha := isColorWithAlpha(src.Fg)

	if !hasBgAlpha && !hasFgAlpha {
		// An opaque source completely replaces the destination.
		return src
	}

	blendedBg := src.Bg
	if hasBgAlpha {
		blendedBg = style.ColorBlendPerceptual(src.Bg, dst.Bg)
	}

	charIsSpace := src.Char == ' ' || src.Char == 0
	dstNotZero := dst.Char != 0
	dstNotSpace := dst.Char != ' '
	dstWidthIsOne := dst.Width == 1

	preserveChar := charIsSpace && dstNotZero && dstNotSpace && dstWidthIsOne

	finalChar := src.Char
	if preserveChar {
		finalChar = dst.Char
	}

	var finalFg style.Color
	if preserveChar {
		// Tint the preserved character with the source background.
		finalFg = style.ColorBlendPerceptual(src.Bg, dst.Fg)
	} else {
		if hasFgAlpha {
			finalFg = style.ColorBlendPerceptual(src.Fg, dst.Bg)
		} else {
			finalFg = src.Fg
		}
	}

	var finalAttrs style.Attrs
	if preserveChar {
		finalAttrs = dst.Attributes
	} else {
		finalAttrs = src.Attributes
	}

	// A transparent source background preserves the destination alpha.
	finalBgAlpha := blendedBg.A
	if src.Bg.A == 0 {
		finalBgAlpha = dst.Bg.A
	} else {
		finalBgAlpha = src.Bg.A
	}

	finalWidth := src.Width
	if preserveChar {
		finalWidth = dst.Width
	}

	return Cell{
		Char:       finalChar,
		Fg:         finalFg,
		Bg:         style.ColorFromInts(blendedBg.R, blendedBg.G, blendedBg.B, finalBgAlpha),
		Attributes: finalAttrs,
		Width:      finalWidth,
	}
}
