package buffer

// scissorStack manages a stack of clipping rectangles.
// When a new rect is pushed, the effective clip is the intersection
// of the new rect with the current effective clip.
type scissorStack struct {
	rects []Rect
}

// Push pushes a clipping rect onto the stack. The effective clip
// becomes the intersection of this rect with the previous effective clip.
func (s *scissorStack) Push(x, y, w, h int) {
	newRect := Rect{X: x, Y: y, Width: w, Height: h}
	if len(s.rects) > 0 {
		// Intersect with current effective clip
		newRect = s.rects[len(s.rects)-1].Intersect(newRect)
	}
	s.rects = append(s.rects, newRect)
}

// Pop removes the topmost rect from the stack.
func (s *scissorStack) Pop() {
	if len(s.rects) > 0 {
		s.rects = s.rects[:len(s.rects)-1]
	}
}

// Clear removes all rects from the stack.
func (s *scissorStack) Clear() {
	s.rects = s.rects[:0]
}

// Current returns the effective clipping rect (intersection of all stacked rects),
// or nil if the stack is empty (no clipping).
func (s *scissorStack) Current() *Rect {
	if len(s.rects) == 0 {
		return nil
	}
	r := s.rects[len(s.rects)-1]
	return &r
}

// Contains checks if a point is within the effective clip region.
// Returns true if the stack is empty (no clipping).
func (s *scissorStack) Contains(x, y int) bool {
	if len(s.rects) == 0 {
		return true
	}
	return s.rects[len(s.rects)-1].Contains(x, y)
}

// Len returns the number of rects on the stack.
func (s *scissorStack) Len() int {
	return len(s.rects)
}
