package textbuf

// ---------------------------------------------------------------------------
// ExtmarksSnapshot captures extmark state for undo/redo.
// ---------------------------------------------------------------------------

// ExtmarksSnapshot is a deep-copy of all extmarks and the ID counter at a
// point in time. Used by ExtmarksHistory to implement undo/redo.
type ExtmarksSnapshot struct {
	extmarks map[int]Extmark
	nextID   int
}

// cloneExtmarks returns a deep copy of the map, including the Fg/Bg pointers.
// Data (interface{}) is intentionally shallow-copied.
func cloneExtmarks(src map[int]Extmark) map[int]Extmark {
	if src == nil {
		return nil
	}
	dst := make(map[int]Extmark, len(src))
	for k, v := range src {
		if v.Fg != nil {
			fg := *v.Fg
			v.Fg = &fg
		}
		if v.Bg != nil {
			bg := *v.Bg
			v.Bg = &bg
		}
		dst[k] = v
	}
	return dst
}

// ---------------------------------------------------------------------------
// ExtmarksHistory manages undo/redo stacks for extmark snapshots.
// ---------------------------------------------------------------------------

// ExtmarksHistory is a pair of stacks that track extmark state changes in
// lockstep with the EditBuffer's undo/redo system.
type ExtmarksHistory struct {
	undoStack []ExtmarksSnapshot
	redoStack []ExtmarksSnapshot
}

// SaveSnapshot pushes a deep-copy of the current extmarks onto the undo stack
// and clears the redo stack (any redo future is forked away).
func (h *ExtmarksHistory) SaveSnapshot(extmarks map[int]Extmark, nextID int) {
	h.undoStack = append(h.undoStack, ExtmarksSnapshot{
		extmarks: cloneExtmarks(extmarks),
		nextID:   nextID,
	})
	h.redoStack = nil
}

// Undo pops and returns the most recent snapshot from the undo stack, or nil.
func (h *ExtmarksHistory) Undo() *ExtmarksSnapshot {
	if len(h.undoStack) == 0 {
		return nil
	}
	snap := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	return &snap
}

// Redo pops and returns the most recent snapshot from the redo stack, or nil.
func (h *ExtmarksHistory) Redo() *ExtmarksSnapshot {
	if len(h.redoStack) == 0 {
		return nil
	}
	snap := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	return &snap
}

// PushUndo pushes a snapshot onto the undo stack (used by redo).
func (h *ExtmarksHistory) PushUndo(snap ExtmarksSnapshot) {
	h.undoStack = append(h.undoStack, snap)
}

// PushRedo pushes a snapshot onto the redo stack (used by undo).
func (h *ExtmarksHistory) PushRedo(snap ExtmarksSnapshot) {
	h.redoStack = append(h.redoStack, snap)
}

// Clear empties both stacks.
func (h *ExtmarksHistory) Clear() {
	h.undoStack = nil
	h.redoStack = nil
}

// CanUndo returns true if there is at least one snapshot on the undo stack.
func (h *ExtmarksHistory) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo returns true if there is at least one snapshot on the redo stack.
func (h *ExtmarksHistory) CanRedo() bool { return len(h.redoStack) > 0 }
