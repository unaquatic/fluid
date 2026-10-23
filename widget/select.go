package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// SelectItem describes one row in a Select widget.
type SelectItem struct {
	// Name is the row's displayed name.
	Name string
	// Hint is displayed after Name using the description color.
	Hint string
	// Description is shown in a column after Name with inline descriptions,
	// otherwise on the row below.
	Description string
	// Value is caller-defined data for the row.
	Value any
	// Disabled prevents selection, for example on a section header.
	Disabled bool
}

// Select is a keyboard-navigable list of items.
type Select struct {
	*render.BaseRenderable

	items         []SelectItem
	selectedIndex int
	scrollOffset  int

	backgroundColor         style.Color
	textColor               style.Color
	focusedBackgroundColor  style.Color
	focusedTextColor        style.Color
	selectedBackgroundColor style.Color
	selectedTextColor       style.Color
	descriptionColor        style.Color
	// hintColor, when set, colors the hint after each name on every row, the
	// selected one included.
	hintColor *style.Color

	showScrollIndicator bool
	wrapSelection       bool
	showDescription     bool
	inlineDescription   bool
	nameWidth           int    // Name-column width when inline descriptions are enabled.
	itemPaddingX        int    // Horizontal inset for the selected row background.
	focusedUseBold      bool   // Uses bold instead of a background to show the focused item.
	showSelectionArrow  bool   // Draws an arrow over the selected row's first cell.
	arrowColumn         bool   // Reserves arrowColumnWidth cells before every name for the arrow.
	selectionIndicator  string // Reserved prefix for the selected item.

	// OnSelectionChanged is called after keyboard or programmatic selection changes.
	OnSelectionChanged func(index int, item SelectItem)
	// OnItemSelected is called when the user activates the selected item.
	OnItemSelected func(index int, item SelectItem)
}

// SelectOption configures a Select during construction.
type SelectOption func(*Select)

// NewSelect returns a focusable selector containing items.
func NewSelect(items []SelectItem, opts ...SelectOption) *Select {
	s := &Select{
		BaseRenderable:          render.NewBaseRenderable(render.WithFocusable(true)),
		items:                   items,
		selectedIndex:           0,
		backgroundColor:         style.ColorTransparent,
		textColor:               style.ColorWhite,
		focusedBackgroundColor:  style.ColorFromHex("#404040"),
		focusedTextColor:        style.ColorWhite,
		selectedBackgroundColor: style.ColorFromHex("#404040"),
		selectedTextColor:       style.ColorWhite,
		descriptionColor:        style.ColorWhite,
		showScrollIndicator:     false,
		wrapSelection:           true,
		showDescription:         true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SelectedIndex returns the currently selected index.
func (s *Select) SelectedIndex() int { return s.selectedIndex }

// ItemCount returns the number of items.
func (s *Select) ItemCount() int { return len(s.items) }

// SelectedItem returns the currently selected item, or nil if none.
func (s *Select) SelectedItem() *SelectItem {
	if s.selectedIndex < 0 || s.selectedIndex >= len(s.items) {
		return nil
	}
	return &s.items[s.selectedIndex]
}

// SetSelectedIndex selects the index after clamping it to the items range.
func (s *Select) SetSelectedIndex(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.items) {
		idx = len(s.items) - 1
	}
	if idx != s.selectedIndex {
		s.selectedIndex = idx
		s.ensureVisible()
		if s.OnSelectionChanged != nil && idx >= 0 && idx < len(s.items) {
			s.OnSelectionChanged(idx, s.items[idx])
		}
		s.RequestRender()
	}
}

// SetScrollOffset sets the first item considered for rendering.
func (s *Select) SetScrollOffset(offset int) {
	s.scrollOffset = offset
}

// SetItems replaces the list's items.
func (s *Select) SetItems(items []SelectItem) {
	s.items = items
	if s.selectedIndex >= len(items) {
		s.selectedIndex = max(len(items)-1, 0)
	}
	s.scrollOffset = 0
	s.RequestRender()
}

// Items returns the current item slice.
func (s *Select) Items() []SelectItem { return s.items }

// RenderSelf draws the select list.
func (s *Select) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := s.Width()
	h := s.Height()
	if w <= 0 || h <= 0 || len(s.items) == 0 {
		return
	}

	x := s.X()
	y := s.Y()

	// Paint the list background before row-specific highlights.
	if !s.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, s.backgroundColor)
	}

	// The draw loop below clips rows directly against the widget height.
	_ = h // visibleCount was computed here but the draw loop uses drawY bounds directly

	px := s.itemPaddingX

	// Render items from the current scroll offset until the viewport is full.
	drawY := y
	for i := s.scrollOffset; i < len(s.items) && drawY < y+h; i++ {
		item := s.items[i]
		isSelected := i == s.selectedIndex

		// Resolve the selected-row appearance from focus state and options.
		fg := s.textColor
		bg := s.backgroundColor
		var attrs uint32

		if isSelected {
			if s.focusedUseBold {
				attrs = style.AttrBold
			} else if s.Focused() {
				fg = s.focusedTextColor
				bg = s.focusedBackgroundColor
			} else {
				fg = s.selectedTextColor
				bg = s.selectedBackgroundColor
			}
		}

		// Draw item row background. itemPaddingX adjusts the selected
		// row highlight: positive insets it, negative extends it outward.
		if !bg.IsTransparent() && bg != s.backgroundColor {
			if isSelected && px != 0 {
				buf.FillRect(x+px, drawY, w-px*2, 1, bg)
			} else {
				buf.FillRect(x, drawY, w, 1, bg)
			}
		}

		// Draw name text, after the arrow column when one is reserved.
		nameX, nameW := x, w
		if s.arrowColumn {
			nameX, nameW = x+arrowColumnWidth, max(w-arrowColumnWidth, 0)
		}
		nameLen := displayWidth(item.Name)
		nameStr := truncateToWidth(item.Name, nameW)
		var nameOpts []buffer.DrawTextOption
		if attrs != 0 {
			nameOpts = append(nameOpts, buffer.WithAttrs(attrs))
		}
		buf.DrawText(nameStr, nameX, drawY, fg, nameOpts...)

		// Draw the arrow last so it replaces the row's leading cell.
		if isSelected && s.showSelectionArrow {
			buf.SetCell(x, drawY, '→', fg, s.backgroundColor, attrs)
		}

		// Draw hint right after the name in description color.
		if item.Hint != "" && nameLen < nameW {
			hintX := nameX + nameLen
			hint := " " + item.Hint
			hint = truncateToWidth(hint, nameW-nameLen)
			hintFg := s.descriptionColor
			if isSelected {
				hintFg = fg
			}
			if s.hintColor != nil {
				hintFg = *s.hintColor
			}
			buf.DrawText(hint, hintX, drawY, hintFg)
			nameLen += displayWidth(hint)
		}

		// Draw inline description after the name column.
		if s.inlineDescription && item.Description != "" {
			descX := nameX + s.nameWidth
			if nameLen > s.nameWidth {
				descX = nameX + nameLen + 1
			}
			if descX < x+w {
				desc := truncateToWidth(item.Description, x+w-descX)
				descFg := s.descriptionColor
				if isSelected {
					descFg = fg
				}
				buf.DrawText(desc, descX, drawY, descFg)
			}
		}
		drawY++

		// A non-inline description consumes the following viewport row.
		if s.showDescription && !s.inlineDescription && item.Description != "" && drawY < y+h {
			desc := truncateToWidth(item.Description, w-2)
			buf.DrawText("  "+desc, x, drawY, s.descriptionColor)
			drawY++
		}
	}

	// Mark hidden items above or below the viewport.
	if s.showScrollIndicator {
		if s.scrollOffset > 0 {
			buf.SetCell(x+w-1, y, symbol.ArrowUp, s.textColor, s.backgroundColor, 0)
		}
		lastVisible := s.scrollOffset + s.visibleOptionCount()
		if lastVisible < len(s.items) {
			buf.SetCell(x+w-1, y+h-1, symbol.ArrowDown, s.textColor, s.backgroundColor, 0)
		}
	}
}

// HandleKeyPress moves or activates the selection for a supported key event.
func (s *Select) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(s.items) == 0 {
		return false
	}

	handled := true

	switch {
	case e.Matches(input.KeyUp) || e.Matches("k"):
		s.MoveUp()

	case e.Matches(input.KeyDown) || e.Matches("j"):
		s.MoveDown()

	case e.Matches(input.KeyHome):
		s.SetSelectedIndex(0)

	case e.Matches(input.KeyEnd):
		s.SetSelectedIndex(len(s.items) - 1)

	case e.Matches(input.KeyReturn) || e.Matches(input.KeySpace):
		if s.OnItemSelected != nil && s.selectedIndex >= 0 && s.selectedIndex < len(s.items) {
			s.OnItemSelected(s.selectedIndex, s.items[s.selectedIndex])
		}

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

// MoveUp selects the previous enabled item.
func (s *Select) MoveUp() {
	start := s.selectedIndex
	for {
		if s.selectedIndex > 0 {
			s.selectedIndex--
		} else if s.wrapSelection {
			s.selectedIndex = len(s.items) - 1
		} else {
			s.selectedIndex = start
			return
		}
		if !s.items[s.selectedIndex].Disabled {
			break
		}
		if s.selectedIndex == start {
			return
		}
	}
	s.ensureVisible()
	if s.OnSelectionChanged != nil {
		s.OnSelectionChanged(s.selectedIndex, s.items[s.selectedIndex])
	}
	s.RequestRender()
}

// MoveDown selects the next enabled item.
func (s *Select) MoveDown() {
	start := s.selectedIndex
	for {
		if s.selectedIndex < len(s.items)-1 {
			s.selectedIndex++
		} else if s.wrapSelection {
			s.selectedIndex = 0
		} else {
			s.selectedIndex = start
			return
		}
		if !s.items[s.selectedIndex].Disabled {
			break
		}
		if s.selectedIndex == start {
			return
		}
	}
	s.ensureVisible()
	if s.OnSelectionChanged != nil {
		s.OnSelectionChanged(s.selectedIndex, s.items[s.selectedIndex])
	}
	s.RequestRender()
}

func (s *Select) visibleOptionCount() int {
	h := s.Height()
	if s.showDescription {
		return h / 2
	}
	return h
}

func (s *Select) ensureVisible() {
	visible := s.visibleOptionCount()
	if visible <= 0 {
		return
	}

	// Keep contiguous disabled headings immediately above the selection visible.
	contextStart := s.selectedIndex
	for contextStart > 0 && s.items[contextStart-1].Disabled {
		contextStart--
	}

	if contextStart < s.scrollOffset {
		s.scrollOffset = contextStart
	}
	if s.selectedIndex >= s.scrollOffset+visible {
		s.scrollOffset = s.selectedIndex - visible + 1
	}
}

// Select options.

// SelectWithBackgroundColor sets the background color.
func SelectWithBackgroundColor(c style.Color) SelectOption {
	return func(s *Select) { s.backgroundColor = c }
}

// SelectWithTextColor sets the text color.
func SelectWithTextColor(c style.Color) SelectOption {
	return func(s *Select) { s.textColor = c }
}

// SelectWithFocusedBackgroundColor sets the focused background color.
func SelectWithFocusedBackgroundColor(c style.Color) SelectOption {
	return func(s *Select) { s.focusedBackgroundColor = c }
}

// SelectWithFocusedTextColor sets the focused text color.
func SelectWithFocusedTextColor(c style.Color) SelectOption {
	return func(s *Select) { s.focusedTextColor = c }
}

// SelectWithSelectedBackgroundColor sets the selected background color.
func SelectWithSelectedBackgroundColor(c style.Color) SelectOption {
	return func(s *Select) { s.selectedBackgroundColor = c }
}

// SelectWithSelectedTextColor sets the selected text color.
func SelectWithSelectedTextColor(c style.Color) SelectOption {
	return func(s *Select) { s.selectedTextColor = c }
}

// SelectWithDescriptionColor sets the description color.
func SelectWithDescriptionColor(c style.Color) SelectOption {
	return func(s *Select) { s.descriptionColor = c }
}

// SelectWithHintColor colors the hint after each name with c on every row,
// the selected one included. Without it, a hint takes the description color,
// and the selected row's text color on that row.
func SelectWithHintColor(c style.Color) SelectOption {
	return func(s *Select) { s.hintColor = &c }
}

// SelectWithShowScrollIndicator controls whether overflow arrows are shown.
func SelectWithShowScrollIndicator(v bool) SelectOption {
	return func(s *Select) { s.showScrollIndicator = v }
}

// SelectWithWrapSelection controls whether movement wraps at the list boundaries.
func SelectWithWrapSelection(v bool) SelectOption {
	return func(s *Select) { s.wrapSelection = v }
}

// SelectWithShowDescription controls whether descriptions are displayed.
func SelectWithShowDescription(v bool) SelectOption {
	return func(s *Select) { s.showDescription = v }
}

// SelectWithInlineDescription renders description on the same row as the name.
func SelectWithInlineDescription(v bool) SelectOption {
	return func(s *Select) { s.inlineDescription = v }
}

// SelectWithNameWidth sets the name-column width used by inline descriptions.
func SelectWithNameWidth(w int) SelectOption {
	return func(s *Select) { s.nameWidth = w }
}

// SetNameWidth sets the name-column width used by inline descriptions.
func (s *Select) SetNameWidth(w int) {
	s.nameWidth = w
	s.RequestRender()
}

// arrowColumnWidth is the width of the column SelectWithArrowColumn reserves:
// the arrow and a space.
const arrowColumnWidth = 2

// SelectWithArrowColumn reserves a column before every item's name for the
// selection arrow, so that the arrow never covers the name. Hints and inline
// descriptions move with the names.
func SelectWithArrowColumn(b bool) SelectOption {
	return func(s *Select) { s.arrowColumn = b }
}

// SelectWithItemPaddingX insets the selected row background by px cells on both sides.
func SelectWithItemPaddingX(px int) SelectOption {
	return func(s *Select) { s.itemPaddingX = px }
}

// SelectWithFocusedBold uses bold text instead of a background color for the focused item.
func SelectWithFocusedBold(b bool) SelectOption {
	return func(s *Select) { s.focusedUseBold = b }
}

// SelectWithSelectionArrow controls whether the selected row starts with an arrow.
func SelectWithSelectionArrow(b bool) SelectOption {
	return func(s *Select) { s.showSelectionArrow = b }
}

// SelectWithOnSelectionChanged sets the callback invoked after selection changes.
func SelectWithOnSelectionChanged(fn func(int, SelectItem)) SelectOption {
	return func(s *Select) { s.OnSelectionChanged = fn }
}

// SelectWithOnItemSelected sets the callback invoked when the user activates an item.
func SelectWithOnItemSelected(fn func(int, SelectItem)) SelectOption {
	return func(s *Select) { s.OnItemSelected = fn }
}

// SelectWithID sets the widget ID.
func SelectWithID(id string) SelectOption {
	return func(s *Select) { s.SetID(id) }
}

// SelectWithWidth sets the widget width.
func SelectWithWidth(w layout.Scalar) SelectOption {
	return func(s *Select) { s.LayoutNode().Width = w }
}

// SelectWithHeight sets the widget height.
func SelectWithHeight(h layout.Scalar) SelectOption {
	return func(s *Select) { s.LayoutNode().Height = h }
}

// SelectWithFlexGrow sets the flex-grow factor.
func SelectWithFlexGrow(v int) SelectOption {
	return func(s *Select) { s.LayoutNode().FlexGrow = v }
}

// SetHeight sets the select widget height dynamically after construction.
func (s *Select) SetHeight(h layout.Scalar) {
	s.LayoutNode().Height = h
	s.LayoutNode().MarkDirty()
	s.RequestRender()
}
