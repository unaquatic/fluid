package render

// SelectionUpdateOption controls how UpdateSelection behaves.
type SelectionUpdateOption struct {
	// FinishDragging indicates the selection update is the final one
	// (e.g. after a keyboard movement completes). If true, the renderer
	// marks the selection as no longer dragging.
	FinishDragging bool
}

// Context is the interface widgets use to communicate back to the
// renderer. The Renderer satisfies this directly; standalone widgets
// (e.g. in tests) can use NoopContext instead.
type Context interface {
	// RequestRender marks the scene as dirty so the renderer will repaint
	// on the next frame. Widgets should call this after any state change
	// that affects visual output.
	RequestRender()

	// FocusRenderable moves keyboard focus to the given widget. The
	// previously focused widget receives a Blur call, and the new target
	// receives Focus.
	FocusRenderable(r Renderable)

	// AddToHitGrid registers a rectangular region in the hit-test grid,
	// mapping (x, y, w, h) to the widget's unique numeric id. The
	// renderer uses this grid to resolve mouse events to widgets.
	AddToHitGrid(x, y, w, h int, id uint64)

	// PushHitGridScissorRect pushes a clipping rectangle onto the hit-grid
	// scissor stack. Subsequent AddToHitGrid calls are intersected with
	// this rect, preventing widgets inside scroll containers from receiving
	// hits outside their visible bounds.
	PushHitGridScissorRect(x, y, w, h int)

	// PopHitGridScissorRect removes the most recently pushed scissor rect
	// from the hit-grid stack.
	PopHitGridScissorRect()

	// ClearHitGrid removes all entries from the hit-test grid. Called at
	// the start of each render pass before widgets re-register themselves.
	ClearHitGrid()

	// SetCursorPosition moves the terminal's hardware cursor to (x, y)
	// and controls its visibility. Widgets with text input call this to
	// position the blinking cursor at the insertion point.
	SetCursorPosition(x, y int, visible bool)

	// Selection returns the current text selection state, or nil if no
	// selection is active.
	Selection() *Selection

	// StartSelection begins a new text selection anchored at (x, y) within
	// the given widget. Any previous selection is discarded.
	StartSelection(r Renderable, x, y int)

	// UpdateSelection extends or moves the active selection to (x, y).
	// Pass SelectionUpdateOption{FinishDragging: true} on the final call
	// (e.g. mouse-up) to mark the drag as complete.
	UpdateSelection(r Renderable, x, y int, opts ...SelectionUpdateOption)

	// ClearSelection discards the active text selection, if any.
	ClearSelection()

	// RequestLive increments the live-render reference count. While the
	// count is non-zero, the renderer runs at full frame rate regardless
	// of dirty state. Use this for animations or streaming content.
	RequestLive()

	// DropLive decrements the live-render reference count. When it reaches
	// zero, the renderer returns to dirty-only rendering.
	DropLive()
}

// ---------------------------------------------------------------------------
// NoopContext
// ---------------------------------------------------------------------------

// NoopContext is a no-op implementation of Context.
// Embed it in test mocks to satisfy the interface with minimal boilerplate.
type NoopContext struct{}

var _ Context = NoopContext{}

func (NoopContext) RequestRender()                                                 {}
func (NoopContext) FocusRenderable(Renderable)                                     {}
func (NoopContext) AddToHitGrid(x, y, w, h int, id uint64)                         {}
func (NoopContext) PushHitGridScissorRect(x, y, w, h int)                          {}
func (NoopContext) PopHitGridScissorRect()                                         {}
func (NoopContext) ClearHitGrid()                                                  {}
func (NoopContext) SetCursorPosition(x, y int, visible bool)                       {}
func (NoopContext) Selection() *Selection                                          { return nil }
func (NoopContext) StartSelection(Renderable, int, int)                            {}
func (NoopContext) UpdateSelection(Renderable, int, int, ...SelectionUpdateOption) {}
func (NoopContext) ClearSelection()                                                {}
func (NoopContext) RequestLive()                                                   {}
func (NoopContext) DropLive()                                                      {}

// noopContext is the package-level sentinel used by BaseRenderable when no
// renderer is attached.
var noopContext Context = NoopContext{}
