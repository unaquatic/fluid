package style

import "testing"

// TestDarculaSolid_Signature pins the most recognisable color choices.
// If any of these drift the theme should be renamed.
func TestDarculaSolid_Signature(t *testing.T) {
	th := DarculaSolid()

	checks := []struct {
		name    string
		got     Color
		wantHex string
	}{
		{"Bg", th.Bg, "#1c1c1c"},
		{"TextBase", th.TextBase, "#cfd2d5"},
		{"Accent", th.Accent, "#cc7832"},
		{"Code.Keyword", th.Code.Keyword.Fg, "#cc7832"},
		{"Code.Function", th.Code.Function.Fg, "#ffc66d"},
		{"Code.Type", th.Code.Type.Fg, "#5ca4a9"},
		{"Code.String", th.Code.String.Fg, "#8ea765"},
		{"Code.Constant", th.Code.Constant.Fg, "#a782bb"},
		{"Code.Comment", th.Code.Comment.Fg, "#8a8a8a"},
		{"Code.Number", th.Code.Number.Fg, "#6897bb"},
		{"SelectionBg", th.SelectionBg, "#176081"},
	}
	for _, c := range checks {
		want := ColorFromHex(c.wantHex)
		if !c.got.Equals(want) {
			t.Errorf("%s: got %s want %s", c.name, c.got.ToHex(), c.wantHex)
		}
	}
}

// TestDarculaSolid_AttributeMasks verifies bold/italic/underline masks
// on keyword, constant, comment, and error tokens.
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

// TestDarculaSolid_FullyPopulated ensures every color role the
// renderer reads is opaque, so the chat UI never falls back to
// zero-alpha.
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
