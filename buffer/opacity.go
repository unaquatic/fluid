package buffer

// opacityStack stores cumulative opacity multipliers. Each value is the
// product of its multiplier and all values below it.
type opacityStack struct {
	values []float32
}

// Push adds an opacity multiplier clamped to [0, 1].
func (s *opacityStack) Push(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	effective := opacity
	if len(s.values) > 0 {
		effective = s.values[len(s.values)-1] * opacity
	}
	s.values = append(s.values, effective)
}

// Pop removes the topmost opacity value.
func (s *opacityStack) Pop() {
	if len(s.values) > 0 {
		s.values = s.values[:len(s.values)-1]
	}
}

// Clear removes all stacked values.
func (s *opacityStack) Clear() {
	s.values = s.values[:0]
}

// Current returns the effective opacity, or 1 if the stack is empty.
func (s *opacityStack) Current() float32 {
	if len(s.values) == 0 {
		return 1.0
	}
	return s.values[len(s.values)-1]
}

// Len returns the number of values on the stack.
func (s *opacityStack) Len() int {
	return len(s.values)
}
