package textbuf

// ExtmarksSnapshot captures extmarks and the next ID for undo or redo.
type ExtmarksSnapshot struct {
	extmarks map[int]Extmark
	nextID   int
}

// cloneExtmarks copies the map and pointed-to colors. User Data remains a
// shallow copy.
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

// ExtmarksHistory tracks extmark snapshots alongside EditBuffer undo and redo.
type ExtmarksHistory struct {
	undoStack []ExtmarksSnapshot
	redoStack []ExtmarksSnapshot
}

// SaveSnapshot copies the current state onto the undo stack and clears redo.
func (h *ExtmarksHistory) SaveSnapshot(extmarks map[int]Extmark, nextID int) {
	h.undoStack = append(h.undoStack, ExtmarksSnapshot{
		extmarks: cloneExtmarks(extmarks),
		nextID:   nextID,
	})
	h.redoStack = nil
}

// Undo removes and returns the newest undo snapshot, or nil when empty.
func (h *ExtmarksHistory) Undo() *ExtmarksSnapshot {
	if len(h.undoStack) == 0 {
		return nil
	}
	snap := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	return &snap
}

// Redo removes and returns the newest redo snapshot, or nil when empty.
func (h *ExtmarksHistory) Redo() *ExtmarksSnapshot {
	if len(h.redoStack) == 0 {
		return nil
	}
	snap := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	return &snap
}

// PushUndo adds snap to the undo stack.
func (h *ExtmarksHistory) PushUndo(snap ExtmarksSnapshot) {
	h.undoStack = append(h.undoStack, snap)
}

// PushRedo adds snap to the redo stack.
func (h *ExtmarksHistory) PushRedo(snap ExtmarksSnapshot) {
	h.redoStack = append(h.redoStack, snap)
}

// Clear empties both stacks.
func (h *ExtmarksHistory) Clear() {
	h.undoStack = nil
	h.redoStack = nil
}

// CanUndo reports whether the undo stack is nonempty.
func (h *ExtmarksHistory) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo reports whether the redo stack is nonempty.
func (h *ExtmarksHistory) CanRedo() bool { return len(h.redoStack) > 0 }
