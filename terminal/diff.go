package terminal

import (
	"bytes"
	"fmt"
	"math"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

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

			// Skip wide char continuations
			if nxt.Width == -1 {
				continue
			}

			// Write character + combining codepoints (ZWJ sequences, etc.)
			ch := nxt.Char
			if ch == 0 {
				ch = ' '
			}
			output.WriteRune(ch)
			if nxt.Combining != "" {
				output.WriteString(nxt.Combining)
			}
			cursorX++
			if nxt.Width == 2 {
				cursorX++ // wide char advances 2 columns
			}
		}
	}

	// Close any open hyperlink and reset
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[0m")

	if output.Len() > 4 { // more than just the reset
		r.backend.WriteOut(output.Bytes())
	}
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

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32
	needsReset := true

	for y := range height {
		output.WriteString(fmt.Sprintf("\x1b[%d;1H", y+1))

		for x := range width {
			cell := buf.Cell(x, y)

			if cell.Width == -1 {
				continue // skip continuation cells
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

			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}
			output.WriteRune(ch)
			if cell.Combining != "" {
				output.WriteString(cell.Combining)
			}
		}
	}

	// Close any open hyperlink and reset
	if curLinkID != 0 {
		output.WriteString("\x1b]8;;\x1b\\")
	}
	output.WriteString("\x1b[0m")
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
