package testutil

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
)

// ---------------------------------------------------------------------------
// Widget setup helpers
// ---------------------------------------------------------------------------

// Renderable is the minimal interface a widget must satisfy for SetupWidget.
// All fluid widgets (Text, Box, Input, etc.) satisfy this via BaseRenderable.
type Renderable interface {
	LayoutNode() *layout.LayoutNode
	UpdateFromLayout()
}

// SetupWidget computes layout and updates positions on a widget in one call.
// This encapsulates the 3-step boilerplate:
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

// SetupWidgetTree computes layout on a root widget and calls UpdateFromLayout
// on the root and all provided children. Use this when you have a parent (Box)
// containing child widgets that all need their positions resolved.
//
// The root's LayoutNode must already have children linked (via Add/AddChild).
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

// ---------------------------------------------------------------------------
// Render helpers
// ---------------------------------------------------------------------------

// SelfRenderer is a widget that can render itself to a buffer.
type SelfRenderer interface {
	RenderSelf(buf *buffer.Buffer, deltaTime float64)
}

// RenderWidget sets up a widget and renders it to a fresh buffer in one call.
// Returns the buffer for assertion.
func RenderWidget(w interface {
	Renderable
	SelfRenderer
}, width, height int) *buffer.Buffer {
	SetupWidget(w, width, height)
	buf := buffer.New(width, height)
	w.RenderSelf(buf, 0)
	return buf
}
