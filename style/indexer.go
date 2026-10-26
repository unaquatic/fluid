package style

import "fmt"

// ANSI is an entry of the terminal's indexed color palette. Entries 0
// through 15 are the ANSI colors, whose colors the terminal's theme picks;
// entries 16 through 255 are the xterm color cube and gray ramp.
type ANSI uint8

// The ANSI colors, entries 0 through 15 of the palette.
const (
	ANSIBlack ANSI = iota
	ANSIRed
	ANSIGreen
	ANSIYellow
	ANSIBlue
	ANSIMagenta
	ANSICyan
	ANSIWhite
	ANSIBrightBlack
	ANSIBrightRed
	ANSIBrightGreen
	ANSIBrightYellow
	ANSIBrightBlue
	ANSIBrightMagenta
	ANSIBrightCyan
	ANSIBrightWhite
)

// FgSeq returns an ANSI foreground-color sequence for a in mode: one of the
// 16 ANSI codes in Color16, and an xterm palette sequence in the other
// modes. Color16 draws an entry past the 16 ANSI colors in the ANSI color
// nearest to it.
func (a ANSI) FgSeq(mode ColorMode) string {
	if mode != Color16 {
		return fmt.Sprintf("\x1b[38;5;%dm", a)
	}
	a = a.ansi16()
	if a < ANSIBrightBlack {
		return fmt.Sprintf("\x1b[%dm", 30+int(a))
	}
	return fmt.Sprintf("\x1b[%dm", 90+int(a-ANSIBrightBlack))
}

// BgSeq returns an ANSI background-color sequence for a in mode, as FgSeq
// returns a foreground one.
func (a ANSI) BgSeq(mode ColorMode) string {
	if mode != Color16 {
		return fmt.Sprintf("\x1b[48;5;%dm", a)
	}
	a = a.ansi16()
	if a < ANSIBrightBlack {
		return fmt.Sprintf("\x1b[%dm", 40+int(a))
	}
	return fmt.Sprintf("\x1b[%dm", 100+int(a-ANSIBrightBlack))
}

// Color returns the color fluid assumes entry a draws in: its guess at the
// terminal theme's color for the 16 ANSI colors, and the standard xterm color
// for the color cube and gray ramp.
func (a ANSI) Color() Color {
	if a <= ANSIBrightWhite {
		c := ansi16Colors[a]
		return Color{R: uint8(c[0]), G: uint8(c[1]), B: uint8(c[2]), A: 255}
	}
	return xtermColor(int(a))
}

// ansi16 returns a, or for an entry past the 16 ANSI colors the ANSI color
// nearest to it.
func (a ANSI) ansi16() ANSI {
	if a <= ANSIBrightWhite {
		return a
	}
	return Nearest16.Index(xtermColor(int(a)))
}

// An Indexer converts colors to the entries of the terminal's palette that
// they draw in.
type Indexer interface {
	// Index returns the entry that c draws in.
	Index(c Color) ANSI
}

// IndexerFunc is an Indexer that converts colors by calling itself.
type IndexerFunc func(Color) ANSI

// Index returns f(c).
func (f IndexerFunc) Index(c Color) ANSI { return f(c) }

// Nearest16 and Nearest256 are the Indexers the color modes with a palette
// convert colors with by default. Nearest16 converts a color to the nearest
// of the 16 ANSI colors, as Color16 does. Nearest256 converts it to the
// perceptually nearest entry of the xterm color cube and gray ramp, 16
// through 255, as Color256 does; it leaves out the ANSI colors, which
// terminal themes redefine.
var (
	Nearest16 Indexer = IndexerFunc(func(c Color) ANSI {
		return ANSI(nearestAnsi16(int(c.R), int(c.G), int(c.B)))
	})
	Nearest256 Indexer = IndexerFunc(func(c Color) ANSI {
		return ANSI(rgbTo256(int(c.R), int(c.G), int(c.B)))
	})
)

// Indexer returns the Indexer that m converts colors with by default:
// Nearest16 for Color16, Nearest256 for Color256, and nil for
// ColorTrueColor, which draws colors as they are.
func (m ColorMode) Indexer() Indexer {
	switch m {
	case Color16:
		return Nearest16
	case Color256:
		return Nearest256
	}
	return nil
}

// IndexMap is an Indexer that converts the colors Map lists to their entries
// and passes the others to Fallback.
type IndexMap struct {
	// Map maps opaque colors to the entries they draw in. A color draws in
	// the entry of the opaque color with its red, green and blue.
	Map map[Color]ANSI
	// Fallback converts the colors Map does not list, such as Nearest16 or
	// Nearest256. A nil Fallback is Nearest16, whose entries every color
	// mode with a palette draws.
	Fallback Indexer
}

// Index returns the entry m.Map lists for c, or the one m.Fallback converts
// c to.
func (m IndexMap) Index(c Color) ANSI {
	if a, ok := m.Map[Color{R: c.R, G: c.G, B: c.B, A: 255}]; ok {
		return a
	}
	if m.Fallback == nil {
		return Nearest16.Index(c)
	}
	return m.Fallback.Index(c)
}

// Indexers are the Indexers that a terminal converts colors with in a color
// mode: Fg those of text and Bg those of backgrounds, so that one color can
// draw in one entry as text and in another behind it. A nil Indexer converts
// colors with the mode's own Indexer, and in ColorTrueColor draws them as
// they are. The zero Indexers convert every color as the mode does by
// default.
type Indexers struct {
	// Fg converts the colors of text.
	Fg Indexer
	// Bg converts the colors of backgrounds.
	Bg Indexer
}

// FgSeq returns an ANSI foreground-color sequence for c in mode, in the
// entry that x.Fg converts c to.
func (x Indexers) FgSeq(c Color, mode ColorMode) string {
	ix := x.Fg
	if ix == nil {
		ix = mode.Indexer()
	}
	if ix == nil {
		return FgColorSeq(c, mode)
	}
	return ix.Index(c).FgSeq(mode)
}

// BgSeq returns an ANSI background-color sequence for c in mode, in the
// entry that x.Bg converts c to.
func (x Indexers) BgSeq(c Color, mode ColorMode) string {
	ix := x.Bg
	if ix == nil {
		ix = mode.Indexer()
	}
	if ix == nil {
		return BgColorSeq(c, mode)
	}
	return ix.Index(c).BgSeq(mode)
}
