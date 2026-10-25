package render

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
)

var nextNum atomic.Uint64 // Assigns numeric hit-grid IDs.

// renderablesByNum resolves hit-grid IDs to renderables.
//
// Entries are weak, so registration never keeps a renderable alive: a discarded
// widget tree is collected whether or not anything calls Destroy. A cleanup
// removes the key once its renderable is gone.
var renderablesByNum sync.Map // map[uint64]weak.Pointer[BaseRenderable]

// LookupRenderable returns the renderable registered under num, or nil when
// nothing is registered or the renderable has been collected.
func LookupRenderable(num uint64) Renderable {
	v, ok := renderablesByNum.Load(num)
	if !ok {
		return nil
	}
	b := v.(weak.Pointer[BaseRenderable]).Value()
	if b == nil {
		return nil
	}
	// Callers need the concrete widget to reach overridden methods.
	if b.self != nil {
		return b.self
	}
	return b
}

// Option configures a BaseRenderable during construction.
type Option func(*BaseRenderable)

// WithID sets the initial string identifier.
func WithID(id string) Option {
	return func(b *BaseRenderable) { b.id = id }
}

// WithVisible sets whether the renderable is initially visible. A hidden one
// keeps its layout space, as SetVisible(false) leaves it.
func WithVisible(v bool) Option {
	return func(b *BaseRenderable) {
		if v {
			b.layoutNode.Visibility = layout.Visible
		} else {
			b.layoutNode.Visibility = layout.Hidden
		}
	}
}

// WithFocusable sets whether the renderable can initially receive focus.
func WithFocusable(f bool) Option {
	return func(b *BaseRenderable) { b.focusable = f }
}

// WithSelectable sets whether the renderable initially supports text selection.
func WithSelectable(s bool) Option {
	return func(b *BaseRenderable) { b.selectable = s }
}

// WithWidth sets the preferred layout width.
func WithWidth(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Width = s }
}

// WithHeight sets the preferred layout height.
func WithHeight(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Height = s }
}

// WithFlexGrow sets the layout growth factor.
func WithFlexGrow(v int) Option {
	return func(b *BaseRenderable) { b.layoutNode.FlexGrow = v }
}

// WithFlexShrink sets the layout shrink factor.
func WithFlexShrink(v int) Option {
	return func(b *BaseRenderable) { b.layoutNode.FlexShrink = v }
}

// WithFlexDirection sets the layout main axis and item order.
func WithFlexDirection(d layout.FlexDirection) Option {
	return func(b *BaseRenderable) { b.layoutNode.FlexDirection = d }
}

// WithJustifyContent sets main-axis item distribution.
func WithJustifyContent(j layout.JustifyContent) Option {
	return func(b *BaseRenderable) { b.layoutNode.JustifyContent = j }
}

// WithAlignItems sets cross-axis item alignment.
func WithAlignItems(a layout.Align) Option {
	return func(b *BaseRenderable) { b.layoutNode.AlignItems = a }
}

// WithPadding sets padding on all four sides.
func WithPadding(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.SetPadding(s) }
}

// WithMargin sets all four margins.
func WithMargin(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.SetMargin(s) }
}

// WithPosition sets whether the renderable participates in flex flow.
func WithPosition(p layout.Position) Option {
	return func(b *BaseRenderable) { b.layoutNode.Position = p }
}

// WithOverflow sets how content outside the layout bounds is handled.
func WithOverflow(o layout.Overflow) Option {
	return func(b *BaseRenderable) { b.layoutNode.Overflow = o }
}

// WithFlexBasis sets the initial main-axis size.
func WithFlexBasis(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.FlexBasis = s }
}

// WithAlignSelf sets the renderable's cross-axis alignment override.
func WithAlignSelf(a layout.Align) Option {
	return func(b *BaseRenderable) { b.layoutNode.AlignSelf = a }
}

// WithFlexWrap sets the line-wrapping mode.
func WithFlexWrap(w layout.FlexWrap) Option {
	return func(b *BaseRenderable) { b.layoutNode.FlexWrap = w }
}

// WithGap sets the fallback gap between flex items.
func WithGap(v int) Option {
	return func(b *BaseRenderable) { b.layoutNode.Gap = v }
}

// WithMinWidth sets the minimum width.
func WithMinWidth(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.MinWidth = s }
}

// WithMaxWidth sets the maximum width.
func WithMaxWidth(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.MaxWidth = s }
}

// WithMinHeight sets the minimum height.
func WithMinHeight(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.MinHeight = s }
}

// WithMaxHeight sets the maximum height.
func WithMaxHeight(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.MaxHeight = s }
}

// WithTop sets the top positioning offset.
func WithTop(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Top = s }
}

// WithRight sets the right positioning offset.
func WithRight(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Right = s }
}

// WithBottom sets the bottom positioning offset.
func WithBottom(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Bottom = s }
}

// WithLeft sets the left positioning offset.
func WithLeft(s layout.Scalar) Option {
	return func(b *BaseRenderable) { b.layoutNode.Left = s }
}

// BaseRenderable implements shared scene graph, layout, input, and rendering
// behavior for widgets. Concrete widgets embed it and override only the methods
// they need.
type BaseRenderable struct {
	mu sync.Mutex

	num uint64 // The process-unique numeric ID.
	id  string // The user-facing string identifier.

	// self stores the concrete outer widget. Calling through this interface
	// reaches methods overridden by a type that embeds BaseRenderable.
	self Renderable

	parent   Renderable
	children []Renderable

	layoutNode *layout.Node

	// Resolved geometry copied from layout traversal.
	x, y, width, height   int
	prevWidth, prevHeight int         // The dimensions used to detect resize callbacks.
	paintRect             buffer.Rect // The region covered by the most recent paint.

	// Configuration and state.
	focusable  bool
	focused    bool
	selectable bool
	destroyed  bool

	translateX    int
	translateY    int
	stickyOffsetX int // A horizontal post-layout offset for sticky positioning.
	stickyOffsetY int // A vertical post-layout offset for sticky positioning.

	dirty      bool
	childDirty bool // Reports whether a descendant changed since the last frame.

	// Renderer that hosts b, or noopHost while detached.
	host Host

	// Optional input handlers.

	// OnMouseHandler runs for every mouse event first; stopping propagation
	// skips the handler for the event's type.
	OnMouseHandler func(*input.MouseEvent)
	// OnMouseDownHandler runs for MouseDown events.
	OnMouseDownHandler func(*input.MouseEvent)
	// OnMouseUpHandler runs for MouseUp events.
	OnMouseUpHandler func(*input.MouseEvent)
	// OnMouseMoveHandler runs for MouseMove events.
	OnMouseMoveHandler func(*input.MouseEvent)
	// OnMouseDragHandler runs for MouseDrag events.
	OnMouseDragHandler func(*input.MouseEvent)
	// OnMouseDragEndHandler runs for MouseDragEnd events.
	OnMouseDragEndHandler func(*input.MouseEvent)
	// OnMouseDropHandler runs for MouseDrop events.
	OnMouseDropHandler func(*input.MouseEvent)
	// OnMouseOverHandler runs for MouseOver events.
	OnMouseOverHandler func(*input.MouseEvent)
	// OnMouseOutHandler runs for MouseOut events.
	OnMouseOutHandler func(*input.MouseEvent)
	// OnMouseScrollHandler runs for MouseScroll events.
	OnMouseScrollHandler func(*input.MouseEvent)
	// OnPasteHandler runs for pasted text.
	OnPasteHandler func(*input.PasteEvent)
	// OnKeyDownHandler runs for key events; stopping propagation keeps a key
	// from the renderable's ancestors.
	OnKeyDownHandler func(*input.KeyEvent)
	// OnSizeChangeHandler runs after layout changes the renderable's size.
	OnSizeChangeHandler func()

	// Optional rendering and lifecycle hooks.

	// RenderBeforeHook runs before RenderSelf, with the frame buffer and the
	// seconds since the last frame.
	RenderBeforeHook func(*buffer.Buffer, time.Duration)
	// RenderAfterHook runs after RenderSelf, with the same arguments.
	RenderAfterHook func(*buffer.Buffer, time.Duration)
	// OnLifecyclePass runs before each layout once the renderable is
	// registered with the root's RegisterLifecyclePass.
	OnLifecyclePass func()
}

// Verify that BaseRenderable implements Renderable.
var _ Renderable = (*BaseRenderable)(nil)

// NewBaseRenderable returns a registered renderable with default layout and
// rendering state.
func NewBaseRenderable(opts ...Option) *BaseRenderable {
	num := nextNum.Add(1)

	b := &BaseRenderable{
		num:        num,
		id:         fmt.Sprintf("renderable-%d", num),
		focusable:  false,
		dirty:      true,
		layoutNode: layout.NewNode(),
		host:       noopHost,
	}

	for _, opt := range opts {
		opt(b)
	}

	// Registration allows the hit grid to resolve the numeric ID. The cleanup
	// runs after b is collected and cannot reference it, so neither the entry
	// nor the cleanup keeps b alive.
	renderablesByNum.Store(num, weak.Make(b))
	runtime.AddCleanup(b, func(num uint64) { renderablesByNum.Delete(num) }, num)

	return b
}

// ID returns the string identifier.
func (b *BaseRenderable) ID() string { return b.id }

// SetID sets the string identifier.
func (b *BaseRenderable) SetID(id string) { b.id = id }

// Num returns the process-unique numeric ID.
func (b *BaseRenderable) Num() uint64 { return b.num }

// SetSelf stores the concrete outer widget used for overridden method dispatch.
func (b *BaseRenderable) SetSelf(r Renderable) { b.self = r }

// Self returns the concrete outer widget, or nil until SetSelf or Add associates
// one with b.
func (b *BaseRenderable) Self() Renderable { return b.self }

// Parent returns the scene graph parent.
func (b *BaseRenderable) Parent() Renderable { return b.parent }

// SetParent sets the scene graph parent.
func (b *BaseRenderable) SetParent(r Renderable) { b.parent = r }

// Children returns a copy of the direct children.
func (b *BaseRenderable) Children() []Renderable {
	result := make([]Renderable, len(b.children))
	copy(result, b.children)
	return result
}

// ChildrenRaw returns the underlying children slice for read-only iteration.
func (b *BaseRenderable) ChildrenRaw() []Renderable {
	return b.children
}

// ChildrenCount returns the number of direct children.
func (b *BaseRenderable) ChildrenCount() int {
	return len(b.children)
}

// Add reparents child to b, propagates b's host, and inserts child at the
// first optional index. An absent or invalid index appends child. Add returns
// the resulting index, or -1 for a nil child.
func (b *BaseRenderable) Add(child Renderable, index ...int) int {
	if child == nil {
		return -1
	}

	// Store the concrete widget so internal calls reach overridden methods.
	// LookupRenderable reads it too, which is why the registry needs no entry
	// of its own for the concrete type.
	type selfSettable interface{ SetSelf(Renderable) }
	if ss, ok := child.(selfSettable); ok {
		ss.SetSelf(child)
	}

	// Keep the render and layout trees single-parented.
	if oldParent := child.Parent(); oldParent != nil {
		oldParent.Remove(child.ID())
	}

	child.SetParent(b)
	child.SetHost(b.host)

	idx := len(b.children)
	if len(index) > 0 && index[0] >= 0 && index[0] <= len(b.children) {
		idx = index[0]
		// Preserve identical ordering in the render and layout trees.
		b.layoutNode.AddChildAt(child.LayoutNode(), idx)
		b.children = append(b.children, nil)
		copy(b.children[idx+1:], b.children[idx:])
		b.children[idx] = child
	} else {
		b.layoutNode.AddChild(child.LayoutNode())
		b.children = append(b.children, child)
	}

	// The new child reports its own region, so b has nothing to add and stays
	// clean. Only the record reaches the ancestors, so traversal descends here.
	b.MarkChildDirty()
	b.host.RequestRender()
	return idx
}

// Remove detaches the first direct child identified by id.
func (b *BaseRenderable) Remove(id string) {
	for i, child := range b.children {
		if child.ID() == id {
			// Keep the layout tree synchronized with the render tree.
			b.layoutNode.RemoveChild(child.LayoutNode())

			child.SetParent(nil)
			b.children = append(b.children[:i], b.children[i+1:]...)

			// A sticky offset pins a child to the viewport of the scroll
			// container that resolved it, so it is meaningless once the child
			// leaves. A parent that does not scroll never recomputes it.
			child.SetStickyOffsetX(0)
			child.SetStickyOffsetY(0)

			// The child painted inside b, so b already carries the region it
			// vacated and repaints it as a dirty renderable.
			b.RequestRender()
			return
		}
	}
}

// InsertBefore inserts child immediately before anchor. It appends child when
// anchor is nil or not a direct child.
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

	b.Add(child)
}

// Renderable returns the direct child identified by id, or nil.
func (b *BaseRenderable) Renderable(id string) Renderable {
	for _, child := range b.children {
		if child.ID() == id {
			return child
		}
	}
	return nil
}

// FindDescendantByID returns the first matching node in a depth-first search of
// b's subtree, including b.
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

// Visible reports whether b is drawn and hit tested: whether its layout
// node's Visibility is layout.Visible.
func (b *BaseRenderable) Visible() bool { return b.layoutNode.Visibility == layout.Visible }

// Gone reports whether b takes part in neither layout nor rendering: whether
// its layout node's Visibility is layout.Gone.
func (b *BaseRenderable) Gone() bool { return b.layoutNode.Visibility == layout.Gone }

// SetVisible controls rendering and hit testing while preserving layout space.
// Hiding a visible b makes it layout.Hidden; hiding a gone one leaves it gone.
// Use SetGone to remove b from layout.
func (b *BaseRenderable) SetVisible(v bool) {
	if b.Visible() == v {
		return
	}
	if v {
		b.layoutNode.Visibility = layout.Visible
	} else {
		b.layoutNode.Visibility = layout.Hidden
	}
	b.layoutNode.MarkDirty()
	b.RequestRender()
}

// SetGone controls whether b participates in layout and rendering. Bringing
// it back makes it visible.
func (b *BaseRenderable) SetGone(g bool) {
	if g {
		b.layoutNode.Visibility = layout.Gone
	} else {
		b.layoutNode.Visibility = layout.Visible
	}
	b.layoutNode.MarkDirty()
	b.RequestRender()
}

// IsDirty reports whether b requires repainting.
func (b *BaseRenderable) IsDirty() bool { return b.dirty }

// Rect returns b's resolved geometry.
func (b *BaseRenderable) Rect() buffer.Rect {
	return buffer.Rect{X: b.x, Y: b.y, Width: b.width, Height: b.height}
}

// Damage returns the region b must repaint, or an empty rectangle when b is
// unchanged. A renderable that moved or resized damages both the region it
// occupied and the region it now occupies.
//
// A hidden or destroyed renderable damages the region it last painted, which
// encloses its descendants because a subtree paints inside its root. A detached
// one is beyond the reach of traversal, and its parent encloses it too.
func (b *BaseRenderable) Damage() buffer.Rect {
	switch {
	case b.destroyed, !b.Visible():
		return b.paintRect
	case b.dirty:
		return b.paintRect.Union(b.Rect())
	default:
		return buffer.Rect{}
	}
}

// ClearDamage forgets the region b vacated, once that region has been
// repainted. Painting clears the damage of a visible renderable, but a hidden
// one is never painted again.
func (b *BaseRenderable) ClearDamage() {
	if b.destroyed || !b.Visible() {
		b.paintRect = buffer.Rect{}
	}
}

// RequestRender marks b dirty and requests a scene repaint.
func (b *BaseRenderable) RequestRender() {
	b.MarkDirty()
	b.host.RequestRender()
}

// MarkDirty marks b for repainting and records the change on b's ancestors, so
// scene traversal descends to b.
//
// Only b repaints. An ancestor carries the record alone, which adds no damage,
// because marking it dirty would repaint everything it covers.
func (b *BaseRenderable) MarkDirty() {
	b.dirty = true
	if b.parent != nil {
		b.parent.MarkChildDirty()
	}
}

// ChildDirty reports whether a descendant changed since the last frame.
func (b *BaseRenderable) ChildDirty() bool { return b.childDirty }

// MarkChildDirty records that a descendant changed. The climb stops at the
// first ancestor already carrying the record, whose own ancestors carry it too.
func (b *BaseRenderable) MarkChildDirty() {
	if b.childDirty {
		return
	}
	b.childDirty = true
	if b.parent != nil {
		b.parent.MarkChildDirty()
	}
}

// ClearChildDirty forgets the record once scene traversal has used it.
func (b *BaseRenderable) ClearChildDirty() { b.childDirty = false }

// IsDestroyed reports whether Destroy was called.
func (b *BaseRenderable) IsDestroyed() bool { return b.destroyed }

// LayoutNode returns b's flexbox node.
func (b *BaseRenderable) LayoutNode() *layout.Node { return b.layoutNode }

// X returns the resolved absolute x-coordinate.
func (b *BaseRenderable) X() int { return b.x }

// Y returns the resolved absolute y-coordinate.
func (b *BaseRenderable) Y() int { return b.y }

// Width returns the resolved width.
func (b *BaseRenderable) Width() int { return b.width }

// Height returns the resolved height.
func (b *BaseRenderable) Height() int { return b.height }

// SetBounds directly sets b's resolved geometry without running layout.
func (b *BaseRenderable) SetBounds(x, y, w, h int) {
	b.x = x
	b.y = y
	b.width = w
	b.height = h
}

// TranslateX returns the horizontal offset inherited by descendants.
func (b *BaseRenderable) TranslateX() int { return b.translateX }

// TranslateY returns the vertical offset inherited by descendants.
func (b *BaseRenderable) TranslateY() int { return b.translateY }

// SetTranslateX sets the horizontal offset inherited by descendants.
func (b *BaseRenderable) SetTranslateX(x int) { b.translateX = x }

// SetTranslateY sets the vertical offset inherited by descendants.
func (b *BaseRenderable) SetTranslateY(y int) { b.translateY = y }

// StickyOffsetX returns b's horizontal sticky offset.
func (b *BaseRenderable) StickyOffsetX() int { return b.stickyOffsetX }

// SetStickyOffsetX sets b's horizontal sticky offset.
func (b *BaseRenderable) SetStickyOffsetX(x int) { b.stickyOffsetX = x }

// StickyOffsetY returns b's vertical sticky offset.
func (b *BaseRenderable) StickyOffsetY() int { return b.stickyOffsetY }

// SetStickyOffsetY sets b's vertical sticky offset.
func (b *BaseRenderable) SetStickyOffsetY(y int) { b.stickyOffsetY = y }

// UpdateFromLayout copies b's absolute flexbox geometry and applies b's own
// translation and sticky offset.
//
// Scene traversal should use UpdateFromLayoutWithOffset so a container's
// translation moves its descendants without moving the container itself.
func (b *BaseRenderable) UpdateFromLayout() {
	b.UpdateFromLayoutWithOffset(b.translateX, b.translateY)
}

// UpdateFromLayoutWithOffset copies b's absolute flexbox geometry and adds
// inherited translation and sticky offsets. It excludes b's own translation,
// which scene traversal passes to descendants.
func (b *BaseRenderable) UpdateFromLayoutWithOffset(offsetX, offsetY int) {
	computed := b.layoutNode.Computed

	b.x = computed.X + offsetX + b.stickyOffsetX
	b.y = computed.Y + offsetY + b.stickyOffsetY
	b.width = computed.Width
	b.height = computed.Height

	// Geometry that no longer matches the last paint leaves stale cells behind,
	// so both the old and the new region need repainting. The mark reaches the
	// ancestors so the damage pass, which runs next, descends to b.
	if b.Rect() != b.paintRect {
		b.MarkDirty()
	}

	// Notify through the concrete widget only when resolved dimensions change.
	if b.width != b.prevWidth || b.height != b.prevHeight {
		b.prevWidth = b.width
		b.prevHeight = b.height
		// Calling through self reaches a concrete widget's OnResize override.
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

// Render draws b's content and registers its hit-test region. Scene traversal
// renders children separately.
func (b *BaseRenderable) Render(buf *buffer.Buffer, deltaTime time.Duration) {
	if !b.Visible() {
		return
	}

	b.renderContent(buf, deltaTime)

	// Only visible renderables register hit-test regions.
	b.host.AddToHitGrid(b.x, b.y, b.width, b.height, b.num)

	b.paintRect = b.Rect()
	b.dirty = false
}

// renderContent runs the before hook, concrete RenderSelf implementation, and
// after hook in order.
func (b *BaseRenderable) renderContent(buf *buffer.Buffer, deltaTime time.Duration) {
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

// RenderSelf is the widget content override point. The default does nothing.
func (b *BaseRenderable) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {}

// RenderBefore calls RenderBeforeHook when it is set.
func (b *BaseRenderable) RenderBefore(buf *buffer.Buffer, deltaTime time.Duration) {
	if b.RenderBeforeHook != nil {
		b.RenderBeforeHook(buf, deltaTime)
	}
}

// RenderAfter calls RenderAfterHook when it is set.
func (b *BaseRenderable) RenderAfter(buf *buffer.Buffer, deltaTime time.Duration) {
	if b.RenderAfterHook != nil {
		b.RenderAfterHook(buf, deltaTime)
	}
}

// Focusable reports whether b can receive keyboard focus.
func (b *BaseRenderable) Focusable() bool { return b.focusable }

// SetFocusable controls whether b can receive keyboard focus.
func (b *BaseRenderable) SetFocusable(f bool) { b.focusable = f }

// Focused reports whether b currently has keyboard focus.
func (b *BaseRenderable) Focused() bool { return b.focused }

// Focus requests keyboard focus for b when it is focusable.
func (b *BaseRenderable) Focus() {
	if !b.focusable {
		return
	}
	b.focused = true
	b.host.FocusRenderable(b)
}

// Blur removes keyboard focus from b.
func (b *BaseRenderable) Blur() {
	b.focused = false
}

// HandleKeyPress calls OnKeyDownHandler when set and reports whether the
// handler stopped event propagation.
func (b *BaseRenderable) HandleKeyPress(e *input.KeyEvent) bool {
	if b.OnKeyDownHandler != nil {
		b.OnKeyDownHandler(e)
		return e.PropagationStopped()
	}
	return false
}

// HandlePaste calls OnPasteHandler when it is set.
func (b *BaseRenderable) HandlePaste(e *input.PasteEvent) {
	if b.OnPasteHandler != nil {
		b.OnPasteHandler(e)
	}
}

// ProcessMouseEvent calls the catch-all handler first. Unless it stops
// propagation, ProcessMouseEvent then calls the matching type-specific handler
// and propagates e to the parent.
func (b *BaseRenderable) ProcessMouseEvent(e *input.MouseEvent) {
	// The catch-all handler runs before the type-specific handler.
	if b.OnMouseHandler != nil {
		b.OnMouseHandler(e)
	}

	if e.PropagationStopped() {
		return
	}

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

	// Continue propagation through the scene graph parent.
	if !e.PropagationStopped() && b.parent != nil {
		b.parent.ProcessMouseEvent(e)
	}
}

// Selectable reports whether b supports text selection.
func (b *BaseRenderable) Selectable() bool { return b.selectable }

// SetSelectable controls whether b supports text selection.
func (b *BaseRenderable) SetSelectable(s bool) { b.selectable = s }

// ShouldStartSelection reports b's selectable state. Widgets can override it
// for coordinate-dependent behavior.
func (b *BaseRenderable) ShouldStartSelection(x, y int) bool {
	return b.selectable
}

// SelectedText returns no text by default.
func (b *BaseRenderable) SelectedText() string { return "" }

// HasSelection reports false by default.
func (b *BaseRenderable) HasSelection() bool { return false }

// OnUpdate runs before layout and does nothing by default.
func (b *BaseRenderable) OnUpdate(deltaTime time.Duration) {}

// OnPostLayout runs after layout and does nothing by default.
func (b *BaseRenderable) OnPostLayout(deltaTime time.Duration) {}

// OnResize handles a resolved size change and does nothing by default.
func (b *BaseRenderable) OnResize(width, height int) {}

// Destroy marks b destroyed, detaches it from its parent, and unregisters its
// numeric ID.
func (b *BaseRenderable) Destroy() {
	b.destroyed = true

	if b.parent != nil {
		b.parent.Remove(b.id)
	}

	renderablesByNum.Delete(b.num)
}

// DestroyRecursively destroys every descendant before destroying b.
func (b *BaseRenderable) DestroyRecursively() {
	// Copy the slice because destroying a child mutates b.children.
	children := make([]Renderable, len(b.children))
	copy(children, b.children)
	for _, child := range children {
		child.DestroyRecursively()
	}
	b.children = nil

	b.Destroy()
}

// Host returns the renderer that hosts b.
func (b *BaseRenderable) Host() Host { return b.host }

// SetHost assigns host to b and every descendant. A nil host is replaced by
// a no-op host.
func (b *BaseRenderable) SetHost(host Host) {
	if host == nil {
		host = noopHost
	}
	b.host = host
	for _, child := range b.children {
		child.SetHost(host)
	}
}
