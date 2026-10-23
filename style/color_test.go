package style

import (
	"math"
	"testing"
)

func assertColorEquals(t *testing.T, got, want Color) {
	t.Helper()
	if !got.Equals(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func assertFloat32Near(t *testing.T, got, want float32, epsilon float32) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > epsilon {
		t.Errorf("got %f, want %f (epsilon=%f)", got, want, epsilon)
	}
}

// --- ColorFromHex tests ---

func TestFromHex_3Char(t *testing.T) {
	c := ColorFromHex("#fff")
	assertColorEquals(t, c, ColorWhite)
}

func TestFromHex_3Char_Color(t *testing.T) {
	c := ColorFromHex("#f00")
	assertColorEquals(t, c, ColorRed)
}

func TestFromHex_4Char(t *testing.T) {
	c := ColorFromHex("#f008")
	r, g, b, a := c.ToInts()
	if r != 255 || g != 0 || b != 0 || a != 136 {
		t.Errorf("got (%d,%d,%d,%d), want (255,0,0,136)", r, g, b, a)
	}
}

func TestFromHex_6Char(t *testing.T) {
	c := ColorFromHex("#ff8000")
	r, g, b, a := c.ToInts()
	if r != 255 || g != 128 || b != 0 || a != 255 {
		t.Errorf("got (%d,%d,%d,%d), want (255,128,0,255)", r, g, b, a)
	}
}

func TestFromHex_8Char(t *testing.T) {
	c := ColorFromHex("#ff000080")
	r, g, b, a := c.ToInts()
	if r != 255 || g != 0 || b != 0 || a != 128 {
		t.Errorf("got (%d,%d,%d,%d), want (255,0,0,128)", r, g, b, a)
	}
}

func TestFromHex_NoHash(t *testing.T) {
	c := ColorFromHex("ff0000")
	assertColorEquals(t, c, ColorRed)
}

func TestFromHex_InvalidLength(t *testing.T) {
	c := ColorFromHex("#12345")
	assertColorEquals(t, c, ColorTransparent)
}

func TestFromHex_UpperCase(t *testing.T) {
	c := ColorFromHex("#FF0000")
	assertColorEquals(t, c, ColorRed)
}

func TestFromHex_MixedCase(t *testing.T) {
	c := ColorFromHex("#Ff0000")
	assertColorEquals(t, c, ColorRed)
}

// --- ColorFromInts tests ---

func TestFromInts(t *testing.T) {
	c := ColorFromInts(255, 0, 0, 255)
	assertColorEquals(t, c, ColorRed)
}

func TestFromInts_HalfAlpha(t *testing.T) {
	c := ColorFromInts(0, 255, 0, 128)
	if c.A != 128 {
		t.Errorf("got A=%d, want 128", c.A)
	}
	if c.G != 255 {
		t.Errorf("got G=%d, want 255", c.G)
	}
}

// --- ColorFromFloats tests ---

func TestFromFloats(t *testing.T) {
	c := ColorFromFloats(0.5, 0.5, 0.5, 1.0)
	if c.R != 128 {
		t.Errorf("got R=%d, want 128", c.R)
	}
	if c.G != 128 {
		t.Errorf("got G=%d, want 128", c.G)
	}
	if c.B != 128 {
		t.Errorf("got B=%d, want 128", c.B)
	}
	if c.A != 255 {
		t.Errorf("got A=%d, want 255", c.A)
	}
}

// --- ToInts tests ---

func TestToInts_ColorWhite(t *testing.T) {
	r, g, b, a := ColorWhite.ToInts()
	if r != 255 || g != 255 || b != 255 || a != 255 {
		t.Errorf("got (%d,%d,%d,%d), want (255,255,255,255)", r, g, b, a)
	}
}

func TestToInts_Roundtrip(t *testing.T) {
	orig := ColorFromInts(123, 45, 67, 200)
	r, g, b, a := orig.ToInts()
	if r != 123 || g != 45 || b != 67 || a != 200 {
		t.Errorf("roundtrip failed: got (%d,%d,%d,%d), want (123,45,67,200)", r, g, b, a)
	}
}

// --- ToHex tests ---

func TestToHex_ColorRed(t *testing.T) {
	hex := ColorRed.ToHex()
	if hex != "#ff0000" {
		t.Errorf("got %q, want %q", hex, "#ff0000")
	}
}

func TestToHex_WithAlpha(t *testing.T) {
	c := ColorFromHex("#ff000080")
	hex := c.ToHex()
	if hex != "#ff000080" {
		t.Errorf("got %q, want %q", hex, "#ff000080")
	}
}

func TestToHex_ColorTransparent(t *testing.T) {
	hex := ColorTransparent.ToHex()
	if hex != "#00000000" {
		t.Errorf("got %q, want %q", hex, "#00000000")
	}
}

// --- Equals tests ---

func TestEquals_Same(t *testing.T) {
	if !ColorRed.Equals(ColorRed) {
		t.Error("ColorRed should equal ColorRed")
	}
}

func TestEquals_Different(t *testing.T) {
	if ColorRed.Equals(ColorBlue) {
		t.Error("ColorRed should not equal ColorBlue")
	}
}

func TestEquals_Exact(t *testing.T) {
	a := Color{26, 51, 77, 255}
	b := Color{26, 51, 77, 255}
	if !a.Equals(b) {
		t.Error("identical colors should be equal")
	}
}

// --- String test ---

func TestString(t *testing.T) {
	s := ColorRed.String()
	want := "rgba(255, 0, 0, 255)"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

// --- Blend tests ---

func TestBlend_Start(t *testing.T) {
	result := ColorBlack.Blend(ColorWhite, 0)
	assertColorEquals(t, result, ColorBlack)
}

func TestBlend_End(t *testing.T) {
	result := ColorBlack.Blend(ColorWhite, 1)
	assertColorEquals(t, result, ColorWhite)
}

func TestBlend_Midpoint(t *testing.T) {
	result := ColorBlack.Blend(ColorWhite, 0.5)
	// Should be approximately 128
	if result.R < 126 || result.R > 129 {
		t.Errorf("R: got %d, want ~128", result.R)
	}
	if result.G < 126 || result.G > 129 {
		t.Errorf("G: got %d, want ~128", result.G)
	}
	if result.B < 126 || result.B > 129 {
		t.Errorf("B: got %d, want ~128", result.B)
	}
}

// --- WithAlpha tests ---

func TestWithAlpha(t *testing.T) {
	c := ColorRed.WithAlpha(128)
	if c.A != 128 {
		t.Errorf("got A=%d, want 128", c.A)
	}
	if c.R != 255 {
		t.Errorf("got R=%d, want 255 (unchanged)", c.R)
	}
}

// --- Premultiply tests ---

func TestPremultiply(t *testing.T) {
	c := Color{255, 128, 64, 128}
	pm := c.Premultiply()
	// alpha = 128/255 ≈ 0.502
	// R: 255 * 0.502 ≈ 128
	// G: 128 * 0.502 ≈ 64
	// B: 64 * 0.502 ≈ 32
	if pm.R < 126 || pm.R > 129 {
		t.Errorf("R: got %d, want ~128", pm.R)
	}
	if pm.G < 62 || pm.G > 66 {
		t.Errorf("G: got %d, want ~64", pm.G)
	}
	if pm.B < 30 || pm.B > 34 {
		t.Errorf("B: got %d, want ~32", pm.B)
	}
	if pm.A != 128 {
		t.Errorf("A: got %d, want 128", pm.A)
	}
}

// --- IsTransparent tests ---

func TestIsTransparent_True(t *testing.T) {
	if !ColorTransparent.IsTransparent() {
		t.Error("ColorTransparent should be transparent")
	}
}

func TestIsTransparent_False(t *testing.T) {
	if ColorRed.IsTransparent() {
		t.Error("ColorRed should not be transparent")
	}
}

// --- Luminance tests ---

func TestLuminance_ColorWhite(t *testing.T) {
	l := ColorWhite.Luminance()
	assertFloat32Near(t, l, 1.0, 0.01)
}

func TestLuminance_ColorBlack(t *testing.T) {
	l := ColorBlack.Luminance()
	assertFloat32Near(t, l, 0.0, 1e-6)
}

func TestLuminance_ColorGreen_Dominant(t *testing.T) {
	// ColorGreen should have highest luminance contribution
	lr := ColorRed.Luminance()
	lg := ColorGreen.Luminance()
	lb := ColorBlue.Luminance()
	if lg <= lr || lg <= lb {
		t.Errorf("green luminance (%f) should be highest (r=%f, b=%f)", lg, lr, lb)
	}
}

// --- ParseColor tests ---

func TestParseColor_Color(t *testing.T) {
	result := ParseColor(ColorRed)
	assertColorEquals(t, result, ColorRed)
}

func TestParseColor_HexString(t *testing.T) {
	result := ParseColor(ColorString("#ff0000"))
	assertColorEquals(t, result, ColorRed)
}

func TestParseColor_NamedColor(t *testing.T) {
	result := ParseColor(ColorString("red"))
	assertColorEquals(t, result, ColorRed)
}

func TestParseColor_NamedColorCaseInsensitive(t *testing.T) {
	result := ParseColor(ColorString("RED"))
	assertColorEquals(t, result, ColorRed)
}

func TestParseColor_TransparentString(t *testing.T) {
	result := ParseColor(ColorString("transparent"))
	assertColorEquals(t, result, ColorTransparent)
}

func TestParseColor_EmptyString(t *testing.T) {
	result := ParseColor(ColorString(""))
	assertColorEquals(t, result, ColorTransparent)
}

// --- Named color map test ---

func TestNamedColors_AllPresent(t *testing.T) {
	names := []string{
		"black", "white", "red", "green", "blue", "yellow", "cyan", "magenta",
		"silver", "gray", "grey", "maroon", "olive", "lime", "aqua", "teal",
		"navy", "fuchsia", "purple", "orange", "transparent",
		"brightblack", "brightred", "brightgreen", "brightyellow",
		"brightblue", "brightmagenta", "brightcyan", "brightwhite",
	}
	for _, name := range names {
		if _, ok := namedColors[name]; !ok {
			t.Errorf("missing named color: %s", name)
		}
	}
}

// --- HSV conversion tests ---

func TestHSVToRGB_ColorRed(t *testing.T) {
	c := ColorHSVToRGB(0, 1, 1)
	assertColorEquals(t, c, ColorRed)
}

func TestHSVToRGB_ColorGreen(t *testing.T) {
	c := ColorHSVToRGB(120, 1, 1)
	assertColorEquals(t, c, ColorGreen)
}

func TestHSVToRGB_ColorBlue(t *testing.T) {
	c := ColorHSVToRGB(240, 1, 1)
	assertColorEquals(t, c, ColorBlue)
}

func TestHSVToRGB_ColorWhite(t *testing.T) {
	c := ColorHSVToRGB(0, 0, 1)
	assertColorEquals(t, c, ColorWhite)
}

func TestHSVToRGB_ColorBlack(t *testing.T) {
	c := ColorHSVToRGB(0, 0, 0)
	want := Color{0, 0, 0, 255}
	assertColorEquals(t, c, want)
}

func TestRGBToHSV_ColorRed(t *testing.T) {
	h, s, v := ColorRGBToHSV(ColorRed)
	assertFloat32Near(t, h, 0, 1)
	assertFloat32Near(t, s, 1, 0.01)
	assertFloat32Near(t, v, 1, 0.01)
}

func TestRGBToHSV_ColorGreen(t *testing.T) {
	h, s, v := ColorRGBToHSV(ColorGreen)
	assertFloat32Near(t, h, 120, 1)
	assertFloat32Near(t, s, 1, 0.01)
	assertFloat32Near(t, v, 1, 0.01)
}

func TestRGBToHSV_Roundtrip(t *testing.T) {
	orig := ColorFromInts(200, 100, 50, 255)
	h, s, v := ColorRGBToHSV(orig)
	back := ColorHSVToRGB(h, s, v)
	// Allow ±1 for rounding
	if absDiff(back.R, orig.R) > 1 {
		t.Errorf("R: got %d, want %d", back.R, orig.R)
	}
	if absDiff(back.G, orig.G) > 1 {
		t.Errorf("G: got %d, want %d", back.G, orig.G)
	}
	if absDiff(back.B, orig.B) > 1 {
		t.Errorf("B: got %d, want %d", back.B, orig.B)
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

// --- ColorHexToRGB / ColorRGBToHex convenience tests ---

func TestHexToRGB(t *testing.T) {
	c := ColorHexToRGB("#00ff00")
	assertColorEquals(t, c, ColorGreen)
}

func TestRGBToHex(t *testing.T) {
	hex := ColorRGBToHex(ColorBlue)
	if hex != "#0000ff" {
		t.Errorf("got %q, want %q", hex, "#0000ff")
	}
}

// --- Edge cases ---

func TestFromHex_AllZeros(t *testing.T) {
	c := ColorFromHex("#000000")
	want := Color{0, 0, 0, 255}
	assertColorEquals(t, c, want)
}

func TestBlend_OutOfRange(t *testing.T) {
	// Blend should still work even with t > 1 (extrapolation)
	result := ColorBlack.Blend(ColorWhite, 2.0)
	if math.IsNaN(float64(result.R)) {
		t.Error("blend should not produce NaN")
	}
}
