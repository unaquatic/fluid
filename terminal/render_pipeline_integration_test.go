package terminal

// These integration tests exercise scene rendering, clipping, hit
// testing, event-driven updates, and painter-order behavior without depending
// on package widget.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// setupRoot returns a root with an in-memory hit-grid context.
func setupRoot(w, h int) (*RootRenderable, *HitGrid) {
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	host := &mockHost{hitGrid: hitGrid}
	root.SetHost(host)
	return root, hitGrid
}

// setupRenderer returns a renderer with in-memory terminal streams.
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

	root.SetHost(r)

	return r, &outBuf
}

// makeRenderChild returns a fixed-size renderable that fills its resolved area.
func makeRenderChild(id string, x, y, w, h int, ch rune, fg, bg style.Color) *render.BaseRenderable {
	child := render.NewBaseRenderable(
		render.WithID(id),
		render.WithWidth(layout.Points(w)),
		render.WithHeight(layout.Points(h)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
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

// makeRenderChildAttrs returns a fixed-size fill renderable with attrs.
func makeRenderChildAttrs(id string, x, y, w, h int, ch rune, fg, bg style.Color, attrs style.Attrs) *render.BaseRenderable {
	child := render.NewBaseRenderable(
		render.WithID(id),
		render.WithWidth(layout.Points(w)),
		render.WithHeight(layout.Points(h)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
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

// TestPipeline_MultiChildRender verifies resolved output from multiple children.
func TestPipeline_MultiChildRender(t *testing.T) {
	root, _ := setupRoot(20, 3)

	// Three one-row children fill successive rows.
	c1 := makeRenderChild("c1", 0, 0, 20, 1, 'A', style.ColorWhite, style.ColorTransparent)
	c2 := makeRenderChild("c2", 0, 1, 20, 1, 'B', style.ColorWhite, style.ColorTransparent)
	c3 := makeRenderChild("c3", 0, 2, 20, 1, 'C', style.ColorWhite, style.ColorTransparent)

	root.Add(c1)
	root.Add(c2)
	root.Add(c3)

	buf := buffer.New(20, 3)
	root.Render(buf, 16*time.Millisecond)

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

// TestPipeline_ChildColorsPreserved verifies that child colors reach the frame.
func TestPipeline_ChildColorsPreserved(t *testing.T) {
	root, _ := setupRoot(10, 1)

	child := makeRenderChild("colored", 0, 0, 5, 1, 'X', style.ColorRed, style.ColorBlue)
	root.Add(child)

	buf := buffer.New(10, 1)
	root.Render(buf, 16*time.Millisecond)

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

// TestPipeline_ChildrenOrderPaintersAlgorithm verifies that later siblings
// paint over earlier siblings.
func TestPipeline_ChildrenOrderPaintersAlgorithm(t *testing.T) {
	root, _ := setupRoot(10, 5)

	// Both children paint the same area.
	first := render.NewBaseRenderable(
		render.WithID("first"),
		render.WithWidth(layout.Points(5)),
		render.WithHeight(layout.Points(3)),
	)
	first.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
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
	second.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		for y := range 3 {
			for x := range 5 {
				buf.SetCell(x, y, 'S', style.ColorWhite, style.ColorTransparent, 0)
			}
		}
	}

	root.Add(first)
	root.Add(second)

	buf := buffer.New(10, 5)
	root.Render(buf, 16*time.Millisecond)

	// The second child paints last.
	cell := buf.Cell(0, 0)
	if cell.Char != 'S' {
		t.Errorf("expected 'S' (last child paints last), got %q", cell.Char)
	}
}

// TestPipeline_ScissorClipsBufferOutput verifies that a child rendering outside
// an OverflowHidden parent is clipped in the frame.
func TestPipeline_ScissorClipsBufferOutput(t *testing.T) {
	root, _ := setupRoot(20, 10)

	// The container clips descendants to ten columns and three rows.
	container := render.NewBaseRenderable(
		render.WithID("container"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(3)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(container)

	// The child attempts to paint twice the container width.
	child := render.NewBaseRenderable(
		render.WithID("wide-child"),
		render.WithWidth(layout.Points(20)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		cx := child.X()
		cy := child.Y()
		for x := range 20 {
			buf.SetCell(cx+x, cy, 'W', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	container.Add(child)

	buf := buffer.New(20, 10)
	root.Render(buf, 16*time.Millisecond)

	// Columns inside the container are painted.
	for x := range 10 {
		cell := buf.Cell(x, 0)
		if cell.Char != 'W' {
			t.Errorf("col %d inside container: expected 'W', got %q", x, cell.Char)
		}
	}

	// Columns beyond the container remain unpainted.
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

	// The child attempts to paint ten rows through a three-row clip.
	child := render.NewBaseRenderable(
		render.WithID("tall-child"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		cx := child.X()
		cy := child.Y()
		for y := range 10 {
			buf.SetCell(cx, cy+y, 'T', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	container.Add(child)

	buf := buffer.New(10, 10)
	root.Render(buf, 16*time.Millisecond)

	// Rows inside the container are painted.
	for y := range 3 {
		if cell := buf.Cell(0, y); cell.Char != 'T' {
			t.Errorf("row %d inside: expected 'T', got %q", y, cell.Char)
		}
	}

	// Rows beyond the container remain unpainted.
	for y := 3; y < 10; y++ {
		if cell := buf.Cell(0, y); cell.Char == 'T' {
			t.Errorf("row %d outside: expected clipped, got 'T'", y)
		}
	}
}

// TestPipeline_NestedScissor verifies intersection of nested clipping regions.
func TestPipeline_NestedScissor(t *testing.T) {
	root, _ := setupRoot(20, 20)

	// The outer clip is ten rows high.
	outer := render.NewBaseRenderable(
		render.WithID("outer"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
		render.WithOverflow(layout.OverflowHidden),
	)
	root.Add(outer)

	// The inner clip restricts descendants to five rows.
	inner := render.NewBaseRenderable(
		render.WithID("inner"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
		render.WithOverflow(layout.OverflowHidden),
	)
	outer.Add(inner)

	// The child attempts to paint all ten rows.
	child := render.NewBaseRenderable(
		render.WithID("child"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(10)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		cx := child.X()
		cy := child.Y()
		for y := range 10 {
			buf.SetCell(cx, cy+y, 'N', style.ColorWhite, style.ColorTransparent, 0)
		}
	}
	inner.Add(child)

	buf := buffer.New(20, 20)
	root.Render(buf, 16*time.Millisecond)

	// Rows inside both clips are painted.
	for y := range 5 {
		if cell := buf.Cell(0, y); cell.Char != 'N' {
			t.Errorf("row %d inside nested: expected 'N', got %q", y, cell.Char)
		}
	}

	// The inner clip excludes all later rows.
	for y := 5; y < 10; y++ {
		if cell := buf.Cell(0, y); cell.Char == 'N' {
			t.Errorf("row %d outside inner: expected clipped, got 'N'", y)
		}
	}
}

// TestPipeline_ScissorClipsHitGrid verifies that visual clipping also restricts
// hit registration.
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
	root.Render(buf, 16*time.Millisecond)

	// A point inside the clip resolves to a target.
	insideID := hitGrid.Check(5, 0)
	if insideID == 0 {
		t.Error("expected hit inside container bounds")
	}

	// A point beyond the horizontal clip does not resolve to the child.
	outsideID := hitGrid.Check(15, 0)
	if outsideID == child.Num() {
		t.Error("expected no hit outside container bounds (scissor should clip hit grid)")
	}
}

// TestPipeline_ExactCellOutput verifies the exact cell a child paints.
func TestPipeline_ExactCellOutput(t *testing.T) {
	root, _ := setupRoot(5, 1)

	child := makeRenderChild("opaque", 0, 0, 5, 1, 'O', style.ColorGreen, style.ColorBlack)
	root.Add(child)

	buf := buffer.New(5, 1)
	root.Render(buf, 16*time.Millisecond)

	cell := buf.Cell(0, 0)
	if cell.Char != 'O' {
		t.Errorf("expected 'O', got %q", cell.Char)
	}
	if !cell.Fg.Equals(style.ColorGreen) {
		t.Errorf("expected Green fg, got %v", cell.Fg)
	}
}

// TestPipeline_HitGridAutoPopulated verifies hit registration for visible
// children after rendering.
func TestPipeline_HitGridAutoPopulated(t *testing.T) {
	root, hitGrid := setupRoot(20, 10)

	// Lay out two fixed children in a column.
	a := makeRenderChild("a", 0, 0, 5, 3, 'A', style.ColorWhite, style.ColorTransparent)
	b := makeRenderChild("b", 0, 0, 5, 3, 'B', style.ColorWhite, style.ColorTransparent)

	root.LayoutNode().FlexDirection = layout.DirectionColumn
	root.Add(a)
	root.Add(b)

	buf := buffer.New(20, 10)
	root.Render(buf, 16*time.Millisecond)

	// The first child's area resolves to its ID.
	idA := hitGrid.Check(2, 1)
	if idA != a.Num() {
		t.Errorf("expected hit on 'a' at (2,1), got num=%d", idA)
	}

	// The second child begins below the first.
	bY := b.Y()
	if bY < 3 {
		t.Errorf("expected b.Y() >= 3, got %d", bY)
	}
	idB := hitGrid.Check(2, bY)
	if idB != b.Num() {
		t.Errorf("expected hit on 'b' at (2,%d), got num=%d", bY, idB)
	}
}

// TestPipeline_MouseDispatchViaAutoHitGrid verifies mouse dispatch through a
// hit grid populated by rendering.
func TestPipeline_MouseDispatchViaAutoHitGrid(t *testing.T) {
	r, _ := setupRenderer(20, 10)

	var clicked string
	a := render.NewBaseRenderable(
		render.WithID("a"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(5)),
	)
	a.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		// Paint one cell while registering the child's full hit region.
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
	b.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		buf.SetCell(b.X(), b.Y(), 'B', style.ColorWhite, style.ColorTransparent, 0)
	}
	b.OnMouseDownHandler = func(e *input.MouseEvent) {
		clicked = "b"
	}

	r.root.Add(a)
	r.root.Add(b)

	// Render once to populate hit targets.
	r.renderFrame()

	// Dispatch a click inside b's resolved area.
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

// TestPipeline_RenderFrameProducesCorrectBuffer verifies that renderFrame copies
// rendered output into CurrentBuffer.
func TestPipeline_RenderFrameProducesCorrectBuffer(t *testing.T) {
	r, _ := setupRenderer(10, 3)
	r.config.BackgroundColor = style.ColorBlack

	child := render.NewBaseRenderable(
		render.WithID("text-like"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		text := "Hello"
		cx := child.X()
		cy := child.Y()
		for i, ch := range text {
			buf.SetCell(cx+i, cy, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.root.Add(child)

	r.renderFrame()

	// renderFrame copies the completed next buffer into currentBuffer.
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

// TestPipeline_EventStateChangeRerender verifies that a key-driven state change
// appears in the next rendered frame.
func TestPipeline_EventStateChangeRerender(t *testing.T) {
	r, _ := setupRenderer(10, 1)
	r.config.BackgroundColor = style.ColorBlack

	// Draw the current label on every frame.
	label := "before"
	child := render.NewBaseRenderable(
		render.WithID("reactive"),
		render.WithFocusable(true),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
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

	// The first frame contains the initial label.
	r.renderFrame()
	buf1 := r.CurrentBuffer()
	if cell := buf1.Cell(0, 0); cell.Char != 'b' {
		t.Errorf("frame 1: expected 'b' at (0,0), got %q", cell.Char)
	}

	// The key handler changes the label.
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key:  &input.KeyEvent{Name: "x", Type: input.KeyPress, Raw: "x", Source: input.KeySourceRaw},
	})

	// The second frame contains the updated label.
	r.renderFrame()
	buf2 := r.CurrentBuffer()
	if cell := buf2.Cell(0, 0); cell.Char != 'a' {
		t.Errorf("frame 2: expected 'a' at (0,0), got %q", cell.Char)
	}
	if cell := buf2.Cell(4, 0); cell.Char != 'r' {
		t.Errorf("frame 2: expected 'r' at (4,0), got %q", cell.Char)
	}
}

// TestPipeline_InvisibleChildNotRendered verifies that hidden children do not
// paint.
func TestPipeline_InvisibleChildNotRendered(t *testing.T) {
	root, _ := setupRoot(10, 3)

	rendered := false
	child := render.NewBaseRenderable(
		render.WithID("hidden"),
		render.WithVisible(false),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		rendered = true
		buf.SetCell(0, 0, 'X', style.ColorWhite, style.ColorTransparent, 0)
	}
	root.Add(child)

	buf := buffer.New(10, 3)
	root.Render(buf, 16*time.Millisecond)

	if rendered {
		t.Error("expected invisible child's render hook NOT to be called")
	}
	if cell := buf.Cell(0, 0); cell.Char == 'X' {
		t.Error("expected no 'X' in buffer from invisible child")
	}
}

// TestPipeline_InvisibleChildNoHitGrid verifies that hidden children are not
// registered for hit testing.
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
	root.Render(buf, 16*time.Millisecond)

	id := hitGrid.Check(5, 2)
	if id == child.Num() {
		t.Error("expected invisible child NOT in hit grid")
	}
}

// TestPipeline_HookOrdering verifies before-hook and after-hook execution order.
func TestPipeline_HookOrdering(t *testing.T) {
	root, _ := setupRoot(10, 1)

	var order []string
	child := render.NewBaseRenderable(
		render.WithID("hooked"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		order = append(order, "before")
		buf.SetCell(child.X(), child.Y(), 'B', style.ColorWhite, style.ColorTransparent, 0)
	}
	child.RenderAfterHook = func(buf *buffer.Buffer, dt time.Duration) {
		order = append(order, "after")
		// The after hook overwrites the before hook's cell.
		buf.SetCell(child.X(), child.Y(), 'A', style.ColorWhite, style.ColorTransparent, 0)
	}
	root.Add(child)

	buf := buffer.New(10, 1)
	root.Render(buf, 16*time.Millisecond)

	if len(order) != 2 || order[0] != "before" || order[1] != "after" {
		t.Errorf("expected [before, after], got %v", order)
	}

	// The final cell contains the after hook's output.
	if cell := buf.Cell(0, 0); cell.Char != 'A' {
		t.Errorf("expected 'A' (after hook), got %q", cell.Char)
	}
}

// TestPipeline_MultipleFramesNoArtifacts verifies that shorter replacement
// content leaves no glyphs from a previous frame.
func TestPipeline_MultipleFramesNoArtifacts(t *testing.T) {
	r, _ := setupRenderer(10, 1)
	r.config.BackgroundColor = style.ColorBlack

	content := "frame1"
	child := render.NewBaseRenderable(
		render.WithID("dynamic"),
		render.WithWidth(layout.Points(10)),
		render.WithHeight(layout.Points(1)),
	)
	child.RenderBeforeHook = func(buf *buffer.Buffer, dt time.Duration) {
		cx := child.X()
		cy := child.Y()
		for i, ch := range content {
			buf.SetCell(cx+i, cy, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.root.Add(child)

	// Render the initial longer content.
	r.renderFrame()
	buf1 := r.CurrentBuffer()
	if cell := buf1.Cell(0, 0); cell.Char != 'f' {
		t.Errorf("frame 1: expected 'f', got %q", cell.Char)
	}

	// Replace it with shorter content for the next frame.
	content = "XY"
	r.renderFrame()
	buf2 := r.CurrentBuffer()

	if cell := buf2.Cell(0, 0); cell.Char != 'X' {
		t.Errorf("frame 2: expected 'X' at (0,0), got %q", cell.Char)
	}
	if cell := buf2.Cell(1, 0); cell.Char != 'Y' {
		t.Errorf("frame 2: expected 'Y' at (1,0), got %q", cell.Char)
	}
	// The cleared frame does not retain the final digit from "frame1".
	cell5 := buf2.Cell(5, 0)
	if cell5.Char == '1' {
		t.Error("frame 2: artifact from frame 1 found at position 5")
	}
}

// TestPipeline_FullscreenStatsCountNodes covers the statistics a fullscreen
// frame reports. Draw never walks the scene, so ExecuteRender has to carry the
// node count over from SyncPositions.
func TestPipeline_FullscreenStatsCountNodes(t *testing.T) {
	r, _ := setupRenderer(10, 4)
	for i := range 3 {
		r.root.Add(render.NewBaseRenderable(
			render.WithID(fmt.Sprintf("child-%d", i)),
			render.WithWidth(layout.Points(10)),
			render.WithHeight(layout.Points(1)),
		))
	}

	r.renderFrame()

	// Three children plus the root.
	if got := r.root.LastFrameStats.NodeCount; got != 4 {
		t.Errorf("NodeCount is %d, want 4", got)
	}
	if got := r.root.LastFrameStats.CmdCount; got != 4 {
		t.Errorf("CmdCount is %d, want 4", got)
	}
}
