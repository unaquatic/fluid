package style

import "testing"

// TestDarculaSolid_Signature pins the palette's identifying colors.
func TestDarculaSolid_Signature(t *testing.T) {
	th := DarculaSolid()

	checks := []struct {
		name    string
		got     Color
		wantHex string
	}{
		{"Bg", th.Bg, "#1a1a1a"},
		{"TextBase", th.TextBase, "#e6e6e6"},
		{"Accent", th.Accent, "#eb7513"},
		{"Code.Keyword", th.Code.Keyword.Fg, "#eb7513"},
		{"Code.Function", th.Code.Function.Fg, "#ffb642"},
		{"Code.Type", th.Code.Type.Fg, "#3dbfc8"},
		{"Code.String", th.Code.String.Fg, "#91b557"},
		{"Code.Constant", th.Code.Constant.Fg, "#aa77c6"},
		{"Code.Comment", th.Code.Comment.Fg, "#8a8a8a"},
		{"Code.Number", th.Code.Number.Fg, "#5a99c9"},
		{"Code.Decorator", th.Code.Decorator.Fg, "#e4be13"},
		{"SelectionBg", th.SelectionBg, "#176081"},
		{"SurfaceRaised", th.SurfaceRaised, "#1f1f1f"},
	}
	for _, c := range checks {
		want := ColorFromHex(c.wantHex)
		if !c.got.Equals(want) {
			t.Errorf("%s: got %s want %s", c.name, c.got.ToHex(), c.wantHex)
		}
	}
}

// TestDarculaSolid_SharedColors verifies that the roles of each group share
// one color, so that changing one color changes them all.
func TestDarculaSolid_SharedColors(t *testing.T) {
	th := DarculaSolid()

	groups := map[string][]struct {
		name string
		c    Color
	}{
		"keyword orange": {
			{"Code.Keyword", th.Code.Keyword.Fg}, {"Code.Primitive", th.Code.Primitive.Fg},
			{"Code.StringEscape", th.Code.StringEscape.Fg}, {"Accent", th.Accent},
			{"TextLink", th.TextLink}, {"Warning", th.Warning},
			{"MarkdownHeading[0]", th.MarkdownHeading[0].Fg}, {"MarkdownStrong", th.MarkdownStrong.Fg},
			{"MarkdownEmph", th.MarkdownEmph.Fg}, {"MarkdownLink", th.MarkdownLink.Fg},
			{"MarkdownBullet", th.MarkdownBullet.Fg}, {"MarkdownImage", th.MarkdownImage.Fg},
		},
		"text": {
			{"TextStrong", th.TextStrong}, {"TextBase", th.TextBase},
			{"SelectionFg", th.SelectionFg}, {"MarkdownText", th.MarkdownText.Fg},
			{"CodeFg", th.CodeFg}, {"Code.Variable", th.Code.Variable.Fg},
			{"Code.Operator", th.Code.Operator.Fg},
			{"Code.Punctuation", th.Code.Punctuation.Fg},
		},
		"string green": {
			{"Code.String", th.Code.String.Fg}, {"Code.Regex", th.Code.Regex.Fg},
			{"MarkdownCode", th.MarkdownCode.Fg},
		},
		"function yellow":  {{"Code.Function", th.Code.Function.Fg}, {"Code.Tag", th.Code.Tag.Fg}},
		"constant purple":  {{"Code.Constant", th.Code.Constant.Fg}, {"Code.Property", th.Code.Property.Fg}},
		"decorator yellow": {{"Code.Decorator", th.Code.Decorator.Fg}, {"Code.Preproc", th.Code.Preproc.Fg}},
		"canvas":           {{"Bg", th.Bg}, {"Surface", th.Surface}, {"CodeBg", th.CodeBg}, {"MarkdownCode.Bg", th.MarkdownCode.Bg}},
	}
	for group, roles := range groups {
		for _, r := range roles[1:] {
			if !r.c.Equals(roles[0].c) {
				t.Errorf("%s: %s is %s, %s is %s", group, r.name, r.c.ToHex(), roles[0].name, roles[0].c.ToHex())
			}
		}
	}
}

// TestDarculaSolid_AttributeMasks verifies decorations on representative syntax
// roles.
func TestDarculaSolid_AttributeMasks(t *testing.T) {
	th := DarculaSolid()

	if th.Code.Keyword.Attrs&AttrBold == 0 {
		t.Error("Keyword missing AttrBold")
	}
	if th.Code.Constant.Attrs&AttrItalic == 0 {
		t.Error("Constant missing AttrItalic")
	}
	if th.Code.Comment.Attrs&AttrItalic == 0 {
		t.Error("Comment missing AttrItalic")
	}
	if th.Code.Error.Attrs&AttrUnderline == 0 {
		t.Error("Error missing AttrUnderline")
	}
}

// TestDarculaSolid_FullyPopulated verifies that every directly consumed color
// role is opaque.
func TestDarculaSolid_FullyPopulated(t *testing.T) {
	th := DarculaSolid()

	roles := []struct {
		name string
		c    Color
	}{
		{"Bg", th.Bg}, {"Surface", th.Surface}, {"SurfaceAlt", th.SurfaceAlt},
		{"SurfaceRaised", th.SurfaceRaised},
		{"Border", th.Border}, {"BorderWeak", th.BorderWeak},
		{"BorderStrong", th.BorderStrong}, {"BorderFocus", th.BorderFocus},
		{"TextStrong", th.TextStrong}, {"TextBase", th.TextBase},
		{"TextWeak", th.TextWeak}, {"TextMuted", th.TextMuted},
		{"Accent", th.Accent}, {"Interactive", th.Interactive},
		{"Success", th.Success}, {"Warning", th.Warning},
		{"Danger", th.Danger}, {"Info", th.Info},
		{"SelectionBg", th.SelectionBg}, {"SelectionFg", th.SelectionFg},
		{"SelectionInactiveBg", th.SelectionInactiveBg},
		{"DiffAddSurface", th.DiffAddSurface},
		{"DiffRemoveSurface", th.DiffRemoveSurface},
		{"CodeFg", th.CodeFg}, {"CodeBg", th.CodeBg},
	}
	for _, r := range roles {
		if r.c.A == 0 {
			t.Errorf("role %s must be opaque; got transparent", r.name)
		}
	}
}
