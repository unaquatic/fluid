package widget

import (
	"fmt"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestScrollBox creates a scrollbox with a given viewport and content size.
// It uses SetContentHeight/SetContentWidth so that auto-scroll logic runs.
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

// ---------------------------------------------------------------------------
// Original tests (preserved)
// ---------------------------------------------------------------------------

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

	// Clamp to max
	sb.SetScrollY(100)
	if sb.ScrollY() > 20 {
		t.Errorf("expected scrollY clamped, got %d", sb.ScrollY())
	}

	// Clamp to min
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

	// Increase content height - should stay at bottom
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
	// Should not panic
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

// ---------------------------------------------------------------------------
// Edge helper tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// 4-directional auto-scroll
// ---------------------------------------------------------------------------

func TestScrollBox_AutoScroll_Bottom(t *testing.T) {
	// Default AutoScroll(true) implies AutoScrollBottom for backward compat.
	sb := newTestScrollBox(20, 10, 20, 10, ScrollBoxWithAutoScroll(AutoScrollBottom))

	// Initially at bottom (content fits viewport, maxScrollY = 0).
	if !sb.IsAtBottom() {
		t.Error("expected at bottom initially")
	}

	// Growing content should keep it at bottom.
	sb.SetContentHeight(30)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom, scrollY=%d maxScrollY=%d", sb.ScrollY(), sb.maxScrollY())
	}

	// Manually scroll away from bottom.
	sb.SetScrollY(5)
	if sb.IsAtBottom() {
		t.Error("should not be at bottom after manual scroll up")
	}

	// Growing content again should NOT snap to bottom because user scrolled away.
	sb.SetContentHeight(40)
	if sb.IsAtBottom() {
		t.Error("should not re-engage auto-scroll after manual scroll away")
	}

	// Scroll back to bottom — userScrolled should clear.
	sb.ScrollToBottom()
	if sb.UserScrolled() {
		t.Error("expected userScrolled to clear when returning to auto-scroll edge")
	}

	// Growing again should stick.
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

	// Content change should keep it at top.
	sb.SetContentHeight(50)
	if !sb.IsAtTop() {
		t.Error("expected auto-scroll top to hold")
	}

	// Manual scroll away.
	sb.SetScrollY(10)
	sb.SetContentHeight(60)
	if sb.ScrollY() == 0 {
		t.Error("should not snap to top after manual scroll")
	}

	// Scroll back to top.
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

// ---------------------------------------------------------------------------
// Keyboard navigation
// ---------------------------------------------------------------------------

func TestScrollBox_KeyNav_UpDown(t *testing.T) {
	sb := newTestScrollBox(20, 10, 20, 50)

	// Scroll down
	if !pressKey(sb, input.KeyDown) {
		t.Error("expected down key handled")
	}
	if sb.ScrollY() == 0 {
		t.Error("expected scrollY > 0 after down")
	}

	yAfterDown := sb.ScrollY()

	// Scroll up
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

	// Move somewhere in the middle.
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
	// Viewport larger than content — nothing to scroll.
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

	// Initially no manual scroll.
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

// ---------------------------------------------------------------------------
// ScrollBoxWithChildren tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Anchor-based scroll tests
// ---------------------------------------------------------------------------

// newAnchorScrollBox creates a column scrollbox with fixed-height children,
// runs layout, and sets content height. Returns the scrollbox.
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

	// Run layout so children get Computed positions.
	ln := sb.LayoutNode()
	ln.Width = layout.Points(vpW)
	ln.Height = layout.Points(vpH)
	ln.ComputeLayout(vpW, vpH)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// Compute total content height.
	total := 0
	for _, h := range childHeights {
		total += h
	}
	sb.SetContentHeight(total)
	return sb
}

// Box_ChildEntry is a local helper to pass children around.
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
	// 3 children: 10px, 20px, 30px = 60px total in a 15px viewport.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Scroll to y=15 → child[1] starts at y=10, so offset = 5.
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

	// Scroll to y=3 → still in child[0], offset = 3.
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

	// Scroll to y=35 → child[2] starts at y=30, offset = 5.
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
	// Viewport 15px, children: 10, 20, 30 = 60px total.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Scroll to y=15 → anchored to child[1] at offset 5.
	sb.SetScrollY(15)
	if sb.ScrollY() != 15 {
		t.Fatalf("expected scrollY=15, got %d", sb.ScrollY())
	}

	// Simulate child[0] growing by 10px. In real usage the layout engine
	// would reposition child[1] from y=10 to y=20. We simulate by
	// manually updating the child's Computed.Y via layout recomputation.
	//
	// To do this properly we resize the first child and re-run layout.
	children := sb.Children()
	children[0].LayoutNode().Height = layout.Points(20) // was 10, now 20
	children[0].LayoutNode().MarkDirty()

	ln := sb.LayoutNode()
	ln.ComputeLayout(20, 15)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// Trigger content height recalculation (70 = 20+20+30).
	sb.SetContentHeight(70)

	// scrollY should follow: child[1] now at y=20 + offset 5 = 25.
	if sb.ScrollY() != 25 {
		t.Errorf("expected scrollY=25 after anchor restore, got %d", sb.ScrollY())
	}
}

func TestScrollBox_AnchorScroll_DoesNotAffectAutoScrollBottom(t *testing.T) {
	// When at auto-scroll bottom, anchor should not interfere.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// maxScrollY = 60 - 15 = 45. Auto-scroll bottom → scrollY = 45.
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom with auto-scroll")
	}

	// Add more content — should stay at bottom.
	sb.SetContentHeight(80) // maxScrollY = 80 - 15 = 65
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

	// Should not panic when updating anchor with no children.
	sb.SetScrollY(0)
	anchor := sb.Anchor()
	if anchor.Valid {
		t.Error("expected invalid anchor with no children")
	}
}

func TestScrollBox_AnchorScroll_ChildRemoved(t *testing.T) {
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Anchor to child[2].
	sb.SetScrollY(35)
	anchor := sb.Anchor()
	if anchor.ChildIndex != 2 {
		t.Fatalf("expected anchor child 2, got %d", anchor.ChildIndex)
	}

	// Remove the last child. Anchor index is now out of bounds.
	lastChild := sb.Children()[2]
	sb.Remove(lastChild.ID())

	// scrollYFromAnchor should clamp to the new last child.
	newY := sb.scrollYFromAnchor()
	if newY < 0 {
		t.Errorf("expected non-negative scrollY from anchor, got %d", newY)
	}
}

func TestScrollBox_AnchorScroll_WithOnPostLayout(t *testing.T) {
	// Simulates the full OnPostLayout flow: content grows, anchor restores.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Scroll up manually, anchoring to child[1].
	sb.SetScrollY(15)

	// Grow child[0] and re-layout.
	children := sb.Children()
	children[0].LayoutNode().Height = layout.Points(25) // was 10
	children[0].LayoutNode().MarkDirty()

	ln := sb.LayoutNode()
	ln.ComputeLayout(20, 15)
	sb.UpdateFromLayout()
	for _, child := range sb.Children() {
		child.UpdateFromLayout()
	}

	// Invoke OnPostLayout which should detect content height change
	// and restore from anchor.
	sb.OnPostLayout(0)

	// child[1] moved from y=10 to y=25. Expected scrollY = 25 + 5 = 30.
	if sb.ScrollY() != 30 {
		t.Errorf("expected scrollY=30 after OnPostLayout anchor restore, got %d", sb.ScrollY())
	}
}

func TestScrollBox_AnchorScroll_ScrollBackToBottomReengagesAutoScroll(t *testing.T) {
	// Scenario: user scrolls up (breaks auto-scroll), then scrolls back to
	// bottom. When new content arrives, auto-scroll should re-engage
	// and snap to the new bottom — the anchor must not prevent this.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	// maxScrollY = 60 - 15 = 45. Starts at auto-scroll bottom.

	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom initially")
	}

	// Scroll up — breaks auto-scroll.
	sb.SetScrollY(15)
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after scrolling up")
	}

	// Scroll back to bottom.
	sb.ScrollToBottom()
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after scrolling back to bottom")
	}
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom after ScrollToBottom")
	}

	// New content arrives — auto-scroll should snap to the new bottom.
	sb.SetContentHeight(80) // maxScrollY = 80 - 15 = 65
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom to re-engage, scrollY=%d maxScrollY=%d",
			sb.ScrollY(), sb.ContentHeight()-sb.ViewportHeight())
	}
}

func TestScrollBox_AnchorScroll_HandleScrollBackToBottomReengagesAutoScroll(t *testing.T) {
	// Same scenario but using handleScroll (mouse wheel) instead of
	// ScrollToBottom, which is the real-world code path.
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)
	// maxScrollY = 60 - 15 = 45.

	// Scroll up via handleScroll (simulates mouse wheel up).
	sb.handleScroll(0, -30) // scrollY = 45 - 30 = 15
	if sb.ScrollY() != 15 {
		t.Fatalf("expected scrollY=15 after scroll up, got %d", sb.ScrollY())
	}
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after mouse scroll up")
	}

	// Scroll back to bottom via handleScroll (simulates mouse wheel down).
	sb.handleScroll(0, 100) // clamped to maxScrollY = 45
	if !sb.IsAtBottom() {
		t.Fatalf("expected at bottom after mouse scroll down, scrollY=%d", sb.ScrollY())
	}
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after scrolling back to bottom via handleScroll")
	}

	// New content arrives — auto-scroll should snap to the new bottom.
	sb.SetContentHeight(80) // maxScrollY = 80 - 15 = 65
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom after content growth, scrollY=%d maxScrollY=%d",
			sb.ScrollY(), sb.ContentHeight()-sb.ViewportHeight())
	}
}

func TestScrollBox_AnchorScroll_KeyNavBackToBottomReengagesAutoScroll(t *testing.T) {
	// Same scenario using HandleKeyPress (keyboard End key).
	sb := newAnchorScrollBox(20, 15, 10, 20, 30)

	// Scroll up via keyboard.
	pressKey(sb, input.KeyPageUp)
	if !sb.UserScrolled() {
		t.Fatal("expected userScrolled after PageUp")
	}

	// Press End to go to bottom.
	pressKey(sb, input.KeyEnd)
	if !sb.IsAtBottom() {
		t.Fatal("expected at bottom after End key")
	}
	if sb.UserScrolled() {
		t.Errorf("expected userScrolled cleared after End key to bottom")
	}

	// New content arrives — auto-scroll should snap to the new bottom.
	sb.SetContentHeight(80)
	if !sb.IsAtBottom() {
		t.Errorf("expected auto-scroll bottom after content growth via key nav, scrollY=%d",
			sb.ScrollY())
	}
}

// ---------------------------------------------------------------------------
// Sticky position tests
// ---------------------------------------------------------------------------

// newStickyScrollBox creates a column scrollbox with children of given heights,
// where specific children are marked StickyTop. stickyIndices lists child
// indices to mark sticky. Returns the scrollbox after layout + content height.
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
	// Viewport 20px. Children: 10, 30, 10 = 50px. Child[0] is sticky.
	// At scrollY=0, child[0] is at the top — no offset needed.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(0)
	child := sb.Children()[0]
	if child.StickyOffsetY() != 0 {
		t.Errorf("expected stickyOffsetY=0 when not scrolled past, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_PinsAtTop(t *testing.T) {
	// Viewport 20px. Children: 10, 30, 10. Child[0] is sticky.
	// Scroll to 5 — child[0] (at y=0) would be at viewport y=-5, pin at 0.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(5)
	child := sb.Children()[0]
	if child.StickyOffsetY() != 5 {
		t.Errorf("expected stickyOffsetY=5, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_SectionScoped_PushedByNext(t *testing.T) {
	// Children: 10, 30, 10, 20, 10. Child[0] and child[2] are sticky.
	// maxOffset for child[0] = 40 - 0 - 10 = 30 (no gap by default).
	// At scrollY=35, rawOffset=35 but clamped to 30.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})

	sb.SetScrollY(35)
	child0 := sb.Children()[0]
	child2 := sb.Children()[2]

	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30, got %d", child0.StickyOffsetY())
	}

	// child[2] at y=40: rawOffset = 35-40 = -5 → clamped to 0
	if child2.StickyOffsetY() != 0 {
		t.Errorf("expected child[2] stickyOffsetY=0 (not yet scrolled past), got %d", child2.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_SecondTakesOver(t *testing.T) {
	// Same layout. Scroll far enough that child[2] pins.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})

	sb.SetScrollY(50)
	child0 := sb.Children()[0]
	child2 := sb.Children()[2]

	// child[0]: maxOffset = 30, rawOffset = 50 → clamped to 30
	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30, got %d", child0.StickyOffsetY())
	}

	// child[2] at y=40: rawOffset = 50-40 = 10
	if child2.StickyOffsetY() != 10 {
		t.Errorf("expected child[2] stickyOffsetY=10, got %d", child2.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_NonStickyChildrenHaveZeroOffset(t *testing.T) {
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, []int{0})

	sb.SetScrollY(15)
	// child[1] is not sticky — should have 0 offset
	child1 := sb.Children()[1]
	if child1.StickyOffsetY() != 0 {
		t.Errorf("expected non-sticky child stickyOffsetY=0, got %d", child1.StickyOffsetY())
	}
}

func TestScrollBox_Sticky_NoStickyChildren(t *testing.T) {
	// No sticky children — should not panic.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, nil)
	sb.SetScrollY(15)
	for i, child := range sb.Children() {
		if child.StickyOffsetY() != 0 {
			t.Errorf("child[%d] expected stickyOffsetY=0 with no sticky children, got %d", i, child.StickyOffsetY())
		}
	}
}

func TestScrollBox_StickyBottom_PinsAtBottom(t *testing.T) {
	// Viewport 20px. Children: 10, 30, 10 = 50. Child[2] is sticky-bottom.
	// Child[2] at y=40, height=10. At scrollY=0, viewport bottom=20.
	// Child[2] bottom (50) > viewport bottom (20) → pin at bottom.
	// raw = (0+20) - (40+10) = -30. Offset = -30.
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
	// Scroll far enough that child[2] is fully in the viewport.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10}, nil)
	sb.Children()[2].LayoutNode().Sticky = layout.StickyBottom
	sb.SetContentHeight(50)
	sb.SetScrollY(30) // viewport 30..50, child[2] at 40..50 → fully visible

	child := sb.Children()[2]
	if child.StickyOffsetY() != 0 {
		t.Errorf("expected stickyOffsetY=0 when fully visible, got %d", child.StickyOffsetY())
	}
}

func TestScrollBox_StickyLeft_PinsAtLeft(t *testing.T) {
	// Horizontal scroll. Row layout, 20px viewport width.
	// Children widths: 10, 30, 10 = 50. Child[0] is sticky-left.
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
	// Child[2] is sticky-right at x=40, width=10.
	// Viewport width=20. At scrollX=0, viewport right=20.
	// Child right (50) > viewport right (20) → pin.
	// raw = (0+20) - (40+10) = -30.
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
	// Children: 10, 30, 10, 20, 10. Child[0] and child[2] sticky.
	// With gap=2: maxOffset for child[0] = 40 - 0 - 10 - 2 = 28.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})
	sb.SetStickyGap(2)

	sb.SetScrollY(35)
	child0 := sb.Children()[0]

	if child0.StickyOffsetY() != 28 {
		t.Errorf("expected child[0] stickyOffsetY=28 (gap=2), got %d", child0.StickyOffsetY())
	}
}

func TestScrollBox_StickyGap_ZeroNoGap(t *testing.T) {
	// With gap=0, the two sticky children touch.
	sb := newStickyScrollBox(20, 20, []int{10, 30, 10, 20, 10}, []int{0, 2})
	sb.SetStickyGap(0)

	sb.SetScrollY(35)
	child0 := sb.Children()[0]

	// maxOffset = 40 - 0 - 10 - 0 = 30
	if child0.StickyOffsetY() != 30 {
		t.Errorf("expected child[0] stickyOffsetY=30 (gap=0), got %d", child0.StickyOffsetY())
	}
}
