package style

// Style combines foreground, background, and text attributes.
type Style struct {
	// Fg is the foreground color.
	Fg Color
	// Bg is the background color.
	Bg Color
	// Attributes is a bitmask of the Attr flags, such as AttrBold.
	Attributes Attrs
}

// New returns white text on a transparent background with no attributes.
func New() Style {
	return Style{
		Fg:         ColorWhite,
		Bg:         ColorTransparent,
		Attributes: AttrNone,
	}
}

// From returns a style with the supplied colors and attributes.
func From(fg, bg Color, attrs Attrs) Style {
	return Style{
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
	}
}

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
func (s Style) WithAttrs(attrs Attrs) Style {
	s.Attributes = attrs
	return s
}

// Func returns a function that applies s to its text argument.
func (s Style) Func() func(string) TextChunk {
	return func(text string) TextChunk {
		return Styled(text, s)
	}
}

// Merge overlays other on s. Non-transparent colors in other replace the
// corresponding colors in s, and nonzero attributes are added to s's attributes.
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

// Equals reports whether s and other have equal colors and attributes.
func (s Style) Equals(other Style) bool {
	return s.Fg.Equals(other.Fg) &&
		s.Bg.Equals(other.Bg) &&
		s.Attributes == other.Attributes
}

// ResetANSI resets all ANSI SGR attributes.
const ResetANSI = "\x1b[0m"
