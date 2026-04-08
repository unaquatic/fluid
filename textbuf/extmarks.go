package textbuf

import (
	"strings"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Extmark — a tracked range in the buffer
// ---------------------------------------------------------------------------

// Extmark is a position-tracked range in the edit buffer. Extmark offsets are
// display-width offsets (including +1 for each newline) — the same coordinate
// space used by EditBuffer.PositionToOffset / OffsetToPosition.
//
// Virtual extmarks represent content that is not part of the user's real text
// (e.g. ghost-text, inline completions). Cursor movement will skip over them.
type Extmark struct {
	ID         int
	Start      int  // display-width offset (inclusive)
	End        int  // display-width offset (exclusive)
	Virtual    bool // if true, cursor movement skips this range
	Fg         *style.Color
	Bg         *style.Color
	Attributes uint32 // text attributes (lower 8 bits) + link ID (upper 24 bits)
	Priority   int
	Data       any // arbitrary user data
	TypeID     int // category identifier (0 = default)
}

// ExtmarkOptions are the parameters for creating an Extmark.
type ExtmarkOptions struct {
	Start      int
	End        int
	Virtual    bool
	Fg         *style.Color
	Bg         *style.Color
	Attributes uint32 // text attributes + link ID (via style.AttributesWithLink)
	Priority   int
	Data       any
	TypeID     int
	Metadata   any // stored separately, accessible via Metadata
}

// ---------------------------------------------------------------------------
// ExtmarksController
// ---------------------------------------------------------------------------

// ExtmarksController manages a set of extmarks that track positions through
// text edits, and optionally cause the cursor to skip over virtual ranges.
//
// It installs override functions on the EditBuffer and EditSurface so that
// every insert, delete, cursor movement, and undo/redo automatically adjusts
// extmark positions (via function fields).
type ExtmarksController struct {
	editBuffer  *EditBuffer
	editSurface *EditSurface

	extmarks       map[int]Extmark
	extmarksByType map[int]map[int]struct{} // typeID → set of extmark IDs
	metadata       map[int]any
	nextID         int
	destroyed      bool

	history ExtmarksHistory

	typeNameToID map[string]int
	typeIDToName map[int]string
	nextTypeID   int
}

// NewExtmarksController creates an ExtmarksController that intercepts
// operations on the given EditBuffer and EditSurface. Only one controller
// should be active per EditBuffer at a time.
func NewExtmarksController(eb *EditBuffer, ev *EditSurface) *ExtmarksController {
	ec := &ExtmarksController{
		editBuffer:     eb,
		editSurface:    ev,
		extmarks:       make(map[int]Extmark),
		extmarksByType: make(map[int]map[int]struct{}),
		metadata:       make(map[int]any),
		nextID:         1,
		typeNameToID:   make(map[string]int),
		typeIDToName:   make(map[int]string),
		nextTypeID:     1,
	}

	ec.installOverrides()
	return ec
}

// ---------------------------------------------------------------------------
// Public API — CRUD
// ---------------------------------------------------------------------------

// Create adds a new extmark and returns its unique ID.
// Returns -1 if the controller has been destroyed.
func (ec *ExtmarksController) Create(opts ExtmarkOptions) int {
	if ec.destroyed {
		return -1
	}

	id := ec.nextID
	ec.nextID++

	em := Extmark{
		ID:         id,
		Start:      opts.Start,
		End:        opts.End,
		Virtual:    opts.Virtual,
		Fg:         opts.Fg,
		Bg:         opts.Bg,
		Attributes: opts.Attributes,
		Priority:   opts.Priority,
		Data:       opts.Data,
		TypeID:     opts.TypeID,
	}
	ec.extmarks[id] = em

	if _, ok := ec.extmarksByType[opts.TypeID]; !ok {
		ec.extmarksByType[opts.TypeID] = make(map[int]struct{})
	}
	ec.extmarksByType[opts.TypeID][id] = struct{}{}

	if opts.Metadata != nil {
		ec.metadata[id] = opts.Metadata
	}

	ec.updateHighlights()
	return id
}

// Delete removes an extmark by ID. Returns true if it existed.
// Returns false if the controller has been destroyed.
func (ec *ExtmarksController) Delete(id int) bool {
	if ec.destroyed {
		return false
	}
	if _, ok := ec.extmarks[id]; !ok {
		return false
	}
	ec.deleteByID(id)
	ec.updateHighlights()
	return true
}

// Get returns the extmark with the given ID, or nil.
func (ec *ExtmarksController) Get(id int) *Extmark {
	if ec.destroyed {
		return nil
	}
	em, ok := ec.extmarks[id]
	if !ok {
		return nil
	}
	return &em
}

// All returns all extmarks.
func (ec *ExtmarksController) All() []Extmark {
	if ec.destroyed {
		return nil
	}
	out := make([]Extmark, 0, len(ec.extmarks))
	for _, em := range ec.extmarks {
		out = append(out, em)
	}
	return out
}

// Virtual returns only virtual extmarks.
func (ec *ExtmarksController) Virtual() []Extmark {
	if ec.destroyed {
		return nil
	}
	var out []Extmark
	for _, em := range ec.extmarks {
		if em.Virtual {
			out = append(out, em)
		}
	}
	return out
}

// AtOffset returns all extmarks whose range contains offset.
func (ec *ExtmarksController) AtOffset(offset int) []Extmark {
	if ec.destroyed {
		return nil
	}
	var out []Extmark
	for _, em := range ec.extmarks {
		if offset >= em.Start && offset < em.End {
			out = append(out, em)
		}
	}
	return out
}

// AllForTypeID returns extmarks with the given type ID.
func (ec *ExtmarksController) AllForTypeID(typeID int) []Extmark {
	if ec.destroyed {
		return nil
	}
	ids, ok := ec.extmarksByType[typeID]
	if !ok {
		return nil
	}
	out := make([]Extmark, 0, len(ids))
	for id := range ids {
		if em, ok := ec.extmarks[id]; ok {
			out = append(out, em)
		}
	}
	return out
}

// Clear removes all extmarks.
func (ec *ExtmarksController) Clear() {
	if ec.destroyed {
		return
	}
	ec.extmarks = make(map[int]Extmark)
	ec.extmarksByType = make(map[int]map[int]struct{})
	ec.metadata = make(map[int]any)
	ec.updateHighlights()
}

// ---------------------------------------------------------------------------
// Public API — Type registry
// ---------------------------------------------------------------------------

// RegisterType maps a human-readable name to a numeric type ID. Calling with
// the same name again returns the same ID.
// Returns -1 if the controller has been destroyed.
func (ec *ExtmarksController) RegisterType(name string) int {
	if ec.destroyed {
		return -1
	}
	if id, ok := ec.typeNameToID[name]; ok {
		return id
	}
	id := ec.nextTypeID
	ec.nextTypeID++
	ec.typeNameToID[name] = id
	ec.typeIDToName[id] = name
	return id
}

// TypeID returns the type ID for a name, or -1 if not registered.
func (ec *ExtmarksController) TypeID(name string) int {
	if ec.destroyed {
		return -1
	}
	if id, ok := ec.typeNameToID[name]; ok {
		return id
	}
	return -1
}

// TypeName returns the name for a type ID, or "".
func (ec *ExtmarksController) TypeName(typeID int) string {
	if ec.destroyed {
		return ""
	}
	return ec.typeIDToName[typeID]
}

// Metadata returns the metadata stored for an extmark, or nil.
func (ec *ExtmarksController) Metadata(id int) any {
	if ec.destroyed {
		return nil
	}
	return ec.metadata[id]
}

// ---------------------------------------------------------------------------
// Destroy
// ---------------------------------------------------------------------------

// Destroy removes all overrides and releases resources. After this call the
// controller is inert and the EditBuffer/EditSurface revert to default behavior.
func (ec *ExtmarksController) Destroy() {
	if ec.destroyed {
		return
	}

	eb := ec.editBuffer
	ev := ec.editSurface

	// Clear all overrides
	eb.overrideMoveCursorLeft = nil
	eb.overrideMoveCursorRight = nil
	eb.overrideSetCursorByOffset = nil
	eb.overrideInsertChar = nil
	eb.overrideInsertText = nil
	eb.overrideDeleteChar = nil
	eb.overrideDeleteCharBackward = nil
	eb.overrideDeleteRange = nil
	eb.overrideDeleteLine = nil
	eb.overrideNewLine = nil
	eb.overrideSetText = nil
	eb.overrideReplaceText = nil
	eb.overrideClear = nil
	eb.overrideUndo = nil
	eb.overrideRedo = nil

	ev.overrideMoveUpVisual = nil
	ev.overrideMoveDownVisual = nil
	ev.overrideDeleteSelectedText = nil

	ec.extmarks = nil
	ec.extmarksByType = nil
	ec.metadata = nil
	ec.typeNameToID = nil
	ec.typeIDToName = nil
	ec.history.Clear()
	ec.destroyed = true
}

// ---------------------------------------------------------------------------
// installOverrides — wires all interceptors
// ---------------------------------------------------------------------------

func (ec *ExtmarksController) installOverrides() {
	eb := ec.editBuffer
	ev := ec.editSurface

	// --- Cursor movement ---

	eb.overrideMoveCursorLeft = func() {
		if ec.destroyed {
			eb.doMoveCursorLeft()
			return
		}

		currentOffset := ev.VisualCursor().Offset
		targetOffset := currentOffset - 1
		if targetOffset < 0 {
			eb.doMoveCursorLeft()
			return
		}

		if vem := ec.findVirtualContaining(targetOffset); vem != nil && currentOffset >= vem.End {
			eb.doSetCursorByOffset(vem.Start)
			return
		}

		eb.doMoveCursorLeft()
	}

	eb.overrideMoveCursorRight = func() {
		if ec.destroyed {
			eb.doMoveCursorRight()
			return
		}

		currentOffset := ev.VisualCursor().Offset
		targetOffset := currentOffset + 1
		textLen := eb.tb.DisplayWidthLength()
		if targetOffset > textLen {
			eb.doMoveCursorRight()
			return
		}

		if vem := ec.findVirtualContaining(targetOffset); vem != nil && targetOffset > vem.Start {
			eb.doSetCursorByOffset(vem.End)
			return
		}

		eb.doMoveCursorRight()
	}

	eb.overrideSetCursorByOffset = func(offset int) {
		if ec.destroyed {
			eb.doSetCursorByOffset(offset)
			return
		}

		currentOffset := ev.VisualCursor().Offset
		movingForward := offset > currentOffset

		if movingForward {
			if vem := ec.findVirtualContaining(offset); vem != nil && offset > vem.Start {
				eb.doSetCursorByOffset(vem.End)
				return
			}
		} else {
			for _, em := range ec.extmarks {
				if em.Virtual && currentOffset >= em.End && offset < em.End && offset >= em.Start {
					eb.doSetCursorByOffset(em.Start)
					return
				}
			}
		}

		eb.doSetCursorByOffset(offset)
	}

	ev.overrideMoveUpVisual = func() {
		if ec.destroyed {
			ev.doMoveUpVisual()
			return
		}

		ev.doMoveUpVisual()
		newOffset := ev.VisualCursor().Offset

		if vem := ec.findVirtualContaining(newOffset); vem != nil {
			// Save desiredCol before the adjustment — SetCursorByOffset
			// triggers notifyCursor → OnCursorChange which clears it.
			savedDesiredCol := ev.desiredCol
			distToStart := newOffset - vem.Start
			distToEnd := vem.End - newOffset
			if distToStart < distToEnd {
				ev.SetCursorByOffset(vem.Start)
			} else {
				ev.SetCursorByOffset(vem.End)
			}
			ev.desiredCol = savedDesiredCol
		}
	}

	ev.overrideMoveDownVisual = func() {
		if ec.destroyed {
			ev.doMoveDownVisual()
			return
		}

		currentOffset := ev.VisualCursor().Offset
		ev.doMoveDownVisual()
		newOffset := ev.VisualCursor().Offset

		if vem := ec.findVirtualContaining(newOffset); vem != nil {
			// Save desiredCol before the adjustment — SetCursorByOffset
			// triggers notifyCursor → OnCursorChange which clears it.
			savedDesiredCol := ev.desiredCol
			distToStart := newOffset - vem.Start
			distToEnd := vem.End - newOffset
			if distToStart < distToEnd {
				adjusted := vem.Start
				if adjusted <= currentOffset {
					adjusted = vem.End
				}
				ev.SetCursorByOffset(adjusted)
			} else {
				ev.SetCursorByOffset(vem.End)
			}
			ev.desiredCol = savedDesiredCol
		}
	}

	// --- Deletion ---

	eb.overrideDeleteCharBackward = func() {
		if ec.destroyed {
			eb.doDeleteCharBackward()
			return
		}

		offsetBefore := ev.VisualCursor().Offset

		if offsetBefore == 0 || ev.HasSelection() {
			eb.doDeleteCharBackward()
			return
		}

		// Save snapshot only after confirming the deletion will happen.
		// doDeleteCharBackward is a no-op at offset 0 and won't push a
		// text undo — saving here prevents orphan extmarks snapshots
		// that desync the two undo stacks.
		ec.saveSnapshot()

		targetOffset := offsetBefore - 1
		if vem := ec.findVirtualContaining(targetOffset); vem != nil && offsetBefore == vem.End {
			// Delete the entire virtual extmark
			startPos := eb.OffsetToPosition(vem.Start)
			endPos := eb.OffsetToPosition(vem.End)
			delLen := vem.End - vem.Start
			ec.deleteByID(vem.ID)
			eb.doDeleteRange(startPos.Row, startPos.Col, endPos.Row, endPos.Col)
			ec.adjustAfterDeletion(vem.Start, delLen)
			ec.updateHighlights()
			return
		}

		eb.doDeleteCharBackward()
		offsetAfter := ev.VisualCursor().Offset
		deletedWidth := offsetBefore - offsetAfter
		ec.adjustAfterDeletion(offsetAfter, deletedWidth)
	}

	eb.overrideDeleteChar = func() {
		if ec.destroyed {
			eb.doDeleteChar()
			return
		}

		currentOffset := ev.VisualCursor().Offset
		textLen := eb.tb.DisplayWidthLength()

		if currentOffset >= textLen || ev.HasSelection() {
			eb.doDeleteChar()
			return
		}

		// Save snapshot only after confirming the deletion will happen.
		// doDeleteChar is a no-op at end of last line and won't push a
		// text undo — saving here prevents orphan extmarks snapshots
		// that desync the two undo stacks.
		ec.saveSnapshot()

		if vem := ec.findVirtualContaining(currentOffset); vem != nil && currentOffset == vem.Start {
			startPos := eb.OffsetToPosition(vem.Start)
			endPos := eb.OffsetToPosition(vem.End)
			delLen := vem.End - vem.Start
			ec.deleteByID(vem.ID)
			eb.doDeleteRange(startPos.Row, startPos.Col, endPos.Row, endPos.Col)
			ec.adjustAfterDeletion(vem.Start, delLen)
			ec.updateHighlights()
			return
		}

		// For forward delete the cursor stays put, so measure the total
		// display-width change to determine deleted width.
		widthBefore := eb.tb.DisplayWidthLength()
		eb.doDeleteChar()
		widthAfter := eb.tb.DisplayWidthLength()
		ec.adjustAfterDeletion(currentOffset, widthBefore-widthAfter)
	}

	eb.overrideDeleteRange = func(startRow, startCol, endRow, endCol int) {
		if ec.destroyed {
			eb.doDeleteRange(startRow, startCol, endRow, endCol)
			return
		}

		ec.saveSnapshot()
		startOff := eb.PositionToOffset(startRow, startCol)
		endOff := eb.PositionToOffset(endRow, endCol)
		length := endOff - startOff

		eb.doDeleteRange(startRow, startCol, endRow, endCol)
		ec.adjustAfterDeletion(startOff, length)
	}

	eb.overrideDeleteLine = func() {
		if ec.destroyed {
			eb.doDeleteLine()
			return
		}

		ec.saveSnapshot()

		row := eb.cursorRow
		lineCount := eb.tb.LineCount()
		lineW := eb.tb.LineWidthAt(row)

		// Compute deletion range in display-width offset space using the
		// buffer's line metadata instead of scanning the raw text string.
		var deleteOffset, deleteLen int
		if lineCount <= 1 {
			// Only line: content is cleared, no newlines involved.
			deleteOffset = 0
			deleteLen = lineW
		} else if row < lineCount-1 {
			// Not the last line: remove line content + trailing newline.
			deleteOffset = eb.positionToOffset(row, 0)
			deleteLen = lineW + 1
		} else {
			// Last line of multiple: remove preceding newline + line content.
			deleteOffset = eb.positionToOffset(row, 0) - 1
			deleteLen = lineW + 1
		}

		eb.doDeleteLine()
		if deleteLen > 0 {
			ec.adjustAfterDeletion(deleteOffset, deleteLen)
		}
	}

	// --- Insertion ---

	eb.overrideInsertText = func(text string) {
		if ec.destroyed {
			eb.doInsertText(text)
			return
		}
		ec.saveSnapshot()
		offsetBefore := ev.VisualCursor().Offset
		eb.doInsertText(text)
		offsetAfter := ev.VisualCursor().Offset
		ec.adjustAfterInsertion(offsetBefore, offsetAfter-offsetBefore)
	}

	eb.overrideInsertChar = func(ch rune) {
		if ec.destroyed {
			eb.doInsertChar(ch)
			return
		}
		ec.saveSnapshot()
		offsetBefore := ev.VisualCursor().Offset
		eb.doInsertChar(ch)
		offsetAfter := ev.VisualCursor().Offset
		ec.adjustAfterInsertion(offsetBefore, offsetAfter-offsetBefore)
	}

	eb.overrideSetText = func(text string) {
		if ec.destroyed {
			eb.doSetText(text)
			return
		}
		ec.Clear()
		eb.doSetText(text)
	}

	eb.overrideReplaceText = func(text string) {
		if ec.destroyed {
			eb.doReplaceText(text)
			return
		}
		ec.saveSnapshot()
		ec.Clear()
		eb.doReplaceText(text)
	}

	eb.overrideClear = func() {
		if ec.destroyed {
			eb.doClear()
			return
		}
		ec.saveSnapshot()
		ec.Clear()
		eb.doClear()
	}

	eb.overrideNewLine = func() {
		if ec.destroyed {
			eb.doNewLine()
			return
		}
		ec.saveSnapshot()
		offsetBefore := ev.VisualCursor().Offset
		eb.doNewLine()
		offsetAfter := ev.VisualCursor().Offset
		ec.adjustAfterInsertion(offsetBefore, offsetAfter-offsetBefore)
	}

	// --- Delete selected text ---

	ev.overrideDeleteSelectedText = func() {
		// Delegate to doDeleteSelectedText which calls the public DeleteRange,
		// so overrideDeleteRange handles snapshot + extmark adjustment.
		ev.doDeleteSelectedText()
	}

	// --- Undo / Redo ---

	eb.overrideUndo = func() bool {
		if ec.destroyed {
			return eb.doUndo()
		}
		if !ec.history.CanUndo() {
			return eb.doUndo()
		}

		// Save current extmark state to redo
		ec.history.PushRedo(ExtmarksSnapshot{
			extmarks: cloneExtmarks(ec.extmarks),
			nextID:   ec.nextID,
		})

		snap := ec.history.Undo()

		// Restore text FIRST so that updateHighlights (called by
		// restoreSnapshot) maps global extmark offsets to per-line
		// highlights against the correct line widths.
		result := eb.doUndo()
		ec.restoreSnapshot(snap)
		return result
	}

	eb.overrideRedo = func() bool {
		if ec.destroyed {
			return eb.doRedo()
		}
		if !ec.history.CanRedo() {
			return eb.doRedo()
		}

		// Save current extmark state to undo
		ec.history.PushUndo(ExtmarksSnapshot{
			extmarks: cloneExtmarks(ec.extmarks),
			nextID:   ec.nextID,
		})

		snap := ec.history.Redo()

		// Restore text FIRST (see undo comment above).
		result := eb.doRedo()
		ec.restoreSnapshot(snap)
		return result
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (ec *ExtmarksController) deleteByID(id int) {
	em, ok := ec.extmarks[id]
	if !ok {
		return
	}
	delete(ec.extmarks, id)
	if s, ok := ec.extmarksByType[em.TypeID]; ok {
		delete(s, id)
	}
	delete(ec.metadata, id)
}

// findVirtualContaining returns the first virtual extmark whose range
// [Start, End) contains offset, or nil.
func (ec *ExtmarksController) findVirtualContaining(offset int) *Extmark {
	for _, em := range ec.extmarks {
		if em.Virtual && offset >= em.Start && offset < em.End {
			return &em
		}
	}
	return nil
}

// adjustAfterInsertion shifts extmark offsets after an insertion at insertOffset
// of the given length.
func (ec *ExtmarksController) adjustAfterInsertion(insertOffset, length int) {
	for id, em := range ec.extmarks {
		if em.Start >= insertOffset {
			em.Start += length
			em.End += length
		} else if em.End > insertOffset {
			em.End += length
		}
		ec.extmarks[id] = em
	}
	ec.updateHighlights()
}

// AdjustAfterDeletion shifts/removes extmarks after a deletion at deleteOffset
// of the given length. Exported for advanced use cases.
func (ec *ExtmarksController) AdjustAfterDeletion(deleteOffset, length int) {
	ec.adjustAfterDeletion(deleteOffset, length)
}

func (ec *ExtmarksController) adjustAfterDeletion(deleteOffset, length int) {
	var toDelete []int

	for id, em := range ec.extmarks {
		if em.End <= deleteOffset {
			// Entirely before deletion — no change
			continue
		}

		if em.Start >= deleteOffset+length {
			// Entirely after deletion — shift back
			em.Start -= length
			em.End -= length
		} else if em.Start >= deleteOffset && em.End <= deleteOffset+length {
			// Entirely within deletion — remove
			toDelete = append(toDelete, id)
			continue
		} else if em.Start < deleteOffset && em.End > deleteOffset+length {
			// Deletion is inside the extmark — shrink
			em.End -= length
		} else if em.Start < deleteOffset && em.End > deleteOffset {
			// Overlaps start of deletion
			em.End -= min(em.End, deleteOffset+length) - deleteOffset
		} else if em.Start < deleteOffset+length && em.End > deleteOffset+length {
			// Overlaps end of deletion
			em.Start = deleteOffset
			em.End -= length
		}
		ec.extmarks[id] = em
	}

	for _, id := range toDelete {
		ec.deleteByID(id)
	}

	ec.updateHighlights()
}

// updateHighlights recomputes TextBuffer highlights from extmarks.
func (ec *ExtmarksController) updateHighlights() {
	tb := ec.editBuffer.TextBuffer()
	tb.ClearAllHighlights()

	for _, em := range ec.extmarks {
		if em.Fg == nil && em.Bg == nil && em.Attributes == 0 {
			continue
		}
		tb.AddHighlightByCharRange(Highlight{
			Start:      em.Start,
			End:        em.End,
			Fg:         em.Fg,
			Bg:         em.Bg,
			Attributes: em.Attributes,
			Ref:        em.ID,
		})
	}
}

func (ec *ExtmarksController) saveSnapshot() {
	ec.history.SaveSnapshot(ec.extmarks, ec.nextID)
}

func (ec *ExtmarksController) restoreSnapshot(snap *ExtmarksSnapshot) {
	if snap == nil {
		return
	}
	ec.extmarks = cloneExtmarks(snap.extmarks)
	ec.nextID = snap.nextID

	// Rebuild the secondary type index from the restored extmarks.
	// Without this, AllForTypeID returns stale results after undo/redo.
	ec.extmarksByType = make(map[int]map[int]struct{})
	for id, em := range ec.extmarks {
		if _, ok := ec.extmarksByType[em.TypeID]; !ok {
			ec.extmarksByType[em.TypeID] = make(map[int]struct{})
		}
		ec.extmarksByType[em.TypeID][id] = struct{}{}
	}

	ec.updateHighlights()
}

// DisplayWidthOfText returns the display width of text including newlines
// as +1 each (matching the offset coordinate space).
func DisplayWidthOfText(text string) int {
	parts := strings.Split(text, "\n")
	total := 0
	for i, part := range parts {
		total += StringWidth(part)
		if i < len(parts)-1 {
			total++ // newline separator
		}
	}
	return total
}
