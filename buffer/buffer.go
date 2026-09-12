package buffer

import (
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// ---------------------------------------------------------------------------
// Rect
// ---------------------------------------------------------------------------

// Rect represents a rectangle in cell coordinates.
type Rect struct {
	X, Y, Width, Height int
}

// Contains returns true if the point (x, y) is inside the rect.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Intersect returns the intersection of two rectangles.
// If there is no overlap the returned Rect will have zero or negative dimensions.
func (r Rect) Intersect(other Rect) Rect {
	x1 := max(r.X, other.X)
	y1 := max(r.Y, other.Y)
	x2 := min(r.Right(), other.Right())
	y2 := min(r.Bottom(), other.Bottom())

	w := x2 - x1
	h := y2 - y1
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x1, Y: y1, Width: w, Height: h}
}

// IsEmpty returns true if the rect has no area.
func (r Rect) IsEmpty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Right returns X + Width.
func (r Rect) Right() int { return r.X + r.Width }

// Bottom returns Y + Height.
func (r Rect) Bottom() int { return r.Y + r.Height }

// ---------------------------------------------------------------------------
// DrawTextOption
// ---------------------------------------------------------------------------

type drawTextConfig struct {
	bg        *style.Color
	attrs     uint32
	selection *textbuf.TextSelection
	maxWidth  int // 0 = no limit (buffer edge)
}

// DrawTextOption configures DrawText behavior.
type DrawTextOption func(*drawTextConfig)

// WithBg sets the background color for drawn text.
func WithBg(bg style.Color) DrawTextOption {
	return func(c *drawTextConfig) {
		c.bg = &bg
	}
}

// WithAttrs sets the text attributes for drawn text.
func WithAttrs(attrs uint32) DrawTextOption {
	return func(c *drawTextConfig) {
		c.attrs = attrs
	}
}

// WithSelection sets a text selection highlight.
func WithSelection(sel textbuf.TextSelection) DrawTextOption {
	return func(c *drawTextConfig) {
		c.selection = &sel
	}
}

// WithMaxWidth sets a maximum width for drawn text.
func WithMaxWidth(w int) DrawTextOption {
	return func(c *drawTextConfig) {
		c.maxWidth = w
	}
}

// ---------------------------------------------------------------------------
// BoxDrawOptions
// ---------------------------------------------------------------------------

// BoxDrawOptions configures the DrawBox method.
type BoxDrawOptions struct {
	X, Y, Width, Height int
	BorderStyle         style.BorderStyle
	Border              style.Border
	BorderColor         style.Color
	BackgroundColor     style.Color
	ShouldFill          bool
	Title               string
	TitleAlignment      string // "left" | "center" | "right"
}

// ---------------------------------------------------------------------------
// Lines  (data access — uses textbuf.Span / textbuf.Line)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Buffer
// ---------------------------------------------------------------------------

// Buffer is a 2D grid of styled cells with clipping and opacity support.
type Buffer struct {
	width        int
	height       int
	cells        []Cell       // row-major flat array, len = width * height
	scissor      scissorStack // from scissor.go
	opacity      opacityStack // from opacity.go
	respectAlpha bool         // whether alpha blending is applied on compositing
	id           string       // buffer identifier
}

// New allocates a buffer of the given size, filled with CellClear.
func New(width, height int) *Buffer {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	b := &Buffer{
		width:  width,
		height: height,
		cells:  make([]Cell, width*height),
	}
	b.fill(CellClear)
	return b
}

// NewWithID allocates a buffer with an identifier.
func NewWithID(width, height int, id string) *Buffer {
	b := New(width, height)
	b.id = id
	return b
}

// BufferOption configures a Buffer during construction.
type BufferOption func(*Buffer)

// WithRespectAlpha enables alpha blending.
func WithRespectAlphaOpt(v bool) BufferOption {
	return func(b *Buffer) { b.respectAlpha = v }
}

// WithIDOpt sets the buffer identifier.
func WithIDOpt(id string) BufferOption {
	return func(b *Buffer) { b.id = id }
}

// NewWithOptions allocates a buffer with configurable options.
func NewWithOptions(width, height int, opts ...BufferOption) *Buffer {
	b := New(width, height)
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// RuneWidthForBuffer returns the display width of a rune.
func (b *Buffer) RuneWidthForBuffer(r rune) int {
	return RuneWidth(r)
}

// StringWidthForBuffer returns the display width of a string.
func (b *Buffer) StringWidthForBuffer(s string) int {
	return textbuf.StringWidth(s)
}

// fill sets every cell to c.
func (b *Buffer) fill(c Cell) {
	for i := range b.cells {
		b.cells[i] = c
	}
}

// ---------------------------------------------------------------------------
// Properties
// ---------------------------------------------------------------------------

// Width returns the buffer width in cells.
func (b *Buffer) Width() int { return b.width }

// Height returns the buffer height in cells.
func (b *Buffer) Height() int { return b.height }

// ID returns the buffer identifier.
func (b *Buffer) ID() string { return b.id }

// Bounds returns the full buffer rect.
func (b *Buffer) Bounds() Rect {
	return Rect{0, 0, b.width, b.height}
}

// SetRespectAlpha toggles alpha blending mode.
func (b *Buffer) SetRespectAlpha(v bool) { b.respectAlpha = v }

// RespectAlpha returns the current alpha blending mode.
func (b *Buffer) RespectAlpha() bool { return b.respectAlpha }

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (b *Buffer) cellIndex(x, y int) int { return y*b.width + x }

func (b *Buffer) inBounds(x, y int) bool {
	return x >= 0 && x < b.width && y >= 0 && y < b.height
}

func (b *Buffer) isClipped(x, y int) bool {
	return !b.scissor.Contains(x, y)
}

// applyOpacity multiplies fg and bg alpha by the current effective opacity.
func (b *Buffer) applyOpacity(fg, bg *style.Color) {
	op := b.opacity.Current()
	if op < 1.0 {
		fg.A = uint8(float32(fg.A) * op)
		bg.A = uint8(float32(bg.A) * op)
	}
}

// ---------------------------------------------------------------------------
// Core Cell Access
// ---------------------------------------------------------------------------

// SetCellCombining sets the Combining field for the cell at (x, y).
// No-op if out of bounds.
func (b *Buffer) SetCellCombining(x, y int, combining string) {
	if !b.inBounds(x, y) {
		return
	}
	b.cells[b.cellIndex(x, y)].Combining = combining
}

// SetGraphemeCluster writes a grapheme cluster at (x, y).
// The first codepoint becomes Cell.Char; remaining combining codepoints
// are stored in Cell.Combining. This is the canonical way to write a
// multi-codepoint cluster (emoji, ZWJ sequence, etc.) to the buffer.
//
// Cell.Width is the width of the whole cluster, which can exceed the base
// rune's width: an emoji with VS16 and a Thai consonant followed by Sara Am
// both occupy two columns. The flush writes every codepoint of the cluster
// from this one cell, so the width must match what the terminal advances.
func (b *Buffer) SetGraphemeCluster(x, y int, cluster string, fg, bg style.Color, attrs uint32) {
	baseRune, baseSize := utf8.DecodeRuneInString(cluster)
	b.setCluster(x, y, baseRune, cluster[baseSize:], textbuf.StringWidth(cluster), fg, bg, attrs)
}

// ClusterStyler is called for each grapheme cluster before it is painted.
// It receives the cluster's column offset (relative to the draw origin) and
// the default fg/bg/attrs. It returns the final colors and attributes to use.
// If it returns skip=true the cluster is not painted but its width is still
// consumed (the column advances).
type ClusterStyler func(col int, fg, bg style.Color, attrs uint32) (outFg, outBg style.Color, outAttrs uint32, skip bool)

// DrawClusters paints grapheme clusters from text into the buffer starting
// at (x, y), bounded to maxWidth columns. Clusters that extend past x or
// x+maxWidth are skipped/clipped. Returns the number of columns consumed.
//
// If styler is nil, each cluster is painted with the provided fg/bg/attrs
// unchanged (fast path for markdown, diff, etc.).
//
// If styler is non-nil, it is called for every visible cluster so the caller
// can override colors per-column (for selection highlighting, transparent bg
// inheritance, etc.).
func (b *Buffer) DrawClusters(text string, x, y, maxWidth int, fg, bg style.Color, attrs uint32, styler ClusterStyler) int {
	cx := 0
	remaining := text
	for len(remaining) > 0 {
		cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
		remaining = rest

		if cx+rw > maxWidth {
			break
		}

		cellFg, cellBg, cellAttrs := fg, bg, attrs
		skip := false
		if styler != nil {
			cellFg, cellBg, cellAttrs, skip = styler(cx, fg, bg, attrs)
		}

		if !skip {
			b.SetGraphemeCluster(x+cx, y, cluster, cellFg, cellBg, cellAttrs)
		}
		cx += rw
	}
	return cx
}

// Cell reads a cell. Returns CellClear if out of bounds.
func (b *Buffer) Cell(x, y int) Cell {
	if !b.inBounds(x, y) {
		return CellClear
	}
	return b.cells[b.cellIndex(x, y)]
}

// SetCell writes a cell at (x, y). Respects scissor clipping and opacity.
// No-op if out of bounds or clipped.
//
// Wide characters (East Asian width 2) require two consecutive buffer columns:
// the main cell and a continuation cell (Width == -1). If the main cell lands
// on the last buffer column there is no room for the continuation, so the
// character is silently dropped. Writing a half-wide char at the right edge
// would produce a stranded Width=2 cell with no continuation; InlineFlush
// would then emit the 2-column rune, making that row 1 column wider than the
// buffer and triggering terminal auto-wrap.
func (b *Buffer) SetCell(x, y int, char rune, fg, bg style.Color, attrs uint32) {
	b.setCluster(x, y, char, "", RuneWidth(char), fg, bg, attrs)
}

// detachWide blanks the other cells of the cluster occupying column x.
// Continuation offsets identify the head even for clusters wider than two.
func (b *Buffer) detachWide(x, y int) {
	head := x
	if c := b.Cell(x, y); c.IsContinuation() {
		head += c.Width
	}
	w := b.Cell(head, y).Width
	if head < 0 || w <= max(x-head, 1) {
		return
	}
	for col := head; col < head+min(w, b.width-head); col++ {
		if col != x && (col == head || b.Cell(col, y).Width == head-col) {
			b.blankAt(col, y)
		}
	}
}

// blankAt replaces the cell at (x, y) with a single-column space, keeping its
// colors so the gap left by a broken wide character matches its surroundings.
//
// The partner of a broken pair can lie outside the caller's scissor rect, and
// a clipped draw must not reach beyond it. Leaving that half alone costs
// nothing: flushRune emits a stranded head or continuation as a space, so the
// row still advances exactly the columns its cells claim.
func (b *Buffer) blankAt(x, y int) {
	if b.isClipped(x, y) {
		return
	}
	idx := b.cellIndex(x, y)
	c := b.cells[idx]
	b.cells[idx] = Cell{
		Char:       ' ',
		Fg:         c.Fg,
		Bg:         c.Bg,
		Attributes: c.Attributes,
		Width:      1,
	}
}

// setCluster writes a cell holding char plus any trailing combining
// codepoints, occupying w columns. Backs SetCell and SetGraphemeCluster.
func (b *Buffer) setCluster(x, y int, char rune, combining string, w int, fg, bg style.Color, attrs uint32) {
	b.SetCellDirect(x, y, Cell{
		Char:       char,
		Combining:  combining,
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      w,
	})
}

// SetCellDirect writes a full Cell struct at (x, y).
// Respects scissor clipping and opacity.
// Transparent backgrounds preserve the existing cell's background.
// Wide cells also write a continuation with matching colors and attributes.
// Clusters that do not fit within the buffer or scissor rect are dropped.
func (b *Buffer) SetCellDirect(x, y int, cell Cell) {
	if !b.inBounds(x, y) || b.isClipped(x, y) {
		return
	}
	if cell.Width > b.width-x {
		return
	}
	for offset := 1; offset < cell.Width; offset++ {
		if b.isClipped(x+offset, y) {
			return
		}
	}
	b.applyOpacity(&cell.Fg, &cell.Bg)
	// Writing a continuation must not detach the head it belongs to.
	if !cell.IsContinuation() {
		for offset := range max(cell.Width, 1) {
			b.detachWide(x+offset, y)
		}
	}
	idx := b.cellIndex(x, y)
	if cell.Bg.IsTransparent() {
		cell.Bg = b.cells[idx].Bg
	}
	b.cells[idx] = cell
	for offset := 1; offset < cell.Width; offset++ {
		b.cells[idx+offset] = Cell{
			Fg:         cell.Fg,
			Bg:         cell.Bg,
			Attributes: cell.Attributes,
			Width:      -offset,
		}
	}
}

// SetCellBlended writes a cell with alpha blending against the existing cell.
// Respects scissor clipping and opacity.
// Wide characters at the last buffer column are dropped (see SetCell).
func (b *Buffer) SetCellBlended(x, y int, char rune, fg, bg style.Color, attrs uint32) {
	if !b.inBounds(x, y) || b.isClipped(x, y) {
		return
	}

	b.applyOpacity(&fg, &bg)

	w := RuneWidth(char)

	// Drop wide chars that can't fit their continuation cell.
	if w == 2 && x+1 >= b.width {
		return
	}

	src := Cell{
		Char:       char,
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      w,
	}

	b.detachWide(x, y)
	if w == 2 {
		b.detachWide(x+1, y)
	}

	idx := b.cellIndex(x, y)
	b.cells[idx] = BlendCell(b.cells[idx], src)

	// If wide character, blend continuation cell
	if w == 2 {
		cont := Cell{
			Char:       0,
			Fg:         fg,
			Bg:         bg,
			Attributes: attrs,
			Width:      -1,
		}
		b.cells[idx+1] = BlendCell(b.cells[idx+1], cont)
	}
}

// ---------------------------------------------------------------------------
// Drawing Methods
// ---------------------------------------------------------------------------

// Clear fills the entire buffer. If no color is given, uses transparent black.
func (b *Buffer) Clear(bg ...style.Color) {
	c := CellClear
	if len(bg) > 0 {
		c.Bg = bg[0]
	}
	b.fill(c)
}

// DrawText draws a string at (x, y) with the given foreground color.
// Handles wide characters and stops at buffer edge or maxWidth.
func (b *Buffer) DrawText(text string, x, y int, fg style.Color, opts ...DrawTextOption) {
	cfg := drawTextConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	hasBg := cfg.bg != nil

	maxX := b.width
	if cfg.maxWidth > 0 && x+cfg.maxWidth < maxX {
		maxX = x + cfg.maxWidth
	}

	cx := x
	charIdx := 0
	remaining := text
	for len(remaining) > 0 {
		// Handle tab: decode first rune to check
		r, rsize := utf8.DecodeRuneInString(remaining)
		if r == '\t' {
			remaining = remaining[rsize:]
			const tabWidth = 2
			for range tabWidth {
				if cx >= maxX {
					break
				}
				var tabBg style.Color
				if hasBg {
					tabBg = *cfg.bg
				} else if b.inBounds(cx, y) {
					tabBg = b.cells[b.cellIndex(cx, y)].Bg
				} else {
					tabBg = style.ColorFromInts(0, 0, 0, 255)
				}
				b.SetCell(cx, y, ' ', fg, tabBg, cfg.attrs)
				cx++
			}
			charIdx++
			continue
		}

		// Extract grapheme cluster
		cluster, rest, rw := textbuf.NextGraphemeCluster(remaining)
		remaining = rest

		if cx+rw > maxX {
			break
		}

		cellFg := fg
		var cellBg style.Color
		if hasBg {
			cellBg = *cfg.bg
		} else if b.inBounds(cx, y) {
			cellBg = b.cells[b.cellIndex(cx, y)].Bg
		} else {
			cellBg = style.ColorFromInts(0, 0, 0, 255)
		}
		cellAttrs := cfg.attrs

		// Apply selection highlighting
		if cfg.selection != nil && charIdx >= cfg.selection.Start && charIdx < cfg.selection.End {
			if cfg.selection.BgColor != nil {
				cellBg = *cfg.selection.BgColor
			}
			if cfg.selection.FgColor != nil {
				cellFg = *cfg.selection.FgColor
			}
		}

		b.SetGraphemeCluster(cx, y, cluster, cellFg, cellBg, cellAttrs)

		cx += rw
		charIdx++
	}
}

// DrawChar draws a single codepoint at (x, y).
func (b *Buffer) DrawChar(char rune, x, y int, fg, bg style.Color, attrs uint32) {
	b.SetCell(x, y, char, fg, bg, attrs)
}

// FillRect fills a rectangle with a background color.
// The filled cells are set to space with white fg.
func (b *Buffer) FillRect(x, y, w, h int, bg style.Color) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			if b.inBounds(col, row) && !b.isClipped(col, row) {
				fg := style.ColorWhite
				cellBg := bg
				b.applyOpacity(&fg, &cellBg)
				b.detachWide(col, row)
				idx := b.cellIndex(col, row)
				b.cells[idx] = Cell{
					Char:  ' ',
					Fg:    fg,
					Bg:    cellBg,
					Width: 1,
				}
			}
		}
	}
}

// DrawBox draws a bordered/filled box with optional title,
// including partial-border vertical extension.
func (b *Buffer) DrawBox(opts BoxDrawOptions) {
	sides := style.GetBorderSides(opts.Border)
	chars := style.CharsForStyle(opts.BorderStyle)

	startX := max(0, opts.X)
	startY := max(0, opts.Y)
	endX := min(b.width-1, opts.X+opts.Width-1)
	endY := min(b.height-1, opts.Y+opts.Height-1)

	if startX > endX || startY > endY {
		return
	}

	isAtActualLeft := startX == opts.X
	isAtActualRight := endX == opts.X+opts.Width-1
	isAtActualTop := startY == opts.Y
	isAtActualBottom := endY == opts.Y+opts.Height-1

	// Title calculation
	shouldDrawTitle := false
	var titleX, titleStartX, titleEndX int
	if opts.Title != "" && sides.Top && isAtActualTop {
		titleLen := b.StringWidthForBuffer(opts.Title)
		minTitleSpace := 4

		if opts.Width >= titleLen+minTitleSpace {
			shouldDrawTitle = true
			padding := 2

			switch opts.TitleAlignment {
			case "center":
				titleX = startX + max(padding, (opts.Width-titleLen)/2)
			case "right":
				titleX = startX + opts.Width - padding - titleLen
			default: // "left"
				titleX = startX + padding
			}

			titleX = max(startX+padding, min(titleX, endX-titleLen))
			titleStartX = titleX
			titleEndX = titleX + titleLen - 1
		}
	}

	// Fill background
	if opts.ShouldFill {
		if !sides.Top && !sides.Right && !sides.Bottom && !sides.Left {
			b.FillRect(startX, startY, endX-startX+1, endY-startY+1, opts.BackgroundColor)
		} else {
			innerStartX := startX
			innerStartY := startY
			innerEndX := endX
			innerEndY := endY
			if sides.Left && isAtActualLeft {
				innerStartX++
			}
			if sides.Top && isAtActualTop {
				innerStartY++
			}
			if sides.Right && isAtActualRight {
				innerEndX--
			}
			if sides.Bottom && isAtActualBottom {
				innerEndY--
			}
			if innerEndX >= innerStartX && innerEndY >= innerStartY {
				b.FillRect(innerStartX, innerStartY, innerEndX-innerStartX+1, innerEndY-innerStartY+1, opts.BackgroundColor)
			}
		}
	}

	// Vertical extension logic:
	// When only vertical borders (no top/bottom), extend verticals to cover top/bottom rows
	leftBorderOnly := sides.Left && isAtActualLeft && !sides.Top && !sides.Bottom
	rightBorderOnly := sides.Right && isAtActualRight && !sides.Top && !sides.Bottom
	bottomOnlyWithVerticals := sides.Bottom && isAtActualBottom && !sides.Top && (sides.Left || sides.Right)
	topOnlyWithVerticals := sides.Top && isAtActualTop && !sides.Bottom && (sides.Left || sides.Right)

	extendVerticalsToTop := leftBorderOnly || rightBorderOnly || bottomOnlyWithVerticals
	extendVerticalsToBottom := leftBorderOnly || rightBorderOnly || topOnlyWithVerticals

	// Draw top border
	if sides.Top && isAtActualTop {
		for drawX := startX; drawX <= endX; drawX++ {
			if shouldDrawTitle && drawX >= titleStartX && drawX <= titleEndX {
				continue
			}
			ch := chars.Horizontal
			if drawX == startX && isAtActualLeft {
				if sides.Left {
					ch = chars.TopLeft
				}
			} else if drawX == endX && isAtActualRight {
				if sides.Right {
					ch = chars.TopRight
				}
			}
			b.SetCell(drawX, startY, ch, opts.BorderColor, opts.BackgroundColor, 0)
		}
	}

	// Draw bottom border
	if sides.Bottom && isAtActualBottom {
		for drawX := startX; drawX <= endX; drawX++ {
			ch := chars.Horizontal
			if drawX == startX && isAtActualLeft {
				if sides.Left {
					ch = chars.BottomLeft
				}
			} else if drawX == endX && isAtActualRight {
				if sides.Right {
					ch = chars.BottomRight
				}
			}
			b.SetCell(drawX, endY, ch, opts.BorderColor, opts.BackgroundColor, 0)
		}
	}

	// Draw vertical borders
	verticalStartY := startY
	if !extendVerticalsToTop && sides.Top && isAtActualTop {
		verticalStartY++
	}
	verticalEndY := endY
	if !extendVerticalsToBottom && sides.Bottom && isAtActualBottom {
		verticalEndY--
	}

	for drawY := verticalStartY; drawY <= verticalEndY; drawY++ {
		if sides.Left && isAtActualLeft {
			b.SetCell(startX, drawY, chars.Vertical, opts.BorderColor, opts.BackgroundColor, 0)
		}
		if sides.Right && isAtActualRight {
			b.SetCell(endX, drawY, chars.Vertical, opts.BorderColor, opts.BackgroundColor, 0)
		}
	}

	// Draw title
	if shouldDrawTitle {
		b.DrawText(opts.Title, titleX, startY, opts.BorderColor, WithBg(opts.BackgroundColor))
	}
}

// DrawHLine draws a horizontal line.
func (b *Buffer) DrawHLine(x, y, length int, char rune, fg, bg style.Color) {
	for i := range length {
		b.SetCell(x+i, y, char, fg, bg, 0)
	}
}

// DrawVLine draws a vertical line.
func (b *Buffer) DrawVLine(x, y, length int, char rune, fg, bg style.Color) {
	for i := range length {
		b.SetCell(x, y+i, char, fg, bg, 0)
	}
}

// ---------------------------------------------------------------------------
// Buffer Compositing
// ---------------------------------------------------------------------------

// DrawFrameBuffer blits a source buffer onto this buffer at (destX, destY).
// An optional srcRect can specify a sub-region of the source.
// Respects scissor rects and opacity. If respectAlpha is true, alpha-blends.
func (b *Buffer) DrawFrameBuffer(destX, destY int, src *Buffer, srcRect ...Rect) {
	sr := src.Bounds()
	if len(srcRect) > 0 {
		sr = srcRect[0]
	}

	for sy := sr.Y; sy < sr.Bottom(); sy++ {
		for sx := sr.X; sx < sr.Right(); sx++ {
			dx := destX + (sx - sr.X)
			dy := destY + (sy - sr.Y)

			if !b.inBounds(dx, dy) || b.isClipped(dx, dy) {
				continue
			}

			srcCell := src.Cell(sx, sy)

			// Skip fully transparent source cells
			if srcCell.Bg.A == 0 && srcCell.Fg.A == 0 {
				continue
			}

			// Apply opacity to source cell
			op := b.opacity.Current()
			if op < 1.0 {
				srcCell.Fg.A = uint8(float32(srcCell.Fg.A) * op)
				srcCell.Bg.A = uint8(float32(srcCell.Bg.A) * op)
			}

			b.detachWide(dx, dy)
			idx := b.cellIndex(dx, dy)
			if src.respectAlpha {
				b.cells[idx] = BlendCell(b.cells[idx], srcCell)
			} else {
				b.cells[idx] = srcCell
			}
		}
	}
}

// CopyFrom performs a full copy from src. Dimensions must match.
func (b *Buffer) CopyFrom(src *Buffer) {
	if b.width != src.width || b.height != src.height {
		return
	}
	copy(b.cells, src.cells)
}

// ---------------------------------------------------------------------------
// Resize & Lifecycle
// ---------------------------------------------------------------------------

// Resize resizes the buffer, clearing all cells and resetting stacks.
func (b *Buffer) Resize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	b.width = width
	b.height = height
	b.cells = make([]Cell, width*height)
	b.fill(CellClear)
	b.scissor.Clear()
	b.opacity.Clear()
}

// ---------------------------------------------------------------------------
// Scissor Rect (delegates to scissorStack)
// ---------------------------------------------------------------------------

// PushScissorRect restricts subsequent draws to the given rect.
func (b *Buffer) PushScissorRect(x, y, w, h int) {
	b.scissor.Push(x, y, w, h)
}

// PopScissorRect removes the topmost scissor rect.
func (b *Buffer) PopScissorRect() {
	b.scissor.Pop()
}

// ClearScissorRects clears all scissor rects.
func (b *Buffer) ClearScissorRects() {
	b.scissor.Clear()
}

// ---------------------------------------------------------------------------
// Opacity (delegates to opacityStack)
// ---------------------------------------------------------------------------

// PushOpacity pushes an opacity multiplier (0.0-1.0).
func (b *Buffer) PushOpacity(opacity float32) {
	b.opacity.Push(opacity)
}

// PopOpacity removes the topmost opacity value.
func (b *Buffer) PopOpacity() {
	b.opacity.Pop()
}

// CurrentOpacity returns the current effective opacity.
func (b *Buffer) CurrentOpacity() float32 {
	return b.opacity.Current()
}

// ClearOpacity resets all stacked opacity.
func (b *Buffer) ClearOpacity() {
	b.opacity.Clear()
}

// ---------------------------------------------------------------------------
// Raw Data Access
// ---------------------------------------------------------------------------

// Chars returns a snapshot of all codepoints in the buffer (row-major).
// Go equivalent of bufferGetCharPtr.
func (b *Buffer) Chars() []rune {
	out := make([]rune, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Char
	}
	return out
}

// Fgs returns a snapshot of all foreground colors in the buffer (row-major).
// Go equivalent of bufferGetFgPtr.
func (b *Buffer) Fgs() []style.Color {
	out := make([]style.Color, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Fg
	}
	return out
}

// Bgs returns a snapshot of all background colors in the buffer (row-major).
// Go equivalent of bufferGetBgPtr.
func (b *Buffer) Bgs() []style.Color {
	out := make([]style.Color, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Bg
	}
	return out
}

// Attributes returns a snapshot of all attribute values in the buffer (row-major).
// Go equivalent of bufferGetAttributesPtr.
func (b *Buffer) Attributes() []uint32 {
	out := make([]uint32, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Attributes
	}
	return out
}

// ResolvedCharSize returns the total number of UTF-8 bytes needed to encode
// all characters in the buffer. Continuation cells (Width < 0) are skipped.
// Go equivalent of bufferGetRealCharSize.
func (b *Buffer) ResolvedCharSize() int {
	size := 0
	for _, c := range b.cells {
		if c.IsContinuation() {
			continue
		}
		ch := c.Char
		if ch == 0 {
			ch = ' '
		}
		size += utf8.RuneLen(ch)
	}
	return size
}

// WriteResolvedChars writes the resolved characters of the buffer into dst.
// Continuation cells are skipped. If addLineBreaks is true, '\n' is inserted
// between rows. Returns the number of bytes written.
// Go equivalent of bufferWriteResolvedChars.
func (b *Buffer) WriteResolvedChars(dst []byte, addLineBreaks bool) int {
	n := 0
	for y := 0; y < b.height; y++ {
		if addLineBreaks && y > 0 {
			if n < len(dst) {
				dst[n] = '\n'
				n++
			}
		}
		for x := 0; x < b.width; x++ {
			cell := b.cells[b.cellIndex(x, y)]
			if cell.IsContinuation() {
				continue
			}
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}
			runeLen := utf8.RuneLen(ch)
			if n+runeLen > len(dst) {
				return n
			}
			utf8.EncodeRune(dst[n:], ch)
			n += runeLen
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Data Access
// ---------------------------------------------------------------------------

// PlainText extracts all text content as a plain string.
// If addLineBreaks is true (default false), newlines are inserted between rows.
func (b *Buffer) PlainText(addLineBreaks ...bool) string {
	breaks := false
	if len(addLineBreaks) > 0 {
		breaks = addLineBreaks[0]
	}

	var sb strings.Builder
	for y := 0; y < b.height; y++ {
		if breaks && y > 0 {
			sb.WriteByte('\n')
		}
		for x := 0; x < b.width; x++ {
			cell := b.cells[b.cellIndex(x, y)]
			if cell.IsContinuation() {
				continue
			}
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// Lines extracts buffer contents as lines of spans.
func (b *Buffer) Lines() []textbuf.Line {
	lines := make([]textbuf.Line, b.height)
	for y := 0; y < b.height; y++ {
		var spans []textbuf.Span
		var current *textbuf.Span

		for x := 0; x < b.width; x++ {
			cell := b.cells[b.cellIndex(x, y)]
			if cell.IsContinuation() {
				if current != nil {
					current.Width++
				}
				continue
			}

			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}

			// Check if we can extend the current span
			if current != nil &&
				current.Fg.Equals(cell.Fg) &&
				current.Bg.Equals(cell.Bg) &&
				current.Attributes == cell.Attributes {
				current.Text += string(ch)
				current.Width += int(cell.Width)
				if cell.Width <= 0 {
					current.Width++
				}
			} else {
				// Start new span
				if current != nil {
					spans = append(spans, *current)
				}
				w := int(cell.Width)
				if w <= 0 {
					w = 1
				}
				s := textbuf.Span{
					Text:       string(ch),
					Fg:         cell.Fg,
					Bg:         cell.Bg,
					Attributes: cell.Attributes,
					Width:      w,
				}
				current = &s
			}
		}
		if current != nil {
			spans = append(spans, *current)
		}
		lines[y] = textbuf.Line{Spans: spans}
	}
	return lines
}

// ---------------------------------------------------------------------------
// BorderConfig / BorderDrawOptions
// ---------------------------------------------------------------------------

// BorderConfig holds the styling information for a border.
type BorderConfig struct {
	Style  style.BorderStyle
	Border style.Border
	Color  style.Color
}

// BorderDrawOptions is the full specification for drawing a bordered box.
type BorderDrawOptions struct {
	X, Y, Width, Height int
	BorderConfig
	BackgroundColor style.Color
	ShouldFill      bool // fill interior with BackgroundColor
	Title           string
	TitleAlignment  layout.Alignment
}

// ---------------------------------------------------------------------------
// Border Drawing
// ---------------------------------------------------------------------------

// DrawBorder draws a box border, background fill, and optional title.
// It handles per-side enable/disable via the Border field.
func (b *Buffer) DrawBorder(opts BorderDrawOptions) {
	if opts.Width <= 0 || opts.Height <= 0 {
		return
	}

	sides := style.GetBorderSides(opts.Border)
	chars := style.CharsForStyle(opts.Style)

	// Fill background if requested
	if opts.ShouldFill {
		inner := InnerRect(opts.X, opts.Y, opts.Width, opts.Height, sides)
		if inner.Width > 0 && inner.Height > 0 {
			b.FillRect(inner.X, inner.Y, inner.Width, inner.Height, opts.BackgroundColor)
		}
	}

	// Draw top border
	if sides.Top {
		b.drawHorizontalBorder(opts.X, opts.Y, opts.Width, chars,
			opts.Color, opts.BackgroundColor, sides.Left, sides.Right)
	}

	// Draw bottom border
	if sides.Bottom {
		by := opts.Y + opts.Height - 1
		// Bottom line: corners are BottomLeft/BottomRight
		for cx := opts.X; cx < opts.X+opts.Width; cx++ {
			ch := chars.Horizontal
			if cx == opts.X && sides.Left {
				ch = chars.BottomLeft
			} else if cx == opts.X+opts.Width-1 && sides.Right {
				ch = chars.BottomRight
			} else if cx == opts.X && !sides.Left {
				// no left corner, just horizontal
			} else if cx == opts.X+opts.Width-1 && !sides.Right {
				// no right corner, just horizontal
			}
			b.SetCell(cx, by, ch, opts.Color, opts.BackgroundColor, 0)
		}
	}

	// Draw left border (excluding corners already drawn)
	if sides.Left {
		startY := opts.Y
		endY := opts.Y + opts.Height
		if sides.Top {
			startY++
		}
		if sides.Bottom {
			endY--
		}
		b.drawVerticalBorder(opts.X, startY, endY-startY, chars.Vertical,
			opts.Color, opts.BackgroundColor)
	}

	// Draw right border (excluding corners already drawn)
	if sides.Right {
		rx := opts.X + opts.Width - 1
		startY := opts.Y
		endY := opts.Y + opts.Height
		if sides.Top {
			startY++
		}
		if sides.Bottom {
			endY--
		}
		b.drawVerticalBorder(rx, startY, endY-startY, chars.Vertical,
			opts.Color, opts.BackgroundColor)
	}

	// Draw title if present (on top border)
	if opts.Title != "" && sides.Top && opts.Width > 2 {
		titleMaxW := opts.Width - 2
		b.drawBorderTitle(opts.Title, opts.X+1, opts.Y, titleMaxW,
			opts.TitleAlignment, opts.Color, opts.BackgroundColor)
	}
}

// drawHorizontalBorder draws a horizontal border line.
// If hasLeft/hasRight are true, the corresponding corners are drawn.
func (b *Buffer) drawHorizontalBorder(x, y, width int, chars style.BorderCharacters, color, bg style.Color, hasLeft, hasRight bool) {
	for cx := x; cx < x+width; cx++ {
		ch := chars.Horizontal
		if cx == x && hasLeft {
			ch = chars.TopLeft
		} else if cx == x+width-1 && hasRight {
			ch = chars.TopRight
		}
		b.SetCell(cx, y, ch, color, bg, 0)
	}
}

// drawVerticalBorder draws a vertical border line.
func (b *Buffer) drawVerticalBorder(x, y, height int, char rune, color, bg style.Color) {
	for cy := y; cy < y+height; cy++ {
		b.SetCell(x, cy, char, color, bg, 0)
	}
}

// drawBorderTitle draws title text within a border, respecting alignment.
// The title is placed at row y, starting no earlier than x, with a maximum width.
func (b *Buffer) drawBorderTitle(title string, x, y, maxWidth int, alignment layout.Alignment, fg, bg style.Color) {
	if maxWidth <= 0 {
		return
	}

	// Truncate title to maxWidth runes
	runes := []rune(title)
	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}
	titleLen := len(runes)

	// Compute starting X based on alignment
	var startX int
	switch alignment {
	case layout.TextAlignCenter:
		startX = x + (maxWidth-titleLen)/2
	case layout.TextAlignRight:
		startX = x + maxWidth - titleLen
	default: // AlignLeft
		startX = x
	}

	// Draw each rune
	for i, r := range runes {
		b.SetCell(startX+i, y, r, fg, bg, 0)
	}
}

// drawBorderTextSimple writes a string character-by-character using SetCell.
func (b *Buffer) drawBorderTextSimple(text string, x, y int, fg, bg style.Color) {
	cx := x
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		if cx >= b.width {
			break
		}
		b.SetCell(cx, y, r, fg, bg, 0)
		cx++
	}
}

// ---------------------------------------------------------------------------
// InnerRect
// ---------------------------------------------------------------------------

// InnerRect computes the content area inside a border.
// Subtracts 1 cell from each enabled border side.
func InnerRect(x, y, w, h int, sides style.BorderSidesConfig) Rect {
	ix, iy, iw, ih := x, y, w, h
	if sides.Left {
		ix++
		iw--
	}
	if sides.Right {
		iw--
	}
	if sides.Top {
		iy++
		ih--
	}
	if sides.Bottom {
		ih--
	}
	if iw < 0 {
		iw = 0
	}
	if ih < 0 {
		ih = 0
	}
	return Rect{X: ix, Y: iy, Width: iw, Height: ih}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------
