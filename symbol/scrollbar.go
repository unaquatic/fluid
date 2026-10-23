package symbol

// ScrollBarSet contains the runes used to draw a scrollbar.
type ScrollBarSet struct {
	Track rune // The background track.
	Thumb rune // The movable thumb.
	Begin rune // The optional start cap; zero omits it.
	End   rune // The optional end cap; zero omits it.
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
