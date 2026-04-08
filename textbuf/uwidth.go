package textbuf

import "unicode/utf8"

// RuneWidth returns the display width of a rune (0, 1, or 2).
//
// Zero-width characters (control codes, combining marks, ZWJ, variation
// selectors, etc.) return 0. East Asian Wide and Fullwidth characters
// return 2. All others return 1.
func RuneWidth(r rune) int {
	// C0 control characters (including NUL)
	if r < 0x20 {
		return 0
	}
	// ASCII printable — fast path for the overwhelmingly common case
	if r < 0x7F {
		return 1
	}
	// DEL + C1 control characters
	if r <= 0x9F {
		return 0
	}
	// Zero-width characters (combining marks, format chars, etc.)
	if isZeroWidth(r) {
		return 0
	}
	// East Asian Wide / Fullwidth
	if isWide(r) {
		return 2
	}
	return 1
}

// StringWidth returns the display width of s using grapheme-cluster-aware
// width calculation (UAX #29). Handles ZWJ sequences, skin tone modifiers,
// flag pairs, combining marks, variation selectors, etc.
func StringWidth(s string) int {
	w := 0
	for len(s) > 0 {
		_, rest, cw := NextGraphemeCluster(s)
		w += cw
		s = rest
	}
	return w
}

// NextGraphemeCluster extracts the first grapheme cluster from s.
// Returns (cluster, rest, clusterWidth).
func NextGraphemeCluster(s string) (cluster, rest string, width int) {
	if s == "" {
		return "", "", 0
	}
	n := firstClusterBreak(s)
	cluster = s[:n]
	rest = s[n:]
	width = graphemeClusterWidth(cluster)
	return
}

// GraphemeClusterCount returns the number of grapheme clusters in s.
func GraphemeClusterCount(s string) int {
	n := 0
	for len(s) > 0 {
		b := firstClusterBreak(s)
		s = s[b:]
		n++
	}
	return n
}

// graphemeClusterWidth returns the display width of a single grapheme cluster.
//
// Width rules:
//   - Regional indicator pairs → 2
//   - Extended_Pictographic base + VS16 (U+FE0F) → 2
//   - Extended_Pictographic base + VS15 (U+FE0E) → 1
//   - Otherwise → RuneWidth of the base (first) rune
func graphemeClusterWidth(cluster string) int {
	if cluster == "" {
		return 0
	}

	r, sz := utf8.DecodeRuneInString(cluster)

	// Regional indicator pair → width 2
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return 2
	}

	baseWidth := RuneWidth(r)

	// For Extended_Pictographic, check for variation selectors.
	if isExtendedPictographic(r) {
		hasVS16 := false
		hasVS15 := false
		for i := sz; i < len(cluster); {
			cr, csz := utf8.DecodeRuneInString(cluster[i:])
			switch cr {
			case 0xFE0F:
				hasVS16 = true
			case 0xFE0E:
				hasVS15 = true
			}
			i += csz
		}
		if hasVS15 {
			return 1
		}
		if hasVS16 || baseWidth == 2 {
			return 2
		}
		return baseWidth
	}

	return baseWidth
}
