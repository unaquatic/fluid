package symbol

// ---------------------------------------------------------------------------
// BorderCharacters
// ---------------------------------------------------------------------------

// BorderCharacters defines the 11 runes used to draw a box border.
type BorderCharacters struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
	TopT        rune // T-junction pointing down  ┬
	BottomT     rune // T-junction pointing up    ┴
	LeftT       rune // T-junction pointing right ├
	RightT      rune // T-junction pointing left  ┤
	Cross       rune // 4-way junction            ┼
}

// ---------------------------------------------------------------------------
// Preset border character sets
// ---------------------------------------------------------------------------

// SingleChars uses thin (single-line) box-drawing characters.
var SingleChars = BorderCharacters{
	TopLeft: '┌', TopRight: '┐', BottomLeft: '└', BottomRight: '┘',
	Horizontal: '─', Vertical: '│',
	TopT: '┬', BottomT: '┴', LeftT: '├', RightT: '┤', Cross: '┼',
}

// DoubleChars uses double-line box-drawing characters.
var DoubleChars = BorderCharacters{
	TopLeft: '╔', TopRight: '╗', BottomLeft: '╚', BottomRight: '╝',
	Horizontal: '═', Vertical: '║',
	TopT: '╦', BottomT: '╩', LeftT: '╠', RightT: '╣', Cross: '╬',
}

// RoundedChars uses single-line with rounded corners.
var RoundedChars = BorderCharacters{
	TopLeft: '╭', TopRight: '╮', BottomLeft: '╰', BottomRight: '╯',
	Horizontal: '─', Vertical: '│',
	TopT: '┬', BottomT: '┴', LeftT: '├', RightT: '┤', Cross: '┼',
}

// HeavyChars uses thick/bold box-drawing characters.
var HeavyChars = BorderCharacters{
	TopLeft: '┏', TopRight: '┓', BottomLeft: '┗', BottomRight: '┛',
	Horizontal: '━', Vertical: '┃',
	TopT: '┳', BottomT: '┻', LeftT: '┣', RightT: '┫', Cross: '╋',
}
