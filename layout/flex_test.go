package layout

import "testing"

func TestGrow_ThreeEqualChildren(t *testing.T) {
	// Three equal growth factors divide the row width into thirds.
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
	// Growth factors one and two divide free space in a one-to-two ratio.
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
	// The growing child receives the height left by the fixed child.
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
	// Items with zero growth retain their basis sizes when free space remains.
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

func TestShrink_Overflow(t *testing.T) {
	// Equal shrink factors divide a 20-column deficit evenly.
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

	assertEqual(t, c1.Computed.Width, 50)
	assertEqual(t, c2.Computed.Width, 50)
}

func TestShrink_DifferentWeights(t *testing.T) {
	// With equal bases, shrink factors one and two split the 20-column deficit
	// proportionally before integer remainder is assigned.
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

	// Integer shares remove 6 columns from c1 and the remaining 14 from c2.
	assertEqual(t, c1.Computed.Width, 54)
	assertEqual(t, c2.Computed.Width, 46)
}

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

	// Seventy free columns form two 35-column spaces between the items.
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

	// SpaceBetween places a single item at the start.
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

	// Sixty free columns give each item 30 columns of surrounding space and a
	// 15-column outer offset.
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

	// Sixty free columns divide into three equal spaces around two items.
	assertEqual(t, root.Children[0].Computed.X, 20)
	assertEqual(t, root.Children[1].Computed.X, 60)
}

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
	// An automatic height stretches across the row's cross axis.
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
	child.AlignSelf = AlignEnd // Override the parent's AlignStart value.
	root.AddChild(child)

	root.ComputeLayout(100, 50)

	assertEqual(t, child.Computed.Y, 40)
}

func TestWrap_Basic(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(70), Points(100))
	root.FlexWrap = WrapWrap

	// Two 30-column items fit within 70 columns; the third starts a new line.
	for range 3 {
		child := NewNode()
		child.SetSize(Points(30), Points(20))
		root.AddChild(child)
	}

	root.ComputeLayout(70, 100)

	// The first two items share the first line.
	assertEqual(t, root.Children[0].Computed.Y, 0)
	assertEqual(t, root.Children[1].Computed.Y, 0)
	// The third item begins after the first line's 20-row cross size.
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

	// WrapReverse places the final line before the line containing the first two items.
	assertEqual(t, root.Children[2].Computed.Y, 0)
	assertEqual(t, root.Children[0].Computed.Y, 20)
}

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

	// Two ten-column gaps leave 80 columns; integer growth assigns the two-column
	// remainder to the final item.
	assertEqual(t, root.Children[0].Computed.Width, 26)
	assertEqual(t, root.Children[1].Computed.Width, 26)
	assertEqual(t, root.Children[2].Computed.Width, 28)

	// Item positions include the configured gap.
	assertEqual(t, root.Children[0].Computed.X, 0)
	assertEqual(t, root.Children[1].Computed.X, 36) // First width plus one gap.
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

	// Two five-row gaps leave 90 rows for three equal items.
	for _, child := range root.Children {
		assertEqual(t, child.Computed.Height, 30)
	}
}

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
	// Column direction advances each item along y.
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

	// RowReverse lays out source items from last to first.
	assertEqual(t, c3.Computed.X, 0)
	assertEqual(t, c2.Computed.X, 30)
	assertEqual(t, c1.Computed.X, 60)
}

func TestNested_RowInsideColumn(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(100), Points(100))

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

	bottomRow := NewNode()
	bottomRow.FlexGrow = 1
	root.AddChild(bottomRow)

	root.ComputeLayout(100, 100)

	// The fixed top row spans the root width.
	assertEqual(t, topRow.Computed.Width, 100)
	assertEqual(t, topRow.Computed.Height, 50)

	// Equal growth factors split the top row in half.
	assertEqual(t, c1.Computed.Width, 50)
	assertEqual(t, c2.Computed.Width, 50)

	// The bottom row receives the remaining root height.
	assertEqual(t, bottomRow.Computed.Height, 50)
	assertEqual(t, bottomRow.Computed.Y, 50)
}

func TestNested_DeepHierarchy(t *testing.T) {
	// A 50-row nested row divides its 100 columns among three equal children.
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

	// Integer division assigns the remaining column to the final child.
	assertEqual(t, row.Children[0].Computed.Width, 33)
	assertEqual(t, row.Children[1].Computed.Width, 33)
	assertEqual(t, row.Children[2].Computed.Width, 34)
	for _, child := range row.Children {
		assertEqual(t, child.Computed.Height, 50)
	}
}

// simpleMeasure returns a measurement function with the given intrinsic size,
// constrained according to each MeasureMode.
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
	// An automatically sized child uses its measured height as its column basis.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	fixed := NewNode()
	fixed.Height = Points(10)
	root.AddChild(fixed)

	measured := NewNode()
	// The default automatic height measures as 25 rows.
	measured.Measure = simpleMeasure(80, 25)
	root.AddChild(measured)

	root.ComputeLayout(80, 100)

	assertEqual(t, fixed.Computed.Height, 10)
	assertEqual(t, measured.Computed.Height, 25)
	assertEqual(t, measured.Computed.Y, 10)
}

func TestMeasureFunc_RowAutoWidth(t *testing.T) {
	// An automatically sized child uses its measured width as its row basis.
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
	// A non-stretched child uses its measured cross-axis width instead of zero.
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
	// An explicit height takes precedence over intrinsic main-axis measurement.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	child := NewNode()
	child.Height = Points(30) // Override the measured height of 99.
	child.Measure = simpleMeasure(80, 99)
	root.AddChild(child)

	root.ComputeLayout(80, 100)

	assertEqual(t, child.Computed.Height, 30)
}

func TestMeasureFunc_GrowOverridesMeasure(t *testing.T) {
	// FlexGrow expands the measured basis with the remaining free space.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	measured := NewNode()
	measured.FlexGrow = 1
	measured.Measure = simpleMeasure(80, 20)
	root.AddChild(measured)

	root.ComputeLayout(80, 100)

	// As the only growing child, it fills the container.
	assertEqual(t, measured.Computed.Height, 100)
}

func TestMeasureFunc_NilDoesNothing(t *testing.T) {
	// An automatic height without measurement has a zero basis.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(100))

	child := NewNode()
	root.AddChild(child)

	root.ComputeLayout(80, 100)

	assertEqual(t, child.Computed.Height, 0)
}

func TestMeasureFunc_MultipleChildrenWithMeasure(t *testing.T) {
	// Measured children stack using their distinct intrinsic heights.
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
	// An explicit width bypasses intrinsic main-axis sizing and remains exact
	// during any cross-axis measurement.
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

	assertEqual(t, child.Computed.Width, 40)
	_ = gotWidthMode
	_ = gotHeightMode
}

func TestColumnAutoHeight_ShrinkWrapsChildren(t *testing.T) {
	// An automatic nested column shrink-wraps its children, gaps, and padding
	// instead of filling its parent.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(200))

	// The container has automatic height, two rows of padding, and two gaps.
	container := NewNode()
	container.FlexDirection = DirectionColumn
	container.PaddingTop = Points(1)
	container.PaddingBottom = Points(1)
	container.Gap = 1
	container.Width = Points(80)

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

	// Child heights 18, gaps 2, and padding 2 produce 22 rows.
	assertEqual(t, container.Computed.Height, 22)
	assertEqual(t, c1.Computed.Height, 5)
	assertEqual(t, c2.Computed.Height, 10)
	assertEqual(t, c3.Computed.Height, 3)
}

func TestColumnAutoHeight_WithMeasureFuncChildren(t *testing.T) {
	// An automatic container includes measured child heights in its intrinsic size.
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

	// The measured 15 rows and fixed 5 rows produce a 20-row container.
	assertEqual(t, container.Computed.Height, 20)
}

func TestColumnAutoHeight_NestedContainers(t *testing.T) {
	// Nested automatic containers shrink-wrap recursively.
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

	// The inner container matches its ten-row child.
	assertEqual(t, inner.Computed.Height, 10)
	// The outer container adds two rows of padding.
	assertEqual(t, outer.Computed.Height, 12)
}

func TestOverflowScroll_ChildrenDoNotShrink(t *testing.T) {
	// A scroll container preserves child bases when content exceeds the viewport.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(20)) // The two children exceed this viewport.
	root.Overflow = OverflowScroll

	c1 := NewNode()
	c1.Height = Points(15)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Height = Points(15)
	root.AddChild(c2)

	root.ComputeLayout(80, 20)

	// Scrolling keeps both 15-row bases instead of shrinking each to ten rows.
	assertEqual(t, c1.Computed.Height, 15)
	assertEqual(t, c2.Computed.Height, 15)
	// The second child can extend beyond the viewport for scrolling.
	assertEqual(t, c2.Computed.Y, 15)
}

func TestOverflowScroll_AutoHeightChildrenDoNotShrink(t *testing.T) {
	// A scroll container preserves an automatic child's shrink-wrapped height.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(10)) // The child is taller than this viewport.
	root.Overflow = OverflowScroll

	// The automatic box includes its children, gap, and vertical padding.
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

	// Child heights 8, one gap, and two rows of padding produce 11 rows.
	assertEqual(t, box.Computed.Height, 11)
	assertEqual(t, leaf1.Computed.Height, 5)
	assertEqual(t, leaf2.Computed.Height, 3)
}

func TestMeasureFunc_AtMostClampsLargeIntrinsic(t *testing.T) {
	// MeasureAtMost clamps an intrinsic height larger than the available space.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(50), Points(30))

	child := NewNode()
	// The intrinsic 100-row height exceeds the 30-row limit.
	child.Measure = simpleMeasure(50, 100)
	root.AddChild(child)

	root.ComputeLayout(50, 30)

	// simpleMeasure applies the 30-row MeasureAtMost constraint.
	assertEqual(t, child.Computed.Height, 30)
}

// wrapMeasure returns a measurement function whose height is the ceiling of
// lineWidth divided by its constrained width.
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
		// Avoid division by zero when the resolved width is empty.
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
	// A row inside a column propagates its constrained width when measuring a
	// child whose intrinsic height depends on wrapping.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(40), Points(100))

	row := NewNode()
	row.FlexDirection = DirectionRow

	// A fixed four-column prompt precedes the measured text area.
	prompt := NewNode()
	prompt.Width = Points(4)
	prompt.Height = Points(1)
	row.AddChild(prompt)

	// The text area wraps 80 columns of content within the remaining width.
	ta := NewNode()
	ta.Width = Percent(100)
	ta.Measure = wrapMeasure(80)
	row.AddChild(ta)

	root.AddChild(row)
	root.ComputeLayout(40, 100)

	// Flex shrink leaves the text area 36 columns. Wrapping 80 columns then
	// requires three rows, which sets the row's shrink-wrapped height.
	assertEqual(t, ta.Computed.Width, 36)
	assertEqual(t, ta.Computed.Height, 3)
	assertEqual(t, row.Computed.Height, 3)
}

func TestIntrinsicMainSize_RowInColumn_CrossMargins(t *testing.T) {
	// Cross-axis margins on a row inside a column reduce the width passed to
	// wrapping descendants. Ignoring them would make text wrap too late.
	//
	// The text area receives 72 columns after root padding, wrapper margins, a
	// one-column accent, and box padding. Its 80 columns of content require two
	// wrapped rows.
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

	// 80 - 2 root padding - 3 wrapper margin - 1 accent - 2 box padding = 72.
	assertEqual(t, textarea.Computed.Width, 72)
	// Wrapping 80 columns at width 72 requires two rows.
	assertEqual(t, textarea.Computed.Height, 2)
	// The wrapper shrink-wraps the box's two content rows and two padding rows.
	assertEqual(t, wrapper.Computed.Height, 4)
}
