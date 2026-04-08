package terminal

import "github.com/unaquatic/fluid/buffer"

// HitGrid maps terminal cell coordinates to Renderable numeric IDs for O(1)
// mouse targeting. Each cell stores the ID of the last widget rendered into
// it (painter's algorithm). The renderer rebuilds the grid every frame during
// the render pass and queries it when dispatching mouse events. Supports a
// scissor stack so that widgets inside scroll containers only receive hits
// within their visible bounds.
type HitGrid struct {
	grid         []uint64 // flat array, width * height
	width        int
	height       int
	scissorStack []buffer.Rect
	dirty        bool
}

// NewHitGrid creates a hit grid of the given size.
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

// Add registers a rect in the hit grid with the given renderable ID.
// Later writes overwrite earlier ones (painter's algorithm: last rendered wins).
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

// AddClipped registers a rect, respecting the active scissor rect.
func (h *HitGrid) AddClipped(x, y, w, hh int, id uint64) {
	clip := h.activeScissor()
	if clip == nil {
		h.Add(x, y, w, hh, id)
		return
	}

	// Intersect the rect with the scissor
	x1 := maxInt(x, clip.X)
	y1 := maxInt(y, clip.Y)
	x2 := minInt(x+w, clip.X+clip.Width)
	y2 := minInt(y+hh, clip.Y+clip.Height)

	if x1 >= x2 || y1 >= y2 {
		return
	}
	h.Add(x1, y1, x2-x1, y2-y1, id)
}

// Check returns the renderable ID at (x, y), or 0 if none.
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

// PushScissorRect pushes a clipping rect onto the scissor stack.
func (h *HitGrid) PushScissorRect(x, y, w, hh int) {
	rect := buffer.Rect{X: x, Y: y, Width: w, Height: hh}

	// If there's already a scissor, intersect with it
	if len(h.scissorStack) > 0 {
		top := h.scissorStack[len(h.scissorStack)-1]
		rect = top.Intersect(rect)
	}

	h.scissorStack = append(h.scissorStack, rect)
}

// PopScissorRect pops the current scissor rect.
func (h *HitGrid) PopScissorRect() {
	if len(h.scissorStack) > 0 {
		h.scissorStack = h.scissorStack[:len(h.scissorStack)-1]
	}
}

// ClearScissorRects clears all scissor rects.
func (h *HitGrid) ClearScissorRects() {
	h.scissorStack = h.scissorStack[:0]
}

// IsDirty returns whether the grid changed since the last Clear.
func (h *HitGrid) IsDirty() bool {
	return h.dirty
}

// Resize resizes the grid, clearing all entries.
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

// Width returns the grid width.
func (h *HitGrid) Width() int { return h.width }

// Height returns the grid height.
func (h *HitGrid) Height() int { return h.height }

// activeScissor returns the current scissor rect, or nil if none.
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
