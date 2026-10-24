package style

import "github.com/unaquatic/fluid/symbol"

// BorderStyle identifies a preset set of border-drawing characters.
type BorderStyle int

const (
	BorderSingle  BorderStyle = iota // Thin box-drawing characters.
	BorderDouble                     // Double-line box-drawing characters.
	BorderRounded                    // Thin lines with rounded corners.
	BorderHeavy                      // Heavy box-drawing characters.
	BorderNone                       // No border.
)

// String returns bs's name.
func (bs BorderStyle) String() string {
	switch bs {
	case BorderSingle:
		return "single"
	case BorderDouble:
		return "double"
	case BorderRounded:
		return "rounded"
	case BorderHeavy:
		return "heavy"
	case BorderNone:
		return "none"
	default:
		return "single"
	}
}

// ParseBorderStyle returns the border style named by s and true, or
// BorderSingle, the zero BorderStyle, and false when s names none.
func ParseBorderStyle(s string) (BorderStyle, bool) {
	switch s {
	case "single":
		return BorderSingle, true
	case "double":
		return BorderDouble, true
	case "rounded":
		return BorderRounded, true
	case "heavy":
		return BorderHeavy, true
	case "none":
		return BorderNone, true
	}
	return BorderSingle, false
}

// IsValidBorderStyle reports whether s names a supported border style.
func IsValidBorderStyle(s string) bool {
	switch s {
	case "single", "double", "rounded", "heavy", "none":
		return true
	default:
		return false
	}
}

// BorderSide identifies a side of a box.
type BorderSide int

const (
	SideTop BorderSide = iota
	SideRight
	SideBottom
	SideLeft
)

// Border selects the sides of a box to draw. Its zero value draws none.
type Border struct {
	// Top selects the top side.
	Top bool
	// Right selects the right side.
	Right bool
	// Bottom selects the bottom side.
	Bottom bool
	// Left selects the left side.
	Left bool
}

// BorderAll returns a Border with all sides enabled.
func BorderAll() Border {
	return Border{Top: true, Right: true, Bottom: true, Left: true}
}

// BorderNoBorder returns a Border with no sides.
func BorderNoBorder() Border {
	return Border{}
}

// BorderSides returns a Border with the supplied sides enabled.
func BorderSides(sides ...BorderSide) Border {
	var b Border
	for _, side := range sides {
		switch side {
		case SideTop:
			b.Top = true
		case SideRight:
			b.Right = true
		case SideBottom:
			b.Bottom = true
		case SideLeft:
			b.Left = true
		}
	}
	return b
}

// CharsForStyle returns the character set of package symbol for bs.
// Unsupported styles use the single-line set.
func CharsForStyle(bs BorderStyle) symbol.BorderCharacters {
	switch bs {
	case BorderSingle:
		return symbol.SingleChars
	case BorderDouble:
		return symbol.DoubleChars
	case BorderRounded:
		return symbol.RoundedChars
	case BorderHeavy:
		return symbol.HeavyChars
	default:
		return symbol.SingleChars
	}
}
