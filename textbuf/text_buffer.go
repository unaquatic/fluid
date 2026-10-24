package textbuf

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/style"
)

// Line is one logical line stored as a sequence of spans.
type Line struct {
	// Spans are the line's runs of text, in order.
	Spans []Span
}

// Text returns the concatenated text of all spans in sl.
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

// Width returns sl's grapheme-aware terminal width.
func (sl *Line) Width() int {
	w := 0
	for _, sp := range sl.Spans {
		w += StringWidth(sp.Text)
	}
	return w
}

// TextBuffer stores styled spans grouped into logical lines.
type TextBuffer struct {
	lines      []Line
	defaultFg  style.Color
	defaultBg  style.Color
	defaultAt  uint32
	tabWidth   int
	expandTabs bool   // Whether supported setters replace tabs with spaces.
	epoch      uint64 // Incremented after content or tab-width changes.

	// Per-line highlights.
	highlights map[int][]Highlight

	// Optional syntax-style registry.
	syntaxStyle *style.SyntaxStyle
}

// Highlight describes a half-open display-column range on one logical line.
type Highlight struct {
	// Start and End are the inclusive start and exclusive end columns.
	Start, End int
	// Fg is the foreground color; nil leaves the text's foreground.
	Fg *style.Color
	// Bg is the background color; nil leaves the text's background.
	Bg *style.Color
	// Attributes is a bitmask of the style.Attr flags, such as style.AttrBold.
	Attributes uint32
	// Ref is a caller-defined identifier used for removal.
	Ref int
}

// NewTextBuffer returns an empty one-line buffer with white foreground,
// transparent background, and a tab width of four.
func NewTextBuffer() *TextBuffer {
	return &TextBuffer{
		lines:     []Line{{Spans: []Span{{Text: ""}}}},
		defaultFg: style.ColorWhite,
		defaultBg: style.ColorTransparent,
		tabWidth:  4,
	}
}

// Length returns the total rune count, including one rune for each newline.
func (tb *TextBuffer) Length() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // Count the logical-line separator.
		}
		total += utf8.RuneCountInString(line.Text())
	}
	return total
}

// DisplayWidthLength returns the sum of line display widths plus one column for
// each newline. EditBuffer offsets use this coordinate space.
func (tb *TextBuffer) DisplayWidthLength() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // Count the logical-line separator.
		}
		total += line.Width()
	}
	return total
}

// ByteSize returns the UTF-8 byte count, including newline separators.
func (tb *TextBuffer) ByteSize() int {
	total := 0
	for i, line := range tb.lines {
		if i > 0 {
			total++ // Count the newline byte.
		}
		total += len(line.Text())
	}
	return total
}

// Reset clears content and highlights and restores default colors, attributes,
// and tab width. It preserves ExpandTabs and SyntaxStyle settings.
func (tb *TextBuffer) Reset() {
	tb.lines = []Line{{Spans: []Span{{Text: ""}}}}
	tb.defaultFg = style.ColorWhite
	tb.defaultBg = style.ColorTransparent
	tb.defaultAt = 0
	tb.tabWidth = 4
	tb.highlights = nil
	tb.epoch++
}

// LoadFile replaces the buffer's content with the content of the file at
// path. If reading the file fails, it returns the error and leaves the
// content unchanged.
func (tb *TextBuffer) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tb.SetText(string(data))
	return nil
}

// ResetDefaults restores white foreground, transparent background, and no
// attributes.
func (tb *TextBuffer) ResetDefaults() {
	tb.defaultFg = style.ColorWhite
	tb.defaultBg = style.ColorTransparent
	tb.defaultAt = 0
}

// SetText replaces all content with text using the default colors.
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

// SetSpans replaces all content with spans, splitting embedded newlines into
// logical lines and applying the default foreground to zero-color spans.
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

// Append adds text to the end of the buffer, splitting newlines into logical
// lines. It does not apply tab expansion.
func (tb *TextBuffer) Append(text string) {
	parts := strings.Split(text, "\n")
	if len(tb.lines) == 0 {
		tb.lines = []Line{{Spans: []Span{{Text: ""}}}}
	}

	// Extend the final span with text before the first newline.
	last := &tb.lines[len(tb.lines)-1]
	if len(last.Spans) == 0 {
		last.Spans = append(last.Spans, Span{Fg: tb.defaultFg, Bg: tb.defaultBg})
	}
	last.Spans[len(last.Spans)-1].Text += parts[0]

	// Each remaining part starts a new default-styled line.
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

// LineCount returns the number of logical lines.
func (tb *TextBuffer) LineCount() int { return len(tb.lines) }

// LineAt returns the line value at row, or a default empty line outside the
// buffer. Its Spans slice shares underlying storage with the buffer.
func (tb *TextBuffer) LineAt(row int) Line {
	if row < 0 || row >= len(tb.lines) {
		return Line{Spans: []Span{{Text: ""}}}
	}
	return tb.lines[row]
}

// LineWidthAt returns the grapheme-aware display width of the stored line at
// row, or zero outside the buffer.
func (tb *TextBuffer) LineWidthAt(row int) int {
	if row < 0 || row >= len(tb.lines) {
		return 0
	}
	return tb.lines[row].Width()
}

// LineTextAt returns the unstyled text at row, or an empty string outside the
// buffer.
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

// Epoch returns the content and tab-width revision.
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

// SetTabWidth stores w and increments Epoch when it changes.
func (tb *TextBuffer) SetTabWidth(w int) {
	if tb.tabWidth == w {
		return
	}
	tb.tabWidth = w
	tb.epoch++
}

// ExpandTabs returns whether tab characters are expanded to spaces on input.
func (tb *TextBuffer) ExpandTabs() bool { return tb.expandTabs }

// SetExpandTabs controls tab replacement in SetText, SetSpans, and SetLines.
func (tb *TextBuffer) SetExpandTabs(v bool) { tb.expandTabs = v }

// SyntaxStyle returns the attached syntax-style registry, or nil.
func (tb *TextBuffer) SyntaxStyle() *style.SyntaxStyle { return tb.syntaxStyle }

// SetSyntaxStyle attaches ss for token coloring; nil detaches it.
func (tb *TextBuffer) SetSyntaxStyle(ss *style.SyntaxStyle) { tb.syntaxStyle = ss }

// Lines returns the underlying line slice without copying it.
func (tb *TextBuffer) Lines() []Line { return tb.lines }

// SetLines replaces content with lines and retains the supplied slice. It
// sanitizes a copied set of spans when tab expansion is enabled.
func (tb *TextBuffer) SetLines(lines []Line) {
	if tb.expandTabs {
		for i := range lines {
			lines[i] = tb.sanitizeLine(lines[i])
		}
	}
	tb.lines = lines
	tb.epoch++
}

// AddHighlight appends hl to lineIdx.
func (tb *TextBuffer) AddHighlight(lineIdx int, hl Highlight) {
	if tb.highlights == nil {
		tb.highlights = make(map[int][]Highlight)
	}
	tb.highlights[lineIdx] = append(tb.highlights[lineIdx], hl)
}

// RemoveHighlightsByRef removes every highlight whose Ref equals ref.
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

// ClearLineHighlights removes all highlights on lineIdx.
func (tb *TextBuffer) ClearLineHighlights(lineIdx int) {
	delete(tb.highlights, lineIdx)
}

// ClearAllHighlights removes all highlights.
func (tb *TextBuffer) ClearAllHighlights() {
	tb.highlights = nil
}

// LineHighlights returns the underlying highlight slice for lineIdx.
func (tb *TextBuffer) LineHighlights(lineIdx int) []Highlight {
	if tb.highlights == nil {
		return nil
	}
	return tb.highlights[lineIdx]
}

// TextRange returns text in the half-open global display-width range [start,
// end). Newline separators each occupy one offset.
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
				// Include a zero-width cluster at the current offset without advancing.
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

// TextRangeByCoords returns text in the half-open range between two logical
// row and display-column positions.
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

		// Include clusters whose starting column lies in [sc, ec).
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

// AddHighlightByCharRange converts hl's global display-width range into
// per-line highlights.
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

		// Skip logical lines outside the requested half-open range.
		if pos >= hl.End {
			break
		}
		if lineEnd <= hl.Start {
			pos = lineEnd
			continue
		}

		// Convert the overlap to line-local columns.
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

// sanitizeText applies configured tab expansion to s.
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

// sanitizeLine copies line and sanitizes each span.
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
