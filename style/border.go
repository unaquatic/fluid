package style

import "github.com/unaquatic/fluid/symbol"

// ---------------------------------------------------------------------------
// BorderStyle
// ---------------------------------------------------------------------------

// BorderStyle defines the visual style of a border.
type BorderStyle int

const (
	BorderSingle  BorderStyle = iota // thin box-drawing characters
	BorderDouble                     // double-line box-drawing
	BorderRounded                    // single with rounded corners
	BorderHeavy                      // thick/bold box-drawing
	BorderNone                       // no border
)

// String returns the string name of a BorderStyle.
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

// ParseBorderStyle parses a string to a BorderStyle.
// If the string is invalid, returns the fallback (default: BorderSingle).
func ParseBorderStyle(s string, fallback ...BorderStyle) BorderStyle {
	fb := BorderSingle
	if len(fallback) > 0 {
		fb = fallback[0]
	}
	switch s {
	case "single":
		return BorderSingle
	case "double":
		return BorderDouble
	case "rounded":
		return BorderRounded
	case "heavy":
		return BorderHeavy
	case "none":
		return BorderNone
	default:
		return fb
	}
}

// IsValidBorderStyle checks if a string is a valid border style name.
func IsValidBorderStyle(s string) bool {
	switch s {
	case "single", "double", "rounded", "heavy", "none":
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// BorderSide / BorderSidesConfig
// ---------------------------------------------------------------------------

// BorderSide identifies a side of a box.
type BorderSide int

const (
	SideTop BorderSide = iota
	SideRight
	SideBottom
	SideLeft
)

// BorderSidesConfig indicates which sides of a border are enabled.
type BorderSidesConfig struct {
	Top    bool
	Right  bool
	Bottom bool
	Left   bool
}

// AllSides returns a config with all sides enabled.
func AllSides() BorderSidesConfig {
	return BorderSidesConfig{Top: true, Right: true, Bottom: true, Left: true}
}

// NoSides returns a config with all sides disabled.
func NoSides() BorderSidesConfig {
	return BorderSidesConfig{}
}

// ---------------------------------------------------------------------------
// Border
// ---------------------------------------------------------------------------

// Border specifies which sides to draw.
type Border struct {
	All   bool         // true = all sides
	Sides []BorderSide // specific sides (only used if All is false)
}

// BorderAll returns a Border with all sides enabled.
func BorderAll() Border {
	return Border{All: true}
}

// BorderNoBorder returns a Border with no sides.
func BorderNoBorder() Border {
	return Border{}
}

// BorderSides returns a Border with specific sides.
func BorderSides(sides ...BorderSide) Border {
	return Border{Sides: sides}
}

// GetBorderSides converts a Border to per-side booleans.
func GetBorderSides(b Border) BorderSidesConfig {
	if b.All {
		return AllSides()
	}
	if len(b.Sides) == 0 {
		return NoSides()
	}
	config := BorderSidesConfig{}
	for _, s := range b.Sides {
		switch s {
		case SideTop:
			config.Top = true
		case SideRight:
			config.Right = true
		case SideBottom:
			config.Bottom = true
		case SideLeft:
			config.Left = true
		}
	}
	return config
}

// GetBorderFromSides converts per-side booleans to a Border.
func GetBorderFromSides(config BorderSidesConfig) Border {
	if config.Top && config.Right && config.Bottom && config.Left {
		return BorderAll()
	}
	var sides []BorderSide
	if config.Top {
		sides = append(sides, SideTop)
	}
	if config.Right {
		sides = append(sides, SideRight)
	}
	if config.Bottom {
		sides = append(sides, SideBottom)
	}
	if config.Left {
		sides = append(sides, SideLeft)
	}
	if len(sides) == 0 {
		return BorderNoBorder()
	}
	return BorderSides(sides...)
}

// ---------------------------------------------------------------------------
// BorderCharacters — type alias for symbols.BorderCharacters
// ---------------------------------------------------------------------------

// BorderCharacters defines the 11 runes used to draw a box border.
// The canonical definition lives in the symbols package; this is
// a type alias so existing code that imports style continues to work.
type BorderCharacters = symbol.BorderCharacters

// Preset border character sets — delegated to symbols.
var (
	SingleChars  = symbol.SingleChars
	DoubleChars  = symbol.DoubleChars
	RoundedChars = symbol.RoundedChars
	HeavyChars   = symbol.HeavyChars
)

// CharsForStyle returns the preset border characters for a given style.
func CharsForStyle(bs BorderStyle) BorderCharacters {
	switch bs {
	case BorderSingle:
		return SingleChars
	case BorderDouble:
		return DoubleChars
	case BorderRounded:
		return RoundedChars
	case BorderHeavy:
		return HeavyChars
	default:
		return SingleChars
	}
}
