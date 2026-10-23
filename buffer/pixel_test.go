package buffer

import (
	"math"
	"testing"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Raw Data Access tests
// ---------------------------------------------------------------------------

func TestChars(t *testing.T) {
	b := New(3, 1)
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	b.SetCell(1, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	b.SetCell(2, 0, 'C', style.ColorWhite, style.ColorBlack, 0)

	chars := b.Chars()
	if len(chars) != 3 {
		t.Fatalf("Chars() len = %d, want 3", len(chars))
	}
	if chars[0] != 'A' || chars[1] != 'B' || chars[2] != 'C' {
		t.Errorf("Chars() = %v, want [A B C]", chars)
	}
}

func TestFgs(t *testing.T) {
	b := New(2, 1)
	b.SetCell(0, 0, 'A', style.ColorRed, style.ColorBlack, 0)
	b.SetCell(1, 0, 'B', style.ColorBlue, style.ColorBlack, 0)

	fgs := b.Fgs()
	if !fgs[0].Equals(style.ColorRed) {
		t.Errorf("Fgs()[0] = %v, want Red", fgs[0])
	}
	if !fgs[1].Equals(style.ColorBlue) {
		t.Errorf("Fgs()[1] = %v, want Blue", fgs[1])
	}
}

func TestBgs(t *testing.T) {
	b := New(2, 1)
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorRed, 0)
	b.SetCell(1, 0, 'B', style.ColorWhite, style.ColorGreen, 0)

	bgs := b.Bgs()
	if !bgs[0].Equals(style.ColorRed) {
		t.Errorf("Bgs()[0] = %v, want Red", bgs[0])
	}
	if !bgs[1].Equals(style.ColorGreen) {
		t.Errorf("Bgs()[1] = %v, want Green", bgs[1])
	}
}

func TestAttributes_Slice(t *testing.T) {
	b := New(2, 1)
	b.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, style.AttrBold)
	b.SetCell(1, 0, 'B', style.ColorWhite, style.ColorBlack, style.AttrItalic)

	attrs := b.Attributes()
	if attrs[0] != style.AttrBold {
		t.Errorf("Attributes()[0] = %d, want AttrBold", attrs[0])
	}
	if attrs[1] != style.AttrItalic {
		t.Errorf("Attributes()[1] = %d, want AttrItalic", attrs[1])
	}
}

func TestResolvedCharSize(t *testing.T) {
	b := New(5, 1)
	b.DrawText("Hi", 0, 0, style.ColorWhite) // H(1) + i(1) = 2 bytes, plus 3 spaces = 5 bytes
	got := b.ResolvedCharSize()
	if got != 5 {
		t.Errorf("ResolvedCharSize() = %d, want 5", got)
	}
}

func TestResolvedCharSize_WideChar(t *testing.T) {
	b := New(4, 1)
	b.DrawText("A\u4e16B", 0, 0, style.ColorWhite) // A(1) + 世(3 UTF-8 bytes, 2 cells, 1 continuation) + B(1) = 5 bytes text + 0 trailing
	got := b.ResolvedCharSize()
	// cells: A(1), 世(2 cells: wide + continuation), B(1), total display = 4 cells
	// Continuation cell is skipped, so: A(1 byte) + 世(3 bytes) + B(1 byte) = 5 bytes
	if got != 5 {
		t.Errorf("ResolvedCharSize() = %d, want 5", got)
	}
}

func TestWriteResolvedChars(t *testing.T) {
	b := New(5, 1)
	b.DrawText("Hello", 0, 0, style.ColorWhite)

	dst := make([]byte, 32)
	n := b.WriteResolvedChars(dst, false)
	got := string(dst[:n])
	if got != "Hello" {
		t.Errorf("WriteResolvedChars() = %q, want %q", got, "Hello")
	}
}

func TestWriteResolvedChars_WithLineBreaks(t *testing.T) {
	b := New(3, 2)
	b.DrawText("Hi", 0, 0, style.ColorWhite)
	b.DrawText("Go", 0, 1, style.ColorWhite)

	dst := make([]byte, 32)
	n := b.WriteResolvedChars(dst, true)
	got := string(dst[:n])
	// Row 0: "Hi " (3 chars), newline, Row 1: "Go " (3 chars)
	want := "Hi \nGo "
	if got != want {
		t.Errorf("WriteResolvedChars(linebreaks) = %q, want %q", got, want)
	}
}

func TestWriteResolvedChars_BufferTooSmall(t *testing.T) {
	b := New(10, 1)
	b.DrawText("HelloWorld", 0, 0, style.ColorWhite)

	dst := make([]byte, 5) // only room for 5 bytes
	n := b.WriteResolvedChars(dst, false)
	got := string(dst[:n])
	if got != "Hello" {
		t.Errorf("WriteResolvedChars(small buf) = %q, want %q", got, "Hello")
	}
}

// ---------------------------------------------------------------------------
// Pixel rendering helper tests
// ---------------------------------------------------------------------------

func TestColorDistanceSq(t *testing.T) {
	a := style.ColorFromInts(255, 0, 0, 255)
	b := style.ColorFromInts(0, 255, 0, 255)
	d := a.DistanceSq(b)
	// (255-0)^2 + (0-255)^2 + (0-0)^2 = 65025 + 65025 = 130050
	if d < 130000 || d > 130100 {
		t.Errorf("DistanceSq(red, green) = %f, want ~130050", d)
	}
}

func TestLuminance(t *testing.T) {
	white := style.ColorFromInts(255, 255, 255, 255)
	l := white.Luminance()
	if !floatEq(l, 1.0) {
		t.Errorf("luminance(white) = %f, want 1.0", l)
	}

	black := style.ColorFromInts(0, 0, 0, 255)
	l = black.Luminance()
	if !floatEq(l, 0.0) {
		t.Errorf("luminance(black) = %f, want 0.0", l)
	}
}

func TestRenderQuadrantBlock_AllSame(t *testing.T) {
	red := style.ColorFromInts(255, 0, 0, 255)
	pixels := [4]style.Color{red, red, red, red}

	result := renderQuadrantBlock(pixels)
	if result.Char != quadrantChars[15] {
		t.Errorf("all same: Char = %q, want full block %q", result.Char, quadrantChars[15])
	}
}

func TestRenderQuadrantBlock_TopBottomSplit(t *testing.T) {
	white := style.ColorFromInts(255, 255, 255, 255)
	black := style.ColorFromInts(0, 0, 0, 255)
	// TL=black, TR=black, BL=white, BR=white → upper half dark
	pixels := [4]style.Color{black, black, white, white}

	result := renderQuadrantBlock(pixels)
	// Bits: TL(8)+TR(4) = 12 → 0x2580 Upper Half Block ▀
	if result.Char != '\u2580' {
		t.Errorf("top/bottom split: Char = %q (%U), want '▀' (U+2580)", result.Char, result.Char)
	}
}

func TestGrayscaleChar(t *testing.T) {
	// intensity 0 → space
	if ch := grayscaleChar(0); ch != ' ' {
		t.Errorf("grayscaleChar(0) = %q, want ' '", ch)
	}
	// intensity 1 → last char in ramp
	runes := []rune(grayscaleRamp)
	if ch := grayscaleChar(1.0); ch != runes[len(runes)-1] {
		t.Errorf("grayscaleChar(1.0) = %q, want %q", ch, runes[len(runes)-1])
	}
}

func TestGetPixelColor_RGBA(t *testing.T) {
	data := []byte{255, 0, 0, 128} // R=255, G=0, B=0, A=128
	c := getPixelColor(data, 0, false)
	if c.R != 255 {
		t.Errorf("R = %d, want 255", c.R)
	}
	if c.G != 0 {
		t.Errorf("G = %d, want 0", c.G)
	}
	if c.A != 128 {
		t.Errorf("A = %d, want 128", c.A)
	}
}

func TestGetPixelColor_BGRA(t *testing.T) {
	data := []byte{0, 0, 255, 255} // B=0, G=0, R=255, A=255
	c := getPixelColor(data, 0, true)
	if c.R != 255 {
		t.Errorf("R = %d, want 255", c.R)
	}
	if c.B != 0 {
		t.Errorf("B = %d, want 0", c.B)
	}
}

func TestGetPixelColor_OOB(t *testing.T) {
	data := []byte{0, 0}
	c := getPixelColor(data, 0, false)
	// Out of bounds → transparent magenta
	if c.A != 0 {
		t.Errorf("OOB pixel A = %d, want 0", c.A)
	}
}

// ---------------------------------------------------------------------------
// DrawSuperSampleBuffer
// ---------------------------------------------------------------------------

func TestDrawSuperSampleBuffer_Solid(t *testing.T) {
	b := New(1, 1)
	// 2x2 pixels, all red Color
	pixelData := []byte{
		255, 0, 0, 255, 255, 0, 0, 255, // TL, TR
		255, 0, 0, 255, 255, 0, 0, 255, // BL, BR
	}
	b.DrawSuperSampleBuffer(0, 0, pixelData, PixelFormatRGBA, 8)

	c := b.Cell(0, 0)
	// All same color → full block
	if c.Char != '\u2588' {
		t.Errorf("solid supersample: Char = %q (%U), want full block", c.Char, c.Char)
	}
}

func TestDrawSuperSampleBuffer_OOB(t *testing.T) {
	b := New(1, 1)
	// Empty pixel data
	b.DrawSuperSampleBuffer(0, 0, []byte{}, PixelFormatRGBA, 8)
	// Should not panic, cell should have OOB pixel results
}

// ---------------------------------------------------------------------------
// DrawPackedBuffer
// ---------------------------------------------------------------------------

// packCell writes a packed cell using float32 LE format (external format).
func packCell(bg, fg style.Color, char uint32) []byte {
	data := make([]byte, 48)
	writeFloat32LE(data, 0, float32(bg.R)/255.0)
	writeFloat32LE(data, 4, float32(bg.G)/255.0)
	writeFloat32LE(data, 8, float32(bg.B)/255.0)
	writeFloat32LE(data, 12, float32(bg.A)/255.0)
	writeFloat32LE(data, 16, float32(fg.R)/255.0)
	writeFloat32LE(data, 20, float32(fg.G)/255.0)
	writeFloat32LE(data, 24, float32(fg.B)/255.0)
	writeFloat32LE(data, 28, float32(fg.A)/255.0)
	writeUint32LE(data, 32, char)
	return data
}

func writeFloat32LE(dst []byte, off int, v float32) {
	bits := math.Float32bits(v)
	dst[off] = byte(bits)
	dst[off+1] = byte(bits >> 8)
	dst[off+2] = byte(bits >> 16)
	dst[off+3] = byte(bits >> 24)
}

func writeUint32LE(dst []byte, off int, v uint32) {
	dst[off] = byte(v)
	dst[off+1] = byte(v >> 8)
	dst[off+2] = byte(v >> 16)
	dst[off+3] = byte(v >> 24)
}

func TestDrawPackedBuffer(t *testing.T) {
	b := New(2, 1)
	data := append(
		packCell(style.ColorBlack, style.ColorWhite, 'A'),
		packCell(style.ColorBlack, style.ColorRed, 'B')...,
	)

	b.DrawPackedBuffer(data, 0, 0, 2, 1)

	c0 := b.Cell(0, 0)
	if c0.Char != 'A' {
		t.Errorf("packed cell 0 Char = %q, want 'A'", c0.Char)
	}
	c1 := b.Cell(1, 0)
	if c1.Char != 'B' {
		t.Errorf("packed cell 1 Char = %q, want 'B'", c1.Char)
	}
}

func TestDrawPackedBuffer_InvalidChar(t *testing.T) {
	b := New(1, 1)
	data := packCell(style.ColorBlack, style.ColorWhite, 0) // null char → should become space
	b.DrawPackedBuffer(data, 0, 0, 1, 1)

	c := b.Cell(0, 0)
	if c.Char != ' ' {
		t.Errorf("null packed char = %q, want space", c.Char)
	}
}

func TestDrawPackedBuffer_ControlChar(t *testing.T) {
	b := New(1, 1)
	data := packCell(style.ColorBlack, style.ColorWhite, 10) // newline → should become full block
	b.DrawPackedBuffer(data, 0, 0, 1, 1)

	c := b.Cell(0, 0)
	if c.Char != '\u2588' {
		t.Errorf("control packed char = %q, want full block", c.Char)
	}
}

// ---------------------------------------------------------------------------
// DrawGrayscaleBuffer
// ---------------------------------------------------------------------------

func TestDrawGrayscaleBuffer(t *testing.T) {
	b := New(3, 1)
	intensities := []float32{0.0, 0.5, 1.0}

	b.DrawGrayscaleBuffer(0, 0, intensities, 3, 1, nil, nil)

	// intensity 0 → skipped (< 0.01)
	c0 := b.Cell(0, 0)
	if c0.Char != ' ' {
		t.Errorf("grayscale[0] = %q, want space (skipped)", c0.Char)
	}

	// intensity 0.5 → some mid-range character
	c1 := b.Cell(1, 0)
	if c1.Char == ' ' {
		t.Error("grayscale[1] should not be space for intensity 0.5")
	}

	// intensity 1.0 → last character in ramp
	c2 := b.Cell(2, 0)
	runes := []rune(grayscaleRamp)
	if c2.Char != runes[len(runes)-1] {
		t.Errorf("grayscale[2] = %q, want %q for intensity 1.0", c2.Char, runes[len(runes)-1])
	}
}

func TestDrawGrayscaleBuffer_CustomColors(t *testing.T) {
	b := New(1, 1)
	fg := style.ColorRed
	intensities := []float32{1.0}

	b.DrawGrayscaleBuffer(0, 0, intensities, 1, 1, &fg, nil)

	c := b.Cell(0, 0)
	// Fg should be based on red with intensity-modulated alpha
	if c.Fg.R != 255 {
		t.Errorf("custom fg R = %d, want 255", c.Fg.R)
	}
}

func TestDrawGrayscaleBuffer_NegativePos(t *testing.T) {
	b := New(3, 1)
	intensities := []float32{1.0, 1.0, 1.0}

	// posX=-1: first pixel should be clipped
	b.DrawGrayscaleBuffer(-1, 0, intensities, 3, 1, nil, nil)

	// Cell 0 should have the second intensity (sx=1)
	c0 := b.Cell(0, 0)
	if c0.Char == ' ' {
		t.Error("cell 0 should have content after negative offset adjustment")
	}
}

// ---------------------------------------------------------------------------
// DrawGrayscaleBufferSupersampled
// ---------------------------------------------------------------------------

func TestDrawGrayscaleBufferSupersampled(t *testing.T) {
	b := New(1, 1)
	// 2x2 source → 1x1 terminal cell
	intensities := []float32{
		0.8, 1.0,
		0.6, 0.8,
	}

	b.DrawGrayscaleBufferSupersampled(0, 0, intensities, 2, 2, nil, nil)

	c := b.Cell(0, 0)
	// avg = (0.8+1.0+0.6+0.8)/4 = 0.8
	if c.Char == ' ' {
		t.Error("supersampled grayscale should not be space for avg=0.8")
	}
}

func TestDrawGrayscaleBufferSupersampled_ZeroDim(t *testing.T) {
	b := New(5, 5)
	b.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
	// srcW=1 → termW=0 → should be no-op
	b.DrawGrayscaleBufferSupersampled(0, 0, []float32{1.0}, 1, 2, nil, nil)
	c := b.Cell(0, 0)
	if c.Char != 'X' {
		t.Error("zero termW should be no-op")
	}
}

// floatEq is declared in opacity_test.go
