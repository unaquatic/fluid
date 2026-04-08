package style

import (
	"testing"
)

func TestSyntaxStyle_New(t *testing.T) {
	ss := NewSyntaxStyle()
	if ss.StyleCount() != 0 {
		t.Errorf("expected 0 styles initially, got %d", ss.StyleCount())
	}
}

func TestSyntaxStyle_Register(t *testing.T) {
	ss := NewSyntaxStyle()

	fg := ColorRed
	bg := ColorBlue
	id := ss.Register("keyword", &fg, &bg, AttrBold)

	if id != 1 {
		t.Errorf("expected first ID=1, got %d", id)
	}
	if ss.StyleCount() != 1 {
		t.Errorf("expected 1 style, got %d", ss.StyleCount())
	}
}

func TestSyntaxStyle_RegisterMultiple(t *testing.T) {
	ss := NewSyntaxStyle()

	fg1 := ColorRed
	id1 := ss.Register("keyword", &fg1, nil, AttrBold)

	fg2 := ColorGreen
	id2 := ss.Register("string", &fg2, nil, AttrNone)

	fg3 := ColorBlue
	id3 := ss.Register("comment", &fg3, nil, AttrItalic)

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Errorf("expected IDs 1,2,3, got %d,%d,%d", id1, id2, id3)
	}
	if ss.StyleCount() != 3 {
		t.Errorf("expected 3 styles, got %d", ss.StyleCount())
	}
}

func TestSyntaxStyle_RegisterOverwrite(t *testing.T) {
	ss := NewSyntaxStyle()

	fg1 := ColorRed
	id1 := ss.Register("keyword", &fg1, nil, AttrBold)

	// Re-register same name with different style
	fg2 := ColorBlue
	id2 := ss.Register("keyword", &fg2, nil, AttrItalic)

	// Should return same ID
	if id1 != id2 {
		t.Errorf("expected same ID on overwrite, got %d vs %d", id1, id2)
	}

	// Count should still be 1
	if ss.StyleCount() != 1 {
		t.Errorf("expected 1 style after overwrite, got %d", ss.StyleCount())
	}

	// Style should be updated
	s := ss.Style(id1)
	if s == nil {
		t.Fatal("expected style, got nil")
	}
	if *s.Fg != fg2 {
		t.Errorf("expected fg ColorBlue after overwrite, got %v", *s.Fg)
	}
	if s.Attributes != AttrItalic {
		t.Errorf("expected AttrItalic after overwrite, got %d", s.Attributes)
	}
}

func TestSyntaxStyle_ResolveByName(t *testing.T) {
	ss := NewSyntaxStyle()

	fg := ColorRed
	ss.Register("keyword", &fg, nil, AttrBold)
	fg2 := ColorGreen
	ss.Register("string", &fg2, nil, AttrNone)

	if id := ss.ResolveByName("keyword"); id != 1 {
		t.Errorf("expected ID 1 for 'keyword', got %d", id)
	}
	if id := ss.ResolveByName("string"); id != 2 {
		t.Errorf("expected ID 2 for 'string', got %d", id)
	}
	if id := ss.ResolveByName("nonexistent"); id != -1 {
		t.Errorf("expected -1 for unknown name, got %d", id)
	}
}

func TestSyntaxStyle_GetStyle(t *testing.T) {
	ss := NewSyntaxStyle()

	fg := ColorRed
	bg := ColorBlue
	id := ss.Register("keyword", &fg, &bg, AttrBold|AttrItalic)

	s := ss.Style(id)
	if s == nil {
		t.Fatal("expected style, got nil")
	}
	if s.Name != "keyword" {
		t.Errorf("expected name 'keyword', got %q", s.Name)
	}
	if s.Fg == nil || *s.Fg != fg {
		t.Errorf("expected fg ColorRed, got %v", s.Fg)
	}
	if s.Bg == nil || *s.Bg != bg {
		t.Errorf("expected bg ColorBlue, got %v", s.Bg)
	}
	if s.Attributes != AttrBold|AttrItalic {
		t.Errorf("expected AttrBold|AttrItalic, got %d", s.Attributes)
	}
}

func TestSyntaxStyle_GetStyle_Invalid(t *testing.T) {
	ss := NewSyntaxStyle()

	// ID 0 is reserved
	if ss.Style(0) != nil {
		t.Error("expected nil for ID 0")
	}
	// Negative ID
	if ss.Style(-1) != nil {
		t.Error("expected nil for ID -1")
	}
	// Out of range
	if ss.Style(99) != nil {
		t.Error("expected nil for ID 99")
	}
}

func TestSyntaxStyle_NilColors(t *testing.T) {
	ss := NewSyntaxStyle()

	id := ss.Register("plain", nil, nil, AttrNone)

	s := ss.Style(id)
	if s == nil {
		t.Fatal("expected style")
	}
	if s.Fg != nil {
		t.Error("expected nil Fg")
	}
	if s.Bg != nil {
		t.Error("expected nil Bg")
	}
}

func TestSyntaxStyle_IDsAreSequential(t *testing.T) {
	ss := NewSyntaxStyle()

	for i := range 50 {
		name := "style_" + string(rune('a'+i))
		id := ss.Register(name, nil, nil, AttrNone)
		if id != i+1 {
			t.Fatalf("expected sequential ID %d, got %d", i+1, id)
		}
	}
	if ss.StyleCount() != 50 {
		t.Errorf("expected 50 styles, got %d", ss.StyleCount())
	}
}
