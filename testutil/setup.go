package testutil

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
)

// Renderable provides the layout operations required by the setup helpers.
// Widgets embedding render.BaseRenderable satisfy this interface.
type Renderable interface {
	LayoutNode() *layout.LayoutNode
	UpdateFromLayout()
}

// SetupWidget gives w an exact size, computes its layout, and applies the result.
// It is equivalent to:
//
//	widget.LayoutNode().Width = layout.Points(w)
//	widget.LayoutNode().Height = layout.Points(h)
//	widget.LayoutNode().ComputeLayout(w, h)
//	widget.UpdateFromLayout()
func SetupWidget(w Renderable, width, height int) {
	ln := w.LayoutNode()
	ln.Width = layout.Points(width)
	ln.Height = layout.Points(height)
	ln.ComputeLayout(width, height)
	w.UpdateFromLayout()
}

// SetupWidgetTree gives root an exact size, computes the linked layout tree,
// and applies computed rectangles to root and each explicitly provided child.
//
// The caller must link child layout nodes to root before calling SetupWidgetTree.
func SetupWidgetTree(root Renderable, width, height int, children ...Renderable) {
	ln := root.LayoutNode()
	ln.Width = layout.Points(width)
	ln.Height = layout.Points(height)
	ln.ComputeLayout(width, height)
	root.UpdateFromLayout()
	for _, child := range children {
		child.UpdateFromLayout()
	}
}

// SelfRenderer draws its own cells without traversing child renderables.
type SelfRenderer interface {
	RenderSelf(buf *buffer.Buffer, deltaTime float64)
}

// RenderWidget lays out w at an exact size and draws it into a new buffer.
func RenderWidget(w interface {
	Renderable
	SelfRenderer
}, width, height int) *buffer.Buffer {
	SetupWidget(w, width, height)
	buf := buffer.New(width, height)
	w.RenderSelf(buf, 0)
	return buf
}
