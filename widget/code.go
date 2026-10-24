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

// HighlightInfo identifies the source and theme supplied to a HighlightFunc.
type HighlightInfo struct {
	// Content is the source being highlighted.
	Content string
	// Filetype is the source's language, as the Code widget was given it.
	Filetype string
	// Theme is the theme whose code palette colors the source.
	Theme *style.Theme
}

// HighlightFunc transforms highlighted lines before Code stores them for rendering.
type HighlightFunc func(lines []textbuf.Line, info HighlightInfo) []textbuf.Line

// SelectionRange contains the start and end character offsets of a selection.
type SelectionRange struct {
	// Start is the offset where the selection begins.
	Start int
	// End is the offset where the selection ends.
	End int
}

// Code displays selectable, scrollable source text with wrapping and syntax highlighting.
// It implements the line and scroll interfaces used by LineNumber.
type Code struct {
	*render.BaseRenderable

	content    string
	filetype   string // language identifier
	fg         style.Color
	bg         style.Color
	wrapMode   WrapMode
	theme      *style.Theme // theme whose code palette colors tokens
	attributes style.Attrs  // default text attributes

	// conceal controls whether formatting text marked as concealed is hidden.
	conceal bool

	// drawUnstyledText allows plain source to remain visible during highlighting.
	drawUnstyledText bool

	// streaming adjusts when incremental content replaces the displayed buffer.
	streaming bool

	// isHighlighting is true while SyntaxHighlighter.Highlight is running.
	isHighlighting bool

	// truncate clips lines at the viewport width instead of allowing horizontal scrolling.
	truncate bool

	// onHighlight can transform the highlighter output before it is stored.
	onHighlight HighlightFunc

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

	// textBuf and textSurface provide shared measurement and rendering.
	textBuf     *textbuf.TextBuffer
	textSurface *textbuf.TextSurface

	linesDirty bool

	// shouldRenderTextBuffer hides stale text during configured streaming updates.
	shouldRenderTextBuffer bool

	// hadInitialContent distinguishes the first streaming update from later updates.
	hadInitialContent bool

	// logicalLineCount counts source lines before wrapping.
	logicalLineCount int

	// lineColors maps zero-based logical lines to row background colors.
	lineColors map[int]style.Color
}

// CodeOption configures a Code widget.
type CodeOption func(*Code)

// NewCode returns a Code widget containing content.
func NewCode(content string, opts ...CodeOption) *Code {
	tb := textbuf.NewTextBuffer()
	tv := textbuf.NewTextSurface(tb)

	c := &Code{
		BaseRenderable:         render.NewBaseRenderable(),
		content:                content,
		fg:                     style.ColorWhite,
		bg:                     style.ColorTransparent,
		wrapMode:               WrapWord,
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

	// Apply widget defaults to the shared text pipeline.
	tv.SetWrapMode(c.wrapMode)
	tb.SetDefaultFg(c.fg)
	tb.SetDefaultBg(c.bg)
	tb.SetExpandTabs(true)

	// Use a Highlighter unless the caller supplied a highlighter.
	if c.theme == nil {
		c.theme = defaultTheme()
	}
	if c.highlighter == nil {
		c.highlighter = NewHighlighter(c.theme)
	}

	// Populate the text buffer before the first measurement or render.
	c.highlightContent()

	// Let the parent scroll when this widget cannot move any further.
	c.OnMouseScrollHandler = func(e *input.MouseEvent) {
		if e.Scroll != nil {
			oldX, oldY := c.scrollX, c.scrollY
			c.handleScroll(e.Scroll.DeltaX, e.Scroll.DeltaY)
			if c.scrollX != oldX || c.scrollY != oldY {
				e.StopPropagation()
			}
		}
	}

	// Measure with the same TextSurface used for rendering.
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

// Syntax highlighting.

// highlightContent replaces the text buffer with highlighted or fallback lines.
func (c *Code) highlightContent() {
	c.logicalLineCount = len(strings.Split(c.content, "\n"))

	// Empty content clears and hides the text buffer.
	if len(c.content) == 0 {
		c.shouldRenderTextBuffer = false
		c.textBuf.Clear()
		c.linesDirty = false
		return
	}

	// A missing file type selects the plain-text fallback.
	if c.filetype == "" {
		c.shouldRenderTextBuffer = true
		c.buildFallbackLines()
		return
	}

	// Decide whether old or unstyled text remains visible during this pass.
	c.ensureVisibleTextBeforeHighlight()

	c.isHighlighting = true

	highlighted, err := c.highlighter.Highlight(c.content, c.filetype, c.fg, c.bg)
	if err != nil {
		// Preserve readable source when tokenization fails.
		c.buildFallbackLines()
		c.shouldRenderTextBuffer = true
		c.isHighlighting = false
		return
	}

	// Allow callers to transform successful highlighter output.
	if c.onHighlight != nil {
		info := HighlightInfo{
			Content:  c.content,
			Filetype: c.filetype,
			Theme:    c.theme,
		}
		highlighted = c.onHighlight(highlighted, info)
	}

	c.textBuf.SetLines(highlighted)
	c.linesDirty = false
	c.shouldRenderTextBuffer = true
	c.isHighlighting = false
}

// ensureVisibleTextBeforeHighlight selects the buffer shown while highlighting runs.
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

// buildFallbackLines stores source lines using the default foreground and background.
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

// Content management.

// SetContent replaces the source text.
// In streaming mode with unstyled text disabled, it leaves the displayed buffer
// unchanged until a later highlighting pass.
func (c *Code) SetContent(content string) {
	if c.content == content {
		return
	}
	c.content = content
	c.linesDirty = true

	// Avoid replacing the visible highlighted buffer with unstyled streaming text.
	if c.streaming && !c.drawUnstyledText && c.filetype != "" {
		return
	}

	c.highlightContent()
	c.RequestRender()
}

// Content returns the current content.
func (c *Code) Content() string { return c.content }

// AppendContent appends text and immediately rebuilds highlighted output.
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

// SetStreaming controls incremental-update behavior and resets initial-content state.
func (c *Code) SetStreaming(v bool) {
	if c.streaming != v {
		c.streaming = v
		c.hadInitialContent = false
		c.linesDirty = true
	}
}

// Streaming returns whether streaming mode is active.
func (c *Code) Streaming() bool { return c.streaming }

// Theme configuration.

// SetTheme sets the theme whose code palette colors the source. A nil
// theme selects style.DefaultDark.
func (c *Code) SetTheme(t *style.Theme) {
	if t == nil {
		t = defaultTheme()
	}
	if c.theme == t {
		return
	}
	c.theme = t
	// Update highlighters that expose mutable theme selection.
	type themeSetter interface{ SetTheme(*style.Theme) }
	if ts, ok := c.highlighter.(themeSetter); ok {
		ts.SetTheme(t)
	}
	c.linesDirty = true
	c.highlightContent()
	c.RequestRender()
}

// Theme returns the theme whose code palette colors the source. Without one
// set, it is a default theme that every such Code shares, which must not be
// changed.
func (c *Code) Theme() *style.Theme { return c.theme }

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

// IsHighlighting reports whether the configured highlighter is currently running.
func (c *Code) IsHighlighting() bool { return c.isHighlighting }

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

// Attributes returns the default text attribute bitmask.
func (c *Code) Attributes() style.Attrs { return c.attributes }

// SetAttributes sets the default text attributes.
func (c *Code) SetAttributes(attrs style.Attrs) {
	if c.attributes != attrs {
		c.attributes = attrs
		c.linesDirty = true
		c.RequestRender()
	}
}

// SetOnHighlight sets the transformation applied to successful highlighter output.
func (c *Code) SetOnHighlight(fn HighlightFunc) {
	c.onHighlight = fn
	c.linesDirty = true
	c.RequestRender()
}

// OnHighlightFn returns the current highlight transformation, or nil.
func (c *Code) OnHighlightFn() HighlightFunc { return c.onHighlight }

// PlainText returns the raw content string (without styling).
func (c *Code) PlainText() string { return c.content }

// TextLength returns the length of the content in bytes.
func (c *Code) TextLength() int { return len(c.content) }

// SetLineColors maps zero-based logical lines to row background colors.
func (c *Code) SetLineColors(colors map[int]style.Color) {
	c.lineColors = colors
	c.RequestRender()
}

// LineCount returns the number of logical lines (before wrapping).
func (c *Code) LineCount() int {
	return c.logicalLineCount
}

// VisualLineCount returns the number of visual lines (after wrapping).
func (c *Code) VisualLineCount() int {
	// Use current dimensions so the returned count reflects current wrapping.
	if w, h := c.Width(), c.Height(); w > 0 && h > 0 {
		c.textSurface.SetViewportSize(w, h)
	}
	return c.textSurface.VirtualLineCount()
}

// Scrolling.

// ScrollX returns the horizontal scroll offset.
func (c *Code) ScrollX() int { return c.scrollX }

// ScrollY returns the vertical scroll offset.
func (c *Code) ScrollY() int { return c.scrollY }

// SetScrollX sets the horizontal scroll offset, clamped to the scrollable range.
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

// SetScrollY sets the vertical scroll offset, clamped to the scrollable range.
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

// SetActiveLine sets the zero-based highlighted line; a negative value disables it.
func (c *Code) SetActiveLine(line int) {
	if c.activeLine != line {
		c.activeLine = line
		c.RequestRender()
	}
}

// ActiveLine returns the highlighted line index, or a negative value if disabled.
func (c *Code) ActiveLine() int { return c.activeLine }

// Selection.

// Selectable returns whether the code is selectable.
func (c *Code) Selectable() bool { return c.selectable_ }

// SetSelectable sets whether the code is selectable.
func (c *Code) SetSelectable(v bool) {
	c.selectable_ = v
	c.BaseRenderable.SetSelectable(v)
}

// SelectedText returns the text covered by the active selection in this widget.
func (c *Code) SelectedText() string {
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return ""
	}

	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	if localSel == nil || !localSel.IsActive {
		return ""
	}

	// Temporarily expose all virtual lines while extracting the selected text.
	origVP := c.textSurface.Viewport()
	totalVLines := c.textSurface.VirtualLineCount()
	c.textSurface.SetViewportSize(origVP.Width, totalVLines)
	vp := c.textSurface.Viewport()
	vp.Y = 0

	allVLines := c.textSurface.VirtualLines()

	// Restore the rendering viewport after extraction.
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

// HasSelection reports whether an active selection intersects this widget.
func (c *Code) HasSelection() bool {
	sel := c.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return false
	}
	localSel := render.ConvertGlobalToLocal(sel, c.X(), c.Y(), c.Width(), c.Height())
	return localSel != nil && localSel.IsActive
}

// ShouldStartSelection reports whether an absolute coordinate can begin selection here.
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

	// Convert visual line coordinates to offsets in the flattened text.
	info := c.textSurface.FullLineInfo()

	startY, startX := localSel.AnchorY, localSel.AnchorX
	endY, endX := localSel.FocusY, localSel.FocusX

	// Normalize the endpoints to document order.
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

// OnSelectionChanged requests a redraw when sel intersects this widget.
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

// ensureWrapped rebuilds highlighted lines when content or styling is dirty.
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

// Appearance and wrapping.

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

// WrapModeValue returns the configured wrap mode.
func (c *Code) WrapModeValue() WrapMode { return c.wrapMode }

// Rendering.

// RenderSelf draws the code content into the buffer.
func (c *Code) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := c.Width()
	h := c.Height()
	if w <= 0 || h <= 0 {
		return
	}

	// Refresh highlighted output before synchronizing the viewport.
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

	// Apply widget dimensions and scroll offsets to TextSurface.
	c.textSurface.SetViewportSize(w, h)
	if vp := c.textSurface.Viewport(); vp != nil {
		vp.Y = c.scrollY
		vp.X = c.scrollX
	}

	x := c.X()
	y := c.Y()

	// Fill the viewport before drawing text.
	if !c.bg.IsTransparent() {
		buf.FillRect(x, y, w, h, c.bg)
	}

	// Apply row backgrounds, active-line styling, and truncation.
	c.textSurface.SetLineBackgrounds(c.lineColors)
	c.textSurface.SetActiveLine(c.activeLine, c.activeLineBg)
	c.textSurface.SetTruncate(c.truncate)

	// Convert the global screen selection into TextSurface coordinates.
	c.syncSelection()

	buf.DrawTextSurface(c.textSurface, x, y)
}

// syncSelection applies the global screen selection to TextSurface.
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

// LogicalLineForVisualLine returns the logical source line for a wrapped visual line.
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

// LineHighlights returns styled chunks for a zero-based logical line.
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

// Code options.

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

// CodeWithStreaming controls incremental-update behavior.
func CodeWithStreaming(v bool) CodeOption {
	return func(c *Code) { c.streaming = v }
}

// CodeWithTheme sets the theme whose code palette colors the source. A nil
// theme selects style.DefaultDark.
func CodeWithTheme(t *style.Theme) CodeOption {
	return func(c *Code) { c.theme = t }
}

// CodeWithSelectable controls whether pointer selections can start in the code.
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

// CodeWithFlexGrow sets the flex-grow factor.
func CodeWithFlexGrow(v int) CodeOption {
	return func(c *Code) { c.LayoutNode().FlexGrow = v }
}

// CodeWithConceal controls whether concealed text is hidden; the default is true.
func CodeWithConceal(v bool) CodeOption {
	return func(c *Code) { c.conceal = v }
}

// CodeWithDrawUnstyledText controls whether plain source is visible during highlighting.
func CodeWithDrawUnstyledText(v bool) CodeOption {
	return func(c *Code) { c.drawUnstyledText = v }
}

// CodeWithTruncate controls whether lines are clipped to the viewport width.
func CodeWithTruncate(v bool) CodeOption {
	return func(c *Code) { c.truncate = v }
}

// CodeWithAttributes sets the default text attributes.
func CodeWithAttributes(attrs style.Attrs) CodeOption {
	return func(c *Code) { c.attributes = attrs }
}

// CodeWithOnHighlight sets the transformation applied after successful highlighting.
func CodeWithOnHighlight(fn HighlightFunc) CodeOption {
	return func(c *Code) { c.onHighlight = fn }
}

// CodeWithHighlighter sets a custom SyntaxHighlighter implementation.
// When nil (default), a Highlighter of the Code's theme is used.
func CodeWithHighlighter(h SyntaxHighlighter) CodeOption {
	return func(c *Code) { c.highlighter = h }
}
