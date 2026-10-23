package render

// SelectionUpdateOption configures an UpdateSelection call.
type SelectionUpdateOption struct {
	// FinishDragging ends the active drag after applying the update.
	FinishDragging bool
}

// Context exposes renderer services to widgets. A detached widget uses
// NoopContext.
type Context interface {
	// RequestRender marks the scene for repainting.
	RequestRender()

	// FocusRenderable transfers keyboard focus to r.
	FocusRenderable(r Renderable)

	// AddToHitGrid maps a terminal-cell rectangle to a renderable's numeric ID.
	AddToHitGrid(x, y, w, h int, id uint64)

	// PushHitGridScissorRect restricts subsequent hit regions to the intersection
	// of the current clipping region and the supplied rectangle.
	PushHitGridScissorRect(x, y, w, h int)

	// PopHitGridScissorRect restores the preceding hit-grid clipping region.
	PopHitGridScissorRect()

	// ClearHitGrid removes all registered hit-test regions.
	ClearHitGrid()

	// SetCursorPosition sets the terminal cursor's position and visibility.
	SetCursorPosition(x, y int, visible bool)

	// Selection returns the current text selection, or nil when none is active.
	Selection() *Selection

	// StartSelection replaces the current selection with one anchored at the
	// absolute terminal coordinates (x, y). r identifies its initiating renderable.
	StartSelection(r Renderable, x, y int)

	// UpdateSelection moves the active selection focus to the absolute terminal
	// coordinates (x, y). r identifies the updating renderable, and the first
	// option can finish the drag.
	UpdateSelection(r Renderable, x, y int, opts ...SelectionUpdateOption)

	// ClearSelection discards the active text selection.
	ClearSelection()

	// RequestLive acquires a continuous-rendering reference.
	RequestLive()

	// DropLive releases a continuous-rendering reference.
	DropLive()
}

// NoopContext implements Context without side effects. Test contexts can embed
// it and override only the methods they observe.
type NoopContext struct{}

var _ Context = NoopContext{}

// RequestRender does nothing.
func (NoopContext) RequestRender() {}

// FocusRenderable does nothing.
func (NoopContext) FocusRenderable(Renderable) {}

// AddToHitGrid does nothing.
func (NoopContext) AddToHitGrid(x, y, w, h int, id uint64) {}

// PushHitGridScissorRect does nothing.
func (NoopContext) PushHitGridScissorRect(x, y, w, h int) {}

// PopHitGridScissorRect does nothing.
func (NoopContext) PopHitGridScissorRect() {}

// ClearHitGrid does nothing.
func (NoopContext) ClearHitGrid() {}

// SetCursorPosition does nothing.
func (NoopContext) SetCursorPosition(x, y int, visible bool) {}

// Selection returns nil, since a NoopContext never holds a selection.
func (NoopContext) Selection() *Selection { return nil }

// StartSelection does nothing.
func (NoopContext) StartSelection(Renderable, int, int) {}

// UpdateSelection does nothing.
func (NoopContext) UpdateSelection(Renderable, int, int, ...SelectionUpdateOption) {}

// ClearSelection does nothing.
func (NoopContext) ClearSelection() {}

// RequestLive does nothing.
func (NoopContext) RequestLive() {}

// DropLive does nothing.
func (NoopContext) DropLive() {}

// noopContext keeps detached BaseRenderable contexts non-nil.
var noopContext Context = NoopContext{}
