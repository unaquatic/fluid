package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// Verify that ScrollBox implements render.Renderable.
var _ render.Renderable = (*ScrollBox)(nil)

// ScrollAnchor records the viewport top relative to one child.
// This relative position remains stable when earlier children change size.
type ScrollAnchor struct {
	ChildIndex int  // Index of the child intersecting the viewport top.
	Offset     int  // Cells from the child's top to the viewport top.
	Valid      bool // Whether ChildIndex and Offset contain a computed anchor.
}

// AutoScrollEdge identifies the viewport edge followed as dimensions change.
type AutoScrollEdge int

const (
	AutoScrollNone   AutoScrollEdge = iota
	AutoScrollTop                   // Keeps scrollY at zero.
	AutoScrollBottom                // Keeps scrollY at its maximum.
	AutoScrollLeft                  // Keeps scrollX at zero.
	AutoScrollRight                 // Keeps scrollX at its maximum.
)

// ScrollBox is a Box that clips and translates child content for scrolling.
// Place a separate ScrollBar beside it when a visible scroll indicator is needed.
type ScrollBox struct {
	*Box

	scrollY       int
	scrollX       int
	contentHeight int
	contentWidth  int

	// Auto-scroll follows one edge until the user moves away from it.
	autoScroll   AutoScrollEdge
	userScrolled bool

	// Anchor-based scrolling preserves a child-relative viewport position.
	anchorScroll bool
	anchor       ScrollAnchor

	// stickyGap separates consecutive sticky children in cells.
	stickyGap int
}

// ScrollBoxOption configures a ScrollBox during construction.
type ScrollBoxOption func(*ScrollBox)

// NewScrollBox returns a clipping, scrollable Box with optional configuration.
func NewScrollBox(opts ...ScrollBoxOption) *ScrollBox {
	sb := &ScrollBox{
		Box:        NewBox(BoxWithOverflow(layout.OverflowHidden)),
		autoScroll: AutoScrollNone,
	}

	// OverflowScroll enables clipping and translated child rendering.
	sb.LayoutNode().Overflow = layout.OverflowScroll

	for _, opt := range opts {
		opt(sb)
	}

	// Options can add a border after NewBox computed its initial padding.
	sb.applyBorderPadding()

	// Initialize the configured sticky edge before content is measured.
	if sb.autoScroll != AutoScrollNone {
		sb.applyAutoScroll(sb.autoScroll)
	}

	// Let scroll events bubble at an edge so an enclosing container can move.
	sb.OnMouseScrollHandler = func(e *input.MouseEvent) {
		if e.Scroll != nil {
			oldY := sb.scrollY
			oldX := sb.scrollX
			sb.handleScroll(e.Scroll.DeltaX, e.Scroll.DeltaY, e.Modifiers.Ctrl)
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

// SetScrollY sets a clamped vertical offset and updates auto-scroll state.
func (sb *ScrollBox) SetScrollY(y int) {
	sb.scrollY = sb.clampScrollY(y)
	if !sb.isAtAutoScrollEdge() && sb.maxScrollY() > 0 {
		sb.userScrolled = true
	}
	if sb.anchorScroll {
		sb.updateAnchor()
	}
	sb.updateAutoScrollState()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// SetScrollX sets a clamped horizontal offset and updates auto-scroll state.
func (sb *ScrollBox) SetScrollX(x int) {
	sb.scrollX = sb.clampScrollX(x)
	if !sb.isAtAutoScrollEdge() && sb.maxScrollX() > 0 {
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

// ScrollBy adds deltaX and deltaY to the current offsets.
func (sb *ScrollBox) ScrollBy(deltaX, deltaY int) {
	if deltaY != 0 {
		sb.SetScrollY(sb.scrollY + deltaY)
	}
	if deltaX != 0 {
		sb.SetScrollX(sb.scrollX + deltaX)
	}
}

// ScrollTo sets absolute horizontal and vertical offsets.
func (sb *ScrollBox) ScrollTo(x, y int) {
	sb.SetScrollY(y)
	sb.SetScrollX(x)
}

// IsAtBottom reports whether the vertical offset is at its maximum.
func (sb *ScrollBox) IsAtBottom() bool {
	return sb.scrollY >= sb.maxScrollY()
}

// IsAtTop reports whether the vertical offset is zero.
func (sb *ScrollBox) IsAtTop() bool {
	return sb.scrollY <= 0
}

// IsAtLeft reports whether the horizontal offset is zero.
func (sb *ScrollBox) IsAtLeft() bool {
	return sb.scrollX <= 0
}

// IsAtRight reports whether the horizontal offset is at its maximum.
func (sb *ScrollBox) IsAtRight() bool {
	return sb.scrollX >= sb.maxScrollX()
}

// AutoScroll returns the configured auto-scroll edge.
func (sb *ScrollBox) AutoScroll() AutoScrollEdge { return sb.autoScroll }

// SetAutoScroll selects the edge followed as content changes; AutoScrollNone disables it.
func (sb *ScrollBox) SetAutoScroll(edge AutoScrollEdge) {
	sb.autoScroll = edge
	sb.updateAutoScrollState()
}

// UserScrolled reports whether scrolling has moved away from the configured auto-scroll edge.
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

// SetContentHeight sets the child-content height used for scrolling and edge following.
func (sb *ScrollBox) SetContentHeight(h int) {
	sb.contentHeight = h
	sb.recalculateAutoScroll()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// SetContentWidth sets the child-content width used for scrolling and edge following.
func (sb *ScrollBox) SetContentWidth(w int) {
	sb.contentWidth = w
	sb.recalculateAutoScroll()
	sb.updateChildTranslation()
	sb.RequestRender()
}

// RenderSelf draws the embedded Box; a separate ScrollBar renders any indicator.
func (sb *ScrollBox) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	sb.Box.RenderSelf(buf, deltaTime)
}

// OnPostLayout recalculates content dimensions from final child layout rectangles.
// It runs after layout because newly added children do not have valid computed
// positions during the earlier update phase.
//
// Computed positions exclude scroll translation. Measuring rendered positions
// would feed the previous frame's translation back into content dimensions.
//
// A child's bottom and right margins count as content, so the space they leave
// after the last child can be scrolled into view. They resolve against the
// content box, as layout resolves them. Gone children take no space.
func (sb *ScrollBox) OnPostLayout(deltaTime float64) {
	ln := sb.LayoutNode()
	sbComputed := ln.Computed
	contentWidth, contentHeight := contentBoxSize(ln)
	maxHeight := 0
	maxWidth := 0
	for _, child := range sb.Children() {
		cn := child.LayoutNode()
		if cn.Visibility == layout.Gone {
			continue
		}
		cc := cn.Computed
		marginBottom, _ := cn.MarginBottom.Resolve(contentHeight)
		marginRight, _ := cn.MarginRight.Resolve(contentWidth)
		bottom := cc.Y - sbComputed.Y + cc.Height + marginBottom
		right := cc.X - sbComputed.X + cc.Width + marginRight
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

	// Reapply edge following even when only the viewport changed, such as after
	// sibling visibility, padding, or border changes.
	sb.recalculateAutoScroll()

	// Save the settled child-relative position for the next content-size change.
	if sb.anchorScroll && sb.userScrolled {
		sb.updateAnchor()
	}
}

// contentBoxSize returns the size of n's content box: its computed size less
// its padding, which resolves against that size.
func contentBoxSize(n *layout.LayoutNode) (width, height int) {
	w, h := n.Computed.Width, n.Computed.Height
	top, _ := n.PaddingTop.Resolve(h)
	right, _ := n.PaddingRight.Resolve(w)
	bottom, _ := n.PaddingBottom.Resolve(h)
	left, _ := n.PaddingLeft.Resolve(w)
	return max(w-left-right, 0), max(h-top-bottom, 0)
}

// Keyboard navigation.

// HandleKeyPress processes keyboard input for scroll navigation.
//
// The default bindings are:
//
//	up / k       scroll up one fifth of the viewport
//	down / j     scroll down one fifth of the viewport
//	left / h     scroll left one fifth of the viewport
//	right / l    scroll right one fifth of the viewport
//	pageup       scroll up half of the viewport
//	pagedown     scroll down half of the viewport
//	home         scroll to the top-left
//	end          scroll to the bottom-right
func (sb *ScrollBox) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease {
		return false
	}

	vpH := sb.ViewportHeight()
	vpW := sb.ViewportWidth()
	handled := true

	switch {
	// Vertical movement uses one fifth of the viewport.
	case e.Matches(input.KeyUp) || e.Matches("k"):
		sb.SetScrollY(sb.scrollY - maxInt(1, vpH/5))

	case e.Matches(input.KeyDown) || e.Matches("j"):
		sb.SetScrollY(sb.scrollY + maxInt(1, vpH/5))

	// Horizontal movement uses one fifth of the viewport.
	case e.Matches(input.KeyLeft) || e.Matches("h"):
		sb.SetScrollX(sb.scrollX - maxInt(1, vpW/5))

	case e.Matches(input.KeyRight) || e.Matches("l"):
		sb.SetScrollX(sb.scrollX + maxInt(1, vpW/5))

	// Page movement uses half of the viewport.
	case e.Matches(input.KeyPageUp):
		sb.SetScrollY(sb.scrollY - maxInt(1, vpH/2))

	case e.Matches(input.KeyPageDown):
		sb.SetScrollY(sb.scrollY + maxInt(1, vpH/2))

	// Home moves to the origin on both axes.
	case e.Matches(input.KeyHome):
		sb.SetScrollY(0)
		sb.SetScrollX(0)

	// End moves to the maximum offset on both axes.
	case e.Matches(input.KeyEnd):
		sb.SetScrollY(sb.maxScrollY())
		sb.SetScrollX(sb.maxScrollX())

	default:
		handled = false
	}

	if handled {
		// The setters clear userScrolled when navigation returns to the followed edge.
		e.StopPropagation()
	}
	return handled
}

// Auto-scroll state.

// updateAutoScrollState re-enables edge following after returning to its edge.
func (sb *ScrollBox) updateAutoScrollState() {
	if sb.autoScroll == AutoScrollNone {
		return
	}
	if sb.isAtAutoScrollEdge() {
		sb.userScrolled = false
	}
}

// applyAutoScroll moves directly to edge without marking a user scroll or changing the anchor.
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

// isAtAutoScrollEdge reports whether the selected auto-scroll edge is visible.
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

// recalculateAutoScroll reapplies edge or anchor positioning after dimensions change.
func (sb *ScrollBox) recalculateAutoScroll() {
	if sb.autoScroll == AutoScrollNone {
		return
	}

	// Continue following the configured edge until the user scrolls away.
	if !sb.userScrolled {
		sb.applyAutoScroll(sb.autoScroll)
		return
	}

	// Away from the edge, restore the saved child-relative vertical position.
	if sb.anchorScroll && sb.userScrolled && sb.anchor.Valid {
		newY := sb.scrollYFromAnchor()
		sb.scrollY = sb.clampScrollY(newY)
		sb.updateAnchor()
		sb.updateChildTranslation()
		return
	}

	// Without an anchor, retain the already clamped absolute offset.
	sb.updateChildTranslation()
}

// Anchor-based scrolling.

// Anchor returns the current child-relative scroll anchor.
// Its coordinates are meaningful only when Valid is true.
func (sb *ScrollBox) Anchor() ScrollAnchor { return sb.anchor }

// updateAnchor records the first child intersecting the viewport top and its offset.
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
	// If the offset is past all children, retain a position relative to the last child.
	last := children[len(children)-1]
	lc := last.LayoutNode().Computed
	sb.anchor = ScrollAnchor{
		ChildIndex: len(children) - 1,
		Offset:     sb.scrollY - (lc.Y - sbY),
		Valid:      true,
	}
}

// scrollYFromAnchor resolves the saved child-relative anchor after layout changes.
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

// Scroll mechanics.

func (sb *ScrollBox) handleScroll(deltaX, deltaY int, ctrl bool) {
	mult := 1
	if ctrl {
		mult = 4
	}
	if deltaY != 0 {
		sb.SetScrollY(sb.scrollY + deltaY*mult)
	}
	if deltaX != 0 {
		sb.SetScrollX(sb.scrollX + deltaX*mult)
	}
	// The setters update userScrolled in both directions, including re-engagement.
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

// computeStickyOffsets pins sticky children to viewport edges.
// The next sticky child on the same edge pushes the current child away.
func (sb *ScrollBox) computeStickyOffsets() {
	children := sb.Children()
	if len(children) == 0 {
		return
	}

	sbComputed := sb.LayoutNode().Computed

	type stickyInfo struct {
		index int
		pos   int // Position on the sticky axis relative to ScrollBox.
		size  int // Size on the sticky axis.
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

	// Top offsets are positive and push children down to the viewport edge.
	for si, s := range top {
		raw := max(sb.scrollY-s.pos, 0)
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

	// Bottom offsets are negative and pull children up to the viewport edge.
	for si, s := range bottom {
		raw := min((sb.scrollY+vpH)-(s.pos+s.size),
			// The child is already above the viewport bottom.
			0)
		// A previous bottom-sticky child limits upward movement.
		limit := -(s.pos) // Do not move above the content origin.
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

	// Left offsets follow the same positive-direction rule as top offsets.
	for si, s := range left {
		raw := max(sb.scrollX-s.pos, 0)
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

	// Right offsets follow the same negative-direction rule as bottom offsets.
	for si, s := range right {
		raw := min((sb.scrollX+vpW)-(s.pos+s.size), 0)
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

// StickyGap returns the gap in cells between consecutive sticky children.
func (sb *ScrollBox) StickyGap() int { return sb.stickyGap }

// SetStickyGap sets the gap in cells between consecutive sticky children.
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

// ScrollBox options.

// ScrollBoxWithAutoScroll sets the edge followed when content or viewport size changes.
// AutoScrollNone disables edge following.
func ScrollBoxWithAutoScroll(edge AutoScrollEdge) ScrollBoxOption {
	return func(sb *ScrollBox) {
		sb.autoScroll = edge
	}
}

// ScrollBoxWithAnchorScroll controls child-relative vertical positioning.
// When enabled, content changes above the viewport do not move the visible child.
// Edge following still takes precedence until the user scrolls away.
func ScrollBoxWithAnchorScroll(v bool) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.anchorScroll = v }
}

// ScrollBoxWithStickyGap sets the gap in cells between consecutive sticky children.
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

// ScrollBoxWithFocusable controls whether the scrollbox can receive keyboard navigation.
func ScrollBoxWithFocusable(v bool) ScrollBoxOption {
	return func(sb *ScrollBox) { sb.SetFocusable(v) }
}

// ScrollBoxWithChildren adds non-nil child renderables during construction.
//
//	widget.NewScrollBox(
//	    widget.ScrollBoxWithAutoScroll(widget.AutoScrollBottom),
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
