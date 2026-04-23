package style

import "math"

// OKLCH is a perceptually-uniform color space derived from Oklab.
//
//   - L (lightness) is in [0, 1]. 0 is black, 1 is white; intermediate
//     values correspond to perceived brightness, not linear intensity.
//   - C (chroma) is in [0, ~0.4]. 0 is fully desaturated (gray); higher
//     values are more saturated. The practical upper bound inside the
//     sRGB gamut depends on L and H.
//   - H (hue) is in degrees [0, 360). 0 is pinkish red, 90 is yellow,
//     180 is cyan-green, 270 is blue.
//
// OKLCH is useful for theme generation because lightness/chroma/hue can
// be adjusted independently without perceptual coupling — e.g. shifting
// L builds a tonal scale that stays within the same hue family.
//
// Reference: https://bottosson.github.io/posts/oklab/
type OKLCH struct {
	L, C, H float64
}

// --- sRGB ↔ linear RGB -----------------------------------------------------

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSrgb(c float64) float64 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

// --- Color ↔ OKLCH ---------------------------------------------------------

// ToOKLCH converts this color to OKLCH, ignoring alpha.
func (c Color) ToOKLCH() OKLCH {
	// sRGB [0, 255] → linear [0, 1]
	lr := srgbToLinear(float64(c.R) / 255.0)
	lg := srgbToLinear(float64(c.G) / 255.0)
	lb := srgbToLinear(float64(c.B) / 255.0)

	// Linear RGB → LMS (Oklab matrix).
	l_ := 0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb
	m_ := 0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb
	s_ := 0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb

	l := math.Cbrt(l_)
	m := math.Cbrt(m_)
	s := math.Cbrt(s_)

	// LMS → Oklab.
	L := 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	a := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	b := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s

	// Oklab → OKLCH (polar form).
	C := math.Sqrt(a*a + b*b)
	H := math.Atan2(b, a) * (180.0 / math.Pi)
	if H < 0 {
		H += 360
	}
	return OKLCH{L: L, C: C, H: H}
}

// linearRGB returns the (unclamped) linear-RGB triple for this OKLCH.
// Callers that need to stay in-gamut should use ToColor, which clips
// chroma to fit.
func (o OKLCH) linearRGB() (lr, lg, lb float64) {
	rad := o.H * (math.Pi / 180.0)
	a := o.C * math.Cos(rad)
	b := o.C * math.Sin(rad)

	l := o.L + 0.3963377774*a + 0.2158037573*b
	m := o.L - 0.1055613458*a - 0.0638541728*b
	s := o.L - 0.0894841775*a - 1.2914855480*b

	l3 := l * l * l
	m3 := m * m * m
	s3 := s * s * s

	lr = 4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3
	lg = -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3
	lb = -0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3
	return
}

// inGamut reports whether the given linear-RGB triple lies within [0, 1].
func inGamut(lr, lg, lb float64) bool {
	return lr >= 0 && lr <= 1 && lg >= 0 && lg <= 1 && lb >= 0 && lb <= 1
}

// gamutEpsilon is the tolerance used by Fit before triggering chroma
// reduction. Round-tripping a saturated sRGB color (e.g. #ff0000)
// produces values that overshoot [0, 1] by ~1e-7 due to FP error in
// the Oklab matrices; treating those as "in gamut" avoids desaturating
// on a no-op round trip. The downstream ToColor path still clamps each
// channel to [0, 255] via clampU8F, so the emitted color is exactly
// the input.
const gamutEpsilon = 1e-5

// Fit clamps L into [0, 1], nudges C to be non-negative, normalises H,
// and iteratively reduces chroma until the color is representable in
// sRGB. This is the "fit" strategy used by the CSS Colors Module Level 4
// when converting wide-gamut OKLCH to sRGB: preserve hue, preserve
// lightness, sacrifice chroma.
func (o OKLCH) Fit() OKLCH {
	base := OKLCH{
		L: clamp64(o.L, 0, 1),
		C: math.Max(0, o.C),
		H: hueNorm(o.H),
	}
	if lr, lg, lb := base.linearRGB(); inGamutEps(lr, lg, lb, gamutEpsilon) {
		return base
	}
	// Geometric falloff: multiply chroma by 0.9 per iteration until the
	// color fits. 24 iterations reduces chroma to ~8% of the original,
	// which is ample for anything sane.
	c := base.C
	for i := 0; i < 24; i++ {
		c *= 0.9
		next := OKLCH{L: base.L, C: c, H: base.H}
		lr, lg, lb := next.linearRGB()
		if inGamut(lr, lg, lb) {
			return next
		}
	}
	return OKLCH{L: base.L, C: 0, H: base.H}
}

// inGamutEps is inGamut with a small tolerance for floating-point
// overshoot at the sRGB gamut boundary. The downstream ToColor clamps
// channels to [0, 255] regardless, so treating (-1e-7, 1+1e-7) as
// "in gamut" is safe and avoids spurious chroma reduction.
func inGamutEps(lr, lg, lb, eps float64) bool {
	return lr >= -eps && lr <= 1+eps &&
		lg >= -eps && lg <= 1+eps &&
		lb >= -eps && lb <= 1+eps
}

// ToColor converts OKLCH back to sRGB, reducing chroma as needed to
// stay within gamut. Alpha is always 255; callers needing translucency
// should wrap with WithAlpha.
func (o OKLCH) ToColor() Color {
	fit := o.Fit()
	lr, lg, lb := fit.linearRGB()
	return Color{
		R: clampU8F(linearToSrgb(lr) * 255),
		G: clampU8F(linearToSrgb(lg) * 255),
		B: clampU8F(linearToSrgb(lb) * 255),
		A: 255,
	}
}

// --- Perceptual transforms -------------------------------------------------

// Lighten shifts L by amount (clamped to [0, 1]) and returns the result
// as an sRGB Color. Positive amounts brighten; negative amounts darken.
func (c Color) Lighten(amount float64) Color {
	o := c.ToOKLCH()
	o.L = clamp64(o.L+amount, 0, 1)
	return o.ToColor()
}

// Darken is the inverse of Lighten.
func (c Color) Darken(amount float64) Color { return c.Lighten(-amount) }

// ShiftHue rotates H by delta degrees and returns the result.
func (c Color) ShiftHue(delta float64) Color {
	o := c.ToOKLCH()
	o.H = hueNorm(o.H + delta)
	return o.ToColor()
}

// ScaleChroma multiplies C by factor (saturate if >1, desaturate if <1).
func (c Color) ScaleChroma(factor float64) Color {
	o := c.ToOKLCH()
	o.C *= factor
	if o.C < 0 {
		o.C = 0
	}
	return o.ToColor()
}

// MixPerceptual blends two colors in OKLCH with t=0 → a, t=1 → b. Hue
// takes the shorter arc (i.e. blue↔red goes via magenta, not via green).
// This is the perceptual analog of Color.Blend, which interpolates in
// sRGB and can produce muddy midpoints.
func MixPerceptual(a, b Color, t float64) Color {
	oa := a.ToOKLCH()
	ob := b.ToOKLCH()
	// Shortest-arc hue interpolation.
	delta := math.Mod(math.Mod(ob.H-oa.H, 360)+540, 360) - 180
	return OKLCH{
		L: oa.L + (ob.L-oa.L)*t,
		C: oa.C + (ob.C-oa.C)*t,
		H: oa.H + delta*t,
	}.ToColor()
}

// --- Scale generation ------------------------------------------------------

// GenerateScale builds a 12-step tonal ramp keyed on seed, tuned for
// either dark or light themes. The ramp is indexed 0..11:
//
//	0..2   near-bg tints (panel backgrounds, subtle fills)
//	3..5   weak fills / low-contrast dividers
//	6..7   borders, weak text
//	8..10  the seed's own family (primary use)
//	11     highest-contrast — approaches the theme's fg
//
// Lightness and chroma curves are hand-tuned to produce perceptually
// even steps across the ramp — the low end stays close to bg, the
// high end approaches fg, and the middle has enough chroma for
// distinct selection/focus tints. Hue is kept constant.
func GenerateScale(seed Color, dark bool) [12]Color {
	base := seed.ToOKLCH()

	var lightSteps [12]float64
	var chromaMul [12]float64

	if dark {
		lightSteps = [12]float64{
			0.118,
			0.138,
			0.167,
			0.202,
			0.246,
			0.304,
			0.378,
			0.468,
			clamp64(base.L*0.825, 0.53, 0.705),
			clamp64(base.L*0.89, 0.61, 0.79),
			clamp64(base.L+0.033, 0.868, 0.943),
			0.984,
		}
		chromaMul = [12]float64{0.52, 0.68, 0.86, 1.02, 1.14, 1.24, 1.36, 1.48, 1.56, 1.64, 1.62, 1.15}
	} else {
		lightSteps = [12]float64{
			0.993, 0.983, 0.962, 0.936, 0.906, 0.866, 0.811, 0.740,
			base.L,
			math.Max(0, base.L-0.036),
			0.49,
			0.27,
		}
		chromaMul = [12]float64{0.12, 0.24, 0.46, 0.68, 0.84, 0.98, 1.08, 1.16, 1.22, 1.26, 1.18, 0.98}
	}

	var scale [12]Color
	for i := 0; i < 12; i++ {
		scale[i] = OKLCH{
			L: lightSteps[i],
			C: base.C * chromaMul[i],
			H: base.H,
		}.ToColor()
	}
	return scale
}

// GenerateNeutralScale builds a 12-step ramp bridging bg and ink.
//
// When ink is zero, the ramp is derived from seed alone with a muted
// chroma — useful for themes that want a single hue to inform the
// whole neutral axis without committing to a specific fg color.
//
// When ink is non-zero, the ramp interpolates perceptually between a
// seed-derived bg and ink. This produces the cleanest canvas-to-
// foreground gradient and is what authors typically want when they
// care about the exact body text color.
func GenerateNeutralScale(seed Color, dark bool, ink Color) [12]Color {
	if ink.A != 0 {
		base := seed.ToOKLCH()

		lift := func(tone float64) Color {
			return OKLCH{
				L: base.L + (1-base.L)*tone,
				C: base.C * math.Max(0, 1-tone),
				H: base.H,
			}.ToColor()
		}
		sink := func(tone float64) Color {
			chromaShrink := tone * 0.3
			if dark {
				chromaShrink = tone * 0.12
			}
			return OKLCH{
				L: base.L * (1 - tone),
				C: base.C * math.Max(0, 1-chromaShrink),
				H: base.H,
			}.ToColor()
		}

		// Derive a bg color from the seed. Dark: sink toward black with
		// a chroma/lightness-dependent amount. Light: lift toward white.
		var bg Color
		if dark {
			tone := clamp64(0.19+math.Max(0, base.L-0.12)*0.33+base.C*1.95, 0.17, 0.27)
			bg = sink(tone)
		} else if base.L < 0.82 {
			bg = lift(0.86)
		} else {
			tone := clamp64(0.1+base.C*3.2+math.Max(0, 0.95-base.L)*0.35, 0.1, 0.28)
			bg = lift(tone)
		}

		var steps [12]float64
		if dark {
			steps = [12]float64{0, 0.018, 0.039, 0.064, 0.097, 0.143, 0.212, 0.310, 0.460, 0.649, 0.845, 0.984}
		} else {
			steps = [12]float64{0, 0.022, 0.042, 0.068, 0.102, 0.146, 0.208, 0.296, 0.432, 0.610, 0.810, 0.965}
		}

		var out [12]Color
		for i, t := range steps {
			out[i] = MixPerceptual(bg, ink, t)
		}
		return out
	}

	// Seed-only neutral: muted chroma ramp keyed on seed's hue.
	base := seed.ToOKLCH()
	maxChroma := 0.04
	if dark {
		maxChroma = 0.068
	}
	neutralC := math.Min(base.C, maxChroma)

	var lightSteps [12]float64
	if dark {
		lightSteps = [12]float64{
			0.138, 0.156, 0.178, 0.202, 0.232, 0.272, 0.326, 0.404,
			clamp64(base.L*0.83, 0.43, 0.55),
			0.596, 0.719, 0.956,
		}
	} else {
		lightSteps = [12]float64{
			0.991, 0.979, 0.964, 0.946, 0.931, 0.913, 0.891, 0.830,
			base.L,
			0.617, 0.542, 0.205,
		}
	}

	var scale [12]Color
	for i := 0; i < 12; i++ {
		scale[i] = OKLCH{L: lightSteps[i], C: neutralC, H: base.H}.ToColor()
	}
	return scale
}

// --- Small utilities -------------------------------------------------------

func clamp64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func hueNorm(h float64) float64 {
	v := math.Mod(h, 360)
	if v < 0 {
		v += 360
	}
	return v
}

func clampU8F(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}
