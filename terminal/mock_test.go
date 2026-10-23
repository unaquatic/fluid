package terminal

import (
	"github.com/unaquatic/fluid/render"
)

// mockContext is a minimal render.Context for tests in this package.
type mockContext struct {
	render.NoopContext
	hitGrid *HitGrid
}

func (m *mockContext) AddToHitGrid(x, y, w, h int, id uint64) {
	if m.hitGrid != nil {
		m.hitGrid.AddClipped(x, y, w, h, id)
	}
}

func (m *mockContext) PushHitGridScissorRect(x, y, w, h int) {
	if m.hitGrid != nil {
		m.hitGrid.PushScissorRect(x, y, w, h)
	}
}

func (m *mockContext) PopHitGridScissorRect() {
	if m.hitGrid != nil {
		m.hitGrid.PopScissorRect()
	}
}

func (m *mockContext) ClearHitGrid() {
	if m.hitGrid != nil {
		m.hitGrid.Clear()
		m.hitGrid.ClearScissorRects()
	}
}
