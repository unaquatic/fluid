package buffer

import (
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// Cell represents a single terminal cell.
type Cell struct {
	Char      rune   // Primary Unicode codepoint (0 = empty)
	Combining string // Zero-width combining codepoints that follow Char
	//                       (ZWJ sequences, skin-tone modifiers, etc.).
	//                       Emitted to the terminal immediately after Char.
	Fg         style.Color // Foreground color
	Bg         style.Color // Background color
	Attributes uint32      // Lower 8 bits: style flags; upper bits: link ID
	Width      int8        // 1 = normal, 2 = wide char first cell, -1 = wide char continuation
}

// CellClear is the default empty cell.
var CellClear = Cell{
	Char:  ' ',
	Fg:    style.ColorWhite,
	Bg:    style.ColorTransparent,
	Width: 1,
}

// NewCell creates a cell with auto-computed width from the rune.
func NewCell(char rune, fg, bg style.Color, attrs uint32) Cell {
	w := RuneWidth(char)
	return Cell{
		Char:       char,
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      int8(w),
	}
}

// IsWide returns true if this cell is the first cell of a wide character.
func (c Cell) IsWide() bool {
	return c.Width == 2
}

// IsContinuation returns true if this cell is a continuation of a wide character.
func (c Cell) IsContinuation() bool {
	return c.Width == -1
}

// IsEmpty returns true if the cell has no visible character.
func (c Cell) IsEmpty() bool {
	return c.Char == 0 || c.Char == ' '
}

// Equals performs deep equality comparison.
func (c Cell) Equals(other Cell) bool {
	return c.Char == other.Char &&
		c.Combining == other.Combining &&
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

// RuneWidth returns the display width of a rune (0, 1, or 2).
// Delegates to textbuf.RuneWidth — the canonical implementation.
func RuneWidth(r rune) int {
	if r == 0 {
		return 1
	}
	return textbuf.RuneWidth(r)
}

// isColorWithAlpha returns true if the color has non-opaque alpha.
func isColorWithAlpha(c style.Color) bool {
	return c.A < 255
}

// BlendCell composites src over dst.
// Uses perceptual alpha blending for background and foreground colors.
func BlendCell(dst, src Cell) Cell {
	hasBgAlpha := isColorWithAlpha(src.Bg)
	hasFgAlpha := isColorWithAlpha(src.Fg)

	if !hasBgAlpha && !hasFgAlpha {
		// Fully opaque overlay: just return src
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
		// Blend overlay bg into destination fg
		finalFg = style.ColorBlendPerceptual(src.Bg, dst.Fg)
	} else {
		if hasFgAlpha {
			finalFg = style.ColorBlendPerceptual(src.Fg, dst.Bg)
		} else {
			finalFg = src.Fg
		}
	}

	var finalAttrs uint32
	if preserveChar {
		finalAttrs = dst.Attributes
	} else {
		finalAttrs = src.Attributes
	}

	// When overlay background is fully transparent, preserve destination background alpha
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
