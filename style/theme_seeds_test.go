package style

import "testing"

// TestThemeFromSeeds_FillsEveryRole guards the basic contract: passing
// only the documented seeds must produce a Theme with every advertised
// color field populated. Ramp-derived fields count as "populated" when
// they have opaque alpha.
func TestThemeFromSeeds_FillsEveryRole(t *testing.T) {
	th := ThemeFromSeeds("test", ThemeSeeds{
		Neutral:     ColorFromInts(15, 15, 20, 255),
		Ink:         ColorWhite,
		Primary:     ColorFromInts(157, 124, 216, 255), // purple
		Interactive: ColorFromInts(250, 178, 131, 255), // peach
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

// TestThemeFromSeeds_InteractiveFallsBackToPrimary verifies the
// documented default: a caller who only sets Primary and leaves
// Interactive zero gets Interactive = Primary.
func TestThemeFromSeeds_InteractiveFallsBackToPrimary(t *testing.T) {
	primary := ColorFromInts(200, 100, 80, 255)
	th := ThemeFromSeeds("fb", ThemeSeeds{
		Neutral: ColorFromInts(20, 20, 20, 255),
		Primary: primary,
		// Interactive intentionally zero.
		Success: ColorBrightGreen,
		Warning: ColorBrightYellow,
		Danger:  ColorBrightRed,
		Info:    ColorBrightCyan,
	}, true)

	// Accent and Interactive both derive from the same ramp (step 10)
	// when Interactive defaults to Primary, so they must be equal.
	if !th.Accent.Equals(th.Interactive) {
		t.Errorf("when Interactive falls back to Primary, Accent should equal Interactive; got %v vs %v",
			th.Accent, th.Interactive)
	}
}

// TestThemeFromSeeds_DarkBgIsDarker guards perceptual direction: in a
// dark theme, Bg must be darker than TextStrong. Reversed themes
// produce unreadable output and this is the cheapest check for
// "someone inverted a ramp".
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

// TestThemeFromSeeds_LightBgIsLighter is the light-theme mirror of the
// above, guarding that the dark flag actually flips the ramp.
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

// TestNeutralCodePalette_DerivedFromRamp pins the mapping between ramp
// indices and token roles so changes to the helper's contract require
// updating this table rather than only the implementation. If the
// mapping shifts, downstream mono themes visibly change — which is a
// signal that should reach code review.
func TestNeutralCodePalette_DerivedFromRamp(t *testing.T) {
	// Build a ramp whose entries are trivially distinguishable so we
	// can assert which index each role draws from. (ramp[i] has R=i.)
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
		{"Builtin", p.Builtin, 9},
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

	// Attribute masks: the monochrome palette must compensate for the
	// missing hue differentiation with bold/italic/underline. Guard
	// every mask so a future tweak can't silently strip them.
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

// TestThemeFromSeeds_CodePaletteIsNeutralRamp pins the constructor
// contract: the default Code palette must be derived from the same
// neutral ramp ThemeFromSeeds uses for surfaces and borders, so a
// seeds-only theme stays visually coherent without a caller having
// to install a code palette themselves.
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

// TestDefaultDark_SeedDerivedInvariants guards structural properties
// that must hold regardless of how the ramps evolve, now that every
// DefaultDark field comes from the seed generator.
func TestDefaultDark_SeedDerivedInvariants(t *testing.T) {
	th := DefaultDark()

	// CodeBg must match Surface so code blocks don't show a visible
	// chip against the message background. ThemeFromSeeds wires this
	// explicitly.
	if !th.CodeBg.Equals(th.Surface) {
		t.Errorf("CodeBg should equal Surface; got %v vs %v", th.CodeBg, th.Surface)
	}

	// Bg must be perceptually darker than TextStrong — the dark theme
	// is unusable otherwise.
	if th.Bg.ToOKLCH().L >= th.TextStrong.ToOKLCH().L {
		t.Errorf("dark theme Bg (L=%.3f) should be darker than TextStrong (L=%.3f)",
			th.Bg.ToOKLCH().L, th.TextStrong.ToOKLCH().L)
	}

	// Heading gradient must descend in lightness (H1 = brightest).
	// This checks the ramp direction hasn't inverted.
	for i := 1; i < len(th.MarkdownHeading); i++ {
		prev := th.MarkdownHeading[i-1].Fg.ToOKLCH().L
		cur := th.MarkdownHeading[i].Fg.ToOKLCH().L
		if cur > prev {
			t.Errorf("H%d (L=%.3f) should not be brighter than H%d (L=%.3f)",
				i+1, cur, i, prev)
		}
	}
}
