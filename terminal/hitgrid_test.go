package terminal

import "testing"

func TestHitGrid_AddAndCheck(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.Add(10, 5, 20, 10, 42)

	// Cells inside the rectangle contain its ID.
	if id := g.Check(10, 5); id != 42 {
		t.Errorf("expected 42 at (10,5), got %d", id)
	}
	if id := g.Check(29, 14); id != 42 {
		t.Errorf("expected 42 at (29,14), got %d", id)
	}

	// Cells outside the rectangle remain empty.
	if id := g.Check(9, 5); id != 0 {
		t.Errorf("expected 0 at (9,5), got %d", id)
	}
	if id := g.Check(30, 14); id != 0 {
		t.Errorf("expected 0 at (30,14), got %d", id)
	}
}

func TestHitGrid_OverlappingRects(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.Add(0, 0, 40, 12, 1)
	g.Add(10, 5, 30, 7, 2) // Overlap the first rectangle.

	// The later registration wins in the overlap.
	if id := g.Check(15, 8); id != 2 {
		t.Errorf("expected 2 in overlap, got %d", id)
	}

	// Cells covered only by the first rectangle keep its ID.
	if id := g.Check(5, 3); id != 1 {
		t.Errorf("expected 1 in first-only region, got %d", id)
	}
}

func TestHitGrid_OutOfBounds(t *testing.T) {
	g := NewHitGrid(10, 10)

	g.Add(0, 0, 10, 10, 99)

	if id := g.Check(-1, 0); id != 0 {
		t.Errorf("expected 0 for negative x, got %d", id)
	}
	if id := g.Check(10, 0); id != 0 {
		t.Errorf("expected 0 for x=width, got %d", id)
	}
	if id := g.Check(0, -1); id != 0 {
		t.Errorf("expected 0 for negative y, got %d", id)
	}
	if id := g.Check(0, 10); id != 0 {
		t.Errorf("expected 0 for y=height, got %d", id)
	}
}

func TestHitGrid_Clear(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.Add(0, 0, 80, 24, 1)
	if !g.IsDirty() {
		t.Error("expected dirty after Add")
	}

	g.Clear()

	if id := g.Check(40, 12); id != 0 {
		t.Errorf("expected 0 after Clear, got %d", id)
	}
	if g.IsDirty() {
		t.Error("expected not dirty after Clear")
	}
}

func TestHitGrid_ScissorClipping(t *testing.T) {
	g := NewHitGrid(80, 24)

	// Restrict registration to x=[10,30) and y=[5,15).
	g.PushScissorRect(10, 5, 20, 10)

	// Register a larger rectangle through the active clip.
	g.AddClipped(0, 0, 80, 24, 1)

	// Cells inside the clip receive the ID.
	if id := g.Check(15, 10); id != 1 {
		t.Errorf("expected 1 inside scissor, got %d", id)
	}

	// Cells outside the clip remain empty despite the requested rectangle.
	if id := g.Check(5, 3); id != 0 {
		t.Errorf("expected 0 outside scissor, got %d", id)
	}
}

func TestHitGrid_NestedScissors(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.PushScissorRect(0, 0, 40, 20)  // Outer clip.
	g.PushScissorRect(10, 5, 20, 10) // Effective inner clip: x=[10,30), y=[5,15).

	g.AddClipped(0, 0, 80, 24, 1)

	// Cells inside the nested clip receive the ID.
	if id := g.Check(15, 10); id != 1 {
		t.Errorf("expected 1 inside nested scissor, got %d", id)
	}

	// The inner clip excludes cells that only the outer clip contains.
	if id := g.Check(5, 3); id != 0 {
		t.Errorf("expected 0 outside inner scissor, got %d", id)
	}

	// Removing the inner clip restores the outer clip.
	g.PopScissorRect()

	// Registration now uses the restored outer clip.
	g.AddClipped(0, 0, 80, 24, 2)

	if id := g.Check(5, 3); id != 2 {
		t.Errorf("expected 2 after popping inner scissor, got %d", id)
	}
}

func TestHitGrid_ClearScissorRects(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.PushScissorRect(10, 5, 20, 10)
	g.ClearScissorRects()

	// Without a clipping region, AddClipped behaves like Add.
	g.AddClipped(0, 0, 80, 24, 1)

	if id := g.Check(5, 3); id != 1 {
		t.Errorf("expected 1 after ClearScissorRects, got %d", id)
	}
}

func TestHitGrid_Resize(t *testing.T) {
	g := NewHitGrid(80, 24)
	g.Add(0, 0, 80, 24, 1)

	g.Resize(40, 12)

	if g.Width() != 40 || g.Height() != 12 {
		t.Errorf("expected 40x12, got %dx%d", g.Width(), g.Height())
	}

	// Resize discards previous IDs.
	if id := g.Check(0, 0); id != 0 {
		t.Errorf("expected 0 after Resize, got %d", id)
	}

	// Registrations use the new dimensions.
	g.Add(0, 0, 40, 12, 2)
	if id := g.Check(20, 6); id != 2 {
		t.Errorf("expected 2 in resized grid, got %d", id)
	}
}

func TestHitGrid_ZeroSize(t *testing.T) {
	g := NewHitGrid(0, 0)
	g.Add(0, 0, 1, 1, 1) // A zero-sized grid safely ignores registration.
	if id := g.Check(0, 0); id != 0 {
		t.Errorf("expected 0 for zero-size grid, got %d", id)
	}
}

func TestHitGrid_NegativeSize(t *testing.T) {
	g := NewHitGrid(-5, -5)
	if g.Width() != 0 || g.Height() != 0 {
		t.Errorf("expected 0x0 for negative size, got %dx%d", g.Width(), g.Height())
	}
}

func TestHitGrid_AddPartiallyOutOfBounds(t *testing.T) {
	g := NewHitGrid(10, 10)
	// Registration clips a rectangle that extends beyond the grid.
	g.Add(8, 8, 5, 5, 42)

	if id := g.Check(9, 9); id != 42 {
		t.Errorf("expected 42 at (9,9), got %d", id)
	}
	if id := g.Check(10, 10); id != 0 {
		t.Errorf("expected 0 at (10,10), got %d", id)
	}
}
