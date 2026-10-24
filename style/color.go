package style

import (
	"fmt"
	"math"
	"strings"
)

// Color stores red, green, blue, and alpha channels in the range [0, 255].
type Color struct {
	// R, G, and B are the red, green, and blue channels, and A the opacity,
	// from 0, transparent, to 255, opaque.
	R, G, B, A uint8
}

// ColorFromHex parses RGB, RGBA, RRGGBB, or RRGGBBAA hexadecimal notation with
// an optional leading '#'. Unsupported lengths return ColorTransparent.
func ColorFromHex(hex string) Color {
	hex = strings.TrimPrefix(hex, "#")

	switch len(hex) {
	case 3: // One hexadecimal digit per RGB channel.
		r := hexDigit(hex[0])
		g := hexDigit(hex[1])
		b := hexDigit(hex[2])
		return Color{
			R: r * 17,
			G: g * 17,
			B: b * 17,
			A: 255,
		}
	case 4: // One hexadecimal digit per RGBA channel.
		r := hexDigit(hex[0])
		g := hexDigit(hex[1])
		b := hexDigit(hex[2])
		a := hexDigit(hex[3])
		return Color{
			R: r * 17,
			G: g * 17,
			B: b * 17,
			A: a * 17,
		}
	case 6: // Two hexadecimal digits per RGB channel.
		r := hexByte(hex[0], hex[1])
		g := hexByte(hex[2], hex[3])
		b := hexByte(hex[4], hex[5])
		return Color{
			R: r,
			G: g,
			B: b,
			A: 255,
		}
	case 8: // Two hexadecimal digits per RGBA channel.
		r := hexByte(hex[0], hex[1])
		g := hexByte(hex[2], hex[3])
		b := hexByte(hex[4], hex[5])
		a := hexByte(hex[6], hex[7])
		return Color{
			R: r,
			G: g,
			B: b,
			A: a,
		}
	default:
		return ColorTransparent
	}
}

// ColorFromInts returns a color from channel values in the range [0, 255].
func ColorFromInts(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// ColorFromFloats returns a color from channel values in the range [0, 1].
func ColorFromFloats(r, g, b, a float64) Color {
	return Color{
		R: uint8(math.Round(float64(r) * 255)),
		G: uint8(math.Round(float64(g) * 255)),
		B: uint8(math.Round(float64(b) * 255)),
		A: uint8(math.Round(float64(a) * 255)),
	}
}

// ToFloats returns c's channels normalized to the range [0, 1].
func (c Color) ToFloats() (r, g, b, a float64) {
	return float64(c.R) / 255.0, float64(c.G) / 255.0, float64(c.B) / 255.0, float64(c.A) / 255.0
}

// ToHex returns c as lowercase #RRGGBB or #RRGGBBAA notation. It omits alpha
// when c is fully opaque.
func (c Color) ToHex() string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// Equals reports whether c and other have identical channels.
func (c Color) Equals(other Color) bool {
	return c == other
}

// String returns c as "rgba(r, g, b, a)" using integer channel values.
func (c Color) String() string {
	return fmt.Sprintf("rgba(%d, %d, %d, %d)", c.R, c.G, c.B, c.A)
}

// Blend linearly interpolates from c at t=0 to other at t=1. Values outside
// that range extrapolate and clamp each resulting channel.
func (c Color) Blend(other Color, t float64) Color {
	r := float64(c.R) + (float64(other.R)-float64(c.R))*t
	g := float64(c.G) + (float64(other.G)-float64(c.G))*t
	b := float64(c.B) + (float64(other.B)-float64(c.B))*t
	a := float64(c.A) + (float64(other.A)-float64(c.A))*t
	return Color{
		R: clampU8(r),
		G: clampU8(g),
		B: clampU8(b),
		A: clampU8(a),
	}
}

// ColorBlendPerceptual composites overlay over text using a perceptually
// adjusted overlay alpha.
func ColorBlendPerceptual(overlay, text Color) Color {
	if overlay.A == 255 {
		return overlay
	}

	if text.A == 0 {
		alpha := float64(overlay.A) / 255.0
		r := float64(overlay.R) * alpha
		g := float64(overlay.G) * alpha
		b := float64(overlay.B) * alpha
		if r < 0.5 && g < 0.5 && b < 0.5 {
			return Color{R: 0, G: 0, B: 0, A: 0}
		}
		return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: overlay.A}
	}

	alpha := float64(overlay.A) / 255.0
	var perceptualAlpha float64

	if alpha > 0.8 {
		normalizedHighAlpha := (alpha - 0.8) * 5.0
		curvedHighAlpha := math.Pow(normalizedHighAlpha, 0.2)
		perceptualAlpha = 0.8 + curvedHighAlpha*0.2
	} else {
		perceptualAlpha = math.Pow(alpha, 0.9)
	}

	oneMinusAlpha := 1.0 - perceptualAlpha
	blendedR := float64(overlay.R)*perceptualAlpha + float64(text.R)*oneMinusAlpha
	blendedG := float64(overlay.G)*perceptualAlpha + float64(text.G)*oneMinusAlpha
	blendedB := float64(overlay.B)*perceptualAlpha + float64(text.B)*oneMinusAlpha

	resultAlpha := alpha + float64(text.A)/255.0*(1.0-alpha)

	return Color{R: clampU8(blendedR), G: clampU8(blendedG), B: clampU8(blendedB), A: clampU8(resultAlpha * 255)}
}

// WithAlpha returns a copy of c with alpha set to a.
func (c Color) WithAlpha(a uint8) Color {
	c.A = a
	return c
}

// Premultiply returns c with each RGB channel multiplied by its alpha.
func (c Color) Premultiply() Color {
	a := float64(c.A) / 255.0
	return Color{
		R: uint8(float64(c.R) * a),
		G: uint8(float64(c.G) * a),
		B: uint8(float64(c.B) * a),
		A: c.A,
	}
}

// IsTransparent reports whether c has zero alpha.
func (c Color) IsTransparent() bool {
	return c.A == 0
}

// Luminance returns c's BT.709 luma in the range [0, 1].
func (c Color) Luminance() float64 {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// DistanceSq returns the squared Euclidean RGB distance from c to other,
// ignoring alpha.
func (c Color) DistanceSq(other Color) float64 {
	dr := float64(int(c.R) - int(other.R))
	dg := float64(int(c.G) - int(other.G))
	db := float64(int(c.B) - int(other.B))
	return dr*dr + dg*dg + db*db
}

// ColorAverage returns the component-wise mean of colors. An empty input
// returns ColorTransparent.
func ColorAverage(colors []Color) Color {
	if len(colors) == 0 {
		return ColorTransparent
	}
	var r, g, b, a int
	for _, c := range colors {
		r += int(c.R)
		g += int(c.G)
		b += int(c.B)
		a += int(c.A)
	}
	n := len(colors)
	return Color{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n)}
}

// Common named colors.
var (
	ColorTransparent = Color{0, 0, 0, 0}
	ColorBlack       = Color{0, 0, 0, 255}
	ColorWhite       = Color{255, 255, 255, 255}
	ColorRed         = Color{255, 0, 0, 255}
	ColorGreen       = Color{0, 255, 0, 255}
	ColorBlue        = Color{0, 0, 255, 255}
	ColorYellow      = Color{255, 255, 0, 255}
	ColorCyan        = Color{0, 255, 255, 255}
	ColorMagenta     = Color{255, 0, 255, 255}

	ColorSilver = Color{192, 192, 192, 255}
	ColorGray   = Color{128, 128, 128, 255}
	ColorMaroon = Color{128, 0, 0, 255}
	ColorOlive  = Color{128, 128, 0, 255}
	ColorTeal   = Color{0, 128, 128, 255}
	ColorNavy   = Color{0, 0, 128, 255}
	ColorPurple = Color{128, 0, 128, 255}
	ColorOrange = Color{255, 165, 0, 255}

	ColorBrightRed     = Color{255, 85, 85, 255}
	ColorBrightGreen   = Color{85, 255, 85, 255}
	ColorBrightYellow  = Color{255, 255, 85, 255}
	ColorBrightBlue    = Color{85, 85, 255, 255}
	ColorBrightMagenta = Color{255, 85, 255, 255}
	ColorBrightCyan    = Color{85, 255, 255, 255}
)

// ColorInput is either a Color or a ColorString accepted by ParseColor.
type ColorInput interface {
	colorInput()
}

// ColorString is a color in hexadecimal notation beginning with '#'.
type ColorString string

func (ColorString) colorInput() {}
func (Color) colorInput()       {}

// ParseColor resolves input to a color. A nil input, or a string that is not
// hexadecimal notation beginning with '#', returns ColorTransparent.
func ParseColor(input ColorInput) Color {
	switch v := input.(type) {
	case Color:
		return v
	case ColorString:
		return parseColorString(string(v))
	default:
		return ColorTransparent
	}
}

func parseColorString(s string) Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return ColorTransparent
	}
	if strings.HasPrefix(s, "#") {
		return ColorFromHex(s)
	}
	return ColorTransparent
}

// ColorHSVToRGB converts hue in degrees and normalized saturation and value to
// an opaque RGB color.
func ColorHSVToRGB(h, s, v float64) Color {
	if s == 0 {
		val := uint8(v * 255)
		return Color{val, val, val, 255}
	}
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	h /= 60
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	default:
		r, g, b = v, v, v
	}
	return Color{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}

// ColorRGBToHSV converts c to hue in degrees and normalized saturation and
// value, ignoring alpha.
func ColorRGBToHSV(c Color) (h, s, v float64) {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	mx := max(r, g, b)
	mn := min(r, g, b)
	delta := mx - mn
	v = mx
	if mx == 0 {
		s = 0
	} else {
		s = delta / mx
	}
	if delta == 0 {
		h = 0
	} else {
		switch mx {
		case r:
			h = (g - b) / delta
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/delta + 2
		case b:
			h = (r-g)/delta + 4
		}
		h *= 60
	}
	return
}

func hexDigit(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

func hexByte(hi, lo byte) uint8 { return hexDigit(hi)*16 + hexDigit(lo) }

func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// ColorMode identifies a terminal color capability level.
type ColorMode int

const (
	// ColorTrueColor supports 24-bit RGB.
	ColorTrueColor ColorMode = iota
	// Color256 supports the xterm 256-color palette.
	Color256
	// Color16 supports the 16 ANSI colors.
	Color16
)

// FgColorSeq returns an ANSI foreground-color sequence for c in mode.
func FgColorSeq(c Color, mode ColorMode) string {
	r, g, b := int(c.R), int(c.G), int(c.B)
	switch mode {
	case ColorTrueColor:
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	case Color256:
		return fmt.Sprintf("\x1b[38;5;%dm", rgbTo256(r, g, b))
	default:
		return fmt.Sprintf("\x1b[%dm", rgbTo16Fg(r, g, b))
	}
}

// BgColorSeq returns an ANSI background-color sequence for c in mode.
func BgColorSeq(c Color, mode ColorMode) string {
	r, g, b := int(c.R), int(c.G), int(c.B)
	switch mode {
	case ColorTrueColor:
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	case Color256:
		return fmt.Sprintf("\x1b[48;5;%dm", rgbTo256(r, g, b))
	default:
		return fmt.Sprintf("\x1b[%dm", rgbTo16Bg(r, g, b))
	}
}

// xtermCubeLevels contains the six channel values in xterm palette entries 16
// through 231. The first step is 95 and each later step is 40.
var xtermCubeLevels = [6]int{0, 95, 135, 175, 215, 255}

// xtermGreyLevels contains the 24 equal-channel values in xterm palette entries
// 232 through 255.
var xtermGreyLevels = [24]int{
	8, 18, 28, 38, 48, 58, 68, 78, 88, 98, 108, 118,
	128, 138, 148, 158, 168, 178, 188, 198, 208, 218, 228, 238,
}

// xtermColor returns the standard color of xterm palette entry i, from 16
// through 255.
func xtermColor(i int) Color {
	if i >= 232 {
		v := uint8(xtermGreyLevels[i-232])
		return Color{R: v, G: v, B: v, A: 255}
	}
	i -= 16
	return Color{
		R: uint8(xtermCubeLevels[i/36]),
		G: uint8(xtermCubeLevels[i/6%6]),
		B: uint8(xtermCubeLevels[i%6]),
		A: 255,
	}
}

// xtermOklab holds the Oklab coordinates of xterm palette entries 16 through
// 255, indexed from entry 16.
var xtermOklab = func() (lab [240][3]float64) {
	for i := range lab {
		L, a, b := xtermColor(16 + i).ToOklab()
		lab[i] = [3]float64{L, a, b}
	}
	return lab
}()

// rgbTo256 returns the xterm palette entry, from 16 through 255, perceptually
// closest to the given color by Oklab distance. Entries 0 through 15 are
// skipped because terminal themes redefine them.
func rgbTo256(r, g, b int) int {
	L, a, bb := Color{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}.ToOklab()
	best, bestDist := 0, math.Inf(1)
	for i, p := range xtermOklab {
		dL, da, db := L-p[0], a-p[1], bb-p[2]
		if d := dL*dL + da*da + db*db; d < bestDist {
			best, bestDist = i, d
		}
	}
	return 16 + best
}

// ansi16Colors contains the eight standard and eight bright ANSI colors.
var ansi16Colors = [][3]int{
	{0, 0, 0},       // 0: black
	{128, 0, 0},     // 1: red
	{0, 128, 0},     // 2: green
	{128, 128, 0},   // 3: yellow
	{0, 0, 128},     // 4: blue
	{128, 0, 128},   // 5: magenta
	{0, 128, 128},   // 6: cyan
	{192, 192, 192}, // 7: white
	{128, 128, 128}, // 8: bright black (gray)
	{255, 0, 0},     // 9: bright red
	{0, 255, 0},     // 10: bright green
	{255, 255, 0},   // 11: bright yellow
	{0, 0, 255},     // 12: bright blue
	{255, 0, 255},   // 13: bright magenta
	{0, 255, 255},   // 14: bright cyan
	{255, 255, 255}, // 15: bright white
}

// rgbTo16Fg returns the nearest ANSI foreground code in 30-37 or 90-97.
func rgbTo16Fg(r, g, b int) int {
	idx := nearestAnsi16(r, g, b)
	if idx < 8 {
		return 30 + idx
	}
	return 90 + (idx - 8)
}

// rgbTo16Bg returns the nearest ANSI background code in 40-47 or 100-107.
func rgbTo16Bg(r, g, b int) int {
	idx := nearestAnsi16(r, g, b)
	if idx < 8 {
		return 40 + idx
	}
	return 100 + (idx - 8)
}

// nearestAnsi16 returns the nearest ansi16Colors index by squared RGB distance.
func nearestAnsi16(r, g, b int) int {
	bestIdx := 0
	bestDist := math.MaxFloat64
	for i, c := range ansi16Colors {
		dr := float64(r - c[0])
		dg := float64(g - c[1])
		db := float64(b - c[2])
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	return bestIdx
}
