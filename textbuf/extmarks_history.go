package textbuf

import "maps"

// extmarksSnapshot captures extmarks and the next ID for undo or redo.
type extmarksSnapshot struct {
	extmarks map[int]Extmark
	nextID   int
}

// cloneExtmarks copies the map. User Data remains a shallow copy.
func cloneExtmarks(src map[int]Extmark) map[int]Extmark {
	return maps.Clone(src)
}

// extmarksHistory tracks extmark snapshots alongside EditBuffer undo and redo.
type extmarksHistory struct {
	undoStack []extmarksSnapshot
	redoStack []extmarksSnapshot
}

// SaveSnapshot copies the current state onto the undo stack and clears redo.
func (h *extmarksHistory) SaveSnapshot(extmarks map[int]Extmark, nextID int) {
	h.undoStack = append(h.undoStack, extmarksSnapshot{
		extmarks: cloneExtmarks(extmarks),
		nextID:   nextID,
	})
	h.redoStack = nil
}

// Undo removes and returns the newest undo snapshot, or nil when empty.
func (h *extmarksHistory) Undo() *extmarksSnapshot {
	if len(h.undoStack) == 0 {
		return nil
	}
	snap := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	return &snap
}

// Redo removes and returns the newest redo snapshot, or nil when empty.
func (h *extmarksHistory) Redo() *extmarksSnapshot {
	if len(h.redoStack) == 0 {
		return nil
	}
	snap := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	return &snap
}

// PushUndo adds snap to the undo stack.
func (h *extmarksHistory) PushUndo(snap extmarksSnapshot) {
	h.undoStack = append(h.undoStack, snap)
}

// PushRedo adds snap to the redo stack.
func (h *extmarksHistory) PushRedo(snap extmarksSnapshot) {
	h.redoStack = append(h.redoStack, snap)
}

// Clear empties both stacks.
func (h *extmarksHistory) Clear() {
	h.undoStack = nil
	h.redoStack = nil
}

// CanUndo reports whether the undo stack is nonempty.
func (h *extmarksHistory) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo reports whether the redo stack is nonempty.
func (h *extmarksHistory) CanRedo() bool { return len(h.redoStack) > 0 }
