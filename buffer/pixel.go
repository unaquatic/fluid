package buffer

import (
	"math"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// PixelFormat describes the byte layout of pixel data.
type PixelFormat int

const (
	// PixelFormatBGRA is BGRA byte order (Blue, Green, Red, Alpha).
	PixelFormatBGRA PixelFormat = iota
	// PixelFormatRGBA is Color byte order (Red, Green, Blue, Alpha).
	PixelFormatRGBA
)

// quadrantChars is an alias for symbols.QuadrantChars. It maps a 4-bit mask
// (TL=8, TR=4, BL=2, BR=1) to a Unicode quadrant block character.
var quadrantChars = symbol.QuadrantChars

// grayscaleRamp maps intensity 0.0–1.0 to an ASCII art character.
const grayscaleRamp = symbol.GrayscaleRamp

// ---------------------------------------------------------------------------
// Sub-cell pixel helpers
// ---------------------------------------------------------------------------

func closestColorIndex(pixel style.Color, a, b style.Color) int {
	if pixel.DistanceSq(a) <= pixel.DistanceSq(b) {
		return 0
	}
	return 1
}

// quadrantResult holds the result of rendering 4 sub-pixels into one cell.
type quadrantResult struct {
	Char rune
	Fg   style.Color
	Bg   style.Color
}

// renderQuadrantBlock determines the best quadrant block character and two
// colors (fg/bg) to approximate 4 sub-pixels in a single terminal cell.
// pixels order: [TL, TR, BL, BR].
func renderQuadrantBlock(pixels [4]style.Color) quadrantResult {
	// 1. Find the most different pair of pixels.
	var idxA, idxB int
	maxDist := float32(-1)
	for i := range 4 {
		for j := i + 1; j < 4; j++ {
			d := pixels[i].DistanceSq(pixels[j])
			if d > maxDist {
				idxA, idxB = i, j
				maxDist = d
			}
		}
	}
	candA := pixels[idxA]
	candB := pixels[idxB]

	// 2. Assign dark/light based on luminance.
	var dark, light style.Color
	if candA.Luminance() <= candB.Luminance() {
		dark, light = candA, candB
	} else {
		dark, light = candB, candA
	}

	// 3. Classify each quadrant. Bit positions: TL=8, TR=4, BL=2, BR=1.
	var bits uint8
	bitValues := [4]uint8{8, 4, 2, 1}
	for i := range 4 {
		if closestColorIndex(pixels[i], dark, light) == 0 {
			bits |= bitValues[i]
		}
	}

	// 4. Construct result.
	all := pixels[:]
	switch bits {
	case 0: // all light
		return quadrantResult{Char: ' ', Fg: dark, Bg: style.ColorAverage(all)}
	case 15: // all dark
		return quadrantResult{Char: quadrantChars[15], Fg: style.ColorAverage(all), Bg: light}
	default: // mixed
		return quadrantResult{Char: quadrantChars[bits], Fg: dark, Bg: light}
	}
}

func getPixelColor(data []byte, idx int, isBGRA bool) style.Color {
	if idx+3 >= len(data) {
		return style.ColorFromInts(255, 0, 255, 0) // transparent magenta for OOB
	}
	if isBGRA {
		return style.ColorFromInts(data[idx+2], data[idx+1], data[idx], data[idx+3])
	}
	return style.ColorFromInts(data[idx], data[idx+1], data[idx+2], data[idx+3])
}

func grayscaleChar(intensity float32) rune {
	if intensity < 0.01 {
		return ' '
	}
	clamped := clampF(intensity, 0, 1)
	runes := []rune(grayscaleRamp)
	idx := int(clamped * float32(len(runes)-1))
	return runes[idx]
}

func clampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------------------------------------------------------------------------
// Public pixel drawing methods
// ---------------------------------------------------------------------------

// DrawSuperSampleBuffer renders raw RGBA/BGRA pixel data into the buffer
// using Unicode quadrant block characters. Each terminal cell represents a
// 2×2 block of source pixels, effectively doubling the resolution.
//
// pixelData must be in 4-bytes-per-pixel format. alignedBytesPerRow is the
// stride (bytes per source row, which may include padding).
//
// Go equivalent of bufferDrawSuperSampleBuffer.
func (b *Buffer) DrawSuperSampleBuffer(posX, posY int, pixelData []byte, format PixelFormat, alignedBytesPerRow int) {
	isBGRA := format == PixelFormatBGRA
	const bytesPerPixel = 4

	for yCel := posY; yCel < b.height; yCel++ {
		for xCel := posX; xCel < b.width; xCel++ {
			if b.isClipped(xCel, yCel) {
				continue
			}

			renderX := (xCel - posX) * 2
			renderY := (yCel - posY) * 2

			tlIdx := renderY*alignedBytesPerRow + renderX*bytesPerPixel
			trIdx := tlIdx + bytesPerPixel
			blIdx := (renderY+1)*alignedBytesPerRow + renderX*bytesPerPixel
			brIdx := blIdx + bytesPerPixel

			var pixels [4]style.Color
			pixels[0] = getPixelColor(pixelData, tlIdx, isBGRA) // TL
			pixels[1] = getPixelColor(pixelData, trIdx, isBGRA) // TR
			pixels[2] = getPixelColor(pixelData, blIdx, isBGRA) // BL
			pixels[3] = getPixelColor(pixelData, brIdx, isBGRA) // BR

			result := renderQuadrantBlock(pixels)
			b.SetCellBlended(xCel, yCel, result.Char, result.Fg, result.Bg, 0)
		}
	}
}

// PackedCellSize is the byte size of one packed cell result (bg[16] + fg[16] + char[4] + padding[12] = 48).
const PackedCellSize = 48

// DrawPackedBuffer renders pre-computed cell results from a packed byte buffer.
// Each cell is 48 bytes: bg Color (4×float32), fg Color (4×float32), char (uint32), padding.
// termW and termH are the logical grid dimensions of the packed data.
//
// Go equivalent of bufferDrawPackedBuffer.
func (b *Buffer) DrawPackedBuffer(data []byte, posX, posY, termW, termH int) {
	numCells := len(data) / PackedCellSize

	for i := range numCells {
		off := i * PackedCellSize
		cellX := posX + (i % termW)
		cellY := posY + (i / termW)

		if cellX >= termW || cellY >= termH {
			continue
		}
		if cellX >= b.width || cellY >= b.height {
			continue
		}
		if b.isClipped(cellX, cellY) {
			continue
		}

		bg := readPackedRGBA(data, off)
		fg := readPackedRGBA(data, off+16)
		ch := readPackedU32(data, off+32)

		if ch == 0 || ch > 0x10FFFF {
			ch = ' '
		}
		if ch < 32 || (ch > 126 && ch < 0x2580) {
			ch = 0x2588 // full block
		}

		b.SetCellBlended(cellX, cellY, rune(ch), fg, bg, 0)
	}
}

// DrawGrayscaleBuffer renders a grid of float32 intensity values (0.0–1.0)
// as ASCII-art characters. Each intensity value maps to one terminal cell.
// fg and bg are optional override colors; pass nil to use defaults
// (white fg, transparent bg).
//
// Go equivalent of bufferDrawGrayscaleBuffer.
func (b *Buffer) DrawGrayscaleBuffer(posX, posY int, intensities []float32, srcW, srcH int, fg, bg *style.Color) {
	if srcW == 0 || srcH == 0 {
		return
	}

	baseFg := style.ColorWhite
	if fg != nil {
		baseFg = *fg
	}
	baseBg := style.ColorTransparent
	if bg != nil {
		baseBg = *bg
	}

	opacity := b.opacity.Current()

	for sy := range srcH {
		destY := posY + sy
		if destY < 0 {
			continue
		}
		if destY >= b.height {
			break
		}
		for sx := range srcW {
			destX := posX + sx
			if destX < 0 {
				continue
			}
			if destX >= b.width {
				break
			}
			if b.isClipped(destX, destY) {
				continue
			}

			intensity := intensities[sy*srcW+sx]
			if intensity < 0.01 {
				continue
			}

			ch := grayscaleChar(intensity)
			gray := clampF(intensity, 0, 1)
			cellFg := baseFg.WithAlpha(uint8(gray * float32(baseFg.A) * opacity))

			b.SetCellBlended(destX, destY, ch, cellFg, baseBg, 0)
		}
	}
}

// DrawGrayscaleBufferSupersampled renders a grid of float32 intensity values
// where the source is 2× the terminal resolution. Each 2×2 block of source
// intensities is averaged into a single terminal cell.
//
// Go equivalent of bufferDrawGrayscaleBufferSupersampled.
func (b *Buffer) DrawGrayscaleBufferSupersampled(posX, posY int, intensities []float32, srcW, srcH int, fg, bg *style.Color) {
	termW := srcW / 2
	termH := srcH / 2
	if termW == 0 || termH == 0 {
		return
	}

	baseFg := style.ColorWhite
	if fg != nil {
		baseFg = *fg
	}
	baseBg := style.ColorTransparent
	if bg != nil {
		baseBg = *bg
	}

	opacity := b.opacity.Current()
	maxIdx := srcH * srcW

	for cellY := range termH {
		destY := posY + cellY
		if destY < 0 {
			continue
		}
		if destY >= b.height {
			break
		}
		for cellX := range termW {
			destX := posX + cellX
			if destX < 0 {
				continue
			}
			if destX >= b.width {
				break
			}
			if b.isClipped(destX, destY) {
				continue
			}

			qx := cellX * 2
			qy := cellY * 2

			tlIdx := qy*srcW + qx
			trIdx := qy*srcW + qx + 1
			blIdx := (qy+1)*srcW + qx
			brIdx := (qy+1)*srcW + qx + 1

			var tl, tr, bl, br float32
			if tlIdx < maxIdx {
				tl = intensities[tlIdx]
			}
			if trIdx < maxIdx && qx+1 < srcW {
				tr = intensities[trIdx]
			}
			if blIdx < maxIdx && qy+1 < srcH {
				bl = intensities[blIdx]
			}
			if brIdx < maxIdx && qx+1 < srcW && qy+1 < srcH {
				br = intensities[brIdx]
			}

			avg := (tl + tr + bl + br) / 4.0
			if avg < 0.01 {
				continue
			}

			ch := grayscaleChar(avg)
			gray := clampF(avg, 0, 1)
			cellFg := baseFg.WithAlpha(uint8(gray * float32(baseFg.A) * opacity))

			b.SetCellBlended(destX, destY, ch, cellFg, baseBg, 0)
		}
	}
}

// ---------------------------------------------------------------------------
// Packed data helpers
// ---------------------------------------------------------------------------

func readPackedU32(data []byte, off int) uint32 {
	return uint32(data[off]) | uint32(data[off+1])<<8 | uint32(data[off+2])<<16 | uint32(data[off+3])<<24
}

func readPackedRGBA(data []byte, off int) style.Color {
	r := math.Float32frombits(readPackedU32(data, off))
	g := math.Float32frombits(readPackedU32(data, off+4))
	b := math.Float32frombits(readPackedU32(data, off+8))
	a := math.Float32frombits(readPackedU32(data, off+12))
	return style.ColorFromFloats(r, g, b, a)
}
