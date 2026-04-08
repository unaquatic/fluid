package layout

import "testing"

// --- Grow Distribution Tests ---

func TestGrow_ThreeEqualChildren(t *testing.T) {
	// Row with 3 equal-grow children -> each gets 1/3 width
	root := NewNode()
	root.SetSize(Points(90), Points(30))

	for range 3 {
		child := NewNode()
		child.FlexGrow = 1
		root.AddChild(child)
	}

	root.ComputeLayout(90, 30)

	for i, child := range root.Children {
		assertEqual(t, child.Computed.Width, 30)
		if i > 0 {
			prev := root.Children[i-1]
			if child.Computed.X <= prev.Computed.X {
				t.Errorf("child %d X (%d) should be after child %d X (%d)", i, child.Computed.X, i-1, prev.Computed.X)
			}
		}
	}
}

func TestGrow_UnequalWeights(t *testing.T) {
	// grow=1 and grow=2 -> 1/3 and 2/3
	root := NewNode()
	root.SetSize(Points(90), Points(30))

	c1 := NewNode()
	c1.FlexGrow = 1
	root.AddChild(c1)

	c2 := NewNode()
	c2.FlexGrow = 2
	root.AddChild(c2)

	root.ComputeLayout(90, 30)

	assertEqual(t, c1.Computed.Width, 30)
	assertEqual(t, c2.Computed.Width, 60)
}

func TestGrow_WithFixedChild(t *testing.T) {
	// Column: fixed 10 + grow=1 in 50-height container
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(30), Points(50))

	fixed := NewNode()
	fixed.Height = Points(10)
	root.AddChild(fixed)

	growChild := NewNode()
	growChild.FlexGrow = 1
	root.AddChild(growChild)

	root.ComputeLayout(30, 50)

	assertEqual(t, fixed.Computed.Height, 10)
	assertEqual(t, growChild.Computed.Height, 40)
}

func TestGrow_NoGrow(t *testing.T) {
	// All items have grow=0, should keep basis size
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	c1 := NewNode()
	c1.Width = Points(20)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(30)
	root.AddChild(c2)

	root.ComputeLayout(100, 30)

	assertEqual(t, c1.Computed.Width, 20)
	assertEqual(t, c2.Computed.Width, 30)
}

// --- Shrink Distribution Tests ---

func TestShrink_Overflow(t *testing.T) {
	// Two items of 60 in a 100 container -> should shrink
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	c1 := NewNode()
	c1.Width = Points(60)
	c1.FlexShrink = 1
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(60)
	c2.FlexShrink = 1
	root.AddChild(c2)

	root.ComputeLayout(100, 30)

	// Both should shrink equally
	assertEqual(t, c1.Computed.Width, 50)
	assertEqual(t, c2.Computed.Width, 50)
}

func TestShrink_DifferentWeights(t *testing.T) {
	// shrink=1 (basis 60) and shrink=2 (basis 60) in 100 container
	// overflow = 20, total weighted = 1*60 + 2*60 = 180
	// c1 shrinks by 20 * 60/180 = 6 (int), c2 shrinks by 20 * 120/180 = 13 (int)
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	c1 := NewNode()
	c1.Width = Points(60)
	c1.FlexShrink = 1
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(60)
	c2.FlexShrink = 2
	root.AddChild(c2)

	root.ComputeLayout(100, 30)

	// With int math: overflow=20, shrink 1:2, basis 60:60
	// c1 shrinks int(20*60/180)=6 -> 54, c2 shrinks 20-6=14 -> 46
	assertEqual(t, c1.Computed.Width, 54)
	assertEqual(t, c2.Computed.Width, 46)
}

// --- JustifyContent Tests ---

func TestJustify_Start(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifyStart

	child := NewNode()
	child.Width = Points(20)
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	assertEqual(t, child.Computed.X, 0)
}

func TestJustify_End(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifyEnd

	child := NewNode()
	child.Width = Points(20)
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	assertEqual(t, child.Computed.X, 80)
}

func TestJustify_Center(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifyCenter

	child := NewNode()
	child.Width = Points(20)
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	assertEqual(t, child.Computed.X, 40)
}

func TestJustify_SpaceBetween_ThreeItems(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifySpaceBetween

	for range 3 {
		child := NewNode()
		child.Width = Points(10)
		root.AddChild(child)
	}

	root.ComputeLayout(100, 30)

	// Items at: 0, 45, 90
	assertEqual(t, root.Children[0].Computed.X, 0)
	assertEqual(t, root.Children[1].Computed.X, 45)
	assertEqual(t, root.Children[2].Computed.X, 90)
}

func TestJustify_SpaceBetween_SingleItem(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifySpaceBetween

	child := NewNode()
	child.Width = Points(10)
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	// Single item with SpaceBetween behaves like Start
	assertEqual(t, child.Computed.X, 0)
}

func TestJustify_SpaceAround(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifySpaceAround

	for range 2 {
		child := NewNode()
		child.Width = Points(20)
		root.AddChild(child)
	}

	root.ComputeLayout(100, 30)

	// Free space = 60, per item = 30, half = 15
	// Item 0 at 15, Item 1 at 65
	assertEqual(t, root.Children[0].Computed.X, 15)
	assertEqual(t, root.Children[1].Computed.X, 65)
}

func TestJustify_SpaceEvenly(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.JustifyContent = JustifySpaceEvenly

	for range 2 {
		child := NewNode()
		child.Width = Points(20)
		root.AddChild(child)
	}

	root.ComputeLayout(100, 30)

	// Free space = 60, slots = 3 (N+1), gap = 20
	// Item 0 at 20, Item 1 at 60
	assertEqual(t, root.Children[0].Computed.X, 20)
	assertEqual(t, root.Children[1].Computed.X, 60)
}

// --- Align Items/Self Tests ---

func TestAlign_Start(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))
	root.AlignItems = AlignStart

	child := NewNode()
	child.SetSize(Points(20), Points(10))
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Y, 0)
}

func TestAlign_End(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))
	root.AlignItems = AlignEnd

	child := NewNode()
	child.SetSize(Points(20), Points(10))
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Y, 40)
}

func TestAlign_Center(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))
	root.AlignItems = AlignCenter

	child := NewNode()
	child.SetSize(Points(20), Points(10))
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Y, 20)
}

func TestAlign_Stretch(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))
	root.AlignItems = AlignStretch

	child := NewNode()
	child.Width = Points(20)
	// No explicit height -> should stretch
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Height, 50)
}

func TestAlignSelf_Override(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))
	root.AlignItems = AlignStart

	child := NewNode()
	child.SetSize(Points(20), Points(10))
	child.AlignSelf = AlignEnd // Override parent's AlignStart
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Y, 40)
}

// --- Flex Wrap Tests ---

func TestWrap_Basic(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(70), Points(100))
	root.FlexWrap = WrapWrap

	// 3 items of width 30 -> 2 fit on first line (30+30=60 <= 70), 1 wraps
	for range 3 {
		child := NewNode()
		child.SetSize(Points(30), Points(20))
		root.AddChild(child)
	}

	root.ComputeLayout(70, 100)

	// First line: items 0 and 1 at y=0
	assertEqual(t, root.Children[0].Computed.Y, 0)
	assertEqual(t, root.Children[1].Computed.Y, 0)
	// Second line: item 2 at y=20
	assertEqual(t, root.Children[2].Computed.Y, 20)
}

func TestWrap_Reverse(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(70), Points(100))
	root.FlexWrap = WrapReverse

	for range 3 {
		child := NewNode()
		child.SetSize(Points(30), Points(20))
		root.AddChild(child)
	}

	root.ComputeLayout(70, 100)

	// With WrapReverse, the second line should come first (lower Y)
	// Line with item 2 should be at y=0, line with items 0,1 at y=20
	assertEqual(t, root.Children[2].Computed.Y, 0)
	assertEqual(t, root.Children[0].Computed.Y, 20)
}

// --- Gap Tests ---

func TestGap_Row(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))
	root.Gap = 10

	for range 3 {
		child := NewNode()
		child.FlexGrow = 1
		root.AddChild(child)
	}

	root.ComputeLayout(100, 30)

	// Available for items = 100 - 2*10 = 80, each gets 80/3 = 26, remainder 2 to last
	assertEqual(t, root.Children[0].Computed.Width, 26)
	assertEqual(t, root.Children[1].Computed.Width, 26)
	assertEqual(t, root.Children[2].Computed.Width, 28)

	// Check positions include gap
	assertEqual(t, root.Children[0].Computed.X, 0)
	assertEqual(t, root.Children[1].Computed.X, 36) // 26 + 10
}

func TestGap_Column(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(30), Points(100))
	root.RowGap = 5

	for range 3 {
		child := NewNode()
		child.FlexGrow = 1
		root.AddChild(child)
	}

	root.ComputeLayout(30, 100)

	// Available for items = 100 - 2*5 = 90, each gets 30
	for _, child := range root.Children {
		assertEqual(t, child.Computed.Height, 30)
	}
}

// --- Direction Tests ---

func TestDirection_Column(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(30), Points(90))

	for range 3 {
		child := NewNode()
		child.FlexGrow = 1
		root.AddChild(child)
	}

	root.ComputeLayout(30, 90)

	for _, child := range root.Children {
		assertEqual(t, child.Computed.Height, 30)
	}
	// Y positions should be 0, 30, 60
	assertEqual(t, root.Children[0].Computed.Y, 0)
	assertEqual(t, root.Children[1].Computed.Y, 30)
	assertEqual(t, root.Children[2].Computed.Y, 60)
}

func TestDirection_RowReverse(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionRowReverse
	root.SetSize(Points(90), Points(30))

	c1 := NewNode()
	c1.Width = Points(30)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(30)
	root.AddChild(c2)

	c3 := NewNode()
	c3.Width = Points(30)
	root.AddChild(c3)

	root.ComputeLayout(90, 30)

	// Reversed order: c3 at 0, c2 at 30, c1 at 60
	assertEqual(t, c3.Computed.X, 0)
	assertEqual(t, c2.Computed.X, 30)
	assertEqual(t, c1.Computed.X, 60)
}

// --- Nested Container Tests ---

func TestNested_RowInsideColumn(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(100), Points(100))

	// Top row
	topRow := NewNode()
	topRow.FlexDirection = DirectionRow
	topRow.Height = Points(50)
	topRow.FlexGrow = 0
	root.AddChild(topRow)

	c1 := NewNode()
	c1.FlexGrow = 1
	topRow.AddChild(c1)

	c2 := NewNode()
	c2.FlexGrow = 1
	topRow.AddChild(c2)

	// Bottom section
	bottomRow := NewNode()
	bottomRow.FlexGrow = 1
	root.AddChild(bottomRow)

	root.ComputeLayout(100, 100)

	// Top row should be 100 wide, 50 tall
	assertEqual(t, topRow.Computed.Width, 100)
	assertEqual(t, topRow.Computed.Height, 50)

	// Children of top row should each be 50 wide
	assertEqual(t, c1.Computed.Width, 50)
	assertEqual(t, c2.Computed.Width, 50)

	// Bottom row should fill remaining space
	assertEqual(t, bottomRow.Computed.Height, 50)
	assertEqual(t, bottomRow.Computed.Y, 50)
}

func TestNested_DeepHierarchy(t *testing.T) {
	// root (column 100x100) -> row (50 tall) -> 3 equal children
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(100), Points(100))

	row := NewNode()
	row.FlexDirection = DirectionRow
	row.Height = Points(50)
	root.AddChild(row)

	for range 3 {
		child := NewNode()
		child.FlexGrow = 1
		row.AddChild(child)
	}

	root.ComputeLayout(100, 100)

	// 100/3 = 33, remainder 1 to last item
	assertEqual(t, row.Children[0].Computed.Width, 33)
	assertEqual(t, row.Children[1].Computed.Width, 33)
	assertEqual(t, row.Children[2].Computed.Width, 34)
	for _, child := range row.Children {
		assertEqual(t, child.Computed.Height, 50)
	}
}

// --- MeasureFunc Tests ---

// simpleMeasure returns a MeasureFunc that always reports (w, h) as intrinsic
// size, clamped by MeasureMode constraints.
func simpleMeasure(intrW, intrH int) MeasureFunc {
	return func(w int, wm MeasureMode, h int, hm MeasureMode) MeasureResult {
		rw, rh := intrW, intrH
		switch wm {
		case MeasureExactly:
			rw = w
		case MeasureAtMost:
			if rw > w {
				rw = w
			}
		}
		switch hm {
		case MeasureExactly:
			rh = h
		case MeasureAtMost:
			if rh > h {
				rh = h
			}
		}
		return MeasureResult{Width: rw, Height: rh}
	}
}

func TestMeasureFunc_ColumnAutoHeight(t *testing.T) {
	// Column container with a fixed child + a MeasureFunc child (Auto height).
	// The measured child should get its intrinsic height as flex basis.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	fixed := NewNode()
	fixed.Height = Points(10)
	root.AddChild(fixed)

	measured := NewNode()
	// Auto height (default), MeasureFunc returns 25 tall
	measured.Measure = simpleMeasure(80, 25)
	root.AddChild(measured)

	root.ComputeLayout(80, 100)

	assertEqual(t, fixed.Computed.Height, 10)
	assertEqual(t, measured.Computed.Height, 25)
	assertEqual(t, measured.Computed.Y, 10)
}

func TestMeasureFunc_RowAutoWidth(t *testing.T) {
	// Row container with a MeasureFunc child that returns intrinsic width.
	root := NewNode()
	root.FlexDirection = DirectionRow
	root.SetSize(Points(100), Points(30))

	measured := NewNode()
	measured.Measure = simpleMeasure(40, 30)
	root.AddChild(measured)

	fixed := NewNode()
	fixed.Width = Points(20)
	root.AddChild(fixed)

	root.ComputeLayout(100, 30)

	assertEqual(t, measured.Computed.Width, 40)
	assertEqual(t, fixed.Computed.Width, 20)
	assertEqual(t, fixed.Computed.X, 40)
}

func TestMeasureFunc_CrossAxisInColumn(t *testing.T) {
	// Column container with AlignStart. The measured child should get its
	// intrinsic width on the cross axis instead of 0.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.AlignItems = AlignStart
	root.SetSize(Points(100), Points(100))

	measured := NewNode()
	measured.Measure = simpleMeasure(50, 20)
	root.AddChild(measured)

	root.ComputeLayout(100, 100)

	assertEqual(t, measured.Computed.Height, 20)
	assertEqual(t, measured.Computed.Width, 50)
}

func TestMeasureFunc_ExplicitSizeOverridesMeasure(t *testing.T) {
	// If a node has explicit Height, MeasureFunc should NOT be consulted
	// for the main axis (height in column layout).
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	child := NewNode()
	child.Height = Points(30) // explicit
	child.Measure = simpleMeasure(80, 99)
	root.AddChild(child)

	root.ComputeLayout(80, 100)

	assertEqual(t, child.Computed.Height, 30)
}

func TestMeasureFunc_GrowOverridesMeasure(t *testing.T) {
	// With FlexGrow, the measured basis should be used but then grown.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	measured := NewNode()
	measured.FlexGrow = 1
	measured.Measure = simpleMeasure(80, 20)
	root.AddChild(measured)

	root.ComputeLayout(80, 100)

	// With grow=1 and no other children, it should fill the container
	assertEqual(t, measured.Computed.Height, 100)
}

func TestMeasureFunc_NilDoesNothing(t *testing.T) {
	// Without MeasureFunc, Auto height should default to 0 basis
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	child := NewNode()
	// No MeasureFunc, Auto height
	root.AddChild(child)

	root.ComputeLayout(80, 100)

	// With default shrink=1, basis=0, no grow -> height stays 0
	assertEqual(t, child.Computed.Height, 0)
}

func TestMeasureFunc_MultipleChildrenWithMeasure(t *testing.T) {
	// Column with three measured children, each reporting different heights
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(60), Points(200))

	heights := []int{10, 20, 30}
	for _, h := range heights {
		child := NewNode()
		child.Measure = simpleMeasure(60, h)
		root.AddChild(child)
	}

	root.ComputeLayout(60, 200)

	y := 0
	for i, child := range root.Children {
		assertEqual(t, child.Computed.Height, heights[i])
		assertEqual(t, child.Computed.Y, y)
		y += heights[i]
	}
}

func TestMeasureFunc_ModeExactlyRespected(t *testing.T) {
	// Verify that when the layout engine passes MeasureExactly,
	// the MeasureFunc returns that exact value.
	root := NewNode()
	root.FlexDirection = DirectionRow
	root.SetSize(Points(100), Points(30))

	var gotWidthMode, gotHeightMode MeasureMode
	child := NewNode()
	child.Width = Points(40)
	child.Measure = func(w int, wm MeasureMode, h int, hm MeasureMode) MeasureResult {
		gotWidthMode = wm
		gotHeightMode = hm
		return MeasureResult{Width: w, Height: h}
	}
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	// Width is explicit (Points) so MeasureFunc shouldn't be called for basis.
	assertEqual(t, child.Computed.Width, 40)
	_ = gotWidthMode
	_ = gotHeightMode
}

func TestColumnAutoHeight_ShrinkWrapsChildren(t *testing.T) {
	// A column container with Auto height inside a larger parent should
	// shrink-wrap its children's heights + gaps + padding instead of
	// expanding to fill the parent.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(200))

	// Container with Auto height, padding, and gap
	container := NewNode()
	container.FlexDirection = DirectionColumn
	container.PaddingTop = Points(1)
	container.PaddingBottom = Points(1)
	container.Gap = 1
	container.Width = Points(80)
	// Height is Auto (default)

	c1 := NewNode()
	c1.Height = Points(5)
	container.AddChild(c1)

	c2 := NewNode()
	c2.Height = Points(10)
	container.AddChild(c2)

	c3 := NewNode()
	c3.Height = Points(3)
	container.AddChild(c3)

	root.AddChild(container)

	root.ComputeLayout(80, 200)

	// Expected: children(5+10+3) + gaps(2) + padding(2) = 22
	assertEqual(t, container.Computed.Height, 22)
	// Children should be positioned correctly within the container
	assertEqual(t, c1.Computed.Height, 5)
	assertEqual(t, c2.Computed.Height, 10)
	assertEqual(t, c3.Computed.Height, 3)
}

func TestColumnAutoHeight_WithMeasureFuncChildren(t *testing.T) {
	// Container with Auto height whose children use MeasureFunc.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(200))

	container := NewNode()
	container.FlexDirection = DirectionColumn
	container.Width = Points(80)

	c1 := NewNode()
	c1.Measure = simpleMeasure(80, 15)
	container.AddChild(c1)

	c2 := NewNode()
	c2.Height = Points(5)
	container.AddChild(c2)

	root.AddChild(container)

	root.ComputeLayout(80, 200)

	// Expected: 15 + 5 = 20
	assertEqual(t, container.Computed.Height, 20)
}

func TestColumnAutoHeight_NestedContainers(t *testing.T) {
	// Nested containers both with Auto height should shrink-wrap recursively.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(200))

	outer := NewNode()
	outer.FlexDirection = DirectionColumn
	outer.PaddingTop = Points(1)
	outer.PaddingBottom = Points(1)
	outer.Width = Points(80)

	inner := NewNode()
	inner.FlexDirection = DirectionColumn
	inner.Width = Points(80)

	leaf := NewNode()
	leaf.Height = Points(10)
	inner.AddChild(leaf)

	outer.AddChild(inner)
	root.AddChild(outer)

	root.ComputeLayout(80, 200)

	// inner: child(10) = 10
	assertEqual(t, inner.Computed.Height, 10)
	// outer: inner(10) + padding(2) = 12
	assertEqual(t, outer.Computed.Height, 12)
}

func TestOverflowScroll_ChildrenDoNotShrink(t *testing.T) {
	// In a scroll container, children should keep their basis even when
	// total content exceeds the viewport. Scrolling handles the overflow.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(20)) // small viewport
	root.Overflow = OverflowScroll

	c1 := NewNode()
	c1.Height = Points(15)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Height = Points(15)
	root.AddChild(c2)

	root.ComputeLayout(80, 20)

	// Without scroll: children would shrink from 15 to 10 each (to fit 20).
	// With scroll: children keep their natural 15 height.
	assertEqual(t, c1.Computed.Height, 15)
	assertEqual(t, c2.Computed.Height, 15)
	// c2 positioned past the viewport -- that's fine, scroll handles it.
	assertEqual(t, c2.Computed.Y, 15)
}

func TestOverflowScroll_AutoHeightChildrenDoNotShrink(t *testing.T) {
	// Scroll container with Auto-height children (containers) that
	// shrink-wrap their own children. They should not be compressed.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(10)) // very small viewport
	root.Overflow = OverflowScroll

	// A box with Auto height containing fixed-height children
	box := NewNode()
	box.FlexDirection = DirectionColumn
	box.PaddingTop = Points(1)
	box.PaddingBottom = Points(1)
	box.Gap = 1

	leaf1 := NewNode()
	leaf1.Height = Points(5)
	box.AddChild(leaf1)

	leaf2 := NewNode()
	leaf2.Height = Points(3)
	box.AddChild(leaf2)

	root.AddChild(box)

	root.ComputeLayout(80, 10)

	// box intrinsic: children(5+3) + gap(1) + padding(2) = 11
	assertEqual(t, box.Computed.Height, 11)
	// Children inside the box should have their full heights
	assertEqual(t, leaf1.Computed.Height, 5)
	assertEqual(t, leaf2.Computed.Height, 3)
}

func TestMeasureFunc_AtMostClampsLargeIntrinsic(t *testing.T) {
	// MeasureFunc returns larger than available; AtMost should clamp.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(50), Points(30))

	child := NewNode()
	// Returns 100 height, but container only has 30 -- AtMost should clamp.
	child.Measure = simpleMeasure(50, 100)
	root.AddChild(child)

	root.ComputeLayout(50, 30)

	// The MeasureFunc is called with AtMost 30 on the height axis,
	// so simpleMeasure clamps to 30.
	assertEqual(t, child.Computed.Height, 30)
}

// wrapMeasure simulates a text widget that wraps: the returned height
// depends on the width. Given an intrinsic line width, it computes
// height = ceil(lineWidth / width).
func wrapMeasure(lineWidth int) MeasureFunc {
	return func(w int, wm MeasureMode, h int, hm MeasureMode) MeasureResult {
		rw := lineWidth
		switch wm {
		case MeasureExactly:
			rw = w
		case MeasureAtMost:
			if rw > w {
				rw = w
			}
		}
		// Compute wrapped height
		effectiveW := rw
		if effectiveW <= 0 {
			effectiveW = 1
		}
		rh := (lineWidth + effectiveW - 1) / effectiveW
		switch hm {
		case MeasureExactly:
			rh = h
		case MeasureAtMost:
			if rh > h {
				rh = h
			}
		}
		return MeasureResult{Width: rw, Height: rh}
	}
}

func TestIntrinsicMainSize_RowInColumn_WidthDependentHeight(t *testing.T) {
	// A Row inside a Column with a MeasureFunc child whose height depends
	// on available width (simulating word-wrapping text). The layout
	// should propagate the parent's cross dimension (width) so the
	// MeasureFunc receives a real width instead of MaxInt16.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(40), Points(100))

	row := NewNode()
	row.FlexDirection = DirectionRow

	// Prompt: fixed 4 columns wide
	prompt := NewNode()
	prompt.Width = Points(4)
	prompt.Height = Points(1)
	row.AddChild(prompt)

	// Textarea: width 100%, wrapping content of 80 display-width chars.
	// At width 36 (40 - 4), ceil(80/36) = 3 lines.
	ta := NewNode()
	ta.Width = Percent(100)
	ta.Measure = wrapMeasure(80)
	row.AddChild(ta)

	root.AddChild(row)
	root.ComputeLayout(40, 100)

	// The row should shrink-wrap to the tallest child's height.
	// Textarea basis is 100% of 40 = 40, but prompt takes 4, so after
	// flex shrink the textarea gets 36 columns.
	// Wrapping 80 chars at width 36: ceil(80/36) = 3 lines.
	assertEqual(t, ta.Computed.Width, 36)
	assertEqual(t, ta.Computed.Height, 3)
	assertEqual(t, row.Computed.Height, 3)
}

func TestIntrinsicMainSize_RowInColumn_CrossMargins(t *testing.T) {
	// Reproduces a bug where a row container with cross-axis margins
	// (MarginLeft/MarginRight in a column parent) would measure its
	// children at the parent's full content width instead of the
	// margin-reduced width. This caused word-wrapping textareas inside
	// such rows to wrap too late (by the sum of the cross margins).
	//
	// Layout:
	//   root (column, 80x100, padding-left: 1, padding-right: 1)
	//     wrapper (row, margin-left: 1, margin-right: 2)
	//       accent (width: 1)
	//       box (column, flex-grow: 1, padding-left: 2)
	//         inner-row (row, flex-grow: 1)
	//           textarea (flex-grow: 1, MeasureFunc)
	//
	// Content = 80 chars of text.
	// Textarea width = 80 - 1 - 1 (root pad) - 1 - 2 (wrapper margin)
	//                    - 1 (accent) - 2 (box pad) = 72
	// Lines = ceil(80/72) = 2
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))
	root.PaddingLeft = Points(1)
	root.PaddingRight = Points(1)

	wrapper := NewNode()
	wrapper.FlexDirection = DirectionRow
	wrapper.FlexShrink = 0
	wrapper.MarginLeft = Points(1)
	wrapper.MarginRight = Points(2)
	wrapper.AlignItems = AlignStretch

	accent := NewNode()
	accent.Width = Points(1)
	wrapper.AddChild(accent)

	box := NewNode()
	box.FlexGrow = 1
	box.FlexDirection = DirectionColumn
	box.AlignItems = AlignStretch
	box.PaddingLeft = Points(2)
	box.PaddingTop = Points(1)
	box.PaddingBottom = Points(1)

	innerRow := NewNode()
	innerRow.FlexDirection = DirectionRow
	innerRow.FlexGrow = 1
	innerRow.AlignItems = AlignStretch

	textarea := NewNode()
	textarea.FlexGrow = 1
	textarea.Measure = wrapMeasure(80)
	innerRow.AddChild(textarea)

	box.AddChild(innerRow)
	wrapper.AddChild(box)
	root.AddChild(wrapper)
	root.ComputeLayout(80, 100)

	// Textarea width should be: 80 - 1(root padL) - 1(root padR)
	//   - 1(marginL) - 2(marginR) - 1(accent) - 2(box padL) = 72
	assertEqual(t, textarea.Computed.Width, 72)
	// Wrapping 80 chars at width 72: ceil(80/72) = 2 lines
	assertEqual(t, textarea.Computed.Height, 2)
	// The wrapper should shrink-wrap to box height: padTop(1) + content(2) + padBottom(1) = 4
	assertEqual(t, wrapper.Computed.Height, 4)
}
