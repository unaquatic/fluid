package symbol

// ScrollBarSet contains the runes used to draw a scrollbar.
type ScrollBarSet struct {
	// Track is the background track.
	Track rune
	// Thumb is the movable thumb.
	Thumb rune
	// Begin is the optional start cap; zero omits it.
	Begin rune
	// End is the optional end cap; zero omits it.
	End rune
}

var (
	// ScrollBarVertical uses a full-block thumb on an empty track.
	ScrollBarVertical = ScrollBarSet{
		Track: ' ',
		Thumb: '█',
	}

	// ScrollBarHorizontal uses an upper-half-block thumb on an empty track.
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
