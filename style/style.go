package style

import (
	"fmt"
	"strings"
)

// Style combines foreground, background, and text attributes.
type Style struct {
	Fg         Color
	Bg         Color
	Attributes uint32
}

// NewStyle returns a default style (white on transparent, no attributes).
func NewStyle() Style {
	return Style{
		Fg:         ColorWhite,
		Bg:         ColorTransparent,
		Attributes: AttrNone,
	}
}

// StyleFrom creates a Style with explicit foreground, background, and attributes.
func StyleFrom(fg, bg Color, attrs uint32) Style {
	return Style{
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
	}
}

// --- Builder methods (return copies) ---

// Bold returns a copy with the bold attribute set.
func (s Style) Bold() Style {
	s.Attributes |= AttrBold
	return s
}

// Italic returns a copy with the italic attribute set.
func (s Style) Italic() Style {
	s.Attributes |= AttrItalic
	return s
}

// Underline returns a copy with the underline attribute set.
func (s Style) Underline() Style {
	s.Attributes |= AttrUnderline
	return s
}

// Dim returns a copy with the dim attribute set.
func (s Style) Dim() Style {
	s.Attributes |= AttrDim
	return s
}

// Strikethrough returns a copy with the strikethrough attribute set.
func (s Style) Strikethrough() Style {
	s.Attributes |= AttrStrikethrough
	return s
}

// Inverse returns a copy with the inverse attribute set.
func (s Style) Inverse() Style {
	s.Attributes |= AttrInverse
	return s
}

// WithFg returns a copy with a new foreground color.
func (s Style) WithFg(fg Color) Style {
	s.Fg = fg
	return s
}

// WithBg returns a copy with a new background color.
func (s Style) WithBg(bg Color) Style {
	s.Bg = bg
	return s
}

// WithAttrs returns a copy with new attributes.
func (s Style) WithAttrs(attrs uint32) Style {
	s.Attributes = attrs
	return s
}

// Func returns a curried function that creates a TextChunk with this style applied.
func (s Style) Func() func(string) TextChunk {
	return func(text string) TextChunk {
		return Styled(text, s)
	}
}

// Merge merges another style into this one. Non-zero/non-transparent fields
// from other override this style's fields.
func (s Style) Merge(other Style) Style {
	if !other.Fg.IsTransparent() {
		s.Fg = other.Fg
	}
	if !other.Bg.IsTransparent() {
		s.Bg = other.Bg
	}
	if other.Attributes != AttrNone {
		s.Attributes |= other.Attributes
	}
	return s
}

// Equals performs deep equality comparison.
func (s Style) Equals(other Style) bool {
	return s.Fg.Equals(other.Fg) &&
		s.Bg.Equals(other.Bg) &&
		s.Attributes == other.Attributes
}

// AttrsANSI generates the SGR sequence for text attributes (bold, italic, etc.).
func AttrsANSI(attrs uint32) string {
	base := GetBaseAttributes(attrs)
	if base == AttrNone {
		return ""
	}

	var parts []string
	if base&AttrBold != 0 {
		parts = append(parts, "1")
	}
	if base&AttrDim != 0 {
		parts = append(parts, "2")
	}
	if base&AttrItalic != 0 {
		parts = append(parts, "3")
	}
	if base&AttrUnderline != 0 {
		parts = append(parts, "4")
	}
	if base&AttrBlink != 0 {
		parts = append(parts, "5")
	}
	if base&AttrInverse != 0 {
		parts = append(parts, "7")
	}
	if base&AttrHidden != 0 {
		parts = append(parts, "8")
	}
	if base&AttrStrikethrough != 0 {
		parts = append(parts, "9")
	}

	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}

// ResetANSI is the escape sequence to reset all attributes.
const ResetANSI = "\x1b[0m"
