package terminal

import (
	"bytes"
	"fmt"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

const inlineMaxLayoutHeight = 10000

// ---------------------------------------------------------------------------
// renderInlineFrame
// ---------------------------------------------------------------------------

func (r *Renderer) renderInlineFrame() {
	frameStart := time.Now()

	r.mu.Lock()
	deltaTime := frameStart.Sub(r.lastFrameTime).Seconds()
	r.lastFrameTime = frameStart
	bgColor := r.config.BackgroundColor
	termWidth := r.width
	r.mu.Unlock()

	// Layout width = terminal width - 1 (right margin prevents auto-wrap).
	layoutWidth := termWidth
	if layoutWidth > 1 {
		layoutWidth = termWidth - 1
	}

	// Only resize the root when the layout width actually changes.
	// The layout system's dirty tracking handles content-driven
	// relayouts automatically (AddChild, SetContent, etc. all call
	// MarkDirty). Forcing MarkDirty every frame defeats this and
	// causes a full flexbox recomputation even when nothing changed.
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

	// --- Visual cursor ---
	// Draw the cursor as an inverted cell in the buffer before flushing.
	// No hardware cursor positioning needed; inline mode keeps it hidden.
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

// ---------------------------------------------------------------------------
// InlineFlush — row-level diff with full-flush fallback
// ---------------------------------------------------------------------------
//
// The flush has two paths:
//
// DIFF (default when a previous frame buffer exists with matching width):
//
//	Compare rows cell-by-cell against the previous frame. Only rewrite
//	rows that actually changed, using cursor movement to skip unchanged
//	rows. When nothing changed at all, zero bytes are written.
//
// FULL (first frame, overflow, or width change):
//
//	Erase all previous output and write every row from scratch. This is
//	the original Ink-style approach and handles overflow (content taller
//	than terminal) by clearing the screen + scrollback.
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

	// Use diff-based flush when we have a compatible previous buffer
	// and are not in overflow mode. This reduces bytes written from
	// O(total_cells) to O(changed_cells).
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

	// Store flush sub-timings for the status bar display.
	r.root.LastFrameStats.FlushBuild = buildDur
	r.root.LastFrameStats.FlushWrite = writeDur
	r.root.LastFrameStats.FlushBytes = outBytes
	_ = flushStart

	// Save current buffer for next frame's diff.
	// After overflow, cursor/scrollback state is unreliable — discard
	// the buffer so the next non-overflow frame starts with full flush.
	if overflow {
		r.inlineCurrentBuf = nil
	} else {
		if r.inlineCurrentBuf == nil || r.inlineCurrentBuf.Width() != width || r.inlineCurrentBuf.Height() != height {
			r.inlineCurrentBuf = buffer.New(width, height)
		}
		r.inlineCurrentBuf.CopyFrom(buf)
	}

	// N+1: content lines + trailing line (cursor is one line below content).
	r.lastInlineHeight = height + 1
}

// ---------------------------------------------------------------------------
// buildInlineFull — erase previous output + write all rows
// ---------------------------------------------------------------------------
//
// Used for:
//   - First frame (no previous buffer to diff against)
//   - Overflow (content taller than terminal)
//   - Width change (row comparison meaningless)
//
// Two erase strategies:
//
//	NORMAL (height <= termHeight): erase previous output line-by-line
//	OVERFLOW: clear screen + scrollback + cursor home
func (r *Renderer) buildInlineFull(out *bytes.Buffer, buf *buffer.Buffer, overflow bool) {
	width := buf.Width()
	height := buf.Height()

	out.Grow(width * height * 4)

	// Synchronize output and prevent width mismatches from wrapping.
	out.WriteString("\x1b[?2026h\x1b[?7l")

	// --- Erase ---
	if overflow {
		// Ink's clearTerminal: clear screen + scrollback + cursor home.
		out.WriteString("\x1b[2J\x1b[3J\x1b[H")
	} else if r.lastInlineHeight > 0 {
		// Normal: erase previous output line-by-line (Ink's eraseLines).
		for i := 0; i < r.lastInlineHeight; i++ {
			out.WriteString("\x1b[2K")
			if i < r.lastInlineHeight-1 {
				out.WriteString("\x1b[A")
			}
		}
		out.WriteString("\x1b[G") // cursor to column 1
	}

	// --- Write all rows ---
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
		anchored := true // \r\n put the cursor at column 0

		for x := 0; x <= lastCol; x++ {
			cell := buf.Cell(x, y)
			ch, ok := flushRune(buf, x, y, cell)
			if !ok {
				continue // continuation covered by its head
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

	// Trailing \r\n — puts cursor at column 0 of the line below content.
	out.WriteString("\r\n")

	// Reset
	if curLinkID != 0 {
		out.WriteString("\x1b]8;;\x1b\\")
	}
	out.WriteString("\x1b[0m")

	out.WriteString("\x1b[?7h\x1b[?2026l")
}

// ---------------------------------------------------------------------------
// buildInlineDiff — row-level diff, only rewrite changed rows
// ---------------------------------------------------------------------------
//
// Compares prev (what's on screen) to next (desired frame) row-by-row.
// Unchanged rows are skipped entirely — no erase, no write, no cursor
// movement. This is the key performance optimisation: for a 200-row
// display where only the input area (2 rows) changed, we write ~2 rows
// instead of ~200.
//
// Height changes are handled:
//   - Grew:   new rows are appended below existing content
//   - Shrunk: removed rows are erased
//
// Cursor invariant: after this function, cursor is at row newH, column 0
// (one line below the last content row), matching the full-flush invariant.
func (r *Renderer) buildInlineDiff(out *bytes.Buffer, prev, next *buffer.Buffer) {
	width := next.Width()
	prevH := prev.Height()
	newH := next.Height()
	commonH := min(prevH, newH)

	// --- Find dirty rows in the common range ---
	dirtyRows := make([]int, 0, 8)
	for y := 0; y < commonH; y++ {
		if !inlineRowsEqual(prev, next, y, width) {
			dirtyRows = append(dirtyRows, y)
		}
	}

	heightGrew := newH > prevH
	heightShrunk := newH < prevH

	// Fast path: nothing changed — zero I/O.
	if len(dirtyRows) == 0 && !heightGrew && !heightShrunk {
		return
	}

	// Estimate output size: ~width*4 bytes per dirty/new row.
	estRows := len(dirtyRows)
	if heightGrew {
		estRows += newH - prevH
	}
	out.Grow(width * estRows * 4)

	out.WriteString("\x1b[?2026h\x1b[?7l") // synchronize output and disable auto-wrap

	// Cursor starts at row prevH (the trailing position from the
	// previous flush — one line below the last content row).
	curRow := prevH
	colorMode := r.backend.ColorMode()

	// Phase 1: Rewrite dirty rows in the common range.
	// Uses cursor movement (\x1b[NA / \x1b[NB) to skip unchanged rows.
	for _, y := range dirtyRows {
		inlineMoveToRow(out, curRow, y)
		curRow = y
		out.WriteString("\x1b[2K") // clear entire line
		r.writeInlineRow(out, next, y, width, colorMode)
	}

	// Phase 2: Append new rows (height grew).
	// Uses \r\n to create new terminal lines (scrolls if at bottom).
	if heightGrew {
		inlineMoveToRow(out, curRow, prevH)
		curRow = prevH
		for y := prevH; y < newH; y++ {
			out.WriteString("\x1b[2K")
			r.writeInlineRow(out, next, y, width, colorMode)
			out.WriteString("\r\n")
			curRow = y + 1
		}
		// curRow == newH after loop
	}

	// Phase 3: Erase removed rows (height shrunk).
	if heightShrunk {
		for y := newH; y < prevH; y++ {
			inlineMoveToRow(out, curRow, y)
			curRow = y
			out.WriteString("\x1b[2K")
		}
	}

	// Restore cursor to the trailing position (row newH, column 0).
	inlineMoveToRow(out, curRow, newH)

	out.WriteString("\x1b[0m")
	out.WriteString("\x1b[?7h\x1b[?2026l") // restore auto-wrap and end synchronized output
}

// ---------------------------------------------------------------------------
// writeInlineRow — emit a single row's content into the output buffer
// ---------------------------------------------------------------------------
//
// Writes cells from column 0 to the last non-clear column, emitting SGR
// sequences as needed. Self-contained: does not carry SGR state across
// rows, since the diff may skip arbitrary rows between calls.
func (r *Renderer) writeInlineRow(out *bytes.Buffer, buf *buffer.Buffer, y, width int, colorMode style.ColorMode) {
	lastCol := inlineRowLastCol(buf, y, width)
	if lastCol < 0 {
		return // row is entirely clear
	}

	curFg := style.Color{}
	curBg := style.Color{}
	var curAttrs uint32
	var curLinkID uint32
	needsReset := true
	anchored := true // the caller left the cursor at column 0

	for x := 0; x <= lastCol; x++ {
		cell := buf.Cell(x, y)
		ch, ok := flushRune(buf, x, y, cell)
		if !ok {
			continue // continuation covered by its head
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// inlineMoveToRow emits escape sequences to move the cursor from row `from`
// to row `to`. Both are 0-indexed relative to the content top. Column is
// always reset to 0 via \r.
func inlineMoveToRow(out *bytes.Buffer, from, to int) {
	if to < from {
		fmt.Fprintf(out, "\x1b[%dA\r", from-to)
	} else if to > from {
		fmt.Fprintf(out, "\x1b[%dB\r", to-from)
	} else {
		out.WriteString("\r")
	}
}

// inlineRowsEqual returns true if row y is identical in both buffers.
func inlineRowsEqual(a, b *buffer.Buffer, y, width int) bool {
	for x := 0; x < width; x++ {
		if !a.Cell(x, y).Equals(b.Cell(x, y)) {
			return false
		}
	}
	return true
}

// inlineRowLastCol returns the last column index that contains a non-clear
// cell, or -1 if the entire row is clear. Used to trim trailing whitespace
// so we don't emit unnecessary escape sequences for empty columns.
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
	// In inline mode the terminal's own background shows through for
	// unwritten cells. Any cell with a non-transparent background must
	// be emitted so the intended background color is visible.
	if !c.Bg.IsTransparent() {
		return false
	}
	return true
}

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
