package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// ScrollBarOrientation identifies the axis represented by a ScrollBar.
type ScrollBarOrientation int

const (
	ScrollBarVertical ScrollBarOrientation = iota
	ScrollBarHorizontal
)

// ScrollBar is a standalone scrollbar widget that participates in the
// layout tree. It can be placed as a sibling of any scrollable content.
//
// The owner is responsible for keeping ScrollSize, ViewportSize, and
// ScrollPosition in sync with the content being scrolled.
type ScrollBar struct {
	*render.BaseRenderable

	orientation ScrollBarOrientation

	scrollSize     int // total content size (lines or columns)
	viewportSize   int // visible area size
	scrollPosition int // current scroll offset

	trackColor style.Color
	thumbColor style.Color

	// OnChange is reserved for interactive position changes. ScrollBar does not
	// currently implement pointer interaction, so it does not call this callback.
	OnChange func(position int)
}

// ScrollBarOption configures a ScrollBar.
type ScrollBarOption func(*ScrollBar)

// NewScrollBar returns a vertical ScrollBar unless an option changes its orientation.
func NewScrollBar(opts ...ScrollBarOption) *ScrollBar {
	sb := &ScrollBar{
		BaseRenderable: render.NewBaseRenderable(),
		orientation:    ScrollBarVertical,
		trackColor:     style.ColorFromInts(40, 40, 40, 255),
		thumbColor:     style.ColorFromInts(100, 100, 100, 255),
	}

	// A vertical scrollbar reserves one column by default.
	sb.LayoutNode().Width = layout.Points(1)

	for _, opt := range opts {
		opt(sb)
	}

	// A horizontal scrollbar reserves one row and leaves its width automatic.
	if sb.orientation == ScrollBarHorizontal {
		sb.LayoutNode().Width = layout.Scalar{} // auto
		sb.LayoutNode().Height = layout.Points(1)
	}

	return sb
}

// Scroll state.

// SetScrollState updates the content size, viewport size, and clamped position in one render request.
func (sb *ScrollBar) SetScrollState(scrollSize, viewportSize, scrollPosition int) {
	sb.scrollSize = scrollSize
	sb.viewportSize = viewportSize
	sb.scrollPosition = clampScrollPosition(scrollPosition, scrollSize, viewportSize)
	sb.updateVisibility()
	sb.RequestRender()
}

// ScrollSize returns the total content size.
func (sb *ScrollBar) ScrollSize() int { return sb.scrollSize }

// SetScrollSize sets the total content size.
func (sb *ScrollBar) SetScrollSize(v int) {
	if sb.scrollSize == v {
		return
	}
	sb.scrollSize = v
	sb.scrollPosition = clampScrollPosition(sb.scrollPosition, sb.scrollSize, sb.viewportSize)
	sb.updateVisibility()
	sb.RequestRender()
}

// ViewportSize returns the visible viewport size.
func (sb *ScrollBar) ViewportSize() int { return sb.viewportSize }

// SetViewportSize sets the visible viewport size.
func (sb *ScrollBar) SetViewportSize(v int) {
	if sb.viewportSize == v {
		return
	}
	sb.viewportSize = v
	sb.scrollPosition = clampScrollPosition(sb.scrollPosition, sb.scrollSize, sb.viewportSize)
	sb.updateVisibility()
	sb.RequestRender()
}

// ScrollPosition returns the current scroll offset.
func (sb *ScrollBar) ScrollPosition() int { return sb.scrollPosition }

// SetScrollPosition sets the current scroll offset, clamped to the scrollable range.
func (sb *ScrollBar) SetScrollPosition(v int) {
	v = clampScrollPosition(v, sb.scrollSize, sb.viewportSize)
	if sb.scrollPosition == v {
		return
	}
	sb.scrollPosition = v
	sb.RequestRender()
}

// ScrollBar rendering.

// RenderSelf draws the track and thumb.
func (sb *ScrollBar) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := sb.Width()
	h := sb.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := sb.X()
	y := sb.Y()

	if sb.orientation == ScrollBarVertical {
		sb.renderVertical(buf, x, y, w, h)
	} else {
		sb.renderHorizontal(buf, x, y, w, h)
	}
}

func (sb *ScrollBar) renderVertical(buf *buffer.Buffer, x, y, w, h int) {
	barX := x + w - 1

	// Paint the last column as the track.
	for row := range h {
		buf.SetCell(barX, y+row, ' ', sb.trackColor, sb.trackColor, 0)
	}

	if sb.scrollSize <= 0 || sb.viewportSize <= 0 {
		return
	}

	viewportFraction := float64(sb.viewportSize) / float64(sb.scrollSize)
	if viewportFraction >= 1.0 {
		return
	}

	thumbHeight := max(int(viewportFraction*float64(h)), 1)

	maxScroll := sb.scrollSize - sb.viewportSize
	if maxScroll <= 0 {
		return
	}
	scrollFraction := float64(sb.scrollPosition) / float64(maxScroll)
	thumbOffset := int(scrollFraction * float64(h-thumbHeight))

	for row := range thumbHeight {
		buf.SetCell(barX, y+thumbOffset+row, symbol.ScrollBarVertical.Thumb, sb.thumbColor, sb.thumbColor, 0)
	}
}

func (sb *ScrollBar) renderHorizontal(buf *buffer.Buffer, x, y, w, h int) {
	barY := y + h - 1

	// Paint the last row as the track.
	for col := range w {
		buf.SetCell(x+col, barY, ' ', sb.trackColor, sb.trackColor, 0)
	}

	if sb.scrollSize <= 0 || sb.viewportSize <= 0 {
		return
	}

	viewportFraction := float64(sb.viewportSize) / float64(sb.scrollSize)
	if viewportFraction >= 1.0 {
		return
	}

	thumbWidth := max(int(viewportFraction*float64(w)), 1)

	maxScroll := sb.scrollSize - sb.viewportSize
	if maxScroll <= 0 {
		return
	}
	scrollFraction := float64(sb.scrollPosition) / float64(maxScroll)
	thumbOffset := int(scrollFraction * float64(w-thumbWidth))

	for col := range thumbWidth {
		buf.SetCell(x+thumbOffset+col, barY, symbol.ScrollBarHorizontal.Thumb, sb.thumbColor, sb.thumbColor, 0)
	}
}

func (sb *ScrollBar) updateVisibility() {
	// Hide the bar when the entire content fits in the viewport.
	sb.SetVisible(sb.scrollSize > sb.viewportSize)
}

func clampScrollPosition(pos, scrollSize, viewportSize int) int {
	max := scrollSize - viewportSize
	if max < 0 {
		max = 0
	}
	if pos < 0 {
		return 0
	}
	if pos > max {
		return max
	}
	return pos
}

// ScrollBar options.

// ScrollBarWithOrientation sets the scrollbar orientation.
func ScrollBarWithOrientation(o ScrollBarOrientation) ScrollBarOption {
	return func(sb *ScrollBar) { sb.orientation = o }
}

// ScrollBarWithTrackColor sets the track background color.
func ScrollBarWithTrackColor(c style.Color) ScrollBarOption {
	return func(sb *ScrollBar) { sb.trackColor = c }
}

// ScrollBarWithThumbColor sets the thumb color.
func ScrollBarWithThumbColor(c style.Color) ScrollBarOption {
	return func(sb *ScrollBar) { sb.thumbColor = c }
}

// ScrollBarWithID sets the widget ID.
func ScrollBarWithID(id string) ScrollBarOption {
	return func(sb *ScrollBar) { sb.SetID(id) }
}

// ScrollBarWithOnChange sets the callback reserved for interactive position changes.
func ScrollBarWithOnChange(fn func(int)) ScrollBarOption {
	return func(sb *ScrollBar) { sb.OnChange = fn }
}

// BindScrollBar keeps scrollBar synchronized with scrollBox's vertical state.
// It installs a hook on scrollBox that updates scrollBar after every render.
func BindScrollBar(scrollBox *ScrollBox, scrollBar *ScrollBar) {
	// Reserve room for the one-column scrollbar in a row layout.
	scrollBox.LayoutNode().FlexBasis = layout.Points(0)

	scrollBox.RenderAfterHook = func(_ *buffer.Buffer, _ float64) {
		scrollBar.SetScrollState(
			scrollBox.ContentHeight(),
			scrollBox.ViewportHeight(),
			scrollBox.ScrollY(),
		)
	}
}
