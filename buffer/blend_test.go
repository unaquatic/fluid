package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestBlendCell_OpaqueOverOpaque(t *testing.T) {
	dst := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	src := NewCell('B', style.ColorRed, style.ColorBlue, 0)

	result := BlendCell(dst, src)

	// An opaque source completely replaces the destination.
	if result.Char != 'B' {
		t.Errorf("Char = %q, want 'B'", result.Char)
	}
	if !result.Fg.Equals(style.ColorRed) {
		t.Errorf("Fg = %v, want Red", result.Fg)
	}
	if !result.Bg.Equals(style.ColorBlue) {
		t.Errorf("Bg = %v, want Blue", result.Bg)
	}
}

func TestBlendCell_TransparentOverOpaque(t *testing.T) {
	dst := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	src := NewCell(' ', style.ColorTransparent, style.ColorTransparent, 0)

	result := BlendCell(dst, src)

	// A transparent source preserves the destination character.
	if result.Char != 'A' {
		t.Errorf("Char = %q, want 'A'", result.Char)
	}
}

func TestBlendCell_SemiTransparent(t *testing.T) {
	dst := NewCell('A', style.ColorWhite, style.ColorFromInts(255, 0, 0, 255), 0)
	src := NewCell(' ', style.ColorTransparent, style.ColorFromInts(0, 0, 255, 128), 0)

	result := BlendCell(dst, src)

	// Perceptual alpha maps 128/255 to about 0.536, producing approximately
	// 118 red and 137 blue while retaining the source alpha of 128.

	if result.Bg.A != 128 {
		t.Errorf("Bg.A = %d, want 128", result.Bg.A)
	}

	// Allow for rounding around the expected perceptual blend.
	if result.Bg.R < 100 || result.Bg.R > 145 {
		t.Errorf("Bg.R = %d, want ~118 (perceptual blend)", result.Bg.R)
	}
	if result.Bg.B < 110 || result.Bg.B > 155 {
		t.Errorf("Bg.B = %d, want ~137 (perceptual blend)", result.Bg.B)
	}

	// An empty source preserves a single-column destination character.
	if result.Char != 'A' {
		t.Errorf("Char = %q, want 'A' (inherited from dst)", result.Char)
	}
}

func TestBlendCell_EmptySourceInheritsChar(t *testing.T) {
	// A translucent background with no character preserves the destination glyph.
	dst := NewCell('Z', style.ColorRed, style.ColorBlack, style.AttrBold)
	src := NewCell(0, style.ColorGreen, style.ColorFromInts(0, 0, 0, 128), 0)

	result := BlendCell(dst, src)
	if result.Char != 'Z' {
		t.Errorf("empty src should inherit dst Char, got %q", result.Char)
	}
	// The source background tints the preserved destination foreground.
	if result.Fg.R < 75 {
		t.Errorf("empty src should inherit blended Fg with dst.Fg Red component, got R=%d", result.Fg.R)
	}
}

func TestBlendCell_ZeroAlphaPreservesDst(t *testing.T) {
	dst := NewCell('D', style.ColorWhite, style.ColorFromInts(255, 0, 0, 255), 0)
	src := NewCell('S', style.ColorRed, style.ColorFromInts(0, 0, 0, 0), 0)

	result := BlendCell(dst, src)
	// A zero source alpha preserves the destination background.
	if result.Bg.R != 255 {
		t.Errorf("Bg.R = %d, want 255 (dst preserved)", result.Bg.R)
	}
}

func TestDrawFrameBuffer_NoAlpha(t *testing.T) {
	dst := New(10, 10)
	dst.Clear(style.ColorBlack)

	src := New(3, 3)
	src.SetCell(0, 0, 'A', style.ColorRed, style.ColorBlue, 0)
	src.SetCell(1, 0, 'B', style.ColorRed, style.ColorBlue, 0)
	src.SetCell(2, 0, 'C', style.ColorRed, style.ColorBlue, 0)

	dst.DrawFrameBuffer(2, 2, src)

	c := dst.Cell(2, 2)
	if c.Char != 'A' {
		t.Errorf("DrawFrameBuffer at (2,2): Char = %q, want 'A'", c.Char)
	}
	c = dst.Cell(3, 2)
	if c.Char != 'B' {
		t.Errorf("DrawFrameBuffer at (3,2): Char = %q, want 'B'", c.Char)
	}
	c = dst.Cell(4, 2)
	if c.Char != 'C' {
		t.Errorf("DrawFrameBuffer at (4,2): Char = %q, want 'C'", c.Char)
	}
}

func TestDrawFrameBuffer_WithAlpha(t *testing.T) {
	dst := New(10, 10)
	dst.Clear(style.ColorRed) // Start with an opaque red destination.

	src := New(3, 1)
	src.SetRespectAlpha(true) // Enable blending when src is copied.
	// Blend a translucent blue source background over red.
	src.SetCell(0, 0, ' ', style.ColorTransparent, style.ColorFromInts(0, 0, 255, 128), 0)

	dst.DrawFrameBuffer(0, 0, src)

	c := dst.Cell(0, 0)
	// Allow for rounding around the expected perceptual blend.
	if c.Bg.R < 100 || c.Bg.R > 145 {
		t.Errorf("blended Bg.R = %d, want ~118 (perceptual)", c.Bg.R)
	}
	if c.Bg.B < 110 || c.Bg.B > 155 {
		t.Errorf("blended Bg.B = %d, want ~137 (perceptual)", c.Bg.B)
	}
}

func TestDrawFrameBuffer_WithSrcRect(t *testing.T) {
	dst := New(10, 10)
	dst.Clear(style.ColorBlack)

	src := New(5, 5)
	src.SetCell(2, 2, 'X', style.ColorWhite, style.ColorRed, 0)

	// Copy only the one-cell source region at (2,2).
	dst.DrawFrameBufferRect(0, 0, src, Rect{X: 2, Y: 2, Width: 1, Height: 1})

	c := dst.Cell(0, 0)
	if c.Char != 'X' {
		t.Errorf("sub-region blit: Char = %q, want 'X'", c.Char)
	}
}

func TestDrawFrameBuffer_ScissorClipping(t *testing.T) {
	dst := New(10, 10)
	dst.Clear(style.ColorBlack)
	dst.PushScissorRect(5, 5, 5, 5) // Permit writes only in the bottom-right quadrant.

	src := New(3, 3)
	src.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorRed, 0)

	// The destination lies outside the clipping region.
	dst.DrawFrameBuffer(0, 0, src)

	c := dst.Cell(0, 0)
	if c.Char == 'Z' {
		t.Error("scissor should have blocked DrawFrameBuffer write at (0,0)")
	}

	dst.PopScissorRect()
}
