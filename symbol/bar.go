package symbol

// Vertical bar runes in one-eighth-cell increments.
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

// BarNineLevels contains vertical fills from empty through full in one-eighth
// increments.
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

// BarThreeLevels contains empty, half, and full vertical fills.
var BarThreeLevels = [3]rune{BarEmpty, BarHalf, BarFull}

// Horizontal bar runes filled from left to right in one-eighth-cell increments.
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
