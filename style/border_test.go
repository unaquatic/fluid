package style

import (
	"testing"
	"unicode"

	"github.com/unaquatic/fluid/symbol"
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
		if got, ok := ParseBorderStyle(tt.input); got != tt.want || !ok {
			t.Errorf("ParseBorderStyle(%q) = %d, %v, want %d, true", tt.input, got, ok, tt.want)
		}
	}
}

func TestParseBorderStyle_Invalid(t *testing.T) {
	// Unknown names return the zero BorderStyle and false.
	got, ok := ParseBorderStyle("invalid")
	if got != BorderSingle || ok {
		t.Errorf("ParseBorderStyle(\"invalid\") = %d, %v, want BorderSingle, false", got, ok)
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
	if chars != symbol.SingleChars {
		t.Errorf("CharsForStyle(Single) != symbol.SingleChars")
	}
}

func TestCharsForStyle_DoubleChars(t *testing.T) {
	chars := CharsForStyle(BorderDouble)
	if chars != symbol.DoubleChars {
		t.Errorf("CharsForStyle(Double) != symbol.DoubleChars")
	}
}

func TestCharsForStyle_RoundedChars(t *testing.T) {
	chars := CharsForStyle(BorderRounded)
	if chars != symbol.RoundedChars {
		t.Errorf("CharsForStyle(Rounded) != symbol.RoundedChars")
	}
}

func TestCharsForStyle_HeavyChars(t *testing.T) {
	chars := CharsForStyle(BorderHeavy)
	if chars != symbol.HeavyChars {
		t.Errorf("CharsForStyle(Heavy) != symbol.HeavyChars")
	}
}

func TestCharsForStyle_None(t *testing.T) {
	// BorderNone uses symbol.SingleChars when characters are requested directly.
	chars := CharsForStyle(BorderNone)
	if chars != symbol.SingleChars {
		t.Errorf("CharsForStyle(None) should fallback to symbol.SingleChars")
	}
}

// TestAllPresetsHaveValidRunes verifies that every preset contains printable
// Unicode border runes.
func TestAllPresetsHaveValidRunes(t *testing.T) {
	presets := map[string]symbol.BorderCharacters{
		"Single":  symbol.SingleChars,
		"Double":  symbol.DoubleChars,
		"Rounded": symbol.RoundedChars,
		"Heavy":   symbol.HeavyChars,
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
	if symbol.SingleChars.TopLeft != '┌' {
		t.Errorf("TopLeft = %q, want '┌'", symbol.SingleChars.TopLeft)
	}
	if symbol.SingleChars.TopRight != '┐' {
		t.Errorf("TopRight = %q, want '┐'", symbol.SingleChars.TopRight)
	}
	if symbol.SingleChars.BottomLeft != '└' {
		t.Errorf("BottomLeft = %q, want '└'", symbol.SingleChars.BottomLeft)
	}
	if symbol.SingleChars.BottomRight != '┘' {
		t.Errorf("BottomRight = %q, want '┘'", symbol.SingleChars.BottomRight)
	}
	if symbol.SingleChars.Horizontal != '─' {
		t.Errorf("Horizontal = %q, want '─'", symbol.SingleChars.Horizontal)
	}
	if symbol.SingleChars.Vertical != '│' {
		t.Errorf("Vertical = %q, want '│'", symbol.SingleChars.Vertical)
	}
}

func TestDoubleChars_CorrectValues(t *testing.T) {
	if symbol.DoubleChars.TopLeft != '╔' {
		t.Errorf("TopLeft = %q, want '╔'", symbol.DoubleChars.TopLeft)
	}
	if symbol.DoubleChars.Horizontal != '═' {
		t.Errorf("Horizontal = %q, want '═'", symbol.DoubleChars.Horizontal)
	}
	if symbol.DoubleChars.Vertical != '║' {
		t.Errorf("Vertical = %q, want '║'", symbol.DoubleChars.Vertical)
	}
}

func TestRoundedChars_CorrectCorners(t *testing.T) {
	if symbol.RoundedChars.TopLeft != '╭' {
		t.Errorf("TopLeft = %q, want '╭'", symbol.RoundedChars.TopLeft)
	}
	if symbol.RoundedChars.TopRight != '╮' {
		t.Errorf("TopRight = %q, want '╮'", symbol.RoundedChars.TopRight)
	}
	if symbol.RoundedChars.BottomLeft != '╰' {
		t.Errorf("BottomLeft = %q, want '╰'", symbol.RoundedChars.BottomLeft)
	}
	if symbol.RoundedChars.BottomRight != '╯' {
		t.Errorf("BottomRight = %q, want '╯'", symbol.RoundedChars.BottomRight)
	}
	// Rounded borders differ from single borders only at their corners.
	if symbol.RoundedChars.Horizontal != symbol.SingleChars.Horizontal {
		t.Error("Rounded horizontal should match Single")
	}
}

func TestHeavyChars_CorrectValues(t *testing.T) {
	if symbol.HeavyChars.TopLeft != '┏' {
		t.Errorf("TopLeft = %q, want '┏'", symbol.HeavyChars.TopLeft)
	}
	if symbol.HeavyChars.Horizontal != '━' {
		t.Errorf("Horizontal = %q, want '━'", symbol.HeavyChars.Horizontal)
	}
	if symbol.HeavyChars.Vertical != '┃' {
		t.Errorf("Vertical = %q, want '┃'", symbol.HeavyChars.Vertical)
	}
}
