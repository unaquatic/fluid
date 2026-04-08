package style

import (
	"testing"
	"unicode"
)

// ---------------------------------------------------------------------------
// BorderStyle tests
// ---------------------------------------------------------------------------

func TestBorderStyle_String(t *testing.T) {
	tests := []struct {
		bs   BorderStyle
		want string
	}{
		{BorderSingle, "single"},
		{BorderDouble, "double"},
		{BorderRounded, "rounded"},
		{BorderHeavy, "heavy"},
		{BorderNone, "none"},
		{BorderStyle(99), "single"}, // unknown defaults to single
	}
	for _, tt := range tests {
		if got := tt.bs.String(); got != tt.want {
			t.Errorf("BorderStyle(%d).String() = %q, want %q", tt.bs, got, tt.want)
		}
	}
}

func TestParseBorderStyle(t *testing.T) {
	tests := []struct {
		input string
		want  BorderStyle
	}{
		{"single", BorderSingle},
		{"double", BorderDouble},
		{"rounded", BorderRounded},
		{"heavy", BorderHeavy},
		{"none", BorderNone},
	}
	for _, tt := range tests {
		if got := ParseBorderStyle(tt.input); got != tt.want {
			t.Errorf("ParseBorderStyle(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseBorderStyle_InvalidDefault(t *testing.T) {
	// Default fallback is BorderSingle
	got := ParseBorderStyle("invalid")
	if got != BorderSingle {
		t.Errorf("ParseBorderStyle(\"invalid\") = %d, want BorderSingle", got)
	}
}

func TestParseBorderStyle_CustomFallback(t *testing.T) {
	got := ParseBorderStyle("invalid", BorderNone)
	if got != BorderNone {
		t.Errorf("ParseBorderStyle(\"invalid\", BorderNone) = %d, want BorderNone", got)
	}
}

func TestIsValidBorderStyle(t *testing.T) {
	valid := []string{"single", "double", "rounded", "heavy", "none"}
	for _, s := range valid {
		if !IsValidBorderStyle(s) {
			t.Errorf("IsValidBorderStyle(%q) = false, want true", s)
		}
	}

	invalid := []string{"", "bold", "Single", "DOUBLE", "dashed"}
	for _, s := range invalid {
		if IsValidBorderStyle(s) {
			t.Errorf("IsValidBorderStyle(%q) = true, want false", s)
		}
	}
}

// ---------------------------------------------------------------------------
// BorderSidesConfig tests
// ---------------------------------------------------------------------------

func TestAllSides(t *testing.T) {
	s := AllSides()
	if !s.Top || !s.Right || !s.Bottom || !s.Left {
		t.Errorf("AllSides() = %+v, want all true", s)
	}
}

func TestNoSides(t *testing.T) {
	s := NoSides()
	if s.Top || s.Right || s.Bottom || s.Left {
		t.Errorf("NoSides() = %+v, want all false", s)
	}
}

// ---------------------------------------------------------------------------
// Border tests
// ---------------------------------------------------------------------------

func TestBorderAll(t *testing.T) {
	b := BorderAll()
	if !b.All {
		t.Error("BorderAll().All should be true")
	}
}

func TestBorderNoBorder(t *testing.T) {
	b := BorderNoBorder()
	if b.All || len(b.Sides) > 0 {
		t.Errorf("BorderNoBorder() = %+v, want empty", b)
	}
}

func TestBorderSidesFunc(t *testing.T) {
	b := BorderSides(SideTop, SideBottom)
	if b.All {
		t.Error("BorderSides should not set All")
	}
	if len(b.Sides) != 2 {
		t.Fatalf("expected 2 sides, got %d", len(b.Sides))
	}
	if b.Sides[0] != SideTop || b.Sides[1] != SideBottom {
		t.Errorf("Sides = %v, want [Top, Bottom]", b.Sides)
	}
}

func TestGetBorderSides_All(t *testing.T) {
	config := GetBorderSides(BorderAll())
	if !config.Top || !config.Right || !config.Bottom || !config.Left {
		t.Errorf("GetBorderSides(All) = %+v, want all true", config)
	}
}

func TestGetBorderSides_None(t *testing.T) {
	config := GetBorderSides(BorderNoBorder())
	if config.Top || config.Right || config.Bottom || config.Left {
		t.Errorf("GetBorderSides(None) = %+v, want all false", config)
	}
}

func TestGetBorderSides_Specific(t *testing.T) {
	config := GetBorderSides(BorderSides(SideTop, SideLeft))
	if !config.Top || config.Right || config.Bottom || !config.Left {
		t.Errorf("GetBorderSides(Top,Left) = %+v, want Top=true, Left=true", config)
	}
}

func TestGetBorderFromSides_AllTrue(t *testing.T) {
	b := GetBorderFromSides(AllSides())
	if !b.All {
		t.Error("GetBorderFromSides(AllSides) should set All=true")
	}
}

func TestGetBorderFromSides_AllFalse(t *testing.T) {
	b := GetBorderFromSides(NoSides())
	if b.All || len(b.Sides) > 0 {
		t.Errorf("GetBorderFromSides(NoSides) = %+v, want empty", b)
	}
}

func TestGetBorderFromSides_Partial(t *testing.T) {
	config := BorderSidesConfig{Top: true, Right: true}
	b := GetBorderFromSides(config)
	if b.All {
		t.Error("partial sides should not set All")
	}
	if len(b.Sides) != 2 {
		t.Fatalf("expected 2 sides, got %d", len(b.Sides))
	}
	// Check roundtrip
	config2 := GetBorderSides(b)
	if config2.Top != config.Top || config2.Right != config.Right ||
		config2.Bottom != config.Bottom || config2.Left != config.Left {
		t.Errorf("roundtrip failed: got %+v, want %+v", config2, config)
	}
}

func TestGetBorderSides_Roundtrip(t *testing.T) {
	// Test all 16 combinations
	bools := []bool{false, true}
	for _, top := range bools {
		for _, right := range bools {
			for _, bottom := range bools {
				for _, left := range bools {
					config := BorderSidesConfig{Top: top, Right: right, Bottom: bottom, Left: left}
					b := GetBorderFromSides(config)
					config2 := GetBorderSides(b)
					if config2 != config {
						t.Errorf("roundtrip failed for %+v: got %+v", config, config2)
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// BorderCharacters tests
// ---------------------------------------------------------------------------

func TestCharsForStyle_SingleChars(t *testing.T) {
	chars := CharsForStyle(BorderSingle)
	if chars != SingleChars {
		t.Errorf("CharsForStyle(Single) != SingleChars")
	}
}

func TestCharsForStyle_DoubleChars(t *testing.T) {
	chars := CharsForStyle(BorderDouble)
	if chars != DoubleChars {
		t.Errorf("CharsForStyle(Double) != DoubleChars")
	}
}

func TestCharsForStyle_RoundedChars(t *testing.T) {
	chars := CharsForStyle(BorderRounded)
	if chars != RoundedChars {
		t.Errorf("CharsForStyle(Rounded) != RoundedChars")
	}
}

func TestCharsForStyle_HeavyChars(t *testing.T) {
	chars := CharsForStyle(BorderHeavy)
	if chars != HeavyChars {
		t.Errorf("CharsForStyle(Heavy) != HeavyChars")
	}
}

func TestCharsForStyle_None(t *testing.T) {
	// BorderNone falls through to SingleChars
	chars := CharsForStyle(BorderNone)
	if chars != SingleChars {
		t.Errorf("CharsForStyle(None) should fallback to SingleChars")
	}
}

// TestAllPresetsHaveValidRunes checks that all runes in all presets are valid
// printable Unicode characters (box-drawing range).
func TestAllPresetsHaveValidRunes(t *testing.T) {
	presets := map[string]BorderCharacters{
		"Single":  SingleChars,
		"Double":  DoubleChars,
		"Rounded": RoundedChars,
		"Heavy":   HeavyChars,
	}

	for name, chars := range presets {
		runes := []rune{
			chars.TopLeft, chars.TopRight, chars.BottomLeft, chars.BottomRight,
			chars.Horizontal, chars.Vertical,
			chars.TopT, chars.BottomT, chars.LeftT, chars.RightT, chars.Cross,
		}
		for i, r := range runes {
			if r == 0 {
				t.Errorf("%s preset: rune %d is zero", name, i)
			}
			if !unicode.IsPrint(r) {
				t.Errorf("%s preset: rune %d (%U) is not printable", name, i, r)
			}
		}
	}
}

func TestSingleChars_CorrectValues(t *testing.T) {
	if SingleChars.TopLeft != '┌' {
		t.Errorf("TopLeft = %q, want '┌'", SingleChars.TopLeft)
	}
	if SingleChars.TopRight != '┐' {
		t.Errorf("TopRight = %q, want '┐'", SingleChars.TopRight)
	}
	if SingleChars.BottomLeft != '└' {
		t.Errorf("BottomLeft = %q, want '└'", SingleChars.BottomLeft)
	}
	if SingleChars.BottomRight != '┘' {
		t.Errorf("BottomRight = %q, want '┘'", SingleChars.BottomRight)
	}
	if SingleChars.Horizontal != '─' {
		t.Errorf("Horizontal = %q, want '─'", SingleChars.Horizontal)
	}
	if SingleChars.Vertical != '│' {
		t.Errorf("Vertical = %q, want '│'", SingleChars.Vertical)
	}
}

func TestDoubleChars_CorrectValues(t *testing.T) {
	if DoubleChars.TopLeft != '╔' {
		t.Errorf("TopLeft = %q, want '╔'", DoubleChars.TopLeft)
	}
	if DoubleChars.Horizontal != '═' {
		t.Errorf("Horizontal = %q, want '═'", DoubleChars.Horizontal)
	}
	if DoubleChars.Vertical != '║' {
		t.Errorf("Vertical = %q, want '║'", DoubleChars.Vertical)
	}
}

func TestRoundedChars_CorrectCorners(t *testing.T) {
	if RoundedChars.TopLeft != '╭' {
		t.Errorf("TopLeft = %q, want '╭'", RoundedChars.TopLeft)
	}
	if RoundedChars.TopRight != '╮' {
		t.Errorf("TopRight = %q, want '╮'", RoundedChars.TopRight)
	}
	if RoundedChars.BottomLeft != '╰' {
		t.Errorf("BottomLeft = %q, want '╰'", RoundedChars.BottomLeft)
	}
	if RoundedChars.BottomRight != '╯' {
		t.Errorf("BottomRight = %q, want '╯'", RoundedChars.BottomRight)
	}
	// Edges should be same as single
	if RoundedChars.Horizontal != SingleChars.Horizontal {
		t.Error("Rounded horizontal should match Single")
	}
}

func TestHeavyChars_CorrectValues(t *testing.T) {
	if HeavyChars.TopLeft != '┏' {
		t.Errorf("TopLeft = %q, want '┏'", HeavyChars.TopLeft)
	}
	if HeavyChars.Horizontal != '━' {
		t.Errorf("Horizontal = %q, want '━'", HeavyChars.Horizontal)
	}
	if HeavyChars.Vertical != '┃' {
		t.Errorf("Vertical = %q, want '┃'", HeavyChars.Vertical)
	}
}
