package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// compile-time check: ScrollBox must satisfy Renderable.
var _ render.Renderable = (*ScrollBox)(nil)

// ScrollAnchor identifies a scroll position relative to a child element.
// Instead of storing an absolute pixel offset, it tracks which child is at
// the viewport top and the pixel distance into that child. This keeps the
// viewport stable when content above the visible area changes size (e.g.
// during streaming or dynamic content insertion).
type ScrollAnchor struct {
	ChildIndex int  // index of the child at the viewport top
	Offset     int  // pixels from the top of that child to the viewport top
	Valid      bool // false when no anchor has been computed yet
}

// AutoScrollEdge identifies which edge the scroll should stick to.
type AutoScrollEdge int

const (
	AutoScrollNone   AutoScrollEdge = iota
	AutoScrollTop                   // stick to top edge (scrollY = 0)
	AutoScrollBottom                // stick to bottom edge (scrollY = max)
	AutoScrollLeft                  // stick to left edge (scrollX = 0)
	AutoScrollRight                 // stick to right edge (scrollX = max)
)

// ScrollBox is a scrollable container. Place a ScrollBar widget as a
// sibling for a visual scrollbar indicator.
// It embeds Box for border/background support and adds scrolling behavior.
type ScrollBox struct {
	*Box

	scrollY       int
	scrollX       int
	contentHeight int
	contentWidth  int

	// Auto-scroll state.
	autoScroll   AutoScrollEdge
	userScrolled bool

	// Anchor-based scroll position tracking.
	anchorScroll bool         // when true, track scroll position as (child + offset)
	anchor       ScrollAnchor // last computed anchor

	// Sticky positioning.
	stickyGap int // pixels between consecutive sticky children (default 0)
}

// ScrollBoxOption configures a ScrollBox during construction.
type ScrollBoxOption func(*ScrollBox)

// NewScrollBox creates a ScrollBox with optional configuration.
func NewScrollBox(opts ...ScrollBoxOption) *ScrollBox {
	sb := &ScrollBox{
		Box:        NewBox(BoxWithOverflow(layout.OverflowHidden)),
		autoScroll: AutoScrollNone,
	}

	// Set overflow: scroll on the layout node for scissor clipping
	sb.LayoutNode().Overflow = layout.OverflowScroll

	for _, opt := range opts {
		opt(sb)
	}

	// Re-apply border padding after ScrollBox opts (which may set a
	// border via ScrollBoxWithBorderConfig after NewBox already ran).
	sb.applyBorderPadding()

	// If an auto-scroll edge is configured, apply the initial position.
	if sb.autoScroll != AutoScrollNone {
		sb.applyAutoScroll(sb.autoScroll)
	}

	// Set up mouse scroll handler.
	// Only stop propagation if the scroll actually moved; when at a
	// boundary let the event bubble to an enclosing scroll container.
	sb.OnMouseScrollHandler = func(e *input.MouseEvent) {
		if e.Scroll != nil {
			oldY := sb.scrollY
			oldX := sb.scrollX
			sb.handleScroll(e.Scroll.DeltaX, e.Scroll.DeltaY)
			if sb.scrollY != oldY || sb.scrollX != oldX {
				e.StopPropagation()
			}
		}
	}

	return sb
}

// ScrollY returns the vertical scroll offset.
func (sb *ScrollBox) ScrollY() int { return sb.scrollY }

// ScrollX returns the horizontal scroll offset.
func (sb *ScrollBox) ScrollX() int { return sb.scrollX }

// SetScrollY sets the vertical scroll offset.
// Marks userScrolled unless we end up at the auto-scroll edge.
func (sb *ScrollBox) SetScrollY(y int) {
	sb.scrollY = sb.clampScrollY(y)
	if !sb.isAtAutoScrollEdge() && sb.maxScrollY() > 1 {
		sb.userScrolled = true
	}
	if sb.anchorScroll {
		sb.updateAnchor()
	}
	sb.updateAutoScrollState()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// SetScrollX sets the horizontal scroll offset.
func (sb *ScrollBox) SetScrollX(x int) {
	sb.scrollX = sb.clampScrollX(x)
	if !sb.isAtAutoScrollEdge() && sb.maxScrollX() > 1 {
		sb.userScrolled = true
	}
	sb.updateAutoScrollState()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// ScrollToBottom scrolls to the bottom of the content.
func (sb *ScrollBox) ScrollToBottom() {
	sb.SetScrollY(sb.maxScrollY())
}

// ScrollToTop scrolls to the top.
func (sb *ScrollBox) ScrollToTop() {
	sb.SetScrollY(0)
}

// ScrollToLeft scrolls to the left edge.
func (sb *ScrollBox) ScrollToLeft() {
	sb.SetScrollX(0)
}

// ScrollToRight scrolls to the right edge.
func (sb *ScrollBox) ScrollToRight() {
	sb.SetScrollX(sb.maxScrollX())
}

// ScrollBy scrolls by delta in both axes.
func (sb *ScrollBox) ScrollBy(deltaX, deltaY int) {
	if deltaY != 0 {
		sb.SetScrollY(sb.scrollY + deltaY)
	}
	if deltaX != 0 {
		sb.SetScrollX(sb.scrollX + deltaX)
	}
}

// ScrollTo scrolls to an absolute position.
func (sb *ScrollBox) ScrollTo(x, y int) {
	sb.SetScrollY(y)
	sb.SetScrollX(x)
}

// IsAtBottom returns true if scrolled to the bottom.
func (sb *ScrollBox) IsAtBottom() bool {
	return sb.scrollY >= sb.maxScrollY()
}

// IsAtTop returns true if scrolled to the top.
func (sb *ScrollBox) IsAtTop() bool {
	return sb.scrollY <= 0
}

// IsAtLeft returns true if scrolled to the left edge.
func (sb *ScrollBox) IsAtLeft() bool {
	return sb.scrollX <= 0
}

// IsAtRight returns true if scrolled to the right edge.
func (sb *ScrollBox) IsAtRight() bool {
	return sb.scrollX >= sb.maxScrollX()
}

// AutoScroll returns the configured auto-scroll edge.
func (sb *ScrollBox) AutoScroll() AutoScrollEdge { return sb.autoScroll }

// SetAutoScroll sets the auto-scroll edge. Use AutoScrollNone to disable.
func (sb *ScrollBox) SetAutoScroll(edge AutoScrollEdge) {
	sb.autoScroll = edge
	sb.updateAutoScrollState()
}

// UserScrolled returns true if the user has scrolled away from the auto-scroll edge.
func (sb *ScrollBox) UserScrolled() bool { return sb.userScrolled }

// ContentHeight returns the total content height.
func (sb *ScrollBox) ContentHeight() int { return sb.contentHeight }

// ContentWidth returns the total content width.
func (sb *ScrollBox) ContentWidth() int { return sb.contentWidth }

// ViewportHeight returns the visible viewport height.
func (sb *ScrollBox) ViewportHeight() int {
	return sb.Box.InnerRect().Height
}

// ViewportWidth returns the visible viewport width.
func (sb *ScrollBox) ViewportWidth() int {
	return sb.Box.InnerRect().Width
}

// SetContentHeight sets the total content height for scrollbar calculations.
func (sb *ScrollBox) SetContentHeight(h int) {
	sb.contentHeight = h
	sb.recalculateAutoScroll()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// SetContentWidth sets the total content width.
func (sb *ScrollBox) SetContentWidth(w int) {
	sb.contentWidth = w
	sb.recalculateAutoScroll()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// RenderSelf draws the scrollbox background and style.
// Scrollbars are rendered by child ScrollBar widgets.
func (sb *ScrollBox) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	sb.Box.RenderSelf(buf, deltaTime)
}

// OnPostLayout recalculates content dimensions from children after the
// layout engine has resolved all Computed positions and sizes. This runs
// in the post-layout phase of the render pipeline, so Computed values are
// fresh — unlike OnUpdate which runs before CalculateLayout and would see
// stale zeros for newly-added children, causing a one-frame lag in
// contentHeight that led to broken auto-scroll (sudden jump to bottom
// on first user scroll).
//
// We use Computed values rather than the rendered positions (child.Y()/X())
// which include scroll translation. Using rendered positions causes a
// 1-frame oscillation: the stale translated positions from the previous
// frame combined with the current scrollY inflate contentHeight by 1,
// which temporarily raises maxScrollY and produces a visible bounce at the
// scroll edge.
func (sb *ScrollBox) OnPostLayout(deltaTime float64) {
	sbComputed := sb.LayoutNode().Computed
	maxHeight := 0
	maxWidth := 0
	for _, child := range sb.Children() {
		cc := child.LayoutNode().Computed
		bottom := cc.Y - sbComputed.Y + cc.Height
		right := cc.X - sbComputed.X + cc.Width
		if bottom > maxHeight {
			maxHeight = bottom
		}
		if right > maxWidth {
			maxWidth = right
		}
	}

	if maxHeight != sb.contentHeight {
		sb.SetContentHeight(maxHeight)
	}
	if maxWidth != sb.contentWidth {
		sb.contentWidth = maxWidth
	}

	// Keep the anchor fresh after layout settles so the next content-size
	// change can restore from up-to-date child positions.
	if sb.anchorScroll && sb.userScrolled {
		sb.updateAnchor()
	}
}

// ---------------------------------------------------------------------------
// Keyboard Navigation
// ---------------------------------------------------------------------------

// HandleKeyPress processes keyboard input for scroll navigation.
//
// Bindings:
//
//	up / k       — scroll up 1/5 viewport
//	down / j     — scroll down 1/5 viewport
//	left / h     — scroll left 1/5 viewport
//	right / l    — scroll right 1/5 viewport
//	pageup       — scroll up 1/2 viewport
//	pagedown     — scroll down 1/2 viewport
//	home         — scroll to top-left
//	end          — scroll to bottom-right
func (sb *ScrollBox) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease {
		return false
	}

	vpH := sb.ViewportHeight()
	vpW := sb.ViewportWidth()
	handled := true

	switch {
	// Vertical: up
	case e.Matches(input.KeyUp) || e.Matches("k"):
		sb.SetScrollY(sb.scrollY - maxInt(1, vpH/5))

	// Vertical: down
	case e.Matches(input.KeyDown) || e.Matches("j"):
		sb.SetScrollY(sb.scrollY + maxInt(1, vpH/5))

	// Horizontal: left
	case e.Matches(input.KeyLeft) || e.Matches("h"):
		sb.SetScrollX(sb.scrollX - maxInt(1, vpW/5))

	// Horizontal: right
	case e.Matches(input.KeyRight) || e.Matches("l"):
		sb.SetScrollX(sb.scrollX + maxInt(1, vpW/5))

	// Page up
	case e.Matches(input.KeyPageUp):
		sb.SetScrollY(sb.scrollY - maxInt(1, vpH/2))

	// Page down
	case e.Matches(input.KeyPageDown):
		sb.SetScrollY(sb.scrollY + maxInt(1, vpH/2))

	// Home — scroll to origin
	case e.Matches(input.KeyHome):
		sb.SetScrollY(0)
		sb.SetScrollX(0)

	// End — scroll to bottom-right
	case e.Matches(input.KeyEnd):
		sb.SetScrollY(sb.maxScrollY())
		sb.SetScrollX(sb.maxScrollX())

	default:
		handled = false
	}

	if handled {
		// Note: SetScrollY/SetScrollX already set userScrolled when away
		// from the auto-scroll edge. We must not override that here — doing so
		// would prevent auto-scroll from re-engaging when the user presses
		// End or PageDown back to the bottom.
		e.StopPropagation()
	}
	return handled
}

// ---------------------------------------------------------------------------
// Internal — auto-scroll
// ---------------------------------------------------------------------------

// updateAutoScrollState clears userScrolled when the user scrolls back to the
// configured auto-scroll edge, allowing auto-scroll to re-engage.
func (sb *ScrollBox) updateAutoScrollState() {
	if sb.autoScroll == AutoScrollNone {
		return
	}
	if sb.isAtAutoScrollEdge() {
		sb.userScrolled = false
	}
}

// applyAutoScroll forces the scroll position to the given edge.
// Sets scrollY/scrollX directly (not via SetScrollY) to avoid
// triggering userScrolled or anchor updates.
func (sb *ScrollBox) applyAutoScroll(edge AutoScrollEdge) {
	switch edge {
	case AutoScrollTop:
		sb.scrollY = 0
	case AutoScrollBottom:
		sb.scrollY = sb.maxScrollY()
	case AutoScrollLeft:
		sb.scrollX = 0
	case AutoScrollRight:
		sb.scrollX = sb.maxScrollX()
	}
	sb.updateChildTranslation()
}

// isAtAutoScrollEdge returns true if the current scroll position matches
// the configured autoScroll edge.
func (sb *ScrollBox) isAtAutoScrollEdge() bool {
	if sb.autoScroll == AutoScrollNone {
		return false
	}
	switch sb.autoScroll {
	case AutoScrollTop:
		return sb.scrollY == 0
	case AutoScrollBottom:
		return sb.scrollY >= sb.maxScrollY()
	case AutoScrollLeft:
		return sb.scrollX == 0
	case AutoScrollRight:
		return sb.scrollX >= sb.maxScrollX()
	}
	return false
}

// recalculateAutoScroll recalculates auto-scroll state.
// Called when content dimensions change.
func (sb *ScrollBox) recalculateAutoScroll() {
	if sb.autoScroll == AutoScrollNone {
		return
	}

	// Primary path: user hasn't scrolled away → snap to the configured edge.
	if !sb.userScrolled {
		sb.applyAutoScroll(sb.autoScroll)
		return
	}

	// Anchor-based scroll: when the user has scrolled away from the auto-scroll
	// edge and content dimensions change, restore scrollY from the anchor
	// so the viewport stays on the same child item.
	if sb.anchorScroll && sb.userScrolled && sb.anchor.Valid {
		newY := sb.scrollYFromAnchor()
		sb.scrollY = sb.clampScrollY(newY)
		sb.updateAnchor()
		sb.updateChildTranslation()
		return
	}

	// No anchor — just keep current scrollY (already clamped by SetContentHeight).
	sb.updateChildTranslation()
}

// ---------------------------------------------------------------------------
// Internal — anchor-based scroll
// ---------------------------------------------------------------------------

// Anchor returns the current scroll anchor. The anchor is only meaningful
// when anchorScroll is enabled and the returned anchor has Valid == true.
func (sb *ScrollBox) Anchor() ScrollAnchor { return sb.anchor }

// updateAnchor recomputes the scroll anchor from the current scrollY and
// children's layout positions. It finds the first child whose bottom edge
// extends below the scroll offset and records the child index plus the
// pixel distance from that child's top to the viewport top.
func (sb *ScrollBox) updateAnchor() {
	children := sb.Children()
	if len(children) == 0 {
		sb.anchor = ScrollAnchor{}
		return
	}

	sbY := sb.LayoutNode().Computed.Y
	for i, child := range children {
		cc := child.LayoutNode().Computed
		childTop := cc.Y - sbY
		childBottom := childTop + cc.Height
		if childBottom > sb.scrollY {
			sb.anchor = ScrollAnchor{
				ChildIndex: i,
				Offset:     sb.scrollY - childTop,
				Valid:      true,
			}
			return
		}
	}
	// Past the end — anchor to last child.
	last := children[len(children)-1]
	lc := last.LayoutNode().Computed
	sb.anchor = ScrollAnchor{
		ChildIndex: len(children) - 1,
		Offset:     sb.scrollY - (lc.Y - sbY),
		Valid:      true,
	}
}

// scrollYFromAnchor converts the saved anchor back to an absolute scrollY
// using the children's current layout positions.
func (sb *ScrollBox) scrollYFromAnchor() int {
	if !sb.anchor.Valid {
		return sb.scrollY
	}
	children := sb.Children()
	idx := sb.anchor.ChildIndex
	if idx >= len(children) {
		idx = len(children) - 1
	}
	if idx < 0 {
		return 0
	}
	child := children[idx]
	cc := child.LayoutNode().Computed
	sbY := sb.LayoutNode().Computed.Y
	childTop := cc.Y - sbY
	return childTop + sb.anchor.Offset
}

// ---------------------------------------------------------------------------
// Internal — scroll mechanics
// ---------------------------------------------------------------------------

func (sb *ScrollBox) handleScroll(deltaX, deltaY int) {
	if deltaY != 0 {
		sb.SetScrollY(sb.scrollY + deltaY)
	}
	if deltaX != 0 {
		sb.SetScrollX(sb.scrollX + deltaX)
	}
	// Note: SetScrollY/SetScrollX already set userScrolled when the
	// resulting position is away from the auto-scroll edge. We must NOT
	// unconditionally set userScrolled here — doing so would prevent
	// auto-scroll from re-engaging when the user scrolls back to the
	// auto-scroll edge (e.g. scrolling back to the bottom).
}

func (sb *ScrollBox) clampScrollY(y int) int {
	if y < 0 {
		return 0
	}
	maxY := sb.maxScrollY()
	if y > maxY {
		return maxY
	}
	return y
}

func (sb *ScrollBox) clampScrollX(x int) int {
	if x < 0 {
		return 0
	}
	maxX := sb.maxScrollX()
	if x > maxX {
		return maxX
	}
	return x
}

func (sb *ScrollBox) maxScrollY() int {
	inner := sb.Box.InnerRect()
	mx := sb.contentHeight - inner.Height
	if mx < 0 {
		return 0
	}
	return mx
}

func (sb *ScrollBox) maxScrollX() int {
	inner := sb.Box.InnerRect()
	mx := sb.contentWidth - inner.Width
	if mx < 0 {
		return 0
	}
	return mx
}

func (sb *ScrollBox) updateChildTranslation() {
	sb.SetTranslateY(-sb.scrollY)
	sb.SetTranslateX(-sb.scrollX)
	sb.computeStickyOffsets()
}

// computeStickyOffsets sets per-child stickyOffsetX/Y on children with a
// Sticky position. Each direction pins the child at the corresponding
// viewport edge and is section-scoped: the next sticky child with the
// same direction pushes the current one away.
func (sb *ScrollBox) computeStickyOffsets() {
	children := sb.Children()
	if len(children) == 0 {
		return
	}

	sbComputed := sb.LayoutNode().Computed

	type stickyInfo struct {
		index int
		pos   int // position on the sticky axis relative to ScrollBox
		size  int // size on the sticky axis
	}

	var top, bottom, left, right []stickyInfo

	for i, child := range children {
		ln := child.LayoutNode()
		cc := ln.Computed
		switch ln.Sticky {
		case layout.StickyTop:
			top = append(top, stickyInfo{i, cc.Y - sbComputed.Y, cc.Height})
		case layout.StickyBottom:
			bottom = append(bottom, stickyInfo{i, cc.Y - sbComputed.Y, cc.Height})
		case layout.StickyLeft:
			left = append(left, stickyInfo{i, cc.X - sbComputed.X, cc.Width})
		case layout.StickyRight:
			right = append(right, stickyInfo{i, cc.X - sbComputed.X, cc.Width})
		default:
			child.SetStickyOffsetX(0)
			child.SetStickyOffsetY(0)
		}
	}

	vpH := sb.ViewportHeight()
	vpW := sb.ViewportWidth()

	// StickyTop: pin at viewport top when child scrolls above it.
	// Offset is positive (pushes child down).
	for si, s := range top {
		raw := sb.scrollY - s.pos
		if raw < 0 {
			raw = 0
		}
		limit := sb.contentHeight - s.pos - s.size
		if si+1 < len(top) {
			limit = top[si+1].pos - s.pos - s.size - sb.stickyGap
		}
		if limit < 0 {
			limit = 0
		}
		if raw > limit {
			raw = limit
		}
		children[s.index].SetStickyOffsetY(raw)
		children[s.index].SetStickyOffsetX(0)
	}

	// StickyBottom: pin at viewport bottom when child scrolls below it.
	// Offset is negative (pulls child up).
	for si, s := range bottom {
		raw := (sb.scrollY + vpH) - (s.pos + s.size)
		if raw > 0 {
			raw = 0 // child is above viewport bottom — no pinning
		}
		// Section limit: previous sticky-bottom child pushes this one down.
		limit := -(s.pos) // can't go above content top
		if si > 0 {
			prev := bottom[si-1]
			limit = -(s.pos - prev.pos - prev.size - sb.stickyGap)
		}
		if limit > 0 {
			limit = 0
		}
		if raw < limit {
			raw = limit
		}
		children[s.index].SetStickyOffsetY(raw)
		children[s.index].SetStickyOffsetX(0)
	}

	// StickyLeft: pin at viewport left when child scrolls past it.
	// Same as StickyTop but on the X axis.
	for si, s := range left {
		raw := sb.scrollX - s.pos
		if raw < 0 {
			raw = 0
		}
		limit := sb.contentWidth - s.pos - s.size
		if si+1 < len(left) {
			limit = left[si+1].pos - s.pos - s.size - sb.stickyGap
		}
		if limit < 0 {
			limit = 0
		}
		if raw > limit {
			raw = limit
		}
		children[s.index].SetStickyOffsetX(raw)
		children[s.index].SetStickyOffsetY(0)
	}

	// StickyRight: pin at viewport right when child scrolls past it.
	// Same as StickyBottom but on the X axis.
	for si, s := range right {
		raw := (sb.scrollX + vpW) - (s.pos + s.size)
		if raw > 0 {
			raw = 0
		}
		limit := -(s.pos)
		if si > 0 {
			prev := right[si-1]
			limit = -(s.pos - prev.pos - prev.size - sb.stickyGap)
		}
		if limit > 0 {
			limit = 0
		}
		if raw < limit {
			raw = limit
		}
		children[s.index].SetStickyOffsetX(raw)
		children[s.index].SetStickyOffsetY(0)
	}
}

// StickyGap returns the gap in pixels between consecutive sticky children.
func (sb *ScrollBox) StickyGap() int { return sb.stickyGap }

// SetStickyGap sets the gap in pixels between consecutive sticky children.
func (sb *ScrollBox) SetStickyGap(gap int) {
	sb.stickyGap = gap
	sb.computeStickyOffsets()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- ScrollBox Options ---

// ScrollBoxWithAutoScroll sets the edge the scroll follows when content
// changes. Use AutoScrollNone to disable, AutoScrollBottom to follow
// the bottom (default for chat-style scrolling), etc.
func ScrollBoxWithAutoScroll(edge AutoScrollEdge) ScrollBoxOption {
	return func(sb *ScrollBox) {
		sb.autoScroll = edge
	}
}

// ScrollBoxWithAnchorScroll enables anchor-based scroll positioning.
// When enabled, the scroll position is tracked as (child index + pixel
// offset) rather than a bare absolute pixel value. This keeps the viewport
// stable when content above the visible area changes size — for example
// during streaming or when earlier messages reflow. The anchor is only
// used when the user has scrolled away from the auto-scroll edge;
// auto-scroll behaviour is unaffected.
func ScrollBoxWithAnchorScroll(v bool) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.anchorScroll = v }
}

// ScrollBoxWithStickyGap sets the gap in pixels between consecutive
// sticky children when they push each other away.
func ScrollBoxWithStickyGap(gap int) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.stickyGap = gap }
}

// ScrollBoxWithBackgroundColor sets the background color.
func ScrollBoxWithBackgroundColor(c style.Color) ScrollBoxOption {
	return func(sb *ScrollBox) {
		sb.BackgroundColor = c
		sb.ShouldFill = true
	}
}

// ScrollBoxWithBorderConfig sets border properties.
func ScrollBoxWithBorderConfig(brd style.Border, bs style.BorderStyle) ScrollBoxOption {
	return func(sb *ScrollBox) {
		sb.Border = brd
		sb.BorderStyle = bs
	}
}

// ScrollBoxWithBorderColor sets the border color.
func ScrollBoxWithBorderColor(c style.Color) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.BorderColor = c }
}

// ScrollBoxWithID sets the widget ID.
func ScrollBoxWithID(id string) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.SetID(id) }
}

// ScrollBoxWithWidth sets the widget width.
func ScrollBoxWithWidth(s layout.Scalar) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.LayoutNode().Width = s }
}

// ScrollBoxWithHeight sets the widget height.
func ScrollBoxWithHeight(s layout.Scalar) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.LayoutNode().Height = s }
}

// ScrollBoxWithFlexDirection sets the flex direction.
func ScrollBoxWithFlexDirection(d layout.FlexDirection) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.LayoutNode().FlexDirection = d }
}

// ScrollBoxWithFlexGrow sets flex grow.
func ScrollBoxWithFlexGrow(v int) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.LayoutNode().FlexGrow = v }
}

// ScrollBoxWithFocusable makes the scrollbox focusable for keyboard navigation.
func ScrollBoxWithFocusable(v bool) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.SetFocusable(v) }
}

// ScrollBoxWithChildren adds child renderables to a ScrollBox during
// construction, enabling declarative tree building:
//
//	widget.NewScrollBox(
//	    widget.ScrollBoxWithAutoScroll(true),
//	    widget.ScrollBoxWithChildren(
//	        widget.NewText("Line 1"),
//	        widget.NewText("Line 2"),
//	    ),
//	)
func ScrollBoxWithChildren(children ...render.Renderable) ScrollBoxOption {
	return func(sb *ScrollBox) {
		for _, child := range children {
			if child != nil {
				sb.Add(child)
			}
		}
	}
}
