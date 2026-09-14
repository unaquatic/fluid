package textbuf

import "testing"

func TestRuneWidth_ASCII(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		{'A', 1}, {'z', 1}, {'0', 1}, {' ', 1}, {'~', 1},
	}
	for _, tt := range tests {
		if got := RuneWidth(tt.r); got != tt.want {
			t.Errorf("RuneWidth(%q) = %d, want %d", tt.r, got, tt.want)
		}
	}
}

func TestRuneWidth_Control(t *testing.T) {
	tests := []rune{0, 1, '\t', '\n', '\r', 0x1F, 0x7F, 0x80, 0x9F}
	for _, r := range tests {
		if got := RuneWidth(r); got != 0 {
			t.Errorf("RuneWidth(0x%04X) = %d, want 0", r, got)
		}
	}
}

func TestRuneWidth_CJK(t *testing.T) {
	tests := []rune{
		0x4E2D, // 中
		0x6587, // 文
		0x3042, // あ (Hiragana)
		0x30A2, // ア (Katakana)
		0xAC00, // 가 (Hangul)
		0xFF01, // ！ (Fullwidth !)
	}
	for _, r := range tests {
		if got := RuneWidth(r); got != 2 {
			t.Errorf("RuneWidth(0x%04X) = %d, want 2", r, got)
		}
	}
}

func TestRuneWidth_Emoji(t *testing.T) {
	tests := []rune{
		0x1F600, // 😀
		0x1F680, // 🚀
		0x1F4A9, // 💩
	}
	for _, r := range tests {
		if got := RuneWidth(r); got != 2 {
			t.Errorf("RuneWidth(0x%04X) = %d, want 2", r, got)
		}
	}
}

func TestRuneWidth_CombiningMark(t *testing.T) {
	tests := []rune{
		0x0301, // Combining Acute Accent
		0x0300, // Combining Grave Accent
		0x0308, // Combining Diaeresis
		0x20E3, // Combining Enclosing Keycap
	}
	for _, r := range tests {
		if got := RuneWidth(r); got != 0 {
			t.Errorf("RuneWidth(0x%04X) = %d, want 0", r, got)
		}
	}
}

func TestRuneWidth_ZWJ(t *testing.T) {
	if got := RuneWidth(0x200D); got != 0 {
		t.Errorf("RuneWidth(ZWJ) = %d, want 0", got)
	}
}

func TestRuneWidth_VariationSelector(t *testing.T) {
	if got := RuneWidth(0xFE0F); got != 0 {
		t.Errorf("RuneWidth(VS16) = %d, want 0", got)
	}
	if got := RuneWidth(0xFE0E); got != 0 {
		t.Errorf("RuneWidth(VS15) = %d, want 0", got)
	}
}

func TestStringWidth_ASCII(t *testing.T) {
	if got := StringWidth("Hello, World!"); got != 13 {
		t.Errorf("StringWidth(ASCII) = %d, want 13", got)
	}
}

func TestStringWidth_CJK(t *testing.T) {
	if got := StringWidth("中文"); got != 4 {
		t.Errorf("StringWidth(CJK) = %d, want 4", got)
	}
}

func TestStringWidth_Emoji(t *testing.T) {
	if got := StringWidth("😀"); got != 2 {
		t.Errorf("StringWidth(emoji) = %d, want 2", got)
	}
}

func TestStringWidth_EmojiZWJ(t *testing.T) {
	// A ZWJ family sequence forms one two-column cluster.
	family := "👨\u200D👩\u200D👧\u200D👦"
	if got := StringWidth(family); got != 2 {
		t.Errorf("StringWidth(ZWJ family) = %d, want 2", got)
	}
}

func TestStringWidth_Flag(t *testing.T) {
	flag := "🇺🇸"
	if got := StringWidth(flag); got != 2 {
		t.Errorf("StringWidth(flag) = %d, want 2", got)
	}
}

func TestStringWidth_SkinTone(t *testing.T) {
	wave := "👋\U0001F3FD"
	if got := StringWidth(wave); got != 2 {
		t.Errorf("StringWidth(skin tone) = %d, want 2", got)
	}
}

func TestStringWidth_VS16(t *testing.T) {
	// VS16 selects two-column emoji presentation for U+263A.
	smiley := "\u263A\uFE0F"
	if got := StringWidth(smiley); got != 2 {
		t.Errorf("StringWidth(VS16) = %d, want 2", got)
	}
}

func TestStringWidth_CombiningMarks(t *testing.T) {
	// A combining acute accent adds no width to its base.
	accented := "e\u0301"
	if got := StringWidth(accented); got != 1 {
		t.Errorf("StringWidth(combining mark) = %d, want 1", got)
	}
}

func TestStringWidth_Mixed(t *testing.T) {
	// Two ASCII cells and two wide cells total six columns.
	if got := StringWidth("Hi中😀"); got != 6 {
		t.Errorf("StringWidth(mixed) = %d, want 6", got)
	}
}

func TestStringWidth_Empty(t *testing.T) {
	if got := StringWidth(""); got != 0 {
		t.Errorf("StringWidth(\"\") = %d, want 0", got)
	}
}

func TestStringWidth_SpacingMark(t *testing.T) {
	// GB9a joins spacing marks to a cluster while their columns still contribute.
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"thai sara am", "\u0E01\u0E33", 2},                 // กำ
		{"thai tone then sara am", "\u0E15\u0E48\u0E33", 2}, // The tone mark is nonspacing.
		{"thai nonspacing only", "\u0E17\u0E35\u0E48", 1},   // ที่
		{"thai word", "\u0E04\u0E33\u0E19\u0E27\u0E13", 5},  // คำนวณ
		{"lao vowel am", "\u0E81\u0EB3", 2},
		{"devanagari matra", "\u0915\u093E", 2}, // का
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringWidth(tt.s); got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestNextGraphemeCluster_ASCII(t *testing.T) {
	c, rest, w := NextGraphemeCluster("abc")
	if c != "a" || rest != "bc" || w != 1 {
		t.Errorf("NextGraphemeCluster(\"abc\") = (%q, %q, %d), want (\"a\", \"bc\", 1)", c, rest, w)
	}
}

func TestNextGraphemeCluster_CJK(t *testing.T) {
	c, rest, w := NextGraphemeCluster("中文")
	if c != "中" || rest != "文" || w != 2 {
		t.Errorf("NextGraphemeCluster(CJK) = (%q, %q, %d), want (\"中\", \"文\", 2)", c, rest, w)
	}
}

func TestNextGraphemeCluster_ZWJ(t *testing.T) {
	family := "👨\u200D👩\u200D👧\u200D👦X"
	c, rest, w := NextGraphemeCluster(family)
	want := "👨\u200D👩\u200D👧\u200D👦"
	if c != want || rest != "X" || w != 2 {
		t.Errorf("NextGraphemeCluster(ZWJ) = (%q, %q, %d), want (%q, \"X\", 2)", c, rest, w, want)
	}
}

func TestNextGraphemeCluster_Flag(t *testing.T) {
	s := "🇺🇸X"
	c, rest, w := NextGraphemeCluster(s)
	if c != "🇺🇸" || rest != "X" || w != 2 {
		t.Errorf("NextGraphemeCluster(flag) = (%q, %q, %d)", c, rest, w)
	}
}

func TestNextGraphemeCluster_CombiningMark(t *testing.T) {
	s := "e\u0301X"
	c, rest, w := NextGraphemeCluster(s)
	if c != "e\u0301" || rest != "X" || w != 1 {
		t.Errorf("NextGraphemeCluster(combining) = (%q, %q, %d)", c, rest, w)
	}
}

func TestNextGraphemeCluster_CRLF(t *testing.T) {
	c, rest, w := NextGraphemeCluster("\r\nX")
	if c != "\r\n" || rest != "X" || w != 0 {
		t.Errorf("NextGraphemeCluster(CRLF) = (%q, %q, %d)", c, rest, w)
	}
}

func TestNextGraphemeCluster_SkinTone(t *testing.T) {
	s := "👋\U0001F3FDX"
	c, rest, w := NextGraphemeCluster(s)
	want := "👋\U0001F3FD"
	if c != want || rest != "X" || w != 2 {
		t.Errorf("NextGraphemeCluster(skin tone) = (%q, %q, %d), want (%q, \"X\", 2)", c, rest, w, want)
	}
}

func TestNextGraphemeCluster_Empty(t *testing.T) {
	c, rest, w := NextGraphemeCluster("")
	if c != "" || rest != "" || w != 0 {
		t.Errorf("NextGraphemeCluster(\"\") = (%q, %q, %d)", c, rest, w)
	}
}

func TestGraphemeClusterCount_ASCII(t *testing.T) {
	if got := GraphemeClusterCount("abc"); got != 3 {
		t.Errorf("GraphemeClusterCount(\"abc\") = %d, want 3", got)
	}
}

func TestGraphemeClusterCount_CombiningMark(t *testing.T) {
	// A base and combining acute accent form one cluster.
	if got := GraphemeClusterCount("e\u0301"); got != 1 {
		t.Errorf("GraphemeClusterCount(\"e\\u0301\") = %d, want 1", got)
	}
}

func TestGraphemeClusterCount_ZWJ(t *testing.T) {
	family := "👨\u200D👩\u200D👧\u200D👦"
	if got := GraphemeClusterCount(family); got != 1 {
		t.Errorf("GraphemeClusterCount(ZWJ family) = %d, want 1", got)
	}
}

func TestGraphemeClusterCount_Flag(t *testing.T) {
	if got := GraphemeClusterCount("🇺🇸"); got != 1 {
		t.Errorf("GraphemeClusterCount(flag) = %d, want 1", got)
	}
}

func TestGraphemeClusterCount_Mixed(t *testing.T) {
	// Two ASCII, one CJK, and one emoji character form four clusters.
	if got := GraphemeClusterCount("Hi中😀"); got != 4 {
		t.Errorf("GraphemeClusterCount(mixed) = %d, want 4", got)
	}
}

func TestGraphemeClusterCount_Empty(t *testing.T) {
	if got := GraphemeClusterCount(""); got != 0 {
		t.Errorf("GraphemeClusterCount(\"\") = %d, want 0", got)
	}
}

func TestStringWidth_MultipleFlags(t *testing.T) {
	// Two flag clusters occupy two columns each.
	flags := "🇺🇸🇯🇵"
	if got := StringWidth(flags); got != 4 {
		t.Errorf("StringWidth(two flags) = %d, want 4", got)
	}
}

func TestStringWidth_ThreeRI(t *testing.T) {
	// Three regional indicators form one pair and one standalone cluster.
	three := "\U0001F1FA\U0001F1F8\U0001F1EF"
	if got := StringWidth(three); got != 4 {
		t.Errorf("StringWidth(3 RI) = %d, want 4", got)
	}
}

func TestStringWidth_MultipleCombining(t *testing.T) {
	// Multiple combining marks add no width to their base.
	s := "a\u0300\u0301"
	if got := StringWidth(s); got != 1 {
		t.Errorf("StringWidth(multi-combining) = %d, want 1", got)
	}
}
