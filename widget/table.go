package widget

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// TableColumn describes the name and sizing policy for one table column.
type TableColumn struct {
	// Name is the column's header text.
	Name string
	// Width is the fixed width; zero divides the available width evenly.
	Width int
	// MinWidth is not applied when Table sizes columns.
	MinWidth int
	// MaxWidth is not applied when Table sizes columns.
	MaxWidth int
	// Alignment is zero for left, one for center, and two for right.
	Alignment int
}

// Table is a keyboard-navigable table with an optional header and striped rows.
type Table struct {
	*render.BaseRenderable

	columns      []TableColumn
	rows         [][]string
	selectedRow  int
	scrollOffset int

	headerFg    style.Color
	headerBg    style.Color
	rowFg       style.Color
	rowBg       style.Color
	altRowBg    style.Color
	selectedFg  style.Color
	selectedBg  style.Color
	borderColor style.Color

	showHeader    bool
	showBorder    bool
	stripedRows   bool
	wrapSelection bool

	// OnRowSelected is called when the user activates the selected row.
	OnRowSelected func(index int, row []string)
}

// TableOption configures a Table.
type TableOption func(*Table)

// NewTable returns a focusable Table containing columns and rows.
func NewTable(columns []TableColumn, rows [][]string, opts ...TableOption) *Table {
	t := &Table{
		BaseRenderable: render.NewBaseRenderable(render.WithFocusable(true)),
		columns:        columns,
		rows:           rows,
		headerFg:       style.ColorWhite,
		headerBg:       style.ColorFromInts(40, 40, 40, 255),
		rowFg:          style.ColorWhite,
		rowBg:          style.ColorTransparent,
		altRowBg:       style.ColorFromInts(25, 25, 25, 255),
		selectedFg:     style.ColorWhite,
		selectedBg:     style.ColorFromInts(51, 153, 255, 255),
		borderColor:    style.ColorGray,
		showHeader:     true,
		showBorder:     false,
		stripedRows:    true,
		wrapSelection:  true,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// SetRows replaces the table rows.
func (t *Table) SetRows(rows [][]string) {
	t.rows = rows
	if t.selectedRow >= len(rows) {
		t.selectedRow = max(len(rows)-1, 0)
	}
	t.RequestRender()
}

// SelectedRow returns the zero-based index of the selected row.
func (t *Table) SelectedRow() int { return t.selectedRow }

// RenderSelf draws the table.
func (t *Table) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	w := t.Width()
	h := t.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := t.X()
	y := t.Y()

	// Resolve fixed and automatic widths for this viewport.
	colWidths := t.calculateColumnWidths(w)

	drawY := y

	// The header, when enabled, consumes the first visible row.
	if t.showHeader && drawY < y+h {
		cx := x
		if !t.headerBg.IsTransparent() {
			buf.FillRect(x, drawY, w, 1, t.headerBg)
		}
		for i, col := range t.columns {
			if i >= len(colWidths) {
				break
			}
			name := truncateToWidth(col.Name, colWidths[i])
			buf.DrawText(name, cx, drawY, t.headerFg)
			cx += colWidths[i] + 1 // Reserve one column between fields.
		}
		drawY++
	}

	// Render rows from the scroll offset until the viewport is full.
	for rowIdx := t.scrollOffset; rowIdx < len(t.rows) && drawY < y+h; rowIdx++ {
		row := t.rows[rowIdx]
		isSelected := rowIdx == t.selectedRow

		fg := t.rowFg
		bg := t.rowBg
		if isSelected && t.Focused() {
			fg = t.selectedFg
			bg = t.selectedBg
		} else if t.stripedRows && rowIdx%2 == 1 {
			bg = t.altRowBg
		}

		if !bg.IsTransparent() {
			buf.FillRect(x, drawY, w, 1, bg)
		}

		cx := x
		for i := 0; i < len(colWidths) && i < len(row); i++ {
			cell := truncateToWidth(row[i], colWidths[i])
			buf.DrawText(cell, cx, drawY, fg)
			cx += colWidths[i] + 1
		}
		drawY++
	}
}

// HandleKeyPress moves or activates the selected row for a supported key event.
func (t *Table) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(t.rows) == 0 {
		return false
	}

	handled := true
	switch {
	case e.Matches(input.KeyUp, 0) || e.Matches("k", 0):
		if t.selectedRow > 0 {
			t.selectedRow--
		} else if t.wrapSelection {
			t.selectedRow = len(t.rows) - 1
		}
		t.ensureRowVisible()
		t.RequestRender()

	case e.Matches(input.KeyDown, 0) || e.Matches("j", 0):
		if t.selectedRow < len(t.rows)-1 {
			t.selectedRow++
		} else if t.wrapSelection {
			t.selectedRow = 0
		}
		t.ensureRowVisible()
		t.RequestRender()

	case e.Matches(input.KeyReturn, 0):
		if t.OnRowSelected != nil && t.selectedRow >= 0 && t.selectedRow < len(t.rows) {
			t.OnRowSelected(t.selectedRow, t.rows[t.selectedRow])
		}

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

func (t *Table) calculateColumnWidths(totalWidth int) []int {
	n := len(t.columns)
	if n == 0 {
		return nil
	}

	separatorSpace := n - 1
	available := totalWidth - separatorSpace

	widths := make([]int, n)
	for i, col := range t.columns {
		if col.Width > 0 {
			widths[i] = col.Width
		} else {
			widths[i] = available / n
		}
	}
	return widths
}

func (t *Table) ensureRowVisible() {
	h := t.Height()
	headerOffset := 0
	if t.showHeader {
		headerOffset = 1
	}
	visible := h - headerOffset

	if t.selectedRow < t.scrollOffset {
		t.scrollOffset = t.selectedRow
	}
	if t.selectedRow >= t.scrollOffset+visible {
		t.scrollOffset = t.selectedRow - visible + 1
	}
}

// Table options.

// TableWithShowHeader controls whether the column header is displayed.
func TableWithShowHeader(v bool) TableOption {
	return func(t *Table) { t.showHeader = v }
}

// TableWithStripedRows controls whether alternate rows use a different background.
func TableWithStripedRows(v bool) TableOption {
	return func(t *Table) { t.stripedRows = v }
}

// TableWithSelectedBg sets the selected row background color.
func TableWithSelectedBg(c style.Color) TableOption {
	return func(t *Table) { t.selectedBg = c }
}

// TableWithOnRowSelected sets the callback invoked when the user activates a row.
func TableWithOnRowSelected(fn func(int, []string)) TableOption {
	return func(t *Table) { t.OnRowSelected = fn }
}

// TableWithID sets the widget ID.
func TableWithID(id string) TableOption {
	return func(t *Table) { t.SetID(id) }
}

// TableWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func TableWithLayout(opts ...layout.Option) TableOption {
	return func(w *Table) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}
