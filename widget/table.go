package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// TableColumn defines a column in the table.
type TableColumn struct {
	Name      string
	Width     int // fixed width (0 = auto)
	MinWidth  int
	MaxWidth  int
	Alignment int // 0=left, 1=center, 2=right
}

// Table is a scrollable, selectable table with headers and row selection.
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

	OnRowSelected func(index int, row []string)
}

// TableOption configures a Table.
type TableOption func(*Table)

// NewTable creates a Table widget.
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

// SelectedRow returns the selected row index.
func (t *Table) SelectedRow() int { return t.selectedRow }

// RenderSelf draws the table.
func (t *Table) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := t.Width()
	h := t.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := t.X()
	y := t.Y()

	// Calculate column widths
	colWidths := t.calculateColumnWidths(w)

	drawY := y

	// Draw header
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
			cx += colWidths[i] + 1 // +1 for separator
		}
		drawY++
	}

	// Draw rows
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

// HandleKeyPress processes keyboard input.
func (t *Table) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(t.rows) == 0 {
		return false
	}

	handled := true
	switch {
	case e.Matches(input.KeyUp) || e.Matches("k"):
		if t.selectedRow > 0 {
			t.selectedRow--
		} else if t.wrapSelection {
			t.selectedRow = len(t.rows) - 1
		}
		t.ensureRowVisible()
		t.RequestRender()

	case e.Matches(input.KeyDown) || e.Matches("j"):
		if t.selectedRow < len(t.rows)-1 {
			t.selectedRow++
		} else if t.wrapSelection {
			t.selectedRow = 0
		}
		t.ensureRowVisible()
		t.RequestRender()

	case e.Matches(input.KeyReturn):
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

// --- Table Options ---

// TableWithShowHeader enables/disables header display.
func TableWithShowHeader(v bool) TableOption {
	return func(t *Table) { t.showHeader = v }
}

// TableWithStripedRows enables/disables striped row backgrounds.
func TableWithStripedRows(v bool) TableOption {
	return func(t *Table) { t.stripedRows = v }
}

// TableWithSelectedBg sets the selected row background color.
func TableWithSelectedBg(c style.Color) TableOption {
	return func(t *Table) { t.selectedBg = c }
}

// TableWithOnRowSelected sets the row selection callback.
func TableWithOnRowSelected(fn func(int, []string)) TableOption {
	return func(t *Table) { t.OnRowSelected = fn }
}

// TableWithID sets the widget ID.
func TableWithID(id string) TableOption {
	return func(t *Table) { t.SetID(id) }
}

// TableWithWidth sets the widget width.
func TableWithWidth(s layout.Scalar) TableOption {
	return func(t *Table) { t.LayoutNode().Width = s }
}

// TableWithHeight sets the widget height.
func TableWithHeight(s layout.Scalar) TableOption {
	return func(t *Table) { t.LayoutNode().Height = s }
}

// TableWithFlexGrow sets flex grow.
func TableWithFlexGrow(v int) TableOption {
	return func(t *Table) { t.LayoutNode().FlexGrow = v }
}
