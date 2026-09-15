package render

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
)

// Renderable is a UI component in the retained scene graph. Widgets typically
// implement it by embedding BaseRenderable and overriding component-specific
// methods.
type Renderable interface {
	// Identity.

	// ID returns the renderable's string identifier.
	ID() string
	// SetID sets the string identifier.
	SetID(id string)
	// Num returns the numeric ID used by hit testing.
	Num() uint64

	// Tree structure.

	// Parent returns the scene graph parent, or nil for a root.
	Parent() Renderable
	// SetParent sets the scene graph parent.
	SetParent(r Renderable)
	// Children returns a copy of the direct children.
	Children() []Renderable
	// ChildrenRaw returns the underlying children slice for read-only iteration.
	ChildrenRaw() []Renderable
	// Add appends child or inserts it at the first optional index. It returns the
	// resulting child index.
	Add(child Renderable, index ...int) int
	// Remove detaches the direct child identified by id.
	Remove(id string)
	// InsertBefore inserts child immediately before anchor in the child list.
	InsertBefore(child, anchor Renderable)
	// Renderable returns the direct child identified by id, or nil.
	Renderable(id string) Renderable
	// FindDescendantByID returns the first matching node in a depth-first search
	// of this subtree, including the receiver.
	FindDescendantByID(id string) Renderable
	// ChildrenCount returns the number of direct children.
	ChildrenCount() int

	// State.

	// Visible reports whether the renderable is drawn and hit tested. A hidden
	// renderable retains its layout space.
	Visible() bool
	// SetVisible shows or hides this renderable.
	SetVisible(v bool)
	// IsDirty reports whether the renderable requires repainting.
	IsDirty() bool
	// Damage returns the region the renderable must repaint, or an empty
	// rectangle when it is unchanged.
	Damage() buffer.Rect
	// ClearDamage forgets a vacated region once it has been repainted.
	ClearDamage()
	// RequestRender marks the renderable dirty and requests a scene repaint.
	RequestRender()
	// IsDestroyed reports whether Destroy was called.
	IsDestroyed() bool

	// Layout.

	// LayoutNode returns the flexbox node configured by the renderable.
	LayoutNode() *layout.LayoutNode
	// X returns the resolved absolute x-coordinate.
	X() int
	// Y returns the resolved absolute y-coordinate.
	Y() int
	// Width returns the resolved width.
	Width() int
	// Height returns the resolved height.
	Height() int
	// SetOpacity sets opacity, where zero is transparent and one is opaque.
	SetOpacity(o float32)
	// Opacity returns the current opacity.
	Opacity() float32
	// TranslateX returns the horizontal offset inherited by descendants after
	// layout.
	TranslateX() int
	// TranslateY returns the vertical offset inherited by descendants after layout.
	TranslateY() int
	// SetTranslateX sets the horizontal post-layout offset.
	SetTranslateX(x int)
	// SetTranslateY sets the vertical post-layout offset.
	SetTranslateY(y int)
	// StickyOffsetX returns the renderable's horizontal sticky offset.
	StickyOffsetX() int
	// SetStickyOffsetX sets the horizontal sticky position offset.
	SetStickyOffsetX(x int)
	// StickyOffsetY returns the renderable's vertical sticky offset.
	StickyOffsetY() int
	// SetStickyOffsetY sets the vertical sticky position offset.
	SetStickyOffsetY(y int)

	// Rendering.

	// Render draws this renderable. Scene graph traversal schedules child
	// renderables separately.
	Render(buf *buffer.Buffer, deltaTime float64)
	// RenderSelf draws the renderable's own content and is the primary widget
	// override point.
	RenderSelf(buf *buffer.Buffer, deltaTime float64)
	// RenderBefore draws content that must precede RenderSelf.
	RenderBefore(buf *buffer.Buffer, deltaTime float64)
	// RenderAfter draws overlays or decorations after the renderable's content.
	RenderAfter(buf *buffer.Buffer, deltaTime float64)

	// Focus and keyboard input.

	// Focusable reports whether the renderable can receive keyboard focus.
	Focusable() bool
	// SetFocusable controls whether the renderable can receive keyboard focus.
	SetFocusable(f bool)
	// Focused reports whether the renderable currently has keyboard focus.
	Focused() bool
	// Focus requests keyboard focus for the renderable.
	Focus()
	// Blur removes keyboard focus from the renderable.
	Blur()
	// HandleKeyPress processes e and reports whether propagation should stop.
	HandleKeyPress(e *input.KeyEvent) bool
	// HandlePaste processes a bracketed paste event.
	HandlePaste(e *input.PasteEvent)

	// Mouse input.

	// ProcessMouseEvent handles a mouse event dispatched by the hit grid.
	ProcessMouseEvent(e *input.MouseEvent)

	// Selection.

	// Selectable reports whether the renderable participates in text selection.
	Selectable() bool
	// SetSelectable controls participation in text selection.
	SetSelectable(s bool)
	// ShouldStartSelection reports whether a mouse press at (x, y) should begin
	// text selection instead of another interaction.
	ShouldStartSelection(x, y int) bool
	// SelectedText returns the text selected within the renderable.
	SelectedText() string
	// HasSelection reports whether the renderable has an active text selection.
	HasSelection() bool

	// Lifecycle.

	// OnUpdate runs before layout for each frame in which the widget is updated.
	OnUpdate(deltaTime float64)
	// OnPostLayout runs after layout and before rendering.
	OnPostLayout(deltaTime float64)
	// OnResize handles a change to the renderable's resolved dimensions.
	OnResize(width, height int)
	// Destroy detaches and unregisters the renderable.
	Destroy()
	// DestroyRecursively destroys the renderable and every descendant.
	DestroyRecursively()

	// Context.

	// Context returns the attached renderer context or a no-op context.
	Context() Context
	// SetContext attaches a renderer context.
	SetContext(ctx Context)

	// Buffered rendering.

	// Buffered reports whether the renderable uses a private frame buffer.
	Buffered() bool
	// SetBuffered controls private frame-buffer rendering.
	SetBuffered(b bool)
	// FrameBuffer returns the private frame buffer, or nil if none is allocated.
	FrameBuffer() *buffer.Buffer

	// Continuous rendering.

	// Live reports whether the renderable requested continuous rendering.
	Live() bool
	// SetLive controls continuous rendering independent of dirty state.
	SetLive(l bool)
	// LiveCount returns the renderable's local live-render request count.
	LiveCount() int

	// Layout synchronization.

	// UpdateFromLayout copies the flexbox result and applies the renderable's own
	// translation and sticky offset.
	UpdateFromLayout()
	// UpdateFromLayoutWithOffset copies the flexbox result with inherited
	// translation offsets. It does not apply the renderable's own translation,
	// which traversal passes to descendants separately.
	UpdateFromLayoutWithOffset(offsetX, offsetY int)
}
