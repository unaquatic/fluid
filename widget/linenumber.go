package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// LineCounter reports the logical line count displayed by a target widget.
type LineCounter interface {
	LineCount() int
}

// Scroller reports a target widget's vertical scroll offset.
type Scroller interface {
	ScrollY() int
}

// CursorTracker reports the cursor position of a target widget, whose
// logical row LineNumber styles as the active line.
type CursorTracker interface {
	// CursorPosition returns the cursor's logical row and display column.
	CursorPosition() (row, col int)
}

// LineMapper maps the wrapped visual lines a target widget shows to its
// logical source lines. LineNumber leaves continuation rows blank when a
// target implements this interface.
type LineMapper interface {
	VisualLineCount() int
	// LogicalLineForVisualLine returns the logical line for a visual line index.
	LogicalLineForVisualLine(visualLine int) int
}

// LineNumberMode controls how line numbers are displayed.
type LineNumberMode int

const (
	LineNumberAbsolute LineNumberMode = iota // Displays absolute line numbers.
	LineNumberRelative                       // Displays distance from the active line.
)

// LineSign configures text displayed before and after a line number.
type LineSign struct {
	// Before is the text before the line number, such as a bookmark marker.
	Before string
	// BeforeColor is the foreground color for Before.
	BeforeColor style.Color
	// After is the text after the line number, such as a diff sign.
	After string
	// AfterColor is the foreground color for After.
	AfterColor style.Color
}

// LineColorConfig contains the gutter and content backgrounds for one logical line.
type LineColorConfig struct {
	// Gutter is the background behind the line number and signs.
	Gutter style.Color
	// Content is the background behind the target content.
	Content style.Color
}

// LineNumber draws a gutter synchronized with a target renderable.
// It supports wrapped and relative numbering, active-line styles, signs,
// custom numbers, and per-line gutter or content backgrounds.
type LineNumber struct {
	*render.BaseRenderable

	target           render.Renderable // Widget that supplies line, scroll, and cursor data.
	fg               style.Color
	bg               style.Color
	activeFg         style.Color // Foreground for the active line number.
	activeBg         style.Color // Background for the active gutter row.
	minWidth         int
	lineNumberOffset int  // Added to zero-based logical line indexes; defaults to one.
	showLineNumbers  bool // Controls gutter visibility; defaults to true.
	mode             LineNumberMode
	separator        rune // Character after the gutter; zero disables it.
	separatorFg      style.Color

	// Per-line maps use zero-based logical line indexes.
	lineColors      map[int]LineColorConfig
	lineSigns       map[int]LineSign
	lineNumbers     map[int]int
	hideLineNumbers map[int]bool

	// Sign widths reserve stable columns across every gutter row.
	maxBeforeWidth int
	maxAfterWidth  int

	// drawn is the state of the target the gutter last drew, which the
	// target can change without the gutter, as by scrolling.
	drawn targetState
	// measuredWidth is the width the gutter reported when the layout last
	// measured it, or -1 when the layout never has, as when options set both
	// its width and height.
	measuredWidth int
}

// targetState holds what of a LineNumber's target its gutter shows.
type targetState struct {
	lineCount, scrollY, activeLine, visualLines, height int
}

// LineNumberOption configures a LineNumber widget.
type LineNumberOption func(*LineNumber)

// NewLineNumber returns a gutter synchronized with target.
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
		measuredWidth:    -1,
	}

	// Size the gutter to its numbers when no option sets its width.
	ln.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		w := ln.ComputedWidth()
		ln.measuredWidth = w
		h := height
		if heightMode == layout.MeasureUndefined {
			h = ln.getLineCount()
		}
		return layout.MeasureResult{Width: w, Height: h}
	}

	for _, opt := range opts {
		opt(ln)
	}

	return ln
}

// targetState returns the state of the target that the gutter shows.
func (ln *LineNumber) targetState() targetState {
	st := targetState{activeLine: -1}
	if ln.target == nil {
		return st
	}
	st.lineCount = ln.getLineCount()
	st.scrollY = ln.getScrollY()
	st.activeLine = ln.getActiveLine()
	if m, ok := ln.target.(LineMapper); ok {
		st.visualLines = m.VisualLineCount()
	}
	st.height = ln.target.Height()
	return st
}

// targetChanged reports whether the target changed what the gutter shows
// since it last drew, as by scrolling or moving its cursor. A gutter with no
// area shows nothing, and never draws, so its target cannot change it.
func (ln *LineNumber) targetChanged() bool {
	return ln.Width() > 0 && ln.Height() > 0 && ln.targetState() != ln.drawn
}

// remeasure lays the gutter out again when the width of its numbers no
// longer matches the width the layout measured, as when the target's line
// count gains a digit or a setter adds a sign column. It reports whether it
// did.
func (ln *LineNumber) remeasure() bool {
	w := ln.ComputedWidth()
	if ln.measuredWidth < 0 || w == ln.measuredWidth {
		return false
	}
	ln.measuredWidth = w
	ln.LayoutNode().MarkDirty()
	return true
}

// IsDirty reports whether the gutter needs repainting: whether the widget
// changed, its target changed what the gutter shows, or the layout resized
// the gutter, as when the target's line count gains a digit, without the
// gutter taking the new size yet.
func (ln *LineNumber) IsDirty() bool {
	c := ln.LayoutNode().Computed
	return ln.BaseRenderable.IsDirty() || ln.targetChanged() ||
		c.Width != ln.Width() || c.Height != ln.Height()
}

// Damage returns the region to repaint, which covers the gutter when its
// target changed what the gutter shows since it last drew.
func (ln *LineNumber) Damage() buffer.Rect {
	d := ln.BaseRenderable.Damage()
	if ln.Visible() && !ln.IsDestroyed() && ln.targetChanged() {
		d = d.Union(ln.Rect())
	}
	return d
}

// SetTarget changes the widget that supplies line, scroll, and cursor data.
func (ln *LineNumber) SetTarget(target render.Renderable) {
	ln.target = target
	ln.remeasure()
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

// SetShowLineNumbers controls whether the gutter is displayed.
func (ln *LineNumber) SetShowLineNumbers(v bool) {
	ln.showLineNumbers = v
	ln.remeasure()
	ln.RequestRender()
}

// ShowLineNumbers returns whether line numbers are shown.
func (ln *LineNumber) ShowLineNumbers() bool { return ln.showLineNumbers }

// Per-line customization.

// SetLineColors sets per-line background colors for gutter and content areas.
func (ln *LineNumber) SetLineColors(colors map[int]LineColorConfig) {
	ln.lineColors = colors
	ln.RequestRender()
}

// SetLineSigns sets per-line sign indicators.
func (ln *LineNumber) SetLineSigns(signs map[int]LineSign) {
	ln.lineSigns = signs
	ln.calculateSignWidths()
	ln.remeasure()
	ln.RequestRender()
}

// SetLineNumbers maps zero-based logical lines to displayed numbers.
func (ln *LineNumber) SetLineNumbers(numbers map[int]int) {
	ln.lineNumbers = numbers
	ln.remeasure()
	ln.RequestRender()
}

// SetHideLineNumbers sets the zero-based logical lines whose numbers are hidden.
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

// ComputedWidth returns the gutter width required by numbers, signs, and the separator.
func (ln *LineNumber) ComputedWidth() int {
	if !ln.showLineNumbers || ln.target == nil {
		return 0
	}

	lineCount := ln.getLineCount()
	maxLine := lineCount + ln.lineNumberOffset

	// Include custom displayed numbers when finding the digit width.
	for _, num := range ln.lineNumbers {
		if num > maxLine {
			maxLine = num
		}
	}

	digits := max(len(fmt.Sprintf("%d", maxLine)), 1)

	w := digits
	if ln.maxAfterWidth > 0 {
		// An after-sign field also provides separation from the content.
		w += ln.maxAfterWidth
	} else {
		w += 1 // Separate a number from adjacent content when no sign follows it.
	}
	w += ln.maxBeforeWidth
	if ln.separator != 0 {
		w++ // Reserve the explicit separator column.
	}
	if w < ln.minWidth {
		w = ln.minWidth
	}
	return w
}

// LineNumber rendering.

// RenderSelf draws the line numbers and per-line content backgrounds.
func (ln *LineNumber) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	ln.drawn = ln.targetState()

	// The target changes its line count without the gutter, so only drawing
	// finds the width it needs. The next frame lays the gutter out at it.
	if ln.remeasure() {
		ln.RequestRender()
	}

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

	// Query optional target interfaces once for this render.
	lineCount := ln.getLineCount()

	scrollY := ln.getScrollY()

	activeLine := ln.getActiveLine()

	// Wrapped targets provide visual-to-logical line mapping.
	hasVisualLines := false
	var lm LineMapper
	if m, ok := ln.target.(LineMapper); ok {
		hasVisualLines = true
		lm = m
	}

	// numWidth excludes sign and separator columns.
	numWidth := ln.computeNumWidth(lineCount)
	sepWidth := 0
	if ln.separator != 0 {
		sepWidth = 1
	}

	// ComputedWidth locates the target content area for row backgrounds. A
	// gutter narrower than its numbers, as for the frame before the layout
	// widens it, clips them at its edge.
	gutterW := min(ln.ComputedWidth(), w)

	// Paint configured target-row backgrounds before drawing the gutter.
	if ln.lineColors != nil && ln.showLineNumbers {
		ln.drawContentBackgrounds(buf, x, y, w, h, gutterW, scrollY, lineCount, hasVisualLines, lm)
	}

	if !ln.showLineNumbers {
		return
	}

	// Draw one gutter row for each visible target row.
	for row := range h {
		visualLine := scrollY + row
		lineY := y + row

		// Resolve the logical line and whether this is its first visual row.
		logicalLine := -1
		isFirstVisualLine := true

		if hasVisualLines {
			totalVisual := lm.VisualLineCount()
			if visualLine < totalVisual {
				logicalLine = lm.LogicalLineForVisualLine(visualLine)
				if visualLine > 0 {
					prevLogical := lm.LogicalLineForVisualLine(visualLine - 1)
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

		// Apply a line-specific gutter background when configured.
		rowBg := style.ColorTransparent
		if lc, ok := ln.lineColors[logicalLine]; ok && !lc.Gutter.IsTransparent() {
			rowBg = lc.Gutter
			buf.FillRect(x, lineY, gutterW, 1, rowBg)
		}

		isActive := activeLine >= 0 && logicalLine == activeLine

		// Active-line styling takes precedence over per-line styling.
		if isActive && !ln.activeBg.IsTransparent() {
			rowBg = ln.activeBg
			buf.FillRect(x, lineY, w, 1, rowBg)
		}

		// Wrapped continuation and explicitly hidden rows keep only their background.
		if ln.hideLineNumbers[logicalLine] || !isFirstVisualLine {
			continue
		}

		fg := ln.fg
		if isActive {
			fg = ln.activeFg
		}

		cx := x

		// Right-align the before-sign in its reserved field.
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

		// Select an absolute, relative, or custom displayed number.
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

		// Right-align the number. An after-sign field replaces normal trailing space.
		trailingSpace := " "
		if ln.maxAfterWidth > 0 {
			trailingSpace = ""
		}
		availDigits := numWidth
		if trailingSpace != "" {
			availDigits-- // Reserve the trailing separator space.
		}
		if availDigits < 1 {
			availDigits = 1
		}
		padded := strings.Repeat(" ", max(availDigits-len(numStr), 0)) + numStr + trailingSpace
		for _, r := range padded {
			if cx-x >= gutterW-sepWidth-ln.maxAfterWidth {
				break
			}
			buf.SetCell(cx, lineY, r, fg, rowBg, 0)
			cx++
		}

		// Draw the after-sign in its reserved field.
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

		// Draw the explicit separator after all number and sign fields.
		if ln.separator != 0 && cx-x < gutterW {
			buf.SetCell(cx, lineY, ln.separator, ln.separatorFg, rowBg, 0)
		}
	}
}

// drawContentBackgrounds paints configured target-row backgrounds to the right of the gutter.
func (ln *LineNumber) drawContentBackgrounds(buf *buffer.Buffer, x, y, w, h, gutterW, scrollY, lineCount int, hasVisualLines bool, lm LineMapper) {
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
			totalVisual := lm.VisualLineCount()
			if visualLine < totalVisual {
				logicalLine = lm.LogicalLineForVisualLine(visualLine)
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

// getLineCount returns the target's logical line count, or zero when unsupported.
func (ln *LineNumber) getLineCount() int {
	if lc, ok := ln.target.(LineCounter); ok {
		return lc.LineCount()
	}
	return 0
}

// getScrollY returns the target's vertical offset, or zero when unsupported.
func (ln *LineNumber) getScrollY() int {
	if sc, ok := ln.target.(Scroller); ok {
		return sc.ScrollY()
	}
	return 0
}

// getActiveLine returns the target's cursor row, or -1 when unsupported.
func (ln *LineNumber) getActiveLine() int {
	if ct, ok := ln.target.(CursorTracker); ok {
		row, _ := ct.CursorPosition()
		return row
	}
	return -1
}

// computeNumWidth returns the number field width available inside the widget.
func (ln *LineNumber) computeNumWidth(lineCount int) int {
	maxLine := lineCount + ln.lineNumberOffset

	// Include custom displayed numbers when finding the digit width.
	for _, num := range ln.lineNumbers {
		if num > maxLine {
			maxLine = num
		}
	}

	digits := max(len(fmt.Sprintf("%d", maxLine)), 1)
	numWidth := digits
	if ln.maxAfterWidth > 0 {
		// The after-sign field separates the number from content.
	} else {
		numWidth++ // Reserve one trailing separator space.
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

// LineNumber appearance.

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
	ln.remeasure()
	ln.RequestRender()
}

// LineNumber options.

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

// LineNumberWithShow controls whether the gutter is displayed.
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

// LineNumberWithLineNumbers maps zero-based logical lines to displayed numbers.
func LineNumberWithLineNumbers(numbers map[int]int) LineNumberOption {
	return func(ln *LineNumber) { ln.lineNumbers = numbers }
}

// LineNumberWithHideLineNumbers sets the zero-based logical lines whose numbers are hidden.
func LineNumberWithHideLineNumbers(hide map[int]bool) LineNumberOption {
	return func(ln *LineNumber) { ln.hideLineNumbers = hide }
}

// LineNumberWithID sets the widget ID.
func LineNumberWithID(id string) LineNumberOption {
	return func(ln *LineNumber) { ln.SetID(id) }
}

// LineNumberWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func LineNumberWithLayout(opts ...layout.Option) LineNumberOption {
	return func(w *LineNumber) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}
