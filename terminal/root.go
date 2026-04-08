package terminal

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
)

// RootRenderable is the top-level node in the scene graph. It orchestrates
// the multi-pass render pipeline: lifecycle → update → layout → sync →
// command generation → render → flush. All user widgets are descendants
// of this node.
type RootRenderable struct {
	*render.BaseRenderable

	lifecyclePasses map[uint64]render.Renderable // registered for lifecycle pass

	// LastFrameStats is populated after each Render call.
	LastFrameStats FrameStats
}

// NewRootRenderable creates a root renderable with the given viewport size.
func NewRootRenderable(width, height int) *RootRenderable {
	r := &RootRenderable{
		BaseRenderable:  render.NewBaseRenderable(render.WithID("root")),
		lifecyclePasses: make(map[uint64]render.Renderable),
	}

	// Set root layout node to fill the viewport
	r.LayoutNode().Width = layout.Points(width)
	r.LayoutNode().Height = layout.Points(height)
	r.LayoutNode().FlexDirection = layout.DirectionColumn

	r.SetBounds(0, 0, width, height)

	return r
}

// RegisterLifecyclePass registers a renderable for the lifecycle pass.
func (r *RootRenderable) RegisterLifecyclePass(renderable render.Renderable) {
	r.lifecyclePasses[renderable.Num()] = renderable
}

// UnregisterLifecyclePass removes a renderable from the lifecycle pass.
func (r *RootRenderable) UnregisterLifecyclePass(num uint64) {
	delete(r.lifecyclePasses, num)
}

// ---------------------------------------------------------------------------
// Render Pipeline
// ---------------------------------------------------------------------------

// Render executes the full render pipeline:
//
//  1. Lifecycle     — registered per-frame callbacks (animations, timers)
//  2. Layout        — flexbox algorithm resolves Computed positions/sizes
//  3. SyncPositions — pull Computed → widget x/y/w/h with translate offsets,
//     then OnPostLayout hooks (react to fresh sizes, e.g. scroll recalc)
//  4. Render commands — build scissor/opacity/render list (children order)
//  5. Execute       — draw to buffer
func (r *RootRenderable) Render(buf *buffer.Buffer, deltaTime float64) {
	prepStats := r.PrepareLayout(deltaTime)
	execStats := r.ExecuteRender(buf, deltaTime)

	r.LastFrameStats = FrameStats{
		Lifecycle: prepStats.Lifecycle,
		Layout:    prepStats.Layout,
		Sync:      execStats.Sync,
		Commands:  execStats.Commands,
		Render:    execStats.Render,
		CmdCount:  execStats.CmdCount,
		NodeCount: execStats.NodeCount,
	}
}

// PrepareLayout runs the lifecycle, update, and layout passes.
// After calling this, the Computed fields on all layout nodes are valid.
// Call ExecuteRender to sync positions, build commands, and draw to a buffer.
//
// This split allows inline mode to inspect computed layout (to derive
// content height) before allocating the render buffer.
func (r *RootRenderable) PrepareLayout(deltaTime float64) FrameStats {
	t0 := time.Now()

	// 1. Lifecycle
	for _, renderable := range r.lifecyclePasses {
		if renderable.IsDestroyed() {
			continue
		}
		if br, ok := renderable.(*render.BaseRenderable); ok && br.OnLifecyclePass != nil {
			br.OnLifecyclePass()
		}
	}
	t1 := time.Now()

	// 2. Layout
	r.CalculateLayout()
	t2 := time.Now()

	return FrameStats{
		Lifecycle: t1.Sub(t0),
		Layout:    t2.Sub(t1),
	}
}

// ExecuteRender syncs positions from the computed layout, builds render
// commands, and executes them into the buffer. Must be called after
// PrepareLayout.
func (r *RootRenderable) ExecuteRender(buf *buffer.Buffer, deltaTime float64) FrameStats {
	t0 := time.Now()

	// 4. Position sync + post-layout hooks
	nodeCount := 0
	r.syncPositionsRecursive(r.BaseRenderable, deltaTime, 0, 0, &nodeCount)
	t1 := time.Now()

	// 5. Build render commands
	var commands []DrawCommand
	r.collectRenderCommands(r.BaseRenderable, &commands)
	t2 := time.Now()

	// Clear hit grid before rebuilding
	r.Context().ClearHitGrid()

	// 6. Execute render commands
	t3 := time.Now()
	for _, cmd := range commands {
		switch cmd.Action {
		case "render":
			if cmd.Target != nil && !cmd.Target.IsDestroyed() {
				cmd.Target.Render(buf, deltaTime)
			}
		case "pushScissorRect":
			buf.PushScissorRect(cmd.X, cmd.Y, cmd.Width, cmd.Height)
			r.Context().PushHitGridScissorRect(cmd.ScreenX, cmd.ScreenY, cmd.Width, cmd.Height)
		case "popScissorRect":
			buf.PopScissorRect()
			r.Context().PopHitGridScissorRect()
		case "pushOpacity":
			buf.PushOpacity(cmd.Opacity)
		case "popOpacity":
			buf.PopOpacity()
		}
	}
	t4 := time.Now()

	return FrameStats{
		Sync:      t1.Sub(t0),
		Commands:  t2.Sub(t1),
		Render:    t4.Sub(t3),
		CmdCount:  len(commands),
		NodeCount: nodeCount,
	}
}

// syncPositionsRecursive syncs widget positions from the freshly-computed
// layout and then runs OnPostLayout hooks. Must be called after
// CalculateLayout.
//
// For each node (pre-order, natural tree order):
//  1. UpdateFromLayoutWithOffset — copies Computed → x/y/w/h with the
//     accumulated parent translate offset. The node's own translate is NOT
//     applied to itself (so a ScrollBox's scissor rect stays at its layout
//     position), but IS accumulated for descendants.
//  2. OnPostLayout — the widget reacts to its final size. For example,
//     ScrollBox recalculates contentHeight and adjusts scrollY/translateY.
//  3. Read TranslateX/Y AFTER OnPostLayout — hooks may have changed the
//     translate (e.g. auto-scroll), so children see the updated offset.
func (r *RootRenderable) syncPositionsRecursive(node render.Renderable, deltaTime float64, parentTX, parentTY int, nodeCount *int) {
	r.syncPositionsWithClip(node, deltaTime, parentTX, parentTY, nil, nodeCount)
}

func (r *RootRenderable) syncPositionsWithClip(node render.Renderable, deltaTime float64, parentTX, parentTY int, clip *scissorRect, nodeCount *int) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	*nodeCount++

	// Sync position: Computed → widget x/y/w/h (triggers OnResize if changed)
	node.UpdateFromLayoutWithOffset(parentTX, parentTY)

	// Viewport culling: after position is synced, skip recursion into
	// children if this node is entirely outside the active scissor.
	if clip != nil {
		nx, ny, nw, nh := node.X(), node.Y(), node.Width(), node.Height()
		if nx+nw <= clip.x || nx >= clip.x+clip.w ||
			ny+nh <= clip.y || ny >= clip.y+clip.h {
			return
		}
	}

	// Post-layout hook (may change translate, e.g. ScrollBox scroll)
	node.OnPostLayout(deltaTime)

	// Accumulate this node's translate for descendants.
	// Read AFTER OnPostLayout so we pick up any translate changes.
	// Include sticky offsets so descendants of a sticky child move with it.
	childTX := parentTX + node.TranslateX() + node.StickyOffsetX()
	childTY := parentTY + node.TranslateY() + node.StickyOffsetY()

	// If this node is a scroll/clip container, create a clip rect for children.
	childClip := clip
	overflow := node.LayoutNode().Overflow
	if overflow == layout.OverflowScroll || overflow == layout.OverflowHidden {
		childClip = &scissorRect{
			x: node.X(), y: node.Y(),
			w: node.Width(), h: node.Height(),
		}
	}

	for _, child := range node.ChildrenRaw() {
		r.syncPositionsWithClip(child, deltaTime, childTX, childTY, childClip, nodeCount)
	}
}

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

// CalculateLayout runs the flexbox algorithm on the layout tree.
func (r *RootRenderable) CalculateLayout() {
	r.LayoutNode().ComputeLayout(r.Width(), r.Height())
}

// UpdateLayout syncs widget positions from the computed layout, runs
// post-layout hooks, and returns the list of render commands. This is
// the public entry point used by tests; Render() calls the underlying
// steps directly with deltaTime.
func (r *RootRenderable) UpdateLayout() []DrawCommand {
	nodeCount := 0
	r.syncPositionsRecursive(r.BaseRenderable, 0, 0, 0, &nodeCount)
	var commands []DrawCommand
	r.collectRenderCommands(r.BaseRenderable, &commands)
	return commands
}

// scissorRect tracks the current clipping region for viewport culling.
type scissorRect struct {
	x, y, w, h int
}

// collectRenderCommands builds the render command list from already-synced
// widget positions. Traverses depth-first in children order for
// correct paint order. Emits scissor, opacity, and render commands.
// Viewport culling: children entirely outside the active scissor rect are
// skipped (no render command emitted, no recursion).
func (r *RootRenderable) collectRenderCommands(node render.Renderable, commands *[]DrawCommand) {
	r.collectRenderCommandsWithScissor(node, commands, nil)
}

func (r *RootRenderable) collectRenderCommandsWithScissor(node render.Renderable, commands *[]DrawCommand, clip *scissorRect) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	// Viewport culling: skip nodes entirely outside the active scissor.
	if clip != nil {
		nx, ny, nw, nh := node.X(), node.Y(), node.Width(), node.Height()
		if nx+nw <= clip.x || nx >= clip.x+clip.w ||
			ny+nh <= clip.y || ny >= clip.y+clip.h {
			return
		}
	}

	needsScissor := node.LayoutNode().Overflow == layout.OverflowHidden ||
		node.LayoutNode().Overflow == layout.OverflowScroll
	needsOpacity := node.Opacity() < 1.0

	if needsOpacity {
		*commands = append(*commands, DrawCommand{
			Action:  "pushOpacity",
			Opacity: node.Opacity(),
		})
	}

	// Render the node itself (border, background) BEFORE the scissor
	// push so borders are not clipped by the inset scissor.
	*commands = append(*commands, DrawCommand{
		Action: "render",
		Target: node,
	})

	// Push scissor rect inset by padding so scrolled children cannot
	// paint over borders. The node already rendered above so its
	// border draws at full bounds.
	sx, sy := node.X(), node.Y()
	sw, sh := node.Width(), node.Height()
	if needsScissor {

		ln := node.LayoutNode()
		if pt, ok := ln.PaddingTop.Resolve(sh); ok && pt > 0 {
			sy += pt
			sh -= pt
		}
		if pb, ok := ln.PaddingBottom.Resolve(sh); ok && pb > 0 {
			sh -= pb
		}
		if pl, ok := ln.PaddingLeft.Resolve(sw); ok && pl > 0 {
			sx += pl
			sw -= pl
		}
		if pr, ok := ln.PaddingRight.Resolve(sw); ok && pr > 0 {
			sw -= pr
		}
		if sw < 0 {
			sw = 0
		}
		if sh < 0 {
			sh = 0
		}

		*commands = append(*commands, DrawCommand{
			Action:  "pushScissorRect",
			X:       sx,
			Y:       sy,
			Width:   sw,
			Height:  sh,
			ScreenX: sx,
			ScreenY: sy,
		})
	}

	// Update the active clip region for child culling.
	childClip := clip
	if needsScissor {
		childClip = &scissorRect{x: sx, y: sy, w: sw, h: sh}
	}

	r.collectChildCommands(node, commands, childClip)

	if needsScissor {
		*commands = append(*commands, DrawCommand{
			Action: "popScissorRect",
		})
	}

	if needsOpacity {
		*commands = append(*commands, DrawCommand{
			Action: "popOpacity",
		})
	}
}

// collectChildCommands emits render commands for a node's children. In scroll
// containers, sticky children are deferred so they draw on top of siblings
// that may scroll beneath them.
func (r *RootRenderable) collectChildCommands(node render.Renderable, commands *[]DrawCommand, clip *scissorRect) {
	children := node.ChildrenRaw()
	if node.LayoutNode().Overflow != layout.OverflowScroll {
		for _, child := range children {
			r.collectRenderCommandsWithScissor(child, commands, clip)
		}
		return
	}
	// Scroll container: non-sticky first, then sticky on top.
	for _, child := range children {
		if child.LayoutNode().Sticky != layout.StickyNone {
			continue
		}
		r.collectRenderCommandsWithScissor(child, commands, clip)
	}
	for _, child := range children {
		if child.LayoutNode().Sticky == layout.StickyNone {
			continue
		}
		r.collectRenderCommandsWithScissor(child, commands, clip)
	}
}

// Resize updates the root viewport size.
func (r *RootRenderable) Resize(width, height int) {
	r.SetBounds(0, 0, width, height)
	r.LayoutNode().Width = layout.Points(width)
	r.LayoutNode().Height = layout.Points(height)
	r.LayoutNode().MarkDirty()
	r.RequestRender()
}
