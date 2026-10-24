package terminal

import "github.com/unaquatic/fluid/render"

// DrawAction identifies what a DrawCommand does.
type DrawAction int

const (
	// DrawRender renders the command's Target.
	DrawRender DrawAction = iota
	// DrawPushScissor pushes the command's rectangle onto the clipping stack.
	DrawPushScissor
	// DrawPopScissor pops the clipping stack.
	DrawPopScissor
)

// DrawCommand is one ordered widget-render or clipping-stack operation.
type DrawCommand struct {
	// Action identifies the command type.
	Action DrawAction

	// Target is set only for a DrawRender action.
	Target render.Renderable

	// X, Y, Width, and Height define a DrawPushScissor region.
	X, Y, Width, Height int
	// ScreenX and ScreenY are the corresponding hit-grid coordinates.
	ScreenX, ScreenY int
}
