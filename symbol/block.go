package symbol

// Block element characters — Unicode quadrant, half-block, and full-block
// characters used for sub-cell pixel rendering and area fills.

// ---------------------------------------------------------------------------
// Individual block elements
// ---------------------------------------------------------------------------

const (
	FullBlock      = '█' // U+2588
	UpperHalfBlock = '▀' // U+2580
	LowerHalfBlock = '▄' // U+2584
	LeftHalfBlock  = '▌' // U+258C
	RightHalfBlock = '▐' // U+2590
)

// ---------------------------------------------------------------------------
// Quadrant block characters
// ---------------------------------------------------------------------------

// QuadrantChars maps a 4-bit mask to a Unicode quadrant block character.
// Bit positions: TL=8, TR=4, BL=2, BR=1. Set bits are drawn as foreground.
//
// Usage: QuadrantChars[mask] where mask = (TL<<3 | TR<<2 | BL<<1 | BR).
var QuadrantChars = [16]rune{
	' ', // 0000
	'▗', // 0001 BR
	'▖', // 0010 BL
	'▄', // 0011 Lower Half
	'▝', // 0100 TR
	'▐', // 0101 Right Half
	'▞', // 0110 TR+BL (diagonal)
	'▟', // 0111 TR+BL+BR
	'▘', // 1000 TL
	'▚', // 1001 TL+BR (diagonal)
	'▌', // 1010 Left Half
	'▙', // 1011 TL+BL+BR
	'▀', // 1100 Upper Half
	'▜', // 1101 TL+TR+BR
	'▛', // 1110 TL+TR+BL
	'█', // 1111 Full Block
}

// ---------------------------------------------------------------------------
// Grayscale ramp
// ---------------------------------------------------------------------------

// GrayscaleRamp maps intensity 0.0–1.0 to an ASCII-art character.
// Index into it with: runes[int(intensity * float64(len(runes)-1))].
const GrayscaleRamp = " .'^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
