package textbuf

import (
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// TextSurface
// ---------------------------------------------------------------------------

// TextSurface provides a viewport-aware, wrap-aware, selection-aware view
// over a TextBuffer. It computes virtual lines (the result of wrapping logical
// lines) and slices them to the viewport.
type TextSurface struct {
	buf       *TextBuffer
	selection *TextSelection
	viewport  *Viewport
	wrapMode  WrapMode
	wrapWidth int // explicit wrap width (0 = use viewport width)
	truncate  bool

	// Tab indicator
	tabIndicator      rune // codepoint shown instead of tab space (0 = none)
	tabIndicatorColor *style.Color

	// Per-line backgrounds
	lineBackgrounds map[int]style.Color // logical line → bg color
	activeLine      int                 // -1 = none
	activeLineBg    style.Color

	// Cached virtual lines
	vlines      []VirtualLine
	lineInfo    LineInfo
	dirty       bool
	cachedEpoch uint64
}

// NewTextSurface creates a view over the given TextBuffer.
func NewTextSurface(buf *TextBuffer) *TextSurface {
	return &TextSurface{
		buf:        buf,
		dirty:      true,
		activeLine: -1,
	}
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// SetViewport sets the viewport. Pass nil to clear.
func (v *TextSurface) SetViewport(vp *Viewport) {
	v.viewport = vp
	v.dirty = true
}

// SetViewportSize sets/updates just the viewport dimensions, preserving offset.
func (v *TextSurface) SetViewportSize(width, height int) {
	if v.viewport != nil {
		v.viewport.Width = width
		v.viewport.Height = height
	} else {
		v.viewport = &Viewport{Width: width, Height: height}
	}
	v.dirty = true
}

// SetWrapMode sets the line wrap mode.
func (v *TextSurface) SetWrapMode(mode WrapMode) {
	if v.wrapMode != mode {
		v.wrapMode = mode
		v.dirty = true
	}
}

// SetWrapWidth sets the wrap width explicitly.
// If set to 0, the viewport width is used (default behavior).
func (v *TextSurface) SetWrapWidth(width int) {
	if v.wrapWidth != width {
		v.wrapWidth = width
		v.dirty = true
	}
}

// SetTruncate enables or disables line truncation.
// When enabled, lines longer than the viewport width are clipped.
func (v *TextSurface) SetTruncate(truncate bool) {
	v.truncate = truncate
}

// Truncate returns whether line truncation is enabled.
func (v *TextSurface) Truncate() bool {
	return v.truncate
}

// SetLineBackgrounds sets per-line background colors (logical line → RGBA).
// Pass nil to clear all line backgrounds.
func (v *TextSurface) SetLineBackgrounds(colors map[int]style.Color) {
	v.lineBackgrounds = colors
}

// SetActiveLine sets the active (highlighted) line index. Pass -1 to disable.
func (v *TextSurface) SetActiveLine(line int, bg style.Color) {
	v.activeLine = line
	v.activeLineBg = bg
}

// LineBackground returns the background color for a source (physical) line.
// Returns transparent if no background is configured for that line.
func (v *TextSurface) LineBackground(lineIdx int) style.Color {
	// Active line takes priority
	if v.activeLine >= 0 && lineIdx == v.activeLine && !v.activeLineBg.IsTransparent() {
		return v.activeLineBg
	}
	// Then per-line backgrounds
	if v.lineBackgrounds != nil {
		if bg, ok := v.lineBackgrounds[lineIdx]; ok && !bg.IsTransparent() {
			return bg
		}
	}
	return style.Color{}
}

// SetTabIndicator sets the codepoint displayed for tab characters.
// Pass 0 to disable the indicator.
func (v *TextSurface) SetTabIndicator(indicator rune) {
	v.tabIndicator = indicator
	v.dirty = true
}

// SetTabIndicatorColor sets the color for tab indicator characters.
func (v *TextSurface) SetTabIndicatorColor(color style.Color) {
	v.tabIndicatorColor = &color
}

// SetSelection sets the selection range.
func (v *TextSurface) SetSelection(sel *TextSelection) {
	v.selection = sel
}

// ResetSelection clears the selection.
func (v *TextSurface) ResetSelection() {
	v.selection = nil
}

// Selection returns the current selection or nil.
func (v *TextSurface) Selection() *TextSelection {
	return v.selection
}

// Buffer returns the underlying TextBuffer.
func (v *TextSurface) Buffer() *TextBuffer {
	return v.buf
}

// ---------------------------------------------------------------------------
// Surface implementation
// ---------------------------------------------------------------------------

// VirtualLines returns the visible virtual lines.
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

// Viewport returns the current viewport or nil.
func (v *TextSurface) Viewport() *Viewport {
	return v.viewport
}

// CachedLineInfo returns layout metadata for visible lines.
// FullLineInfo returns the complete (unsliced) line info for all virtual
// lines, regardless of viewport. Use this for global visual-to-logical
// mappings (e.g. LogicalLineForVisualLine).
func (v *TextSurface) FullLineInfo() LineInfo {
	v.ensureUpdated()
	return v.lineInfo
}

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

// LineHighlights returns the highlights for a source (physical) line.
func (v *TextSurface) LineHighlights(lineIdx int) []Highlight {
	return v.buf.LineHighlights(lineIdx)
}

// VirtualLineCount returns the total number of virtual lines.
func (v *TextSurface) VirtualLineCount() int {
	v.ensureUpdated()
	return len(v.vlines)
}

// ---------------------------------------------------------------------------
// Logical line info
// ---------------------------------------------------------------------------

// LogicalLineInfo returns line info based on logical (unwrapped) lines,
// regardless of wrap mode. Each entry corresponds to one source line.
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

// ---------------------------------------------------------------------------
// Local (visual-coordinate) selection
// ---------------------------------------------------------------------------

// SetLocalSelection sets a selection by visual (viewport-relative) coordinates.
// Returns true if a valid selection was established.
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

// UpdateSelection updates the selection end offset.
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

// UpdateLocalSelection updates the selection by visual coordinates.
// Returns true if valid.
func (v *TextSurface) UpdateLocalSelection(anchorX, anchorY, focusX, focusY int, bg, fg *style.Color) bool {
	return v.SetLocalSelection(anchorX, anchorY, focusX, focusY, bg, fg)
}

// ResetLocalSelection clears the local selection (same as ResetSelection).
func (v *TextSurface) ResetLocalSelection() {
	v.selection = nil
}

// ---------------------------------------------------------------------------
// Text extraction
// ---------------------------------------------------------------------------

// SelectedText returns the selected text, or "" if no selection.
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

// PlainText returns the full plain text of the underlying buffer.
func (v *TextSurface) PlainText() string {
	return v.buf.PlainText()
}

// ---------------------------------------------------------------------------
// Measurement
// ---------------------------------------------------------------------------

// MeasureDimensions represents the result of MeasureForDimensions.
type MeasureDimensions struct {
	LineCount int
	MaxWidth  int
}

// MeasureForDimensions computes the layout that would result from the given
// dimensions without actually changing the viewport. It uses the same virtual
// line computation as the rendering path (calculateVirtualLines) so that
// measurement and rendering always agree on line counts and widths.
func (v *TextSurface) MeasureForDimensions(width, height int) *MeasureDimensions {
	wrapWidth := 0
	if v.wrapMode != WrapNone && width > 0 {
		wrapWidth = width
		if v.wrapWidth > 0 {
			wrapWidth = v.wrapWidth
		}
	}

	// Use the same wrapping path as updateVirtualLines.
	vlines, info := v.calculateVirtualLines(wrapWidth)

	return &MeasureDimensions{LineCount: len(vlines), MaxWidth: info.MaxWidth}
}

// ---------------------------------------------------------------------------
// Internal: visual-to-offset conversion
// ---------------------------------------------------------------------------

// visualToCharOffset converts viewport-relative visual coordinates to a
// global character offset.
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

	// Walk through chunks by grapheme cluster to find the column offset
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

// ---------------------------------------------------------------------------
// Virtual line computation
// ---------------------------------------------------------------------------

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

// calculateVirtualLines is the single source of truth for virtual line
// computation. Both updateVirtualLines (for rendering) and
// MeasureForDimensions (for layout) call this so they always agree.
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

// buildUnwrappedVLine creates a single virtual line from a logical line.
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

// ---------------------------------------------------------------------------
// styledRune is a flattened rune with style info, used during wrapping.
// ---------------------------------------------------------------------------

type styledRune struct {
	r    rune
	w    int // display width
	fg   style.Color
	bg   style.Color
	attr uint32
}

// flattenSpans converts a Line's spans into a flat []styledRune stream,
// expanding tab characters into spaces (with optional tab indicator glyph).
// This is the single place where tab handling occurs for all virtual line paths.
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
			// Decode the base rune for the styledRune; store the full
			// cluster width so wrapping uses grapheme-aware widths.
			r, _ := utf8.DecodeRuneInString(cluster)
			runes = append(runes, styledRune{r: r, w: cw, fg: fg, bg: bg, attr: span.Attributes})
			// Append remaining codepoints of the cluster as zero-width
			// so they are included in the text output but don't affect
			// column arithmetic.
			for j := utf8.RuneLen(r); j < len(cluster); {
				cr, sz := utf8.DecodeRuneInString(cluster[j:])
				j += sz
				runes = append(runes, styledRune{r: cr, w: 0, fg: fg, bg: bg, attr: span.Attributes})
			}
		}
	}
	return runes
}

// wrapLine wraps a single logical line into multiple virtual lines.
func (v *TextSurface) wrapLineInto(
	line Line, lineIdx, wrapWidth int,
	globalCharOffset *int,
	vlines *[]VirtualLine,
	starts, widths, sources *[]int,
) {
	runes := v.flattenSpans(line)

	if len(runes) == 0 {
		// Empty line still produces one virtual line.
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

		// Determine how many runes fit on this virtual line.
		endIdx, lineWidth := v.fitRunesOnLine(runes, runeIdx, wrapWidth)

		// Handle case where nothing fits (single wide char exceeds wrapWidth).
		if endIdx == runeIdx && runeIdx < len(runes) {
			endIdx = runeIdx + 1
			lineWidth = runes[runeIdx].w
		}

		// Build chunks from runes[runeIdx..endIdx], grouping by style.
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

// fitRunesOnLine determines how many runes from runes[start:] fit within
// wrapWidth columns. For WrapWord mode, it tries to break at word boundaries.
// Returns (endIdx, totalWidth).
func (v *TextSurface) fitRunesOnLine(runes []styledRune, start, wrapWidth int) (int, int) {
	width := 0
	lastBreakIdx := -1 // index AFTER the last word-break rune
	lastBreakWidth := 0

	for i := start; i < len(runes); i++ {
		rw := runes[i].w
		if width+rw > wrapWidth {
			// Won't fit. For word wrap, try to break at last boundary.
			if v.wrapMode == WrapWord && lastBreakIdx > start {
				return lastBreakIdx, lastBreakWidth
			}
			return i, width
		}
		width += rw

		// Track word break positions (break AFTER space/punct).
		if v.wrapMode == WrapWord && isWordBreak(runes[i].r) {
			lastBreakIdx = i + 1
			lastBreakWidth = width
		}
	}

	// Everything fits.
	return len(runes), width
}

// buildChunksFromRunes groups consecutive runes with the same style into chunks.
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

		// Build chunk from runes[chunkStart..i]
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

// isWordBreak returns true if the rune is a word break character.
// This uses an explicit set of characters rather than the broad
// unicode.IsPunct category, which would break at quotes, asterisks, etc.
// and produce ugly mid-token wraps.
func isWordBreak(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '-', '/', '\\', // dashes & slashes
		'.', ',', ';', ':', '!', '?', // punctuation
		'(', ')', '[', ']', '{', '}': // brackets
		return true
	}
	return false
}
