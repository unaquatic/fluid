package terminal

import (
	"bytes"
	"fmt"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

const inlineMaxLayoutHeight = 10000

// renderInlineFrame lays out content-height output, renders it, and flushes it
// into the normal terminal screen.
func (r *Renderer) renderInlineFrame() {
	frameStart := time.Now()

	r.mu.Lock()
	deltaTime := frameStart.Sub(r.lastFrameTime).Seconds()
	r.lastFrameTime = frameStart
	bgColor := r.config.BackgroundColor
	termWidth := r.width
	r.mu.Unlock()

	// Reserve the final terminal column to prevent automatic wrapping.
	layoutWidth := termWidth
	if layoutWidth > 1 {
		layoutWidth = termWidth - 1
	}

	// Resize the root only when width changes; content mutations already dirty
	// the layout tree.
	if r.lastInlineLayoutWidth != layoutWidth {
		r.lastInlineLayoutWidth = layoutWidth
		r.root.Resize(layoutWidth, inlineMaxLayoutHeight)
	}

	prepStats := r.root.PrepareLayout(deltaTime)

	contentHeight := max(r.computeContentHeight(), 1)

	if r.nextBuffer.Width() != layoutWidth || r.nextBuffer.Height() != contentHeight {
		r.nextBuffer = buffer.New(layoutWidth, contentHeight)
	}
	tClear0 := time.Now()
	r.nextBuffer.Clear(bgColor)
	clearDur := time.Since(tClear0)

	if r.hitGrid != nil {
		r.hitGrid.Resize(layoutWidth, contentHeight)
	}

	execStats := r.root.ExecuteRender(r.nextBuffer, deltaTime)

	r.root.LastFrameStats = FrameStats{
		Lifecycle: prepStats.Lifecycle,
		Update:    prepStats.Update,
		Layout:    prepStats.Layout,
		Sync:      execStats.Sync,
		Commands:  execStats.Commands,
		Render:    execStats.Render,
		BufClear:  clearDur,
		CmdCount:  execStats.CmdCount,
		NodeCount: prepStats.NodeCount,
	}

	r.mu.Lock()
	fns := make([]func(*buffer.Buffer, float64), len(r.postProcessFns))
	copy(fns, r.postProcessFns)
	r.mu.Unlock()
	for _, fn := range fns {
		fn(r.nextBuffer, deltaTime)
	}

	// Inline mode represents the cursor by inverting a cell while keeping the
	// hardware cursor hidden.
	r.mu.Lock()
	cx := r.cursorX
	cy := r.cursorY
	cursorVisible := r.cursorVisible
	r.mu.Unlock()

	if cursorVisible && cx >= 0 && cx < r.nextBuffer.Width() && cy >= 0 && cy < contentHeight {
		cell := r.nextBuffer.Cell(cx, cy)
		cell.Fg, cell.Bg = cell.Bg, cell.Fg
		if cell.Char == 0 {
			cell.Char = ' '
		}
		r.nextBuffer.SetCellDirect(cx, cy, cell)
	}

	tFlush0 := time.Now()
	r.InlineFlush(r.nextBuffer)
	flushDur := time.Since(tFlush0)
	r.root.LastFrameStats.Flush = flushDur
	totalDur := time.Since(frameStart)
	r.root.LastFrameStats.Total = totalDur

	if r.config.GatherStats {
		renderTime := time.Since(frameStart).Seconds()
		r.mu.Lock()
		r.stats.record(renderTime)
		r.mu.Unlock()
	}
}

// InlineFlush updates inline output. It rewrites only changed rows when a
// compatible previous frame exists. The first frame, a width change, or content
// taller than the terminal triggers a complete rewrite.
func (r *Renderer) InlineFlush(buf *buffer.Buffer) {
	flushStart := time.Now()
	width := buf.Width()
	height := buf.Height()
	r.mu.Lock()
	termHeight := r.height
	r.mu.Unlock()
	if termHeight < 1 {
		termHeight = height
	}

	overflow := height > termHeight || r.lastInlineHeight > termHeight

	// Row-level diffing requires matching widths and reliable on-screen cursor
	// state.
	prev := r.inlineCurrentBuf
	useDiff := !overflow && prev != nil && prev.Width() == width

	var out bytes.Buffer
	tBuild0 := time.Now()

	if useDiff {
		r.buildInlineDiff(&out, prev, buf)
	} else {
		r.buildInlineFull(&out, buf, overflow)
	}

	buildDur := time.Since(tBuild0)
	outBytes := out.Len()
	tWrite0 := time.Now()
	if outBytes > 0 {
		r.backend.WriteOut(out.Bytes())
	}
	writeDur := time.Since(tWrite0)

	// Expose build and write timings through the frame statistics.
	r.root.LastFrameStats.FlushBuild = buildDur
	r.root.LastFrameStats.FlushWrite = writeDur
	r.root.LastFrameStats.FlushBytes = outBytes
	_ = flushStart

	// Overflow makes scrollback position unreliable, so the next non-overflow
	// frame must start with a complete rewrite.
	if overflow {
		r.inlineCurrentBuf = nil
	} else {
		if r.inlineCurrentBuf == nil || r.inlineCurrentBuf.Width() != width || r.inlineCurrentBuf.Height() != height {
			r.inlineCurrentBuf = buffer.New(width, height)
		}
		r.inlineCurrentBuf.CopyFrom(buf)
	}

	// Track content rows plus the trailing cursor row.
	r.lastInlineHeight = height + 1
}

// buildInlineFull erases prior inline output and writes every current row.
// Overflow clears the screen and scrollback; normal output erases the known
// previous rows in place.
func (r *Renderer) buildInlineFull(out *bytes.Buffer, buf *buffer.Buffer, overflow bool) {
	width := buf.Width()
	height := buf.Height()

	out.Grow(width * height * 4)

	// Use synchronized output and disable automatic wrapping during the rewrite.
	out.WriteString("\x1b[?2026h\x1b[?7l")

	if overflow {
		// Reset screen, scrollback, and cursor when content exceeds the viewport.
		out.WriteString("\x1b[2J\x1b[3J\x1b[H")
	} else if r.lastInlineHeight > 0 {
		// Erase each known output row while walking back to the content origin.
		for i := 0; i < r.lastInlineHeight; i++ {
			out.WriteString("\x1b[2K")
			if i < r.lastInlineHeight-1 {
				out.WriteString("\x1b[A")
			}
		}
		out.WriteString("\x1b[G") // Return to the first column.
	}

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32
	needsReset := true
	linesWritten := 0

	for y := range height {
		lastCol := inlineRowLastCol(buf, y, width)

		if linesWritten > 0 {
			out.WriteString("\r\n")
		}
		linesWritten++
		anchored := true // CRLF placed the cursor at column zero.

		for x := 0; x <= lastCol; x++ {
			cell := buf.Cell(x, y)
			ch, ok := flushRune(buf, x, y, cell)
			if !ok {
				continue // The cluster head accounts for this continuation.
			}

			if !anchored {
				anchorColumn(out, x)
				anchored = true
			}

			if needsReset || !cell.Fg.Equals(curFg) || !cell.Bg.Equals(curBg) || cell.Attributes != curAttrs {
				out.WriteString(buildSGR(cell.Fg, cell.Bg, cell.Attributes, r.backend.ColorMode()))
				curFg = cell.Fg
				curBg = cell.Bg
				curAttrs = cell.Attributes
				needsReset = false
			}

			linkID := style.GetLinkID(cell.Attributes)
			if linkID != curLinkID {
				if linkID != 0 {
					url := r.linkRegistry.URL(linkID)
					out.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\", url))
				} else {
					out.WriteString("\x1b]8;;\x1b\\")
				}
				curLinkID = linkID
			}

			out.WriteRune(ch)
			if ch == cell.Char {
				out.WriteString(cell.Combining)
				anchored = !widthUncertain(cell)
			}
		}
	}

	// Leave the cursor at column zero on the line below the content.
	out.WriteString("\r\n")

	// Close an active link and reset SGR before restoring terminal modes.
	if curLinkID != 0 {
		out.WriteString("\x1b]8;;\x1b\\")
	}
	out.WriteString("\x1b[0m")

	out.WriteString("\x1b[?7h\x1b[?2026l")
}

// buildInlineDiff rewrites changed common rows, appends added rows, and erases
// removed rows. Unchanged rows produce no output.
//
// On return, the cursor is at column zero on the row after the new content.
func (r *Renderer) buildInlineDiff(out *bytes.Buffer, prev, next *buffer.Buffer) {
	width := next.Width()
	prevH := prev.Height()
	newH := next.Height()
	commonH := min(prevH, newH)

	// Compare rows shared by both frames.
	dirtyRows := make([]int, 0, 8)
	for y := 0; y < commonH; y++ {
		if !inlineRowsEqual(prev, next, y, width) {
			dirtyRows = append(dirtyRows, y)
		}
	}

	heightGrew := newH > prevH
	heightShrunk := newH < prevH

	// Identical dimensions and rows require no terminal output.
	if len(dirtyRows) == 0 && !heightGrew && !heightShrunk {
		return
	}

	// Reserve approximately four bytes per cell in rows that will be written.
	estRows := len(dirtyRows)
	if heightGrew {
		estRows += newH - prevH
	}
	out.Grow(width * estRows * 4)

	out.WriteString("\x1b[?2026h\x1b[?7l") // Synchronize output and disable automatic wrapping.

	// The previous flush left the cursor one row after the old content.
	curRow := prevH
	colorMode := r.backend.ColorMode()

	// Rewrite changed rows while cursor movement skips unchanged rows.
	for _, y := range dirtyRows {
		inlineMoveToRow(out, curRow, y)
		curRow = y
		out.WriteString("\x1b[2K") // Clear the complete previous row.
		r.writeInlineRow(out, next, y, width, colorMode)
	}

	// Append rows added below the previous content.
	if heightGrew {
		inlineMoveToRow(out, curRow, prevH)
		curRow = prevH
		for y := prevH; y < newH; y++ {
			out.WriteString("\x1b[2K")
			r.writeInlineRow(out, next, y, width, colorMode)
			out.WriteString("\r\n")
			curRow = y + 1
		}
	}

	// Erase rows removed from the previous frame.
	if heightShrunk {
		for y := newH; y < prevH; y++ {
			inlineMoveToRow(out, curRow, y)
			curRow = y
			out.WriteString("\x1b[2K")
		}
	}

	// Restore the trailing cursor position below the new content.
	inlineMoveToRow(out, curRow, newH)

	out.WriteString("\x1b[0m")
	out.WriteString("\x1b[?7h\x1b[?2026l") // Restore wrapping and end synchronized output.
}

// writeInlineRow writes through the final non-clear cell in row y. Each call
// starts with independent style state because diffing can skip arbitrary rows.
func (r *Renderer) writeInlineRow(out *bytes.Buffer, buf *buffer.Buffer, y, width int, colorMode style.ColorMode) {
	lastCol := inlineRowLastCol(buf, y, width)
	if lastCol < 0 {
		return // The row is entirely clear.
	}

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32
	needsReset := true
	anchored := true // The caller positioned the cursor at column zero.

	for x := 0; x <= lastCol; x++ {
		cell := buf.Cell(x, y)
		ch, ok := flushRune(buf, x, y, cell)
		if !ok {
			continue // The cluster head accounts for this continuation.
		}

		if !anchored {
			anchorColumn(out, x)
			anchored = true
		}

		if needsReset || !cell.Fg.Equals(curFg) || !cell.Bg.Equals(curBg) || cell.Attributes != curAttrs {
			out.WriteString(buildSGR(cell.Fg, cell.Bg, cell.Attributes, colorMode))
			curFg = cell.Fg
			curBg = cell.Bg
			curAttrs = cell.Attributes
			needsReset = false
		}

		linkID := style.GetLinkID(cell.Attributes)
		if linkID != curLinkID {
			if linkID != 0 {
				url := r.linkRegistry.URL(linkID)
				out.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\", url))
			} else {
				out.WriteString("\x1b]8;;\x1b\\")
			}
			curLinkID = linkID
		}

		out.WriteRune(ch)
		if ch == cell.Char {
			out.WriteString(cell.Combining)
			anchored = !widthUncertain(cell)
		}
	}

	// Close any open hyperlink within this row.
	if curLinkID != 0 {
		out.WriteString("\x1b]8;;\x1b\\")
	}
}

// inlineMoveToRow moves between zero-based content rows and returns to column
// zero.
func inlineMoveToRow(out *bytes.Buffer, from, to int) {
	if to < from {
		fmt.Fprintf(out, "\x1b[%dA\r", from-to)
	} else if to > from {
		fmt.Fprintf(out, "\x1b[%dB\r", to-from)
	} else {
		out.WriteString("\r")
	}
}

// inlineRowsEqual reports whether the first width cells of row y match.
func inlineRowsEqual(a, b *buffer.Buffer, y, width int) bool {
	for x := 0; x < width; x++ {
		if !a.Cell(x, y).Equals(b.Cell(x, y)) {
			return false
		}
	}
	return true
}

// inlineRowLastCol returns the last non-clear column in row y, or -1 when the
// row can be omitted entirely.
func inlineRowLastCol(buf *buffer.Buffer, y, width int) int {
	for x := width - 1; x >= 0; x-- {
		if !isInlineClearCell(buf.Cell(x, y)) {
			return x
		}
	}
	return -1
}

func isInlineClearCell(c buffer.Cell) bool {
	if c.Char != ' ' && c.Char != 0 {
		return false
	}
	if c.Attributes != 0 {
		return false
	}
	if !c.Fg.Equals(style.ColorWhite) {
		return false
	}
	// A non-transparent background must be emitted even for an empty glyph.
	if !c.Bg.IsTransparent() {
		return false
	}
	return true
}

// computeContentHeight returns the bottom edge of the root's laid-out children,
// including their bottom margins and the root's bottom padding.
func (r *Renderer) computeContentHeight() int {
	maxBottom := 0
	for _, child := range r.root.LayoutNode().Children {
		mb := 0
		if v, ok := child.MarginBottom.Resolve(0); ok {
			mb = v
		}
		bottom := child.Computed.Y + child.Computed.Height + mb
		if bottom > maxBottom {
			maxBottom = bottom
		}
	}
	if pb, ok := r.root.LayoutNode().PaddingBottom.Resolve(maxBottom); ok {
		maxBottom += pb
	}
	return maxBottom
}
