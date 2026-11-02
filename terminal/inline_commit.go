package terminal

import (
	"slices"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
)

// CommitInline hands rows, the leading children of the root, over to the
// terminal, which keeps them in its scrollback as it shows them now. They
// leave the scene, and the frame starts below them, so the rows that stay are
// all a frame lays out, draws and holds, however much output scrolled away
// above them.
//
// It commits rows in order, stops at the first it cannot commit, and returns
// how many it committed. A row can be committed once it scrolled wholly into
// the scrollback, above the screen, as the terminal shows it: unchanged since
// the last frame was written. The children of the root before it have to be
// committed with it, or be gone, which takes no rows. CommitInline removes a
// committed row from the root without destroying it, and the row's later
// changes no longer reach the terminal.
//
// Writing the whole output again, as a resize, a change to a row in the
// scrollback, a return from the alternate screen and ReprintInline make the
// renderer do, erases the screen and the scrollback, and the committed rows
// with them. The function SetInlineReprint sets puts them back first. Without
// one, they are lost.
//
// It commits nothing outside inline mode, before the first frame is written
// or while the output is due to be written again, or when the root does not
// stack its children from its first row, as top padding or gaps between them
// prevent. Nor does it commit again before the next frame is laid out, as the
// rows are where the last frame put them, which a commit moves.
//
// Call it from the event-loop goroutine before a frame is laid out, for
// example from a lifecycle pass.
func (r *Renderer) CommitInline(rows ...render.Renderable) int {
	if !r.inlineCommitReady() {
		return 0
	}

	// The rows the root skips are gone, and take no rows. Removing by ID takes
	// the first child that has it, so a committed row must not share one.
	var skipped []string
	n, height := 0, 0
	for _, child := range r.root.ChildrenRaw() {
		if n == len(rows) {
			break
		}
		if child.Num() != rows[n].Num() {
			if !gone(child) {
				break
			}
			skipped = append(skipped, child.ID())
			continue
		}
		bottom, ok := r.inlineCommittable(child)
		if !ok || slices.Contains(skipped, child.ID()) {
			break
		}
		height = max(height, bottom)
		n++
	}
	if n == 0 {
		return 0
	}

	for _, row := range rows[:n] {
		r.root.Remove(row.ID())
	}
	// The root stays in place, so the frame's traversal skips its clean
	// children, but every child that stays moved up by the committed rows.
	if children := r.root.ChildrenRaw(); len(children) > 0 {
		children[0].MarkDirty()
	}
	if height > 0 {
		r.dropInlineRows(height)
	}
	return n
}

// inlineCommitReady reports whether CommitInline can commit rows: the frame
// last written is the terminal's, and the root stacks its children from its
// first row, so the rows a child covers end where the next one's begin.
func (r *Renderer) inlineCommitReady() bool {
	r.mu.Lock()
	inline, w, h := r.config.InlineMode, r.width, r.height
	r.mu.Unlock()

	switch {
	case !inline, r.inlineCurrentBuf == nil, r.inlineReprintDue, r.inlineRebased:
		return false
	case r.inlineCurrentBuf.Width() != inlineLayoutWidth(w), r.inlineTermHeight != h:
		return false // A resize makes the next frame write every row again.
	}

	root := r.root.LayoutNode()
	gap := root.Gap
	if root.RowGap > 0 {
		gap = root.RowGap
	}
	pad, _ := root.PaddingTop.Resolve(0)
	return root.FlexDirection == layout.DirectionColumn && stacksFromStart(root) && gap == 0 && pad == 0
}

// inlineCommittable reports whether CommitInline can commit row, a child of
// the root, and returns the frame row the rows after it begin on, below it
// and its bottom margin. A gone row takes no rows and returns zero.
//
// Only a row wholly above the screen can be committed: the terminal can no
// longer redraw it but by writing every row again.
func (r *Renderer) inlineCommittable(row render.Renderable) (bottom int, ok bool) {
	if gone(row) {
		return 0, true
	}

	// What the terminal shows of row is its last paint, which its geometry
	// matches while nothing changed it, its layout included.
	n := row.LayoutNode()
	if !row.Visible() || row.IsDestroyed() || row.IsDirty() || row.ChildDirty() || n.IsDirty() ||
		n.Computed.Y != row.Y() || n.Computed.Height != row.Height() {
		return 0, false
	}
	margin, _ := n.MarginBottom.Resolve(0)
	if margin < 0 {
		return 0, false // The rows after it overlap it.
	}
	bottom = row.Y() + row.Height() + margin
	if bottom > r.inlineViewportTop {
		return 0, false // Some of it is still on screen.
	}
	return bottom, true
}

// gone reports whether row takes part in neither layout nor rendering.
func gone(row render.Renderable) bool {
	return row.LayoutNode().Visibility == layout.Gone
}

// dropInlineRows removes the first n rows from the frame the terminal shows,
// and moves the rows the next frame starts from up by n. The rows are in the
// scrollback, so the frame's first row is now n rows nearer the screen.
//
// The frame being drawn, and the rows it tracks while it is, need no moving:
// removing rows from the root damages all of the next one.
func (r *Renderer) dropInlineRows(n int) {
	r.inlineCurrentBuf.DropRows(n)
	r.inlineViewportTop -= n
	r.inlineCursorRow -= n
	r.inlineLiveWas = max(r.inlineLiveWas-n, 0)
	r.inlineCommitted, r.inlineRebased = true, true
}

// SetInlineReprint sets fn, the function that puts back the rows committed
// with CommitInline before inline output is written again in full. A resize,
// a change to a row in the scrollback, a return from the alternate screen and
// ReprintInline make the renderer do so, which erases the screen and the
// scrollback, committed rows too.
//
// fn adds the committed rows back to the root, at the top, as they are to
// look now. It may replace the rows the root still holds as well. The frame
// that called it then erases the screen and the scrollback and writes every
// row, after which they can be committed again.
//
// fn runs on the event loop at the start of a frame, before its lifecycle
// passes and layout. A nil fn, the default, puts nothing back, so such a
// rewrite loses the committed rows.
//
// Call it from the event-loop goroutine, for example through Enqueue, or
// before the renderer runs.
func (r *Renderer) SetInlineReprint(fn func()) { r.inlineReprint = fn }

// ReprintInline writes the whole of inline output again on the next frame,
// the rows committed with CommitInline too, which the function
// SetInlineReprint sets puts back first. Call it after a change that reaches
// committed rows, such as a setting that changes how every row looks: they
// left the scene, so nothing else repaints them.
//
// It does nothing outside inline mode, or while no row is committed: the
// scene then holds every row, and repaints those the change reaches.
//
// Call it from the event-loop goroutine.
func (r *Renderer) ReprintInline() {
	if !r.InlineMode() || !r.inlineCommitted {
		return
	}
	r.inlineReprintDue = true
	r.RequestRender()
}

// reprintInline has the program put back the rows committed to the terminal,
// and makes the frame write every row in place of the screen and the
// scrollback. A reprint is due only while rows are committed.
func (r *Renderer) reprintInline() {
	r.inlineReprintDue = false
	if r.inlineReprint != nil {
		r.inlineReprint()
	}
	r.inlineCommitted = false
	r.inlineCurrentBuf = nil
	r.inlineEraseNext = true
}
