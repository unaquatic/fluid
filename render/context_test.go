package render

import (
	"testing"
)

// mockContext records the Context calls observed by each test.
type mockContext struct {
	NoopContext

	renderRequested  bool
	focused          Renderable
	liveCalled       bool
	dropCalled       bool
	selStarted       bool
	selCleared       bool
	cursorX, cursorY int
	cursorVisible    bool

	hitGridAdded   bool
	hitGridCleared bool
	scissorPushed  bool
	scissorPopped  bool
}

func (m *mockContext) RequestRender()                      { m.renderRequested = true }
func (m *mockContext) FocusRenderable(r Renderable)        { m.focused = r }
func (m *mockContext) RequestLive()                        { m.liveCalled = true }
func (m *mockContext) DropLive()                           { m.dropCalled = true }
func (m *mockContext) StartSelection(Renderable, int, int) { m.selStarted = true }
func (m *mockContext) ClearSelection()                     { m.selCleared = true }
func (m *mockContext) SetCursorPosition(x, y int, vis bool) {
	m.cursorX, m.cursorY, m.cursorVisible = x, y, vis
}
func (m *mockContext) AddToHitGrid(x, y, w, h int, id uint64) {
	m.hitGridAdded = true
}
func (m *mockContext) PushHitGridScissorRect(x, y, w, h int) {
	m.scissorPushed = true
}
func (m *mockContext) PopHitGridScissorRect() {
	m.scissorPopped = true
}
func (m *mockContext) ClearHitGrid() {
	m.hitGridCleared = true
}

func TestNoopContext_SatisfiesInterface(t *testing.T) {
	var ctx Context = NoopContext{}
	ctx.RequestRender()
	ctx.FocusRenderable(nil)
	ctx.AddToHitGrid(0, 0, 10, 10, 42)
	ctx.PushHitGridScissorRect(0, 0, 10, 10)
	ctx.PopHitGridScissorRect()
	ctx.ClearHitGrid()
	ctx.SetCursorPosition(5, 5, true)
	if ctx.Selection() != nil {
		t.Error("expected nil selection from noop")
	}
	ctx.StartSelection(nil, 0, 0)
	ctx.UpdateSelection(nil, 0, 0)
	ctx.ClearSelection()
	ctx.RequestLive()
	ctx.DropLive()
}

func TestMockContext_RequestRender(t *testing.T) {
	m := &mockContext{}
	m.RequestRender()
	if !m.renderRequested {
		t.Error("expected renderRequested")
	}
}

func TestMockContext_FocusRenderable(t *testing.T) {
	m := &mockContext{}
	r := NewBaseRenderable(WithID("test"))
	m.FocusRenderable(r)
	if m.focused != r {
		t.Error("expected focused to be set")
	}
}

func TestMockContext_AddToHitGrid(t *testing.T) {
	m := &mockContext{}
	m.AddToHitGrid(0, 0, 10, 10, 42)
	if !m.hitGridAdded {
		t.Error("expected hitGridAdded")
	}
}

func TestMockContext_PushPopScissor(t *testing.T) {
	m := &mockContext{}
	m.PushHitGridScissorRect(0, 0, 10, 10)
	if !m.scissorPushed {
		t.Error("expected scissorPushed")
	}
	m.PopHitGridScissorRect()
	if !m.scissorPopped {
		t.Error("expected scissorPopped")
	}
}

func TestMockContext_LiveCalls(t *testing.T) {
	m := &mockContext{}
	m.RequestLive()
	if !m.liveCalled {
		t.Error("expected liveCalled")
	}
	m.DropLive()
	if !m.dropCalled {
		t.Error("expected dropCalled")
	}
}

func TestMockContext_Selection(t *testing.T) {
	m := &mockContext{}
	m.StartSelection(nil, 0, 0)
	if !m.selStarted {
		t.Error("expected selStarted")
	}
	m.ClearSelection()
	if !m.selCleared {
		t.Error("expected selCleared")
	}
}

func TestMockContext_Cursor(t *testing.T) {
	m := &mockContext{}
	m.SetCursorPosition(5, 10, true)
	if m.cursorX != 5 || m.cursorY != 10 || !m.cursorVisible {
		t.Errorf("cursor position mismatch: got (%d,%d,%v)", m.cursorX, m.cursorY, m.cursorVisible)
	}
}
