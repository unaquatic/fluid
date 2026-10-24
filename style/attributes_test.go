package style

import "testing"

func TestCreateTextAttributes_None(t *testing.T) {
	attr := CreateTextAttributes(AttributeOptions{})
	if attr != AttrNone {
		t.Errorf("got %d, want %d", attr, AttrNone)
	}
}

func TestCreateTextAttributes_Bold(t *testing.T) {
	attr := CreateTextAttributes(AttributeOptions{Bold: true})
	if attr != AttrBold {
		t.Errorf("got %d, want %d", attr, AttrBold)
	}
}

func TestCreateTextAttributes_Multiple(t *testing.T) {
	attr := CreateTextAttributes(AttributeOptions{
		Bold:      true,
		Italic:    true,
		Underline: true,
	})
	want := AttrBold | AttrItalic | AttrUnderline
	if attr != want {
		t.Errorf("got %d, want %d", attr, want)
	}
}

func TestCreateTextAttributes_All(t *testing.T) {
	attr := CreateTextAttributes(AttributeOptions{
		Bold:          true,
		Dim:           true,
		Italic:        true,
		Underline:     true,
		Blink:         true,
		Inverse:       true,
		Hidden:        true,
		Strikethrough: true,
	})
	want := AttrBold | AttrDim | AttrItalic | AttrUnderline |
		AttrBlink | AttrInverse | AttrHidden | AttrStrikethrough
	if attr != want {
		t.Errorf("got %d, want %d", attr, want)
	}
	// Enabling every decoration sets all eight low bits.
	if attr != AttributeBaseMask {
		t.Errorf("all attributes should equal AttributeBaseMask (%d), got %d", AttributeBaseMask, attr)
	}
}

func TestBaseAttributes(t *testing.T) {
	// Upper link-ID bits do not affect the extracted decorations.
	attr := AttrBold | AttrItalic | (42 << AttributeBaseBits)
	base := BaseAttributes(attr)
	want := AttrBold | AttrItalic
	if base != want {
		t.Errorf("got %d, want %d", base, want)
	}
}

func TestAttributesWithLink(t *testing.T) {
	base := AttrBold | AttrUnderline
	linkID := uint32(12345)
	packed := AttributesWithLink(base, linkID)

	// Packing a link preserves the decoration bits.
	gotBase := BaseAttributes(packed)
	if gotBase != base {
		t.Errorf("base attrs: got %d, want %d", gotBase, base)
	}

	// The link ID round-trips through the upper bits.
	gotLink := LinkID(packed)
	if gotLink != linkID {
		t.Errorf("link ID: got %d, want %d", gotLink, linkID)
	}
}

func TestAttributesWithLink_ZeroLinkID(t *testing.T) {
	base := AttrItalic
	packed := AttributesWithLink(base, 0)

	gotBase := BaseAttributes(packed)
	if gotBase != base {
		t.Errorf("base: got %d, want %d", gotBase, base)
	}

	gotLink := LinkID(packed)
	if gotLink != 0 {
		t.Errorf("link: got %d, want 0", gotLink)
	}
}

func TestAttributesWithLink_MaxLinkID(t *testing.T) {
	base := AttrBold
	// The largest 24-bit link ID fits without changing the decorations.
	maxLinkID := uint32((1 << 24) - 1)
	packed := AttributesWithLink(base, maxLinkID)

	gotBase := BaseAttributes(packed)
	if gotBase != base {
		t.Errorf("base: got %d, want %d", gotBase, base)
	}

	gotLink := LinkID(packed)
	if gotLink != maxLinkID {
		t.Errorf("link: got %d, want %d", gotLink, maxLinkID)
	}
}

func TestHasAttribute(t *testing.T) {
	attr := AttrBold | AttrItalic | AttrUnderline

	if !HasAttribute(attr, AttrBold) {
		t.Error("should have Bold")
	}
	if !HasAttribute(attr, AttrItalic) {
		t.Error("should have Italic")
	}
	if !HasAttribute(attr, AttrUnderline) {
		t.Error("should have Underline")
	}
	if HasAttribute(attr, AttrDim) {
		t.Error("should not have Dim")
	}
	if HasAttribute(attr, AttrStrikethrough) {
		t.Error("should not have Strikethrough")
	}
}

func TestHasAttribute_None(t *testing.T) {
	if HasAttribute(AttrNone, AttrBold) {
		t.Error("AttrNone should not have Bold")
	}
}

func TestLinkIDRoundtrip(t *testing.T) {
	// Representative link IDs round-trip without affecting decorations.
	testIDs := []uint32{0, 1, 100, 1000, 65535, (1 << 24) - 1}
	bases := []uint32{AttrNone, AttrBold, AttrBold | AttrItalic, AttributeBaseMask}

	for _, base := range bases {
		for _, id := range testIDs {
			packed := AttributesWithLink(base, id)
			gotBase := BaseAttributes(packed)
			gotID := LinkID(packed)

			if gotBase != base {
				t.Errorf("base roundtrip failed for base=%d, id=%d: got %d", base, id, gotBase)
			}
			if gotID != id {
				t.Errorf("id roundtrip failed for base=%d, id=%d: got %d", base, id, gotID)
			}
		}
	}
}

func TestBitmaskConstants(t *testing.T) {
	// Every decoration flag occupies one distinct bit.
	attrs := []uint32{AttrBold, AttrDim, AttrItalic, AttrUnderline, AttrBlink, AttrInverse, AttrHidden, AttrStrikethrough}
	for i, a := range attrs {
		expected := uint32(1 << i)
		if a != expected {
			t.Errorf("attr[%d] = %d, want %d", i, a, expected)
		}
	}
}
