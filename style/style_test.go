package style

import (
	"strings"
	"testing"
)

func TestNewStyle(t *testing.T) {
	s := NewStyle()
	if !s.Fg.Equals(ColorWhite) {
		t.Errorf("default fg should be White, got %s", s.Fg)
	}
	if !s.Bg.Equals(ColorTransparent) {
		t.Errorf("default bg should be Transparent, got %s", s.Bg)
	}
	if s.Attributes != AttrNone {
		t.Errorf("default attrs should be AttrNone, got %d", s.Attributes)
	}
}

func TestStyleFrom(t *testing.T) {
	s := StyleFrom(ColorRed, ColorBlue, AttrBold|AttrItalic)
	if !s.Fg.Equals(ColorRed) {
		t.Errorf("fg should be Red, got %s", s.Fg)
	}
	if !s.Bg.Equals(ColorBlue) {
		t.Errorf("bg should be Blue, got %s", s.Bg)
	}
	if s.Attributes != AttrBold|AttrItalic {
		t.Errorf("attrs should be Bold|Italic, got %d", s.Attributes)
	}
}

func TestStyleBold(t *testing.T) {
	s := NewStyle().Bold()
	if !HasAttribute(s.Attributes, AttrBold) {
		t.Error("should have Bold")
	}
}

func TestStyleItalic(t *testing.T) {
	s := NewStyle().Italic()
	if !HasAttribute(s.Attributes, AttrItalic) {
		t.Error("should have Italic")
	}
}

func TestStyleUnderline(t *testing.T) {
	s := NewStyle().Underline()
	if !HasAttribute(s.Attributes, AttrUnderline) {
		t.Error("should have Underline")
	}
}

func TestStyleDim(t *testing.T) {
	s := NewStyle().Dim()
	if !HasAttribute(s.Attributes, AttrDim) {
		t.Error("should have Dim")
	}
}

func TestStyleStrikethrough(t *testing.T) {
	s := NewStyle().Strikethrough()
	if !HasAttribute(s.Attributes, AttrStrikethrough) {
		t.Error("should have Strikethrough")
	}
}

func TestStyleInverse(t *testing.T) {
	s := NewStyle().Inverse()
	if !HasAttribute(s.Attributes, AttrInverse) {
		t.Error("should have Inverse")
	}
}

func TestStyleChaining(t *testing.T) {
	s := NewStyle().Bold().Italic().Underline().WithFg(ColorRed).WithBg(ColorBlue)
	if !HasAttribute(s.Attributes, AttrBold) {
		t.Error("should have Bold")
	}
	if !HasAttribute(s.Attributes, AttrItalic) {
		t.Error("should have Italic")
	}
	if !HasAttribute(s.Attributes, AttrUnderline) {
		t.Error("should have Underline")
	}
	if !s.Fg.Equals(ColorRed) {
		t.Errorf("fg should be Red, got %s", s.Fg)
	}
	if !s.Bg.Equals(ColorBlue) {
		t.Errorf("bg should be Blue, got %s", s.Bg)
	}
}

func TestStyleWithFg(t *testing.T) {
	s := NewStyle().WithFg(ColorGreen)
	if !s.Fg.Equals(ColorGreen) {
		t.Errorf("fg should be Green, got %s", s.Fg)
	}
}

func TestStyleWithBg(t *testing.T) {
	s := NewStyle().WithBg(ColorYellow)
	if !s.Bg.Equals(ColorYellow) {
		t.Errorf("bg should be Yellow, got %s", s.Bg)
	}
}

func TestStyleWithAttrs(t *testing.T) {
	s := NewStyle().WithAttrs(AttrBlink | AttrHidden)
	if s.Attributes != AttrBlink|AttrHidden {
		t.Errorf("attrs should be Blink|Hidden, got %d", s.Attributes)
	}
}

func TestStyleImmutability(t *testing.T) {
	s1 := NewStyle()
	s2 := s1.Bold()
	if HasAttribute(s1.Attributes, AttrBold) {
		t.Error("original style should not be modified")
	}
	if !HasAttribute(s2.Attributes, AttrBold) {
		t.Error("new style should have Bold")
	}
}

func TestStyleMerge_FgOverride(t *testing.T) {
	base := NewStyle().WithFg(ColorWhite)
	overlay := Style{Fg: ColorRed, Bg: ColorTransparent, Attributes: AttrNone}
	merged := base.Merge(overlay)
	if !merged.Fg.Equals(ColorRed) {
		t.Errorf("merged fg should be Red, got %s", merged.Fg)
	}
}

func TestStyleMerge_BgOverride(t *testing.T) {
	base := NewStyle()
	overlay := Style{Fg: ColorTransparent, Bg: ColorBlue, Attributes: AttrNone}
	merged := base.Merge(overlay)
	if !merged.Bg.Equals(ColorBlue) {
		t.Errorf("merged bg should be Blue, got %s", merged.Bg)
	}
}

func TestStyleMerge_TransparentNoOverride(t *testing.T) {
	base := NewStyle().WithFg(ColorGreen)
	overlay := Style{Fg: ColorTransparent, Bg: ColorTransparent, Attributes: AttrNone}
	merged := base.Merge(overlay)
	if !merged.Fg.Equals(ColorGreen) {
		t.Errorf("merged fg should stay Green, got %s", merged.Fg)
	}
}

func TestStyleMerge_AttributesCombine(t *testing.T) {
	base := NewStyle().Bold()
	overlay := Style{Fg: ColorTransparent, Bg: ColorTransparent, Attributes: AttrItalic}
	merged := base.Merge(overlay)
	if !HasAttribute(merged.Attributes, AttrBold) {
		t.Error("merged should have Bold from base")
	}
	if !HasAttribute(merged.Attributes, AttrItalic) {
		t.Error("merged should have Italic from overlay")
	}
}

func TestStyleEquals_Same(t *testing.T) {
	s1 := NewStyle().Bold().WithFg(ColorRed)
	s2 := NewStyle().Bold().WithFg(ColorRed)
	if !s1.Equals(s2) {
		t.Error("identical styles should be equal")
	}
}

func TestStyleEquals_Different(t *testing.T) {
	s1 := NewStyle().Bold()
	s2 := NewStyle().Italic()
	if s1.Equals(s2) {
		t.Error("different styles should not be equal")
	}
}

func TestAttrsANSI_None(t *testing.T) {
	result := AttrsANSI(AttrNone)
	if result != "" {
		t.Errorf("AttrNone should produce empty string, got %q", result)
	}
}

func TestAttrsANSI_Multiple(t *testing.T) {
	result := AttrsANSI(AttrBold | AttrUnderline | AttrStrikethrough)
	if !strings.Contains(result, "1") || !strings.Contains(result, "4") || !strings.Contains(result, "9") {
		t.Errorf("should contain codes 1, 4, 9, got %q", result)
	}
}

func TestResetANSI(t *testing.T) {
	if ResetANSI != "\x1b[0m" {
		t.Errorf("ResetANSI should be \\x1b[0m, got %q", ResetANSI)
	}
}
