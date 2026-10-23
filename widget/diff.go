package widget

import (
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// DiffViewMode controls how the diff is displayed.
type DiffViewMode int

const (
	DiffUnified DiffViewMode = iota // Unified diff view
	DiffSplit                       // Side-by-side split view
)

// diffLineType classifies a parsed diff line.
type diffLineType int

const (
	diffContext diffLineType = iota
	diffAdded
	diffRemoved
	diffHeader
	diffHunk
	diffEmpty // padding line in split view
)

// Diff displays a unified or split diff with color-coded lines, line numbers,
// scrolling, and optional syntax highlighting + inline word diff.
//
// Architecture: Diff is a compositor that delegates rendering to child widgets:
//   - Code widget(s) handle text rendering (tabs, wrapping, syntax highlighting)
//   - LineNumber widget(s) handle gutter rendering (line numbers, signs, per-line bg)
//
// Unified view:  LineNumber → Code (100% width)
// Split view:    LineNumber → Code (50%) │ LineNumber → Code (50%)
type Diff struct {
	*render.BaseRenderable

	patch       string // raw unified diff text
	viewMode    DiffViewMode
	filetype    string            // language for syntax highlighting
	theme       string            // theme name
	wrapMode    *WrapMode         // nil means inherit Code default (WrapWord)
	highlighter SyntaxHighlighter // nil means use default (ChromaHighlighter)

	showLineNumbers bool
	inlineHighlight bool // word-level inline diff highlighting
	showSigns       bool // show +/- signs in gutter
	gutterSeparator bool // show vertical separator between gutter and content
	splitSeparator  rune // separator character between split panels

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

	// Inline highlight colors
	inlineAddedBg   style.Color
	inlineRemovedBg style.Color

	// Scroll
	scrollY int
	scrollX int

	// Parsed state
	parsed *StructuredPatch
	layout *DiffLayout // view-independent layout (drives both unified & split)

	// Child widgets
	leftCode  *Code
	rightCode *Code
	leftSide  *LineNumber
	rightSide *LineNumber

	// Split view: total visual rows after wrapping (cached from last render).
	splitVisualCount int

	// Rebuilt state
	needsRebuild bool
}

// DiffOption configures a Diff widget.
type DiffOption func(*Diff)

// NewDiff creates a Diff widget.
func NewDiff(patch string, opts ...DiffOption) *Diff {
	d := &Diff{
		BaseRenderable: render.NewBaseRenderable(
			render.WithFlexDirection(layout.DirectionColumn),
		),
		patch:           patch,
		viewMode:        DiffUnified,
		theme:           "monokai",
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

	// Set up mouse scroll handler.
	// Only stop propagation if the scroll actually moved; when at a
	// boundary (top/bottom) let the event bubble to the parent so an
	// enclosing ScrollBox can scroll instead.
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

	// Set up intrinsic measurement so the layout engine can auto-size.
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

// ---------------------------------------------------------------------------
// Content Management
// ---------------------------------------------------------------------------

// SetPatch sets the diff patch content.
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

// SetViewMode sets the view mode.
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

// SetFiletype sets the language for syntax highlighting within diff.
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

// SetTheme sets the theme for syntax highlighting.
func (d *Diff) SetTheme(name string) {
	if d.theme == name {
		return
	}
	d.theme = name
	d.needsRebuild = true
	d.rebuild()
	d.RequestRender()
}

// Theme returns the current theme.
func (d *Diff) Theme() string { return d.theme }

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

// ---------------------------------------------------------------------------
// Scroll (implements ScrollInfoProvider)
// ---------------------------------------------------------------------------

// ScrollY returns the vertical scroll offset.
func (d *Diff) ScrollY() int { return d.scrollY }

// ScrollX returns the horizontal scroll offset.
func (d *Diff) ScrollX() int { return d.scrollX }

// SetScrollY sets the vertical scroll offset (clamped).
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
		// Propagate to child Code widgets
		if d.leftCode != nil {
			d.leftCode.SetScrollY(v)
		}
		if d.rightCode != nil {
			d.rightCode.SetScrollY(v)
		}
		d.RequestRender()
	}
}

// SetScrollX sets the horizontal scroll offset (clamped).
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

// OnResize invalidates state when size changes.
func (d *Diff) OnResize(width, height int) {
	d.needsRebuild = true
}

// RenderSelf orchestrates child widget rendering.
// Diff manually positions and renders its Code + LineNumber children
// rather than relying on the layout pipeline, ensuring correct behavior
// both in production and in test scenarios.
func (d *Diff) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
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

// renderUnifiedView renders the unified view: one LineNumber+Code pair at full width.
func (d *Diff) renderUnifiedView(buf *buffer.Buffer, x, y, w, h int, deltaTime float64) {
	if d.leftCode == nil || d.leftSide == nil {
		return
	}

	gutterW := d.leftSide.ComputedWidth()
	contentW := max(w-gutterW, 1)

	// Position Code first so wrapping is computed at the correct width
	// before LineNumber queries VisualLineCount / LogicalLineForVisualLine.
	d.positionWidget(d.leftCode, x+gutterW, y, contentW, h)

	// Position and render gutter (LineNumber)
	d.positionWidget(d.leftSide, x, y, gutterW, h)
	d.leftSide.RenderSelf(buf, deltaTime)

	// Render content (Code)
	d.leftCode.RenderSelf(buf, deltaTime)
}

// renderSplitView renders the split view from DiffLayout blocks.
// Headers/hunks span the full width; context and change blocks are rendered
// in two columns with a separator. Wrapping is computed at render time when
// the actual width is known.
func (d *Diff) renderSplitView(buf *buffer.Buffer, x, y, w, h int, deltaTime float64) {
	if d.layout == nil || len(d.layout.Blocks) == 0 {
		return
	}

	// Determine effective wrap mode.
	effectiveWrap := WrapWord
	if d.wrapMode != nil {
		effectiveWrap = *d.wrapMode
	}

	// Compute gutter width from pre-computed max line number.
	gutterW := d.computeSplitGutterWidth()

	// Layout: left_gutter | left_content | separator(1) | right_gutter | right_content
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
		// for full-width header rows
		w-gutterW, 1)

	sepX := x + halfW
	rightX := sepX + 1

	// Draw separator column for the full viewport height.
	// Full-width rows will overwrite this.
	for row := range h {
		buf.SetCell(sepX, y+row, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)
	}

	visualRow := 0
	emptyLine := wrappedLine{} // empty line for padding rows

	for _, block := range d.layout.Blocks {
		switch block.Kind {
		case DiffBlockFullWidth:
			// Header/hunk: wrap at full width, render spanning both columns.
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

		case DiffBlockContext:
			// Context: same content on both sides.
			bg := d.splitBgForType(diffContext)
			fg := d.splitFgForType(diffContext)

			leftLines := splitWrapText(block.Text, leftContentW, effectiveWrap, fg, bg)
			rightLines := splitWrapText(block.Text, rightContentW, effectiveWrap, fg, bg)

			maxLen := max(len(rightLines), len(leftLines))

			for j := 0; j < maxLen; j++ {
				screenRow := visualRow - d.scrollY
				if screenRow >= 0 && screenRow < h {
					ry := y + screenRow

					// Left gutter + content
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

					// Separator
					buf.SetCell(sepX, ry, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)

					// Right gutter + content
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

		case DiffBlockChange:
			// Pair removes (left) with adds (right), padding the shorter side.
			pairLen := max(len(block.Adds), len(block.Removes))

			for j := 0; j < pairLen; j++ {
				// Determine left side (removed or empty).
				var leftContent string
				var leftLineNum int
				leftType := diffEmpty
				if j < len(block.Removes) {
					leftContent = block.Removes[j].Content
					leftLineNum = block.Removes[j].LineNum
					leftType = diffRemoved
				}

				// Determine right side (added or empty).
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

				for k := 0; k < wrapMax; k++ {
					screenRow := visualRow - d.scrollY
					if screenRow >= 0 && screenRow < h {
						ry := y + screenRow

						// Left gutter + content
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

						// Separator
						buf.SetCell(sepX, ry, d.splitSeparator, d.separatorFg, style.ColorTransparent, 0)

						// Right gutter + content
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

// ---------------------------------------------------------------------------
// Split View Rendering Helpers
// ---------------------------------------------------------------------------

// computeSplitGutterWidth returns the gutter width for split view based on
// the pre-computed max line number and whether signs are shown.
func (d *Diff) computeSplitGutterWidth() int {
	maxNum := 0
	if d.layout != nil {
		maxNum = d.layout.MaxLineNum
	}

	digits := max(len(strconv.Itoa(maxNum)), 2)

	w := digits + 1 // digits + leading pad
	if d.showSigns {
		w += 3 // " + " sign area
	} else {
		w += 1 // trailing space
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

// renderSplitGutter draws one gutter cell (line number + optional sign).
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

	// Line number (right-aligned in numFieldW).
	if d.showLineNumbers && !hideNum && lineNum > 0 {
		numStr := strconv.Itoa(lineNum)
		startCol := max(numFieldW-len(numStr), 0)
		for i, r := range numStr {
			buf.SetCell(bx+startCol+i, by, r, d.lineNumFg, bg, 0)
		}
	}

	// Sign.
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
				fg = d.lineNumFg // use dim color for empty sign area
			}
			buf.SetCell(bx+numFieldW+i, by, r, fg, bg, 0)
		}
	}
}

// renderSplitLine draws a wrappedLine (from the shared wrapping infrastructure)
// into the buffer at the given position and width. Each chunk carries its own
// fg/bg/attributes. A fallback bg is used for the full-row background fill.
func (d *Diff) renderSplitLine(buf *buffer.Buffer, bx, by, width int, line wrappedLine, fallbackBg style.Color) {
	if !fallbackBg.IsTransparent() {
		buf.FillRect(bx, by, width, 1, fallbackBg)
	}
	col := 0
	for _, chunk := range line.chunks {
		col += buf.DrawClusters(chunk.text, bx+col, by, width-col, chunk.fg, chunk.bg, chunk.attributes, nil)
	}
}

// splitWrapText wraps text for split view rendering using the shared wrapping
// infrastructure from text.go (wrapSegmentsWord / wrapSegmentsChar).
// Tabs are expanded to spaces since Diff doesn't use TextSurface.
func splitWrapText(text string, width int, mode WrapMode, fg, bg style.Color) []wrappedLine {
	segments := []styledSegment{{text: text, fg: fg, bg: bg}}

	if mode == WrapNone || width <= 0 {
		return []wrappedLine{segmentsToWrappedLine(segments)}
	}

	switch mode {
	case WrapChar:
		return wrapSegmentsChar(segments, width)
	case WrapWord:
		return wrapSegmentsWord(segments, width)
	default:
		return []wrappedLine{segmentsToWrappedLine(segments)}
	}
}

// positionWidget sets a child widget's position and size for manual rendering.
func (d *Diff) positionWidget(w render.Renderable, x, y, width, height int) {
	ln := w.LayoutNode()
	ln.Computed.X = x
	ln.Computed.Y = y
	ln.Computed.Width = width
	ln.Computed.Height = height
	w.UpdateFromLayout()
}

// ---------------------------------------------------------------------------
// Rebuild: parse patch and create/update child widgets
// ---------------------------------------------------------------------------

func (d *Diff) rebuild() {
	// Remove existing children
	d.removeChildren()

	// Parse patch and build view-independent layout once.
	d.parsed = nil
	d.layout = nil
	if d.patch != "" {
		d.parsed = ParsePatch(d.patch)
		d.layout = BuildDiffLayout(d.parsed)

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

// ---------------------------------------------------------------------------
// Unified View
// ---------------------------------------------------------------------------

func (d *Diff) buildUnifiedView() {
	d.LayoutNode().FlexDirection = layout.DirectionColumn

	if d.layout == nil || len(d.layout.Blocks) == 0 {
		return
	}

	// Flatten DiffLayout blocks into sequential display lines.
	// The block structure naturally produces the standard git unified-diff
	// order: for change blocks all removes are emitted before all adds.
	var contentLines []string
	lineColors := make(map[int]LineColorConfig)
	lineSigns := make(map[int]LineSign)
	lineNumbers := make(map[int]int)
	hideLineNums := make(map[int]bool)
	lineIndex := 0

	for _, block := range d.layout.Blocks {
		switch block.Kind {
		case DiffBlockFullWidth:
			// File header or hunk header — no line number.
			contentLines = append(contentLines, block.Text)
			hideLineNums[lineIndex] = true
			lineIndex++

		case DiffBlockContext:
			// Context line — show new-side line number.
			contentLines = append(contentLines, block.Text)
			if !d.contextBg.IsTransparent() {
				lineColors[lineIndex] = LineColorConfig{
					Gutter:  d.contextBg,
					Content: d.contextBg,
				}
			}
			lineNumbers[lineIndex] = block.NewLineNum
			lineIndex++

		case DiffBlockChange:
			// Removes first, then adds (standard unified diff order).
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

	// Create Code widget
	codeOpts := []CodeOption{
		CodeWithFiletype(d.filetype),
		CodeWithTheme(d.theme),
		CodeWithFg(d.contextFg),
		CodeWithBg(style.ColorTransparent),
		CodeWithWidth(layout.Percent(100)),
		CodeWithHeight(layout.Percent(100)),
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

	// Build per-line bg for Code (content area colors)
	codeLineColors := make(map[int]style.Color)
	for idx, lc := range lineColors {
		if !lc.Content.IsTransparent() {
			codeLineColors[idx] = lc.Content
		}
	}
	d.leftCode.SetLineColors(codeLineColors)

	// If signs not shown, clear them
	if !d.showSigns {
		lineSigns = nil
	}

	// Create LineNumber widget wrapping the Code
	// Minimum gutter width: 3 digits + sign area (3 when showSigns, 1 otherwise)
	minGutterW := 3
	if d.showSigns {
		minGutterW += 3 // " + " sign area
	} else {
		minGutterW += 1 // trailing space
	}

	d.leftSide = NewLineNumber(d.leftCode,
		LineNumberWithFg(d.lineNumFg),
		LineNumberWithOffset(0), // we use custom lineNumbers map
		LineNumberWithMinWidth(minGutterW),
		LineNumberWithShow(d.showLineNumbers),
		LineNumberWithLineColors(lineColors),
		LineNumberWithLineSigns(lineSigns),
		LineNumberWithLineNumbers(lineNumbers),
		LineNumberWithHideLineNumbers(hideLineNums),
		LineNumberWithWidth(layout.Percent(100)),
		LineNumberWithHeight(layout.Percent(100)),
	)

	if d.gutterSeparator {
		d.leftSide.SetSeparator(symbol.Vertical)
		d.leftSide.separatorFg = d.separatorFg
	}

	d.Add(d.leftSide)
}

// ---------------------------------------------------------------------------
// Split View
// ---------------------------------------------------------------------------

func (d *Diff) buildSplitView() {
	d.LayoutNode().FlexDirection = layout.DirectionRow
	d.splitVisualCount = 0
}

// ---------------------------------------------------------------------------
// Inline Word Diff
// ---------------------------------------------------------------------------

// inlineDiffSegment represents a segment with diff status.
type inlineDiffSegment struct {
	text    string
	changed bool
}

// computeInlineDiff compares two strings and returns segments marked as changed/unchanged.
func computeInlineDiff(oldText, newText string) (oldSegs, newSegs []inlineDiffSegment) {
	oldRunes := []rune(oldText)
	newRunes := []rune(newText)

	// Find common prefix
	prefixLen := 0
	for prefixLen < len(oldRunes) && prefixLen < len(newRunes) && oldRunes[prefixLen] == newRunes[prefixLen] {
		prefixLen++
	}

	// Find common suffix
	suffixLen := 0
	for suffixLen < len(oldRunes)-prefixLen && suffixLen < len(newRunes)-prefixLen &&
		oldRunes[len(oldRunes)-1-suffixLen] == newRunes[len(newRunes)-1-suffixLen] {
		suffixLen++
	}

	// Common prefix
	if prefixLen > 0 {
		prefix := string(oldRunes[:prefixLen])
		oldSegs = append(oldSegs, inlineDiffSegment{text: prefix, changed: false})
		newSegs = append(newSegs, inlineDiffSegment{text: prefix, changed: false})
	}

	// Changed middle
	oldMiddle := string(oldRunes[prefixLen : len(oldRunes)-suffixLen])
	newMiddle := string(newRunes[prefixLen : len(newRunes)-suffixLen])
	if len(oldMiddle) > 0 {
		oldSegs = append(oldSegs, inlineDiffSegment{text: oldMiddle, changed: true})
	}
	if len(newMiddle) > 0 {
		newSegs = append(newSegs, inlineDiffSegment{text: newMiddle, changed: true})
	}

	// Common suffix
	if suffixLen > 0 {
		suffix := string(oldRunes[len(oldRunes)-suffixLen:])
		oldSegs = append(oldSegs, inlineDiffSegment{text: suffix, changed: false})
		newSegs = append(newSegs, inlineDiffSegment{text: suffix, changed: false})
	}

	return oldSegs, newSegs
}

// ---------------------------------------------------------------------------
// Diff Options
// ---------------------------------------------------------------------------

// DiffWithViewMode sets the view mode.
func DiffWithViewMode(mode DiffViewMode) DiffOption {
	return func(d *Diff) { d.viewMode = mode }
}

// DiffWithFiletype sets the filetype for syntax highlighting within diff.
func DiffWithFiletype(ft string) DiffOption {
	return func(d *Diff) { d.filetype = ft }
}

// DiffWithTheme sets the syntax highlighting theme name.
func DiffWithTheme(name string) DiffOption {
	return func(d *Diff) { d.theme = name }
}

// DiffWithShowLineNumbers enables/disables line numbers.
func DiffWithShowLineNumbers(v bool) DiffOption {
	return func(d *Diff) { d.showLineNumbers = v }
}

// DiffWithInlineHighlight enables/disables word-level inline diff.
func DiffWithInlineHighlight(v bool) DiffOption {
	return func(d *Diff) { d.inlineHighlight = v }
}

// DiffWithShowSigns enables/disables +/- signs in gutter.
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

// DiffWithWidth sets the widget width.
func DiffWithWidth(s layout.Scalar) DiffOption {
	return func(d *Diff) { d.LayoutNode().Width = s }
}

// DiffWithHeight sets the widget height.
func DiffWithHeight(s layout.Scalar) DiffOption {
	return func(d *Diff) { d.LayoutNode().Height = s }
}

// DiffWithFlexGrow sets flex grow.
func DiffWithFlexGrow(v int) DiffOption {
	return func(d *Diff) { d.LayoutNode().FlexGrow = v }
}

// DiffWithWrapMode sets the wrap mode for the Code widgets inside the diff.
// When nil (default), Code inherits its own default (WrapWord).
func DiffWithWrapMode(m WrapMode) DiffOption {
	return func(d *Diff) { d.wrapMode = &m }
}

// DiffWithHighlighter sets a custom SyntaxHighlighter for the diff and its
// child Code widgets. When nil (default), a ChromaHighlighter is used.
func DiffWithHighlighter(h SyntaxHighlighter) DiffOption {
	return func(d *Diff) { d.highlighter = h }
}
