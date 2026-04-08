package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// TabSelectOption represents a single tab option.
type TabSelectOption struct {
	Name  string
	Value any
}

// TabSelect is a horizontal tab-style selector.
type TabSelect struct {
	*render.BaseRenderable

	options       []TabSelectOption
	selectedIndex int
	tabWidth      int  // fixed tab width (0 = auto)
	showUnderline bool // show underline on selected tab

	textColor         style.Color
	selectedTextColor style.Color
	selectedBgColor   style.Color
	underlineColor    style.Color
	separatorColor    style.Color
	backgroundColor   style.Color

	// Event callbacks
	OnSelectionChanged func(index int)
	OnItemSelected     func(index int)
}

// TabSelectOpt configures a TabSelect during construction.
type TabSelectOpt func(*TabSelect)

// NewTabSelect creates a TabSelect widget.
func NewTabSelect(options []TabSelectOption, opts ...TabSelectOpt) *TabSelect {
	ts := &TabSelect{
		BaseRenderable:    render.NewBaseRenderable(render.WithFocusable(true)),
		options:           options,
		showUnderline:     true,
		textColor:         style.ColorGray,
		selectedTextColor: style.ColorWhite,
		selectedBgColor:   style.ColorTransparent,
		underlineColor:    style.ColorBrightBlue,
		separatorColor:    style.ColorFromInts(60, 60, 60, 255),
		backgroundColor:   style.ColorTransparent,
	}

	// Default height: 1 row (2 if underline)
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

// SetSelectedIndex sets the selected tab index.
func (ts *TabSelect) SetSelectedIndex(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ts.options) {
		idx = len(ts.options) - 1
	}
	if idx != ts.selectedIndex {
		ts.selectedIndex = idx
		if ts.OnSelectionChanged != nil {
			ts.OnSelectionChanged(idx)
		}
		ts.RequestRender()
	}
}

// SetOptions replaces the tab options.
func (ts *TabSelect) SetOptions(options []TabSelectOption) {
	ts.options = options
	if ts.selectedIndex >= len(options) {
		ts.selectedIndex = max(len(options)-1, 0)
	}
	ts.RequestRender()
}

// RenderSelf draws the tab selector.
func (ts *TabSelect) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := ts.Width()
	h := ts.Height()
	if w <= 0 || h <= 0 || len(ts.options) == 0 {
		return
	}

	x := ts.X()
	y := ts.Y()

	// Fill background
	if !ts.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, ts.backgroundColor)
	}

	// Calculate tab widths
	tabWidths := ts.calculateTabWidths(w)

	// Draw tabs
	cx := x
	for i, opt := range ts.options {
		if cx >= x+w {
			break
		}

		tw := tabWidths[i]
		isSelected := i == ts.selectedIndex

		// Determine colors
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

		// Draw tab text (centered)
		name := opt.Name
		nameWidth := ts.StringWidth(name)
		if nameWidth > tw {
			name = truncateToWidth(name, tw)
			nameWidth = ts.StringWidth(name)
		}
		textX := cx + (tw-nameWidth)/2
		buf.DrawText(name, textX, y, fg)

		// Draw underline for selected tab
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

		// Draw separator between tabs
		if i < len(ts.options)-1 && cx < x+w {
			buf.SetCell(cx, y, symbol.Vertical, ts.separatorColor, ts.backgroundColor, 0)
			if ts.showUnderline && h >= 2 {
				buf.SetCell(cx, y+1, symbol.Horizontal, ts.separatorColor, ts.backgroundColor, 0)
			}
			cx++
		}
	}
}

// HandleKeyPress processes keyboard input.
func (ts *TabSelect) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(ts.options) == 0 {
		return false
	}

	handled := true

	switch {
	case e.Matches(input.KeyLeft) || e.Matches("h"):
		if ts.selectedIndex > 0 {
			ts.selectedIndex--
			if ts.OnSelectionChanged != nil {
				ts.OnSelectionChanged(ts.selectedIndex)
			}
			ts.RequestRender()
		}

	case e.Matches(input.KeyRight) || e.Matches("l"):
		if ts.selectedIndex < len(ts.options)-1 {
			ts.selectedIndex++
			if ts.OnSelectionChanged != nil {
				ts.OnSelectionChanged(ts.selectedIndex)
			}
			ts.RequestRender()
		}

	case e.Matches(input.KeyReturn) || e.Matches(input.KeySpace):
		if ts.OnItemSelected != nil {
			ts.OnItemSelected(ts.selectedIndex)
		}

	case e.Matches(input.KeyHome):
		ts.SetSelectedIndex(0)

	case e.Matches(input.KeyEnd):
		ts.SetSelectedIndex(len(ts.options) - 1)

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

// calculateTabWidths distributes available width among tabs.
func (ts *TabSelect) calculateTabWidths(totalWidth int) []int {
	n := len(ts.options)
	if n == 0 {
		return nil
	}

	// Account for separators
	separatorSpace := n - 1
	availableWidth := totalWidth - separatorSpace

	if ts.tabWidth > 0 {
		// Fixed tab width
		widths := make([]int, n)
		for i := range widths {
			widths[i] = ts.tabWidth
		}
		return widths
	}

	// Distribute evenly
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

// --- TabSelect Options ---

// TabSelectWithTabWidth sets fixed tab width.
func TabSelectWithTabWidth(w int) TabSelectOpt {
	return func(ts *TabSelect) { ts.tabWidth = w }
}

// TabSelectWithShowUnderline enables/disables underline.
func TabSelectWithShowUnderline(v bool) TabSelectOpt {
	return func(ts *TabSelect) { ts.showUnderline = v }
}

// TabSelectWithTextColor sets the default text color.
func TabSelectWithTextColor(c style.Color) TabSelectOpt {
	return func(ts *TabSelect) { ts.textColor = c }
}

// TabSelectWithSelectedTextColor sets the selected text color.
func TabSelectWithSelectedTextColor(c style.Color) TabSelectOpt {
	return func(ts *TabSelect) { ts.selectedTextColor = c }
}

// TabSelectWithSelectedBgColor sets the selected background color.
func TabSelectWithSelectedBgColor(c style.Color) TabSelectOpt {
	return func(ts *TabSelect) { ts.selectedBgColor = c }
}

// TabSelectWithUnderlineColor sets the underline color.
func TabSelectWithUnderlineColor(c style.Color) TabSelectOpt {
	return func(ts *TabSelect) { ts.underlineColor = c }
}

// TabSelectWithBackgroundColor sets the background color.
func TabSelectWithBackgroundColor(c style.Color) TabSelectOpt {
	return func(ts *TabSelect) { ts.backgroundColor = c }
}

// TabSelectWithOnSelectionChanged sets the selection changed callback.
func TabSelectWithOnSelectionChanged(fn func(int)) TabSelectOpt {
	return func(ts *TabSelect) { ts.OnSelectionChanged = fn }
}

// TabSelectWithOnItemSelected sets the item selected callback.
func TabSelectWithOnItemSelected(fn func(int)) TabSelectOpt {
	return func(ts *TabSelect) { ts.OnItemSelected = fn }
}

// TabSelectWithID sets the widget ID.
func TabSelectWithID(id string) TabSelectOpt {
	return func(ts *TabSelect) { ts.SetID(id) }
}

// TabSelectWithWidth sets the widget width.
func TabSelectWithWidth(s layout.Scalar) TabSelectOpt {
	return func(ts *TabSelect) { ts.LayoutNode().Width = s }
}

// TabSelectWithHeight sets the widget height.
func TabSelectWithHeight(s layout.Scalar) TabSelectOpt {
	return func(ts *TabSelect) { ts.LayoutNode().Height = s }
}
