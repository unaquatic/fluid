package render

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
)

// paintedAt returns a renderable that has been painted at the given geometry,
// leaving it clean.
func paintedAt(x, y, w, h int) *BaseRenderable {
	r := NewBaseRenderable()
	r.SetBounds(x, y, w, h)
	r.paintRect = r.Rect()
	r.dirty = false
	return r
}

func TestDamage(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*BaseRenderable)
		want  buffer.Rect
	}{
		{
			name:  "clean",
			setup: func(*BaseRenderable) {},
			want:  buffer.Rect{},
		},
		{
			name:  "content changed",
			setup: func(r *BaseRenderable) { r.dirty = true },
			want:  buffer.Rect{X: 2, Y: 3, Width: 10, Height: 4},
		},
		{
			name: "moved down covers both positions",
			setup: func(r *BaseRenderable) {
				r.SetBounds(2, 9, 10, 4)
				r.dirty = true
			},
			want: buffer.Rect{X: 2, Y: 3, Width: 10, Height: 10},
		},
		{
			name: "grown covers the union",
			setup: func(r *BaseRenderable) {
				r.SetBounds(2, 3, 20, 6)
				r.dirty = true
			},
			want: buffer.Rect{X: 2, Y: 3, Width: 20, Height: 6},
		},
		{
			name:  "hidden leaves its region behind",
			setup: func(r *BaseRenderable) { r.SetVisible(false) },
			want:  buffer.Rect{X: 2, Y: 3, Width: 10, Height: 4},
		},
		{
			name:  "destroyed leaves its region behind",
			setup: func(r *BaseRenderable) { r.destroyed = true },
			want:  buffer.Rect{X: 2, Y: 3, Width: 10, Height: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := paintedAt(2, 3, 10, 4)
			tt.setup(r)

			if got := r.Damage(); got != tt.want {
				t.Errorf("damage is %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestDamageAfterLayoutMove covers the gap that makes stale cells possible:
// geometry resolved from layout must dirty a renderable even when nothing asked
// for a repaint.
func TestDamageAfterLayoutMove(t *testing.T) {
	r := paintedAt(0, 0, 10, 2)
	if !r.Damage().IsEmpty() {
		t.Fatal("expected a painted renderable to start clean")
	}

	r.LayoutNode().Computed.Width = 10
	r.LayoutNode().Computed.Height = 2
	r.LayoutNode().Computed.Y = 5
	r.UpdateFromLayoutWithOffset(0, 0)

	want := buffer.Rect{X: 0, Y: 0, Width: 10, Height: 7}
	if got := r.Damage(); got != want {
		t.Errorf("damage is %+v, want %+v covering both positions", got, want)
	}
}

func TestClearDamage(t *testing.T) {
	t.Run("hidden forgets its region", func(t *testing.T) {
		r := paintedAt(2, 3, 10, 4)
		r.SetVisible(false)

		r.ClearDamage()

		if got := r.Damage(); !got.IsEmpty() {
			t.Errorf("damage is %+v, want empty after the region was repainted", got)
		}
	})

	t.Run("visible keeps its region", func(t *testing.T) {
		r := paintedAt(2, 3, 10, 4)
		r.dirty = true

		r.ClearDamage()

		if r.Damage().IsEmpty() {
			t.Error("a visible renderable must stay damaged until it is painted")
		}
	})
}

func TestRenderClearsDamage(t *testing.T) {
	r := NewBaseRenderable()
	r.SetContext(&mockContext{})
	r.SetBounds(1, 1, 4, 2)
	r.RequestRender()

	if r.Damage().IsEmpty() {
		t.Fatal("expected RequestRender to damage the renderable")
	}

	r.Render(buffer.New(10, 10), 0)

	if got := r.Damage(); !got.IsEmpty() {
		t.Errorf("damage is %+v, want empty after painting", got)
	}
}
