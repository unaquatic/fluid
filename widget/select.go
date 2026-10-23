package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// SelectOption describes one row in a Select widget.
type SelectOption struct {
	Name        string
	Hint        string // Displayed after Name using the description color.
	Description string
	Value       any
	Disabled    bool // Prevents selection, for example on a section header.
}

// Select is a keyboard-navigable option list.
type Select struct {
	*render.BaseRenderable

	options       []SelectOption
	selectedIndex int
	scrollOffset  int

	backgroundColor         style.Color
	textColor               style.Color
	focusedBackgroundColor  style.Color
	focusedTextColor        style.Color
	selectedBackgroundColor style.Color
	selectedTextColor       style.Color
	descriptionColor        style.Color

	showScrollIndicator bool
	wrapSelection       bool
	showDescription     bool
	inlineDescription   bool
	nameWidth           int    // Name-column width when inline descriptions are enabled.
	itemPaddingX        int    // Horizontal inset for the selected row background.
	focusedUseBold      bool   // Uses bold instead of a background to show the focused item.
	showSelectionArrow  bool   // Draws an arrow over the selected row's first cell.
	selectionIndicator  string // Reserved prefix for the selected option.

	// OnSelectionChanged is called after keyboard or programmatic selection changes.
	OnSelectionChanged func(index int, option SelectOption)
	// OnItemSelected is called when the user activates the selected option.
	OnItemSelected func(index int, option SelectOption)
}

// SelectOpt configures a Select during construction.
type SelectOpt func(*Select)

// NewSelect returns a focusable selector containing options.
func NewSelect(options []SelectOption, opts ...SelectOpt) *Select {
	s := &Select{
		BaseRenderable:          render.NewBaseRenderable(render.WithFocusable(true)),
		options:                 options,
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

// OptionCount returns the number of options.
func (s *Select) OptionCount() int { return len(s.options) }

// SelectedOption returns the currently selected option, or nil if none.
func (s *Select) SelectedOption() *SelectOption {
	if s.selectedIndex < 0 || s.selectedIndex >= len(s.options) {
		return nil
	}
	return &s.options[s.selectedIndex]
}

// SetSelectedIndex selects the index after clamping it to the options range.
func (s *Select) SetSelectedIndex(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.options) {
		idx = len(s.options) - 1
	}
	if idx != s.selectedIndex {
		s.selectedIndex = idx
		s.ensureVisible()
		if s.OnSelectionChanged != nil && idx >= 0 && idx < len(s.options) {
			s.OnSelectionChanged(idx, s.options[idx])
		}
		s.RequestRender()
	}
}

// SetScrollOffset sets the first option considered for rendering.
func (s *Select) SetScrollOffset(offset int) {
	s.scrollOffset = offset
}

// SetOptions replaces the options list.
func (s *Select) SetOptions(options []SelectOption) {
	s.options = options
	if s.selectedIndex >= len(options) {
		s.selectedIndex = max(len(options)-1, 0)
	}
	s.scrollOffset = 0
	s.RequestRender()
}

// Options returns the current option slice.
func (s *Select) Options() []SelectOption { return s.options }

// RenderSelf draws the select list.
func (s *Select) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := s.Width()
	h := s.Height()
	if w <= 0 || h <= 0 || len(s.options) == 0 {
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

	// Render options from the current scroll offset until the viewport is full.
	drawY := y
	for i := s.scrollOffset; i < len(s.options) && drawY < y+h; i++ {
		opt := s.options[i]
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

		// Draw option row background. itemPaddingX adjusts the selected
		// row highlight: positive insets it, negative extends it outward.
		if !bg.IsTransparent() && bg != s.backgroundColor {
			if isSelected && px != 0 {
				buf.FillRect(x+px, drawY, w-px*2, 1, bg)
			} else {
				buf.FillRect(x, drawY, w, 1, bg)
			}
		}

		// Draw name text.
		nameLen := displayWidth(opt.Name)
		nameStr := truncateToWidth(opt.Name, w)
		var nameOpts []buffer.DrawTextOption
		if attrs != 0 {
			nameOpts = append(nameOpts, buffer.WithAttrs(attrs))
		}
		buf.DrawText(nameStr, x, drawY, fg, nameOpts...)

		// Draw the arrow last so it replaces the row's leading cell.
		if isSelected && s.showSelectionArrow {
			buf.SetCell(x, drawY, '→', fg, s.backgroundColor, attrs)
		}

		// Draw hint right after the name in description color.
		if opt.Hint != "" && nameLen < w {
			hintX := x + nameLen
			hint := " " + opt.Hint
			hint = truncateToWidth(hint, w-nameLen)
			hintFg := s.descriptionColor
			if isSelected {
				hintFg = fg
			}
			buf.DrawText(hint, hintX, drawY, hintFg)
			nameLen += displayWidth(hint)
		}

		// Draw inline description after the name column.
		if s.inlineDescription && opt.Description != "" {
			descX := x + s.nameWidth
			if nameLen > s.nameWidth {
				descX = x + nameLen + 1
			}
			if descX < x+w {
				desc := truncateToWidth(opt.Description, x+w-descX)
				descFg := s.descriptionColor
				if isSelected {
					descFg = fg
				}
				buf.DrawText(desc, descX, drawY, descFg)
			}
		}
		drawY++

		// A non-inline description consumes the following viewport row.
		if s.showDescription && !s.inlineDescription && opt.Description != "" && drawY < y+h {
			desc := truncateToWidth(opt.Description, w-2)
			buf.DrawText("  "+desc, x, drawY, s.descriptionColor)
			drawY++
		}
	}

	// Mark hidden options above or below the viewport.
	if s.showScrollIndicator {
		if s.scrollOffset > 0 {
			buf.SetCell(x+w-1, y, symbol.ArrowUp, s.textColor, s.backgroundColor, 0)
		}
		lastVisible := s.scrollOffset + s.visibleOptionCount()
		if lastVisible < len(s.options) {
			buf.SetCell(x+w-1, y+h-1, symbol.ArrowDown, s.textColor, s.backgroundColor, 0)
		}
	}
}

// HandleKeyPress moves or activates the selection for a supported key event.
func (s *Select) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(s.options) == 0 {
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
		s.SetSelectedIndex(len(s.options) - 1)

	case e.Matches(input.KeyReturn) || e.Matches(input.KeySpace):
		if s.OnItemSelected != nil && s.selectedIndex >= 0 && s.selectedIndex < len(s.options) {
			s.OnItemSelected(s.selectedIndex, s.options[s.selectedIndex])
		}

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

// MoveUp selects the previous enabled option.
func (s *Select) MoveUp() {
	start := s.selectedIndex
	for {
		if s.selectedIndex > 0 {
			s.selectedIndex--
		} else if s.wrapSelection {
			s.selectedIndex = len(s.options) - 1
		} else {
			s.selectedIndex = start
			return
		}
		if !s.options[s.selectedIndex].Disabled {
			break
		}
		if s.selectedIndex == start {
			return
		}
	}
	s.ensureVisible()
	if s.OnSelectionChanged != nil {
		s.OnSelectionChanged(s.selectedIndex, s.options[s.selectedIndex])
	}
	s.RequestRender()
}

// MoveDown selects the next enabled option.
func (s *Select) MoveDown() {
	start := s.selectedIndex
	for {
		if s.selectedIndex < len(s.options)-1 {
			s.selectedIndex++
		} else if s.wrapSelection {
			s.selectedIndex = 0
		} else {
			s.selectedIndex = start
			return
		}
		if !s.options[s.selectedIndex].Disabled {
			break
		}
		if s.selectedIndex == start {
			return
		}
	}
	s.ensureVisible()
	if s.OnSelectionChanged != nil {
		s.OnSelectionChanged(s.selectedIndex, s.options[s.selectedIndex])
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
	for contextStart > 0 && s.options[contextStart-1].Disabled {
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
func SelectWithBackgroundColor(c style.Color) SelectOpt {
	return func(s *Select) { s.backgroundColor = c }
}

// SelectWithTextColor sets the text color.
func SelectWithTextColor(c style.Color) SelectOpt {
	return func(s *Select) { s.textColor = c }
}

// SelectWithFocusedBackgroundColor sets the focused background color.
func SelectWithFocusedBackgroundColor(c style.Color) SelectOpt {
	return func(s *Select) { s.focusedBackgroundColor = c }
}

// SelectWithFocusedTextColor sets the focused text color.
func SelectWithFocusedTextColor(c style.Color) SelectOpt {
	return func(s *Select) { s.focusedTextColor = c }
}

// SelectWithSelectedBackgroundColor sets the selected background color.
func SelectWithSelectedBackgroundColor(c style.Color) SelectOpt {
	return func(s *Select) { s.selectedBackgroundColor = c }
}

// SelectWithSelectedTextColor sets the selected text color.
func SelectWithSelectedTextColor(c style.Color) SelectOpt {
	return func(s *Select) { s.selectedTextColor = c }
}

// SelectWithDescriptionColor sets the description color.
func SelectWithDescriptionColor(c style.Color) SelectOpt {
	return func(s *Select) { s.descriptionColor = c }
}

// SelectWithShowScrollIndicator controls whether overflow arrows are shown.
func SelectWithShowScrollIndicator(v bool) SelectOpt {
	return func(s *Select) { s.showScrollIndicator = v }
}

// SelectWithWrapSelection controls whether movement wraps at the list boundaries.
func SelectWithWrapSelection(v bool) SelectOpt {
	return func(s *Select) { s.wrapSelection = v }
}

// SelectWithShowDescription controls whether descriptions are displayed.
func SelectWithShowDescription(v bool) SelectOpt {
	return func(s *Select) { s.showDescription = v }
}

// SelectWithInlineDescription renders description on the same row as the name.
func SelectWithInlineDescription(v bool) SelectOpt {
	return func(s *Select) { s.inlineDescription = v }
}

// SelectWithNameWidth sets the name-column width used by inline descriptions.
func SelectWithNameWidth(w int) SelectOpt {
	return func(s *Select) { s.nameWidth = w }
}

// SelectWithItemPaddingX insets the selected row background by px cells on both sides.
func SelectWithItemPaddingX(px int) SelectOpt {
	return func(s *Select) { s.itemPaddingX = px }
}

// SelectWithFocusedBold uses bold text instead of a background color for the focused item.
func SelectWithFocusedBold(b bool) SelectOpt {
	return func(s *Select) { s.focusedUseBold = b }
}

// SelectWithSelectionArrow controls whether the selected row starts with an arrow.
func SelectWithSelectionArrow(b bool) SelectOpt {
	return func(s *Select) { s.showSelectionArrow = b }
}

// SelectWithOnSelectionChanged sets the callback invoked after selection changes.
func SelectWithOnSelectionChanged(fn func(int, SelectOption)) SelectOpt {
	return func(s *Select) { s.OnSelectionChanged = fn }
}

// SelectWithOnItemSelected sets the callback invoked when the user activates an option.
func SelectWithOnItemSelected(fn func(int, SelectOption)) SelectOpt {
	return func(s *Select) { s.OnItemSelected = fn }
}

// SelectWithID sets the widget ID.
func SelectWithID(id string) SelectOpt {
	return func(s *Select) { s.SetID(id) }
}

// SelectWithWidth sets the widget width.
func SelectWithWidth(w layout.Scalar) SelectOpt {
	return func(s *Select) { s.LayoutNode().Width = w }
}

// SelectWithHeight sets the widget height.
func SelectWithHeight(h layout.Scalar) SelectOpt {
	return func(s *Select) { s.LayoutNode().Height = h }
}

// SelectWithFlexGrow sets the flex-grow factor.
func SelectWithFlexGrow(v int) SelectOpt {
	return func(s *Select) { s.LayoutNode().FlexGrow = v }
}

// SetHeight sets the select widget height dynamically after construction.
func (s *Select) SetHeight(h layout.Scalar) {
	s.LayoutNode().Height = h
	s.LayoutNode().MarkDirty()
	s.RequestRender()
}
