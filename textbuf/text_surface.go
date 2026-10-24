package textbuf

import (
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/style"
)

// TextSurface presents a TextBuffer as wrapped virtual lines within an optional
// viewport and selection.
type TextSurface struct {
	buf       *TextBuffer
	selection *TextSelection
	viewport  *Viewport
	wrapMode  WrapMode
	wrapWidth int // Explicit wrap width; zero uses viewport width.
	truncate  bool

	// Configured tab indicator.
	tabIndicator      rune // Zero disables the indicator.
	tabIndicatorColor *style.Color

	// Per-line backgrounds.
	lineBackgrounds map[int]style.Color // Logical line to background color.
	activeLine      int                 // Negative means no active line.
	activeLineBg    style.Color

	// Cached virtual-line layout.
	vlines      []VirtualLine
	lineInfo    LineInfo
	dirty       bool
	cachedEpoch uint64
}

// NewTextSurface returns a dirty view over buf with no active line.
func NewTextSurface(buf *TextBuffer) *TextSurface {
	return &TextSurface{
		buf:        buf,
		dirty:      true,
		activeLine: -1,
	}
}

// SetViewport sets vp and invalidates virtual-line layout. Nil removes the
// viewport.
func (v *TextSurface) SetViewport(vp *Viewport) {
	v.viewport = vp
	v.dirty = true
}

// SetViewportSize changes viewport dimensions while preserving its offset.
func (v *TextSurface) SetViewportSize(width, height int) {
	if v.viewport != nil {
		v.viewport.Width = width
		v.viewport.Height = height
	} else {
		v.viewport = &Viewport{Width: width, Height: height}
	}
	v.dirty = true
}

// SetWrapMode sets the line-wrapping mode.
func (v *TextSurface) SetWrapMode(mode WrapMode) {
	if v.wrapMode != mode {
		v.wrapMode = mode
		v.dirty = true
	}
}

// SetWrapWidth sets an explicit wrap width. Zero uses viewport width.
func (v *TextSurface) SetWrapWidth(width int) {
	if v.wrapWidth != width {
		v.wrapWidth = width
		v.dirty = true
	}
}

// SetTruncate controls whether rendering replaces viewport overflow with an
// ellipsis.
func (v *TextSurface) SetTruncate(truncate bool) {
	v.truncate = truncate
}

// Truncate reports whether line truncation is enabled.
func (v *TextSurface) Truncate() bool {
	return v.truncate
}

// SetLineBackgrounds retains colors as logical-line backgrounds. Nil clears
// them.
func (v *TextSurface) SetLineBackgrounds(colors map[int]style.Color) {
	v.lineBackgrounds = colors
}

// SetActiveLine sets a logical line background that overrides other line
// backgrounds. A negative line disables it.
func (v *TextSurface) SetActiveLine(line int, bg style.Color) {
	v.activeLine = line
	v.activeLineBg = bg
}

// LineBackground returns the active or configured background for a logical
// source line, or a transparent color.
func (v *TextSurface) LineBackground(lineIdx int) style.Color {
	// The active-line color has highest priority.
	if v.activeLine >= 0 && lineIdx == v.activeLine && !v.activeLineBg.IsTransparent() {
		return v.activeLineBg
	}
	// Fall back to the per-line map.
	if v.lineBackgrounds != nil {
		if bg, ok := v.lineBackgrounds[lineIdx]; ok && !bg.IsTransparent() {
			return bg
		}
	}
	return style.Color{}
}

// SetTabIndicator stores the configured tab indicator. Zero disables it.
func (v *TextSurface) SetTabIndicator(indicator rune) {
	v.tabIndicator = indicator
	v.dirty = true
}

// SetTabIndicatorColor stores the configured tab-indicator color.
func (v *TextSurface) SetTabIndicatorColor(color style.Color) {
	v.tabIndicatorColor = &color
}

// SetSelection sets the global display-width selection range.
func (v *TextSurface) SetSelection(sel *TextSelection) {
	v.selection = sel
}

// ResetSelection removes the current selection.
func (v *TextSurface) ResetSelection() {
	v.selection = nil
}

// Selection returns the current selection, or nil when absent.
func (v *TextSurface) Selection() *TextSelection {
	return v.selection
}

// Buffer returns the underlying text buffer.
func (v *TextSurface) Buffer() *TextBuffer {
	return v.buf
}

// VirtualLines returns virtual lines within the vertical viewport.
func (v *TextSurface) VirtualLines() []VirtualLine {
	v.ensureUpdated()
	if v.viewport != nil {
		start := max(v.viewport.Y, 0)
		if start >= len(v.vlines) {
			return nil
		}
		end := min(start+v.viewport.Height, len(v.vlines))
		return v.vlines[start:end]
	}
	return v.vlines
}

// Viewport returns the current viewport, or nil when unset.
func (v *TextSurface) Viewport() *Viewport {
	return v.viewport
}

// FullLineInfo returns metadata for all virtual lines, ignoring the viewport.
func (v *TextSurface) FullLineInfo() LineInfo {
	v.ensureUpdated()
	return v.lineInfo
}

// CachedLineInfo returns metadata sliced to the vertical viewport.
func (v *TextSurface) CachedLineInfo() LineInfo {
	v.ensureUpdated()
	if v.viewport != nil {
		start := max(v.viewport.Y, 0)
		if start >= len(v.lineInfo.Starts) {
			return LineInfo{}
		}
		end := min(start+v.viewport.Height, len(v.lineInfo.Starts))
		maxW := 0
		widthSlice := v.lineInfo.Widths[start:end]
		for _, w := range widthSlice {
			if w > maxW {
				maxW = w
			}
		}
		return LineInfo{
			Starts:   v.lineInfo.Starts[start:end],
			Widths:   widthSlice,
			Sources:  v.lineInfo.Sources[start:end],
			MaxWidth: maxW,
		}
	}
	return v.lineInfo
}

// DefaultFg returns the default foreground color.
func (v *TextSurface) DefaultFg() style.Color { return v.buf.DefaultFg() }

// DefaultBg returns the default background color.
func (v *TextSurface) DefaultBg() style.Color { return v.buf.DefaultBg() }

// LineHighlights returns highlights for a logical source line.
func (v *TextSurface) LineHighlights(lineIdx int) []Highlight {
	return v.buf.LineHighlights(lineIdx)
}

// VirtualLineCount returns the total number of virtual lines before viewport
// slicing.
func (v *TextSurface) VirtualLineCount() int {
	v.ensureUpdated()
	return len(v.vlines)
}

// LogicalLineInfo returns metadata for unwrapped logical lines.
func (v *TextSurface) LogicalLineInfo() LineInfo {
	var starts, widths, sources []int
	maxW := 0
	pos := 0
	for i := 0; i < v.buf.LineCount(); i++ {
		if i > 0 {
			pos++
		}
		w := v.buf.LineWidthAt(i)
		starts = append(starts, pos)
		widths = append(widths, w)
		sources = append(sources, i)
		pos += w
		if w > maxW {
			maxW = w
		}
	}
	return LineInfo{Starts: starts, Widths: widths, Sources: sources, MaxWidth: maxW}
}

// SetLocalSelection maps viewport-relative endpoints to a global selection and
// reports whether both coordinates were valid.
func (v *TextSurface) SetLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg *style.Color) bool {
	startOff := v.visualToCharOffset(anchorX, anchorY)
	endOff := v.visualToCharOffset(focusX, focusY)
	if startOff < 0 || endOff < 0 {
		return false
	}
	if startOff > endOff {
		startOff, endOff = endOff, startOff
	}
	v.selection = &TextSelection{Start: startOff, End: endOff, BgColor: bg, FgColor: fg}
	return true
}

// UpdateSelection sets the selection end offset and replaces non-nil colors.
func (v *TextSurface) UpdateSelection(end int, bg, fg *style.Color) {
	if v.selection == nil {
		v.selection = &TextSelection{Start: end, End: end, BgColor: bg, FgColor: fg}
		return
	}
	v.selection.End = end
	if bg != nil {
		v.selection.BgColor = bg
	}
	if fg != nil {
		v.selection.FgColor = fg
	}
}

// UpdateLocalSelection replaces the selection from viewport-relative endpoints.
func (v *TextSurface) UpdateLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg *style.Color) bool {
	return v.SetLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
}

// ResetLocalSelection removes the current selection.
func (v *TextSurface) ResetLocalSelection() {
	v.selection = nil
}

// SelectedText returns text in the normalized selection range, or an empty
// string when no selection exists.
func (v *TextSurface) SelectedText() string {
	if v.selection == nil {
		return ""
	}
	start, end := v.selection.Start, v.selection.End
	if start > end {
		start, end = end, start
	}
	return v.buf.TextRange(start, end)
}

// PlainText returns all unstyled buffer content.
func (v *TextSurface) PlainText() string {
	return v.buf.PlainText()
}

// MeasureDimensions contains virtual-line count and maximum width.
type MeasureDimensions struct {
	// LineCount is the number of virtual lines.
	LineCount int
	// MaxWidth is the display width of the widest virtual line.
	MaxWidth int
}

// MeasureForDimensions calculates virtual-line dimensions without changing the
// viewport. width constrains wrapping; height is currently unused.
func (v *TextSurface) MeasureForDimensions(width, height int) *MeasureDimensions {
	wrapWidth := 0
	if v.wrapMode != WrapNone && width > 0 {
		wrapWidth = width
		if v.wrapWidth > 0 {
			wrapWidth = v.wrapWidth
		}
	}

	// Share virtual-line construction with rendering.
	vlines, info := v.calculateVirtualLines(wrapWidth)

	return &MeasureDimensions{LineCount: len(vlines), MaxWidth: info.MaxWidth}
}

// visualToCharOffset maps viewport-relative coordinates to a global
// display-width offset, or -1 for an invalid row.
func (v *TextSurface) visualToCharOffset(x, y int) int {
	v.ensureUpdated()

	vpY := 0
	if v.viewport != nil {
		vpY = v.viewport.Y
	}

	absY := y + vpY
	if absY < 0 || absY >= len(v.vlines) {
		return -1
	}

	vl := v.vlines[absY]
	offset := vl.CharOffset

	// Advance through complete grapheme clusters to x.
	col := 0
	for _, chunk := range vl.Chunks {
		remaining := chunk.Text
		for len(remaining) > 0 {
			if col >= x {
				return offset
			}
			cluster, rest, cw := NextGraphemeCluster(remaining)
			_ = cluster
			remaining = rest
			col += cw
			offset += cw
		}
	}
	return offset
}

func (v *TextSurface) ensureUpdated() {
	bufEpoch := v.buf.Epoch()
	if !v.dirty && v.cachedEpoch == bufEpoch {
		return
	}
	v.updateVirtualLines()
	v.dirty = false
	v.cachedEpoch = bufEpoch
}

func (v *TextSurface) updateVirtualLines() {
	wrapWidth := 0
	if v.wrapMode != WrapNone {
		if v.wrapWidth > 0 {
			wrapWidth = v.wrapWidth
		} else if v.viewport != nil {
			wrapWidth = v.viewport.Width
		}
	}

	v.vlines, v.lineInfo = v.calculateVirtualLines(wrapWidth)
}

// calculateVirtualLines builds wrapped lines and their parallel metadata.
func (v *TextSurface) calculateVirtualLines(wrapWidth int) ([]VirtualLine, LineInfo) {
	var vlines []VirtualLine
	var starts, widths, sources []int
	globalCharOffset := 0

	for lineIdx := 0; lineIdx < v.buf.LineCount(); lineIdx++ {
		line := v.buf.LineAt(lineIdx)

		if lineIdx > 0 {
			globalCharOffset++
		}

		if wrapWidth > 0 {
			v.wrapLineInto(line, lineIdx, wrapWidth, &globalCharOffset, &vlines, &starts, &widths, &sources)
		} else {
			vl := v.buildUnwrappedVLine(line, lineIdx, globalCharOffset)
			vlines = append(vlines, vl)
			starts = append(starts, globalCharOffset)
			widths = append(widths, vl.Width)
			sources = append(sources, lineIdx)
			globalCharOffset += vl.Width
		}
	}

	maxW := 0
	for _, w := range widths {
		if w > maxW {
			maxW = w
		}
	}

	info := LineInfo{
		Starts:   starts,
		Widths:   widths,
		Sources:  sources,
		MaxWidth: maxW,
	}
	return vlines, info
}

// buildUnwrappedVLine converts one logical line without wrapping it.
func (v *TextSurface) buildUnwrappedVLine(line Line, lineIdx, charOffset int) VirtualLine {
	runes := v.flattenSpans(line)
	if len(runes) == 0 {
		return VirtualLine{
			CharOffset: charOffset,
			SourceLine: lineIdx,
		}
	}
	chunks := buildChunksFromRunes(runes, 0)
	totalWidth := 0
	for _, ch := range chunks {
		totalWidth += ch.Width
	}
	return VirtualLine{
		Chunks:     chunks,
		Width:      totalWidth,
		CharOffset: charOffset,
		SourceLine: lineIdx,
	}
}

// styledRune stores one code point and its cluster-head width during wrapping.
type styledRune struct {
	r    rune
	w    int // Display width, or zero for trailing cluster code points.
	fg   style.Color
	bg   style.Color
	attr style.Attrs
}

// flattenSpans expands a line into styled code points while recording display
// width only on each grapheme cluster's first code point.
func (v *TextSurface) flattenSpans(line Line) []styledRune {
	var runes []styledRune

	for _, span := range line.Spans {
		fg := span.Fg
		bg := span.Bg
		if fg == (style.Color{}) {
			fg = v.buf.defaultFg
		}
		remaining := span.Text
		for len(remaining) > 0 {
			cluster, rest, cw := NextGraphemeCluster(remaining)
			remaining = rest
			// Store cluster width on its first code point for wrapping.
			r, _ := utf8.DecodeRuneInString(cluster)
			runes = append(runes, styledRune{r: r, w: cw, fg: fg, bg: bg, attr: span.Attributes})
			// Preserve trailing code points without counting their width again.
			for j := utf8.RuneLen(r); j < len(cluster); {
				cr, sz := utf8.DecodeRuneInString(cluster[j:])
				j += sz
				runes = append(runes, styledRune{r: cr, w: 0, fg: fg, bg: bg, attr: span.Attributes})
			}
		}
	}
	return runes
}

// wrapLineInto appends one logical line as one or more virtual lines.
func (v *TextSurface) wrapLineInto(
	line Line, lineIdx, wrapWidth int,
	globalCharOffset *int,
	vlines *[]VirtualLine,
	starts, widths, sources *[]int,
) {
	runes := v.flattenSpans(line)

	if len(runes) == 0 {
		// Preserve an empty logical line as one empty visual row.
		vl := VirtualLine{
			CharOffset: *globalCharOffset,
			SourceLine: lineIdx,
		}
		*vlines = append(*vlines, vl)
		*starts = append(*starts, *globalCharOffset)
		*widths = append(*widths, 0)
		*sources = append(*sources, lineIdx)
		return
	}

	colOffset := 0
	runeIdx := 0

	for runeIdx < len(runes) {
		lineStart := *globalCharOffset
		lineColOffset := colOffset

		// Find the next segment that fits this visual row.
		endIdx, lineWidth := v.fitRunesOnLine(runes, runeIdx, wrapWidth)

		// Keep an oversized cluster intact on its own row.
		if endIdx == runeIdx && runeIdx < len(runes) {
			endIdx = runeIdx + 1
			lineWidth = runes[runeIdx].w
		}

		// Regroup the selected code points into styled chunks.
		chunks := buildChunksFromRunes(runes[runeIdx:endIdx], colOffset-lineColOffset)

		vl := VirtualLine{
			Chunks:          chunks,
			Width:           lineWidth,
			CharOffset:      lineStart,
			SourceLine:      lineIdx,
			SourceColOffset: lineColOffset,
		}
		*vlines = append(*vlines, vl)
		*starts = append(*starts, lineStart)
		*widths = append(*widths, lineWidth)
		*sources = append(*sources, lineIdx)

		colOffset += lineWidth
		*globalCharOffset += lineWidth
		runeIdx = endIdx
	}
}

// fitRunesOnLine returns the exclusive code-point index and display width that
// fit from start. WrapWord prefers the latest word boundary.
func (v *TextSurface) fitRunesOnLine(runes []styledRune, start, wrapWidth int) (int, int) {
	width := 0
	lastBreakIdx := -1 // Exclusive index after the latest word break.
	lastBreakWidth := 0

	for i := start; i < len(runes); i++ {
		rw := runes[i].w
		if width+rw > wrapWidth {
			// Prefer a prior word boundary before splitting at the width limit.
			if v.wrapMode == WrapWord && lastBreakIdx > start {
				return lastBreakIdx, lastBreakWidth
			}
			return i, width
		}
		width += rw

		// Word wrapping breaks after recognized spaces or punctuation.
		if v.wrapMode == WrapWord && isWordBreak(runes[i].r) {
			lastBreakIdx = i + 1
			lastBreakWidth = width
		}
	}

	// The remaining code points fit on one row.
	return len(runes), width
}

// buildChunksFromRunes groups adjacent code points with equal styles.
func buildChunksFromRunes(runes []styledRune, graphemeStartBase int) []VirtualChunk {
	if len(runes) == 0 {
		return nil
	}

	var chunks []VirtualChunk
	chunkStart := 0
	colOffset := 0

	for i := 1; i <= len(runes); i++ {
		if i < len(runes) &&
			runes[i].fg == runes[chunkStart].fg &&
			runes[i].bg == runes[chunkStart].bg &&
			runes[i].attr == runes[chunkStart].attr {
			continue
		}

		// Materialize the completed half-open code-point range.
		var text []rune
		w := 0
		for j := chunkStart; j < i; j++ {
			text = append(text, runes[j].r)
			w += runes[j].w
		}
		chunks = append(chunks, VirtualChunk{
			Text:          string(text),
			GraphemeStart: graphemeStartBase + colOffset,
			Width:         w,
			Fg:            runes[chunkStart].fg,
			Bg:            runes[chunkStart].bg,
			Attributes:    runes[chunkStart].attr,
		})
		colOffset += w
		chunkStart = i
	}
	return chunks
}

// isWordBreak reports whether wrapping can break after r. It uses a limited
// punctuation set to avoid splitting punctuation-heavy tokens too aggressively.
func isWordBreak(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '-', '/', '\\', // Dashes and slashes.
		'.', ',', ';', ':', '!', '?', // Sentence punctuation.
		'(', ')', '[', ']', '{', '}': // Brackets.
		return true
	}
	return false
}
