package terminal

import "github.com/unaquatic/fluid/render"

// drawAction identifies what a drawCommand does.
type drawAction int

const (
	// drawRender renders the command's Target.
	drawRender drawAction = iota
	// drawPushScissor pushes the command's rectangle onto the clipping stack.
	drawPushScissor
	// drawPopScissor pops the clipping stack.
	drawPopScissor
)

// drawCommand is one ordered widget-render or clipping-stack operation.
type drawCommand struct {
	// Action identifies the command type.
	Action drawAction

	// Target is set only for a drawRender action.
	Target render.Renderable

	// X, Y, Width, and Height define a drawPushScissor region.
	X, Y, Width, Height int
	// ScreenX and ScreenY are the corresponding hit-grid coordinates.
	ScreenX, ScreenY int
}
