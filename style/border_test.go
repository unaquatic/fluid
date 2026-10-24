package style

import (
	"testing"
	"unicode"
)

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
		{BorderStyle(99), "single"}, // Unknown values use the single-line name.
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
	// Without an explicit fallback, unknown names use BorderSingle.
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

func TestBorderAll(t *testing.T) {
	b := BorderAll()
	if !b.Top || !b.Right || !b.Bottom || !b.Left {
		t.Errorf("BorderAll() = %+v, want all true", b)
	}
}

func TestBorderNoBorder(t *testing.T) {
	if b := BorderNoBorder(); b != (Border{}) {
		t.Errorf("BorderNoBorder() = %+v, want empty", b)
	}
}

func TestBorderSides(t *testing.T) {
	tests := []struct {
		sides []BorderSide
		want  Border
	}{
		{nil, Border{}},
		{[]BorderSide{SideTop, SideBottom}, Border{Top: true, Bottom: true}},
		{[]BorderSide{SideLeft, SideTop, SideLeft}, Border{Top: true, Left: true}},
		{[]BorderSide{SideTop, SideRight, SideBottom, SideLeft}, BorderAll()},
	}
	for _, tt := range tests {
		if got := BorderSides(tt.sides...); got != tt.want {
			t.Errorf("BorderSides(%v) = %+v, want %+v", tt.sides, got, tt.want)
		}
	}
}

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
	// BorderNone uses SingleChars when characters are requested directly.
	chars := CharsForStyle(BorderNone)
	if chars != SingleChars {
		t.Errorf("CharsForStyle(None) should fallback to SingleChars")
	}
}

// TestAllPresetsHaveValidRunes verifies that every preset contains printable
// Unicode border runes.
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
	// Rounded borders differ from single borders only at their corners.
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
