package render

import (
	"sync"
	"testing"
)

// mockSelectableRenderable provides fixed geometry and selection text.
type mockSelectableRenderable struct {
	x, y, w, h   int
	selectedText string
}

func (m *mockSelectableRenderable) X() int               { return m.x }
func (m *mockSelectableRenderable) Y() int               { return m.y }
func (m *mockSelectableRenderable) Width() int           { return m.w }
func (m *mockSelectableRenderable) Height() int          { return m.h }
func (m *mockSelectableRenderable) SelectedText() string { return m.selectedText }

// Verify that mockSelectableRenderable implements SelectableRenderable.
var _ SelectableRenderable = (*mockSelectableRenderable)(nil)

func TestNew(t *testing.T) {
	sel := NewSelection()
	if sel == nil {
		t.Fatal("NewSelection() returned nil")
	}
	if sel.IsActive() {
		t.Error("new selection should not be active")
	}
	if sel.IsDragging() {
		t.Error("new selection should not be dragging")
	}
	if sel.IsStart() {
		t.Error("new selection should not be start")
	}
	if sel.Anchor() != (Point{}) {
		t.Errorf("anchor should be zero, got %v", sel.Anchor())
	}
	if sel.Focus() != (Point{}) {
		t.Errorf("focus should be zero, got %v", sel.Focus())
	}
}

func TestStartUpdateClear(t *testing.T) {
	sel := NewSelection()

	sel.Start(5, 10)
	if !sel.IsActive() {
		t.Error("selection should be active after Start")
	}
	if !sel.IsDragging() {
		t.Error("selection should be dragging after Start")
	}
	if !sel.IsStart() {
		t.Error("selection should be start after Start")
	}
	if sel.Anchor() != (Point{5, 10}) {
		t.Errorf("anchor = %v, want (5,10)", sel.Anchor())
	}
	if sel.Focus() != (Point{5, 10}) {
		t.Errorf("focus = %v, want (5,10)", sel.Focus())
	}

	// Update moves only the focus endpoint.
	sel.Update(20, 15)
	if sel.Anchor() != (Point{5, 10}) {
		t.Errorf("anchor should not change on Update, got %v", sel.Anchor())
	}
	if sel.Focus() != (Point{20, 15}) {
		t.Errorf("focus = %v, want (20,15)", sel.Focus())
	}

	sel.Clear()
	if sel.IsActive() {
		t.Error("selection should not be active after Clear")
	}
	if sel.IsDragging() {
		t.Error("selection should not be dragging after Clear")
	}
	if sel.IsStart() {
		t.Error("selection should not be start after Clear")
	}
	if sel.Anchor() != (Point{}) {
		t.Errorf("anchor should be zero after Clear, got %v", sel.Anchor())
	}
	if sel.Focus() != (Point{}) {
		t.Errorf("focus should be zero after Clear, got %v", sel.Focus())
	}
	if len(sel.SelectedRenderables()) != 0 {
		t.Error("selected renderables should be empty after Clear")
	}
	if len(sel.TouchedRenderables()) != 0 {
		t.Error("touched renderables should be empty after Clear")
	}
}

func TestBounds_TopLeftToBottomRight(t *testing.T) {
	sel := NewSelection()
	sel.Start(5, 3)
	sel.Update(15, 8)

	b := sel.Bounds()
	if b.X != 5 || b.Y != 3 || b.Width != 10 || b.Height != 5 {
		t.Errorf("bounds = %+v, want {5 3 10 5}", b)
	}
}

func TestBounds_BottomRightToTopLeft(t *testing.T) {
	sel := NewSelection()
	sel.Start(15, 8)
	sel.Update(5, 3)

	b := sel.Bounds()
	if b.X != 5 || b.Y != 3 || b.Width != 10 || b.Height != 5 {
		t.Errorf("bounds = %+v, want {5 3 10 5}", b)
	}
}

func TestBounds_SinglePoint(t *testing.T) {
	sel := NewSelection()
	sel.Start(7, 4)

	b := sel.Bounds()
	if b.X != 7 || b.Y != 4 || b.Width != 0 || b.Height != 0 {
		t.Errorf("bounds = %+v, want {7 4 0 0}", b)
	}
}

func TestBounds_HorizontalLine(t *testing.T) {
	sel := NewSelection()
	sel.Start(2, 5)
	sel.Update(10, 5)

	b := sel.Bounds()
	if b.X != 2 || b.Y != 5 || b.Width != 8 || b.Height != 0 {
		t.Errorf("bounds = %+v, want {2 5 8 0}", b)
	}
}

func TestBounds_VerticalLine(t *testing.T) {
	sel := NewSelection()
	sel.Start(5, 2)
	sel.Update(5, 10)

	b := sel.Bounds()
	if b.X != 5 || b.Y != 2 || b.Width != 0 || b.Height != 8 {
		t.Errorf("bounds = %+v, want {5 2 0 8}", b)
	}
}

func TestSetFocus(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.SetFocus(Point{X: 42, Y: 17})

	if sel.Focus() != (Point{42, 17}) {
		t.Errorf("focus = %v, want (42,17)", sel.Focus())
	}
}

func TestStateSetters(t *testing.T) {
	sel := NewSelection()

	sel.SetActive(true)
	if !sel.IsActive() {
		t.Error("SetActive(true) failed")
	}
	sel.SetActive(false)
	if sel.IsActive() {
		t.Error("SetActive(false) failed")
	}

	sel.SetDragging(true)
	if !sel.IsDragging() {
		t.Error("SetDragging(true) failed")
	}
	sel.SetDragging(false)
	if sel.IsDragging() {
		t.Error("SetDragging(false) failed")
	}

	sel.SetStart(true)
	if !sel.IsStart() {
		t.Error("SetStart(true) failed")
	}
	sel.SetStart(false)
	if sel.IsStart() {
		t.Error("SetStart(false) failed")
	}
}

func TestRenderableTracking(t *testing.T) {
	sel := NewSelection()

	r1 := &mockSelectableRenderable{x: 0, y: 0, w: 10, h: 5}
	r2 := &mockSelectableRenderable{x: 10, y: 0, w: 10, h: 5}
	r3 := &mockSelectableRenderable{x: 5, y: 5, w: 10, h: 5}

	sel.UpdateSelectedRenderables([]SelectableRenderable{r1, r2})
	sel.UpdateTouchedRenderables([]SelectableRenderable{r3})

	selected := sel.SelectedRenderables()
	if len(selected) != 2 {
		t.Fatalf("selected count = %d, want 2", len(selected))
	}
	if selected[0] != r1 || selected[1] != r2 {
		t.Error("selected renderables don't match")
	}

	touched := sel.TouchedRenderables()
	if len(touched) != 1 || touched[0] != r3 {
		t.Error("touched renderables don't match")
	}

	// Mutating the returned slice must not change the tracked list.
	selected[0] = nil
	if sel.SelectedRenderables()[0] == nil {
		t.Error("SelectedRenderables should return a copy")
	}
}

func TestGetSelectedText_ReadingOrder(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(50, 50)

	// Supply renderables out of reading order.
	r1 := &mockSelectableRenderable{x: 10, y: 5, w: 10, h: 1, selectedText: "second"}
	r2 := &mockSelectableRenderable{x: 0, y: 0, w: 10, h: 1, selectedText: "first"}
	r3 := &mockSelectableRenderable{x: 0, y: 10, w: 10, h: 1, selectedText: "third"}

	sel.UpdateSelectedRenderables([]SelectableRenderable{r1, r2})
	sel.UpdateTouchedRenderables([]SelectableRenderable{r3})

	text := sel.SelectedText()
	expected := "first\nsecond\nthird"
	if text != expected {
		t.Errorf("text = %q, want %q", text, expected)
	}
}

func TestGetSelectedText_Empty(t *testing.T) {
	sel := NewSelection()
	text := sel.SelectedText()
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
}

func TestGetSelectedText_NoText(t *testing.T) {
	sel := NewSelection()
	r1 := &mockSelectableRenderable{x: 0, y: 0, w: 10, h: 1, selectedText: ""}
	r2 := &mockSelectableRenderable{x: 0, y: 1, w: 10, h: 1, selectedText: ""}
	sel.UpdateSelectedRenderables([]SelectableRenderable{r1, r2})

	text := sel.SelectedText()
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
}

func TestGetSelectedText_NilItems(t *testing.T) {
	sel := NewSelection()
	r1 := &mockSelectableRenderable{x: 0, y: 0, w: 10, h: 1, selectedText: "hello"}
	// Nil entries contribute no text.
	sel.UpdateSelectedRenderables([]SelectableRenderable{nil, r1, nil})

	text := sel.SelectedText()
	if text != "hello" {
		t.Errorf("text = %q, want %q", text, "hello")
	}
}

func TestGetSelectedText_SameRow(t *testing.T) {
	sel := NewSelection()
	r1 := &mockSelectableRenderable{x: 20, y: 5, w: 10, h: 1, selectedText: "right"}
	r2 := &mockSelectableRenderable{x: 0, y: 5, w: 10, h: 1, selectedText: "left"}
	sel.UpdateSelectedRenderables([]SelectableRenderable{r1, r2})

	text := sel.SelectedText()
	if text != "left\nright" {
		t.Errorf("text = %q, want %q", text, "left\nright")
	}
}

func TestConvertGlobalToLocal_FullyInside(t *testing.T) {
	sel := NewSelection()
	sel.Start(12, 7)
	sel.Update(18, 9)

	// Map into a 20 by 10 renderable whose absolute origin is (10,5).
	lb := ConvertGlobalToLocal(sel, 10, 5, 20, 10)
	if lb == nil {
		t.Fatal("expected non-nil LocalSelectionBounds")
	}
	if !lb.IsActive {
		t.Error("expected IsActive = true")
	}
	// The local anchor is (12-10, 7-5) = (2,2).
	if lb.AnchorX != 2 || lb.AnchorY != 2 {
		t.Errorf("anchor = (%d,%d), want (2,2)", lb.AnchorX, lb.AnchorY)
	}
	// The local focus is (18-10, 9-5) = (8,4).
	if lb.FocusX != 8 || lb.FocusY != 4 {
		t.Errorf("focus = (%d,%d), want (8,4)", lb.FocusX, lb.FocusY)
	}
}

func TestConvertGlobalToLocal_PartialOverlap(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(15, 8)

	// Map into a 20 by 10 renderable whose absolute origin is (10,5).
	lb := ConvertGlobalToLocal(sel, 10, 5, 20, 10)
	if lb == nil {
		t.Fatal("expected non-nil LocalSelectionBounds")
	}
	// The negative local anchor clamps to the top-left cell.
	if lb.AnchorX != 0 || lb.AnchorY != 0 {
		t.Errorf("anchor = (%d,%d), want (0,0)", lb.AnchorX, lb.AnchorY)
	}
	// The local focus is (15-10, 8-5) = (5,3).
	if lb.FocusX != 5 || lb.FocusY != 3 {
		t.Errorf("focus = (%d,%d), want (5,3)", lb.FocusX, lb.FocusY)
	}
}

func TestConvertGlobalToLocal_ClampedMax(t *testing.T) {
	sel := NewSelection()
	sel.Start(5, 3)
	sel.Update(100, 100)

	// Map into a 10 by 5 renderable at the terminal origin.
	lb := ConvertGlobalToLocal(sel, 0, 0, 10, 5)
	if lb == nil {
		t.Fatal("expected non-nil LocalSelectionBounds")
	}
	// The anchor already lies within the renderable.
	if lb.AnchorX != 5 || lb.AnchorY != 3 {
		t.Errorf("anchor = (%d,%d), want (5,3)", lb.AnchorX, lb.AnchorY)
	}
	// The focus clamps to the bottom-right cell (9,4).
	if lb.FocusX != 9 || lb.FocusY != 4 {
		t.Errorf("focus = (%d,%d), want (9,4)", lb.FocusX, lb.FocusY)
	}
}

func TestConvertGlobalToLocal_NoIntersection(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(5, 5)

	// A renderable at (20,20) does not intersect this selection.
	lb := ConvertGlobalToLocal(sel, 20, 20, 10, 10)
	if lb != nil {
		t.Errorf("expected nil, got %+v", lb)
	}
}

func TestConvertGlobalToLocal_Inactive(t *testing.T) {
	sel := NewSelection()
	// A new selection is inactive.
	lb := ConvertGlobalToLocal(sel, 0, 0, 10, 10)
	if lb != nil {
		t.Error("expected nil for inactive selection")
	}
}

func TestConvertGlobalToLocal_Nil(t *testing.T) {
	lb := ConvertGlobalToLocal(nil, 0, 0, 10, 10)
	if lb != nil {
		t.Error("expected nil for nil selection")
	}
}

func TestConvertGlobalToLocal_ZeroSize(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(10, 10)

	lb := ConvertGlobalToLocal(sel, 5, 5, 0, 0)
	if lb != nil {
		t.Error("expected nil for zero-size renderable")
	}
}

func TestConvertGlobalToLocal_ReverseSelection(t *testing.T) {
	sel := NewSelection()
	sel.Start(15, 8)
	sel.Update(5, 3)

	// Map into a 20 by 10 renderable whose absolute origin is (3,2).
	lb := ConvertGlobalToLocal(sel, 3, 2, 20, 10)
	if lb == nil {
		t.Fatal("expected non-nil LocalSelectionBounds")
	}
	// The local anchor is (15-3, 8-2) = (12,6).
	if lb.AnchorX != 12 || lb.AnchorY != 6 {
		t.Errorf("anchor = (%d,%d), want (12,6)", lb.AnchorX, lb.AnchorY)
	}
	// The local focus is (5-3, 3-2) = (2,1).
	if lb.FocusX != 2 || lb.FocusY != 1 {
		t.Errorf("focus = (%d,%d), want (2,1)", lb.FocusX, lb.FocusY)
	}
}

func TestConvertGlobalToLocal_EdgeTouch(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(10, 10)

	// Inclusive selection endpoints intersect a renderable beginning at (10,10).
	lb := ConvertGlobalToLocal(sel, 10, 10, 5, 5)
	if lb == nil {
		t.Fatal("expected non-nil for edge-touching renderable")
	}
	// The anchor clamps to the renderable's top-left cell.
	if lb.AnchorX != 0 || lb.AnchorY != 0 {
		t.Errorf("anchor = (%d,%d), want (0,0)", lb.AnchorX, lb.AnchorY)
	}
	// The focus maps directly to the top-left cell.
	if lb.FocusX != 0 || lb.FocusY != 0 {
		t.Errorf("focus = (%d,%d), want (0,0)", lb.FocusX, lb.FocusY)
	}
}

func TestSortByReadingOrder(t *testing.T) {
	r1 := &mockSelectableRenderable{x: 10, y: 5}
	r2 := &mockSelectableRenderable{x: 0, y: 5}
	r3 := &mockSelectableRenderable{x: 0, y: 0}
	r4 := &mockSelectableRenderable{x: 5, y: 10}

	sorted := sortByReadingOrder([]SelectableRenderable{r1, r2, r3, r4})

	expected := []SelectableRenderable{r3, r2, r1, r4}
	for i, want := range expected {
		got := sorted[i]
		if got.X() != want.X() || got.Y() != want.Y() {
			t.Errorf("sorted[%d] = (%d,%d), want (%d,%d)", i, got.X(), got.Y(), want.X(), want.Y())
		}
	}
}

func TestSortByReadingOrder_Empty(t *testing.T) {
	sorted := sortByReadingOrder(nil)
	if len(sorted) != 0 {
		t.Errorf("expected empty, got %d items", len(sorted))
	}
}

func TestConcurrentAccess(t *testing.T) {
	sel := NewSelection()
	var wg sync.WaitGroup

	// Exercise state reads and writes concurrently.
	for i := range 100 {
		wg.Add(3)
		go func(x int) {
			defer wg.Done()
			sel.Start(x, x)
		}(i)
		go func(x int) {
			defer wg.Done()
			sel.Update(x*2, x*2)
			_ = sel.Bounds()
			_ = sel.IsActive()
			_ = sel.IsDragging()
			_ = sel.IsStart()
			_ = sel.Anchor()
			_ = sel.Focus()
		}(i)
		go func() {
			defer wg.Done()
			sel.Clear()
		}()
	}

	wg.Wait()
	// The race detector verifies synchronization.
}

func TestConcurrentGetSelectedText(t *testing.T) {
	sel := NewSelection()
	sel.Start(0, 0)
	sel.Update(100, 100)

	r1 := &mockSelectableRenderable{x: 0, y: 0, w: 10, h: 1, selectedText: "hello"}
	r2 := &mockSelectableRenderable{x: 0, y: 1, w: 10, h: 1, selectedText: "world"}
	sel.UpdateSelectedRenderables([]SelectableRenderable{r1, r2})

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = sel.SelectedText()
		})
	}
	wg.Wait()
}

func TestConvertGlobalToLocal_Concurrent(t *testing.T) {
	sel := NewSelection()
	sel.Start(5, 5)
	sel.Update(20, 20)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = ConvertGlobalToLocal(sel, 0, 0, 30, 30)
		})
	}
	wg.Wait()
}

func TestDragLifecycle(t *testing.T) {
	sel := NewSelection()

	// Mouse down starts an active drag.
	sel.Start(5, 10)
	sel.SetActive(true)
	sel.SetDragging(true)
	sel.SetStart(true)

	if !sel.IsActive() || !sel.IsDragging() || !sel.IsStart() {
		t.Fatal("state after start is incorrect")
	}

	// Mouse movement updates the focus and ends the initial state.
	sel.Update(15, 12)
	sel.SetStart(false)

	if sel.IsStart() {
		t.Error("should not be start after first drag update")
	}

	b := sel.Bounds()
	if b.Width != 10 || b.Height != 2 {
		t.Errorf("bounds = %+v, want width=10 height=2", b)
	}

	// Mouse up ends dragging but retains the active selection.
	sel.SetDragging(false)

	if !sel.IsActive() {
		t.Error("selection should remain active after mouse up")
	}
	if sel.IsDragging() {
		t.Error("should not be dragging after mouse up")
	}

	// Text remains available after dragging ends.
	r := &mockSelectableRenderable{x: 5, y: 10, w: 10, h: 3, selectedText: "selected text"}
	sel.UpdateSelectedRenderables([]SelectableRenderable{r})
	text := sel.SelectedText()
	if text != "selected text" {
		t.Errorf("text = %q, want %q", text, "selected text")
	}

	// Explicit clearing ends the selection.
	sel.Clear()
	if sel.IsActive() {
		t.Error("should not be active after Clear")
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want int
	}{
		{5, 0, 10, 5},   // Within the range.
		{-1, 0, 10, 0},  // Below the minimum.
		{15, 0, 10, 10}, // Above the maximum.
		{0, 0, 0, 0},    // All values are zero.
		{5, 5, 5, 5},    // The range contains one value.
	}

	for _, tt := range tests {
		got := clamp(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}
