package terminal

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
)

// RootRenderable is the scene graph root. It coordinates lifecycle callbacks,
// flexbox layout, position synchronization, and ordered render commands.
type RootRenderable struct {
	*render.BaseRenderable

	lifecyclePasses map[uint64]render.Renderable // Renderables with lifecycle callbacks.

	// LastFrameStats contains measurements from the most recent frame.
	LastFrameStats FrameStats
}

// NewRootRenderable returns a scene root sized to the viewport.
func NewRootRenderable(width, height int) *RootRenderable {
	r := &RootRenderable{
		BaseRenderable:  render.NewBaseRenderable(render.WithID("root")),
		lifecyclePasses: make(map[uint64]render.Renderable),
	}

	// The root is a column that exactly fills the viewport.
	r.LayoutNode().Width = layout.Points(width)
	r.LayoutNode().Height = layout.Points(height)
	r.LayoutNode().FlexDirection = layout.DirectionColumn

	r.SetBounds(0, 0, width, height)

	return r
}

// RegisterLifecyclePass adds renderable to the per-frame lifecycle pass.
func (r *RootRenderable) RegisterLifecyclePass(renderable render.Renderable) {
	r.lifecyclePasses[renderable.Num()] = renderable
}

// UnregisterLifecyclePass removes the renderable identified by num from the
// lifecycle pass.
func (r *RootRenderable) UnregisterLifecyclePass(num uint64) {
	delete(r.lifecyclePasses, num)
}

// Render executes these root rendering passes:
//
//  1. Run registered lifecycle callbacks.
//  2. Resolve flexbox layout.
//  3. Synchronize widget positions and run post-layout hooks.
//  4. Build and execute render, clipping, and opacity commands.
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

// PrepareLayout runs lifecycle callbacks and computes layout. On return, each
// layout node's Computed field is current.
//
// Inline rendering calls this separately so it can derive content height before
// allocating its frame buffer.
func (r *RootRenderable) PrepareLayout(deltaTime float64) FrameStats {
	t0 := time.Now()

	// Run registered lifecycle hooks before layout.
	for _, renderable := range r.lifecyclePasses {
		if renderable.IsDestroyed() {
			continue
		}
		if br, ok := renderable.(*render.BaseRenderable); ok && br.OnLifecyclePass != nil {
			br.OnLifecyclePass()
		}
	}
	t1 := time.Now()

	r.CalculateLayout()
	t2 := time.Now()

	return FrameStats{
		Lifecycle: t1.Sub(t0),
		Layout:    t2.Sub(t1),
	}
}

// ExecuteRender synchronizes positions from prepared layout, builds commands,
// and executes them into buf. Call it after PrepareLayout.
func (r *RootRenderable) ExecuteRender(buf *buffer.Buffer, deltaTime float64) FrameStats {
	t0 := time.Now()

	// Synchronize positions and run post-layout hooks.
	nodeCount := 0
	r.syncPositionsRecursive(r.BaseRenderable, deltaTime, 0, 0, &nodeCount)
	t1 := time.Now()

	// Build the complete ordered command stream.
	var commands []DrawCommand
	r.collectRenderCommands(r.BaseRenderable, &commands)
	t2 := time.Now()

	// Visible render commands repopulate the hit grid.
	r.Context().ClearHitGrid()

	// Execute render and stack operations in order.
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

// syncPositionsRecursive synchronizes a laid-out subtree and runs post-layout
// hooks in pre-order.
//
// Each node receives inherited translation without its own translation. After
// OnPostLayout, the node's current translation is added for descendants.
func (r *RootRenderable) syncPositionsRecursive(node render.Renderable, deltaTime float64, parentTX, parentTY int, nodeCount *int) {
	r.syncPositionsWithClip(node, deltaTime, parentTX, parentTY, nil, nodeCount)
}

func (r *RootRenderable) syncPositionsWithClip(node render.Renderable, deltaTime float64, parentTX, parentTY int, clip *scissorRect, nodeCount *int) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	*nodeCount++

	// Copy computed geometry and trigger resize handling when dimensions changed.
	node.UpdateFromLayoutWithOffset(parentTX, parentTY)

	// Do not run post-layout hooks or descend when the node is outside the clip.
	if clip != nil {
		nx, ny, nw, nh := node.X(), node.Y(), node.Width(), node.Height()
		if nx+nw <= clip.x || nx >= clip.x+clip.w ||
			ny+nh <= clip.y || ny >= clip.y+clip.h {
			return
		}
	}

	// Post-layout hooks can update descendant translation.
	node.OnPostLayout(deltaTime)

	// Descendants inherit current translation and sticky movement.
	childTX := parentTX + node.TranslateX() + node.StickyOffsetX()
	childTY := parentTY + node.TranslateY() + node.StickyOffsetY()

	// A clipping container supplies the child culling rectangle.
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

// CalculateLayout computes flexbox geometry for the current root dimensions.
func (r *RootRenderable) CalculateLayout() {
	r.LayoutNode().ComputeLayout(r.Width(), r.Height())
}

// UpdateLayout synchronizes positions, runs post-layout hooks with zero delta,
// and returns render commands without executing them.
func (r *RootRenderable) UpdateLayout() []DrawCommand {
	nodeCount := 0
	r.syncPositionsRecursive(r.BaseRenderable, 0, 0, 0, &nodeCount)
	var commands []DrawCommand
	r.collectRenderCommands(r.BaseRenderable, &commands)
	return commands
}

// scissorRect contains a clipping region used for subtree culling.
type scissorRect struct {
	x, y, w, h int
}

// collectRenderCommands builds a depth-first command stream from synchronized
// widget positions. Nodes outside an active clipping region are omitted with
// their subtrees.
func (r *RootRenderable) collectRenderCommands(node render.Renderable, commands *[]DrawCommand) {
	r.collectRenderCommandsWithScissor(node, commands, nil)
}

func (r *RootRenderable) collectRenderCommandsWithScissor(node render.Renderable, commands *[]DrawCommand, clip *scissorRect) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	// Cull a complete subtree when its root lies outside the active clip.
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

	// Render the container before clipping its descendants so its border remains
	// visible.
	*commands = append(*commands, DrawCommand{
		Action: "render",
		Target: node,
	})

	// Inset descendant clipping by resolved padding.
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

	// Use the emitted scissor rectangle when culling descendants.
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

// collectChildCommands appends child command streams. In a scroll container,
// sticky children follow non-sticky siblings so they paint on top.
func (r *RootRenderable) collectChildCommands(node render.Renderable, commands *[]DrawCommand, clip *scissorRect) {
	children := node.ChildrenRaw()
	if node.LayoutNode().Overflow != layout.OverflowScroll {
		for _, child := range children {
			r.collectRenderCommandsWithScissor(child, commands, clip)
		}
		return
	}
	// Paint non-sticky children before sticky children.
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

// Resize updates the root viewport dimensions and requests a new frame.
func (r *RootRenderable) Resize(width, height int) {
	r.SetBounds(0, 0, width, height)
	r.LayoutNode().Width = layout.Points(width)
	r.LayoutNode().Height = layout.Points(height)
	r.LayoutNode().MarkDirty()
	r.RequestRender()
}
