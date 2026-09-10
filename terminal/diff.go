package terminal

import (
	"bytes"
	"fmt"
	"math"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// flushRune
// ---------------------------------------------------------------------------

// flushRune returns the rune to emit for the cell at (x, y) and whether the
// cell emits anything at all.
//
// A wide character is emitted as its head rune, and its Width=-1 continuation
// emits nothing, because the head already advanced two columns. That only
// holds while the pair is intact. Buffer.detachWide keeps it intact, and this
// is the backstop: a half of a broken pair is emitted as a space so the row
// still advances exactly the columns its cells claim. Letting the drift reach
// the terminal desynchronises the cursor and corrupts the rest of the frame.
//
// The returned rune differs from cell.Char whenever it is a substitution, so
// callers can tell whether to emit Cell.Combining and whether two columns were
// consumed.
//
// See widthUncertain for what has to happen after emitting the result.
func flushRune(buf *buffer.Buffer, x, y int, cell buffer.Cell) (rune, bool) {
	switch {
	case cell.Width == -1:
		if x > 0 && buf.Cell(x-1, y).Width == 2 {
			return 0, false // covered by its head
		}
		return ' ', true // stranded continuation: claim its column
	case cell.Width == 2 && buf.Cell(x+1, y).Width != -1:
		return ' ', true // stranded head: would claim a column it no longer owns
	case cell.Char == 0:
		return ' ', true
	}
	return cell.Char, true
}

// widthUncertain reports whether the terminal's advance for cell may differ
// from the width the buffer assigned it.
//
// A single-codepoint narrow cell is safe: every terminal advances exactly one
// column for it. Wide characters and multi-codepoint clusters are not — their
// advance depends on the terminal's own width tables and grapheme handling,
// which fluid can neither know nor always probe. iTerm composes
// U+1100 U+1161 U+11A8 into one narrow-looking glyph while advancing four
// columns; other terminals render emoji sequences one column wide.
//
// After emitting one of these the flush re-anchors the cursor to an absolute
// column. That turns a disagreement into a single misplaced glyph rather than
// a row that drifts from that point on — and, once the drift runs past the
// right margin, a wrap that desynchronises every row after it.
func widthUncertain(c buffer.Cell) bool {
	return c.Width == 2 || c.Combining != ""
}

// anchorColumn emits an absolute cursor column move (CHA). Used to re-anchor
// after a cell whose advance the terminal may disagree about.
func anchorColumn(out *bytes.Buffer, x int) {
	fmt.Fprintf(out, "\x1b[%dG", x+1)
}

// ---------------------------------------------------------------------------
// DiffAndFlush
// ---------------------------------------------------------------------------

// DiffAndFlush compares the current and next buffers cell-by-cell
// and writes only the changed cells to the terminal.
func (r *Renderer) DiffAndFlush(current, next *buffer.Buffer) {
	width := next.Width()
	height := next.Height()
	var output bytes.Buffer
	output.Grow(width * height * 2) // rough estimate
	output.WriteString("\x1b[?7l")  // prevent width mismatches from wrapping or scrolling

	cursorX, cursorY := -1, -1
	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32

	needsReset := true // force first SGR

	for y := range height {
		for x := range width {
			cur := current.Cell(x, y)
			nxt := next.Cell(x, y)

			if nxt.Equals(cur) {
				continue // skip unchanged cells
			}

			// Move cursor if not already at (x, y)
			if cursorX != x || cursorY != y {
				output.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
				cursorX, cursorY = x, y
			}

			// Update SGR if colors/attrs changed
			if needsReset || !nxt.Fg.Equals(curFg) || !nxt.Bg.Equals(curBg) || nxt.Attributes != curAttrs {
				output.WriteString(buildSGR(nxt.Fg, nxt.Bg, nxt.Attributes, r.backend.ColorMode()))
				curFg = nxt.Fg
				curBg = nxt.Bg
				curAttrs = nxt.Attributes
				needsReset = false
			}

			// OSC 8 hyperlink transitions
			linkID := style.GetLinkID(nxt.Attributes)
			if linkID != curLinkID {
				if linkID != 0 {
					url := r.linkRegistry.URL(linkID)
					output.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\", url))
				} else {
					output.WriteString("\x1b]8;;\x1b\\")
				}
				curLinkID = linkID
			}

			ch, ok := flushRune(next, x, y, nxt)
			if !ok {
				continue // continuation covered by its head
			}

			// Write character + combining codepoints (ZWJ sequences, etc.)
			output.WriteRune(ch)
			cursorX++
			if ch == nxt.Char {
				output.WriteString(nxt.Combining)
				if nxt.Width == 2 {
					cursorX++ // wide char advances 2 columns
				}
				if widthUncertain(nxt) {
					// The terminal may not have advanced what we budgeted.
					// Forget our position so the next cell re-anchors.
					cursorX = -1
				}
			}
		}
	}

	if output.Len() == len("\x1b[?7l") {
		return
	}

	// Close any open hyperlink, restore auto-wrap, and reset.
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[?7h\x1b[0m")
	r.backend.WriteOut(output.Bytes())
}

// ---------------------------------------------------------------------------
// buildSGR
// ---------------------------------------------------------------------------

// buildSGR generates an ANSI SGR escape sequence for the given style.
// Produces sequences like: \x1b[0;1;38;2;R;G;B;48;2;R;G;Bm
func buildSGR(fg, bg style.Color, attrs uint32, mode style.ColorMode) string {
	var buf bytes.Buffer
	buf.WriteString("\x1b[0") // always reset first

	// Text attributes
	if attrs&style.AttrBold != 0 {
		buf.WriteString(";1")
	}
	if attrs&style.AttrDim != 0 {
		buf.WriteString(";2")
	}
	if attrs&style.AttrItalic != 0 {
		buf.WriteString(";3")
	}
	if attrs&style.AttrUnderline != 0 {
		buf.WriteString(";4")
	}
	if attrs&style.AttrBlink != 0 {
		buf.WriteString(";5")
	}
	if attrs&style.AttrInverse != 0 {
		buf.WriteString(";7")
	}
	if attrs&style.AttrHidden != 0 {
		buf.WriteString(";8")
	}
	if attrs&style.AttrStrikethrough != 0 {
		buf.WriteString(";9")
	}

	// Foreground color
	if fg.A > 0 {
		seq := style.FgColorSeq(fg, mode)
		if len(seq) > 3 {
			params := seq[2 : len(seq)-1]
			buf.WriteString(";" + params)
		}
	}

	// Background color
	if bg.A > 0 {
		seq := style.BgColorSeq(bg, mode)
		if len(seq) > 3 {
			params := seq[2 : len(seq)-1]
			buf.WriteString(";" + params)
		}
	}

	buf.WriteByte('m')
	return buf.String()
}

// ---------------------------------------------------------------------------
// FullFlush
// ---------------------------------------------------------------------------

// FullFlush writes the entire buffer to the terminal (no diffing).
// Used for initial render or after suspend/resume.
func (r *Renderer) FullFlush(buf *buffer.Buffer) {
	width := buf.Width()
	height := buf.Height()
	var output bytes.Buffer
	output.Grow(width * height * 4)
	output.WriteString("\x1b[?7l") // prevent width mismatches from wrapping or scrolling

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32
	needsReset := true

	for y := range height {
		output.WriteString(fmt.Sprintf("\x1b[%d;1H", y+1))
		anchored := true // the row-start CUP just placed the cursor

		for x := range width {
			cell := buf.Cell(x, y)

			ch, ok := flushRune(buf, x, y, cell)
			if !ok {
				continue // continuation covered by its head
			}

			if !anchored {
				anchorColumn(&output, x)
				anchored = true
			}

			if needsReset || !cell.Fg.Equals(curFg) || !cell.Bg.Equals(curBg) || cell.Attributes != curAttrs {
				output.WriteString(buildSGR(cell.Fg, cell.Bg, cell.Attributes, r.backend.ColorMode()))
				curFg = cell.Fg
				curBg = cell.Bg
				curAttrs = cell.Attributes
				needsReset = false
			}

			// OSC 8 hyperlink transitions
			linkID := style.GetLinkID(cell.Attributes)
			if linkID != curLinkID {
				if linkID != 0 {
					url := r.linkRegistry.URL(linkID)
					output.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\", url))
				} else {
					output.WriteString("\x1b]8;;\x1b\\")
				}
				curLinkID = linkID
			}

			output.WriteRune(ch)
			if ch == cell.Char {
				output.WriteString(cell.Combining)
				anchored = !widthUncertain(cell)
			}
		}
	}

	// Close any open hyperlink, restore auto-wrap, and reset.
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[?7h\x1b[0m")
	r.backend.WriteOut(output.Bytes())
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

// Stats holds rendering performance statistics collected across
// frames. Enable collection via Config.GatherStats and read the
// current values via Renderer.Stats(). The stats maintain a sliding window
// of recent frame times for computing averages.
type Stats struct {
	// FrameCount is the total number of frames rendered.
	FrameCount uint64
	// FPS is the current frames per second, computed from recent samples.
	FPS float64
	// LastRenderTime is the wall-clock duration of the most recent frame
	// in seconds.
	LastRenderTime float64
	// AvgRenderTime is the average frame duration across the sample window
	// in seconds.
	AvgRenderTime float64
	// MaxRenderTime is the longest frame duration across the sample window
	// in seconds.
	MaxRenderTime float64
	samples       []float64
	maxSamples    int
}

// NewStats creates a Stats with the given sample capacity.
func NewStats(maxSamples int) Stats {
	if maxSamples <= 0 {
		maxSamples = 60
	}
	return Stats{
		maxSamples: maxSamples,
		samples:    make([]float64, 0, maxSamples),
	}
}

// record adds a frame time sample and updates statistics.
func (s *Stats) record(renderTimeSec float64) {
	s.FrameCount++
	s.LastRenderTime = renderTimeSec

	if renderTimeSec > s.MaxRenderTime {
		s.MaxRenderTime = renderTimeSec
	}

	// Sliding window average
	s.samples = append(s.samples, renderTimeSec)
	if len(s.samples) > s.maxSamples {
		s.samples = s.samples[1:]
	}

	sum := 0.0
	for _, v := range s.samples {
		sum += v
	}
	s.AvgRenderTime = sum / float64(len(s.samples))

	// FPS from average frame time
	if s.AvgRenderTime > 0 {
		s.FPS = math.Min(1.0/s.AvgRenderTime, 999)
	}
}

// reset clears all statistics.
func (s *Stats) reset() {
	s.FrameCount = 0
	s.FPS = 0
	s.LastRenderTime = 0
	s.AvgRenderTime = 0
	s.MaxRenderTime = 0
	s.samples = s.samples[:0]
}
