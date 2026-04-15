package terminal

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

func newInlineTestRenderer() (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	w, h := 40, 20
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	cfg := DefaultConfig()
	cfg.InlineMode = true
	cfg.Fullscreen = false
	r := &Renderer{
		root: root, hitGrid: hitGrid,
		backend:         NewBackend(strings.NewReader(""), &outBuf),
		config:          cfg,
		currentBuffer:   buffer.New(w, h),
		nextBuffer:      buffer.New(w, h),
		width:           w,
		height:          h,
		controlState:    StateIdle,
		parser:          input.NewParser(),
		eventBus:        input.NewEventBus(),
		pendingWork:     make(chan func(), 512),
		stopChan:        make(chan struct{}, 1),
		done:            make(chan struct{}),
		sel:             render.NewSelection(),
		linkRegistry:    NewLinkRegistry(),
		stats:           NewStats(60),
		lastFrameTime:   time.Now(),
	}
	root.SetContext(r)
	return r, &outBuf
}

func TestInlineMode_ConfigForcesNoAlternateScreen(t *testing.T) {
	cfg := mergeConfig([]Config{{InlineMode: true, Fullscreen: true}})
	if cfg.Fullscreen {
		t.Error("expected Fullscreen=false")
	}
}

func TestComputeContentHeight(t *testing.T) {
	r, _ := newInlineTestRenderer()
	if h := r.computeContentHeight(); h != 0 {
		t.Errorf("empty root: expected 0, got %d", h)
	}
	r.root.Add(render.NewBaseRenderable(render.WithID("c1"), render.WithWidth(layout.Points(40)), render.WithHeight(layout.Points(3))))
	r.root.Add(render.NewBaseRenderable(render.WithID("c2"), render.WithWidth(layout.Points(40)), render.WithHeight(layout.Points(5))))
	r.root.LayoutNode().Width = layout.Points(40)
	r.root.LayoutNode().Height = layout.Points(inlineMaxLayoutHeight)
	r.root.CalculateLayout()
	if h := r.computeContentHeight(); h != 8 {
		t.Errorf("with children: expected 8, got %d", h)
	}
}

func TestInlineFlush_FirstFrame_NoErase(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	buf := buffer.New(5, 2)
	buf.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 1, 'C', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf)
	if strings.Contains(outBuf.String(), "\x1b[2K") {
		t.Error("first frame should not erase")
	}
	// N+1: 2 content lines + 1 trailing = 3
	if r.lastInlineHeight != 3 {
		t.Errorf("expected lastInlineHeight=3 (2+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_SecondFrame_Diff(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 3 rows (A, B, C)
	buf1 := buffer.New(5, 3)
	for y := range 3 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: 2 rows (X, Y) — height shrunk, content changed
	buf2 := buffer.New(5, 2)
	buf2.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 1, 'Y', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Diff path: 2 dirty rows (0,1 changed) + 1 erased row (2 removed) = 3 erases
	if c := strings.Count(output, "\x1b[2K"); c != 3 {
		t.Errorf("expected 3 erases (2 dirty + 1 removed), got %d", c)
	}
	// Should contain the new content
	if !strings.Contains(output, "X") || !strings.Contains(output, "Y") {
		t.Error("expected new content X and Y in output")
	}
	// lastInlineHeight should reflect new height
	if r.lastInlineHeight != 3 {
		t.Errorf("expected lastInlineHeight=3 (2+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_Overflow_ClearTerminal(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// termHeight = 20. Write 25 rows (overflow).
	buf := buffer.New(10, 25)
	for y := range 25 {
		buf.SetCell(0, y, rune('a'+y%26), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf)
	output := outBuf.String()

	// Should use clearTerminal (clear screen + scrollback + home)
	if !strings.Contains(output, "\x1b[2J\x1b[3J\x1b[H") {
		t.Error("overflow should use clearTerminal")
	}
	// Should NOT use per-line erase
	if strings.Contains(output, "\x1b[2K") {
		t.Error("overflow should not use per-line erase")
	}
	// 25 content + 1 trailing = 26
	if r.lastInlineHeight != 26 {
		t.Errorf("expected lastInlineHeight=26 (25+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_Overflow_Subsequent_StillClears(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: overflow
	buf1 := buffer.New(10, 25)
	for y := range 25 {
		buf1.SetCell(0, y, 'A', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: still overflow (lastInlineHeight=26 > termHeight=20)
	buf2 := buffer.New(10, 22)
	for y := range 22 {
		buf2.SetCell(0, y, 'B', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Should still use clearTerminal because lastInlineHeight > termHeight
	if !strings.Contains(output, "\x1b[2J\x1b[3J\x1b[H") {
		t.Error("subsequent overflow should still use clearTerminal")
	}
}

func TestInlineFlush_Overflow_Discards_CurrentBuf(t *testing.T) {
	r, _ := newInlineTestRenderer()
	// Frame 1: non-overflow
	buf1 := buffer.New(10, 5)
	buf1.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf1)
	if r.inlineCurrentBuf == nil {
		t.Fatal("expected inlineCurrentBuf after non-overflow flush")
	}

	// Frame 2: overflow
	buf2 := buffer.New(10, 25)
	for y := range 25 {
		buf2.SetCell(0, y, 'B', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	if r.inlineCurrentBuf != nil {
		t.Error("expected inlineCurrentBuf=nil after overflow")
	}
}

// ---------------------------------------------------------------------------
// Diff path tests
// ---------------------------------------------------------------------------

func TestInlineDiff_NothingChanged_ZeroOutput(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	buf := buffer.New(10, 5)
	buf.SetCell(0, 0, 'H', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 1, 'e', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 2, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 3, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 4, 'o', style.ColorWhite, style.ColorBlack, 0)

	// Frame 1: full flush
	r.InlineFlush(buf)
	outBuf.Reset()

	// Frame 2: identical content
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'H', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 1, 'e', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 2, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 3, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 4, 'o', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)

	// Should produce zero output — nothing changed
	if outBuf.Len() != 0 {
		t.Errorf("expected 0 bytes for identical frame, got %d", outBuf.Len())
	}
}

func TestInlineDiff_OnlyChangedRowsRewritten(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 5 rows with multi-char content to avoid false positives
	// from escape sequences (e.g. \x1b[5A contains 'A').
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: same 5 rows, only row 2 changed
	buf2 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	// Change row 2 to all 'X'
	for x := range 5 {
		buf2.SetCell(x, 2, 'X', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Should erase only 1 row (the changed row)
	if c := strings.Count(output, "\x1b[2K"); c != 1 {
		t.Errorf("expected 1 erase for 1 dirty row, got %d", c)
	}
	// Should contain the changed content
	if !strings.Contains(output, "XXXXX") {
		t.Error("expected changed row content XXXXX in output")
	}
	// Unchanged row content (e.g. "aaaaa") should NOT appear
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "bbbbb") ||
		strings.Contains(output, "ddddd") || strings.Contains(output, "eeeee") {
		t.Error("unchanged rows should not appear in output")
	}
}

func TestInlineDiff_HeightGrew_AppendsNewRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 3 rows
	buf1 := buffer.New(10, 3)
	for y := range 3 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: 5 rows, first 3 unchanged, 2 new rows added
	buf2 := buffer.New(10, 5)
	for y := range 3 {
		buf2.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	buf2.SetCell(0, 3, 'D', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 4, 'E', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Should only write the 2 new rows, not rewrite old ones
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 2 {
		t.Errorf("expected 2 erases (2 new rows), got %d", eraseCount)
	}
	// Should contain new content
	if !strings.Contains(output, "D") || !strings.Contains(output, "E") {
		t.Error("expected new content D and E")
	}
	// Should NOT contain old content
	if strings.Contains(output, "A") || strings.Contains(output, "B") || strings.Contains(output, "C") {
		t.Error("unchanged rows A, B, C should not be in output")
	}
	// lastInlineHeight updated
	if r.lastInlineHeight != 6 {
		t.Errorf("expected lastInlineHeight=6 (5+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineDiff_HeightShrunk_ErasesRemovedRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 5 rows with multi-char content
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: 3 rows, same content for rows 0-2
	buf2 := buffer.New(10, 3)
	for y := range 3 {
		for x := range 5 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Should erase 2 removed rows (rows 3 and 4)
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 2 {
		t.Errorf("expected 2 erases (2 removed rows), got %d", eraseCount)
	}
	// Unchanged common rows should NOT be rewritten
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "bbbbb") || strings.Contains(output, "ccccc") {
		t.Error("unchanged common rows should not be rewritten")
	}
	if r.lastInlineHeight != 4 {
		t.Errorf("expected lastInlineHeight=4 (3+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineDiff_HeightGrewAndRowsChanged(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 3 rows with multi-char content
	buf1 := buffer.New(10, 3)
	for y := range 3 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: 5 rows, row 1 changed, 2 new rows
	buf2 := buffer.New(10, 5)
	for x := range 5 {
		buf2.SetCell(x, 0, 'a', style.ColorWhite, style.ColorBlack, 0) // same
		buf2.SetCell(x, 1, 'X', style.ColorWhite, style.ColorBlack, 0) // changed
		buf2.SetCell(x, 2, 'c', style.ColorWhite, style.ColorBlack, 0) // same
		buf2.SetCell(x, 3, 'd', style.ColorWhite, style.ColorBlack, 0) // new
		buf2.SetCell(x, 4, 'e', style.ColorWhite, style.ColorBlack, 0) // new
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// 1 dirty row + 2 new rows = 3 erases
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 3 {
		t.Errorf("expected 3 erases (1 dirty + 2 new), got %d", eraseCount)
	}
	if !strings.Contains(output, "XXXXX") || !strings.Contains(output, "ddddd") || !strings.Contains(output, "eeeee") {
		t.Error("expected changed/new content XXXXX, ddddd, eeeee")
	}
	// Unchanged rows should not be rewritten
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "ccccc") {
		t.Error("unchanged rows should not be rewritten")
	}
}

func TestInlineDiff_WidthChange_FallsBackToFull(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: width 10
	buf1 := buffer.New(10, 3)
	buf1.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: width 15 (different from prev buffer's width)
	buf2 := buffer.New(15, 3)
	buf2.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Width changed: should use full flush (erase + rewrite all)
	// lastInlineHeight was 4 (3+1), so should erase 4 lines
	if c := strings.Count(output, "\x1b[2K"); c != 4 {
		t.Errorf("expected 4 erases from full flush, got %d", c)
	}
}

func TestInlineDiff_AfterOverflow_FullFlush(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: overflow (clears inlineCurrentBuf)
	buf1 := buffer.New(10, 25) // > termHeight=20
	for y := range 25 {
		buf1.SetCell(0, y, 'A', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	if r.inlineCurrentBuf != nil {
		t.Fatal("expected nil inlineCurrentBuf after overflow")
	}

	// Frame 2: height fits, but lastInlineHeight (26) > termHeight (20)
	// → still overflow. Need this frame to bring lastInlineHeight down.
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	// After this overflow frame: lastInlineHeight = 6, inlineCurrentBuf = nil

	// Frame 3: now lastInlineHeight (6) <= termHeight (20) → non-overflow.
	// But inlineCurrentBuf is nil → full flush. Saves buffer.
	outBuf.Reset()
	buf3 := buffer.New(10, 5)
	buf3.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf3)
	if r.inlineCurrentBuf == nil {
		t.Error("expected inlineCurrentBuf after non-overflow full flush")
	}

	// Frame 4: same content → diff path → zero output
	outBuf.Reset()
	buf4 := buffer.New(10, 5)
	buf4.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf4)
	if outBuf.Len() != 0 {
		t.Errorf("expected 0 bytes for identical frame after recovery, got %d", outBuf.Len())
	}
}

func TestInlineDiff_CursorMovement_Up(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 5 rows
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// Frame 2: only row 0 changed
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0) // changed
	for y := 1; y < 5; y++ {
		buf2.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Cursor starts at row 5 (trailing), needs to move to row 0: up 5
	if !strings.Contains(output, "\x1b[5A") {
		t.Error("expected cursor up 5 to reach row 0")
	}
	// After writing row 0, needs to return to row 5: down 5
	if !strings.Contains(output, "\x1b[5B") {
		t.Error("expected cursor down 5 to return to trailing position")
	}
}

func TestInlineDiff_MultipleRows_MinimalOutput(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Frame 1: 10 rows of wide content (30 chars each).
	// Wide content ensures the diff savings outweigh per-row overhead.
	buf1 := buffer.New(40, 10)
	for y := range 10 {
		for x := range 30 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	firstBytes := outBuf.Len()
	outBuf.Reset()

	// Frame 2: only rows 3 and 7 changed
	buf2 := buffer.New(40, 10)
	for y := range 10 {
		for x := range 30 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	for x := range 30 {
		buf2.SetCell(x, 3, 'X', style.ColorWhite, style.ColorBlack, 0)
		buf2.SetCell(x, 7, 'Y', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	diffBytes := outBuf.Len()

	// Diff output should be significantly smaller than full flush
	if diffBytes >= firstBytes {
		t.Errorf("diff (%d bytes) should be much smaller than full flush (%d bytes)", diffBytes, firstBytes)
	}
	// Only 2 rows changed
	if c := strings.Count(outBuf.String(), "\x1b[2K"); c != 2 {
		t.Errorf("expected 2 erases, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// Existing tests (unchanged)
// ---------------------------------------------------------------------------

func TestRenderInlineFrame_LayoutWidthMargin(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.renderInlineFrame()
	if r.root.LayoutNode().Width.Value != 39 {
		t.Errorf("expected layout width=39, got %v", r.root.LayoutNode().Width.Value)
	}
}

func TestRenderInlineFrame_DynamicHeight(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.root.Add(render.NewBaseRenderable(render.WithID("c1"), render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(3))))
	r.renderInlineFrame()
	// 3 content + 1 trailing = 4
	if r.lastInlineHeight != 4 {
		t.Errorf("expected 4 (3+1), got %d", r.lastInlineHeight)
	}
	r.root.Add(render.NewBaseRenderable(render.WithID("c2"), render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(5))))
	r.renderInlineFrame()
	// 8 content + 1 trailing = 9
	if r.lastInlineHeight != 9 {
		t.Errorf("expected 9 (8+1), got %d", r.lastInlineHeight)
	}
}

func TestProcessResize_InlineMode_Initial(t *testing.T) {
	r, _ := newInlineTestRenderer()
	// Before any render, lastInlineHeight == 0.
	r.processResize(100, 50)
	if r.width != 100 {
		t.Errorf("expected width=100, got %d", r.width)
	}
	// Should stay 0 — no previous output to erase, don't nuke screen.
	if r.lastInlineHeight != 0 {
		t.Errorf("expected lastInlineHeight=0 (initial), got %d", r.lastInlineHeight)
	}
}

func TestProcessResize_InlineMode_AfterRender(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.lastInlineHeight = 5 // simulate having rendered
	r.processResize(100, 50)
	// Should be termHeight+1 to force clearTerminal on next render.
	if r.lastInlineHeight != 51 {
		t.Errorf("expected lastInlineHeight=51 (termHeight+1), got %d", r.lastInlineHeight)
	}
}

func TestPrepareLayout_ExecuteRender_Split(t *testing.T) {
	root := NewRootRenderable(40, 20)
	hitGrid := NewHitGrid(40, 20)
	root.SetContext(&mockContext{hitGrid: hitGrid})
	child := render.NewBaseRenderable(render.WithID("child"), render.WithWidth(layout.Points(20)), render.WithHeight(layout.Points(5)))
	root.Add(child)
	root.PrepareLayout(0.016)
	if child.LayoutNode().Computed.Width != 20 {
		t.Errorf("expected 20, got %d", child.LayoutNode().Computed.Width)
	}
	buf := buffer.New(40, 20)
	execStats := root.ExecuteRender(buf, 0.016)
	if execStats.CmdCount == 0 {
		t.Error("expected at least one render command")
	}
}

func TestDestroy_InlineMode(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.lastInlineHeight = 5
	r.Destroy()
	if !strings.Contains(outBuf.String(), "\r\n") {
		t.Error("expected \\r\\n on destroy")
	}
}
