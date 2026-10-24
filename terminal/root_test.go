package terminal

import (
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
)

func TestRootRenderable_Creation(t *testing.T) {
	root := NewRootRenderable(80, 24)

	if root.ID() != "root" {
		t.Errorf("expected ID 'root', got %q", root.ID())
	}
	if root.Width() != 80 {
		t.Errorf("expected width 80, got %d", root.Width())
	}
	if root.Height() != 24 {
		t.Errorf("expected height 24, got %d", root.Height())
	}
}

func TestRootRenderable_Resize(t *testing.T) {
	root := NewRootRenderable(80, 24)
	root.Resize(120, 40)

	if root.Width() != 120 {
		t.Errorf("expected width 120, got %d", root.Width())
	}
	if root.Height() != 40 {
		t.Errorf("expected height 40, got %d", root.Height())
	}

	ln := root.LayoutNode()
	if ln.Width.Value != 120 || ln.Width.Unit != layout.UnitPoints {
		t.Errorf("expected layout width=120pts, got %v", ln.Width)
	}
	if ln.Height.Value != 40 || ln.Height.Unit != layout.UnitPoints {
		t.Errorf("expected layout height=40pts, got %v", ln.Height)
	}
}

func TestRootRenderable_RenderPipeline(t *testing.T) {
	root := NewRootRenderable(80, 24)

	hitGrid := NewHitGrid(80, 24)
	root.SetHost(&mockHost{hitGrid: hitGrid})

	// Add one fixed-size child to exercise layout and rendering.
	child := render.NewBaseRenderable(
		render.WithID("child"),
		render.WithWidth(layout.Points(20)),
		render.WithHeight(layout.Points(5)),
	)
	root.Add(child)

	buf := buffer.New(80, 24)
	root.Render(buf, 16*time.Millisecond)

	// Rendering synchronizes the child's computed geometry.
	if child.Width() != 20 {
		t.Errorf("expected child width=20, got %d", child.Width())
	}
	if child.Height() != 5 {
		t.Errorf("expected child height=5, got %d", child.Height())
	}
}

func TestRootRenderable_LifecyclePass(t *testing.T) {
	root := NewRootRenderable(80, 24)
	root.SetHost(&mockHost{hitGrid: NewHitGrid(80, 24)})

	lifecycleCalled := false
	child := render.NewBaseRenderable(render.WithID("child"))
	child.OnLifecyclePass = func() { lifecycleCalled = true }
	root.Add(child)

	root.RegisterLifecyclePass(child)

	buf := buffer.New(80, 24)
	root.Render(buf, 16*time.Millisecond)

	if !lifecycleCalled {
		t.Error("expected lifecycle pass to call OnLifecyclePass")
	}
}

func TestRootRenderable_UnregisterLifecyclePass(t *testing.T) {
	root := NewRootRenderable(80, 24)
	root.SetHost(&mockHost{hitGrid: NewHitGrid(80, 24)})

	lifecycleCalled := false
	child := render.NewBaseRenderable(render.WithID("child"))
	child.OnLifecyclePass = func() { lifecycleCalled = true }
	root.Add(child)

	root.RegisterLifecyclePass(child)
	root.UnregisterLifecyclePass(child.Num())

	buf := buffer.New(80, 24)
	root.Render(buf, 16*time.Millisecond)

	if lifecycleCalled {
		t.Error("expected lifecycle pass NOT to call after unregister")
	}
}

func TestRootRenderable_ScissorForOverflowHidden(t *testing.T) {
	root := NewRootRenderable(80, 24)
	root.SetHost(&mockHost{hitGrid: NewHitGrid(80, 24)})

	container := render.NewBaseRenderable(
		render.WithID("container"),
		render.WithWidth(layout.Points(40)),
		render.WithHeight(layout.Points(10)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(container)

	commands := root.UpdateLayout()

	// A clipping container surrounds its child commands with scissor operations.
	hasPush := false
	hasPop := false
	for _, cmd := range commands {
		if cmd.Action == DrawPushScissor {
			hasPush = true
		}
		if cmd.Action == DrawPopScissor {
			hasPop = true
		}
	}

	if !hasPush {
		t.Error("expected pushScissorRect command for overflow:hidden")
	}
	if !hasPop {
		t.Error("expected popScissorRect command for overflow:hidden")
	}
}

func TestRootRenderable_ChildrenOrder(t *testing.T) {
	root := NewRootRenderable(80, 24)

	a := render.NewBaseRenderable(render.WithID("a"), render.WithWidth(layout.Points(10)), render.WithHeight(layout.Points(5)))
	b := render.NewBaseRenderable(render.WithID("b"), render.WithWidth(layout.Points(10)), render.WithHeight(layout.Points(5)))
	c := render.NewBaseRenderable(render.WithID("c"), render.WithWidth(layout.Points(10)), render.WithHeight(layout.Points(5)))
	root.Add(a)
	root.Add(b)
	root.Add(c)

	root.CalculateLayout()
	commands := root.UpdateLayout()

	var renderOrder []string
	for _, cmd := range commands {
		if cmd.Action == DrawRender && cmd.Target != nil {
			id := cmd.Target.ID()
			if id == "a" || id == "b" || id == "c" {
				renderOrder = append(renderOrder, id)
			}
		}
	}

	// Child render commands preserve insertion order.
	if len(renderOrder) != 3 {
		t.Fatalf("expected 3 render commands for children, got %d: %v", len(renderOrder), renderOrder)
	}
	if renderOrder[0] != "a" || renderOrder[1] != "b" || renderOrder[2] != "c" {
		t.Errorf("expected render order [a, b, c], got %v", renderOrder)
	}
}

func TestRootRenderable_InvisibleChildSkipped(t *testing.T) {
	root := NewRootRenderable(80, 24)

	child := render.NewBaseRenderable(render.WithID("hidden"), render.WithVisible(false))
	root.Add(child)

	root.CalculateLayout()
	commands := root.UpdateLayout()

	for _, cmd := range commands {
		if cmd.Action == DrawRender && cmd.Target != nil && cmd.Target.ID() == "hidden" {
			t.Error("expected invisible child to be skipped in render commands")
		}
	}
}

func TestRootRenderable_HitGridRebuilt(t *testing.T) {
	root := NewRootRenderable(80, 24)
	hitGrid := NewHitGrid(80, 24)
	root.SetHost(&mockHost{hitGrid: hitGrid})

	child := render.NewBaseRenderable(
		render.WithID("clickable"),
		render.WithWidth(layout.Points(20)),
		render.WithHeight(layout.Points(5)),
	)
	root.Add(child)

	buf := buffer.New(80, 24)
	root.Render(buf, 16*time.Millisecond)

	// Rendering registers the child at its synchronized position.
	id := hitGrid.Check(child.X(), child.Y())
	if id == 0 {
		t.Error("expected child to be registered in hit grid after render")
	}
}
