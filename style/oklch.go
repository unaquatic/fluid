package style

import "math"

// OKLCH represents a color in the perceptually uniform cylindrical form of
// Oklab.
//
// L is lightness in [0, 1], C is nonnegative chroma, and H is hue in degrees.
// The maximum in-gamut chroma depends on lightness and hue.
//
// See https://bottosson.github.io/posts/oklab/.
type OKLCH struct {
	L, C, H float64
}

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

// ToOKLCH converts c from sRGB to OKLCH and ignores alpha.
func (c Color) ToOKLCH() OKLCH {
	L, a, b := c.ToOklab()

	// Convert Cartesian Oklab to polar chroma and hue.
	C := math.Sqrt(a*a + b*b)
	H := math.Atan2(b, a) * (180.0 / math.Pi)
	if H < 0 {
		H += 360
	}
	return OKLCH{L: L, C: C, H: H}
}

// ToOklab converts c from sRGB to Cartesian Oklab and ignores alpha. Euclidean
// distance between Oklab coordinates approximates perceived color difference.
func (c Color) ToOklab() (L, a, b float64) {
	// Convert normalized sRGB channels to linear RGB.
	lr := srgbToLinear(float64(c.R) / 255.0)
	lg := srgbToLinear(float64(c.G) / 255.0)
	lb := srgbToLinear(float64(c.B) / 255.0)

	// Transform linear RGB into Oklab's LMS intermediate space.
	l_ := 0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb
	m_ := 0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb
	s_ := 0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb

	l := math.Cbrt(l_)
	m := math.Cbrt(m_)
	s := math.Cbrt(s_)

	// Transform cube-root LMS into Cartesian Oklab.
	L = 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	a = 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	b = 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return L, a, b
}

// linearRGB returns o as unclamped linear RGB. Use ToColor when the result must
// fit within sRGB.
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

// inGamut reports whether all three linear RGB channels lie in [0, 1].
func inGamut(lr, lg, lb float64) bool {
	return lr >= 0 && lr <= 1 && lg >= 0 && lg <= 1 && lb >= 0 && lb <= 1
}

// gamutEpsilon tolerates small matrix round-trip errors at the sRGB boundary so
// an in-gamut source color is not needlessly desaturated.
const gamutEpsilon = 1e-5

// Fit normalizes o and reduces chroma until it fits in sRGB. It preserves hue
// and clamped lightness while fitting.
func (o OKLCH) Fit() OKLCH {
	base := OKLCH{
		L: clamp64(o.L, 0, 1),
		C: math.Max(0, o.C),
		H: hueNorm(o.H),
	}
	if lr, lg, lb := base.linearRGB(); inGamutEps(lr, lg, lb, gamutEpsilon) {
		return base
	}
	// Reduce chroma geometrically for at most 24 attempts, then fall back to gray.
	c := base.C
	for range 24 {
		c *= 0.9
		next := OKLCH{L: base.L, C: c, H: base.H}
		lr, lg, lb := next.linearRGB()
		if inGamut(lr, lg, lb) {
			return next
		}
	}
	return OKLCH{L: base.L, C: 0, H: base.H}
}

// inGamutEps applies a floating-point tolerance to the sRGB gamut boundary.
func inGamutEps(lr, lg, lb, eps float64) bool {
	return lr >= -eps && lr <= 1+eps &&
		lg >= -eps && lg <= 1+eps &&
		lb >= -eps && lb <= 1+eps
}

// ToColor converts o to an opaque sRGB color after fitting it to the gamut.
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

// Lighten adds amount to OKLCH lightness and returns an opaque sRGB color.
// Positive amounts brighten and negative amounts darken.
func (c Color) Lighten(amount float64) Color {
	o := c.ToOKLCH()
	o.L = clamp64(o.L+amount, 0, 1)
	return o.ToColor()
}

// Darken subtracts amount from OKLCH lightness.
func (c Color) Darken(amount float64) Color { return c.Lighten(-amount) }

// ShiftHue rotates c's OKLCH hue by delta degrees and returns an opaque color.
func (c Color) ShiftHue(delta float64) Color {
	o := c.ToOKLCH()
	o.H = hueNorm(o.H + delta)
	return o.ToColor()
}

// ScaleChroma multiplies c's OKLCH chroma by factor. Negative results clamp to
// zero.
func (c Color) ScaleChroma(factor float64) Color {
	o := c.ToOKLCH()
	o.C *= factor
	if o.C < 0 {
		o.C = 0
	}
	return o.ToColor()
}

// MixPerceptual interpolates a and b in OKLCH, following the shorter hue arc.
// t=0 selects a and t=1 selects b; output alpha is always opaque.
func MixPerceptual(a, b Color, t float64) Color {
	oa := a.ToOKLCH()
	ob := b.ToOKLCH()
	// Normalize the hue delta to [-180, 180).
	delta := math.Mod(math.Mod(ob.H-oa.H, 360)+540, 360) - 180
	return OKLCH{
		L: oa.L + (ob.L-oa.L)*t,
		C: oa.C + (ob.C-oa.C)*t,
		H: oa.H + delta*t,
	}.ToColor()
}

// GenerateScale returns a 12-step tonal ramp in seed's hue family. The dark
// flag selects a dark or light theme progression. Indices have these roles:
//
//	0..2   low-contrast background tints
//	3..5   fills and subtle dividers
//	6..7   borders and weak text
//	8..10  prominent seed-family colors
//	11     highest-contrast endpoint
//
// The lightness and chroma curves are fixed; hue remains constant.
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
	for i := range 12 {
		scale[i] = OKLCH{
			L: lightSteps[i],
			C: base.C * chromaMul[i],
			H: base.H,
		}.ToColor()
	}
	return scale
}

// GenerateNeutralScale returns a 12-step ramp derived from seed.
//
// A transparent ink produces a muted, seed-hued neutral ramp. An opaque ink
// produces a perceptual interpolation from a seed-derived background to ink.
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

		// Derive the background by moving the seed toward black or white.
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

	// Without ink, retain seed's hue but cap chroma for neutral surfaces.
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
	for i := range 12 {
		scale[i] = OKLCH{L: lightSteps[i], C: neutralC, H: base.H}.ToColor()
	}
	return scale
}

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
