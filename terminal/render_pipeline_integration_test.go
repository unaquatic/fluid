package terminal

// Integration tests for the render pipeline machinery.
// These tests verify that the framework correctly orchestrates:
//   - Multi-child rendering with hooks writing to the buffer
//   - Scissor clipping (overflow:hidden) actually clipping buffer output
//   - Opacity compositing producing blended cell colors
//   - HitGrid auto-population via the render pipeline + mouse dispatch routing
//   - Z-index ordering affecting buffer output (painter's algorithm)
//   - Nested scissor clipping
//   - Full-frame render producing correct buffer after event-driven state change
//
// These tests use BaseRenderable with RenderBeforeHook since the render/
// package cannot import widget/ (circular dependency). This tests the
// framework machinery, not specific widget behavior.

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupRoot creates a RootRenderable with a HitGrid and Context wired up.
func setupRoot(w, h int) (*RootRenderable, *HitGrid) {
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	ctx := &mockContext{hitGrid: hitGrid}
	root.SetContext(ctx)
	return root, hitGrid
}

// setupRenderer creates a full Renderer with fake terminal for synchronous testing.
func setupRenderer(w, h int) (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)

	r := &Renderer{
		root:          root,
		hitGrid:       hitGrid,
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		config:        DefaultConfig(),
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		width:         w,
		height:        h,
		controlState:  StateExplicitStarted,
		parser:        input.NewParser(),
		eventBus:      input.NewEventBus(),
		pendingWork:   make(chan func()),
		wake:          make(chan struct{}, 1),
		stopChan:      make(chan struct{}, 1),
		done:          make(chan struct{}),
		sel:           render.NewSelection(),
		linkRegistry:  NewLinkRegistry(),
		stats:         NewStats(60),
		lastFrameTime: time.Now(),
	}

	root.SetContext(r)

	return r, &outBuf
}

// makeRenderChild creates a BaseRenderable positioned at (x, y) with size (w, h)
// that renders a fill character across its entire area.
func makeRenderChild(id string, x, y, w, h int, ch rune, fg, bg style.Color) *render.BaseRenderable {
	child := render.NewBaseRenderable(
		render.WithID(id),
		render.WithWidth(layout.Points(w)),
		render.WithHeight(layout.Points(h)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		cw := child.Width()
		ch2 := child.Height()
		for row := range ch2 {
			for col := range cw {
				buf.SetCell(cx+col, cy+row, ch, fg, bg, 0)
			}
		}
	}
	return child
}

// makeRenderChildAttrs creates a child that renders with specific attributes.
func makeRenderChildAttrs(id string, x, y, w, h int, ch rune, fg, bg style.Color, attrs uint32) *render.BaseRenderable {
	child := render.NewBaseRenderable(
		render.WithID(id),
		render.WithWidth(layout.Points(w)),
		render.WithHeight(layout.Points(h)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		cw := child.Width()
		ch2 := child.Height()
		for row := range ch2 {
			for col := range cw {
				buf.SetCell(cx+col, cy+row, ch, fg, bg, attrs)
			}
		}
	}
	return child
}

// ---------------------------------------------------------------------------
// Multi-child render pipeline → buffer output
// ---------------------------------------------------------------------------

// TestPipeline_MultiChildRender verifies that multiple children render their
// content to the correct positions in the buffer through the full pipeline.
func TestPipeline_MultiChildRender(t *testing.T) {
	root, _ := setupRoot(20, 3)

	// Child 1: fills row 0 with 'A' (20 wide, 1 tall)
	c1 := makeRenderChild("c1", 0, 0, 20, 1, 'A', style.ColorWhite, style.ColorTransparent)
	// Child 2: fills row 1 with 'B'
	c2 := makeRenderChild("c2", 0, 1, 20, 1, 'B', style.ColorWhite, style.ColorTransparent)
	// Child 3: fills row 2 with 'C'
	c3 := makeRenderChild("c3", 0, 2, 20, 1, 'C', style.ColorWhite, style.ColorTransparent)

	root.Add(c1)
	root.Add(c2)
	root.Add(c3)

	buf := buffer.New(20, 3)
	root.Render(buf, 0.016)

	// Verify each row has the correct character
	for x := range 5 {
		if cell := buf.Cell(x, 0); cell.Char != 'A' {
			t.Errorf("row 0 col %d: expected 'A', got %q", x, cell.Char)
		}
		if cell := buf.Cell(x, 1); cell.Char != 'B' {
			t.Errorf("row 1 col %d: expected 'B', got %q", x, cell.Char)
		}
		if cell := buf.Cell(x, 2); cell.Char != 'C' {
			t.Errorf("row 2 col %d: expected 'C', got %q", x, cell.Char)
		}
	}
}

// TestPipeline_ChildColorsPreserved verifies fg/bg colors written by children
// are preserved in the buffer.
func TestPipeline_ChildColorsPreserved(t *testing.T) {
	root, _ := setupRoot(10, 1)

	child := makeRenderChild("colored", 0, 0, 5, 1, 'X', style.ColorRed, style.ColorBlue)
	root.Add(child)

	buf := buffer.New(10, 1)
	root.Render(buf, 0.016)

	cell := buf.Cell(0, 0)
	if cell.Char != 'X' {
		t.Errorf("expected 'X', got %q", cell.Char)
	}
	if !cell.Fg.Equals(style.ColorRed) {
		t.Errorf("expected Red fg, got %v", cell.Fg)
	}
	if !cell.Bg.Equals(style.ColorBlue) {
		t.Errorf("expected Blue bg, got %v", cell.Bg)
	}
}

// ---------------------------------------------------------------------------
// Z-index ordering → buffer output (painter's algorithm)
// ---------------------------------------------------------------------------

// TestPipeline_ChildrenOrderPaintersAlgorithm verifies that later children
// paint over earlier children in the buffer (insertion order).
func TestPipeline_ChildrenOrderPaintersAlgorithm(t *testing.T) {
	root, _ := setupRoot(10, 5)

	// Both children overlap at the same position
	first := render.NewBaseRenderable(
		render.WithID("first"),
		render.WithWidth(layout.Points(5)),
		render.WithHeight(layout.Points(3)),
	)
	first.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		for y := range 3 {
			for x := range 5 {
				buf.SetCell(x, y, 'F', style.ColorWhite, style.ColorTransparent, 0)
			}
		}
	}

	second := render.NewBaseRenderable(
		render.WithID("second"),
		render.WithWidth(layout.Points(5)),
		render.WithHeight(layout.Points(3)),
	)
	second.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		for y := range 3 {
			for x := range 5 {
				buf.SetCell(x, y, 'S', style.ColorWhite, style.ColorTransparent, 0)
			}
		}
	}

	root.Add(first)
	root.Add(second)

	buf := buffer.New(10, 5)
	root.Render(buf, 0.016)

	// Second child paints last → 'S' should be visible
	cell := buf.Cell(0, 0)
	if cell.Char != 'S' {
		t.Errorf("expected 'S' (last child paints last), got %q", cell.Char)
	}
}

// ---------------------------------------------------------------------------
// Scissor clipping (overflow:hidden) → buffer output
// ---------------------------------------------------------------------------

// TestPipeline_ScissorClipsBufferOutput verifies that a child rendering outside
// its overflow:hidden parent is clipped in the buffer.
func TestPipeline_ScissorClipsBufferOutput(t *testing.T) {
	root, _ := setupRoot(20, 10)

	// Container: 10x3 at position (0,0), overflow:hidden
	container := render.NewBaseRenderable(
		render.WithID("container"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(3)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(container)

	// Child: tries to render 20 chars wide (wider than container)
	child := render.NewBaseRenderable(
		render.WithID("wide-child"),
		render.WithWidth(layout.Points(20)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		for x := range 20 {
			buf.SetCell(cx+x, cy, 'W', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	container.Add(child)

	buf := buffer.New(20, 10)
	root.Render(buf, 0.016)

	// Characters inside the container (0..9) should be 'W'
	for x := range 10 {
		cell := buf.Cell(x, 0)
		if cell.Char != 'W' {
			t.Errorf("col %d inside container: expected 'W', got %q", x, cell.Char)
		}
	}

	// Characters outside the container (10..19) should be clipped (default space)
	for x := 10; x < 20; x++ {
		cell := buf.Cell(x, 0)
		if cell.Char == 'W' {
			t.Errorf("col %d outside container: expected clipped (not 'W'), got 'W'", x)
		}
	}
}

// TestPipeline_ScissorClipsVertically verifies vertical scissor clipping.
func TestPipeline_ScissorClipsVertically(t *testing.T) {
	root, _ := setupRoot(10, 10)

	container := render.NewBaseRenderable(
		render.WithID("container"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(3)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(container)

	// Child: tries to render 10 rows tall (taller than container's 3)
	child := render.NewBaseRenderable(
		render.WithID("tall-child"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		for y := range 10 {
			buf.SetCell(cx, cy+y, 'T', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	container.Add(child)

	buf := buffer.New(10, 10)
	root.Render(buf, 0.016)

	// Rows 0..2 (inside) should have 'T'
	for y := range 3 {
		if cell := buf.Cell(0, y); cell.Char != 'T' {
			t.Errorf("row %d inside: expected 'T', got %q", y, cell.Char)
		}
	}

	// Rows 3..9 (outside) should be clipped
	for y := 3; y < 10; y++ {
		if cell := buf.Cell(0, y); cell.Char == 'T' {
			t.Errorf("row %d outside: expected clipped, got 'T'", y)
		}
	}
}

// TestPipeline_NestedScissor verifies nested overflow:hidden containers
// produce the intersection of both scissor rects.
func TestPipeline_NestedScissor(t *testing.T) {
	root, _ := setupRoot(20, 20)

	// Outer container: 10x10
	outer := render.NewBaseRenderable(
		render.WithID("outer"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(outer)

	// Inner container: 10x5 (fits in height)
	inner := render.NewBaseRenderable(
		render.WithID("inner"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
		render.WithOverflow(layout.OverflowHidden),
	)
	outer.Add(inner)

	// Child: tries to render full 10x10
	child := render.NewBaseRenderable(
		render.WithID("child"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		for y := range 10 {
			buf.SetCell(cx, cy+y, 'N', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	inner.Add(child)

	buf := buffer.New(20, 20)
	root.Render(buf, 0.016)

	// Inner is 5 tall, so only rows 0..4 should have 'N'
	for y := range 5 {
		if cell := buf.Cell(0, y); cell.Char != 'N' {
			t.Errorf("row %d inside nested: expected 'N', got %q", y, cell.Char)
		}
	}

	// Row 5+ should be clipped by inner container
	for y := 5; y < 10; y++ {
		if cell := buf.Cell(0, y); cell.Char == 'N' {
			t.Errorf("row %d outside inner: expected clipped, got 'N'", y)
		}
	}
}

// ---------------------------------------------------------------------------
// Scissor clipping → HitGrid
// ---------------------------------------------------------------------------

// TestPipeline_ScissorClipsHitGrid verifies that overflow:hidden containers
// clip the hit grid so mouse events outside are not routed to the child.
func TestPipeline_ScissorClipsHitGrid(t *testing.T) {
	root, hitGrid := setupRoot(20, 10)

	container := render.NewBaseRenderable(
		render.WithID("container"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(container)

	child := render.NewBaseRenderable(
		render.WithID("clipped-child"),
		render.WithWidth(layout.Points(20)),
		render.WithHeight(layout.Points(1)),
	)
	container.Add(child)

	buf := buffer.New(20, 10)
	root.Render(buf, 0.016)

	// Inside the container: should hit the child
	insideID := hitGrid.Check(5, 0)
	if insideID == 0 {
		t.Error("expected hit inside container bounds")
	}

	// Outside the container horizontally: should NOT hit the child
	outsideID := hitGrid.Check(15, 0)
	if outsideID == child.Num() {
		t.Error("expected no hit outside container bounds (scissor should clip hit grid)")
	}
}

// ---------------------------------------------------------------------------
// Opacity compositing → buffer output
// ---------------------------------------------------------------------------

// TestPipeline_OpacityCompositing verifies that opacity < 1.0 causes cell
// colors to be blended rather than opaque overwrites.
func TestPipeline_OpacityCompositing(t *testing.T) {
	root, _ := setupRoot(10, 3)

	// First: paint a solid red background
	bg := render.NewBaseRenderable(
		render.WithID("bg"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(3)),
	)
	bg.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		for y := range 3 {
			for x := range 10 {
				buf.SetCell(x, y, ' ', style.ColorWhite, style.ColorRed, 0)
			}
		}
	}
	root.Add(bg)

	// Second: overlay with 50% opacity blue, positioned absolutely over the bg.
	// Use PositionAbsolute so layout doesn't stack it below the bg.
	overlay := render.NewBaseRenderable(
		render.WithID("overlay"),
		render.WithOpacity(0.5),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(3)),
		render.WithPosition(layout.PositionAbsolute),
		render.WithTop(layout.Points(0)),
		render.WithLeft(layout.Points(0)),
	)
	overlay.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := overlay.X()
		cy := overlay.Y()
		for y := range 3 {
			for x := range 10 {
				buf.SetCell(cx+x, cy+y, ' ', style.ColorWhite, style.ColorBlue, 0)
			}
		}
	}
	root.Add(overlay)

	buf := buffer.New(10, 3)
	root.Render(buf, 0.016)

	// The cell should NOT be pure red or pure blue — it should be blended
	cell := buf.Cell(0, 0)
	if cell.Bg.Equals(style.ColorRed) {
		t.Error("expected bg NOT to be pure red after 50% blue overlay")
	}
	if cell.Bg.Equals(style.ColorBlue) {
		t.Error("expected bg NOT to be pure blue (opacity < 1)")
	}
	// The blended color should have some blue component from the overlay
	if cell.Bg.B < 1 {
		t.Errorf("expected blended color with blue component > 0, got %v", cell.Bg)
	}
}

// TestPipeline_FullOpacity verifies opacity=1.0 does NOT produce blend artifacts.
func TestPipeline_FullOpacity(t *testing.T) {
	root, _ := setupRoot(5, 1)

	child := makeRenderChild("opaque", 0, 0, 5, 1, 'O', style.ColorGreen, style.ColorBlack)
	root.Add(child)

	buf := buffer.New(5, 1)
	root.Render(buf, 0.016)

	cell := buf.Cell(0, 0)
	if cell.Char != 'O' {
		t.Errorf("expected 'O', got %q", cell.Char)
	}
	if !cell.Fg.Equals(style.ColorGreen) {
		t.Errorf("expected Green fg, got %v", cell.Fg)
	}
}

// ---------------------------------------------------------------------------
// HitGrid auto-population + mouse dispatch routing
// ---------------------------------------------------------------------------

// TestPipeline_HitGridAutoPopulated verifies that after a full render, the hit
// grid is populated for each visible child, enabling correct mouse dispatch.
func TestPipeline_HitGridAutoPopulated(t *testing.T) {
	root, hitGrid := setupRoot(20, 10)

	// Two children at different positions
	a := makeRenderChild("a", 0, 0, 5, 3, 'A', style.ColorWhite, style.ColorTransparent)
	b := makeRenderChild("b", 0, 0, 5, 3, 'B', style.ColorWhite, style.ColorTransparent)

	// Use layout to position them: a is 5x3 fixed, b is 5x3 below
	root.LayoutNode().FlexDirection = layout.DirectionColumn
	root.Add(a)
	root.Add(b)

	buf := buffer.New(20, 10)
	root.Render(buf, 0.016)

	// a should be at (0,0), b should be below a
	idA := hitGrid.Check(2, 1)
	if idA != a.Num() {
		t.Errorf("expected hit on 'a' at (2,1), got num=%d", idA)
	}

	// b should be at y=3 (below a which is 3 tall)
	bY := b.Y()
	if bY < 3 {
		t.Errorf("expected b.Y() >= 3, got %d", bY)
	}
	idB := hitGrid.Check(2, bY)
	if idB != b.Num() {
		t.Errorf("expected hit on 'b' at (2,%d), got num=%d", bY, idB)
	}
}

// TestPipeline_MouseDispatchViaAutoHitGrid verifies the full flow:
// render populates hit grid → mouse event dispatches to the correct child.
func TestPipeline_MouseDispatchViaAutoHitGrid(t *testing.T) {
	r, _ := setupRenderer(20, 10)

	var clicked string
	a := render.NewBaseRenderable(
		render.WithID("a"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
	)
	a.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		// render something so the child is visible
		buf.SetCell(a.X(), a.Y(), 'A', style.ColorWhite, style.ColorTransparent, 0)
	}
	a.OnMouseDownHandler = func(e *input.MouseEvent) {
		clicked = "a"
	}

	b := render.NewBaseRenderable(
		render.WithID("b"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
	)
	b.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		buf.SetCell(b.X(), b.Y(), 'B', style.ColorWhite, style.ColorTransparent, 0)
	}
	b.OnMouseDownHandler = func(e *input.MouseEvent) {
		clicked = "b"
	}

	r.root.Add(a)
	r.root.Add(b)

	// Render a frame to auto-populate the hit grid
	r.renderFrame()

	// Click on b (should be at y=5 since a takes y=0..4)
	bY := b.Y()
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      2,
			Y:      bY + 1,
		},
	})

	if clicked != "b" {
		t.Errorf("expected click dispatched to 'b', got %q", clicked)
	}
}

// ---------------------------------------------------------------------------
// renderFrame: full-frame render producing correct buffer
// ---------------------------------------------------------------------------

// TestPipeline_RenderFrameProducesCorrectBuffer verifies that renderFrame()
// produces the correct buffer content visible via the current buffer.
func TestPipeline_RenderFrameProducesCorrectBuffer(t *testing.T) {
	r, _ := setupRenderer(10, 3)
	r.config.BackgroundColor = style.ColorBlack

	child := render.NewBaseRenderable(
		render.WithID("text-like"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		text := "Hello"
		cx := child.X()
		cy := child.Y()
		for i, ch := range text {
			buf.SetCell(cx+i, cy, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.root.Add(child)

	r.renderFrame()

	// After renderFrame, the buffers are swapped. currentBuffer has the just-rendered frame.
	buf := r.CurrentBuffer()
	cell := buf.Cell(0, 0)
	if cell.Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %q", cell.Char)
	}
	cell4 := buf.Cell(4, 0)
	if cell4.Char != 'o' {
		t.Errorf("expected 'o' at (4,0), got %q", cell4.Char)
	}
}

// ---------------------------------------------------------------------------
// Event → state change → re-render
// ---------------------------------------------------------------------------

// TestPipeline_EventStateChangeRerender verifies the full loop:
// render → key event → handler mutates state → re-render → buffer updated.
func TestPipeline_EventStateChangeRerender(t *testing.T) {
	r, _ := setupRenderer(10, 1)
	r.config.BackgroundColor = style.ColorBlack

	// Create a renderable that draws its "label" state
	label := "before"
	child := render.NewBaseRenderable(
		render.WithID("reactive"),
		render.WithFocusable(true),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		for i, ch := range label {
			buf.SetCell(cx+i, cy, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	child.OnKeyDownHandler = func(e *input.KeyEvent) {
		if e.Name == "x" {
			label = "after"
		}
	}
	r.root.Add(child)
	r.focused = child

	// First render: should show "before"
	r.renderFrame()
	buf1 := r.CurrentBuffer()
	if cell := buf1.Cell(0, 0); cell.Char != 'b' {
		t.Errorf("frame 1: expected 'b' at (0,0), got %q", cell.Char)
	}

	// Dispatch key event
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key:  &input.KeyEvent{Name: "x", Type: input.KeyPress, Raw: "x", Source: "raw"},
	})

	// Second render: should show "after"
	r.renderFrame()
	buf2 := r.CurrentBuffer()
	if cell := buf2.Cell(0, 0); cell.Char != 'a' {
		t.Errorf("frame 2: expected 'a' at (0,0), got %q", cell.Char)
	}
	if cell := buf2.Cell(4, 0); cell.Char != 'r' {
		t.Errorf("frame 2: expected 'r' at (4,0), got %q", cell.Char)
	}
}

// ---------------------------------------------------------------------------
// Invisible children
// ---------------------------------------------------------------------------

// TestPipeline_InvisibleChildNotRendered verifies that an invisible child's
// render hook is never called and no content appears in the buffer.
func TestPipeline_InvisibleChildNotRendered(t *testing.T) {
	root, _ := setupRoot(10, 3)

	rendered := false
	child := render.NewBaseRenderable(
		render.WithID("hidden"),
		render.WithVisible(false),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		rendered = true
		buf.SetCell(0, 0, 'X', style.ColorWhite, style.ColorTransparent, 0)
	}
	root.Add(child)

	buf := buffer.New(10, 3)
	root.Render(buf, 0.016)

	if rendered {
		t.Error("expected invisible child's render hook NOT to be called")
	}
	if cell := buf.Cell(0, 0); cell.Char == 'X' {
		t.Error("expected no 'X' in buffer from invisible child")
	}
}

// TestPipeline_InvisibleChildNoHitGrid verifies that invisible children
// are not registered in the hit grid.
func TestPipeline_InvisibleChildNoHitGrid(t *testing.T) {
	root, hitGrid := setupRoot(10, 5)

	child := render.NewBaseRenderable(
		render.WithID("hidden-hittable"),
		render.WithVisible(false),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
	)
	root.Add(child)

	buf := buffer.New(10, 5)
	root.Render(buf, 0.016)

	id := hitGrid.Check(5, 2)
	if id == child.Num() {
		t.Error("expected invisible child NOT in hit grid")
	}
}

// ---------------------------------------------------------------------------
// RenderBeforeHook / RenderAfterHook ordering
// ---------------------------------------------------------------------------

// TestPipeline_HookOrdering verifies RenderBeforeHook fires before
// RenderAfterHook and both write to the buffer correctly.
func TestPipeline_HookOrdering(t *testing.T) {
	root, _ := setupRoot(10, 1)

	var order []string
	child := render.NewBaseRenderable(
		render.WithID("hooked"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		order = append(order, "before")
		buf.SetCell(child.X(), child.Y(), 'B', style.ColorWhite, style.ColorTransparent, 0)
	}
	child.RenderAfterHook = func(buf *buffer.Buffer, dt float64) {
		order = append(order, "after")
		// Overwrite with 'A'
		buf.SetCell(child.X(), child.Y(), 'A', style.ColorWhite, style.ColorTransparent, 0)
	}
	root.Add(child)

	buf := buffer.New(10, 1)
	root.Render(buf, 0.016)

	if len(order) != 2 || order[0] != "before" || order[1] != "after" {
		t.Errorf("expected [before, after], got %v", order)
	}

	// After hook wrote 'A' last, so buffer should have 'A'
	if cell := buf.Cell(0, 0); cell.Char != 'A' {
		t.Errorf("expected 'A' (after hook), got %q", cell.Char)
	}
}

// ---------------------------------------------------------------------------
// OnUpdate propagation
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Post-process functions
// ---------------------------------------------------------------------------

// TestPipeline_PostProcessModifiesBuffer verifies that post-process functions
// can modify the buffer after the main render pass.
func TestPipeline_PostProcessModifiesBuffer(t *testing.T) {
	r, _ := setupRenderer(10, 1)
	r.config.BackgroundColor = style.ColorBlack

	// Render a child with 'X'
	child := render.NewBaseRenderable(
		render.WithID("content"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		buf.SetCell(child.X(), child.Y(), 'X', style.ColorWhite, style.ColorBlack, 0)
	}
	r.root.Add(child)

	// Add post-process that overwrites (0,0) with 'P'
	r.AddPostProcessFn(func(buf *buffer.Buffer, dt float64) {
		buf.SetCell(0, 0, 'P', style.ColorYellow, style.ColorBlack, 0)
	})

	r.renderFrame()

	buf := r.CurrentBuffer()
	if cell := buf.Cell(0, 0); cell.Char != 'P' {
		t.Errorf("expected 'P' from post-process, got %q", cell.Char)
	}
}

// ---------------------------------------------------------------------------
// Multiple frames
// ---------------------------------------------------------------------------

// TestPipeline_MultipleFramesNoArtifacts verifies that rendering multiple
// frames correctly clears and redraws without artifacts from previous frames.
func TestPipeline_MultipleFramesNoArtifacts(t *testing.T) {
	r, _ := setupRenderer(10, 1)
	r.config.BackgroundColor = style.ColorBlack

	content := "frame1"
	child := render.NewBaseRenderable(
		render.WithID("dynamic"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt float64) {
		cx := child.X()
		cy := child.Y()
		for i, ch := range content {
			buf.SetCell(cx+i, cy, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.root.Add(child)

	// Frame 1
	r.renderFrame()
	buf1 := r.CurrentBuffer()
	if cell := buf1.Cell(0, 0); cell.Char != 'f' {
		t.Errorf("frame 1: expected 'f', got %q", cell.Char)
	}

	// Change content and render frame 2
	content = "XY"
	r.renderFrame()
	buf2 := r.CurrentBuffer()

	if cell := buf2.Cell(0, 0); cell.Char != 'X' {
		t.Errorf("frame 2: expected 'X' at (0,0), got %q", cell.Char)
	}
	if cell := buf2.Cell(1, 0); cell.Char != 'Y' {
		t.Errorf("frame 2: expected 'Y' at (1,0), got %q", cell.Char)
	}
	// Position 5 should NOT have 'e' from "frame1" — buffer was cleared
	cell5 := buf2.Cell(5, 0)
	if cell5.Char == '1' {
		t.Error("frame 2: artifact from frame 1 found at position 5")
	}
}
