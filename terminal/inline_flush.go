package terminal

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

const inlineMaxLayoutHeight = 10000

// Terminal control sequences used by inline output.
const (
	inlineBeginSync = "\x1b[?2026h"
	inlineEndSync   = "\x1b[?2026l"
	inlineNoWrap    = "\x1b[?7l"
	inlineWrap      = "\x1b[?7h"
	inlineEraseRow  = "\x1b[2K"
	inlineResetSGR  = "\x1b[0m"
	inlineEraseAll  = "\x1b[2J\x1b[H\x1b[3J"
)

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
	syncStats := r.root.SyncPositions(deltaTime)

	contentHeight := max(r.computeContentHeight(), 1)

	// A resized frame has no usable previous content, so everything is damaged.
	resized := r.nextBuffer.Width() != layoutWidth || r.nextBuffer.Height() != contentHeight
	if resized {
		r.nextBuffer = buffer.New(layoutWidth, contentHeight)
	}

	damage := r.inlineDamage(layoutWidth, contentHeight, resized)

	var clearDur time.Duration
	var execStats FrameStats
	if !damage.IsEmpty() {
		tClear0 := time.Now()
		r.nextBuffer.FillRect(damage.X, damage.Y, damage.Width, damage.Height, bgColor)
		clearDur = time.Since(tClear0)

		// The hit grid is not maintained inline, so it is neither sized nor filled.
		execStats = r.root.Draw(r.nextBuffer, deltaTime, damage)
	}

	r.root.LastFrameStats = FrameStats{
		Lifecycle: prepStats.Lifecycle,
		Update:    prepStats.Update,
		Layout:    prepStats.Layout,
		Sync:      syncStats.Sync,
		Commands:  execStats.Commands,
		Render:    execStats.Render,
		BufClear:  clearDur,
		CmdCount:  execStats.CmdCount,
		NodeCount: syncStats.NodeCount,
	}

	r.mu.Lock()
	fns := make([]func(*buffer.Buffer, float64), len(r.postProcessFns))
	copy(fns, r.postProcessFns)
	r.mu.Unlock()
	for _, fn := range fns {
		fn(r.nextBuffer, deltaTime)
	}

	r.drawInlineCursor(r.nextBuffer)

	// Widgets publish the cursor while they paint, so the cell it inverted is
	// only known now. The flush has to compare and store that row even when the
	// scene damage does not reach it.
	r.inlineDamaged = damage.Union(r.inlineCursorCell)

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

// inlineDamage returns the region of the frame to repaint. Rows above the
// viewport have scrolled into the scrollback, where a repaint could not reach
// them, so damage never extends above it.
//
// A resized frame, or one whose previous content was discarded, is damaged in
// full.
func (r *Renderer) inlineDamage(width, height int, resized bool) buffer.Rect {
	full := buffer.Rect{Width: width, Height: height}
	if resized || r.inlineCurrentBuf == nil {
		r.root.ResetDamage() // A full repaint leaves nothing for the next frame.
		return full
	}

	viewport := buffer.Rect{
		Y:      r.inlineViewportTop,
		Width:  width,
		Height: height - r.inlineViewportTop,
	}
	// The cell the cursor inverted last frame has to be repainted before the
	// inversion moves. Where it moves to is not known until widgets paint, so
	// the frame widens its damage again once they have.
	damage := r.root.TakeDamage().Union(r.inlineCursorCell)
	return damage.Intersect(viewport)
}

// drawInlineCursor inverts the cell under the cursor. Inline mode keeps the
// hardware cursor hidden because it cannot be placed in the scrollback.
//
// Widgets publish their cursor position while painting, so this reads the
// position after the scene is drawn and records the cell it inverted.
func (r *Renderer) drawInlineCursor(buf *buffer.Buffer) {
	r.mu.Lock()
	x, y, visible := r.cursorX, r.cursorY, r.cursorVisible
	r.mu.Unlock()

	r.inlineCursorCell = buffer.Rect{}
	if !visible || x < 0 || x >= buf.Width() || y < 0 || y >= buf.Height() {
		return
	}

	cell := buf.Cell(x, y)
	cell.Fg, cell.Bg = cell.Bg, cell.Fg
	if cell.Char == 0 {
		cell.Char = ' '
	}
	buf.SetCellDirect(x, y, cell)
	r.inlineCursorCell = buffer.Rect{X: x, Y: y, Width: 1, Height: 1}
}

// InlineFlush updates inline output to match buf.
//
// Rows above the viewport have scrolled into the terminal's scrollback, where
// they cannot be addressed and are never rewritten. Content reaching past the
// last terminal row scrolls the terminal, which moves the rows at the top into
// the scrollback.
//
// A first frame writes every row. A width change, a height change, or a change
// to a row already in the scrollback clears the screen and rewrites every row.
// Every other frame writes only the rows that differ.
func (r *Renderer) InlineFlush(buf *buffer.Buffer) {
	termHeight := r.inlineTermRows(buf.Height())
	prev := r.inlineCurrentBuf

	switch {
	case prev == nil:
		r.inlineFullFlush(buf, termHeight, r.inlineEraseNext)
		return
	case prev.Width() != buf.Width(), r.inlineTermHeight != termHeight:
		r.inlineFullFlush(buf, termHeight, true)
		return
	}

	// Only repainted rows can differ, so the comparison starts at the damage.
	// Dropped rows lie below the frame and are always considered.
	from := min(r.inlineDamaged.Y, buf.Height())
	first, last := inlineChangedRows(prev, buf, from)
	switch {
	case first < 0:
		r.storeInline(buf, termHeight, r.inlineViewportTop, r.inlineCursorRow)
	case first < r.inlineViewportTop:
		r.inlineFullFlush(buf, termHeight, true)
	default:
		r.inlineDiffFlush(buf, prev, first, last, termHeight)
	}
}

// inlineFullFlush writes every row of buf. Erasing the screen and scrollback
// first is required whenever previously written rows can no longer be located.
func (r *Renderer) inlineFullFlush(buf *buffer.Buffer, termHeight int, erase bool) {
	width, height := buf.Width(), buf.Height()

	var out bytes.Buffer
	out.Grow(width * height * 4)
	out.WriteString(inlineBeginSync + inlineNoWrap)
	if erase {
		out.WriteString(inlineEraseAll)
	}

	colorMode := r.backend.ColorMode()
	for y := range height {
		if y > 0 {
			out.WriteString("\r\n")
		}
		out.WriteString(inlineEraseRow)
		r.writeInlineRow(&out, buf, y, width, colorMode)
	}

	// The cursor stays on the last written row, at column zero so the next frame
	// can move by whole rows.
	out.WriteString("\r" + inlineResetSGR + inlineWrap + inlineEndSync)
	r.writeInline(&out, "full")

	// Writing past the last terminal row scrolled the rows above out of reach.
	r.storeInline(buf, termHeight, height-termHeight, max(height-1, 0))
}

// inlineDiffFlush rewrites rows first through last. It scrolls the terminal
// when the change sits below the viewport, and erases rows that buf dropped.
func (r *Renderer) inlineDiffFlush(buf, prev *buffer.Buffer, first, last, termHeight int) {
	newHeight, prevHeight := buf.Height(), prev.Height()
	renderEnd := min(last, newHeight-1)
	bottom := max(newHeight-1, 0)

	// An appended row does not exist on screen yet, so the cursor stops on the
	// row above it and a newline opens it.
	target, appendStart := first, false
	switch {
	case first > renderEnd:
		target = bottom // Only dropped rows changed.
	case first == prevHeight && first > 0:
		target, appendStart = first-1, true
	}

	var out bytes.Buffer
	out.WriteString(inlineBeginSync + inlineNoWrap)

	viewportTop, cursorRow := r.inlineScroll(&out, target, termHeight)
	inlineMoveRows(&out, target-cursorRow)
	cursorRow = target

	if appendStart {
		out.WriteString("\r\n")
		cursorRow = first
	} else {
		out.WriteString("\r")
	}

	colorMode := r.backend.ColorMode()
	for y := first; y <= renderEnd; y++ {
		if y > first {
			out.WriteString("\r\n")
		}
		out.WriteString(inlineEraseRow)
		r.writeInlineRow(&out, buf, y, buf.Width(), colorMode)
		cursorRow = y
	}

	// Park on the last row of content so every frame starts from one position.
	if cursorRow < bottom {
		inlineMoveRows(&out, bottom-cursorRow)
		cursorRow = bottom
	}

	// Erasing dropped rows walks below the cursor and returns to it.
	deepest := cursorRow + inlineEraseDropped(&out, newHeight, prevHeight)

	out.WriteString("\r" + inlineResetSGR + inlineWrap + inlineEndSync)
	r.writeInline(&out, "diff")

	// Rows written past the last terminal row scrolled the viewport down.
	r.storeInline(buf, termHeight, max(viewportTop, deepest-termHeight+1), cursorRow)
}

// inlineScroll scrolls the terminal until row target can be addressed, and
// returns the resulting viewport and cursor rows. Scrolling moves the rows at
// the top of the screen into the scrollback.
func (r *Renderer) inlineScroll(out *bytes.Buffer, target, termHeight int) (viewportTop, cursorRow int) {
	viewportTop, cursorRow = r.inlineViewportTop, r.inlineCursorRow

	bottom := viewportTop + termHeight - 1
	if target <= bottom {
		return viewportTop, cursorRow
	}

	inlineMoveRows(out, bottom-cursorRow)
	scroll := target - bottom
	out.WriteString(strings.Repeat("\r\n", scroll))
	return viewportTop + scroll, target
}

// inlineEraseDropped erases the rows a shorter frame no longer occupies and
// returns the cursor to the row it started on. It returns the number of rows it
// walked below that row, which is how far the terminal may have scrolled.
func inlineEraseDropped(out *bytes.Buffer, newHeight, prevHeight int) int {
	dropped := max(prevHeight-newHeight, 0)
	if dropped == 0 {
		return 0
	}

	for range dropped {
		out.WriteString("\r\n" + inlineEraseRow)
	}
	inlineMoveRows(out, -dropped)
	return dropped
}

// storeInline records the frame the terminal now shows.
//
// Only the repainted rows are copied. The frame buffer is never cleared outside
// its damage, so both buffers already agree everywhere else.
func (r *Renderer) storeInline(buf *buffer.Buffer, termHeight, viewportTop, cursorRow int) {
	current := r.inlineCurrentBuf
	if current == nil || current.Width() != buf.Width() || current.Height() != buf.Height() {
		current = buffer.New(buf.Width(), buf.Height())
		r.inlineCurrentBuf = current
		current.CopyFrom(buf)
	} else {
		current.CopyRowsFrom(buf, r.inlineDamaged.Y, r.inlineDamaged.Height)
	}

	r.inlineTermHeight = termHeight
	r.inlineViewportTop = max(viewportTop, 0)
	r.inlineCursorRow = cursorRow
	r.inlineEraseNext = false
}

// writeInline sends out to the terminal and records flush statistics.
func (r *Renderer) writeInline(out *bytes.Buffer, kind string) {
	start := time.Now()
	r.backend.WriteOut(out.Bytes())

	r.root.LastFrameStats.FlushWrite = time.Since(start)
	r.root.LastFrameStats.FlushBytes = out.Len()
	if r.dbg != nil {
		r.dbg.Printf("[flush] %s\tbytes=%d\ttop=%d\tcursor=%d", kind, out.Len(), r.inlineViewportTop, r.inlineCursorRow)
	}
}

// inlineTermRows returns the terminal height, or fallback when it is unknown.
func (r *Renderer) inlineTermRows(fallback int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.height < 1 {
		return fallback
	}
	return r.height
}

// inlineChangedRows returns the first and last rows at or below from in which
// prev and next differ, or -1, -1 when they match. Rows beyond a frame's height
// are clear.
func inlineChangedRows(prev, next *buffer.Buffer, from int) (first, last int) {
	width := next.Width()
	first, last = -1, -1

	for y := max(from, 0); y < max(prev.Height(), next.Height()); y++ {
		if inlineRowsEqual(prev, next, y, width) {
			continue
		}
		if first < 0 {
			first = y
		}
		last = y
	}

	// Appended rows are new even when they are clear.
	if next.Height() > prev.Height() {
		if first < 0 {
			first = prev.Height()
		}
		last = next.Height() - 1
	}
	return first, last
}

// inlineMoveRows moves the cursor down by delta rows, or up when delta is
// negative.
func inlineMoveRows(out *bytes.Buffer, delta int) {
	switch {
	case delta > 0:
		fmt.Fprintf(out, "\x1b[%dB", delta)
	case delta < 0:
		fmt.Fprintf(out, "\x1b[%dA", -delta)
	}
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
