package terminal

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/widget"
)

// glyphRow paints every cell it occupies with one glyph, so a stale cell in the
// frame is visible as the wrong rune.
type glyphRow struct {
	*render.BaseRenderable
	glyph rune
}

func newGlyphRow(id string, glyph rune, width, height int) *glyphRow {
	g := &glyphRow{
		BaseRenderable: render.NewBaseRenderable(render.WithID(id),
			render.WithWidth(layout.Points(width)), render.WithHeight(layout.Points(height))),
		glyph: glyph,
	}
	g.SetSelf(g)
	return g
}

func (g *glyphRow) RenderSelf(buf *buffer.Buffer, _ time.Duration) {
	for y := range g.Height() {
		for x := range g.Width() {
			buf.SetCell(g.X()+x, g.Y()+y, g.glyph, style.ColorWhite, style.ColorBlack, 0)
		}
	}
}

func (g *glyphRow) setGlyph(r rune) {
	g.glyph = r
	g.RequestRender()
}

// frameText returns column zero of every frame row, which identifies the widget
// that painted each one.
func frameText(buf *buffer.Buffer) string {
	var sb strings.Builder
	for y := range buf.Height() {
		sb.WriteRune(buf.Cell(0, y).Char)
	}
	return sb.String()
}

func newDamageTestRenderer(t *testing.T, rows ...*glyphRow) *Renderer {
	t.Helper()
	r, _ := newInlineTestRenderer()
	for _, row := range rows {
		r.root.Add(row)
	}
	r.renderInlineFrame() // Establish a painted baseline.
	return r
}

// TestTakeDamage pins the contract the name promises. A vacated region is
// reported once and then forgotten, because nothing will repaint it. A visible
// renderable keeps reporting until it paints.
func TestTakeDamage(t *testing.T) {
	t.Run("a vacated region is taken once", func(t *testing.T) {
		b := newGlyphRow("b", 'b', 39, 2)
		r := newDamageTestRenderer(t, newGlyphRow("a", 'a', 39, 2), b)

		// Hiding the last row moves no sibling, so only b is damaged.
		b.SetGone(true)
		r.root.PrepareLayout(0)
		r.root.SyncPositions(0)

		want := buffer.Rect{X: 0, Y: 2, Width: 39, Height: 2}
		if got := r.root.TakeDamage(); got != want {
			t.Fatalf("damage is %+v, want %+v", got, want)
		}
		if got := r.root.TakeDamage(); !got.IsEmpty() {
			t.Errorf("second call reports %+v, want nothing left to take", got)
		}
	})

	t.Run("an unpainted renderable keeps its damage", func(t *testing.T) {
		b := newGlyphRow("b", 'b', 39, 2)
		r := newDamageTestRenderer(t, newGlyphRow("a", 'a', 39, 2), b)

		b.setGlyph('B')
		r.root.PrepareLayout(0)
		r.root.SyncPositions(0)

		first := r.root.TakeDamage()
		if first.IsEmpty() {
			t.Fatal("expected the changed row to be damaged")
		}
		if got := r.root.TakeDamage(); got != first {
			t.Errorf("damage is %+v, want %+v until it paints", got, first)
		}
	})
}

// newHiddenSubtreeRenderer builds a frame whose rows 2-3 are painted by wrap's
// child, with wrap inside a fixed-height container so that hiding or removing
// it cannot change the frame height. A height change would repaint everything
// and hide what these tests are checking.
func newHiddenSubtreeRenderer(t *testing.T) (*Renderer, *render.BaseRenderable) {
	t.Helper()

	wrap := render.NewBaseRenderable(render.WithID("wrap"),
		render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(2)))
	wrap.Add(newGlyphRow("c", 'c', 39, 2))

	container := render.NewBaseRenderable(render.WithID("container"),
		render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(2)))
	container.Add(wrap)

	r, _ := newInlineTestRenderer()
	r.root.Add(newGlyphRow("head", 'H', 39, 2))
	r.root.Add(container)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "HHcc" {
		t.Fatalf("baseline frame is %q, want %q", got, "HHcc")
	}
	return r, wrap
}

// TestInlineDamage_HiddenSubtreeErases covers hiding a parent. Traversal stops
// at it, so it has to report the region it painted or its child's rows keep
// their content forever.
func TestInlineDamage_HiddenSubtreeErases(t *testing.T) {
	r, wrap := newHiddenSubtreeRenderer(t)

	// SetVisible keeps the layout space, so the frame height does not change and
	// the damage path stays in play.
	wrap.SetVisible(false)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "HH  " {
		t.Errorf("frame is %q, want %q with the child's rows erased", got, "HH  ")
	}
}

// TestInlineDamage_DetachedSubtreeErases covers removing that parent. A
// detached subtree cannot report anything, so its region becomes the parent's
// to repaint.
func TestInlineDamage_DetachedSubtreeErases(t *testing.T) {
	r, wrap := newHiddenSubtreeRenderer(t)

	wrap.Destroy()
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "HH  " {
		t.Errorf("frame is %q, want %q with the child's rows erased", got, "HH  ")
	}
}

// newClippedBoxRenderer builds a four-row scroll container holding ten rows, so
// rows two onward are clipped away. Clipped rows never paint, so they stay
// dirty and report their region for as long as the scene lives.
func newClippedBoxRenderer(t *testing.T) (*Renderer, []*glyphRow) {
	t.Helper()

	box := render.NewBaseRenderable(render.WithID("box"),
		render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(4)))
	box.LayoutNode().Overflow = layout.OverflowScroll
	box.LayoutNode().FlexDirection = layout.DirectionColumn
	rows := make([]*glyphRow, 10)
	for i := range rows {
		rows[i] = newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 2)
		box.Add(rows[i])
	}

	r, _ := newInlineTestRenderer()
	r.root.Add(box)
	r.root.Add(newGlyphRow("footer", 'F', 39, 2)) // Keeps the clipped rows in frame.
	r.renderInlineFrame()
	return r, rows
}

// TestInlineDamage_ScissoredContentSettles covers an idle frame over clipped
// content. The clipped rows stay dirty forever, so a frame that consulted them
// would never settle.
func TestInlineDamage_ScissoredContentSettles(t *testing.T) {
	r, _ := newClippedBoxRenderer(t)

	// Nothing changes, so the second frame must find the scene settled.
	r.renderInlineFrame()

	if got := r.inlineDamaged; !got.IsEmpty() {
		t.Errorf("damage is %+v, want empty once the scene settles", got)
	}
	if got := r.root.LastFrameStats.CmdCount; got != 0 {
		t.Errorf("issued %d draw commands, want 0", got)
	}
}

// TestInlineDamage_ClippedRowsStayOutOfTheDamage covers the bound a clipping
// container puts on its descendants' damage. Changing a visible row makes
// traversal descend into the container, which then reaches the clipped rows
// behind it. Unbounded, their region would drag the damage over the whole
// content rather than the one row that changed.
func TestInlineDamage_ClippedRowsStayOutOfTheDamage(t *testing.T) {
	r, rows := newClippedBoxRenderer(t)

	rows[0].setGlyph('A')
	r.root.PrepareLayout(0)
	r.root.SyncPositions(0)

	want := buffer.Rect{X: 0, Y: 0, Width: 39, Height: 2}
	if got := r.root.TakeDamage(); got != want {
		t.Errorf("damage is %+v, want %+v bounded to the visible row", got, want)
	}
}

// TestInlineDamage_CullsKeepTheCommandStreamSmall pins how many draw commands a
// clipped scene issues. Both culls in the command pass are invisible to a
// correctness check, because dropping either one still paints the right frame
// and only grows the work behind it.
func TestInlineDamage_CullsKeepTheCommandStreamSmall(t *testing.T) {
	r, rows := newClippedBoxRenderer(t)

	// The first frame repaints in full, so the damage reaches the rows the box
	// clips away and the clipping bound alone keeps them out of the stream.
	const wantFull = 7
	if got := r.root.LastFrameStats.CmdCount; got != wantFull {
		t.Errorf("full repaint issued %d commands, want %d", got, wantFull)
	}

	// One visible row changes, so the damage alone excludes every other row.
	rows[0].setGlyph('A')
	r.renderInlineFrame()

	const wantChange = 5
	if got := r.root.LastFrameStats.CmdCount; got != wantChange {
		t.Errorf("one visible change issued %d commands, want %d", got, wantChange)
	}
}

// TestInlineDamage_PruneSkipsCleanSiblings pins what one change behind many
// clean siblings costs. Traversal descends on the record an ancestor carries,
// so the record has to be forgotten once a frame has used it. A record left
// behind makes its bubble look changed forever, and every bubble after it is
// followed once traversal stops skipping.
//
// The earlier change is what gives the stale record somewhere to survive. A
// regression here is silent: the frame stays correct and only gets slower, in
// proportion to the transcript.
func TestInlineDamage_PruneSkipsCleanSiblings(t *testing.T) {
	const bubbleCount = 200

	r, _ := newInlineTestRenderer()
	rows := make([]*glyphRow, bubbleCount)
	for i := range rows {
		bubble := render.NewBaseRenderable(render.WithID("bubble"+strconv.Itoa(i)),
			render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(1)))
		rows[i] = newGlyphRow("row"+strconv.Itoa(i), 'a', 39, 1)
		bubble.Add(rows[i])
		r.root.Add(bubble)
	}
	r.renderInlineFrame() // Establish a painted baseline.

	// Mark the first bubble, which every later frame has to skip again.
	rows[0].setGlyph('A')
	r.renderInlineFrame()

	// The last row changes, so every bubble before it holds no change and
	// nothing before the first change could have moved.
	rows[bubbleCount-1].setGlyph('Z')
	r.renderInlineFrame()

	// The root, the changed bubble, and its row, with room for structure.
	const want = 8
	if got := r.root.LastFrameStats.NodeCount; got > want {
		t.Errorf("visited %d nodes for one change behind %d clean bubbles, want at most %d",
			got, bubbleCount-1, want)
	}
}

func TestInlineDamage_CleanFrameDrawsNothing(t *testing.T) {
	r := newDamageTestRenderer(t, newGlyphRow("a", 'a', 39, 2), newGlyphRow("b", 'b', 39, 2))

	r.renderInlineFrame()

	if got := r.inlineDamaged; !got.IsEmpty() {
		t.Errorf("damage is %+v, want empty for an unchanged scene", got)
	}
	if got := r.root.LastFrameStats.CmdCount; got != 0 {
		t.Errorf("issued %d draw commands, want 0", got)
	}
}

func TestInlineDamage_RepaintsOnlyTheChangedWidget(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	b.setGlyph('B')
	r.renderInlineFrame()

	if want := (buffer.Rect{X: 0, Y: 2, Width: 39, Height: 2}); r.inlineDamaged != want {
		t.Errorf("damage is %+v, want %+v", r.inlineDamaged, want)
	}
	// The untouched widget keeps its cells, and the changed one is repainted.
	if got := frameText(r.nextBuffer); got != "aaBB" {
		t.Errorf("frame is %q, want %q", got, "aaBB")
	}
}

// TestInlineDamage_MovedWidgetLeavesNoStaleCells is the regression this design
// risks: a widget that moves must repaint the rows it vacated.
func TestInlineDamage_MovedWidgetLeavesNoStaleCells(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 4)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	if got := frameText(r.nextBuffer); got != "aaaabb" {
		t.Fatalf("baseline frame is %q, want %q", got, "aaaabb")
	}

	// Shrinking the first widget pulls the second one up by two rows. Only the
	// first is marked; the second must be dirtied by the geometry change alone.
	a.LayoutNode().Height = layout.Points(2)
	a.LayoutNode().MarkDirty()
	a.RequestRender()
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "aabb" {
		t.Errorf("frame is %q, want %q with no stale rows", got, "aabb")
	}
}

func TestInlineDamage_HiddenWidgetIsErased(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	a.SetGone(true)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "bb" {
		t.Errorf("frame is %q, want %q", got, "bb")
	}
}

// TestInlineDamage_StopsAtTheViewport keeps repaints out of the scrollback,
// which the terminal cannot address.
// scrolledRenderer returns a renderer whose frame is taller than the terminal,
// so the rows at the top have scrolled into the scrollback.
func scrolledRenderer(t *testing.T) (*Renderer, *bytes.Buffer, []*glyphRow) {
	t.Helper()
	rows := make([]*glyphRow, 0, 15)
	for i := range 15 {
		rows = append(rows, newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 2))
	}
	r, out := newInlineTestRenderer()
	for _, row := range rows {
		r.root.Add(row)
	}
	r.renderInlineFrame()

	if r.inlineViewportTop == 0 {
		t.Fatalf("expected the 30-row frame to scroll a %d-row terminal", testTermHeight)
	}
	return r, out, rows
}

func TestInlineDamage_StopsAtTheViewport(t *testing.T) {
	r, _, rows := scrolledRenderer(t)

	// The last widget is on screen, below the viewport top.
	rows[len(rows)-1].setGlyph('Z')
	r.renderInlineFrame()

	if got := r.inlineDamaged; got.Y < r.inlineViewportTop {
		t.Errorf("damage %+v reaches above viewportTop=%d", got, r.inlineViewportTop)
	}
}

// TestInlineDamage_ScrollbackChangeRewritesTheScreen covers the case that
// cannot be repainted in place. A row in the scrollback cannot be addressed, so
// the only way to show the change is to erase and rewrite everything.
func TestInlineDamage_ScrollbackChangeRewritesTheScreen(t *testing.T) {
	r, out, rows := scrolledRenderer(t)
	out.Reset()

	// The first widget is entirely inside the scrollback.
	rows[0].setGlyph('Z')
	r.renderInlineFrame()

	if got := r.inlineDamaged; got.Y != 0 {
		t.Errorf("damage is %+v, want the whole frame", got)
	}
	if !strings.Contains(out.String(), inlineEraseAll) {
		t.Error("expected the screen and scrollback to be rewritten")
	}
	if !strings.Contains(out.String(), "ZZZ") {
		t.Error("expected the changed row to reach the terminal")
	}
}

// strayRow paints one row above its own rectangle, the way a background drawn
// wider or taller than its layout box does.
type strayRow struct {
	*render.BaseRenderable
	glyph rune
}

func newStrayRow(id string, glyph rune, width, height int) *strayRow {
	s := &strayRow{
		BaseRenderable: render.NewBaseRenderable(render.WithID(id),
			render.WithWidth(layout.Points(width)), render.WithHeight(layout.Points(height))),
		glyph: glyph,
	}
	s.SetSelf(s)
	return s
}

func (s *strayRow) RenderSelf(buf *buffer.Buffer, _ time.Duration) {
	for y := s.Y() - 1; y < s.Y()+s.Height(); y++ {
		for x := range s.Width() {
			buf.SetCell(x, y, s.glyph, style.ColorWhite, style.ColorBlack, 0)
		}
	}
}

// TestInlineDamage_PaintingStaysInsideTheDamage covers a renderable that paints
// outside its own rectangle. The frame clears and stores only the damaged rows,
// so a write beyond them would leave the stored frame describing something the
// terminal never received.
func TestInlineDamage_PaintingStaysInsideTheDamage(t *testing.T) {
	stray := newStrayRow("stray", 'o', 39, 2)

	r, _ := newInlineTestRenderer()
	r.root.Add(newGlyphRow("a", 'a', 39, 2))
	r.root.Add(newGlyphRow("b", 'b', 39, 2))
	r.root.Add(stray)
	r.renderInlineFrame()

	// Only the stray row changes, so the damage covers its rectangle alone.
	stray.glyph = 'O'
	stray.RequestRender()
	r.renderInlineFrame()

	if got, want := frameText(r.nextBuffer), frameText(r.inlineCurrentBuf); got != want {
		t.Errorf("frame is %q but the stored copy is %q; a write escaped the damage", got, want)
	}
}

// cursorMover moves the cursor while it paints. Painting runs after the frame
// decides its damage and before the frame inverts the cursor cell, so a frame
// that samples the cursor twice sees two different positions.
type cursorMover struct {
	*render.BaseRenderable
	renderer *Renderer
	x, y     int
}

func newCursorMover(r *Renderer, width, height int) *cursorMover {
	c := &cursorMover{
		BaseRenderable: render.NewBaseRenderable(render.WithID("mover"),
			render.WithWidth(layout.Points(width)), render.WithHeight(layout.Points(height))),
		renderer: r,
	}
	c.SetSelf(c)
	return c
}

func (c *cursorMover) RenderSelf(*buffer.Buffer, time.Duration) {
	c.renderer.SetCursorPosition(c.x, c.y, true)
}

// TestInlineDamage_CursorPublishedWhilePainting covers a widget that publishes
// its cursor position as it paints, which is how Textarea reports the caret.
//
// The frame has to invert the cell the widget just published, not the one the
// cursor held beforehand, and that cell has to sit inside the damage so the
// flush compares and stores it.
func TestInlineDamage_CursorPublishedWhilePainting(t *testing.T) {
	r, _ := newInlineTestRenderer()
	mover := newCursorMover(r, 39, 2)

	r.root.Add(newGlyphRow("a", 'a', 39, 2))
	r.root.Add(mover)
	r.root.Add(newGlyphRow("c", 'c', 39, 2))

	r.cursorVisible = true
	r.cursorX, r.cursorY = 0, 0
	mover.x, mover.y = 0, 0
	r.renderInlineFrame()

	// The move targets the last row, outside the damage the frame computed.
	mover.x, mover.y = 38, 5
	mover.RequestRender()
	r.renderInlineFrame()

	want := buffer.Rect{X: 38, Y: 5, Width: 1, Height: 1}
	if got := r.inlineCursorCell; got != want {
		t.Errorf("inverted cell is %+v, want %+v published during the paint", got, want)
	}
	if got := r.inlineDamaged.Intersect(want); got != want {
		t.Errorf("cursor cell %+v lies outside damage %+v", want, r.inlineDamaged)
	}
}

// resize changes a row's height the way a widget setter does.
func (g *glyphRow) resize(rows int) {
	g.LayoutNode().Height = layout.Points(rows)
	g.LayoutNode().MarkDirty()
	g.RequestRender()
}

// TestInlineDamage_GrowthPaintsAppendedRows is the quiet failure this design
// risks. Rows added below the previous height start cleared, so damage that
// failed to reach them would flush blank rows rather than visibly wrong ones.
func TestInlineDamage_GrowthPaintsAppendedRows(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	b.resize(5)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "aabbbbb" {
		t.Errorf("frame is %q, want %q with every appended row painted", got, "aabbbbb")
	}
}

func TestInlineDamage_GrowthKeepsRowsAbove(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	b.resize(4)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "aabbbb" {
		t.Errorf("frame is %q, want %q", got, "aabbbb")
	}
	// The untouched widget sits above the damage and is carried over, not redrawn.
	if got := r.inlineDamaged.Y; got < 2 {
		t.Errorf("damage starts at row %d, want the grown widget at row 2", got)
	}
}

func TestInlineDamage_ShrinkDropsRows(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 4)
	r := newDamageTestRenderer(t, a, b)

	b.resize(1)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "aab" {
		t.Errorf("frame is %q, want %q", got, "aab")
	}
}

// TestInlineDamage_TrackedFrameMatchesAfterGrowth checks the snapshot the diff
// compares against. A height change reallocates it, and rows it fails to carry
// over would show up as phantom differences on the next frame.
func TestInlineDamage_TrackedFrameMatchesAfterGrowth(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	b.resize(5)
	r.renderInlineFrame()

	tracked := r.inlineCurrentBuf
	if tracked.Height() != r.nextBuffer.Height() {
		t.Fatalf("tracked frame is %d rows, want %d", tracked.Height(), r.nextBuffer.Height())
	}
	for y := range tracked.Height() {
		if !inlineRowsEqual(tracked, r.nextBuffer, y, tracked.Width()) {
			t.Errorf("tracked row %d does not match the frame", y)
		}
	}
}

// TestInlineDamage_InsertDamagesTheNewRowsOnly covers appending a renderable to
// a root whose height does not track its content. The new renderable and the
// siblings it pushed down report the affected rows, and the root adds nothing.
func TestInlineDamage_InsertDamagesTheNewRowsOnly(t *testing.T) {
	r, _ := newInlineTestRenderer()
	for i := range 8 {
		r.root.Add(newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 2))
	}
	input := newGlyphRow("input", 'p', 39, 3)
	r.root.Add(input)
	r.renderInlineFrame()

	before := r.nextBuffer.Height()
	r.root.InsertBefore(newGlyphRow("new", 'N', 39, 2), input)
	r.renderInlineFrame()

	// The new rows and the input that moved, not the eight rows above them.
	if got := r.inlineDamaged.Y; got != before-input.Height() {
		t.Errorf("damage starts at row %d, want %d", got, before-input.Height())
	}
	if got := frameText(r.nextBuffer); got != "aabbccddeeffgghhNNppp" {
		t.Errorf("frame is %q, want the new row above the input", got)
	}
}

// TestInlineDamage_ContainerGrowthDamagesItsWholeRect documents why a growing
// transcript does not yet benefit from damage tracking.
//
// Damage is the union of a renderable's previous and current rectangle, so a
// container reports everything it covers as soon as its own height changes,
// even though only the last row moved. Reporting the difference instead would
// confine the repaint to the rows that actually changed.
func TestInlineDamage_ContainerGrowthDamagesItsWholeRect(t *testing.T) {
	container := widget.NewBox(widget.BoxWithID("transcript"),
		widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionColumn)))
	bubbles := make([]*glyphRow, 0, 10)
	for i := range 10 {
		b := newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 2)
		bubbles = append(bubbles, b)
		container.Add(b)
	}

	r, _ := newInlineTestRenderer()
	r.root.Add(container)
	r.root.Add(newGlyphRow("prompt", 'p', 39, 3))
	r.renderInlineFrame()

	// The last bubble grows one row, as a streaming message does.
	bubbles[len(bubbles)-1].resize(3)
	r.root.PrepareLayout(0)
	r.root.SyncPositions(0)

	if got := container.Damage(); got.Y != 0 {
		t.Errorf("container damage is %+v, want it to start at row 0", got)
	}
	if got := r.root.TakeDamage(); got.Y != 0 {
		t.Errorf("scene damage is %+v, want it to start at row 0", got)
	}
}

// TestInlineDamage_CursorCellIsRestored covers the inverted cell the inline
// cursor leaves behind when it moves.
func TestInlineDamage_CursorCellIsRestored(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	r, _ := newInlineTestRenderer()
	r.root.Add(a)
	r.cursorVisible = true
	r.cursorX, r.cursorY = 0, 0
	r.renderInlineFrame()

	inverted := r.nextBuffer.Cell(0, 0)
	if inverted.Fg.Equals(style.ColorWhite) {
		t.Fatal("expected the cursor cell to be inverted")
	}

	r.cursorX = 5
	r.renderInlineFrame()

	if got := r.nextBuffer.Cell(0, 0); !got.Fg.Equals(style.ColorWhite) {
		t.Error("expected the vacated cursor cell to be repainted")
	}
	if got := r.nextBuffer.Cell(5, 0); got.Fg.Equals(style.ColorWhite) {
		t.Error("expected the new cursor cell to be inverted")
	}
}

// TestInlineDamage_AncestorsAddNoDamage covers the split between the two marks.
// An ancestor only records that a descendant changed, so a change deep inside a
// tall container repaints the change rather than the container.
func TestInlineDamage_AncestorsAddNoDamage(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	r := newDamageTestRenderer(t, a, b)

	b.setGlyph('B')
	if r.root.IsDirty() {
		t.Fatal("expected the root to stay clean")
	}
	if !r.root.ChildDirty() {
		t.Fatal("expected the record to reach the root")
	}

	r.root.PrepareLayout(0)
	r.root.SyncPositions(0)

	if got := r.root.Damage(); !got.IsEmpty() {
		t.Errorf("root damage is %+v, want empty", got)
	}
	want := buffer.Rect{X: 0, Y: 2, Width: 39, Height: 2}
	if got := r.root.TakeDamage(); got != want {
		t.Errorf("scene damage is %+v, want %+v", got, want)
	}
}

// TestInlineDamage_DeepChangeStaysAtTheChange is the regression this split
// exists for. A container taller than the terminal would otherwise repaint in
// full for every token, which forces a rewrite of the whole scrollback.
func TestInlineDamage_DeepChangeStaysAtTheChange(t *testing.T) {
	bubble := widget.NewBox(widget.BoxWithID("bubble"),
		widget.BoxWithLayout(
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithFlexShrink(0),
		))
	parts := make([]*glyphRow, 0, 5)
	for i := range 5 {
		p := newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 4)
		parts = append(parts, p)
		bubble.Add(p)
	}

	r, _ := newInlineTestRenderer()
	r.root.Add(bubble)
	r.renderInlineFrame()

	last := parts[len(parts)-1]
	last.setGlyph('Z')
	r.root.PrepareLayout(0)
	r.root.SyncPositions(0)

	if got, want := r.root.TakeDamage(), last.Rect(); got != want {
		t.Errorf("damage is %+v, want the changed part %+v", got, want)
	}
}

// TestInlineDamage_WalksSkipCleanLeadingRows is the point of the whole design.
// Streaming changes the last row, so the rows above it must not be visited.
func TestInlineDamage_WalksSkipCleanLeadingRows(t *testing.T) {
	rows := make([]*glyphRow, 0, 20)
	for i := range 20 {
		rows = append(rows, newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 1))
	}
	r := newDamageTestRenderer(t, rows...)

	rows[len(rows)-1].setGlyph('Z')
	r.root.PrepareLayout(0)

	stats := r.root.SyncPositions(0)
	if stats.NodeCount > 3 {
		t.Errorf("walked %d nodes, want the root and the changed row only", stats.NodeCount)
	}
}

// TestInlineDamage_WalksFollowEverythingBelowAChange covers the other half of
// the rule. A row that grows displaces every row under it, and those rows are
// not marked when the change happens.
func TestInlineDamage_WalksFollowEverythingBelowAChange(t *testing.T) {
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	c := newGlyphRow("c", 'c', 39, 2)
	r := newDamageTestRenderer(t, a, b, c)

	a.resize(4)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "aaaabbcc" {
		t.Errorf("frame is %q, want the rows below the growth moved down", got)
	}
	if got := c.Y(); got != 6 {
		t.Errorf("row c is at %d, want 6", got)
	}
}

// TestInlineDamage_CentredContainerIsNotPruned covers the layouts the rule does
// not hold for. Centring moves the children before a change too, so none of
// them may keep the geometry of the previous frame.
func TestInlineDamage_CentredContainerIsNotPruned(t *testing.T) {
	box := widget.NewBox(widget.BoxWithID("centred"),
		widget.BoxWithLayout(
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithJustifyContent(layout.JustifyCenter),
			layout.WithHeight(layout.Points(10)),
		))
	a := newGlyphRow("a", 'a', 39, 2)
	b := newGlyphRow("b", 'b', 39, 2)
	box.Add(a)
	box.Add(b)

	r, _ := newInlineTestRenderer()
	r.root.Add(box)
	r.renderInlineFrame()

	before := a.Y()
	b.resize(4)
	r.renderInlineFrame()

	if a.Y() == before {
		t.Fatal("test is not exercising the case: row a did not move")
	}
	if got := a.Rect(); got.Y != a.Y() {
		t.Errorf("row a kept stale geometry %+v", got)
	}
}
