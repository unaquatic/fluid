package buffer

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// Rect describes a rectangular region in buffer cell coordinates.
type Rect struct {
	// X and Y are the top-left cell, and Width and Height the size in cells.
	X, Y, Width, Height int
}

// Contains reports whether (x, y) lies within r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Intersect returns the intersection of r and other, or the zero rectangle
// when they do not overlap.
func (r Rect) Intersect(other Rect) Rect {
	x1 := max(r.X, other.X)
	y1 := max(r.Y, other.Y)
	x2 := min(r.Right(), other.Right())
	y2 := min(r.Bottom(), other.Bottom())

	// An empty result carries no position. Callers read X and Y without
	// rechecking IsEmpty, so a positioned empty rectangle misdirects them.
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// IsEmpty reports whether r has no area.
func (r Rect) IsEmpty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Union returns the smallest rectangle containing both r and other, or the zero
// rectangle when both are empty. An empty operand contributes nothing.
func (r Rect) Union(other Rect) Rect {
	switch {
	case r.IsEmpty() && other.IsEmpty():
		// Neither operand has area, so neither contributes a position.
		return Rect{}
	case r.IsEmpty():
		return other
	case other.IsEmpty():
		return r
	}

	x1 := min(r.X, other.X)
	y1 := min(r.Y, other.Y)
	x2 := max(r.Right(), other.Right())
	y2 := max(r.Bottom(), other.Bottom())
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// Right returns the exclusive right edge of r.
func (r Rect) Right() int { return r.X + r.Width }

// Bottom returns the exclusive bottom edge of r.
func (r Rect) Bottom() int { return r.Y + r.Height }

type drawTextConfig struct {
	bg        *style.Color
	attrs     uint32
	selection *textbuf.TextSelection
	maxWidth  int // Zero extends drawing to the buffer edge.
}

// DrawTextOption configures DrawText behavior.
type DrawTextOption func(*drawTextConfig)

// WithBg sets the background color of text drawn by DrawText.
func WithBg(bg style.Color) DrawTextOption {
	return func(c *drawTextConfig) {
		c.bg = &bg
	}
}

// WithAttrs sets the attributes of text drawn by DrawText.
func WithAttrs(attrs uint32) DrawTextOption {
	return func(c *drawTextConfig) {
		c.attrs = attrs
	}
}

// WithSelection highlights a selection within text drawn by DrawText.
func WithSelection(sel textbuf.TextSelection) DrawTextOption {
	return func(c *drawTextConfig) {
		c.selection = &sel
	}
}

// WithMaxWidth limits text drawn by DrawText to w columns.
func WithMaxWidth(w int) DrawTextOption {
	return func(c *drawTextConfig) {
		c.maxWidth = w
	}
}

// BoxDrawOptions configures the DrawBox method.
type BoxDrawOptions struct {
	// X and Y are the box's top-left cell, and Width and Height its size.
	X, Y, Width, Height int
	// BorderStyle selects the border's line characters.
	BorderStyle style.BorderStyle
	// Border selects the sides to draw.
	Border style.Border
	// BorderColor colors the border.
	BorderColor style.Color
	// BackgroundColor is the background of the border and, with ShouldFill,
	// of the whole box.
	BackgroundColor style.Color
	// ShouldFill fills the whole box with BackgroundColor before the border.
	ShouldFill bool
	// Title is drawn in the top border when the top side is drawn and wide
	// enough.
	Title string
	// TitleAlignment places Title along the top border.
	TitleAlignment layout.Alignment
}

// Buffer is a 2D grid of styled cells with clipping support.
type Buffer struct {
	width        int
	height       int
	cells        []Cell // Cells are stored in row-major order.
	scissor      scissorStack
	respectAlpha bool // Whether DrawFrameBuffer alpha blends this buffer as a source.
	id           string
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

// Option configures a Buffer during construction.
type Option func(*Buffer)

// WithRespectAlpha configures whether DrawFrameBuffer alpha blends the new
// buffer when it is used as a source.
func WithRespectAlpha(v bool) Option {
	return func(b *Buffer) { b.respectAlpha = v }
}

// WithID sets the new buffer's identifier.
func WithID(id string) Option {
	return func(b *Buffer) { b.id = id }
}

// NewWithOptions allocates a buffer with configurable options.
func NewWithOptions(width, height int, opts ...Option) *Buffer {
	b := New(width, height)
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// RuneWidthForBuffer returns the number of terminal columns occupied by r.
func (b *Buffer) RuneWidthForBuffer(r rune) int {
	return RuneWidth(r)
}

// StringWidthForBuffer returns the number of terminal columns occupied by s.
func (b *Buffer) StringWidthForBuffer(s string) int {
	return textbuf.StringWidth(s)
}

// fill sets every cell to c.
func (b *Buffer) fill(c Cell) {
	for i := range b.cells {
		b.cells[i] = c
	}
}

// Width returns the buffer width in cells.
func (b *Buffer) Width() int { return b.width }

// Height returns the buffer height in cells.
func (b *Buffer) Height() int { return b.height }

// ID returns the buffer identifier.
func (b *Buffer) ID() string { return b.id }

// Bounds returns the rectangular bounds of b.
func (b *Buffer) Bounds() Rect {
	return Rect{0, 0, b.width, b.height}
}

// SetRespectAlpha controls whether DrawFrameBuffer alpha blends b when b is
// used as a source.
func (b *Buffer) SetRespectAlpha(v bool) { b.respectAlpha = v }

// RespectAlpha reports whether DrawFrameBuffer alpha blends b when b is used
// as a source.
func (b *Buffer) RespectAlpha() bool { return b.respectAlpha }

func (b *Buffer) cellIndex(x, y int) int { return y*b.width + x }

func (b *Buffer) inBounds(x, y int) bool {
	return x >= 0 && x < b.width && y >= 0 && y < b.height
}

func (b *Buffer) isClipped(x, y int) bool {
	return !b.scissor.Contains(x, y)
}

// SetCellCombining sets the grapheme-cluster continuation for the cell at
// (x, y). It does nothing when (x, y) is outside the buffer.
func (b *Buffer) SetCellCombining(x, y int, combining string) {
	if !b.inBounds(x, y) {
		return
	}
	b.cells[b.cellIndex(x, y)].combining = combiningHandle(combining)
}

// SetGraphemeCluster writes cluster at (x, y). It stores the first code point
// in Cell.Char and the remaining code points in Cell.Combining.
//
// Cell.Width records the display width of the complete cluster, which can
// exceed the width of its first rune. The terminal writes every code point in
// the cluster from this cell, so the recorded width must equal the number of
// columns by which the terminal advances.
func (b *Buffer) SetGraphemeCluster(x, y int, cluster string, fg, bg style.Color, attrs uint32) {
	baseRune, baseSize := utf8.DecodeRuneInString(cluster)
	b.setCluster(x, y, baseRune, cluster[baseSize:], textbuf.StringWidth(cluster), fg, bg, attrs)
}

// ClusterStyler customizes a grapheme cluster before DrawClusters paints it.
// The col argument is the cluster's column offset from the drawing origin. If
// skip is true, DrawClusters advances past the cluster without painting it.
type ClusterStyler func(col int, fg, bg style.Color, attrs uint32) (outFg, outBg style.Color, outAttrs uint32, skip bool)

// DrawClusters paints grapheme clusters from text at (x, y), using at most
// maxWidth columns. It stops before the first cluster that would cross the
// right boundary and returns the number of columns consumed.
//
// If styler is nil, every cluster uses the supplied colors and attributes.
// Otherwise, DrawClusters calls styler once for each cluster that fits.
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

// Cell returns the cell at (x, y), or CellClear when the coordinates are
// outside the buffer.
func (b *Buffer) Cell(x, y int) Cell {
	if !b.inBounds(x, y) {
		return CellClear
	}
	return b.cells[b.cellIndex(x, y)]
}

// SetCell writes char at (x, y), subject to the current clipping region. It
// does nothing when (x, y) is outside the buffer or the clipping region.
//
// A wide character occupies a head cell followed by continuation cells. If
// the complete character does not fit, SetCell discards the write to prevent
// the terminal from advancing beyond the buffer width.
func (b *Buffer) SetCell(x, y int, char rune, fg, bg style.Color, attrs uint32) {
	b.setCluster(x, y, char, "", RuneWidth(char), fg, bg, attrs)
}

// detachWide replaces the other cells in the cluster at (x, y) with spaces.
// A continuation cell's negative offset identifies the cluster head.
func (b *Buffer) detachWide(x, y int) {
	head := x
	if c := b.Cell(x, y); c.IsContinuation() {
		head += int(c.Width)
	}
	w := int(b.Cell(head, y).Width)
	if head < 0 || w <= max(x-head, 1) {
		return
	}
	for col := head; col < head+min(w, b.width-head); col++ {
		if col != x && (col == head || int(b.Cell(col, y).Width) == head-col) {
			b.blankAt(col, y)
		}
	}
}

// blankAt replaces the cell at (x, y) with a single-column space while
// preserving its style.
//
// A related cluster cell can lie outside the current clipping region. In that
// case blankAt leaves it unchanged; the terminal renderer emits a stranded
// head or continuation as a space and preserves the row width.
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

// setCluster writes char and its trailing code points as one cluster spanning
// w columns.
func (b *Buffer) setCluster(x, y int, char rune, combining string, w int, fg, bg style.Color, attrs uint32) {
	b.SetCellDirect(x, y, Cell{
		Char:       char,
		combining:  combiningHandle(combining),
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      int16(w),
	})
}

// SetCellDirect writes cell at (x, y), subject to the current clipping region.
// Transparent backgrounds preserve the existing cell's background.
// Multi-column cells create continuations with matching styles. If the entire
// cell does not fit within the buffer and clipping region, the write is
// discarded.
func (b *Buffer) SetCellDirect(x, y int, cell Cell) {
	if !b.inBounds(x, y) || b.isClipped(x, y) {
		return
	}
	width := int(cell.Width)
	if width > b.width-x {
		return
	}
	for offset := 1; offset < width; offset++ {
		if b.isClipped(x+offset, y) {
			return
		}
	}
	// Rewriting a continuation must not detach it from its existing head.
	if !cell.IsContinuation() {
		for offset := range max(width, 1) {
			b.detachWide(x+offset, y)
		}
	}
	idx := b.cellIndex(x, y)
	if cell.Bg.IsTransparent() {
		cell.Bg = b.cells[idx].Bg
	}
	b.cells[idx] = cell
	for offset := 1; offset < width; offset++ {
		b.cells[idx+offset] = Cell{
			Fg:         cell.Fg,
			Bg:         cell.Bg,
			Attributes: cell.Attributes,
			Width:      int16(-offset),
		}
	}
}

// SetCellBlended alpha blends char over the cell at (x, y), subject to the
// current clipping region. It discards a wide character that does
// not fit in the buffer.
func (b *Buffer) SetCellBlended(x, y int, char rune, fg, bg style.Color, attrs uint32) {
	if !b.inBounds(x, y) || b.isClipped(x, y) {
		return
	}

	w := RuneWidth(char)

	// A wide character is invalid without room for its continuation cell.
	if w == 2 && x+1 >= b.width {
		return
	}

	src := Cell{
		Char:       char,
		Fg:         fg,
		Bg:         bg,
		Attributes: attrs,
		Width:      int16(w),
	}

	b.detachWide(x, y)
	if w == 2 {
		b.detachWide(x+1, y)
	}

	idx := b.cellIndex(x, y)
	b.cells[idx] = BlendCell(b.cells[idx], src)

	// Give the continuation cell the same blended style as the head.
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

// Clear resets every cell using the first background color, if present. With
// no argument, it uses CellClear's transparent background.
func (b *Buffer) Clear(bg ...style.Color) {
	c := CellClear
	if len(bg) > 0 {
		c.Bg = bg[0]
	}
	b.fill(c)
}

// DrawText draws text at (x, y) with fg. It preserves grapheme clusters and
// stops at the buffer edge or the configured maximum width.
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
		// Draw each tab as two spaces with the surrounding background.
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

		// Advance and clip complete grapheme clusters.
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

		// Selection colors override the text colors.
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

// FillRect fills the given rectangle with spaces using a white foreground and
// the supplied background color.
func (b *Buffer) FillRect(x, y, w, h int, bg style.Color) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			if b.inBounds(col, row) && !b.isClipped(col, row) {
				fg := style.ColorWhite
				cellBg := bg
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

// DrawBox draws a clipped box with optional border sides, background fill, and
// title.
func (b *Buffer) DrawBox(opts BoxDrawOptions) {
	sides := opts.Border.SidesConfig()
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

	// Reserve the title's cells within the visible top border.
	shouldDrawTitle := false
	var titleX, titleStartX, titleEndX int
	if opts.Title != "" && sides.Top && isAtActualTop {
		titleLen := b.StringWidthForBuffer(opts.Title)
		minTitleSpace := 4

		if opts.Width >= titleLen+minTitleSpace {
			shouldDrawTitle = true
			padding := 2

			switch opts.TitleAlignment {
			case layout.TextAlignCenter:
				titleX = startX + max(padding, (opts.Width-titleLen)/2)
			case layout.TextAlignRight:
				titleX = startX + opts.Width - padding - titleLen
			default:
				titleX = startX + padding
			}

			titleX = max(startX+padding, min(titleX, endX-titleLen))
			titleStartX = titleX
			titleEndX = titleX + titleLen - 1
		}
	}

	// Fill only the interior when the box has border sides.
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

	// Extend vertical sides into rows that have no horizontal border.
	leftBorderOnly := sides.Left && isAtActualLeft && !sides.Top && !sides.Bottom
	rightBorderOnly := sides.Right && isAtActualRight && !sides.Top && !sides.Bottom
	bottomOnlyWithVerticals := sides.Bottom && isAtActualBottom && !sides.Top && (sides.Left || sides.Right)
	topOnlyWithVerticals := sides.Top && isAtActualTop && !sides.Bottom && (sides.Left || sides.Right)

	extendVerticalsToTop := leftBorderOnly || rightBorderOnly || bottomOnlyWithVerticals
	extendVerticalsToBottom := leftBorderOnly || rightBorderOnly || topOnlyWithVerticals

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

	// Replace the reserved top-border cells with the title.
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

// DrawFrameBuffer copies src to b at (destX, destY). The first optional source
// rectangle restricts the copied region. The copy respects b's clipping
// region. If src.RespectAlpha reports true, the copy alpha blends each source
// cell over its destination.
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

			// Fully transparent source cells leave the destination unchanged.
			if srcCell.Bg.A == 0 && srcCell.Fg.A == 0 {
				continue
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

// CopyFrom replaces b's cells with src's cells. It does nothing unless their
// dimensions match.
func (b *Buffer) CopyFrom(src *Buffer) {
	if b.width != src.width || b.height != src.height {
		return
	}
	copy(b.cells, src.cells)
}

// CopyRowsFrom replaces rows [y, y+height) of b with the corresponding rows of
// src, copying only the rows both buffers hold.
//
// Buffers of differing width lay their rows out differently, so no row
// corresponds and it copies nothing.
func (b *Buffer) CopyRowsFrom(src *Buffer, y, height int) {
	if b.width != src.width {
		return
	}

	top := max(y, 0)
	bottom := min(y+height, min(b.height, src.height))
	if top >= bottom {
		return
	}
	copy(b.cells[top*b.width:bottom*b.width], src.cells[top*b.width:bottom*b.width])
}

// SetHeight changes b's height, keeping the rows both sizes hold and clearing
// the rows the change adds. Existing cells are reused when they have room, so
// growing by a row costs a row rather than a copy of the whole buffer.
//
// It resets the clipping stack, as Resize does.
func (b *Buffer) SetHeight(height int) {
	if height < 0 {
		height = 0
	}

	need := b.width * height
	if need > cap(b.cells) {
		b.cells = slices.Grow(b.cells, need-len(b.cells))
	}

	kept := min(len(b.cells), need)
	b.cells = b.cells[:need]
	for i := kept; i < need; i++ {
		b.cells[i] = CellClear
	}

	b.height = height
	b.scissor.Clear()
}

// Resize changes the dimensions of b, clears its cells, and resets its
// clipping stack.
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
}

// PushScissorRect restricts subsequent drawing to the intersection of the
// current clipping region and the given rectangle.
func (b *Buffer) PushScissorRect(x, y, w, h int) {
	b.scissor.Push(x, y, w, h)
}

// PopScissorRect restores the clipping region that preceded the most recent
// PushScissorRect call.
func (b *Buffer) PopScissorRect() {
	b.scissor.Pop()
}

// ClearScissorRects removes all clipping restrictions.
func (b *Buffer) ClearScissorRects() {
	b.scissor.Clear()
}

// Chars returns a row-major snapshot of the primary code point in each cell.
func (b *Buffer) Chars() []rune {
	out := make([]rune, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Char
	}
	return out
}

// Fgs returns a row-major snapshot of the foreground color in each cell.
func (b *Buffer) Fgs() []style.Color {
	out := make([]style.Color, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Fg
	}
	return out
}

// Bgs returns a row-major snapshot of the background color in each cell.
func (b *Buffer) Bgs() []style.Color {
	out := make([]style.Color, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Bg
	}
	return out
}

// Attributes returns a row-major snapshot of the attributes in each cell.
func (b *Buffer) Attributes() []uint32 {
	out := make([]uint32, len(b.cells))
	for i, c := range b.cells {
		out[i] = c.Attributes
	}
	return out
}

// ResolvedCharSize returns the number of UTF-8 bytes needed to encode the
// primary character of each non-continuation cell. It encodes a zero character
// as a space.
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

// WriteResolvedChars writes the primary character of each non-continuation
// cell to dst. It writes a zero character as a space and, when addLineBreaks
// is true, inserts a newline between rows. It returns the number of bytes
// written without splitting a UTF-8 encoding.
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

// PlainText returns the buffer's primary characters as plain text. When the
// first optional argument is true, it inserts a newline between rows.
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

// Lines returns the buffer contents as lines of adjacent, equally styled spans.
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

			// Merge adjacent cells that have the same style.
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
				// Flush the previous span before starting a new style.
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

// BorderConfig defines a border's sides, characters, and color.
type BorderConfig struct {
	// Style selects the border's line characters.
	Style style.BorderStyle
	// Border selects the sides to draw.
	Border style.Border
	// Color colors the border.
	Color style.Color
}

// BorderDrawOptions configures DrawBorder.
type BorderDrawOptions struct {
	// X and Y are the border's top-left cell, and Width and Height its size.
	X, Y, Width, Height int
	BorderConfig
	// BackgroundColor is the background of the border and, with ShouldFill,
	// of the area inside it.
	BackgroundColor style.Color
	// ShouldFill fills the area inside the border with BackgroundColor.
	ShouldFill bool
	// Title is drawn in the top border when the top side is drawn and wide
	// enough.
	Title string
	// TitleAlignment places Title along the top border.
	TitleAlignment layout.Alignment
}

// DrawBorder draws the configured border, optional interior fill, and optional
// title. BorderConfig.Border selects the visible sides.
func (b *Buffer) DrawBorder(opts BorderDrawOptions) {
	if opts.Width <= 0 || opts.Height <= 0 {
		return
	}

	sides := opts.Border.SidesConfig()
	chars := style.CharsForStyle(opts.Style)

	// Fill the area enclosed by the enabled sides.
	if opts.ShouldFill {
		inner := InnerRect(opts.X, opts.Y, opts.Width, opts.Height, sides)
		if inner.Width > 0 && inner.Height > 0 {
			b.FillRect(inner.X, inner.Y, inner.Width, inner.Height, opts.BackgroundColor)
		}
	}

	if sides.Top {
		b.drawHorizontalBorder(opts.X, opts.Y, opts.Width, chars,
			opts.Color, opts.BackgroundColor, sides.Left, sides.Right)
	}

	if sides.Bottom {
		by := opts.Y + opts.Height - 1
		// Use bottom corner runes where vertical sides meet the line.
		for cx := opts.X; cx < opts.X+opts.Width; cx++ {
			ch := chars.Horizontal
			if cx == opts.X && sides.Left {
				ch = chars.BottomLeft
			} else if cx == opts.X+opts.Width-1 && sides.Right {
				ch = chars.BottomRight
			} else if cx == opts.X && !sides.Left {
				// Keep the horizontal rune when there is no left side.
			} else if cx == opts.X+opts.Width-1 && !sides.Right {
				// Keep the horizontal rune when there is no right side.
			}
			b.SetCell(cx, by, ch, opts.Color, opts.BackgroundColor, 0)
		}
	}

	// Horizontal borders already drew their corner cells.
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

	// Horizontal borders already drew their corner cells.
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

	// A title replaces part of the top border between its corners.
	if opts.Title != "" && sides.Top && opts.Width > 2 {
		titleMaxW := opts.Width - 2
		b.drawBorderTitle(opts.Title, opts.X+1, opts.Y, titleMaxW,
			opts.TitleAlignment, opts.Color, opts.BackgroundColor)
	}
}

// drawHorizontalBorder draws a top border and any enabled corner cells.
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

// drawBorderTitle draws an aligned title within the maxWidth columns beginning
// at (x, y).
func (b *Buffer) drawBorderTitle(title string, x, y, maxWidth int, alignment layout.Alignment, fg, bg style.Color) {
	if maxWidth <= 0 {
		return
	}

	// Limit the title by rune count before calculating its alignment.
	runes := []rune(title)
	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}
	titleLen := len(runes)

	// Align the truncated title within the available columns.
	var startX int
	switch alignment {
	case layout.TextAlignCenter:
		startX = x + (maxWidth-titleLen)/2
	case layout.TextAlignRight:
		startX = x + maxWidth - titleLen
	default: // AlignLeft
		startX = x
	}

	// Write each rune at its rune index within the title.
	for i, r := range runes {
		b.SetCell(startX+i, y, r, fg, bg, 0)
	}
}

// drawBorderTextSimple writes text one rune per cell using SetCell.
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

// InnerRect returns the content area inside the enabled border sides. Each
// enabled side consumes one row or column, and the result never has negative
// dimensions.
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
