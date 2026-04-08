package widget

import (
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// HighlightContext provides context to the OnHighlight callback.
type HighlightContext struct {
	Content  string
	Filetype string
	Theme    string
}

// OnHighlightCallback is called after highlighting completes, allowing
// callers to modify the styled lines before they are rendered.
type OnHighlightCallback func(lines []textbuf.Line, ctx HighlightContext) []textbuf.Line

// SelectionRange represents a character-offset selection range.
type SelectionRange struct {
	Start int
	End   int
}

// Code displays syntax-highlighted code with scroll and wrap support.
//
// Features:
//   - Pluggable syntax highlighting via the SyntaxHighlighter interface
//   - Horizontal and vertical scrolling
//   - Word/character/no-wrap modes
//   - Selection support for copy
//   - Streaming mode for incremental content updates
//   - Conceal mode for hiding formatting characters
//   - Truncation mode
//   - OnHighlight callback for modifying highlights
//   - Implements LineInfoProvider and ScrollInfoProvider for LineNumber integration
type Code struct {
	*render.BaseRenderable

	content    string
	filetype   string // language identifier
	fg         style.Color
	bg         style.Color
	wrapMode   WrapMode
	theme      string // theme name (e.g. "monokai", "dracula")
	attributes uint32 // default text attributes

	// Conceal: when true, concealed text (e.g. markdown formatting) is hidden (default true).
	conceal bool

	// DrawUnstyledText: when true, plain text is drawn before highlighting
	// completes (default true).
	drawUnstyledText bool

	// Streaming
	streaming bool

	// IsHighlighting indicates whether async highlighting is in progress.
	isHighlighting bool

	// Truncate: when true, lines exceeding the viewport width are truncated
	// instead of being scrollable.
	truncate bool

	// OnHighlight is called after highlighting, allowing callers to modify
	// the styled segments.
	onHighlight OnHighlightCallback

	// Scroll
	scrollX int
	scrollY int

	// Selection
	selectable_ bool
	selectionBg style.Color
	selectionFg *style.Color

	// Active line highlighting
	activeLine   int // -1 = none
	activeLineBg style.Color

	// Syntax highlighting
	highlighter SyntaxHighlighter

	// Internal: TextBuffer + TextSurface (shared measurement & rendering)
	textBuf     *textbuf.TextBuffer
	textSurface *textbuf.TextSurface

	linesDirty bool

	// Internal: whether the text buffer should be rendered (for streaming
	// flow where highlighting may not be ready yet).
	shouldRenderTextBuffer bool

	// Internal: whether initial content has been received during streaming.
	hadInitialContent bool

	// Logical line count (before wrapping)
	logicalLineCount int

	// Per-line background colors (logical line → RGBA). Used by Diff and
	// other composing widgets to paint line-level backgrounds.
	lineColors map[int]style.Color
}

// CodeOption configures a Code widget.
type CodeOption func(*Code)

// NewCode creates a Code widget with optional configuration.
func NewCode(content string, opts ...CodeOption) *Code {
	tb := textbuf.NewTextBuffer()
	tv := textbuf.NewTextSurface(tb)

	c := &Code{
		BaseRenderable:         render.NewBaseRenderable(),
		content:                content,
		fg:                     style.ColorWhite,
		bg:                     style.ColorTransparent,
		wrapMode:               WrapWord,
		theme:                  "monokai",
		conceal:                true,
		drawUnstyledText:       true,
		shouldRenderTextBuffer: true,
		selectionBg:            style.ColorFromInts(51, 153, 255, 255),
		activeLine:             -1,
		activeLineBg:           style.ColorFromInts(40, 40, 40, 255),
		linesDirty:             true,
		textBuf:                tb,
		textSurface:            tv,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Sync settings to TextSurface
	tv.SetWrapMode(c.wrapMode)
	tb.SetDefaultFg(c.fg)
	tb.SetDefaultBg(c.bg)
	tb.SetExpandTabs(true)

	// Default highlighter if not set via option
	if c.highlighter == nil {
		c.highlighter = NewChromaHighlighter(c.theme)
	}

	// Build highlighted lines into TextBuffer
	c.highlightContent()

	// Set up mouse scroll handler
	c.OnMouseScrollHandler = func(e *input.MouseEvent) {
		if e.Scroll != nil {
			c.handleScroll(e.Scroll.DeltaX, e.Scroll.DeltaY)
			e.StopPropagation()
		}
	}

	// Set up intrinsic measurement via TextSurface.
	c.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		availW := width
		if widthMode == layout.MeasureUndefined {
			availW = 0
		}
		availH := height
		if heightMode == layout.MeasureUndefined {
			availH = 0
		}
		m := c.textSurface.MeasureForDimensions(availW, availH)
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
		if widthMode == layout.MeasureExactly {
			measuredW = width
		} else if widthMode == layout.MeasureAtMost && measuredW > width {
			measuredW = width
		}
		if heightMode == layout.MeasureExactly {
			measuredH = height
		} else if heightMode == layout.MeasureAtMost && measuredH > height {
			measuredH = height
		}
		return layout.MeasureResult{Width: measuredW, Height: measuredH}
	}

	return c
}

// ---------------------------------------------------------------------------
// Syntax Highlighting
// ---------------------------------------------------------------------------

// highlightContent tokenizes the source via the SyntaxHighlighter and
// populates the TextBuffer with highlighted lines.
func (c *Code) highlightContent() {
	c.logicalLineCount = len(strings.Split(c.content, "\n"))

	// Empty content — nothing to render
	if len(c.content) == 0 {
		c.shouldRenderTextBuffer = false
		c.textBuf.Clear()
		c.linesDirty = false
		return
	}

	// No filetype — always render plain text
	if c.filetype == "" {
		c.shouldRenderTextBuffer = true
		c.buildFallbackLines()
		return
	}

	// Ensure visible text before highlight
	c.ensureVisibleTextBeforeHighlight()

	c.isHighlighting = true

	highlighted, err := c.highlighter.Highlight(c.content, c.filetype, c.fg, c.bg)
	if err != nil {
		// Fallback: plain text
		c.buildFallbackLines()
		c.shouldRenderTextBuffer = true
		c.isHighlighting = false
		return
	}

	// Invoke onHighlight callback if set
	if c.onHighlight != nil {
		ctx := HighlightContext{
			Content:  c.content,
			Filetype: c.filetype,
			Theme:    c.theme,
		}
		highlighted = c.onHighlight(highlighted, ctx)
	}

	c.textBuf.SetLines(highlighted)
	c.linesDirty = false
	c.shouldRenderTextBuffer = true
	c.isHighlighting = false
}

// ensureVisibleTextBeforeHighlight decides whether to show plain text while
// highlighting runs.
func (c *Code) ensureVisibleTextBeforeHighlight() {
	if c.filetype == "" {
		c.shouldRenderTextBuffer = true
		return
	}

	isInitialContent := c.streaming && !c.hadInitialContent
	shouldDrawUnstyledNow := c.drawUnstyledText
	if c.streaming {
		shouldDrawUnstyledNow = isInitialContent && c.drawUnstyledText
	}

	if c.streaming && !isInitialContent {
		c.shouldRenderTextBuffer = true
	} else if shouldDrawUnstyledNow {
		c.buildFallbackLines()
		c.shouldRenderTextBuffer = true
	} else {
		c.shouldRenderTextBuffer = false
	}

	if isInitialContent {
		c.hadInitialContent = true
	}
}

// buildFallbackLines creates plain lines in the TextBuffer (fallback).
func (c *Code) buildFallbackLines() {
	rawLines := strings.Split(c.content, "\n")
	result := make([]textbuf.Line, len(rawLines))
	for i, line := range rawLines {
		result[i] = textbuf.Line{
			Spans: []textbuf.Span{{
				Text: line,
				Fg:   c.fg,
				Bg:   c.bg,
			}},
		}
	}
	c.textBuf.SetLines(result)
	c.linesDirty = false
}

// ---------------------------------------------------------------------------
// Content Management
// ---------------------------------------------------------------------------

// SetContent sets the code content.
// When streaming with a filetype and drawUnstyledText disabled, the content
// is stored but rendering is deferred until highlighting completes.
func (c *Code) SetContent(content string) {
	if c.content == content {
		return
	}
	c.content = content
	c.linesDirty = true

	// When streaming + !drawUnstyledText + filetype set,
	// defer the text buffer update until highlighting finishes.
	if c.streaming && !c.drawUnstyledText && c.filetype != "" {
		return
	}

	c.highlightContent()
	c.RequestRender()
}

// Content returns the current content.
func (c *Code) Content() string { return c.content }

// AppendContent appends text to the content (optimized for streaming).
func (c *Code) AppendContent(text string) {
	c.content += text
	c.linesDirty = true
	c.highlightContent()
	c.RequestRender()
}

// SetFiletype sets the language for syntax highlighting.
func (c *Code) SetFiletype(ft string) {
	if c.filetype == ft {
		return
	}
	c.filetype = ft
	c.linesDirty = true
	c.highlightContent()
	c.RequestRender()
}

// Filetype returns the current language.
func (c *Code) Filetype() string { return c.filetype }

// SetStreaming sets whether content is being streamed.
// Resets internal highlight state when the value changes.
func (c *Code) SetStreaming(v bool) {
	if c.streaming != v {
		c.streaming = v
		c.hadInitialContent = false
		c.linesDirty = true
	}
}

// Streaming returns whether streaming mode is active.
func (c *Code) Streaming() bool { return c.streaming }

// ---------------------------------------------------------------------------
// Theme
// ---------------------------------------------------------------------------

// SetTheme sets the syntax highlighting theme by name (e.g. "monokai", "dracula", "github").
func (c *Code) SetTheme(name string) {
	if c.theme == name {
		return
	}
	c.theme = name
	// Propagate to highlighter if it supports theme changes.
	type themeSetter interface{ SetTheme(string) }
	if ts, ok := c.highlighter.(themeSetter); ok {
		ts.SetTheme(name)
	}
	c.linesDirty = true
	c.highlightContent()
	c.RequestRender()
}

// Theme returns the current theme name.
func (c *Code) Theme() string { return c.theme }

// ---------------------------------------------------------------------------
// Conceal
// ---------------------------------------------------------------------------

// Conceal returns whether concealed text is hidden.
func (c *Code) Conceal() bool { return c.conceal }

// SetConceal sets whether concealed text (e.g. markdown formatting) is hidden.
func (c *Code) SetConceal(v bool) {
	if c.conceal != v {
		c.conceal = v
		c.linesDirty = true
		c.RequestRender()
	}
}

// ---------------------------------------------------------------------------
// DrawUnstyledText
// ---------------------------------------------------------------------------

// DrawUnstyledText returns whether plain text is drawn before highlighting.
func (c *Code) DrawUnstyledText() bool { return c.drawUnstyledText }

// SetDrawUnstyledText sets whether plain text is drawn while highlighting runs.
func (c *Code) SetDrawUnstyledText(v bool) {
	if c.drawUnstyledText != v {
		c.drawUnstyledText = v
		c.linesDirty = true
		c.RequestRender()
	}
}

// ---------------------------------------------------------------------------
// IsHighlighting
// ---------------------------------------------------------------------------

// IsHighlighting returns whether async highlighting is currently in progress.
func (c *Code) IsHighlighting() bool { return c.isHighlighting }

// ---------------------------------------------------------------------------
// Truncate
// ---------------------------------------------------------------------------

// Truncate returns whether lines are truncated to the viewport width.
func (c *Code) Truncate() bool { return c.truncate }

// SetTruncate sets whether lines exceeding the viewport width are truncated.
func (c *Code) SetTruncate(v bool) {
	if c.truncate != v {
		c.truncate = v
		c.RequestRender()
	}
}

// TextBuffer returns the underlying TextBuffer.
func (c *Code) TextBuffer() *textbuf.TextBuffer { return c.textBuf }

// ---------------------------------------------------------------------------
// Default Attributes
// ---------------------------------------------------------------------------

// Attributes returns the default text attributes (bold, italic, etc.).
func (c *Code) Attributes() uint32 { return c.attributes }

// SetAttributes sets the default text attributes.
func (c *Code) SetAttributes(attrs uint32) {
	if c.attributes != attrs {
		c.attributes = attrs
		c.linesDirty = true
		c.RequestRender()
	}
}

// ---------------------------------------------------------------------------
// OnHighlight callback
// ---------------------------------------------------------------------------

// SetOnHighlight sets a callback invoked after highlighting, allowing callers
// to modify the styled segments before rendering.
func (c *Code) SetOnHighlight(fn OnHighlightCallback) {
	c.onHighlight = fn
	c.linesDirty = true
	c.RequestRender()
}

// OnHighlightFn returns the current OnHighlight callback, or nil.
func (c *Code) OnHighlightFn() OnHighlightCallback { return c.onHighlight }

// ---------------------------------------------------------------------------
// PlainText / TextLength
// ---------------------------------------------------------------------------

// PlainText returns the raw content string (without styling).
func (c *Code) PlainText() string { return c.content }

// TextLength returns the length of the content in bytes.
func (c *Code) TextLength() int { return len(c.content) }

// ---------------------------------------------------------------------------
// Per-line Colors
// ---------------------------------------------------------------------------

// SetLineColors sets per-line background colors (logical line → RGBA).
// Used by Diff to paint line-level backgrounds (green for adds, red for removes).
func (c *Code) SetLineColors(colors map[int]style.Color) {
	c.lineColors = colors
	c.RequestRender()
}

// ---------------------------------------------------------------------------
// Line Info (implements LineInfoProvider)
// ---------------------------------------------------------------------------

// LineCount returns the number of logical lines (before wrapping).
func (c *Code) LineCount() int {
	return c.logicalLineCount
}

// VisualLineCount returns the number of visual lines (after wrapping).
func (c *Code) VisualLineCount() int {
	// Ensure viewport is synced to current widget dimensions so wrapping is correct.
	if w, h := c.Width(), c.Height(); w > 0 && h > 0 {
		c.textSurface.SetViewportSize(w, h)
	}
	return c.textSurface.VirtualLineCount()
}

// ---------------------------------------------------------------------------
// Scroll (implements ScrollInfoProvider)
// ---------------------------------------------------------------------------

// ScrollX returns the horizontal scroll offset.
func (c *Code) ScrollX() int { return c.scrollX }

// ScrollY returns the vertical scroll offset.
func (c *Code) ScrollY() int { return c.scrollY }

// SetScrollX sets the horizontal scroll offset (clamped).
func (c *Code) SetScrollX(v int) {
	maxX := c.MaxScrollX()
	if v < 0 {
		v = 0
	}
	if v > maxX {
		v = maxX
	}
	if c.scrollX != v {
		c.scrollX = v
		c.RequestRender()
	}
}

// SetScrollY sets the vertical scroll offset (clamped).
func (c *Code) SetScrollY(v int) {
	maxY := c.MaxScrollY()
	if v < 0 {
		v = 0
	}
	if v > maxY {
		v = maxY
	}
	if c.scrollY != v {
		c.scrollY = v
		c.RequestRender()
	}
}

// MaxScrollX returns the maximum horizontal scroll offset.
func (c *Code) MaxScrollX() int {
	m := c.textSurface.MeasureForDimensions(0, 0)
	maxW := 0
	if m != nil {
		maxW = m.MaxWidth
	}
	result := maxW - c.Width()
	if result < 0 {
		return 0
	}
	return result
}

// MaxScrollY returns the maximum vertical scroll offset.
func (c *Code) MaxScrollY() int {
	result := c.textSurface.VirtualLineCount() - c.Height()
	if result < 0 {
		return 0
	}
	return result
}

// ScrollWidth returns the widest line width across all wrapped lines.
func (c *Code) ScrollWidth() int {
	m := c.textSurface.MeasureForDimensions(0, 0)
	if m != nil {
		return m.MaxWidth
	}
	return 0
}

// ScrollHeight returns the total number of visual lines.
func (c *Code) ScrollHeight() int {
	return c.textSurface.VirtualLineCount()
}

// ScrollToBottom scrolls to show the last line.
func (c *Code) ScrollToBottom() {
	c.SetScrollY(c.MaxScrollY())
}

// handleScroll processes mouse scroll events.
func (c *Code) handleScroll(deltaX, deltaY int) {
	c.SetScrollY(c.scrollY + deltaY)
	if c.wrapMode == WrapNone {
		c.SetScrollX(c.scrollX + deltaX)
	}
}

// ---------------------------------------------------------------------------
// Active Line
// ---------------------------------------------------------------------------

// SetActiveLine sets the highlighted line index (-1 to disable).
func (c *Code) SetActiveLine(line int) {
	if c.activeLine != line {
		c.activeLine = line
		c.RequestRender()
	}
}

// ActiveLine returns the active line index (-1 if none).
func (c *Code) ActiveLine() int { return c.activeLine }

// ---------------------------------------------------------------------------
// Selection
// ---------------------------------------------------------------------------

// Selectable returns whether the code is selectable.
func (c *Code) Selectable() bool { return c.selectable_ }

// SetSelectable sets whether the code is selectable.
func (c *Code) SetSelectable(v bool) {
	c.selectable_ = v
	c.BaseRenderable.SetSelectable(v)
}

// SelectedText returns the text within the current render.
func (c *Code) SelectedText() string {
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return ""
	}

	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	if localSel == nil || !localSel.IsActive {
		return ""
	}

	// Save/restore viewport to iterate all virtual lines
	origVP := c.textSurface.Viewport()
	totalVLines := c.textSurface.VirtualLineCount()
	c.textSurface.SetViewportSize(origVP.Width, totalVLines)
	vp := c.textSurface.Viewport()
	vp.Y = 0

	allVLines := c.textSurface.VirtualLines()

	// Restore viewport
	c.textSurface.SetViewportSize(origVP.Width, origVP.Height)
	if rvp := c.textSurface.Viewport(); rvp != nil {
		rvp.Y = origVP.Y
	}

	var sb strings.Builder
	for lineIdx, vl := range allVLines {
		var lineText strings.Builder
		for _, chunk := range vl.Chunks {
			lineText.WriteString(chunk.Text)
		}
		for cx, r := range []rune(lineText.String()) {
			if isInSelection(cx, lineIdx, localSel) {
				sb.WriteRune(r)
			}
		}
		if lineIdx < len(allVLines)-1 {
			if lineIdx+1 <= localSel.FocusY || lineIdx+1 <= localSel.AnchorY {
				sb.WriteByte('\n')
			}
		}
	}

	return sb.String()
}

// HasSelection returns true if there's an active selection intersecting this widget.
func (c *Code) HasSelection() bool {
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return false
	}
	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	return localSel != nil && localSel.IsActive
}

// ShouldStartSelection returns true if the given absolute coordinate falls
// within this widget's bounds and the widget is selectable.
func (c *Code) ShouldStartSelection(x, y int) bool {
	if !c.selectable_ {
		return false
	}
	localX := x - c.X()
	localY := y - c.Y()
	return localX >= 0 && localX < c.Width() && localY >= 0 && localY < c.Height()
}

// Selection returns the character-offset range of the current selection,
// or nil if there is no active selection intersecting this widget.
func (c *Code) Selection() *SelectionRange {
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return nil
	}

	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	if localSel == nil || !localSel.IsActive {
		return nil
	}

	// Convert 2D selection bounds to a linear character range
	info := c.textSurface.FullLineInfo()

	startY, startX := localSel.AnchorY, localSel.AnchorX
	endY, endX := localSel.FocusY, localSel.FocusX

	// Normalize so start <= end
	if startY > endY || (startY == endY && startX > endX) {
		startY, startX, endY, endX = endY, endX, startY, startX
	}

	startOff := 0
	endOff := 0
	charOff := 0
	for lineIdx := 0; lineIdx < len(info.Widths); lineIdx++ {
		lineLen := info.Widths[lineIdx]
		if lineIdx == startY {
			startOff = charOff + startX
		}
		if lineIdx == endY {
			endOff = charOff + endX
			break
		}
		charOff += lineLen + 1
	}

	return &SelectionRange{Start: startOff, End: endOff}
}

// OnSelectionChanged updates the widget's local selection state from a global
// Selection object and returns true if the selection intersects this widget.
func (c *Code) OnSelectionChanged(sel *render.Selection) bool {
	if sel == nil || !sel.IsActive() {
		return false
	}
	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	if localSel == nil || !localSel.IsActive {
		return false
	}
	c.RequestRender()
	return true
}

// ---------------------------------------------------------------------------
// Wrapping
// ---------------------------------------------------------------------------

// ensureWrapped recomputes lines if dirty.
func (c *Code) ensureWrapped() {
	if !c.linesDirty {
		return
	}
	c.highlightContent()
}

// OnResize updates the TextSurface viewport when size changes.
func (c *Code) OnResize(width, height int) {
	c.textSurface.SetViewportSize(width, height)
}

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

// SetFg sets the default foreground color.
func (c *Code) SetFg(fg style.Color) {
	c.fg = fg
	c.linesDirty = true
	c.RequestRender()
}

// SetBg sets the default background color.
func (c *Code) SetBg(bg style.Color) {
	c.bg = bg
	c.RequestRender()
}

// SetWrapMode sets the wrap mode.
func (c *Code) SetWrapMode(m WrapMode) {
	if c.wrapMode != m {
		c.wrapMode = m
		c.textSurface.SetWrapMode(m)
		c.RequestRender()
	}
}

// Fg returns the foreground color.
func (c *Code) Fg() style.Color { return c.fg }

// Bg returns the background color.
func (c *Code) Bg() style.Color { return c.bg }

// WrapModeValue returns the current wrap mode.
func (c *Code) WrapModeValue() WrapMode { return c.wrapMode }

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// RenderSelf draws the code content into the buffer.
func (c *Code) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := c.Width()
	h := c.Height()
	if w <= 0 || h <= 0 {
		return
	}

	// Re-highlight if dirty
	if c.linesDirty {
		if len(c.content) == 0 {
			c.shouldRenderTextBuffer = false
			c.linesDirty = false
		} else if c.filetype == "" {
			c.shouldRenderTextBuffer = true
			c.linesDirty = false
		} else {
			c.linesDirty = false
			c.highlightContent()
		}
	}

	if !c.shouldRenderTextBuffer {
		return
	}

	// Sync viewport (scroll + dimensions) to TextSurface
	c.textSurface.SetViewportSize(w, h)
	if vp := c.textSurface.Viewport(); vp != nil {
		vp.Y = c.scrollY
		vp.X = c.scrollX
	}

	x := c.X()
	y := c.Y()

	// Fill background
	if !c.bg.IsTransparent() {
		buf.FillRect(x, y, w, h, c.bg)
	}

	// Push per-line config to TextSurface
	c.textSurface.SetLineBackgrounds(c.lineColors)
	c.textSurface.SetActiveLine(c.activeLine, c.activeLineBg)
	c.textSurface.SetTruncate(c.truncate)

	// Convert screen selection → text buffer selection
	c.syncSelection()

	buf.DrawTextSurface(c.textSurface, x, y)
}

// syncSelection converts global screen selection to a TextSurface render.
func (c *Code) syncSelection() {
	c.textSurface.ResetSelection()
	if !c.selectable_ {
		return
	}
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return
	}
	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	if localSel == nil || !localSel.IsActive {
		return
	}
	bg := c.selectionBg
	c.textSurface.SetLocalSelection(localSel.AnchorX, localSel.AnchorY, localSel.FocusX, localSel.FocusY, &bg, c.selectionFg)
}

// LogicalLineForVisualLine maps a visual line index to its logical line index.
// Implements VisualLineProvider for LineNumber integration with word wrap.
func (c *Code) LogicalLineForVisualLine(visualLine int) int {
	info := c.textSurface.FullLineInfo()
	if len(info.Sources) == 0 {
		return visualLine
	}
	if visualLine < 0 {
		return 0
	}
	if visualLine >= len(info.Sources) {
		return info.Sources[len(info.Sources)-1]
	}
	return info.Sources[visualLine]
}

// ---------------------------------------------------------------------------
// Line Highlights (for LineNumber integration)
// ---------------------------------------------------------------------------

// LineHighlights returns the styled chunks for a given logical line.
// Used by Diff and other widgets that compose Code.
func (c *Code) LineHighlights(lineIdx int) []style.TextChunk {
	if lineIdx < 0 || lineIdx >= c.textBuf.LineCount() {
		return nil
	}

	line := c.textBuf.LineAt(lineIdx)
	chunks := make([]style.TextChunk, len(line.Spans))
	for i, sp := range line.Spans {
		fg := sp.Fg
		bg := sp.Bg
		chunks[i] = style.TextChunk{
			Text:       sp.Text,
			Fg:         &fg,
			Bg:         &bg,
			Attributes: sp.Attributes,
		}
	}
	return chunks
}

// ---------------------------------------------------------------------------
// Code Options
// ---------------------------------------------------------------------------

// CodeWithFiletype sets the language for syntax highlighting.
func CodeWithFiletype(ft string) CodeOption {
	return func(c *Code) { c.filetype = ft }
}

// CodeWithFg sets the foreground color.
func CodeWithFg(color style.Color) CodeOption {
	return func(c *Code) { c.fg = color }
}

// CodeWithBg sets the background color.
func CodeWithBg(color style.Color) CodeOption {
	return func(c *Code) { c.bg = color }
}

// CodeWithWrapMode sets the wrap mode.
func CodeWithWrapMode(m WrapMode) CodeOption {
	return func(c *Code) { c.wrapMode = m }
}

// CodeWithStreaming sets streaming mode.
func CodeWithStreaming(v bool) CodeOption {
	return func(c *Code) { c.streaming = v }
}

// CodeWithTheme sets the syntax highlighting theme name.
func CodeWithTheme(name string) CodeOption {
	return func(c *Code) { c.theme = name }
}

// CodeWithSelectable makes the code selectable.
func CodeWithSelectable(v bool) CodeOption {
	return func(c *Code) { c.selectable_ = v }
}

// CodeWithSelectionBg sets the selection background color.
func CodeWithSelectionBg(color style.Color) CodeOption {
	return func(c *Code) { c.selectionBg = color }
}

// CodeWithSelectionFg sets the selection foreground color.
func CodeWithSelectionFg(color style.Color) CodeOption {
	return func(c *Code) { c.selectionFg = &color }
}

// CodeWithScrollX sets the initial horizontal scroll offset.
func CodeWithScrollX(v int) CodeOption {
	return func(c *Code) { c.scrollX = v }
}

// CodeWithScrollY sets the initial vertical scroll offset.
func CodeWithScrollY(v int) CodeOption {
	return func(c *Code) { c.scrollY = v }
}

// CodeWithActiveLine sets the initially highlighted active line.
func CodeWithActiveLine(line int) CodeOption {
	return func(c *Code) { c.activeLine = line }
}

// CodeWithActiveLineBg sets the active line background color.
func CodeWithActiveLineBg(color style.Color) CodeOption {
	return func(c *Code) { c.activeLineBg = color }
}

// CodeWithID sets the widget ID.
func CodeWithID(id string) CodeOption {
	return func(c *Code) { c.SetID(id) }
}

// CodeWithWidth sets the widget width.
func CodeWithWidth(s layout.Scalar) CodeOption {
	return func(c *Code) { c.LayoutNode().Width = s }
}

// CodeWithHeight sets the widget height.
func CodeWithHeight(s layout.Scalar) CodeOption {
	return func(c *Code) { c.LayoutNode().Height = s }
}

// CodeWithFlexGrow sets flex grow.
func CodeWithFlexGrow(v int) CodeOption {
	return func(c *Code) { c.LayoutNode().FlexGrow = v }
}

// CodeWithConceal sets whether concealed text is hidden (default true).
func CodeWithConceal(v bool) CodeOption {
	return func(c *Code) { c.conceal = v }
}

// CodeWithDrawUnstyledText sets whether plain text is drawn before highlighting (default true).
func CodeWithDrawUnstyledText(v bool) CodeOption {
	return func(c *Code) { c.drawUnstyledText = v }
}

// CodeWithTruncate sets whether lines are truncated to viewport width.
func CodeWithTruncate(v bool) CodeOption {
	return func(c *Code) { c.truncate = v }
}

// CodeWithAttributes sets the default text attributes.
func CodeWithAttributes(attrs uint32) CodeOption {
	return func(c *Code) { c.attributes = attrs }
}

// CodeWithOnHighlight sets the highlight callback.
func CodeWithOnHighlight(fn OnHighlightCallback) CodeOption {
	return func(c *Code) { c.onHighlight = fn }
}

// CodeWithHighlighter sets a custom SyntaxHighlighter implementation.
// When nil (default), a ChromaHighlighter is used.
func CodeWithHighlighter(h SyntaxHighlighter) CodeOption {
	return func(c *Code) { c.highlighter = h }
}
