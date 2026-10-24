package widget

import (
	"strings"
	"unicode"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// WrapMode aliases textbuf.WrapMode for widget text APIs.
type WrapMode = textbuf.WrapMode

const (
	WrapNone = textbuf.WrapNone // Preserves logical lines without wrapping.
	WrapChar = textbuf.WrapChar // Wraps at grapheme-cluster boundaries.
	WrapWord = textbuf.WrapWord // Wraps at whitespace boundaries when possible.
)

// wrappedLine contains one line produced by the local wrapping helpers.
type wrappedLine struct {
	chunks []wrappedChunk
}

// lineWidth returns the terminal width of the line.
func (wl wrappedLine) lineWidth() int {
	w := 0
	for _, c := range wl.chunks {
		w += textbuf.StringWidth(c.text)
	}
	return w
}

// wrappedChunk contains adjacent wrapped text with one style.
type wrappedChunk struct {
	text       string
	fg         style.Color
	bg         style.Color
	attributes style.Attrs
}

// styledSegment contains unwrapped text with one style.
type styledSegment struct {
	text       string
	fg         style.Color
	bg         style.Color
	attributes style.Attrs
}

// Text displays styled, selectable text with wrapping, truncation, and scrolling.
//
// Content can come directly from SetContent or SetStyledContent, or it can be
// assembled from a TextNode tree through AddNode and AddText.
//
// Direct content takes precedence over TextNode children.
type Text struct {
	*render.BaseRenderable

	content     style.StyledText
	fg          style.Color
	bg          style.Color
	attributes  style.Attrs
	wrapMode    WrapMode
	truncate    bool
	selectable_ bool
	selectionBg style.Color
	selectionFg style.Color

	// scrollX and scrollY are offsets in the rendered TextSurface.
	scrollX int
	scrollY int

	// rootTextNode supplies content until a direct content setter is called.
	rootTextNode     *TextNode
	hasManualContent bool

	// textBuf and textSurface provide shared measurement and rendering.
	textBuf     *textbuf.TextBuffer
	textSurface *textbuf.TextSurface

	linesDirty bool
}

// TextOption configures a Text widget during construction.
type TextOption func(*Text)

// NewText returns a Text widget with plain content.
// An empty content string leaves the TextNode tree as the content source.
func NewText(content string, opts ...TextOption) *Text {
	t := newTextBase(opts...)
	if content != "" {
		t.content = style.NewStyledText(style.Plain(content))
		t.hasManualContent = true
	}
	t.finishInit()
	return t
}

// NewStyledText returns a Text widget with pre-styled content.
func NewStyledText(content style.StyledText, opts ...TextOption) *Text {
	t := newTextBase(opts...)
	t.content = content
	t.hasManualContent = true
	t.finishInit()
	return t
}

// newTextBase creates the shared state used by both Text constructors.
func newTextBase(opts ...TextOption) *Text {
	tb := textbuf.NewTextBuffer()
	tv := textbuf.NewTextSurface(tb)

	t := &Text{
		BaseRenderable: render.NewBaseRenderable(),
		fg:             style.ColorWhite,
		bg:             style.ColorTransparent,
		wrapMode:       WrapNone,
		selectionBg:    style.ColorFromInts(51, 153, 255, 255),
		linesDirty:     true,
		textBuf:        tb,
		textSurface:    tv,
	}

	// Every Text has a stable root for later TextNode additions.
	t.rootTextNode = NewTextNode(TextNodeWithID(t.ID() + "-root"))
	t.rootTextNode.ownerText = t

	for _, opt := range opts {
		opt(t)
	}
	return t
}

// finishInit connects configured content and styles to the text pipeline.
func (t *Text) finishInit() {
	// Apply widget configuration to TextBuffer and TextSurface.
	t.textSurface.SetWrapMode(t.wrapMode)
	t.textSurface.SetTruncate(t.truncate)
	t.textBuf.SetDefaultFg(t.fg)
	t.textBuf.SetDefaultBg(t.bg)
	t.textBuf.SetExpandTabs(true)

	// Populate TextBuffer before the first measurement.
	t.syncContentToTextBuffer()

	if t.selectable_ {
		t.SetSelectable(true)
	}

	// Measure with the same TextSurface used for rendering.
	t.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		// Include pending TextNode changes in intrinsic dimensions.
		t.ensureWrapped()
		availW := width
		if widthMode == layout.MeasureUndefined {
			availW = 0
		}
		availH := height
		if heightMode == layout.MeasureUndefined {
			availH = 0
		}
		m := t.textSurface.MeasureForDimensions(availW, availH)
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
		// Let containers collapse an empty Text widget.
		if len(t.content.Chunks) == 0 {
			measuredW = 0
			measuredH = 0
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
}

// Content management.

// SetContent replaces the content with a plain string and disables TextNode content.
func (t *Text) SetContent(content string) {
	t.content = style.NewStyledText(style.Plain(content))
	t.hasManualContent = true
	t.linesDirty = true
	t.LayoutNode().MarkDirty()
	t.RequestRender()
}

// SetStyledContent replaces the content with styled text and disables TextNode content.
func (t *Text) SetStyledContent(content style.StyledText) {
	t.content = content
	t.hasManualContent = true
	t.linesDirty = true
	t.LayoutNode().MarkDirty()
	t.RequestRender()
}

// SetStyledContentQuiet replaces styled content without requesting another render.
// Use it from a render hook where a render is already in progress.
func (t *Text) SetStyledContentQuiet(content style.StyledText) {
	t.content = content
	t.hasManualContent = true
	t.linesDirty = true
}

// Content returns the current styled text content.
func (t *Text) Content() style.StyledText {
	return t.content
}

// PlainText returns the content without styling.
func (t *Text) PlainText() string {
	return t.content.PlainText()
}

// LineCount returns the number of wrapped lines.
func (t *Text) LineCount() int {
	t.ensureWrapped()
	if len(t.content.Chunks) == 0 {
		return 0
	}
	if w := t.Width(); w > 0 {
		t.textSurface.SetViewportSize(w, t.Height())
	}
	return t.textSurface.VirtualLineCount()
}

// TextNode integration.

// textNodeDirtied schedules a rebuild while the node tree remains the content source.
func (t *Text) textNodeDirtied() {
	if !t.hasManualContent {
		t.linesDirty = true
		t.RequestRender()
	}
}

// TextNode returns the root node used to assemble inline content.
func (t *Text) TextNode() *TextNode {
	return t.rootTextNode
}

// AddNode appends a TextNode child to this Text widget.
func (t *Text) AddNode(child *TextNode) int {
	return t.rootTextNode.AddNode(child)
}

// AddText appends a plain string child to the TextNode tree.
func (t *Text) AddText(s string) int {
	return t.rootTextNode.AddString(s)
}

// RemoveNode removes a TextNode child by ID.
func (t *Text) RemoveNode(id string) bool {
	return t.rootTextNode.Remove(id)
}

// ClearNodes removes all TextNode children.
func (t *Text) ClearNodes() {
	t.rootTextNode.Clear()
}

// rebuildFromTextNodes flattens the current TextNode tree into styled content.
func (t *Text) rebuildFromTextNodes() {
	inherited := InheritedStyle{
		Fg:         t.fg,
		Bg:         t.bg,
		Attributes: t.attributes,
	}
	chunks := t.rootTextNode.GatherChunks(inherited)
	if len(chunks) == 0 {
		t.content = style.StyledText{}
	} else {
		t.content = style.StyledText{Chunks: chunks}
	}
	t.rootTextNode.ClearDirty()
}

// Scrolling.

// ScrollX returns the horizontal scroll offset.
func (t *Text) ScrollX() int { return t.scrollX }

// ScrollY returns the vertical scroll offset.
func (t *Text) ScrollY() int { return t.scrollY }

// SetScrollX sets the horizontal scroll offset, clamped to the scrollable range.
func (t *Text) SetScrollX(v int) {
	maxX := t.MaxScrollX()
	if v < 0 {
		v = 0
	}
	if v > maxX {
		v = maxX
	}
	if t.scrollX != v {
		t.scrollX = v
		t.RequestRender()
	}
}

// SetScrollY sets the vertical scroll offset, clamped to the scrollable range.
func (t *Text) SetScrollY(v int) {
	maxY := t.MaxScrollY()
	if v < 0 {
		v = 0
	}
	if v > maxY {
		v = maxY
	}
	if t.scrollY != v {
		t.scrollY = v
		t.RequestRender()
	}
}

// MaxScrollX returns the maximum horizontal scroll offset.
func (t *Text) MaxScrollX() int {
	m := t.textSurface.MeasureForDimensions(0, 0)
	maxW := 0
	if m != nil {
		maxW = m.MaxWidth
	}
	result := maxW - t.Width()
	if result < 0 {
		return 0
	}
	return result
}

// MaxScrollY returns the maximum vertical scroll offset.
func (t *Text) MaxScrollY() int {
	result := t.textSurface.VirtualLineCount() - t.Height()
	if result < 0 {
		return 0
	}
	return result
}

// ScrollWidth returns the maximum line width across all wrapped lines.
func (t *Text) ScrollWidth() int {
	m := t.textSurface.MeasureForDimensions(0, 0)
	if m != nil {
		return m.MaxWidth
	}
	return 0
}

// ScrollHeight returns the total number of wrapped lines.
func (t *Text) ScrollHeight() int {
	return t.textSurface.VirtualLineCount()
}

// Text rendering.

// RenderSelf draws the text content into the buffer.
func (t *Text) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := t.Width()
	h := t.Height()
	if w <= 0 || h <= 0 {
		return
	}

	t.ensureWrapped()

	// Apply widget dimensions and scroll offsets to TextSurface.
	t.textSurface.SetViewportSize(w, h)
	if vp := t.textSurface.Viewport(); vp != nil {
		vp.Y = t.scrollY
		vp.X = t.scrollX
	}

	// Convert the global screen selection to local text coordinates.
	t.syncSelection()

	buf.DrawTextSurface(t.textSurface, t.X(), t.Y())
}

// syncSelection applies the global screen selection to TextSurface.
func (t *Text) syncSelection() {
	t.textSurface.ResetSelection()
	if !t.Selectable() {
		return
	}
	sel := t.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return
	}
	localSel := render.ConvertGlobalToLocal(sel, t.X(), t.Y(), t.Width(), t.Height())
	if localSel == nil || !localSel.IsActive {
		return
	}
	bg := t.selectionBg
	t.textSurface.SetLocalSelection(localSel.AnchorX, localSel.AnchorY, localSel.FocusX, localSel.FocusY, bg, t.selectionFg)
}

// isInSelection reports whether a visual coordinate lies within sel.
func isInSelection(x, y int, sel *render.LocalSelectionBounds) bool {
	if sel == nil || !sel.IsActive {
		return false
	}

	// Normalize endpoints to visual document order.
	startY, startX := sel.AnchorY, sel.AnchorX
	endY, endX := sel.FocusY, sel.FocusX
	if startY > endY || (startY == endY && startX > endX) {
		startY, startX, endY, endX = endY, endX, startY, startX
	}

	if y < startY || y > endY {
		return false
	}
	if y == startY && y == endY {
		return x >= startX && x <= endX
	}
	if y == startY {
		return x >= startX
	}
	if y == endY {
		return x <= endX
	}
	return true
}

// SelectedText returns the text covered by the active selection in this widget.
func (t *Text) SelectedText() string {
	sel := t.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return ""
	}

	localSel := render.ConvertGlobalToLocal(sel, t.X(), t.Y(), t.Width(), t.Height())
	if localSel == nil || !localSel.IsActive {
		return ""
	}

	t.ensureWrapped()

	// Temporarily expose all virtual lines while extracting selected text.
	origVP := t.textSurface.Viewport()
	totalVLines := t.textSurface.VirtualLineCount()
	if totalVLines > 0 {
		t.textSurface.SetViewportSize(origVP.Width, totalVLines)
		if vp := t.textSurface.Viewport(); vp != nil {
			vp.Y = 0
		}
	}

	allVLines := t.textSurface.VirtualLines()

	// Restore the rendering viewport after extraction.
	if origVP != nil {
		t.textSurface.SetViewportSize(origVP.Width, origVP.Height)
		if vp := t.textSurface.Viewport(); vp != nil {
			vp.Y = origVP.Y
		}
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
func (t *Text) HasSelection() bool {
	sel := t.Context().Selection()
	if sel == nil || !sel.IsActive() {
		return false
	}
	localSel := render.ConvertGlobalToLocal(sel, t.X(), t.Y(), t.Width(), t.Height())
	return localSel != nil && localSel.IsActive
}

// Text-buffer conversion and wrapping helpers.

// syncContentToTextBuffer converts styled content into logical TextBuffer lines.
func (t *Text) syncContentToTextBuffer() {
	if len(t.content.Chunks) == 0 {
		t.textBuf.Clear()
		return
	}

	// Split hard newlines before TextSurface applies visual wrapping.
	hardLines := t.splitChunksIntoHardLines()
	tbLines := make([]textbuf.Line, len(hardLines))
	for i, segments := range hardLines {
		spans := make([]textbuf.Span, len(segments))
		for j, seg := range segments {
			spans[j] = textbuf.Span{
				Text:       seg.text,
				Fg:         seg.fg,
				Bg:         seg.bg,
				Attributes: seg.attributes,
			}
		}
		tbLines[i] = textbuf.Line{Spans: spans}
	}
	t.textBuf.SetLines(tbLines)
}

func (t *Text) ensureWrapped() {
	if !t.linesDirty {
		return
	}

	// Rebuild node-backed content only while no direct content has been set.
	if !t.hasManualContent && t.rootTextNode != nil && t.rootTextNode.ChildrenCount() > 0 {
		t.rebuildFromTextNodes()
	}

	t.syncContentToTextBuffer()
	t.linesDirty = false
}

// splitChunksIntoHardLines splits styled content at explicit newline boundaries.
func (t *Text) splitChunksIntoHardLines() [][]styledSegment {
	var result [][]styledSegment
	var currentLine []styledSegment

	for _, chunk := range t.content.Chunks {
		fg := t.fg
		bg := t.bg
		attrs := t.attributes

		if !chunk.Fg.IsTransparent() {
			fg = chunk.Fg
		}
		if !chunk.Bg.IsTransparent() {
			bg = chunk.Bg
		}
		// Chunk attributes add to the widget defaults.
		attrs = attrs | chunk.Attributes

		// Preserve style across every hard-line fragment of the chunk.
		parts := strings.Split(chunk.Text, "\n")
		for i, part := range parts {
			if i > 0 {
				// A newline completes the current logical line, including an empty one.
				result = append(result, currentLine)
				currentLine = nil
			}
			if len(part) > 0 {
				currentLine = append(currentLine, styledSegment{
					text:       part,
					fg:         fg,
					bg:         bg,
					attributes: attrs,
				})
			}
		}
	}

	// Retain the final empty line after a trailing newline.
	result = append(result, currentLine)
	return result
}

// segmentsToWrappedLine converts styled segments into a wrappedLine.
func segmentsToWrappedLine(segments []styledSegment) wrappedLine {
	chunks := make([]wrappedChunk, len(segments))
	for i, seg := range segments {
		chunks[i] = wrappedChunk{
			text:       seg.text,
			fg:         seg.fg,
			bg:         seg.bg,
			attributes: seg.attributes,
		}
	}
	return wrappedLine{chunks: chunks}
}

// wrapSegmentsChar wraps styled segments at grapheme-cluster boundaries.
func wrapSegmentsChar(segments []styledSegment, maxWidth int) []wrappedLine {
	var result []wrappedLine
	var currentChunks []wrappedChunk
	currentWidth := 0

	for _, seg := range segments {
		fg := seg.fg
		bg := seg.bg
		attrs := seg.attributes

		var sb strings.Builder
		remaining := seg.text
		for len(remaining) > 0 {
			cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
			remaining = rest
			if currentWidth+rw > maxWidth {
				// Finish the current styled chunk before breaking the line.
				if sb.Len() > 0 {
					currentChunks = append(currentChunks, wrappedChunk{
						text: sb.String(), fg: fg, bg: bg, attributes: attrs,
					})
					sb.Reset()
				}
				// Start a new line with the current grapheme cluster.
				result = append(result, wrappedLine{chunks: currentChunks})
				currentChunks = nil
				currentWidth = 0
			}
			sb.WriteString(cluster)
			currentWidth += rw
		}
		// Preserve any remaining segment text on the current line.
		if sb.Len() > 0 {
			currentChunks = append(currentChunks, wrappedChunk{
				text: sb.String(), fg: fg, bg: bg, attributes: attrs,
			})
		}
	}

	// Emit the final partially filled line.
	if len(currentChunks) > 0 {
		result = append(result, wrappedLine{chunks: currentChunks})
	}

	if len(result) == 0 {
		result = append(result, wrappedLine{})
	}

	return result
}

// styledWord groups text through its trailing whitespace while preserving styles.
type styledWord struct {
	parts []styledSegment
	width int
}

// wrapSegmentsWord wraps styled segments at word boundaries.
func wrapSegmentsWord(segments []styledSegment, maxWidth int) []wrappedLine {
	// Group non-space clusters with the whitespace that follows them.
	words := splitSegmentsIntoWords(segments)

	var result []wrappedLine
	var currentChunks []wrappedChunk
	currentWidth := 0

	for _, word := range words {
		wordWidth := word.width

		if currentWidth == 0 {
			// A long first word falls back to grapheme wrapping.
			if wordWidth > maxWidth {
				charWrapped := wrapSegmentsChar(word.parts, maxWidth)
				result = append(result, charWrapped...)
				continue
			}
			for _, p := range word.parts {
				currentChunks = append(currentChunks, wrappedChunk{
					text: p.text, fg: p.fg, bg: p.bg, attributes: p.attributes,
				})
			}
			currentWidth = wordWidth
			continue
		}

		if currentWidth+wordWidth > maxWidth {
			// Move this word to a fresh line.
			result = append(result, wrappedLine{chunks: currentChunks})
			currentChunks = nil
			currentWidth = 0

			// Do not carry separator whitespace onto the new line.
			trimmedParts := trimLeadingWhitespace(word.parts)
			trimmedWidth := 0
			for _, p := range trimmedParts {
				trimmedWidth += textbuf.StringWidth(p.text)
			}

			if trimmedWidth > maxWidth {
				charWrapped := wrapSegmentsChar(trimmedParts, maxWidth)
				result = append(result, charWrapped...)
				continue
			}

			for _, p := range trimmedParts {
				if len(p.text) > 0 {
					currentChunks = append(currentChunks, wrappedChunk{
						text: p.text, fg: p.fg, bg: p.bg, attributes: p.attributes,
					})
				}
			}
			currentWidth = trimmedWidth
		} else {
			for _, p := range word.parts {
				currentChunks = append(currentChunks, wrappedChunk{
					text: p.text, fg: p.fg, bg: p.bg, attributes: p.attributes,
				})
			}
			currentWidth += wordWidth
		}
	}

	if len(currentChunks) > 0 {
		result = append(result, wrappedLine{chunks: currentChunks})
	}

	if len(result) == 0 {
		result = append(result, wrappedLine{})
	}

	return result
}

// splitSegmentsIntoWords groups non-space graphemes with their trailing whitespace.
func splitSegmentsIntoWords(segments []styledSegment) []styledWord {
	// styledCluster is one indivisible display unit with its source style.
	type styledCluster struct {
		text       string
		baseRune   rune // First rune used for whitespace detection.
		width      int
		fg         style.Color
		bg         style.Color
		attributes style.Attrs
	}

	// Flatten segments so style boundaries do not affect word detection.
	var allClusters []styledCluster
	for _, seg := range segments {
		remaining := seg.text
		for len(remaining) > 0 {
			cluster, rest, cw := textbuf.NextGraphemeCluster(remaining)
			remaining = rest
			// Unicode whitespace classification needs one representative rune.
			var base rune
			for _, r := range cluster {
				base = r
				break
			}
			allClusters = append(allClusters, styledCluster{
				text: cluster, baseRune: base, width: cw,
				fg: seg.fg, bg: seg.bg, attributes: seg.attributes,
			})
		}
	}

	if len(allClusters) == 0 {
		return nil
	}

	// Include trailing whitespace in each word to preserve source spacing.
	var words []styledWord
	i := 0
	for i < len(allClusters) {
		var wordClusters []styledCluster

		// Consume the visible word body.
		for i < len(allClusters) && !unicode.IsSpace(allClusters[i].baseRune) {
			wordClusters = append(wordClusters, allClusters[i])
			i++
		}
		// Attach separator whitespace to that body.
		for i < len(allClusters) && unicode.IsSpace(allClusters[i].baseRune) {
			wordClusters = append(wordClusters, allClusters[i])
			i++
		}

		if len(wordClusters) == 0 {
			continue
		}

		// Rebuild minimal styled segments for this word.
		var parts []styledSegment
		var sb strings.Builder
		curFg := wordClusters[0].fg
		curBg := wordClusters[0].bg
		curAttrs := wordClusters[0].attributes
		width := 0

		for _, sc := range wordClusters {
			if sc.fg != curFg || sc.bg != curBg || sc.attributes != curAttrs {
				if sb.Len() > 0 {
					parts = append(parts, styledSegment{
						text: sb.String(), fg: curFg, bg: curBg, attributes: curAttrs,
					})
					sb.Reset()
				}
				curFg = sc.fg
				curBg = sc.bg
				curAttrs = sc.attributes
			}
			sb.WriteString(sc.text)
			width += sc.width
		}
		if sb.Len() > 0 {
			parts = append(parts, styledSegment{
				text: sb.String(), fg: curFg, bg: curBg, attributes: curAttrs,
			})
		}

		words = append(words, styledWord{parts: parts, width: width})
	}

	return words
}

// trimLeadingWhitespace trims leading whitespace from styled segments.
func trimLeadingWhitespace(parts []styledSegment) []styledSegment {
	if len(parts) == 0 {
		return parts
	}

	result := make([]styledSegment, len(parts))
	copy(result, parts)

	for i := range result {
		trimmed := strings.TrimLeftFunc(result[i].text, unicode.IsSpace)
		if trimmed != result[i].text {
			result[i].text = trimmed
			if len(trimmed) > 0 {
				break // Stop after reaching visible text.
			}
			// Continue through segments containing only whitespace.
		} else {
			break // The first visible segment needs no change.
		}
	}

	// Drop segments emptied by trimming.
	for len(result) > 0 && len(result[0].text) == 0 {
		result = result[1:]
	}

	return result
}

// splitWords groups each run of non-space text with the whitespace that follows it.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	inSpace := false

	for _, r := range s {
		isSpace := unicode.IsSpace(r)
		if isSpace && !inSpace {
			current.WriteRune(r)
			inSpace = true
		} else if !isSpace && inSpace {
			words = append(words, current.String())
			current.Reset()
			current.WriteRune(r)
			inSpace = false
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// OnResize updates the TextSurface viewport dimensions.
func (t *Text) OnResize(width, height int) {
	t.textSurface.SetViewportSize(width, height)
}

// Text appearance and wrapping.

// SetFg sets the default foreground and schedules content synchronization.
func (t *Text) SetFg(c style.Color) {
	t.fg = c
	t.linesDirty = true
	if t.rootTextNode != nil {
		t.rootTextNode.fg = c
	}
	t.RequestRender()
}

// SetBg sets the default background and schedules content synchronization.
func (t *Text) SetBg(c style.Color) {
	t.bg = c
	t.linesDirty = true
	if t.rootTextNode != nil {
		t.rootTextNode.bg = c
	}
	t.RequestRender()
}

// SetAttributes sets default text attributes and schedules content synchronization.
func (t *Text) SetAttributes(a style.Attrs) {
	t.attributes = a
	t.linesDirty = true
	if t.rootTextNode != nil {
		t.rootTextNode.attributes = a
	}
	t.RequestRender()
}

// SetWrapMode sets the wrap mode and schedules content synchronization.
func (t *Text) SetWrapMode(m WrapMode) {
	t.wrapMode = m
	t.linesDirty = true
	t.RequestRender()
}

// SetTruncate sets whether text is truncated with an ellipsis.
func (t *Text) SetTruncate(v bool) {
	t.truncate = v
	t.textSurface.SetTruncate(v)
	t.linesDirty = true
	t.RequestRender()
}

// Fg returns the foreground color.
func (t *Text) Fg() style.Color { return t.fg }

// Bg returns the background color.
func (t *Text) Bg() style.Color { return t.bg }

// Attributes returns the text attributes bitmask.
func (t *Text) Attributes() style.Attrs { return t.attributes }

// WrapModeValue returns the configured wrap mode.
func (t *Text) WrapModeValue() WrapMode { return t.wrapMode }

// Truncate returns whether truncation is enabled.
func (t *Text) Truncate() bool { return t.truncate }

// Text options.

// TextWithFg sets the foreground color.
func TextWithFg(c style.Color) TextOption {
	return func(t *Text) { t.fg = c }
}

// TextWithBg sets the background color.
func TextWithBg(c style.Color) TextOption {
	return func(t *Text) { t.bg = c }
}

// TextWithAttributes sets text attributes.
func TextWithAttributes(a style.Attrs) TextOption {
	return func(t *Text) { t.attributes = a }
}

// TextWithWrapMode sets the wrapping mode.
func TextWithWrapMode(m WrapMode) TextOption {
	return func(t *Text) { t.wrapMode = m }
}

// TextWithTruncate controls whether overflowing lines are truncated.
func TextWithTruncate(v bool) TextOption {
	return func(t *Text) { t.truncate = v }
}

// TextWithSelectable controls whether selections can start in the text.
func TextWithSelectable(v bool) TextOption {
	return func(t *Text) { t.selectable_ = v }
}

// TextWithSelectionBg sets the selection background color.
func TextWithSelectionBg(c style.Color) TextOption {
	return func(t *Text) { t.selectionBg = c }
}

// TextWithSelectionFg sets the selection foreground color.
func TextWithSelectionFg(c style.Color) TextOption {
	return func(t *Text) { t.selectionFg = c }
}

// TextWithID sets the widget ID.
func TextWithID(id string) TextOption {
	return func(t *Text) { t.SetID(id) }
}

// TextWithWidth sets the widget width.
func TextWithWidth(s layout.Scalar) TextOption {
	return func(t *Text) { t.LayoutNode().Width = s }
}

// TextWithHeight sets the widget height.
func TextWithHeight(s layout.Scalar) TextOption {
	return func(t *Text) { t.LayoutNode().Height = s }
}

// TextWithFlexGrow sets the flex-grow factor.
func TextWithFlexGrow(v int) TextOption {
	return func(t *Text) { t.LayoutNode().FlexGrow = v }
}

// TextWithFlexShrink sets the flex-shrink factor.
func TextWithFlexShrink(v int) TextOption {
	return func(t *Text) { t.LayoutNode().FlexShrink = v }
}

// TextWithScrollX sets the initial horizontal scroll offset.
func TextWithScrollX(v int) TextOption {
	return func(t *Text) { t.scrollX = v }
}

// TextWithScrollY sets the initial vertical scroll offset.
func TextWithScrollY(v int) TextOption {
	return func(t *Text) { t.scrollY = v }
}
