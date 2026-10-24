package style

import "testing"

func TestAttrsAllFlags(t *testing.T) {
	// Enabling every decoration sets all eight low bits.
	attr := AttrBold | AttrDim | AttrItalic | AttrUnderline |
		AttrBlink | AttrInverse | AttrHidden | AttrStrikethrough
	if attr != AttributeBaseMask {
		t.Errorf("all attributes should equal AttributeBaseMask (%d), got %d", AttributeBaseMask, attr)
	}
}

func TestAttrsBase(t *testing.T) {
	// Upper link-ID bits do not affect the extracted decorations.
	attr := AttrBold | AttrItalic | (42 << AttributeBaseBits)
	base := attr.Base()
	want := AttrBold | AttrItalic
	if base != want {
		t.Errorf("got %d, want %d", base, want)
	}
}

func TestAttrsWithLink(t *testing.T) {
	base := AttrBold | AttrUnderline
	linkID := uint32(12345)
	packed := base.WithLink(linkID)

	// Packing a link preserves the decoration bits.
	if gotBase := packed.Base(); gotBase != base {
		t.Errorf("base attrs: got %d, want %d", gotBase, base)
	}

	// The link ID round-trips through the upper bits.
	if gotLink := packed.LinkID(); gotLink != linkID {
		t.Errorf("link ID: got %d, want %d", gotLink, linkID)
	}
}

func TestAttrsWithLink_ZeroLinkID(t *testing.T) {
	base := AttrItalic
	packed := base.WithLink(0)

	if gotBase := packed.Base(); gotBase != base {
		t.Errorf("base: got %d, want %d", gotBase, base)
	}
	if gotLink := packed.LinkID(); gotLink != 0 {
		t.Errorf("link: got %d, want 0", gotLink)
	}
}

func TestAttrsWithLink_MaxLinkID(t *testing.T) {
	base := AttrBold
	// The largest 24-bit link ID fits without changing the decorations.
	maxLinkID := uint32((1 << 24) - 1)
	packed := base.WithLink(maxLinkID)

	if gotBase := packed.Base(); gotBase != base {
		t.Errorf("base: got %d, want %d", gotBase, base)
	}
	if gotLink := packed.LinkID(); gotLink != maxLinkID {
		t.Errorf("link: got %d, want %d", gotLink, maxLinkID)
	}
}

func TestAttrsHas(t *testing.T) {
	attr := AttrBold | AttrItalic | AttrUnderline

	if !attr.Has(AttrBold) {
		t.Error("should have Bold")
	}
	if !attr.Has(AttrItalic) {
		t.Error("should have Italic")
	}
	if !attr.Has(AttrUnderline) {
		t.Error("should have Underline")
	}
	if attr.Has(AttrDim) {
		t.Error("should not have Dim")
	}
	if attr.Has(AttrStrikethrough) {
		t.Error("should not have Strikethrough")
	}
}

func TestAttrsHas_None(t *testing.T) {
	if AttrNone.Has(AttrBold) {
		t.Error("AttrNone should not have Bold")
	}
}

func TestLinkIDRoundtrip(t *testing.T) {
	// Representative link IDs round-trip without affecting decorations.
	testIDs := []uint32{0, 1, 100, 1000, 65535, (1 << 24) - 1}
	bases := []Attrs{AttrNone, AttrBold, AttrBold | AttrItalic, AttributeBaseMask}

	for _, base := range bases {
		for _, id := range testIDs {
			packed := base.WithLink(id)
			if gotBase := packed.Base(); gotBase != base {
				t.Errorf("base roundtrip failed for base=%d, id=%d: got %d", base, id, gotBase)
			}
			if gotID := packed.LinkID(); gotID != id {
				t.Errorf("id roundtrip failed for base=%d, id=%d: got %d", base, id, gotID)
			}
		}
	}
}

func TestBitmaskConstants(t *testing.T) {
	// Every decoration flag occupies one distinct bit.
	attrs := []Attrs{AttrBold, AttrDim, AttrItalic, AttrUnderline, AttrBlink, AttrInverse, AttrHidden, AttrStrikethrough}
	for i, a := range attrs {
		if expected := Attrs(1 << i); a != expected {
			t.Errorf("attr[%d] = %d, want %d", i, a, expected)
		}
	}
}
