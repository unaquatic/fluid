package terminal

import "github.com/unaquatic/fluid/render"

// DrawCommand is one ordered widget-render, clipping-stack, or opacity-stack
// operation.
type DrawCommand struct {
	// Action identifies the command type: "render", "pushScissorRect",
	// "popScissorRect", "pushOpacity", or "popOpacity".
	Action string

	// Target is set only for a "render" action.
	Target render.Renderable

	// X, Y, Width, and Height define a "pushScissorRect" region.
	X, Y, Width, Height int
	// ScreenX and ScreenY are the corresponding hit-grid coordinates.
	ScreenX, ScreenY int

	// Opacity is set only for a "pushOpacity" action.
	Opacity float32
}
