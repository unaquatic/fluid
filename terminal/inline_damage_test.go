package terminal

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
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

func (g *glyphRow) RenderSelf(buf *buffer.Buffer, _ float64) {
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
func TestInlineDamage_StopsAtTheViewport(t *testing.T) {
	rows := make([]*glyphRow, 0, 15)
	for i := range 15 {
		rows = append(rows, newGlyphRow(string(rune('a'+i)), rune('a'+i), 39, 2))
	}
	r := newDamageTestRenderer(t, rows...)

	if r.inlineViewportTop == 0 {
		t.Fatalf("expected the 30-row frame to scroll a %d-row terminal", testTermHeight)
	}

	// The first widget is entirely inside the scrollback.
	rows[0].setGlyph('Z')
	r.renderInlineFrame()

	if got := r.inlineDamaged; !got.IsEmpty() && got.Y < r.inlineViewportTop {
		t.Errorf("damage %+v reaches above viewportTop=%d", got, r.inlineViewportTop)
	}
}

// TestInlineDamage_RepaintsATranslatedChild covers a child placed outside its
// parent's rectangle. A parent's rectangle says nothing about where its
// descendants paint, so damage must not cull the subtree along with the parent.
func TestInlineDamage_RepaintsATranslatedChild(t *testing.T) {
	wrap := render.NewBaseRenderable(render.WithID("wrap"),
		render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(2)))
	wrap.SetTranslateY(-6) // The child is pulled up into the spacer rows.
	child := newGlyphRow("c", 'c', 39, 2)
	wrap.Add(child)

	r, _ := newInlineTestRenderer()
	r.root.Add(newGlyphRow("head", 'H', 39, 2))
	r.root.Add(newGlyphRow("spacer", ' ', 39, 6))
	r.root.Add(wrap)
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "HHcc      " {
		t.Fatalf("baseline frame is %q, want %q", got, "HHcc      ")
	}

	// The damage covers the child's rows but not the parent's.
	child.setGlyph('C')
	r.renderInlineFrame()

	if got := frameText(r.nextBuffer); got != "HHCC      " {
		t.Errorf("frame is %q, want %q", got, "HHCC      ")
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

func (s *strayRow) RenderSelf(buf *buffer.Buffer, _ float64) {
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

func (c *cursorMover) RenderSelf(*buffer.Buffer, float64) {
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
