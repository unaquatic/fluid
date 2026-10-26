package terminal

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// flushRune returns the base rune emitted for cell and reports whether the cell
// emits output.
//
// An intact cluster emits from its head while continuations emit nothing. A
// broken head or continuation emits a space to preserve cursor accounting.
//
// A substituted rune differs from cell.Char, which tells callers to omit the
// cluster's combining code points.
func flushRune(buf *buffer.Buffer, x, y int, cell buffer.Cell) (rune, bool) {
	if cell.IsWide() || cell.IsContinuation() {
		if !clusterIntact(buf, x, y) {
			return ' ', true
		}
		if cell.IsContinuation() {
			return 0, false
		}
	}
	if cell.Char == 0 {
		return ' ', true
	}
	return cell.Char, true
}

func clusterIntact(buf *buffer.Buffer, x, y int) bool {
	head := x
	if c := buf.Cell(x, y); c.IsContinuation() {
		head += int(c.Width)
	}
	w := int(buf.Cell(head, y).Width)
	if head < 0 || w <= max(x-head, 1) || w > buf.Width()-head {
		return false
	}
	for offset := 1; offset < w; offset++ {
		if int(buf.Cell(head+offset, y).Width) != -offset {
			return false
		}
	}
	return true
}

// widthUncertain reports whether the terminal might advance by a different
// number of columns than the buffer records for c.
//
// Wide and multi-code-point clusters depend on terminal-specific width and
// grapheme handling. Flush code re-anchors after them so disagreement does not
// accumulate across the row.
func widthUncertain(c buffer.Cell) bool {
	return c.IsWide() || c.HasCombining()
}

// anchorColumn writes an absolute, one-based CHA sequence for zero-based x.
func anchorColumn(out *bytes.Buffer, x int) {
	fmt.Fprintf(out, "\x1b[%dG", x+1)
}

// diffAndFlush compares current and next and writes changed cells to the
// terminal.
func (r *Renderer) diffAndFlush(current, next *buffer.Buffer) {
	width := next.Width()
	height := next.Height()
	var output bytes.Buffer
	output.Grow(width * height * 2) // Reserve a small estimate before building escapes.
	output.WriteString("\x1b[?7l")  // Disable automatic wrapping during the flush.

	cursorX, cursorY := -1, -1
	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs style.Attrs
	var curLinkID uint32

	needsReset := true // Emit SGR before the first changed cell.

	for y := range height {
		for x := range width {
			cur := current.Cell(x, y)
			nxt := next.Cell(x, y)

			if nxt.Equals(cur) {
				continue // Unchanged cells require no terminal output.
			}

			// Position the cursor when changed cells are not contiguous.
			if cursorX != x || cursorY != y {
				output.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
				cursorX, cursorY = x, y
			}

			// Emit SGR only when the output style changes.
			if needsReset || !nxt.Fg.Equals(curFg) || !nxt.Bg.Equals(curBg) || nxt.Attributes != curAttrs {
				output.WriteString(buildSGR(nxt.Fg, nxt.Bg, nxt.Attributes, r.backend.ColorMode(), r.backend.Indexers()))
				curFg = nxt.Fg
				curBg = nxt.Bg
				curAttrs = nxt.Attributes
				needsReset = false
			}

			// Open or close OSC 8 links at link-ID boundaries.
			linkID := nxt.Attributes.LinkID()
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
				continue // The intact cluster head accounts for this continuation.
			}

			// Emit trailing code points only for the original cluster head.
			output.WriteRune(ch)
			cursorX++
			if ch == nxt.Char {
				output.WriteString(nxt.Combining())
				cursorX += max(int(nxt.Width)-1, 0)
				if widthUncertain(nxt) {
					// Force absolute positioning before the next emitted cell.
					cursorX = -1
				}
			}
		}
	}

	if output.Len() == len("\x1b[?7l") {
		return
	}

	// Close an active link, restore automatic wrapping, and reset SGR.
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[?7h\x1b[0m")
	r.backend.WriteOut(output.Bytes())
}

// buildSGR returns one reset-prefixed ANSI SGR sequence for the supplied style,
// with the colors in mode, in the palette entries that x converts them to.
func buildSGR(fg, bg style.Color, attrs style.Attrs, mode style.ColorMode, x style.Indexers) string {
	var buf bytes.Buffer
	buf.WriteString("\x1b[0") // Reset attributes before applying the complete style.

	// Append enabled text-decoration parameters.
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

	if fg.A > 0 {
		seq := x.FgSeq(fg, mode)
		if len(seq) > 3 {
			params := seq[2 : len(seq)-1]
			buf.WriteString(";" + params)
		}
	}

	if bg.A > 0 {
		seq := x.BgSeq(bg, mode)
		if len(seq) > 3 {
			params := seq[2 : len(seq)-1]
			buf.WriteString(";" + params)
		}
	}

	buf.WriteByte('m')
	return buf.String()
}

// FullFlush writes the complete buffer without comparing a previous frame.
func (r *Renderer) FullFlush(buf *buffer.Buffer) {
	width := buf.Width()
	height := buf.Height()
	var output bytes.Buffer
	output.Grow(width * height * 4)
	output.WriteString("\x1b[?7l") // Disable automatic wrapping during the flush.

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs style.Attrs
	var curLinkID uint32
	needsReset := true

	for y := range height {
		output.WriteString(fmt.Sprintf("\x1b[%d;1H", y+1))
		anchored := true // CUP positioned the cursor at the row start.

		for x := range width {
			cell := buf.Cell(x, y)

			ch, ok := flushRune(buf, x, y, cell)
			if !ok {
				continue // The intact cluster head accounts for this continuation.
			}

			if !anchored {
				anchorColumn(&output, x)
				anchored = true
			}

			if needsReset || !cell.Fg.Equals(curFg) || !cell.Bg.Equals(curBg) || cell.Attributes != curAttrs {
				output.WriteString(buildSGR(cell.Fg, cell.Bg, cell.Attributes, r.backend.ColorMode(), r.backend.Indexers()))
				curFg = cell.Fg
				curBg = cell.Bg
				curAttrs = cell.Attributes
				needsReset = false
			}

			// Open or close OSC 8 links at link-ID boundaries.
			linkID := cell.Attributes.LinkID()
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
				output.WriteString(cell.Combining())
				anchored = !widthUncertain(cell)
			}
		}
	}

	// Close an active link, restore automatic wrapping, and reset SGR.
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[?7h\x1b[0m")
	r.backend.WriteOut(output.Bytes())
}

// Stats contains renderer timing statistics. Average duration and FPS use a
// sliding sample window; frame count and maximum duration accumulate until
// ResetStats.
type Stats struct {
	// FrameCount is the total number of frames rendered.
	FrameCount uint64
	// FPS is the reciprocal of the average frame duration, capped at 999.
	FPS float64
	// LastRenderTime is the wall-clock duration of the most recent frame.
	LastRenderTime time.Duration
	// AvgRenderTime is the average frame duration across the sample window.
	AvgRenderTime time.Duration
	// MaxRenderTime is the longest frame duration since the last reset.
	MaxRenderTime time.Duration
	samples       []time.Duration
	maxSamples    int
}

// newStats returns empty statistics with a sliding window of maxSamples. A
// nonpositive capacity uses 60 samples.
func newStats(maxSamples int) Stats {
	if maxSamples <= 0 {
		maxSamples = 60
	}
	return Stats{
		maxSamples: maxSamples,
		samples:    make([]time.Duration, 0, maxSamples),
	}
}

// record adds a frame-duration sample and updates derived values.
func (s *Stats) record(renderTime time.Duration) {
	s.FrameCount++
	s.LastRenderTime = renderTime
	s.MaxRenderTime = max(s.MaxRenderTime, renderTime)

	// Keep only the most recent samples for the average.
	s.samples = append(s.samples, renderTime)
	if len(s.samples) > s.maxSamples {
		s.samples = s.samples[1:]
	}

	var sum time.Duration
	for _, v := range s.samples {
		sum += v
	}
	s.AvgRenderTime = sum / time.Duration(len(s.samples))

	// Derive FPS from the same sliding-window average.
	if s.AvgRenderTime > 0 {
		s.FPS = math.Min(1/s.AvgRenderTime.Seconds(), 999)
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
