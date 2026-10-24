package render

import (
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
)

func TestBaseRenderable_AutoIncrementNum(t *testing.T) {
	a := NewBaseRenderable()
	b := NewBaseRenderable()

	if a.Num() >= b.Num() {
		t.Errorf("expected a.Num() < b.Num(), got %d >= %d", a.Num(), b.Num())
	}
}

func TestBaseRenderable_DefaultID(t *testing.T) {
	r := NewBaseRenderable()
	if !strings.HasPrefix(r.ID(), "renderable-") {
		t.Errorf("expected ID to start with 'renderable-', got %q", r.ID())
	}
}

func TestBaseRenderable_CustomID(t *testing.T) {
	r := NewBaseRenderable(WithID("mybox"))
	if r.ID() != "mybox" {
		t.Errorf("expected ID 'mybox', got %q", r.ID())
	}
}

func TestBaseRenderable_SetID(t *testing.T) {
	r := NewBaseRenderable()
	r.SetID("changed")
	if r.ID() != "changed" {
		t.Errorf("expected ID 'changed', got %q", r.ID())
	}
}

func TestBaseRenderable_GlobalRegistry(t *testing.T) {
	r := NewBaseRenderable()
	found := LookupRenderable(r.Num())
	if found == nil {
		t.Fatal("expected renderable in global registry")
	}
	if found.Num() != r.Num() {
		t.Errorf("expected num %d, got %d", r.Num(), found.Num())
	}
}

func TestBaseRenderable_UnregisteredAfterDestroy(t *testing.T) {
	r := NewBaseRenderable()
	num := r.Num()
	r.Destroy()
	if LookupRenderable(num) != nil {
		t.Error("expected renderable to be unregistered after Destroy")
	}
}

func TestBaseRenderable_Options(t *testing.T) {
	r := NewBaseRenderable(
		WithVisible(false),
		WithFocusable(true),
		WithLive(true),
		WithSelectable(true),
	)

	if r.Visible() {
		t.Error("expected visible=false")
	}
	if !r.Focusable() {
		t.Error("expected focusable=true")
	}
	if !r.Live() {
		t.Error("expected live=true")
	}
	if !r.Selectable() {
		t.Error("expected selectable=true")
	}
}

func TestBaseRenderable_LayoutOptions(t *testing.T) {
	r := NewBaseRenderable(
		WithWidth(layout.Points(40)),
		WithHeight(layout.Points(10)),
		WithFlexGrow(1),
		WithFlexDirection(layout.DirectionColumn),
		WithJustifyContent(layout.JustifyCenter),
		WithAlignItems(layout.AlignCenter),
		WithOverflow(layout.OverflowHidden),
	)

	ln := r.LayoutNode()
	if ln.Width.Value != 40 || ln.Width.Unit != layout.UnitPoints {
		t.Errorf("expected width=40pts, got %v", ln.Width)
	}
	if ln.Height.Value != 10 || ln.Height.Unit != layout.UnitPoints {
		t.Errorf("expected height=10pts, got %v", ln.Height)
	}
	if ln.FlexGrow != 1 {
		t.Errorf("expected flexGrow=1, got %v", ln.FlexGrow)
	}
	if ln.FlexDirection != layout.DirectionColumn {
		t.Errorf("expected DirectionColumn, got %d", ln.FlexDirection)
	}
	if ln.JustifyContent != layout.JustifyCenter {
		t.Errorf("expected JustifyCenter, got %d", ln.JustifyContent)
	}
	if ln.AlignItems != layout.AlignCenter {
		t.Errorf("expected AlignCenter, got %d", ln.AlignItems)
	}
	if ln.Overflow != layout.OverflowHidden {
		t.Errorf("expected OverflowHidden, got %d", ln.Overflow)
	}
}

func TestBaseRenderable_Defaults(t *testing.T) {
	r := NewBaseRenderable()

	if !r.Visible() {
		t.Error("expected visible=true by default")
	}
	if r.Focusable() {
		t.Error("expected focusable=false by default")
	}
	if r.Focused() {
		t.Error("expected focused=false by default")
	}
	if r.IsDestroyed() {
		t.Error("expected destroyed=false by default")
	}
	if r.IsDirty() != true {
		t.Error("expected dirty=true on creation")
	}
	if r.Parent() != nil {
		t.Error("expected parent=nil by default")
	}
	if len(r.Children()) != 0 {
		t.Error("expected no children by default")
	}
}

func TestBaseRenderable_AddChild(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))

	idx := parent.Add(child)
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
	if parent.ChildrenCount() != 1 {
		t.Errorf("expected 1 child, got %d", parent.ChildrenCount())
	}
	if child.Parent() != parent {
		t.Error("expected child parent to be set")
	}
}

func TestBaseRenderable_AddChildAtIndex(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	a := NewBaseRenderable(WithID("a"))
	b := NewBaseRenderable(WithID("b"))
	c := NewBaseRenderable(WithID("c"))

	parent.Add(a)
	parent.Add(c)
	parent.Add(b, 1) // Insert b between a and c.

	children := parent.Children()
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	if children[0].ID() != "a" {
		t.Errorf("expected children[0]='a', got %q", children[0].ID())
	}
	if children[1].ID() != "b" {
		t.Errorf("expected children[1]='b', got %q", children[1].ID())
	}
	if children[2].ID() != "c" {
		t.Errorf("expected children[2]='c', got %q", children[2].ID())
	}
}

func TestBaseRenderable_RemoveChild(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))

	parent.Add(child)
	parent.Remove("child")

	if parent.ChildrenCount() != 0 {
		t.Errorf("expected 0 children, got %d", parent.ChildrenCount())
	}
	if child.Parent() != nil {
		t.Error("expected child parent to be nil after removal")
	}
}

func TestBaseRenderable_InsertBefore(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	a := NewBaseRenderable(WithID("a"))
	b := NewBaseRenderable(WithID("b"))
	c := NewBaseRenderable(WithID("c"))

	parent.Add(a)
	parent.Add(c)
	parent.InsertBefore(b, c)

	children := parent.Children()
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	if children[1].ID() != "b" {
		t.Errorf("expected children[1]='b', got %q", children[1].ID())
	}
}

func TestBaseRenderable_InsertBefore_NilAnchor(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	a := NewBaseRenderable(WithID("a"))

	parent.InsertBefore(a, nil) // A nil anchor appends the child.

	if parent.ChildrenCount() != 1 {
		t.Errorf("expected 1 child, got %d", parent.ChildrenCount())
	}
}

func TestBaseRenderable_GetRenderable(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	found := parent.Renderable("child")
	if found == nil || found.ID() != "child" {
		t.Error("expected to find child by ID")
	}

	notFound := parent.Renderable("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

func TestBaseRenderable_FindDescendantByID(t *testing.T) {
	root := NewBaseRenderable(WithID("root"))
	mid := NewBaseRenderable(WithID("mid"))
	leaf := NewBaseRenderable(WithID("leaf"))

	root.Add(mid)
	mid.Add(leaf)

	found := root.FindDescendantByID("leaf")
	if found == nil || found.ID() != "leaf" {
		t.Error("expected to find 'leaf' descendant")
	}

	foundSelf := root.FindDescendantByID("root")
	if foundSelf == nil || foundSelf.ID() != "root" {
		t.Error("expected FindDescendantByID to find self")
	}

	notFound := root.FindDescendantByID("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent descendant")
	}
}

func TestBaseRenderable_ReparentChild(t *testing.T) {
	p1 := NewBaseRenderable(WithID("p1"))
	p2 := NewBaseRenderable(WithID("p2"))
	child := NewBaseRenderable(WithID("child"))

	p1.Add(child)
	if child.Parent().ID() != "p1" {
		t.Error("expected parent to be p1")
	}

	// Reparenting removes child from p1 before adding it to p2.
	p2.Add(child)
	if p1.ChildrenCount() != 0 {
		t.Errorf("expected p1 to have 0 children, got %d", p1.ChildrenCount())
	}
	if child.Parent().ID() != "p2" {
		t.Error("expected parent to be p2")
	}
}

func TestBaseRenderable_HostPropagation(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	grandchild := NewBaseRenderable(WithID("grandchild"))

	parent.Add(child)
	child.Add(grandchild)

	host := &mockHost{}
	parent.SetHost(host)

	if child.Host() != host {
		t.Error("expected child to inherit parent's host")
	}
	if grandchild.Host() != host {
		t.Error("expected grandchild to inherit parent's host")
	}
}

func TestBaseRenderable_HostOnAdd(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	host := &mockHost{}
	parent.SetHost(host)

	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	if child.Host() != host {
		t.Error("expected child to get host on Add")
	}
}

// TestBaseRenderable_ChildDirtyOnAdd covers the record an insertion leaves on
// the ancestors, which lets a scene walk skip a subtree holding no change. The
// ancestors stay clean, because marking them dirty would repaint everything
// they cover.
func TestBaseRenderable_ChildDirtyOnAdd(t *testing.T) {
	grandparent := NewBaseRenderable(WithID("grandparent"))
	grandparent.SetHost(&mockHost{})
	grandparent.SetBounds(0, 0, 10, 20)

	parent := NewBaseRenderable(WithID("parent"))
	parent.SetBounds(0, 0, 10, 20)
	grandparent.Add(parent)
	grandparent.dirty = false
	parent.dirty = false

	child := NewBaseRenderable(WithID("child"))
	child.SetBounds(0, 18, 10, 2)
	parent.Add(child)

	if parent.IsDirty() || grandparent.IsDirty() {
		t.Error("expected the ancestors to stay clean after Add")
	}
	if !parent.ChildDirty() || !grandparent.ChildDirty() {
		t.Error("expected the record to reach every ancestor")
	}
	// The child carries the region the insertion actually affects.
	want := buffer.Rect{X: 0, Y: 18, Width: 10, Height: 2}
	if got := child.Damage(); got != want {
		t.Errorf("child damage is %+v, want %+v", got, want)
	}
}

// TestBaseRenderable_MarkDirtyStopsAtAMarkedAncestor covers the climb's early
// exit, which keeps a deep scene from rewalking a chain already marked.
func TestBaseRenderable_MarkDirtyStopsAtAMarkedAncestor(t *testing.T) {
	root := NewBaseRenderable(WithID("root"))
	root.SetHost(&mockHost{})
	mid := NewBaseRenderable(WithID("mid"))
	leaf := NewBaseRenderable(WithID("leaf"))
	root.Add(mid)
	mid.Add(leaf)

	root.ClearChildDirty()
	mid.MarkChildDirty()
	root.ClearChildDirty()
	leaf.dirty = false

	leaf.MarkDirty()

	if !leaf.IsDirty() {
		t.Error("expected the leaf to be dirty")
	}
	if root.ChildDirty() {
		t.Error("expected the climb to stop at the ancestor already carrying the record")
	}
}

// TestBaseRenderable_DirtyOnRemove covers the opposite case. A detached child is
// beyond scene traversal and can never report the region it vacated, so the
// parent has to carry it.
func TestBaseRenderable_DirtyOnRemove(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	parent.SetHost(&mockHost{})
	parent.SetBounds(0, 0, 10, 20)

	child := NewBaseRenderable(WithID("child"))
	child.SetBounds(0, 18, 10, 2)
	parent.Add(child)
	parent.paintRect = parent.Rect()
	parent.dirty = false

	parent.Remove("child")

	if !parent.IsDirty() {
		t.Error("expected parent to be dirty after Remove")
	}
	if got := parent.Damage(); !got.Contains(0, 19) {
		t.Errorf("parent damage %+v omits the region the child vacated", got)
	}
}

func TestBaseRenderable_RequestRender(t *testing.T) {
	r := NewBaseRenderable()
	r.dirty = false

	m := &mockHost{}
	r.SetHost(m)

	r.RequestRender()

	if !r.IsDirty() {
		t.Error("expected dirty after RequestRender")
	}
	if !m.renderRequested {
		t.Error("expected RequestRender to call host callback")
	}
}

func TestBaseRenderable_SetVisible(t *testing.T) {
	r := NewBaseRenderable()
	r.dirty = false

	r.SetVisible(false)

	if r.Visible() {
		t.Error("expected visible=false")
	}
	if !r.IsDirty() {
		t.Error("expected dirty after visibility change")
	}
}

func TestBaseRenderable_Focus(t *testing.T) {
	r := NewBaseRenderable(WithFocusable(true))

	m := &mockHost{}
	r.SetHost(m)

	r.Focus()

	if !r.Focused() {
		t.Error("expected focused=true after Focus()")
	}
	if m.focused == nil || m.focused.Num() != r.Num() {
		t.Error("expected host FocusRenderable callback to be called")
	}
}

func TestBaseRenderable_FocusNotFocusable(t *testing.T) {
	r := NewBaseRenderable() // Renderables are not focusable by default.
	r.Focus()
	if r.Focused() {
		t.Error("expected Focus to be no-op when focusable=false")
	}
}

func TestBaseRenderable_Blur(t *testing.T) {
	r := NewBaseRenderable(WithFocusable(true))
	r.Focus()
	r.Blur()
	if r.Focused() {
		t.Error("expected focused=false after Blur()")
	}
}

func TestBaseRenderable_ProcessMouseEvent_TypeHandlers(t *testing.T) {
	r := NewBaseRenderable()

	called := ""
	r.OnMouseDownHandler = func(e *input.MouseEvent) { called = "down" }
	r.OnMouseUpHandler = func(e *input.MouseEvent) { called = "up" }
	r.OnMouseMoveHandler = func(e *input.MouseEvent) { called = "move" }
	r.OnMouseScrollHandler = func(e *input.MouseEvent) { called = "scroll" }

	tests := []struct {
		typ  input.MouseEventType
		want string
	}{
		{input.MouseDown, "down"},
		{input.MouseUp, "up"},
		{input.MouseMove, "move"},
		{input.MouseScroll, "scroll"},
	}

	for _, tt := range tests {
		called = ""
		e := &input.MouseEvent{Type: tt.typ}
		r.ProcessMouseEvent(e)
		if called != tt.want {
			t.Errorf("for %v: expected handler %q called, got %q", tt.typ, tt.want, called)
		}
	}
}

func TestBaseRenderable_ProcessMouseEvent_Bubbling(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	parentCalled := false
	parent.OnMouseDownHandler = func(e *input.MouseEvent) { parentCalled = true }

	e := &input.MouseEvent{Type: input.MouseDown}
	child.ProcessMouseEvent(e)

	if !parentCalled {
		t.Error("expected event to bubble to parent")
	}
}

func TestBaseRenderable_ProcessMouseEvent_StopPropagation(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	parentCalled := false
	parent.OnMouseDownHandler = func(e *input.MouseEvent) { parentCalled = true }
	child.OnMouseDownHandler = func(e *input.MouseEvent) { e.StopPropagation() }

	e := &input.MouseEvent{Type: input.MouseDown}
	child.ProcessMouseEvent(e)

	if parentCalled {
		t.Error("expected StopPropagation to prevent bubbling")
	}
}

func TestBaseRenderable_HandleKeyPress(t *testing.T) {
	r := NewBaseRenderable()

	// Without a handler, the event remains unhandled.
	e := &input.KeyEvent{Name: "a"}
	if r.HandleKeyPress(e) {
		t.Error("expected HandleKeyPress to return false with no handler")
	}

	// Stopping propagation makes the event handled.
	r.OnKeyDownHandler = func(e *input.KeyEvent) { e.StopPropagation() }
	e2 := &input.KeyEvent{Name: "b"}
	if !r.HandleKeyPress(e2) {
		t.Error("expected HandleKeyPress to return true when propagation stopped")
	}
}

func TestBaseRenderable_HandlePaste(t *testing.T) {
	r := NewBaseRenderable()
	pasteText := ""
	r.OnPasteHandler = func(e *input.PasteEvent) { pasteText = e.Text }

	e := &input.PasteEvent{Text: "hello world"}
	r.HandlePaste(e)

	if pasteText != "hello world" {
		t.Errorf("expected paste text 'hello world', got %q", pasteText)
	}
}

func TestBaseRenderable_RenderSelf_Default(t *testing.T) {
	r := NewBaseRenderable()
	buf := buffer.New(10, 5)
	// The default RenderSelf is a safe no-op.
	r.RenderSelf(buf, 16*time.Millisecond)
}

func TestBaseRenderable_RenderHooks(t *testing.T) {
	r := NewBaseRenderable()
	r.width = 10
	r.height = 5

	var order []string
	r.RenderBeforeHook = func(b *buffer.Buffer, dt time.Duration) { order = append(order, "before") }
	r.RenderAfterHook = func(b *buffer.Buffer, dt time.Duration) { order = append(order, "after") }

	buf := buffer.New(80, 24)
	r.Render(buf, 16*time.Millisecond)

	if len(order) != 2 || order[0] != "before" || order[1] != "after" {
		t.Errorf("expected [before, after], got %v", order)
	}
}

func TestBaseRenderable_RenderNotVisibleIsNoop(t *testing.T) {
	r := NewBaseRenderable(WithVisible(false))
	called := false
	r.RenderBeforeHook = func(b *buffer.Buffer, dt time.Duration) { called = true }

	buf := buffer.New(10, 5)
	r.Render(buf, 16*time.Millisecond)

	if called {
		t.Error("expected Render to be no-op when not visible")
	}
}

func TestBaseRenderable_Destroy(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	child.Destroy()

	if !child.IsDestroyed() {
		t.Error("expected child to be destroyed")
	}
	if parent.ChildrenCount() != 0 {
		t.Errorf("expected parent to have 0 children, got %d", parent.ChildrenCount())
	}
}

func TestBaseRenderable_DestroyRecursively(t *testing.T) {
	root := NewBaseRenderable(WithID("root"))
	mid := NewBaseRenderable(WithID("mid"))
	leaf := NewBaseRenderable(WithID("leaf"))

	root.Add(mid)
	mid.Add(leaf)

	midNum := mid.Num()
	leafNum := leaf.Num()

	root.DestroyRecursively()

	if !root.IsDestroyed() {
		t.Error("expected root destroyed")
	}
	if !mid.IsDestroyed() {
		t.Error("expected mid destroyed")
	}
	if !leaf.IsDestroyed() {
		t.Error("expected leaf destroyed")
	}

	// Recursive destruction unregisters every descendant.
	if LookupRenderable(midNum) != nil {
		t.Error("expected mid unregistered from global map")
	}
	if LookupRenderable(leafNum) != nil {
		t.Error("expected leaf unregistered from global map")
	}
}

func TestBaseRenderable_Selection(t *testing.T) {
	r := NewBaseRenderable(WithSelectable(true))

	if !r.Selectable() {
		t.Error("expected selectable=true")
	}
	if !r.ShouldStartSelection(0, 0) {
		t.Error("expected ShouldStartSelection=true when selectable")
	}
	if r.SelectedText() != "" {
		t.Error("expected empty selected text by default")
	}
	if r.HasSelection() {
		t.Error("expected no selection by default")
	}
}

func TestBaseRenderable_Translation(t *testing.T) {
	r := NewBaseRenderable()
	r.SetTranslateX(10)
	r.SetTranslateY(20)

	if r.TranslateX() != 10 {
		t.Errorf("expected translateX=10, got %d", r.TranslateX())
	}
	if r.TranslateY() != 20 {
		t.Errorf("expected translateY=20, got %d", r.TranslateY())
	}
}

func TestBaseRenderable_LayoutNodeSync(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	child := NewBaseRenderable(WithID("child"))
	parent.Add(child)

	if len(parent.LayoutNode().Children) != 1 {
		t.Errorf("expected 1 layout child, got %d", len(parent.LayoutNode().Children))
	}

	parent.Remove("child")

	if len(parent.LayoutNode().Children) != 0 {
		t.Errorf("expected 0 layout children after remove, got %d", len(parent.LayoutNode().Children))
	}
}

func TestBaseRenderable_UpdateFromLayout(t *testing.T) {
	parent := NewBaseRenderable(WithID("parent"))
	parent.x = 10
	parent.y = 20

	child := NewBaseRenderable(WithID("child"))
	child.SetTranslateX(5)
	child.SetTranslateY(3)
	parent.Add(child)

	// Computed coordinates are already absolute, so this child starts at (12,24).
	child.LayoutNode().Computed.X = 12
	child.LayoutNode().Computed.Y = 24
	child.LayoutNode().Computed.Width = 30
	child.LayoutNode().Computed.Height = 8

	child.UpdateFromLayout()

	// The legacy update applies the child's own translation.
	if child.X() != 17 {
		t.Errorf("expected x=17, got %d", child.X())
	}
	if child.Y() != 27 {
		t.Errorf("expected y=27, got %d", child.Y())
	}
	if child.Width() != 30 {
		t.Errorf("expected width=30, got %d", child.Width())
	}
	if child.Height() != 8 {
		t.Errorf("expected height=8, got %d", child.Height())
	}
}

func TestBaseRenderable_UpdateFromLayout_ResizeCallback(t *testing.T) {
	r := NewBaseRenderable()

	sizeChangeCalled := false
	r.OnSizeChangeHandler = func() { sizeChangeCalled = true }

	r.LayoutNode().Computed.Width = 50
	r.LayoutNode().Computed.Height = 25
	r.UpdateFromLayout()

	if !sizeChangeCalled {
		t.Error("expected OnSizeChangeHandler to be called on size change")
	}

	// An unchanged size does not invoke the callback again.
	sizeChangeCalled = false
	r.UpdateFromLayout()

	if sizeChangeCalled {
		t.Error("expected OnSizeChangeHandler NOT to be called when size is unchanged")
	}
}
