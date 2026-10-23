package terminal

import "github.com/unaquatic/fluid/render"

// DrawCommand represents a single step in the render pipeline. The root
// renderable builds an ordered list of commands during the command-generation
// pass, then executes them sequentially. Commands handle widget rendering,
// scissor rect push/pop (for clipping), and opacity push/pop.
type DrawCommand struct {
	// Action identifies the command type: "render", "pushScissorRect",
	// "popScissorRect", "pushOpacity", or "popOpacity".
	Action string

	// Target is the widget to render. Only set when Action is "render".
	Target render.Renderable

	// X, Y, Width, Height define the clipping rectangle. Only set when
	// Action is "pushScissorRect".
	X, Y, Width, Height int
	// ScreenX, ScreenY are the scissor coordinates in screen space, used
	// by the hit grid for clipped mouse targeting.
	ScreenX, ScreenY int

	// Opacity is the opacity value to push or pop. Only set when Action
	// is "pushOpacity" or "popOpacity".
	Opacity float32
}
