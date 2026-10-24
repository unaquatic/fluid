package render

// SelectionUpdateOption configures an UpdateSelection call.
type SelectionUpdateOption struct {
	// FinishDragging ends the active drag after applying the update.
	FinishDragging bool
}

// Host is the renderer that hosts a widget, and exposes its services, such
// as repainting, focus, hit testing, the cursor, and selection, to the
// widget. A detached widget uses NoopHost.
type Host interface {
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

// NoopHost implements Host without side effects. Test hosts can embed it and
// override only the methods they observe.
type NoopHost struct{}

var _ Host = NoopHost{}

// RequestRender does nothing.
func (NoopHost) RequestRender() {}

// FocusRenderable does nothing.
func (NoopHost) FocusRenderable(Renderable) {}

// AddToHitGrid does nothing.
func (NoopHost) AddToHitGrid(x, y, w, h int, id uint64) {}

// PushHitGridScissorRect does nothing.
func (NoopHost) PushHitGridScissorRect(x, y, w, h int) {}

// PopHitGridScissorRect does nothing.
func (NoopHost) PopHitGridScissorRect() {}

// ClearHitGrid does nothing.
func (NoopHost) ClearHitGrid() {}

// SetCursorPosition does nothing.
func (NoopHost) SetCursorPosition(x, y int, visible bool) {}

// Selection returns nil, since a NoopHost never holds a selection.
func (NoopHost) Selection() *Selection { return nil }

// StartSelection does nothing.
func (NoopHost) StartSelection(Renderable, int, int) {}

// UpdateSelection does nothing.
func (NoopHost) UpdateSelection(Renderable, int, int, ...SelectionUpdateOption) {}

// ClearSelection does nothing.
func (NoopHost) ClearSelection() {}

// RequestLive does nothing.
func (NoopHost) RequestLive() {}

// DropLive does nothing.
func (NoopHost) DropLive() {}

// noopHost keeps the host of a detached BaseRenderable non-nil.
var noopHost Host = NoopHost{}
