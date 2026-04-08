package terminal

import "testing"

// ---------------------------------------------------------------------------
// Basic Add/Check
// ---------------------------------------------------------------------------

func TestHitGrid_AddAndCheck(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.Add(10, 5, 20, 10, 42)

	// Inside the rect
	if id := g.Check(10, 5); id != 42 {
		t.Errorf("expected 42 at (10,5), got %d", id)
	}
	if id := g.Check(29, 14); id != 42 {
		t.Errorf("expected 42 at (29,14), got %d", id)
	}

	// Outside the rect
	if id := g.Check(9, 5); id != 0 {
		t.Errorf("expected 0 at (9,5), got %d", id)
	}
	if id := g.Check(30, 14); id != 0 {
		t.Errorf("expected 0 at (30,14), got %d", id)
	}
}

// ---------------------------------------------------------------------------
// Overlapping rects (later wins)
// ---------------------------------------------------------------------------

func TestHitGrid_OverlappingRects(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.Add(0, 0, 40, 12, 1)
	g.Add(10, 5, 30, 7, 2) // overlaps

	// In overlap region: later wins
	if id := g.Check(15, 8); id != 2 {
		t.Errorf("expected 2 in overlap, got %d", id)
	}

	// In first-only region
	if id := g.Check(5, 3); id != 1 {
		t.Errorf("expected 1 in first-only region, got %d", id)
	}
}

// ---------------------------------------------------------------------------
// Out of bounds
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Scissor clipping
// ---------------------------------------------------------------------------

func TestHitGrid_ScissorClipping(t *testing.T) {
	g := NewHitGrid(80, 24)

	// Push a scissor rect: only (10,5)-(30,15)
	g.PushScissorRect(10, 5, 20, 10)

	// AddClipped should respect scissor
	g.AddClipped(0, 0, 80, 24, 1)

	// Inside scissor
	if id := g.Check(15, 10); id != 1 {
		t.Errorf("expected 1 inside scissor, got %d", id)
	}

	// Outside scissor (but inside the added rect)
	if id := g.Check(5, 3); id != 0 {
		t.Errorf("expected 0 outside scissor, got %d", id)
	}
}

func TestHitGrid_NestedScissors(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.PushScissorRect(0, 0, 40, 20)  // outer
	g.PushScissorRect(10, 5, 20, 10) // inner (intersection: 10,5 to 30,15)

	g.AddClipped(0, 0, 80, 24, 1)

	// Inside inner scissor
	if id := g.Check(15, 10); id != 1 {
		t.Errorf("expected 1 inside nested scissor, got %d", id)
	}

	// Outside inner but inside outer
	if id := g.Check(5, 3); id != 0 {
		t.Errorf("expected 0 outside inner scissor, got %d", id)
	}

	// Pop inner
	g.PopScissorRect()

	// Now outer is active -- AddClipped should use outer
	g.AddClipped(0, 0, 80, 24, 2)

	if id := g.Check(5, 3); id != 2 {
		t.Errorf("expected 2 after popping inner scissor, got %d", id)
	}
}

func TestHitGrid_ClearScissorRects(t *testing.T) {
	g := NewHitGrid(80, 24)

	g.PushScissorRect(10, 5, 20, 10)
	g.ClearScissorRects()

	// AddClipped should behave as unclipped
	g.AddClipped(0, 0, 80, 24, 1)

	if id := g.Check(5, 3); id != 1 {
		t.Errorf("expected 1 after ClearScissorRects, got %d", id)
	}
}

// ---------------------------------------------------------------------------
// Resize
// ---------------------------------------------------------------------------

func TestHitGrid_Resize(t *testing.T) {
	g := NewHitGrid(80, 24)
	g.Add(0, 0, 80, 24, 1)

	g.Resize(40, 12)

	if g.Width() != 40 || g.Height() != 12 {
		t.Errorf("expected 40x12, got %dx%d", g.Width(), g.Height())
	}

	// Old data should be cleared
	if id := g.Check(0, 0); id != 0 {
		t.Errorf("expected 0 after Resize, got %d", id)
	}

	// New bounds should work
	g.Add(0, 0, 40, 12, 2)
	if id := g.Check(20, 6); id != 2 {
		t.Errorf("expected 2 in resized grid, got %d", id)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestHitGrid_ZeroSize(t *testing.T) {
	g := NewHitGrid(0, 0)
	g.Add(0, 0, 1, 1, 1) // should not panic
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
	// Add a rect that extends beyond the grid
	g.Add(8, 8, 5, 5, 42)

	// Inside grid
	if id := g.Check(9, 9); id != 42 {
		t.Errorf("expected 42 at (9,9), got %d", id)
	}
	// Outside grid
	if id := g.Check(10, 10); id != 0 {
		t.Errorf("expected 0 at (10,10), got %d", id)
	}
}
