package widget

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// ListItem contains the displayed text and caller-defined value for one list row.
type ListItem struct {
	// Text is the row's displayed text.
	Text string
	// Value is caller-defined data for the row.
	Value any
}

// List is a keyboard-navigable, scrollable list with one selected item.
type List struct {
	*render.BaseRenderable

	items         []ListItem
	selectedIndex int
	scrollOffset  int

	textColor       style.Color
	selectedFg      style.Color
	selectedBg      style.Color
	backgroundColor style.Color

	showBullet    bool
	bulletChar    rune
	wrapSelection bool

	// OnItemSelected is called when the user activates the selected item.
	OnItemSelected func(index int, item ListItem)
}

// ListOption configures a List.
type ListOption func(*List)

// NewList returns a focusable List containing items.
func NewList(items []ListItem, opts ...ListOption) *List {
	l := &List{
		BaseRenderable:  render.NewBaseRenderable(render.WithFocusable(true)),
		items:           items,
		textColor:       style.ColorWhite,
		selectedFg:      style.ColorWhite,
		selectedBg:      style.ColorFromInts(51, 153, 255, 255),
		backgroundColor: style.ColorTransparent,
		showBullet:      true,
		bulletChar:      symbol.BulletFilled,
		wrapSelection:   true,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// SelectedIndex returns the zero-based index of the selected item.
func (l *List) SelectedIndex() int { return l.selectedIndex }

// SetItems replaces the list items.
func (l *List) SetItems(items []ListItem) {
	l.items = items
	if l.selectedIndex >= len(items) {
		l.selectedIndex = max(len(items)-1, 0)
	}
	l.RequestRender()
}

// RenderSelf draws the list.
func (l *List) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	w := l.Width()
	h := l.Height()
	if w <= 0 || h <= 0 || len(l.items) == 0 {
		return
	}

	x := l.X()
	y := l.Y()

	if !l.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, l.backgroundColor)
	}

	for i := l.scrollOffset; i < len(l.items) && i-l.scrollOffset < h; i++ {
		item := l.items[i]
		screenY := y + (i - l.scrollOffset)
		isSelected := i == l.selectedIndex

		fg := l.textColor
		bg := l.backgroundColor

		if isSelected && l.Focused() {
			fg = l.selectedFg
			bg = l.selectedBg
		}

		if !bg.IsTransparent() {
			buf.FillRect(x, screenY, w, 1, bg)
		}

		prefix := ""
		if l.showBullet {
			if isSelected {
				prefix = string(l.bulletChar) + " "
			} else {
				prefix = "  "
			}
		}

		text := truncateToWidth(prefix+item.Text, w)
		buf.DrawText(text, x, screenY, fg)
	}
}

// HandleKeyPress moves or activates the selection for a supported key event.
func (l *List) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease || len(l.items) == 0 {
		return false
	}

	handled := true
	switch {
	case e.Matches(input.KeyUp, 0) || e.Matches("k", 0):
		if l.selectedIndex > 0 {
			l.selectedIndex--
		} else if l.wrapSelection {
			l.selectedIndex = len(l.items) - 1
		}
		l.ensureVisible()
		l.RequestRender()

	case e.Matches(input.KeyDown, 0) || e.Matches("j", 0):
		if l.selectedIndex < len(l.items)-1 {
			l.selectedIndex++
		} else if l.wrapSelection {
			l.selectedIndex = 0
		}
		l.ensureVisible()
		l.RequestRender()

	case e.Matches(input.KeyReturn, 0) || e.Matches(input.KeySpace, 0):
		if l.OnItemSelected != nil {
			l.OnItemSelected(l.selectedIndex, l.items[l.selectedIndex])
		}

	default:
		handled = false
	}

	if handled {
		e.StopPropagation()
	}
	return handled
}

func (l *List) ensureVisible() {
	h := l.Height()
	if l.selectedIndex < l.scrollOffset {
		l.scrollOffset = l.selectedIndex
	}
	if l.selectedIndex >= l.scrollOffset+h {
		l.scrollOffset = l.selectedIndex - h + 1
	}
}

// List options.

// ListWithFg sets the text color.
func ListWithFg(c style.Color) ListOption {
	return func(l *List) { l.textColor = c }
}

// ListWithSelectedBg sets the selected row's background color.
func ListWithSelectedBg(c style.Color) ListOption {
	return func(l *List) { l.selectedBg = c }
}

// ListWithShowBullet controls whether rows reserve space for the selection bullet.
func ListWithShowBullet(v bool) ListOption {
	return func(l *List) { l.showBullet = v }
}

// ListWithOnItemSelected sets the callback invoked when a user activates an item.
func ListWithOnItemSelected(fn func(int, ListItem)) ListOption {
	return func(l *List) { l.OnItemSelected = fn }
}

// ListWithID sets the widget ID.
func ListWithID(id string) ListOption {
	return func(l *List) { l.SetID(id) }
}

// ListWithWidth sets the widget width.
func ListWithWidth(s layout.Scalar) ListOption {
	return func(l *List) { l.LayoutNode().Width = s }
}

// ListWithHeight sets the widget height.
func ListWithHeight(s layout.Scalar) ListOption {
	return func(l *List) { l.LayoutNode().Height = s }
}

// ListWithFlexGrow sets the flex-grow factor.
func ListWithFlexGrow(v int) ListOption {
	return func(l *List) { l.LayoutNode().FlexGrow = v }
}
