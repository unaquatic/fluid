package textbuf

import (
	"strings"

	"github.com/unaquatic/fluid/style"
)

// Extmark is a tracked half-open range in global display-width coordinates.
// Newlines occupy one offset, matching EditBuffer position conversion.
//
// Cursor movement skips a Virtual extmark as one range.
type Extmark struct {
	// ID is the identifier the controller assigned, starting at 1.
	ID int
	// Start is the inclusive global display-width offset.
	Start int
	// End is the exclusive global display-width offset.
	End int
	// Virtual reports whether cursor movement skips the range.
	Virtual bool
	// Fg is the foreground color; nil leaves the text's foreground.
	Fg *style.Color
	// Bg is the background color; nil leaves the text's background.
	Bg *style.Color
	// Attributes holds decoration bits and an optional packed link ID.
	Attributes uint32
	// Priority is caller-defined; textbuf stores it but does not order by it.
	Priority int
	// Data is caller-defined data.
	Data any
	// TypeID is the category identifier; zero is the default category.
	TypeID int
}

// ExtmarkOptions configures a new Extmark. Its fields mean what the Extmark
// fields of the same names do.
type ExtmarkOptions struct {
	// Start is the inclusive global display-width offset.
	Start int
	// End is the exclusive global display-width offset.
	End int
	// Virtual reports whether cursor movement skips the range.
	Virtual bool
	// Fg is the foreground color; nil leaves the text's foreground.
	Fg *style.Color
	// Bg is the background color; nil leaves the text's background.
	Bg *style.Color
	// Attributes holds decoration bits and an optional packed link ID.
	Attributes uint32
	// Priority is caller-defined; textbuf stores it but does not order by it.
	Priority int
	// Data is caller-defined data.
	Data any
	// TypeID is the category identifier; zero is the default category.
	TypeID int
	// Metadata is data stored separately and returned by Metadata.
	Metadata any
}

// ExtmarksController tracks extmark positions through EditBuffer mutations and
// coordinates their highlights with an EditSurface.
//
// It intercepts cursor movement, edits, and history operations. Only one active
// controller can safely own a given EditBuffer and EditSurface.
type ExtmarksController struct {
	editBuffer  *EditBuffer
	editSurface *EditSurface

	extmarks       map[int]Extmark
	extmarksByType map[int]map[int]struct{} // Type ID to extmark ID set.
	metadata       map[int]any
	nextID         int
	destroyed      bool

	history ExtmarksHistory

	typeNameToID map[string]int
	typeIDToName map[int]string
	nextTypeID   int
}

// NewExtmarksController returns a controller and installs its edit and movement
// interceptors on eb and ev.
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

// Create adds an extmark and returns its positive ID, or -1 after destruction.
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

// Delete removes id and reports whether it existed in an active controller.
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

// Get returns a copy of the extmark identified by id, or nil when absent or
// destroyed.
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

// All returns all extmarks in unspecified order, or nil after destruction.
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

// Virtual returns virtual extmarks in unspecified order, or nil after
// destruction.
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

// AtOffset returns extmarks whose half-open range contains offset.
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

// AllForTypeID returns extmarks in typeID in unspecified order.
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

// Clear removes all extmarks and metadata from an active controller.
func (ec *ExtmarksController) Clear() {
	if ec.destroyed {
		return
	}
	ec.extmarks = make(map[int]Extmark)
	ec.extmarksByType = make(map[int]map[int]struct{})
	ec.metadata = make(map[int]any)
	ec.updateHighlights()
}

// RegisterType returns a stable positive ID for name, or -1 after destruction.
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

// TypeID returns the ID registered for name, or -1 when absent or destroyed.
func (ec *ExtmarksController) TypeID(name string) int {
	if ec.destroyed {
		return -1
	}
	if id, ok := ec.typeNameToID[name]; ok {
		return id
	}
	return -1
}

// TypeName returns the name registered for typeID, or an empty string.
func (ec *ExtmarksController) TypeName(typeID int) string {
	if ec.destroyed {
		return ""
	}
	return ec.typeIDToName[typeID]
}

// Metadata returns separately stored metadata for id, or nil.
func (ec *ExtmarksController) Metadata(id int) any {
	if ec.destroyed {
		return nil
	}
	return ec.metadata[id]
}

// Destroy removes installed interceptors and makes the controller inert.
func (ec *ExtmarksController) Destroy() {
	if ec.destroyed {
		return
	}

	eb := ec.editBuffer
	ev := ec.editSurface

	// Restore default edit and movement behavior.
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

func (ec *ExtmarksController) installOverrides() {
	eb := ec.editBuffer
	ev := ec.editSurface

	// Cursor movement skips virtual ranges.

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
			// Preserve the vertical-navigation column across the corrective move.
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
			// Preserve the vertical-navigation column across the corrective move.
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

	// Deletion records extmark history and adjusts tracked ranges.

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

		// Save only when text deletion will also create an undo snapshot.
		ec.saveSnapshot()

		targetOffset := offsetBefore - 1
		if vem := ec.findVirtualContaining(targetOffset); vem != nil && offsetBefore == vem.End {
			// Backspace at a virtual range boundary removes the entire range.
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

		// Save only when text deletion will also create an undo snapshot.
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

		// Forward delete keeps the cursor fixed, so compare total widths.
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

		// Derive the deleted global range from logical line widths.
		var deleteOffset, deleteLen int
		if lineCount <= 1 {
			// Clearing the sole line removes no newline.
			deleteOffset = 0
			deleteLen = lineW
		} else if row < lineCount-1 {
			// A nonfinal line owns its trailing newline.
			deleteOffset = eb.positionToOffset(row, 0)
			deleteLen = lineW + 1
		} else {
			// The final line is removed with its preceding newline.
			deleteOffset = eb.positionToOffset(row, 0) - 1
			deleteLen = lineW + 1
		}

		eb.doDeleteLine()
		if deleteLen > 0 {
			ec.adjustAfterDeletion(deleteOffset, deleteLen)
		}
	}

	// Insertion shifts ranges at or after the insertion point.

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

	ev.overrideDeleteSelectedText = func() {
		// The public DeleteRange interceptor handles history and range adjustment.
		ev.doDeleteSelectedText()
	}

	// Undo and redo restore text before rebuilding extmark highlights.

	eb.overrideUndo = func() bool {
		if ec.destroyed {
			return eb.doUndo()
		}
		if !ec.history.CanUndo() {
			return eb.doUndo()
		}

		// Preserve current extmarks for redo.
		ec.history.PushRedo(ExtmarksSnapshot{
			extmarks: cloneExtmarks(ec.extmarks),
			nextID:   ec.nextID,
		})

		snap := ec.history.Undo()

		// Restore text first so highlights use the restored line widths.
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

		// Preserve current extmarks for undo.
		ec.history.PushUndo(ExtmarksSnapshot{
			extmarks: cloneExtmarks(ec.extmarks),
			nextID:   ec.nextID,
		})

		snap := ec.history.Redo()

		// Restore text first so highlights use the restored line widths.
		result := eb.doRedo()
		ec.restoreSnapshot(snap)
		return result
	}
}

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

// findVirtualContaining returns the first virtual extmark containing offset, or
// nil. Map iteration makes the selected match unspecified when ranges overlap.
func (ec *ExtmarksController) findVirtualContaining(offset int) *Extmark {
	for _, em := range ec.extmarks {
		if em.Virtual && offset >= em.Start && offset < em.End {
			return &em
		}
	}
	return nil
}

// adjustAfterInsertion expands or shifts extmarks for a display-width insertion.
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

// AdjustAfterDeletion adjusts extmarks for a display-width deletion beginning at
// deleteOffset.
func (ec *ExtmarksController) AdjustAfterDeletion(deleteOffset, length int) {
	ec.adjustAfterDeletion(deleteOffset, length)
}

func (ec *ExtmarksController) adjustAfterDeletion(deleteOffset, length int) {
	var toDelete []int

	for id, em := range ec.extmarks {
		if em.End <= deleteOffset {
			// Ranges before the deletion remain unchanged.
			continue
		}

		if em.Start >= deleteOffset+length {
			// Ranges after the deletion shift backward.
			em.Start -= length
			em.End -= length
		} else if em.Start >= deleteOffset && em.End <= deleteOffset+length {
			// Ranges fully deleted are removed.
			toDelete = append(toDelete, id)
			continue
		} else if em.Start < deleteOffset && em.End > deleteOffset+length {
			// A deletion contained by the extmark shortens it.
			em.End -= length
		} else if em.Start < deleteOffset && em.End > deleteOffset {
			// Preserve the prefix before an overlapping deletion.
			em.End -= min(em.End, deleteOffset+length) - deleteOffset
		} else if em.Start < deleteOffset+length && em.End > deleteOffset+length {
			// Move a surviving suffix to the deletion start.
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

// updateHighlights replaces TextBuffer highlights with styled extmark ranges.
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

	// Rebuild the type index from restored primary records.
	ec.extmarksByType = make(map[int]map[int]struct{})
	for id, em := range ec.extmarks {
		if _, ok := ec.extmarksByType[em.TypeID]; !ok {
			ec.extmarksByType[em.TypeID] = make(map[int]struct{})
		}
		ec.extmarksByType[em.TypeID][id] = struct{}{}
	}

	ec.updateHighlights()
}

// DisplayWidthOfText returns text's display width with each newline counted as
// one offset.
func DisplayWidthOfText(text string) int {
	parts := strings.Split(text, "\n")
	total := 0
	for i, part := range parts {
		total += StringWidth(part)
		if i < len(parts)-1 {
			total++ // Count the logical-line separator.
		}
	}
	return total
}
