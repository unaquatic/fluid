package textbuf

import (
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

// CursorPosition identifies one location in logical and global display-width
// coordinates.
type CursorPosition struct {
	// Row is the logical line index.
	Row int
	// Col is the display-column offset within the line.
	Col int
	// Offset is the global display-width offset from the buffer start.
	Offset int
}

// editSnapshot captures text and cursor state for undo or redo.
type editSnapshot struct {
	lines     []Line
	cursorRow int
	cursorCol int
}

// editBufferIDCounter assigns EditBuffer IDs.
var editBufferIDCounter atomic.Uint64

// EditBuffer adds cursor movement, text editing, and history to a TextBuffer.
type EditBuffer struct {
	id        uint64
	tb        *TextBuffer
	cursorRow int
	cursorCol int

	undoStack []editSnapshot
	redoStack []editSnapshot
	maxUndo   int

	// OnCursorChange runs after a cursor movement notification.
	OnCursorChange func(row, col int)

	// ExtmarksController installs these interceptors to track edits.
	overrideMoveCursorLeft     func()
	overrideMoveCursorRight    func()
	overrideSetCursorByOffset  func(offset int)
	overrideInsertChar         func(ch rune)
	overrideInsertText         func(text string)
	overrideDeleteChar         func()
	overrideDeleteCharBackward func()
	overrideDeleteRange        func(startRow, startCol, endRow, endCol int)
	overrideDeleteLine         func()
	overrideNewLine            func()
	overrideSetText            func(text string)
	overrideReplaceText        func(text string)
	overrideClear              func()
	overrideUndo               func() bool
	overrideRedo               func() bool
}

// NewEditBuffer returns an edit buffer backed by tb with a 100-entry undo limit.
func NewEditBuffer(tb *TextBuffer) *EditBuffer {
	return &EditBuffer{
		id:      editBufferIDCounter.Add(1),
		tb:      tb,
		maxUndo: 100,
	}
}

// ID returns the process-unique edit-buffer identifier.
func (eb *EditBuffer) ID() uint64 { return eb.id }

// TextBuffer returns the underlying TextBuffer.
func (eb *EditBuffer) TextBuffer() *TextBuffer { return eb.tb }

// CursorRow returns the current logical row.
func (eb *EditBuffer) CursorRow() int { return eb.cursorRow }

// CursorCol returns the current display column.
func (eb *EditBuffer) CursorCol() int { return eb.cursorCol }

// CursorPosition returns the cursor in logical and global coordinates.
func (eb *EditBuffer) CursorPosition() CursorPosition {
	return CursorPosition{
		Row:    eb.cursorRow,
		Col:    eb.cursorCol,
		Offset: eb.positionToOffset(eb.cursorRow, eb.cursorCol),
	}
}

// SetCursor moves to a logical row and display column, clamped to the buffer.
func (eb *EditBuffer) SetCursor(row, col int) {
	row = clampInt(row, 0, eb.tb.LineCount()-1)
	lineW := eb.tb.LineWidthAt(row)
	col = clampInt(col, 0, lineW)
	if row != eb.cursorRow || col != eb.cursorCol {
		eb.cursorRow = row
		eb.cursorCol = col
		eb.notifyCursor()
	}
}

// SetCursorByOffset moves to a global display-width offset.
func (eb *EditBuffer) SetCursorByOffset(offset int) {
	if eb.overrideSetCursorByOffset != nil {
		eb.overrideSetCursorByOffset(offset)
		return
	}
	eb.doSetCursorByOffset(offset)
}

func (eb *EditBuffer) doSetCursorByOffset(offset int) {
	row, col := eb.offsetToPosition(offset)
	eb.SetCursor(row, col)
}

// MoveCursorLeft moves left by one grapheme cluster or across the preceding
// newline.
func (eb *EditBuffer) MoveCursorLeft() {
	if eb.overrideMoveCursorLeft != nil {
		eb.overrideMoveCursorLeft()
		return
	}
	eb.doMoveCursorLeft()
}

func (eb *EditBuffer) doMoveCursorLeft() {
	if eb.cursorCol > 0 {
		eb.cursorCol = eb.prevCol(eb.cursorRow, eb.cursorCol)
	} else if eb.cursorRow > 0 {
		eb.cursorRow--
		eb.cursorCol = eb.tb.LineWidthAt(eb.cursorRow)
	}
	eb.notifyCursor()
}

// MoveCursorRight moves right by one grapheme cluster or across the following
// newline.
func (eb *EditBuffer) MoveCursorRight() {
	if eb.overrideMoveCursorRight != nil {
		eb.overrideMoveCursorRight()
		return
	}
	eb.doMoveCursorRight()
}

func (eb *EditBuffer) doMoveCursorRight() {
	lineW := eb.tb.LineWidthAt(eb.cursorRow)
	if eb.cursorCol < lineW {
		eb.cursorCol = eb.nextCol(eb.cursorRow, eb.cursorCol)
	} else if eb.cursorRow < eb.tb.LineCount()-1 {
		eb.cursorRow++
		eb.cursorCol = 0
	}
	eb.notifyCursor()
}

// MoveCursorUp moves up one logical line and clamps the display column.
func (eb *EditBuffer) MoveCursorUp() {
	if eb.cursorRow > 0 {
		eb.cursorRow--
		lineW := eb.tb.LineWidthAt(eb.cursorRow)
		if eb.cursorCol > lineW {
			eb.cursorCol = lineW
		}
	}
	eb.notifyCursor()
}

// MoveCursorDown moves down one logical line and clamps the display column.
func (eb *EditBuffer) MoveCursorDown() {
	if eb.cursorRow < eb.tb.LineCount()-1 {
		eb.cursorRow++
		lineW := eb.tb.LineWidthAt(eb.cursorRow)
		if eb.cursorCol > lineW {
			eb.cursorCol = lineW
		}
	}
	eb.notifyCursor()
}

// GotoLine moves to the start of line, clamped to the buffer.
func (eb *EditBuffer) GotoLine(line int) {
	eb.SetCursor(line, 0)
}

// SetText replaces all text, moves the cursor to the start, and clears history.
func (eb *EditBuffer) SetText(text string) {
	if eb.overrideSetText != nil {
		eb.overrideSetText(text)
		return
	}
	eb.doSetText(text)
}

func (eb *EditBuffer) doSetText(text string) {
	eb.tb.SetText(text)
	eb.cursorRow = 0
	eb.cursorCol = 0
	eb.undoStack = nil
	eb.redoStack = nil
	eb.notifyCursor()
}

// ReplaceText replaces all text as an undoable edit and moves the cursor to the
// start.
func (eb *EditBuffer) ReplaceText(text string) {
	if eb.overrideReplaceText != nil {
		eb.overrideReplaceText(text)
		return
	}
	eb.doReplaceText(text)
}

func (eb *EditBuffer) doReplaceText(text string) {
	eb.saveUndo()
	eb.tb.SetText(text)
	eb.cursorRow = 0
	eb.cursorCol = 0
	eb.redoStack = nil
	eb.notifyCursor()
}

// Text returns the complete unstyled content.
func (eb *EditBuffer) Text() string {
	return eb.tb.PlainText()
}

// InsertChar inserts ch at the cursor.
func (eb *EditBuffer) InsertChar(ch rune) {
	if eb.overrideInsertChar != nil {
		eb.overrideInsertChar(ch)
		return
	}
	eb.doInsertChar(ch)
}

func (eb *EditBuffer) doInsertChar(ch rune) {
	eb.saveUndo()
	eb.insertAtCursor(string(ch))
	eb.redoStack = nil
}

// InsertText inserts text at the cursor.
func (eb *EditBuffer) InsertText(text string) {
	if eb.overrideInsertText != nil {
		eb.overrideInsertText(text)
		return
	}
	eb.doInsertText(text)
}

func (eb *EditBuffer) doInsertText(text string) {
	eb.saveUndo()
	eb.insertAtCursor(text)
	eb.redoStack = nil
}

// DeleteChar removes the grapheme cluster at the cursor. At end of line, it
// joins the following line.
func (eb *EditBuffer) DeleteChar() {
	if eb.overrideDeleteChar != nil {
		eb.overrideDeleteChar()
		return
	}
	eb.doDeleteChar()
}

func (eb *EditBuffer) doDeleteChar() {
	lineW := eb.tb.LineWidthAt(eb.cursorRow)
	if eb.cursorCol >= lineW {
		// Forward delete at end of line removes the following newline.
		if eb.cursorRow < eb.tb.LineCount()-1 {
			eb.saveUndo()
			eb.mergeLineForward()
			eb.redoStack = nil
		}
		return
	}
	eb.saveUndo()
	eb.deleteAtCursor(1)
	eb.redoStack = nil
}

// DeleteCharBackward removes the preceding grapheme cluster. At start of line,
// it joins the line to its predecessor.
func (eb *EditBuffer) DeleteCharBackward() {
	if eb.overrideDeleteCharBackward != nil {
		eb.overrideDeleteCharBackward()
		return
	}
	eb.doDeleteCharBackward()
}

func (eb *EditBuffer) doDeleteCharBackward() {
	if eb.cursorCol > 0 {
		eb.saveUndo()
		prevCol := eb.prevCol(eb.cursorRow, eb.cursorCol)
		count := eb.cursorCol - prevCol
		eb.cursorCol = prevCol
		eb.deleteAtCursor(count)
		eb.redoStack = nil
	} else if eb.cursorRow > 0 {
		// Backspace at start of line removes the preceding newline.
		eb.saveUndo()
		prevRow := eb.cursorRow - 1
		prevLineW := eb.tb.LineWidthAt(prevRow)
		eb.mergeLineBackward()
		eb.cursorRow = prevRow
		eb.cursorCol = prevLineW
		eb.redoStack = nil
	}
}

// NewLine splits the current logical line at the cursor.
func (eb *EditBuffer) NewLine() {
	if eb.overrideNewLine != nil {
		eb.overrideNewLine()
		return
	}
	eb.doNewLine()
}

func (eb *EditBuffer) doNewLine() {
	eb.saveUndo()
	eb.splitLine()
	eb.redoStack = nil
}

// DeleteLine removes the current logical line or clears the sole remaining line.
func (eb *EditBuffer) DeleteLine() {
	if eb.overrideDeleteLine != nil {
		eb.overrideDeleteLine()
		return
	}
	eb.doDeleteLine()
}

func (eb *EditBuffer) doDeleteLine() {
	if eb.tb.LineCount() <= 1 {
		eb.saveUndo()
		eb.tb.SetLines([]Line{{Spans: []Span{{
			Text: "",
			Fg:   eb.tb.defaultFg,
			Bg:   eb.tb.defaultBg,
		}}}})
		eb.cursorCol = 0
		eb.redoStack = nil
		return
	}
	eb.saveUndo()
	lines := eb.tb.Lines()
	newLines := make([]Line, 0, len(lines)-1)
	newLines = append(newLines, lines[:eb.cursorRow]...)
	newLines = append(newLines, lines[eb.cursorRow+1:]...)
	eb.tb.SetLines(newLines)
	if eb.cursorRow >= len(newLines) {
		eb.cursorRow = len(newLines) - 1
	}
	lineW := eb.tb.LineWidthAt(eb.cursorRow)
	if eb.cursorCol > lineW {
		eb.cursorCol = lineW
	}
	eb.redoStack = nil
}

// Clear removes all content as an undoable edit.
func (eb *EditBuffer) Clear() {
	if eb.overrideClear != nil {
		eb.overrideClear()
		return
	}
	eb.doClear()
}

func (eb *EditBuffer) doClear() {
	eb.saveUndo()
	eb.tb.Clear()
	eb.cursorRow = 0
	eb.cursorCol = 0
	eb.redoStack = nil
}

// DeleteRange removes the half-open range between two logical row and
// display-column positions.
func (eb *EditBuffer) DeleteRange(startRow, startCol, endRow, endCol int) {
	if eb.overrideDeleteRange != nil {
		eb.overrideDeleteRange(startRow, startCol, endRow, endCol)
		return
	}
	eb.doDeleteRange(startRow, startCol, endRow, endCol)
}

func (eb *EditBuffer) doDeleteRange(startRow, startCol, endRow, endCol int) {
	eb.saveUndo()
	eb.deleteRange(startRow, startCol, endRow, endCol)
	eb.cursorRow = startRow
	eb.cursorCol = startCol
	eb.redoStack = nil
}

// Undo restores the previous text and cursor snapshot and reports whether one
// existed.
func (eb *EditBuffer) Undo() bool {
	if eb.overrideUndo != nil {
		return eb.overrideUndo()
	}
	return eb.doUndo()
}

func (eb *EditBuffer) doUndo() bool {
	if len(eb.undoStack) == 0 {
		return false
	}
	// Preserve the current state for Redo.
	eb.redoStack = append(eb.redoStack, eb.snapshot())

	last := eb.undoStack[len(eb.undoStack)-1]
	eb.undoStack = eb.undoStack[:len(eb.undoStack)-1]
	eb.restore(last)
	return true
}

// Redo restores the newest redo snapshot and reports whether one existed.
func (eb *EditBuffer) Redo() bool {
	if eb.overrideRedo != nil {
		return eb.overrideRedo()
	}
	return eb.doRedo()
}

func (eb *EditBuffer) doRedo() bool {
	if len(eb.redoStack) == 0 {
		return false
	}
	eb.undoStack = append(eb.undoStack, eb.snapshot())

	last := eb.redoStack[len(eb.redoStack)-1]
	eb.redoStack = eb.redoStack[:len(eb.redoStack)-1]
	eb.restore(last)
	return true
}

// CanUndo reports whether an undo snapshot is available.
func (eb *EditBuffer) CanUndo() bool { return len(eb.undoStack) > 0 }

// CanRedo reports whether a redo snapshot is available.
func (eb *EditBuffer) CanRedo() bool { return len(eb.redoStack) > 0 }

// ClearHistory removes all undo and redo snapshots.
func (eb *EditBuffer) ClearHistory() {
	eb.undoStack = nil
	eb.redoStack = nil
}

// SetMaxUndo sets the undo capacity to at least one and immediately discards
// excess oldest entries.
func (eb *EditBuffer) SetMaxUndo(n int) {
	if n < 1 {
		n = 1
	}
	eb.maxUndo = n
	if len(eb.undoStack) > eb.maxUndo {
		eb.undoStack = eb.undoStack[len(eb.undoStack)-eb.maxUndo:]
	}
}

// LineCount returns the number of logical lines.
func (eb *EditBuffer) LineCount() int { return eb.tb.LineCount() }

// NextWordBoundary skips the current non-space run and following whitespace.
func (eb *EditBuffer) NextWordBoundary() CursorPosition {
	row, col := eb.cursorRow, eb.cursorCol
	lineText := eb.tb.LineTextAt(row)
	runes := []rune(lineText)
	runeIdx := colToRuneIndex(runes, col)

	// Advance through the current non-space run.
	for runeIdx < len(runes) && !unicode.IsSpace(runes[runeIdx]) {
		runeIdx++
	}
	// Advance through whitespace to the next run.
	for runeIdx < len(runes) && unicode.IsSpace(runes[runeIdx]) {
		runeIdx++
	}

	if runeIdx >= len(runes) && row < eb.tb.LineCount()-1 {
		return CursorPosition{Row: row + 1, Col: 0, Offset: eb.positionToOffset(row+1, 0)}
	}

	newCol := runeIndexToCol(runes, runeIdx)
	return CursorPosition{Row: row, Col: newCol, Offset: eb.positionToOffset(row, newCol)}
}

// PrevWordBoundary returns the start of the preceding non-space run.
func (eb *EditBuffer) PrevWordBoundary() CursorPosition {
	row, col := eb.cursorRow, eb.cursorCol
	lineText := eb.tb.LineTextAt(row)
	runes := []rune(lineText)
	runeIdx := colToRuneIndex(runes, col)

	if runeIdx <= 0 {
		if row > 0 {
			prevLineW := eb.tb.LineWidthAt(row - 1)
			return CursorPosition{Row: row - 1, Col: prevLineW, Offset: eb.positionToOffset(row-1, prevLineW)}
		}
		return CursorPosition{Row: 0, Col: 0, Offset: 0}
	}

	runeIdx--
	// Move left through preceding whitespace.
	for runeIdx > 0 && unicode.IsSpace(runes[runeIdx]) {
		runeIdx--
	}
	// Continue to the start of the preceding non-space run.
	for runeIdx > 0 && !unicode.IsSpace(runes[runeIdx-1]) {
		runeIdx--
	}

	newCol := runeIndexToCol(runes, runeIdx)
	return CursorPosition{Row: row, Col: newCol, Offset: eb.positionToOffset(row, newCol)}
}

// EOL returns the end of the current logical line.
func (eb *EditBuffer) EOL() CursorPosition {
	lineW := eb.tb.LineWidthAt(eb.cursorRow)
	return CursorPosition{Row: eb.cursorRow, Col: lineW, Offset: eb.positionToOffset(eb.cursorRow, lineW)}
}

func (eb *EditBuffer) positionToOffset(row, col int) int {
	offset := 0
	for i := 0; i < row && i < eb.tb.LineCount(); i++ {
		offset += eb.tb.LineWidthAt(i) + 1 // Each newline occupies one offset.
	}
	offset += col
	return offset
}

func (eb *EditBuffer) offsetToPosition(offset int) (int, int) {
	pos := 0
	for i := 0; i < eb.tb.LineCount(); i++ {
		lineW := eb.tb.LineWidthAt(i)
		if pos+lineW >= offset {
			return i, offset - pos
		}
		pos += lineW + 1 // Each newline occupies one offset.
	}
	// Clamp offsets beyond the content to the final line end.
	lastRow := max(eb.tb.LineCount()-1, 0)
	return lastRow, eb.tb.LineWidthAt(lastRow)
}

// TextRange returns the half-open global display-width range [startOffset,
// endOffset).
func (eb *EditBuffer) TextRange(startOffset, endOffset int) string {
	return eb.tb.TextRange(startOffset, endOffset)
}

// LineStartOffset returns the global display-width offset at the start of row.
func (eb *EditBuffer) LineStartOffset(row int) int {
	return eb.positionToOffset(row, 0)
}

// OffsetToPosition converts a global display-width offset to a logical position.
func (eb *EditBuffer) OffsetToPosition(offset int) CursorPosition {
	row, col := eb.offsetToPosition(offset)
	return CursorPosition{Row: row, Col: col, Offset: offset}
}

// PositionToOffset converts a logical position to a global display-width offset.
func (eb *EditBuffer) PositionToOffset(row, col int) int {
	return eb.positionToOffset(row, col)
}

// TextRangeByCoords returns a half-open logical coordinate range.
func (eb *EditBuffer) TextRangeByCoords(startRow, startCol, endRow, endCol int) string {
	return eb.tb.TextRangeByCoords(startRow, startCol, endRow, endCol)
}

// SetCursorByByteOffset converts byteOffset within line to a display column and
// moves the cursor there.
func (eb *EditBuffer) SetCursorByByteOffset(line, byteOffset int) {
	line = clampInt(line, 0, eb.tb.LineCount()-1)
	lineText := eb.tb.LineTextAt(line)

	end := min(byteOffset, len(lineText))
	col := StringWidth(lineText[:end])
	eb.SetCursor(line, col)
}

func (eb *EditBuffer) insertAtCursor(text string) {
	lines := eb.tb.Lines()
	if eb.cursorRow >= len(lines) {
		eb.cursorRow = len(lines) - 1
	}

	lineText := lines[eb.cursorRow].Text()
	byteOff := colToByteOffset(lineText, eb.cursorCol)

	before := lineText[:byteOff]
	after := lineText[byteOff:]
	combined := before + text + after

	// New content inherits the first span's style, or buffer defaults.
	fg := eb.tb.defaultFg
	bg := eb.tb.defaultBg
	var attrs uint32
	if len(lines[eb.cursorRow].Spans) > 0 {
		fg = lines[eb.cursorRow].Spans[0].Fg
		bg = lines[eb.cursorRow].Spans[0].Bg
		attrs = lines[eb.cursorRow].Spans[0].Attributes
	}

	newParts := strings.Split(combined, "\n")
	newLines := make([]Line, 0, len(lines)-1+len(newParts))
	newLines = append(newLines, lines[:eb.cursorRow]...)
	for _, part := range newParts {
		newLines = append(newLines, Line{Spans: []Span{{
			Text: part, Fg: fg, Bg: bg, Attributes: attrs,
		}}})
	}
	newLines = append(newLines, lines[eb.cursorRow+1:]...)
	eb.tb.SetLines(newLines)

	// Move the cursor after the inserted text.
	insertedLines := strings.Split(text, "\n")
	if len(insertedLines) == 1 {
		eb.cursorCol += StringWidth(text)
	} else {
		eb.cursorRow += len(insertedLines) - 1
		eb.cursorCol = StringWidth(insertedLines[len(insertedLines)-1])
	}
	eb.notifyCursor()
}

func (eb *EditBuffer) deleteAtCursor(count int) {
	lines := eb.tb.Lines()
	lineText := lines[eb.cursorRow].Text()

	// Convert the cursor column to the start byte of its cluster.
	startByte := colToByteOffset(lineText, eb.cursorCol)

	// Extend the deletion through at least count display columns.
	deleted := 0
	endByte := startByte
	remaining := lineText[startByte:]
	for deleted < count && len(remaining) > 0 {
		cluster, rest, cw := NextGraphemeCluster(remaining)
		deleted += ClusterWidth(cw)
		endByte += len(cluster)
		remaining = rest
	}

	newText := lineText[:startByte] + lineText[endByte:]

	fg := eb.tb.defaultFg
	bg := eb.tb.defaultBg
	if len(lines[eb.cursorRow].Spans) > 0 {
		fg = lines[eb.cursorRow].Spans[0].Fg
		bg = lines[eb.cursorRow].Spans[0].Bg
	}
	lines[eb.cursorRow] = Line{Spans: []Span{{Text: newText, Fg: fg, Bg: bg}}}
	eb.tb.SetLines(lines)
}

func (eb *EditBuffer) mergeLineForward() {
	lines := eb.tb.Lines()
	if eb.cursorRow >= len(lines)-1 {
		return
	}
	merged := lines[eb.cursorRow].Text() + lines[eb.cursorRow+1].Text()
	fg := eb.tb.defaultFg
	if len(lines[eb.cursorRow].Spans) > 0 {
		fg = lines[eb.cursorRow].Spans[0].Fg
	}
	newLines := make([]Line, 0, len(lines)-1)
	newLines = append(newLines, lines[:eb.cursorRow]...)
	newLines = append(newLines, Line{Spans: []Span{{Text: merged, Fg: fg, Bg: eb.tb.defaultBg}}})
	newLines = append(newLines, lines[eb.cursorRow+2:]...)
	eb.tb.SetLines(newLines)
}

func (eb *EditBuffer) mergeLineBackward() {
	lines := eb.tb.Lines()
	if eb.cursorRow <= 0 {
		return
	}
	prevRow := eb.cursorRow - 1
	merged := lines[prevRow].Text() + lines[eb.cursorRow].Text()
	fg := eb.tb.defaultFg
	if len(lines[prevRow].Spans) > 0 {
		fg = lines[prevRow].Spans[0].Fg
	}
	newLines := make([]Line, 0, len(lines)-1)
	newLines = append(newLines, lines[:prevRow]...)
	newLines = append(newLines, Line{Spans: []Span{{Text: merged, Fg: fg, Bg: eb.tb.defaultBg}}})
	newLines = append(newLines, lines[eb.cursorRow+1:]...)
	eb.tb.SetLines(newLines)
}

func (eb *EditBuffer) splitLine() {
	lines := eb.tb.Lines()
	lineText := lines[eb.cursorRow].Text()
	byteOff := colToByteOffset(lineText, eb.cursorCol)

	before := lineText[:byteOff]
	after := lineText[byteOff:]

	fg := eb.tb.defaultFg
	bg := eb.tb.defaultBg
	if len(lines[eb.cursorRow].Spans) > 0 {
		fg = lines[eb.cursorRow].Spans[0].Fg
		bg = lines[eb.cursorRow].Spans[0].Bg
	}

	newLines := make([]Line, 0, len(lines)+1)
	newLines = append(newLines, lines[:eb.cursorRow]...)
	newLines = append(newLines, Line{Spans: []Span{{Text: before, Fg: fg, Bg: bg}}})
	newLines = append(newLines, Line{Spans: []Span{{Text: after, Fg: fg, Bg: bg}}})
	newLines = append(newLines, lines[eb.cursorRow+1:]...)
	eb.tb.SetLines(newLines)

	eb.cursorRow++
	eb.cursorCol = 0
	eb.notifyCursor()
}

func (eb *EditBuffer) deleteRange(startRow, startCol, endRow, endCol int) {
	lines := eb.tb.Lines()
	if startRow >= len(lines) {
		return
	}
	if endRow >= len(lines) {
		endRow = len(lines) - 1
		endCol = eb.tb.LineWidthAt(endRow)
	}

	startText := lines[startRow].Text()
	endText := lines[endRow].Text()
	startByteOff := colToByteOffset(startText, startCol)
	endByteOff := colToByteOffset(endText, endCol)

	merged := startText[:startByteOff] + endText[endByteOff:]

	fg := eb.tb.defaultFg
	bg := eb.tb.defaultBg
	if len(lines[startRow].Spans) > 0 {
		fg = lines[startRow].Spans[0].Fg
		bg = lines[startRow].Spans[0].Bg
	}

	newLines := make([]Line, 0, len(lines)-(endRow-startRow))
	newLines = append(newLines, lines[:startRow]...)
	newLines = append(newLines, Line{Spans: []Span{{Text: merged, Fg: fg, Bg: bg}}})
	newLines = append(newLines, lines[endRow+1:]...)
	eb.tb.SetLines(newLines)
}

func (eb *EditBuffer) saveUndo() {
	snap := eb.snapshot()
	eb.undoStack = append(eb.undoStack, snap)
	if len(eb.undoStack) > eb.maxUndo {
		eb.undoStack = eb.undoStack[1:]
	}
}

func (eb *EditBuffer) snapshot() editSnapshot {
	lines := eb.tb.Lines()
	copyLines := make([]Line, len(lines))
	for i, l := range lines {
		spans := make([]Span, len(l.Spans))
		copy(spans, l.Spans)
		copyLines[i] = Line{Spans: spans}
	}
	return editSnapshot{
		lines:     copyLines,
		cursorRow: eb.cursorRow,
		cursorCol: eb.cursorCol,
	}
}

func (eb *EditBuffer) restore(snap editSnapshot) {
	eb.tb.SetLines(snap.lines)
	eb.cursorRow = snap.cursorRow
	eb.cursorCol = snap.cursorCol
	eb.notifyCursor()
}

func (eb *EditBuffer) notifyCursor() {
	if eb.OnCursorChange != nil {
		eb.OnCursorChange(eb.cursorRow, eb.cursorCol)
	}
}

// prevCol returns the start column of the cluster preceding col on row.
func (eb *EditBuffer) prevCol(row, col int) int {
	lineText := eb.tb.LineTextAt(row)
	w := 0
	remaining := lineText
	for len(remaining) > 0 {
		cluster, rest, cw := NextGraphemeCluster(remaining)
		_ = cluster
		cw = ClusterWidth(cw)
		if w+cw >= col {
			return w
		}
		w += cw
		remaining = rest
	}
	return w
}

// nextCol returns the end column of the cluster beginning at col.
func (eb *EditBuffer) nextCol(row, col int) int {
	lineText := eb.tb.LineTextAt(row)
	w := 0
	remaining := lineText
	for len(remaining) > 0 {
		_, rest, cw := NextGraphemeCluster(remaining)
		cw = ClusterWidth(cw)
		if w >= col {
			return w + cw
		}
		w += cw
		remaining = rest
	}
	return w
}

// colToByteOffset returns the byte offset at or following display column col.
func colToByteOffset(s string, col int) int {
	w := 0
	pos := 0
	for pos < len(s) {
		if w >= col {
			return pos
		}
		cluster, _, cw := NextGraphemeCluster(s[pos:])
		w += ClusterWidth(cw)
		pos += len(cluster)
	}
	return len(s)
}

// colToRuneIndex maps a display column to an index in runes using grapheme
// boundaries.
func colToRuneIndex(runes []rune, col int) int {
	s := string(runes)
	byteOff := colToByteOffset(s, col)
	// Count rune starts preceding byteOff.
	n := 0
	for i := range s {
		if i >= byteOff {
			break
		}
		n++
	}
	return n
}

// runeIndexToCol maps an index in runes to a display column.
func runeIndexToCol(runes []rune, idx int) int {
	s := string(runes)
	// Convert the rune index back to a UTF-8 byte offset.
	byteOff := 0
	for i := 0; i < idx && byteOff < len(s); {
		_, size := utf8.DecodeRuneInString(s[byteOff:])
		byteOff += size
		i++
	}
	return StringWidth(s[:byteOff])
}

// clampInt restricts v to the inclusive range [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
