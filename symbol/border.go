package symbol

// BorderCharacters contains the line, corner, and junction runes needed to draw
// a rectangular grid.
type BorderCharacters struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
	TopT        rune // Joins left, right, and down.
	BottomT     rune // Joins left, right, and up.
	LeftT       rune // Joins up, down, and right.
	RightT      rune // Joins up, down, and left.
	Cross       rune // Joins all four directions.
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
