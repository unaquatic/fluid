package buffer

import (
	"math"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// PixelFormat identifies the channel order of four-byte pixel data.
type PixelFormat int

const (
	// PixelFormatBGRA stores channels in blue, green, red, alpha order.
	PixelFormatBGRA PixelFormat = iota
	// PixelFormatRGBA stores channels in red, green, blue, alpha order.
	PixelFormatRGBA
)

// quadrantChars maps a four-bit quadrant mask to a Unicode block rune. The
// bits from most to least significant represent top-left, top-right,
// bottom-left, and bottom-right.
var quadrantChars = symbol.QuadrantChars

// grayscaleRamp orders ASCII runes from lowest to highest intensity.
const grayscaleRamp = symbol.GrayscaleRamp

func closestColorIndex(pixel style.Color, a, b style.Color) int {
	if pixel.DistanceSq(a) <= pixel.DistanceSq(b) {
		return 0
	}
	return 1
}

// quadrantResult approximates four source pixels as one styled terminal cell.
type quadrantResult struct {
	Char rune
	Fg   style.Color
	Bg   style.Color
}

// renderQuadrantBlock approximates four pixels with a quadrant block rune and
// two colors. Pixels are ordered top-left, top-right, bottom-left, bottom-right.
func renderQuadrantBlock(pixels [4]style.Color) quadrantResult {
	// Use the most distant pair of colors as clustering candidates.
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

	// The darker candidate is the foreground represented by set mask bits.
	var dark, light style.Color
	if candA.Luminance() <= candB.Luminance() {
		dark, light = candA, candB
	} else {
		dark, light = candB, candA
	}

	// Assign each quadrant to its nearest candidate color.
	var bits uint8
	bitValues := [4]uint8{8, 4, 2, 1}
	for i := range 4 {
		if closestColorIndex(pixels[i], dark, light) == 0 {
			bits |= bitValues[i]
		}
	}

	all := pixels[:]
	switch bits {
	case 0: // All quadrants use the background candidate.
		return quadrantResult{Char: ' ', Fg: dark, Bg: style.ColorAverage(all)}
	case 15: // All quadrants use the foreground candidate.
		return quadrantResult{Char: quadrantChars[15], Fg: style.ColorAverage(all), Bg: light}
	default: // The block rune separates the two candidate colors.
		return quadrantResult{Char: quadrantChars[bits], Fg: dark, Bg: light}
	}
}

func getPixelColor(data []byte, idx int, isBGRA bool) style.Color {
	if idx+3 >= len(data) {
		return style.ColorFromInts(255, 0, 255, 0) // Transparent magenta marks missing pixels.
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

// DrawSuperSampleBuffer renders four-channel pixel data with Unicode quadrant
// blocks. Each terminal cell represents a 2 by 2 block of source pixels.
//
// pixelData must contain four bytes per pixel in the specified format.
// alignedBytesPerRow is the source row stride in bytes, including any padding.
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
			pixels[0] = getPixelColor(pixelData, tlIdx, isBGRA) // Top-left.
			pixels[1] = getPixelColor(pixelData, trIdx, isBGRA) // Top-right.
			pixels[2] = getPixelColor(pixelData, blIdx, isBGRA) // Bottom-left.
			pixels[3] = getPixelColor(pixelData, brIdx, isBGRA) // Bottom-right.

			result := renderQuadrantBlock(pixels)
			b.SetCellBlended(xCel, yCel, result.Char, result.Fg, result.Bg, 0)
		}
	}
}

// PackedCellSize is the size in bytes of one packed cell.
const PackedCellSize = 48

// DrawPackedBuffer renders precomputed cells from data. Each 48-byte cell
// contains four little-endian float32 background channels, four little-endian
// float32 foreground channels, a little-endian uint32 character, and 12 padding
// bytes. termW and termH specify the packed grid's dimensions.
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
			ch = 0x2588 // Replace unsupported control runes with a full block.
		}

		b.SetCellBlended(cellX, cellY, rune(ch), fg, bg, 0)
	}
}

// DrawGrayscaleBuffer renders a srcW by srcH grid of intensities as ASCII art,
// with one source value per terminal cell. Intensities are clamped to [0, 1].
// A nil foreground uses white, and a nil background uses transparency.
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
			cellFg := baseFg.WithAlpha(uint8(gray * float32(baseFg.A)))

			b.SetCellBlended(destX, destY, ch, cellFg, baseBg, 0)
		}
	}
}

// DrawGrayscaleBufferSupersampled renders a srcW by srcH intensity grid at half
// its width and height. Each terminal cell represents the average of a 2 by 2
// source block. A nil foreground uses white, and a nil background uses
// transparency.
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
			cellFg := baseFg.WithAlpha(uint8(gray * float32(baseFg.A)))

			b.SetCellBlended(destX, destY, ch, cellFg, baseBg, 0)
		}
	}
}

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
