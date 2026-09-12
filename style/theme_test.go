package style

import "testing"

func TestDefaultDark_PopulatesEveryRole(t *testing.T) {
	th := DefaultDark()
	if th == nil {
		t.Fatal("DefaultDark returned nil")
	}
	if th.Name == "" {
		t.Error("Name should be set")
	}

	checks := []struct {
		name string
		c    Color
	}{
		{"Bg", th.Bg},
		{"Surface", th.Surface},
		{"SurfaceAlt", th.SurfaceAlt},
		{"Border", th.Border},
		{"BorderStrong", th.BorderStrong},
		{"TextStrong", th.TextStrong},
		{"TextWeak", th.TextWeak},
		{"TextMuted", th.TextMuted},
		{"Accent", th.Accent},
		{"AccentStrong", th.AccentStrong},
		{"Success", th.Success},
		{"Warning", th.Warning},
		{"Danger", th.Danger},
		{"Info", th.Info},
		{"CodeFg", th.CodeFg},
		{"CodeBg", th.CodeBg},
	}
	for _, chk := range checks {
		// A role is considered populated when it's not the zero value of
		// Color (which is fully transparent). We allow Bg/Surface/CodeBg
		// to collapse to the same value; just require they're opaque so
		// widgets that fill them don't emit ColorTransparent.
		if chk.c.A == 0 {
			t.Errorf("role %s should not be fully transparent", chk.name)
		}
	}
	// Entry roles carry a color plus attributes; populated means the
	// foreground is opaque.
	entries := []struct {
		name string
		e    Entry
	}{
		{"MarkdownLink", th.MarkdownLink},
		{"MarkdownBullet", th.MarkdownBullet},
		{"MarkdownQuote", th.MarkdownQuote},
		{"MarkdownQuoteBorder", th.MarkdownQuoteBorder},
		{"MarkdownRule", th.MarkdownRule},
		// Core code palette should be populated so widgets can derive a
		// Chroma style without the caller having to fill every field.
		{"Code.Keyword", th.Code.Keyword},
		{"Code.Type", th.Code.Type},
		{"Code.Function", th.Code.Function},
		{"Code.String", th.Code.String},
		{"Code.Number", th.Code.Number},
		{"Code.Comment", th.Code.Comment},
		{"Code.Operator", th.Code.Operator},
		{"Code.Constant", th.Code.Constant},
		{"Code.Decorator", th.Code.Decorator},
		{"Code.Error", th.Code.Error},
	}
	for _, chk := range entries {
		if chk.e.Fg.A == 0 {
			t.Errorf("role %s should not be fully transparent", chk.name)
		}
	}
	for i, h := range th.MarkdownHeading {
		if h.Fg.A == 0 {
			t.Errorf("MarkdownHeading[%d] should not be fully transparent", i)
		}
	}
}

func TestDefaultDark_CodeBgMatchesSurface(t *testing.T) {
	// The chat UI relies on CodeBg blending with Surface so code blocks
	// don't show a visible chip. Guard that invariant. ThemeFromSeeds
	// sets CodeBg = Surface explicitly, so this is a constructor
	// contract rather than a coincidence.
	th := DefaultDark()
	if !th.CodeBg.Equals(th.Surface) {
		t.Errorf("expected CodeBg == Surface for default dark theme; got %v vs %v", th.CodeBg, th.Surface)
	}
}

func TestResolvers_InheritWhenUnset(t *testing.T) {
	// A theme that only fills the "legacy" fields should resolve every
	// new role to its parent. This guards that inheritance is wired up
	// the way the doc comments promise.
	th := &Theme{
		Bg:           ColorFromInts(10, 10, 10, 255),
		Surface:      ColorFromInts(20, 20, 20, 255),
		SurfaceAlt:   ColorFromInts(30, 30, 30, 255),
		Border:       ColorFromInts(60, 60, 60, 255),
		BorderStrong: ColorFromInts(90, 90, 90, 255),
		TextStrong:   ColorWhite,
		TextWeak:     ColorGray,
		TextMuted:    ColorFromInts(100, 100, 100, 255),
		Accent:       ColorBrightBlue,
		AccentStrong: ColorBrightCyan,
		Success:      ColorBrightGreen,
		Warning:      ColorBrightYellow,
		Danger:       ColorBrightRed,
		Info:         ColorBrightCyan,
		MarkdownLink: Entry{Fg: ColorBrightBlue, Attrs: AttrUnderline},
	}

	cases := []struct {
		name string
		got  Color
		want Color
	}{
		{"ResolveInteractive→Accent", th.ResolveInteractive(), th.Accent},
		{"ResolveInteractiveStrong→AccentStrong", th.ResolveInteractiveStrong(), th.AccentStrong},
		{"ResolveTextBase→TextStrong", th.ResolveTextBase(), th.TextStrong},
		{"ResolveTextLink→MarkdownLink", th.ResolveTextLink(), th.MarkdownLink.Fg},
		{"ResolveSurfaceRaised→SurfaceAlt", th.ResolveSurfaceRaised(), th.SurfaceAlt},
		{"ResolveBorderFocus→Accent", th.ResolveBorderFocus(), th.Accent},
		{"ResolveDiffAddFg→Success", th.ResolveDiffAddFg(), th.Success},
		{"ResolveDiffRemoveFg→Danger", th.ResolveDiffRemoveFg(), th.Danger},
	}
	for _, tc := range cases {
		if !tc.got.Equals(tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, tc.got, tc.want)
		}
	}

	// Entry roles inherit as a whole record, and pick up the default
	// attributes that go with the role.
	entryCases := []struct {
		name string
		got  Entry
		want Entry
	}{
		{"ResolveMarkdownText→TextStrong", th.ResolveMarkdownText(), Entry{Fg: th.TextStrong}},
		{"ResolveMarkdownCode→CodeFg (zero)", th.ResolveMarkdownCode(), Entry{Fg: th.CodeFg, Bg: th.CodeBg}},
		{"ResolveMarkdownStrong→TextStrong bold", th.ResolveMarkdownStrong(), Entry{Fg: th.TextStrong, Attrs: AttrBold}},
		{"ResolveMarkdownEmph→TextWeak italic", th.ResolveMarkdownEmph(), Entry{Fg: th.TextWeak, Attrs: AttrItalic}},
		{"ResolveMarkdownListEnum→MarkdownBullet", th.ResolveMarkdownListEnum(), th.ResolveMarkdownBullet()},
		{"ResolveMarkdownImage→MarkdownLink", th.ResolveMarkdownImage(), th.MarkdownLink},
		{"ResolveMarkdownHeading→bold body", th.ResolveMarkdownHeading(1), Entry{Fg: th.TextStrong, Attrs: AttrBold}},
	}
	for _, tc := range entryCases {
		if tc.got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.name, tc.got, tc.want)
		}
	}
}

func TestResolvers_PreferExplicitFields(t *testing.T) {
	// When a field is set explicitly, resolvers must return it rather
	// than the parent role.
	th := DefaultDark()
	th.Interactive = ColorFromInts(1, 2, 3, 255)
	th.TextBase = ColorFromInts(4, 5, 6, 255)
	th.DiffAddFg = ColorFromInts(7, 8, 9, 255)
	th.MarkdownEmph = Entry{Fg: ColorFromInts(10, 11, 12, 255)}

	if !th.ResolveInteractive().Equals(th.Interactive) {
		t.Errorf("Interactive: resolver did not prefer explicit field")
	}
	if !th.ResolveTextBase().Equals(th.TextBase) {
		t.Errorf("TextBase: resolver did not prefer explicit field")
	}
	if !th.ResolveDiffAddFg().Equals(th.DiffAddFg) {
		t.Errorf("DiffAddFg: resolver did not prefer explicit field")
	}
	if th.ResolveMarkdownEmph() != th.MarkdownEmph {
		t.Errorf("MarkdownEmph: resolver did not prefer explicit field")
	}
}

func TestResolveSelection_DerivedFromInteractive(t *testing.T) {
	// With no SelectionBg set, ResolveSelectionBg must derive a
	// tint from Interactive (or its fallback) blended with Surface,
	// and SelectionFg must pick a contrasting on-color.
	th := &Theme{
		Surface:     ColorFromInts(20, 20, 20, 255),
		Bg:          ColorFromInts(20, 20, 20, 255),
		Interactive: ColorBrightBlue,
		TextStrong:  ColorWhite,
	}
	bg := th.ResolveSelectionBg()
	if bg.A == 0 {
		t.Fatal("ResolveSelectionBg should produce an opaque color")
	}
	// The derived bg is a 70% surface blend of a saturated blue, so
	// it should sit perceptibly between Surface and Interactive.
	if bg.Equals(th.Surface) || bg.Equals(th.Interactive) {
		t.Errorf("derived SelectionBg should differ from both Surface and Interactive; got %v", bg)
	}
	fg := th.ResolveSelectionFg()
	// Against a dim tint near #161830, white wins auto-contrast.
	if !fg.Equals(ColorWhite) {
		t.Errorf("expected white fg on dim selection, got %v (bg=%v)", fg, bg)
	}
}

func TestResolveSelection_PreferExplicit(t *testing.T) {
	th := DefaultDark()
	th.SelectionBg = ColorFromInts(30, 60, 90, 255)
	th.SelectionFg = ColorFromInts(200, 210, 220, 255)
	if !th.ResolveSelectionBg().Equals(th.SelectionBg) {
		t.Errorf("SelectionBg: resolver did not prefer explicit field")
	}
	if !th.ResolveSelectionFg().Equals(th.SelectionFg) {
		t.Errorf("SelectionFg: resolver did not prefer explicit field")
	}
}

func TestResolveSelectionInactive_DarkerThanActive(t *testing.T) {
	// The inactive tint must be strictly darker than the active one
	// so a focused/defocused pair reads as a dimming, not a shift.
	th := DefaultDark()
	active := th.ResolveSelectionBg()
	inactive := th.ResolveSelectionInactiveBg()
	if inactive.Luminance() >= active.Luminance() {
		t.Errorf("inactive selection (L=%.3f) should be dimmer than active (L=%.3f)",
			inactive.Luminance(), active.Luminance())
	}
}

func TestChooseOnColor(t *testing.T) {
	cases := []struct {
		name string
		fill Color
		want Color
	}{
		{"black → white", ColorBlack, ColorWhite},
		{"white → black", ColorWhite, ColorBlack},
		{"midgray → white", ColorFromInts(90, 90, 90, 255), ColorWhite},
		{"bright yellow → black", ColorBrightYellow, ColorBlack},
	}
	for _, tc := range cases {
		if got := ChooseOnColor(tc.fill); !got.Equals(tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
