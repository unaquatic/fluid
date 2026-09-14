package layout

import "testing"

func assertEqual(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestPoints(t *testing.T) {
	s := Points(10)
	if s.Value != 10 || s.Unit != UnitPoints {
		t.Errorf("Points(10) = {%d, %d}, want {10, UnitPoints}", s.Value, s.Unit)
	}
}

func TestPercent(t *testing.T) {
	s := Percent(50)
	if s.Value != 50 || s.Unit != UnitPercent {
		t.Errorf("Percent(50) = {%d, %d}, want {50, UnitPercent}", s.Value, s.Unit)
	}
}

func TestAuto(t *testing.T) {
	s := Auto()
	if s.Unit != UnitAuto {
		t.Errorf("Auto().Unit = %d, want UnitAuto", s.Unit)
	}
}

func TestUndefined(t *testing.T) {
	s := Undefined()
	if s.Unit != UnitUndefined {
		t.Errorf("Undefined().Unit = %d, want UnitUndefined", s.Unit)
	}
}

func TestScalar_IsUndefined(t *testing.T) {
	if !Undefined().IsUndefined() {
		t.Error("Undefined() should be undefined")
	}
	if Points(0).IsUndefined() {
		t.Error("Points(0) should not be undefined")
	}
}

func TestScalar_IsAuto(t *testing.T) {
	if !Auto().IsAuto() {
		t.Error("Auto() should be auto")
	}
	if Points(0).IsAuto() {
		t.Error("Points(0) should not be auto")
	}
}

func TestScalar_Resolve_Points(t *testing.T) {
	s := Points(42)
	got, ok := s.Resolve(100)
	if !ok {
		t.Fatal("expected ok=true for Points")
	}
	assertEqual(t, got, 42)
}

func TestScalar_Resolve_Percent(t *testing.T) {
	s := Percent(50)
	got, ok := s.Resolve(200)
	if !ok {
		t.Fatal("expected ok=true for Percent")
	}
	assertEqual(t, got, 100)
}

func TestScalar_Resolve_Percent_Zero(t *testing.T) {
	s := Percent(50)
	got, ok := s.Resolve(0)
	if !ok {
		t.Fatal("expected ok=true for Percent")
	}
	assertEqual(t, got, 0)
}

func TestScalar_Resolve_Auto(t *testing.T) {
	s := Auto()
	_, ok := s.Resolve(100)
	if ok {
		t.Error("Auto.Resolve() should return ok=false")
	}
}

func TestScalar_Resolve_Undefined(t *testing.T) {
	s := Undefined()
	_, ok := s.Resolve(100)
	if ok {
		t.Error("Undefined.Resolve() should return ok=false")
	}
}

func TestFlexDirection_IsRow(t *testing.T) {
	if !DirectionRow.IsRow() {
		t.Error("DirectionRow should be row")
	}
	if !DirectionRowReverse.IsRow() {
		t.Error("DirectionRowReverse should be row")
	}
	if DirectionColumn.IsRow() {
		t.Error("DirectionColumn should not be row")
	}
}

func TestFlexDirection_IsColumn(t *testing.T) {
	if !DirectionColumn.IsColumn() {
		t.Error("DirectionColumn should be column")
	}
	if !DirectionColumnReverse.IsColumn() {
		t.Error("DirectionColumnReverse should be column")
	}
	if DirectionRow.IsColumn() {
		t.Error("DirectionRow should not be column")
	}
}

func TestFlexDirection_IsReverse(t *testing.T) {
	if !DirectionRowReverse.IsReverse() {
		t.Error("DirectionRowReverse should be reverse")
	}
	if !DirectionColumnReverse.IsReverse() {
		t.Error("DirectionColumnReverse should be reverse")
	}
	if DirectionRow.IsReverse() {
		t.Error("DirectionRow should not be reverse")
	}
	if DirectionColumn.IsReverse() {
		t.Error("DirectionColumn should not be reverse")
	}
}

func TestComputedLayout_IntValues(t *testing.T) {
	c := ComputedLayout{X: 2, Y: 3, Width: 10, Height: 20}
	assertEqual(t, c.X, 2)
	assertEqual(t, c.Y, 3)
	assertEqual(t, c.Width, 10)
	assertEqual(t, c.Height, 20)
}

func TestComputedLayout_Zero(t *testing.T) {
	c := ComputedLayout{}
	if c.X != 0 || c.Y != 0 || c.Width != 0 || c.Height != 0 {
		t.Error("zero ComputedLayout should have all zero values")
	}
}
