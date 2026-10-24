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

// Textarea is a multiline editor with wrapping, selection, undo, mouse input,
// syntax highlighting, and configurable keybindings.
//
// EditBuffer owns text, cursor, and undo state. EditSurface owns wrapping,
// viewport, visual cursor, and selection state.
type Textarea struct {
	*render.BaseRenderable

	editSurface *textbuf.EditSurface

	// Focus selects between configured text and background colors.
	unfocusedTextColor style.Color
	unfocusedBgColor   style.Color
	focusedTextColor   style.Color
	focusedBgColor     style.Color
	textColor          style.Color // Active foreground selected by Focus or Blur.
	backgroundColor    style.Color // Active background selected by Focus or Blur.
	cursorColor        style.Color
	selectionBg        style.Color
	selectionFg        style.Color // Transparent preserves the text foreground.

	showCursor bool
	readOnly   bool

	// Custom bindings and aliases are compiled only after they change.
	keyBindings   []input.KeyBinding
	keyAliases    input.KeyAliasMap
	bindingMap    map[string]string
	bindingsDirty bool

	// actionHandlers dispatches resolved action names.
	actionHandlers map[string]func() bool

	// Selection uses the render.Host for screen coordinates and an offset anchor
	// for EditSurface text coordinates.
	selAnchorOffset         int
	keyboardSelectionActive bool // True while a movement action extends selection.

	// defaultAttributes applies to text without an explicit highlight style.
	defaultAttributes style.Attrs

	// Drag selection accumulates fractional automatic scrolling over time.
	autoScrollVelocity    float64
	autoScrollAccumulator float64
	scrollSpeed           float64

	// Mouse selection coordinates are local to the viewport.
	mouseSelAnchorX int
	mouseSelAnchorY int
	mouseSelecting  bool

	// extmarksCtrl is created only when advanced extmark access is requested.
	extmarksCtrl *textbuf.ExtmarksController

	// OnCursorChange is called after the logical cursor position changes.
	OnCursorChange func(row, col int)
	// OnContentChange is called after an undoable content edit.
	OnContentChange func(text string)
	// OnSubmit is called when the configured submit action runs.
	OnSubmit func(text string)
}

// TextareaOption configures a Textarea during construction.
type TextareaOption func(*Textarea)

// NewTextarea returns an empty, focusable Textarea with optional configuration.
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
		bindingsDirty:      true,
		scrollSpeed:        16,
	}

	// Notify EditSurface before the public callback so visual cursor state remains current.
	eb.OnCursorChange = func(row, col int) {
		ev.NotifyCursorChange()
		if ta.OnCursorChange != nil {
			ta.OnCursorChange(row, col)
		}
	}

	for _, opt := range opts {
		opt(ta)
	}

	// Apply configured text defaults after all options run.
	tb.SetDefaultFg(ta.textColor)
	tb.SetExpandTabs(true)
	if ta.defaultAttributes != 0 {
		tb.SetDefaultAttributes(ta.defaultAttributes)
	}

	// Action handlers are stable for the lifetime of the widget.
	ta.actionHandlers = ta.buildActionHandlers()

	// Measure with the same EditSurface used for rendering.
	ta.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		availW := width
		if widthMode == layout.MeasureUndefined {
			availW = 0 // An unconstrained width measures unwrapped content.
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

// Text content.

// SetText replaces all text content and clears undo history.
func (ta *Textarea) SetText(text string) {
	ta.editBuffer().SetText(text)
	ta.clearSelection()
	ta.syncViewport()
	ta.LayoutNode().MarkDirty()
	ta.RequestRender()
}

// ReplaceText replaces all text as one undoable edit.
func (ta *Textarea) ReplaceText(text string) {
	ta.editBuffer().ReplaceText(text)
	ta.clearSelection()
	ta.syncViewport()
	ta.emitContentChange()
	ta.RequestRender()
}

// Clear removes all text as an undoable edit.
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

// Cursor movement.

// SetCursor sets a logical row and display-column position.
func (ta *Textarea) SetCursor(row, col int) {
	ta.editBuffer().SetCursor(row, col)
	ta.syncViewport()
	ta.RequestRender()
}

// A LineNumber beside a Textarea styles the cursor's line as active.
var _ CursorTracker = (*Textarea)(nil)

// CursorPosition returns the logical row and display-column position.
func (ta *Textarea) CursorPosition() (row, col int) {
	return ta.editBuffer().CursorRow(), ta.editBuffer().CursorCol()
}

// SetCursorByOffset sets the cursor from a buffer-wide display-column offset.
func (ta *Textarea) SetCursorByOffset(offset int) {
	ta.editSurface.SetCursorByOffset(offset)
	ta.RequestRender()
}

// CursorOffset returns the cursor as a buffer-wide display-column offset.
func (ta *Textarea) CursorOffset() int {
	eb := ta.editBuffer()
	return eb.PositionToOffset(eb.CursorRow(), eb.CursorCol())
}

// GotoLine moves the cursor to the start of a logical line.
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

// GotoLineEnd moves the cursor to the end of the current logical line.
func (ta *Textarea) GotoLineEnd() {
	eol := ta.editBuffer().EOL()
	ta.editBuffer().SetCursor(eol.Row, eol.Col)
	ta.RequestRender()
}

// Selection.

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
		BgColor: ta.selectionBg,
		FgColor: ta.selectionFg,
	}
	ta.editSurface.SetSelection(sel)

	// Start a corresponding screen-coordinate selection in the host.
	host := ta.Host()
	host.StartSelection(ta, ta.X(), ta.Y())

	// Finish the selection with the cursor at the end of the buffer.
	lastRow := eb.LineCount() - 1
	lastCol := ta.textBuffer().LineWidthAt(lastRow)
	eb.SetCursor(lastRow, lastCol)

	// Synchronize the host's selection focus with the final visual cursor.
	vc := ta.editSurface.VisualCursor()
	host.UpdateSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow,
		render.SelectionUpdateOption{FinishDragging: true})

	ta.RequestRender()
}

// Undo and redo.

// Undo reverses the last edit and reports whether an edit was available.
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

// Redo reapplies the last undone edit and reports whether an edit was available.
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

// Scrolling.

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

// ScrollY returns the current vertical viewport offset.
func (ta *Textarea) ScrollY() int {
	vp := ta.editSurface.TextSurface().Viewport()
	if vp == nil {
		return 0
	}
	return vp.Y
}

// Wrapped-line information.

// VirtualLineCount returns the number of wrapped lines visible in the viewport.
func (ta *Textarea) VirtualLineCount() int {
	return ta.editSurface.VirtualLineCount()
}

// TotalVirtualLineCount returns the total number of wrapped lines.
func (ta *Textarea) TotalVirtualLineCount() int {
	return ta.editSurface.TotalVirtualLineCount()
}

// Text ranges.

// TextRange returns text between buffer-wide display-column offsets.
func (ta *Textarea) TextRange(startOffset, endOffset int) string {
	return ta.editBuffer().TextRange(startOffset, endOffset)
}

// TextRangeByCoords returns text between two logical row and display-column positions.
func (ta *Textarea) TextRangeByCoords(startRow, startCol, endRow, endCol int) string {
	return ta.editBuffer().TextRangeByCoords(startRow, startCol, endRow, endCol)
}

// LineInfo returns logical-line layout metadata.
func (ta *Textarea) LineInfo() textbuf.LineInfo {
	return ta.editSurface.LogicalLineInfo()
}

// Syntax highlighting.

// AddHighlight adds hl to a zero-based logical line.
func (ta *Textarea) AddHighlight(lineIdx int, hl textbuf.Highlight) {
	ta.textBuffer().AddHighlight(lineIdx, hl)
	ta.RequestRender()
}

// AddHighlightByCharRange adds a highlight whose range uses buffer-wide offsets.
func (ta *Textarea) AddHighlightByCharRange(hl textbuf.Highlight) {
	ta.textBuffer().AddHighlightByCharRange(hl)
	ta.RequestRender()
}

// RemoveHighlightsByRef removes every highlight with ref.
func (ta *Textarea) RemoveHighlightsByRef(ref int) {
	ta.textBuffer().RemoveHighlightsByRef(ref)
	ta.RequestRender()
}

// ClearLineHighlights removes highlights from a zero-based logical line.
func (ta *Textarea) ClearLineHighlights(lineIdx int) {
	ta.textBuffer().ClearLineHighlights(lineIdx)
	ta.RequestRender()
}

// ClearAllHighlights removes all highlights.
func (ta *Textarea) ClearAllHighlights() {
	ta.textBuffer().ClearAllHighlights()
	ta.RequestRender()
}

// LineHighlights returns highlights for a zero-based logical line.
func (ta *Textarea) LineHighlights(lineIdx int) []textbuf.Highlight {
	return ta.textBuffer().LineHighlights(lineIdx)
}

// Textarea configuration.

// SetWrapMode sets the text wrapping mode.
func (ta *Textarea) SetWrapMode(m textbuf.WrapMode) {
	ta.editSurface.SetWrapMode(m)
	ta.RequestRender()
}

// SetReadOnly controls whether editing actions can change content.
func (ta *Textarea) SetReadOnly(v bool) {
	ta.readOnly = v
}

// ReadOnly reports whether editing actions are disabled.
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
	ta.selectionFg = c
}

// SetSelectionBg sets the selection background color.
func (ta *Textarea) SetSelectionBg(c style.Color) {
	ta.selectionBg = c
}

// SetPlaceholder sets plain placeholder text shown when the editor is empty.
func (ta *Textarea) SetPlaceholder(text string) {
	ta.editSurface.SetPlaceholderText(text, style.ColorGray)
}

// SetPlaceholderSpans sets a styled placeholder shown when the editor is empty.
func (ta *Textarea) SetPlaceholderSpans(spans []textbuf.Span) {
	ta.editSurface.SetPlaceholderSpans(spans)
}

// SetDefaultAttributes sets the default text attributes (bold, italic, etc.).
func (ta *Textarea) SetDefaultAttributes(attrs style.Attrs) {
	ta.defaultAttributes = attrs
	ta.textBuffer().SetDefaultAttributes(attrs)
	ta.RequestRender()
}

// DefaultAttributes returns the default text attributes.
func (ta *Textarea) DefaultAttributes() style.Attrs {
	return ta.defaultAttributes
}

// SetScrollSpeed sets drag-selection auto-scroll speed in lines per second.
func (ta *Textarea) SetScrollSpeed(speed float64) {
	if speed < 0 {
		speed = 0
	}
	ta.scrollSpeed = speed
}

// EditSurface returns the underlying surface for advanced viewport access.
func (ta *Textarea) EditSurface() *textbuf.EditSurface {
	return ta.editSurface
}

// EditBuffer returns the underlying editable buffer.
func (ta *Textarea) EditBuffer() *textbuf.EditBuffer {
	return ta.editBuffer()
}

// TextBuffer returns the underlying styled text buffer.
func (ta *Textarea) TextBuffer() *textbuf.TextBuffer {
	return ta.textBuffer()
}

// Extmarks returns the extmark controller, creating it on first access.
// The controller tracks extmarks across cursor movement, edits, undo, and redo.
func (ta *Textarea) Extmarks() *textbuf.ExtmarksController {
	if ta.extmarksCtrl == nil {
		ta.extmarksCtrl = textbuf.NewExtmarksController(ta.editBuffer(), ta.editSurface)
	}
	return ta.extmarksCtrl
}

// Focus state.

// Focus activates focused colors and cursor rendering.
func (ta *Textarea) Focus() {
	ta.BaseRenderable.Focus()
	ta.textColor = ta.focusedTextColor
	ta.backgroundColor = ta.focusedBgColor
	ta.RequestRender()
}

// Blur activates unfocused colors and hides the terminal cursor.
func (ta *Textarea) Blur() {
	ta.BaseRenderable.Blur()
	ta.textColor = ta.unfocusedTextColor
	ta.backgroundColor = ta.unfocusedBgColor
	ta.Host().SetCursorPosition(0, 0, false)
	ta.RequestRender()
}

// OnResize updates the EditSurface viewport dimensions.
func (ta *Textarea) OnResize(width, height int) {
	ta.editSurface.SetViewportSize(width, height)
}

// Textarea rendering.

// RenderSelf draws the textarea into the buffer.
func (ta *Textarea) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := ta.Width()
	h := ta.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := ta.X()
	y := ta.Y()

	// Synchronize the editing viewport with the layout rectangle.
	ta.editSurface.SetViewportSize(w, h)

	// Apply the foreground selected by the current focus state.
	ta.textBuffer().SetDefaultFg(ta.textColor)

	// Register the viewport for pointer-event routing.
	ta.Host().AddToHitGrid(x, y, w, h, ta.Num())

	// Draw background
	if !ta.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, ta.backgroundColor)
	}

	// Empty editors render a placeholder but still expose a focused cursor.
	if ta.editSurface.IsEmpty() {
		phText, phColor := ta.editSurface.PlaceholderText()
		if phText != "" {
			drawText := truncateToWidth(phText, w)
			buf.DrawText(drawText, x, y, phColor)
		}
		if ta.Focused() && ta.showCursor {
			ta.Host().SetCursorPosition(x, y, true)
		}
		return
	}

	// Draw text, highlights, and selection through EditSurface.
	buf.DrawEditSurface(ta.editSurface, x, y)

	// Publish the visual cursor position to the terminal renderer.
	if ta.Focused() && ta.showCursor {
		vc := ta.editSurface.VisualCursor()
		cursorScreenX := x + vc.VisualCol
		cursorScreenY := y + vc.VisualRow
		// Permit one-past-the-right-edge at the end of a full-width line.
		if cursorScreenX >= x && cursorScreenX <= x+w &&
			cursorScreenY >= y && cursorScreenY < y+h {
			ta.Host().SetCursorPosition(cursorScreenX, cursorScreenY, true)
		}
	}
}

// Keyboard input.

// HandleKeyPress resolves and runs an editing action for e.
func (ta *Textarea) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease {
		return false
	}

	// Printable runes bypass the action binding table.
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

	// Compile custom bindings lazily after configuration changes.
	action, found := ta.resolveAction(e)
	if !found {
		return false
	}

	// Read-only mode permits only movement and selection actions.
	if ta.readOnly && !navActions[action] {
		return false
	}

	// Dispatch the resolved action through the fixed handler table.
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

// Mouse input.

// ProcessMouseEvent handles editor scrolling and selection before base event dispatch.
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

	// Preserve base handlers and event bubbling.
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

		newY := min(max(vp.Y+delta, 0), maxY)
		if newY != vp.Y {
			vp.Y = newY
			ta.RequestRender()
		}
	}

	// Horizontal deltas move the no-wrap viewport.
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

	// Ignore presses outside the local viewport.
	if localX < 0 || localX >= ta.Width() || localY < 0 || localY >= ta.Height() {
		return
	}

	// Resolve the clicked visual coordinate through EditSurface.
	ta.editSurface.SetLocalSelection(localX, localY, localX, localY,
		ta.selectionBg, ta.selectionFg, true, false)

	// Store a viewport-local anchor for subsequent drag events.
	ta.mouseSelAnchorX = localX
	ta.mouseSelAnchorY = localY
	ta.mouseSelecting = true

	// Replace any keyboard selection with a drag selection in the host.
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.editSurface.ResetSelection()
	ta.Host().StartSelection(ta, e.X, e.Y)

	e.StopPropagation()
}

func (ta *Textarea) handleMouseDrag(e *input.MouseEvent) {
	if !ta.mouseSelecting {
		return
	}

	localX := e.X - ta.X()
	localY := e.Y - ta.Y()

	// Update text offsets from the fixed mouse anchor to the drag position.
	ok := ta.editSurface.UpdateLocalSelection(
		ta.mouseSelAnchorX, ta.mouseSelAnchorY,
		localX, localY,
		ta.selectionBg, ta.selectionFg,
	)

	// Start automatic vertical scrolling within the outer fifth of the viewport.
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
		// Keep screen-coordinate selection synchronized with EditSurface.
		ta.Host().UpdateSelection(ta, e.X, e.Y)
		ta.RequestRender()
	}

	e.StopPropagation()
}

func (ta *Textarea) handleMouseDragEnd(e *input.MouseEvent) {
	if ta.mouseSelecting {
		// Mark the host's drag as complete.
		ta.Host().UpdateSelection(ta, e.X, e.Y,
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

// OnPostLayout advances drag-selection auto-scroll using elapsed milliseconds.
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
			newY := min(max(vp.Y+direction*linesToScroll, 0), maxY)
			if newY != vp.Y {
				vp.Y = newY
				ta.RequestRender()
			}
		}

		ta.autoScrollAccumulator -= float64(direction * linesToScroll)
	}
}

// ShouldStartSelection reports whether an absolute coordinate can begin selection here.
func (ta *Textarea) ShouldStartSelection(x, y int) bool {
	if !ta.Selectable() {
		return false
	}
	localX := x - ta.X()
	localY := y - ta.Y()
	return localX >= 0 && localX < ta.Width() && localY >= 0 && localY < ta.Height()
}

// Destroy releases extmark state, hides the cursor, and destroys the base renderable.
func (ta *Textarea) Destroy() {
	if ta.extmarksCtrl != nil {
		ta.extmarksCtrl.Destroy()
		ta.extmarksCtrl = nil
	}
	if ta.Focused() {
		ta.Host().SetCursorPosition(0, 0, false)
	}
	ta.BaseRenderable.Destroy()
}

// Action dispatch.

// buildActionHandlers maps each supported action name to its handler.
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

// Cursor movement actions.

func (ta *Textarea) actionMoveLeft() bool {
	// Left movement collapses an active selection to its start.
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
	// Right movement collapses an active selection to its end.
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
	// At column zero, move across the newline to the previous line's end.
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
	// At line end, move across the newline to the next line's start.
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

// Selection movement actions.
//
// Each handler records an anchor before moving and updates the focus afterward.

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

// Editing actions.

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

// Selection synchronization.

// HasSelection reports whether EditSurface contains an active text selection.
func (ta *Textarea) HasSelection() bool {
	return ta.editSurface.HasSelection()
}

// updateSelectionForMovement synchronizes selection before and after cursor movement.
// A call before movement records the anchor; a call afterward updates the focus.
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
		// Record both screen and text-offset anchors when selection begins.
		if !ta.HasSelection() {
			vc := ta.editSurface.VisualCursor()
			ta.Host().StartSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow)
			eb := ta.editBuffer()
			ta.selAnchorOffset = eb.PositionToOffset(eb.CursorRow(), eb.CursorCol())
		}
		return
	}

	// Move the screen-coordinate focus to the new visual cursor.
	vc := ta.editSurface.VisualCursor()
	ta.Host().UpdateSelection(ta, ta.X()+vc.VisualCol, ta.Y()+vc.VisualRow,
		render.SelectionUpdateOption{FinishDragging: true})

	// Apply the same range to EditSurface for rendering and editing.
	ta.syncSelectionToEditBufferView()
}

// syncSelectionToEditBufferView applies the anchor-to-cursor range to EditSurface.
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
		BgColor: ta.selectionBg,
		FgColor: ta.selectionFg,
	}
	ta.editSurface.SetSelection(sel)
}

// clearSelection clears both the context-level and EditSurface selection.
func (ta *Textarea) clearSelection() {
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.editSurface.ResetSelection()
	ta.Host().ClearSelection()
}

// deleteSelectedText removes the active range and reports whether one existed.
func (ta *Textarea) deleteSelectedText() bool {
	if !ta.HasSelection() {
		return false
	}
	ta.editSurface.DeleteSelectedText()
	ta.keyboardSelectionActive = false
	ta.selAnchorOffset = 0
	ta.Host().ClearSelection()
	return true
}

// Internal state access.

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
	ta.RequestRender()
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

// defaultBindings returns the standard multiline editor bindings.
func (ta *Textarea) defaultBindings() []input.KeyBinding {
	return DefaultTextareaBindings
}

// Textarea options.

// TextareaWithText sets the initial text.
func TextareaWithText(text string) TextareaOption {
	return func(ta *Textarea) {
		ta.editBuffer().SetText(text)
	}
}

// TextareaWithPlaceholder sets the placeholder text.
func TextareaWithPlaceholder(text string) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetPlaceholderText(text, style.ColorGray)
	}
}

// TextareaWithPlaceholderFg sets the placeholder color.
func TextareaWithPlaceholderFg(fg style.Color) TextareaOption {
	return func(ta *Textarea) {
		phText, _ := ta.editSurface.PlaceholderText()
		ta.editSurface.SetPlaceholderText(phText, fg)
	}
}

// TextareaWithFg sets both focused and unfocused foreground colors.
func TextareaWithFg(c style.Color) TextareaOption {
	return func(ta *Textarea) {
		ta.unfocusedTextColor = c
		ta.focusedTextColor = c
		ta.textColor = c
	}
}

// TextareaWithFocusedFg sets the text color used when focused.
func TextareaWithFocusedFg(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.focusedTextColor = c }
}

// TextareaWithBg sets both focused and unfocused background colors.
func TextareaWithBg(c style.Color) TextareaOption {
	return func(ta *Textarea) {
		ta.unfocusedBgColor = c
		ta.focusedBgColor = c
		ta.backgroundColor = c
	}
}

// TextareaWithFocusedBg sets the background color when focused.
func TextareaWithFocusedBg(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.focusedBgColor = c }
}

// TextareaWithReadOnly controls whether editing actions can change content.
func TextareaWithReadOnly(v bool) TextareaOption {
	return func(ta *Textarea) { ta.readOnly = v }
}

// TextareaWithWrapMode sets the wrap mode.
func TextareaWithWrapMode(m textbuf.WrapMode) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetWrapMode(m)
	}
}

// TextareaWithScrollMargin sets the viewport fraction kept around the cursor.
func TextareaWithScrollMargin(m float32) TextareaOption {
	return func(ta *Textarea) {
		ta.editSurface.SetScrollMargin(m)
	}
}

// TextareaWithShowCursor controls cursor rendering while focused.
func TextareaWithShowCursor(v bool) TextareaOption {
	return func(ta *Textarea) { ta.showCursor = v }
}

// TextareaWithCursorColor sets the cursor color.
func TextareaWithCursorColor(c style.Color) TextareaOption {
	return func(ta *Textarea) { ta.cursorColor = c }
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
	return func(ta *Textarea) { ta.selectionFg = c }
}

// TextareaWithOnCursorChange sets the callback invoked after cursor movement.
func TextareaWithOnCursorChange(fn func(row, col int)) TextareaOption {
	return func(ta *Textarea) { ta.OnCursorChange = fn }
}

// TextareaWithOnContentChange sets the callback invoked after undoable edits.
func TextareaWithOnContentChange(fn func(text string)) TextareaOption {
	return func(ta *Textarea) { ta.OnContentChange = fn }
}

// TextareaWithOnSubmit sets the callback invoked by the submit action.
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

// TextareaWithFlexGrow sets the flex-grow factor.
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

// TextareaWithDefaultAttributes sets the default text attributes (bold, italic, etc.).
func TextareaWithDefaultAttributes(attrs style.Attrs) TextareaOption {
	return func(ta *Textarea) { ta.defaultAttributes = attrs }
}

// TextareaWithSelectable controls whether mouse selection can start in the editor.
func TextareaWithSelectable(v bool) TextareaOption {
	return func(ta *Textarea) { ta.SetSelectable(v) }
}

// TextareaWithScrollSpeed sets drag-selection auto-scroll speed in lines per second.
func TextareaWithScrollSpeed(speed float64) TextareaOption {
	return func(ta *Textarea) { ta.scrollSpeed = speed }
}
