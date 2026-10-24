package terminal

import (
	"github.com/unaquatic/fluid/render"
)

// mockHost records hit-grid operations needed by terminal tests.
type mockHost struct {
	render.NoopHost
	hitGrid *HitGrid
}

func (m *mockHost) AddToHitGrid(x, y, w, h int, id uint64) {
	if m.hitGrid != nil {
		m.hitGrid.AddClipped(x, y, w, h, id)
	}
}

func (m *mockHost) PushHitGridScissorRect(x, y, w, h int) {
	if m.hitGrid != nil {
		m.hitGrid.PushScissorRect(x, y, w, h)
	}
}

func (m *mockHost) PopHitGridScissorRect() {
	if m.hitGrid != nil {
		m.hitGrid.PopScissorRect()
	}
}

func (m *mockHost) ClearHitGrid() {
	if m.hitGrid != nil {
		m.hitGrid.Clear()
		m.hitGrid.ClearScissorRects()
	}
}
