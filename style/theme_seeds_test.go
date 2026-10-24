package style

import "testing"

// TestThemeFromSeeds_FillsEveryRole verifies that the seed constructor
// populates every advertised role with an opaque color.
func TestThemeFromSeeds_FillsEveryRole(t *testing.T) {
	th := ThemeFromSeeds("test", ThemeSeeds{
		Neutral:     ColorFromInts(15, 15, 20, 255),
		Ink:         ColorWhite,
		Primary:     ColorFromInts(157, 124, 216, 255), // Purple.
		Interactive: ColorFromInts(250, 178, 131, 255), // Peach.
		Success:     ColorFromInts(127, 216, 143, 255),
		Warning:     ColorFromInts(245, 167, 66, 255),
		Danger:      ColorFromInts(224, 108, 117, 255),
		Info:        ColorFromInts(86, 182, 194, 255),
	}, true)

	checks := []struct {
		name string
		c    Color
	}{
		{"Bg", th.Bg},
		{"Surface", th.Surface},
		{"SurfaceAlt", th.SurfaceAlt},
		{"SurfaceRaised", th.SurfaceRaised},
		{"Border", th.Border},
		{"BorderWeak", th.BorderWeak},
		{"BorderStrong", th.BorderStrong},
		{"BorderFocus", th.BorderFocus},
		{"TextStrong", th.TextStrong},
		{"TextBase", th.TextBase},
		{"TextWeak", th.TextWeak},
		{"TextMuted", th.TextMuted},
		{"TextLink", th.TextLink},
		{"Accent", th.Accent},
		{"AccentStrong", th.AccentStrong},
		{"Interactive", th.Interactive},
		{"InteractiveStrong", th.InteractiveStrong},
		{"Success", th.Success},
		{"Warning", th.Warning},
		{"Danger", th.Danger},
		{"Info", th.Info},
		{"OnAccent", th.OnAccent},
		{"OnInteractive", th.OnInteractive},
		{"OnSuccess", th.OnSuccess},
		{"OnWarning", th.OnWarning},
		{"OnDanger", th.OnDanger},
		{"OnInfo", th.OnInfo},
		{"DiffAddSurface", th.DiffAddSurface},
		{"DiffRemoveSurface", th.DiffRemoveSurface},
		{"DiffAddFg", th.DiffAddFg},
		{"DiffRemoveFg", th.DiffRemoveFg},
		{"CodeFg", th.CodeFg},
		{"CodeBg", th.CodeBg},
	}
	for _, chk := range checks {
		if chk.c.A == 0 {
			t.Errorf("role %s should be populated, got transparent", chk.name)
		}
	}
	entries := []struct {
		name string
		e    Entry
	}{
		{"MarkdownText", th.MarkdownText},
		{"MarkdownLink", th.MarkdownLink},
		{"MarkdownLinkText", th.MarkdownLinkText},
		{"MarkdownCode", th.MarkdownCode},
		{"MarkdownEmph", th.MarkdownEmph},
		{"MarkdownStrong", th.MarkdownStrong},
		{"MarkdownBullet", th.MarkdownBullet},
		{"MarkdownListEnum", th.MarkdownListEnum},
		{"MarkdownQuote", th.MarkdownQuote},
		{"MarkdownQuoteBorder", th.MarkdownQuoteBorder},
		{"MarkdownRule", th.MarkdownRule},
		{"MarkdownImage", th.MarkdownImage},
		{"MarkdownImageText", th.MarkdownImageText},
	}
	for _, chk := range entries {
		if chk.e.Fg.A == 0 {
			t.Errorf("role %s should be populated, got transparent", chk.name)
		}
	}
	for i, h := range th.MarkdownHeading {
		if h.Fg.A == 0 {
			t.Errorf("MarkdownHeading[%d] should be populated", i)
		}
	}
}

// TestThemeFromSeeds_InteractiveFallsBackToPrimary verifies that a transparent
// Interactive seed inherits Primary.
func TestThemeFromSeeds_InteractiveFallsBackToPrimary(t *testing.T) {
	primary := ColorFromInts(200, 100, 80, 255)
	th := ThemeFromSeeds("fb", ThemeSeeds{
		Neutral: ColorFromInts(20, 20, 20, 255),
		Primary: primary,
		// Leave Interactive transparent to request its fallback.
		Success: ColorBrightGreen,
		Warning: ColorBrightYellow,
		Danger:  ColorBrightRed,
		Info:    ColorBrightCyan,
	}, true)

	// Both roles use step ten of the same ramp after fallback.
	if !th.Accent.Equals(th.Interactive) {
		t.Errorf("when Interactive falls back to Primary, Accent should equal Interactive; got %v vs %v",
			th.Accent, th.Interactive)
	}
}

// TestThemeFromSeeds_DarkBgIsDarker verifies the dark ramp's perceptual
// direction.
func TestThemeFromSeeds_DarkBgIsDarker(t *testing.T) {
	th := ThemeFromSeeds("t", ThemeSeeds{
		Neutral:     ColorFromInts(20, 20, 20, 255),
		Primary:     ColorBrightBlue,
		Interactive: ColorBrightBlue,
		Success:     ColorBrightGreen,
		Warning:     ColorBrightYellow,
		Danger:      ColorBrightRed,
		Info:        ColorBrightCyan,
	}, true)

	if th.Bg.ToOKLCH().L >= th.TextStrong.ToOKLCH().L {
		t.Errorf("dark theme Bg (%v, L=%.3f) should be darker than TextStrong (%v, L=%.3f)",
			th.Bg, th.Bg.ToOKLCH().L, th.TextStrong, th.TextStrong.ToOKLCH().L)
	}
}

// TestThemeFromSeeds_LightBgIsLighter verifies that a light ramp reverses the
// dark ramp's perceptual direction.
func TestThemeFromSeeds_LightBgIsLighter(t *testing.T) {
	th := ThemeFromSeeds("t", ThemeSeeds{
		Neutral:     ColorFromInts(240, 240, 240, 255),
		Ink:         ColorBlack,
		Primary:     ColorFromInts(59, 125, 216, 255),
		Interactive: ColorFromInts(59, 125, 216, 255),
		Success:     ColorFromInts(61, 154, 87, 255),
		Warning:     ColorFromInts(214, 140, 39, 255),
		Danger:      ColorFromInts(209, 56, 61, 255),
		Info:        ColorFromInts(49, 135, 149, 255),
	}, false)

	if th.Bg.ToOKLCH().L <= th.TextStrong.ToOKLCH().L {
		t.Errorf("light theme Bg (%v, L=%.3f) should be lighter than TextStrong (%v, L=%.3f)",
			th.Bg, th.Bg.ToOKLCH().L, th.TextStrong, th.TextStrong.ToOKLCH().L)
	}
}

// TestNeutralCodePalette_DerivedFromRamp pins each syntax role to its documented
// neutral-ramp index.
func TestNeutralCodePalette_DerivedFromRamp(t *testing.T) {
	// Encode each ramp index in its red channel for direct comparison.
	var ramp [12]Color
	for i := range ramp {
		ramp[i] = Color{R: uint8(i), G: uint8(i), B: uint8(i), A: 255}
	}
	p := NeutralCodePalette(ramp)

	cases := []struct {
		name string
		got  Entry
		want uint8
	}{
		{"Error", p.Error, 11},
		{"Keyword", p.Keyword, 11},
		{"Function", p.Function, 11},
		{"Type", p.Type, 10},
		{"Constant", p.Constant, 10},
		{"Decorator", p.Decorator, 10},
		{"Variable", p.Variable, 9},
		{"Number", p.Number, 9},
		{"String", p.String, 8},
		{"Operator", p.Operator, 8},
		{"Punctuation", p.Punctuation, 8},
		{"Comment", p.Comment, 6},
	}
	for _, c := range cases {
		if c.got.Fg.R != c.want {
			t.Errorf("%s should draw from ramp[%d] (R=%d), got R=%d",
				c.name, c.want, c.want, c.got.Fg.R)
		}
	}

	// Decorations distinguish monochrome roles that share similar colors.
	if p.Keyword.Attrs&AttrBold == 0 {
		t.Error("Keyword should include AttrBold")
	}
	if p.Function.Attrs&AttrBold == 0 {
		t.Error("Function should include AttrBold")
	}
	if p.Type.Attrs&AttrBold == 0 {
		t.Error("Type should include AttrBold")
	}
	if p.Comment.Attrs&AttrItalic == 0 {
		t.Error("Comment should include AttrItalic")
	}
	if p.Error.Attrs&(AttrBold|AttrUnderline) != (AttrBold | AttrUnderline) {
		t.Error("Error should include Bold|Underline")
	}
}

// TestThemeFromSeeds_CodePaletteIsNeutralRamp verifies that surfaces and the
// default syntax palette use the same neutral ramp.
func TestThemeFromSeeds_CodePaletteIsNeutralRamp(t *testing.T) {
	seeds := ThemeSeeds{
		Neutral:     Color{R: 20, G: 20, B: 20, A: 255},
		Ink:         ColorWhite,
		Primary:     ColorBrightCyan,
		Interactive: ColorBrightBlue,
		Success:     ColorBrightGreen,
		Warning:     ColorBrightYellow,
		Danger:      ColorBrightRed,
		Info:        ColorBrightCyan,
	}
	th := ThemeFromSeeds("t", seeds, true)
	expected := NeutralCodePalette(
		GenerateNeutralScale(seeds.Neutral, true, seeds.Ink),
	)
	if th.Code.Keyword != expected.Keyword {
		t.Errorf("Keyword should come from neutral ramp: got %v want %v",
			th.Code.Keyword, expected.Keyword)
	}
	if th.Code.String != expected.String {
		t.Errorf("String should come from neutral ramp: got %v want %v",
			th.Code.String, expected.String)
	}
	if th.Code.Comment != expected.Comment {
		t.Errorf("Comment should come from neutral ramp: got %v want %v",
			th.Code.Comment, expected.Comment)
	}
}

// TestDefaultDark_SeedDerivedInvariants verifies relationships required of the
// generated default theme.
func TestDefaultDark_SeedDerivedInvariants(t *testing.T) {
	th := DefaultDark()

	// Code blocks share the panel surface instead of drawing a separate chip.
	if !th.CodeBg.Equals(th.Surface) {
		t.Errorf("CodeBg should equal Surface; got %v vs %v", th.CodeBg, th.Surface)
	}

	// The background is perceptually darker than primary text.
	if th.Bg.ToOKLCH().L >= th.TextStrong.ToOKLCH().L {
		t.Errorf("dark theme Bg (L=%.3f) should be darker than TextStrong (L=%.3f)",
			th.Bg.ToOKLCH().L, th.TextStrong.ToOKLCH().L)
	}

	// Heading lightness descends from H1 through H6.
	for i := 1; i < len(th.MarkdownHeading); i++ {
		prev := th.MarkdownHeading[i-1].Fg.ToOKLCH().L
		cur := th.MarkdownHeading[i].Fg.ToOKLCH().L
		if cur > prev {
			t.Errorf("H%d (L=%.3f) should not be brighter than H%d (L=%.3f)",
				i+1, cur, i, prev)
		}
	}
}
