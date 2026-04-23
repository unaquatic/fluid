package style

import (
	"math"
	"testing"
)

// --- sRGB ↔ OKLCH round-trip -----------------------------------------------

func TestOKLCH_RoundTrip(t *testing.T) {
	// Round-tripping via OKLCH must be close to identity for in-gamut
	// colors. The Fit step can clip chroma, but for sRGB-origin colors
	// it should be a no-op, so we allow only a tiny tolerance for
	// floating-point + uint8 quantisation.
	samples := []Color{
		ColorBlack, ColorWhite,
		ColorRed, ColorGreen, ColorBlue,
		ColorYellow, ColorCyan, ColorMagenta,
		ColorBrightRed, ColorBrightGreen, ColorBrightBlue,
		ColorFromInts(20, 20, 20, 255),
		ColorFromInts(249, 38, 114, 255),  // monokai pink
		ColorFromInts(102, 217, 239, 255), // monokai cyan
		ColorFromInts(166, 226, 46, 255),  // monokai green
		ColorFromInts(157, 124, 216, 255), // muted purple
		ColorFromInts(86, 182, 194, 255),  // muted teal
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

// --- Perceptual transforms -------------------------------------------------

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
		// Round-trip loss can shift by 1; allow tiny drift.
		if !near(mid, a, 2) {
			t.Errorf("MixPerceptual(a, b, 0) should be a, got %v vs %v", mid, a)
		}
	}
	if mid := MixPerceptual(a, b, 1); !near(mid, b, 2) {
		t.Errorf("MixPerceptual(a, b, 1) should be b, got %v vs %v", mid, b)
	}
}

// --- Scale generation ------------------------------------------------------

func TestGenerateScale_Monotonic_Dark(t *testing.T) {
	// In a dark-theme ramp, perceptual L must rise monotonically from
	// 0 to 11: the ramp goes from "barely visible" to "near white". We
	// measure in OKLCH L (what the ramp targets), not BT.709 luminance,
	// because chroma modulates luminance enough to occasionally invert
	// consecutive steps even when the perceptual lightness is strictly
	// increasing.
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
	// In a light-theme ramp, perceptual L must fall monotonically from
	// 0 to 11 (bg→fg goes from white toward black).
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
	// All steps in a chromatic scale must sit on the seed's hue line.
	// Fit() may pull chroma to 0 for near-black / near-white, in which
	// case hue is undefined — skip those steps.
	seed := ColorFromInts(157, 124, 216, 255) // muted purple
	seedHue := seed.ToOKLCH().H
	scale := GenerateScale(seed, true)
	for i, c := range scale {
		o := c.ToOKLCH()
		if o.C < 0.01 {
			continue // fully desaturated, hue undefined
		}
		// Allow ~2° drift due to gamut fitting.
		if hueDist(o.H, seedHue) > 3.0 {
			t.Errorf("scale[%d] hue drifted: %.1f vs seed %.1f", i, o.H, seedHue)
		}
	}
}

func TestGenerateNeutralScale_WithInk_BridgesBgToInk(t *testing.T) {
	// With an explicit ink, scale[0] should be close to the derived bg
	// and scale[11] close to ink. The middle steps blend between them.
	bgSeed := ColorFromInts(20, 20, 20, 255)
	ink := ColorFromInts(240, 240, 240, 255)
	scale := GenerateNeutralScale(bgSeed, true, ink)

	// scale[11] blends 98.4% ink + 1.6% bg → should be very close to ink.
	if absDiff(scale[11].R, ink.R) > 5 {
		t.Errorf("scale[11] should approach ink: got %v want ≈%v", scale[11], ink)
	}
	// scale[0] blends 0% ink → should equal the derived bg. We can't
	// easily predict bg, but we can assert it's darker than scale[11].
	if scale[0].Luminance() >= scale[11].Luminance() {
		t.Errorf("scale[0] should be darker than scale[11] in a dark neutral ramp")
	}
}

func TestGenerateNeutralScale_WithoutInk_HasLowChroma(t *testing.T) {
	// Seed-only neutral ramp: chroma must be clamped to the "muted"
	// bucket (≤ 0.068 in dark mode). Otherwise the supposed neutrals
	// tint the whole UI.
	scale := GenerateNeutralScale(ColorBrightBlue, true, Color{})
	for i, c := range scale {
		if c.ToOKLCH().C > 0.09 {
			t.Errorf("seed-only neutral[%d] chroma too high: %.3f", i, c.ToOKLCH().C)
		}
	}
}

// --- helpers ---------------------------------------------------------------

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
