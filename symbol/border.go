package symbol

// BorderCharacters contains the line, corner, and junction runes needed to draw
// a rectangular grid.
type BorderCharacters struct {
	// TopLeft is the top-left corner.
	TopLeft rune
	// TopRight is the top-right corner.
	TopRight rune
	// BottomLeft is the bottom-left corner.
	BottomLeft rune
	// BottomRight is the bottom-right corner.
	BottomRight rune
	// Horizontal is the horizontal line.
	Horizontal rune
	// Vertical is the vertical line.
	Vertical rune
	// TopT joins left, right, and down.
	TopT rune
	// BottomT joins left, right, and up.
	BottomT rune
	// LeftT joins up, down, and right.
	LeftT rune
	// RightT joins up, down, and left.
	RightT rune
	// Cross joins all four directions.
	Cross rune
}

// SingleChars contains thin single-line border characters.
var SingleChars = BorderCharacters{
	TopLeft: '┌', TopRight: '┐', BottomLeft: '└', BottomRight: '┘',
	Horizontal: '─', Vertical: '│',
	TopT: '┬', BottomT: '┴', LeftT: '├', RightT: '┤', Cross: '┼',
}

// DoubleChars contains double-line border characters.
var DoubleChars = BorderCharacters{
	TopLeft: '╔', TopRight: '╗', BottomLeft: '╚', BottomRight: '╝',
	Horizontal: '═', Vertical: '║',
	TopT: '╦', BottomT: '╩', LeftT: '╠', RightT: '╣', Cross: '╬',
}

// RoundedChars contains thin lines with rounded corners.
var RoundedChars = BorderCharacters{
	TopLeft: '╭', TopRight: '╮', BottomLeft: '╰', BottomRight: '╯',
	Horizontal: '─', Vertical: '│',
	TopT: '┬', BottomT: '┴', LeftT: '├', RightT: '┤', Cross: '┼',
}

// HeavyChars contains heavy border characters.
var HeavyChars = BorderCharacters{
	TopLeft: '┏', TopRight: '┓', BottomLeft: '┗', BottomRight: '┛',
	Horizontal: '━', Vertical: '┃',
	TopT: '┳', BottomT: '┻', LeftT: '┣', RightT: '┫', Cross: '╋',
}
