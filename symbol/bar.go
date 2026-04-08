package symbol

// Bar symbols — vertical fill characters used by sparklines, bar charts,
// and similar widgets. Each rune represents an increasing fraction of a
// cell height, from empty to full.

const (
	BarEmpty         = ' '
	BarOneEighth     = '▁'
	BarOneQuarter    = '▂'
	BarThreeEighths  = '▃'
	BarHalf          = '▄'
	BarFiveEighths   = '▅'
	BarThreeQuarters = '▆'
	BarSevenEighths  = '▇'
	BarFull          = '█'
)

// BarNineLevels contains nine vertical-fill runes from empty to full,
// indexed 0–8. Suitable for sparkline rendering.
var BarNineLevels = [9]rune{
	BarEmpty,
	BarOneEighth,
	BarOneQuarter,
	BarThreeEighths,
	BarHalf,
	BarFiveEighths,
	BarThreeQuarters,
	BarSevenEighths,
	BarFull,
}

// BarThreeLevels is a compact 3-level vertical fill: empty, half, full.
var BarThreeLevels = [3]rune{BarEmpty, BarHalf, BarFull}

// ---------------------------------------------------------------------------
// Horizontal bar characters (left-to-right fill)
// ---------------------------------------------------------------------------

const (
	HBarOneEighth     = '▏'
	HBarOneQuarter    = '▎'
	HBarThreeEighths  = '▍'
	HBarHalf          = '▌'
	HBarFiveEighths   = '▋'
	HBarThreeQuarters = '▊'
	HBarSevenEighths  = '▉'
	HBarFull          = '█'
)
