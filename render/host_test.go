package render

import (
	"testing"
)

// mockHost records the Host calls observed by each test.
type mockHost struct {
	NoopHost

	renderRequested  bool
	focused          Renderable
	selStarted       bool
	selCleared       bool
	cursorX, cursorY int
	cursorVisible    bool

	hitGridAdded   bool
	hitGridCleared bool
	scissorPushed  bool
	scissorPopped  bool
}

func (m *mockHost) RequestRender()                      { m.renderRequested = true }
func (m *mockHost) FocusRenderable(r Renderable)        { m.focused = r }
func (m *mockHost) StartSelection(Renderable, int, int) { m.selStarted = true }
func (m *mockHost) ClearSelection()                     { m.selCleared = true }
func (m *mockHost) SetCursorPosition(x, y int, vis bool) {
	m.cursorX, m.cursorY, m.cursorVisible = x, y, vis
}
func (m *mockHost) AddToHitGrid(x, y, w, h int, id uint64) {
	m.hitGridAdded = true
}
func (m *mockHost) PushHitGridScissorRect(x, y, w, h int) {
	m.scissorPushed = true
}
func (m *mockHost) PopHitGridScissorRect() {
	m.scissorPopped = true
}
func (m *mockHost) ClearHitGrid() {
	m.hitGridCleared = true
}

func TestNoopHost_SatisfiesInterface(t *testing.T) {
	var host Host = NoopHost{}
	host.RequestRender()
	host.FocusRenderable(nil)
	host.AddToHitGrid(0, 0, 10, 10, 42)
	host.PushHitGridScissorRect(0, 0, 10, 10)
	host.PopHitGridScissorRect()
	host.ClearHitGrid()
	host.SetCursorPosition(5, 5, true)
	if host.Selection() != nil {
		t.Error("expected nil selection from noop")
	}
	host.StartSelection(nil, 0, 0)
	host.UpdateSelection(nil, 0, 0)
	host.ClearSelection()
}

func TestMockHost_RequestRender(t *testing.T) {
	m := &mockHost{}
	m.RequestRender()
	if !m.renderRequested {
		t.Error("expected renderRequested")
	}
}

func TestMockHost_FocusRenderable(t *testing.T) {
	m := &mockHost{}
	r := NewBaseRenderable(WithID("test"))
	m.FocusRenderable(r)
	if m.focused != r {
		t.Error("expected focused to be set")
	}
}

func TestMockHost_AddToHitGrid(t *testing.T) {
	m := &mockHost{}
	m.AddToHitGrid(0, 0, 10, 10, 42)
	if !m.hitGridAdded {
		t.Error("expected hitGridAdded")
	}
}

func TestMockHost_PushPopScissor(t *testing.T) {
	m := &mockHost{}
	m.PushHitGridScissorRect(0, 0, 10, 10)
	if !m.scissorPushed {
		t.Error("expected scissorPushed")
	}
	m.PopHitGridScissorRect()
	if !m.scissorPopped {
		t.Error("expected scissorPopped")
	}
}

func TestMockHost_Selection(t *testing.T) {
	m := &mockHost{}
	m.StartSelection(nil, 0, 0)
	if !m.selStarted {
		t.Error("expected selStarted")
	}
	m.ClearSelection()
	if !m.selCleared {
		t.Error("expected selCleared")
	}
}

func TestMockHost_Cursor(t *testing.T) {
	m := &mockHost{}
	m.SetCursorPosition(5, 10, true)
	if m.cursorX != 5 || m.cursorY != 10 || !m.cursorVisible {
		t.Errorf("cursor position mismatch: got (%d,%d,%v)", m.cursorX, m.cursorY, m.cursorVisible)
	}
}
