package layout

import "testing"

// --- NewNode Tests ---

func TestNewNode_Defaults(t *testing.T) {
	n := NewNode()

	if n.FlexDirection != DirectionRow {
		t.Errorf("FlexDirection = %d, want DirectionRow", n.FlexDirection)
	}
	if n.FlexWrap != WrapNone {
		t.Errorf("FlexWrap = %d, want WrapNone", n.FlexWrap)
	}
	if n.JustifyContent != JustifyStart {
		t.Errorf("JustifyContent = %d, want JustifyStart", n.JustifyContent)
	}
	if n.AlignItems != AlignStretch {
		t.Errorf("AlignItems = %d, want AlignStretch", n.AlignItems)
	}
	if n.FlexGrow != 0 {
		t.Errorf("FlexGrow = %v, want 0", n.FlexGrow)
	}
	if n.FlexShrink != 1 {
		t.Errorf("FlexShrink = %v, want 1", n.FlexShrink)
	}
	if !n.FlexBasis.IsAuto() {
		t.Error("FlexBasis should be Auto")
	}
	if n.AlignSelf != AlignAuto {
		t.Errorf("AlignSelf = %d, want AlignAuto", n.AlignSelf)
	}
	if !n.Width.IsAuto() {
		t.Error("Width should be Auto")
	}
	if !n.Height.IsAuto() {
		t.Error("Height should be Auto")
	}
	if n.Position != PositionRelative {
		t.Errorf("Position = %d, want PositionRelative", n.Position)
	}
	if n.Overflow != OverflowVisible {
		t.Errorf("Overflow = %d, want OverflowVisible", n.Overflow)
	}
	if !n.IsDirty() {
		t.Error("new node should be dirty")
	}
}

// --- AddChild Tests ---

func TestAddChild(t *testing.T) {
	parent := NewNode()
	child := NewNode()

	parent.AddChild(child)

	if parent.ChildCount() != 1 {
		t.Errorf("ChildCount() = %d, want 1", parent.ChildCount())
	}
	if child.Parent != parent {
		t.Error("child.Parent should be parent")
	}
	if parent.Children[0] != child {
		t.Error("parent.Children[0] should be child")
	}
}

func TestAddChild_Multiple(t *testing.T) {
	parent := NewNode()
	c1 := NewNode()
	c2 := NewNode()
	c3 := NewNode()

	parent.AddChild(c1)
	parent.AddChild(c2)
	parent.AddChild(c3)

	if parent.ChildCount() != 3 {
		t.Errorf("ChildCount() = %d, want 3", parent.ChildCount())
	}
	if parent.Children[0] != c1 || parent.Children[1] != c2 || parent.Children[2] != c3 {
		t.Error("children should be in order c1, c2, c3")
	}
}

func TestAddChild_ReparentsFromOldParent(t *testing.T) {
	p1 := NewNode()
	p2 := NewNode()
	child := NewNode()

	p1.AddChild(child)
	if p1.ChildCount() != 1 {
		t.Fatalf("p1 should have 1 child, got %d", p1.ChildCount())
	}

	p2.AddChild(child) // should remove from p1
	if p1.ChildCount() != 0 {
		t.Errorf("p1 ChildCount() = %d, want 0 after reparent", p1.ChildCount())
	}
	if p2.ChildCount() != 1 {
		t.Errorf("p2 ChildCount() = %d, want 1", p2.ChildCount())
	}
	if child.Parent != p2 {
		t.Error("child.Parent should be p2")
	}
}

// --- AddChildAt Tests ---

func TestAddChildAt_Beginning(t *testing.T) {
	parent := NewNode()
	c1 := NewNode()
	c2 := NewNode()

	parent.AddChild(c1)
	parent.AddChildAt(c2, 0)

	if parent.Children[0] != c2 {
		t.Error("c2 should be at index 0")
	}
	if parent.Children[1] != c1 {
		t.Error("c1 should be at index 1")
	}
}

func TestAddChildAt_Middle(t *testing.T) {
	parent := NewNode()
	c1 := NewNode()
	c2 := NewNode()
	c3 := NewNode()

	parent.AddChild(c1)
	parent.AddChild(c3)
	parent.AddChildAt(c2, 1)

	if parent.Children[0] != c1 || parent.Children[1] != c2 || parent.Children[2] != c3 {
		t.Error("children should be in order c1, c2, c3")
	}
}

func TestAddChildAt_OutOfRange(t *testing.T) {
	parent := NewNode()
	child := NewNode()

	parent.AddChildAt(child, 99) // should append
	if parent.ChildCount() != 1 {
		t.Errorf("ChildCount() = %d, want 1", parent.ChildCount())
	}
}

// --- RemoveChild Tests ---

func TestRemoveChild(t *testing.T) {
	parent := NewNode()
	child := NewNode()

	parent.AddChild(child)
	parent.RemoveChild(child)

	if parent.ChildCount() != 0 {
		t.Errorf("ChildCount() = %d, want 0", parent.ChildCount())
	}
	if child.Parent != nil {
		t.Error("child.Parent should be nil after remove")
	}
}

func TestRemoveChild_NotFound(t *testing.T) {
	parent := NewNode()
	child := NewNode()
	other := NewNode()

	parent.AddChild(child)
	parent.RemoveChild(other) // should be a no-op

	if parent.ChildCount() != 1 {
		t.Errorf("ChildCount() = %d, want 1", parent.ChildCount())
	}
}

func TestRemoveChild_Middle(t *testing.T) {
	parent := NewNode()
	c1 := NewNode()
	c2 := NewNode()
	c3 := NewNode()

	parent.AddChild(c1)
	parent.AddChild(c2)
	parent.AddChild(c3)
	parent.RemoveChild(c2)

	if parent.ChildCount() != 2 {
		t.Errorf("ChildCount() = %d, want 2", parent.ChildCount())
	}
	if parent.Children[0] != c1 || parent.Children[1] != c3 {
		t.Error("remaining children should be c1, c3")
	}
}

// --- Dirty Propagation Tests ---

func TestMarkDirty_Propagates(t *testing.T) {
	root := NewNode()
	child := NewNode()
	grandchild := NewNode()

	root.AddChild(child)
	child.AddChild(grandchild)

	// Clear dirty flags
	root.dirty = false
	child.dirty = false
	grandchild.dirty = false

	grandchild.MarkDirty()

	if !grandchild.IsDirty() {
		t.Error("grandchild should be dirty")
	}
	if !child.IsDirty() {
		t.Error("child should be dirty (propagated from grandchild)")
	}
	if !root.IsDirty() {
		t.Error("root should be dirty (propagated from grandchild)")
	}
}

func TestMarkDirty_StopsAtAlreadyDirty(t *testing.T) {
	root := NewNode()
	child := NewNode()
	grandchild := NewNode()

	root.AddChild(child)
	child.AddChild(grandchild)

	// Only root is dirty
	root.dirty = true
	child.dirty = false
	grandchild.dirty = false

	// When child is marked dirty, it should stop at root since it's already dirty
	child.MarkDirty()

	if !child.IsDirty() {
		t.Error("child should be dirty")
	}
}

// --- Shorthand Setter Tests ---

func TestSetMargin(t *testing.T) {
	n := NewNode()
	n.dirty = false
	n.SetMargin(Points(5))

	assertEqual(t, n.MarginTop.Value, 5)
	assertEqual(t, n.MarginRight.Value, 5)
	assertEqual(t, n.MarginBottom.Value, 5)
	assertEqual(t, n.MarginLeft.Value, 5)
	if !n.IsDirty() {
		t.Error("SetMargin should mark dirty")
	}
}

func TestSetMarginXY(t *testing.T) {
	n := NewNode()
	n.SetMarginXY(Points(10), Points(5))

	assertEqual(t, n.MarginLeft.Value, 10)
	assertEqual(t, n.MarginRight.Value, 10)
	assertEqual(t, n.MarginTop.Value, 5)
	assertEqual(t, n.MarginBottom.Value, 5)
}

func TestSetPadding(t *testing.T) {
	n := NewNode()
	n.SetPadding(Points(3))

	assertEqual(t, n.PaddingTop.Value, 3)
	assertEqual(t, n.PaddingRight.Value, 3)
	assertEqual(t, n.PaddingBottom.Value, 3)
	assertEqual(t, n.PaddingLeft.Value, 3)
}

func TestSetPaddingXY(t *testing.T) {
	n := NewNode()
	n.SetPaddingXY(Points(8), Points(4))

	assertEqual(t, n.PaddingLeft.Value, 8)
	assertEqual(t, n.PaddingRight.Value, 8)
	assertEqual(t, n.PaddingTop.Value, 4)
	assertEqual(t, n.PaddingBottom.Value, 4)
}

func TestSetSize(t *testing.T) {
	n := NewNode()
	n.dirty = false
	n.SetSize(Points(100), Points(50))

	assertEqual(t, n.Width.Value, 100)
	assertEqual(t, n.Height.Value, 50)
	if !n.IsDirty() {
		t.Error("SetSize should mark dirty")
	}
}
