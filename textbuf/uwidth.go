package textbuf

import "unicode/utf8"

// RuneWidth returns r's terminal width as zero, one, or two columns.
//
// Controls, combining marks, format characters, and variation selectors have
// zero width. East Asian Wide and Fullwidth runes have width two. Other runes
// have width one.
func RuneWidth(r rune) int {
	// C0 controls, including NUL, have no display width.
	if r < 0x20 {
		return 0
	}
	// Handle printable ASCII without table lookup.
	if r < 0x7F {
		return 1
	}
	// DEL and C1 controls have no display width.
	if r <= 0x9F {
		return 0
	}
	if isZeroWidth(r) {
		return 0
	}
	if isWide(r) {
		return 2
	}
	return 1
}

// StringWidth returns s's terminal width after UAX #29 grapheme segmentation.
func StringWidth(s string) int {
	w := 0
	for len(s) > 0 {
		_, rest, cw := NextGraphemeCluster(s)
		w += cw
		s = rest
	}
	return w
}

// NextGraphemeCluster returns s's first grapheme cluster, remaining text, and
// cluster display width.
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

// graphemeClusterWidth returns one grapheme cluster's terminal width.
//
// Regional-indicator clusters use two columns. VS16 selects two-column emoji
// presentation and VS15 selects one-column text presentation. Other clusters
// begin with the base rune's RuneWidth.
//
// Spacing marks joined by GB9a add their own display widths.
func graphemeClusterWidth(cluster string) int {
	if cluster == "" {
		return 0
	}

	r, sz := utf8.DecodeRuneInString(cluster)

	// Regional-indicator clusters represent flags in two columns.
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return 2
	}

	rest := cluster[sz:]
	w := RuneWidth(r)
	if isExtendedPictographic(r) {
		w = pictographicWidth(w, rest)
	}
	return w + spacingMarkWidth(rest)
}

// pictographicWidth applies variation selectors after an
// Extended_Pictographic base. VS15 selects one-column text presentation and
// VS16 selects two-column emoji presentation.
func pictographicWidth(baseWidth int, rest string) int {
	hasVS16 := false
	for _, r := range rest {
		switch r {
		case 0xFE0E:
			return 1
		case 0xFE0F:
			hasVS16 = true
		}
	}
	if hasVS16 || baseWidth == 2 {
		return 2
	}
	return baseWidth
}

// spacingMarkWidth returns the combined width of SpacingMark code points in
// rest.
//
// GB9a joins these marks to the preceding cluster even though they can still
// advance the terminal cursor.
func spacingMarkWidth(rest string) int {
	w := 0
	for _, r := range rest {
		if isGCBSpacingMark(r) {
			w += RuneWidth(r)
		}
	}
	return w
}
