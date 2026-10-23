package symbol

// Scrollbar symbol sets for vertical and horizontal scrollbars.

// ScrollBarSet defines the characters used to draw a scrollbar.
type ScrollBarSet struct {
	Track rune // background track character
	Thumb rune // thumb/handle character
	Begin rune // optional start cap (0 = none)
	End   rune // optional end cap (0 = none)
}

// Preset scrollbar character sets.
var (
	// ScrollBarVertical uses a block thumb on a space track.
	ScrollBarVertical = ScrollBarSet{
		Track: ' ',
		Thumb: '█',
	}

	// ScrollBarHorizontal uses upper-half block on a space track.
	ScrollBarHorizontal = ScrollBarSet{
		Track: ' ',
		Thumb: '▀',
	}

	// ScrollBarDoubleVertical uses double-line box-drawing characters.
	ScrollBarDoubleVertical = ScrollBarSet{
		Track: '║',
		Thumb: '█',
	}

	// ScrollBarDoubleHorizontal uses double-line box-drawing characters.
	ScrollBarDoubleHorizontal = ScrollBarSet{
		Track: '═',
		Thumb: '█',
	}
)
