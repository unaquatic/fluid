package render

import (
	"sort"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Minimal interfaces to avoid circular import with render package
// ---------------------------------------------------------------------------

// SelectableRenderable is the minimal interface a renderable must satisfy
// to participate in text selection. The selection system uses these methods
// to determine whether a widget overlaps the selection region and to
// extract its selected text.
type SelectableRenderable interface {
	// X returns the widget's absolute x position.
	X() int
	// Y returns the widget's absolute y position.
	Y() int
	// Width returns the widget's rendered width.
	Width() int
	// Height returns the widget's rendered height.
	Height() int
	// SelectedText returns the text currently selected within this widget.
	SelectedText() string
}

// ---------------------------------------------------------------------------
// Point
// ---------------------------------------------------------------------------

// Point represents a 2D coordinate in absolute terminal space (columns and
// rows, 0-indexed from the top-left corner).
type Point struct {
	X, Y int
}

// ---------------------------------------------------------------------------
// SelectionBounds
// ---------------------------------------------------------------------------

// SelectionBounds represents the normalized bounding rectangle of a text
// selection in absolute terminal coordinates. Width and Height are always
// non-negative regardless of the direction the user dragged.
type SelectionBounds struct {
	// X, Y is the top-left corner of the selection rectangle.
	X, Y int
	// Width, Height is the extent of the selection (always >= 0).
	Width, Height int
}

// ---------------------------------------------------------------------------
// Selection
// ---------------------------------------------------------------------------

// Selection represents a text selection across one or more renderables.
// It tracks an anchor (where the user started) and a focus (where the
// mouse currently is). The renderer drives this during mouse drag events;
// widgets query it in RenderSelf to highlight selected text. All methods
// are safe for concurrent use.
type Selection struct {
	anchor Point // Selection start point (absolute terminal coordinates)
	focus  Point // Selection end point (tracks mouse during drag)

	isActive   bool
	isDragging bool
	isStart    bool // Whether this is the beginning of a new selection

	// Renderables within the selection region.
	// Set by the renderer during selection update.
	selectedRenderables []any // Renderables fully contained in selection bounds
	touchedRenderables  []any // Renderables overlapping selection bounds

	mu sync.Mutex
}

// New creates a new Selection in the inactive state.
func NewSelection() *Selection {
	return &Selection{}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// Anchor returns the selection start point.
func (s *Selection) Anchor() Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.anchor
}

// Focus returns the selection end point.
func (s *Selection) Focus() Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.focus
}

// SetFocus updates the focus (end) point.
func (s *Selection) SetFocus(p Point) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.focus = p
}

// Bounds computes the bounding rectangle of the selection.
// The returned bounds are normalized: Width and Height are always >= 0,
// regardless of the direction the user dragged.
func (s *Selection) Bounds() SelectionBounds {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.computeBounds()
}

// computeBounds is the unlocked implementation of Bounds.
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

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

// IsActive returns whether the selection is currently active.
func (s *Selection) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActive
}

// SetActive sets the active state.
func (s *Selection) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isActive = active
}

// IsDragging returns whether the user is currently dragging.
func (s *Selection) IsDragging() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isDragging
}

// SetDragging sets the dragging state.
func (s *Selection) SetDragging(dragging bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isDragging = dragging
}

// IsStart returns whether this is the start of a new selection.
func (s *Selection) IsStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isStart
}

// SetStart sets the start state.
func (s *Selection) SetStart(start bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isStart = start
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Start begins a new selection at (x, y).
// Sets anchor and focus to the same point and marks the selection as active.
func (s *Selection) Start(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anchor = Point{X: x, Y: y}
	s.focus = Point{X: x, Y: y}
	s.isActive = true
	s.isDragging = true
	s.isStart = true
}

// Update moves the focus point during a drag.
func (s *Selection) Update(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.focus = Point{X: x, Y: y}
}

// Clear resets the selection to its inactive state.
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

// ---------------------------------------------------------------------------
// Renderable tracking
// ---------------------------------------------------------------------------

// SelectedRenderables returns the renderables fully contained within the
// selection bounds.
func (s *Selection) SelectedRenderables() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]any, len(s.selectedRenderables))
	copy(result, s.selectedRenderables)
	return result
}

// TouchedRenderables returns the renderables that overlap (but may not be
// fully contained in) the selection bounds.
func (s *Selection) TouchedRenderables() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]any, len(s.touchedRenderables))
	copy(result, s.touchedRenderables)
	return result
}

// UpdateSelectedRenderables sets the list of fully-selected renderables.
func (s *Selection) UpdateSelectedRenderables(renderables []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedRenderables = renderables
}

// UpdateTouchedRenderables sets the list of touched renderables.
func (s *Selection) UpdateTouchedRenderables(renderables []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touchedRenderables = renderables
}

// ---------------------------------------------------------------------------
// Text extraction
// ---------------------------------------------------------------------------

// SelectedText collects text from all selected and touched renderables,
// sorted in reading order (top-to-bottom, left-to-right).
// Only renderables implementing SelectableRenderable contribute text.
func (s *Selection) SelectedText() string {
	s.mu.Lock()
	// Collect all renderables that may contribute text
	all := make([]any, 0, len(s.selectedRenderables)+len(s.touchedRenderables))
	all = append(all, s.selectedRenderables...)
	all = append(all, s.touchedRenderables...)
	s.mu.Unlock()

	if len(all) == 0 {
		return ""
	}

	sorted := sortByReadingOrder(all)

	var b strings.Builder
	for _, r := range sorted {
		if sr, ok := r.(SelectableRenderable); ok {
			text := sr.SelectedText()
			if text != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(text)
			}
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Reading order sort
// ---------------------------------------------------------------------------

// sortByReadingOrder sorts renderables by their position in reading order:
// top-to-bottom, then left-to-right. Items that don't implement
// SelectableRenderable are placed at the end.
func sortByReadingOrder(renderables []any) []any {
	result := make([]any, len(renderables))
	copy(result, renderables)

	sort.SliceStable(result, func(i, j int) bool {
		ri, oki := result[i].(SelectableRenderable)
		rj, okj := result[j].(SelectableRenderable)

		if !oki && !okj {
			return false
		}
		if !oki {
			return false // non-selectable goes to end
		}
		if !okj {
			return true // selectable before non-selectable
		}

		if ri.Y() != rj.Y() {
			return ri.Y() < rj.Y()
		}
		return ri.X() < rj.X()
	})

	return result
}

// ---------------------------------------------------------------------------
// LocalSelectionBounds
// ---------------------------------------------------------------------------

// LocalSelectionBounds represents a selection mapped into a renderable's
// local coordinate space. Widgets receive this from ConvertGlobalToLocal
// and use it to determine which characters to highlight.
type LocalSelectionBounds struct {
	// AnchorX, AnchorY is the selection start position in local coordinates.
	AnchorX, AnchorY int
	// FocusX, FocusY is the selection end position in local coordinates.
	FocusX, FocusY int
	// IsActive is true when the selection is currently being dragged.
	IsActive bool
}

// ---------------------------------------------------------------------------
// Coordinate conversion
// ---------------------------------------------------------------------------

// ConvertGlobalToLocal converts global selection coordinates to a renderable's
// local coordinate space. Returns nil if the selection doesn't intersect the
// renderable's bounds.
//
// localX, localY are the renderable's absolute position on screen.
// width, height are the renderable's dimensions.
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

	// Check intersection between selection bounds and renderable rect.
	// Selection rect: [bounds.X, bounds.X+bounds.Width) x [bounds.Y, bounds.Y+bounds.Height)
	// Renderable rect: [localX, localX+width) x [localY, localY+height)
	if bounds.X+bounds.Width < localX || bounds.X >= localX+width {
		return nil
	}
	if bounds.Y+bounds.Height < localY || bounds.Y >= localY+height {
		return nil
	}

	// Translate anchor and focus to local coordinates
	anchorX := anchor.X - localX
	anchorY := anchor.Y - localY
	focusX := focus.X - localX
	focusY := focus.Y - localY

	// Clamp to renderable bounds [0, dimension)
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

// clamp restricts v to [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
