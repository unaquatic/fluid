package render

import (
	"cmp"
	"slices"
	"strings"
	"sync"
)

// SelectableRenderable exposes the geometry and text needed to order selected
// renderables and collect their text.
type SelectableRenderable interface {
	// X returns the absolute x-coordinate.
	X() int
	// Y returns the absolute y-coordinate.
	Y() int
	// Width returns the rendered width.
	Width() int
	// Height returns the rendered height.
	Height() int
	// SelectedText returns the selected text.
	SelectedText() string
}

// Point is a zero-based coordinate in absolute terminal-cell space.
type Point struct {
	// X is the column and Y the row.
	X, Y int
}

// SelectionBounds contains normalized absolute selection endpoints. Width and
// Height are the nonnegative differences between those inclusive endpoints.
type SelectionBounds struct {
	// X and Y identify the top-left endpoint.
	X, Y int
	// Width and Height are the distances to the bottom-right endpoint.
	Width, Height int
}

// Selection tracks an anchor, a moving focus, and the renderables covered by
// a text selection. Its methods are safe for concurrent use.
type Selection struct {
	anchor Point // The fixed absolute start point.
	focus  Point // The moving absolute end point.

	isActive   bool
	isDragging bool
	isStart    bool // Whether no drag update has followed Start.

	// The renderer updates these lists as selection bounds change.
	selectedRenderables []SelectableRenderable // Renderables fully contained by the selection.
	touchedRenderables  []SelectableRenderable // Renderables that overlap the selection.

	mu sync.Mutex
}

// NewSelection returns an inactive selection.
func NewSelection() *Selection {
	return &Selection{}
}

// Anchor returns the fixed selection start point.
func (s *Selection) Anchor() Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.anchor
}

// Focus returns the moving selection end point.
func (s *Selection) Focus() Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.focus
}

// SetFocus sets the selection end point.
func (s *Selection) SetFocus(p Point) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.focus = p
}

// Bounds returns normalized selection endpoints regardless of drag direction.
func (s *Selection) Bounds() SelectionBounds {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.computeBounds()
}

// computeBounds returns normalized bounds while the caller holds s.mu.
func (s *Selection) computeBounds() SelectionBounds {
	minX := s.anchor.X
	maxX := s.focus.X
	if minX > maxX {
		minX, maxX = maxX, minX
	}

	minY := s.anchor.Y
	maxY := s.focus.Y
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	return SelectionBounds{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// IsActive reports whether the selection is active.
func (s *Selection) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActive
}

// SetActive sets whether the selection is active.
func (s *Selection) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isActive = active
}

// IsDragging reports whether a drag is in progress.
func (s *Selection) IsDragging() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isDragging
}

// SetDragging sets whether a drag is in progress.
func (s *Selection) SetDragging(dragging bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isDragging = dragging
}

// IsStart reports whether the selection is awaiting its first drag update.
func (s *Selection) IsStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isStart
}

// SetStart sets whether the selection is awaiting its first drag update.
func (s *Selection) SetStart(start bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isStart = start
}

// Start begins an active drag with both endpoints at (x, y).
func (s *Selection) Start(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anchor = Point{X: x, Y: y}
	s.focus = Point{X: x, Y: y}
	s.isActive = true
	s.isDragging = true
	s.isStart = true
}

// Update moves the focus to (x, y).
func (s *Selection) Update(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.focus = Point{X: x, Y: y}
}

// Clear resets all selection state and tracked renderables.
func (s *Selection) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anchor = Point{}
	s.focus = Point{}
	s.isActive = false
	s.isDragging = false
	s.isStart = false
	s.selectedRenderables = nil
	s.touchedRenderables = nil
}

// SelectedRenderables returns a copy of the fully contained renderables.
func (s *Selection) SelectedRenderables() []SelectableRenderable {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.selectedRenderables)
}

// TouchedRenderables returns a copy of the overlapping renderables.
func (s *Selection) TouchedRenderables() []SelectableRenderable {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.touchedRenderables)
}

// UpdateSelectedRenderables replaces the fully contained renderable list. It
// retains the supplied slice.
func (s *Selection) UpdateSelectedRenderables(renderables []SelectableRenderable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedRenderables = renderables
}

// UpdateTouchedRenderables replaces the overlapping renderable list. It retains
// the supplied slice.
func (s *Selection) UpdateTouchedRenderables(renderables []SelectableRenderable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touchedRenderables = renderables
}

// SelectedText joins nonempty text from the tracked renderables in
// top-to-bottom, left-to-right order. It separates renderables with newlines
// and skips nil entries.
func (s *Selection) SelectedText() string {
	s.mu.Lock()
	// Copy the tracked values before calling into renderables without the lock.
	all := slices.Concat(s.selectedRenderables, s.touchedRenderables)
	s.mu.Unlock()
	all = slices.DeleteFunc(all, func(r SelectableRenderable) bool { return r == nil })

	var b strings.Builder
	for _, r := range sortByReadingOrder(all) {
		text := r.SelectedText()
		if text != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(text)
		}
	}
	return b.String()
}

// sortByReadingOrder returns a stable copy of renderables ordered from top to
// bottom, then left to right.
func sortByReadingOrder(renderables []SelectableRenderable) []SelectableRenderable {
	result := slices.Clone(renderables)
	slices.SortStableFunc(result, func(a, b SelectableRenderable) int {
		return cmp.Or(cmp.Compare(a.Y(), b.Y()), cmp.Compare(a.X(), b.X()))
	})
	return result
}

// LocalSelectionBounds contains selection endpoints mapped and clamped to a
// renderable's local coordinates.
type LocalSelectionBounds struct {
	// AnchorX and AnchorY identify the local start point.
	AnchorX, AnchorY int
	// FocusX and FocusY identify the local end point.
	FocusX, FocusY int
	// IsActive reports whether the source selection is active.
	IsActive bool
}

// ConvertGlobalToLocal maps an active selection into a renderable's local
// coordinate space. It returns nil when sel is inactive or does not intersect
// the renderable.
//
// localX and localY are the renderable's absolute position. width and height
// are its dimensions.
func ConvertGlobalToLocal(sel *Selection, localX, localY, width, height int) *LocalSelectionBounds {
	if sel == nil || width <= 0 || height <= 0 {
		return nil
	}

	sel.mu.Lock()
	active := sel.isActive
	bounds := sel.computeBounds()
	anchor := sel.anchor
	focus := sel.focus
	sel.mu.Unlock()

	if !active {
		return nil
	}

	// Selection endpoints are inclusive; renderable bounds are half-open.
	if bounds.X+bounds.Width < localX || bounds.X >= localX+width {
		return nil
	}
	if bounds.Y+bounds.Height < localY || bounds.Y >= localY+height {
		return nil
	}

	// Translate both endpoints from absolute to local coordinates.
	anchorX := anchor.X - localX
	anchorY := anchor.Y - localY
	focusX := focus.X - localX
	focusY := focus.Y - localY

	// Clamp both endpoints to valid local cells.
	anchorX = clamp(anchorX, 0, width-1)
	anchorY = clamp(anchorY, 0, height-1)
	focusX = clamp(focusX, 0, width-1)
	focusY = clamp(focusY, 0, height-1)

	return &LocalSelectionBounds{
		AnchorX:  anchorX,
		AnchorY:  anchorY,
		FocusX:   focusX,
		FocusY:   focusY,
		IsActive: true,
	}
}

// clamp restricts v to the inclusive range [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
