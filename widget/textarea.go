package widget

import (
	"math"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// Textarea is a multi-line text editor with cursor, undo/redo, word wrap,
// selection, mouse support, syntax highlighting, and configurable keybindings.
//
// Architecture:
//   - Delegates text storage, cursor, undo/redo to textbuf.EditBuffer.
//   - Delegates viewport, scrolling, visual cursor, wrap to textbuf.EditSurface.
//   - Renders via buffer.DrawEditSurface (single call).
//   - Uses a handler-map dispatch pattern: action → func() bool.
//   - Read-only gating uses a navActions whitelist.
//   - Keybinding compilation is lazy: dirty flag avoids redundant rebuilds.
type Textarea struct {
	*render.BaseRenderable

	editSurface *textbuf.EditSurface

	// Visual config — separate focused/unfocused colors
	unfocusedTextColor style.Color
	unfocusedBgColor   style.Color
	focusedTextColor   style.Color
	focusedBgColor     style.Color
	textColor          style.Color // active text color (swapped on focus/blur)
	backgroundColor    style.Color // active bg color
	cursorColor        style.Color
	selectionBg        style.Color
	selectionFg        *style.Color // optional selection foreground

	showCursor     bool
	cursorShape    string // "block", "line", or "underline"
	cursorBlinking bool
	readOnly       bool

	// Keybindings — lazy-compiled
	keyBindings   []input.KeyBinding // custom overrides
	keyAliases    input.KeyAliasMap  // key alias overrides
	bindingMap    map[string]string  // compiled lookup
	bindingsDirty bool

	// Handler map — action string → handler func
	actionHandlers map[string]func() bool

	// Selection tracking
	// When a Context is available, selection is driven through it
	// (ctx.StartSelection / ctx.UpdateSelection / ctx.ClearSelection).
	// When no context is present (e.g. tests), we fall back to internal
	// offset-based tracking.
	selAnchorOffset         int  // anchor character offset (internal fallback)
	keyboardSelectionActive bool // true while shift+nav is extending a selection

	// Default text attributes (bold, italic, etc.)
	defaultAttributes uint32

	// initialValue one-shot guard
	initialValueSet bool

	// Mouse scroll auto-scroll state
	autoScrollVelocity    float64
	autoScrollAccumulator float64
	scrollSpeed           float64

	// Mouse selection anchor (viewport-local)
	mouseSelAnchorX int
	mouseSelAnchorY int
	mouseSelecting  bool

	// Extmarks — lazily created
	extmarksCtrl *textbuf.ExtmarksController

	// Event callbacks
	OnCursorChange  func(row, col int)
	OnContentChange func(text string)
	OnSubmit        func(text string)
}

// TextareaOption configures a Textarea during construction.
type TextareaOption func(*Textarea)

// NewTextarea creates a Textarea with optional configuration.
func NewTextarea(opts ...TextareaOption) *Textarea {
	tb := textbuf.NewTextBuffer()
	eb := textbuf.NewEditBuffer(tb)
	ev := textbuf.NewEditSurface(eb, 0, 0)

	defText := style.ColorWhite
	defBg := style.ColorTransparent

	ta := &Textarea{
		BaseRenderable:     render.NewBaseRenderable(render.WithFocusable(true), render.WithSelectable(true)),
		editSurface:        ev,
		unfocusedTextColor: defText,
		unfocusedBgColor:   defBg,
		focusedTextColor:   defText,
		focusedBgColor:     defBg,
		textColor:          defText,
		backgroundColor:    defBg,
		cursorColor:        style.ColorWhite,
		selectionBg:        style.ColorFromInts(51, 153, 255, 255),
		showCursor:         true,
		cursorShape:        "block",
		cursorBlinking:     true,
		bindingsDirty:      true,
		scrollSpeed:        16,
	}

	// Wire cursor change callback — chain with EditSurface's internal
	// handler so desiredCol is reset and cursor visibility is maintained
	// on non-visual cursor movements (left/right, home/end, word jumps).
	eb.OnCursorChange = func(row, col int) {
		ev.NotifyCursorChange()
		if ta.OnCursorChange != nil {
			ta.OnCursorChange(row, col)
		}
	}

	for _, opt := range opts {
		opt(ta)
	}

	// Apply text color and tab expansion on the TextBuffer
	tb.SetDefaultFg(ta.textColor)
	tb.SetExpandTabs(true)
	if ta.defaultAttributes != 0 {
		tb.SetDefaultAttributes(ta.defaultAttributes)
	}

	// Build the handler map
	ta.actionHandlers = ta.buildActionHandlers()

	// Set up intrinsic measurement so the layout engine can auto-size
	// when Width or Height is Auto. Follows the same pattern as Code widget.
	ta.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		availW := width
		if widthMode == layout.MeasureUndefined {
			availW = 0 // no constraint → max-content (no wrapping)
		}
		availH := height
		if heightMode == layout.MeasureUndefined {
			availH = 0
		}

		m := ta.editSurface.MeasureForDimensions(availW, availH)
		measuredW := 1
		measuredH := 1
		if m != nil {
			if m.MaxWidth > 0 {
				measuredW = m.MaxWidth
			}
			if m.LineCount > 0 {
				measuredH = m.LineCount
			}
		}

		switch widthMode {
		case layout.MeasureExactly:
			measuredW = width
		case layout.MeasureAtMost:
			if measuredW > width {
				measuredW = width
			}
		}

		switch heightMode {
		case layout.MeasureExactly:
			measuredH = height
		case layout.MeasureAtMost:
			if measuredH > height {
				measuredH = height
			}
		}

		return layout.MeasureResult{Width: measuredW, Height: measuredH}
	}

	return ta
}

// ---------------------------------------------------------------------------
// Public API — Text
// ---------------------------------------------------------------------------

// SetText replaces all text content and clears undo history.
func (ta *Textarea) SetText(text string) {
	ta.editBuffer().SetText(text)
	ta.clearSelection()
	ta.syncViewport()
	ta.LayoutNode().MarkDirty()
	ta.RequestRender()
}

// ReplaceText replaces all text while preserving undo history (creates an undo point).
func (ta *Textarea) ReplaceText(text string) {
	ta.editBuffer().ReplaceText(text)
	ta.clearSelection()
	ta.syncViewport()
	ta.emitContentChange()
	ta.RequestRender()
}

// Clear clears all text content (undoable).
func (ta *Textarea) Clear() {
	ta.editBuffer().Clear()
	ta.clearSelection()
	ta.syncViewport()
	ta.emitContentChange()
	ta.RequestRender()
}

// Text returns all text joined by newlines.
func (ta *Textarea) Text() string {
	return ta.editBuffer().Text()
}

// InsertText inserts text at the current cursor position.
func (ta *Textarea) InsertText(text string) {
	if ta.readOnly {
		return
	}
	ta.deleteSelectedText()
	ta.editBuffer().InsertText(text)
	ta.emitContentChange()
	ta.RequestRender()
}

// LineCount returns the number of logical lines.
func (ta *Textarea) LineCount() int {
	return ta.editBuffer().LineCount()
}

// ---------------------------------------------------------------------------
// Public API — Cursor
// ---------------------------------------------------------------------------

// SetCursor sets the cursor position (row, col in display columns).
func (ta *Textarea) SetCursor(row, col int) {
	ta.editBuffer().SetCursor(row, col)
	ta.syncViewport()
	ta.RequestRender()
}

// CursorPosition returns the current cursor position.
func (ta *Textarea) CursorPosition() (row, col int) {
	return ta.editBuffer().CursorRow(), ta.editBuffer().CursorCol()
}

// SetCursorByOffset sets the cursor by global display-width offset.
func (ta *Textarea) SetCursorByOffset(offset int) {
	ta.editSurface.SetCursorByOffset(offset)
	ta.RequestRender()
}

// CursorOffset returns the cursor position as a global display-width offset.
func (ta *Textarea) CursorOffset() int {
	eb := ta.editBuffer()
	return eb.PositionToOffset(eb.CursorRow(), eb.CursorCol())
}

// GotoLine moves the cursor to the start of the given line.
func (ta *Textarea) GotoLine(line int) {
	ta.editBuffer().GotoLine(line)
	ta.syncViewport()
	ta.RequestRender()
}

// GotoBufferEnd moves the cursor to the end of the buffer.
func (ta *Textarea) GotoBufferEnd() {
	eb := ta.editBuffer()
	lastRow := eb.LineCount() - 1
	lastCol := ta.textBuffer().LineWidthAt(lastRow)
	eb.SetCursor(lastRow, lastCol)
	ta.syncViewport()
	ta.RequestRender()
}

// GotoLineEnd moves cursor to the end of the current line.
func (ta *Textarea) GotoLineEnd() {
	eol := ta.editBuffer().EOL()
	ta.editBuffer().SetCursor(eol.Row, eol.Col)
	ta.RequestRender()
}

// ---------------------------------------------------------------------------
// Public API — Selection
// ---------------------------------------------------------------------------

// SelectedText returns the currently selected text.
func (ta *Textarea) SelectedText() string {
	return ta.editSurface.SelectedText()
}

// SelectAll selects all text.
func (ta *Textarea) SelectAll() {
	eb := ta.editBuffer()
	totalLen := ta.textBuffer().Length()
	if totalLen == 0 {
		return
	}

	ta.selAnchorOffset = 0
	ta.keyboardSelectionActive = true

	sel := &textbuf.TextSelection{
		Start:   0,
		End:     totalLen,
		BgColor: &ta.selectionBg,
		FgColor: ta.selectionFg,
	}
	ta.editSurface.SetSelection(sel)

	// Inform context
	ctx := ta.Context()
	ctx.StartSelection(ta, ta.X(), ta.Y())

	// Move cursor to end
	lastRow := eb.LineCount() - 1
	lastCol := ta.textBuffer().LineWidthAt(lastRow)
	eb.SetCursor(lastRow, lastCol)

	// Update context focus
	vc := ta.editSurface.VisualCursor()
	ctx.UpdateSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow,
		render.SelectionUpdateOption{FinishDragging: true})

	ta.RequestRender()
}

// ---------------------------------------------------------------------------
// Public API — Undo / Redo
// ---------------------------------------------------------------------------

// Undo undoes the last operation. Returns true if there was something to undo.
func (ta *Textarea) Undo() bool {
	ok := ta.editBuffer().Undo()
	if ok {
		ta.clearSelection()
		ta.syncViewport()
		ta.emitContentChange()
		ta.RequestRender()
	}
	return ok
}

// Redo redoes the last undone operation. Returns true if there was something to redo.
func (ta *Textarea) Redo() bool {
	ok := ta.editBuffer().Redo()
	if ok {
		ta.clearSelection()
		ta.syncViewport()
		ta.emitContentChange()
		ta.RequestRender()
	}
	return ok
}

// CanUndo returns whether undo is available.
func (ta *Textarea) CanUndo() bool { return ta.editBuffer().CanUndo() }

// CanRedo returns whether redo is available.
func (ta *Textarea) CanRedo() bool { return ta.editBuffer().CanRedo() }

// ---------------------------------------------------------------------------
// Public API — Scroll
// ---------------------------------------------------------------------------

// ScrollTo scrolls so the given row is visible.
func (ta *Textarea) ScrollTo(row int) {
	vp := ta.editSurface.TextSurface().Viewport()
	if vp == nil {
		return
	}
	if row < 0 {
		row = 0
	}
	maxRow := ta.editBuffer().LineCount() - 1
	if row > maxRow {
		row = maxRow
	}
	vp.Y = row
	ta.RequestRender()
}

// ScrollY returns the current vertical scroll offset (viewport Y).
func (ta *Textarea) ScrollY() int {
	vp := ta.editSurface.TextSurface().Viewport()
	if vp == nil {
		return 0
	}
	return vp.Y
}

// ---------------------------------------------------------------------------
// Public API — Virtual lines
// ---------------------------------------------------------------------------

// VirtualLineCount returns the number of visible virtual (wrapped) lines in the viewport.
func (ta *Textarea) VirtualLineCount() int {
	return ta.editSurface.VirtualLineCount()
}

// TotalVirtualLineCount returns the total number of virtual (wrapped) lines.
func (ta *Textarea) TotalVirtualLineCount() int {
	return ta.editSurface.TotalVirtualLineCount()
}

// ---------------------------------------------------------------------------
// Public API — Text range access
// ---------------------------------------------------------------------------

// TextRange returns text between start and end display-width offsets.
func (ta *Textarea) TextRange(startOffset, endOffset int) string {
	return ta.editBuffer().TextRange(startOffset, endOffset)
}

// TextRangeByCoords returns text between (startRow, startCol) and (endRow, endCol).
func (ta *Textarea) TextRangeByCoords(startRow, startCol, endRow, endCol int) string {
	return ta.editBuffer().TextRangeByCoords(startRow, startCol, endRow, endCol)
}

// ---------------------------------------------------------------------------
// Public API — Line info
// ---------------------------------------------------------------------------

// LineInfo returns the logical line layout metadata.
func (ta *Textarea) LineInfo() textbuf.LineInfo {
	return ta.editSurface.LogicalLineInfo()
}

// ---------------------------------------------------------------------------
// Public API — Syntax highlighting
// ---------------------------------------------------------------------------

// SetSyntaxStyle attaches a SyntaxStyle registry for token coloring.
func (ta *Textarea) SetSyntaxStyle(ss *style.SyntaxStyle) {
	ta.textBuffer().SetSyntaxStyle(ss)
	ta.RequestRender()
}

// SyntaxStyle returns the attached SyntaxStyle registry, or nil.
func (ta *Textarea) SyntaxStyle() *style.SyntaxStyle {
	return ta.textBuffer().SyntaxStyle()
}

// AddHighlight adds a highlight to the given line.
func (ta *Textarea) AddHighlight(lineIdx int, hl textbuf.Highlight) {
	ta.textBuffer().AddHighlight(lineIdx, hl)
	ta.RequestRender()
}

// AddHighlightByCharRange adds a highlight using global character offset range.
func (ta *Textarea) AddHighlightByCharRange(hl textbuf.Highlight) {
	ta.textBuffer().AddHighlightByCharRange(hl)
	ta.RequestRender()
}

// RemoveHighlightsByRef removes all highlights with the given ref.
func (ta *Textarea) RemoveHighlightsByRef(ref int) {
	ta.textBuffer().RemoveHighlightsByRef(ref)
	ta.RequestRender()
}

// ClearLineHighlights removes all highlights on the given line.
func (ta *Textarea) ClearLineHighlights(lineIdx int) {
	ta.textBuffer().ClearLineHighlights(lineIdx)
	ta.RequestRender()
}

// ClearAllHighlights removes all highlights.
func (ta *Textarea) ClearAllHighlights() {
	ta.textBuffer().ClearAllHighlights()
	ta.RequestRender()
}

// LineHighlights returns the highlights for a specific line.
func (ta *Textarea) LineHighlights(lineIdx int) []textbuf.Highlight {
	return ta.textBuffer().LineHighlights(lineIdx)
}

// ---------------------------------------------------------------------------
// Public API — Configuration
// ---------------------------------------------------------------------------

// SetWrapMode sets the text wrapping mode.
func (ta *Textarea) SetWrapMode(m WrapMode) {
	ta.editSurface.SetWrapMode(m)
	ta.RequestRender()
}

// SetReadOnly sets the read-only state.
func (ta *Textarea) SetReadOnly(v bool) {
	ta.readOnly = v
}

// ReadOnly returns the read-only state.
func (ta *Textarea) ReadOnly() bool {
	return ta.readOnly
}

// SetKeyBindings sets custom keybinding overrides.
func (ta *Textarea) SetKeyBindings(bindings []input.KeyBinding) {
	ta.keyBindings = bindings
	ta.bindingsDirty = true
}

// SetKeyAliases sets custom key alias overrides.
func (ta *Textarea) SetKeyAliases(aliases input.KeyAliasMap) {
	ta.keyAliases = aliases
	ta.bindingsDirty = true
}

// SetSelectionFg sets the selection foreground color.
func (ta *Textarea) SetSelectionFg(c style.Color) {
	ta.selectionFg = &c
}

// SetSelectionBg sets the selection background color.
func (ta *Textarea) SetSelectionBg(c style.Color) {
	ta.selectionBg = c
}

// SetPlaceholder sets the placeholder text shown when empty and unfocused.
func (ta *Textarea) SetPlaceholder(text string) {
	ta.editSurface.SetPlaceholderText(text, style.ColorGray)
}

// SetPlaceholderSpans sets a span-based placeholder.
func (ta *Textarea) SetPlaceholderSpans(spans []textbuf.Span) {
	ta.editSurface.SetPlaceholderSpans(spans)
}

// SetCursorShape sets the cursor shape: "block", "line", or "underline".
func (ta *Textarea) SetCursorShape(shape string) {
	ta.cursorShape = shape
	if ta.Focused() {
		ta.RequestRender()
	}
}

// CursorShape returns the current cursor shape.
func (ta *Textarea) CursorShape() string {
	return ta.cursorShape
}

// SetCursorBlinking sets whether the cursor blinks.
func (ta *Textarea) SetCursorBlinking(b bool) {
	ta.cursorBlinking = b
	if ta.Focused() {
		ta.RequestRender()
	}
}

// SetDefaultAttributes sets the default text attributes (bold, italic, etc.).
func (ta *Textarea) SetDefaultAttributes(attrs uint32) {
	ta.defaultAttributes = attrs
	ta.textBuffer().SetDefaultAttributes(attrs)
	ta.RequestRender()
}

// DefaultAttributes returns the default text attributes.
func (ta *Textarea) DefaultAttributes() uint32 {
	return ta.defaultAttributes
}

// SetScrollSpeed sets the mouse scroll speed (lines per second during auto-scroll).
func (ta *Textarea) SetScrollSpeed(speed float64) {
	if speed < 0 {
		speed = 0
	}
	ta.scrollSpeed = speed
}

// EditSurface returns the underlying EditSurface for advanced access.
func (ta *Textarea) EditSurface() *textbuf.EditSurface {
	return ta.editSurface
}

// EditBuffer returns the underlying EditBuffer for advanced access.
func (ta *Textarea) EditBuffer() *textbuf.EditBuffer {
	return ta.editBuffer()
}

// TextBuffer returns the underlying TextBuffer for advanced access.
func (ta *Textarea) TextBuffer() *textbuf.TextBuffer {
	return ta.textBuffer()
}

// Extmarks returns the ExtmarksController, creating it on first access.
// The controller intercepts cursor movement, edits, and undo/redo to
// keep extmark positions in sync with the buffer content.
func (ta *Textarea) Extmarks() *textbuf.ExtmarksController {
	if ta.extmarksCtrl == nil {
		ta.extmarksCtrl = textbuf.NewExtmarksController(ta.editBuffer(), ta.editSurface)
	}
	return ta.extmarksCtrl
}

// SetInitialValue sets the text only once (subsequent calls are ignored).
// Useful for reactive frameworks where props re-apply.
func (ta *Textarea) SetInitialValue(text string) {
	if !ta.initialValueSet {
		ta.SetText(text)
		ta.initialValueSet = true
	}
}

// ---------------------------------------------------------------------------
// Focus / Blur
// ---------------------------------------------------------------------------

// Focus overrides BaseRenderable.Focus to swap active colors and set cursor style.
func (ta *Textarea) Focus() {
	ta.BaseRenderable.Focus()
	ta.textColor = ta.focusedTextColor
	ta.backgroundColor = ta.focusedBgColor
	ta.applyCursorStyle()
	ta.RequestRender()
}

// Blur overrides BaseRenderable.Blur to swap active colors and hide cursor.
func (ta *Textarea) Blur() {
	ta.BaseRenderable.Blur()
	ta.textColor = ta.unfocusedTextColor
	ta.backgroundColor = ta.unfocusedBgColor
	ta.Context().SetCursorPosition(0, 0, false)
	ta.RequestRender()
}

// ---------------------------------------------------------------------------
// OnResize
// ---------------------------------------------------------------------------

// OnResize is called when the widget dimensions change.
func (ta *Textarea) OnResize(width, height int) {
	ta.editSurface.SetViewportSize(width, height)
}

// ---------------------------------------------------------------------------
// RenderSelf
// ---------------------------------------------------------------------------

// RenderSelf draws the textarea into the buffer.
func (ta *Textarea) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := ta.Width()
	h := ta.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := ta.X()
	y := ta.Y()

	// Update viewport size to match widget dimensions
	ta.editSurface.SetViewportSize(w, h)

	// Apply text color as default fg
	ta.textBuffer().SetDefaultFg(ta.textColor)

	// Register in hit grid for mouse dispatch
	ta.Context().AddToHitGrid(x, y, w, h, ta.Num())

	// Draw background
	if !ta.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, ta.backgroundColor)
	}

	// Draw placeholder if empty
	if ta.editSurface.IsEmpty() {
		phText, phColor := ta.editSurface.PlaceholderText()
		if phText != "" {
			drawText := truncateToWidth(phText, w)
			buf.DrawText(drawText, x, y, phColor)
		}
		// If focused, still show cursor at position 0
		if ta.Focused() && ta.showCursor {
			ta.Context().SetCursorPosition(x, y, true)
			ta.applyCursorStyle()
		}
		return
	}

	// Draw text content via EditSurface → Surface pipeline
	buf.DrawEditSurface(ta.editSurface, x, y)

	// Position cursor
	if ta.Focused() && ta.showCursor {
		vc := ta.editSurface.VisualCursor()
		cursorScreenX := x + vc.VisualCol
		cursorScreenY := y + vc.VisualRow
		// Allow the cursor to sit one column past the content (at the right
		// border) when the cursor is at the end of a full-width line. This
		// keeps the cursor visually "after the last character" instead of
		// on top of it.
		if cursorScreenX >= x && cursorScreenX <= x+w &&
			cursorScreenY >= y && cursorScreenY < y+h {
			ta.Context().SetCursorPosition(cursorScreenX, cursorScreenY, true)
			ta.applyCursorStyle()
		}
	}
}

// ---------------------------------------------------------------------------
// HandleKeyPress
// ---------------------------------------------------------------------------

// HandleKeyPress processes keyboard input.
func (ta *Textarea) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease {
		return false
	}

	// Printable rune — not matched by binding table
	if e.IsRune() {
		if ta.readOnly {
			return false
		}
		ta.deleteSelectedText()
		ta.editBuffer().InsertChar(e.Rune())
		ta.emitContentChange()
		e.StopPropagation()
		ta.RequestRender()
		return true
	}

	// Resolve action from keybindings (lazy rebuild)
	action, found := ta.resolveAction(e)
	if !found {
		return false
	}

	// Read-only: block non-navigation actions (navActions whitelist)
	if ta.readOnly && !navActions[action] {
		return false
	}

	// Dispatch via handler map — O(1) lookup, extensible
	if handler, ok := ta.actionHandlers[action]; ok {
		handled := handler()
		if handled {
			e.StopPropagation()
		}
		return handled
	}

	return false
}

// HandlePaste handles paste events.
func (ta *Textarea) HandlePaste(e *input.PasteEvent) {
	if ta.readOnly {
		return
	}
	ta.deleteSelectedText()
	ta.editBuffer().InsertText(e.Text)
	ta.emitContentChange()
	ta.RequestRender()
}

// ---------------------------------------------------------------------------
// Mouse handling
// ---------------------------------------------------------------------------

// ProcessMouseEvent overrides BaseRenderable to handle scroll and selection.
func (ta *Textarea) ProcessMouseEvent(e *input.MouseEvent) {
	switch e.Type {
	case input.MouseScroll:
		ta.handleMouseScroll(e)
	case input.MouseDown:
		ta.handleMouseDown(e)
	case input.MouseDrag:
		ta.handleMouseDrag(e)
	case input.MouseDragEnd:
		ta.handleMouseDragEnd(e)
	case input.MouseUp:
		ta.handleMouseUp(e)
	}

	// Still call base for bubbling and other handlers
	ta.BaseRenderable.ProcessMouseEvent(e)
}

func (ta *Textarea) handleMouseScroll(e *input.MouseEvent) {
	if e.Scroll == nil {
		return
	}

	vp := ta.editSurface.TextSurface().Viewport()
	if vp == nil {
		return
	}

	delta := e.Scroll.DeltaY
	if delta != 0 {
		totalVLines := ta.editSurface.TotalVirtualLineCount()
		maxY := max(totalVLines-vp.Height, 0)

		newY := max(vp.Y+delta, 0)
		if newY > maxY {
			newY = maxY
		}
		if newY != vp.Y {
			vp.Y = newY
			ta.RequestRender()
		}
	}

	// Horizontal scroll (for no-wrap mode)
	deltaX := e.Scroll.DeltaX
	if deltaX != 0 {
		newX := max(vp.X+deltaX, 0)
		if newX != vp.X {
			vp.X = newX
			ta.RequestRender()
		}
	}

	e.StopPropagation()
}

func (ta *Textarea) handleMouseDown(e *input.MouseEvent) {
	if !ta.Selectable() {
		return
	}

	localX := e.X - ta.X()
	localY := e.Y - ta.Y()

	// Bounds check
	if localX < 0 || localX >= ta.Width() || localY < 0 || localY >= ta.Height() {
		return
	}

	// Move cursor to click position
	ta.editSurface.SetLocalSelection(localX, localY, localX, localY,
		&ta.selectionBg, ta.selectionFg, true, false)

	// Start mouse selection
	ta.mouseSelAnchorX = localX
	ta.mouseSelAnchorY = localY
	ta.mouseSelecting = true

	// Clear any keyboard selection and start mouse selection in context
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.editSurface.ResetSelection()
	ta.Context().StartSelection(ta, e.X, e.Y)

	e.StopPropagation()
}

func (ta *Textarea) handleMouseDrag(e *input.MouseEvent) {
	if !ta.mouseSelecting {
		return
	}

	localX := e.X - ta.X()
	localY := e.Y - ta.Y()

	// Update selection from anchor to current drag position
	ok := ta.editSurface.UpdateLocalSelection(
		ta.mouseSelAnchorX, ta.mouseSelAnchorY,
		localX, localY,
		&ta.selectionBg, ta.selectionFg,
	)

	// Auto-scroll when dragging near edges
	vp := ta.editSurface.TextSurface().Viewport()
	if vp != nil {
		scrollMargin := int(math.Max(1, float64(vp.Height)*0.2))
		if localY < scrollMargin {
			ta.autoScrollVelocity = -ta.scrollSpeed
		} else if localY >= vp.Height-scrollMargin {
			ta.autoScrollVelocity = ta.scrollSpeed
		} else {
			ta.autoScrollVelocity = 0
		}
	}

	if ok {
		// Update context-level selection to track the mouse drag
		ta.Context().UpdateSelection(ta, e.X, e.Y)
		ta.RequestRender()
	}

	e.StopPropagation()
}

func (ta *Textarea) handleMouseDragEnd(e *input.MouseEvent) {
	if ta.mouseSelecting {
		// Finalize the context selection
		ta.Context().UpdateSelection(ta, e.X, e.Y,
			render.SelectionUpdateOption{FinishDragging: true})
	}
	ta.mouseSelecting = false
	ta.autoScrollVelocity = 0
	ta.autoScrollAccumulator = 0
}

func (ta *Textarea) handleMouseUp(e *input.MouseEvent) {
	ta.mouseSelecting = false
	ta.autoScrollVelocity = 0
	ta.autoScrollAccumulator = 0
}

// OnPostLayout handles auto-scroll during mouse drag selection.
// Runs after layout as part of the sync pass — only for visible widgets.
func (ta *Textarea) OnPostLayout(deltaTime float64) {
	if ta.autoScrollVelocity == 0 || !ta.mouseSelecting {
		return
	}

	deltaSec := deltaTime / 1000.0
	ta.autoScrollAccumulator += ta.autoScrollVelocity * deltaSec

	linesToScroll := int(math.Abs(ta.autoScrollAccumulator))
	if linesToScroll > 0 {
		direction := 1
		if ta.autoScrollVelocity < 0 {
			direction = -1
		}

		vp := ta.editSurface.TextSurface().Viewport()
		if vp != nil {
			totalVLines := ta.editSurface.TotalVirtualLineCount()
			maxY := max(totalVLines-vp.Height, 0)
			newY := max(vp.Y+direction*linesToScroll, 0)
			if newY > maxY {
				newY = maxY
			}
			if newY != vp.Y {
				vp.Y = newY
				ta.RequestRender()
			}
		}

		ta.autoScrollAccumulator -= float64(direction * linesToScroll)
	}
}

// ShouldStartSelection returns true if a mouse selection should start at (x, y).
func (ta *Textarea) ShouldStartSelection(x, y int) bool {
	if !ta.Selectable() {
		return false
	}
	localX := x - ta.X()
	localY := y - ta.Y()
	return localX >= 0 && localX < ta.Width() && localY >= 0 && localY < ta.Height()
}

// ---------------------------------------------------------------------------
// Destroy
// ---------------------------------------------------------------------------

// Destroy cleans up the textarea and its resources.
func (ta *Textarea) Destroy() {
	if ta.extmarksCtrl != nil {
		ta.extmarksCtrl.Destroy()
		ta.extmarksCtrl = nil
	}
	if ta.Focused() {
		ta.Context().SetCursorPosition(0, 0, false)
	}
	ta.BaseRenderable.Destroy()
}

// ---------------------------------------------------------------------------
// Handler map (action → func() bool)
// ---------------------------------------------------------------------------

// buildActionHandlers constructs the map from action name → handler.
func (ta *Textarea) buildActionHandlers() map[string]func() bool {
	return map[string]func() bool{
		// Navigation
		ActionMoveLeft:      ta.actionMoveLeft,
		ActionMoveRight:     ta.actionMoveRight,
		ActionMoveUp:        ta.actionMoveUp,
		ActionMoveDown:      ta.actionMoveDown,
		ActionMoveWordLeft:  ta.actionMoveWordLeft,
		ActionMoveWordRight: ta.actionMoveWordRight,
		ActionMoveHome:      ta.actionMoveHome,
		ActionMoveEnd:       ta.actionMoveEnd,
		ActionMoveDocStart:  ta.actionMoveDocStart,
		ActionMoveDocEnd:    ta.actionMoveDocEnd,
		ActionPageUp:        ta.actionPageUp,
		ActionPageDown:      ta.actionPageDown,
		ActionVisualHome:    ta.actionVisualHome,
		ActionVisualEnd:     ta.actionVisualEnd,

		// Selection
		ActionSelectLeft:       ta.actionSelectLeft,
		ActionSelectRight:      ta.actionSelectRight,
		ActionSelectUp:         ta.actionSelectUp,
		ActionSelectDown:       ta.actionSelectDown,
		ActionSelectWordLeft:   ta.actionSelectWordLeft,
		ActionSelectWordRight:  ta.actionSelectWordRight,
		ActionSelectHome:       ta.actionSelectHome,
		ActionSelectEnd:        ta.actionSelectEnd,
		ActionSelectDocStart:   ta.actionSelectDocStart,
		ActionSelectDocEnd:     ta.actionSelectDocEnd,
		ActionSelectAll:        ta.actionSelectAll,
		ActionSelectVisualHome: ta.actionSelectVisualHome,
		ActionSelectVisualEnd:  ta.actionSelectVisualEnd,

		// Editing
		ActionBackspace:     ta.actionBackspace,
		ActionDelete:        ta.actionDelete,
		ActionDeleteLine:    ta.actionDeleteLine,
		ActionDeleteToEnd:   ta.actionDeleteToEnd,
		ActionDeleteToStart: ta.actionDeleteToStart,
		ActionDeleteWordFwd: ta.actionDeleteWordFwd,
		ActionDeleteWordBwd: ta.actionDeleteWordBwd,
		ActionNewline:       ta.actionNewline,
		ActionTab:           ta.actionTab,
		ActionUndo:          ta.actionUndo,
		ActionRedo:          ta.actionRedo,
		ActionSubmit:        ta.actionSubmit,
	}
}

// --- Navigation actions ---

func (ta *Textarea) actionMoveLeft() bool {
	// Collapse selection to start
	if ta.HasSelection() {
		sel := ta.editSurface.Selection()
		if sel != nil {
			ta.editSurface.SetCursorByOffset(sel.Start)
		}
		ta.clearSelection()
		ta.RequestRender()
		return true
	}
	ta.editBuffer().MoveCursorLeft()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveRight() bool {
	// Collapse selection to end
	if ta.HasSelection() {
		sel := ta.editSurface.Selection()
		if sel != nil {
			ta.editSurface.SetCursorByOffset(sel.End)
		}
		ta.clearSelection()
		ta.RequestRender()
		return true
	}
	ta.editBuffer().MoveCursorRight()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveUp() bool {
	ta.clearSelection()
	ta.editSurface.MoveUpVisual()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveDown() bool {
	ta.clearSelection()
	ta.editSurface.MoveDownVisual()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveWordLeft() bool {
	ta.clearSelection()
	pos := ta.editBuffer().PrevWordBoundary()
	ta.editBuffer().SetCursor(pos.Row, pos.Col)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveWordRight() bool {
	ta.clearSelection()
	pos := ta.editBuffer().NextWordBoundary()
	ta.editBuffer().SetCursor(pos.Row, pos.Col)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveHome() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	row := eb.CursorRow()
	col := eb.CursorCol()
	// Smart home: if already at col 0 and there's a previous line, jump to its end
	if col == 0 && row > 0 {
		prevLineW := ta.textBuffer().LineWidthAt(row - 1)
		eb.SetCursor(row-1, prevLineW)
	} else {
		eb.SetCursor(row, 0)
	}
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveEnd() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	eol := eb.EOL()
	row := eb.CursorRow()
	col := eb.CursorCol()
	// Smart end: if already at EOL and there's a next line, jump to its start
	if col == eol.Col && row < eb.LineCount()-1 {
		eb.SetCursor(row+1, 0)
	} else {
		eb.SetCursor(eol.Row, eol.Col)
	}
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveDocStart() bool {
	ta.clearSelection()
	ta.editBuffer().SetCursor(0, 0)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionMoveDocEnd() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	lastRow := eb.LineCount() - 1
	lastCol := ta.textBuffer().LineWidthAt(lastRow)
	eb.SetCursor(lastRow, lastCol)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionPageUp() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	newRow := max(eb.CursorRow()-ta.Height(), 0)
	eb.SetCursor(newRow, eb.CursorCol())
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionPageDown() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	newRow := eb.CursorRow() + ta.Height()
	maxRow := eb.LineCount() - 1
	if newRow > maxRow {
		newRow = maxRow
	}
	eb.SetCursor(newRow, eb.CursorCol())
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionVisualHome() bool {
	ta.clearSelection()
	sol := ta.editSurface.VisualSOL()
	ta.editBuffer().SetCursor(sol.LogicalRow, sol.LogicalCol)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionVisualEnd() bool {
	ta.clearSelection()
	eol := ta.editSurface.VisualEOL()
	ta.editBuffer().SetCursor(eol.LogicalRow, eol.LogicalCol)
	ta.RequestRender()
	return true
}

// --- Selection actions ---
//
// Each follows the before/after pattern:
//   updateSelectionForMovement(true, true)   // before: record anchor
//   <move cursor>
//   updateSelectionForMovement(true, false)   // after: update focus
//   RequestRender()

func (ta *Textarea) actionSelectLeft() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editBuffer().MoveCursorLeft()
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectRight() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editBuffer().MoveCursorRight()
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectUp() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editSurface.MoveUpVisual()
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectDown() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editSurface.MoveDownVisual()
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectWordLeft() bool {
	ta.updateSelectionForMovement(true, true)
	pos := ta.editBuffer().PrevWordBoundary()
	ta.editBuffer().SetCursor(pos.Row, pos.Col)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectWordRight() bool {
	ta.updateSelectionForMovement(true, true)
	pos := ta.editBuffer().NextWordBoundary()
	ta.editBuffer().SetCursor(pos.Row, pos.Col)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectHome() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editBuffer().SetCursor(ta.editBuffer().CursorRow(), 0)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectEnd() bool {
	ta.updateSelectionForMovement(true, true)
	eol := ta.editBuffer().EOL()
	ta.editBuffer().SetCursor(eol.Row, eol.Col)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectDocStart() bool {
	ta.updateSelectionForMovement(true, true)
	ta.editBuffer().SetCursor(0, 0)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectDocEnd() bool {
	ta.updateSelectionForMovement(true, true)
	eb := ta.editBuffer()
	lastRow := eb.LineCount() - 1
	lastCol := ta.textBuffer().LineWidthAt(lastRow)
	eb.SetCursor(lastRow, lastCol)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectAll() bool {
	ta.SelectAll()
	return true
}

func (ta *Textarea) actionSelectVisualHome() bool {
	ta.updateSelectionForMovement(true, true)
	sol := ta.editSurface.VisualSOL()
	ta.editBuffer().SetCursor(sol.LogicalRow, sol.LogicalCol)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionSelectVisualEnd() bool {
	ta.updateSelectionForMovement(true, true)
	eol := ta.editSurface.VisualEOL()
	ta.editBuffer().SetCursor(eol.LogicalRow, eol.LogicalCol)
	ta.updateSelectionForMovement(true, false)
	ta.RequestRender()
	return true
}

// --- Editing actions ---

func (ta *Textarea) actionBackspace() bool {
	if !ta.deleteSelectedText() {
		ta.editBuffer().DeleteCharBackward()
	}
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDelete() bool {
	if !ta.deleteSelectedText() {
		ta.editBuffer().DeleteChar()
	}
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDeleteLine() bool {
	ta.clearSelection()
	ta.editBuffer().DeleteLine()
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDeleteToEnd() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	eol := eb.EOL()
	if eb.CursorCol() < eol.Col {
		eb.DeleteRange(eb.CursorRow(), eb.CursorCol(), eol.Row, eol.Col)
	}
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDeleteToStart() bool {
	ta.clearSelection()
	eb := ta.editBuffer()
	row := eb.CursorRow()
	if eb.CursorCol() > 0 {
		eb.DeleteRange(row, 0, row, eb.CursorCol())
	}
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDeleteWordFwd() bool {
	if ta.HasSelection() {
		ta.deleteSelectedText()
	} else {
		eb := ta.editBuffer()
		pos := eb.NextWordBoundary()
		eb.DeleteRange(eb.CursorRow(), eb.CursorCol(), pos.Row, pos.Col)
	}
	ta.clearSelection()
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionDeleteWordBwd() bool {
	if ta.HasSelection() {
		ta.deleteSelectedText()
	} else {
		eb := ta.editBuffer()
		pos := eb.PrevWordBoundary()
		eb.DeleteRange(pos.Row, pos.Col, eb.CursorRow(), eb.CursorCol())
	}
	ta.clearSelection()
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionNewline() bool {
	ta.deleteSelectedText()
	ta.editBuffer().NewLine()
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionTab() bool {
	ta.deleteSelectedText()
	ta.editBuffer().InsertChar('\t')
	ta.emitContentChange()
	ta.RequestRender()
	return true
}

func (ta *Textarea) actionUndo() bool {
	ta.Undo()
	return true
}

func (ta *Textarea) actionRedo() bool {
	ta.Redo()
	return true
}

func (ta *Textarea) actionSubmit() bool {
	if ta.OnSubmit != nil {
		ta.OnSubmit(ta.Text())
	}
	return true
}

// ---------------------------------------------------------------------------
// Selection helpers
// ---------------------------------------------------------------------------

// HasSelection returns true if there is an active text selection.
// Overrides BaseRenderable.HasSelection.
func (ta *Textarea) HasSelection() bool {
	return ta.editSurface.HasSelection()
}

// updateSelectionForMovement manages selection state around cursor movement.
// Call with isBeforeMovement=true BEFORE moving the cursor, then again with
// isBeforeMovement=false AFTER. When shiftPressed is false the selection is
// cleared.
func (ta *Textarea) updateSelectionForMovement(shiftPressed, isBeforeMovement bool) {
	if !ta.Selectable() {
		return
	}

	if !shiftPressed {
		ta.keyboardSelectionActive = false
		ta.clearSelection()
		return
	}

	ta.keyboardSelectionActive = true

	if isBeforeMovement {
		// Before the cursor moves: if no selection yet, record the anchor
		if !ta.HasSelection() {
			vc := ta.editSurface.VisualCursor()
			ta.Context().StartSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow)
			// Also record offset-based anchor for EditSurface selection
			eb := ta.editBuffer()
			ta.selAnchorOffset = eb.PositionToOffset(eb.CursorRow(), eb.CursorCol())
		}
		return
	}

	// After the cursor moves: update focus position
	vc := ta.editSurface.VisualCursor()
	ta.Context().UpdateSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow,
		render.SelectionUpdateOption{FinishDragging: true})

	// Update the EditSurface's internal selection for rendering
	ta.syncSelectionToEditBufferView()
}

// syncSelectionToEditBufferView computes the offset-based selection from the
// anchor and current cursor, then applies it to the EditSurface for rendering.
func (ta *Textarea) syncSelectionToEditBufferView() {
	eb := ta.editBuffer()
	focusOffset := eb.PositionToOffset(eb.CursorRow(), eb.CursorCol())

	start, end := ta.selAnchorOffset, focusOffset
	if start > end {
		start, end = end, start
	}

	sel := &textbuf.TextSelection{
		Start:   start,
		End:     end,
		BgColor: &ta.selectionBg,
		FgColor: ta.selectionFg,
	}
	ta.editSurface.SetSelection(sel)
}

// clearSelection clears both the context-level and EditSurface selection.
func (ta *Textarea) clearSelection() {
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.editSurface.ResetSelection()
	ta.Context().ClearSelection()
}

// deleteSelectedText deletes selected text. Returns true if there was a selection.
func (ta *Textarea) deleteSelectedText() bool {
	if !ta.HasSelection() {
		return false
	}
	ta.editSurface.DeleteSelectedText()
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.Context().ClearSelection()
	return true
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (ta *Textarea) editBuffer() *textbuf.EditBuffer {
	return ta.editSurface.EditBuffer()
}

func (ta *Textarea) textBuffer() *textbuf.TextBuffer {
	return ta.editSurface.EditBuffer().TextBuffer()
}

func (ta *Textarea) syncViewport() {
	w := ta.Width()
	h := ta.Height()
	if w > 0 && h > 0 {
		ta.editSurface.SetViewportSize(w, h)
	}
}

func (ta *Textarea) emitContentChange() {
	ta.LayoutNode().MarkDirty()
	if ta.OnContentChange != nil {
		ta.OnContentChange(ta.Text())
	}
}

func (ta *Textarea) resolveAction(e *input.KeyEvent) (string, bool) {
	if ta.bindingsDirty {
		ta.rebuildBindingMap()
	}
	return input.ResolveAction(*e, ta.bindingMap)
}

func (ta *Textarea) rebuildBindingMap() {
	defaults := ta.defaultBindings()
	merged := input.MergeKeyBindings(defaults, ta.keyBindings)
	aliases := input.MergeKeyAliases(input.DefaultAliases, ta.keyAliases)
	ta.bindingMap = input.BuildKeyBindingsMap(merged, aliases)
	ta.bindingsDirty = false
}

// defaultBindings returns the default keybinding table for this widget.
func (ta *Textarea) defaultBindings() []input.KeyBinding {
	return DefaultTextareaBindings
}

// applyCursorStyle is a no-op retained for call-site compatibility.
// Cursor style is applied by the Renderer during the render pass.
func (ta *Textarea) applyCursorStyle() {}

// ---------------------------------------------------------------------------
// Textarea Options
// ---------------------------------------------------------------------------

// TextareaWithText sets the initial text.
func TextareaWithText(text string) TextareaOption {
	return func(ta *Textarea) {
		ta.editBuffer().SetText(text)
	}
}

// TextareaWithInitialValue sets the initial text (one-shot: subsequent calls ignored).
func TextareaWithInitialValue(text string) TextareaOption {
	return func(ta *Textarea) {
		ta.editBuffer().SetText(text)
		ta.initialValueSet = true
	}
}

// TextareaWithPlaceholder sets the placeholder text.
func TextareaWithPlaceholder(text string) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetPlaceholderText(text, style.ColorGray)
	}
}

// TextareaWithPlaceholderColor sets the placeholder color.
func TextareaWithPlaceholderColor(fg style.Color) TextareaOption {
	return func(ta *Textarea) {
		phText, _ := ta.editSurface.PlaceholderText()
		ta.editSurface.SetPlaceholderText(phText, fg)
	}
}

// TextareaWithTextColor sets the text color (both focused and unfocused).
func TextareaWithTextColor(c style.Color) TextareaOption {
	return func(ta *Textarea) {
		ta.unfocusedTextColor = c
		ta.focusedTextColor = c
		ta.textColor = c
	}
}

// TextareaWithFocusedTextColor sets the text color used when focused.
func TextareaWithFocusedTextColor(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.focusedTextColor = c }
}

// TextareaWithBackgroundColor sets the background color (both focused and unfocused).
func TextareaWithBackgroundColor(c style.Color) TextareaOption {
	return func(ta *Textarea) {
		ta.unfocusedBgColor = c
		ta.focusedBgColor = c
		ta.backgroundColor = c
	}
}

// TextareaWithFocusedBackgroundColor sets the background color when focused.
func TextareaWithFocusedBackgroundColor(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.focusedBgColor = c }
}

// TextareaWithReadOnly makes the textarea read-only.
func TextareaWithReadOnly(v bool) TextareaOption {
	return func(ta *Textarea) { ta.readOnly = v }
}

// TextareaWithWrapMode sets the wrap mode.
func TextareaWithWrapMode(m WrapMode) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetWrapMode(m)
	}
}

// TextareaWithScrollMargin sets the scroll margin fraction.
func TextareaWithScrollMargin(m float32) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetScrollMargin(m)
	}
}

// TextareaWithShowCursor enables or disables cursor rendering.
func TextareaWithShowCursor(v bool) TextareaOption {
	return func(ta *Textarea) { ta.showCursor = v }
}

// TextareaWithCursorColor sets the cursor color.
func TextareaWithCursorColor(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.cursorColor = c }
}

// TextareaWithCursorStyle sets the cursor shape (block/line/underline) and blinking.
func TextareaWithCursorStyle(shape string, blinking bool) TextareaOption {
	return func(ta *Textarea) {
		ta.cursorShape = shape
		ta.cursorBlinking = blinking
	}
}

// TextareaWithMaxUndo sets the maximum undo history size.
func TextareaWithMaxUndo(n int) TextareaOption {
	return func(ta *Textarea) {
		ta.editBuffer().SetMaxUndo(n)
	}
}

// TextareaWithKeyBindings sets custom keybinding overrides.
func TextareaWithKeyBindings(bindings []input.KeyBinding) TextareaOption {
	return func(ta *Textarea) {
		ta.keyBindings = bindings
		ta.bindingsDirty = true
	}
}

// TextareaWithKeyAliases sets custom key alias overrides.
func TextareaWithKeyAliases(aliases input.KeyAliasMap) TextareaOption {
	return func(ta *Textarea) {
		ta.keyAliases = aliases
		ta.bindingsDirty = true
	}
}

// TextareaWithSelectionBg sets the selection background color.
func TextareaWithSelectionBg(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.selectionBg = c }
}

// TextareaWithSelectionFg sets the selection foreground color.
func TextareaWithSelectionFg(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.selectionFg = &c }
}

// TextareaWithOnCursorChange sets the cursor change callback.
func TextareaWithOnCursorChange(fn func(row, col int)) TextareaOption {
	return func(ta *Textarea) { ta.OnCursorChange = fn }
}

// TextareaWithOnContentChange sets the content change callback.
func TextareaWithOnContentChange(fn func(text string)) TextareaOption {
	return func(ta *Textarea) { ta.OnContentChange = fn }
}

// TextareaWithOnSubmit sets the submit callback.
func TextareaWithOnSubmit(fn func(text string)) TextareaOption {
	return func(ta *Textarea) { ta.OnSubmit = fn }
}

// TextareaWithID sets the widget ID.
func TextareaWithID(id string) TextareaOption {
	return func(ta *Textarea) { ta.SetID(id) }
}

// TextareaWithWidth sets the widget width.
func TextareaWithWidth(s layout.Scalar) TextareaOption {
	return func(ta *Textarea) { ta.LayoutNode().Width = s }
}

// TextareaWithHeight sets the widget height.
func TextareaWithHeight(s layout.Scalar) TextareaOption {
	return func(ta *Textarea) { ta.LayoutNode().Height = s }
}

// TextareaWithFlexGrow sets flex grow.
func TextareaWithFlexGrow(v int) TextareaOption {
	return func(ta *Textarea) { ta.LayoutNode().FlexGrow = v }
}

// TextareaWithMinHeight sets the minimum height constraint.
func TextareaWithMinHeight(s layout.Scalar) TextareaOption {
	return func(ta *Textarea) { ta.LayoutNode().MinHeight = s }
}

// TextareaWithMaxHeight sets the maximum height constraint.
func TextareaWithMaxHeight(s layout.Scalar) TextareaOption {
	return func(ta *Textarea) { ta.LayoutNode().MaxHeight = s }
}

// TextareaWithTabIndicator sets the tab indicator character.
// TextareaWithDefaultAttributes sets the default text attributes (bold, italic, etc.).
func TextareaWithDefaultAttributes(attrs uint32) TextareaOption {
	return func(ta *Textarea) { ta.defaultAttributes = attrs }
}

// TextareaWithSyntaxStyle attaches a SyntaxStyle registry.
func TextareaWithSyntaxStyle(ss *style.SyntaxStyle) TextareaOption {
	return func(ta *Textarea) {
		ta.textBuffer().SetSyntaxStyle(ss)
	}
}

// TextareaWithSelectable sets the selectable flag (for mouse selection).
func TextareaWithSelectable(v bool) TextareaOption {
	return func(ta *Textarea) { ta.SetSelectable(v) }
}

// TextareaWithScrollSpeed sets the auto-scroll speed during mouse drag.
func TextareaWithScrollSpeed(speed float64) TextareaOption {
	return func(ta *Textarea) { ta.scrollSpeed = speed }
}
