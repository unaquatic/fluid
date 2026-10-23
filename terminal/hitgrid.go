package terminal

import "github.com/unaquatic/fluid/buffer"

// HitGrid maps each terminal cell to the numeric ID of the last renderable
// registered there. A clipping stack restricts registration for clipped
// descendants.
type HitGrid struct {
	grid         []uint64 // IDs in row-major order.
	width        int
	height       int
	scissorStack []buffer.Rect
	dirty        bool
}

// NewHitGrid returns an empty hit grid with nonnegative dimensions.
func NewHitGrid(width, height int) *HitGrid {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &HitGrid{
		grid:   make([]uint64, width*height),
		width:  width,
		height: height,
	}
}

// Add assigns id to each in-bounds cell in the rectangle. Later calls replace
// earlier IDs in overlapping cells.
func (h *HitGrid) Add(x, y, w, hh int, id uint64) {
	for row := y; row < y+hh; row++ {
		for col := x; col < x+w; col++ {
			if col >= 0 && col < h.width && row >= 0 && row < h.height {
				h.grid[row*h.width+col] = id
				h.dirty = true
			}
		}
	}
}

// AddClipped assigns id within the intersection of the rectangle and active
// clipping region.
func (h *HitGrid) AddClipped(x, y, w, hh int, id uint64) {
	clip := h.activeScissor()
	if clip == nil {
		h.Add(x, y, w, hh, id)
		return
	}

	// Restrict the requested rectangle to the effective clip.
	x1 := maxInt(x, clip.X)
	y1 := maxInt(y, clip.Y)
	x2 := minInt(x+w, clip.X+clip.Width)
	y2 := minInt(y+hh, clip.Y+clip.Height)

	if x1 >= x2 || y1 >= y2 {
		return
	}
	h.Add(x1, y1, x2-x1, y2-y1, id)
}

// Check returns the renderable ID at (x, y), or zero outside the grid or where
// no renderable is registered.
func (h *HitGrid) Check(x, y int) uint64 {
	if x < 0 || x >= h.width || y < 0 || y >= h.height {
		return 0
	}
	return h.grid[y*h.width+x]
}

// Clear clears all entries and resets the dirty flag.
func (h *HitGrid) Clear() {
	for i := range h.grid {
		h.grid[i] = 0
	}
	h.dirty = false
}

// PushScissorRect adds a clipping rectangle intersected with the active region.
func (h *HitGrid) PushScissorRect(x, y, w, hh int) {
	rect := buffer.Rect{X: x, Y: y, Width: w, Height: hh}

	// Store the effective intersection for constant-time access.
	if len(h.scissorStack) > 0 {
		top := h.scissorStack[len(h.scissorStack)-1]
		rect = top.Intersect(rect)
	}

	h.scissorStack = append(h.scissorStack, rect)
}

// PopScissorRect removes the most recently pushed clipping rectangle.
func (h *HitGrid) PopScissorRect() {
	if len(h.scissorStack) > 0 {
		h.scissorStack = h.scissorStack[:len(h.scissorStack)-1]
	}
}

// ClearScissorRects removes all clipping restrictions.
func (h *HitGrid) ClearScissorRects() {
	h.scissorStack = h.scissorStack[:0]
}

// IsDirty reports whether Add wrote an in-bounds cell since the last Clear or
// Resize.
func (h *HitGrid) IsDirty() bool {
	return h.dirty
}

// Resize changes the grid dimensions and clears all entries and clipping state.
func (h *HitGrid) Resize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	h.width = width
	h.height = height
	h.grid = make([]uint64, width*height)
	h.dirty = false
	h.scissorStack = h.scissorStack[:0]
}

// Width returns the grid width in cells.
func (h *HitGrid) Width() int { return h.width }

// Height returns the grid height in cells.
func (h *HitGrid) Height() int { return h.height }

// activeScissor returns the effective clipping rectangle, or nil when
// unrestricted.
func (h *HitGrid) activeScissor() *buffer.Rect {
	if len(h.scissorStack) == 0 {
		return nil
	}
	r := h.scissorStack[len(h.scissorStack)-1]
	return &r
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
