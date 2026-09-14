package symbol

// Common full- and half-cell block elements.
const (
	FullBlock      = '█' // U+2588
	UpperHalfBlock = '▀' // U+2580
	LowerHalfBlock = '▄' // U+2584
	LeftHalfBlock  = '▌' // U+258C
	RightHalfBlock = '▐' // U+2590
)

// QuadrantChars maps a four-bit mask to a quadrant block rune. From most to
// least significant, the bits represent top-left, top-right, bottom-left, and
// bottom-right. A set bit uses the foreground color.
var QuadrantChars = [16]rune{
	' ', // 0000: empty.
	'▗', // 0001: bottom-right.
	'▖', // 0010: bottom-left.
	'▄', // 0011: lower half.
	'▝', // 0100: top-right.
	'▐', // 0101: right half.
	'▞', // 0110: top-right and bottom-left.
	'▟', // 0111: all except top-left.
	'▘', // 1000: top-left.
	'▚', // 1001: top-left and bottom-right.
	'▌', // 1010: left half.
	'▙', // 1011: all except top-right.
	'▀', // 1100: upper half.
	'▜', // 1101: all except bottom-left.
	'▛', // 1110: all except bottom-right.
	'█', // 1111: full block.
}

// GrayscaleRamp orders ASCII-art characters from lowest to highest visual
// density.
const GrayscaleRamp = " .'^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
