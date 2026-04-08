package buffer

import "testing"

func TestScissorStack_Empty(t *testing.T) {
	var s scissorStack
	if s.Len() != 0 {
		t.Errorf("empty stack Len() = %d, want 0", s.Len())
	}
	if s.Current() != nil {
		t.Error("empty stack Current() should be nil")
	}
	// No clipping: should contain any point
	if !s.Contains(0, 0) {
		t.Error("empty stack should contain (0,0)")
	}
	if !s.Contains(100, 200) {
		t.Error("empty stack should contain any point")
	}
}

func TestScissorStack_PushPop(t *testing.T) {
	var s scissorStack

	s.Push(5, 5, 10, 10)
	if s.Len() != 1 {
		t.Errorf("after push Len() = %d, want 1", s.Len())
	}
	r := s.Current()
	if r == nil {
		t.Fatal("Current() should not be nil after push")
	}
	if r.X != 5 || r.Y != 5 || r.Width != 10 || r.Height != 10 {
		t.Errorf("Current() = %+v, want {5 5 10 10}", *r)
	}

	// Point inside
	if !s.Contains(5, 5) {
		t.Error("should contain (5,5)")
	}
	if !s.Contains(14, 14) {
		t.Error("should contain (14,14)")
	}
	// Point outside
	if s.Contains(4, 5) {
		t.Error("should not contain (4,5)")
	}
	if s.Contains(15, 5) {
		t.Error("should not contain (15,5)")
	}

	s.Pop()
	if s.Len() != 0 {
		t.Errorf("after pop Len() = %d, want 0", s.Len())
	}
	if s.Current() != nil {
		t.Error("after pop Current() should be nil")
	}
}

func TestScissorStack_Intersection(t *testing.T) {
	var s scissorStack

	// First rect: (0,0)-(20,20)
	s.Push(0, 0, 20, 20)
	// Second rect: (10,10)-(30,30) -> intersection should be (10,10)-(20,20) = {10,10,10,10}
	s.Push(10, 10, 20, 20)

	if s.Len() != 2 {
		t.Errorf("Len() = %d, want 2", s.Len())
	}

	r := s.Current()
	if r == nil {
		t.Fatal("Current() should not be nil")
	}
	if r.X != 10 || r.Y != 10 || r.Width != 10 || r.Height != 10 {
		t.Errorf("intersection = %+v, want {10 10 10 10}", *r)
	}

	if !s.Contains(10, 10) {
		t.Error("should contain (10,10)")
	}
	if !s.Contains(19, 19) {
		t.Error("should contain (19,19)")
	}
	if s.Contains(5, 5) {
		t.Error("should not contain (5,5)")
	}
	if s.Contains(20, 20) {
		t.Error("should not contain (20,20)")
	}

	// Pop inner, should revert to outer
	s.Pop()
	r = s.Current()
	if r.X != 0 || r.Y != 0 || r.Width != 20 || r.Height != 20 {
		t.Errorf("after pop = %+v, want {0 0 20 20}", *r)
	}
}

func TestScissorStack_NonOverlapping(t *testing.T) {
	var s scissorStack

	s.Push(0, 0, 5, 5)
	s.Push(10, 10, 5, 5) // no overlap

	r := s.Current()
	if r == nil {
		t.Fatal("Current() should not be nil")
	}
	if !r.IsEmpty() {
		t.Errorf("non-overlapping intersection should be empty, got %+v", *r)
	}

	// Nothing should be contained
	if s.Contains(0, 0) {
		t.Error("should not contain any point in empty intersection")
	}
	if s.Contains(12, 12) {
		t.Error("should not contain any point in empty intersection")
	}
}

func TestScissorStack_Clear(t *testing.T) {
	var s scissorStack
	s.Push(0, 0, 10, 10)
	s.Push(5, 5, 10, 10)

	s.Clear()
	if s.Len() != 0 {
		t.Errorf("after Clear() Len() = %d, want 0", s.Len())
	}
	if s.Current() != nil {
		t.Error("after Clear() Current() should be nil")
	}
}

func TestScissorStack_PopEmpty(t *testing.T) {
	var s scissorStack
	// Should not panic
	s.Pop()
	if s.Len() != 0 {
		t.Error("Pop on empty stack should leave Len()=0")
	}
}
