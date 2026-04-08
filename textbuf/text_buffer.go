package textbuf

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Line
// ---------------------------------------------------------------------------

// Line is one logical line stored as a sequence of spans.
type Line struct {
	Spans []Span
}

// Text returns the concatenated plain text of the line.
func (sl *Line) Text() string {
	if len(sl.Spans) == 1 {
		return sl.Spans[0].Text
	}
	var sb strings.Builder
	for _, sp := range sl.Spans {
		sb.WriteString(sp.Text)
	}
	return sb.String()
}

// Width returns the display width of the line using grapheme-cluster-aware
// width calculation.
func (sl *Line) Width() int {
	w := 0
	for _, sp := range sl.Spans {
		w += StringWidth(sp.Text)
	}
	return w
}

// ---------------------------------------------------------------------------
// TextBuffer
// ---------------------------------------------------------------------------

// TextBuffer is a styled text content store. It stores text as a list of
// logical lines, each composed of spans.
type TextBuffer struct {
	lines      []Line
	defaultFg  style.Color
	defaultBg  style.Color
	defaultAt  uint32
	tabWidth   int
	expandTabs bool   // when true, replace \t with spaces on input
	epoch      uint64 // incremented on every content change

	// Highlight support (per-line highlights)
	highlights map[int][]Highlight

	// Syntax highlighting
	syntaxStyle *style.SyntaxStyle
}

// Highlight describes a colored range on a single line.
type Highlight struct {
	Start, End int // column offsets within the line
	Fg         *style.Color
	Bg         *style.Color
	Attributes uint32
	Ref        int // user-provided reference for removal
}

// NewTextBuffer creates a new empty TextBuffer.
func NewTextBuffer() *TextBuffer {
	return &TextBuffer{
		lines:     []Line{{Spans: []Span{{Text: ""}}}},
		defaultFg: style.ColorWhite,
		defaultBg: style.ColorTransparent,
		tabWidth:  4,
	}
}

// ---------------------------------------------------------------------------
// Length / Size
// ---------------------------------------------------------------------------

// Length returns the total character (rune) count of the buffer.
func (tb *TextBuffer) Length() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // newline
		}
		total += utf8.RuneCountInString(line.Text())
	}
	return total
}

// DisplayWidthLength returns the total display-width length of the buffer,
// counting each character by its terminal display width and each newline as 1.
// This is the same coordinate space used by EditBuffer.PositionToOffset.
func (tb *TextBuffer) DisplayWidthLength() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // newline
		}
		total += line.Width()
	}
	return total
}

// ByteSize returns the total byte size of the buffer content.
func (tb *TextBuffer) ByteSize() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // newline byte
		}
		total += len(line.Text())
	}
	return total
}

// ---------------------------------------------------------------------------
// Content methods
// ---------------------------------------------------------------------------

// Reset clears all content, highlights, and resets defaults to initial state.
func (tb *TextBuffer) Reset() {
	tb.lines = []Line{{Spans: []Span{{Text: ""}}}}
	tb.defaultFg = style.ColorWhite
	tb.defaultBg = style.ColorTransparent
	tb.defaultAt = 0
	tb.tabWidth = 4
	tb.highlights = nil
	tb.epoch++
}

// LoadFile reads the file at the given path and replaces the buffer content.
// Returns true on success, false on error.
func (tb *TextBuffer) LoadFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	tb.SetText(string(data))
	return true
}

// ResetDefaults resets the default fg, bg, and attributes to their initial values.
func (tb *TextBuffer) ResetDefaults() {
	tb.defaultFg = style.ColorWhite
	tb.defaultBg = style.ColorTransparent
	tb.defaultAt = 0
}

// SetText replaces the entire content with plain text using default colors.
func (tb *TextBuffer) SetText(text string) {
	text = tb.sanitizeText(text)
	rawLines := strings.Split(text, "\n")
	tb.lines = make([]Line, len(rawLines))
	for i, line := range rawLines {
		tb.lines[i] = Line{
			Spans: []Span{{
				Text: line,
				Fg:   tb.defaultFg,
				Bg:   tb.defaultBg,
			}},
		}
	}
	tb.epoch++
	tb.highlights = nil
}

// SetSpans replaces the entire content with the given spans.
// Newlines within span text cause line breaks.
func (tb *TextBuffer) SetSpans(spans []Span) {
	tb.lines = nil
	current := Line{}

	for _, span := range spans {
		parts := strings.Split(span.Text, "\n")
		for i, part := range parts {
			s := Span{
				Text:       tb.sanitizeText(part),
				Fg:         span.Fg,
				Bg:         span.Bg,
				Attributes: span.Attributes,
			}
			if s.Fg == (style.Color{}) {
				s.Fg = tb.defaultFg
			}
			current.Spans = append(current.Spans, s)
			if i < len(parts)-1 {
				tb.lines = append(tb.lines, current)
				current = Line{}
			}
		}
	}
	tb.lines = append(tb.lines, current)
	tb.epoch++
	tb.highlights = nil
}

// Append appends plain text to the end of the buffer.
func (tb *TextBuffer) Append(text string) {
	parts := strings.Split(text, "\n")
	if len(tb.lines) == 0 {
		tb.lines = []Line{{Spans: []Span{{Text: ""}}}}
	}

	// Append first part to last line
	last := &tb.lines[len(tb.lines)-1]
	if len(last.Spans) == 0 {
		last.Spans = append(last.Spans, Span{Fg: tb.defaultFg, Bg: tb.defaultBg})
	}
	last.Spans[len(last.Spans)-1].Text += parts[0]

	// Remaining parts become new lines
	for _, part := range parts[1:] {
		tb.lines = append(tb.lines, Line{
			Spans: []Span{{
				Text: part,
				Fg:   tb.defaultFg,
				Bg:   tb.defaultBg,
			}},
		})
	}
	tb.epoch++
}

// Clear removes all content.
func (tb *TextBuffer) Clear() {
	tb.lines = []Line{{Spans: []Span{{Text: ""}}}}
	tb.epoch++
	tb.highlights = nil
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// LineCount returns the number of logical lines.
func (tb *TextBuffer) LineCount() int { return len(tb.lines) }

// LineAt returns the Line at the given row, or an empty line if out of range.
func (tb *TextBuffer) LineAt(row int) Line {
	if row < 0 || row >= len(tb.lines) {
		return Line{Spans: []Span{{Text: ""}}}
	}
	return tb.lines[row]
}

// LineWidthAt returns the display width of the line at row,
// accounting for the buffer's tab width setting.
func (tb *TextBuffer) LineWidthAt(row int) int {
	if row < 0 || row >= len(tb.lines) {
		return 0
	}
	return tb.lines[row].Width()
}

// LineTextAt returns the plain text of the line at row.
func (tb *TextBuffer) LineTextAt(row int) string {
	if row < 0 || row >= len(tb.lines) {
		return ""
	}
	return tb.lines[row].Text()
}

// PlainText returns the entire content as a plain string.
func (tb *TextBuffer) PlainText() string {
	var sb strings.Builder
	for i, line := range tb.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(line.Text())
	}
	return sb.String()
}

// Epoch returns the content epoch (incremented on every mutation).
func (tb *TextBuffer) Epoch() uint64 { return tb.epoch }

// DefaultFg returns the default foreground color.
func (tb *TextBuffer) DefaultFg() style.Color { return tb.defaultFg }

// DefaultBg returns the default background color.
func (tb *TextBuffer) DefaultBg() style.Color { return tb.defaultBg }

// DefaultAttributes returns the default text attributes.
func (tb *TextBuffer) DefaultAttributes() uint32 { return tb.defaultAt }

// SetDefaultFg sets the default foreground color.
func (tb *TextBuffer) SetDefaultFg(fg style.Color) { tb.defaultFg = fg }

// SetDefaultBg sets the default background color.
func (tb *TextBuffer) SetDefaultBg(bg style.Color) { tb.defaultBg = bg }

// SetDefaultAttributes sets the default text attributes.
func (tb *TextBuffer) SetDefaultAttributes(attrs uint32) { tb.defaultAt = attrs }

// TabWidth returns the tab stop width.
func (tb *TextBuffer) TabWidth() int { return tb.tabWidth }

// SetTabWidth sets the tab stop width. Bumps the epoch so that all
// attached views recompute virtual lines with the new width.
func (tb *TextBuffer) SetTabWidth(w int) {
	if tb.tabWidth == w {
		return
	}
	tb.tabWidth = w
	tb.epoch++
}

// ExpandTabs returns whether tab characters are expanded to spaces on input.
func (tb *TextBuffer) ExpandTabs() bool { return tb.expandTabs }

// SetExpandTabs enables or disables automatic tab-to-spaces expansion.
// When enabled, all content entering the buffer via SetText, SetLines,
// InsertLine, etc. has \t replaced with tabWidth spaces.
func (tb *TextBuffer) SetExpandTabs(v bool) { tb.expandTabs = v }

// SyntaxStyle returns the attached SyntaxStyle registry, or nil.
func (tb *TextBuffer) SyntaxStyle() *style.SyntaxStyle { return tb.syntaxStyle }

// SetSyntaxStyle attaches a SyntaxStyle registry for token coloring.
// Pass nil to detach.
func (tb *TextBuffer) SetSyntaxStyle(ss *style.SyntaxStyle) { tb.syntaxStyle = ss }

// Lines returns the internal lines slice (for direct access by views).
func (tb *TextBuffer) Lines() []Line { return tb.lines }

// SetLines replaces the internal lines (used by EditBuffer).
func (tb *TextBuffer) SetLines(lines []Line) {
	if tb.expandTabs {
		for i := range lines {
			lines[i] = tb.sanitizeLine(lines[i])
		}
	}
	tb.lines = lines
	tb.epoch++
}

// ---------------------------------------------------------------------------
// Highlight support
// ---------------------------------------------------------------------------

// AddHighlight adds a highlight to the given line.
func (tb *TextBuffer) AddHighlight(lineIdx int, hl Highlight) {
	if tb.highlights == nil {
		tb.highlights = make(map[int][]Highlight)
	}
	tb.highlights[lineIdx] = append(tb.highlights[lineIdx], hl)
}

// RemoveHighlightsByRef removes all highlights with the given ref.
func (tb *TextBuffer) RemoveHighlightsByRef(ref int) {
	for lineIdx, hls := range tb.highlights {
		filtered := hls[:0]
		for _, hl := range hls {
			if hl.Ref != ref {
				filtered = append(filtered, hl)
			}
		}
		if len(filtered) == 0 {
			delete(tb.highlights, lineIdx)
		} else {
			tb.highlights[lineIdx] = filtered
		}
	}
}

// ClearLineHighlights removes all highlights on the given line.
func (tb *TextBuffer) ClearLineHighlights(lineIdx int) {
	delete(tb.highlights, lineIdx)
}

// ClearAllHighlights removes all highlights.
func (tb *TextBuffer) ClearAllHighlights() {
	tb.highlights = nil
}

// LineHighlights returns the highlights for a specific line.
func (tb *TextBuffer) LineHighlights(lineIdx int) []Highlight {
	if tb.highlights == nil {
		return nil
	}
	return tb.highlights[lineIdx]
}

// ---------------------------------------------------------------------------
// Text range access
// ---------------------------------------------------------------------------

// TextRange returns text between start and end character offsets.
// Offsets are in display-width units across the whole buffer.
func (tb *TextBuffer) TextRange(start, end int) string {
	if start >= end {
		return ""
	}

	var sb strings.Builder
	pos := 0
	for i, line := range tb.lines {
		if i > 0 {
			if pos >= start && pos < end {
				sb.WriteByte('\n')
			}
			pos++
		}
		lineText := line.Text()
		remaining := lineText
		for len(remaining) > 0 {
			cluster, rest, cw := NextGraphemeCluster(remaining)
			remaining = rest
			if cw == 0 {
				// Zero-width cluster — include if in range, don't advance.
				if pos >= start && pos < end {
					sb.WriteString(cluster)
				}
				continue
			}
			if pos >= start && pos < end {
				sb.WriteString(cluster)
			}
			pos += cw
			if pos >= end {
				return sb.String()
			}
		}
	}
	return sb.String()
}

// TextRangeByCoords returns text between (startRow, startCol) and (endRow, endCol).
// Coordinates are in display-width columns.
func (tb *TextBuffer) TextRangeByCoords(startRow, startCol, endRow, endCol int) string {
	if startRow > endRow || (startRow == endRow && startCol >= endCol) {
		return ""
	}
	if startRow < 0 {
		startRow = 0
		startCol = 0
	}
	if endRow >= len(tb.lines) {
		endRow = len(tb.lines) - 1
		endCol = tb.lines[endRow].Width()
	}

	var sb strings.Builder
	for row := startRow; row <= endRow && row < len(tb.lines); row++ {
		lineText := tb.lines[row].Text()

		sc := 0
		if row == startRow {
			sc = startCol
		}
		ec := tb.lines[row].Width()
		if row == endRow {
			ec = endCol
		}

		// Walk grapheme clusters, collecting those in [sc, ec).
		w := 0
		remaining := lineText
		for len(remaining) > 0 {
			cluster, rest, cw := NextGraphemeCluster(remaining)
			remaining = rest
			if w >= ec {
				break
			}
			cw = ClusterWidth(cw)
			if w >= sc {
				sb.WriteString(cluster)
			}
			w += cw
		}

		if row < endRow {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Highlight by char range
// ---------------------------------------------------------------------------

// AddHighlightByCharRange adds a highlight using global character offset range.
// The highlight's Start and End fields are treated as global display-width offsets;
// the method resolves them to per-line highlights.
func (tb *TextBuffer) AddHighlightByCharRange(hl Highlight) {
	if hl.Start >= hl.End {
		return
	}
	if tb.highlights == nil {
		tb.highlights = make(map[int][]Highlight)
	}

	pos := 0
	for lineIdx := 0; lineIdx < len(tb.lines); lineIdx++ {
		if lineIdx > 0 {
			pos++ // newline
		}
		lineW := tb.lines[lineIdx].Width()
		lineEnd := pos + lineW

		// Check if this line overlaps [hl.Start, hl.End)
		if pos >= hl.End {
			break
		}
		if lineEnd <= hl.Start {
			pos = lineEnd
			continue
		}

		// Compute overlap within this line
		localStart := 0
		if hl.Start > pos {
			localStart = hl.Start - pos
		}
		localEnd := min(hl.End-pos, lineW)

		lineHL := Highlight{
			Start:      localStart,
			End:        localEnd,
			Fg:         hl.Fg,
			Bg:         hl.Bg,
			Attributes: hl.Attributes,
			Ref:        hl.Ref,
		}
		tb.highlights[lineIdx] = append(tb.highlights[lineIdx], lineHL)

		pos = lineEnd
	}
}

// HighlightCount returns the total number of highlights across all lines.
func (tb *TextBuffer) HighlightCount() int {
	total := 0
	for _, hls := range tb.highlights {
		total += len(hls)
	}
	return total
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// sanitizeText applies input transformations (e.g. tab expansion) to a string.
func (tb *TextBuffer) sanitizeText(s string) string {
	if !tb.expandTabs || !strings.ContainsRune(s, '\t') {
		return s
	}
	tw := tb.tabWidth
	if tw <= 0 {
		tw = 4
	}
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tw))
}

// sanitizeLine applies input transformations to all spans of a Line.
func (tb *TextBuffer) sanitizeLine(line Line) Line {
	if !tb.expandTabs {
		return line
	}
	spans := make([]Span, len(line.Spans))
	for i, sp := range line.Spans {
		spans[i] = sp
		spans[i].Text = tb.sanitizeText(sp.Text)
	}
	return Line{Spans: spans}
}
