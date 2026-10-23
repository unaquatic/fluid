package widget

import (
	"fmt"
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// LineInfoProvider is implemented by widgets that can provide line information
// for the LineNumber gutter to sync with.
type LineInfoProvider interface {
	LineCount() int
}

// ScrollInfoProvider is optionally implemented to sync scroll position.
type ScrollInfoProvider interface {
	ScrollY() int
}

// CursorInfoProvider is implemented by widgets that can report their cursor row.
// Used for active line highlighting.
type CursorInfoProvider interface {
	CursorRow() int
}

// VisualLineProvider is implemented by widgets that support word wrap.
// When present, the line number gutter shows the logical line number only on
// the first visual line of each logical line; continuation lines show empty gutter.
type VisualLineProvider interface {
	VisualLineCount() int
	// LogicalLineForVisualLine maps a visual line index to its logical line index.
	LogicalLineForVisualLine(visualLine int) int
}

// LineNumberMode controls how line numbers are displayed.
type LineNumberMode int

const (
	LineNumberAbsolute LineNumberMode = iota // Show absolute line numbers
	LineNumberRelative                       // Show distance from active line
)

// LineSign configures sign indicators displayed before/after line numbers.
type LineSign struct {
	Before      string      // text before line number (e.g. for bookmarks)
	BeforeColor style.Color // color for before text
	After       string      // text after line number (e.g. " +" or " -")
	AfterColor  style.Color // color for after text
}

// LineColorConfig configures per-line background colors for both the gutter
// and content areas.
type LineColorConfig struct {
	Gutter  style.Color // background for the gutter (line number + signs)
	Content style.Color // background for the content area (code)
}

// LineNumber displays line numbers synced with a target widget.
//
// Features:
//   - Active line highlighting with configurable foreground and background
//   - Wrap-aware numbering (shows number only on first visual line of each logical line)
//   - Relative line number mode
//   - Auto-width calculation from max line number digit count
//   - Scroll sync with target widget
//   - Per-line background colors (gutter + content), signs, custom line numbers
//   - Content area background painting (for diff-style coloring)
type LineNumber struct {
	*render.BaseRenderable

	target           render.Renderable // the widget to sync with
	fg               style.Color
	bg               style.Color
	activeFg         style.Color // color for the current/active line
	activeBg         style.Color // background for the active line row
	minWidth         int
	lineNumberOffset int  // offset added to display line numbers (default 1)
	showLineNumbers  bool // whether to show line numbers (default true)
	mode             LineNumberMode
	separator        rune // separator char after gutter (0 = none)
	separatorFg      style.Color

	// Per-line customization (used by Diff and other composing widgets)
	lineColors      map[int]LineColorConfig // logical line → gutter/content bg
	lineSigns       map[int]LineSign        // logical line → sign indicators
	lineNumbers     map[int]int             // logical line → custom display number
	hideLineNumbers map[int]bool            // logical lines where number is hidden

	// Cached sign column widths
	maxBeforeWidth int
	maxAfterWidth  int
}

// LineNumberOption configures a LineNumber widget.
type LineNumberOption func(*LineNumber)

// NewLineNumber creates a LineNumber widget.
func NewLineNumber(target render.Renderable, opts ...LineNumberOption) *LineNumber {
	ln := &LineNumber{
		BaseRenderable:   render.NewBaseRenderable(),
		target:           target,
		fg:               style.ColorGray,
		bg:               style.ColorTransparent,
		activeFg:         style.ColorWhite,
		activeBg:         style.ColorTransparent,
		minWidth:         4,
		lineNumberOffset: 1,
		showLineNumbers:  true,
		mode:             LineNumberAbsolute,
		separator:        0,
		separatorFg:      style.ColorGray,
	}

	for _, opt := range opts {
		opt(ln)
	}

	return ln
}

// SetTarget sets the target widget to sync with.
func (ln *LineNumber) SetTarget(target render.Renderable) {
	ln.target = target
	ln.RequestRender()
}

// Target returns the current target widget.
func (ln *LineNumber) Target() render.Renderable { return ln.target }

// SetMode sets the line number display mode.
func (ln *LineNumber) SetMode(mode LineNumberMode) {
	if ln.mode != mode {
		ln.mode = mode
		ln.RequestRender()
	}
}

// Mode returns the current line number mode.
func (ln *LineNumber) Mode() LineNumberMode { return ln.mode }

// SetShowLineNumbers enables/disables line number display.
func (ln *LineNumber) SetShowLineNumbers(v bool) {
	ln.showLineNumbers = v
	ln.RequestRender()
}

// ShowLineNumbers returns whether line numbers are shown.
func (ln *LineNumber) ShowLineNumbers() bool { return ln.showLineNumbers }

// ---------------------------------------------------------------------------
// Per-line customization setters (used by Diff)
// ---------------------------------------------------------------------------

// SetLineColors sets per-line background colors for gutter and content areas.
func (ln *LineNumber) SetLineColors(colors map[int]LineColorConfig) {
	ln.lineColors = colors
	ln.RequestRender()
}

// SetLineSigns sets per-line sign indicators.
func (ln *LineNumber) SetLineSigns(signs map[int]LineSign) {
	ln.lineSigns = signs
	ln.calculateSignWidths()
	ln.RequestRender()
}

// SetLineNumbers sets custom line number mapping (logical line → display number).
func (ln *LineNumber) SetLineNumbers(numbers map[int]int) {
	ln.lineNumbers = numbers
	ln.RequestRender()
}

// SetHideLineNumbers sets which logical lines should hide their line number.
func (ln *LineNumber) SetHideLineNumbers(hide map[int]bool) {
	ln.hideLineNumbers = hide
	ln.RequestRender()
}

// calculateSignWidths computes the maximum before/after sign text widths.
func (ln *LineNumber) calculateSignWidths() {
	ln.maxBeforeWidth = 0
	ln.maxAfterWidth = 0
	for _, sign := range ln.lineSigns {
		if w := len(sign.Before); w > ln.maxBeforeWidth {
			ln.maxBeforeWidth = w
		}
		if w := len(sign.After); w > ln.maxAfterWidth {
			ln.maxAfterWidth = w
		}
	}
}

// ---------------------------------------------------------------------------
// Computed Width
// ---------------------------------------------------------------------------

// ComputedWidth returns the width the gutter needs based on the target's line count.
func (ln *LineNumber) ComputedWidth() int {
	if !ln.showLineNumbers || ln.target == nil {
		return 0
	}

	lineCount := ln.getLineCount()
	maxLine := lineCount + ln.lineNumberOffset

	// Account for custom line numbers which may be larger
	for _, num := range ln.lineNumbers {
		if num > maxLine {
			maxLine = num
		}
	}

	digits := max(len(fmt.Sprintf("%d", maxLine)), 1)

	w := digits
	if ln.maxAfterWidth > 0 {
		// When after signs present, the sign provides spacing
		w += ln.maxAfterWidth
	} else {
		w += 1 // trailing space when no signs
	}
	w += ln.maxBeforeWidth
	if ln.separator != 0 {
		w++ // separator char
	}
	if w < ln.minWidth {
		w = ln.minWidth
	}
	return w
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// RenderSelf draws the line numbers and per-line content backgrounds.
func (ln *LineNumber) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := ln.Width()
	h := ln.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := ln.X()
	y := ln.Y()

	// Fill background
	if !ln.bg.IsTransparent() {
		buf.FillRect(x, y, w, h, ln.bg)
	}

	if ln.target == nil {
		return
	}

	// Get line count from target
	lineCount := ln.getLineCount()

	// Get scroll offset from target
	scrollY := ln.getScrollY()

	// Get cursor row from target (for active line)
	activeLine := ln.getActiveLine()

	// Check if target supports visual line mapping (word wrap)
	hasVisualLines := false
	var vlp VisualLineProvider
	if vl, ok := ln.target.(VisualLineProvider); ok {
		hasVisualLines = true
		vlp = vl
	}

	// Calculate number display width (excluding sign columns)
	numWidth := ln.computeNumWidth(lineCount)
	sepWidth := 0
	if ln.separator != 0 {
		sepWidth = 1
	}

	// Total gutter width (for content bg positioning)
	gutterW := ln.ComputedWidth()

	// Draw content area backgrounds (behind the target code widget).
	// This paints the full-width row bg for each line.
	if ln.lineColors != nil && ln.showLineNumbers {
		ln.drawContentBackgrounds(buf, x, y, w, h, gutterW, scrollY, lineCount, hasVisualLines, vlp)
	}

	if !ln.showLineNumbers {
		return
	}

	// Draw gutter rows
	for row := range h {
		visualLine := scrollY + row
		lineY := y + row

		// Determine logical line number
		logicalLine := -1
		isFirstVisualLine := true

		if hasVisualLines {
			totalVisual := vlp.VisualLineCount()
			if visualLine < totalVisual {
				logicalLine = vlp.LogicalLineForVisualLine(visualLine)
				if visualLine > 0 {
					prevLogical := vlp.LogicalLineForVisualLine(visualLine - 1)
					isFirstVisualLine = prevLogical != logicalLine
				}
			}
		} else {
			if visualLine < lineCount {
				logicalLine = visualLine
			}
		}

		if logicalLine < 0 {
			break
		}

		// Per-line gutter background
		rowBg := style.ColorTransparent
		if lc, ok := ln.lineColors[logicalLine]; ok && !lc.Gutter.IsTransparent() {
			rowBg = lc.Gutter
			buf.FillRect(x, lineY, gutterW, 1, rowBg)
		}

		// Determine if this is the active line
		isActive := activeLine >= 0 && logicalLine == activeLine

		// Active line background (takes precedence)
		if isActive && !ln.activeBg.IsTransparent() {
			rowBg = ln.activeBg
			buf.FillRect(x, lineY, w, 1, rowBg)
		}

		// Skip number/sign rendering for hidden lines or continuation lines
		if ln.hideLineNumbers[logicalLine] || !isFirstVisualLine {
			continue
		}

		fg := ln.fg
		if isActive {
			fg = ln.activeFg
		}

		cx := x

		// Before sign
		if ln.maxBeforeWidth > 0 {
			if sign, ok := ln.lineSigns[logicalLine]; ok && sign.Before != "" {
				padding := ln.maxBeforeWidth - len(sign.Before)
				cx += padding
				signFg := sign.BeforeColor
				if signFg.IsTransparent() {
					signFg = fg
				}
				for _, r := range sign.Before {
					if cx-x >= gutterW {
						break
					}
					buf.SetCell(cx, lineY, r, signFg, rowBg, 0)
					cx++
				}
			} else {
				cx += ln.maxBeforeWidth
			}
		}

		// Line number (right-aligned within numWidth)
		numStr := ""
		if customNum, ok := ln.lineNumbers[logicalLine]; ok {
			displayNum := customNum
			if ln.mode == LineNumberRelative && activeLine >= 0 && !isActive {
				dist := logicalLine - activeLine
				if dist < 0 {
					dist = -dist
				}
				numStr = fmt.Sprintf("%d", dist)
			} else {
				numStr = fmt.Sprintf("%d", displayNum)
			}
		} else {
			displayNum := logicalLine + ln.lineNumberOffset
			if ln.mode == LineNumberRelative && activeLine >= 0 {
				if isActive {
					numStr = fmt.Sprintf("%d", displayNum)
				} else {
					dist := logicalLine - activeLine
					if dist < 0 {
						dist = -dist
					}
					numStr = fmt.Sprintf("%d", dist)
				}
			} else {
				numStr = fmt.Sprintf("%d", displayNum)
			}
		}

		// Right-align the number within numWidth.
		// When after signs are present, the sign provides spacing so we omit
		// the trailing space from the number area.
		trailingSpace := " "
		if ln.maxAfterWidth > 0 {
			trailingSpace = ""
		}
		availDigits := numWidth
		if trailingSpace != "" {
			availDigits-- // reserve space for trailing space
		}
		if availDigits < 1 {
			availDigits = 1
		}
		padded := strings.Repeat(" ", availDigits-len(numStr)) + numStr + trailingSpace
		for _, r := range padded {
			if cx-x >= gutterW-sepWidth-ln.maxAfterWidth {
				break
			}
			buf.SetCell(cx, lineY, r, fg, rowBg, 0)
			cx++
		}

		// After sign
		if ln.maxAfterWidth > 0 {
			if sign, ok := ln.lineSigns[logicalLine]; ok && sign.After != "" {
				signFg := sign.AfterColor
				if signFg.IsTransparent() {
					signFg = fg
				}
				for _, r := range sign.After {
					if cx-x >= gutterW-sepWidth {
						break
					}
					buf.SetCell(cx, lineY, r, signFg, rowBg, 0)
					cx++
				}
			}
		}

		// Separator
		if ln.separator != 0 && cx-x < gutterW {
			buf.SetCell(cx, lineY, ln.separator, ln.separatorFg, rowBg, 0)
		}
	}
}

// drawContentBackgrounds paints the content area background for each visible line.
// This is separate from the gutter rendering and fills the area to the right of the gutter.
func (ln *LineNumber) drawContentBackgrounds(buf *buffer.Buffer, x, y, w, h, gutterW, scrollY, lineCount int, hasVisualLines bool, vlp VisualLineProvider) {
	contentX := x + gutterW
	contentW := w - gutterW
	if contentW <= 0 {
		return
	}

	for row := range h {
		visualLine := scrollY + row
		lineY := y + row

		logicalLine := -1
		if hasVisualLines {
			totalVisual := vlp.VisualLineCount()
			if visualLine < totalVisual {
				logicalLine = vlp.LogicalLineForVisualLine(visualLine)
			}
		} else {
			if visualLine < lineCount {
				logicalLine = visualLine
			}
		}

		if logicalLine < 0 {
			break
		}

		if lc, ok := ln.lineColors[logicalLine]; ok && !lc.Content.IsTransparent() {
			buf.FillRect(contentX, lineY, contentW, 1, lc.Content)
		}
	}
}

// getLineCount gets the line count from the target.
func (ln *LineNumber) getLineCount() int {
	if lip, ok := ln.target.(LineInfoProvider); ok {
		return lip.LineCount()
	}
	return 0
}

// getScrollY gets the scroll offset from the target.
func (ln *LineNumber) getScrollY() int {
	if sip, ok := ln.target.(ScrollInfoProvider); ok {
		return sip.ScrollY()
	}
	return 0
}

// getActiveLine gets the cursor row from the target, or -1.
func (ln *LineNumber) getActiveLine() int {
	if cip, ok := ln.target.(CursorInfoProvider); ok {
		return cip.CursorRow()
	}
	return -1
}

// computeNumWidth calculates the column width needed for line numbers.
func (ln *LineNumber) computeNumWidth(lineCount int) int {
	maxLine := lineCount + ln.lineNumberOffset

	// Account for custom line numbers
	for _, num := range ln.lineNumbers {
		if num > maxLine {
			maxLine = num
		}
	}

	digits := max(len(fmt.Sprintf("%d", maxLine)), 1)
	numWidth := digits
	if ln.maxAfterWidth > 0 {
		// When signs present, sign text provides spacing
	} else {
		numWidth++ // trailing space
	}
	if numWidth < ln.minWidth {
		numWidth = ln.minWidth
	}
	w := ln.Width()
	sepWidth := 0
	if ln.separator != 0 {
		sepWidth = 1
	}
	maxNumW := w - sepWidth - ln.maxBeforeWidth - ln.maxAfterWidth
	if numWidth > maxNumW {
		numWidth = maxNumW
	}
	return numWidth
}

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

// SetFg sets the default line number foreground color.
func (ln *LineNumber) SetFg(c style.Color) {
	ln.fg = c
	ln.RequestRender()
}

// SetBg sets the gutter background color.
func (ln *LineNumber) SetBg(c style.Color) {
	ln.bg = c
	ln.RequestRender()
}

// SetActiveFg sets the active line foreground color.
func (ln *LineNumber) SetActiveFg(c style.Color) {
	ln.activeFg = c
	ln.RequestRender()
}

// SetActiveBg sets the active line background color.
func (ln *LineNumber) SetActiveBg(c style.Color) {
	ln.activeBg = c
	ln.RequestRender()
}

// SetSeparator sets the separator character drawn after the gutter (0 to disable).
func (ln *LineNumber) SetSeparator(r rune) {
	ln.separator = r
	ln.RequestRender()
}

// ---------------------------------------------------------------------------
// LineNumber Options
// ---------------------------------------------------------------------------

// LineNumberWithFg sets the foreground color.
func LineNumberWithFg(c style.Color) LineNumberOption {
	return func(ln *LineNumber) { ln.fg = c }
}

// LineNumberWithBg sets the background color.
func LineNumberWithBg(c style.Color) LineNumberOption {
	return func(ln *LineNumber) { ln.bg = c }
}

// LineNumberWithActiveFg sets the active line foreground color.
func LineNumberWithActiveFg(c style.Color) LineNumberOption {
	return func(ln *LineNumber) { ln.activeFg = c }
}

// LineNumberWithActiveBg sets the active line background color.
func LineNumberWithActiveBg(c style.Color) LineNumberOption {
	return func(ln *LineNumber) { ln.activeBg = c }
}

// LineNumberWithMinWidth sets the minimum gutter width.
func LineNumberWithMinWidth(w int) LineNumberOption {
	return func(ln *LineNumber) { ln.minWidth = w }
}

// LineNumberWithOffset sets the starting line number offset.
func LineNumberWithOffset(offset int) LineNumberOption {
	return func(ln *LineNumber) { ln.lineNumberOffset = offset }
}

// LineNumberWithShow enables/disables line number display.
func LineNumberWithShow(v bool) LineNumberOption {
	return func(ln *LineNumber) { ln.showLineNumbers = v }
}

// LineNumberWithMode sets the line number mode (absolute or relative).
func LineNumberWithMode(mode LineNumberMode) LineNumberOption {
	return func(ln *LineNumber) { ln.mode = mode }
}

// LineNumberWithSeparator sets the separator character.
func LineNumberWithSeparator(r rune) LineNumberOption {
	return func(ln *LineNumber) { ln.separator = r }
}

// LineNumberWithSeparatorFg sets the separator foreground color.
func LineNumberWithSeparatorFg(c style.Color) LineNumberOption {
	return func(ln *LineNumber) { ln.separatorFg = c }
}

// LineNumberWithLineColors sets per-line background colors.
func LineNumberWithLineColors(colors map[int]LineColorConfig) LineNumberOption {
	return func(ln *LineNumber) { ln.lineColors = colors }
}

// LineNumberWithLineSigns sets per-line sign indicators.
func LineNumberWithLineSigns(signs map[int]LineSign) LineNumberOption {
	return func(ln *LineNumber) {
		ln.lineSigns = signs
		ln.calculateSignWidths()
	}
}

// LineNumberWithLineNumbers sets custom line number mapping.
func LineNumberWithLineNumbers(numbers map[int]int) LineNumberOption {
	return func(ln *LineNumber) { ln.lineNumbers = numbers }
}

// LineNumberWithHideLineNumbers sets which lines hide their number.
func LineNumberWithHideLineNumbers(hide map[int]bool) LineNumberOption {
	return func(ln *LineNumber) { ln.hideLineNumbers = hide }
}

// LineNumberWithID sets the widget ID.
func LineNumberWithID(id string) LineNumberOption {
	return func(ln *LineNumber) { ln.SetID(id) }
}

// LineNumberWithWidth sets the widget width.
func LineNumberWithWidth(s layout.Scalar) LineNumberOption {
	return func(ln *LineNumber) { ln.LayoutNode().Width = s }
}

// LineNumberWithHeight sets the widget height.
func LineNumberWithHeight(s layout.Scalar) LineNumberOption {
	return func(ln *LineNumber) { ln.LayoutNode().Height = s }
}
