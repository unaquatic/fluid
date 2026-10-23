package textbuf

import "testing"

// allTables lists every interval table that requires sorted order for binary
// search.
var allTables = map[string][]urange{
	"zeroWidthExtra":   zeroWidthExtra,
	"wideRanges":       wideRanges,
	"extPictRanges":    extPictRanges,
	"prependRanges":    prependRanges,
	"gcbExtendExtra":   gcbExtendExtra,
	"spacingMarkExtra": spacingMarkExtra,
}

func TestTablesSorted(t *testing.T) {
	for name, table := range allTables {
		t.Run(name, func(t *testing.T) {
			for i, rg := range table {
				if rg[0] > rg[1] {
					t.Errorf("range %d is inverted: {%04X, %04X}", i, rg[0], rg[1])
				}
				if i > 0 && rg[0] <= table[i-1][1] {
					t.Errorf("range %d {%04X, %04X} is not after range %d {%04X, %04X}",
						i, rg[0], rg[1], i-1, table[i-1][0], table[i-1][1])
				}
			}
		})
	}
}

func TestInRangeMatchesLinearScan(t *testing.T) {
	linear := func(r rune, table []urange) bool {
		for _, rg := range table {
			if r >= rg[0] && r <= rg[1] {
				return true
			}
		}
		return false
	}

	for name, table := range allTables {
		t.Run(name, func(t *testing.T) {
			// Compare binary and linear lookup at each interval boundary and neighbor.
			for _, rg := range table {
				for _, r := range []rune{rg[0] - 1, rg[0], rg[1], rg[1] + 1} {
					if got, want := inRange(r, table), linear(r, table); got != want {
						t.Errorf("inRange(U+%04X) = %v, want %v", r, got, want)
					}
				}
			}
		})
	}
}

func TestRuneWidthZeroWidthExtra(t *testing.T) {
	// These code points depend on zeroWidthExtra remaining sorted.
	zero := []rune{
		0x1160, // Hangul Jungseong Filler
		0x11FF, // Hangul Jongseong Ssangnieun
		0x200B, // Zero Width Space
		0x200D, // Zero Width Joiner
		0x2060, // Word Joiner
		0xD7B0, // Hangul Jamo Extended-B
		0xFEFF, // BOM / Zero Width No-Break Space
		0xE0100,
	}
	for _, r := range zero {
		if got := RuneWidth(r); got != 0 {
			t.Errorf("RuneWidth(U+%04X) = %d, want 0", r, got)
		}
	}
}
