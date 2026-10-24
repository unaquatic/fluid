package textbuf

import (
	"github.com/unaquatic/fluid/style"
)

// VisualCursor contains one cursor position in viewport-relative, logical, and
// global display-width coordinates.
type VisualCursor struct {
	// VisualRow is the viewport-relative row.
	VisualRow int
	// VisualCol is the viewport-relative display column.
	VisualCol int
	// LogicalRow is the logical document row.
	LogicalRow int
	// LogicalCol is the logical display column.
	LogicalCol int
	// Offset is the global display-width offset from the buffer start.
	Offset int
}

// EditSurface combines an EditBuffer with a viewport-aware TextSurface. It
// manages visual cursor navigation, scrolling, selection, and placeholder data.
type EditSurface struct {
	surface      *TextSurface
	editBuf      *EditBuffer
	scrollMargin float32 // Fraction of viewport height in [0, 0.5].
	desiredCol   *int    // Visual column preserved during vertical movement.

	// Placeholder data read by widgets when the buffer is empty.
	placeholderText  string
	placeholderFg    style.Color
	placeholderSpans []Span // Styled placeholder, taking priority over plain text.

	// Movement and deletion interceptors installed by ExtmarksController.
	overrideMoveUpVisual       func()
	overrideMoveDownVisual     func()
	overrideDeleteSelectedText func()
}

// NewEditSurface returns an edit surface with the given viewport dimensions and
// a 15-percent scroll margin.
func NewEditSurface(editBuf *EditBuffer, vpWidth, vpHeight int) *EditSurface {
	surface := NewTextSurface(editBuf.TextBuffer())
	surface.SetViewport(&Viewport{Width: vpWidth, Height: vpHeight})

	ev := &EditSurface{
		surface:      surface,
		editBuf:      editBuf,
		scrollMargin: 0.15,
	}

	// Cursor notifications reset the desired column and update scrolling.
	editBuf.OnCursorChange = func(row, col int) {
		ev.desiredCol = nil
		ev.ensureCursorVisible()
	}

	return ev
}

// SetViewportSize updates dimensions and clamps vertical scrolling to content.
func (ev *EditSurface) SetViewportSize(width, height int) {
	ev.surface.SetViewportSize(width, height)
	ev.clampScrollPosition()
}

// clampScrollPosition removes blank space below the final virtual line.
func (ev *EditSurface) clampScrollPosition() {
	vp := ev.surface.viewport
	if vp == nil || vp.Height <= 0 {
		return
	}
	ev.surface.ensureUpdated()
	totalVLines := len(ev.surface.vlines)
	if totalVLines <= vp.Height {
		// Content that fits resets vertical scrolling.
		vp.Y = 0
	} else if vp.Y > totalVLines-vp.Height {
		vp.Y = totalVLines - vp.Height
	}
}

// SetViewport replaces the viewport without scrolling it. Call ScrollToCursor
// afterward to bring the cursor into view.
func (ev *EditSurface) SetViewport(vp *Viewport) {
	ev.surface.SetViewport(vp)
}

// ScrollToCursor scrolls the viewport, if there is one, so that the cursor is
// visible.
func (ev *EditSurface) ScrollToCursor() {
	ev.ensureCursorVisible()
}

// SetWrapMode sets the underlying line-wrapping mode.
func (ev *EditSurface) SetWrapMode(mode WrapMode) {
	ev.surface.SetWrapMode(mode)
}

// SetScrollMargin sets the viewport-height fraction kept around the cursor,
// clamped to [0, 0.5].
func (ev *EditSurface) SetScrollMargin(margin float32) {
	if margin < 0 {
		margin = 0
	}
	if margin > 0.5 {
		margin = 0.5
	}
	ev.scrollMargin = margin
}

// SetSelection sets the text selection.
func (ev *EditSurface) SetSelection(sel *TextSelection) {
	ev.surface.SetSelection(sel)
}

// ResetSelection removes the current selection.
func (ev *EditSurface) ResetSelection() {
	ev.surface.ResetSelection()
}

// SetPlaceholderText stores a plain placeholder and clears styled placeholders.
func (ev *EditSurface) SetPlaceholderText(text string, fg style.Color) {
	ev.placeholderText = text
	ev.placeholderFg = fg
	ev.placeholderSpans = nil
}

// SetPlaceholderSpans retains a styled placeholder and clears plain placeholder
// text.
func (ev *EditSurface) SetPlaceholderSpans(spans []Span) {
	ev.placeholderSpans = spans
	ev.placeholderText = ""
}

// NotifyCursorChange resets the desired vertical-navigation column and scrolls
// to the current cursor. Call it after cursor changes made outside EditBuffer's
// notification path.
func (ev *EditSurface) NotifyCursorChange() {
	ev.desiredCol = nil
	ev.ensureCursorVisible()
}

// EditBuffer returns the underlying edit buffer.
func (ev *EditSurface) EditBuffer() *EditBuffer { return ev.editBuf }

// TextSurface returns the underlying text surface.
func (ev *EditSurface) TextSurface() *TextSurface { return ev.surface }

// VirtualLines scrolls to the cursor and returns viewport-visible lines.
func (ev *EditSurface) VirtualLines() []VirtualLine {
	ev.ensureCursorVisible()
	return ev.surface.VirtualLines()
}

// Viewport returns the underlying viewport.
func (ev *EditSurface) Viewport() *Viewport {
	return ev.surface.Viewport()
}

// CachedLineInfo scrolls to the cursor and returns viewport-visible metadata.
func (ev *EditSurface) CachedLineInfo() LineInfo {
	ev.ensureCursorVisible()
	return ev.surface.CachedLineInfo()
}

// Selection returns the current selection, or nil when absent.
func (ev *EditSurface) Selection() *TextSelection {
	return ev.surface.Selection()
}

// HasSelection reports whether selection data exists.
func (ev *EditSurface) HasSelection() bool {
	return ev.surface.Selection() != nil
}

// DefaultFg returns the default foreground color.
func (ev *EditSurface) DefaultFg() style.Color { return ev.surface.DefaultFg() }

// DefaultBg returns the default background color.
func (ev *EditSurface) DefaultBg() style.Color { return ev.surface.DefaultBg() }

// LineHighlights returns highlights for a logical source line.
func (ev *EditSurface) LineHighlights(lineIdx int) []Highlight {
	return ev.surface.LineHighlights(lineIdx)
}

// LineBackground returns the background for a logical source line.
func (ev *EditSurface) LineBackground(lineIdx int) style.Color {
	return ev.surface.LineBackground(lineIdx)
}

// Truncate reports whether line truncation is enabled.
func (ev *EditSurface) Truncate() bool {
	return ev.surface.Truncate()
}

// VisualCursor returns the current cursor in all supported coordinate spaces.
func (ev *EditSurface) VisualCursor() VisualCursor {
	return ev.logicalToVisualCursor(ev.editBuf.CursorRow(), ev.editBuf.CursorCol())
}

// Cursor returns the logical row and display column.
func (ev *EditSurface) Cursor() (int, int) {
	return ev.editBuf.CursorRow(), ev.editBuf.CursorCol()
}

// logicalToVisualCursor maps a logical row and display column into all cursor
// coordinate spaces.
func (ev *EditSurface) logicalToVisualCursor(row, col int) VisualCursor {
	ev.surface.ensureUpdated()

	vlines := ev.surface.vlines
	vpY := 0
	if ev.surface.viewport != nil {
		vpY = ev.surface.viewport.Y
	}

	visualRow := 0
	for i, vl := range vlines {
		if vl.SourceLine == row {
			// A wrap-boundary column belongs to the following virtual line; only a
			// logical line's true end remains on its final virtual line.
			if col >= vl.SourceColOffset && col < vl.SourceColOffset+vl.Width {
				visualRow = i - vpY
				break
			}
			// Use the final virtual line for the logical end position.
			if i+1 >= len(vlines) || vlines[i+1].SourceLine != row {
				visualRow = i - vpY
				break
			}
		} else if vl.SourceLine > row {
			// If the logical row was skipped, use the preceding virtual line.
			if i > 0 {
				visualRow = (i - 1) - vpY
			}
			break
		}
	}

	// Convert the logical column to its wrapped-line column.
	visualCol := col
	if visualRow+vpY >= 0 && visualRow+vpY < len(vlines) {
		vl := vlines[visualRow+vpY]
		visualCol = col - vl.SourceColOffset
	}

	offset := ev.editBuf.PositionToOffset(row, col)

	return VisualCursor{
		VisualRow:  visualRow,
		VisualCol:  visualCol,
		LogicalRow: row,
		LogicalCol: col,
		Offset:     offset,
	}
}

// MoveUpVisual moves the cursor up one wrapped visual line.
func (ev *EditSurface) MoveUpVisual() {
	if ev.overrideMoveUpVisual != nil {
		ev.overrideMoveUpVisual()
		return
	}
	ev.doMoveUpVisual()
}

func (ev *EditSurface) doMoveUpVisual() {
	vc := ev.VisualCursor()
	vpY := 0
	if ev.surface.viewport != nil {
		vpY = ev.surface.viewport.Y
	}

	targetVisIdx := vc.VisualRow + vpY - 1
	if targetVisIdx < 0 {
		// At the first visual line, move to its start.
		ev.editBuf.cursorCol = 0
		ev.desiredCol = nil
		ev.ensureCursorVisible()
		return
	}

	ev.surface.ensureUpdated()
	if targetVisIdx >= len(ev.surface.vlines) {
		return
	}

	desiredCol := vc.VisualCol
	if ev.desiredCol != nil {
		desiredCol = *ev.desiredCol
	} else {
		dc := desiredCol
		ev.desiredCol = &dc
	}

	vl := ev.surface.vlines[targetVisIdx]
	newCol := min(vl.SourceColOffset+desiredCol, vl.SourceColOffset+vl.Width)

	ev.editBuf.cursorRow = vl.SourceLine
	ev.editBuf.cursorCol = newCol
	ev.ensureCursorVisible()
}

// MoveDownVisual moves the cursor down one wrapped visual line.
func (ev *EditSurface) MoveDownVisual() {
	if ev.overrideMoveDownVisual != nil {
		ev.overrideMoveDownVisual()
		return
	}
	ev.doMoveDownVisual()
}

func (ev *EditSurface) doMoveDownVisual() {
	vc := ev.VisualCursor()
	vpY := 0
	if ev.surface.viewport != nil {
		vpY = ev.surface.viewport.Y
	}

	targetVisIdx := vc.VisualRow + vpY + 1

	ev.surface.ensureUpdated()
	if targetVisIdx >= len(ev.surface.vlines) {
		// At the final visual line, move to the logical line end.
		lineText := ev.editBuf.tb.LineTextAt(ev.editBuf.cursorRow)
		ev.editBuf.cursorCol = DisplayWidthOfText(lineText)
		ev.desiredCol = nil
		ev.ensureCursorVisible()
		return
	}

	desiredCol := vc.VisualCol
	if ev.desiredCol != nil {
		desiredCol = *ev.desiredCol
	} else {
		dc := desiredCol
		ev.desiredCol = &dc
	}

	vl := ev.surface.vlines[targetVisIdx]
	newCol := min(vl.SourceColOffset+desiredCol, vl.SourceColOffset+vl.Width)

	ev.editBuf.cursorRow = vl.SourceLine
	ev.editBuf.cursorCol = newCol
	ev.ensureCursorVisible()
}

// VirtualLineCount returns the total number of virtual lines.
func (ev *EditSurface) VirtualLineCount() int {
	return ev.surface.VirtualLineCount()
}

// TotalVirtualLineCount returns the same unsliced count as VirtualLineCount.
func (ev *EditSurface) TotalVirtualLineCount() int {
	return ev.surface.VirtualLineCount()
}

// SetCursorByOffset moves to a global display-width offset.
func (ev *EditSurface) SetCursorByOffset(offset int) {
	ev.editBuf.SetCursorByOffset(offset)
}

// NextWordBoundary returns the next EditBuffer word boundary as a visual cursor.
func (ev *EditSurface) NextWordBoundary() VisualCursor {
	pos := ev.editBuf.NextWordBoundary()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// PrevWordBoundary returns the previous EditBuffer word boundary as a visual
// cursor.
func (ev *EditSurface) PrevWordBoundary() VisualCursor {
	pos := ev.editBuf.PrevWordBoundary()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// EOL returns the logical line end as a visual cursor.
func (ev *EditSurface) EOL() VisualCursor {
	pos := ev.editBuf.EOL()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// VisualSOL returns the start of the current wrapped line.
func (ev *EditSurface) VisualSOL() VisualCursor {
	vc := ev.VisualCursor()
	ev.surface.ensureUpdated()

	vpY := 0
	if ev.surface.viewport != nil {
		vpY = ev.surface.viewport.Y
	}
	absIdx := vc.VisualRow + vpY
	if absIdx < 0 || absIdx >= len(ev.surface.vlines) {
		return vc
	}

	vl := ev.surface.vlines[absIdx]
	return ev.logicalToVisualCursor(vl.SourceLine, vl.SourceColOffset)
}

// VisualEOL returns the final cursor position on the current wrapped line. A
// continued line ends one column before its wrap boundary; the final wrapped
// line ends at the logical line end.
func (ev *EditSurface) VisualEOL() VisualCursor {
	vc := ev.VisualCursor()
	ev.surface.ensureUpdated()

	vpY := 0
	if ev.surface.viewport != nil {
		vpY = ev.surface.viewport.Y
	}
	absIdx := vc.VisualRow + vpY
	if absIdx < 0 || absIdx >= len(ev.surface.vlines) {
		return vc
	}

	vl := ev.surface.vlines[absIdx]
	logicalRow := vl.SourceLine

	// Keep a continued line before the boundary owned by the next virtual line.
	if absIdx+1 < len(ev.surface.vlines) && ev.surface.vlines[absIdx+1].SourceLine == logicalRow {
		endCol := vl.SourceColOffset + vl.Width
		if vl.Width > 0 {
			endCol = vl.SourceColOffset + vl.Width - 1
		}
		return ev.logicalToVisualCursor(logicalRow, endCol)
	}

	// The final virtual line owns the logical end position.
	lineW := ev.editBuf.TextBuffer().LineWidthAt(logicalRow)
	return ev.logicalToVisualCursor(logicalRow, lineW)
}

// Text returns the complete unstyled content.
func (ev *EditSurface) Text() string {
	return ev.editBuf.Text()
}

// DeleteSelectedText removes the normalized current selection.
func (ev *EditSurface) DeleteSelectedText() {
	if ev.overrideDeleteSelectedText != nil {
		ev.overrideDeleteSelectedText()
		return
	}
	ev.doDeleteSelectedText()
}

func (ev *EditSurface) doDeleteSelectedText() {
	sel := ev.surface.Selection()
	if sel == nil {
		return
	}
	start, end := sel.Start, sel.End
	if start > end {
		start, end = end, start
	}

	// Convert global offsets to logical deletion endpoints.
	startPos := ev.editBuf.OffsetToPosition(start)
	endPos := ev.editBuf.OffsetToPosition(end)
	ev.editBuf.DeleteRange(startPos.Row, startPos.Col, endPos.Row, endPos.Col)
	ev.surface.ResetSelection()
}

// SetLocalSelection maps viewport-relative endpoints into a selection. It can
// also move the cursor to the focus and scroll to follow it.
func (ev *EditSurface) SetLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg style.Color, updateCursor, followCursor bool) bool {
	ok := ev.surface.SetLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
	if ok && updateCursor {
		// Move the cursor to a valid focus offset when requested.
		off := ev.surface.visualToCharOffset(focusX, focusY)
		if off >= 0 {
			ev.editBuf.SetCursorByOffset(off)
			if followCursor {
				ev.ensureCursorVisible()
			}
		}
	}
	return ok
}

// UpdateSelection updates the selection end offset.
func (ev *EditSurface) UpdateSelection(end int, bg, fg style.Color) {
	ev.surface.UpdateSelection(end, bg, fg)
}

// UpdateLocalSelection replaces selection from viewport-relative coordinates.
func (ev *EditSurface) UpdateLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg style.Color) bool {
	return ev.surface.UpdateLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
}

// ResetLocalSelection clears the local selection.
func (ev *EditSurface) ResetLocalSelection() {
	ev.surface.ResetLocalSelection()
}

// SelectedText returns selected text, or an empty string.
func (ev *EditSurface) SelectedText() string {
	return ev.surface.SelectedText()
}

// LogicalLineInfo returns unwrapped logical-line metadata.
func (ev *EditSurface) LogicalLineInfo() LineInfo {
	return ev.surface.LogicalLineInfo()
}

func (ev *EditSurface) ensureCursorVisible() {
	vp := ev.surface.viewport
	if vp == nil {
		return
	}
	if vp.Height == 0 {
		return
	}

	vc := ev.logicalToVisualCursor(ev.editBuf.CursorRow(), ev.editBuf.CursorCol())
	ev.surface.ensureUpdated()
	totalVLines := len(ev.surface.vlines)

	// Recover the cursor's absolute virtual-line index.
	absVisRow := vc.VisualRow + vp.Y

	marginLines := min(max(int(float32(vp.Height)*ev.scrollMargin), 1), (vp.Height-1)/2)

	newY := vp.Y

	// Scroll upward when the cursor enters the top margin.
	if absVisRow < vp.Y+marginLines {
		newY = max(absVisRow-marginLines, 0)
	}
	// Scroll downward when the cursor enters the bottom margin.
	if absVisRow >= vp.Y+vp.Height-marginLines {
		newY = max(min(absVisRow-vp.Height+marginLines+1, totalVLines-vp.Height), 0)
	}

	if newY != vp.Y {
		vp.Y = newY
	}

	// Unwrapped text scrolls horizontally to follow the logical column.
	if ev.surface.wrapMode == WrapNone {
		if vc.LogicalCol < vp.X {
			vp.X = vc.LogicalCol
		}
		if vc.LogicalCol >= vp.X+vp.Width {
			vp.X = vc.LogicalCol - vp.Width + 1
		}
	}
}

// IsEmpty reports whether the buffer contains one empty logical line.
func (ev *EditSurface) IsEmpty() bool {
	tb := ev.editBuf.TextBuffer()
	return tb.LineCount() == 1 && tb.LineTextAt(0) == ""
}

// PlaceholderText returns the configured plain placeholder and foreground.
func (ev *EditSurface) PlaceholderText() (string, style.Color) {
	return ev.placeholderText, ev.placeholderFg
}

// PlaceholderSpans returns the underlying configured placeholder spans.
func (ev *EditSurface) PlaceholderSpans() []Span {
	return ev.placeholderSpans
}

// MeasureForDimensions delegates nonmutating measurement to TextSurface.
func (ev *EditSurface) MeasureForDimensions(width, height int) *MeasureDimensions {
	return ev.surface.MeasureForDimensions(width, height)
}
