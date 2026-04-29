package render

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/textbuf"
)

// ---------------------------------------------------------------------------
// Global counter & registry
// ---------------------------------------------------------------------------

var nextNum uint64 // atomic counter

// renderablesByNum is used by the hit grid for O(1) mouse dispatch.
var renderablesByNum sync.Map // map[uint64]Renderable

// LookupRenderable returns the Renderable registered with the given num, or nil.
func LookupRenderable(num uint64) Renderable {
	v, ok := renderablesByNum.Load(num)
	if !ok {
		return nil
	}
	return v.(Renderable)
}

// ---------------------------------------------------------------------------
// RenderableOption
// ---------------------------------------------------------------------------

// RenderableOption is a functional option for configuring a BaseRenderable
// at construction time. Pass these to NewBaseRenderable or widget constructors
// that accept them (e.g. WithID, WithVisible, WithFocusable).
type RenderableOption func(*BaseRenderable)

// WithID sets the string identifier.
func WithID(id string) RenderableOption {
	return func(b *BaseRenderable) { b.id = id }
}

// WithVisible sets the initial visibility.
func WithVisible(v bool) RenderableOption {
	return func(b *BaseRenderable) { b.visible = v }
}

// WithFocusable sets the initial focusability.
func WithFocusable(f bool) RenderableOption {
	return func(b *BaseRenderable) { b.focusable = f }
}

// WithOpacity sets the initial opacity.
func WithOpacity(o float32) RenderableOption {
	return func(b *BaseRenderable) { b.opacity = o }
}

// WithBuffered enables or disables private frame buffer mode.
func WithBuffered(v bool) RenderableOption {
	return func(b *BaseRenderable) { b.buffered = v }
}

// WithLive enables or disables live (continuous rendering) mode.
func WithLive(l bool) RenderableOption {
	return func(b *BaseRenderable) { b.live = l }
}

// WithSelectable sets the initial selectability.
func WithSelectable(s bool) RenderableOption {
	return func(b *BaseRenderable) { b.selectable = s }
}

// --- Layout options ---

// WithWidth sets the layout node width.
func WithWidth(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Width = s }
}

// WithHeight sets the layout node height.
func WithHeight(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Height = s }
}

// WithFlexGrow sets the flex grow factor.
func WithFlexGrow(v int) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.FlexGrow = v }
}

// WithFlexShrink sets the flex shrink factor.
func WithFlexShrink(v int) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.FlexShrink = v }
}

// WithFlexDirection sets the flex direction.
func WithFlexDirection(d layout.FlexDirection) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.FlexDirection = d }
}

// WithJustifyContent sets the justify content mode.
func WithJustifyContent(j layout.JustifyContent) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.JustifyContent = j }
}

// WithAlignItems sets the align items mode.
func WithAlignItems(a layout.Align) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.AlignItems = a }
}

// WithPadding sets all four paddings to the same value.
func WithPadding(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.SetPadding(s) }
}

// WithMargin sets all four margins to the same value.
func WithMargin(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.SetMargin(s) }
}

// WithPosition sets the position type.
func WithPosition(p layout.Position) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Position = p }
}

// WithOverflow sets the overflow mode.
func WithOverflow(o layout.Overflow) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Overflow = o }
}

// WithFlexBasis sets the flex basis.
func WithFlexBasis(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.FlexBasis = s }
}

// WithAlignSelf sets the align self mode.
func WithAlignSelf(a layout.Align) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.AlignSelf = a }
}

// WithFlexWrap sets the flex wrap mode.
func WithFlexWrap(w layout.FlexWrap) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.FlexWrap = w }
}

// WithGap sets the gap between items.
func WithGap(v int) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Gap = v }
}

// WithMinWidth sets the minimum width.
func WithMinWidth(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.MinWidth = s }
}

// WithMaxWidth sets the maximum width.
func WithMaxWidth(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.MaxWidth = s }
}

// WithMinHeight sets the minimum height.
func WithMinHeight(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.MinHeight = s }
}

// WithMaxHeight sets the maximum height.
func WithMaxHeight(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.MaxHeight = s }
}

// WithTop sets the top position offset.
func WithTop(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Top = s }
}

// WithRight sets the right position offset.
func WithRight(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Right = s }
}

// WithBottom sets the bottom position offset.
func WithBottom(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Bottom = s }
}

// WithLeft sets the left position offset.
func WithLeft(s layout.Scalar) RenderableOption {
	return func(b *BaseRenderable) { b.layoutNode.Left = s }
}

// ---------------------------------------------------------------------------
// BaseRenderable
// ---------------------------------------------------------------------------

// BaseRenderable provides the default implementation of the Renderable
// interface. All widget structs embed *BaseRenderable, inheriting tree
// management, layout integration, focus handling, selection, and the
// standard render pipeline (RenderBefore → RenderSelf → children →
// RenderAfter). Widgets override only the methods they need.
type BaseRenderable struct {
	mu sync.Mutex

	num uint64 // unique auto-incremented ID
	id  string // string identifier

	// self holds the outer Renderable (e.g. *Box, *Text) so that internal
	// calls like renderContent can dispatch to the widget's RenderSelf
	// override.  Go embedding does NOT provide virtual dispatch: when
	// BaseRenderable.Render calls b.RenderSelf(), the receiver is
	// *BaseRenderable, not *Box.  Storing the interface value here lets us
	// call b.self.RenderSelf() which goes through the vtable and reaches
	// the correct concrete method.  The field is set automatically in Add().
	self Renderable

	parent   Renderable
	children []Renderable

	layoutNode *layout.LayoutNode

	// Computed absolute position (set during layout traversal)
	x, y, width, height   int
	prevWidth, prevHeight int // for resize detection

	// Configuration
	visible    bool
	focusable  bool
	focused    bool
	live       bool
	selectable bool
	buffered   bool
	destroyed  bool

	opacity       float32
	translateX    int
	translateY    int
	stickyOffsetX int // post-layout X offset applied by ScrollBox for sticky children
	stickyOffsetY int // post-layout Y offset applied by ScrollBox for sticky children

	dirty bool

	// Private frame buffer (when buffered=true)
	frameBuffer *buffer.Buffer

	// Live count tracking
	liveCount int

	// Context (set by renderer)
	ctx Context

	// Event handlers (set by widgets or users)
	OnMouseHandler        func(*input.MouseEvent)
	OnMouseDownHandler    func(*input.MouseEvent)
	OnMouseUpHandler      func(*input.MouseEvent)
	OnMouseMoveHandler    func(*input.MouseEvent)
	OnMouseDragHandler    func(*input.MouseEvent)
	OnMouseDragEndHandler func(*input.MouseEvent)
	OnMouseDropHandler    func(*input.MouseEvent)
	OnMouseOverHandler    func(*input.MouseEvent)
	OnMouseOutHandler     func(*input.MouseEvent)
	OnMouseScrollHandler  func(*input.MouseEvent)
	OnPasteHandler        func(*input.PasteEvent)
	OnKeyDownHandler      func(*input.KeyEvent)
	OnSizeChangeHandler   func()

	// Render hooks
	RenderBeforeHook func(*buffer.Buffer, float64)
	RenderAfterHook  func(*buffer.Buffer, float64)
	OnLifecyclePass  func()
}

// Compile-time interface check.
var _ Renderable = (*BaseRenderable)(nil)

// NewBaseRenderable creates a BaseRenderable with sensible defaults.
func NewBaseRenderable(opts ...RenderableOption) *BaseRenderable {
	num := atomic.AddUint64(&nextNum, 1)

	b := &BaseRenderable{
		num:        num,
		id:         fmt.Sprintf("renderable-%d", num),
		visible:    true,
		opacity:    1.0,
		focusable:  false,
		dirty:      true,
		layoutNode: layout.NewNode(),
		ctx:        noopContext,
	}

	for _, opt := range opts {
		opt(b)
	}

	// Register in global map
	renderablesByNum.Store(num, b)

	return b
}

// ---------------------------------------------------------------------------
// Identity
// ---------------------------------------------------------------------------

// ID returns the string identifier.
func (b *BaseRenderable) ID() string { return b.id }

// SetID sets the string identifier.
func (b *BaseRenderable) SetID(id string) { b.id = id }

// Num returns the unique numeric ID.
func (b *BaseRenderable) Num() uint64 { return b.num }

// SetSelf stores the outer Renderable interface value (e.g. *Box wrapping
// this BaseRenderable) so that internal method calls like renderContent can
// dispatch to the widget's RenderSelf override.  Called automatically by Add.
func (b *BaseRenderable) SetSelf(r Renderable) { b.self = r }

// Self returns the outer Renderable, or nil if not yet set.
func (b *BaseRenderable) Self() Renderable { return b.self }

// ---------------------------------------------------------------------------
// Tree Management
// ---------------------------------------------------------------------------

// Parent returns the parent renderable.
func (b *BaseRenderable) Parent() Renderable { return b.parent }

// SetParent sets the parent renderable.
func (b *BaseRenderable) SetParent(r Renderable) { b.parent = r }

// Children returns a copy of the direct children slice.
// Safe for callers that may modify the slice. For read-only iteration
// in hot paths, use ChildrenRaw() to avoid allocation.
func (b *BaseRenderable) Children() []Renderable {
	result := make([]Renderable, len(b.children))
	copy(result, b.children)
	return result
}

// ChildrenRaw returns the underlying children slice without copying.
// The caller must NOT modify the returned slice. Use this in hot
// render/layout paths to avoid 183k+ allocations per frame.
func (b *BaseRenderable) ChildrenRaw() []Renderable {
	return b.children
}

// ChildrenCount returns the number of direct children.
func (b *BaseRenderable) ChildrenCount() int {
	return len(b.children)
}

// Add adds a child at the optional index (default: append).
// Sets child's parent and propagates the render context.
// Returns the insertion index.
func (b *BaseRenderable) Add(child Renderable, index ...int) int {
	if child == nil {
		return -1
	}

	// Re-register child in the global map with the correct outer type.
	// NewBaseRenderable() registers the embedded *BaseRenderable, but the
	// caller passes the outer widget type (e.g., *widget.Input) here.
	// This mirrors JS/TS where `this` in a base constructor is the subclass.
	renderablesByNum.Store(child.Num(), child)

	// Store the outer Renderable so internal dispatch (e.g. renderContent
	// calling RenderSelf) goes through the interface vtable and reaches
	// the widget's override instead of the BaseRenderable no-op.
	type selfSettable interface{ SetSelf(Renderable) }
	if ss, ok := child.(selfSettable); ok {
		ss.SetSelf(child)
	}

	// Remove from old parent if any
	if oldParent := child.Parent(); oldParent != nil {
		oldParent.Remove(child.ID())
	}

	child.SetParent(b)
	child.SetContext(b.ctx)

	// Also wire up the layout node tree
	b.layoutNode.AddChild(child.LayoutNode())

	idx := len(b.children)
	if len(index) > 0 && index[0] >= 0 && index[0] <= len(b.children) {
		idx = index[0]
		// Insert at position
		b.children = append(b.children, nil)
		copy(b.children[idx+1:], b.children[idx:])
		b.children[idx] = child
	} else {
		b.children = append(b.children, child)
	}

	b.dirty = true
	b.ctx.RequestRender()
	return idx
}

// Remove removes a child by ID.
func (b *BaseRenderable) Remove(id string) {
	for i, child := range b.children {
		if child.ID() == id {
			// Remove from layout tree
			b.layoutNode.RemoveChild(child.LayoutNode())

			child.SetParent(nil)
			b.children = append(b.children[:i], b.children[i+1:]...)
			b.dirty = true
			b.ctx.RequestRender()
			return
		}
	}
}

// InsertBefore inserts child before the anchor child.
// If anchor is nil or not found, appends.
func (b *BaseRenderable) InsertBefore(child, anchor Renderable) {
	if anchor == nil {
		b.Add(child)
		return
	}

	for i, c := range b.children {
		if c.ID() == anchor.ID() {
			b.Add(child, i)
			return
		}
	}

	// Anchor not found, append
	b.Add(child)
}

// Renderable returns a direct child by ID, or nil.
func (b *BaseRenderable) Renderable(id string) Renderable {
	for _, child := range b.children {
		if child.ID() == id {
			return child
		}
	}
	return nil
}

// FindDescendantByID searches the entire subtree for a renderable by ID.
func (b *BaseRenderable) FindDescendantByID(id string) Renderable {
	if b.id == id {
		return b
	}
	for _, child := range b.children {
		if found := child.FindDescendantByID(id); found != nil {
			return found
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

// Visible returns whether this renderable is visible.
func (b *BaseRenderable) Visible() bool { return b.visible }

// SetVisible sets visibility. When hidden, the node still participates in
// layout (like CSS visibility:hidden). Use SetGone to remove from layout.
func (b *BaseRenderable) SetVisible(v bool) {
	if b.visible != v {
		b.visible = v
		if v {
			b.layoutNode.Visibility = layout.Visible
		} else {
			b.layoutNode.Visibility = layout.Hidden
		}
		b.layoutNode.MarkDirty()
		b.RequestRender()
	}
}

// SetGone removes this renderable from both layout and rendering.
// The node occupies zero space (like Android View.GONE).
func (b *BaseRenderable) SetGone(g bool) {
	if g {
		b.visible = false
		b.layoutNode.Visibility = layout.Gone
	} else {
		b.visible = true
		b.layoutNode.Visibility = layout.Visible
	}
	b.layoutNode.MarkDirty()
	b.RequestRender()
}

// IsDirty returns whether this renderable needs re-rendering.
func (b *BaseRenderable) IsDirty() bool { return b.dirty }

// RequestRender marks this renderable as dirty and requests a render frame.
func (b *BaseRenderable) RequestRender() {
	b.dirty = true
	b.ctx.RequestRender()
}

// IsDestroyed returns whether this renderable has been destroyed.
func (b *BaseRenderable) IsDestroyed() bool { return b.destroyed }

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

// LayoutNode returns the underlying layout node.
func (b *BaseRenderable) LayoutNode() *layout.LayoutNode { return b.layoutNode }

// X returns the computed absolute x position.
func (b *BaseRenderable) X() int { return b.x }

// Y returns the computed absolute y position.
func (b *BaseRenderable) Y() int { return b.y }

// Width returns the computed width.
func (b *BaseRenderable) Width() int { return b.width }

// Height returns the computed height.
func (b *BaseRenderable) Height() int { return b.height }

// SetBounds sets the absolute position and size directly. This bypasses
// layout computation and is intended for tests or manual positioning.
func (b *BaseRenderable) SetBounds(x, y, w, h int) {
	b.x = x
	b.y = y
	b.width = w
	b.height = h
}

// SetOpacity sets the opacity (0.0 - 1.0).
func (b *BaseRenderable) SetOpacity(o float32) {
	if o < 0 {
		o = 0
	}
	if o > 1 {
		o = 1
	}
	b.opacity = o
}

// Opacity returns the opacity.
func (b *BaseRenderable) Opacity() float32 { return b.opacity }

// TranslateX returns the x translation offset.
func (b *BaseRenderable) TranslateX() int { return b.translateX }

// TranslateY returns the y translation offset.
func (b *BaseRenderable) TranslateY() int { return b.translateY }

// SetTranslateX sets the x translation offset.
func (b *BaseRenderable) SetTranslateX(x int) { b.translateX = x }

// SetTranslateY sets the y translation offset.
func (b *BaseRenderable) SetTranslateY(y int) { b.translateY = y }

// StickyOffsetX returns the horizontal sticky position offset.
func (b *BaseRenderable) StickyOffsetX() int { return b.stickyOffsetX }

// SetStickyOffsetX sets the horizontal sticky position offset.
func (b *BaseRenderable) SetStickyOffsetX(x int) { b.stickyOffsetX = x }

// StickyOffsetY returns the vertical sticky position offset.
func (b *BaseRenderable) StickyOffsetY() int { return b.stickyOffsetY }

// SetStickyOffsetY sets the vertical sticky position offset.
func (b *BaseRenderable) SetStickyOffsetY(y int) { b.stickyOffsetY = y }

// UpdateFromLayout reads computed layout values and updates absolute position.
// The layout engine (ComputeLayout / layoutFlexContainer) already produces
// absolute Computed.X/Y values (accumulated from the root). We must NOT add
// the parent's position again — that would double-count for nested widgets.
//
// NOTE: This legacy method applies the node's own translate directly. For
// correct scroll behaviour use UpdateFromLayoutWithOffset which separates
// the node's translate (applied to descendants) from inherited offsets.
func (b *BaseRenderable) UpdateFromLayout() {
	b.UpdateFromLayoutWithOffset(b.translateX, b.translateY)
}

// UpdateFromLayoutWithOffset reads computed layout values and updates
// absolute position using an inherited offset from ancestor translates.
// The node's own translateX/Y is NOT added here — collectCommands
// accumulates it and passes it to descendant calls instead. This ensures
// a scroll container's translate shifts its children without moving the
// container (or its scissor rect) itself.
func (b *BaseRenderable) UpdateFromLayoutWithOffset(offsetX, offsetY int) {
	computed := b.layoutNode.Computed

	b.x = computed.X + offsetX + b.stickyOffsetX
	b.y = computed.Y + offsetY + b.stickyOffsetY
	b.width = computed.Width
	b.height = computed.Height

	// Check for size change
	if b.width != b.prevWidth || b.height != b.prevHeight {
		b.prevWidth = b.width
		b.prevHeight = b.height
		// Dispatch through self (the outer Renderable) so widget
		// overrides of OnResize are reached. Same pattern as
		// renderContent → self.RenderSelf.
		if b.self != nil {
			b.self.OnResize(b.width, b.height)
		} else {
			b.OnResize(b.width, b.height)
		}
		if b.OnSizeChangeHandler != nil {
			b.OnSizeChangeHandler()
		}
	}
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// Render executes the render pipeline for this renderable.
func (b *BaseRenderable) Render(buf *buffer.Buffer, deltaTime float64) {
	if !b.visible {
		return
	}

	// If buffered mode, render into private buffer then blit
	if b.buffered && b.frameBuffer != nil {
		b.frameBuffer.Clear()
		b.renderContent(b.frameBuffer, deltaTime)
		buf.DrawFrameBuffer(b.x, b.y, b.frameBuffer)
	} else {
		b.renderContent(buf, deltaTime)
	}

	// Register in hit grid
	b.ctx.AddToHitGrid(b.x, b.y, b.width, b.height, b.num)

	b.dirty = false
}

// renderContent runs the hooks + RenderSelf sequence.
// Dispatches through b.self (when set) so the concrete widget's RenderSelf
// override is called instead of the BaseRenderable no-op.
func (b *BaseRenderable) renderContent(buf *buffer.Buffer, deltaTime float64) {
	if b.RenderBeforeHook != nil {
		b.RenderBeforeHook(buf, deltaTime)
	}

	if b.self != nil {
		b.self.RenderSelf(buf, deltaTime)
	} else {
		b.RenderSelf(buf, deltaTime)
	}

	if b.RenderAfterHook != nil {
		b.RenderAfterHook(buf, deltaTime)
	}
}

// RenderSelf is the override point for widgets. Default: no-op.
func (b *BaseRenderable) RenderSelf(buf *buffer.Buffer, deltaTime float64) {}

// RenderBefore calls the RenderBeforeHook if set.
func (b *BaseRenderable) RenderBefore(buf *buffer.Buffer, deltaTime float64) {
	if b.RenderBeforeHook != nil {
		b.RenderBeforeHook(buf, deltaTime)
	}
}

// RenderAfter calls the RenderAfterHook if set.
func (b *BaseRenderable) RenderAfter(buf *buffer.Buffer, deltaTime float64) {
	if b.RenderAfterHook != nil {
		b.RenderAfterHook(buf, deltaTime)
	}
}

// ---------------------------------------------------------------------------
// Focus
// ---------------------------------------------------------------------------

// Focusable returns whether this renderable can receive focus.
func (b *BaseRenderable) Focusable() bool { return b.focusable }

// SetFocusable sets whether this renderable can receive focus.
func (b *BaseRenderable) SetFocusable(f bool) { b.focusable = f }

// Focused returns whether this renderable currently has focus.
func (b *BaseRenderable) Focused() bool { return b.focused }

// Focus gives this renderable keyboard focus.
func (b *BaseRenderable) Focus() {
	if !b.focusable {
		return
	}
	b.focused = true
	b.ctx.FocusRenderable(b)
}

// Blur removes keyboard focus from this renderable.
func (b *BaseRenderable) Blur() {
	b.focused = false
}

// HandleKeyPress handles a keyboard event. Returns true if consumed.
// Default: invokes OnKeyDownHandler if set, otherwise returns false.
func (b *BaseRenderable) HandleKeyPress(e *input.KeyEvent) bool {
	if b.OnKeyDownHandler != nil {
		b.OnKeyDownHandler(e)
		return e.PropagationStopped()
	}
	return false
}

// HandlePaste handles a paste event.
func (b *BaseRenderable) HandlePaste(e *input.PasteEvent) {
	if b.OnPasteHandler != nil {
		b.OnPasteHandler(e)
	}
}

// ---------------------------------------------------------------------------
// Mouse
// ---------------------------------------------------------------------------

// ProcessMouseEvent dispatches a mouse event to type-specific handlers,
// then bubbles to parent if propagation is not stopped.
func (b *BaseRenderable) ProcessMouseEvent(e *input.MouseEvent) {
	// Catch-all handler
	if b.OnMouseHandler != nil {
		b.OnMouseHandler(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Type-specific handlers
	switch e.Type {
	case input.MouseDown:
		if b.OnMouseDownHandler != nil {
			b.OnMouseDownHandler(e)
		}
	case input.MouseUp:
		if b.OnMouseUpHandler != nil {
			b.OnMouseUpHandler(e)
		}
	case input.MouseMove:
		if b.OnMouseMoveHandler != nil {
			b.OnMouseMoveHandler(e)
		}
	case input.MouseDrag:
		if b.OnMouseDragHandler != nil {
			b.OnMouseDragHandler(e)
		}
	case input.MouseDragEnd:
		if b.OnMouseDragEndHandler != nil {
			b.OnMouseDragEndHandler(e)
		}
	case input.MouseDrop:
		if b.OnMouseDropHandler != nil {
			b.OnMouseDropHandler(e)
		}
	case input.MouseOver:
		if b.OnMouseOverHandler != nil {
			b.OnMouseOverHandler(e)
		}
	case input.MouseOut:
		if b.OnMouseOutHandler != nil {
			b.OnMouseOutHandler(e)
		}
	case input.MouseScroll:
		if b.OnMouseScrollHandler != nil {
			b.OnMouseScrollHandler(e)
		}
	}

	// Bubble to parent if not stopped
	if !e.PropagationStopped() && b.parent != nil {
		b.parent.ProcessMouseEvent(e)
	}
}

// ---------------------------------------------------------------------------
// Selection
// ---------------------------------------------------------------------------

// Selectable returns whether this renderable supports text selection.
func (b *BaseRenderable) Selectable() bool { return b.selectable }

// SetSelectable sets whether this renderable supports text selection.
func (b *BaseRenderable) SetSelectable(s bool) { b.selectable = s }

// ShouldStartSelection returns whether a selection should start at (x, y).
// Default: returns selectable state.
func (b *BaseRenderable) ShouldStartSelection(x, y int) bool {
	return b.selectable
}

// SelectedText returns the currently selected text. Default: empty.
func (b *BaseRenderable) SelectedText() string { return "" }

// HasSelection returns whether there is an active selection. Default: false.
func (b *BaseRenderable) HasSelection() bool { return false }

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// OnUpdate is called each frame before layout. Default: no-op.
// Widgets override this to modify layout properties (sizes, flex, etc.)
// before the layout engine runs.
func (b *BaseRenderable) OnUpdate(deltaTime float64) {}

// OnPostLayout is called each frame after CalculateLayout has resolved
// all Computed positions and sizes. Widgets override this when they need
// to react to fresh layout results — e.g. recalculating content
// dimensions or adjusting scroll position. Default: no-op.
func (b *BaseRenderable) OnPostLayout(deltaTime float64) {}

// OnResize is called when the computed size changes. Default: no-op.
func (b *BaseRenderable) OnResize(width, height int) {}

// Destroy removes this renderable from the scene graph and unregisters it.
func (b *BaseRenderable) Destroy() {
	b.destroyed = true

	// Remove from parent
	if b.parent != nil {
		b.parent.Remove(b.id)
	}

	// Unregister from global map
	renderablesByNum.Delete(b.num)
}

// DestroyRecursively destroys this renderable and all descendants.
func (b *BaseRenderable) DestroyRecursively() {
	// Destroy children first (copy slice to avoid mutation during iteration)
	children := make([]Renderable, len(b.children))
	copy(children, b.children)
	for _, child := range children {
		child.DestroyRecursively()
	}
	b.children = nil

	b.Destroy()
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

// Context returns the render context.
func (b *BaseRenderable) Context() Context { return b.ctx }

// SetContext sets the render context and propagates to children.
// A nil ctx is replaced with the package-level no-op context so that
// b.ctx is never nil.
func (b *BaseRenderable) SetContext(ctx Context) {
	if ctx == nil {
		ctx = noopContext
	}
	b.ctx = ctx
	for _, child := range b.children {
		child.SetContext(ctx)
	}
}

// ---------------------------------------------------------------------------
// Width helpers
// ---------------------------------------------------------------------------

// StringWidth returns the display width of a string using grapheme-cluster-aware
// width calculation.
func (b *BaseRenderable) StringWidth(s string) int {
	return textbuf.StringWidth(s)
}

// ---------------------------------------------------------------------------
// Buffered mode
// ---------------------------------------------------------------------------

// Buffered returns whether this renderable uses a private frame buffer.
func (b *BaseRenderable) Buffered() bool { return b.buffered }

// SetBuffered enables or disables private frame buffer mode.
func (b *BaseRenderable) SetBuffered(v bool) {
	b.buffered = v
	if v && b.frameBuffer == nil && b.width > 0 && b.height > 0 {
		b.frameBuffer = buffer.New(b.width, b.height)
	}
}

// FrameBuffer returns the private frame buffer (nil if not buffered).
func (b *BaseRenderable) FrameBuffer() *buffer.Buffer { return b.frameBuffer }

// ---------------------------------------------------------------------------
// Live mode
// ---------------------------------------------------------------------------

// Live returns whether this renderable is in continuous rendering mode.
func (b *BaseRenderable) Live() bool { return b.live }

// SetLive enables or disables live mode.
func (b *BaseRenderable) SetLive(l bool) {
	if b.live == l {
		return
	}
	b.live = l
	if l {
		b.liveCount++
		b.ctx.RequestLive()
	} else {
		b.liveCount--
		if b.liveCount <= 0 {
			b.liveCount = 0
			b.ctx.DropLive()
		}
	}
}

// LiveCount returns the live mode reference count.
func (b *BaseRenderable) LiveCount() int { return b.liveCount }

// ---------------------------------------------------------------------------
// AnimationTarget implementation
// ---------------------------------------------------------------------------

// Property returns an animatable property value by name.
// Supported properties: "opacity", "translateX", "translateY".
func (b *BaseRenderable) Property(name string) float64 {
	switch name {
	case "opacity":
		return float64(b.opacity)
	case "translateX":
		return float64(b.translateX)
	case "translateY":
		return float64(b.translateY)
	default:
		return 0
	}
}

// SetProperty sets an animatable property value by name and marks the
// renderable as dirty. Supported: "opacity", "translateX", "translateY".
func (b *BaseRenderable) SetProperty(name string, value float64) {
	switch name {
	case "opacity":
		b.SetOpacity(float32(value))
	case "translateX":
		b.translateX = int(math.Round(value))
	case "translateY":
		b.translateY = int(math.Round(value))
	default:
		return
	}
	b.RequestRender()
}
