package textbuf

import (
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// VisualCursor
// ---------------------------------------------------------------------------

// VisualCursor represents a cursor position with both visual and logical coords.
// Visual coordinates (VisualRow, VisualCol) are VIEWPORT-RELATIVE.
type VisualCursor struct {
	VisualRow  int // viewport-relative row (0 = top of viewport)
	VisualCol  int // viewport-relative column
	LogicalRow int // document-absolute row
	LogicalCol int // document-absolute column
	Offset     int // global display-width offset from buffer start
}

// ---------------------------------------------------------------------------
// EditSurface
// ---------------------------------------------------------------------------

// EditSurface wraps a TextSurface and an EditBuffer. It manages viewport
// scrolling to keep the cursor visible, visual cursor computation, and
// placeholder text. It implements Surface.
type EditSurface struct {
	surface      *TextSurface
	editBuf      *EditBuffer
	scrollMargin float32 // fraction of viewport height (0.0–0.5)
	desiredCol   *int    // preserved visual column for up/down

	// Placeholder (shown when buffer is empty)
	placeholderText  string
	placeholderFg    style.Color
	placeholderSpans []Span // span-based placeholder (takes priority over plain)

	// Tab indicator (delegated to surface)
	tabIndicator      rune
	tabIndicatorColor *style.Color

	// Override function fields — set by ExtmarksController (same package).
	overrideMoveUpVisual       func()
	overrideMoveDownVisual     func()
	overrideDeleteSelectedText func()
}

// NewEditSurface creates an EditSurface.
func NewEditSurface(editBuf *EditBuffer, vpWidth, vpHeight int) *EditSurface {
	surface := NewTextSurface(editBuf.TextBuffer())
	surface.SetViewport(&Viewport{Width: vpWidth, Height: vpHeight})

	ev := &EditSurface{
		surface:      surface,
		editBuf:      editBuf,
		scrollMargin: 0.15,
	}

	// Hook cursor changes to auto-scroll.
	editBuf.OnCursorChange = func(row, col int) {
		ev.desiredCol = nil
		ev.ensureCursorVisible()
	}

	return ev
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// SetViewportSize updates the viewport dimensions and clamps the scroll
// position so that content fills the viewport (no blank space at the bottom).
func (ev *EditSurface) SetViewportSize(width, height int) {
	ev.surface.SetViewportSize(width, height)
	ev.clampScrollPosition()
}

// clampScrollPosition adjusts vp.Y so there is no blank space beyond the
// content. When the viewport is large enough to show all virtual lines the
// scroll offset is reset to 0.
func (ev *EditSurface) clampScrollPosition() {
	vp := ev.surface.viewport
	if vp == nil || vp.Height <= 0 {
		return
	}
	ev.surface.ensureUpdated()
	totalVLines := len(ev.surface.vlines)
	if totalVLines <= vp.Height {
		// All content fits — no scrolling needed.
		vp.Y = 0
	} else if vp.Y > totalVLines-vp.Height {
		vp.Y = totalVLines - vp.Height
	}
}

// SetViewport sets the viewport fully.
// If moveCursor is true, the cursor is adjusted to stay within the viewport.
func (ev *EditSurface) SetViewport(vp *Viewport, moveCursor ...bool) {
	ev.surface.SetViewport(vp)
	if len(moveCursor) > 0 && moveCursor[0] && vp != nil {
		ev.ensureCursorVisible()
	}
}

// SetWrapMode sets the wrap mode on the underlying surface.
func (ev *EditSurface) SetWrapMode(mode WrapMode) {
	ev.surface.SetWrapMode(mode)
}

// SetScrollMargin sets the scroll margin as a fraction of viewport height.
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

// ResetSelection clears selection.
func (ev *EditSurface) ResetSelection() {
	ev.surface.ResetSelection()
}

// SetPlaceholderText sets placeholder text shown when content is empty.
func (ev *EditSurface) SetPlaceholderText(text string, fg style.Color) {
	ev.placeholderText = text
	ev.placeholderFg = fg
	ev.placeholderSpans = nil
}

// SetPlaceholderSpans sets a span-based placeholder shown when the buffer is empty.
func (ev *EditSurface) SetPlaceholderSpans(spans []Span) {
	ev.placeholderSpans = spans
	ev.placeholderText = ""
}

// SetTabIndicator sets the tab indicator codepoint.
func (ev *EditSurface) SetTabIndicator(indicator rune) {
	ev.tabIndicator = indicator
	ev.surface.SetTabIndicator(indicator)
}

// SetTabIndicatorColor sets the tab indicator color.
func (ev *EditSurface) SetTabIndicatorColor(color style.Color) {
	ev.tabIndicatorColor = &color
	ev.surface.SetTabIndicatorColor(color)
}

// NotifyCursorChange should be called when the cursor position changes
// through non-visual means (e.g. left/right movement, home/end, word jumps).
// It resets the sticky desired column used by visual up/down navigation
// and ensures the cursor remains visible in the viewport.
func (ev *EditSurface) NotifyCursorChange() {
	ev.desiredCol = nil
	ev.ensureCursorVisible()
}

// EditBuffer returns the underlying EditBuffer.
func (ev *EditSurface) EditBuffer() *EditBuffer { return ev.editBuf }

// TextSurface returns the underlying TextSurface.
func (ev *EditSurface) TextSurface() *TextSurface { return ev.surface }

// ---------------------------------------------------------------------------
// Surface implementation
// ---------------------------------------------------------------------------

// VirtualLines returns the visible virtual lines.
func (ev *EditSurface) VirtualLines() []VirtualLine {
	ev.ensureCursorVisible()
	return ev.surface.VirtualLines()
}

// Viewport returns the viewport.
func (ev *EditSurface) Viewport() *Viewport {
	return ev.surface.Viewport()
}

// CachedLineInfo returns layout metadata.
func (ev *EditSurface) CachedLineInfo() LineInfo {
	ev.ensureCursorVisible()
	return ev.surface.CachedLineInfo()
}

// Selection returns the current selection or nil.
func (ev *EditSurface) Selection() *TextSelection {
	return ev.surface.Selection()
}

// HasSelection returns true if a selection is active.
func (ev *EditSurface) HasSelection() bool {
	return ev.surface.Selection() != nil
}

// DefaultFg returns the default foreground color.
func (ev *EditSurface) DefaultFg() style.Color { return ev.surface.DefaultFg() }

// DefaultBg returns the default background color.
func (ev *EditSurface) DefaultBg() style.Color { return ev.surface.DefaultBg() }

// LineHighlights returns the highlights for a source (physical) line.
func (ev *EditSurface) LineHighlights(lineIdx int) []Highlight {
	return ev.surface.LineHighlights(lineIdx)
}

// LineBackground returns the per-line background for a source line.
func (ev *EditSurface) LineBackground(lineIdx int) style.Color {
	return ev.surface.LineBackground(lineIdx)
}

// Truncate returns whether line truncation is enabled.
func (ev *EditSurface) Truncate() bool {
	return ev.surface.Truncate()
}

// ---------------------------------------------------------------------------
// Visual cursor
// ---------------------------------------------------------------------------

// VisualCursor returns the cursor position in visual coordinates.
func (ev *EditSurface) VisualCursor() VisualCursor {
	return ev.logicalToVisualCursor(ev.editBuf.CursorRow(), ev.editBuf.CursorCol())
}

// Cursor returns the logical cursor (row, col).
func (ev *EditSurface) Cursor() (int, int) {
	return ev.editBuf.CursorRow(), ev.editBuf.CursorCol()
}

// logicalToVisualCursor converts logical (row, col) to a VisualCursor.
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
			// Check if this virtual line contains our column.
			// Use < (not <=) so that a cursor exactly at a wrap boundary
			// maps to the START of the next virtual line, not the end of
			// the previous one. The fallback path (last-vline check below)
			// correctly handles the cursor at the true end of a logical line.
			if col >= vl.SourceColOffset && col < vl.SourceColOffset+vl.Width {
				visualRow = i - vpY
				break
			}
			// If it's the last vline for this source line, use it
			if i+1 >= len(vlines) || vlines[i+1].SourceLine != row {
				visualRow = i - vpY
				break
			}
		} else if vl.SourceLine > row {
			// Passed our line, use previous
			if i > 0 {
				visualRow = (i - 1) - vpY
			}
			break
		}
	}

	// Visual column is relative to the vline's source col offset
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

// ---------------------------------------------------------------------------
// Visual navigation
// ---------------------------------------------------------------------------

// MoveUpVisual moves the cursor up by one visual line.
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
		// Already on the first visual line — move to start of line.
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

// MoveDownVisual moves the cursor down by one visual line.
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
		// Already on the last visual line — move to end of line.
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

// TotalVirtualLineCount returns the total virtual line count.
func (ev *EditSurface) TotalVirtualLineCount() int {
	return ev.surface.VirtualLineCount()
}

// ---------------------------------------------------------------------------
// Cursor by offset
// ---------------------------------------------------------------------------

// SetCursorByOffset sets the cursor by global display-width offset.
func (ev *EditSurface) SetCursorByOffset(offset int) {
	ev.editBuf.SetCursorByOffset(offset)
}

// ---------------------------------------------------------------------------
// Word boundary helpers (visual)
// ---------------------------------------------------------------------------

// NextWordBoundary returns the next word boundary as a VisualCursor.
func (ev *EditSurface) NextWordBoundary() VisualCursor {
	pos := ev.editBuf.NextWordBoundary()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// PrevWordBoundary returns the previous word boundary as a VisualCursor.
func (ev *EditSurface) PrevWordBoundary() VisualCursor {
	pos := ev.editBuf.PrevWordBoundary()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// EOL returns the logical end-of-line as a VisualCursor.
func (ev *EditSurface) EOL() VisualCursor {
	pos := ev.editBuf.EOL()
	return ev.logicalToVisualCursor(pos.Row, pos.Col)
}

// ---------------------------------------------------------------------------
// Visual SOL / EOL (wrap-aware)
// ---------------------------------------------------------------------------

// VisualSOL returns the visual start of the current wrapped line.
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

// VisualEOL returns the visual end of the current wrapped line.
// For wrapped continuation lines, returns width-1 to keep the cursor on the
// current visual line. For the last virtual line of a logical line, returns
// the true end-of-line position.
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

	// Check if the next virtual line is a continuation of the same logical line.
	// If so, return width-1 to stay on the current visual line (the position at
	// SourceColOffset+Width is a wrap boundary that maps to the next vline).
	if absIdx+1 < len(ev.surface.vlines) && ev.surface.vlines[absIdx+1].SourceLine == logicalRow {
		endCol := vl.SourceColOffset + vl.Width
		if vl.Width > 0 {
			endCol = vl.SourceColOffset + vl.Width - 1
		}
		return ev.logicalToVisualCursor(logicalRow, endCol)
	}

	// Last virtual line for this logical line: use the true end of the logical line.
	lineW := ev.editBuf.TextBuffer().LineWidthAt(logicalRow)
	return ev.logicalToVisualCursor(logicalRow, lineW)
}

// ---------------------------------------------------------------------------
// Text access
// ---------------------------------------------------------------------------

// Text returns the full text content.
func (ev *EditSurface) Text() string {
	return ev.editBuf.Text()
}

// ---------------------------------------------------------------------------
// Selection helpers
// ---------------------------------------------------------------------------

// DeleteSelectedText deletes the currently selected text.
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

	// Convert offsets to row/col
	startPos := ev.editBuf.OffsetToPosition(start)
	endPos := ev.editBuf.OffsetToPosition(end)
	ev.editBuf.DeleteRange(startPos.Row, startPos.Col, endPos.Row, endPos.Col)
	ev.surface.ResetSelection()
}

// SetLocalSelection sets selection by visual viewport-relative coordinates.
// Returns true if valid.
func (ev *EditSurface) SetLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg *style.Color, updateCursor, followCursor bool) bool {
	ok := ev.surface.SetLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
	if ok && updateCursor {
		// Move cursor to focus position
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
func (ev *EditSurface) UpdateSelection(end int, bg, fg *style.Color) {
	ev.surface.UpdateSelection(end, bg, fg)
}

// UpdateLocalSelection updates the selection by visual coordinates.
func (ev *EditSurface) UpdateLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg *style.Color) bool {
	return ev.surface.UpdateLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
}

// ResetLocalSelection clears the local selection.
func (ev *EditSurface) ResetLocalSelection() {
	ev.surface.ResetLocalSelection()
}

// SelectedText returns the selected text, or "".
func (ev *EditSurface) SelectedText() string {
	return ev.surface.SelectedText()
}

// ---------------------------------------------------------------------------
// Line info
// ---------------------------------------------------------------------------

// LogicalLineInfo returns logical (unwrapped) line info.
func (ev *EditSurface) LogicalLineInfo() LineInfo {
	return ev.surface.LogicalLineInfo()
}

// ---------------------------------------------------------------------------
// Scroll management
// ---------------------------------------------------------------------------

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

	// Find absolute visual line index of cursor
	absVisRow := vc.VisualRow + vp.Y

	marginLines := max(int(float32(vp.Height)*ev.scrollMargin), 1)
	if marginLines > (vp.Height-1)/2 {
		marginLines = (vp.Height - 1) / 2
	}

	newY := vp.Y

	// Cursor above viewport
	if absVisRow < vp.Y+marginLines {
		newY = max(absVisRow-marginLines, 0)
	}
	// Cursor below viewport
	if absVisRow >= vp.Y+vp.Height-marginLines {
		newY = min(absVisRow-vp.Height+marginLines+1, totalVLines-vp.Height)
		if newY < 0 {
			newY = 0
		}
	}

	if newY != vp.Y {
		vp.Y = newY
	}

	// Horizontal scroll (for no-wrap mode)
	if ev.surface.wrapMode == WrapNone {
		// Ensure cursor column is visible
		if vc.LogicalCol < vp.X {
			vp.X = vc.LogicalCol
		}
		if vc.LogicalCol >= vp.X+vp.Width {
			vp.X = vc.LogicalCol - vp.Width + 1
		}
	}
}

// IsEmpty returns true if the edit buffer has no content.
func (ev *EditSurface) IsEmpty() bool {
	tb := ev.editBuf.TextBuffer()
	return tb.LineCount() == 1 && tb.LineTextAt(0) == ""
}

// PlaceholderText returns the placeholder text and color.
func (ev *EditSurface) PlaceholderText() (string, style.Color) {
	return ev.placeholderText, ev.placeholderFg
}

// PlaceholderSpans returns the placeholder spans, or nil.
func (ev *EditSurface) PlaceholderSpans() []Span {
	return ev.placeholderSpans
}

// MeasureForDimensions computes the layout that would result from the given
// dimensions without changing the viewport. Delegates to the underlying
// TextSurface. Returns nil if the buffer is empty.
func (ev *EditSurface) MeasureForDimensions(width, height int) *MeasureDimensions {
	return ev.surface.MeasureForDimensions(width, height)
}
