package textbuf

import (
	"sort"
	"unicode"
)

// urange is an inclusive code-point interval.
type urange = [2]rune

// inRange reports whether r belongs to any interval in a table sorted by upper
// bound.
func inRange(r rune, table []urange) bool {
	i := sort.Search(len(table), func(i int) bool {
		return table[i][1] >= r
	})
	return i < len(table) && table[i][0] <= r
}

// isZeroWidth reports whether r is a combining mark or belongs to an additional
// zero-width format range.
func isZeroWidth(r rune) bool {
	// Mn and Me contain standard nonspacing and enclosing marks.
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return true
	}
	// Additional format characters are listed separately.
	return inRange(r, zeroWidthExtra)
}

// zeroWidthExtra contains zero-width format characters, controls, tags, and
// trailing Hangul jamo outside the Mn and Me categories.
//
// Intervals are sorted by their first code point.
var zeroWidthExtra = []urange{
	{0x1160, 0x11FF}, // Hangul Jungseong + Jongseong (V+T jamo)
	{0x200B, 0x200F}, // ZWSP, ZWNJ, ZWJ, LRM, RLM
	{0x2028, 0x202E}, // Line/Paragraph Separator, Bidi controls
	{0x2060, 0x206F}, // Word Joiner, invisible operators, deprecated Cf
	{0xD7B0, 0xD7C6}, // Hangul Jamo Extended-B (V jamo)
	{0xD7CB, 0xD7FB}, // Hangul Jamo Extended-B (T jamo)
	{0xFEFF, 0xFEFF}, // BOM / ZWNBSP

	{0xE0001, 0xE0001}, // Language Tag
	{0xE0020, 0xE007F}, // Tag characters
	{0xE0100, 0xE01EF}, // Variation Selectors Supplement
}

// isWide reports whether r has East Asian Width W or F.
func isWide(r rune) bool {
	return inRange(r, wideRanges)
}

// wideRanges contains Unicode 16.0 East Asian Width W and F intervals.
//
// Intervals are sorted by their first code point.
var wideRanges = []urange{
	// Hangul leading consonant jamo.
	{0x1100, 0x115F},

	// Wide miscellaneous symbols and emoji.
	{0x231A, 0x231B},
	{0x2329, 0x232A},
	{0x23E9, 0x23F3},
	{0x23F8, 0x23FA},
	{0x25FD, 0x25FE},
	{0x2614, 0x2615},
	{0x2648, 0x2653},
	{0x267F, 0x267F},
	{0x2693, 0x2693},
	{0x26A1, 0x26A1},
	{0x26AA, 0x26AB},
	{0x26BD, 0x26BE},
	{0x26C4, 0x26C5},
	{0x26CE, 0x26CE},
	{0x26D4, 0x26D4},
	{0x26EA, 0x26EA},
	{0x26F2, 0x26F3},
	{0x26F5, 0x26F5},
	{0x26FA, 0x26FA},
	{0x26FD, 0x26FD},
	{0x2702, 0x2702},
	{0x2705, 0x2705},
	{0x2708, 0x270D},
	{0x270F, 0x270F},
	{0x2728, 0x2728},
	{0x274C, 0x274C},
	{0x274E, 0x274E},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2795, 0x2797},
	{0x27B0, 0x27B0},
	{0x27BF, 0x27BF},
	{0x2B1B, 0x2B1C},
	{0x2B50, 0x2B50},
	{0x2B55, 0x2B55},

	// CJK Radicals Supplement.
	{0x2E80, 0x2E99},
	{0x2E9B, 0x2EF3},

	// Kangxi Radicals.
	{0x2F00, 0x2FD5},

	// Ideographic description characters, symbols, and punctuation.
	{0x2FF0, 0x303E},

	// Hiragana and related kana.
	{0x3041, 0x3096},
	{0x3099, 0x30FF},

	// Bopomofo.
	{0x3105, 0x312F},

	// Hangul Compatibility Jamo.
	{0x3131, 0x318E},

	// Kanbun and CJK Strokes.
	{0x3190, 0x31E3},

	// Katakana Phonetic Extensions.
	{0x31F0, 0x321E},

	// Enclosed CJK Letters and Months.
	{0x3220, 0x3247},

	// CJK ranges through Unified Ideographs Extension A.
	{0x3250, 0x4DBF},

	// CJK Unified Ideographs and Yi.
	{0x4E00, 0xA48C},

	// Yi Radicals.
	{0xA490, 0xA4C6},

	// Hangul Jamo Extended-A.
	{0xA960, 0xA97C},

	// Hangul Syllables.
	{0xAC00, 0xD7A3},

	// CJK Compatibility Ideographs.
	{0xF900, 0xFAFF},

	// Vertical Forms.
	{0xFE10, 0xFE19},

	// CJK Compatibility Forms.
	{0xFE30, 0xFE6B},

	// Fullwidth Latin characters and signs.
	{0xFF01, 0xFF60},
	{0xFFE0, 0xFFE6},

	// Ideographic symbols, Tangut, Khitan, and Nushu.
	{0x16FE0, 0x16FE4},
	{0x16FF0, 0x16FF1},
	{0x17000, 0x187F7},
	{0x18800, 0x18CD5},
	{0x18D00, 0x18D08},

	// Kana extensions and supplements.
	{0x1AFF0, 0x1AFF3},
	{0x1AFF5, 0x1AFFB},
	{0x1AFFD, 0x1AFFE},
	{0x1B000, 0x1B122},
	{0x1B132, 0x1B132},
	{0x1B150, 0x1B152},
	{0x1B155, 0x1B155},
	{0x1B164, 0x1B167},
	{0x1B170, 0x1B2FB},

	// Individual wide Mahjong and playing-card symbols.
	{0x1F004, 0x1F004},
	{0x1F0CF, 0x1F0CF},
	{0x1F18E, 0x1F18E},
	{0x1F191, 0x1F19A},

	// Enclosed Ideographic Supplement.
	{0x1F200, 0x1F202},
	{0x1F210, 0x1F23B},
	{0x1F240, 0x1F248},
	{0x1F250, 0x1F251},
	{0x1F260, 0x1F265},

	// Miscellaneous Symbols and Pictographs.
	{0x1F300, 0x1F320},
	{0x1F32D, 0x1F335},
	{0x1F337, 0x1F37C},
	{0x1F37E, 0x1F393},
	{0x1F3A0, 0x1F3CA},
	{0x1F3CF, 0x1F3D3},
	{0x1F3E0, 0x1F3F0},
	{0x1F3F4, 0x1F3F4},
	{0x1F3F8, 0x1F43E},
	{0x1F440, 0x1F440},
	{0x1F442, 0x1F4FC},
	{0x1F4FF, 0x1F53D},
	{0x1F54B, 0x1F54E},
	{0x1F550, 0x1F567},
	{0x1F57A, 0x1F57A},
	{0x1F595, 0x1F596},
	{0x1F5A4, 0x1F5A4},
	{0x1F5FB, 0x1F64F},

	// Transport and Map Symbols.
	{0x1F680, 0x1F6C5},
	{0x1F6CC, 0x1F6CC},
	{0x1F6D0, 0x1F6D2},
	{0x1F6D5, 0x1F6D7},
	{0x1F6DC, 0x1F6DF},
	{0x1F6EB, 0x1F6EC},
	{0x1F6F4, 0x1F6FC},

	// Geometric Shapes Extended.
	{0x1F7E0, 0x1F7EB},
	{0x1F7F0, 0x1F7F0},

	// Supplemental Symbols and Pictographs.
	{0x1F90C, 0x1F93A},
	{0x1F93C, 0x1F945},
	{0x1F947, 0x1F9FF},

	// Symbols and Pictographs Extended-A.
	{0x1FA70, 0x1FA7C},
	{0x1FA80, 0x1FA88},
	{0x1FA90, 0x1FABD},
	{0x1FABF, 0x1FAC5},
	{0x1FACE, 0x1FADB},
	{0x1FAE0, 0x1FAE8},
	{0x1FAF0, 0x1FAF8},

	// CJK Unified Ideographs Extensions B through I.
	{0x20000, 0x2FFFD},
	{0x30000, 0x3FFFD},
}

// isExtendedPictographic reports whether r has the Extended_Pictographic
// property used by GB11 and emoji width calculation.
func isExtendedPictographic(r rune) bool {
	// No earlier code point has the property.
	if r < 0x00A9 {
		return false
	}
	return inRange(r, extPictRanges)
}

// extPictRanges contains Unicode 16.0 Extended_Pictographic intervals.
//
// Consolidated intervals include reserved code points assigned the property.
var extPictRanges = []urange{
	{0x00A9, 0x00A9}, // ©
	{0x00AE, 0x00AE}, // ®
	{0x203C, 0x203C},
	{0x2049, 0x2049},
	{0x2122, 0x2122},
	{0x2139, 0x2139},
	{0x2194, 0x2199},
	{0x21A9, 0x21AA},
	{0x231A, 0x231B},
	{0x2328, 0x2328},
	{0x23CF, 0x23CF},
	{0x23E9, 0x23F3},
	{0x23F8, 0x23FA},
	{0x24C2, 0x24C2},
	{0x25AA, 0x25AB},
	{0x25B6, 0x25B6},
	{0x25C0, 0x25C0},
	{0x25FB, 0x25FE},
	{0x2600, 0x2605},
	{0x2607, 0x2612},
	{0x2614, 0x2615},
	{0x2616, 0x2685},
	{0x2690, 0x2705},
	{0x2708, 0x2712},
	{0x2714, 0x2714},
	{0x2716, 0x2716},
	{0x271D, 0x271D},
	{0x2721, 0x2721},
	{0x2728, 0x2728},
	{0x2733, 0x2734},
	{0x2744, 0x2744},
	{0x2747, 0x2747},
	{0x274C, 0x274C},
	{0x274E, 0x274E},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2763, 0x2767},
	{0x2795, 0x2797},
	{0x27A1, 0x27A1},
	{0x27B0, 0x27B0},
	{0x27BF, 0x27BF},
	{0x2934, 0x2935},
	{0x2B05, 0x2B07},
	{0x2B1B, 0x2B1C},
	{0x2B50, 0x2B50},
	{0x2B55, 0x2B55},
	{0x3030, 0x3030},
	{0x303D, 0x303D},
	{0x3297, 0x3297},
	{0x3299, 0x3299},

	// Mahjong Tiles.
	{0x1F000, 0x1F02C},
	// Domino Tiles and Playing Cards.
	{0x1F030, 0x1F093},
	{0x1F0A0, 0x1F0AE},
	{0x1F0B1, 0x1F0BF},
	{0x1F0C1, 0x1F0CF},
	{0x1F0D1, 0x1F0F5},

	// Enclosed Alphanumeric and Ideographic Supplement.
	{0x1F10D, 0x1F10F},
	{0x1F12F, 0x1F12F},
	{0x1F16C, 0x1F171},
	{0x1F17E, 0x1F17F},
	{0x1F18E, 0x1F18E},
	{0x1F191, 0x1F19A},
	{0x1F1AD, 0x1F1AD},
	{0x1F1E6, 0x1F1FF},
	{0x1F200, 0x1F202},
	{0x1F210, 0x1F23B},
	{0x1F240, 0x1F248},
	{0x1F250, 0x1F251},
	{0x1F260, 0x1F265},

	// Miscellaneous Symbols, Pictographs, and Emoticons.
	{0x1F300, 0x1F579},
	{0x1F57B, 0x1F5A3},
	{0x1F5A5, 0x1F64F},

	// Transport and Map Symbols.
	{0x1F680, 0x1F6FF},

	// Alchemical Symbols and extended geometric shapes.
	{0x1F774, 0x1F77F},
	{0x1F7D5, 0x1F7FF},

	// Supplemental Symbols.
	{0x1F80C, 0x1F80F},
	{0x1F848, 0x1F84F},
	{0x1F85A, 0x1F85F},
	{0x1F888, 0x1F88F},
	{0x1F8AE, 0x1F8FF},

	// Supplemental Symbols and Pictographs.
	{0x1F900, 0x1F9FF},

	// Symbols and Pictographs Extended-A.
	{0x1FA00, 0x1FA53},
	{0x1FA60, 0x1FA6D},
	{0x1FA70, 0x1FAFF},

	// Symbols Extended-B.
	{0x1FC00, 0x1FFFD},
}

// isPrepend reports whether r has Grapheme_Cluster_Break=Prepend.
func isPrepend(r rune) bool {
	return inRange(r, prependRanges)
}

// prependRanges contains Unicode 16.0 Grapheme_Cluster_Break=Prepend intervals.
var prependRanges = []urange{
	{0x0600, 0x0605},
	{0x06DD, 0x06DD},
	{0x070F, 0x070F},
	{0x0890, 0x0891},
	{0x08E2, 0x08E2},
	{0x0D4E, 0x0D4E},
	{0x110BD, 0x110BD},
	{0x110CD, 0x110CD},
	{0x111C2, 0x111C3},
	{0x1193F, 0x1193F},
	{0x11941, 0x11941},
	{0x11A3A, 0x11A3A},
	{0x11A84, 0x11A89},
	{0x11D46, 0x11D46},
	{0x11F02, 0x11F02},
}

// isGCBExtend reports whether r has Grapheme_Cluster_Break=Extend. This includes
// Mn, Me, and additional format and modifier characters.
func isGCBExtend(r rune) bool {
	// Standard combining marks cover most Extend characters.
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return true
	}
	// The remaining Extend characters are table-driven.
	return inRange(r, gcbExtendExtra)
}

// gcbExtendExtra contains Unicode 16.0 Extend characters outside Mn and Me.
//
// Intervals are sorted by their first code point.
var gcbExtendExtra = []urange{
	{0x0E31, 0x0E31},   // Thai Character Mai Han Akat
	{0x0E34, 0x0E3A},   // Thai vowel signs
	{0x0E47, 0x0E4E},   // Thai marks
	{0x0EB1, 0x0EB1},   // Lao vowel sign
	{0x0EB4, 0x0EBC},   // Lao vowel signs
	{0x0EC8, 0x0ECE},   // Lao tone marks
	{0x0F18, 0x0F19},   // Tibetan astrological signs
	{0x0F35, 0x0F35},   // Tibetan
	{0x0F37, 0x0F37},   // Tibetan
	{0x0F39, 0x0F39},   // Tibetan
	{0x0F71, 0x0F7E},   // Tibetan vowels
	{0x0F80, 0x0F84},   // Tibetan
	{0x0F86, 0x0F87},   // Tibetan
	{0x0FC6, 0x0FC6},   // Tibetan
	{0x200C, 0x200C},   // ZWNJ
	{0xFF9E, 0xFF9F},   // Halfwidth katakana voiced marks
	{0x1F3FB, 0x1F3FF}, // Emoji skin tone modifiers
	{0xE0020, 0xE007F}, // Tags
	{0xE0100, 0xE01EF}, // Variation Selectors Supplement
}

// isGCBSpacingMark reports whether r has Grapheme_Cluster_Break=SpacingMark.
func isGCBSpacingMark(r rune) bool {
	// Most Mc characters are spacing marks.
	if unicode.Is(unicode.Mc, r) {
		// Extend takes precedence for Mc characters assigned both categories.
		if isGCBExtend(r) {
			return false
		}
		return true
	}
	// A small table covers spacing marks outside Mc.
	return inRange(r, spacingMarkExtra)
}

// spacingMarkExtra contains spacing marks outside the Mc category.
var spacingMarkExtra = []urange{
	{0x0E33, 0x0E33}, // Thai Sara Am
	{0x0EB3, 0x0EB3}, // Lao Vowel Sign Am
}
