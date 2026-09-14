package style

import (
	"math"
	"testing"
)

func TestOKLCH_RoundTrip(t *testing.T) {
	// In-gamut sRGB colors survive an OKLCH round trip with only matrix and
	// integer-quantization drift.
	samples := []Color{
		ColorBlack, ColorWhite,
		ColorRed, ColorGreen, ColorBlue,
		ColorYellow, ColorCyan, ColorMagenta,
		ColorBrightRed, ColorBrightGreen, ColorBrightBlue,
		ColorFromInts(20, 20, 20, 255),
		ColorFromInts(249, 38, 114, 255),  // Monokai pink.
		ColorFromInts(102, 217, 239, 255), // Monokai cyan.
		ColorFromInts(166, 226, 46, 255),  // Monokai green.
		ColorFromInts(157, 124, 216, 255), // Muted purple.
		ColorFromInts(86, 182, 194, 255),  // Muted teal.
	}
	for _, orig := range samples {
		got := orig.ToOKLCH().ToColor()
		if absDiff(orig.R, got.R) > 1 ||
			absDiff(orig.G, got.G) > 1 ||
			absDiff(orig.B, got.B) > 1 {
			t.Errorf("round-trip drift: %v → OKLCH → %v", orig, got)
		}
	}
}

func TestOKLCH_BlackHasL0(t *testing.T) {
	o := ColorBlack.ToOKLCH()
	if o.L > 0.01 {
		t.Errorf("black should have L≈0, got %f", o.L)
	}
}

func TestOKLCH_WhiteHasL1(t *testing.T) {
	o := ColorWhite.ToOKLCH()
	if math.Abs(o.L-1.0) > 0.01 {
		t.Errorf("white should have L≈1, got %f", o.L)
	}
}

func TestOKLCH_PureGrayHasZeroChroma(t *testing.T) {
	o := ColorFromInts(128, 128, 128, 255).ToOKLCH()
	if o.C > 0.005 {
		t.Errorf("neutral gray should have C≈0, got %f", o.C)
	}
}

func TestLightenIncreasesLuminance(t *testing.T) {
	base := ColorFromInts(100, 80, 60, 255)
	lighter := base.Lighten(0.2)
	if lighter.Luminance() <= base.Luminance() {
		t.Errorf("Lighten should raise luminance: base=%.3f lighter=%.3f",
			base.Luminance(), lighter.Luminance())
	}
}

func TestDarkenDecreasesLuminance(t *testing.T) {
	base := ColorFromInts(180, 160, 140, 255)
	darker := base.Darken(0.2)
	if darker.Luminance() >= base.Luminance() {
		t.Errorf("Darken should lower luminance: base=%.3f darker=%.3f",
			base.Luminance(), darker.Luminance())
	}
}

func TestShiftHuePreservesLightnessApproximately(t *testing.T) {
	base := ColorFromInts(249, 38, 114, 255)
	rotated := base.ShiftHue(120)
	if absDiff64(base.ToOKLCH().L, rotated.ToOKLCH().L) > 0.005 {
		t.Errorf("ShiftHue should preserve L: base=%.3f rotated=%.3f",
			base.ToOKLCH().L, rotated.ToOKLCH().L)
	}
}

func TestScaleChroma(t *testing.T) {
	base := ColorFromInts(249, 38, 114, 255)
	desat := base.ScaleChroma(0.3)
	if desat.ToOKLCH().C >= base.ToOKLCH().C {
		t.Errorf("ScaleChroma(<1) should reduce chroma")
	}
	if desat.ToOKLCH().C <= 0 {
		t.Errorf("ScaleChroma(0.3) should not fully desaturate")
	}
}

func TestMixPerceptualEndpoints(t *testing.T) {
	a := ColorBrightBlue
	b := ColorBrightRed
	if mid := MixPerceptual(a, b, 0); !mid.Equals(a) {
		// The color-space round trip permits one value of channel drift.
		if !near(mid, a, 2) {
			t.Errorf("MixPerceptual(a, b, 0) should be a, got %v vs %v", mid, a)
		}
	}
	if mid := MixPerceptual(a, b, 1); !near(mid, b, 2) {
		t.Errorf("MixPerceptual(a, b, 1) should be b, got %v vs %v", mid, b)
	}
}

func TestGenerateScale_Monotonic_Dark(t *testing.T) {
	// A dark ramp's OKLCH lightness rises monotonically from low-contrast
	// surfaces to its high-contrast endpoint.
	scale := GenerateScale(ColorBrightBlue, true)
	for i := 1; i < 12; i++ {
		prev := scale[i-1].ToOKLCH().L
		cur := scale[i].ToOKLCH().L
		if cur <= prev {
			t.Errorf("dark scale[%d].L (%f) should exceed scale[%d].L (%f)",
				i, cur, i-1, prev)
		}
	}
}

func TestGenerateScale_Monotonic_Light(t *testing.T) {
	// A light ramp's OKLCH lightness falls monotonically toward dark text.
	scale := GenerateScale(ColorBrightBlue, false)
	for i := 1; i < 12; i++ {
		prev := scale[i-1].ToOKLCH().L
		cur := scale[i].ToOKLCH().L
		if cur >= prev {
			t.Errorf("light scale[%d].L (%f) should be less than scale[%d].L (%f)",
				i, cur, i-1, prev)
		}
	}
}

func TestGenerateScale_PreservesHue(t *testing.T) {
	// Chromatic steps retain seed hue unless gamut fitting removes all chroma.
	seed := ColorFromInts(157, 124, 216, 255) // Muted purple.
	seedHue := seed.ToOKLCH().H
	scale := GenerateScale(seed, true)
	for i, c := range scale {
		o := c.ToOKLCH()
		if o.C < 0.01 {
			continue // Hue is undefined for a fully desaturated step.
		}
		// Gamut fitting permits approximately two degrees of hue drift.
		if hueDist(o.H, seedHue) > 3.0 {
			t.Errorf("scale[%d] hue drifted: %.1f vs seed %.1f", i, o.H, seedHue)
		}
	}
}

func TestGenerateNeutralScale_WithInk_BridgesBgToInk(t *testing.T) {
	// Explicit ink anchors the high-contrast end of the neutral ramp.
	bgSeed := ColorFromInts(20, 20, 20, 255)
	ink := ColorFromInts(240, 240, 240, 255)
	scale := GenerateNeutralScale(bgSeed, true, ink)

	// Step 11 contains 98.4 percent ink and remains close to it.
	if absDiff(scale[11].R, ink.R) > 5 {
		t.Errorf("scale[11] should approach ink: got %v want ≈%v", scale[11], ink)
	}
	// Step zero is the derived background and is darker than step 11.
	if scale[0].Luminance() >= scale[11].Luminance() {
		t.Errorf("scale[0] should be darker than scale[11] in a dark neutral ramp")
	}
}

func TestGenerateNeutralScale_WithoutInk_HasLowChroma(t *testing.T) {
	// A dark seed-only neutral ramp caps chroma at 0.068 before conversion.
	scale := GenerateNeutralScale(ColorBrightBlue, true, Color{})
	for i, c := range scale {
		if c.ToOKLCH().C > 0.09 {
			t.Errorf("seed-only neutral[%d] chroma too high: %.3f", i, c.ToOKLCH().C)
		}
	}
}

func absDiff64(a, b float64) float64 {
	if a < b {
		return b - a
	}
	return a - b
}

func hueDist(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func near(a, b Color, tol uint8) bool {
	return absDiff(a.R, b.R) <= tol &&
		absDiff(a.G, b.G) <= tol &&
		absDiff(a.B, b.B) <= tol
}
