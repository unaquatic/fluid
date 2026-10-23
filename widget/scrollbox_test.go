package widget

import (
	"fmt"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// ScrollBox test helpers.

// newTestScrollBox returns a laid-out ScrollBox with explicit content dimensions.
func newTestScrollBox(vpW, vpH, contentW, contentH int, opts ...ScrollBoxOption) *ScrollBox {
	sb := NewScrollBox(opts...)
	testutil.SetupWidget(sb, vpW, vpH)
	sb.SetContentHeight(contentH)
	sb.SetContentWidth(contentW)
	return sb
}

func pressKey(sb *ScrollBox, name string) bool {
	e := &input.KeyEvent{Name: name, Type: input.KeyPress}
	return sb.HandleKeyPress(e)
}

// Core scrolling behavior.

func TestNewScrollBox_Defaults(t *testing.T) {
	sb := NewScrollBox()
	if sb == nil {
		t.Fatal("NewScrollBox returned nil")
	}
	if sb.ScrollY() != 0 {
		t.Errorf("expected scrollY 0, got %d", sb.ScrollY())
	}
	if sb.ScrollX() != 0 {
		t.Errorf("expected scrollX 0, got %d", sb.ScrollX())
	}

}

func TestScrollBox_SetScroll(t *testing.T) {
	sb := NewScrollBox()
	testutil.SetupWidget(sb, 20, 10)

	sb.contentHeight = 30
	sb.SetScrollY(5)
	if sb.ScrollY() != 5 {
		t.Errorf("expected scrollY 5, got %d", sb.ScrollY())
	}

	// Offsets beyond the content clamp to the maximum.
	sb.SetScrollY(100)
	if sb.ScrollY() > 20 {
		t.Errorf("expected scrollY clamped, got %d", sb.ScrollY())
	}

	// Negative offsets clamp to zero.
	sb.SetScrollY(-5)
	if sb.ScrollY() != 0 {
		t.Errorf("expected scrollY 0, got %d", sb.ScrollY())
	}
}

func TestScrollBox_ScrollToBottom(t *testing.T) {
	sb := NewScrollBox()
	testutil.SetupWidget(sb, 20, 10)

	sb.contentHeight = 30
	sb.ScrollToBottom()
	if !sb.IsAtBottom() {
		t.Error("expected IsAtBottom true after ScrollToBottom")
	}
}

func TestScrollBox_ScrollToTop(t *testing.T) {
	sb := NewScrollBox()
	testutil.SetupWidget(sb, 20, 10)

	sb.contentHeight = 30
	sb.SetScrollY(10)
	sb.ScrollToTop()
	if sb.ScrollY() != 0 {
		t.Errorf("expected scrollY 0 after ScrollToTop, got %d", sb.ScrollY())
	}
}

func TestScrollBox_AutoScroll(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithAutoScroll(AutoScrollBottom))
	testutil.SetupWidget(sb, 20, 10)

	sb.contentHeight = 10
	sb.ScrollToBottom()
	if !sb.IsAtBottom() {
		t.Error("expected at bottom")
	}

	// Content growth keeps a followed bottom edge visible.
	sb.SetContentHeight(20)
	if !sb.IsAtBottom() {
		t.Error("expected auto-scroll to keep at bottom")
	}
}

func TestScrollBox_Render(t *testing.T) {
	sb := NewScrollBox(
		ScrollBoxWithBackgroundColor(style.ColorBlack),
	)
	testutil.SetupWidget(sb, 20, 10)

	buf := buffer.New(20, 10)
	// An empty ScrollBox must render safely.
	sb.RenderSelf(buf, 0)
}

func TestScrollBox_Options(t *testing.T) {
	sb := NewScrollBox(
		ScrollBoxWithID("my-scroll"),
		ScrollBoxWithWidth(layout.Points(30)),
		ScrollBoxWithHeight(layout.Points(20)),
		ScrollBoxWithFlexGrow(1),
	)

	if sb.ID() != "my-scroll" {
		t.Errorf("expected id 'my-scroll', got %q", sb.ID())
	}
}

// Edge queries.

func TestScrollBox_IsAtEdges(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 30)

	if !sb.IsAtTop() {
		t.Error("expected IsAtTop at init")
	}
	if !sb.IsAtLeft() {
		t.Error("expected IsAtLeft at init")
	}

	sb.ScrollToBottom()
	if !sb.IsAtBottom() {
		t.Error("expected IsAtBottom")
	}
	if sb.IsAtTop() {
		t.Error("did not expect IsAtTop at bottom")
	}

	sb.ScrollToRight()
	if !sb.IsAtRight() {
		t.Error("expected IsAtRight")
	}
	if sb.IsAtLeft() {
		t.Error("did not expect IsAtLeft at right")
	}
}

func TestScrollBox_ScrollBy(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 30)

	sb.ScrollBy(5, 3)
	if sb.ScrollX() != 5 {
		t.Errorf("expected scrollX 5, got %d", sb.ScrollX())
	}
	if sb.ScrollY() != 3 {
		t.Errorf("expected scrollY 3, got %d", sb.ScrollY())
	}
}

func TestScrollBox_ScrollTo(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 30)

	sb.ScrollTo(7, 12)
	if sb.ScrollX() != 7 {
		t.Errorf("expected scrollX 7, got %d", sb.ScrollX())
	}
	if sb.ScrollY() != 12 {
		t.Errorf("expected scrollY 12, got %d", sb.ScrollY())
	}
}

// TestScrollBox_ContentSizeIncludesTrailingMargins checks that a child's bottom
// and right margins count as content, so a viewport at the end shows the space
// they leave. A Gone child takes no space, margins included.
func TestScrollBox_ContentSizeIncludesTrailingMargins(t *testing.T) {
	gone := NewBox(BoxWithHeight(layout.Points(1)), BoxWithMarginBottom(layout.Points(5)))
	gone.SetGone(true)

	for _, tt := range []struct {
		name          string
		dir           layout.FlexDirection
		children      []render.Renderable
		width, height int // The wanted content size.
	}{
		{
			name: "bottom margin",
			dir:  layout.DirectionColumn,
			children: []render.Renderable{NewBox(BoxWithWidth(layout.Points(4)),
				BoxWithHeight(layout.Points(3)), BoxWithFlexShrink(0), BoxWithMarginBottom(layout.Points(1)))},
			width: 4, height: 4,
		},
		{
			name: "right margin",
			dir:  layout.DirectionRow,
			children: []render.Renderable{NewBox(BoxWithWidth(layout.Points(5)),
				BoxWithHeight(layout.Points(1)), BoxWithFlexShrink(0), BoxWithMarginRight(layout.Points(2)))},
			width: 7, height: 1,
		},
		{
			name: "gone child",
			dir:  layout.DirectionColumn,
			children: []render.Renderable{NewBox(BoxWithWidth(layout.Points(4)),
				BoxWithHeight(layout.Points(2)), BoxWithFlexShrink(0)), gone},
			width: 4, height: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewScrollBox(ScrollBoxWithFlexDirection(tt.dir), ScrollBoxWithChildren(tt.children...))
			testutil.SetupWidget(sb, 3, 1)
			sb.OnPostLayout(0)

			if sb.ContentWidth() != tt.width || sb.ContentHeight() != tt.height {
				t.Errorf("content is %dx%d, want %dx%d", sb.ContentWidth(), sb.ContentHeight(), tt.width, tt.height)
			}
		})
	}
}

// TestScrollBox_AutoScroll_BottomShowsTrailingMargin checks that following the
// bottom edge keeps the last child's bottom margin in view.
func TestScrollBox_AutoScroll_BottomShowsTrailingMargin(t *testing.T) {
	last := NewBox(BoxWithHeight(layout.Points(3)), BoxWithFlexShrink(0), BoxWithMarginBottom(layout.Points(1)))
	sb := NewScrollBox(
		ScrollBoxWithFlexDirection(layout.DirectionColumn),
		ScrollBoxWithAutoScroll(AutoScrollBottom),
		ScrollBoxWithChildren(last),
	)
	testutil.SetupWidget(sb, 20, 2)
	sb.OnPostLayout(0)

	// Three rows and the margin leave the last row of the box and the margin
	// in the two-row viewport.
	if got := sb.ScrollY(); got != 2 {
		t.Errorf("scrollY = %d, want 2", got)
	}
}

// Auto-scroll in each direction.

func TestScrollBox_AutoScroll_Bottom(t *testing.T) {
	// Start with content that fits at the followed bottom edge.
	sb := newTestScrollBox(20, 10, 20, 10, ScrollBoxWithAutoScroll(AutoScrollBottom))

	// Fitting content has a zero maximum offset and is therefore at the bottom.
	if !sb.IsAtBottom() {
		t.Error("expected at bottom initially")
	}

	// Growing content moves the offset to the new bottom.
	sb.SetContentHeight(30)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom, scrollY=%d maxScrollY=%d", sb.ScrollY(), sb.maxScrollY())
	}

	// A manual offset away from the bottom suspends edge following.
	sb.SetScrollY(5)
	if sb.IsAtBottom() {
		t.Error("should not be at bottom after manual scroll up")
	}

	// Later growth retains the manual viewport.
	sb.SetContentHeight(40)
	if sb.IsAtBottom() {
		t.Error("should not re-engage auto-scroll after manual scroll away")
	}

	// Returning to the bottom re-enables edge following.
	sb.ScrollToBottom()
	if sb.UserScrolled() {
		t.Error("expected userScrolled to clear when returning to auto-scroll edge")
	}

	// Subsequent growth follows the bottom again.
	sb.SetContentHeight(50)
	if !sb.IsAtBottom() {
		t.Error("expected auto-scroll to re-engage after scrolling back to bottom")
	}
}

func TestScrollBox_AutoScroll_Top(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 30, ScrollBoxWithAutoScroll(AutoScrollTop))

	if !sb.IsAtTop() {
		t.Error("expected at top with AutoScrollTop")
	}

	// Content growth does not move a followed top edge.
	sb.SetContentHeight(50)
	if !sb.IsAtTop() {
		t.Error("expected auto-scroll top to hold")
	}

	// Manual movement suspends top-edge following.
	sb.SetScrollY(10)
	sb.SetContentHeight(60)
	if sb.ScrollY() == 0 {
		t.Error("should not snap to top after manual scroll")
	}

	// Returning to the top re-enables edge following.
	sb.ScrollToTop()
	if sb.UserScrolled() {
		t.Error("expected userScrolled to clear at top")
	}
	sb.SetContentHeight(70)
	if !sb.IsAtTop() {
		t.Error("expected auto-scroll top to re-engage")
	}
}

func TestScrollBox_AutoScroll_Left(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 10, ScrollBoxWithAutoScroll(AutoScrollLeft))

	if !sb.IsAtLeft() {
		t.Error("expected at left with AutoScrollLeft")
	}

	sb.SetContentWidth(60)
	if !sb.IsAtLeft() {
		t.Error("expected auto-scroll left to hold")
	}

	sb.SetScrollX(10)
	sb.SetContentWidth(80)
	if sb.ScrollX() == 0 {
		t.Error("should not snap to left after manual scroll")
	}

	sb.ScrollToLeft()
	sb.SetContentWidth(100)
	if !sb.IsAtLeft() {
		t.Error("expected auto-scroll left to re-engage")
	}
}

func TestScrollBox_AutoScroll_Right(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 10, ScrollBoxWithAutoScroll(AutoScrollRight))

	if !sb.IsAtRight() {
		t.Error("expected at right with AutoScrollRight")
	}

	sb.SetContentWidth(60)
	if !sb.IsAtRight() {
		t.Error("expected auto-scroll right to hold after grow")
	}

	sb.SetScrollX(5)
	sb.SetContentWidth(80)
	if sb.IsAtRight() {
		t.Error("should not snap to right after manual scroll")
	}

	sb.ScrollToRight()
	sb.SetContentWidth(100)
	if !sb.IsAtRight() {
		t.Error("expected auto-scroll right to re-engage")
	}
}

func TestScrollBox_AutoScroll_Option(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithAutoScroll(AutoScrollTop))
	if sb.AutoScroll() != AutoScrollTop {
		t.Errorf("expected AutoScrollTop, got %d", sb.AutoScroll())
	}
}

func TestScrollBox_AutoScrollOption_DefaultsToBottom(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithAutoScroll(AutoScrollBottom))
	if sb.autoScroll != AutoScrollBottom {
		t.Errorf("expected AutoScrollBottom default, got %d", sb.autoScroll)
	}
}

// Keyboard navigation.

func TestScrollBox_KeyNav_UpDown(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)

	// Down advances the vertical offset.
	if !pressKey(sb, input.KeyDown) {
		t.Error("expected down key handled")
	}
	if sb.ScrollY() == 0 {
		t.Error("expected scrollY > 0 after down")
	}

	yAfterDown := sb.ScrollY()

	// Up reduces the vertical offset.
	pressKey(sb, input.KeyUp)
	if sb.ScrollY() >= yAfterDown {
		t.Errorf("expected scrollY < %d after up, got %d", yAfterDown, sb.ScrollY())
	}
}

func TestScrollBox_KeyNav_JK(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)

	pressKey(sb, "j")
	if sb.ScrollY() == 0 {
		t.Error("expected j to scroll down")
	}

	y := sb.ScrollY()
	pressKey(sb, "k")
	if sb.ScrollY() >= y {
		t.Error("expected k to scroll up")
	}
}

func TestScrollBox_KeyNav_LeftRight(t *testing.T) {
	sb := newTestScrollBox(20, 10, 50, 10)

	pressKey(sb, input.KeyRight)
	if sb.ScrollX() == 0 {
		t.Error("expected scrollX > 0 after right")
	}

	x := sb.ScrollX()
	pressKey(sb, input.KeyLeft)
	if sb.ScrollX() >= x {
		t.Error("expected scrollX to decrease after left")
	}
}

func TestScrollBox_KeyNav_HL(t *testing.T) {
	sb := newTestScrollBox(20, 10, 50, 10)

	pressKey(sb, "l")
	if sb.ScrollX() == 0 {
		t.Error("expected l to scroll right")
	}

	x := sb.ScrollX()
	pressKey(sb, "h")
	if sb.ScrollX() >= x {
		t.Error("expected h to scroll left")
	}
}

func TestScrollBox_KeyNav_PageUpDown(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 100)

	pressKey(sb, input.KeyPageDown)
	yDown := sb.ScrollY()
	if yDown < 5 {
		t.Errorf("expected page down to scroll at least 5 (1/2 viewport), got %d", yDown)
	}

	pressKey(sb, input.KeyPageUp)
	if sb.ScrollY() >= yDown {
		t.Error("expected page up to scroll back")
	}
}

func TestScrollBox_KeyNav_HomeEnd(t *testing.T) {
	sb := newTestScrollBox(20, 10, 50, 100)

	// Start from nonzero offsets on both axes.
	sb.SetScrollY(30)
	sb.SetScrollX(10)

	pressKey(sb, input.KeyHome)
	if sb.ScrollY() != 0 {
		t.Errorf("expected home to scroll to top, got scrollY=%d", sb.ScrollY())
	}
	if sb.ScrollX() != 0 {
		t.Errorf("expected home to scroll to left, got scrollX=%d", sb.ScrollX())
	}

	pressKey(sb, input.KeyEnd)
	if !sb.IsAtBottom() {
		t.Error("expected end to scroll to bottom")
	}
	if !sb.IsAtRight() {
		t.Error("expected end to scroll to right")
	}
}

func TestScrollBox_KeyNav_NoContentNoScroll(t *testing.T) {
	// Fitting content cannot move in response to navigation.
	sb := newTestScrollBox(20, 10, 20, 10)

	pressKey(sb, input.KeyDown)
	if sb.ScrollY() != 0 {
		t.Error("expected no scroll when content fits viewport")
	}
}

func TestScrollBox_KeyNav_ReleaseIgnored(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)
	e := &input.KeyEvent{Name: input.KeyDown, Type: input.KeyRelease}
	if sb.HandleKeyPress(e) {
		t.Error("expected key release to be ignored")
	}
}

func TestScrollBox_KeyNav_UnhandledKey(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)
	e := &input.KeyEvent{Name: "x", Type: input.KeyPress}
	if sb.HandleKeyPress(e) {
		t.Error("expected 'x' to not be handled")
	}
}

func TestScrollBox_KeyNav_SetsManualScroll(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50, ScrollBoxWithAutoScroll(AutoScrollBottom))

	// Edge-following state starts engaged.
	if sb.UserScrolled() {
		t.Error("expected no manual scroll initially")
	}

	pressKey(sb, input.KeyUp)
	if !sb.UserScrolled() {
		t.Error("expected keyboard scroll to set userScrolled")
	}
}

func TestScrollBox_KeyNav_StopsPropagation(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)

	e := &input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress}
	sb.HandleKeyPress(e)
	if !e.PropagationStopped() {
		t.Error("expected propagation stopped for handled key")
	}
}

func TestScrollBox_FocusableOption(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithFocusable(true))
	if !sb.Focusable() {
		t.Error("expected focusable after option")
	}
}

func TestScrollBox_ViewportDimensions(t *testing.T) {
	sb := newTestScrollBox(20, 10, 40, 30)

	if sb.ViewportHeight() != 10 {
		t.Errorf("expected viewport height 10, got %d", sb.ViewportHeight())
	}
	if sb.ViewportWidth() != 20 {
		t.Errorf("expected viewport width 20, got %d", sb.ViewportWidth())
	}
}

// ScrollBoxWithChildren behavior.

func TestScrollBox_WithChildren_AddsChildren(t *testing.T) {
	txt1 := NewText("line 1")
	txt2 := NewText("line 2")
	sb := NewScrollBox(
		ScrollBoxWithChildren(txt1, txt2),
	)

	if sb.ChildrenCount() != 2 {
		t.Fatalf("expected 2 children, got %d", sb.ChildrenCount())
	}
}

func TestScrollBox_WithChildren_SetsParent(t *testing.T) {
	txt := NewText("child")
	sb := NewScrollBox(
		ScrollBoxWithID("scroll-parent"),
		ScrollBoxWithChildren(txt),
	)

	if txt.Parent() == nil {
		t.Fatal("expected child to have parent set")
	}
	if txt.Parent().ID() != sb.ID() {
		t.Errorf("expected parent %q, got %q", sb.ID(), txt.Parent().ID())
	}
}

func TestScrollBox_WithChildren_SkipsNil(t *testing.T) {
	txt := NewText("valid")
	sb := NewScrollBox(ScrollBoxWithChildren(nil, txt, nil))

	if sb.ChildrenCount() != 1 {
		t.Fatalf("expected 1 child (nils skipped), got %d", sb.ChildrenCount())
	}
}

func TestScrollBox_WithChildren_WithOtherOptions(t *testing.T) {
	txt := NewText("hello")
	sb := NewScrollBox(
		ScrollBoxWithID("combo-scroll"),
		ScrollBoxWithAutoScroll(AutoScrollBottom),
		ScrollBoxWithChildren(txt),
	)

	if sb.ID() != "combo-scroll" {
		t.Errorf("expected id 'combo-scroll', got %q", sb.ID())
	}
	if sb.ChildrenCount() != 1 {
		t.Errorf("expected 1 child, got %d", sb.ChildrenCount())
	}
}

// Anchor-based scrolling.

// newAnchorScrollBox returns a laid-out column ScrollBox with fixed-height children.
func newAnchorScrollBox(vpW, vpH int, childHeights ...int) *ScrollBox {
	var children []Box_ChildEntry
	for i, h := range childHeights {
		txt := NewText(
			fmt.Sprintf("child-%d", i),
			TextWithHeight(layout.Points(h)),
		)
		children = append(children, Box_ChildEntry{child: txt})
	}

	sb := NewScrollBox(
		ScrollBoxWithFlexDirection(layout.DirectionColumn),
		ScrollBoxWithAnchorScroll(true),
		ScrollBoxWithAutoScroll(AutoScrollBottom),
	)
	for _, c := range children {
		sb.Add(c.child)
	}

	// Anchors require final child positions from layout.
	ln := sb.LayoutNode()
	ln.Width = layout.Points(vpW)
	ln.Height = layout.Points(vpH)
	ln.ComputeLayout(vpW, vpH)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// Content height is the sum of fixed child heights.
	total := 0
	for _, h := range childHeights {
		total += h
	}
	sb.SetContentHeight(total)
	return sb
}

// Box_ChildEntry stores a Text child while building an anchored ScrollBox.
type Box_ChildEntry struct {
	child *Text
}

func TestScrollBox_AnchorScroll_Option(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithAnchorScroll(true))
	if !sb.anchorScroll {
		t.Error("expected anchorScroll to be true")
	}
}

func TestScrollBox_AnchorScroll_ComputesAnchor(t *testing.T) {
	// Three children total 60 cells in a 15-cell viewport.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// At y=15, child 1 starts at y=10 and has an anchor offset of 5.
	sb.SetScrollY(15)

	anchor := sb.Anchor()
	if !anchor.Valid {
		t.Fatal("expected valid anchor")
	}
	if anchor.ChildIndex != 1 {
		t.Errorf("expected anchor child 1, got %d", anchor.ChildIndex)
	}
	if anchor.Offset != 5 {
		t.Errorf("expected anchor offset 5, got %d", anchor.Offset)
	}
}

func TestScrollBox_AnchorScroll_AnchorAtFirstChild(t *testing.T) {
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// At y=3, child 0 intersects the viewport top at offset 3.
	sb.SetScrollY(3)

	anchor := sb.Anchor()
	if !anchor.Valid {
		t.Fatal("expected valid anchor")
	}
	if anchor.ChildIndex != 0 {
		t.Errorf("expected anchor child 0, got %d", anchor.ChildIndex)
	}
	if anchor.Offset != 3 {
		t.Errorf("expected anchor offset 3, got %d", anchor.Offset)
	}
}

func TestScrollBox_AnchorScroll_AnchorAtLastChild(t *testing.T) {
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// At y=35, child 2 starts at y=30 and has an anchor offset of 5.
	sb.SetScrollY(35)

	anchor := sb.Anchor()
	if !anchor.Valid {
		t.Fatal("expected valid anchor")
	}
	if anchor.ChildIndex != 2 {
		t.Errorf("expected anchor child 2, got %d", anchor.ChildIndex)
	}
	if anchor.Offset != 5 {
		t.Errorf("expected anchor offset 5, got %d", anchor.Offset)
	}
}

func TestScrollBox_AnchorScroll_RestoresOnContentGrowth(t *testing.T) {
	// Use a 15-cell viewport over children with heights 10, 20, and 30.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Anchor five cells into child 1.
	sb.SetScrollY(15)
	if sb.ScrollY() != 15 {
		t.Fatalf("expected scrollY=15, got %d", sb.ScrollY())
	}

	// Grow child 0 by ten cells and recompute all child positions.
	children := sb.Children()
	children[0].LayoutNode().Height = layout.Points(20) // Increase from 10 to 20.
	children[0].LayoutNode().MarkDirty()

	ln := sb.LayoutNode()
	ln.ComputeLayout(20, 15)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// Recalculate against the new total height of 70.
	sb.SetContentHeight(70)

	// Child 1 now starts at 20, so the saved offset resolves to y=25.
	if sb.ScrollY() != 25 {
		t.Errorf("expected scrollY=25 after anchor restore, got %d", sb.ScrollY())
	}
}

func TestScrollBox_AnchorScroll_DoesNotAffectAutoScrollBottom(t *testing.T) {
	// Edge following takes precedence over a saved anchor.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// The initial maximum offset is 45.
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom with auto-scroll")
	}

	// Content growth moves to the new maximum offset of 65.
	sb.SetContentHeight(80)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom to hold, scrollY=%d", sb.ScrollY())
	}
}

func TestScrollBox_AnchorScroll_NoChildren(t *testing.T) {
	sb := NewScrollBox(
		ScrollBoxWithAnchorScroll(true),
	)
	testutil.SetupWidget(sb, 20, 15)
	sb.SetContentHeight(0)

	// An empty child list cannot produce a valid anchor.
	sb.SetScrollY(0)
	anchor := sb.Anchor()
	if anchor.Valid {
		t.Error("expected invalid anchor with no children")
	}
}

func TestScrollBox_AnchorScroll_ChildRemoved(t *testing.T) {
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Start with an anchor in the final child.
	sb.SetScrollY(35)
	anchor := sb.Anchor()
	if anchor.ChildIndex != 2 {
		t.Fatalf("expected anchor child 2, got %d", anchor.ChildIndex)
	}

	// Removing that child leaves the saved index out of range.
	lastChild := sb.Children()[2]
	sb.Remove(lastChild.ID())

	// Anchor resolution falls back to the new final child.
	newY := sb.scrollYFromAnchor()
	if newY < 0 {
		t.Errorf("expected non-negative scrollY from anchor, got %d", newY)
	}
}

func TestScrollBox_AnchorScroll_WithOnPostLayout(t *testing.T) {
	// Exercise anchor restoration through the post-layout hook.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Anchor five cells into child 1.
	sb.SetScrollY(15)

	// Grow child 0 and recompute positions.
	children := sb.Children()
	children[0].LayoutNode().Height = layout.Points(25) // Increase from 10 to 25.
	children[0].LayoutNode().MarkDirty()

	ln := sb.LayoutNode()
	ln.ComputeLayout(20, 15)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// OnPostLayout detects the new content height and restores the anchor.
	sb.OnPostLayout(0)

	// Child 1 moved to y=25, so the saved offset resolves to y=30.
	if sb.ScrollY() != 30 {
		t.Errorf("expected scrollY=30 after OnPostLayout anchor restore, got %d", sb.ScrollY())
	}
}

func TestScrollBox_AnchorScroll_ScrollBackToBottomReengagesAutoScroll(t *testing.T) {
	// Returning to the followed bottom must disable anchor restoration.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	// The ScrollBox starts at its maximum offset of 45.

	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom initially")
	}

	// Moving upward suspends bottom-edge following.
	sb.SetScrollY(15)
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after scrolling up")
	}

	// Returning to the bottom clears the manual-scroll state.
	sb.ScrollToBottom()
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after scrolling back to bottom")
	}
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom after ScrollToBottom")
	}

	// New content follows the bottom to its new maximum offset.
	sb.SetContentHeight(80)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom to re-engage, scrollY=%d maxScrollY=%d",
			sb.ScrollY(), sb.ContentHeight()-sb.ViewportHeight())
	}
}

func TestScrollBox_AnchorScroll_HandleScrollBackToBottomReengagesAutoScroll(t *testing.T) {
	// Exercise re-engagement through mouse-wheel handling.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	// The initial maximum offset is 45.

	// A 30-row upward delta moves from 45 to 15.
	sb.handleScroll(0, -30, false)
	if sb.ScrollY() != 15 {
		t.Fatalf("expected scrollY=15 after scroll up, got %d", sb.ScrollY())
	}
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after mouse scroll up")
	}

	// A large downward delta clamps to the bottom.
	sb.handleScroll(0, 100, false)
	if !sb.IsAtBottom() {
		t.Fatalf("expected at bottom after mouse scroll down, scrollY=%d", sb.ScrollY())
	}
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after scrolling back to bottom via handleScroll")
	}

	// Content growth now follows the bottom again.
	sb.SetContentHeight(80)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom after content growth, scrollY=%d maxScrollY=%d",
			sb.ScrollY(), sb.ContentHeight()-sb.ViewportHeight())
	}
}

func TestScrollBox_AnchorScroll_KeyNavBackToBottomReengagesAutoScroll(t *testing.T) {
	// Exercise re-engagement through keyboard handling.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// PageUp suspends bottom-edge following.
	pressKey(sb, input.KeyPageUp)
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after PageUp")
	}

	// End returns to the followed edge.
	pressKey(sb, input.KeyEnd)
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom after End key")
	}
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after End key to bottom")
	}

	// Content growth follows the bottom again.
	sb.SetContentHeight(80)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom after content growth via key nav, scrollY=%d",
			sb.ScrollY())
	}
}

// resizeChild sets the height of sb's child i to h, lays sb out again in a
// vpW by vpH viewport, and runs its post-layout hook, as a frame does.
func resizeChild(sb *ScrollBox, i, h, vpW, vpH int) {
	child := sb.Children()[i]
	child.LayoutNode().Height = layout.Points(h)
	child.LayoutNode().MarkDirty()
	sb.LayoutNode().ComputeLayout(vpW, vpH)
	sb.UpdateFromLayout()
	for _, c := range sb.Children() {
		c.UpdateFromLayout()
	}
	sb.OnPostLayout(0)
}

func TestScrollBox_StopFollowing_GrowthExtendsBelow(t *testing.T) {
	// The 15-cell viewport follows the bottom at y=45, the last 15 cells of
	// child 2, which spans 30 to 60.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom initially")
	}

	sb.StopFollowing()
	if !sb.UserScrolled() {
		t.Fatal("expected StopFollowing to suspend edge following")
	}

	// Child 2 grows by ten cells below what the viewport shows.
	resizeChild(sb, 2, 40, 20, 15)
	if sb.ScrollY() != 45 {
		t.Errorf("expected scrollY to stay 45 as the content grows below, got %d", sb.ScrollY())
	}
	if sb.IsAtBottom() || !sb.UserScrolled() {
		t.Errorf("expected the viewport above the new bottom, not following it; scrollY=%d", sb.ScrollY())
	}
}

func TestScrollBox_StopFollowing_ShrinkToEdgeFollowsAgain(t *testing.T) {
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	sb.StopFollowing()
	resizeChild(sb, 2, 40, 20, 15)

	// Shrinking back brings the bottom edge into view at y=45 again.
	sb.StopFollowing()
	resizeChild(sb, 2, 30, 20, 15)
	if sb.ScrollY() != 45 || !sb.IsAtBottom() {
		t.Fatalf("expected scrollY=45 at the bottom after shrinking, got %d", sb.ScrollY())
	}
	if sb.UserScrolled() {
		t.Fatal("expected edge following to resume at the bottom")
	}

	// New content is followed once more.
	resizeChild(sb, 2, 50, 20, 15)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom after growth, scrollY=%d", sb.ScrollY())
	}
}

func TestScrollBox_StopFollowing_NoAutoScroll(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)
	sb.StopFollowing()
	if sb.UserScrolled() {
		t.Error("expected StopFollowing to do nothing without an auto-scroll edge")
	}
}

// Sticky positioning.

// newStickyScrollBox returns a laid-out column whose indexed children stick to the top.
func newStickyScrollBox(vpW, vpH int, childHeights []int, stickyIndices []int) *ScrollBox {
	stickySet := map[int]bool{}
	for _, i := range stickyIndices {
		stickySet[i] = true
	}

	sb := NewScrollBox(
		ScrollBoxWithFlexDirection(layout.DirectionColumn),
		ScrollBoxWithAutoScroll(AutoScrollBottom),
	)
	for i, h := range childHeights {
		txt := NewText(
			fmt.Sprintf("child-%d", i),
			TextWithHeight(layout.Points(h)),
		)
		if stickySet[i] {
			txt.LayoutNode().Sticky = layout.StickyTop
		}
		sb.Add(txt)
	}

	ln := sb.LayoutNode()
	ln.Width = layout.Points(vpW)
	ln.Height = layout.Points(vpH)
	ln.ComputeLayout(vpW, vpH)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	total := 0
	for _, h := range childHeights {
		total += h
	}
	sb.SetContentHeight(total)
	return sb
}

func TestScrollBox_Sticky_NoOffset_WhenNotScrolledPast(t *testing.T) {
	// Child 0 already touches the viewport top at scrollY zero.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(0)
	child := sb.Children()[0]
	if child.StickyOffsetY() != 0 {
		t.Errorf("expected stickyOffsetY=0 when not scrolled past, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_PinsAtTop(t *testing.T) {
	// At scrollY five, child 0 needs a positive five-cell offset to remain visible.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(5)
	child := sb.Children()[0]
	if child.StickyOffsetY() != 5 {
		t.Errorf("expected stickyOffsetY=5, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_SectionScoped_PushedByNext(t *testing.T) {
	// Child 2 starts at 40, limiting child 0's sticky offset to 30.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})

	sb.SetScrollY(35)
	child0 := sb.Children()[0]
	child2 := sb.Children()[2]

	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30, got %d", child0.StickyOffsetY())
	}

	// Child 2 has not yet crossed the viewport top, so its offset remains zero.
	if child2.StickyOffsetY() != 0 {
		t.Errorf("expected child[2] stickyOffsetY=0 (not yet scrolled past), got %d", child2.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_SecondTakesOver(t *testing.T) {
	// Scroll far enough for the second sticky section to take over.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})

	sb.SetScrollY(50)
	child0 := sb.Children()[0]
	child2 := sb.Children()[2]

	// Child 0 remains clamped at its section limit of 30.
	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30, got %d", child0.StickyOffsetY())
	}

	// Child 2 sticks with a ten-cell offset.
	if child2.StickyOffsetY() != 10 {
		t.Errorf("expected child[2] stickyOffsetY=10, got %d", child2.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_NonStickyChildrenHaveZeroOffset(t *testing.T) {
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(15)
	// Non-sticky child 1 must retain zero offset.
	child1 := sb.Children()[1]
	if child1.StickyOffsetY() != 0 {
		t.Errorf("expected non-sticky child stickyOffsetY=0, got %d", child1.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_NoStickyChildren(t *testing.T) {
	// A container without sticky children leaves every offset at zero.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, nil)
	sb.SetScrollY(15)
	for i, child := range sb.Children() {
		if child.StickyOffsetY() != 0 {
			t.Errorf("child[%d] expected stickyOffsetY=0 with no sticky children, got %d", i, child.StickyOffsetY())
		}
	}
}

func TestScrollBox_StickyBottom_PinsAtBottom(t *testing.T) {
	// Child 2 ends at 50 while the viewport ends at 20, requiring offset -30.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, nil)
	sb.Children()[2].LayoutNode().Sticky = layout.StickyBottom
	sb.SetScrollY(0)
	sb.SetContentHeight(50)

	child := sb.Children()[2]
	if child.StickyOffsetY() != -30 {
		t.Errorf("expected stickyOffsetY=-30 (pinned at bottom), got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_StickyBottom_NoOffset_WhenVisible(t *testing.T) {
	// A fully visible bottom-sticky child needs no offset.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, nil)
	sb.Children()[2].LayoutNode().Sticky = layout.StickyBottom
	sb.SetContentHeight(50)
	sb.SetScrollY(30) // Viewport 30..50 contains child 2 at 40..50.

	child := sb.Children()[2]
	if child.StickyOffsetY() != 0 {
		t.Errorf("expected stickyOffsetY=0 when fully visible, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_StickyLeft_PinsAtLeft(t *testing.T) {
	// Child 0 sticks to the left in a horizontally scrolling row.
	sb := NewScrollBox(
		ScrollBoxWithFlexDirection(layout.DirectionRow),
	)
	for i, w := range []int{10, 30, 10} {
		txt := NewText(fmt.Sprintf("c%d", i), TextWithWidth(layout.Points(w)))
		if i == 0 {
			txt.LayoutNode().Sticky = layout.StickyLeft
		}
		sb.Add(txt)
	}
	ln := sb.LayoutNode()
	ln.Width = layout.Points(20)
	ln.Height = layout.Points(10)
	ln.ComputeLayout(20, 10)
	sb.UpdateFromLayout()
	for _, c := range sb.Children() {
		c.UpdateFromLayout()
	}
	sb.SetContentWidth(50)

	sb.SetScrollX(5)
	child := sb.Children()[0]
	if child.StickyOffsetX() != 5 {
		t.Errorf("expected stickyOffsetX=5, got %d", child.StickyOffsetX())
	}
}

func TestScrollBox_StickyRight_PinsAtRight(t *testing.T) {
	// Child 2 ends at 50 while the viewport ends at 20, requiring offset -30.
	sb := NewScrollBox(
		ScrollBoxWithFlexDirection(layout.DirectionRow),
	)
	for i, w := range []int{10, 30, 10} {
		txt := NewText(fmt.Sprintf("c%d", i), TextWithWidth(layout.Points(w)))
		if i == 2 {
			txt.LayoutNode().Sticky = layout.StickyRight
		}
		sb.Add(txt)
	}
	ln := sb.LayoutNode()
	ln.Width = layout.Points(20)
	ln.Height = layout.Points(10)
	ln.ComputeLayout(20, 10)
	sb.UpdateFromLayout()
	for _, c := range sb.Children() {
		c.UpdateFromLayout()
	}
	sb.SetContentWidth(50)

	sb.SetScrollX(0)
	child := sb.Children()[2]
	if child.StickyOffsetX() != -30 {
		t.Errorf("expected stickyOffsetX=-30 (pinned at right), got %d", child.StickyOffsetX())
	}
}

func TestScrollBox_StickyGap_Default(t *testing.T) {
	sb := NewScrollBox()
	if sb.StickyGap() != 0 {
		t.Errorf("expected default stickyGap=0, got %d", sb.StickyGap())
	}
}

func TestScrollBox_StickyGap_Option(t *testing.T) {
	sb := NewScrollBox(ScrollBoxWithStickyGap(2))
	if sb.StickyGap() != 2 {
		t.Errorf("expected stickyGap=2, got %d", sb.StickyGap())
	}
}

func TestScrollBox_StickyGap_PushAway(t *testing.T) {
	// A two-cell gap reduces child 0's section limit from 30 to 28.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})
	sb.SetStickyGap(2)

	sb.SetScrollY(35)
	child0 := sb.Children()[0]

	if child0.StickyOffsetY() != 28 {
		t.Errorf("expected child[0] stickyOffsetY=28 (gap=2), got %d", child0.StickyOffsetY())
	}
}

func TestScrollBox_StickyGap_ZeroNoGap(t *testing.T) {
	// A zero gap lets consecutive sticky children touch.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})
	sb.SetStickyGap(0)

	sb.SetScrollY(35)
	child0 := sb.Children()[0]

	// Child 0's section limit remains 30.
	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30 (gap=0), got %d", child0.StickyOffsetY())
	}
}
