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

// ParseBorderStyle returns the border style named by s. An unknown name returns
// the first fallback, or BorderSingle when no fallback is supplied.
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

// BorderSidesConfig selects individual border sides.
type BorderSidesConfig struct {
	Top    bool
	Right  bool
	Bottom bool
	Left   bool
}

// AllSides returns a configuration with every side enabled.
func AllSides() BorderSidesConfig {
	return BorderSidesConfig{Top: true, Right: true, Bottom: true, Left: true}
}

// NoSides returns a configuration with every side disabled.
func NoSides() BorderSidesConfig {
	return BorderSidesConfig{}
}

// Border specifies which sides to draw.
type Border struct {
	All   bool         // Whether every side is enabled.
	Sides []BorderSide // Enabled sides when All is false.
}

// BorderAll returns a Border with all sides enabled.
func BorderAll() Border {
	return Border{All: true}
}

// BorderNoBorder returns a Border with no sides.
func BorderNoBorder() Border {
	return Border{}
}

// BorderSides returns a Border containing the supplied sides.
func BorderSides(sides ...BorderSide) Border {
	return Border{Sides: sides}
}

// GetBorderSides expands b into per-side flags.
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

// GetBorderFromSides compacts per-side flags into a Border.
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

// BorderCharacters is an alias for [symbol.BorderCharacters], the runes needed
// to draw border lines and junctions.
type BorderCharacters = symbol.BorderCharacters

// Preset border character sets provided by the symbol package.
var (
	SingleChars  = symbol.SingleChars
	DoubleChars  = symbol.DoubleChars
	RoundedChars = symbol.RoundedChars
	HeavyChars   = symbol.HeavyChars
)

// CharsForStyle returns the character set for bs. Unsupported styles use the
// single-line set.
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
