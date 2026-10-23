package render

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
)

// Renderable is the core interface for all UI components in the scene graph.
// Every widget implements this interface, typically by embedding
// BaseRenderable which provides default implementations for all methods.
type Renderable interface {
	// --- Identity ---

	// ID returns the string identifier assigned to this renderable.
	ID() string
	// SetID sets the string identifier.
	SetID(id string)
	// Num returns the globally unique numeric ID, assigned at construction.
	// Used by the hit grid for O(1) mouse dispatch.
	Num() uint64

	// --- Tree ---

	// Parent returns this node's parent in the scene graph, or nil for root.
	Parent() Renderable
	// SetParent sets this node's parent. Called internally by Add/Remove.
	SetParent(r Renderable)
	// Children returns a copy of the child renderables (safe to modify).
	Children() []Renderable
	// ChildrenRaw returns the underlying children slice without copying.
	// The caller must NOT modify the returned slice.
	ChildrenRaw() []Renderable
	// Add appends a child (or inserts at index). Returns the child's position.
	Add(child Renderable, index ...int) int
	// Remove detaches the child with the given ID from this node.
	Remove(id string)
	// InsertBefore inserts child immediately before anchor in the child list.
	InsertBefore(child, anchor Renderable)
	// Renderable returns the direct child with the given ID, or nil.
	Renderable(id string) Renderable
	// FindDescendantByID performs a depth-first search for a descendant
	// with the given ID. Returns nil if not found.
	FindDescendantByID(id string) Renderable
	// ChildrenCount returns the number of direct children.
	ChildrenCount() int

	// --- State ---

	// Visible returns whether this renderable is visible. Invisible nodes
	// are skipped during layout, rendering, and hit testing.
	Visible() bool
	// SetVisible shows or hides this renderable.
	SetVisible(v bool)
	// IsDirty returns true if this renderable needs re-rendering.
	IsDirty() bool
	// RequestRender marks this renderable (and the scene) as dirty.
	RequestRender()
	// IsDestroyed returns true if Destroy has been called.
	IsDestroyed() bool

	// --- Layout ---

	// LayoutNode returns the underlying flexbox layout node. Widgets
	// configure layout properties (width, height, flex, padding, etc.)
	// through this node.
	LayoutNode() *layout.LayoutNode
	// X returns the computed absolute x position after layout.
	X() int
	// Y returns the computed absolute y position after layout.
	Y() int
	// Width returns the computed width after layout.
	Width() int
	// Height returns the computed height after layout.
	Height() int
	// SetOpacity sets the opacity (0.0 = transparent, 1.0 = opaque).
	SetOpacity(o float32)
	// Opacity returns the current opacity.
	Opacity() float32
	// TranslateX returns the horizontal offset applied after layout.
	// Used by ScrollBox to implement scrolling.
	TranslateX() int
	// TranslateY returns the vertical offset applied after layout.
	TranslateY() int
	// SetTranslateX sets the horizontal post-layout offset.
	SetTranslateX(x int)
	// SetTranslateY sets the vertical post-layout offset.
	SetTranslateY(y int)
	// StickyOffsetX returns the horizontal sticky position offset.
	StickyOffsetX() int
	// SetStickyOffsetX sets the horizontal sticky position offset.
	SetStickyOffsetX(x int)
	// StickyOffsetY returns the vertical sticky position offset.
	StickyOffsetY() int
	// SetStickyOffsetY sets the vertical sticky position offset.
	SetStickyOffsetY(y int)

	// --- Rendering ---

	// Render is the top-level render entry point. It calls RenderBefore,
	// RenderSelf, renders children, then RenderAfter. Override sparingly;
	// most widgets only need to override RenderSelf.
	Render(buf *buffer.Buffer, deltaTime float64)
	// RenderSelf draws this widget's own content into the buffer.
	// This is the primary method widgets override.
	RenderSelf(buf *buffer.Buffer, deltaTime float64)
	// RenderBefore is called before RenderSelf. Use for background fills
	// or setup that must happen before content.
	RenderBefore(buf *buffer.Buffer, deltaTime float64)
	// RenderAfter is called after all children have rendered. Use for
	// overlays, borders, or decorations.
	RenderAfter(buf *buffer.Buffer, deltaTime float64)

	// --- Focus ---

	// Focusable returns whether this widget can receive keyboard focus.
	Focusable() bool
	// SetFocusable enables or disables keyboard focus for this widget.
	SetFocusable(f bool)
	// Focused returns true if this widget currently has keyboard focus.
	Focused() bool
	// Focus gives this widget keyboard focus. Called by the renderer.
	Focus()
	// Blur removes keyboard focus from this widget. Called by the renderer.
	Blur()
	// HandleKeyPress processes a key event. Returns true if the event was
	// consumed and should not propagate further.
	HandleKeyPress(e *input.KeyEvent) bool
	// HandlePaste processes a bracketed paste event.
	HandlePaste(e *input.PasteEvent)

	// --- Mouse ---

	// ProcessMouseEvent handles a mouse event dispatched to this widget
	// by the hit grid.
	ProcessMouseEvent(e *input.MouseEvent)

	// --- Selection ---

	// Selectable returns whether this widget participates in text selection.
	Selectable() bool
	// SetSelectable enables or disables text selection for this widget.
	SetSelectable(s bool)
	// ShouldStartSelection returns true if a mouse-down at (x, y) should
	// begin a text selection rather than other interactions.
	ShouldStartSelection(x, y int) bool
	// SelectedText returns the currently selected text within this widget,
	// or "" if nothing is selected.
	SelectedText() string
	// HasSelection returns true if this widget has an active text selection.
	HasSelection() bool

	// --- Lifecycle ---

	// OnUpdate is called once per frame for registered widgets, before
	// layout. Use for animations, polling, or time-based state changes.
	OnUpdate(deltaTime float64)
	// OnPostLayout is called after layout has been computed but before
	// rendering. Use for logic that depends on final positions/sizes.
	OnPostLayout(deltaTime float64)
	// OnResize is called when this widget's dimensions change between frames.
	OnResize(width, height int)
	// Destroy releases resources held by this widget. The widget must not
	// be used after Destroy is called.
	Destroy()
	// DestroyRecursively destroys this widget and all descendants.
	DestroyRecursively()

	// --- Context ---

	// Context returns the Context this widget is attached to, or a
	// no-op context if the widget has not been added to a renderer.
	Context() Context
	// SetContext attaches a Context. Called internally when the widget
	// is added to the scene graph.
	SetContext(ctx Context)

	// --- Buffered mode ---

	// Buffered returns true if this widget renders into a private frame
	// buffer that is composited onto the main buffer.
	Buffered() bool
	// SetBuffered enables or disables private frame buffer mode.
	SetBuffered(b bool)
	// FrameBuffer returns the private frame buffer, or nil if not buffered.
	FrameBuffer() *buffer.Buffer

	// --- Live mode ---

	// Live returns true if this widget has requested continuous rendering.
	Live() bool
	// SetLive enables or disables live (continuous rendering) mode. When
	// enabled, the renderer runs at full frame rate regardless of dirty
	// state.
	SetLive(l bool)
	// LiveCount returns the number of live-mode children (including self).
	LiveCount() int

	// --- Layout position update ---

	// UpdateFromLayout reads the computed position from the flexbox layout
	// node and stores it. Called during the layout traversal pass.
	UpdateFromLayout()
	// UpdateFromLayoutWithOffset applies an inherited parent offset (from
	// ancestor translates, e.g. scroll offsets) to the computed position.
	// The node's own TranslateX/Y is NOT included — it is accumulated
	// separately and passed to descendants.
	UpdateFromLayoutWithOffset(offsetX, offsetY int)
}
