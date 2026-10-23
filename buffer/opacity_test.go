package buffer

import (
	"math"
	"testing"
)

func floatEq(a, b float32) bool {
	return math.Abs(float64(a-b)) < 1e-6
}

func TestOpacityStack_Empty(t *testing.T) {
	var s opacityStack
	if s.Len() != 0 {
		t.Errorf("empty stack Len() = %d, want 0", s.Len())
	}
	if !floatEq(s.Current(), 1.0) {
		t.Errorf("empty stack Current() = %f, want 1.0", s.Current())
	}
}

func TestOpacityStack_PushPop(t *testing.T) {
	var s opacityStack

	s.Push(0.5)
	if !floatEq(s.Current(), 0.5) {
		t.Errorf("after Push(0.5) Current() = %f, want 0.5", s.Current())
	}

	s.Pop()
	if !floatEq(s.Current(), 1.0) {
		t.Errorf("after Pop Current() = %f, want 1.0", s.Current())
	}
}

func TestOpacityStack_Multiplication(t *testing.T) {
	var s opacityStack

	s.Push(0.5)
	s.Push(0.5)
	if !floatEq(s.Current(), 0.25) {
		t.Errorf("0.5 * 0.5 = %f, want 0.25", s.Current())
	}

	s.Push(0.5)
	if !floatEq(s.Current(), 0.125) {
		t.Errorf("0.5 * 0.5 * 0.5 = %f, want 0.125", s.Current())
	}

	s.Pop()
	if !floatEq(s.Current(), 0.25) {
		t.Errorf("after pop = %f, want 0.25", s.Current())
	}
}

func TestOpacityStack_Clamping(t *testing.T) {
	var s opacityStack

	s.Push(-0.5)
	if !floatEq(s.Current(), 0.0) {
		t.Errorf("negative should clamp to 0, got %f", s.Current())
	}

	s.Pop()
	s.Push(1.5)
	if !floatEq(s.Current(), 1.0) {
		t.Errorf(">1 should clamp to 1.0, got %f", s.Current())
	}
}

func TestOpacityStack_Clear(t *testing.T) {
	var s opacityStack
	s.Push(0.5)
	s.Push(0.3)

	s.Clear()
	if s.Len() != 0 {
		t.Errorf("after Clear() Len() = %d, want 0", s.Len())
	}
	if !floatEq(s.Current(), 1.0) {
		t.Errorf("after Clear() Current() = %f, want 1.0", s.Current())
	}
}

func TestOpacityStack_PopEmpty(t *testing.T) {
	var s opacityStack
	// Popping an empty stack is a safe no-op.
	s.Pop()
	if !floatEq(s.Current(), 1.0) {
		t.Error("Pop on empty stack should return 1.0")
	}
}

func TestOpacityStack_ZeroOpacity(t *testing.T) {
	var s opacityStack
	s.Push(0.0)
	if !floatEq(s.Current(), 0.0) {
		t.Errorf("zero opacity = %f, want 0.0", s.Current())
	}

	// Multiplying an effective opacity of zero keeps it at zero.
	s.Push(0.5)
	if !floatEq(s.Current(), 0.0) {
		t.Errorf("0.0 * 0.5 = %f, want 0.0", s.Current())
	}
}
