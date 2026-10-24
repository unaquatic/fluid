package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// TabSelectItem contains the label and caller-defined value for one tab.
type TabSelectItem struct {
	// Name is the tab's label.
	Name string
	// Value is caller-defined data for the tab.
	Value any
}

// TabSelect is a horizontal tab-style selector.
type TabSelect struct {
	*render.BaseRenderable

	items         []TabSelectItem
	selectedIndex int
	tabWidth      int  // Fixed tab width; zero distributes available width.
	showUnderline bool // Reserves a second row for the tab underline.

	textColor         style.Color
	selectedTextColor style.Color
	selectedBgColor   style.Color
	underlineColor    style.Color
	separatorColor    style.Color
	backgroundColor   style.Color

	// OnSelectionChanged is called after the selected tab changes.
	OnSelectionChanged func(index int)
	// OnItemSelected is called when the user activates the selected tab.
	OnItemSelected func(index int)
}

// TabSelectOption configures a TabSelect during construction.
type TabSelectOption func(*TabSelect)

// NewTabSelect returns a focusable horizontal selector containing items.
func NewTabSelect(items []TabSelectItem, opts ...TabSelectOption) *TabSelect {
	ts := &TabSelect{
		BaseRenderable:    render.NewBaseRenderable(render.WithFocusable(true)),
		items:             items,
		showUnderline:     true,
		textColor:         style.ColorGray,
		selectedTextColor: style.ColorWhite,
		selectedBgColor:   style.ColorTransparent,
		underlineColor:    style.ColorBrightBlue,
		separatorColor:    style.ColorFromInts(60, 60, 60, 255),
		backgroundColor:   style.ColorTransparent,
	}

	// Reserve one row initially; underline mode adds a second row below.
	ts.LayoutNode().Height = layout.Points(1)

	for _, opt := range opts {
		opt(ts)
	}

	if ts.showUnderline {
		ts.LayoutNode().Height = layout.Points(2)
	}

	return ts
}

// SelectedIndex returns the currently selected tab index.
func (ts *TabSelect) SelectedIndex() int { return ts.selectedIndex }

// SetSelectedIndex selects the index after clamping it to the items range.
func (ts *TabSelect) SetSelectedIndex(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ts.items) {
		idx = len(ts.items) - 1
	}
	if idx != ts.selectedIndex {
		ts.selectedIndex = idx
		if ts.OnSelectionChanged != nil {
			ts.OnSelectionChanged(idx)
		}
		ts.RequestRender()
	}
}

// SetItems replaces the tab items.
func (ts *TabSelect) SetItems(items []TabSelectItem) {
	ts.items = items
	if ts.selectedIndex >= len(items) {
		ts.selectedIndex = max(len(items)-1, 0)
	}
	ts.RequestRender()
}

// RenderSelf draws the tab selector.
func (ts *TabSelect) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := ts.Width()
	h := ts.Height()
	if w <= 0 || h <= 0 || len(ts.items) == 0 {
		return
	}

	x := ts.X()
	y := ts.Y()

	// Paint the selector background before tab-specific backgrounds.
	if !ts.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, ts.backgroundColor)
	}

	// Divide the viewport among the tabs and their separators.
	tabWidths := ts.calculateTabWidths(w)

	// Render tabs left to right until the viewport is full.
	cx := x
	for i, item := range ts.items {
		if cx >= x+w {
			break
		}

		tw := tabWidths[i]
		isSelected := i == ts.selectedIndex

		// Apply the selected colors to the active tab.
		fg := ts.textColor
		bg := ts.backgroundColor
		if isSelected {
			fg = ts.selectedTextColor
			if !ts.selectedBgColor.IsTransparent() {
				bg = ts.selectedBgColor
			}
		}

		// Fill tab background
		if !bg.IsTransparent() {
			buf.FillRect(cx, y, tw, 1, bg)
		}

		// Truncate and center the tab label within its allocated width.
		name := item.Name
		nameWidth := ts.StringWidth(name)
		if nameWidth > tw {
			name = truncateToWidth(name, tw)
			nameWidth = ts.StringWidth(name)
		}
		textX := cx + (tw-nameWidth)/2
		buf.DrawText(name, textX, y, fg)

		// Use a heavy underline for the selected tab and a light rule elsewhere.
		if ts.showUnderline && h >= 2 {
			if isSelected {
				for ux := cx; ux < cx+tw; ux++ {
					buf.SetCell(ux, y+1, symbol.HeavyHorizontal, ts.underlineColor, ts.backgroundColor, 0)
				}
			} else {
				for ux := cx; ux < cx+tw; ux++ {
					buf.SetCell(ux, y+1, symbol.Horizontal, ts.separatorColor, ts.backgroundColor, 0)
				}
			}
		}

		cx += tw

		// Separate adjacent tabs in both rows.
		if i < len(ts.items)-1 && cx < x+w {
			buf.SetCell(cx, y, symbol.Vertical, ts.separatorColor, ts.backgroundColor, 0)
			if ts.showUnderline && h >= 2 {
				buf.SetCell(cx, y+1, symbol.Horizontal, ts.separatorColor, ts.backgroundColor, 0)
			}
			cx++
		}
	}
}

// HandleKeyPress moves or activates the selected tab for a supported key event.
func (ts *TabSelect) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(ts.items) == 0 {
		return false
	}

	handled := true

	switch {
	case e.Matches(input.KeyLeft, 0) || e.Matches("h", 0):
		if ts.selectedIndex > 0 {
			ts.selectedIndex--
			if ts.OnSelectionChanged != nil {
				ts.OnSelectionChanged(ts.selectedIndex)
			}
			ts.RequestRender()
		}

	case e.Matches(input.KeyRight, 0) || e.Matches("l", 0):
		if ts.selectedIndex < len(ts.items)-1 {
			ts.selectedIndex++
			if ts.OnSelectionChanged != nil {
				ts.OnSelectionChanged(ts.selectedIndex)
			}
			ts.RequestRender()
		}

	case e.Matches(input.KeyReturn, 0) || e.Matches(input.KeySpace, 0):
		if ts.OnItemSelected != nil {
			ts.OnItemSelected(ts.selectedIndex)
		}

	case e.Matches(input.KeyHome, 0):
		ts.SetSelectedIndex(0)

	case e.Matches(input.KeyEnd, 0):
		ts.SetSelectedIndex(len(ts.items) - 1)

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

// calculateTabWidths divides the available width among tabs.
func (ts *TabSelect) calculateTabWidths(totalWidth int) []int {
	n := len(ts.items)
	if n == 0 {
		return nil
	}

	// Exclude one separator column between each pair of tabs.
	separatorSpace := n - 1
	availableWidth := totalWidth - separatorSpace

	if ts.tabWidth > 0 {
		// A configured width applies equally to every tab.
		widths := make([]int, n)
		for i := range widths {
			widths[i] = ts.tabWidth
		}
		return widths
	}

	// Give remainder columns to the leftmost tabs.
	baseWidth := availableWidth / n
	remainder := availableWidth % n

	widths := make([]int, n)
	for i := range widths {
		widths[i] = baseWidth
		if i < remainder {
			widths[i]++
		}
	}

	return widths
}

// TabSelect options.

// TabSelectWithTabWidth sets a fixed width for every tab; zero uses even distribution.
func TabSelectWithTabWidth(w int) TabSelectOption {
	return func(ts *TabSelect) { ts.tabWidth = w }
}

// TabSelectWithShowUnderline controls whether the selector renders an underline row.
func TabSelectWithShowUnderline(v bool) TabSelectOption {
	return func(ts *TabSelect) { ts.showUnderline = v }
}

// TabSelectWithTextColor sets the default text color.
func TabSelectWithTextColor(c style.Color) TabSelectOption {
	return func(ts *TabSelect) { ts.textColor = c }
}

// TabSelectWithSelectedTextColor sets the selected text color.
func TabSelectWithSelectedTextColor(c style.Color) TabSelectOption {
	return func(ts *TabSelect) { ts.selectedTextColor = c }
}

// TabSelectWithSelectedBgColor sets the selected background color.
func TabSelectWithSelectedBgColor(c style.Color) TabSelectOption {
	return func(ts *TabSelect) { ts.selectedBgColor = c }
}

// TabSelectWithUnderlineColor sets the underline color.
func TabSelectWithUnderlineColor(c style.Color) TabSelectOption {
	return func(ts *TabSelect) { ts.underlineColor = c }
}

// TabSelectWithBackgroundColor sets the background color.
func TabSelectWithBackgroundColor(c style.Color) TabSelectOption {
	return func(ts *TabSelect) { ts.backgroundColor = c }
}

// TabSelectWithOnSelectionChanged sets the callback invoked after selection changes.
func TabSelectWithOnSelectionChanged(fn func(int)) TabSelectOption {
	return func(ts *TabSelect) { ts.OnSelectionChanged = fn }
}

// TabSelectWithOnItemSelected sets the callback invoked when the user activates a tab.
func TabSelectWithOnItemSelected(fn func(int)) TabSelectOption {
	return func(ts *TabSelect) { ts.OnItemSelected = fn }
}

// TabSelectWithID sets the widget ID.
func TabSelectWithID(id string) TabSelectOption {
	return func(ts *TabSelect) { ts.SetID(id) }
}

// TabSelectWithWidth sets the widget width.
func TabSelectWithWidth(s layout.Scalar) TabSelectOption {
	return func(ts *TabSelect) { ts.LayoutNode().Width = s }
}

// TabSelectWithHeight sets the widget height.
func TabSelectWithHeight(s layout.Scalar) TabSelectOption {
	return func(ts *TabSelect) { ts.LayoutNode().Height = s }
}
