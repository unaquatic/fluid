package style

import (
	"fmt"
	"math"
	"strings"
)

// Color represents a color with uint8 channels in [0, 255] range.
type Color struct {
	R, G, B, A uint8
}

// --- Constructors ---

// ColorFromHex parses a hex string ("#RGB", "#RGBA", "#RRGGBB", "#RRGGBBAA") into a Color.
func ColorFromHex(hex string) Color {
	hex = strings.TrimPrefix(hex, "#")

	switch len(hex) {
	case 3: // RGB
		r := hexDigit(hex[0])
		g := hexDigit(hex[1])
		b := hexDigit(hex[2])
		return Color{
			R: r * 17,
			G: g * 17,
			B: b * 17,
			A: 255,
		}
	case 4: // RGBA
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
	case 6: // RRGGBB
		r := hexByte(hex[0], hex[1])
		g := hexByte(hex[2], hex[3])
		b := hexByte(hex[4], hex[5])
		return Color{
			R: r,
			G: g,
			B: b,
			A: 255,
		}
	case 8: // RRGGBBAA
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

// ColorFromInts creates a Color from 0-255 integer values.
func ColorFromInts(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// ColorFromFloats creates a Color from 0.0-1.0 float values.
func ColorFromFloats(r, g, b, a float32) Color {
	return Color{
		R: uint8(math.Round(float64(r) * 255)),
		G: uint8(math.Round(float64(g) * 255)),
		B: uint8(math.Round(float64(b) * 255)),
		A: uint8(math.Round(float64(a) * 255)),
	}
}

// --- Methods ---

// ToInts converts to 0-255 integer values.
func (c Color) ToInts() (r, g, b, a uint8) {
	return c.R, c.G, c.B, c.A
}

// ToFloats converts to 0.0-1.0 float32 values.
func (c Color) ToFloats() (r, g, b, a float32) {
	return float32(c.R) / 255.0, float32(c.G) / 255.0, float32(c.B) / 255.0, float32(c.A) / 255.0
}

// ToHex converts to "#RRGGBB" or "#RRGGBBAA" string.
// If alpha is 255, the AA suffix is omitted.
func (c Color) ToHex() string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// Equals performs deep equality comparison.
func (c Color) Equals(other Color) bool {
	return c == other
}

// String returns "rgba(r, g, b, a)" with uint8 values.
func (c Color) String() string {
	return fmt.Sprintf("rgba(%d, %d, %d, %d)", c.R, c.G, c.B, c.A)
}

// Blend performs linear interpolation between two colors.
// t=0 returns c, t=1 returns other.
func (c Color) Blend(other Color, t float32) Color {
	r := float32(c.R) + (float32(other.R)-float32(c.R))*t
	g := float32(c.G) + (float32(other.G)-float32(c.G))*t
	b := float32(c.B) + (float32(other.B)-float32(c.B))*t
	a := float32(c.A) + (float32(other.A)-float32(c.A))*t
	return Color{
		R: clampU8(r),
		G: clampU8(g),
		B: clampU8(b),
		A: clampU8(a),
	}
}

// ColorBlendPerceptual composites overlay over text using perceptual alpha curves.
func ColorBlendPerceptual(overlay, text Color) Color {
	if overlay.A == 255 {
		return overlay
	}

	if text.A == 0 {
		alpha := float32(overlay.A) / 255.0
		r := float32(overlay.R) * alpha
		g := float32(overlay.G) * alpha
		b := float32(overlay.B) * alpha
		if r < 0.5 && g < 0.5 && b < 0.5 {
			return Color{R: 0, G: 0, B: 0, A: 0}
		}
		return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: overlay.A}
	}

	alpha := float32(overlay.A) / 255.0
	var perceptualAlpha float32

	if alpha > 0.8 {
		normalizedHighAlpha := (alpha - 0.8) * 5.0
		curvedHighAlpha := pow32(normalizedHighAlpha, 0.2)
		perceptualAlpha = 0.8 + curvedHighAlpha*0.2
	} else {
		perceptualAlpha = pow32(alpha, 0.9)
	}

	oneMinusAlpha := 1.0 - perceptualAlpha
	blendedR := float32(overlay.R)*perceptualAlpha + float32(text.R)*oneMinusAlpha
	blendedG := float32(overlay.G)*perceptualAlpha + float32(text.G)*oneMinusAlpha
	blendedB := float32(overlay.B)*perceptualAlpha + float32(text.B)*oneMinusAlpha

	resultAlpha := alpha + float32(text.A)/255.0*(1.0-alpha)

	return Color{R: clampU8(blendedR), G: clampU8(blendedG), B: clampU8(blendedB), A: clampU8(resultAlpha * 255)}
}

// WithAlpha returns a copy with modified alpha.
func (c Color) WithAlpha(a uint8) Color {
	c.A = a
	return c
}

// Premultiply returns the premultiplied-alpha version.
func (c Color) Premultiply() Color {
	a := float32(c.A) / 255.0
	return Color{
		R: uint8(float32(c.R) * a),
		G: uint8(float32(c.G) * a),
		B: uint8(float32(c.B) * a),
		A: c.A,
	}
}

// IsTransparent returns true if alpha is 0.
func (c Color) IsTransparent() bool {
	return c.A == 0
}

// Luminance returns perceived luminance using the BT.709 formula.
func (c Color) Luminance() float32 {
	r := float32(c.R) / 255.0
	g := float32(c.G) / 255.0
	b := float32(c.B) / 255.0
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// DistanceSq returns the squared Euclidean distance between two colors in RGB space.
func (c Color) DistanceSq(other Color) float32 {
	dr := float32(int(c.R) - int(other.R))
	dg := float32(int(c.G) - int(other.G))
	db := float32(int(c.B) - int(other.B))
	return dr*dr + dg*dg + db*db
}

// ColorAverage returns the component-wise average of the given colors.
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

// --- Color Constants ---

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

	ColorSilver  = Color{192, 192, 192, 255}
	ColorGray    = Color{128, 128, 128, 255}
	ColorMaroon  = Color{128, 0, 0, 255}
	ColorOlive   = Color{128, 128, 0, 255}
	ColorLime    = Color{0, 255, 0, 255}
	ColorAqua    = Color{0, 255, 255, 255}
	ColorTeal    = Color{0, 128, 128, 255}
	ColorNavy    = Color{0, 0, 128, 255}
	ColorFuchsia = Color{255, 0, 255, 255}
	ColorPurple  = Color{128, 0, 128, 255}
	ColorOrange  = Color{255, 165, 0, 255}

	ColorBrightBlack   = Color{128, 128, 128, 255}
	ColorBrightRed     = Color{255, 85, 85, 255}
	ColorBrightGreen   = Color{85, 255, 85, 255}
	ColorBrightYellow  = Color{255, 255, 85, 255}
	ColorBrightBlue    = Color{85, 85, 255, 255}
	ColorBrightMagenta = Color{255, 85, 255, 255}
	ColorBrightCyan    = Color{85, 255, 255, 255}
	ColorBrightWhite   = Color{255, 255, 255, 255}
)

// --- ColorInput Resolution ---

type ColorInput interface {
	colorInput()
}

type ColorString string

func (ColorString) colorInput() {}
func (Color) colorInput()       {}

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
	lower := strings.ToLower(s)
	if c, ok := namedColors[lower]; ok {
		return c
	}
	return ColorTransparent
}

// --- Color Conversion Functions ---

func ColorHexToRGB(hex string) Color { return ColorFromHex(hex) }
func ColorRGBToHex(c Color) string   { return c.ToHex() }

func ColorHSVToRGB(h, s, v float32) Color {
	if s == 0 {
		val := uint8(v * 255)
		return Color{val, val, val, 255}
	}
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	h /= 60
	i := float32(math.Floor(float64(h)))
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var r, g, b float32
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

func ColorRGBToHSV(c Color) (h, s, v float32) {
	r := float32(c.R) / 255.0
	g := float32(c.G) / 255.0
	b := float32(c.B) / 255.0
	mx := maxf(r, maxf(g, b))
	mn := minf(r, minf(g, b))
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

// --- Named Color Map ---

var namedColors = map[string]Color{
	"transparent":   ColorTransparent,
	"black":         ColorBlack,
	"white":         ColorWhite,
	"red":           ColorRed,
	"green":         ColorGreen,
	"blue":          ColorBlue,
	"yellow":        ColorYellow,
	"cyan":          ColorCyan,
	"magenta":       ColorMagenta,
	"silver":        ColorSilver,
	"gray":          ColorGray,
	"grey":          ColorGray,
	"maroon":        ColorMaroon,
	"olive":         ColorOlive,
	"lime":          ColorLime,
	"aqua":          ColorAqua,
	"teal":          ColorTeal,
	"navy":          ColorNavy,
	"fuchsia":       ColorFuchsia,
	"purple":        ColorPurple,
	"orange":        ColorOrange,
	"brightblack":   ColorBrightBlack,
	"brightred":     ColorBrightRed,
	"brightgreen":   ColorBrightGreen,
	"brightyellow":  ColorBrightYellow,
	"brightblue":    ColorBrightBlue,
	"brightmagenta": ColorBrightMagenta,
	"brightcyan":    ColorBrightCyan,
	"brightwhite":   ColorBrightWhite,
}

// --- Internal helpers ---

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

func clampU8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func pow32(base, exp float32) float32 {
	return float32(math.Pow(float64(base), float64(exp)))
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// ColorMode
// ---------------------------------------------------------------------------

// ColorMode represents a terminal's color capability level.
type ColorMode int

const (
	// ColorTrueColor supports 24-bit RGB (16 million colors).
	ColorTrueColor ColorMode = iota
	// Color256 supports 256-color palette.
	Color256
	// Color16 supports basic 16 ANSI colors.
	Color16
)

// ---------------------------------------------------------------------------
// ANSI color sequence generation
// ---------------------------------------------------------------------------

// FgColorSeq generates the ANSI foreground color escape sequence for the
// given Color using the specified color mode.
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

// BgColorSeq generates the ANSI background color escape sequence for the
// given Color using the specified color mode.
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

// rgbTo256 maps an RGB color to the nearest xterm-256 color index.
func rgbTo256(r, g, b int) int {
	// Check if it's close to a grayscale ramp value (indices 232-255).
	if r == g && g == b {
		if r < 8 {
			return 16 // black
		}
		if r > 248 {
			return 231 // white
		}
		return int(math.Round(float64(r-8)/247*23)) + 232
	}

	// Map to the 6x6x6 color cube (indices 16-231).
	ri := int(math.Round(float64(r) / 255 * 5))
	gi := int(math.Round(float64(g) / 255 * 5))
	bi := int(math.Round(float64(b) / 255 * 5))
	return 16 + 36*ri + 6*gi + bi
}

// ansi16Colors are the standard 16 ANSI colors (normal + bright).
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

// rgbTo16Fg maps RGB to the nearest ANSI 16-color foreground code (30-37, 90-97).
func rgbTo16Fg(r, g, b int) int {
	idx := nearestAnsi16(r, g, b)
	if idx < 8 {
		return 30 + idx
	}
	return 90 + (idx - 8)
}

// rgbTo16Bg maps RGB to the nearest ANSI 16-color background code (40-47, 100-107).
func rgbTo16Bg(r, g, b int) int {
	idx := nearestAnsi16(r, g, b)
	if idx < 8 {
		return 40 + idx
	}
	return 100 + (idx - 8)
}

// nearestAnsi16 finds the nearest ANSI 16-color index by Euclidean distance.
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
