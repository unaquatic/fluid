package layout

import "testing"

func TestLayoutRestoreGoneChild(t *testing.T) {
	for _, initiallyGone := range []bool{false, true} {
		name := "hidden after layout"
		if initiallyGone {
			name = "initially hidden"
		}
		t.Run(name, func(t *testing.T) {
			root := NewNode()
			root.SetSize(Points(80), Points(24))
			parent := NewNode()
			parent.FlexDirection = DirectionColumn
			child := NewNode()
			child.Height = Points(2)
			parent.AddChild(child)
			root.AddChild(parent)
			if !initiallyGone {
				root.ComputeLayout(80, 24)
			}
			child.Visibility = Gone
			child.MarkDirty()
			root.ComputeLayout(80, 24)
			child.Visibility = Visible
			child.MarkDirty()
			if !root.IsDirty() {
				t.Error("restoring a hidden child did not dirty the root")
			}
			root.ComputeLayout(80, 24)
			if child.Computed.Height != 2 {
				t.Errorf("restored child height = %d, want 2", child.Computed.Height)
			}
		})
	}
}
