package textbuf

import (
	"unicode"
	"unicode/utf8"
)

// gcbProp identifies a UAX #29 Grapheme_Cluster_Break property.
type gcbProp int

const (
	gcbOther gcbProp = iota
	gcbCR
	gcbLF
	gcbControl
	gcbExtend
	gcbZWJ
	gcbRI
	gcbPrepend
	gcbSpacingMark
	gcbL
	gcbV
	gcbT
	gcbLV
	gcbLVT
)

// gcbOf returns r's Grapheme_Cluster_Break property.
func gcbOf(r rune) gcbProp {
	// Handle exact singleton properties first.
	if r == '\n' {
		return gcbLF
	}
	if r == '\r' {
		return gcbCR
	}
	if r == 0x200D {
		return gcbZWJ
	}

	// CR and LF were separated from the remaining C0 controls above.
	if r >= 0 && r < 0x0A {
		return gcbControl
	}
	if r > 0x0A && r < 0x0D {
		return gcbControl
	}
	if r > 0x0D && r <= 0x1F {
		return gcbControl
	}
	// DEL and C1 controls share the Control property.
	if r >= 0x7F && r <= 0x9F {
		return gcbControl
	}

	// Prepend format characters must be classified before the general Cf case.
	if isPrepend(r) {
		return gcbPrepend
	}

	// Regional indicator symbols form flag pairs.
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return gcbRI
	}

	// Classify Hangul jamo and syllables algorithmically.
	if r >= 0x1100 && r <= 0x115F {
		return gcbL
	}
	if r >= 0xA960 && r <= 0xA97C {
		return gcbL
	}
	if r >= 0x1160 && r <= 0x11A7 {
		return gcbV
	}
	if r >= 0xD7B0 && r <= 0xD7C6 {
		return gcbV
	}
	if r >= 0x11A8 && r <= 0x11FF {
		return gcbT
	}
	if r >= 0xD7CB && r <= 0xD7FB {
		return gcbT
	}
	// A zero trailing index identifies an LV syllable; all others are LVT.
	if r >= 0xAC00 && r <= 0xD7A3 {
		if (r-0xAC00)%28 == 0 {
			return gcbLV
		}
		return gcbLVT
	}

	if isGCBExtend(r) {
		return gcbExtend
	}

	if isGCBSpacingMark(r) {
		return gcbSpacingMark
	}

	// Remaining format characters behave as controls for segmentation.
	if unicode.Is(unicode.Cf, r) {
		return gcbControl
	}

	// Treat surrogate code points and Unicode noncharacters as controls.
	if r >= 0xD800 && r <= 0xDFFF {
		return gcbControl
	}
	if r >= 0xFDD0 && r <= 0xFDEF {
		return gcbControl
	}
	if r&0xFFFE == 0xFFFE { // The final two code points in every plane.
		return gcbControl
	}

	return gcbOther
}

// firstClusterBreak returns the byte length of the first grapheme cluster
// in s according to these UAX #29 boundary rules:
//
//	GB3:  CR × LF
//	GB4:  (Control | CR | LF) ÷
//	GB5:  ÷ (Control | CR | LF)
//	GB6:  L × (L | V | LV | LVT)
//	GB7:  (LV | V) × (V | T)
//	GB8:  (LVT | T) × T
//	GB9:  × (Extend | ZWJ)
//	GB9a: × SpacingMark
//	GB9b: Prepend ×
//	GB11: ExtPict Extend* ZWJ × ExtPict
//	GB12: sot (RI RI)* RI × RI
//	GB13: [^RI] (RI RI)* RI × RI
//	GB999: Any ÷ Any
func firstClusterBreak(s string) int {
	if len(s) == 0 {
		return 0
	}

	r0, sz0 := utf8.DecodeRuneInString(s)
	pos := sz0
	if pos >= len(s) {
		return pos
	}

	prev := gcbOf(r0)

	// epState tracks the Extended_Pictographic Extend* ZWJ sequence for GB11.
	// Zero means inactive, one means the pictograph and optional extenders were
	// seen, and two means the trailing ZWJ was also seen.
	epState := 0
	if isExtendedPictographic(r0) {
		epState = 1
	}

	// riCount tracks regional indicators for pairing under GB12 and GB13.
	riCount := 0
	if prev == gcbRI {
		riCount = 1
	}

	for pos < len(s) {
		r, sz := utf8.DecodeRuneInString(s[pos:])
		cur := gcbOf(r)
		curEP := isExtendedPictographic(r)

		// UAX #29 requires these boundary rules in priority order.
		shouldBreak := true

		switch {
		// GB3: CR × LF
		case prev == gcbCR && cur == gcbLF:
			shouldBreak = false

		// GB4: (Control | CR | LF) ÷
		case prev == gcbControl || prev == gcbCR || prev == gcbLF:
			shouldBreak = true

		// GB5: ÷ (Control | CR | LF)
		case cur == gcbControl || cur == gcbCR || cur == gcbLF:
			shouldBreak = true

		// GB6: L × (L | V | LV | LVT)
		case prev == gcbL && (cur == gcbL || cur == gcbV || cur == gcbLV || cur == gcbLVT):
			shouldBreak = false

		// GB7: (LV | V) × (V | T)
		case (prev == gcbLV || prev == gcbV) && (cur == gcbV || cur == gcbT):
			shouldBreak = false

		// GB8: (LVT | T) × T
		case (prev == gcbLVT || prev == gcbT) && cur == gcbT:
			shouldBreak = false

		// GB9: × (Extend | ZWJ)
		case cur == gcbExtend || cur == gcbZWJ:
			shouldBreak = false

		// GB9a: × SpacingMark
		case cur == gcbSpacingMark:
			shouldBreak = false

		// GB9b: Prepend ×
		case prev == gcbPrepend:
			shouldBreak = false

		// GB11: ExtPict Extend* ZWJ × ExtPict
		case epState == 2 && curEP:
			shouldBreak = false

		// GB12/13: RI × RI when an odd count starts the current pair.
		case prev == gcbRI && cur == gcbRI && riCount%2 == 1:
			shouldBreak = false

		// GB999: Any ÷ Any
		default:
			shouldBreak = true
		}

		if shouldBreak {
			break
		}

		// Update extended-pictographic state for the next boundary.
		switch {
		case curEP:
			epState = 1
		case cur == gcbExtend && epState >= 1:
			// Extenders preserve state one.
		case cur == gcbZWJ && epState >= 1:
			epState = 2
		default:
			epState = 0
		}

		// Reset regional-indicator pairing after any other property.
		if cur == gcbRI {
			riCount++
		} else {
			riCount = 0
		}

		prev = cur
		pos += sz
	}

	return pos
}
