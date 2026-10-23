package layout

import "testing"

func TestComputeLayout_SimpleRow(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(50))

	c1 := NewNode()
	c1.Width = Points(40)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(30)
	root.AddChild(c2)

	root.ComputeLayout(100, 50)

	assertEqual(t, root.Computed.X, 0)
	assertEqual(t, root.Computed.Y, 0)
	assertEqual(t, root.Computed.Width, 100)
	assertEqual(t, root.Computed.Height, 50)

	assertEqual(t, c1.Computed.X, 0)
	assertEqual(t, c1.Computed.Width, 40)

	assertEqual(t, c2.Computed.X, 40)
	assertEqual(t, c2.Computed.Width, 30)
}

func TestComputeLayout_SimpleColumn(t *testing.T) {
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(50), Points(100))

	c1 := NewNode()
	c1.Height = Points(30)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Height = Points(20)
	root.AddChild(c2)

	root.ComputeLayout(50, 100)

	assertEqual(t, c1.Computed.Y, 0)
	assertEqual(t, c1.Computed.Height, 30)

	assertEqual(t, c2.Computed.Y, 30)
	assertEqual(t, c2.Computed.Height, 20)
}

func TestComputeLayout_PercentageSizing(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(200), Points(100))

	child := NewNode()
	child.Width = Percent(50)
	child.Height = Percent(75)
	root.AddChild(child)

	root.ComputeLayout(200, 100)

	assertEqual(t, child.Computed.Width, 100)
	assertEqual(t, child.Computed.Height, 75)
}

func TestComputeLayout_MinWidth(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	c1 := NewNode()
	c1.Width = Points(80)
	c1.FlexShrink = 1
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(80)
	c2.FlexShrink = 1
	c2.MinWidth = Points(60)
	root.AddChild(c2)

	root.ComputeLayout(100, 30)

	// Flexible shrinkage must respect c2's minimum width.
	if c2.Computed.Width < 60 {
		t.Errorf("c2.Width = %d, should not be below MinWidth 60", c2.Computed.Width)
	}
}

func TestComputeLayout_MaxWidth(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	child := NewNode()
	child.FlexGrow = 1
	child.MaxWidth = Points(50)
	root.AddChild(child)

	root.ComputeLayout(100, 30)

	assertEqual(t, child.Computed.Width, 50)
}

func TestComputeLayout_Padding(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))
	root.SetPadding(Points(10))

	child := NewNode()
	child.FlexGrow = 1
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	// Padding reduces the child's available box on every side.
	assertEqual(t, child.Computed.X, 10)
	assertEqual(t, child.Computed.Y, 10)
	assertEqual(t, child.Computed.Width, 80)
	assertEqual(t, child.Computed.Height, 80)
}

func TestComputeLayout_Margin(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	child := NewNode()
	child.Width = Points(60)
	child.SetMargin(Points(10))
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	assertEqual(t, child.Computed.X, 10) // The left margin offsets the child.
	assertEqual(t, child.Computed.Width, 60)
}

func TestComputeLayout_AbsoluteTopLeft(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	child := NewNode()
	child.Position = PositionAbsolute
	child.Top = Points(10)
	child.Left = Points(20)
	child.SetSize(Points(30), Points(30))
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	assertEqual(t, child.Computed.X, 20)
	assertEqual(t, child.Computed.Y, 10)
	assertEqual(t, child.Computed.Width, 30)
	assertEqual(t, child.Computed.Height, 30)
}

func TestComputeLayout_AbsoluteBottomRight(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	child := NewNode()
	child.Position = PositionAbsolute
	child.Bottom = Points(10)
	child.Right = Points(20)
	child.SetSize(Points(30), Points(30))
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	assertEqual(t, child.Computed.X, 50) // 100 width - 20 right - 30 child width.
	assertEqual(t, child.Computed.Y, 60) // 100 height - 10 bottom - 30 child height.
}

func TestComputeLayout_AbsoluteLeftRight_DeriveWidth(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	child := NewNode()
	child.Position = PositionAbsolute
	child.Left = Points(10)
	child.Right = Points(20)
	child.Top = Points(5)
	child.Height = Points(30)
	// Opposing horizontal offsets derive the automatic width: 100 - 10 - 20 = 70.
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	assertEqual(t, child.Computed.X, 10)
	assertEqual(t, child.Computed.Width, 70)
}

func TestComputeLayout_AbsoluteDoesNotAffectFlex(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	// An absolute child is excluded from flex flow.
	abs := NewNode()
	abs.Position = PositionAbsolute
	abs.Top = Points(0)
	abs.Left = Points(0)
	abs.SetSize(Points(50), Points(50))
	root.AddChild(abs)

	flex := NewNode()
	flex.FlexGrow = 1
	root.AddChild(flex)

	root.ComputeLayout(100, 100)

	// The flex child therefore receives the full width.
	assertEqual(t, flex.Computed.Width, 100)
}

func TestComputeLayout_ClearsDirty(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))

	child := NewNode()
	child.FlexGrow = 1
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	if root.IsDirty() {
		t.Error("root should not be dirty after ComputeLayout")
	}
	if child.IsDirty() {
		t.Error("child should not be dirty after ComputeLayout")
	}
}

func TestComputeLayout_ComplexNested(t *testing.T) {
	// Model an 80 by 24 column containing a one-row header, a growing body
	// split into a 20-column sidebar and growing content, and a one-row footer.
	root := NewNode()
	root.FlexDirection = DirectionColumn
	root.SetSize(Points(80), Points(24))

	header := NewNode()
	header.Height = Points(1)
	root.AddChild(header)

	body := NewNode()
	body.FlexDirection = DirectionRow
	body.FlexGrow = 1
	root.AddChild(body)

	sidebar := NewNode()
	sidebar.Width = Points(20)
	body.AddChild(sidebar)

	content := NewNode()
	content.FlexGrow = 1
	body.AddChild(content)

	footer := NewNode()
	footer.Height = Points(1)
	root.AddChild(footer)

	root.ComputeLayout(80, 24)

	// The header occupies the first row.
	assertEqual(t, header.Computed.X, 0)
	assertEqual(t, header.Computed.Y, 0)
	assertEqual(t, header.Computed.Width, 80)
	assertEqual(t, header.Computed.Height, 1)

	// The body receives the 22 rows between header and footer.
	assertEqual(t, body.Computed.X, 0)
	assertEqual(t, body.Computed.Y, 1)
	assertEqual(t, body.Computed.Width, 80)
	assertEqual(t, body.Computed.Height, 22)

	// The sidebar occupies the first 20 body columns.
	assertEqual(t, sidebar.Computed.X, 0)
	assertEqual(t, sidebar.Computed.Y, 1)
	assertEqual(t, sidebar.Computed.Width, 20)
	assertEqual(t, sidebar.Computed.Height, 22)

	// The content receives the remaining 60 body columns.
	assertEqual(t, content.Computed.X, 20)
	assertEqual(t, content.Computed.Y, 1)
	assertEqual(t, content.Computed.Width, 60)
	assertEqual(t, content.Computed.Height, 22)

	// The footer occupies the final row.
	assertEqual(t, footer.Computed.X, 0)
	assertEqual(t, footer.Computed.Y, 23)
	assertEqual(t, footer.Computed.Width, 80)
	assertEqual(t, footer.Computed.Height, 1)
}

func TestComputeLayout_ZeroSizeContainer(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(0), Points(0))

	child := NewNode()
	child.FlexGrow = 1
	root.AddChild(child)

	root.ComputeLayout(0, 0)

	assertEqual(t, child.Computed.Width, 0)
	assertEqual(t, child.Computed.Height, 0)
}

func TestComputeLayout_AutoSizeRoot(t *testing.T) {
	root := NewNode()
	// An automatic root fills the available viewport.

	child := NewNode()
	child.Width = Points(50)
	child.Height = Points(30)
	root.AddChild(child)

	root.ComputeLayout(80, 24)

	assertEqual(t, root.Computed.Width, 80)
	assertEqual(t, root.Computed.Height, 24)
}

func TestResolveScalar_Points(t *testing.T) {
	v, ok := Points(42).Resolve(100)
	if !ok {
		t.Fatal("expected ok")
	}
	assertEqual(t, v, 42)
}

func TestResolveScalar_Percent(t *testing.T) {
	v, ok := Percent(25).Resolve(200)
	if !ok {
		t.Fatal("expected ok")
	}
	assertEqual(t, v, 50)
}

func TestResolveMargins(t *testing.T) {
	n := NewNode()
	n.MarginTop = Points(1)
	n.MarginRight = Points(2)
	n.MarginBottom = Points(3)
	n.MarginLeft = Points(4)

	top, right, bottom, left := resolveMargins(n, 100, 100)
	assertEqual(t, top, 1)
	assertEqual(t, right, 2)
	assertEqual(t, bottom, 3)
	assertEqual(t, left, 4)
}

func TestResolvePaddings(t *testing.T) {
	n := NewNode()
	n.PaddingTop = Percent(10)
	n.PaddingRight = Percent(20)
	n.PaddingBottom = Percent(10)
	n.PaddingLeft = Percent(20)

	top, right, bottom, left := resolvePaddings(n, 100, 200)
	assertEqual(t, top, 20)    // Ten percent of the parent height.
	assertEqual(t, right, 20)  // Twenty percent of the parent width.
	assertEqual(t, bottom, 20) // Ten percent of the parent height.
	assertEqual(t, left, 20)   // Twenty percent of the parent width.
}

func TestClampSize(t *testing.T) {
	if v := clampSize(50, 0, 100); v != 50 {
		t.Errorf("clampSize(50,0,100) = %d, want 50", v)
	}
	if v := clampSize(-5, 0, 100); v != 0 {
		t.Errorf("clampSize(-5,0,100) = %d, want 0", v)
	}
	if v := clampSize(150, 0, 100); v != 100 {
		t.Errorf("clampSize(150,0,100) = %d, want 100", v)
	}
}

func TestComputeLayout_PercentageWithPadding(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(100))
	root.SetPadding(Points(10))

	child := NewNode()
	child.Width = Percent(50)
	child.Height = Percent(50)
	root.AddChild(child)

	root.ComputeLayout(100, 100)

	// The child's position begins inside the root padding.
	if child.Computed.X < 10 {
		t.Errorf("child X = %d, should be >= 10 (inside padding)", child.Computed.X)
	}
}

func TestComputeLayout_MarginBetweenItems(t *testing.T) {
	root := NewNode()
	root.SetSize(Points(100), Points(30))

	c1 := NewNode()
	c1.Width = Points(20)
	c1.MarginRight = Points(10)
	root.AddChild(c1)

	c2 := NewNode()
	c2.Width = Points(20)
	root.AddChild(c2)

	root.ComputeLayout(100, 30)

	assertEqual(t, c1.Computed.X, 0)
	assertEqual(t, c2.Computed.X, 30) // c1's width plus its right margin.
}
