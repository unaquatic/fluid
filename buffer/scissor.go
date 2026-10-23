package buffer

// scissorStack stores cumulative clipping rectangles. Each rectangle is the
// intersection of the requested region and all regions below it.
type scissorStack struct {
	rects []Rect
}

// Push adds a clipping rectangle intersected with the current region.
func (s *scissorStack) Push(x, y, w, h int) {
	newRect := Rect{X: x, Y: y, Width: w, Height: h}
	if len(s.rects) > 0 {
		// Store the effective region so reads remain constant time.
		newRect = s.rects[len(s.rects)-1].Intersect(newRect)
	}
	s.rects = append(s.rects, newRect)
}

// Pop removes the most recently pushed rectangle.
func (s *scissorStack) Pop() {
	if len(s.rects) > 0 {
		s.rects = s.rects[:len(s.rects)-1]
	}
}

// Clear removes all clipping rectangles.
func (s *scissorStack) Clear() {
	s.rects = s.rects[:0]
}

// Current returns the effective clipping rectangle, or nil when drawing is
// unrestricted.
func (s *scissorStack) Current() *Rect {
	if len(s.rects) == 0 {
		return nil
	}
	r := s.rects[len(s.rects)-1]
	return &r
}

// Contains reports whether (x, y) lies in the effective clipping region. It
// reports true when drawing is unrestricted.
func (s *scissorStack) Contains(x, y int) bool {
	if len(s.rects) == 0 {
		return true
	}
	return s.rects[len(s.rects)-1].Contains(x, y)
}

// Len returns the number of clipping rectangles on the stack.
func (s *scissorStack) Len() int {
	return len(s.rects)
}
