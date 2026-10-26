package widget

import (
	"strconv"
	"strings"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
	"github.com/unaquatic/fluid/textbuf"
)

// DiffViewMode controls how the diff is displayed.
type DiffViewMode int

const (
	DiffUnified DiffViewMode = iota // One sequential column.
	DiffSplit                       // Old and new content in side-by-side columns.
)

// diffLineType classifies a parsed diff line.
type diffLineType int

const (
	diffContext diffLineType = iota
	diffAdded
	diffRemoved
	diffHeader
	diffHunk
	diffEmpty // Padding row in split view.
)

// Diff displays a scrollable unified patch in sequential or side-by-side form.
// It supports line numbers, syntax highlighting, signs, and line backgrounds.
//
// Diff composes Code widgets for text with LineNumber widgets for gutters.
//
// Unified view uses one gutter and Code pane. Split view uses one pair per side.
type Diff struct {
	*render.BaseRenderable

	patch       string // Raw unified diff text.
	viewMode    DiffViewMode
	filetype    string            // Language used for syntax highlighting.
	theme       *style.Theme      // Theme whose code palette colors tokens.
	wrapMode    *textbuf.WrapMode // Nil uses Code's default textbuf.WrapWord mode.
	highlighter SyntaxHighlighter // Nil uses Code's default Highlighter.

	showLineNumbers bool
	inlineHighlight bool // Enables character-level change highlighting.
	showSigns       bool // Displays plus and minus signs in gutters.
	gutterSeparator bool // Separates the gutter from content with a vertical rule.
	splitSeparator  rune // Separates the two split panes.

	// Colors
	addedBg       style.Color
	removedBg     style.Color
	contextBg     style.Color
	headerFg      style.Color
	hunkFg        style.Color
	lineNumFg     style.Color
	addedFg       style.Color
	removedFg     style.Color
	contextFg     style.Color
	signAddedFg   style.Color
	signRemovedFg style.Color
	separatorFg   style.Color

	// Inline colors apply to changed portions within added and removed lines.
	inlineAddedBg   style.Color
	inlineRemovedBg style.Color

	// Scroll
	scrollY int
	scrollX int

	// parsed and layout are rebuilt from patch when needsRebuild is true.
	parsed *structuredPatch
	layout *diffLayout

	// Child widgets
	leftCode  *Code
	rightCode *Code
	leftSide  *LineNumber
	rightSide *LineNumber

	// splitVisualCount caches the wrapped split-view height from the last render.
	splitVisualCount int

	// needsRebuild marks parsed state and child widgets as stale.
	needsRebuild bool
}

// DiffOption configures a Diff widget.
type DiffOption func(*Diff)

// NewDiff returns a unified Diff for patch unless options select another view.
func NewDiff(patch string, opts ...DiffOption) *Diff {
	d := &Diff{
		BaseRenderable: render.NewBaseRenderable(
			render.WithFlexDirection(layout.DirectionColumn),
		),
		patch:           patch,
		viewMode:        DiffUnified,
		theme:           defaultTheme(),
		showLineNumbers: true,
		inlineHighlight: true,
		showSigns:       true,
		gutterSeparator: false,
		splitSeparator:  symbol.Vertical,

		addedBg:       style.ColorFromInts(22, 39, 27, 255),
		removedBg:     style.ColorFromInts(50, 20, 24, 255),
		contextBg:     style.ColorFromInts(30, 30, 30, 255),
		headerFg:      style.ColorBrightCyan,
		hunkFg:        style.ColorBrightMagenta,
		lineNumFg:     style.ColorGray,
		addedFg:       style.ColorBrightGreen,
		removedFg:     style.ColorBrightRed,
		contextFg:     style.ColorWhite,
		signAddedFg:   style.ColorBrightGreen,
		signRemovedFg: style.ColorBrightRed,
		separatorFg:   style.ColorGray,

		inlineAddedBg:   style.ColorFromInts(36, 65, 40, 255),
		inlineRemovedBg: style.ColorFromInts(80, 28, 32, 255),

		needsRebuild: true,
	}

	for _, opt := range opts {
		opt(d)
	}

	// Let scroll events bubble at an edge so an enclosing ScrollBox can move.
	d.OnMouseScrollHandler = func(e *input.MouseEvent) {
		if e.Scroll != nil {
			oldY := d.scrollY
			oldX := d.scrollX
			d.handleScroll(e.Scroll.DeltaX, e.Scroll.DeltaY)
			if d.scrollY != oldY || d.scrollX != oldX {
				e.StopPropagation()
			}
		}
	}

	d.rebuild()

	// Report patch height to containers that auto-size this widget.
	d.LayoutNode().Measure = func(
		width int, widthMode layout.MeasureMode,
		height int, heightMode layout.MeasureMode,
	) layout.MeasureResult {
		h := max(d.LineCount(), 1)
		w := width
		if widthMode == layout.MeasureUndefined {
			w = 0
		}
		if heightMode == layout.MeasureExactly {
			h = height
		} else if heightMode == layout.MeasureAtMost && h > height {
			h = height
		}
		return layout.MeasureResult{Width: w, Height: h}
	}

	return d
}

// Content and view configuration.

// SetPatch replaces the patch, resets scrolling, and rebuilds the view.
func (d *Diff) SetPatch(patch string) {
	if d.patch == patch {
		return
	}
	d.patch = patch
	d.scrollY = 0
	d.scrollX = 0
	d.needsRebuild = true
	d.rebuild()
	d.RequestRender()
}

// Patch returns the current patch.
func (d *Diff) Patch() string { return d.patch }

// SetViewMode changes the presentation mode and resets scrolling.
func (d *Diff) SetViewMode(mode DiffViewMode) {
	if d.viewMode == mode {
		return
	}
	d.viewMode = mode
	d.scrollY = 0
	d.scrollX = 0
	d.needsRebuild = true
	d.rebuild()
	d.RequestRender()
}

// ViewMode returns the current view mode.
func (d *Diff) ViewMode() DiffViewMode { return d.viewMode }

// SetFiletype sets the language used to highlight diff content.
func (d *Diff) SetFiletype(ft string) {
	if d.filetype == ft {
		return
	}
	d.filetype = ft
	d.needsRebuild = true
	d.rebuild()
	d.RequestRender()
}

// Filetype returns the current filetype.
func (d *Diff) Filetype() string { return d.filetype }

// SetTheme sets the theme whose code palette colors the diff's source. A
// nil theme selects style.DefaultDark.
func (d *Diff) SetTheme(t *style.Theme) {
	if t == nil {
		t = defaultTheme()
	}
	if d.theme == t {
		return
	}
	d.theme = t
	d.needsRebuild = true
	d.rebuild()
	d.RequestRender()
}

// Theme returns the theme whose code palette colors the diff's source.
// Without one set, it is a default theme that every such Diff shares, which
// must not be changed.
func (d *Diff) Theme() *style.Theme { return d.theme }

// LineCount returns the total number of display lines.
func (d *Diff) LineCount() int {
	if d.viewMode == DiffSplit {
		return d.splitVisualCount
	}
	if d.leftCode != nil {
		return d.leftCode.LineCount()
	}
	return 0
}

// Scrolling.

// ScrollY returns the vertical scroll offset.
func (d *Diff) ScrollY() int { return d.scrollY }

// ScrollX returns the horizontal scroll offset.
func (d *Diff) ScrollX() int { return d.scrollX }

// SetScrollY sets the vertical scroll offset, clamped to the current content.
func (d *Diff) SetScrollY(v int) {
	maxY := d.MaxScrollY()
	if v < 0 {
		v = 0
	}
	if v > maxY {
		v = maxY
	}
	if d.scrollY != v {
		d.scrollY = v
		// Keep child panes aligned with the compositor.
		if d.leftCode != nil {
			d.leftCode.SetScrollY(v)
		}
		if d.rightCode != nil {
			d.rightCode.SetScrollY(v)
		}
		d.RequestRender()
	}
}

// SetScrollX sets a non-negative horizontal offset on both content panes.
func (d *Diff) SetScrollX(v int) {
	if v < 0 {
		v = 0
	}
	if d.scrollX != v {
		d.scrollX = v
		if d.leftCode != nil {
			d.leftCode.SetScrollX(v)
		}
		if d.rightCode != nil {
			d.rightCode.SetScrollX(v)
		}
		d.RequestRender()
	}
}

// MaxScrollY returns the maximum vertical scroll offset.
func (d *Diff) MaxScrollY() int {
	result := d.LineCount() - d.Height()
	if result < 0 {
		return 0
	}
	return result
}

// ScrollToBottom scrolls to the last line.
func (d *Diff) ScrollToBottom() {
	d.SetScrollY(d.MaxScrollY())
}

func (d *Diff) handleScroll(deltaX, deltaY int) {
	d.SetScrollY(d.scrollY + deltaY)
	d.SetScrollX(d.scrollX + deltaX)
}

// OnResize marks width-dependent diff state for rebuilding.
func (d *Diff) OnResize(width, height int) {
	d.needsRebuild = true
}

// RenderSelf positions and draws the active diff view.
// Diff renders its Code and LineNumber children manually because pane widths
// and wrapped row pairing depend on the final viewport size.
func (d *Diff) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	if d.needsRebuild {
		d.rebuild()
		d.needsRebuild = false
	}

	w := d.Width()
	h := d.Height()
	x := d.X()
	y := d.Y()
	if w <= 0 || h <= 0 {
		return
	}

	if d.viewMode == DiffSplit {
		d.renderSplitView(buf, x, y, w, h, deltaTime)
	} else {
		d.renderUnifiedView(buf, x, y, w, h, deltaTime)
	}
}

// renderUnifiedView draws one full-width gutter and content pane.
func (d *Diff) renderUnifiedView(buf *buffer.Buffer, x, y, w, h int, deltaTime time.Duration) {
	if d.leftCode == nil || d.leftSide == nil {
		return
	}

	gutterW := d.leftSide.ComputedWidth()
	contentW := max(w-gutterW, 1)

	// Size Code first so LineNumber observes wrapping at the final content width.
	d.positionWidget(d.leftCode, x+gutterW, y, contentW, h)

	// Draw the gutter before content overlays its row backgrounds and text.
	d.positionWidget(d.leftSide, x, y, gutterW, h)
	d.leftSide.RenderSelf(buf, deltaTime)

	d.leftCode.RenderSelf(buf, deltaTime)
}

// renderSplitView draws headers across the viewport and content in paired panes.
// It wraps each side at render time using the final pane widths.
func (d *Diff) renderSplitView(buf *buffer.Buffer, x, y, w, h int, deltaTime time.Duration) {
	if d.layout == nil || len(d.layout.Blocks) == 0 {
		return
	}

	// Match Code's default wrap mode when no explicit mode was configured.
	effectiveWrap := textbuf.WrapWord
	if d.wrapMode != nil {
		effectiveWrap = *d.wrapMode
	}

	// Use one gutter width for both panes.
	gutterW := d.computeSplitGutterWidth()

	// Divide the viewport into gutter/content pairs around one separator column.
	halfW := max((w-1)/2, 1)
	rightHalfW := w - halfW - 1
	leftContentW := halfW - gutterW
	rightContentW := rightHalfW - gutterW
	if leftContentW < 1 {
		leftContentW = 1
	}
	if rightContentW < 1 {
		rightContentW = 1
	}
	fullContentW := max(
		// Full-width header rows use only the left gutter.
		w-gutterW, 1)

	sepX := x + halfW
	rightX := sepX + 1

	// Draw the separator once; full-width header rows overwrite it as needed.
	for row := range h {
		buf.SetCell(sepX, y+row, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)
	}

	visualRow := 0
	emptyLine := wrappedLine{} // Pads the shorter side of paired changes.

	for _, block := range d.layout.Blocks {
		switch block.Kind {
		case diffBlockFullWidth:
			// Headers and hunk markers span both content panes.
			fg := d.splitFgForType(block.Type)
			lines := splitWrapText(block.Text, fullContentW, effectiveWrap, fg, style.ColorTransparent)

			for _, line := range lines {
				screenRow := visualRow - d.scrollY
				if screenRow >= 0 && screenRow < h {
					ry := y + screenRow
					d.renderSplitGutter(buf, x, ry, gutterW, 0, true, diffEmpty, style.ColorTransparent)
					buf.SetCell(sepX, ry, ' ', style.ColorTransparent, style.ColorTransparent, 0)
					d.renderSplitLine(buf, x+gutterW, ry, w-gutterW, line, style.ColorTransparent)
				}
				visualRow++
			}

		case diffBlockContext:
			// Context rows repeat the unchanged content on both sides.
			bg := d.splitBgForType(diffContext)
			fg := d.splitFgForType(diffContext)

			leftLines := splitWrapText(block.Text, leftContentW, effectiveWrap, fg, bg)
			rightLines := splitWrapText(block.Text, rightContentW, effectiveWrap, fg, bg)

			maxLen := max(len(rightLines), len(leftLines))

			for j := range maxLen {
				screenRow := visualRow - d.scrollY
				if screenRow >= 0 && screenRow < h {
					ry := y + screenRow

					// Draw the old-file side.
					lType := diffContext
					if j > 0 {
						lType = diffEmpty
					}
					lNum := 0
					if j == 0 {
						lNum = block.OldLineNum
					}
					d.renderSplitGutter(buf, x, ry, gutterW, lNum, j > 0, lType, bg)
					ll := emptyLine
					llBg := bg
					if j < len(leftLines) {
						ll = leftLines[j]
					} else {
						llBg = style.ColorTransparent
					}
					d.renderSplitLine(buf, x+gutterW, ry, leftContentW, ll, llBg)

					// Restore the separator if row content painted over it.
					buf.SetCell(sepX, ry, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)

					// Draw the new-file side.
					rNum := 0
					if j == 0 {
						rNum = block.NewLineNum
					}
					d.renderSplitGutter(buf, rightX, ry, gutterW, rNum, j > 0, lType, bg)
					rl := emptyLine
					rlBg := bg
					if j < len(rightLines) {
						rl = rightLines[j]
					} else {
						rlBg = style.ColorTransparent
					}
					d.renderSplitLine(buf, rightX+gutterW, ry, rightContentW, rl, rlBg)
				}
				visualRow++
			}

		case diffBlockChange:
			// Pair removed and added lines by index, padding the shorter side.
			pairLen := max(len(block.Adds), len(block.Removes))

			for j := range pairLen {
				// The old-file side contains a removed line or padding.
				var leftContent string
				var leftLineNum int
				leftType := diffEmpty
				if j < len(block.Removes) {
					leftContent = block.Removes[j].Content
					leftLineNum = block.Removes[j].LineNum
					leftType = diffRemoved
				}

				// The new-file side contains an added line or padding.
				var rightContent string
				var rightLineNum int
				rightType := diffEmpty
				if j < len(block.Adds) {
					rightContent = block.Adds[j].Content
					rightLineNum = block.Adds[j].LineNum
					rightType = diffAdded
				}

				leftFg := d.splitFgForType(leftType)
				leftBg := d.splitBgForType(leftType)
				rightFg := d.splitFgForType(rightType)
				rightBg := d.splitBgForType(rightType)

				leftLines := splitWrapText(leftContent, leftContentW, effectiveWrap, leftFg, leftBg)
				rightLines := splitWrapText(rightContent, rightContentW, effectiveWrap, rightFg, rightBg)

				wrapMax := max(len(rightLines), len(leftLines))

				for k := range wrapMax {
					screenRow := visualRow - d.scrollY
					if screenRow >= 0 && screenRow < h {
						ry := y + screenRow

						// Draw the old-file side.
						lHide := leftType == diffEmpty || k > 0
						lNum := 0
						if k == 0 {
							lNum = leftLineNum
						}
						lType := leftType
						if k > 0 || leftType == diffEmpty {
							lType = diffEmpty
						}
						d.renderSplitGutter(buf, x, ry, gutterW, lNum, lHide, lType, leftBg)

						ll := emptyLine
						llBg := leftBg
						if k < len(leftLines) {
							ll = leftLines[k]
						} else {
							llBg = style.ColorTransparent
						}
						d.renderSplitLine(buf, x+gutterW, ry, leftContentW, ll, llBg)

						// Restore the pane separator.
						buf.SetCell(sepX, ry, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)

						// Draw the new-file side.
						rHide := rightType == diffEmpty || k > 0
						rNum := 0
						if k == 0 {
							rNum = rightLineNum
						}
						rType := rightType
						if k > 0 || rightType == diffEmpty {
							rType = diffEmpty
						}
						d.renderSplitGutter(buf, rightX, ry, gutterW, rNum, rHide, rType, rightBg)

						rl := emptyLine
						rlBg := rightBg
						if k < len(rightLines) {
							rl = rightLines[k]
						} else {
							rlBg = style.ColorTransparent
						}
						d.renderSplitLine(buf, rightX+gutterW, ry, rightContentW, rl, rlBg)
					}
					visualRow++
				}
			}
		}
	}

	d.splitVisualCount = visualRow
}

// Split-view rendering helpers.

// computeSplitGutterWidth returns the width needed for numbers, padding, and signs.
func (d *Diff) computeSplitGutterWidth() int {
	maxNum := 0
	if d.layout != nil {
		maxNum = d.layout.MaxLineNum
	}

	digits := max(len(strconv.Itoa(maxNum)), 2)

	w := digits + 1 // One leading padding column.
	if d.showSigns {
		w += 3 // Width of " + " or " - ".
	} else {
		w += 1 // One trailing padding column.
	}
	return w
}

// splitFgForType returns the foreground color for a line type in split view.
func (d *Diff) splitFgForType(lt diffLineType) style.Color {
	switch lt {
	case diffHeader:
		return d.headerFg
	case diffHunk:
		return d.hunkFg
	case diffEmpty:
		return style.ColorTransparent
	default:
		return d.contextFg
	}
}

// splitBgForType returns the background color for a line type in split view.
func (d *Diff) splitBgForType(lt diffLineType) style.Color {
	switch lt {
	case diffAdded:
		return d.addedBg
	case diffRemoved:
		return d.removedBg
	case diffContext:
		return d.contextBg
	default:
		return style.ColorTransparent
	}
}

// renderSplitGutter draws one gutter row with an optional line number and sign.
func (d *Diff) renderSplitGutter(buf *buffer.Buffer, bx, by, gutterW, lineNum int, hideNum bool, lt diffLineType, bg style.Color) {
	// Fill background.
	if !bg.IsTransparent() {
		buf.FillRect(bx, by, gutterW, 1, bg)
	}

	signW := 0
	if d.showSigns {
		signW = 3
	}
	numFieldW := gutterW - signW

	// Right-align the line number before the sign field.
	if d.showLineNumbers && !hideNum && lineNum > 0 {
		numStr := strconv.Itoa(lineNum)
		startCol := max(numFieldW-len(numStr), 0)
		for i, r := range numStr {
			buf.SetCell(bx+startCol+i, by, r, d.lineNumFg, bg, 0)
		}
	}

	// Draw a fixed-width sign field for added, removed, or empty rows.
	if d.showSigns {
		sign := "   "
		signFg := style.ColorTransparent
		switch lt {
		case diffAdded:
			sign = " + "
			signFg = d.signAddedFg
		case diffRemoved:
			sign = " - "
			signFg = d.signRemovedFg
		}
		for i, r := range sign {
			fg := signFg
			if fg.A == 0 {
				fg = d.lineNumFg // Use the gutter color for an empty sign field.
			}
			buf.SetCell(bx+numFieldW+i, by, r, fg, bg, 0)
		}
	}
}

// renderSplitLine draws one wrapped line and fills unused cells with fallbackBg.
func (d *Diff) renderSplitLine(buf *buffer.Buffer, bx, by, width int, line wrappedLine, fallbackBg style.Color) {
	if !fallbackBg.IsTransparent() {
		buf.FillRect(bx, by, width, 1, fallbackBg)
	}
	col := 0
	for _, chunk := range line.chunks {
		col += buf.DrawClusters(chunk.text, bx+col, by, width-col, chunk.fg, chunk.bg, chunk.attributes, nil)
	}
}

// splitWrapText applies the shared Text wrapping functions to one diff line.
func splitWrapText(text string, width int, mode textbuf.WrapMode, fg, bg style.Color) []wrappedLine {
	segments := []styledSegment{{text: text, fg: fg, bg: bg}}

	if mode == textbuf.WrapNone || width <= 0 {
		return []wrappedLine{segmentsToWrappedLine(segments)}
	}

	switch mode {
	case textbuf.WrapChar:
		return wrapSegmentsChar(segments, width)
	case textbuf.WrapWord:
		return wrapSegmentsWord(segments, width)
	default:
		return []wrappedLine{segmentsToWrappedLine(segments)}
	}
}

// positionWidget applies a computed rectangle to a manually rendered child.
func (d *Diff) positionWidget(w render.Renderable, x, y, width, height int) {
	ln := w.LayoutNode()
	ln.Computed.X = x
	ln.Computed.Y = y
	ln.Computed.Width = width
	ln.Computed.Height = height
	w.UpdateFromLayout()
}

// Parsed-state and child-widget rebuilding.

func (d *Diff) rebuild() {
	// Discard child widgets tied to the previous parsed layout.
	d.removeChildren()

	// Parse once, then build the children required by the selected view.
	d.parsed = nil
	d.layout = nil
	if d.patch != "" {
		d.parsed = parsePatch(d.patch)
		d.layout = buildDiffLayout(d.parsed)

		if d.viewMode == DiffSplit {
			d.buildSplitView()
		} else {
			d.buildUnifiedView()
		}
	}

	d.needsRebuild = false
}

func (d *Diff) removeChildren() {
	if d.leftSide != nil {
		d.Remove(d.leftSide.ID())
		d.leftSide = nil
	}
	if d.rightSide != nil {
		d.Remove(d.rightSide.ID())
		d.rightSide = nil
	}
	d.leftCode = nil
	d.rightCode = nil
}

func (d *Diff) buildUnifiedView() {
	d.LayoutNode().FlexDirection = layout.DirectionColumn

	if d.layout == nil || len(d.layout.Blocks) == 0 {
		return
	}

	// Flatten blocks in unified-diff order, with removals before additions.
	var contentLines []string
	lineColors := make(map[int]LineColorConfig)
	lineSigns := make(map[int]LineSign)
	lineNumbers := make(map[int]int)
	hideLineNums := make(map[int]bool)
	lineIndex := 0

	for _, block := range d.layout.Blocks {
		switch block.Kind {
		case diffBlockFullWidth:
			// File and hunk headers do not display source line numbers.
			contentLines = append(contentLines, block.Text)
			hideLineNums[lineIndex] = true
			lineIndex++

		case diffBlockContext:
			// Context rows display their new-file line number.
			contentLines = append(contentLines, block.Text)
			if !d.contextBg.IsTransparent() {
				lineColors[lineIndex] = LineColorConfig{
					Gutter:  d.contextBg,
					Content: d.contextBg,
				}
			}
			lineNumbers[lineIndex] = block.NewLineNum
			lineIndex++

		case diffBlockChange:
			// Emit all removals before additions in each change block.
			for _, rm := range block.Removes {
				contentLines = append(contentLines, rm.Content)
				lineColors[lineIndex] = LineColorConfig{
					Gutter:  d.removedBg,
					Content: d.removedBg,
				}
				lineSigns[lineIndex] = LineSign{
					After:      " - ",
					AfterColor: d.signRemovedFg,
				}
				lineNumbers[lineIndex] = rm.LineNum
				lineIndex++
			}
			for _, add := range block.Adds {
				contentLines = append(contentLines, add.Content)
				lineColors[lineIndex] = LineColorConfig{
					Gutter:  d.addedBg,
					Content: d.addedBg,
				}
				lineSigns[lineIndex] = LineSign{
					After:      " + ",
					AfterColor: d.signAddedFg,
				}
				lineNumbers[lineIndex] = add.LineNum
				lineIndex++
			}
		}
	}

	content := strings.Join(contentLines, "\n")

	// Create the unified content pane.
	codeOpts := []CodeOption{
		CodeWithFiletype(d.filetype),
		CodeWithTheme(d.theme),
		CodeWithFg(d.contextFg),
		CodeWithBg(style.ColorTransparent),
		CodeWithLayout(
			layout.WithWidth(layout.Percent(100)),
			layout.WithHeight(layout.Percent(100)),
		),
		CodeWithScrollY(d.scrollY),
		CodeWithScrollX(d.scrollX),
	}
	if d.wrapMode != nil {
		codeOpts = append(codeOpts, CodeWithWrapMode(*d.wrapMode))
	}
	if d.highlighter != nil {
		codeOpts = append(codeOpts, CodeWithHighlighter(d.highlighter))
	}
	d.leftCode = NewCode(content, codeOpts...)

	// Give Code the content-area background for each logical line.
	codeLineColors := make(map[int]style.Color)
	for idx, lc := range lineColors {
		if !lc.Content.IsTransparent() {
			codeLineColors[idx] = lc.Content
		}
	}
	d.leftCode.SetLineColors(codeLineColors)

	// Omit sign metadata entirely when gutter signs are disabled.
	if !d.showSigns {
		lineSigns = nil
	}

	// Reserve at least three digits plus the sign or trailing-space field.
	minGutterW := 3
	if d.showSigns {
		minGutterW += 3 // Width of " + " or " - ".
	} else {
		minGutterW += 1 // One trailing padding column.
	}

	d.leftSide = NewLineNumber(d.leftCode,
		LineNumberWithFg(d.lineNumFg),
		LineNumberWithOffset(0), // lineNumbers supplies every displayed number.
		LineNumberWithMinWidth(minGutterW),
		LineNumberWithShow(d.showLineNumbers),
		LineNumberWithLineColors(lineColors),
		LineNumberWithLineSigns(lineSigns),
		LineNumberWithLineNumbers(lineNumbers),
		LineNumberWithHideLineNumbers(hideLineNums),
		LineNumberWithLayout(
			layout.WithWidth(layout.Percent(100)),
			layout.WithHeight(layout.Percent(100)),
		),
	)

	if d.gutterSeparator {
		d.leftSide.SetSeparator(symbol.Vertical)
		d.leftSide.separatorFg = d.separatorFg
	}

	d.Add(d.leftSide)
}

func (d *Diff) buildSplitView() {
	d.LayoutNode().FlexDirection = layout.DirectionRow
	d.splitVisualCount = 0
}

// inlineDiffSegment contains text and whether it differs from the opposite side.
type inlineDiffSegment struct {
	text    string
	changed bool
}

// computeInlineDiff splits two strings into their common prefix, changed middle, and common suffix.
func computeInlineDiff(oldText, newText string) (oldSegs, newSegs []inlineDiffSegment) {
	oldRunes := []rune(oldText)
	newRunes := []rune(newText)

	// Find the longest shared rune prefix.
	prefixLen := 0
	for prefixLen < len(oldRunes) && prefixLen < len(newRunes) && oldRunes[prefixLen] == newRunes[prefixLen] {
		prefixLen++
	}

	// Find the longest shared suffix that does not overlap the prefix.
	suffixLen := 0
	for suffixLen < len(oldRunes)-prefixLen && suffixLen < len(newRunes)-prefixLen &&
		oldRunes[len(oldRunes)-1-suffixLen] == newRunes[len(newRunes)-1-suffixLen] {
		suffixLen++
	}

	// Emit the shared prefix on both sides.
	if prefixLen > 0 {
		prefix := string(oldRunes[:prefixLen])
		oldSegs = append(oldSegs, inlineDiffSegment{text: prefix, changed: false})
		newSegs = append(newSegs, inlineDiffSegment{text: prefix, changed: false})
	}

	// Mark the unmatched middle portions as changed.
	oldMiddle := string(oldRunes[prefixLen : len(oldRunes)-suffixLen])
	newMiddle := string(newRunes[prefixLen : len(newRunes)-suffixLen])
	if len(oldMiddle) > 0 {
		oldSegs = append(oldSegs, inlineDiffSegment{text: oldMiddle, changed: true})
	}
	if len(newMiddle) > 0 {
		newSegs = append(newSegs, inlineDiffSegment{text: newMiddle, changed: true})
	}

	// Emit the shared suffix on both sides.
	if suffixLen > 0 {
		suffix := string(oldRunes[len(oldRunes)-suffixLen:])
		oldSegs = append(oldSegs, inlineDiffSegment{text: suffix, changed: false})
		newSegs = append(newSegs, inlineDiffSegment{text: suffix, changed: false})
	}

	return oldSegs, newSegs
}

// Diff options.

// DiffWithViewMode sets the view mode.
func DiffWithViewMode(mode DiffViewMode) DiffOption {
	return func(d *Diff) { d.viewMode = mode }
}

// DiffWithFiletype sets the filetype for syntax highlighting within diff.
func DiffWithFiletype(ft string) DiffOption {
	return func(d *Diff) { d.filetype = ft }
}

// DiffWithTheme sets the theme whose code palette colors the diff's source.
// A nil theme selects style.DefaultDark.
func DiffWithTheme(t *style.Theme) DiffOption {
	return func(d *Diff) {
		if t == nil {
			t = defaultTheme()
		}
		d.theme = t
	}
}

// DiffWithShowLineNumbers controls whether gutters display line numbers.
func DiffWithShowLineNumbers(v bool) DiffOption {
	return func(d *Diff) { d.showLineNumbers = v }
}

// DiffWithInlineHighlight controls character-level highlighting within changed lines.
func DiffWithInlineHighlight(v bool) DiffOption {
	return func(d *Diff) { d.inlineHighlight = v }
}

// DiffWithShowSigns controls whether gutters display plus and minus signs.
func DiffWithShowSigns(v bool) DiffOption {
	return func(d *Diff) { d.showSigns = v }
}

// DiffWithAddedBg sets the added line background color.
func DiffWithAddedBg(c style.Color) DiffOption {
	return func(d *Diff) { d.addedBg = c }
}

// DiffWithRemovedBg sets the removed line background color.
func DiffWithRemovedBg(c style.Color) DiffOption {
	return func(d *Diff) { d.removedBg = c }
}

// DiffWithContextBg sets the context line background color.
func DiffWithContextBg(c style.Color) DiffOption {
	return func(d *Diff) { d.contextBg = c }
}

// DiffWithScrollY sets the initial vertical scroll offset.
func DiffWithScrollY(v int) DiffOption {
	return func(d *Diff) { d.scrollY = v }
}

// DiffWithID sets the widget ID.
func DiffWithID(id string) DiffOption {
	return func(d *Diff) { d.SetID(id) }
}

// DiffWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func DiffWithLayout(opts ...layout.Option) DiffOption {
	return func(w *Diff) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}

// DiffWithWrapMode sets the content wrapping mode; the default is textbuf.WrapWord.
func DiffWithWrapMode(m textbuf.WrapMode) DiffOption {
	return func(d *Diff) { d.wrapMode = &m }
}

// DiffWithHighlighter sets a custom SyntaxHighlighter for the diff and its
// child Code widgets. When nil (default), a Highlighter of the diff's
// theme is used.
func DiffWithHighlighter(h SyntaxHighlighter) DiffOption {
	return func(d *Diff) { d.highlighter = h }
}
