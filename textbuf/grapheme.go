package textbuf

import (
	"unicode"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Grapheme_Cluster_Break (GCB) property
// ---------------------------------------------------------------------------

// gcbProp represents the UAX #29 Grapheme_Cluster_Break property value.
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

// gcbOf returns the Grapheme_Cluster_Break property for r.
func gcbOf(r rune) gcbProp {
	// --- Exact single-codepoint values ---
	if r == '\n' {
		return gcbLF
	}
	if r == '\r' {
		return gcbCR
	}
	if r == 0x200D {
		return gcbZWJ
	}

	// --- C0 control characters (excluding CR/LF already handled) ---
	if r >= 0 && r < 0x0A {
		return gcbControl
	}
	if r > 0x0A && r < 0x0D {
		return gcbControl
	}
	if r > 0x0D && r <= 0x1F {
		return gcbControl
	}
	// DEL + C1 controls
	if r >= 0x7F && r <= 0x9F {
		return gcbControl
	}

	// --- Prepend (must check before general Cf check) ---
	if isPrepend(r) {
		return gcbPrepend
	}

	// --- Regional Indicator ---
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return gcbRI
	}

	// --- Hangul Jamo algorithmic classification ---
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
	// Hangul syllables: LV if trailing index == 0, else LVT
	if r >= 0xAC00 && r <= 0xD7A3 {
		if (r-0xAC00)%28 == 0 {
			return gcbLV
		}
		return gcbLVT
	}

	// --- Extend (combining marks + supplementary) ---
	if isGCBExtend(r) {
		return gcbExtend
	}

	// --- SpacingMark ---
	if isGCBSpacingMark(r) {
		return gcbSpacingMark
	}

	// --- Remaining format characters → Control ---
	if unicode.Is(unicode.Cf, r) {
		return gcbControl
	}

	// --- Surrogates and noncharacters → Control ---
	if r >= 0xD800 && r <= 0xDFFF {
		return gcbControl
	}
	if r >= 0xFDD0 && r <= 0xFDEF {
		return gcbControl
	}
	if r&0xFFFE == 0xFFFE { // U+xFFFE and U+xFFFF
		return gcbControl
	}

	return gcbOther
}

// ---------------------------------------------------------------------------
// firstClusterBreak returns the byte length of the first grapheme cluster
// in s, implementing the UAX #29 grapheme cluster boundary rules:
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
//
// ---------------------------------------------------------------------------
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

	// --- State for GB11 ---
	// epState tracks Extended_Pictographic + Extend* + ZWJ context:
	//   0 = not tracking
	//   1 = saw ExtPict (possibly followed by Extend*)
	//   2 = saw ExtPict + Extend* + ZWJ
	epState := 0
	if isExtendedPictographic(r0) {
		epState = 1
	}

	// --- State for GB12/GB13 ---
	riCount := 0
	if prev == gcbRI {
		riCount = 1
	}

	for pos < len(s) {
		r, sz := utf8.DecodeRuneInString(s[pos:])
		cur := gcbOf(r)
		curEP := isExtendedPictographic(r)

		// Apply break rules in priority order.
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

		// GB12/13: RI × RI (only pairs — odd count means we're mid-pair)
		case prev == gcbRI && cur == gcbRI && riCount%2 == 1:
			shouldBreak = false

		// GB999: Any ÷ Any
		default:
			shouldBreak = true
		}

		if shouldBreak {
			break
		}

		// --- Update state for next iteration ---

		// Update EP tracking for GB11.
		switch {
		case curEP:
			epState = 1
		case cur == gcbExtend && epState >= 1:
			// Extend preserves EP context: ExtPict Extend*
			// epState stays at 1
		case cur == gcbZWJ && epState >= 1:
			epState = 2
		default:
			epState = 0
		}

		// Update RI count for GB12/13.
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
