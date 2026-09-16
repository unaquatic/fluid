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

// SyncPositions copies prepared layout into resolved geometry and runs
// post-layout hooks. Call it after PrepareLayout and before Damage or Draw.
//
// It always walks the complete scene. Skipping a subtree would leave stale
// geometry behind and hide the damage a moved renderable causes.
func (r *RootRenderable) SyncPositions(deltaTime float64) FrameStats {
	t0 := time.Now()
	nodeCount := 0
	r.syncPositionsRecursive(r.BaseRenderable, deltaTime, 0, 0, &nodeCount)
	return FrameStats{Sync: time.Since(t0), NodeCount: nodeCount}
}

// TakeDamage returns the region of the scene that changed since the last frame,
// or an empty rectangle when nothing did. Call it after SyncPositions.
//
// It consumes the regions that nothing will repaint. A hidden or destroyed
// renderable never paints again, so the region it vacated is reported once and
// then forgotten. A visible renderable keeps reporting until it paints, which
// is what clears it.
func (r *RootRenderable) TakeDamage() buffer.Rect {
	var damage buffer.Rect
	r.collectDamage(r.BaseRenderable, nil, &damage)
	return damage
}

// ResetDamage discards the damage of every renderable in the scene. Call it
// when the frame repaints in full, which leaves nothing for the next one to
// restore.
func (r *RootRenderable) ResetDamage() {
	r.TakeDamage()
}

// collectDamage accumulates the damage of node's subtree.
//
// A clipping container bounds the damage of everything inside it, because
// nothing inside can reach a cell the container clips away. Without that bound
// a node scissored out of view never paints, so its geometry never matches its
// last paint and it reports damage on every frame forever.
func (r *RootRenderable) collectDamage(node render.Renderable, scissor *buffer.Rect, damage *buffer.Rect) {
	nodeDamage := node.Damage()
	if scissor != nil {
		nodeDamage = nodeDamage.Intersect(*scissor)
	}
	*damage = damage.Union(nodeDamage)
	node.ClearDamage()
	node.ClearChildDirty()

	// A hidden or destroyed subtree reports everything it covered, so its root
	// already carries what its descendants left behind.
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	childScissor := scissor
	if rect, clips := clipRect(node); clips {
		childScissor = &rect
	}

	stacked := stacksFromStart(node.LayoutNode())
	for _, child := range node.ChildrenRaw() {
		// A clean child of a container that stacks from the start holds no
		// change, and nothing before the first change could have moved.
		if stacked && !changedBelow(child) {
			continue
		}
		stacked = false
		r.collectDamage(child, childScissor, damage)
	}
}

// changedBelow reports whether node or anything under it changed since the last
// frame, which is what traversal must descend for.
func changedBelow(node render.Renderable) bool {
	return node.IsDirty() || node.ChildDirty()
}

// stacksFromStart reports whether n places its children from the start of the
// main axis, where a change to one child cannot move the children before it.
//
// Centring, end alignment, and space distribution all move earlier children
// when a later one resizes, and wrapping moves them across lines.
func stacksFromStart(n *layout.LayoutNode) bool {
	return n.JustifyContent == layout.JustifyStart &&
		n.FlexWrap == layout.WrapNone &&
		!n.FlexDirection.IsReverse()
}

// ExecuteRender synchronizes positions from prepared layout, builds commands,
// and executes them into buf. Call it after PrepareLayout.
func (r *RootRenderable) ExecuteRender(buf *buffer.Buffer, deltaTime float64) FrameStats {
	sync := r.SyncPositions(deltaTime)
	stats := r.Draw(buf, deltaTime, buffer.Rect{})
	stats.Sync = sync.Sync
	stats.NodeCount = sync.NodeCount
	return stats
}

// Draw builds and executes render commands for the renderables intersecting
// damage. An empty rectangle draws the complete scene. Call it after
// SyncPositions.
//
// Callers must clear the damaged region first, because renderables paint their
// content over whatever the buffer already holds.
//
// Painting is confined to the damaged region. A renderable that draws outside
// its own rectangle would otherwise write cells the frame never cleared, which
// no longer match what the caller records as painted.
func (r *RootRenderable) Draw(buf *buffer.Buffer, deltaTime float64, damage buffer.Rect) FrameStats {
	var damaged *buffer.Rect
	if !damage.IsEmpty() {
		damaged = &damage
	}

	// Build the ordered command stream for the damaged region.
	t0 := time.Now()
	var commands []DrawCommand
	r.collectRenderCommandsWithScissor(r.BaseRenderable, &commands, nil, damaged)
	t1 := time.Now()

	// Visible render commands repopulate the hit grid.
	r.Context().ClearHitGrid()

	// Confine every write to the damage. Nested pushes intersect, so a clipping
	// container still narrows this further.
	if damaged != nil {
		buf.PushScissorRect(damage.X, damage.Y, damage.Width, damage.Height)
	}

	// Execute render and stack operations in order.
	t2 := time.Now()
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
	t3 := time.Now()

	if damaged != nil {
		buf.PopScissorRect()
	}

	// Sync and NodeCount belong to SyncPositions, which Draw does not run.
	return FrameStats{
		Commands: t1.Sub(t0),
		Render:   t3.Sub(t2),
		CmdCount: len(commands),
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

// syncPositionsWithClip resolves the geometry of node and every descendant it
// reached. A hidden or destroyed subtree paints nothing and is not descended
// into.
//
// A subtree the clip culls is not descended into either. Every pass culls on
// the same cullRect, so the command pass discards that subtree too.
func (r *RootRenderable) syncPositionsWithClip(node render.Renderable, deltaTime float64, parentTX, parentTY int, clip *buffer.Rect, nodeCount *int) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	*nodeCount++

	// Copy computed geometry and trigger resize handling when dimensions changed.
	prev := node.Rect()
	node.UpdateFromLayoutWithOffset(parentTX, parentTY)

	// A node that moved or resized carries every descendant with it, so none of
	// them can keep the geometry of the previous frame.
	moved := node.Rect() != prev

	// Do not run post-layout hooks or descend when the node is outside the clip.
	if excludes(clip, node.Rect()) {
		return
	}

	// Post-layout hooks can update descendant translation.
	node.OnPostLayout(deltaTime)

	// Descendants inherit current translation and sticky movement.
	childTX := parentTX + node.TranslateX() + node.StickyOffsetX()
	childTY := parentTY + node.TranslateY() + node.StickyOffsetY()

	// A clipping container supplies the child culling rectangle.
	childClip := clip
	if rect, clips := clipRect(node); clips {
		childClip = &rect
	}

	stacked := !moved && stacksFromStart(node.LayoutNode())
	for _, child := range node.ChildrenRaw() {
		// A clean child of a container that stayed put holds the geometry of
		// the previous frame, and so does its subtree.
		if stacked && !changedBelow(child) {
			continue
		}
		stacked = false
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
// clipRect returns the rectangle node bounds its descendants to, and reports
// whether node bounds them at all. Every pass uses this one, so they agree on
// what a container hides.
//
// It is the container's whole box, not the box its scissor command carries,
// which padding insets further. Culling on the wider rectangle only keeps
// renderables the buffer then clips away.
func clipRect(node render.Renderable) (buffer.Rect, bool) {
	switch node.LayoutNode().Overflow {
	case layout.OverflowScroll, layout.OverflowHidden:
		return node.Rect(), true
	}
	return buffer.Rect{}, false
}

// excludes reports whether rect lies entirely outside clip. A nil clip excludes
// nothing.
func excludes(clip *buffer.Rect, rect buffer.Rect) bool {
	if clip == nil {
		return false
	}
	return rect.Right() <= clip.X || rect.X >= clip.Right() ||
		rect.Bottom() <= clip.Y || rect.Y >= clip.Bottom()
}

// collectRenderCommands builds a depth-first command stream from synchronized
// widget positions. Nodes outside an active clipping region are omitted with
// their subtrees.
func (r *RootRenderable) collectRenderCommands(node render.Renderable, commands *[]DrawCommand) {
	r.collectRenderCommandsWithScissor(node, commands, nil, nil)
}

// collectRenderCommandsWithScissor appends the command stream for node's
// subtree. Either region may be nil.
//
// scissor comes from a clipping container and applies to descendants, so a
// subtree outside it cannot paint and is culled whole.
//
// damage is fixed for the frame. A subtree paints inside its root, so a node
// outside the damage encloses nothing inside it and is culled whole too.
func (r *RootRenderable) collectRenderCommandsWithScissor(node render.Renderable, commands *[]DrawCommand, scissor, damage *buffer.Rect) {
	if node.IsDestroyed() || !node.Visible() {
		return
	}

	// Cull a complete subtree when its root lies outside the active scissor.
	if excludes(scissor, node.Rect()) {
		return
	}

	// A subtree whose root misses the damage holds nothing inside it, because a
	// subtree paints inside its root.
	if excludes(damage, node.Rect()) {
		return
	}

	cull, needsScissor := clipRect(node)
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

	// Cull descendants on the container's whole box, which is what layout
	// traversal culls on too. The emitted scissor still clips painting to the
	// narrower inset box.
	childScissor := scissor
	if needsScissor {
		childScissor = &cull
	}

	r.collectChildCommands(node, commands, childScissor, damage)

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
func (r *RootRenderable) collectChildCommands(node render.Renderable, commands *[]DrawCommand, scissor, damage *buffer.Rect) {
	children := node.ChildrenRaw()
	if node.LayoutNode().Overflow != layout.OverflowScroll {
		for _, child := range children {
			r.collectRenderCommandsWithScissor(child, commands, scissor, damage)
		}
		return
	}
	// Paint non-sticky children before sticky children.
	for _, child := range children {
		if child.LayoutNode().Sticky != layout.StickyNone {
			continue
		}
		r.collectRenderCommandsWithScissor(child, commands, scissor, damage)
	}
	for _, child := range children {
		if child.LayoutNode().Sticky == layout.StickyNone {
			continue
		}
		r.collectRenderCommandsWithScissor(child, commands, scissor, damage)
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
