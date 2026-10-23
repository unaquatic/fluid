package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestBlendCell_OpaqueOverOpaque(t *testing.T) {
	dst := NewCell('A', style.ColorWhite, style.ColorBlack, 0)
	src := NewCell('B', style.ColorRed, style.ColorBlue, 0)

	result := BlendCell(dst, src)

	// Opaque source should fully cover destination
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

	// Transparent source: should keep destination
	if result.Char != 'A' {
		t.Errorf("Char = %q, want 'A'", result.Char)
	}
}

func TestBlendCell_SemiTransparent(t *testing.T) {
	dst := NewCell('A', style.ColorWhite, style.ColorFromInts(255, 0, 0, 255), 0)
	src := NewCell(' ', style.ColorTransparent, style.ColorFromInts(0, 0, 255, 128), 0)

	result := BlendCell(dst, src)

	// With perceptual alpha:
	// alpha=128/255≈0.502, perceptualAlpha = pow(0.502, 0.9) ≈ 0.536
	// blendedBg.R = 0*0.536 + 255*0.464 ≈ 118
	// blendedBg.B = 255*0.536 + 0*0.464 ≈ 137
	// finalBgAlpha = src.Bg.A = 128

	if result.Bg.A != 128 {
		t.Errorf("Bg.A = %d, want 128", result.Bg.A)
	}

	// Perceptual blend: R should be around 102-140
	if result.Bg.R < 100 || result.Bg.R > 145 {
		t.Errorf("Bg.R = %d, want ~118 (perceptual blend)", result.Bg.R)
	}
	if result.Bg.B < 110 || result.Bg.B > 155 {
		t.Errorf("Bg.B = %d, want ~137 (perceptual blend)", result.Bg.B)
	}

	// Source has empty char, dst has 'A' with width=1 → preserveChar=true
	if result.Char != 'A' {
		t.Errorf("Char = %q, want 'A' (inherited from dst)", result.Char)
	}
}

func TestBlendCell_EmptySourceInheritsChar(t *testing.T) {
	// When src bg has alpha, empty src char should inherit dst char
	dst := NewCell('Z', style.ColorRed, style.ColorBlack, style.AttrBold)
	src := NewCell(0, style.ColorGreen, style.ColorFromInts(0, 0, 0, 128), 0)

	result := BlendCell(dst, src)
	// src char=0, dst char='Z' width=1 → preserveChar=true
	if result.Char != 'Z' {
		t.Errorf("empty src should inherit dst Char, got %q", result.Char)
	}
	// When preserving char, fg = blendColors(src.Bg, dst.Fg)
	// The blended fg should be influenced by both src.Bg and dst.Fg (Red)
	if result.Fg.R < 75 {
		t.Errorf("empty src should inherit blended Fg with dst.Fg Red component, got R=%d", result.Fg.R)
	}
}

func TestBlendCell_ZeroAlphaPreservesDst(t *testing.T) {
	dst := NewCell('D', style.ColorWhite, style.ColorFromInts(255, 0, 0, 255), 0)
	src := NewCell('S', style.ColorRed, style.ColorFromInts(0, 0, 0, 0), 0)

	result := BlendCell(dst, src)
	// Zero src alpha → preserve dst bg
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
	dst.Clear(style.ColorRed) // all red bg

	src := New(3, 1)
	src.SetRespectAlpha(true) // src.respectAlpha triggers blending
	// Semi-transparent blue bg
	src.SetCell(0, 0, ' ', style.ColorTransparent, style.ColorFromInts(0, 0, 255, 128), 0)

	dst.DrawFrameBuffer(0, 0, src)

	c := dst.Cell(0, 0)
	// Perceptual alpha blending: R should be around 100-140, B around 115-155
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

	// Only blit sub-region {2,2,1,1}
	dst.DrawFrameBuffer(0, 0, src, Rect{X: 2, Y: 2, Width: 1, Height: 1})

	c := dst.Cell(0, 0)
	if c.Char != 'X' {
		t.Errorf("sub-region blit: Char = %q, want 'X'", c.Char)
	}
}

func TestDrawFrameBuffer_ScissorClipping(t *testing.T) {
	dst := New(10, 10)
	dst.Clear(style.ColorBlack)
	dst.PushScissorRect(5, 5, 5, 5) // only allow writes in bottom-right quadrant

	src := New(3, 3)
	src.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorRed, 0)

	// Blit at (0,0) which is outside scissor
	dst.DrawFrameBuffer(0, 0, src)

	c := dst.Cell(0, 0)
	if c.Char == 'Z' {
		t.Error("scissor should have blocked DrawFrameBuffer write at (0,0)")
	}

	dst.PopScissorRect()
}

func TestDrawFrameBuffer_OpacityApplied(t *testing.T) {
	dst := New(10, 1)
	dst.Clear(style.ColorBlack)

	src := New(1, 1)
	src.SetCell(0, 0, 'A', style.ColorFromInts(255, 255, 255, 255), style.ColorFromInts(255, 0, 0, 255), 0)

	dst.PushOpacity(0.5)
	dst.DrawFrameBuffer(0, 0, src)
	dst.PopOpacity()

	c := dst.Cell(0, 0)
	// With 0.5 opacity, A should be approximately 128
	if c.Fg.A < 125 || c.Fg.A > 130 {
		t.Errorf("with opacity 0.5, Fg.A = %d, want ~128", c.Fg.A)
	}
}
